package randomizedpaxos

import (
	"genericsmr"
	"io"
	"runtime"
	"state"
	"sync"
	"testing"
	"time"
)

type network struct {
	conn	[][]*genericsmr.SimConn // unidirectional connection from i to j
	n	int
}

var CLIENTID uint32 = 0

// func (net *network) Connect(i, j int) {
// 	net.conn[i][j].Connect()
// 	net.conn[j][i].Connect()
// }

func (cfg *config) Connect(i int) {
	for j := 0; j < cfg.n; j++ {
		cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
		cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
		cfg.replicas[i].TestingState.IsConnected.Connected[j] = true
		cfg.replicas[j].TestingState.IsConnected.Connected[i] = true
		cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
		cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
	}
	cfg.connectedToNet[i] = true
}

func (cfg *config) Disconnect(i int) {
	for j := 0; j < cfg.n; j++ {
		cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
		cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
		cfg.replicas[i].TestingState.IsConnected.Connected[j] = false
		cfg.replicas[j].TestingState.IsConnected.Connected[i] = false
		cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
		cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
	}
	cfg.connectedToNet[i] = false
}

func MakeNetwork(n int) *network {
	var net *network = new(network)
	net.conn = make([][]*genericsmr.SimConn, n)
	for i := 0; i < n; i++ {
		net.conn[i] = make([]*genericsmr.SimConn, n)
	}
	for i := 0; i < n; i++ {
		for j := i+1; j < n; j++ {
			r1, w1 := io.Pipe()
			r2, w2 := io.Pipe()
			net.conn[i][j] = genericsmr.NewSimConn(r1, w2)
			net.conn[j][i] = genericsmr.NewSimConn(r2, w1)
		}
	}
	net.n = n
	return net
}

type config struct {
	mu        sync.Mutex
	t         *testing.T
	net 	  *network
	n         int
	replicas  []*Replica
	connectedToNet []bool
	requestChan   []chan state.Command
	responseChan  chan genericsmr.RepCommand
	repExecutions [][]state.Command
}

// this is for testing purposes only
func make_config(t *testing.T, n int, unreliable bool) *config {
	runtime.GOMAXPROCS(4)
	cfg := &config{
		t: t,
		net: MakeNetwork(n),
		n: n,
		replicas: make([]*Replica, n),
		connectedToNet: make([]bool, n),
		requestChan: make([]chan state.Command, n),
		responseChan: make(chan genericsmr.RepCommand),	
		repExecutions: make([][]state.Command, n),
	}

	for i := 0; i < n; i++ {
		cfg.requestChan[i] = make(chan state.Command)
		cfg.repExecutions[i] = make([]state.Command, 0)
	}

	// create a full set of Rafts.
	for i := 0; i < n; i++ {
		cfg.replicas[i] = NewReplicaMoreParam(i, make([]string, n), false, false, false, false, false)
	}

	cfg.Start()

	return cfg
}

func make_config_full(t *testing.T, n int, unreliable bool, 
			electionTimeout int, heartbeatTimeout int, benOrStartTimeout int, benOrResendTimeout int) *config {
	runtime.GOMAXPROCS(4)
	cfg := &config{
		t: t,
		net: MakeNetwork(n),
		n: n,
		replicas: make([]*Replica, n),
		connectedToNet: make([]bool, n),
		requestChan: make([]chan state.Command, n),
		responseChan: make(chan genericsmr.RepCommand),	
		repExecutions: make([][]state.Command, n),
	}

	for i := 0; i < n; i++ {
		cfg.requestChan[i] = make(chan state.Command)
		cfg.repExecutions[i] = make([]state.Command, 0)
	}

	// create a full set of Rafts.
	for i := 0; i < n; i++ {
		cfg.replicas[i] = NewReplicaFullParam(i, make([]string, n), false, false, false, false, false, 
							electionTimeout, heartbeatTimeout, benOrStartTimeout, benOrResendTimeout)
	}

	cfg.Start()

	return cfg
}

func (cfg *config) Start() {
	// connect replicas
	for i := 0; i < cfg.n; i++ {
		for j := 0; j < cfg.n; j++ {
			if i != j {
				cfg.replicas[i].ConnectToPeersSim(j, cfg.net.conn[i][j])
			}
		}
		cfg.connectedToNet[i] = true
	}

	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].TestingState.RequestChan = cfg.requestChan[i]
		cfg.replicas[i].TestingState.ResponseChan = cfg.responseChan
	}

	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].ConnectListenToPeers()
	}

	go cfg.listen()
}

func (cfg *config) listen() {
	for {
		data := <-cfg.responseChan
		replica := data.ServerId
		cmd := data.Command
		cfg.repExecutions[replica] = append(cfg.repExecutions[replica], cmd)
		idx := len(cfg.repExecutions[replica]) - 1

		for i := 0; i < cfg.n; i++ {
			if idx < len(cfg.repExecutions[i]) && cfg.repExecutions[i][idx] != cmd {
				cfg.t.Fatal("Replica ", replica, " executed ", cmd, " at index ", idx, " but replica ", i, " executed ", cfg.repExecutions[i][idx])
			}
		}
	}
}

func (cfg *config) runReplicas() {
	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].runReplica()
	}
}

// check that there's exactly one leader.
// try a few times in case re-elections are needed.
func (cfg *config) checkOneLeader() int {
	for iters := 0; iters < 10; iters++ {
		time.Sleep(500 * time.Millisecond)
		leaders := make(map[int][]int)
		for i := 0; i < cfg.n; i++ {
			if cfg.connectedToNet[i] {
				if leader, term, _, _ := cfg.replicas[i].getState(); leader {
					leaders[term] = append(leaders[term], i)
				}
			}
		}

		lastTermWithLeader := -1
		for t, leaders := range leaders {
			if len(leaders) > 1 {
				cfg.t.Fatalf("Term %d has %d (>1) leaders\n", t, len(leaders))
			}
			if t > lastTermWithLeader {
				lastTermWithLeader = t
			}
		}

		if len(leaders) != 0 {
			return leaders[lastTermWithLeader][0]
		}
	}
	cfg.t.Fatal("expected one leader, got none")
	return -1
}