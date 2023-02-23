package paxosproto

import "state"

const (
	REPLICATE_ENTRIES uint8 = iota
	REPLICATE_ENTRIES_REPLY
	REQUEST_VOTE
	REQUEST_VOTE_REPLY
	BEN_OR_BROADCAST
	BEN_OR_BROADCAST_REPLY
	BEN_OR_CONSENSUS
	BEN_OR_CONSENSUS_REPLY
	BROADCAST
	BROADCAST_REPLY
)

type ReplicateEntries struct {
	Term				int32
	PrevLogIndex		int32
	PrevLogTerm			int32
	Entries				[]state.Command
	BenOrIndex			int32
	LeaderCommit		int32
}

type ReplicateEntriesReply struct {
	Term				int32
	CommitIndex			int32
	BenOrCommitEntry	state.Command
	ReplicateEntries	[]state.Command
	Success				bool
}

type RequestVote struct {
	Term 				int32
	LeaderCommit		int32
	BenOrIndex			int32
}

type RequestVoteReply struct {
	Term				int32
	VoteGranted			bool
	CommitIndex			int32
	BenOrIndex			int32
	Entries				[]state.Command
	BenOrEntry			state.Command
}

type BenOrBroadcast struct {
	Term				int32
	ClientReq			state.Command
	Timestamp			int64
	RequestTerm			int32
	Index				int32
}

type BenOrBroadcastReply struct {
	Term				int32
	CommittedEntry		bool
}

type BenOrConsensus struct {
	Phase				int32
	Vote				int32
	MajRequest			state.Command
	LeaderRequest		state.Command
	EntryType			int32
}

type BenOrConsensusReply struct {
	Term				int32
	CommittedEntry		state.Command
}

type Broadcast struct {
	Term				int32
	ClientReq			state.Command
}

type BroadcastReply struct {
	Term				int32
}