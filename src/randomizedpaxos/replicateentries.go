package randomizedpaxos

import (
	"dlog"
	"fmt"
	"log"
	"math/rand"
	"time"
)

func (r *Replica) handleReplicateEntries(rpc *ReplicateEntries) {
	r.handleIncomingTerm(rpc)

	if r.term > int(rpc.Term) || r.isLogMoreUpToDate(rpc) == HigherOrder {
		fmt.Println("strange", r.Id, r.term, int(rpc.Term), r.isLogMoreUpToDate(rpc), "our data", r.commitIndex, r.logTerm, len(r.log), "rpc data", rpc.GetCommitIndex(), rpc.GetLogTerm(), rpc.GetLogLength())

		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			ReplyTimestamp: time.Now().UnixNano(), StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: -1,
		}

		fmt.Println("Replica", r.Id, "sending to", rpc.SenderId, "ReplicateEntriesReply", "AAAA")

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)

		if r.term == int(rpc.Term) {
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		return
	}

	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	timeout = rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
	setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)

	r.lastHeardFromLeader = time.Now()

	if int(rpc.PrevLogIndex) >= len(r.log) || 
		(r.log[rpc.PrevLogIndex].SenderId != rpc.PrevLogSenderId && r.log[rpc.PrevLogIndex].Timestamp != rpc.PrevLogTimestamp) {
		
		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			ReplyTimestamp: time.Now().UnixNano(), StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex) + 1,
		}

		fmt.Println("Replica", r.Id, "sending to", rpc.SenderId, "ReplicateEntriesReply", "BBBB")

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	if r.leaderState.isLeader {
		log.Fatal("Should not be leader when receiving a ReplicateEntries RPC, and have a greater term!")
	}

	oldCommitIndex := r.commitIndex

	dlog.Println("Replica", r.Id, "is accepting entries from", rpc.SenderId, "before log", logToString(r.log))
	potentialEntries := r.replaceExistingLog(rpc, int(rpc.PrevLogIndex) + 1)
	dlog.Println("Replica", r.Id, "is accepting entries from", rpc.SenderId, "after log", logToString(r.log))

	for _, v := range(potentialEntries) {
		if !r.seenBefore(v) { r.pq.push(v) }
	}

	fmt.Println("Replica", r.Id, "pq values4", r.pq.extractList())

	r.commitIndex = int(rpc.CommitIndex)
	r.logTerm = max(r.logTerm, int(rpc.LogTerm))
	if r.commitIndex > oldCommitIndex {
		// if !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
		// 	r.pq.push(r.benOrState.benOrBroadcastEntry)
		// }
		fmt.Println("Replica", r.Id, "pq values2", r.pq.extractList())
		r.benOrState = emptyBenOrState
		timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
		setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)

		clearTimer(r.benOrResendTimer)
	}

	if r.benOrState.benOrStage == Broadcasting && r.commitIndex + 1 < len(r.log) && 
		r.benOrState.benOrBroadcastEntry != r.log[r.commitIndex + 1] {
		r.benOrState.biasedCoin = true
		fmt.Println("Replica", r.Id, "set biased coin to true")
	}

	entries := make([]Entry, 0)
	if rpc.CommitIndex + 1 < int32(len(r.log)) {
		entries = r.log[rpc.CommitIndex + 1:]
	}

	if !r.benOrState.benOrRunning || r.benOrState.benOrStage == Broadcasting {
		fmt.Println("HUH", r.Id, "sending to", rpc.SenderId, rpc.GetStartIndex() + int32(len(rpc.Entries)), logToString(r.pq.extractList()))
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			ReplyTimestamp: time.Now().UnixNano(), StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: True, NewRequestedIndex: rpc.GetStartIndex() + int32(len(rpc.Entries)),
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	} else {
		fmt.Println("HUH2", r.Id, "sending to", rpc.SenderId, int32(r.commitIndex) + 1, logToString(r.pq.extractList()))
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			ReplyTimestamp: time.Now().UnixNano(), StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex) + 1,
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	}
}



// ISSUE: Replica 4 is the leader, and commits up to idx 100. It updates its commit index to 100, but the other replicas don't before replica 4 is disconnected.
// Now, replica 2 is elected leader, and adds a bunch of entries to its log. It updates its commit index to 110, but the other replicas don't before replica 2 is disconnected.
// Replica 4 is reelected leader (since it has a higher commitindex), and then tells the other replicas to change entries 100 to 110. This is a problem!!!!

// Potential solution: when you are elected, you only compare logterm and index. You can be the leader with a lower commitIndex.
// When updating your log, you only replace your log with that from the rpc, if entries up to commitindex are different or if your logterm is lower.


func (r *Replica) handleReplicateEntriesReply (rpc *ReplicateEntriesReply) {
	r.handleIncomingTerm(rpc)
	dlog.Println("Replica", r.Id, "sees term", rpc.GetTerm(), "success", rpc.GetSuccess(), "commitidx", rpc.GetCommitIndex(), "newreqindex", rpc.NewRequestedIndex, "from", rpc.SenderId)
	
	if r.term > int(rpc.Term) || !r.leaderState.isLeader {
		return
	}

	replyTimestamp := time.Unix(0, rpc.ReplyTimestamp)
	if replyTimestamp.Before(r.leaderState.lastReplicaTimestamp[rpc.SenderId]) {
		return
	}

	r.leaderState.lastReplicaTimestamp[rpc.SenderId] = replyTimestamp

	fmt.Println("Leader", r.Id, ": ", r.commitIndex, r.logTerm, len(r.log), ", Replica", rpc.SenderId, ": ", rpc.CommitIndex, rpc.LogTerm, rpc.LogLength, "PQEntries", "[", logToString(rpc.PQEntries), "]")
	if r.isLogMoreUpToDate(rpc) == LowerOrder {
		// todo: leader needs to handle the case where the committed entries match up with its own!
		r.updateLogFromRPC(rpc)
	} else {
		for _, v := range(rpc.Entries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}
	
		for _, v := range(rpc.PQEntries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}

		fmt.Println("Replica", r.Id, "pq values3", r.pq.extractList())

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

		if rpc.Success == True {
			r.leaderState.repMatchIndex[rpc.SenderId] = int(rpc.NewRequestedIndex) - 1
			r.leaderState.repNextIndex[rpc.SenderId] = int(rpc.NewRequestedIndex)
			dlog.Println("BRUH", len(r.log))
		} else {
			if rpc.NewRequestedIndex > 0 { // otherwise, the replica is actually just telling us to update ourselves first
				r.leaderState.repNextIndex[rpc.SenderId] = int(rpc.NewRequestedIndex)
			}
		}
	}
}

/************************************** Replicate Entries **********************************************/

func (r *Replica) broadcastReplicateEntries() {
	// r.replicateEntriesCounter = rpcCounter{r.currentTerm, r.replicateEntriesCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			// dlog.Println(r.Id, "HUHUHUHUHUH1", r.leaderState.repNextIndex)
			prevLogIndex := int32(r.leaderState.repNextIndex[i]) - 1
			dlog.Println("server", r.Id, "has data", logToString(r.log), "and is sending to", i, "with prevLogIndex", prevLogIndex, "and nextIndex", r.leaderState.repNextIndex[i], "and matchIndex", r.leaderState.repMatchIndex[i], "and commit Index", r.commitIndex)
			// dlog.Println("HUHUHUHUHUH2")

			args := &ReplicateEntries{
				SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
				PrevLogIndex: prevLogIndex, PrevLogSenderId: r.log[prevLogIndex].SenderId, PrevLogTimestamp: r.log[prevLogIndex].Timestamp, Entries: r.log[r.leaderState.repNextIndex[i]:],
			}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}