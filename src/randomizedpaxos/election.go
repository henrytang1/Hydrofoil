package randomizedpaxos

import (
	"math/rand"
	"time"
)

/************************************** Election **********************************************/

func (r *Replica) startElection() {
	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	pqEntries := r.pq.extractList()
	for _, v := range(pqEntries) {
		r.pq.push(v)
	}

	r.term++
	r.candidateState = CandidateState{
		votesReceived: 1,
		votedFor: int(r.Id),
		requestVoteEntries: append(r.log[r.preparedIndex+1:], pqEntries...),
	}

	args := &RequestVote{
		SenderId: r.Id,
		Term: int32(r.term),
		BenOrIndex: int32(r.benOrIndex),
	}

	r.SendMsg(r.Id, r.requestVoteRPC, args)
}


func (r *Replica) handleRequestVote (rpc *RequestVote) {
	replicaEntries := make([]Entry, 0)
	if int(rpc.BenOrIndex) < len(r.log) {
		replicaEntries = r.log[rpc.BenOrIndex:]
	}
	
	args := &RequestVoteReply{
		SenderId: r.Id,
		Term: int32(r.term),
		// Counter: int32(r.infoBroadcastCounter.count),
		VoteGranted: False,
		BenOrIndex: int32(r.benOrIndex),
		PreparedIndex: int32(r.preparedIndex),
		Entries: replicaEntries,
		CandidateBenOrIndex: rpc.BenOrIndex,
	}

	if int(rpc.Term) < r.term || (int(rpc.Term) == r.term && r.candidateState.votedFor != -1) {
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}
	
	if int(rpc.Term) > r.term {
		r.term = int(rpc.Term)

		if (r.leaderState.isLeader) {
			clearTimer(r.heartbeatTimer)
		}

		r.leaderState = LeaderState{
			isLeader: false,
			repNextIndex: make([]int, r.N),
			repCommitIndex: make([]int, r.N),
			repPreparedIndex: make([]int, r.N),
		}

		timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
		setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
	}

	if rpc.Term < int32(r.term) {
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}

	if rpc.Term == int32(r.term) && r.candidateState.votedFor != -1 {
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}

	if int(rpc.Term) > r.term {
		r.term = int(rpc.Term)
		r.leaderState.isLeader = false
	}

	args.VoteGranted = True
	r.candidateState.votedFor = int(rpc.SenderId)
	r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
	return
}


func (r *Replica) handleRequestVoteReply (rpc *RequestVoteReply) {
	if int(rpc.Term) < r.term {
		// ignore these entries
		return
	}
	if int(rpc.Term) > r.term { // you are no longer a candidate!
		r.term = int(rpc.Term)

		if (r.leaderState.isLeader) {
			clearTimer(r.heartbeatTimer)
		}

		r.leaderState = LeaderState{
			isLeader: false,
			repNextIndex: make([]int, r.N),
			repCommitIndex: make([]int, r.N),
			repPreparedIndex: make([]int, r.N),
		}

		r.candidateState = CandidateState{
			votesReceived: 0,
			votedFor: -1,
			requestVoteEntries: make([]Entry, 0),
		}
		return
	}

	// if it's the same term, then this is a vote for the current request votes being sent out
	currentCommitPoint := r.benOrIndex-1
	firstEntryIndex := int(rpc.StartIndex)
	potentialEntries := make([]Entry, 0)

	for i := currentCommitPoint+1; i <= int(rpc.PreparedIndex); i++ {
		if i >= len(r.log) {
			r.log = append(r.log, benOrUncommittedLogEntry())
		}

		if (i != int(rpc.BenOrIndex) && 
			((i != r.benOrIndex && r.log[i].Term < rpc.Entries[i-firstEntryIndex].Term) || i == r.benOrIndex)) || 
			(i == r.benOrIndex && i == int(rpc.BenOrIndex) && r.log[i].Term < rpc.Entries[i-firstEntryIndex].Term) {
				r.inLog.remove(UniqueCommand{senderId: r.log[i].SenderId, time: r.log[i].Timestamp})
				potentialEntries = append(potentialEntries, r.log[i])
				r.log[i] = rpc.Entries[i-firstEntryIndex]
				r.inLog.add(UniqueCommand{senderId: r.log[i].SenderId, time: r.log[i].Timestamp})
		}
	}

	for i := int(rpc.PreparedIndex)+1; i < len(rpc.Entries) + firstEntryIndex; i++ {
		potentialEntries = append(potentialEntries, rpc.Entries[i-firstEntryIndex])
	}

	for _, v := range(potentialEntries) {
		if !r.seenBefore(v) {
			r.pq.push(v)
		}
	}

	for _, v := range(rpc.PQEntries) {
		if !r.seenBefore(v) {
			r.pq.push(v)
		}
	}

	if r.leaderState.isLeader {
		for !r.pq.isEmpty() {
			entry := r.pq.pop()
			if !r.inLog.contains(UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}) {
				r.addNewEntry(entry)
				r.inLog.add(UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp})
			}
		}
	}

	r.preparedIndex = max(r.preparedIndex, int(rpc.PreparedIndex))
	if r.benOrIndex != int(rpc.BenOrIndex) && r.benOrIndex <= int(rpc.PreparedIndex) {
		r.benOrState = BenOrState{
			benOrStage: Stopped,
			benOrIteration: 0,
			benOrPhase: 0,
			benOrVote: -1,
			benOrMajRequest: benOrUncommittedLogEntry(),
			benOrBroadcastRequest: benOrUncommittedLogEntry(),
			benOrRepliesReceived: 0,
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		}
		r.benOrIndex = r.preparedIndex + 1
	}






	
	currentCommitPoint := r.benOrIndex-1
	currentPreparedPoint := r.preparedIndex
	newCommitPoint := max(currentCommitPoint, int(rpc.BenOrIndex)-1)
	newPreparedPoint := max(r.preparedIndex, int(rpc.PreparedIndex))
	firstEntryIndex := int(rpc.CandidateBenOrIndex) + 1

	// add new committed entries from returning rpc
	for i := r.benOrIndex; i < int(rpc.PreparedIndex); i++ {
		idx := i - firstEntryIndex
		if i < len(r.log) {
			if (r.log[i].Term != rpc.Entries[idx].Term) {
				if r.log[i].BenOrActive == True {
					if rpc.Entries[idx].BenOrActive == False && i <= newPreparedPoint {
						// r.requestVoteEntries[i] = rpc.ReplicaEntries[idx]
						r.log[i] = rpc.Entries[idx]
						continue
					}
					// else, use current entry
				} else if rpc.Entries[idx].BenOrActive == True {
					// use current entry instead
					continue
				} else { // neither requestVoteEntries[i] or rpc.ReplicaEntries[idx] is benOrActive
					// r.requestVoteEntries[idx] = rpc.ReplicaEntries[idx]
					r.log = append(r.log[:i], rpc.Entries[idx:newPreparedPoint+1]...)
					break
				}
			}
		} else {
			r.log = append(r.log, rpc.Entries[idx:newPreparedPoint+1]...)
			// r.requestVoteEntries = append(r.requestVoteEntries, rpc.ReplicaEntries[idx])
		}
	}

	r.benOrIndex = newCommitPoint+1
	r.preparedIndex = newPreparedPoint

	// remove elements elements that have for sure been committed from requestVoteEntries
	r.candidateState.requestVoteEntries = r.candidateState.requestVoteEntries[r.preparedIndex-currentPreparedPoint:]

	if rpc.VoteGranted == True {
		r.candidateState.votesReceived++

		start := newPreparedPoint + 1
		if r.benOrRunning() && r.log[start].BenOrActive == True {
			start++
		}

		for i := newPreparedPoint + 1; i < firstEntryIndex + len(rpc.Entries); i++ {
			idxRPC := i - firstEntryIndex
			idxRVEntries := i - newPreparedPoint - 1
			if idxRVEntries < len(r.candidateState.requestVoteEntries) {
				if (r.candidateState.requestVoteEntries[idxRVEntries].Term < rpc.Entries[idxRPC].Term) {
					r.candidateState.requestVoteEntries[idxRVEntries] = rpc.Entries[idxRPC]
				}
			} else {
				r.candidateState.requestVoteEntries = append(r.candidateState.requestVoteEntries, rpc.Entries[idxRPC])
				break
			}
		}
	}

	// TODO: change term to new term
	// update ben or entry term

	if (r.candidateState.votesReceived > r.N/2) {
		// become the leader
		r.leaderState = LeaderState{
			isLeader: true,
			repNextIndex: make([]int, r.N),
			repCommitIndex: make([]int, r.N),
			repPreparedIndex: make([]int, r.N),
			repLastMsgTimestamp: make([]int64, r.N),
		}

		// copy over values from requestVoteEntries to log
		r.log = append(r.log[:r.preparedIndex+1], r.candidateState.requestVoteEntries...)
		r.candidateState.votesReceived = 0
		r.candidateState.requestVoteEntries = make([]Entry, 0)

		for i := 0; i < r.N ; i++ {
			r.leaderState.repNextIndex[i] = r.benOrIndex
			r.leaderState.repPreparedIndex[i] = 0
			r.leaderState.repCommitIndex[i] = 0
			r.leaderState.repLastMsgTimestamp[i] = 0
		}

		timeout := rand.Intn(r.heartbeatTimeout/2) + r.heartbeatTimeout/2
		resetTimer(r.heartbeatTimer, time.Duration(timeout)*time.Millisecond)
		// send out replicate entries rpcs
		r.broadcastReplicateEntries()
	}
}