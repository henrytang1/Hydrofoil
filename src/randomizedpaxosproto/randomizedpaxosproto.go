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
    Data 	        state.Command
    ReceiverId      int32
    Term 	        int32
    Index	        int32
    BenOrActive     uint8 // bool
    Timestamp       int64
    FromLeader      uint8 // bool
    // Valid           uint8 // bool
}

type ReplicateEntries struct {
    SenderId		   				int32
    Term							int32
    // Counter							int32
    PrevLogIndex					int32
    PrevLogTerm						int32
    Entries							[]Entry
    LeaderBenOrIndex				int32
    LeaderPreparedIndex				int32
}

func (t *ReplicateEntries) GetSenderId() int32 { return t.SenderId }
func (t *ReplicateEntries) GetTerm() int32 { return t.Term }
func (t *ReplicateEntries) GetPrevLogIndex() int32 { return t.PrevLogIndex }
func (t *ReplicateEntries) GetPrevLogTerm() int32 { return t.PrevLogTerm }
func (t *ReplicateEntries) GetEntries() []Entry { return t.Entries }
func (t *ReplicateEntries) GetSetLeaderBenOrIndex(leaderBenOrIndex int32) { t.LeaderBenOrIndex = leaderBenOrIndex }
func (t *ReplicateEntries) GetLeaderBenOrIndex() int32 { return t.LeaderBenOrIndex }
func (t *ReplicateEntries) GetLeaderPreparedIndex() int32 { return t.LeaderPreparedIndex }

type ReplicateEntriesReply struct {
    SenderId	   					int32
    Term							int32
    // Counter							int32
    ReplicaBenOrIndex				int32
    ReplicaPreparedIndex			int32
    ReplicaEntries				    []Entry
    // LeaderPreparedIndex				int32
    // LeaderBenOrIndex				int32
    // EntryAtLeaderBenOrIndex     	Entry
    PrevLogIndex					int32
    RequestedIndex					int32
    Success							uint8 // bool
}

func (t *ReplicateEntriesReply) GetSenderId() int32 { return t.SenderId }
func (t *ReplicateEntriesReply) GetTerm() int32 { return t.Term }
func (t *ReplicateEntriesReply) GetReplicaBenOrIndex() int32 { return t.ReplicaBenOrIndex }
func (t *ReplicateEntriesReply) GetReplicaPreparedIndex() int32 { return t.ReplicaPreparedIndex }
func (t *ReplicateEntriesReply) GetReplicaEntries() []Entry { return t.ReplicaEntries }
func (t *ReplicateEntriesReply) GetPrevLogIndex() int32 { return t.PrevLogIndex }
func (t *ReplicateEntriesReply) GetRequestedIndex() int32 { return t.RequestedIndex }
func (t *ReplicateEntriesReply) GetSuccess() uint8 { return t.Success }

type RequestVote struct {
    SenderId		   				int32
    Term 							int32
    // Counter							int32
    CandidateBenOrIndex				int32
}

type RequestVoteReply struct {
    SenderId	   					int32
    Term							int32
    // Counter							int32
    VoteGranted						uint8 // bool
    ReplicaBenOrIndex				int32
    ReplicaPreparedIndex			int32
    ReplicaEntries					[]Entry
    CandidateBenOrIndex				int32
    EntryAtCandidateBenOrIndex		Entry
}

type BenOrBroadcast struct {
    SenderId		   				int32
    Term							int32
    Index                           int32
    Iteration						int32
    ClientReq						Entry
    // Timestamp						int64
    // RequestTerm						int32
    // Index							int32
}

func (t *BenOrBroadcast) GetSenderId() int32 { return t.SenderId }
func (t *BenOrBroadcast) GetTerm() int32 { return t.Term }
func (t *BenOrBroadcast) GetIndex() int32 { return t.Index }
func (t *BenOrBroadcast) GetIteration() int32 { return t.Iteration }
func (t *BenOrBroadcast) GetClientReq() Entry { return t.ClientReq }

type BenOrBroadcastReply struct {
    SenderId	   					int32
    Term							int32
    // Counter                         int32
    Index                           int32
    Iteration                       int32
    ClientReq   					Entry
}

func (t *BenOrBroadcastReply) GetSenderId() int32 { return t.SenderId }
func (t *BenOrBroadcastReply) GetTerm() int32 { return t.Term }
func (t *BenOrBroadcastReply) GetIndex() int32 { return t.Index }
func (t *BenOrBroadcastReply) GetIteration() int32 { return t.Iteration }
func (t *BenOrBroadcastReply) GetClientReq() Entry { return t.ClientReq }

type BenOrConsensus struct {
    SenderId		   				int32
    Term							int32
    Index                           int32
    Iteration						int32 // iteration
    Phase							int32
    Vote							int32
    MajRequest						Entry
    LeaderRequest					Entry
    Stage   						int32 // 0 or 1 depending on BenOr stage
}

func (t *BenOrConsensus) GetSenderId() int32 { return t.SenderId }
func (t *BenOrConsensus) GetTerm() int32 { return t.Term }
func (t *BenOrConsensus) GetIndex() int32 { return t.Index }
func (t *BenOrConsensus) GetIteration() int32 { return t.Iteration }
func (t *BenOrConsensus) GetPhase() int32 { return t.Phase }
func (t *BenOrConsensus) GetVote() int32 { return t.Vote }
func (t *BenOrConsensus) GetMajRequest() Entry { return t.MajRequest }
func (t *BenOrConsensus) GetLeaderRequest() Entry { return t.LeaderRequest }
func (t *BenOrConsensus) GetStage() int32 { return t.Stage }

type BenOrConsensusReply struct {
    SenderId	   					int32
    Term							int32
    Index                           int32
    Iteration                       int32
    Phase                           int32
    Vote                            int32
    MajRequest                      Entry
    LeaderRequest                   Entry
    Stage                           int32
    // IndexCommitted                  uint8 // bool // true if replica has committed entry
    // CommittedEntry					Entry
}

func (t *BenOrConsensusReply) GetSenderId() int32 { return t.SenderId } // returns the replica that first sent to BenOrConsensusReply
func (t *BenOrConsensusReply) GetTerm() int32 { return t.Term }
func (t *BenOrConsensusReply) GetIndex() int32 { return t.Index }
func (t *BenOrConsensusReply) GetIteration() int32 { return t.Iteration }
func (t *BenOrConsensusReply) GetPhase() int32 { return t.Phase }
func (t *BenOrConsensusReply) GetVote() int32 { return t.Vote }
func (t *BenOrConsensusReply) GetMajRequest() Entry { return t.MajRequest }
func (t *BenOrConsensusReply) GetLeaderRequest() Entry { return t.LeaderRequest }
func (t *BenOrConsensusReply) GetStage() int32 { return t.Stage }

type GetCommittedData struct {
    SenderId           				int32
    Term               				int32
    StartIndex                      int32
    EndIndex                        int32 // inclusive
}

type GetCommittedDataReply struct {
    SenderId           				int32
    Term               				int32
    StartIndex                      int32
    EndIndex                        int32 // inclusive
    Entries                         []Entry
}

type InfoBroadcast struct {
    SenderId		   				int32
    Term							int32
    // Counter							int32
    ClientReq						Entry
}

type InfoBroadcastReply struct {
    SenderId	   					int32
    Term							int32
    // Counter                         int32
}