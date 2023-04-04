package randomizedpaxos

import (
	"math/rand"
	"time"
)

/************************************** Election **********************************************/

func (r *Replica) startElection() {
	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	r.currentTerm++
	r.votedFor = int(r.Id)
	r.votesReceived = 1 // itself
	r.requestVoteEntries = append(r.log[r.preparedIndex+1:], r.pq.extractList()...)

	args := &RequestVote{
		SenderId: r.Id,
		Term: int32(r.currentTerm),
		CandidateBenOrIndex: int32(r.benOrIndex),
	}

	r.SendMsg(r.Id, r.requestVoteRPC, args)
}


func (r *Replica) handleRequestVote (rpc *RequestVote) {
	replicaEntries := make([]Entry, 0)
	if int(rpc.CandidateBenOrIndex)+1 < len(r.log) {
		replicaEntries = r.log[rpc.CandidateBenOrIndex:]
	}

	entryAtCandidateBenOrIndex := benOrUncommittedLogEntry(-1)
	entryAtCandidateBenOrIndex.Term = -1
	if int(rpc.CandidateBenOrIndex) < len(r.log) {
		entryAtCandidateBenOrIndex = r.log[rpc.CandidateBenOrIndex]
	}
	
	args := &RequestVoteReply{
		SenderId: r.Id,
		Term: int32(r.currentTerm),
		// Counter: int32(r.infoBroadcastCounter.count),
		VoteGranted: False,
		ReplicaBenOrIndex: int32(r.benOrIndex),
		ReplicaPreparedIndex: int32(r.preparedIndex),
		ReplicaEntries: replicaEntries,
		CandidateBenOrIndex: rpc.CandidateBenOrIndex,
		EntryAtCandidateBenOrIndex: entryAtCandidateBenOrIndex}

	if rpc.Term < int32(r.currentTerm) {
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}

	if rpc.Term == int32(r.currentTerm) && r.votedFor != -1 {
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}

	if int(rpc.Term) > r.currentTerm {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
	}

	args.VoteGranted = True
	r.votedFor = int(rpc.SenderId)
	r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
	return
}


func (r *Replica) handleRequestVoteReply (rpc *RequestVoteReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		
		if (r.isLeader) {
			clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		r.isLeader = false
		r.votesReceived = 0
		r.requestVoteEntries = make([]Entry, 0)
		return
	}
	
	if (r.isLeader || int(rpc.Term) < r.currentTerm) {
		// ignore these entries
		return
	}

	currentCommitPoint := r.benOrIndex-1
	currentPreparedPoint := r.preparedIndex
	newCommitPoint := max(currentCommitPoint, int(rpc.ReplicaBenOrIndex)-1)
	newPreparedPoint := max(r.preparedIndex, int(rpc.ReplicaPreparedIndex))
	firstEntryIndex := int(rpc.CandidateBenOrIndex) + 1

	// add new committed entries from returning rpc
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

	r.benOrIndex = newCommitPoint+1
	r.preparedIndex = newPreparedPoint

	// remove elements elements that have for sure been committed from requestVoteEntries
	r.requestVoteEntries = r.requestVoteEntries[r.preparedIndex-currentPreparedPoint:]

	if rpc.VoteGranted == True {
		r.votesReceived++

		start := newPreparedPoint + 1
		if r.benOrRunning() && r.log[start].BenOrActive == True {
			start++
		}

		for i := newPreparedPoint + 1; i < firstEntryIndex + len(rpc.ReplicaEntries); i++ {
			idxRPC := i - firstEntryIndex
			idxRVEntries := i - newPreparedPoint - 1
			if idxRVEntries < len(r.requestVoteEntries) {
				if (r.requestVoteEntries[idxRVEntries].Term < rpc.ReplicaEntries[idxRPC].Term) {
					r.requestVoteEntries[idxRVEntries] = rpc.ReplicaEntries[idxRPC]
				}
			} else {
				r.requestVoteEntries = append(r.requestVoteEntries, rpc.ReplicaEntries[idxRPC])
				break
			}
		}
	}

	if (r.votesReceived > r.N/2) {
		// become the leader
		r.isLeader = true
		r.votesReceived = 0

		// copy over values from requestVoteEntries to log
		r.log = append(r.log[:r.preparedIndex+1], r.requestVoteEntries...)
		r.requestVoteEntries = make([]Entry, 0)

		for i := 0; i < r.N ; i++ {
			r.nextIndex[i] = len(r.log)
			r.matchIndex[i] = 0
			r.commitIndex[i] = 0
		}

		timeout := rand.Intn(r.heartbeatTimeout/2) + r.heartbeatTimeout/2
		resetTimer(r.heartbeatTimer, time.Duration(timeout)*time.Millisecond)
		// send out replicate entries rpcs
		r.bcastReplicateEntries()
	}
}