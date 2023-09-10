package hydrofoil

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// this defines the tests that are run

// The tester generously allows solutions to complete elections in one second
// (much more than the paper's range of timeouts).
const electionTimeout = 150
const heartbeatTimeout = 15
const benOrStartTimeout = 40
const benOrResendTimeout = 20

func assert(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Fatal(msg)
	}
}

func TestInitialElection(t *testing.T) {
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

// func TestPQ(t *testing.T) {
// 	pq := newExtendedPriorityQueue()
// 	pq.push(Entry{Data: state.Command{}, SenderId: 1, Timestamp: 5})
// 	pq.push(Entry{Data: state.Command{}, SenderId: 2, Timestamp: 3})

// 	fmt.Println(logToString(pq.extractList()))
// }

func TestFailAgree(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, 1e9, 1e9)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: fail agree...")
	leader := cfg.checkOneLeader()

	cfg.disconnect((leader + 1) % 5)
	cfg.disconnect((leader + 2) % 5)

	fmt.Println("Testing agreement on 3 replicas (disconnected replica)...")
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

	fmt.Println("Testing agreement on 3 replicas (disconnected leader)...")
	iters = 6
	for ; index < iters+1; index++ {
		res := cfg.sendCommandLeader(index * 100)
		if !res {
			t.Fatal("Failed agreement on entry")
		}
	}
	assert(t, len(cfg.checkLogData()) == 6, "Log length is not 6")

	cfg.disconnect((leader+3) % 5)
	fmt.Println("Testing no agreement on replicas (disconnected leader)...")
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
	cfg := make_config_full(t, servers, false, 1e9, 1e9, benOrStartTimeout, 10)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: ben or rapid entries...")
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

	time.Sleep(500 * time.Millisecond)

	res, newTime = cfg.sendCommandCheckResults(0, 1400, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

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
	for iters := 0; iters < 300; iters++ {
		leader := -1
		for i := 0; i < servers; i++ {
			isLeader := cfg.sendCommand(i, iters*servers+i+1)
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
	res = cfg.sendCommandLeaderCheckReplicas(10000, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	commands := cfg.checkLogData()
	fmt.Println("length: ", len(commands), "data: ", commandToString(commands))
	assert(t, len(commands) == 1502, "Log length is not 1502")

	loc := make([]int, servers)
	for i := 0; i < 1500; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := (opId + servers - 1) % servers
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
	cfg := make_config_full(t, servers, false, 1e9, 1e9, benOrStartTimeout, 10)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: ben or many reconnects...")
	res := cfg.sendCommandCheckCommit(0, -1)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	nup := servers
	for iters := 0; iters < 50; iters++ {
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

	for nup < 3 {
		rep := rand.Int() % servers
		if cfg.connectedToNet[rep] == false {
			cfg.connect(rep)
			nup += 1
		}
	}

	for i := 0; i< servers; i++ {
		for j := 0; j < servers; j++ {
			fmt.Println("Connection from", i, "to", j, "is", cfg.replicas[i].Connected[j])
		}
	}

	time.Sleep(25 * time.Second)


	for i := 0; i < servers; i++ {
		if cfg.connectedToNet[i] == false {
			cfg.connect(i)
		}
	}

	time.Sleep(3 * time.Second)

	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))
	assert(t, len(commands) == 251, "Log length is not 251")

	loc := make([]int, servers)
	for i := 0; i < 250; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := (opId + servers - 1) % servers
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

func TestRaftWithBenOrSimple(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, benOrStartTimeout, benOrResendTimeout)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: raft with ben or...")
	leader1 := cfg.checkOneLeader()

	for i := 0; i < servers; i++ {
		cfg.replicas[i].electionTimeout = 1e9
	}

	time.Sleep(1 * time.Second)

	leader2 := cfg.checkOneLeader()
	assert(t, leader1 == leader2, "Leader changed")

	// fmt.Println("Testing agreement on 3 entries...")
	for iters := 0; iters < 10; iters++ {
		for i := 0; i < servers; i++ {
			cfg.sendCommand(i, iters*servers+i+1)
			fmt.Println("Just sent", iters*servers+i+1, "to", i)
		}

		if (rand.Int() % 1000) < 100 {
			ms := rand.Int63() % (1000 / 2)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		} else {
			ms := (rand.Int63() % 13)
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}

	cfg.commandsExecuted()
	// fmt.Println("Commands of length", len(commands), ": ", commandToString(commands))

	cfg.disconnect(leader1)

	time.Sleep(5 * time.Second)

	commands := cfg.checkLogData()
	fmt.Println("Commands of length", len(commands), ": ", commandToString(commands))
	
	cfg.connect(leader1)
	time.Sleep(1 * time.Second)
	commands = cfg.checkLogData()
	fmt.Println("Commands of length", len(commands), ": ", commandToString(commands))
	assert(t, len(commands) == 50, "Log length is not 50")

	loc := make([]int, servers)
	for i := 0; i < 30; i++ {
		opId := int(commands[i].OpId) % servers
		pos := (opId + servers - 1) % servers
		if opId < loc[pos] {
			t.Fatal("Out of order")
		}
		loc[pos] = opId
	}

	fmt.Println("None out of order")

	fmt.Println("... Passed")
}

func TestRaftWithBenOrNoFailures(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, benOrStartTimeout, benOrResendTimeout)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: raft with ben or no failures...")
	res := cfg.sendCommandCheckCommit(0, -1)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	// for iters := 0; iters < 1000; iters++ {
	for iters := 0; iters < 300; iters++ {
		for i := 0; i < servers; i++ {
			cfg.sendCommand(i, iters*servers+i+1)
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
	}

	time.Sleep(20 * time.Second)
	res = cfg.sendCommandLeaderCheckReplicas(10000, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))
	assert(t, len(commands) >= 1501, "Log length is larger than 1501")

	for i := 1501; i < len(commands); i++ {
		assert(t, commands[i].OpId >= 10000, "Log entry isn't at least 10000")
	}

	loc := make([]int, servers)
	for i := 0; i < 1500; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := (opId + servers - 1) % servers
		if opId < loc[pos] {
			t.Fatal("Out of order")
		}
		loc[pos] = opId
	}

	fmt.Println("None out of order")

	fmt.Println("... Passed")
}

func TestRaftWithBenOrComplex(t *testing.T) {
	servers := 5
	cfg := make_config_full(t, servers, false, electionTimeout, heartbeatTimeout, benOrStartTimeout, benOrResendTimeout)
	cfg.runReplicas()
	defer cfg.cleanup()

	fmt.Println("Test: raft with ben or complex...")
	res := cfg.sendCommandCheckCommit(0, -1)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	nup := servers
	for iters := 0; iters < 300; iters++ {
	// for iters := 0; iters < 300; iters++ {
		leader := -1
		for i := 0; i < servers; i++ {
			isLeader := cfg.sendCommand(i, iters*servers+i+1)
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

	time.Sleep(10 * time.Second)
	res = cfg.sendCommandLeaderCheckReplicas(10000, 5)
	if !res {
		t.Fatal("Failed agreement on entry")
	}

	commands := cfg.checkLogData()
	fmt.Println(commandToString(commands))
	assert(t, len(commands) >= 1501, "Log length is larger than 1501")

	for i := 1501; i < len(commands); i++ {
		assert(t, commands[i].OpId >= 10000, "Log entry isn't at least 10000")
	}

	loc := make([]int, servers)
	for i := 0; i < 1500; i++ {
		opId := int(commands[i+1].OpId) % servers
		pos := (opId + servers - 1) % servers
		if opId < loc[pos] {
			t.Fatal("Out of order")
		}
		loc[pos] = opId
	}

	fmt.Println("None out of order")

	fmt.Println("... Passed")
}