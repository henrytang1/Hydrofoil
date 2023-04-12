package randomizedpaxos

import (
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
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex, Entries: entries, PQEntries: r.pq.extractList(),
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
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex),
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	}

	if r.leaderState.isLeader {
		log.Fatal("Should not be leader when receiving a ReplicateEntries RPC, and have a greater term!")
	}

	oldCommitIndex := r.commitIndex

	potentialEntries := r.replaceExistingLog(rpc, int(rpc.PrevLogIndex) + 1)

	for _, v := range(potentialEntries) {
		if !r.seenBefore(v) { r.pq.push(v) }
	}

	r.commitIndex = int(rpc.CommitIndex)
	r.logTerm = max(r.logTerm, int(rpc.LogTerm))
	if r.commitIndex > oldCommitIndex {
		if !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
			r.pq.push(r.benOrState.benOrBroadcastEntry)
		}
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
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex, Entries: entries, PQEntries: r.pq.extractList(),
			Success: True, NewRequestedIndex: rpc.GetStartIndex() + int32(len(rpc.Entries)),
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	} else {
		args := &ReplicateEntriesReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			LeaderTimestamp: rpc.LeaderTimestamp, StartIndex: rpc.CommitIndex, Entries: entries, PQEntries: r.pq.extractList(),
			Success: False, NewRequestedIndex: int32(r.commitIndex) + 1,
		}

		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	}
}

func (r *Replica) handleReplicateEntriesReply (rpc *ReplicateEntriesReply) {
	r.handleIncomingTerm(rpc)
	fmt.Println("Replica", r.Id, "sees", rpc.GetTerm(), rpc.GetSuccess(), rpc.GetCommitIndex(), rpc.NewRequestedIndex)
	if r.term > int(rpc.Term) || r.leaderState.lastRepEntriesTimestamp != rpc.LeaderTimestamp {
		return
	}

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
				fmt.Println("BRUH", len(r.log))
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
			// fmt.Println(r.Id, "HUHUHUHUHUH1", r.leaderState.repNextIndex)
			prevLogIndex := int32(r.leaderState.repNextIndex[i]) - 1
			// fmt.Println("HUHUHUHUHUH2")

			args := &ReplicateEntries{
				SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
				LeaderTimestamp: r.leaderState.lastRepEntriesTimestamp, PrevLogIndex: prevLogIndex,
				PrevLogSenderId: r.log[prevLogIndex].SenderId, PrevLogTimestamp: r.log[prevLogIndex].Timestamp, Entries: r.log[r.leaderState.repNextIndex[i]:],
			}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}