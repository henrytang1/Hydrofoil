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

	fmt.Println("... Passed")
}