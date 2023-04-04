package randomizedpaxos

import (
	"math/rand"
	"randomizedpaxosproto"
	"state"
	"time"
)

type Set struct {
	m map[UniqueCommand]bool
}

func newSet() Set {
	return Set{make(map[UniqueCommand]bool)}
}

func (s *Set) add(item UniqueCommand) {
	s.m[item] = false
}

func (s *Set) remove(item UniqueCommand) {
	delete(s.m, item)
}

func (s *Set) contains(item UniqueCommand) bool {
	_, ok := s.m[item]
	return ok
}

func (s *Set) commit(item UniqueCommand) {
	s.m[item] = true
}

func (s *Set) isCommitted(item UniqueCommand) bool {
	if _, ok := s.m[item]; !ok {
		return false
	}
	return s.m[item]
}

func (s *Set) commitSlice(items []UniqueCommand) {
	for _, item := range items {
		s.commit(item)
	}
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

func (r *Replica) addNewEntry(newLogEntry Entry) {
	if r.leaderState.isLeader {
		newLogEntry.Term = int32(r.currentTerm)
		if r.benOrIndex == len(r.log) - 1 && r.benOrRunning() {
			newLogEntry.Index = int32(len(r.log)) - 1
			r.log[len(r.log)-1] = newLogEntry
		} else {
			newLogEntry.Index = int32(len(r.log))
			r.log = append(r.log, newLogEntry)
		}
	} else {
		r.pq.push(newLogEntry)
	}
}

// called when the timer has not yet fired
func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		<-t.C
	}
	t.Reset(d)
}

// called when the timer has already fired
func setTimer(t *time.Timer, d time.Duration) {
	t.Reset(d)
}

func clearTimer(t *time.Timer) {
	if !t.Stop() {
		<-t.C
	}
}

func benOrUncommittedLogEntry(idx int) randomizedpaxosproto.Entry {
	return randomizedpaxosproto.Entry{
		Data: state.Command{},
		SenderId: -1,
		Term: -1, // term -1 means that this is a Ben-Or+ entry that hasn't yet committed
		Index: int32(idx),
		BenOrActive: True,
		Timestamp: -1,
		FromLeader: False,
	}
}


// Returns true if term <= newTerm
func (r *Replica) handleIncomingRPCTerm (newTerm int) bool {
	if newTerm < r.currentTerm {
		// ignore these entries
		return false
	}
	if newTerm > r.currentTerm {
		r.currentTerm = newTerm

		if (r.leaderState.isLeader) {
			clearTimer(r.heartbeatTimer)
		}

		r.leaderState = LeaderState{
			isLeader: false,
			repNextIndex: make([]int, r.N),
			repCommitIndex: make([]int, r.N),
			repPreparedIndex: make([]int, r.N),
		}
	}
	timeout := rand.Intn(r.electionTimeout/2) + r.electionTimeout/2
	setTimer(r.electionTimer, time.Duration(timeout)*time.Millisecond)

	return true
}