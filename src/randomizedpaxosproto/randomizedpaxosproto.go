package randomizedpaxosproto

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
    REQUEST_BROADCAST
    REQUEST_BROADCAST_REPLY
)

type Entry struct {
    Data 	    state.Command
    SenderId    int32
    Term 	    int32
    Index	    int32
    Timestamp   int64
}

type ReplicateEntries struct {
    SenderId		   				int32
    Term							int32
    Counter							int32
    PrevLogIndex					int32
    PrevLogTerm						int32
    Entries							[]Entry
    LeaderBenOrIndex				int32
    LeaderPreparedIndex				int32
}

type ReplicateEntriesReply struct {
    ReplicaId	   					int32
    Term							int32
    Counter							int32
    ReplicaBenOrIndex				int32
    ReplicaPreparedIndex			int32
    ReplicaEntries				    []Entry
    EntryAtLeaderBenOrIndex     	Entry
    Success							bool
}

type RequestVote struct {
    SenderId		   				int32
    Term 							int32
    Counter							int32
    CandidatePreparedIndex			int32
    CandidateBenOrIndex				int32
}

type RequestVoteReply struct {
    ReplicaId	   					int32
    Term							int32
    Counter							int32
    VoteGranted						bool
    ReplicaBenOrIndex				int32
    ReplicaPreparedIndex			int32
    ReplicaEntries					[]Entry
    EntryAtCandidateBenOrIndex		Entry
}

type BenOrBroadcast struct {
    SenderId		   				int32
    Term							int32
    Counter							int32
    ClientReq						state.Command
    Timestamp						int64
    RequestTerm						int32
    Index							int32
}

type BenOrBroadcastReply struct {
    ReplicaId	   					int32
    Term							int32
    Counter                         int32
    CommittedEntry					Entry
}

type BenOrConsensus struct {
    SenderId		   				int32
    Term							int32
    Counter							int32
    Phase							int32
    Vote							int32
    MajRequest						state.Command
    LeaderRequest					state.Command
    EntryType						int32
}

type BenOrConsensusReply struct {
    ReplicaId	   					int32
    Term							int32
    Counter                         int32
    CommittedEntry					Entry
}

type InfoBroadcast struct {
    SenderId		   				int32
    Term							int32
    Counter							int32
    ClientReq						Entry
}

type InfoBroadcastReply struct {
    ReplicaId	   					int32
    Term							int32
    Counter                         int32
}