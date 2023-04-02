package randomizedpaxos

import (
	"randomizedpaxosproto"
	"state"
	"time"
)

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
	return (r.benOrState.benOrStatus == Broadcasting || 
		r.benOrState.benOrStatus == StageOne || 
		r.benOrState.benOrStatus == StageTwo)
}

// called when the timer has not yet fired
func (r *Replica) resetTimer(t *time.Timer, d time.Duration) {
	if t == nil {
		t = time.NewTimer(d)
		return
	}
	if !t.Stop() {
		<-t.C
	}
	t.Reset(d)
}

// called when the timer has already fired
func (r *Replica) setTimer(t *time.Timer, d time.Duration) {
	if t == nil {
		t = time.NewTimer(d)
		return
	}
	t.Reset(d)
}

func (r *Replica) clearTimer(t *time.Timer) {
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