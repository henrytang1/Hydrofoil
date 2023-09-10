package hydrofoil

import (
	"dlog"
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

func (cfg *config) connect(i int) {
	for j := 0; j < cfg.n; j++ {
		if i != j && cfg.connectedToNet[j] {
			cfg.replicas[i].Connected[j] = true
			cfg.replicas[j].Connected[i] = true
		}
	}
	cfg.connectedToNet[i] = true
	dlog.Println("Server", i, " connected to network")
}

func (cfg *config) disconnect(i int) {
	for j := 0; j < cfg.n; j++ {
		if i != j {
			cfg.replicas[i].Connected[j] = false
			cfg.replicas[j].Connected[i] = false
		}
	}
	cfg.connectedToNet[i] = false
	dlog.Println("Server", i, " disconnected from network")
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
func make_config(t *testing.T, n int, reliable bool) *config {
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

	cfg.start(reliable)

	return cfg
}

func make_config_full(t *testing.T, n int, reliable bool, 
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

	cfg.start(reliable)

	return cfg
}

func (cfg *config) start(reliable bool) {
	// connect replicas
	for i := 0; i < cfg.n; i++ {
		for j := 0; j < cfg.n; j++ {
			if i != j {
				cfg.replicas[i].ConnectToPeersSim(j, cfg.net.conn[i][j], reliable)
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
	for data := range cfg.responseChan {
		replica := data.ServerId
		cmd := data.Command
		// dlog.Println("Got execution", cmd.OpId, "from replica", replica, "before append")
		cfg.repExecutions[replica] = append(cfg.repExecutions[replica], cmd)
		// dlog.Println("Got execution", cmd.OpId, "from replica", replica, "after append")
		// idx := len(cfg.repExecutions[replica]) - 1
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

func (cfg *config) commandsExecuted() []state.Command {
	maxLength := 0
	commands := make([]state.Command, 0)
	for i := 0; i < cfg.n; i++ {
		if len(cfg.repExecutions[i]) > maxLength {
			maxLength = len(cfg.repExecutions[i])
			commands = cfg.repExecutions[i]
		}
		dlog.Println("Replica ", i, " execution log: ", commandToString(cfg.repExecutions[i]))
	}
	return commands
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
						dlog.Println("Replica ", j, " log: ", commandToString(cfg.repExecutions[j]))
					}
					cfg.t.Fatal("Log data is not the same for all replicas")
				}
			}
		}
	}
	return logData
}

func (cfg *config) sendCommand(rep int, cmdId int) bool {
	dlog.Println(cmdId)
	cmd := state.Command{ClientId: CLIENTID, OpId: int32(cmdId), Op: state.PUT, K: 0, V: 0}
	dlog.Println("SEND COMMAND: ", cmd.OpId, " TO ", rep, "START")
	cfg.requestChan[rep] <- cmd
	dlog.Println("SEND COMMAND: ", cmd.OpId, " TO ", rep, "END")
	isLeader, _, _, _ := cfg.replicas[rep].getState()
	return isLeader
}

func (cfg *config) cleanup() {
	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].shutdown()
	}
}

func (cfg *config) sendCommandCheckResults(rep int, cmdId int, expectedReplicas int) (bool, time.Time) {
	cfg.sendCommand(rep, cmdId)
	t1 := time.Now()
	dlog.Println("TIME BEGIN: ", t1, time.Now().UnixMilli())
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
		if numFound >= expectedReplicas {
			dlog.Println("TIME END: ", time.Since(t1).Milliseconds())
			return true, time.Now()
		}
		time.Sleep(3 * time.Millisecond)
	}
	dlog.Println("TIME END: ", time.Since(t1).Milliseconds())
	return false, time.Now()
}

func (cfg *config) sendCommandCheckCommit(rep int, cmdId int) bool {
	res, _ := cfg.sendCommandCheckResults(rep, cmdId, cfg.n/2 + 1)
	return res
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
					dlog.Println("Sent command", cmdId, "to", starts)
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
		dlog.Println("Replica ", j, " log: ", commandToString(cfg.repExecutions[j]))
	}
	return false
}

func (cfg *config) sendCommandLeader(cmdId int) bool {
	return cfg.sendCommandLeaderCheckReplicas(cmdId, cfg.n/2+1)
}