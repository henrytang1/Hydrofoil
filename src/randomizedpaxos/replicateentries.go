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
		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: -1,
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)

		if r.term == int(rpc.Term) {
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		return
	}

	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	if int(rpc.PrevLogIndex) > len(r.log) || 
		(r.log[rpc.PrevLogIndex].SenderId != rpc.PrevLogSenderId && r.log[rpc.PrevLogIndex].Timestamp != rpc.PrevLogTimestamp) {
		
		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		// NewRequestedIndex is not used because the leader will either switch to a follower (because of the term) or ignore this reply (because it has an older LeaderTimestamp)
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex) + 1,
		}

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
	}

	if r.benOrState.benOrStage == Broadcasting && r.commitIndex + 1 < len(r.log) && 
		r.benOrState.benOrBroadcastEntry != r.log[r.commitIndex + 1] {
		r.benOrState.biasedCoin = true
	}

	entries := make([]Entry, 0)
	if rpc.CommitIndex + 1 < int32(len(r.log)) {
		entries = r.log[rpc.CommitIndex + 1:]
	}

	if !r.benOrState.benOrRunning || r.benOrState.benOrStage == Broadcasting {
		fmt.Println("HUH", r.Id, logToString(r.pq.extractList()))
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: True, NewRequestedIndex: rpc.GetStartIndex() + int32(len(rpc.Entries)),
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	} else {
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex) + 1,
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	}
}

func (r *Replica) handleReplicateEntriesReply (rpc *ReplicateEntriesReply) {
	r.handleIncomingTerm(rpc)
	dlog.Println("Replica", r.Id, "sees term", rpc.GetTerm(), "success", rpc.GetSuccess(), "commitidx", rpc.GetCommitIndex(), "newreqindex", rpc.NewRequestedIndex, "from", rpc.SenderId)
	if r.term > int(rpc.Term) || r.leaderState.lastRepEntriesTimestamp != rpc.LeaderTimestamp {
		return
	}

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

		if r.leaderState.isLeader {
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
				r.leaderState.repNextIndex[rpc.SenderId] = int(rpc.NewRequestedIndex)
			}
		}
	}
}

/************************************** Replicate Entries **********************************************/

func (r *Replica) broadcastReplicateEntries() {
	// r.replicateEntriesCounter = rpcCounter{r.currentTerm, r.replicateEntriesCounter.count+1}
	r.leaderState.lastRepEntriesTimestamp = time.Now().UnixNano()
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			// dlog.Println(r.Id, "HUHUHUHUHUH1", r.leaderState.repNextIndex)
			prevLogIndex := int32(r.leaderState.repNextIndex[i]) - 1
			dlog.Println("server", r.Id, "has data", logToString(r.log), "and is sending to", i, "with prevLogIndex", prevLogIndex, "and nextIndex", r.leaderState.repNextIndex[i], "and matchIndex", r.leaderState.repMatchIndex[i], "and commit Index", r.commitIndex)
			// dlog.Println("HUHUHUHUHUH2")

			args := &ReplicateEntries{
				SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
				LeaderTimestamp: r.leaderState.lastRepEntriesTimestamp, PrevLogIndex: prevLogIndex,
				PrevLogSenderId: r.log[prevLogIndex].SenderId, PrevLogTimestamp: r.log[prevLogIndex].Timestamp, Entries: r.log[r.leaderState.repNextIndex[i]:],
			}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}