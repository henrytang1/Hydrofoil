package randomizedpaxos

import (
	"math/rand"
	"time"
)

/************************************** Election **********************************************/

func (r *Replica) startElection() {
	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	r.term++
	r.candidateState = CandidateState{
		isCandidate: true,
		votesReceived: 1,
	}

	r.votedFor = int(r.Id)

	args := &RequestVote{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
	}

	r.SendMsg(r.Id, r.requestVoteRPC, args)
}


func (r *Replica) handleRequestVote (rpc *RequestVote) {
	r.handleIncomingTerm(rpc)

	entries := make([]Entry, 0)
	if rpc.CommitIndex + 1 < int32(len(r.log)) {
		entries = r.log[rpc.CommitIndex + 1:]
	}

	if r.term > int(rpc.Term) || r.isLogMoreUpToDate(rpc) == HigherOrder || r.votedFor != -1 {
		args := &RequestVoteReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			VoteGranted: False, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	} else {
		r.votedFor = int(rpc.SenderId)
		args := &RequestVoteReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			VoteGranted: True, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	}
}


func (r *Replica) handleRequestVoteReply (rpc *RequestVoteReply) {
	// if r.term < int(rpc.Term), then you would no longer be the candidate after r.handleIncomingTerm(rpc)
	r.handleIncomingTerm(rpc)

	if int(rpc.Term) < r.term {
		return
	}

	if !r.candidateState.isCandidate {
		return
	}

	if r.isLogMoreUpToDate(rpc) == LowerOrder {
		r.updateLogFromRPC(rpc)

		// stop being a candidate
		r.candidateState = emptyCandidateState
		return
	} else {
		for _, v := range(rpc.Entries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}
	
		for _, v := range(rpc.PQEntries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}
	}

	if rpc.VoteGranted == True {
		r.candidateState.votesReceived++
	}

	if r.candidateState.votesReceived > r.N/2 {
		r.leaderState = LeaderState{
			isLeader: true,
			repNextIndex: make([]int, r.N),
			repMatchIndex: make([]int, r.N),
			lastRepEntriesTimestamp: 0,
		}
		r.candidateState = emptyCandidateState
		r.logTerm = r.term

		for !r.pq.isEmpty() {
			entry := r.pq.pop()
			entry.Term = int32(r.term)
			entry.Index = int32(len(r.log))

			if !r.inLog.contains(entry) {
				r.log = append(r.log, entry)
				r.inLog.add(entry)
			}
		}

		for i := 0; i < r.N; i++ {
			if i != int(r.Id) {
				r.leaderState.repNextIndex[i] = len(r.log)
				r.leaderState.repMatchIndex[i] = 0
			}

			r.leaderState.repNextIndex[r.Id] = len(r.log)
			r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
		}

		r.sendHeartbeat()
	}
}