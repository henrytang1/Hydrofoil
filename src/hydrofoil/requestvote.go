package hydrofoil

import (
	"dlog"
	"fmt"
	"math/rand"
	"time"
)

/************************************** Election **********************************************/

func (r *Replica) startElection() {
	dlog.Println("Replica", r.Id, "starting election")

	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	r.term++
	r.candidateState = CandidateState{
		isCandidate: true,
		votesReceived: 1,
	}

	r.votedFor = int(r.Id)

	args := &RequestVote{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
	}

	fmt.Println("Replica", r.Id, "sending to all RequestVote", "AAAA")
	for i := 0; i < r.N; i++ {
		if i != int(r.Id) {
			r.SendMsg(int32(i), r.requestVoteRPC, args)
		}
	}
}


func (r *Replica) handleRequestVote(rpc *RequestVote) {
	dlog.Println("Replica", r.Id, "has term", r.term, "and received a RequestVote RPC from", rpc.SenderId, "with term", rpc.Term)
	r.handleIncomingTerm(rpc)

	entries := make([]Entry, 0)
	if rpc.CommitIndex + 1 < int32(len(r.log)) {
		entries = r.log[rpc.CommitIndex + 1:]
	}

	if r.term > int(rpc.Term) || r.leaderTerm > int(rpc.GetLogTerm()) || (r.leaderTerm == int(rpc.GetLogTerm()) && len(r.log) > int(rpc.LogLength)) || r.votedFor != -1 {
		args := &RequestVoteReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			VoteGranted: False, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		fmt.Println("Replica", r.Id, "sending to", rpc.SenderId, "RequestVoteReply", "BBBB")
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
		return
	} else {
		r.votedFor = int(rpc.SenderId)
		args := &RequestVoteReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			VoteGranted: True, StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		fmt.Println("Replica", r.Id, "sending to", rpc.SenderId, "RequestVoteReply", "CCCC")
		r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)

		timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
		setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
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

	dlog.Println("Replica", r.Id, "term", r.term, "um1")

	if r.isLogMoreUpToDate(rpc) == LessUpToDate {
		r.updateLogFromRPC(rpc)

		// // stop being a candidate
		// r.candidateState = emptyCandidateState
		// return
	} else {
		for _, v := range(rpc.Entries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}

		for _, v := range(rpc.PQEntries) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}

		fmt.Println("Replica", r.Id, "pq values1", logToString(r.pq.extractList()))
	}

	if rpc.VoteGranted == True {
		r.candidateState.votesReceived++
	}

	dlog.Println("Replica", r.Id, "term", r.term, "um2", r.candidateState.votesReceived, rpc.VoteGranted)

	if r.candidateState.votesReceived > r.N/2 {
		// become the leader
		r.becomeLeader()
	}
}

func (r *Replica) becomeLeader() {
	dlog.Println("Replica", r.Id, "becoming the leader")

	r.leaderState = LeaderState{
		isLeader: true,
		repNextIndex: make([]int, r.N),
		repMatchIndex: make([]int, r.N),
		lastMsgTimestamp: make([]time.Time, r.N),
	}

	for i := 0; i < r.N; i++ {
		r.leaderState.lastMsgTimestamp[i] = zeroTime
	}

	r.candidateState = emptyCandidateState
	r.leaderTerm = r.term

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
	}

	r.leaderState.repNextIndex[r.Id] = len(r.log)
	r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1

	clearTimer(r.electionTimer)
	clearTimer(r.benOrStartTimer)
	clearTimer(r.benOrResendTimer)

	r.sendHeartbeat()
}