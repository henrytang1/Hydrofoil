package hydrofoilproto

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

// if Term != -1, then this entry is from the leader
// if Term == -1, then this entry is running BenOr or has been committed using BenOr+
type Entry struct {
    Data 	        state.Command
    ServerId        int32
    Term 	        int32
    Index	        int32
    Timestamp       int64
}

type ReplicateEntries struct {
    SenderId		   				int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    PrevLogIndex					int32
    PrevLogServerId                 int32
    PrevLogTimestamp                int64

    Entries							[]Entry
}

const (
	False uint8 		= 0
	True        		= 1
)

func (t *ReplicateEntries) GetSenderId() int32 { return t.SenderId }
func (t *ReplicateEntries) GetTerm() int32 { return t.Term }
func (t *ReplicateEntries) GetCommitIndex() int32 { return t.CommitIndex }
func (t *ReplicateEntries) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *ReplicateEntries) GetLogLength() int32 { return t.LogLength }
func (t *ReplicateEntries) GetPrevLogIndex() int32 { return t.PrevLogIndex }
func (t *ReplicateEntries) GetPrevLogServerId() int32 { return t.PrevLogServerId }
func (t *ReplicateEntries) GetPrevLogTimestamp() int64 { return t.PrevLogTimestamp }
func (t *ReplicateEntries) GetStartIndex() int32 { return t.PrevLogIndex + 1 }
func (t *ReplicateEntries) GetEntries() []Entry { return t.Entries }

type ReplicateEntriesReply struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    StartIndex                      int32 // return entries starting from this index
    Entries				            []Entry
    PQEntries                       []Entry

    NewRequestedIndex				int32

    MsgTimestamp                    int64
}

func (t *ReplicateEntriesReply) GetSenderId() int32 { return t.SenderId }
func (t *ReplicateEntriesReply) GetTerm() int32 { return t.Term }
func (t *ReplicateEntriesReply) GetCommitIndex() int32 { return t.CommitIndex }
func (t *ReplicateEntriesReply) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *ReplicateEntriesReply) GetLogLength() int32 { return t.LogLength }
func (t *ReplicateEntriesReply) GetStartIndex() int32 { return t.StartIndex }
func (t *ReplicateEntriesReply) GetEntries() []Entry { return t.Entries }
func (t *ReplicateEntriesReply) GetPQEntries() []Entry { return t.PQEntries }
func (t *ReplicateEntriesReply) GetNewRequestedIndex() int32 { return t.NewRequestedIndex }
func (t *ReplicateEntriesReply) GetMsgTimestamp() int64 { return t.MsgTimestamp }

type RequestVote struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32
}

func (t *RequestVote) GetSenderId() int32 { return t.SenderId }
func (t *RequestVote) GetTerm() int32 { return t.Term }
func (t *RequestVote) GetCommitIndex() int32 { return t.CommitIndex }
func (t *RequestVote) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *RequestVote) GetLogLength() int32 { return t.LogLength }

type RequestVoteReply struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      			    int32
    LeaderTerm                         int32
    LogLength                       int32

    VoteGranted						uint8 // bool
    StartIndex          			int32 // return entries starting from this index
    Entries     					[]Entry
    PQEntries                       []Entry
}

func (t *RequestVoteReply) GetSenderId() int32 { return t.SenderId }
func (t *RequestVoteReply) GetTerm() int32 { return t.Term }
func (t *RequestVoteReply) GetCommitIndex() int32 { return t.CommitIndex }
func (t *RequestVoteReply) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *RequestVoteReply) GetLogLength() int32 { return t.LogLength }
func (t *RequestVoteReply) GetVoteGranted() uint8 { return t.VoteGranted }
func (t *RequestVoteReply) GetStartIndex() int32 { return t.StartIndex }
func (t *RequestVoteReply) GetEntries() []Entry { return t.Entries }
func (t *RequestVoteReply) GetPQEntries() []Entry { return t.PQEntries }

type BenOrBroadcast struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32 // Ben Or index is commit index + 1
    LeaderTerm                         int32
    LogLength                       int32

    Iteration						int32
    BroadcastEntry					Entry

    StartIndex          			int32 // entries start from this index
    Entries     					[]Entry
    PQEntries                       []Entry
}

func (t *BenOrBroadcast) GetSenderId() int32 { return t.SenderId }
func (t *BenOrBroadcast) GetTerm() int32 { return t.Term }
func (t *BenOrBroadcast) GetCommitIndex() int32 { return t.CommitIndex }
func (t *BenOrBroadcast) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *BenOrBroadcast) GetLogLength() int32 { return t.LogLength }
func (t *BenOrBroadcast) GetBenOrMsgValid() uint8 { return True }
func (t *BenOrBroadcast) GetIteration() int32 { return t.Iteration }
func (t *BenOrBroadcast) GetBroadcastEntry() Entry { return t.BroadcastEntry }
func (t *BenOrBroadcast) GetStartIndex() int32 { return t.StartIndex }
func (t *BenOrBroadcast) GetEntries() []Entry { return t.Entries }
func (t *BenOrBroadcast) GetPQEntries() []Entry { return t.PQEntries }

type BenOrBroadcastReply struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    BenOrMsgValid                   uint8 // if this is false, then the replica hasn't started Ben-Or yet (and thus Broadcast Entry is meaningless)
    Iteration                       int32
    BroadcastEntry					Entry

    StartIndex          			int32 // return entries starting from this index    
    Entries     					[]Entry
    PQEntries                       []Entry
}

func (t *BenOrBroadcastReply) GetSenderId() int32 { return t.SenderId }
func (t *BenOrBroadcastReply) GetTerm() int32 { return t.Term }
func (t *BenOrBroadcastReply) GetCommitIndex() int32 { return t.CommitIndex }
func (t *BenOrBroadcastReply) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *BenOrBroadcastReply) GetLogLength() int32 { return t.LogLength }
func (t *BenOrBroadcastReply) GetBenOrMsgValid() uint8 { return t.BenOrMsgValid }
func (t *BenOrBroadcastReply) GetIteration() int32 { return t.Iteration }
func (t *BenOrBroadcastReply) GetBroadcastEntry() Entry { return t.BroadcastEntry }
func (t *BenOrBroadcastReply) GetStartIndex() int32 { return t.StartIndex }
func (t *BenOrBroadcastReply) GetEntries() []Entry { return t.Entries }
func (t *BenOrBroadcastReply) GetPQEntries() []Entry { return t.PQEntries }

type BenOrConsensus struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    Iteration						int32
    Phase							int32
    Stage   						uint8 // StageOne or StageTwo depending on BenOr stage

    Vote							uint8
    PrevPhaseFinalValue             uint8

    HaveMajEntry   					uint8 // bool
    MajEntry						Entry

    Entries     					[]Entry
    PQEntries                       []Entry
}

func (t *BenOrConsensus) GetSenderId() int32 { return t.SenderId }
func (t *BenOrConsensus) GetTerm() int32 { return t.Term }
func (t *BenOrConsensus) GetCommitIndex() int32 { return t.CommitIndex }
func (t *BenOrConsensus) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *BenOrConsensus) GetLogLength() int32 { return t.LogLength }
func (t *BenOrConsensus) GetBenOrMsgValid() uint8 { return True }
func (t *BenOrConsensus) GetIteration() int32 { return t.Iteration }
func (t *BenOrConsensus) GetPhase() int32 { return t.Phase }
func (t *BenOrConsensus) GetStage() uint8 { return t.Stage }
func (t *BenOrConsensus) GetVote() uint8 { return t.Vote }
func (t *BenOrConsensus) GetPrevPhaseFinalValue() uint8 { return t.PrevPhaseFinalValue }
func (t *BenOrConsensus) GetHaveMajEntry() uint8 { return t.HaveMajEntry }
func (t *BenOrConsensus) GetMajEntry() Entry { return t.MajEntry }
func (t *BenOrConsensus) GetStartIndex() int32 { return t.CommitIndex + 1 }
func (t *BenOrConsensus) GetEntries() []Entry { return t.Entries }
func (t *BenOrConsensus) GetPQEntries() []Entry { return t.PQEntries }

type BenOrConsensusReply struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    BenOrMsgValid                   uint8 // if this is false, then the replica hasn't started Ben-Or yet (and thus Vote is meaningless)
    Iteration						int32
    Phase							int32
    Stage   						uint8 // StageOne or StageTwo depending on BenOr stage

    Vote							uint8
    PrevPhaseFinalValue             uint8

    HaveMajEntry   					uint8 // bool
    MajEntry						Entry

    StartIndex          			int32 // entries start from this index
    Entries     					[]Entry
    PQEntries                       []Entry
}

func (t *BenOrConsensusReply) GetSenderId() int32 { return t.SenderId }
func (t *BenOrConsensusReply) GetTerm() int32 { return t.Term }
func (t *BenOrConsensusReply) GetCommitIndex() int32 { return t.CommitIndex }
func (t *BenOrConsensusReply) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *BenOrConsensusReply) GetLogLength() int32 { return t.LogLength }
func (t *BenOrConsensusReply) GetBenOrMsgValid() uint8 { return t.BenOrMsgValid }
func (t *BenOrConsensusReply) GetIteration() int32 { return t.Iteration }
func (t *BenOrConsensusReply) GetPhase() int32 { return t.Phase }
func (t *BenOrConsensusReply) GetStage() uint8 { return t.Stage }
func (t *BenOrConsensusReply) GetVote() uint8 { return t.Vote }
func (t *BenOrConsensusReply) GetPrevPhaseFinalValue() uint8 { return t.PrevPhaseFinalValue }
func (t *BenOrConsensusReply) GetHaveMajEntry() uint8 { return t.HaveMajEntry }
func (t *BenOrConsensusReply) GetMajEntry() Entry { return t.MajEntry }
func (t *BenOrConsensusReply) GetStartIndex() int32 { return t.StartIndex }
func (t *BenOrConsensusReply) GetEntries() []Entry { return t.Entries }
func (t *BenOrConsensusReply) GetPQEntries() []Entry { return t.PQEntries }

type GetCommittedData struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    StartIndex                      int32
    Entries                         []Entry
    PQEntries                       []Entry
}

func (t *GetCommittedData) GetSenderId() int32 { return t.SenderId }
func (t *GetCommittedData) GetTerm() int32 { return t.Term }
func (t *GetCommittedData) GetCommitIndex() int32 { return t.CommitIndex }
func (t *GetCommittedData) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *GetCommittedData) GetLogLength() int32 { return t.LogLength }
func (t *GetCommittedData) GetStartIndex() int32 { return t.StartIndex }
func (t *GetCommittedData) GetEntries() []Entry { return t.Entries }
func (t *GetCommittedData) GetPQEntries() []Entry { return t.PQEntries }

type GetCommittedDataReply struct {
    SenderId	   					int32
    Term							int32
    CommitIndex      				int32
    LeaderTerm                         int32
    LogLength                       int32

    StartIndex                      int32
    Entries                         []Entry
    PQEntries                       []Entry
}

func (t *GetCommittedDataReply) GetSenderId() int32 { return t.SenderId }
func (t *GetCommittedDataReply) GetTerm() int32 { return t.Term }
func (t *GetCommittedDataReply) GetCommitIndex() int32 { return t.CommitIndex }
func (t *GetCommittedDataReply) GetLeaderTerm() int32 { return t.LeaderTerm }
func (t *GetCommittedDataReply) GetLogLength() int32 { return t.LogLength }
func (t *GetCommittedDataReply) GetStartIndex() int32 { return t.StartIndex }
func (t *GetCommittedDataReply) GetEntries() []Entry { return t.Entries }
func (t *GetCommittedDataReply) GetPQEntries() []Entry { return t.PQEntries }

// type InfoBroadcast struct {
//     SenderId	   					int32
//     Term							int32
//     CommitIndex      				int32
//     LeaderTerm                         int32

//     PQEntries                       []Entry
// }

// func (t *InfoBroadcast) GetSenderId() int32 { return t.SenderId }
// func (t *InfoBroadcast) GetTerm() int32 { return t.Term }
// func (t *InfoBroadcast) GetCommitIndex() int32 { return t.CommitIndex }
// func (t *InfoBroadcast) GetLeaderTerm() int32 { return t.LeaderTerm }
// func (t *InfoBroadcast) GetPQEntries() []Entry { return t.PQEntries }

// type InfoBroadcastReply struct {
//     SenderId	   					int32
//     Term							int32
// }