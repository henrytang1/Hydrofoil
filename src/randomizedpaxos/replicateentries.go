package randomizedpaxos

import (
	"math/rand"
	"randomizedpaxosproto"
	"time"
)

func (r *Replica) handleReplicateEntries(rpc *randomizedpaxosproto.ReplicateEntries) {
	replicaEntries := make([]Entry, 0)
	if int(rpc.LeaderPreparedIndex)+1 < len(r.log) {
		replicaEntries = r.log[rpc.LeaderPreparedIndex+1:]
	}

	entryAtLeaderBenOrIndex := benOrUncommittedLogEntry(-1)
	entryAtLeaderBenOrIndex.Term = -1
	if int(rpc.LeaderBenOrIndex) < len(r.log) {
		entryAtLeaderBenOrIndex = r.log[rpc.LeaderBenOrIndex]
	}

	args := &randomizedpaxosproto.ReplicateEntriesReply{
		ReplicaId: r.Id,
		Term: int32(r.currentTerm),
		// Counter: int32(r.infoBroadcastCounter.count),
		ReplicaBenOrIndex: int32(r.benOrIndex),
		ReplicaPreparedIndex: int32(r.preparedIndex),
		ReplicaEntries: replicaEntries,
		// LeaderPreparedIndex: rpc.LeaderPreparedIndex,
		// LeaderBenOrIndex: rpc.LeaderBenOrIndex,
		// EntryAtLeaderBenOrIndex: entryAtLeaderBenOrIndex,
		PrevLogIndex: rpc.PrevLogIndex,
		RequestedIndex: int32(rpc.PrevLogIndex)-1,
		Success: False}

	// reject if the term is out of date
	if (int(rpc.Term) < r.currentTerm) {
		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	// update to follower if received term is newer
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			r.clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		r.isLeader = false
	}

	// a term of -1 means that the entry is committed using Ben-Or+
	// we assume that r.entries[rpc.PrevLogIndex] is not currently running Ben-Or+
	// reject if the previous term doesn't match
	// if entry at previous term is running Ben-Or+, then we must reject and ask the leader to send over another entry
	if (r.entries[rpc.PrevLogIndex].Term != rpc.PrevLogTerm || r.entries[rpc.PrevLogIndex].BenOrActive == True) {
		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	// tell leader to update itself if it's out of date
	// alternatively, this is an out of date message
	if r.preparedIndex > int(rpc.LeaderPreparedIndex) {
		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	// TODO: get rid of this later since I think it's uneccessary logic
	// if this is a heartbeat, then we can just return
	if (len(rpc.Entries) == 0) {
		// this is a heartbeat
		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	// args = &randomizedpaxosproto.ReplicateEntriesReply{}
	// if (rpc.LeaderBenOrIndex <= int32(r.benOrIndex)) {
	// 	args.EntryAtLeaderBenOrIndex = r.log[rpc.LeaderBenOrIndex]
	// }
	currentCommitPoint := r.benOrIndex-1
	newCommitPoint := max(currentCommitPoint, int(rpc.LeaderBenOrIndex)-1)
	newPreparedPoint := int(rpc.LeaderPreparedIndex)
	firstEntryIndex := int(rpc.PrevLogIndex)+1

	logLength := len(r.log)

	benOrIndexChanged := false
	i := currentCommitPoint+1

	for ; i < logLength; i++ {
		if (i < firstEntryIndex) {
			continue
		}
		if r.log[i].Term != rpc.Entries[i-firstEntryIndex].Term {
			if (r.log[i].BenOrActive == True) {
				r.log[i] = rpc.Entries[i-firstEntryIndex]
				if rpc.Entries[i-firstEntryIndex].BenOrActive == False && i <= newPreparedPoint {
					benOrIndexChanged = true
					continue
				}

				if r.benOrState.benOrStatus == Stopped {
					// don't need to do anything else
				} else if r.benOrState.benOrStatus == Broadcasting {
					r.benOrState.biasedCoin = true
				} else { // r.benOrStatus == BenOrRunning
					// can't do anything here
				}
			} else if rpc.Entries[i-firstEntryIndex].BenOrActive == True {
				// use current entry instead
				continue
			} else {
				r.log = append(r.log[:i], rpc.Entries[i-firstEntryIndex:]...)
				break
			}
		}
	}

	if i == logLength {
		r.log = append(r.log, rpc.Entries[i-firstEntryIndex:]...)
	}

	if benOrIndexChanged {
		r.benOrIndex = newCommitPoint+1
		r.benOrState.benOrStatus = Stopped
		r.benOrState.biasedCoin = false
		if (r.benOrIndex < len(r.log)) {
			r.log[r.benOrIndex].BenOrActive = True
		} else {
			r.log[r.benOrIndex] = benOrUncommittedLogEntry(len(r.log))
		}
	}

	for i := currentCommitPoint+1; i <= len(r.log); i++ {
		r.pq.remove(r.log[i])
	}

	// extract values from priority queue and append them
	replicaEntries = append(replicaEntries, r.pq.extractList()...)

	args = &randomizedpaxosproto.ReplicateEntriesReply{
		ReplicaId: r.Id,
		Term: int32(r.currentTerm),
		// Counter: int32(r.infoBroadcastCounter.count),
		ReplicaBenOrIndex: int32(r.benOrIndex),
		ReplicaPreparedIndex: int32(r.preparedIndex),
		ReplicaEntries: replicaEntries,
		// LeaderPreparedIndex: rpc.LeaderPreparedIndex,
		// LeaderBenOrIndex: rpc.LeaderBenOrIndex,
		// EntryAtLeaderBenOrIndex: entryAtLeaderBenOrIndex,
		PrevLogIndex: rpc.PrevLogIndex,
		RequestedIndex: int32(r.benOrIndex),
		Success: True}

	r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	return
}

func (r *Replica) handleReplicateEntriesReply (rpc *randomizedpaxosproto.ReplicateEntriesReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			r.clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		r.isLeader = false
		r.votesReceived = 0
	}

	if (r.isLeader || int(rpc.Term) < r.currentTerm) {
		// ignore these entries
		return
	}

	// currentCommitPoint := r.benOrIndex-1
	// currentPreparedPoint := r.preparedIndex
	// newCommitPoint := max(currentCommitPoint, int(rpc.ReplicaBenOrIndex)-1)
	newPreparedPoint := max(r.preparedIndex, int(rpc.ReplicaPreparedIndex))
	firstEntryIndex := int(rpc.PrevLogIndex) + 1

	// update log with new entries if they exist
	for i := r.benOrIndex; i < int(rpc.ReplicaPreparedIndex); i++ {
		idx := i - firstEntryIndex
		if i < len(r.log) {
			if (r.log[i].Term != rpc.ReplicaEntries[idx].Term) {
				if r.log[i].BenOrActive == True {
					if rpc.ReplicaEntries[idx].BenOrActive == False && i <= newPreparedPoint {
						// r.requestVoteEntries[i] = rpc.ReplicaEntries[idx]
						r.log[i] = rpc.ReplicaEntries[idx]
						continue
					}
					// else, use current entry
				} else if rpc.ReplicaEntries[idx].BenOrActive == True {
					// use current entry instead
					continue
				} else { // neither requestVoteEntries[i] or rpc.ReplicaEntries[idx] is benOrActive
					// r.requestVoteEntries[idx] = rpc.ReplicaEntries[idx]
					r.log = append(r.log[:i], rpc.ReplicaEntries[idx:newPreparedPoint+1]...)
					break
				}
			}
		} else {
			r.log = append(r.log, rpc.ReplicaEntries[idx:newPreparedPoint+1]...)
			// r.requestVoteEntries = append(r.requestVoteEntries, rpc.ReplicaEntries[idx])
		}
	}

	r.commitIndex[rpc.ReplicaId] = max(r.commitIndex[rpc.ReplicaId], int(rpc.ReplicaBenOrIndex))-1
	r.matchIndex[rpc.ReplicaId] = max(r.matchIndex[rpc.ReplicaId], int(rpc.ReplicaPreparedIndex))

	if rpc.Success == True {
		r.nextIndex[rpc.ReplicaId] = r.commitIndex[rpc.ReplicaId] + 1
	} else {
		r.nextIndex[rpc.ReplicaId] = int(rpc.RequestedIndex)
		// TODO: can optimize this out
	}
}

/************************************** Replicate Entries **********************************************/

func (r *Replica) bcastReplicateEntries() {
	// r.replicateEntriesCounter = rpcCounter{r.currentTerm, r.replicateEntriesCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			args := &randomizedpaxosproto.ReplicateEntries{
				SenderId: r.Id, Term: int32(r.currentTerm),
				PrevLogIndex: int32(r.nextIndex[i]-1), PrevLogTerm: int32(r.log[r.nextIndex[i]-1].Term),
				Entries: r.log[r.nextIndex[i]:], LeaderBenOrIndex: int32(r.benOrIndex), LeaderPreparedIndex: int32(r.preparedIndex)}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}