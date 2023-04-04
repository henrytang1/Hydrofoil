package randomizedpaxos

import (
	"genericsmr"
	"io"
	"runtime"
	"sync"
	"testing"
)

type network struct {
	conn	  [][]*genericsmr.SimConn // unidirectional connection from i to j
	n		  int
}

// func (net *network) Connect(i, j int) {
// 	net.conn[i][j].Connect()
// 	net.conn[j][i].Connect()
// }

func (cfg *config) Connect(i, j int) {
	cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
	cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
	cfg.replicas[i].TestingState.IsConnected.Connected[j] = true
	cfg.replicas[j].TestingState.IsConnected.Connected[i] = true
	cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
	cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
}

func (cfg *config) Disconnect(i, j int) {
	cfg.replicas[i].TestingState.IsConnected.Mu.Lock()
	cfg.replicas[j].TestingState.IsConnected.Mu.Lock()
	cfg.replicas[i].TestingState.IsConnected.Connected[j] = false
	cfg.replicas[j].TestingState.IsConnected.Connected[i] = false
	cfg.replicas[i].TestingState.IsConnected.Mu.Unlock()
	cfg.replicas[j].TestingState.IsConnected.Mu.Unlock()
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
	// net       *labrpc.Network
	net 	  *network
	n         int
	// done      int32 // tell internal threads to die
	replicas  []*Replica
	// applyErr  []string // from apply channel readers
	// connected []bool   // whether each server is on the net
	// // saved     []*Persister
	// endnames  [][]string    // the port file names each sends to
	// logs      []map[int]int // copy of each server's committed entries
}

// this is for testing purposes only
func make_config(t *testing.T, n int, unreliable bool) *config {
	runtime.GOMAXPROCS(4)
	cfg := &config{}
	cfg.t = t
	cfg.net = MakeNetwork(n)
	cfg.n = n
	// cfg.applyErr = make([]string, cfg.n)
	cfg.replicas = make([]*Replica, cfg.n)
	// cfg.connected = make([]bool, cfg.n)
	// cfg.saved = make([]*Persister, cfg.n)
	// cfg.endnames = make([][]string, cfg.n)
	// cfg.logs = make([]map[int]int, cfg.n)

	// cfg.setunreliable(unreliable)

	// cfg.net.LongDelays(true)

	// create a full set of Rafts.
	for i := 0; i < n; i++ {
		// cfg.logs[i] = map[int]int{}
		rep := NewReplica(i, make([]string, n), false, false, false, false, false)
		// rep := Make(ends, i, cfg.saved[i], applyCh)
		cfg.replicas[i] = rep
	}

	// connect everyone
	// for i := 0; i < cfg.n; i++ {
	// 	for j := 0; j < cfg.n; j++ {
	// 		cfg.replicas[i].IsConnected.Mu.Lock()
	// 		cfg.replicas[j].IsConnected.Mu.Lock()
	// 		cfg.replicas[i].IsConnected.Connected[j] = true
	// 		cfg.replicas[j].IsConnected.Connected[i] = true
	// 		cfg.replicas[i].IsConnected.Mu.Unlock()
	// 		cfg.replicas[j].IsConnected.Mu.Unlock()
	// 	}
	// }

	for i := 0; i < cfg.n; i++ {
		for j := 0; j < cfg.n; j++ {
			if i != j {
				cfg.replicas[i].ConnectToPeersSim(j, cfg.net.conn[i][j])
			}
		}
	}

	for i := 0; i < cfg.n; i++ {
		cfg.replicas[i].ConnectListenToPeers()
	}

	// cfg.replicas[0].PeerWriters[1].Write([]byte("hmmmmm"))
	// bs, _ := io.ReadAll(cfg.replicas[1].PeerReaders[0])
	// fmt.Println(string(bs))

	return cfg
}