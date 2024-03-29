package genericsmr

import (
	"bufio"
	"dlog"
	"encoding/binary"
	"fastrpc"
	"fmt"
	"genericsmrproto"
	"io"
	"log"
	"net"
	"os"
	"rdtsc"
	"state"
	"sync"
	"time"

	mrand "math/rand"
)

const CHAN_BUFFER_SIZE = 200000

type RPCPair struct {
	Obj  fastrpc.Serializable
	Chan chan fastrpc.Serializable
}

type Propose struct {
	*genericsmrproto.Propose
	Reply *bufio.Writer
}

type Beacon struct {
	Rid       int32
	Timestamp uint64
}

type Client struct {
	*genericsmrproto.RegisterClientIdArgs
	Reply *bufio.Writer
}

type GetView struct {
	*genericsmrproto.GetView
	Reply *bufio.Writer
}

type GetState struct {
	*genericsmrproto.GetState
	Reply *bufio.Writer
}

type Slowdown struct {
	*genericsmrproto.Slowdown
	Reply *bufio.Writer
}

type Connect struct {
	*genericsmrproto.Connect
	Reply *bufio.Writer
}

type Disconnect struct {
	*genericsmrproto.Disconnect
	Reply *bufio.Writer
}

// type IsConnectedStatus struct {
// 	// Mu 		sync.Mutex
// 	Connected	map[int]bool
// }

type RepCommand struct {
	ServerId int
	Command  state.Command
}

type Testing struct { // for testing purposes only
	IsProduction bool
	// IsConnected  IsConnectedStatus
	IsReliable   map[int]bool
	RequestChan  chan state.Command
	ResponseChan chan RepCommand
}

type LockedReader struct {
	*bufio.Reader
	mu *sync.Mutex
}

type LockedWriter struct {
	*bufio.Writer
	mu *sync.Mutex
}

type Replica struct {
	N            int        // total number of replicas
	Id           int32      // the ID of the current replica
	PeerAddrList []string   // array with the IP:port address of every replica
	Peers        []net.Conn // cache of connections to all other replicas
	PeerReaders  []LockedReader
	PeerWriters  []LockedWriter
	Alive        []bool // connection status
	Listener     net.Listener

	State *state.State

	ProposeChan chan *Propose // channel for client proposals
	BeaconChan  chan *Beacon  // channel for beacons from peer replicas

	Shutdown bool

	Thrifty bool // send only as many messages as strictly required?
	Exec    bool // execute commands?
	Dreply  bool // reply to client after command has been executed?
	Beacon  bool // send beacons to detect how fast are the other replicas?

	Durable     bool     // log to a stable store?
	StableStore *os.File // file support for the persistent log

	PreferredPeerOrder []int32 // replicas in the preferred order of communication

	rpcTable map[uint8]*RPCPair
	rpcCode  uint8

	Ewma []float64

	OnClientConnect chan bool

	RegisterClientIdChan chan *Client // channel for registering client id

	GetViewChan chan *GetView

	GetStateChan chan *GetState

	SlowdownChan chan *Slowdown

	ConnectChan chan *Connect

	DisconnectChan chan *Disconnect

	TestingState Testing

	Connected	map[int]bool
}

func NewReplica(id int, peerAddrList []string, thrifty bool, exec bool, dreply bool) *Replica {
	r := &Replica{
		len(peerAddrList),
		int32(id),
		peerAddrList,
		make([]net.Conn, len(peerAddrList)),
		make([]LockedReader, len(peerAddrList)),
		make([]LockedWriter, len(peerAddrList)),
		make([]bool, len(peerAddrList)),
		nil,
		state.InitState(),
		make(chan *Propose, CHAN_BUFFER_SIZE),
		make(chan *Beacon, CHAN_BUFFER_SIZE),
		false,
		thrifty,
		exec,
		dreply,
		false,
		false,
		nil,
		make([]int32, len(peerAddrList)),
		make(map[uint8]*RPCPair),
		genericsmrproto.GENERIC_SMR_BEACON_REPLY + 1,
		make([]float64, len(peerAddrList)),
		make(chan bool, 1200),
		make(chan *Client, CHAN_BUFFER_SIZE),
		make(chan *GetView, CHAN_BUFFER_SIZE),
		make(chan *GetState, CHAN_BUFFER_SIZE),
		make(chan *Slowdown, CHAN_BUFFER_SIZE),
		make(chan *Connect, CHAN_BUFFER_SIZE),
		make(chan *Disconnect, CHAN_BUFFER_SIZE),
		Testing{true, make(map[int]bool), nil, nil},
		make(map[int]bool),
	}

	var err error

	if r.StableStore, err = os.Create(fmt.Sprintf("stable-store-replica%d", r.Id)); err != nil {
		log.Fatal(err)
	}

	for i := 0; i < r.N; i++ {
		r.PreferredPeerOrder[i] = int32((int(r.Id) + 1 + i) % r.N)
		r.Ewma[i] = 0.0
	}

	return r
}

/* Client API */

func (r *Replica) Ping(args *genericsmrproto.PingArgs, reply *genericsmrproto.PingReply) error {
	return nil
}

func (r *Replica) BeTheLeader(args *genericsmrproto.BeTheLeaderArgs, reply *genericsmrproto.BeTheLeaderReply) error {
	return nil
}

func (r *Replica) BeTheLeader2(args *genericsmrproto.BeTheLeaderArgs, reply *genericsmrproto.BeTheLeaderReply) error {
	return nil
}

type SimConn struct {
	*io.PipeReader
	*io.PipeWriter
}

func NewSimConn(pr *io.PipeReader, pw *io.PipeWriter) *SimConn{
	return &SimConn{
		PipeReader: pr,
		PipeWriter: pw,
	}
}

func (r *Replica) ConnectToPeersSim(serverId int, simConn *SimConn, reliable bool) {
	// a, b := io.Pipe()

	r.Alive[serverId] = true
	r.PeerReaders[serverId] = LockedReader{bufio.NewReader(simConn), &sync.Mutex{}}
	r.PeerWriters[serverId] = LockedWriter{bufio.NewWriter(simConn), &sync.Mutex{}}

	// r.TestingState.IsConnected.Mu.Lock()
	r.Connected[serverId] = true
	r.TestingState.IsReliable[serverId] = reliable
	dlog.Println("server", r.Id, "connected to server", serverId)
	// r.TestingState.IsConnected.Mu.Unlock()
}

func (r *Replica) ConnectListenToPeers() {
	for rid, reader := range r.PeerReaders {
		if int32(rid) == r.Id {
			continue
		}
		go r.replicaListener(rid, reader)
	}
}

/* ============= */

func (r *Replica) ConnectToPeers() {
	var b [4]byte
	bs := b[:4]
	done := make(chan bool)

	go r.waitForPeerConnections(done)

	//connect to peers
	for i := 0; i < int(r.Id); i++ {
		for done := false; !done; {
			if conn, err := net.Dial("tcp", r.PeerAddrList[i]); err == nil {
				r.Peers[i] = conn
				done = true
			} else {
				time.Sleep(1e9)
			}
		}
		binary.LittleEndian.PutUint32(bs, uint32(r.Id))
		if _, err := r.Peers[i].Write(bs); err != nil {
			fmt.Println("Write id error:", err)
			continue
		}
		r.Alive[i] = true
		r.PeerReaders[i] = LockedReader{bufio.NewReader(r.Peers[i]), &sync.Mutex{}}
		r.PeerWriters[i] = LockedWriter{bufio.NewWriter(r.Peers[i]), &sync.Mutex{}}
		r.Connected[i] = true
	}
	<-done
	log.Printf("Replica id: %d. Done connecting to peers\n", r.Id)

	for rid, reader := range r.PeerReaders {
		if int32(rid) == r.Id {
			continue
		}
		go r.replicaListener(rid, reader)
	}
}

func (r *Replica) ConnectToPeersNoListeners() {
	var b [4]byte
	bs := b[:4]
	done := make(chan bool)

	go r.waitForPeerConnections(done)

	//connect to peers
	for i := 0; i < int(r.Id); i++ {
		for done := false; !done; {
			if conn, err := net.Dial("tcp", r.PeerAddrList[i]); err == nil {
				r.Peers[i] = conn
				done = true
			} else {
				time.Sleep(1e9)
			}
		}
		binary.LittleEndian.PutUint32(bs, uint32(r.Id))
		if _, err := r.Peers[i].Write(bs); err != nil {
			fmt.Println("Write id error:", err)
			continue
		}
		r.Alive[i] = true
		r.PeerReaders[i] = LockedReader{bufio.NewReader(r.Peers[i]), &sync.Mutex{}}
		r.PeerWriters[i] = LockedWriter{bufio.NewWriter(r.Peers[i]), &sync.Mutex{}}
	}
	<-done
	log.Printf("Replica id: %d. Done connecting to peers\n", r.Id)
}

/* Peer (replica) connections dispatcher */
func (r *Replica) waitForPeerConnections(done chan bool) {
	var b [4]byte
	bs := b[:4]

	r.Listener, _ = net.Listen("tcp", r.PeerAddrList[r.Id])
	for i := r.Id + 1; i < int32(r.N); i++ {
		conn, err := r.Listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}
		if _, err := io.ReadFull(conn, bs); err != nil {
			fmt.Println("Connection establish error:", err)
			continue
		}
		id := int32(binary.LittleEndian.Uint32(bs))
		r.Peers[id] = conn
		r.PeerReaders[id] = LockedReader{bufio.NewReader(conn), &sync.Mutex{}}
		r.PeerWriters[id] = LockedWriter{bufio.NewWriter(conn), &sync.Mutex{}}
		r.Alive[id] = true
		r.Connected[int(id)] = true
	}

	done <- true
}

/* Client connections dispatcher */
func (r *Replica) WaitForClientConnections() {
	for !r.Shutdown {
		conn, err := r.Listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go r.clientListener(conn)

		fmt.Println("Client connected on replica", r.Id, "!")
		r.OnClientConnect <- true
	}
}

type HasSenderId interface {
	GetSenderId() int32
}

func (r *Replica) replicaListener(rid int, reader LockedReader) {
	var msgType uint8
	var err error = nil
	var gbeacon genericsmrproto.Beacon
	var gbeaconReply genericsmrproto.BeaconReply

	// var timer05ms *time.Timer
	// var timer1ms *time.Timer
	// var timer2ms *time.Timer
	// var timer5ms *time.Timer
	// var timer10ms *time.Timer
	// var timer20ms *time.Timer
	// var timer40ms *time.Timer
	// var timer80ms *time.Timer
	// allFired := false
	// INJECT_SLOWDOWN := false
	// if r.Id == int32(0) && INJECT_SLOWDOWN {
	// 	timer05ms = time.NewTimer(48 * time.Second)
	// 	timer1ms = time.NewTimer(49 * time.Second)
	// 	timer2ms = time.NewTimer(50 * time.Second)
	// 	timer5ms = time.NewTimer(51 * time.Second)
	// 	timer10ms = time.NewTimer(52 * time.Second)
	// 	timer20ms = time.NewTimer(53 * time.Second)
	// 	timer40ms = time.NewTimer(54 * time.Second)
	// 	timer80ms = time.NewTimer(55 * time.Second)
	// }
	for err == nil && !r.Shutdown {

		// if r.Id == int32(0) && INJECT_SLOWDOWN && !allFired {
		// 	select {
		// 	case <-timer05ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 0.5ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(500 * time.Microsecond)

		// 	case <-timer1ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 1ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(1 * time.Millisecond)

		// 	case <-timer2ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 2ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(2 * time.Millisecond)

		// 	case <-timer5ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 5ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(5 * time.Millisecond)

		// 	case <-timer10ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 10ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(10 * time.Millisecond)

		// 	case <-timer20ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 20ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(20 * time.Millisecond)

		// 	case <-timer40ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 40ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(40 * time.Millisecond)

		// 	case <-timer80ms.C:
		// 		fmt.Printf("Replica %v: replicaListenerTimer 80ms fired at %v\n", r.Id, time.Now())
		// 		allFired = true
		// 		time.Sleep(80 * time.Millisecond)

		// 	default:
		// 		break

		// 	}
		// }
		// fmt.Println("Replica", r.Id, "is waiting for message from replica", rid)

		// if !r.IsProduction {
		// 	for _, err := reader.Peek(1); err != nil; _, err = reader.Peek(1) {
		// 		time.Sleep(10 * time.Millisecond)
		// 	}
		// }
		if msgType, err = reader.ReadByte(); err != nil {
			break
		}
		// fmt.Println(r.Id, "finished1", msgType)

		// fmt.Println("HUH")
		switch uint8(msgType) {

		case genericsmrproto.GENERIC_SMR_BEACON:
			// if !r.IsProduction {
			// 	for _, err := reader.Peek(1); err != nil; _, err = reader.Peek(1) {
			// 		time.Sleep(10 * time.Millisecond)
			// 	}
			// }
			if err = gbeacon.Unmarshal(reader); err != nil {
				break
			}
			beacon := &Beacon{int32(rid), gbeacon.Timestamp}
			r.BeaconChan <- beacon
			break

		case genericsmrproto.GENERIC_SMR_BEACON_REPLY:
			// if !r.IsProduction {
			// 	for _, err := reader.Peek(1); err != nil; _, err = reader.Peek(1) {
			// 		time.Sleep(10 * time.Millisecond)
			// 	}
			// }
			if err = gbeaconReply.Unmarshal(reader); err != nil {
				break
			}
			//TODO: UPDATE STUFF
			r.Ewma[rid] = 0.99*r.Ewma[rid] + 0.01*float64(rdtsc.Cputicks()-gbeaconReply.Timestamp)
			log.Println(r.Ewma)
			break

		default:
			// NOTE: sends value here
			// fmt.Println("Replica", r.Id, "received message from replica", rid, "of type", msgType)

			// if msgType, err = reader.ReadByte(); err != nil {
			// 	break
			// }
			// fmt.Println(r.Id, "finished2222", msgType)

			// fmt.Println("umm2")
			// buf := make([]byte, 1024)
			// var n int
			// if n, err = reader.Read(buf); err != nil {
			// 	break
			// }
			// fmt.Println("WHATTT", string(buf[:n]))

			if rpair, present := r.rpcTable[msgType]; present {
				obj := rpair.Obj.New()
				// if !r.IsProduction {
				// 	for _, err := reader.Peek(1); err != nil; _, err = reader.Peek(1) {
				// 		time.Sleep(10 * time.Millisecond)
				// 	}
				// }

				// fmt.Println("umm2")
				// buf := make([]byte, 1024)
				// var n int
				// if n, err = reader.Read(buf); err != nil {
				// 	break
				// }
				// fmt.Println("WHATTT", string(buf[:n]))
				// log.Println("Error: Replica", r.Id, "received", msgType, "type from", rid)
				if err = obj.Unmarshal(reader); err != nil {
					break
				}

				// if !r.TestingState.IsProduction {
				// 	if t, ok := obj.(HasSenderId); ok {
				// 		// r.TestingState.IsConnected.Mu.Lock()
				// 		if !r.Connected[int(t.GetSenderId())] {
				// 			// r.TestingState.IsConnected.Mu.Unlock()
				// 			fmt.Println("Replica", r.Id, "received message from replica", rid, "of type", msgType, "but replica", t.GetSenderId(), "is not connected");
				// 			continue
				// 		} else {
				// 			// r.TestingState.IsConnected.Mu.Unlock()
				// 			rpair.Chan <- obj
				// 		}
				// 	}
				// } else {
				// 	rpair.Chan <- obj
				// }
				if t, ok := obj.(HasSenderId); ok {
					// r.TestingState.IsConnected.Mu.Lock()
					if !r.Connected[int(t.GetSenderId())] {
						// r.TestingState.IsConnected.Mu.Unlock()
						// fmt.Println("Replica", r.Id, "received message from replica", rid, "of type", msgType, "but replica", t.GetSenderId(), "is not connected");
						continue
					} else {
						// r.TestingState.IsConnected.Mu.Unlock()
						rpair.Chan <- obj
					}
				} else {
					rpair.Chan <- obj
				}

				// rpair.Chan <- obj
			} else {
				log.Println("Error: Replica", r.Id, "received unknown message type from", rid)
			}
		}
	}
}

func (r *Replica) clientListener(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	var msgType byte //:= make([]byte, 1)
	var err error

	// var timer05ms *time.Timer
	// var timer1ms *time.Timer
	// var timer2ms *time.Timer
	// var timer5ms *time.Timer
	// var timer10ms *time.Timer
	// var timer20ms *time.Timer
	// var timer40ms *time.Timer
	// var timer80ms *time.Timer
	// allFired := false
	// INJECT_SLOWDOWN := false
	// if r.Id == int32(0) && INJECT_SLOWDOWN {
	// 	timer05ms = time.NewTimer(42 * time.Second)
	// 	timer1ms = time.NewTimer(43 * time.Second)
	// 	timer2ms = time.NewTimer(44 * time.Second)
	// 	timer5ms = time.NewTimer(45 * time.Second)
	// 	timer10ms = time.NewTimer(46 * time.Second)
	// 	timer20ms = time.NewTimer(47 * time.Second)
	// 	timer40ms = time.NewTimer(48 * time.Second)
	// 	timer80ms = time.NewTimer(49 * time.Second)
	// }

	for !r.Shutdown && err == nil {

		// if r.Id == int32(0) && INJECT_SLOWDOWN && !allFired {
		// 	select {
		// 	case <-timer05ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 0.5ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(500 * time.Microsecond)

		// 	case <-timer1ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 1ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(1 * time.Millisecond)

		// 	case <-timer2ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 2ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(2 * time.Millisecond)

		// 	case <-timer5ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 5ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(5 * time.Millisecond)

		// 	case <-timer10ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 10ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(10 * time.Millisecond)

		// 	case <-timer20ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 20ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(20 * time.Millisecond)

		// 	case <-timer40ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 40ms fired at %v\n", r.Id, time.Now())
		// 		time.Sleep(40 * time.Millisecond)

		// 	case <-timer80ms.C:
		// 		fmt.Printf("Replica %v: clientListenerTimer 80ms fired at %v\n", r.Id, time.Now())
		// 		allFired = true
		// 		time.Sleep(80 * time.Millisecond)

		// 	default:
		// 		break

		// 	}
		// }

		if msgType, err = reader.ReadByte(); err != nil {
			break
		}

		switch uint8(msgType) {

		case genericsmrproto.PROPOSE:
			prop := new(genericsmrproto.Propose)
			if err = prop.Unmarshal(reader); err != nil {
				break
			}
			//if (time.Now().UnixNano() - prop.Timestamp) >= int64(5000000) /*5ms*/ {
			//		fmt.Printf("Replica %v: clientListener: request %v-%v takes %v (us)\n", r.Id, prop.Command.ClientId, prop.CommandId, (time.Now().UnixNano()-prop.Timestamp)/int64(1000))

			//}
			r.ProposeChan <- &Propose{prop, writer}
			break

		case genericsmrproto.READ:
			read := new(genericsmrproto.Read)
			if err = read.Unmarshal(reader); err != nil {
				break
			}
			//r.ReadChan <- read
			break

		case genericsmrproto.PROPOSE_AND_READ:
			pr := new(genericsmrproto.ProposeAndRead)
			if err = pr.Unmarshal(reader); err != nil {
				break
			}
			//r.ProposeAndReadChan <- pr
			break

		case genericsmrproto.REGISTER_CLIENT_ID:

			rci := new(genericsmrproto.RegisterClientIdArgs)
			if err = rci.Unmarshal(reader); err != nil {
				fmt.Println("Error reading from client", err)
				break
			}
			dlog.Println("Receiving registration from client", rci.ClientId)
			r.RegisterClientIdChan <- &Client{rci, writer}
			break

		case genericsmrproto.GET_VIEW:
			gv := new(genericsmrproto.GetView)
			if err = gv.Unmarshal(reader); err != nil {
				break
			}
			r.GetViewChan <- &GetView{gv, writer}
			break

		case genericsmrproto.GET_STATE:
			// fmt.Println("Replica %d: GET_STATE received", r.Id)
			gst := new(genericsmrproto.GetState)
			if err = gst.Unmarshal(reader); err != nil {
				break
			}
			r.GetStateChan <- &GetState{gst, writer}
			break

		case genericsmrproto.SLOWDOWN:
			slowdown := new(genericsmrproto.Slowdown)
			if err = slowdown.Unmarshal(reader); err != nil {
				break
			}
			r.SlowdownChan <- &Slowdown{slowdown, writer}
			break

		case genericsmrproto.CONNECT:
			connect := new(genericsmrproto.Connect)
			if err = connect.Unmarshal(reader); err != nil {
				break
			}
			r.ConnectChan <- &Connect{connect, writer}
			break
		
		case genericsmrproto.DISCONNECT:
			disconnect := new(genericsmrproto.Disconnect)
			if err = disconnect.Unmarshal(reader); err != nil {
				break
			}
			r.DisconnectChan <- &Disconnect{disconnect, writer}
			break
		}
	}
	if err != nil && err != io.EOF {
		log.Println("Error when reading from client connection:", err)
	}
}

func (r *Replica) RegisterRPC(msgObj fastrpc.Serializable, notify chan fastrpc.Serializable) uint8 {
	code := r.rpcCode
	r.rpcCode++
	r.rpcTable[code] = &RPCPair{msgObj, notify}
	return code
}

func (r *Replica) SendMsg(peerId int32, code uint8, msg fastrpc.Serializable) {
	// if !r.TestingState.IsProduction{
		// r.TestingState.IsConnected.Mu.Lock()
		if !r.Connected[int(peerId)] {
			// r.TestingState.IsConnected.Mu.Unlock()
			// fmt.Println("Replica", r.Id, "wants to send message to peer", peerId, "but it is not connected")
			return
		}
		// r.TestingState.IsConnected.Mu.Unlock()
	// }

	if !r.TestingState.IsProduction && !r.TestingState.IsReliable[int(peerId)] {
		// short delay
		go func() {
			time.Sleep(time.Duration(mrand.Int() % 27) * time.Millisecond)

			if mrand.Int()%1000 < 100 {
				// drop the request, return as if timeout
				return
			}

			// r.TestingState.IsConnected.Mu.Lock()
			if r.Connected[int(peerId)] {
				// r.TestingState.IsConnected.Mu.Unlock()
				// fmt.Println("Replica", r.Id, "sending message to peer", peerId)
				w := r.PeerWriters[peerId]
				w.mu.Lock()
				w.WriteByte(code)
				msg.Marshal(w)
				w.Flush()
				w.mu.Unlock()
			} else {
				// r.TestingState.IsConnected.Mu.Unlock()
			}
			// else {
			// 	fmt.Println("Replica", r.Id, "wants to send message to peer", peerId, "but it is not connected")
			// }
		}()
		return
	}

	// fmt.Println("Replica", r.Id, "sending message to peerrrrr", peerId)
	w := r.PeerWriters[peerId]
	w.WriteByte(code)
	msg.Marshal(w)
	w.Flush()
}

func (r *Replica) SendMsgNoFlush(peerId int32, code uint8, msg fastrpc.Serializable) {
	w := r.PeerWriters[peerId]
	w.WriteByte(code)
	msg.Marshal(w)
}

func (r *Replica) ReplyPropose(reply *genericsmrproto.ProposeReply, w *bufio.Writer) {
	//r.clientMutex.Lock()
	//defer r.clientMutex.Unlock()
	//w.WriteByte(genericsmrproto.PROPOSE_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyProposeTS(reply *genericsmrproto.ProposeReplyTS, w *bufio.Writer) {
	//r.clientMutex.Lock()
	//defer r.clientMutex.Unlock()
	w.WriteByte(genericsmrproto.PROPOSE_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) SendBeacon(peerId int32) {
	w := r.PeerWriters[peerId]
	w.WriteByte(genericsmrproto.GENERIC_SMR_BEACON)
	beacon := &genericsmrproto.Beacon{rdtsc.Cputicks()}
	beacon.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyBeacon(beacon *Beacon) {
	w := r.PeerWriters[beacon.Rid]
	w.WriteByte(genericsmrproto.GENERIC_SMR_BEACON_REPLY)
	rb := &genericsmrproto.BeaconReply{beacon.Timestamp}
	rb.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyRegisterClientId(reply *genericsmrproto.RegisterClientIdReply, w *bufio.Writer) {
	w.WriteByte(genericsmrproto.REGISTER_CLIENT_ID_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyGetView(reply *genericsmrproto.GetViewReply, w *bufio.Writer) {
	w.WriteByte(genericsmrproto.GET_VIEW_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyGetState(reply *genericsmrproto.GetStateReply, w *bufio.Writer) {
	// fmt.Println("Replying get state")
	w.WriteByte(genericsmrproto.GET_STATE_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplySlowdown(reply *genericsmrproto.SlowdownReply, w *bufio.Writer) {
	w.WriteByte(genericsmrproto.SLOWDOWN_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyConnect(reply *genericsmrproto.ConnectReply, w *bufio.Writer) {
	w.WriteByte(genericsmrproto.CONNECT_REPLY)
	reply.Marshal(w)
	w.Flush()
}

func (r *Replica) ReplyDisconnect(reply *genericsmrproto.DisconnectReply, w *bufio.Writer) {
	w.WriteByte(genericsmrproto.DISCONNECT_REPLY)
	reply.Marshal(w)
	w.Flush()
}

// updates the preferred order in which to communicate with peers according to a preferred quorum
func (r *Replica) UpdatePreferredPeerOrder(quorum []int32) {
	aux := make([]int32, r.N)
	i := 0
	for _, p := range quorum {
		if p == r.Id {
			continue
		}
		aux[i] = p
		i++
	}

	for _, p := range r.PreferredPeerOrder {
		found := false
		for j := 0; j < i; j++ {
			if aux[j] == p {
				found = true
				break
			}
		}
		if !found {
			aux[i] = p
			i++
		}
	}

	r.PreferredPeerOrder = aux
}
