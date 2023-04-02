package randomizedpaxos

import (
	"log"
	"math/rand"
	"randomizedpaxosproto"
	"sort"
	"time"
)

type BenOrState struct {
	benOrStatus						uint8
	benOrIteration					int
	benOrPhase						int

	benOrVote						int
	benOrMajRequest					Entry

	// benOrStage					uint8
	benOrBroadcastRequest 			Entry
	// benOrWaitingForIndex			int
	benOrRepliesReceived			int
	benOrConsensusMessages			[]randomizedpaxosproto.BenOrConsensus
	benOrBroadcastMessages			[]BenOrBroadcastData
	biasedCoin						bool
}

type BenOrBroadcastData struct {
	SenderId		   				int32
    Term							int32
    Index                           int32
    ClientReq						Entry
}

func getBenOrBroadcastData (rpc interface{}) BenOrBroadcastData {
	switch t := rpc.(type) {
		case randomizedpaxosproto.BenOrBroadcast:
			return BenOrBroadcastData{
				SenderId: t.SenderId,
				Term: t.Term,
				Index: t.Index,
				ClientReq: t.ClientReq,
			}
		case randomizedpaxosproto.BenOrBroadcastReply:
			return BenOrBroadcastData{
				SenderId: t.ReplicaId,
				Term: t.Term,
				Index: t.Index,
				ClientReq: t.ClientReq,
			}
		default:
			log.Fatalf("Wrong type for Ben Or Broadcast!\n")
			return BenOrBroadcastData{}
	}
}

/************************************** Ben Or Braodcast **********************************************/
func (r *Replica) startBenOrPlus () {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	r.setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	iteration := r.benOrState.benOrIteration

	r.benOrState = BenOrState{
		benOrStatus: Broadcasting,
		benOrPhase: 0,
		biasedCoin: false,
		benOrRepliesReceived: 0,
		benOrIteration: iteration + 1,
		benOrBroadcastMessages: make([]BenOrBroadcastData, 0),
		benOrConsensusMessages: make([]randomizedpaxosproto.BenOrConsensus, 0),
	}
	
	var request Entry
	if (r.log[r.benOrIndex].Term != -1) {
		request = r.log[r.benOrIndex]
	} else {
		request = r.pq.pop()
	}

	args := &randomizedpaxosproto.BenOrBroadcast{
		r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(iteration), request}
	// r.SendMsg(r.Id, r.benOrBroadcastRPC, args)
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
		}
	}

	// received your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, getBenOrBroadcastData(*args))
	r.benOrState.benOrBroadcastRequest = request
}


func (r *Replica) handleBenOrBroadcast (rpc *randomizedpaxosproto.BenOrBroadcast) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			r.clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		r.isLeader = false
	}

	if (int(rpc.Index) < r.benOrIndex) {
		args := &randomizedpaxosproto.GetCommittedDataReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.log[rpc.Index:r.benOrIndex+1],
		}
		r.SendMsg(rpc.SenderId, r.getCommittedDataReplyRPC, args)
		return
	}

	if (r.benOrIndex < int(rpc.Index)) {
		args := &randomizedpaxosproto.GetCommittedData{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), rpc.Index,
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.getCommittedDataRPC, args)
			}
		}

		// r.benOrStatus = WaitingToBeUpdated	
		return 
	}

	if (r.benOrState.benOrIteration < int(rpc.Iteration)) {
		r.benOrState.benOrIteration = int(rpc.Iteration)

		r.benOrState.benOrRepliesReceived = 1
		r.benOrState.benOrBroadcastMessages = make([]BenOrBroadcastData, 0)
		r.benOrState.benOrConsensusMessages = make([]randomizedpaxosproto.BenOrConsensus, 0)

		r.pq.push(r.benOrState.benOrBroadcastRequest)

		// if r.benOrStatus != WaitingToBeUpdated {
			r.benOrState.benOrStatus = Broadcasting

			var request Entry
			if (r.log[r.benOrIndex].Term != -1) {
				request = r.log[r.benOrIndex]
			} else {
				request = r.pq.pop()
			}

			r.benOrState.benOrBroadcastRequest = request

			args := &randomizedpaxosproto.BenOrBroadcastReply{
				r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), request,
			}
			r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		// }
		return
	}

	if (r.benOrState.benOrIteration > int(rpc.Iteration)) {
		args := &randomizedpaxosproto.BenOrBroadcastReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrState.benOrBroadcastRequest,
		}
		r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		return
	}

	// r.benOrIteration == int(rpc.Iteration) and r.benOrIndex == int(rpc.Index)
	notReceivedYet := true
	for _, msg := range r.benOrState.benOrBroadcastMessages {
		if msg.SenderId == rpc.SenderId {
			notReceivedYet = false
			break
		}
	}

	if r.benOrState.benOrStatus == Broadcasting && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, getBenOrBroadcastData(*rpc))

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.startBenOrConsensusStage1(true)
		}
	}

	args := &randomizedpaxosproto.BenOrBroadcastReply{
		r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrState.benOrBroadcastRequest,
	}
	r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
}


func (r *Replica) handleBenOrBroadcastReply(rpc *randomizedpaxosproto.BenOrBroadcastReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			r.clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		r.isLeader = false
	}

	if (int(rpc.Index) < r.benOrIndex) {
		args := &randomizedpaxosproto.GetCommittedDataReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.log[rpc.Index:r.benOrIndex+1],
		}
		r.SendMsg(rpc.ReplicaId, r.getCommittedDataReplyRPC, args)
		return
	}

	if (r.benOrIndex < int(rpc.Index)) {
		args := &randomizedpaxosproto.GetCommittedData{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), rpc.Index,
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.getCommittedDataRPC, args)
			}
		}

		// r.benOrStatus = WaitingToBeUpdated	
		return 
	}

	if (r.benOrState.benOrIteration < int(rpc.Iteration)) {
		r.benOrState.benOrIteration = int(rpc.Iteration)

		r.benOrState.benOrRepliesReceived = 1
		r.benOrState.benOrBroadcastMessages = make([]BenOrBroadcastData, 0)
		r.benOrState.benOrConsensusMessages = make([]randomizedpaxosproto.BenOrConsensus, 0)

		r.pq.push(r.benOrState.benOrBroadcastRequest)

		// if r.benOrStatus != WaitingToBeUpdated {
			r.benOrState.benOrStatus = Broadcasting

			var request Entry
			if (r.log[r.benOrIndex].Term != -1) {
				request = r.log[r.benOrIndex]
			} else {
				request = r.pq.pop()
			}

			r.benOrState.benOrBroadcastRequest = request

			// args := &randomizedpaxosproto.BenOrBroadcastReply{
			// 	r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), request,
			// }
			// r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		// }
		return
	}

	if (r.benOrState.benOrIteration > int(rpc.Iteration)) {
		// args := &randomizedpaxosproto.BenOrBroadcastReply{
		// 	r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrBroadcastRequest,
		// }
		// r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		return
	}

	// r.benOrIteration == int(rpc.Iteration) and r.benOrIndex == int(rpc.Index)
	notReceivedYet := true
	for _, msg := range r.benOrState.benOrBroadcastMessages {
		if msg.SenderId == rpc.ReplicaId {
			notReceivedYet = false
			break
		}
	}

	if r.benOrState.benOrStatus == Broadcasting && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet{
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, getBenOrBroadcastData(*rpc))

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.startBenOrConsensusStage1(true)
		}
	}

	// args := &randomizedpaxosproto.BenOrBroadcastReply{
	// 	r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrBroadcastRequest,
	// }
	// r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
}

/************************************** Ben Or Consensus **********************************************/

// only called if actually need to run benOr at this index
func (r *Replica) startBenOrConsensusStage1(initialize bool) {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	r.setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	var foundMajRequest int32;
	var majRequest Entry;

	if initialize {
		r.benOrState.benOrPhase++
		r.benOrState.benOrStatus = StageOne
		r.benOrState.benOrRepliesReceived = 0

		var n int = len(r.benOrState.benOrBroadcastMessages)

		msgs := r.benOrState.benOrBroadcastMessages

		sort.Slice(msgs, func(i, j int) bool {
			if msgs[i].Index != msgs[j].Index {
				return msgs[i].Index < msgs[j].Index
			}
			if msgs[i].ClientReq.Timestamp != msgs[j].ClientReq.Timestamp {
				return msgs[i].ClientReq.Timestamp < msgs[j].ClientReq.Timestamp
			}
			return msgs[i].ClientReq.SenderId < msgs[j].ClientReq.SenderId
		})

		foundMajRequest = int32(Vote0)
		majRequest = benOrUncommittedLogEntry(-1)
		curRequest := benOrUncommittedLogEntry(-1)

		counter := 0
		timestamp := int64(-1)
		index := int32(-1)
		senderId := int32(-1)

		for i := 0; i < n; i++ {
			if msgs[i].Index == index && msgs[i].ClientReq.Timestamp == timestamp && 
						msgs[i].ClientReq.SenderId == senderId {
				counter++
				if msgs[i].ClientReq.Term > curRequest.Term {
					curRequest = msgs[i].ClientReq
				}
			} else {
				counter = 1
				index = msgs[i].Index
				timestamp = msgs[i].ClientReq.Timestamp
				senderId = msgs[i].ClientReq.SenderId
				curRequest = msgs[i].ClientReq
			}
			
			if counter >= r.N/2 {
				foundMajRequest = int32(Vote1)
				majRequest = curRequest
				break
			}
			// if i + r.N/2 < n && msgs[i].Index == msgs[i+r.N/2].Index &&
			// msgs[i].ClientReq.Timestamp == msgs[i+r.N/2].ClientReq.Timestamp && 
			// msgs[i].ClientReq.SenderId == msgs[i+r.N/2].ClientReq.SenderId {
				
			// 	majRequest = msgs[i].ClientReq
			// 	foundMajRequest = int32(Vote1)
			// 	break
			// }
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

	args := &randomizedpaxosproto.BenOrConsensus{
		r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(r.benOrState.benOrIteration), 
		int32(r.benOrState.benOrPhase), foundMajRequest, majRequest, leaderEntry, 1} // last entry represents the phase
	// r.SendMsg(r.Id, r.benOrBroadcastRPC, args)
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}

	// received your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, *args)
}

func (r *Replica) startBenOrConsensusStage2(initialize bool) {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	r.setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	var vote int32
	var majRequest Entry;

	if initialize {
		r.benOrState.benOrPhase++
		r.benOrState.benOrStatus = StageOne
		r.benOrState.benOrRepliesReceived = 0

		// var n int = len(r.benOrState.benOrConsensusMessages)

		msgs := r.benOrState.benOrConsensusMessages
	
		voteCount := [2]int{0, 0}
		for _, msg := range msgs {
			voteCount[msg.Vote]++
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

	args := &randomizedpaxosproto.BenOrConsensus{
		r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(r.benOrState.benOrIteration),
		int32(r.benOrState.benOrPhase), vote, majRequest, leaderEntry, 2} // last entry represents the phase

	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}

	// received your own message
	r.benOrState.benOrRepliesReceived++
	r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, *args)
}

func (r *Replica) handleBenOrConsensus (rpc *randomizedpaxosproto.BenOrConsensus) {
	if (int(rpc.Term) > r.currentTerm) { // update term but don't return
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			r.clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}

		r.isLeader = false
	}

	if (int(rpc.Index) < r.benOrIndex) {
		args := &randomizedpaxosproto.GetCommittedDataReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.log[rpc.Index:r.benOrIndex+1],
		}
		r.SendMsg(rpc.SenderId, r.getCommittedDataReplyRPC, args)
		return
	}

	if (r.benOrIndex < int(rpc.Index)) {
		args := &randomizedpaxosproto.GetCommittedData{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), rpc.Index,
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

	if (r.benOrState.benOrStatus == Stopped) {
		return
	}

	if (r.benOrState.benOrIteration < int(rpc.Iteration)) {
		r.benOrState.benOrIteration = int(rpc.Iteration)

		r.benOrState.benOrRepliesReceived = 1
		r.benOrState.benOrBroadcastMessages = make([]BenOrBroadcastData, 0)
		r.benOrState.benOrConsensusMessages = make([]randomizedpaxosproto.BenOrConsensus, 0)

		// r.pq.push(r.benOrBroadcastRequest)

		r.benOrState.benOrStatus = Broadcasting

		var request Entry
		if (r.log[r.benOrIndex].Term != -1) {
			request = r.log[r.benOrIndex]
		} else {
			request = r.pq.pop()
		}

		r.benOrState.benOrBroadcastRequest = request

		args := &randomizedpaxosproto.BenOrBroadcastReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), request,
		}
		r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		return
	}

	if (r.benOrState.benOrIteration > int(rpc.Iteration)) {
		args := &randomizedpaxosproto.BenOrBroadcastReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrState.benOrBroadcastRequest,
		}
		r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
		return
	}

	// r.benOrIteration == int(rpc.Iteration) and r.benOrIndex == int(rpc.Index)
	
	// TODO: check to make sure it's the same phase
	if (r.benOrState.benOrPhase < int(rpc.Phase)) {
		r.benOrState.benOrPhase = int(rpc.Phase)
		r.benOrState.benOrRepliesReceived = 2

		r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, *rpc, *rpc)
		r.benOrState.benOrVote = int(rpc.Vote)
		r.benOrState.benOrMajRequest = rpc.MajRequest

		if (r.benOrState.benOrPhase == 1) {
			r.startBenOrConsensusStage1(false)
		} else if (r.benOrState.benOrPhase == 2) {
			r.startBenOrConsensusStage2(false)
		}
		return
	}

	if (r.benOrState.benOrPhase > int(rpc.Phase)) {
		args := &randomizedpaxosproto.BenOrConsensusReply{
			r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrState.benOrIteration), 
			int32(r.benOrState.benOrPhase), int32(r.benOrState.benOrVote), r.benOrState.benOrMajRequest, 
			r.log[r.benOrIndex], int32(r.benOrState.benOrStatus),
		}
		r.SendMsg(rpc.SenderId, r.benOrConsensusReplyRPC, args)
		return
	}

	// TODO: if same phase, then handle consensus request

	notReceivedYet := true
	for _, msg := range r.benOrState.benOrConsensusMessages {
		if msg.SenderId == rpc.SenderId {
			notReceivedYet = false
			break
		}
	}

	if r.benOrState.benOrStatus == StageOne && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, *rpc)

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.startBenOrConsensusStage2(true)
		}
	}

	if r.benOrState.benOrStatus == StageTwo && r.benOrState.benOrRepliesReceived < r.N/2 && notReceivedYet {
		r.benOrState.benOrRepliesReceived++
		r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, *rpc)

		if r.benOrState.benOrRepliesReceived >= r.N/2 {
			// move past broadcasting stage
			r.handleBenOrStageEnd()
		}
	}

	args := &randomizedpaxosproto.BenOrBroadcastReply{
		r.Id, int32(r.currentTerm), rpc.Index, int32(r.benOrIndex), r.benOrState.benOrBroadcastRequest,
	}
	r.SendMsg(rpc.SenderId, r.benOrBroadcastReplyRPC, args)
}

func (r *Replica) handleBenOrStageEnd() {
	var n int = len(r.benOrState.benOrConsensusMessages)

	msgs := r.benOrState.benOrConsensusMessages

	numVotes := make([]int, 3) // values initialized to 0

	for i := 0; i < n; i++ {
		numVotes[msgs[i].Vote]++
	}
	
	if numVotes[Vote1] > r.N/2 {
		// commit entry
		r.benOrState.benOrMajRequest.BenOrActive = False
		r.log[r.benOrIndex] = r.benOrState.benOrMajRequest

		r.benOrState = BenOrState{
			benOrStatus: Stopped,
			benOrIteration: 0,
			benOrPhase: 0,
			benOrVote: 0,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastData, 0),
			benOrConsensusMessages: make([]randomizedpaxosproto.BenOrConsensus, 0),
			biasedCoin: false,
		}

		r.preparedIndex++
		r.benOrIndex = r.preparedIndex

		// TODO: close retry channel
		r.clearTimer(r.benOrResendTimer)

		return
	} else if numVotes[Vote0] > r.N/2 {
		// TODO: add a check to see if this is already a leader entry?
		r.pq.push(r.benOrState.benOrBroadcastRequest)

		r.benOrState = BenOrState{
			benOrStatus: Stopped,
			benOrIteration: r.benOrState.benOrIteration+1,
			benOrPhase: 0,
			benOrVote: 0,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastData, 0),
			benOrConsensusMessages: make([]randomizedpaxosproto.BenOrConsensus, 0),
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
			benOrStatus: StageOne,
			benOrIteration: r.benOrState.benOrIteration+1,
			benOrPhase: 0,
			benOrVote: initialVote,
			benOrMajRequest: r.benOrState.benOrMajRequest,
			benOrRepliesReceived: 0,
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastMessages: make([]BenOrBroadcastData, 0),
			benOrConsensusMessages: make([]randomizedpaxosproto.BenOrConsensus, 0),
			biasedCoin: r.benOrState.biasedCoin,
		}

		r.startBenOrConsensusStage1(false)
	}
}

/************************************** Ben Or **********************************************/

func (r *Replica) resendBenOrTimer() {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	r.setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	if r.benOrState.benOrStatus == Broadcasting {
		args := &randomizedpaxosproto.BenOrBroadcast{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(r.benOrState.benOrIteration), r.benOrState.benOrBroadcastRequest}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	} else if r.benOrState.benOrStatus == StageOne {
		foundMajRequest := int32(r.benOrState.benOrVote)
		majRequest := r.benOrState.benOrMajRequest
		leaderEntry := benOrUncommittedLogEntry(-1)
		if r.log[r.benOrIndex].FromLeader == True {
			leaderEntry = r.log[r.benOrIndex]
		}
		args := &randomizedpaxosproto.BenOrConsensus{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(r.benOrState.benOrIteration), int32(r.benOrState.benOrPhase), foundMajRequest, majRequest, leaderEntry, 1}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	} else if r.benOrState.benOrStatus == StageTwo {
		vote := int32(r.benOrState.benOrVote)
		majRequest := r.benOrState.benOrMajRequest
		leaderEntry := benOrUncommittedLogEntry(-1)
		if r.log[r.benOrIndex].FromLeader == True{
			leaderEntry = r.log[r.benOrIndex]
		}
		args := &randomizedpaxosproto.BenOrConsensus{
			r.Id, int32(r.currentTerm), int32(r.benOrIndex), int32(r.benOrState.benOrIteration),
			int32(r.benOrState.benOrPhase), vote, majRequest, leaderEntry, 2} // last entry represents the phase
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrConsensusRPC, args)
			}
		}
	}
}

func (r *Replica) handleBenOrConsensusReply(rpc *randomizedpaxosproto.BenOrConsensusReply) {
	// TODO: implement
}