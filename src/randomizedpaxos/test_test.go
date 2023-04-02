package randomizedpaxos

import (
	"fmt"
	"testing"
	"time"
)

// this defines the tests that are run

// The tester generously allows solutions to complete elections in one second
// (much more than the paper's range of timeouts).
const RaftElectionTimeout = 1000 * time.Millisecond

func TestInitialElection(t *testing.T) {
	servers := 3
	cfg := make_config(t, servers, false)
	// defer cfg.cleanup()

	fmt.Println("Test: initial election...")

	// // is a leader elected?
	// cfg.checkOneLeader()

	// // does the leader+Term stay the same there is no failure?
	// term1 := cfg.checkTerms()
	// time.Sleep(2 * RaftElectionTimeout)
	// term2 := cfg.checkTerms()
	// if term1 != term2 {
	// 	fmt.Println("warning: Term changed even though there were no failures")
	// }

	// cfg.replicas[0].PeerWriters[1].Write([]byte("hello"))
	// cfg.replicas[0].PeerWriters[1].Flush()

	// cfg.replicas[0].PeerWriters[1].Write([]byte(" world"))
	// cfg.replicas[0].PeerWriters[1].Flush()

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

	fmt.Println("... Passed")
}