package hydrofoil

import (
	"log"
	"math/rand"
	"time"
)

func (r *Replica) handleReplicateEntries(rpc *ReplicateEntries) {
	// startTime := time.Now().UnixMicro()
	r.handleIncomingTerm(rpc)
	if r.term > int(rpc.Term) || r.isLogMoreUpToDate(rpc) == MoreUpToDate {
		if r.term == int(rpc.Term) {
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		
			if !r.benOrState.benOrRunning {
				timeout = rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
				setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
			}
		
			// dlog.Println("Replica", r.Id, "lastHeardFromLeader", time.Now(), "diff is", time.Now().Sub(r.lastHeardFromLeader))
			r.lastHeardFromLeader = time.Now()
		}

		// dlog.Println("strange", r.Id, r.term, int(rpc.Term), r.isLogMoreUpToDate(rpc), "our data", r.commitIndex, r.leaderTerm, len(r.log), "rpc data", rpc.GetCommitIndex(), rpc.GetLeaderTerm(), rpc.GetLogLength())

		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(), NewRequestedIndex: int32(r.commitIndex) + 1, 
			MsgTimestamp: time.Now().UnixNano(),
		}

		// dlog.Println("Replica", r.Id, "sending to", rpc.SenderId, "ReplicateEntriesReply", "AAAA")

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)

		if r.term == int(rpc.Term) {
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		// fmt.Println("Replica", r.Id, "type 1", time.Now().UnixMicro() - startTime)
		return
	}

	// at this point, r.term == rpc.Term
	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	if r.candidateState.isCandidate {
		r.candidateState = emptyCandidateState
	}

	if !r.benOrState.benOrRunning {
		timeout = rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
		setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
	}
	// fmt.Println("Replica", r.Id, "type 2", time.Now().UnixMicro() - startTime)

	// startTime = time.Now().UnixMicro()

	// dlog.Println("Replica", r.Id, "lastHeardFromLeader", time.Now(), "diff is", time.Now().Sub(r.lastHeardFromLeader))
	r.lastHeardFromLeader = time.Now()

	// fmt.Println("Replica", r.Id, "type 2.1", time.Now().UnixMicro() - startTime)
	// startTime = time.Now().UnixMicro()

	if int(rpc.PrevLogIndex) >= len(r.log) || 
		(r.log[rpc.PrevLogIndex].ServerId != rpc.PrevLogServerId && r.log[rpc.PrevLogIndex].Timestamp != rpc.PrevLogTimestamp) {
		
		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(), NewRequestedIndex: int32(r.commitIndex) + 1,
			MsgTimestamp: time.Now().UnixNano(), 
		}

		// dlog.Println("Replica", r.Id, "sending to", rpc.SenderId, "ReplicateEntriesReply", "BBBB")

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		// fmt.Println("Replica", r.Id, "type 2.5", time.Now().UnixMicro() - startTime)
		return
	}

	// fmt.Println("Replica", r.Id, "type 2.1", time.Now().UnixMicro() - startTime)
	// startTime = time.Now().UnixMicro()

	if r.leaderState.isLeader {
		log.Fatal("Should not be leader when receiving a ReplicateEntries RPC, and have a greater term!")
	}

	oldCommitIndex := r.commitIndex

	// dlog.Println("Replica", r.Id, "is accepting entries from", rpc.SenderId, "before log", logToString(r.log))

	// fmt.Println("Replica", r.Id, "type 3", time.Now().UnixMicro() - startTime)
	// startTime = time.Now().UnixMicro()

	var potentialEntries []Entry
	shouldReplaceLog, startReplacementIdx := r.shouldLogBeReplaced(rpc, max(r.commitIndex, int(rpc.PrevLogIndex)) + 1)
	if shouldReplaceLog {
		potentialEntries = r.replaceExistingLog(rpc, startReplacementIdx)
	} else {
		potentialEntries = rpc.GetEntries()
	}

	// dlog.Println("Replica", r.Id, "is accepting entries from", rpc.SenderId, "after log", logToString(r.log))

	for _, v := range(potentialEntries) {
		if !r.seenBefore(v) { r.pq.push(v) }
	}

	// dlog.Println("Replica", r.Id, "pq values4", r.pq.extractList())

	r.commitIndex = int(rpc.CommitIndex)
	r.leaderTerm = max(r.leaderTerm, int(rpc.LeaderTerm))
	if r.commitIndex > oldCommitIndex {
		// if !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
		// 	r.pq.push(r.benOrState.benOrBroadcastEntry)
		// }
		// dlog.Println("Replica", r.Id, "pq values2", r.pq.extractList())
		r.benOrState = emptyBenOrState

		if !r.leaderState.isLeader {
			timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
			setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
		}

		clearTimer(r.benOrResendTimer)
	}

	entries := make([]Entry, 0)
	if rpc.CommitIndex + 1 < int32(len(r.log)) {
		entries = r.log[rpc.CommitIndex + 1:]
	}

	if !r.benOrState.benOrRunning || r.benOrState.benOrStage == Broadcasting {
		// dlog.Println("HUH", r.Id, "sending to", rpc.SenderId, rpc.GetStartIndex() + int32(len(rpc.Entries)), logToString(r.pq.extractList()))
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(), NewRequestedIndex: rpc.LogLength,
			MsgTimestamp: time.Now().UnixNano(),
		}

		if rpc.LogLength != rpc.GetStartIndex() + int32(len(rpc.Entries)) {
			log.Fatal("log length and start index + entries length do not match for replica", r.Id, rpc.LogLength, rpc.GetStartIndex(), int32(len(rpc.Entries)))
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	} else {
		// dlog.Println("HUH2", r.Id, "sending to", rpc.SenderId, int32(r.commitIndex) + 1, logToString(r.pq.extractList()))
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(), NewRequestedIndex: int32(r.commitIndex) + 1,
			MsgTimestamp: time.Now().UnixNano(),
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	}

	// fmt.Println("Replica", r.Id, "type 5", time.Now().UnixMicro() - startTime)
}

func (r *Replica) handleReplicateEntriesReply (rpc *ReplicateEntriesReply) {
	r.handleIncomingTerm(rpc)
	// dlog.Println("Replica", r.Id, "sees term", rpc.GetTerm(), "commitidx", rpc.GetCommitIndex(), "newreqindex", rpc.NewRequestedIndex, "from", rpc.SenderId)
	
	if r.term > int(rpc.Term) || !r.leaderState.isLeader {
		return
	}

	replyTimestamp := time.Unix(0, rpc.MsgTimestamp)
	// timeDiff := time.Now().Sub(replyTimestamp)
	// dlog.Println("Time diff", timeDiff)

	if replyTimestamp.Before(r.leaderState.lastMsgTimestamp[rpc.SenderId]) {
		return
	}

	r.leaderState.lastMsgTimestamp[rpc.SenderId] = replyTimestamp

	// dlog.Println("Leader", r.Id, ": ", r.commitIndex, r.leaderTerm, len(r.log), ", Replica", rpc.SenderId, ": ", rpc.CommitIndex, rpc.LeaderTerm, rpc.LogLength, "PQEntries", "[", logToString(rpc.PQEntries), "]")
	if r.isLogMoreUpToDate(rpc) == LessUpToDate {
		r.updateLogFromRPC(rpc)
	} else {
		for _, v := range(rpc.Entries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}
	
		for _, v := range(rpc.PQEntries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}

		// dlog.Println("Replica", r.Id, "pq values3", r.pq.extractList())

		for !r.pq.isEmpty() {
			entry := r.pq.pop()
			entry.Term = int32(r.term)
			entry.Index = int32(len(r.log))

			if !r.inLog.contains(entry) {
				r.log = append(r.log, entry)
				r.inLog.add(entry)
			}
		}

		r.leaderState.repNextIndex[r.Id] = len(r.log)
		r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1

		r.leaderState.repMatchIndex[rpc.SenderId] = int(rpc.NewRequestedIndex) - 1
		r.leaderState.repNextIndex[rpc.SenderId] = int(rpc.NewRequestedIndex)
	}
}

/************************************** Replicate Entries **********************************************/

func (r *Replica) broadcastReplicateEntries() {
	// r.replicateEntriesCounter = rpcCounter{r.currentTerm, r.replicateEntriesCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			// dlog.Println(r.Id, "HUHUHUHUHUH1", r.leaderState.repNextIndex)
			prevLogIndex := int32(r.leaderState.repNextIndex[i]) - 1

			args := &ReplicateEntries{
				SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
				PrevLogIndex: prevLogIndex, PrevLogServerId: r.log[prevLogIndex].ServerId, PrevLogTimestamp: r.log[prevLogIndex].Timestamp, Entries: r.log[r.leaderState.repNextIndex[i]:],
			}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}