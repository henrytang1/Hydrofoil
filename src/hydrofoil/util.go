package hydrofoil

import (
	"dlog"
	"log"
	"math/rand"
	"state"
	"strconv"
	"time"
)

type ClientReqStatus struct {
	logOccurrences	int
	committed	bool
	executed 	bool
}

type Set struct {
	m map[UniqueCommand]ClientReqStatus
}

func newSet() Set {
	return Set{make(map[UniqueCommand]ClientReqStatus)}
}

func (s *Set) dec(item UniqueCommand) {
	status, found := s.m[item]
	if !found {
		log.Fatal("dec: not found")
	} else {
		status.logOccurrences--
		s.m[item] = status
	}
}

func (s *Set) add(entry Entry) {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, false, false}
	} else {
		status.logOccurrences++
		s.m[item] = status
	}
}

func (s *Set) remove(entry Entry) {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	if item.senderId == -1 {
		return
	}
	delete(s.m, item)
}

func (s *Set) contains(entry Entry) bool {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		return false
	}
	return status.logOccurrences > 0
}

func (s *Set) commit(entry Entry) {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, true, false}
	} else {
		status.committed = true
		s.m[item] = status
	}
}

func (s *Set) execute(entry Entry) {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, true, true}
	} else {
		status.executed = true
		s.m[item] = status
	}
}

func (s *Set) isCommitted(entry Entry) bool {
	item := UniqueCommand{senderId: entry.ServerId, time: entry.Timestamp}
	if _, ok := s.m[item]; !ok {
		return false
	}
	return s.m[item].committed
}

func (s *Set) commitSlice(entries []Entry) {
	for _, entry := range entries {
		s.commit(entry)
	}
}

func (r *Replica) seenBefore (entry Entry) bool {
	return r.inLog.contains(entry) || r.pq.contains(entry)
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

func (r *Replica) benOrRunning() bool {
	return (r.benOrState.benOrStage == Broadcasting || 
		r.benOrState.benOrStage == StageOne || 
		r.benOrState.benOrStage == StageTwo)
}

type ServerTimer struct {
	active bool // true if the timer is currently active
	timer *time.Timer
}

func newTimer() *ServerTimer {
	t := ServerTimer{active: false, timer: time.NewTimer(0)}
	// initialize these timers, since otherwise calling timer.Reset() on them will panic
	<-t.timer.C
	return &t
}

// Sets timer to duration. If timer is already active, it is stopped and reset.
func setTimer(t *ServerTimer, d time.Duration) {
	if t.active {
		if !t.timer.Stop() {
			<-t.timer.C
		}
	}
	t.timer.Reset(d)
	t.active = true
}

func clearTimer(t *ServerTimer) {
	if t.active {
		if !t.timer.Stop() {
			<-t.timer.C
		}
	}
	t.active = false
}

func (r *Replica) handleIncomingTerm(rpc RPC) {
	if r.term < int(rpc.GetTerm()) {
		r.term = int(rpc.GetTerm())
		if r.leaderState.isLeader {
			dlog.Println("Leader", r.Id, "became follower")
			r.leaderState = emptyLeaderState
			clearTimer(r.heartbeatTimer)

			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

			timeout = rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
			setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
		}
		if r.candidateState.isCandidate {
			r.candidateState = emptyCandidateState
		}
		r.votedFor = -1
	}
}

const ( // for benOrStatus
	LessUpToDate uint8	= 0
	EquallyUpToDate	   	= 1
	MoreUpToDate	   	= 2
)

func convertBoolToOrder (b bool) uint8 {
	if b { return MoreUpToDate }
	return LessUpToDate
}

func (r *Replica) isLogMoreUpToDate(rpc RPC) uint8 {
	if r.commitIndex != int(rpc.GetCommitIndex()) {
		return convertBoolToOrder(r.commitIndex > int(rpc.GetCommitIndex()))
	}
	if r.leaderTerm != int(rpc.GetLeaderTerm()) {
		return convertBoolToOrder(r.leaderTerm > int(rpc.GetLeaderTerm()))
	}
	if len(r.log) != int(rpc.GetLogLength()) {
		return convertBoolToOrder(len(r.log) > int(rpc.GetLogLength()))
	}
	return EquallyUpToDate
}

func entryEqual (e1 Entry, e2 Entry) bool {
	return e1.ServerId == e2.ServerId && e1.Timestamp == e2.Timestamp
}

func logToString (log []Entry) string {
	var s string = ""
	for i := 0; i < len(log); i++ {
		s += strconv.Itoa(int(log[i].Data.OpId)) + " "
	}
	return s
}

func commandToString (log []state.Command) string {
	var s string = ""
	for i := 0; i < len(log); i++ {
		s += strconv.Itoa(int(log[i].OpId)) + " "
	}
	return s
}