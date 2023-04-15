package randomizedpaxos

import (
	"fmt"
	"math/rand"
	"state"
	"testing"
	"time"
)

// this defines the tests that are run

// The tester generously allows solutions to complete elections in one second
// (much more than the paper's range of timeouts).
const electionTimeout = 150
const heartbeatTimeout = 15
const benOrStartTimeout = 30
const benOrResendTimeout = 15

func TestInitialElection(t *testing.T) {
	// replica := NewReplicaMoreParam(0, make([]string, 3), false, false, false, false, false)
	// setTimer(replica.heartbeatTimer, time.Duration(6000)*time.Millisecond)

	// <-replica.heartbeatTimer.timer.C
	// fmt.Println("hello")

	// dlog.Println("TEST")

	servers := 3
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, 1e9, 1e9)
	cfg.runReplicas()
	defer cfg.cleanup()

	// defer cfg.cleanup()

	fmt.Println("Test: initial election...")

	// is a leader elected?
	leader1 := cfg.checkOneLeader()
	cfg.disconnect(leader1)

	time.Sleep(4 * electionTimeout)

	cfg.checkOneLeader()
	cfg.connect(leader1)

	time.Sleep(4 * electionTimeout)

	cfg.checkOneLeader()

	fmt.Println("... Passed")
}

func TestBasicAgree(t *testing.T) {
	servers := 3
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, 1e9, 1e9)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: basic agreement...")
	cfg.checkOneLeader()

	fmt.Println("Testing agreement on 3 entries...")
	iters := 3
	for index := 1; index < iters+1; index++ {
		res := cfg.sendCommandLeaderCheckReplicas(index * 100, servers)
		if !res {
			t.Fatal("Failed agreement on entry")
		}
	}

	fmt.Println(cfg.checkLogData())

	fmt.Println("... Passed")
}

func assert(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Fatal(msg)
	}
}

func TestPQ(t *testing.T) {
	pq := newExtendedPriorityQueue()
	pq.push(Entry{Data: state.Command{}, SenderId: 1, Timestamp: 5})
	pq.push(Entry{Data: state.Command{}, SenderId: 2, Timestamp: 3})

	fmt.Println(logToString(pq.extractList()))
}

func TestFailAgree(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, 1e9, 1e9)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: fail agree...")
	leader := cfg.checkOneLeader()

	cfg.disconnect((leader + 1) % 5)
	cfg.disconnect((leader + 2) % 5)

	fmt.Println("Testing agreement on 3 replicas (fail replica)...")
	iters := 3
	index := 1
	for ; index < iters+1; index++ {
		res := cfg.sendCommandLeader(index * 100)
		if !res {
			t.Fatal("Failed agreement on entry")
		}
	}
	fmt.Println(cfg.checkLogData())

	cfg.connect((leader + 1) % 5)
	cfg.connect((leader + 2) % 5)
	time.Sleep(4 * electionTimeout)

	leader = cfg.checkOneLeader()
	cfg.disconnect((leader+1) % 5)
	cfg.disconnect((leader+2) % 5)

	fmt.Println("Testing agreement on 3333 replicas (fail leader)...")
	iters = 6
	for ; index < iters+1; index++ {
		res := cfg.sendCommandLeader(index * 100)
		if !res {
			t.Fatal("Failed agreement on entry")
		}
	}
	assert(t, len(cfg.checkLogData()) == 6, "Log length is not 6")

	cfg.disconnect((leader+3) % 5)
	fmt.Println("Testing no agreement on asdf replicas (fail leader)...")
	iters = 8
	for ; index < iters+1; index++ {
		res := cfg.sendCommandCheckCommit((leader+4)%5, index * 100)
		if res {
			t.Fatal("Reached agreement even though majority of replicas are down")
		}
	}

	fmt.Println("Reconnecting everyone")
	cfg.connect((leader+1) % 5)
	cfg.connect((leader+2) % 5)
	cfg.connect((leader+3) % 5)

	fmt.Println("Testing agreement on 5 replicas...")
	time.Sleep(4 * electionTimeout)
	cfg.checkOneLeader()
	// fmt.Println(cfg.checkLogData())
	// assert(t, len(cfg.checkLogData()) == 6, "Log length is not 6")

	fmt.Println("... Passed")
}

func TestBenOrSimple(t *testing.T) {
	servers := 3
	cfg := make_config_full(t, servers, false, 1e9, 1e9, benOrStartTimeout, benOrResendTimeout)
	cfg.runReplicas()
	defer cfg.cleanup()
	fmt.Println("Test: ben or simple...")

	res := cfg.sendCommandCheckCommit(0, 100)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	fmt.Println("... Passed")
}

func TestBenOrMultiple(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, 1e9, 1e9, benOrStartTimeout, benOrResendTimeout)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: ben or multiple...")

	cfg.sendCommand(0, 100)
	cfg.sendCommand(1, 200)
	cfg.sendCommand(2, 300)
	cfg.sendCommand(3, 400)
	cfg.sendCommand(4, 500)

	time.Sleep(3 * time.Second)
	// fmt.Println("SLEEP DONE")
	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))
	// fmt.Println("POKEMON")

	res, _ := cfg.sendCommandCheckResults(0, 600, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	commands = cfg.checkLogData()
	assert(t, len(commands) == 6, "Log length is not 6")
	fmt.Println(commandToString(commands))

	fmt.Println("... Passed")
}

func TestBenOrRapidEntries(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, true, 1e9, 1e9, benOrStartTimeout, 10)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: ben or multiple...")
	fmt.Println("Test 6 commands")

	oldTime := time.Now()
	
	cfg.sendCommand(0, 1)
	cfg.sendCommand(1, 2)
	cfg.sendCommand(2, 3)
	cfg.sendCommand(3, 4)
	cfg.sendCommand(4, 5)

	res, newTime := cfg.sendCommandCheckResults(0, 6, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	fmt.Println("Time taken (5 commands): ", newTime.Sub(oldTime))
	time.Sleep(2 * time.Second)

	fmt.Println("Test 14 commands")

	oldTime = time.Now()
	
	cfg.sendCommand(0, 100)
	cfg.sendCommand(1, 200)
	cfg.sendCommand(2, 300)
	cfg.sendCommand(3, 400)
	cfg.sendCommand(4, 500)
	cfg.sendCommand(4, 600)
	cfg.sendCommand(3, 700)
	cfg.sendCommand(2, 800)
	cfg.sendCommand(1, 900)
	cfg.sendCommand(0, 1000)
	cfg.sendCommand(2, 1100)
	cfg.sendCommand(1, 1200)
	cfg.sendCommand(0, 1300)

	res, newTime = cfg.sendCommandCheckResults(0, 1400, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	fmt.Println("Time taken (14 commands): ", newTime.Sub(oldTime))

	commands := cfg.checkLogData()
	assert(t, len(commands) == 20, "Log length is not 20")
	fmt.Println(commandToString(commands))

	fmt.Println("... Passed")
}

func TestFigure8(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, 1e9, 1e9)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: figure 8...")
	res := cfg.sendCommandCheckCommit(0, -1)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	nup := servers
	for iters := 0; iters < 100; iters++ {
		leader := -1
		for i := 0; i < servers; i++ {
			isLeader := cfg.sendCommand(i, iters*servers+i)
			// _, _, ok := cfg.rafts[i].Start(rand.Int() % 10000)
			// if ok && cfg.connected[i] {
			// 	leader = i
			// }
			if isLeader && cfg.connectedToNet[i] {
				leader = i
			}
		}

		if (rand.Int() % 1000) < 100 {
			ms := rand.Int63() % (1000 / 2)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		} else {
			ms := (rand.Int63() % 13)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}

		if leader != -1 && (rand.Int()%1000) < int(1000)/2 {
			cfg.disconnect(leader)
			nup -= 1
		}

		if nup < 3 {
			rep := rand.Int() % servers
			if cfg.connectedToNet[rep] == false {
				cfg.connect(rep)
				nup += 1
			}
		}
	}

	for i := 0; i < servers; i++ {
		if cfg.connectedToNet[i] == false {
			cfg.connect(i)
		}
	}

	time.Sleep(1 * time.Second)
	res = cfg.sendCommandLeaderCheckReplicas(1000, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))
	assert(t, len(commands) == 502, "Log length is not 502")

	loc := make([]int, 5)
	for i := 0; i < 500; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := opId % servers
		if opId < loc[pos] {
			t.Fatal("Out of order")
		}
		loc[pos] = opId
	}

	fmt.Println("None out of order")

	fmt.Println("... Passed")
}

func TestBenOrManyReconnects(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, true, 1e9, 1e9, benOrStartTimeout, 10)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: ben or many reconnects...")
	res := cfg.sendCommandCheckCommit(0, -1)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	nup := servers
	for iters := 0; iters < 100; iters++ {
		for i := 0; i < servers; i++ {
			cfg.sendCommand(i, iters*servers+i)
			fmt.Println("Just sent", iters*servers+i, "to", i)
			// _, _, ok := cfg.rafts[i].Start(rand.Int() % 10000)
			// if ok && cfg.connected[i] {
			// 	leader = i
			// }
		}

		if (rand.Int() % 1000) < 100 {
			ms := rand.Int63() % (1000 / 2)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		} else {
			ms := (rand.Int63() % 13)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}

		if nup > 2 {
			rep := rand.Int() % servers
			if cfg.connectedToNet[rep] && (rand.Int()%1000) < int(1000)/2 {
				cfg.disconnect(rep)
				nup -= 1
				continue
			}
		}

		if nup < 3 {
			rep := rand.Int() % servers
			if cfg.connectedToNet[rep] == false {
				cfg.connect(rep)
				nup += 1
			}
		}
	}

	for i := 0; i < servers; i++ {
		fmt.Println("Replica", i, "execution log length: ", len(cfg.repExecutions[i]), ", connected status", cfg.connectedToNet[i])
	}

	for i := 0; i< servers; i++ {
		for j := 0; j < servers; j++ {
			fmt.Println("Connection from", i, "to", j, "is", cfg.replicas[i].TestingState.IsConnected.Connected[j])
		}
	}

	for nup < 3 {
		rep := rand.Int() % servers
		if cfg.connectedToNet[rep] == false {
			cfg.connect(rep)
			nup += 1
		}
	}

	time.Sleep(20 * time.Second)

	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))

	for i := 0; i < servers; i++ {
		fmt.Println("Replica", i, "execution log length: ", len(cfg.repExecutions[i]), ", connected status", cfg.connectedToNet[i])
	}

	for i := 0; i < servers; i++ {
		if cfg.connectedToNet[i] == false {
			cfg.connect(i)
		}
	}

	time.Sleep(10 * time.Second)

	commands = cfg.checkLogData()
	fmt.Println(commandToString(commands))
	assert(t, len(commands) == 501, "Log length is not 501")

	loc := make([]int, 5)
	for i := 0; i < 500; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := opId % servers
		if opId < loc[pos] {
			t.Fatal("Out of order")
		}
		loc[pos] = opId
	}

	fmt.Println("None out of order")

	res, _ = cfg.sendCommandCheckResults(0, 1000, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	fmt.Println("... Passed")
}

	// asdf := newExtendedPriorityQueue()

	// k := Entry{Data: state.Command{}, SenderId: 1, Timestamp: 4}

	// asdf.push(k)
	// asdf.push(Entry{Data: state.Command{}, SenderId: 1, Timestamp: 5})
	// asdf.push(Entry{Data: state.Command{}, SenderId: 2, Timestamp: 3})

	// asdf.remove(k)
	// fmt.Println(asdf.pop())
	// fmt.Println(asdf.pop())

	// fmt.Println(asdf.isEmpty())
	// // fmt.Println(asdf.pop())

	// fmt.Println(asdf.isEmpty())


	// r1, w1 := io.Pipe()
	// r2, w2 := io.Pipe()
	// c1 := genericsmr.NewSimConn(r1, w2)
	// c2 := genericsmr.NewSimConn(r2, w1)

	// c1.Connect()
	// c2.Connect()

	// // b1r := bufio.NewReader(c1.PipeReader)
	// b1w := bufio.NewWriter(c1)
	// b2r := bufio.NewReader(c2)
	// // b2w := bufio.NewWriter(c2.PipeWriter)

	// go func() {
	// 	b1w.WriteString("hello")
	// 	b1w.Flush()
	// }()

	// buf := make([]byte, 10)
	// by, _ := b2r.ReadByte()

	// n, err := b2r.Read(buf)
	// fmt.Println(string(by))
	// fmt.Println(n, err, string(buf[:n]))

	// time.Sleep(100 * time.Second)



	    // // create a pipe
		// r, w := io.Pipe()

		// // example of concurrent reading and writing
		// go func() {
		// 	// write some data to the pipe
		// 	_, err := w.Write([]byte("strange"))
		// 	if err != nil {
		// 		panic(err)
		// 	}
	
		// 	// don't close the writer
		// 	w.Close()
		// }()
	
		// // read from the pipe
		// time.Sleep(1 * time.Second)
		// buf := make([]byte, 10)
		// for {
		// 	n, err := r.Read(buf)
		// 	fmt.Println("err", err)
		// 	if err != nil {
		// 		// panic(err)
		// 		break
		// 	}
		// 	message := string(buf[:n])
		// 	println(message)
		// }
	
		// // sleep to simulate some work being done
		// time.Sleep(1 * time.Second)
		
	// a, b := io.Pipe()
	// b.Write([]byte("hello"))
	// var data []byte
	// n, err = a.Read(data)
	// fmt.Println(n, err, data)

	// // is a leader elected?
	// cfg.checkOneLeader()

	// // does the leader+Term stay the same there is no failure?
	// term1 := cfg.checkTerms()
	// time.Sleep(2 * RaftElectionTimeout)
	// term2 := cfg.checkTerms()
	// if term1 != term2 {
	// 	fmt.Println("warning: Term changed even though there were no failures")
	// }

	// time.Sleep(5 * time.Second)

	

	// servers := 3
	// cfg := make_config(t, servers, false)
	// // defer cfg.cleanup()

	// fmt.Println("Test: initial election...")

	// args := &randomizedpaxosproto.InfoBroadcastReply{
	// 	SenderId: 0, Term: 100}
	// cfg.replicas[0].SendMsg(1, cfg.replicas[0].infoBroadcastReplyRPC, args)

	// cfg.Disconnect(0, 1)

	// args = &randomizedpaxosproto.InfoBroadcastReply{
	// 	SenderId: 0, Term: 300}
	// cfg.replicas[0].SendMsg(1, cfg.replicas[0].infoBroadcastReplyRPC, args)

	// cfg.Connect(0, 1)

	// args = &randomizedpaxosproto.InfoBroadcastReply{
	// 	SenderId: 0, Term: 500}
	// cfg.replicas[0].SendMsg(1, cfg.replicas[0].infoBroadcastReplyRPC, args)

	// time.Sleep(2000 * time.Second)







	// infoBroadcastReplyS := <- cfg.replicas[1].infoBroadcastReplyChan
	
	// fmt.Println(infoBroadcastReplyS)

	// cfg.replicas[0].PeerWriters[1].Write([]byte("hello"))
	// cfg.replicas[0].PeerWriters[1].Flush()

	// cfg.replicas[0].PeerWriters[1].Write([]byte(" world"))
	// cfg.replicas[0].PeerWriters[1].Flush()

	// <- cfg.replicas[1].infoBroadcastReplyChan

	// fmt.Println(cfg.replicas[1].PeerReaders[0].Buffered())
	// _, err := cfg.replicas[1].PeerReaders[0].Peek(1)
	// if err != nil {
	// 	fmt.Println("asdf ", err)
	// }

	// bs, _ := io.ReadAll(cfg.replicas[1].PeerReaders[0])
	// fmt.Println(string(bs))

	// bs, _ = io.ReadAll(cfg.replicas[1].PeerReaders[0])
	// fmt.Println(string(bs))

	// a := genericsmr.NewSimConn()
	// b := bufio.NewReader(a)
	// c := bufio.NewWriter(a)
	// c.Write([]byte("hello"))
	// c.Flush()
	// bs, _ = io.ReadAll(b)
	// fmt.Println(string(bs))