package randomizedpaxos

import (
	"fmt"
	"genericsmr"
	"io"
	"reflect"
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

func (cfg *config) connect(i int) {
	for j := 0; j < cfg.n; j++ {
		if i != j {
			cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
			cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
			cfg.replicas[i].TestingState.IsConnected.Connected[j] = true
			cfg.replicas[j].TestingState.IsConnected.Connected[i] = true
			cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
			cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
		}
	}
	cfg.connectedToNet[i] = true
	fmt.Println("Server", i, " connected to network")
}

func (cfg *config) disconnect(i int) {
	for j := 0; j < cfg.n; j++ {
		if i != j {
			cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
			cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
			cfg.replicas[i].TestingState.IsConnected.Connected[j] = false
			cfg.replicas[j].TestingState.IsConnected.Connected[i] = false
			cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
			cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
		}
	}
	cfg.connectedToNet[i] = false
	fmt.Println("Server", i, " disconnected from network")
}

func makeNetwork(n int) *network {
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
		net: makeNetwork(n),
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
		cfg.replicas[i] = newReplicaMoreParam(i, make([]string, n), false, false, false, false, false)
	}

	cfg.start()

	return cfg
}

func make_config_full(t *testing.T, n int, unreliable bool, 
			electionTimeout int, heartbeatTimeout int, benOrStartTimeout int, benOrResendTimeout int) *config {
	runtime.GOMAXPROCS(4)
	cfg := &config{
		t: t,
		net: makeNetwork(n),
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
		cfg.replicas[i] = newReplicaFullParam(i, make([]string, n), false, false, false, false, false, 
							electionTimeout, heartbeatTimeout, benOrStartTimeout, benOrResendTimeout)
	}

	cfg.start()

	return cfg
}

func (cfg *config) start() {
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

func (cfg *config) checkLogData() []state.Command {
	var logData []state.Command
	logData = nil
	for i := 0; i < cfg.n; i++ {
		if cfg.connectedToNet[i] {
			if logData == nil {
				logData = cfg.repExecutions[i]
			} else {
				if !reflect.DeepEqual(logData, cfg.repExecutions[i]) {
					for j := 0; j < cfg.n; j++ {
						fmt.Println("Replica ", j, " log: ", commandToString(cfg.repExecutions[j]))
					}
					cfg.t.Fatal("Log data is not the same for all replicas")
				}
			}
		}
	}
	return logData
}

func (cfg *config) sendCommand(rep int, cmdId int) {
	cmd := state.Command{ClientId: CLIENTID, OpId: int32(cmdId), Op: state.PUT, K: 0, V: 0}
	cfg.requestChan[rep] <- cmd
}

func (cfg *config) cleanup() {
	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].shutdown()
	}
}

func (cfg *config) sendCommandReplica(rep int, cmdId int) bool {
	cfg.sendCommand(rep, cmdId)
	t1 := time.Now()
	for time.Since(t1).Seconds() < 2 {
		numFound := 0
		for i := 0; i < cfg.n; i++ {
			logData := cfg.repExecutions[i]
			for j := 0; j < len(logData); j++ {
				if logData[j].ClientId == CLIENTID && logData[j].OpId == int32(cmdId) {
					numFound++
				}
			}
		}
		if numFound >= cfg.n/2+1 {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func (cfg *config) sendCommandLeaderCheckReplicas(cmdId int, expectedReplicas int) bool {
	t0 := time.Now()
	starts := 0
	inc := 0
	for time.Since(t0).Seconds() < 5 {
		// try all the servers, maybe one is the leader.
		index := -1
		for si := 0; si < cfg.n; si++ {
			starts = (starts + 1) % cfg.n
			if cfg.connectedToNet[starts] {
				isLeader, _, _, _ := cfg.replicas[starts].getState()
				if isLeader {
					cfg.sendCommand(starts, cmdId + inc)
					fmt.Println("Sent command", cmdId, "to", starts)
					index = starts
					break
				}
			}
		}

		if index != -1 {
			// somebody claimed to be the leader and to have
			// submitted our command; wait a while for agreement.
			t1 := time.Now()
			for time.Since(t1).Seconds() < 2 {
				numFound := 0
				for i := 0; i < cfg.n; i++ {
					logData := cfg.repExecutions[i]
					for j := 0; j < len(logData); j++ {
						if logData[j].ClientId == CLIENTID && logData[j].OpId == int32(cmdId + inc) {
							numFound++
						}
					}
				}
				if numFound >= expectedReplicas {
					return true
				}
				time.Sleep(20 * time.Millisecond)
			}
			inc++
		} else {
			time.Sleep(50 * time.Millisecond)
		}
	}
	for j := 0; j < cfg.n; j++ {
		fmt.Println("Replica ", j, " log: ", cfg.repExecutions[j])
	}
	return false
}

func (cfg *config) sendCommandLeader(cmdId int) bool {
	return cfg.sendCommandLeaderCheckReplicas(cmdId, cfg.n/2+1)
}