package randomizedpaxos

import (
	"math/rand"
	"sort"
	"time"
)

type BenOrState struct {
	benOrStage				uint8
	benOrIteration				int
	benOrPhase				int

	benOrVote				int
	benOrMajRequest				Entry

	// benOrStage				uint8
	benOrBroadcastRequest 			Entry
	// benOrWaitingForIndex			int
	benOrRepliesReceived			int
	benOrBroadcastMessages			[]BenOrBroadcastMsg
	benOrConsensusMessages			[]BenOrConsensusMsg
	biasedCoin				bool
}

/************************************** Ben Or Braodcast **********************************************/
func (r *Replica) startBenOrPlus () {
	r.startBenOrIteration(true)
}


func (r *Replica) startBenOrIteration(startFromBeginning bool) {
	if startFromBeginning {
		r.benOrState = BenOrState{
			benOrStage: Broadcasting,
			benOrIteration: 0,
			benOrPhase: 0,
			benOrVote: -1,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		}
	}
	
	var request Entry
	if (r.log[r.benOrIndex].Term != -1) {
		request = r.log[r.benOrIndex]
	} else if (!r.pq.isEmpty()) {
		request = r.pq.pop()
	} else {
		return
	}

	args := &BenOrBroadcast{SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), 
		Iteration: int32(r.benOrState.benOrIteration), ClientReq: request}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
		}
	}

	// always receive your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, args)
	r.benOrState.benOrBroadcastRequest = request
}


func (r *Replica) handleBenOrBroadcast (rpc BenOrBroadcastMsg) {
	if int(rpc.GetTerm()) > r.currentTerm {
		r.currentTerm = int(rpc.GetTerm())

		if (r.isLeader) {
			clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		r.isLeader = false
	}

	cmd := UniqueCommand{senderId: rpc.GetClientReq().SenderId, time: rpc.GetClientReq().Timestamp}
	if !r.seenEntries.contains(cmd) {
		r.addNewEntry(rpc.GetClientReq())
	}

	if int(rpc.GetIndex()) < r.benOrIndex {
		args := &GetCommittedDataReply{
			SenderId: r.Id, Term: int32(r.currentTerm), StartIndex: rpc.GetIndex(), EndIndex: int32(r.benOrIndex), Entries: r.log[rpc.GetIndex():r.benOrIndex+1],
		}
		r.SendMsg(rpc.GetSenderId(), r.getCommittedDataReplyRPC, args)
		return
	}

	if r.benOrIndex < int(rpc.GetIndex()) {
		args := &GetCommittedData{
			SenderId: r.Id, Term: int32(r.currentTerm), StartIndex: int32(r.benOrIndex), EndIndex: rpc.GetIndex(),
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.getCommittedDataRPC, args)
			}
		}

		// r.benOrStatus = WaitingToBeUpdated	
		return 
	}

	if r.benOrState.benOrIteration < int(rpc.GetIteration()) {
		if !r.isLeader {
			r.pq.push(r.benOrState.benOrBroadcastRequest) // add own value back
		}

		r.benOrState = BenOrState{
			benOrStage: Broadcasting,
			benOrIteration: int(rpc.GetIteration()),
			benOrPhase: 0,
			benOrVote: -1,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 1,
			benOrBroadcastMessages: []BenOrBroadcastMsg{rpc},
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		}

		r.startBenOrIteration(false)
		return
	}

	if r.benOrState.benOrIteration > int(rpc.GetIteration()) {
		if t, ok := rpc.(*BenOrBroadcast); ok {
			args := &BenOrBroadcastReply{
				SenderId: r.Id, Term: int32(r.currentTerm), Index: t.Index, Iteration: int32(r.benOrIndex), ClientReq: r.benOrState.benOrBroadcastRequest,
			}
			r.SendMsg(rpc.GetSenderId(), r.benOrBroadcastReplyRPC, args)
		}
		return
	}

	// r.benOrIteration == int(rpc.Iteration) and r.benOrIndex == int(rpc.Index)
	notReceivedYet := true
	for _, msg := range r.benOrState.benOrBroadcastMessages {
		if msg.GetSenderId() == rpc.GetSenderId() {
			notReceivedYet = false
			break
		}
	}

	if r.benOrState.benOrStage == Broadcasting && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, rpc)

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.startBenOrConsensusStage1(true)
		}
	}

	if _, ok := rpc.(*BenOrBroadcast); ok {
		args := &BenOrBroadcastReply{
			SenderId: r.Id, Term: int32(r.currentTerm), Index: rpc.GetIndex(), Iteration: int32(r.benOrIndex), ClientReq: r.benOrState.benOrBroadcastRequest,
		}
		r.SendMsg(rpc.GetSenderId(), r.benOrBroadcastReplyRPC, args)
	}
}

/************************************** Ben Or Consensus **********************************************/

// only called if actually need to run benOr at this index
func (r *Replica) startBenOrConsensusStage1(initialize bool) {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	var foundMajRequest int32;
	var majRequest Entry;

	if initialize {
		r.benOrState.benOrPhase++
		r.benOrState.benOrStage = StageOne
		r.benOrState.benOrRepliesReceived = 0

		var n int = len(r.benOrState.benOrBroadcastMessages)

		msgs := r.benOrState.benOrBroadcastMessages

		sort.Slice(msgs, func(i, j int) bool {
			if msgs[i].GetIndex() != msgs[j].GetIndex() {
				return msgs[i].GetIndex() < msgs[j].GetIndex()
			}
			if msgs[i].GetClientReq().Timestamp != msgs[j].GetClientReq().Timestamp {
				return msgs[i].GetClientReq().Timestamp < msgs[j].GetClientReq().Timestamp
			}
			return msgs[i].GetClientReq().SenderId < msgs[j].GetClientReq().SenderId
		})

		foundMajRequest = int32(Vote0)
		majRequest = benOrUncommittedLogEntry(-1)
		curRequest := benOrUncommittedLogEntry(-1)

		counter := 0
		timestamp := int64(-1)
		index := int32(-1)
		senderId := int32(-1)

		for i := 0; i < n; i++ {
			if msgs[i].GetIndex() == index && msgs[i].GetClientReq().Timestamp == timestamp && 
				msgs[i].GetClientReq().SenderId == senderId {
				counter++
				if msgs[i].GetClientReq().Term > curRequest.Term {
					curRequest = msgs[i].GetClientReq()
				}
			} else {
				counter = 1
				index = msgs[i].GetIndex()
				timestamp = msgs[i].GetClientReq().Timestamp
				senderId = msgs[i].GetClientReq().SenderId
				curRequest = msgs[i].GetClientReq()
			}
			
			if counter >= r.N/2 {
				foundMajRequest = int32(Vote1)
				majRequest = curRequest
				break
			}
		}

		r.benOrState.benOrVote = int(foundMajRequest)
		r.benOrState.benOrMajRequest = majRequest
	} else {
		foundMajRequest = int32(r.benOrState.benOrVote)
		majRequest = r.benOrState.benOrMajRequest
	}

	leaderEntry := benOrUncommittedLogEntry(-1)
	if r.log[r.benOrIndex].FromLeader == True {
		leaderEntry = r.log[r.benOrIndex]
	}

	args := &BenOrConsensus{
		SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), Iteration: int32(r.benOrState.benOrIteration), 
		Phase: int32(r.benOrState.benOrPhase), Vote: foundMajRequest, MajRequest: majRequest, LeaderRequest: leaderEntry, Stage: 1} // last entry represents the phase
	// r.SendMsg(r.Id, r.benOrBroadcastRPC, args)
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}

	// received your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, args)
}


func (r *Replica) startBenOrConsensusStage2(initialize bool) {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	var vote int32
	var majRequest Entry;

	if initialize {
		r.benOrState.benOrPhase++
		r.benOrState.benOrStage = StageOne
		r.benOrState.benOrRepliesReceived = 0

		// var n int = len(r.benOrState.benOrConsensusMessages)

		msgs := r.benOrState.benOrConsensusMessages
	
		voteCount := [2]int{0, 0}
		for _, msg := range msgs {
			voteCount[msg.GetVote()]++
		}

		var vote int32;

		if voteCount[0] > r.N/2 {
			vote = int32(Vote0)
		} else if voteCount[1] > r.N/2 {
			vote = int32(Vote1)
		} else {
			vote = int32(VoteQuestionMark)
		}

		r.benOrState.benOrVote = int(vote)
		// r.benOrState.benOrMajRequest = majRequest
	}

	vote = int32(r.benOrState.benOrVote)
	majRequest = r.benOrState.benOrMajRequest

	leaderEntry := benOrUncommittedLogEntry(-1)
	if r.log[r.benOrIndex].FromLeader == True {
		leaderEntry = r.log[r.benOrIndex]
	}

	args := &BenOrConsensus{
		SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), Iteration: int32(r.benOrState.benOrIteration),
		Phase: int32(r.benOrState.benOrPhase), Vote: vote, MajRequest: majRequest, LeaderRequest: leaderEntry, Stage: 2} // last entry represents the phase

	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}

	// received your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, args)
}


func (r *Replica) handleBenOrConsensus (rpc BenOrConsensusMsg) {
	if (int(rpc.GetTerm()) > r.currentTerm) { // update term but don't return
		r.currentTerm = int(rpc.GetTerm())

		if (r.isLeader) {
			clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		r.isLeader = false
	}

	if int(rpc.GetIndex()) < r.benOrIndex {
		args := &GetCommittedDataReply{
			SenderId: r.Id, Term: int32(r.currentTerm), StartIndex: rpc.GetIndex(), EndIndex: int32(r.benOrIndex), 
			Entries: r.log[rpc.GetIndex():r.benOrIndex+1],
		}
		r.SendMsg(rpc.GetSenderId(), r.getCommittedDataReplyRPC, args)
		return
	}

	if r.benOrIndex < int(rpc.GetIndex()) {
		args := &GetCommittedData{
			SenderId: r.Id, Term: int32(r.currentTerm), StartIndex: int32(r.benOrIndex), EndIndex: rpc.GetIndex(),
		}

		// request data from other replicas
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.getCommittedDataRPC, args)
			}
		}

		// r.benOrStatus = WaitingToBeUpdated	
		return 
	}

	if r.benOrState.benOrStage == Stopped {
		return
	}

	if r.benOrState.benOrIteration < int(rpc.GetIteration()) {
		if !r.isLeader {
			r.pq.push(r.benOrState.benOrBroadcastRequest) // add own value back
		}

		var request Entry
		if (r.log[r.benOrIndex].Term != -1) {
			request = r.log[r.benOrIndex]
		} else if (!r.pq.isEmpty()) {
			request = r.pq.pop()
		} else {
			return
		}

		r.benOrState = BenOrState{
			benOrStage: uint8(rpc.GetStage()),
			benOrIteration: int(rpc.GetIteration()),
			benOrPhase: int(rpc.GetPhase()),
			benOrVote: int(rpc.GetVote()),
			benOrMajRequest: rpc.GetMajRequest(),
			benOrBroadcastRequest: request,
			benOrRepliesReceived: 1,
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: []BenOrConsensusMsg{rpc},
			biasedCoin: false,
		}

		if r.benOrState.benOrStage == StageOne {
			r.startBenOrConsensusStage1(false)
		} else if r.benOrState.benOrStage == StageTwo {
			r.startBenOrConsensusStage2(false)
		}
		return
	}

	if r.benOrState.benOrIteration > int(rpc.GetIteration()) {
		if _, ok := rpc.(*BenOrConsensus); ok {
			args := &BenOrBroadcastReply{
				SenderId: r.Id, Term: int32(r.currentTerm), Index: rpc.GetIndex(), Iteration: int32(r.benOrIndex), ClientReq: r.benOrState.benOrBroadcastRequest,
			}
			r.SendMsg(rpc.GetSenderId(), r.benOrBroadcastReplyRPC, args)
		}
		return
	}

	// r.benOrIteration == int(rpc.Iteration) and r.benOrIndex == int(rpc.Index)
	if r.benOrState.benOrPhase < int(rpc.GetPhase()) || 
		(r.benOrState.benOrPhase == int(rpc.GetPhase()) && int(r.benOrState.benOrStage) < int(rpc.GetStage())) {
		r.benOrState = BenOrState{
			benOrStage: uint8(rpc.GetStage()),
			benOrIteration: r.benOrState.benOrIteration,
			benOrPhase: int(rpc.GetPhase()),
			benOrVote: int(rpc.GetVote()),
			benOrMajRequest: rpc.GetMajRequest(),
			benOrBroadcastRequest: r.benOrState.benOrBroadcastRequest,
			benOrRepliesReceived: 1,
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: []BenOrConsensusMsg{rpc},
			biasedCoin: false,
		}

		// r.benOrState.benOrPhase = int(rpc.GetPhase())
		// r.benOrState.benOrRepliesReceived = 1

		// r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, rpc)
		// r.benOrState.benOrVote = int(rpc.GetVote())
		// r.benOrState.benOrMajRequest = rpc.GetMajRequest()

		if (r.benOrState.benOrStage == StageOne) {
			r.startBenOrConsensusStage1(false)
		} else if (r.benOrState.benOrStage == StageTwo) {
			r.startBenOrConsensusStage2(false)
		}
		return
	}

	if r.benOrState.benOrPhase > int(rpc.GetPhase()) || 
		(r.benOrState.benOrPhase == int(rpc.GetPhase()) && int(r.benOrState.benOrStage) > int(rpc.GetStage())){
		if t, ok := rpc.(*BenOrConsensus); ok {
			args := &BenOrConsensusReply{
				SenderId: r.Id, Term: int32(r.currentTerm), Index: t.Index, Iteration: int32(r.benOrState.benOrIteration), 
				Phase: int32(r.benOrState.benOrPhase), Vote: int32(r.benOrState.benOrVote), MajRequest: r.benOrState.benOrMajRequest, 
				LeaderRequest: r.log[r.benOrIndex], Stage: int32(r.benOrState.benOrStage),
			}
			r.SendMsg(rpc.GetSenderId(), r.benOrConsensusReplyRPC, args)
		}
		return
	}

	// Same phase, iteration, and stage
	notReceivedYet := true
	for _, msg := range r.benOrState.benOrConsensusMessages {
		if msg.GetSenderId() == rpc.GetSenderId() {
			notReceivedYet = false
			break
		}
	}

	if r.benOrState.benOrStage == StageOne && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, rpc)

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.startBenOrConsensusStage2(true)
		}
	}

	if r.benOrState.benOrStage == StageTwo && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, rpc)

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.handleBenOrStageEnd()
		}
	}

	if t, ok := rpc.(*BenOrConsensus); ok {
		args := &BenOrBroadcastReply{
			SenderId: r.Id, Term: int32(r.currentTerm), Index: t.Index, Iteration: int32(r.benOrIndex), ClientReq: r.benOrState.benOrBroadcastRequest,
		}
		r.SendMsg(rpc.GetSenderId(), r.benOrBroadcastReplyRPC, args)
	}
}


func (r *Replica) handleBenOrStageEnd() {
	var n int = len(r.benOrState.benOrConsensusMessages)

	msgs := r.benOrState.benOrConsensusMessages

	numVotes := make([]int, 3) // values initialized to 0

	for i := 0; i < n; i++ {
		numVotes[msgs[i].GetVote()]++
	}
	
	if numVotes[Vote1] > r.N/2 {
		// commit entry
		r.benOrState.benOrMajRequest.BenOrActive = False
		r.log[r.benOrIndex] = r.benOrState.benOrMajRequest

		r.benOrState = BenOrState{
			benOrStage: Stopped,
			benOrIteration: 0,
			benOrPhase: 0,
			benOrVote: 0,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		}

		r.preparedIndex++
		r.benOrIndex = r.preparedIndex

		// TODO: close retry channel
		clearTimer(r.benOrResendTimer)

		// restart BenOr Timer
		timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
		setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

		return
	} else if numVotes[Vote0] > r.N/2 {
		// TODO: add a check to see if this is already a leader entry?
		r.pq.push(r.benOrState.benOrBroadcastRequest)

		r.benOrState = BenOrState{
			benOrStage: Stopped,
			benOrIteration: r.benOrState.benOrIteration+1,
			benOrPhase: 0,
			benOrVote: 0,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		}
		return
	} else { // at least one question mark
		r.benOrState.benOrPhase++

		var initialVote int
		if r.benOrState.biasedCoin {
			initialVote = 0
		} else {
			initialVote = rand.Intn(2)
		}

		r.benOrState = BenOrState{
			benOrStage: StageOne,
			benOrIteration: r.benOrState.benOrIteration+1,
			benOrPhase: 0,
			benOrVote: initialVote,
			benOrMajRequest: r.benOrState.benOrMajRequest,
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: r.benOrState.biasedCoin,
		}

		r.startBenOrConsensusStage1(false)
	}
}

/************************************** Resend Ben Or **********************************************/

func (r *Replica) resendBenOrTimer() {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	if r.benOrState.benOrStage == Broadcasting {
		args := &BenOrBroadcast{
			SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), Iteration: int32(r.benOrState.benOrIteration), ClientReq: r.benOrState.benOrBroadcastRequest}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	} else if r.benOrState.benOrStage == StageOne {
		foundMajRequest := int32(r.benOrState.benOrVote)
		majRequest := r.benOrState.benOrMajRequest
		leaderEntry := benOrUncommittedLogEntry(-1)
		if r.log[r.benOrIndex].FromLeader == True {
			leaderEntry = r.log[r.benOrIndex]
		}
		args := &BenOrConsensus{
			SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), 
			Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), 
			Vote: foundMajRequest, MajRequest: majRequest, LeaderRequest: leaderEntry, Stage: 1}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	} else if r.benOrState.benOrStage == StageTwo {
		vote := int32(r.benOrState.benOrVote)
		majRequest := r.benOrState.benOrMajRequest
		leaderEntry := benOrUncommittedLogEntry(-1)
		if r.log[r.benOrIndex].FromLeader == True{
			leaderEntry = r.log[r.benOrIndex]
		}
		args := &BenOrConsensus{
			SenderId: r.Id, Term: int32(r.currentTerm), Index: int32(r.benOrIndex), 
			Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), 
			Vote: vote, MajRequest: majRequest, LeaderRequest: leaderEntry, Stage: 2} // last entry represents the phase
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrConsensusRPC, args)
			}
		}
	}
}