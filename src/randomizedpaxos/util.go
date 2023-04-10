package randomizedpaxos

import (
	"log"
	"math/rand"
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
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, false, false}
	} else {
		status.logOccurrences++
		s.m[item] = status
	}
}

func (s *Set) remove(entry Entry) {
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
	if item.senderId == -1 {
		return
	}
	delete(s.m, item)
}

func (s *Set) contains(entry Entry) bool {
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		return false
	}
	return status.logOccurrences > 0
}

func (s *Set) commit(entry Entry) {
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, true, false}
	} else {
		status.committed = true
		s.m[item] = status
	}
}

func (s *Set) execute(entry Entry) {
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
	status, found := s.m[item]
	if !found {
		s.m[item] = ClientReqStatus{1, true, true}
	} else {
		status.executed = true
		s.m[item] = status
	}
}

func (s *Set) isCommitted(entry Entry) bool {
	item := UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp}
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

// func (r *Replica) addNewEntry(newLogEntry Entry) {
// 	if r.leaderState.isLeader {
// 		newLogEntry.Term = int32(r.term)
// 		if r.benOrIndex == len(r.log) - 1 && r.benOrRunning() {
// 			newLogEntry.Index = int32(len(r.log)) - 1
// 			r.log[len(r.log)-1] = newLogEntry
// 		} else {
// 			newLogEntry.Index = int32(len(r.log))
// 			r.log = append(r.log, newLogEntry)
// 		}
// 		r.inLog.add(UniqueCommand{senderId: newLogEntry.SenderId, time: newLogEntry.Timestamp})
// 	} else {
// 		r.pq.push(newLogEntry)
// 	}
// }

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
			r.leaderState = emptyLeaderState
			clearTimer(r.heartbeatTimer)

			timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
			setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)
		}
		if r.candidateState.isCandidate {
			r.candidateState = emptyCandidateState
		}
	}
}

const ( // for benOrStatus
	LowerOrder uint8	= 0
	SameOrder	   	= 1
	HigherOrder	   	= 2
)

func convertBoolToOrder (b bool) uint8 {
	if b { return HigherOrder }
	return LowerOrder
}

func (r *Replica) isLogMoreUpToDate(rpc RPC) uint8 {
	if r.commitIndex != int(rpc.GetCommitIndex()) {
		return convertBoolToOrder(r.commitIndex > int(rpc.GetCommitIndex()))
	}
	if r.logTerm != int(rpc.GetLogTerm()) {
		return convertBoolToOrder(r.logTerm > int(rpc.GetLogTerm()))
	}
	if len(r.log) != int(rpc.GetLogLength()) {
		return convertBoolToOrder(len(r.log) > int(rpc.GetLogLength()))
	}
	return SameOrder
}

// func benOrUncommittedLogEntry() Entry {
// 	return randomizedpaxosproto.Entry{
// 		Data: state.Command{},
// 		SenderId: -1,
// 		Term: -1, // term -1 means that this is a Ben-Or+ entry that hasn't yet committed
// 		Index: -1,
// 		// BenOrActive: True,
// 		Timestamp: -1,
// 		// FromLeader: False,
// 	}
// }


// Returns true if term <= newTerm
// func (r *Replica) handleIncomingRPCTerm (newTerm int) bool {
// 	if newTerm < r.term {
// 		// ignore these entries
// 		return false
// 	}
// 	if newTerm > r.term {
// 		r.term = newTerm

// 		if (r.leaderState.isLeader) {
// 			clearTimer(r.heartbeatTimer)
// 		}

// 		r.leaderState = LeaderState{
// 			isLeader: false,
// 			repNextIndex: make([]int, r.N),
// 			repCommitIndex: make([]int, r.N),
// 			repPreparedIndex: make([]int, r.N),
// 		}
// 	}

// 	return true
// }

// // Returns true if term <= newTerm
// func (r *Replica) handleReplicateEntriesRPCTerm (newTerm int) bool {
// 	if newTerm < r.term {
// 		// ignore these entries
// 		return false
// 	}
// 	if newTerm > r.term {
// 		r.term = newTerm

// 		if (r.leaderState.isLeader) {
// 			clearTimer(r.heartbeatTimer)
// 		}

// 		r.leaderState = LeaderState{
// 			isLeader: false,
// 			repNextIndex: make([]int, r.N),
// 			repCommitIndex: make([]int, r.N),
// 			repPreparedIndex: make([]int, r.N),
// 		}
// 	}
// 	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
// 	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

// 	return true
// }

func entryEqual (e1 Entry, e2 Entry) bool {
	return e1.SenderId == e2.SenderId && e1.Timestamp == e2.Timestamp
}

// func (r *Replica) addEntryToLogIndex (entry Entry, idx int) {
// 	if r.inLog.is
	
// 	if idx == len(r.log) {
// 		r.log = append(r.log, entry)
// 	} else {
// 		r.log[idx] = entry
// 	}
// }

// func (r *Replica) addToPQ (entry Entry) {
// 	if r.seenBefore(entry) {
// 		return
// 	}
// 	r.inLog.add(UniqueCommand{senderId: entry.SenderId, time: entry.Timestamp})
// 	r.pq.push(entry)
// }