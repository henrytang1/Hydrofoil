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

const (
	Zero int = iota
	One
	Unknown
)

const (
	Broadcasting uint8 = iota
	BenOrRunning
	Stopped
)

const INJECT_SLOWDOWN = false

const CHAN_BUFFER_SIZE = 200000

const MAX_BATCH = 5000
const BATCH_INTERVAL = 100 * time.Microsecond

const TRUE = uint8(1)
const FALSE = uint8(0)

type rpcCounter struct {
	term	int
	count	int
}

type Replica struct {
	*genericsmr.Replica // extends a generic Paxos replica
	replicateEntriesChan         	chan fastrpc.Serializable
	replicateEntriesReplyChan		chan fastrpc.Serializable
	requestVoteChan		          	chan fastrpc.Serializable
	requestVoteReplyChan     		chan fastrpc.Serializable
	benOrBroadcastChan    			chan fastrpc.Serializable
	benOrBroadcastReplyChan    		chan fastrpc.Serializable
	benOrConsensusChan    			chan fastrpc.Serializable
	benOrConsensusReplyChan   		chan fastrpc.Serializable
	infoBroadcastChan				chan fastrpc.Serializable
	infoBroadcastReplyChan			chan fastrpc.Serializable
	replicateEntriesRPC          	uint8
	replicateEntriesReplyRPC     	uint8
	requestVoteRPC           		uint8
	requestVoteReplyRPC      		uint8
	benOrBroadcastRPC           	uint8
	benOrBroadcastReplyRPC      	uint8
	benOrConsensusRPC      			uint8
	benOrConsensusReplyRPC     		uint8
	infoBroadcastRPC      			uint8
	infoBroadcastReplyRPC     		uint8

	// used to ignore entries in the past
	replicateEntriesCounter			rpcCounter
	requestVoteCounter				rpcCounter // not necessary since the term already determines a unique entry
	benOrBroadcastCounter			rpcCounter
	benOrConsensusCounter			rpcCounter
	infoBroadcastCounter			rpcCounter

	isLeader						bool
	electionTimeout         		int
	heartbeatTimeout         		int
	benOrStartWaitTimeout			int
	currentTerm						int
	log								[]randomizedpaxosproto.Entry
	pq								ExtendedPriorityQueue // to be fixed
	benOrStatus						uint8
	benOrIndex						int
	biasedCoin						bool
	preparedIndex					int // length of the log that has been prepared except for at most 1 entry that's still running benOr
	lastApplied						int
	entries							[]randomizedpaxosproto.Entry
	nextIndex						[]int
	matchIndex						[]int // highest known prepared index for each replica
	commitIndex						[]int // highest known commit index for each replica
	currentTimer					time.Time
	highestTimestamp				[]int64 // highest timestamp seen from each replica (used to ignore old requests)
	votesReceived					int
	votedFor						int
	requestVoteEntries				[]randomizedpaxosproto.Entry // only stores entries that we're not sure if they've already been committed when requesting a vote
	// requestVoteBenOrIndex			int
	// requestVotePreparedIndex		int

	clientWriters      				map[uint32]*bufio.Writer
	heartbeatTimer					*time.Timer
	electionTimer					*time.Timer
	benOrStartWaitTimer				*time.Timer
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

func NewReplica(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool, durable bool) *Replica {
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
		infoBroadcastRPC: 0,
		infoBroadcastReplyRPC: 0,
		replicateEntriesCounter: rpcCounter{0,0},
		requestVoteCounter: rpcCounter{0,0},
		benOrBroadcastCounter: rpcCounter{0,0},
		benOrConsensusCounter: rpcCounter{0,0},
		infoBroadcastCounter: rpcCounter{0,0},
		isLeader: false,
		electionTimeout: 0,
		heartbeatTimeout: 0,
		benOrStartWaitTimeout: 0,
		currentTerm: 0,
		log: make([]randomizedpaxosproto.Entry, 0),
		pq: newExtendedPriorityQueue(),
		benOrStatus: Stopped,
		benOrIndex: 0,
		biasedCoin: false,
		preparedIndex: 0,
		lastApplied: -1,
		entries: make([]randomizedpaxosproto.Entry, 0),
		nextIndex: make([]int, 0),
		matchIndex: make([]int, 0),
		commitIndex: make([]int, 0),
		currentTimer: time.Now(),
		highestTimestamp: make([]int64, 0),
		votesReceived: 0,
		votedFor: -1,
		requestVoteEntries: make([]randomizedpaxosproto.Entry, 0),
		clientWriters: make(map[uint32]*bufio.Writer),
		heartbeatTimer: nil,
		electionTimer: nil,
		benOrStartWaitTimer: nil}

	r.Durable = durable

	r.replicateEntriesRPC = r.RegisterRPC(new(randomizedpaxosproto.ReplicateEntries), r.replicateEntriesChan)
	r.replicateEntriesReplyRPC = r.RegisterRPC(new(randomizedpaxosproto.ReplicateEntriesReply), r.replicateEntriesReplyChan)
	r.requestVoteRPC = r.RegisterRPC(new(randomizedpaxosproto.RequestVote), r.requestVoteChan)
	r.requestVoteReplyRPC = r.RegisterRPC(new(randomizedpaxosproto.RequestVoteReply), r.requestVoteReplyChan)
	r.benOrBroadcastRPC = r.RegisterRPC(new(randomizedpaxosproto.BenOrBroadcast), r.benOrBroadcastChan)
	r.benOrBroadcastReplyRPC = r.RegisterRPC(new(randomizedpaxosproto.BenOrBroadcastReply), r.benOrBroadcastReplyChan)
	r.benOrConsensusRPC = r.RegisterRPC(new(randomizedpaxosproto.BenOrConsensus), r.benOrConsensusChan)
	r.benOrConsensusReplyRPC = r.RegisterRPC(new(randomizedpaxosproto.BenOrConsensusReply), r.benOrConsensusReplyChan)
	r.infoBroadcastRPC = r.RegisterRPC(new(randomizedpaxosproto.InfoBroadcast), r.infoBroadcastChan)
	r.infoBroadcastReplyRPC = r.RegisterRPC(new(randomizedpaxosproto.InfoBroadcastReply), r.infoBroadcastReplyChan)

	// go r.run()

	return r
}

func (r *Replica) replyReplicateEntries(replicaId int32, reply *randomizedpaxosproto.ReplicateEntriesReply) {
	r.SendMsg(replicaId, r.replicateEntriesReplyRPC, reply)
}

func (r *Replica) replyRequestVote(replicaId int32, reply *randomizedpaxosproto.RequestVoteReply) {
	r.SendMsg(replicaId, r.requestVoteReplyRPC, reply)
}

func (r *Replica) replyBenOrBroadcast(replicaId int32, reply *randomizedpaxosproto.BenOrBroadcastReply) {
	r.SendMsg(replicaId, r.benOrBroadcastReplyRPC, reply)
}

func (r *Replica) replyBenOrConsensus(replicaId int32, reply *randomizedpaxosproto.BenOrConsensusReply) {
	r.SendMsg(replicaId, r.benOrConsensusReplyRPC, reply)
}

func (r *Replica) replyInfoBroadcast(replicaId int32, reply *randomizedpaxosproto.InfoBroadcastReply) {
	r.SendMsg(replicaId, r.infoBroadcastReplyRPC, reply)
}

var clockChan chan bool

func (r *Replica) clock() {
	for !r.Shutdown {
		time.Sleep(BATCH_INTERVAL)
		clockChan <- true
	}
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
	if a > b { 
		return a
	}
	return b
}

// called when the timer has not yet fired
func (r *Replica) resetTimer(t *time.Timer, d time.Duration) {
	if t == nil {
		t = time.NewTimer(d)
		return
	}
	if !t.Stop() {
		<-t.C
	}
	t.Reset(d)
}

// called when the timer has already fired
func (r *Replica) setTimer(t *time.Timer, d time.Duration) {
	if t == nil {
		t = time.NewTimer(d)
		return
	}
	t.Reset(d)
}

/* Main event processing loop */
func (r *Replica) run() {

	r.ConnectToPeers()

	dlog.Println("Waiting for client connections")

	go r.WaitForClientConnections()

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

			// Execution of the leader's state machine
			for i := r.lastApplied + 1; i <= commitIndices; i++ {
				if writer, ok := r.clientWriters[r.log[i].Data.ClientId]; ok {
					val := r.log[i].Data.Execute(r.State)
					propreply := &genericsmrproto.ProposeReplyTS{
						TRUE,
						r.log[i].Data.OpId,
						val,
						r.log[i].Timestamp} // TODO: check if timestamp is correct
					r.ReplyProposeTS(propreply, writer)
				}
			}

			// Update preparedIndex
			r.preparedIndex = matchIndices[r.N/2].Idx
		}

		select {
			case client := <-r.RegisterClientIdChan:
				r.registerClient(client.ClientId, client.Reply)

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
				replicateEntries := replicateEntriesS.(*randomizedpaxosproto.ReplicateEntries)
				//got a ReplicateEntries message
				dlog.Printf("Received ReplicateEntries from replica %d, for instance %d\n", replicateEntries.SenderId, replicateEntries.Term)
				r.handleReplicateEntries(replicateEntries)
				break

			case replicateEntriesReplyS := <-r.replicateEntriesReplyChan:
				replicateEntriesReply := replicateEntriesReplyS.(*randomizedpaxosproto.ReplicateEntriesReply)
				//got a ReplicateEntriesReply message
				dlog.Printf("Received ReplicateEntriesReply from replica %d\n", replicateEntriesReply.Term)
				r.handleReplicateEntriesReply(replicateEntriesReply)
				break

			case requestVoteS := <-r.requestVoteChan:
				requestVote := requestVoteS.(*randomizedpaxosproto.RequestVote)
				//got a RequestVote message
				dlog.Printf("Received RequestVote from replica %d, for instance %d\n", requestVote.SenderId, requestVote.Term)
				r.handleRequestVote(requestVote)
				break

			case requestVoteReplyS := <-r.requestVoteReplyChan:
				requestVoteReply := requestVoteReplyS.(*randomizedpaxosproto.RequestVoteReply)
				//got a RequestVoteReply message
				dlog.Printf("Received RequestVoteReply from replica %d\n", requestVoteReply.Term)
				r.handleRequestVoteReply(requestVoteReply)
				break

			case benOrBroadcastS := <-r.benOrBroadcastChan:
				benOrBroadcast := benOrBroadcastS.(*randomizedpaxosproto.BenOrBroadcast)
				//got a BenOrBroadcast message
				dlog.Printf("Received BenOrBroadcast from replica %d, for instance %d\n", benOrBroadcast.SenderId, benOrBroadcast.Term)
				r.handleBenOrBroadcast(benOrBroadcast)
				break

			case benOrBroadcastReplyS := <-r.benOrBroadcastReplyChan:
				benOrBroadcastReply := benOrBroadcastReplyS.(*randomizedpaxosproto.BenOrBroadcastReply)
				//got a BenOrBroadcastReply message
				dlog.Printf("Received BenOrBroadcastReply from replica %d\n", benOrBroadcastReply.Term)
				r.handleBenOrBroadcastReply(benOrBroadcastReply)
				break

			case benOrConsensusS := <-r.benOrConsensusChan:
				benOrConsensus := benOrConsensusS.(*randomizedpaxosproto.BenOrConsensus)
				//got a BenOrConsensus message
				dlog.Printf("Received BenOrConsensus from replica %d, for instance %d\n", benOrConsensus.SenderId, benOrConsensus.Term)
				r.handleBenOrConsensus(benOrConsensus)
				break

			case benOrConsensusReplyS := <-r.benOrConsensusReplyChan:
				benOrConsensusReply := benOrConsensusReplyS.(*randomizedpaxosproto.BenOrConsensusReply)
				//got a BenOrConsensusReply message
				dlog.Printf("Received BenOrConsensusReply from replica %d\n", benOrConsensusReply.Term)
				r.handleBenOrConsensusReply(benOrConsensusReply)
				break

			case infoBroadcastS := <-r.infoBroadcastChan:
				infoBroadcast := infoBroadcastS.(*randomizedpaxosproto.InfoBroadcast)
				//got a InfoBroadcast message
				dlog.Printf("Received InfoBroadcast from replica %d, for instance %d\n", infoBroadcast.SenderId, infoBroadcast.Term)
				r.handleInfoBroadcast(infoBroadcast)
				break

			case infoBroadcastReplyS := <-r.infoBroadcastReplyChan:
				infoBroadcastReply := infoBroadcastReplyS.(*randomizedpaxosproto.InfoBroadcastReply)
				//got a InfoBroadcastReply message
				dlog.Printf("Received InfoBroadcastReply from replica %d\n", infoBroadcastReply.Term)
				r.handleInfoBroadcastReply(infoBroadcastReply)
				break
			
			case <- r.heartbeatTimer.C:
				//got a heartbeat timeout
				r.sendHeartbeat()
			
			case <- r.electionTimer.C:
				//got an election timeout
				timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
				r.setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
				r.startElection()
			
			case <- r.benOrStartWaitTimer.C:
				//got a benOrStartWait timeout
				r.startBenOrPlus()
		}
	}
}

// Manage Client Writers
func (r *Replica) registerClient(clientId uint32, writer *bufio.Writer) uint8 {
	w, exists := r.clientWriters[clientId]

	if !exists {
		r.clientWriters[clientId] = writer
		return TRUE
	}

	if w == writer {
		return TRUE
	}

	return FALSE
}

func (r *Replica) bcastReplicateEntries() {
	r.replicateEntriesCounter = rpcCounter{r.currentTerm, r.replicateEntriesCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			args := &randomizedpaxosproto.ReplicateEntries{
				r.Id, int32(r.currentTerm), int32(r.replicateEntriesCounter.count), 
				int32(r.matchIndex[i]+1), int32(r.log[i].Term),
				r.log[r.matchIndex[i]+1:], int32(r.benOrIndex), int32(r.preparedIndex)}

			r.SendMsg(int32(i), r.replicateEntriesRPC, args)
		}
	}
}

func (r *Replica) bcastInfoBroadcast(clientReq randomizedpaxosproto.Entry) {
	r.infoBroadcastCounter = rpcCounter{r.currentTerm, r.infoBroadcastCounter.count+1}
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			args := &randomizedpaxosproto.InfoBroadcast{
				r.Id, int32(r.currentTerm), int32(r.infoBroadcastCounter.count), 
				clientReq}

			r.SendMsg(int32(i), r.infoBroadcastRPC, args)
		}
	}
}

func (r *Replica) handlePropose(propose *genericsmr.Propose) {
	newLogEntry := randomizedpaxosproto.Entry{
		Data: propose.Command,
		SenderId: r.Id,
		Term: int32(r.currentTerm),
		Index: int32(len(r.log)),
		Timestamp: r.currentTimer.UnixNano()}

	if r.isLeader {
		r.log = append(r.log, newLogEntry)
		if r.benOrIndex == len(r.log) { r.benOrIndex++ }
		r.bcastReplicateEntries()
	}

	r.bcastInfoBroadcast(newLogEntry)
	r.pq.push(newLogEntry)
}

func (r *Replica) handleInfoBroadcast(rpc *randomizedpaxosproto.InfoBroadcast) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
	}

	args := &randomizedpaxosproto.InfoBroadcastReply{
		r.Id, int32(r.currentTerm), int32(r.infoBroadcastCounter.count)}
	r.SendMsg(rpc.SenderId, r.infoBroadcastRPC, args)
	return

	r.pq.push(rpc.ClientReq)
}

func (r *Replica) benOrRunning() bool {
	return (r.benOrStatus == Broadcasting || r.benOrStatus == BenOrRunning)
}

func benOrUncommittedLogEntry(idx int) randomizedpaxosproto.Entry {
	return randomizedpaxosproto.Entry{
		Data: state.Command{},
		SenderId: -1,
		Term: -1, // term -1 means that this is a Ben-Or+ entry that hasn't yet committed
		Index: int32(idx),
		BenOrActive: true,
		Timestamp: -1}
}

func (r *Replica) handleReplicateEntries(rpc *randomizedpaxosproto.ReplicateEntries) {
	replicaEntries := make([]randomizedpaxosproto.Entry, 0)
	if int(rpc.LeaderPreparedIndex)+1 < len(r.log) {
		replicaEntries = r.log[rpc.LeaderPreparedIndex+1:]
	}

	entryAtLeaderBenOrIndex := randomizedpaxosproto.Entry{}
	entryAtLeaderBenOrIndex.Term = -1
	if int(rpc.LeaderBenOrIndex) < len(r.log) {
		entryAtLeaderBenOrIndex = r.log[rpc.LeaderBenOrIndex]
	}

	args := &randomizedpaxosproto.ReplicateEntriesReply{
		ReplicaId: r.Id,
		Term: int32(r.currentTerm),
		Counter: int32(r.infoBroadcastCounter.count),
		ReplicaBenOrIndex: int32(r.benOrIndex),
		ReplicaPreparedIndex: int32(r.preparedIndex),
		ReplicaEntries: replicaEntries,
		LeaderPreparedIndex: rpc.LeaderPreparedIndex,
		LeaderBenOrIndex: rpc.LeaderBenOrIndex,
		EntryAtLeaderBenOrIndex: entryAtLeaderBenOrIndex,
		Success: false}

	// reject if the term is out of date
	if (int(rpc.Term) < r.currentTerm) {
		r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
		return
	}

	// update to follower if received term is newer
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
	}

	// a term of -1 means that the entry is committed using Ben-Or+
	// we assume that r.entries[rpc.PrevLogIndex] is not currently running Ben-Or+
	// reject if the previous term doesn't match
	// if entry at previous term is running Ben-Or+, then we must reject and ask the leader to send over another entry
	if (r.entries[rpc.PrevLogIndex].Term != rpc.PrevLogTerm || r.entries[rpc.PrevLogIndex].BenOrActive) {
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
		if (r.log[i].Term != rpc.Entries[i-firstEntryIndex].Term) {
			if (r.log[i].BenOrActive) {
				r.log[i] = rpc.Entries[i-firstEntryIndex]
				if !rpc.Entries[i-firstEntryIndex].BenOrActive && i <= newPreparedPoint {
					benOrIndexChanged = true
					continue
				}

				if r.benOrStatus == Stopped {
					// don't need to do anything else
				} else if r.benOrStatus == Broadcasting {
					r.biasedCoin = true
				} else { // r.benOrStatus == BenOrRunning
					// can't do anything here
				}
			} else if (rpc.Entries[i-firstEntryIndex].BenOrActive) {
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
		r.benOrStatus = Stopped
		r.biasedCoin = false
		if (r.benOrIndex < len(r.log)) {
			r.log[r.benOrIndex].BenOrActive = true
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
		Counter: int32(r.infoBroadcastCounter.count),
		ReplicaBenOrIndex: int32(r.benOrIndex),
		ReplicaPreparedIndex: int32(r.preparedIndex),
		ReplicaEntries: replicaEntries,
		LeaderPreparedIndex: rpc.LeaderPreparedIndex,
		LeaderBenOrIndex: rpc.LeaderBenOrIndex,
		EntryAtLeaderBenOrIndex: entryAtLeaderBenOrIndex,
		Success: true}

	r.SendMsg(rpc.SenderId, r.replicateEntriesReplyRPC, args)
	return
}


func (r *Replica) startElection () {
	r.currentTerm++
	r.votedFor = int(r.Id)
	r.votesReceived = 1 // itself
	r.requestVoteEntries = append(r.log[r.preparedIndex+1:], r.pq.extractList()...)

	args := &randomizedpaxosproto.RequestVote{
		SenderId: r.Id,
		Term: int32(r.currentTerm),
		Counter: 0,
		CandidateBenOrIndex: int32(r.benOrIndex),
	}

	r.sendMsg(r.Id, r.requestVoteRPC, args)
}


func (r *Replica) handleRequestVote (rpc *randomizedpaxosproto.RequestVote) {
	replicaEntries := make([]randomizedpaxosproto.Entry, 0)
	if int(rpc.CandidateBenOrIndex)+1 < len(r.log) {
		replicaEntries = r.log[rpc.CandidateBenOrIndex:]
	}

	entryAtCandidateBenOrIndex := randomizedpaxosproto.Entry{}
	entryAtCandidateBenOrIndex.Term = -1
	if int(rpc.CandidateBenOrIndex) < len(r.log) {
		entryAtCandidateBenOrIndex = r.log[rpc.CandidateBenOrIndex]
	}
	
	args := &randomizedpaxosproto.RequestVoteReply{
		ReplicaId: r.Id,
		Term: int32(r.currentTerm),
		Counter: int32(r.infoBroadcastCounter.count),
		VoteGranted: false,
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

	args.VoteGranted = true
	r.votedFor = int(rpc.SenderId)
	r.SendMsg(rpc.SenderId, r.requestVoteReplyRPC, args)
	return
}

func (r *Replica) handleRequestVoteReply (rpc *randomizedpaxosproto.RequestVoteReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
		r.votesReceived = 0
		r.requestVoteEntries = make([]randomizedpaxosproto.Entry, 0)
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
				if (r.log[i].BenOrActive) {
					if !rpc.ReplicaEntries[idx].BenOrActive && i <= newPreparedPoint {
						// r.requestVoteEntries[i] = rpc.ReplicaEntries[idx]
						r.log[i] = rpc.ReplicaEntries[idx]
						continue
					}
					// else, use current entry
				} else if (rpc.ReplicaEntries[idx].BenOrActive) {
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

	if rpc.VoteGranted {
		r.votesReceived++

		start := newPreparedPoint + 1
		if r.benOrRunning() && r.log[start].BenOrActive {
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
		r.requestVoteEntries = make([]randomizedpaxosproto.Entry, 0)

		for i := 0; i < r.N ; i++ {
			r.nextIndex[i] = len(r.log)
			r.matchIndex[i] = 0
			r.commitIndex[i] = 0
		}

		timeout := rand.Intn(r.heartbeatTimeout/2) + r.heartbeatTimeout/2
		r.resetTimer(r.heartbeatTimer, time.Duration(timeout)*time.Millisecond)
		// send out replicate entries rpcs
		r.bcastReplicateEntries()
	}
}

func (r *Replica) handleInfoBroadcastReply (rpc *randomizedpaxosproto.InfoBroadcastReply) {
	if (int(rpc.Term) > r.currentTerm) {
		r.currentTerm = int(rpc.Term)
		r.isLeader = false
		r.votesReceived = 0
	}
	return
}

func (r *Replica) handleReplicateEntriesReply (rpc *randomizedpaxosproto.ReplicateEntriesReply) {
	
}