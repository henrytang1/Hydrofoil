package randomizedpaxos

import (
	"bufio"
	"dlog"
	"encoding/binary"
	"fastrpc"
	"genericsmr"
	"genericsmrproto"
	"io"
	"math/rand"
	"randomizedpaxosproto"
	"sort"
	"state"
	"time"
)

type Entry = randomizedpaxosproto.Entry
type ReplicateEntries = randomizedpaxosproto.ReplicateEntries
type ReplicateEntriesReply = randomizedpaxosproto.ReplicateEntriesReply
type RequestVote = randomizedpaxosproto.RequestVote
type RequestVoteReply = randomizedpaxosproto.RequestVoteReply
type BenOrBroadcast = randomizedpaxosproto.BenOrBroadcast
type BenOrBroadcastReply = randomizedpaxosproto.BenOrBroadcastReply
type BenOrConsensus = randomizedpaxosproto.BenOrConsensus
type BenOrConsensusReply = randomizedpaxosproto.BenOrConsensusReply
type GetCommittedData = randomizedpaxosproto.GetCommittedData
type GetCommittedDataReply = randomizedpaxosproto.GetCommittedDataReply
type InfoBroadcast = randomizedpaxosproto.InfoBroadcast
type InfoBroadcastReply = randomizedpaxosproto.InfoBroadcastReply

type BenOrBroadcastMsg interface {
	GetSenderId() int32
	GetTerm() int32
	GetIndex() int32
	GetIteration() int32
	GetClientReq() Entry
}

type BenOrConsensusMsg interface {
	GetSenderId() int32
	GetTerm() int32
	GetIndex() int32
	GetIteration() int32
	GetPhase() int32
	GetVote() int32
	GetMajRequest() Entry
	GetLeaderRequest() Entry
	GetStage() int32
}

const (
	False uint8 = 0
	True        = 1
)

const ( // for benOrStatus
	Broadcasting uint8 = 1
	StageOne	   = 2
	StageTwo	   = 3
	Stopped 	   = 4 // just means that Ben Or isn't running at this moment
	// WaitingToBeUpdated
)

const (
	Vote0 uint8 	   = 1
	Vote1		   = 2
	VoteQuestionMark   = 3
)

const INJECT_SLOWDOWN = false

const CHAN_BUFFER_SIZE = 200000

const MAX_BATCH = 5000
const BATCH_INTERVAL = 100 * time.Microsecond

// const TRUE = uint8(1)
// const FALSE = uint8(0)

// type RPCCounter struct {
// 	term	int
// 	count	int
// }

type Set struct {
	m map[UniqueCommand]bool
}

func newSet() Set {
	return Set{make(map[UniqueCommand]bool)}
}

func (s *Set) add(item UniqueCommand) {
	s.m[item] = false
}

func (s *Set) remove(item UniqueCommand) {
	delete(s.m, item)
}

func (s *Set) contains(item UniqueCommand) bool {
	_, ok := s.m[item]
	return ok
}

func (s *Set) commit(item UniqueCommand) {
	s.m[item] = true
}

func (s *Set) isCommitted(item UniqueCommand) bool {
	if _, ok := s.m[item]; !ok {
		return false
	}
	return s.m[item]
}

func (s *Set) commitSlice(items []UniqueCommand) {
	for _, item := range items {
		s.commit(item)
	}
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
	infoBroadcastChan		chan fastrpc.Serializable
	infoBroadcastReplyChan		chan fastrpc.Serializable
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
	infoBroadcastRPC      		uint8
	infoBroadcastReplyRPC     	uint8

	// used to ignore entries in the past
	// replicateEntriesCounter			rpcCounter
	// requestVoteCounter				rpcCounter // not necessary since the term already determines a unique entry
	// benOrBroadcastCounter			rpcCounter
	// benOrConsensusCounter			rpcCounter
	// infoBroadcastCounter				rpcCounter

	isLeader			bool
	electionTimeout         	int
	heartbeatTimeout         	int
	benOrStartWaitTimeout		int
	benOrResendTimeout		int
	currentTerm			int
	log				[]Entry
	pq				ExtendedPriorityQueue // to be fixed
	seenEntries			Set

	benOrState			BenOrState
	benOrIndex			int
	preparedIndex			int // length of the log that has been prepared except for at most 1 entry that's still running benOr
	lastApplied			int
	entries				[]Entry
	nextIndex			[]int
	matchIndex			[]int // highest known prepared index for each replica
	commitIndex			[]int // highest known commit index for each replica
	currentTimer			time.Time
	highestTimestamp		[]int64 // highest timestamp seen from each replica (used to ignore old requests)
	votesReceived			int
	votedFor			int
	requestVoteEntries		[]Entry // only stores entries that we're not sure if they've already been committed when requesting a vote
	// requestVoteBenOrIndex		int
	// requestVotePreparedIndex		int

	clientWriters      		map[uint32]*bufio.Writer
	heartbeatTimer			*time.Timer
	electionTimer			*time.Timer
	benOrStartWaitTimer		*time.Timer
	benOrResendTimer		*time.Timer
}


type Instance struct {
	cmds   []state.Command
	ballot int32
}

type Pair struct {
	Idx  int
	Term int
}

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

// Entry point
func NewReplica(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool, durable bool, isProduction bool) *Replica {
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
		infoBroadcastChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
		infoBroadcastReplyChan: make(chan fastrpc.Serializable, genericsmr.CHAN_BUFFER_SIZE),
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
		infoBroadcastRPC: 0,
		infoBroadcastReplyRPC: 0,
		// replicateEntriesCounter: rpcCounter{0,0},
		// requestVoteCounter: rpcCounter{0,0},
		// benOrBroadcastCounter: rpcCounter{0,0},
		// benOrConsensusCounter: rpcCounter{0,0},
		// infoBroadcastCounter: rpcCounter{0,0},
		isLeader: false,
		electionTimeout: 0, // TODO: NEED TO UPDATE THESE VALUES
		heartbeatTimeout: 0,
		benOrStartWaitTimeout: 0,
		benOrResendTimeout: 0,
		currentTerm: 0,
		log: make([]Entry, 0),
		pq: newExtendedPriorityQueue(),
		seenEntries: newSet(),
		benOrState: BenOrState{
			benOrStage: Stopped,
			benOrIteration: 0,
			benOrPhase: 0,

			benOrVote: 0,
			benOrMajRequest: benOrUncommittedLogEntry(-1),
			benOrBroadcastRequest: benOrUncommittedLogEntry(-1),
			benOrRepliesReceived: 0,
			// benOrStage: StageOne,
			benOrBroadcastMessages: make([]BenOrBroadcastMsg, 0),
			benOrConsensusMessages: make([]BenOrConsensusMsg, 0),
			biasedCoin: false,
		},
		benOrIndex: 1,
		preparedIndex: 0,
		lastApplied: 0, // 0 entry is already applied (it's an empty entry)
		// first entry is a default useless entry (to make edge cases easier)
		// second entry is the initial BenOr entry (but BenOr is not started yet)
		entries: make([]Entry, 2),
		nextIndex: make([]int, 0),
		matchIndex: make([]int, 0),
		commitIndex: make([]int, 0),
		currentTimer: time.Now(),
		highestTimestamp: make([]int64, 0),
		votesReceived: 0,
		votedFor: -1,
		requestVoteEntries: make([]Entry, 0),
		clientWriters: make(map[uint32]*bufio.Writer),
		heartbeatTimer: time.NewTimer(0),
		electionTimer: time.NewTimer(0),
		benOrStartWaitTimer: time.NewTimer(0),
		benOrResendTimer: time.NewTimer(0),
	}

	// initialize these timers, since otherwise calling timer.Reset() on them will panic
	<-r.heartbeatTimer.C
	<-r.electionTimer.C
	<-r.benOrStartWaitTimer.C
	<-r.benOrResendTimer.C

	r.Durable = durable
	r.TestingState.IsProduction = isProduction

	r.entries[0] = Entry{
		Data: state.Command{},
		SenderId: -1,
		Term: -1,
		Index: 0,
		BenOrActive: False,
		Timestamp: -1,
		FromLeader: False, // this initial entry shouldn't matter
	}

	r.entries[1] = Entry{
		Data: state.Command{},
		SenderId: -1,
		Term: -1,
		Index: 1,
		BenOrActive: True,
		Timestamp: -1,
		FromLeader: False,
	}

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
	r.infoBroadcastRPC = r.RegisterRPC(new(InfoBroadcast), r.infoBroadcastChan)
	r.infoBroadcastReplyRPC = r.RegisterRPC(new(InfoBroadcastReply), r.infoBroadcastReplyChan)

	// go r.run()

	return r
}

// func (r *Replica) replyReplicateEntries(replicaId int32, reply *ReplicateEntriesReply) {
// 	r.SendMsg(replicaId, r.replicateEntriesReplyRPC, reply)
// }

// func (r *Replica) replyRequestVote(replicaId int32, reply *RequestVoteReply) {
// 	r.SendMsg(replicaId, r.requestVoteReplyRPC, reply)
// }

// func (r *Replica) replyBenOrBroadcast(replicaId int32, reply *BenOrBroadcastReply) {
// 	r.SendMsg(replicaId, r.benOrBroadcastReplyRPC, reply)
// }

// func (r *Replica) replyBenOrConsensus(replicaId int32, reply *BenOrConsensusReply) {
// 	r.SendMsg(replicaId, r.benOrConsensusReplyRPC, reply)
// }

// func (r *Replica) replyInfoBroadcast(replicaId int32, reply *InfoBroadcastReply) {
// 	r.SendMsg(replicaId, r.infoBroadcastReplyRPC, reply)
// }

var clockChan chan bool

func (r *Replica) clock() {
	for !r.Shutdown {
		time.Sleep(BATCH_INTERVAL)
		clockChan <- true
	}
}

func (r *Replica) runReplica() {
	go r.runInNewProcess()
}

/* Main event processing loop */
func (r *Replica) runInNewProcess() {
	if (r.TestingState.IsProduction) { 
		r.ConnectToPeers()

		dlog.Println("Waiting for client connections")

		go r.WaitForClientConnections() // TODO: remove for testing 
	}

	// if r.Exec {
	// 	go r.executeCommands()
	// }

	// if r.Id == 0 {
	// 	r.IsLeader = true
	// }

	clockChan = make(chan bool, 1)
	go r.clock()

	onOffProposeChan := r.ProposeChan

	for !r.Shutdown {
		if r.isLeader {
			commitIndices := -1
			matchIndices := make([]Pair, 0)

			for i := 0; i < r.N; i++ {
				idx := r.matchIndex[i]
				term := 0
				if idx >= 0 {
					term = int(r.log[idx].Term)
				}

				commitIndices = min(commitIndices, r.nextIndex[i]-1)
				matchIndices = append(matchIndices, Pair{
					Idx:  idx,
					Term: term,
				})
			}
			sort.Slice(matchIndices, func(i, j int) bool {
				return matchIndices[i].Idx > matchIndices[j].Idx
			})

			// TODO: figure out if this execution procedure actually makes any sense and how to capture it
			// Execution of the leader's state machine
			for i := r.lastApplied + 1; i <= commitIndices; i++ {
				if writer, ok := r.clientWriters[r.log[i].Data.ClientId]; ok {
					val := r.log[i].Data.Execute(r.State)
					propreply := &genericsmrproto.ProposeReplyTS{
						OK: True,
						CommandId: r.log[i].Data.OpId,
						Value: val,
						Timestamp: r.log[i].Timestamp} // TODO: check if timestamp is correct
					r.ReplyProposeTS(propreply, writer)
				}
			}

			// Update preparedIndex
			r.preparedIndex = matchIndices[r.N/2].Idx
		}

		select {
			case client := <-r.RegisterClientIdChan:
				r.registerClient(client.ClientId, client.Reply)
				break

			case <-clockChan:
				//activate the new proposals channel
				onOffProposeChan = r.ProposeChan
				break

			case propose := <-onOffProposeChan:
				//got a Propose from a client
				dlog.Printf("Proposal with op %d\n", propose.Command.Op)
				r.handlePropose(propose)
				//deactivate the new proposals channel to prioritize the handling of protocol messages
				if MAX_BATCH > 100 {
					onOffProposeChan = nil
				}
				break

			case replicateEntriesS := <-r.replicateEntriesChan:
				replicateEntries := replicateEntriesS.(*ReplicateEntries)
				//got a ReplicateEntries message
				dlog.Printf("Received ReplicateEntries from replica %d, for instance %d\n", replicateEntries.SenderId, replicateEntries.Term)
				r.handleReplicateEntries(replicateEntries)
				break

			case replicateEntriesReplyS := <-r.replicateEntriesReplyChan:
				replicateEntriesReply := replicateEntriesReplyS.(*ReplicateEntriesReply)
				//got a ReplicateEntriesReply message
				dlog.Printf("Received ReplicateEntriesReply from replica %d\n", replicateEntriesReply.Term)
				r.handleReplicateEntriesReply(replicateEntriesReply)
				break

			case requestVoteS := <-r.requestVoteChan:
				requestVote := requestVoteS.(*RequestVote)
				//got a RequestVote message
				dlog.Printf("Received RequestVote from replica %d, for instance %d\n", requestVote.SenderId, requestVote.Term)
				r.handleRequestVote(requestVote)
				break

			case requestVoteReplyS := <-r.requestVoteReplyChan:
				requestVoteReply := requestVoteReplyS.(*RequestVoteReply)
				//got a RequestVoteReply message
				dlog.Printf("Received RequestVoteReply from replica %d\n", requestVoteReply.Term)
				r.handleRequestVoteReply(requestVoteReply)
				break

			case benOrBroadcastS := <-r.benOrBroadcastChan:
				benOrBroadcast := benOrBroadcastS.(*BenOrBroadcast)
				//got a BenOrBroadcast message
				dlog.Printf("Received BenOrBroadcast from replica %d, for instance %d\n", benOrBroadcast.SenderId, benOrBroadcast.Term)
				r.handleBenOrBroadcast(benOrBroadcast)
				break

			case benOrBroadcastReplyS := <-r.benOrBroadcastReplyChan:
				benOrBroadcastReply := benOrBroadcastReplyS.(*BenOrBroadcastReply)
				//got a BenOrBroadcastReply message
				dlog.Printf("Received BenOrBroadcastReply from replica %d\n", benOrBroadcastReply.Term)
				r.handleBenOrBroadcast(benOrBroadcastReply)
				break

			case benOrConsensusS := <-r.benOrConsensusChan:
				benOrConsensus := benOrConsensusS.(*BenOrConsensus)
				//got a BenOrConsensus message
				dlog.Printf("Received BenOrConsensus from replica %d, for instance %d\n", benOrConsensus.SenderId, benOrConsensus.Term)
				r.handleBenOrConsensus(benOrConsensus)
				break

			case benOrConsensusReplyS := <-r.benOrConsensusReplyChan:
				benOrConsensusReply := benOrConsensusReplyS.(*BenOrConsensusReply)
				//got a BenOrConsensusReply message
				dlog.Printf("Received BenOrConsensusReply from replica %d\n", benOrConsensusReply.Term)
				r.handleBenOrConsensus(benOrConsensusReply)
				break

			case infoBroadcastS := <-r.infoBroadcastChan:
				infoBroadcast := infoBroadcastS.(*InfoBroadcast)
				//got a InfoBroadcast message
				dlog.Printf("Received InfoBroadcast from replica %d, for instance %d\n", infoBroadcast.SenderId, infoBroadcast.Term)
				r.handleInfoBroadcast(infoBroadcast)
				break

			case infoBroadcastReplyS := <-r.infoBroadcastReplyChan:
				infoBroadcastReply := infoBroadcastReplyS.(*InfoBroadcastReply)
				//got a InfoBroadcastReply message
				dlog.Printf("Received InfoBroadcastReply from replica %d\n", infoBroadcastReply.Term)
				r.handleInfoBroadcastReply(infoBroadcastReply)
				break
			
			case <- r.heartbeatTimer.C:
				//got a heartbeat timeout
				r.sendHeartbeat()
			
			case <- r.electionTimer.C:
				//got an election timeout
				r.startElection()
			
			case <- r.benOrStartWaitTimer.C:
				//got a benOrStartWait timeout
				r.startBenOrPlus()

			case <- r.benOrResendTimer.C:
				r.resendBenOrTimer()
		}
	}
}

// Manage Client Writers
func (r *Replica) registerClient(clientId uint32, writer *bufio.Writer) uint8 {
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

func (r *Replica) sendHeartbeat () {
	timeout := rand.Intn(r.heartbeatTimeout/2) + r.heartbeatTimeout/2
	setTimer(r.heartbeatTimer, time.Duration(timeout)*time.Millisecond)
	r.bcastReplicateEntries()
}

func (r *Replica) bcastInfoBroadcast(clientReq Entry) {
	// r.infoBroadcastCounter = rpcCounter{r.currentTerm, r.infoBroadcastCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			args := &InfoBroadcast{
				SenderId: r.Id, Term: int32(r.currentTerm), ClientReq: clientReq}

			r.SendMsg(int32(i), r.infoBroadcastRPC, args)
		}
	}
}

func (r *Replica) handlePropose(propose *genericsmr.Propose) {
	newLogEntry := Entry{
		Data: propose.Command,
		SenderId: r.Id,
		// Term: int32(r.currentTerm),
		// Index: int32(len(r.log)),
		Term: -1,
		Index: -1,
		Timestamp: r.currentTimer.UnixNano(),
		FromLeader: False,
	}
	
	uniq := UniqueCommand{senderId: r.Id, time: newLogEntry.Timestamp}
	r.seenEntries.add(uniq)

	r.addNewEntry(newLogEntry)

	if r.isLeader {
		r.bcastReplicateEntries()
	} else {
		r.bcastInfoBroadcast(newLogEntry)
	}
}

func (r *Replica) handleInfoBroadcast(rpc *InfoBroadcast) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
	}

	uniq := UniqueCommand{senderId: rpc.ClientReq.SenderId, time: rpc.ClientReq.Timestamp}
	r.seenEntries.add(uniq)

	args := &InfoBroadcastReply{
		SenderId: r.Id, Term: int32(r.currentTerm)}
	r.SendMsg(rpc.SenderId, r.infoBroadcastRPC, args)

	r.addNewEntry(rpc.ClientReq)
	return
}

func (r *Replica) handleInfoBroadcastReply (rpc *InfoBroadcastReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)

		if (r.isLeader) {
			clearTimer(r.heartbeatTimer)
			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		r.isLeader = false
		r.votesReceived = 0
	}
	return
}