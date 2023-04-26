package hydrofoil

import (
	"bufio"
	"dlog"
	"encoding/binary"
	"fastrpc"
	"fmt"
	"genericsmr"
	"genericsmrproto"
	"hydrofoilproto"
	"io"
	"log"
	"math/rand"
	"sort"
	"state"
	"time"
)

type Entry = hydrofoilproto.Entry
type ReplicateEntries = hydrofoilproto.ReplicateEntries
type ReplicateEntriesReply = hydrofoilproto.ReplicateEntriesReply
type RequestVote = hydrofoilproto.RequestVote
type RequestVoteReply = hydrofoilproto.RequestVoteReply
type BenOrBroadcast = hydrofoilproto.BenOrBroadcast
type BenOrBroadcastReply = hydrofoilproto.BenOrBroadcastReply
type BenOrConsensus = hydrofoilproto.BenOrConsensus
type BenOrConsensusReply = hydrofoilproto.BenOrConsensusReply
type GetCommittedData = hydrofoilproto.GetCommittedData
type GetCommittedDataReply = hydrofoilproto.GetCommittedDataReply
// type InfoBroadcast = randomizedpaxosproto.InfoBroadcast
// type InfoBroadcastReply = randomizedpaxosproto.InfoBroadcastReply

type RPC interface {
	GetSenderId() int32
	GetTerm() int32
	GetCommitIndex() int32
	GetLeaderTerm() int32
	GetLogLength() int32
}

type UpdateMsg interface {
	RPC
	GetStartIndex() int32
	GetEntries() []Entry
}

type ReplyMsg interface {
	UpdateMsg
	GetPQEntries() []Entry
}

type BenOrBroadcastMsg interface {
	RPC
	GetBenOrMsgValid() uint8
	GetIteration() int32
	GetBroadcastEntry() Entry
	GetStartIndex() int32
	GetEntries() []Entry
	GetPQEntries() []Entry
}

type BenOrConsensusMsg interface {
	RPC
	GetBenOrMsgValid() uint8
	GetIteration() int32
	GetPhase() int32
	GetStage() uint8
	GetVote() uint8
	GetPrevPhaseFinalValue() uint8
	GetHaveMajEntry() uint8
	GetMajEntry() Entry
	GetStartIndex() int32
	GetEntries() []Entry
	GetPQEntries() []Entry
}

const ( // for benOrStatus
	Broadcasting uint8	= 0
	StageOne	   	= 1
	StageTwo	   	= 2
	NotRunning	   	= 3
)

const (
	Vote0 uint8 	   	= 0
	Vote1		   	= 1
	VoteQuestionMark  	= 2
	VoteUninitialized 	= 3
)

const (
	False uint8 		= hydrofoilproto.False
	True        		= hydrofoilproto.True
)

func convertBoolToInteger(b bool) uint8 {
	if b { return True }
	return False
}

func convertIntegerToBool(i uint8) bool {
	if i == True { return true }
	return false
}

var zeroTime time.Time

// const INJECT_SLOWDOWN = false
// const CHAN_BUFFER_SIZE = 200000
// const MAX_BATCH = 5000
// const BATCH_INTERVAL = 100 * time.Microsecond

type LeaderState struct {
	isLeader			bool
	repNextIndex			[]int // next index to send to each replica
	repMatchIndex			[]int // index of highest known replicated entry on replica
	numEntries			int
	lastMsgTimestamp		[]time.Time // time we last heard a replicateentriesreply from each replica
}

type CandidateState struct {
	isCandidate			bool
	votesReceived			int
}

type BenOrState struct {
	// benOrIndex = commitIndex + 1 at all times!
	benOrRunning			bool

	benOrIteration			int
	benOrPhase			int
	benOrStage			uint8

	benOrBroadcastEntry 		Entry // entry that you initially broadcast this iteration
	benOrBroadcastMsgs		[]Entry
	heardServerFromBroadcast	[]bool

	haveMajEntry			bool
	benOrMajEntry			Entry // majority entry received in BenOrBroadcast. If none, then this is emptyEntry.

	benOrVote			uint8
	benOrConsensusMsgs		[]uint8
	heardServerFromConsensus	[]bool

	prevPhaseFinalValue		uint8 // final value from previous phase
	
	biasedCoin			bool
}

type Replica struct {
	*genericsmr.Replica // extends a generic Paxos replica

	replicateEntriesChan         	chan fastrpc.Serializable
	replicateEntriesReplyChan	chan fastrpc.Serializable
	requestVoteChan		        chan fastrpc.Serializable
	requestVoteReplyChan     	chan fastrpc.Serializable
	benOrBroadcastChan    		chan fastrpc.Serializable
	benOrBroadcastReplyChan    	chan fastrpc.Serializable
	benOrConsensusChan    		chan fastrpc.Serializable
	benOrConsensusReplyChan   	chan fastrpc.Serializable
	getCommittedDataChan		chan fastrpc.Serializable
	getCommittedDataReplyChan	chan fastrpc.Serializable
	replicateEntriesRPC          	uint8
	replicateEntriesReplyRPC     	uint8
	requestVoteRPC           	uint8
	requestVoteReplyRPC      	uint8
	benOrBroadcastRPC           	uint8
	benOrBroadcastReplyRPC      	uint8
	benOrConsensusRPC      		uint8
	benOrConsensusReplyRPC     	uint8
	getCommittedDataRPC		uint8
	getCommittedDataReplyRPC	uint8

	term				int
	votedFor			int
	log				[]Entry
	pq				ExtendedPriorityQueue // to be fixed
	inLog				Set

	commitIndex			int // index of highest log entry known to be committed
	leaderTerm			int // highest term seen from a leader
	lastApplied			int // index of last log entry applied to state machine
	lastHeardFromLeader		time.Time // time we last heard from the leader

	leaderState			LeaderState
	benOrState			BenOrState
	candidateState			CandidateState

	// proposals			map[uint32]*genericsmr.Propose

	// recentlySentGetCommit		bool // true if we've recently sent out a GetCommittedData request

	electionTimeout         	int
	heartbeatTimeout         	int
	benOrStartTimeout		int
	benOrResendTimeout		int
	// getCommittedTimeout		int // time before we issue another GetCommitData request

	clientWriters      		map[uint32]*bufio.Writer
	heartbeatTimer			*ServerTimer
	electionTimer			*ServerTimer
	benOrStartTimer			*ServerTimer
	benOrResendTimer		*ServerTimer
	// getCommittedTimer		*ServerTimer
}


type Instance struct {
	cmds   []state.Command
	ballot int32
}

// type Pair struct {
// 	Idx  int
// 	Term int
// }

// UNSURE IF WE EVER NEED THIS ACTUALLY
//append a log entry to stable storage
func (r *Replica) recordInstanceMetadata(inst *Instance) {
	if !r.Durable {
		return
	}

	// var b [5]byte
	var b [4]byte
	binary.LittleEndian.PutUint32(b[0:4], uint32(inst.ballot))
	// b[4] = byte(inst.status)
	r.StableStore.Write(b[:])
}

//write a sequence of commands to stable storage
func (r *Replica) recordCommands(cmds []state.Command) {
	if !r.Durable {
		return
	}

	if cmds == nil {
		return
	}
	for i := 0; i < len(cmds); i++ {
		cmds[i].Marshal(io.Writer(r.StableStore))
	}
}

//sync with the stable store
func (r *Replica) sync() {
	if !r.Durable {
		return
	}

	r.StableStore.Sync()
}

// Manage Client Writers
func (r *Replica) registerClient(clientId uint32, writer *bufio.Writer) uint8 {
	fmt.Println("Registering client", clientId)
	w, exists := r.clientWriters[clientId]

	if !exists {
		r.clientWriters[clientId] = writer
		return True
	}

	if w == writer {
		return True
	}

	return False
}

var emptyEntry = Entry{
	Data: state.Command{},
	ServerId: -1,
	Term: -1,
	Index: -1,
	Timestamp: -1,
}

var emptyLeaderState = LeaderState{
	isLeader: false,
	repNextIndex: make([]int, 0), // replica next index
	repMatchIndex: make([]int, 0),
	numEntries: 1,
	lastMsgTimestamp: make([]time.Time, 0),
}

var emptyCandidateState = CandidateState{
	isCandidate: false,
	votesReceived: 0,
}

var emptyBenOrState = BenOrState{
	benOrRunning: false,

	benOrIteration: 0,
	benOrPhase: 0,
	benOrStage: NotRunning,

	benOrBroadcastEntry: emptyEntry,
	benOrBroadcastMsgs: make([]Entry, 0),
	heardServerFromBroadcast: make([]bool, 0),

	haveMajEntry: false,
	benOrMajEntry: emptyEntry,

	benOrVote: VoteUninitialized,
	benOrConsensusMsgs: make([]uint8, 0),
	heardServerFromConsensus: make([]bool, 0),

	prevPhaseFinalValue: VoteUninitialized,
	
	biasedCoin: false,
}

func newReplicaFullParam(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool, durable bool, 
	isProduction bool, electionTimeout int, heartbeatTimeout int, benOrStartTimeout int, benOrResendTimeout int) *Replica {
	r := &Replica{
		Replica: genericsmr.NewReplica(id, peerAddrList, thrifty, exec, dreply),
		replicateEntriesChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		replicateEntriesReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		requestVoteChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		requestVoteReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		benOrBroadcastChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		benOrBroadcastReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		benOrConsensusChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		benOrConsensusReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		getCommittedDataChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		getCommittedDataReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		replicateEntriesRPC: 0,
		replicateEntriesReplyRPC: 0, 
		requestVoteRPC: 0,
		requestVoteReplyRPC: 0,
		benOrBroadcastRPC: 0,
		benOrBroadcastReplyRPC: 0,
		benOrConsensusRPC: 0,
		benOrConsensusReplyRPC: 0,
		getCommittedDataRPC: 0,
		getCommittedDataReplyRPC: 0,

		term: 0,
		votedFor: -1,
		log: []Entry{emptyEntry},
		pq: newExtendedPriorityQueue(),
		inLog: newSet(),

		commitIndex: 0,
		leaderTerm: 0,
		lastApplied: 0, // 0 entry is already applied (it's an empty entry)
		lastHeardFromLeader: time.Now(),

		leaderState: emptyLeaderState,
		benOrState: emptyBenOrState,
		candidateState: emptyCandidateState,

		// recentlySentGetCommit: false,
		// proposals: make(map[uint32]*genericsmr.Propose),

		electionTimeout: electionTimeout,
		heartbeatTimeout: heartbeatTimeout,
		benOrStartTimeout: benOrStartTimeout,
		benOrResendTimeout: benOrResendTimeout,
		// getCommittedTimeout: getCommittedTimeout,

		clientWriters: make(map[uint32]*bufio.Writer),
		heartbeatTimer: newTimer(),
		electionTimer: newTimer(),
		benOrStartTimer: newTimer(),
		benOrResendTimer: newTimer(),
		// getCommittedTimer: newTimer(),
	}

	r.Durable = durable
	r.TestingState.IsProduction = isProduction
	// r.TestingState.IsConnected.Mu.Lock()
	for i := 0; i < r.N; i++ {
		r.Connected[i] = false
	}
	// r.TestingState.IsConnected.Mu.Unlock()

	r.replicateEntriesRPC = r.RegisterRPC(new(ReplicateEntries), r.replicateEntriesChan)
	r.replicateEntriesReplyRPC = r.RegisterRPC(new(ReplicateEntriesReply), r.replicateEntriesReplyChan)
	r.requestVoteRPC = r.RegisterRPC(new(RequestVote), r.requestVoteChan)
	r.requestVoteReplyRPC = r.RegisterRPC(new(RequestVoteReply), r.requestVoteReplyChan)
	r.benOrBroadcastRPC = r.RegisterRPC(new(BenOrBroadcast), r.benOrBroadcastChan)
	r.benOrBroadcastReplyRPC = r.RegisterRPC(new(BenOrBroadcastReply), r.benOrBroadcastReplyChan)
	r.benOrConsensusRPC = r.RegisterRPC(new(BenOrConsensus), r.benOrConsensusChan)
	r.benOrConsensusReplyRPC = r.RegisterRPC(new(BenOrConsensusReply), r.benOrConsensusReplyChan)
	r.getCommittedDataRPC = r.RegisterRPC(new(GetCommittedData), r.getCommittedDataChan)
	r.getCommittedDataReplyRPC = r.RegisterRPC(new(GetCommittedDataReply), r.getCommittedDataReplyChan)

	if r.TestingState.IsProduction {
		go r.run()
	}

	// dlog.Println("Replica", r.Id)

	return r
}

func newReplicaMoreParam(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool, durable bool, isProduction bool) *Replica {
	// return newReplicaFullParam(id, peerAddrList, thrifty, exec, dreply, durable, isProduction, 1e9, 1e9, 50, 40)
	return newReplicaFullParam(id, peerAddrList, thrifty, exec, dreply, durable, isProduction, 150, 15, 1e9, 1e9)
}

// Entry point
func NewReplica(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool, durable bool) *Replica {
	return newReplicaMoreParam(id, peerAddrList, thrifty, exec, dreply, durable, true)
}

// func (r *Replica) clock() {
// 	for !r.Shutdown {
// 		time.Sleep(BATCH_INTERVAL)
// 		clockChan <- true
// 	}
// }

func (r *Replica) getState() (bool, int, int, int) {
	return r.leaderState.isLeader, r.term, r.leaderTerm, r.commitIndex
}

func (r *Replica) executeCommand(i int) {
	if r.TestingState.IsProduction {
		val := r.log[i].Data.Execute(r.State)
		fmt.Println(r.log[i].ServerId, r.Id, i)
		// if r.log[i].ServerId == r.Id {
			if writer, ok := r.clientWriters[r.log[i].Data.ClientId]; ok {
				propreply := &genericsmrproto.ProposeReplyTS{
					OK: True,
					CommandId: r.log[i].Data.OpId,
					Value: val,
					Timestamp: r.log[i].Timestamp} // TODO: check if timestamp is correct
				r.ReplyProposeTS(propreply, writer)
			}
		// }
		// else {
		// 	// fmt.Println(r.log[i].ServerId, r.Id, r.inLog.isCommitted(r.log[i]))
		// 	if (r.log[i].ServerId == r.Id) { 
		// 		propreply := &genericsmrproto.ProposeReplyTS{
		// 			OK: True,
		// 			CommandId: r.log[i].Data.OpId,
		// 			Value: val,
		// 			Timestamp: r.log[i].Timestamp} // TODO: check if timestamp is correct
		// 		r.ReplyProposeTS(propreply, r.proposals[r.log[i].Data.ClientId].Reply)
		// 		// fmt.Println("executeCommand", i, "with opid", r.log[i].Data.OpId)
		// 	}
		// }
		// if !r.inLog.isCommitted(r.log[i]) {
			// r.inLog.commit(r.log[i])
		// }
		// } else if !ok {
		// 	fmt.Println("Client", r.log[i].Data.ClientId, "not connected")
		// }
	} else {
		// dlog.Println("Replica", r.Id, "executing command", i, "with opid", r.log[i].Data.OpId, "START")
		r.TestingState.ResponseChan <- genericsmr.RepCommand{int(r.Id), r.log[i].Data}
		// dlog.Println("Replica", r.Id, "executing command", i, "with opid", r.log[i].Data.OpId, "END")
	}
}

func (r *Replica) runReplica() {
	go r.run()
}

func (r *Replica) executeCommands() {
	for !r.Shutdown {
		executed := false
		if r.commitIndex > r.lastApplied {
			// dlog.Println("Replica", r.Id, "executing from", r.lastApplied + 1, "to", r.commitIndex, "log is: ", logToString(r.log))
		}

		// Execution of the leader's state machine
		for i := r.lastApplied + 1; i <= r.commitIndex; i++ {
			r.executeCommand(i)
			executed = true
		}
		r.lastApplied = r.commitIndex

		if !executed {
			time.Sleep(1000)
		}
	}
}

/* Main event processing loop */
func (r *Replica) run() {
	if (r.TestingState.IsProduction) { 
		r.ConnectToPeers()

		// dlog.Println("Waiting for client connections")

		go r.WaitForClientConnections()
	}

	go r.executeCommands()

	var timeout int
	timeout = rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout) * time.Millisecond)
	
	timeout = rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
	setTimer(r.benOrStartTimer, time.Duration(timeout) * time.Millisecond)

	// dlog.Println("Starting replica", r.Id)

	// if r.Id == 0 {
	// 	r.becomeLeader()
	// }

	// idx := 0
	// times := make([]int64, 10)
	for !r.Shutdown {
		// timesss := time.Now().UnixMicro()
		if r.leaderState.isLeader {
			matchIndices := append(make([]int, 0, r.N), r.leaderState.repMatchIndex...)
			sort.Sort(sort.Reverse(sort.IntSlice(matchIndices)))
			// dlog.Println("Replica", r.Id, "matchIndices", matchIndices)
			if r.commitIndex < matchIndices[r.N/2] && (!r.benOrState.benOrRunning || r.log[matchIndices[r.N/2]] == r.benOrState.benOrMajEntry) {
				// dlog.Println("Leader", r.Id, "committing using RAFT from", r.commitIndex + 1, "to", matchIndices[r.N/2])
				r.commitIndex = matchIndices[r.N/2]
				r.benOrState = emptyBenOrState
				// timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
				// setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
				clearTimer(r.benOrResendTimer)
			}

			if len(r.log) > r.leaderState.numEntries {
				r.leaderState.numEntries = len(r.log)
				r.sendHeartbeat()
			}
		} else if !r.benOrState.benOrRunning && time.Since(r.lastHeardFromLeader) > time.Duration(3 * r.benOrStartTimeout)*time.Millisecond &&
			(r.commitIndex + 1 < len(r.log) || !r.pq.isEmpty()) {
			r.startBenOrPlus()
		}

		select {
			case client := <-r.RegisterClientIdChan:
				fmt.Printf("Client %d registering\n", client.ClientId)
				r.registerClient(client.ClientId, client.Reply)

			case cmd := <-r.TestingState.RequestChan:
				// dlog.Printf("Replica %d received testing command with op %d\n", r.Id, cmd.OpId)
				r.handleProposeCommand(cmd)

			// case <-clockChan:
			// 	//activate the new proposals channel
			// 	onOffProposeChan = r.ProposeChan
			// 	break

			case propose := <-r.ProposeChan:
				//got a Propose from a client
				dlog.Printf("Replica %d received proposal with op %d\n", r.Id, propose.Command.Op)
				r.handlePropose(propose)
				//deactivate the new proposals channel to prioritize the handling of protocol messages
				// if MAX_BATCH > 100 {
				// 	onOffProposeChan = nil
				// }

			case getState := <-r.GetStateChan:
				//got a GetState request from a client
				dlog.Printf("Replica %d received GetState from client\n", r.Id)
				r.handleGetState(getState)
			
			case slowdown := <-r.SlowdownChan:
				//got a Slowdown request from a client
				dlog.Printf("Replica %d received Slowdown from client\n", r.Id)
				r.handleSlowdown(slowdown)
			
			case connect := <-r.ConnectChan:
				//got a Connect request from a client
				dlog.Printf("Replica %d received Connect from client\n", r.Id)
				r.handleConnect(connect)
			
			case disconnect := <-r.DisconnectChan:
				//got a Disconnect request from a client
				dlog.Printf("Replica %d received Disconnect from client\n", r.Id)
				r.handleDisconnect(disconnect)

			case replicateEntriesS := <-r.replicateEntriesChan:
				startTime := time.Now().UnixMicro()
				replicateEntries := replicateEntriesS.(*ReplicateEntries)
				//got a ReplicateEntries message
				// dlog.Printf("Replica %d received ReplicateEntries from replica %d, for term %d with log up to idx %d\n", r.Id, replicateEntries.SenderId, replicateEntries.Term, int(replicateEntries.PrevLogIndex) + len(replicateEntries.Entries))
				r.handleReplicateEntries(replicateEntries)
				fmt.Println("ReplicateEntries took", time.Now().UnixMicro() - startTime, "microseconds")

			case replicateEntriesReplyS := <-r.replicateEntriesReplyChan:
				startTime := time.Now().UnixMicro()
				replicateEntriesReply := replicateEntriesReplyS.(*ReplicateEntriesReply)
				//got a ReplicateEntriesReply message
				// dlog.Printf("Replica %d received ReplicateEntriesReply from replica %d, for term %d\n", r.Id, replicateEntriesReply.SenderId, replicateEntriesReply.Term)
				r.handleReplicateEntriesReply(replicateEntriesReply)
				fmt.Println("ReplicateEntriesReply took", time.Now().UnixMicro() - startTime, "microseconds")

			case requestVoteS := <-r.requestVoteChan:
				startTime := time.Now().UnixMicro()
				requestVote := requestVoteS.(*RequestVote)
				//got a RequestVote message
				// dlog.Printf("Replica %d received RequestVote from replica %d, for term %d\n", r.Id, requestVote.SenderId, requestVote.Term)
				r.handleRequestVote(requestVote)
				fmt.Println("RequestVote took", time.Now().UnixMicro() - startTime, "microseconds")

			case requestVoteReplyS := <-r.requestVoteReplyChan:
				startTime := time.Now().UnixMicro()
				requestVoteReply := requestVoteReplyS.(*RequestVoteReply)
				//got a RequestVoteReply message
				// dlog.Printf("Replica %d received RequestVoteReply from replica %d, for term %d\n", r.Id, requestVoteReply.SenderId, requestVoteReply.Term)
				r.handleRequestVoteReply(requestVoteReply)
				fmt.Println("RequestVoteReply took", time.Now().UnixMicro() - startTime, "microseconds")

			case benOrBroadcastS := <-r.benOrBroadcastChan:
				startTime := time.Now().UnixMicro()
				benOrBroadcast := benOrBroadcastS.(*BenOrBroadcast)
				//got a BenOrBroadcast message
				// dlog.Printf("Replica %d received BenOrBroadcast from replica %d, for term %d and index %d\n", r.Id, benOrBroadcast.SenderId, benOrBroadcast.Term, benOrBroadcast.CommitIndex+1)
				// debug.PrintStack()
				r.handleBenOrBroadcast(benOrBroadcast)
				fmt.Println("BenOrBroadcast from replica", benOrBroadcast.SenderId, "for index", benOrBroadcast.CommitIndex+1, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("BenOrBroadcast took", time.Now().UnixMicro() - startTime, "microseconds")

			case benOrBroadcastReplyS := <-r.benOrBroadcastReplyChan:
				startTime := time.Now().UnixMicro()
				benOrBroadcastReply := benOrBroadcastReplyS.(*BenOrBroadcastReply)
				//got a BenOrBroadcastReply message
				// dlog.Printf("Replica %d received BenOrBroadcastReply from replica %d, for term %d and index %d and validity %d\n", r.Id, benOrBroadcastReply.SenderId, benOrBroadcastReply.Term, benOrBroadcastReply.CommitIndex+1, benOrBroadcastReply.BenOrMsgValid)
				r.handleBenOrBroadcast(benOrBroadcastReply)
				fmt.Println("BenOrBroadcastReply from replica", benOrBroadcastReply.SenderId, "for index", benOrBroadcastReply.CommitIndex+1, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("BenOrBroadcastReply took", time.Now().UnixMicro() - startTime, "microseconds")

			case benOrConsensusS := <-r.benOrConsensusChan:
				startTime := time.Now().UnixMicro()
				benOrConsensus := benOrConsensusS.(*BenOrConsensus)
				//got a BenOrConsensus message
				// dlog.Printf("Replica %d received BenOrConsensus from replica %d, for term %d with vote %d\n", r.Id, benOrConsensus.SenderId, benOrConsensus.Term, benOrConsensus.Vote)
				r.handleBenOrConsensus(benOrConsensus)
				fmt.Println("BenOrBroadcastConsensus Stage", benOrConsensus.Stage, "from replica", benOrConsensus.SenderId, "for index", benOrConsensus.CommitIndex+1, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("BenOrConsensus took", time.Now().UnixMicro() - startTime, "microseconds")

			case benOrConsensusReplyS := <-r.benOrConsensusReplyChan:
				startTime := time.Now().UnixMicro()
				benOrConsensusReply := benOrConsensusReplyS.(*BenOrConsensusReply)
				//got a BenOrConsensusReply message
				// dlog.Printf("Replica %d received BenOrConsensusReply from replica %d, for term %d with vote %d and validity %d\n", r.Id, benOrConsensusReply.SenderId, benOrConsensusReply.Term, benOrConsensusReply.Vote, benOrConsensusReply.BenOrMsgValid)
				r.handleBenOrConsensus(benOrConsensusReply)
				fmt.Println("BenOrBroadcastConsensusReply Stage", benOrConsensusReply.Stage, "from replica", benOrConsensusReply.SenderId, "for index", benOrConsensusReply.CommitIndex+1, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("BenOrConsensusReply took", time.Now().UnixMicro() - startTime, "microseconds")

			case getCommittedDataS := <-r.getCommittedDataChan:
				startTime := time.Now().UnixMicro()
				getCommittedData := getCommittedDataS.(*GetCommittedData)
				//got a GetCommittedData message
				// dlog.Printf("Replica %d received GetCommittedData from replica %d\n", r.Id, getCommittedData.SenderId)
				r.handleGetCommittedData(getCommittedData)
				fmt.Println("GetCommittedData from replica", getCommittedData.SenderId, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("GetCommittedData took", time.Now().UnixMicro() - startTime, "microseconds")
			
			case getCommittedDataReplyS := <-r.getCommittedDataReplyChan:
				startTime := time.Now().UnixMicro()
				getCommittedDataReply := getCommittedDataReplyS.(*GetCommittedDataReply)
				//got a GetCommittedData message
				// dlog.Printf("Replica %d received GetCommittedDataReply from replica %d\n", r.Id, getCommittedDataReply.SenderId)
				r.handleGetCommittedDataReply(getCommittedDataReply)
				fmt.Println("GetCommittedDataReply from replica", getCommittedDataReply.SenderId, "took", time.Now().UnixMicro() - startTime, "microseconds")
				// fmt.Println("GetCommittedDataReply took", time.Now().UnixMicro() - startTime, "microseconds")

			// case infoBroadcastS := <-r.infoBroadcastChan:
			// 	infoBroadcast := infoBroadcastS.(*InfoBroadcast)
			// 	//got a InfoBroadcast message
			// 	dlog.Printf("Received InfoBroadcast from replica %d, for instance %d\n", infoBroadcast.SenderId, infoBroadcast.Term)
			// 	r.handleInfoBroadcast(infoBroadcast)
			// 	break

			// case infoBroadcastReplyS := <-r.infoBroadcastReplyChan:
			// 	infoBroadcastReply := infoBroadcastReplyS.(*InfoBroadcastReply)
			// 	//got a InfoBroadcastReply message
			// 	dlog.Printf("Received InfoBroadcastReply from replica %d\n", infoBroadcastReply.Term)
			// 	r.handleInfoBroadcastReply(infoBroadcastReply)
			// 	break
			
			case <- r.heartbeatTimer.timer.C:
				// startTime := time.Now().UnixMicro()
				r.heartbeatTimer.active = false
				//got a heartbeat timeout
				r.sendHeartbeat()
				// fmt.Println("Heartbeat took", time.Now().UnixMicro() - startTime, "microseconds")
			
			case <- r.electionTimer.timer.C:
				startTime := time.Now().UnixMicro()
				r.electionTimer.active = false
				//got an election timeout
				r.startElection()
				fmt.Println("Election took", time.Now().UnixMicro() - startTime, "microseconds")
			
			case <- r.benOrStartTimer.timer.C:
				startTime := time.Now().UnixMicro()
				r.benOrStartTimer.active = false
				//got a benOrStartWait timeout
				r.startBenOrPlus()
				fmt.Println("BenOrStartWait took", time.Now().UnixMicro() - startTime, "microseconds")

			case <- r.benOrResendTimer.timer.C:
				startTime := time.Now().UnixMicro()
				// dlog.Println("Replica", r.Id, "got a benOrResend timeout")
				r.benOrResendTimer.active = false
				//got a benOrResend timeout
				r.resendBenOrTimer()
				fmt.Println("BenOrResend took", time.Now().UnixMicro() - startTime, "microseconds")
			
			// case <- r.getCommittedTimer.timer.C:
			// 	//got a getCommitted timeout
			// 	r.clearGetCommittedTimer()
		}
		// times[idx % 10] = time.Now().UnixMicro() - timesss
		// if idx % 10 == 9 {
		// 	var sum int64 = 0
		// 	for i := 0; i < 10; i++ {
		// 		sum += times[i]
		// 	}
		// 	average := sum / 10
		// 	// dlog.Println("Replica", r.Id, "average time for 10 iterations:", average)
		// 	// fmt.Println("Replica", r.Id, "average time for 10 iterations:", average)
		// }
		// idx++
	}

	// dlog.Println("Replica", r.Id, "exiting main loop")

	clearTimer(r.heartbeatTimer)
	clearTimer(r.electionTimer)
	clearTimer(r.benOrStartTimer)
	clearTimer(r.benOrResendTimer)
}

func (r *Replica) sendHeartbeat() {
	timeout := rand.Intn(r.heartbeatTimeout/2) + r.heartbeatTimeout/2
	setTimer(r.heartbeatTimer, time.Duration(timeout)*time.Millisecond)

	if !r.leaderState.isLeader {
		log.Fatal("Replica", r.Id, "is not leader, but sending heartbeat")
	}
	
	// dlog.Printf("Replica %d sending heartbeat\n", r.Id)
	r.broadcastReplicateEntries()
	// else {
	// 	r.broadcastInfo()
	// }
}

func (r *Replica) shutdown() {
	r.Shutdown = true
}

// func (r *Replica) clearGetCommittedTimer() {
// 	clearTimer(r.getCommittedTimer)
// }

func (r *Replica) handleProposeCommand(cmd state.Command) {
	newLogEntry := Entry{
		Data: cmd,
		ServerId: r.Id,
		Term: -1,
		Index: -1,
		Timestamp: time.Now().UnixNano(),
	}
	
	// we only send entries at hearbeats
	if r.leaderState.isLeader {
		newLogEntry.Term = int32(r.term)
		newLogEntry.Index = int32(len(r.log))
		r.inLog.add(newLogEntry)
		r.log = append(r.log, newLogEntry)
		r.leaderState.repNextIndex[r.Id] = len(r.log)
		r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
		// dlog.Println("I HATE THIS", len(r.log))

		// r.sendHeartbeat()
	} else {
		r.pq.push(newLogEntry)
		r.sendGetCommittedData()
		// dlog.Println(newLogEntry)
		// dlog.Println(r.pq.extractList())
		// dlog.Println(r.Id, "ADDED TO PQ", logToString(r.pq.extractList()))

	}
}

func (r *Replica) handlePropose(propose *genericsmr.Propose) {
	// r.proposals[propose.Command.ClientId] = propose
	r.handleProposeCommand(propose.Command)
}

// func (r *Replica) getUpToDateData () {
// 	if r.getCommittedTimer.active {
// 		return
// 	}
// 	timeout := rand.Intn(r.getCommittedTimeout/2) + r.getCommittedTimeout/2
// 	setTimer(r.getCommittedTimer, time.Duration(timeout)*time.Millisecond)

// 	args := &GetCommittedData{
// 		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.LeaderTerm), LogLength: int32(len(r.log)),
// 	}
// 	for i := 0; i < r.N; i++ {
// 		if int32(i) != r.Id {
// 			r.SendMsg(int32(i), r.getCommittedDataRPC, args)
// 		}
// 	}
// }

func (r *Replica) handleGetState(getState *genericsmr.GetState) {
	args := &genericsmrproto.GetStateReply{
		IsLeader: convertBoolToInteger(r.leaderState.isLeader),
	}
	r.ReplyGetState(args, getState.Reply)
}

func (r *Replica) handleSlowdown(slowdown *genericsmr.Slowdown) {
	args := &genericsmrproto.SlowdownReply{
		Success: 1,
	}
	r.ReplySlowdown(args, slowdown.Reply)
	time.Sleep(time.Duration(slowdown.TimeInMs) * time.Millisecond)
}

func (r *Replica) handleConnect(connect *genericsmr.Connect) {
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.Connected[i] = true
		}
	}

	args := &genericsmrproto.ConnectReply{
		Success: 1,
	}
	r.ReplyConnect(args, connect.Reply)
}

func (r *Replica) handleDisconnect(disconnect *genericsmr.Disconnect) {
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.Connected[i] = false
		}
	}
	args := &genericsmrproto.DisconnectReply{
		Success: 1,
	}
	r.ReplyDisconnect(args, disconnect.Reply)
}

func (r *Replica) shouldLogBeReplaced(rpc UpdateMsg, logStartIdx int) (bool, int) { // should you replace, and if you should, the first index to start replacing
	firstEntryIndex := int(rpc.GetStartIndex())

	for i := logStartIdx; i <= int(rpc.GetCommitIndex()); i++ {
		if i >= len(r.log) || !entryEqual(r.log[i], rpc.GetEntries()[i - firstEntryIndex]) { return true, i }
	}

	if r.leaderTerm > int(rpc.GetLeaderTerm()) || (r.leaderTerm == int(rpc.GetLeaderTerm()) && len(r.log) >= int(rpc.GetLogLength())) {
		return false, -1
	}

	for i := max(logStartIdx, int(rpc.GetCommitIndex()) + 1); i < int(firstEntryIndex) + len(rpc.GetEntries()); i++ {
		if i >= len(r.log) || !entryEqual(r.log[i], rpc.GetEntries()[i - firstEntryIndex]) { return true, i }
	}

	// you have a lower log term, but the entries are the same
	return false, -1
}

func (r *Replica) replaceExistingLog (rpc UpdateMsg, logStartIdx int) []Entry {
	// dlog.Println("Replica", r.Id, "replacing existing log", len(r.log), rpc.GetStartIndex(), len(rpc.GetEntries()), logStartIdx)
	firstEntryIndex := int(rpc.GetStartIndex())

	potentialEntries := make([]Entry, 0)
	for i := logStartIdx; i < len(r.log); i++ {
		r.inLog.remove(r.log[i])
		potentialEntries = append(potentialEntries, r.log[i])
	}
	r.log = r.log[:logStartIdx]

	for i := logStartIdx - firstEntryIndex; i < len(rpc.GetEntries()); i++ {
		r.inLog.add(rpc.GetEntries()[i])
		r.log = append(r.log, rpc.GetEntries()[i])
		r.pq.remove(rpc.GetEntries()[i])
	}

	if r.leaderState.isLeader {
		r.leaderState.repNextIndex[r.Id] = len(r.log)
		r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
	}

	return potentialEntries
}

func (r *Replica) updateLogFromRPC (rpc ReplyMsg) bool {
	// timeStart := time.Now().UnixMicro()
	oldCommitIndex := r.commitIndex

	var potentialEntries []Entry
	shouldReplaceLog, startReplacementIdx := r.shouldLogBeReplaced(rpc, r.commitIndex + 1)
	if shouldReplaceLog {
		potentialEntries = r.replaceExistingLog(rpc, startReplacementIdx)
	} else {
		potentialEntries = rpc.GetEntries()
	}
	
	r.commitIndex = int(rpc.GetCommitIndex())
	r.leaderTerm = max(r.leaderTerm, int(rpc.GetLeaderTerm()))
	if r.commitIndex > oldCommitIndex {
		// dlog.Println("Replica", r.Id, "committing to", r.commitIndex, "from", oldCommitIndex)
		// if !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
		// 	r.pq.push(r.benOrState.benOrBroadcastEntry)
		// }
		r.benOrState = emptyBenOrState

		if !r.leaderState.isLeader {
			timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
			setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
		}

		clearTimer(r.benOrResendTimer)
	}

	for _, v := range(potentialEntries) {
		if !r.seenBefore(v) { r.pq.push(v) }
	}

	for _, v := range(rpc.GetPQEntries()) {
		if !r.seenBefore(v) { r.pq.push(v) }
	}

	// dlog.Println("Replica", r.Id, "pq values5", logToString(r.pq.extractList()))

	if r.leaderState.isLeader {
		// the following case actually shouldn't be an error because it's possible in a unique case
		// ex: consider 5 replicas. Replica 0 is the leader, and it replicates 5 entries on all replicas, but only replica 1 knows the new commitIndex before replica 0 goes down.
		// Now, replica 2 is the new leader (receiving votes from replicas 3, and 4), and sends a replicate entries to replica 1.
		// Replica 1 has a higher commitIndex!
		// if rpc.GetStartIndex() + int32(len(rpc.GetEntries())) > rpc.GetCommitIndex() {
		// 	log.Fatal("Leader", r.Id, "got an RPC with a commit index that is less than the last entry in the RPC from", rpc.GetSenderId())
		// }

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

		if shouldReplaceLog {
			r.leaderState.repNextIndex[rpc.GetSenderId()] = int(rpc.GetLogLength())
			r.leaderState.repMatchIndex[rpc.GetSenderId()] = int(rpc.GetLogLength()) - 1

			for i := 0; i < r.N; i++ {
				if i != int(r.Id) || i != int(rpc.GetSenderId()) {
					r.leaderState.repNextIndex[i] = min(r.leaderState.repNextIndex[i], startReplacementIdx)
					r.leaderState.repMatchIndex[i] = min(r.leaderState.repMatchIndex[i], startReplacementIdx - 1)
				}
			}
		}
	}

	// fmt.Println("updateLogFromRPC took", time.Now().UnixMicro() - timeStart, "microseconds")
	return shouldReplaceLog
}

// func (r *Replica) broadcastInfo() {
// 	for i := 0; i < r.N; i++ {
// 		if int32(i) != r.Id {
// 			args := &InfoBroadcast{
// 				SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.LeaderTerm), PQEntries: r.pq.extractList()}

// 			r.SendMsg(int32(i), r.infoBroadcastRPC, args)
// 		}
// 	}
// }

// func (r *Replica) handleInfoBroadcast(rpc *InfoBroadcast) {
// 	if r.term < int(rpc.Term) {
// 		r.term = int(rpc.Term)
// 		if r.leaderState.isLeader {
// 			r.leaderState = LeaderState{
// 				isLeader: false,
// 				repNextIndex: make([]int, r.N),
// 				repMatchIndex: make([]int, r.N),
// 				lastRepEntriesTimestamp: 0,
// 			}
// 			clearTimer(r.heartbeatTimer)
// 		}
// 		if r.candidateState.isCandidate {
// 			r.candidateState = CandidateState{
// 				isCandidate: false,
// 				votesReceived: 0,
// 			}
// 		}
// 	}

// 	for _, v := range(rpc.PQEntries) {
// 		if !r.seenBefore(v) {
// 			if r.leaderState.isLeader {
// 				r.inLog.add(v)
// 				r.log = append(r.log, v)
// 			} else {
// 				r.pq.push(v)
// 			}
// 		}
// 	}


// 	r.handleIncomingRPCTerm(int(rpc.Term))

// 	uniq := UniqueCommand{senderId: rpc.ClientReq.SenderId, time: rpc.ClientReq.Timestamp}
// 	r.inLog.add(uniq)

// 	r.addNewEntry(rpc.ClientReq)

// 	args := &InfoBroadcastReply{
// 		SenderId: r.Id, Term: int32(r.term)}
// 	r.SendMsg(rpc.SenderId, r.infoBroadcastRPC, args)
// 	return
// }

// func (r *Replica) handleInfoBroadcastReply (rpc *InfoBroadcastReply) {
// 	r.handleIncomingRPCTerm(int(rpc.Term))
// 	return
// }

func (r *Replica) sendGetCommittedData() {
	// dlog.Println("Replica", r.Id, "sending get committed data", r.commitIndex, len(r.log))

	entries := make([]Entry, 0)
	if r.commitIndex + 1 < len(r.log) {
		entries = r.log[r.commitIndex + 1:]
	}

	args := &GetCommittedData{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
		StartIndex: int32(r.commitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
	}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.getCommittedDataRPC, args)
		}
	}
}

func (r *Replica) handleGetCommittedData(rpc *GetCommittedData) {
	r.handleIncomingTerm(rpc)

	// only respond if we have a higher commit index
	// if r.isLogMoreUpToDate(rpc) != MoreUpToDate {
	// 	return
	// }

	var moreUpToDate bool
	if r.isLogMoreUpToDate(rpc) != MoreUpToDate {
		// fmt.Println("huh", r.Id, r.commitIndex, r.leaderTerm, len(r.log), rpc.CommitIndex, rpc.LeaderTerm, rpc.LogLength, rpc.StartIndex, len(rpc.Entries))
		if r.commitIndex + 1 >= int(rpc.GetStartIndex()) {
			r.updateLogFromRPC(rpc)
		}
		moreUpToDate = false
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
		}
		moreUpToDate = true
	}

	if moreUpToDate {
		entries := make([]Entry, 0)
		if rpc.CommitIndex + 1 < int32(len(r.log)) {
			entries = r.log[rpc.CommitIndex + 1:]
		}

		args := &GetCommittedDataReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			StartIndex: rpc.CommitIndex + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		// dlog.Println("Replica", r.Id, "sending get committed data reply to", rpc.SenderId, "with", r.commitIndex, len(r.log))
		r.SendMsg(rpc.SenderId, r.getCommittedDataReplyRPC, args)
	}

	// if r.leaderState.isLeader {
	// 	r.sendHeartbeat()
	// }
}

func (r *Replica) handleGetCommittedDataReply(rpc *GetCommittedDataReply) {
	r.handleIncomingTerm(rpc)

	if r.isLogMoreUpToDate(rpc) == LessUpToDate {
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
		}
	}
}

// // if something actually changed!
// if updated {
// 	for i := int(rpc.PreparedIndex)+1; i < len(r.log); i++ {
// 		r.inLog.remove(UniqueCommand{senderId: r.log[i].SenderId, time: r.log[i].Timestamp})
// 		// removedEntries = append(removedEntries, r.log[i])
// 		// if !r.seenBefore(r.log[i]) {
// 		// 	r.pq.push(r.log[i])
// 		// }
// 		potentialEntries = append(potentialEntries, r.log[i])
// 	}
// 	r.log = r.log[:rpc.PreparedIndex+1]

// 	for i := int(rpc.PreparedIndex)+1; i < len(rpc.Entries) + firstEntryIndex; i++ {
// 		r.inLog.add(UniqueCommand{senderId: rpc.Entries[i-firstEntryIndex].SenderId, time: rpc.Entries[i-firstEntryIndex].Timestamp})
// 	}
// 	r.log = append(r.log, rpc.Entries[int(rpc.PreparedIndex)+1-firstEntryIndex:]...)
// } else {
// 	for i := int(rpc.PreparedIndex)+1; i < len(rpc.Entries) + firstEntryIndex; i++ {
// 		// if !r.seenBefore(rpc.Entries[i-firstEntryIndex]) {
// 		// 	r.pq.push(rpc.Entries[i-firstEntryIndex])
// 		// }
// 		potentialEntries = append(potentialEntries, rpc.Entries[i-firstEntryIndex])
// 	}
// }

// TODO: if updated, append their log to ours
// update ben or index based on new vs old commit index

// if i == logLength && i - firstEntryIndex < len(rpc.Entries) {
// 	r.log = append(r.log, rpc.Entries[i-firstEntryIndex:]...)
// }

// if benOrIndexChanged {
// 	r.benOrIndex = newCommitPoint+1
// 	r.benOrState.benOrStage = Stopped
// 	r.benOrState.biasedCoin = false
// 	if (r.benOrIndex < len(r.log)) {
// 		r.log[r.benOrIndex].BenOrActive = True
// 	} else {
// 		r.log[r.benOrIndex] = benOrUncommittedLogEntry(len(r.log))
// 	}
// }

// for i := currentCommitPoint+1; i <= len(r.log); i++ {
// 	r.pq.remove(r.log[i])
// }

// can only have one entry with term == -1 after commit point
// if r.log[i].Term != rpc.Entries[i-firstEntryIndex].Term {
// 	if r.benOrIndex == i {
// 		if r.log[i].Term < rpc.Entries[i-firstEntryIndex].Term {
// 			removedEntries = append(removedEntries, r.log[i])
// 			r.log[i] = rpc.Entries[i-firstEntryIndex]
// 		}

// 		// if BenOrActive is false, then this entry has for sure been committed already since
// 		// rpc is sending back data only up to replicatedIndex
// 		if rpc.Entries[i-firstEntryIndex].Term != -1 && i <= newPreparedPoint {
// 			benOrIndexChanged = true
// 			continue
// 		}

// 		if r.benOrState.benOrStage == Stopped {
// 			// don't need to do anything else
// 		} else if r.benOrState.benOrStage == Broadcasting {
// 			r.benOrState.biasedCoin = true
// 		} else { // r.benOrStatus == BenOrRunning
// 			continue // Ben Or needs to keep running, although it will commit the value we've just added
// 		}
// 	} else if rpc.Entries[i-firstEntryIndex].BenOrActive == True {
// 		// use current entry instead
// 		continue
// 	} else {
// 		r.log = append(r.log[:i], rpc.Entries[i-firstEntryIndex:]...)
// 		break
// 	}
// }