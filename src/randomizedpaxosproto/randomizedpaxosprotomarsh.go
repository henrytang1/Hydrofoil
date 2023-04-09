package randomizedpaxosproto

import (
	"bufio"
	"encoding/binary"
	"fastrpc"
	"io"
	"sync"
)

type byteReader interface {
	io.Reader
	ReadByte() (c byte, err error)
}

func (t *ReplicateEntries) New() fastrpc.Serializable {
	return new(ReplicateEntries)
}
func (t *ReplicateEntriesReply) New() fastrpc.Serializable {
	return new(ReplicateEntriesReply)
}
func (t *RequestVote) New() fastrpc.Serializable {
	return new(RequestVote)
}
func (t *RequestVoteReply) New() fastrpc.Serializable {
	return new(RequestVoteReply)
}
func (t *BenOrBroadcast) New() fastrpc.Serializable {
	return new(BenOrBroadcast)
}
func (t *BenOrBroadcastReply) New() fastrpc.Serializable {
	return new(BenOrBroadcastReply)
}
func (t *BenOrConsensus) New() fastrpc.Serializable {
	return new(BenOrConsensus)
}
func (t *BenOrConsensusReply) New() fastrpc.Serializable {
	return new(BenOrConsensusReply)
}
func (t *GetCommittedData) New() fastrpc.Serializable {
	return new(GetCommittedData)
}
func (t *SendCommittedData) New() fastrpc.Serializable {
	return new(SendCommittedData)
}    

func (t *RequestVoteReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type RequestVoteReplyCache struct {
	mu	sync.Mutex
	cache	[]*RequestVoteReply
}

func NewRequestVoteReplyCache() *RequestVoteReplyCache {
	c := &RequestVoteReplyCache{}
	c.cache = make([]*RequestVoteReply, 0)
	return c
}

func (p *RequestVoteReplyCache) Get() *RequestVoteReply {
	var t *RequestVoteReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &RequestVoteReply{}
	}
	return t
}
func (p *RequestVoteReplyCache) Put(t *RequestVoteReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *RequestVoteReply) Marshal(wire io.Writer) {
	var b [25]byte
	var bs []byte
	bs = b[:25]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	bs[20] = byte(t.VoteGranted)
	tmp32 = t.StartIndex
	bs[21] = byte(tmp32)
	bs[22] = byte(tmp32 >> 8)
	bs[23] = byte(tmp32 >> 16)
	bs[24] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *RequestVoteReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [25]byte
	var bs []byte
	bs = b[:25]
	if _, err := io.ReadAtLeast(wire, bs, 25); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.VoteGranted = uint8(bs[20])
	t.StartIndex = int32((uint32(bs[21]) | (uint32(bs[22]) << 8) | (uint32(bs[23]) << 16) | (uint32(bs[24]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}

func (t *BenOrBroadcastReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type BenOrBroadcastReplyCache struct {
	mu	sync.Mutex
	cache	[]*BenOrBroadcastReply
}

func NewBenOrBroadcastReplyCache() *BenOrBroadcastReplyCache {
	c := &BenOrBroadcastReplyCache{}
	c.cache = make([]*BenOrBroadcastReply, 0)
	return c
}

func (p *BenOrBroadcastReplyCache) Get() *BenOrBroadcastReply {
	var t *BenOrBroadcastReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BenOrBroadcastReply{}
	}
	return t
}
func (p *BenOrBroadcastReplyCache) Put(t *BenOrBroadcastReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BenOrBroadcastReply) Marshal(wire io.Writer) {
	var b [25]byte
	var bs []byte
	bs = b[:25]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	bs[20] = byte(t.BenOrMsgValid)
	tmp32 = t.Iteration
	bs[21] = byte(tmp32)
	bs[22] = byte(tmp32 >> 8)
	bs[23] = byte(tmp32 >> 16)
	bs[24] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.BroadcastEntry.Marshal(wire)
	bs = b[:4]
	tmp32 = t.StartIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *BenOrBroadcastReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [25]byte
	var bs []byte
	bs = b[:25]
	if _, err := io.ReadAtLeast(wire, bs, 25); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.BenOrMsgValid = uint8(bs[20])
	t.Iteration = int32((uint32(bs[21]) | (uint32(bs[22]) << 8) | (uint32(bs[23]) << 16) | (uint32(bs[24]) << 24)))
	t.BroadcastEntry.Unmarshal(wire)
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.StartIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}

func (t *BenOrConsensus) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type BenOrConsensusCache struct {
	mu	sync.Mutex
	cache	[]*BenOrConsensus
}

func NewBenOrConsensusCache() *BenOrConsensusCache {
	c := &BenOrConsensusCache{}
	c.cache = make([]*BenOrConsensus, 0)
	return c
}

func (p *BenOrConsensusCache) Get() *BenOrConsensus {
	var t *BenOrConsensus
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BenOrConsensus{}
	}
	return t
}
func (p *BenOrConsensusCache) Put(t *BenOrConsensus) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BenOrConsensus) Marshal(wire io.Writer) {
	var b [36]byte
	var bs []byte
	bs = b[:36]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	tmp32 = t.Iteration
	bs[20] = byte(tmp32)
	bs[21] = byte(tmp32 >> 8)
	bs[22] = byte(tmp32 >> 16)
	bs[23] = byte(tmp32 >> 24)
	tmp32 = t.Phase
	bs[24] = byte(tmp32)
	bs[25] = byte(tmp32 >> 8)
	bs[26] = byte(tmp32 >> 16)
	bs[27] = byte(tmp32 >> 24)
	tmp32 = t.Stage
	bs[28] = byte(tmp32)
	bs[29] = byte(tmp32 >> 8)
	bs[30] = byte(tmp32 >> 16)
	bs[31] = byte(tmp32 >> 24)
	tmp32 = t.Vote
	bs[32] = byte(tmp32)
	bs[33] = byte(tmp32 >> 8)
	bs[34] = byte(tmp32 >> 16)
	bs[35] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.MajRequest.Marshal(wire)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *BenOrConsensus) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [36]byte
	var bs []byte
	bs = b[:36]
	if _, err := io.ReadAtLeast(wire, bs, 36); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.Iteration = int32((uint32(bs[20]) | (uint32(bs[21]) << 8) | (uint32(bs[22]) << 16) | (uint32(bs[23]) << 24)))
	t.Phase = int32((uint32(bs[24]) | (uint32(bs[25]) << 8) | (uint32(bs[26]) << 16) | (uint32(bs[27]) << 24)))
	t.Stage = int32((uint32(bs[28]) | (uint32(bs[29]) << 8) | (uint32(bs[30]) << 16) | (uint32(bs[31]) << 24)))
	t.Vote = int32((uint32(bs[32]) | (uint32(bs[33]) << 8) | (uint32(bs[34]) << 16) | (uint32(bs[35]) << 24)))
	t.MajRequest.Unmarshal(wire)
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}

func (t *SendCommittedData) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type SendCommittedDataCache struct {
	mu	sync.Mutex
	cache	[]*SendCommittedData
}

func NewSendCommittedDataCache() *SendCommittedDataCache {
	c := &SendCommittedDataCache{}
	c.cache = make([]*SendCommittedData, 0)
	return c
}

func (p *SendCommittedDataCache) Get() *SendCommittedData {
	var t *SendCommittedData
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &SendCommittedData{}
	}
	return t
}
func (p *SendCommittedDataCache) Put(t *SendCommittedData) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *SendCommittedData) Marshal(wire io.Writer) {
	var b [24]byte
	var bs []byte
	bs = b[:24]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	tmp32 = t.StartIndex
	bs[20] = byte(tmp32)
	bs[21] = byte(tmp32 >> 8)
	bs[22] = byte(tmp32 >> 16)
	bs[23] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *SendCommittedData) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [24]byte
	var bs []byte
	bs = b[:24]
	if _, err := io.ReadAtLeast(wire, bs, 24); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.StartIndex = int32((uint32(bs[20]) | (uint32(bs[21]) << 8) | (uint32(bs[22]) << 16) | (uint32(bs[23]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}

func (t *Entry) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type EntryCache struct {
	mu	sync.Mutex
	cache	[]*Entry
}

func NewEntryCache() *EntryCache {
	c := &EntryCache{}
	c.cache = make([]*Entry, 0)
	return c
}

func (p *EntryCache) Get() *Entry {
	var t *Entry
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Entry{}
	}
	return t
}
func (p *EntryCache) Put(t *Entry) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Entry) Marshal(wire io.Writer) {
	var b [20]byte
	var bs []byte
	t.Data.Marshal(wire)
	bs = b[:20]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.Index
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp64 := t.Timestamp
	bs[12] = byte(tmp64)
	bs[13] = byte(tmp64 >> 8)
	bs[14] = byte(tmp64 >> 16)
	bs[15] = byte(tmp64 >> 24)
	bs[16] = byte(tmp64 >> 32)
	bs[17] = byte(tmp64 >> 40)
	bs[18] = byte(tmp64 >> 48)
	bs[19] = byte(tmp64 >> 56)
	wire.Write(bs)
}

func (t *Entry) Unmarshal(wire io.Reader) error {
	var b [20]byte
	var bs []byte
	t.Data.Unmarshal(wire)
	bs = b[:20]
	if _, err := io.ReadAtLeast(wire, bs, 20); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Index = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.Timestamp = int64((uint64(bs[12]) | (uint64(bs[13]) << 8) | (uint64(bs[14]) << 16) | (uint64(bs[15]) << 24) | (uint64(bs[16]) << 32) | (uint64(bs[17]) << 40) | (uint64(bs[18]) << 48) | (uint64(bs[19]) << 56)))
	return nil
}

func (t *ReplicateEntriesReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ReplicateEntriesReplyCache struct {
	mu	sync.Mutex
	cache	[]*ReplicateEntriesReply
}

func NewReplicateEntriesReplyCache() *ReplicateEntriesReplyCache {
	c := &ReplicateEntriesReplyCache{}
	c.cache = make([]*ReplicateEntriesReply, 0)
	return c
}

func (p *ReplicateEntriesReplyCache) Get() *ReplicateEntriesReply {
	var t *ReplicateEntriesReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ReplicateEntriesReply{}
	}
	return t
}
func (p *ReplicateEntriesReplyCache) Put(t *ReplicateEntriesReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ReplicateEntriesReply) Marshal(wire io.Writer) {
	var b [32]byte
	var bs []byte
	bs = b[:32]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	tmp64 := t.LeaderTimestamp
	bs[20] = byte(tmp64)
	bs[21] = byte(tmp64 >> 8)
	bs[22] = byte(tmp64 >> 16)
	bs[23] = byte(tmp64 >> 24)
	bs[24] = byte(tmp64 >> 32)
	bs[25] = byte(tmp64 >> 40)
	bs[26] = byte(tmp64 >> 48)
	bs[27] = byte(tmp64 >> 56)
	tmp32 = t.StartIndex
	bs[28] = byte(tmp32)
	bs[29] = byte(tmp32 >> 8)
	bs[30] = byte(tmp32 >> 16)
	bs[31] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
	bs = b[:5]
	bs[0] = byte(t.Success)
	tmp32 = t.NewRequestedIndex
	bs[1] = byte(tmp32)
	bs[2] = byte(tmp32 >> 8)
	bs[3] = byte(tmp32 >> 16)
	bs[4] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *ReplicateEntriesReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [32]byte
	var bs []byte
	bs = b[:32]
	if _, err := io.ReadAtLeast(wire, bs, 32); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.LeaderTimestamp = int64((uint64(bs[20]) | (uint64(bs[21]) << 8) | (uint64(bs[22]) << 16) | (uint64(bs[23]) << 24) | (uint64(bs[24]) << 32) | (uint64(bs[25]) << 40) | (uint64(bs[26]) << 48) | (uint64(bs[27]) << 56)))
	t.StartIndex = int32((uint32(bs[28]) | (uint32(bs[29]) << 8) | (uint32(bs[30]) << 16) | (uint32(bs[31]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	bs = b[:5]
	if _, err := io.ReadAtLeast(wire, bs, 5); err != nil {
		return err
	}
	t.Success = uint8(bs[0])
	t.NewRequestedIndex = int32((uint32(bs[1]) | (uint32(bs[2]) << 8) | (uint32(bs[3]) << 16) | (uint32(bs[4]) << 24)))
	return nil
}

func (t *RequestVote) BinarySize() (nbytes int, sizeKnown bool) {
	return 20, true
}

type RequestVoteCache struct {
	mu	sync.Mutex
	cache	[]*RequestVote
}

func NewRequestVoteCache() *RequestVoteCache {
	c := &RequestVoteCache{}
	c.cache = make([]*RequestVote, 0)
	return c
}

func (p *RequestVoteCache) Get() *RequestVote {
	var t *RequestVote
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &RequestVote{}
	}
	return t
}
func (p *RequestVoteCache) Put(t *RequestVote) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *RequestVote) Marshal(wire io.Writer) {
	var b [20]byte
	var bs []byte
	bs = b[:20]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *RequestVote) Unmarshal(wire io.Reader) error {
	var b [20]byte
	var bs []byte
	bs = b[:20]
	if _, err := io.ReadAtLeast(wire, bs, 20); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	return nil
}

func (t *GetCommittedData) BinarySize() (nbytes int, sizeKnown bool) {
	return 20, true
}

type GetCommittedDataCache struct {
	mu	sync.Mutex
	cache	[]*GetCommittedData
}

func NewGetCommittedDataCache() *GetCommittedDataCache {
	c := &GetCommittedDataCache{}
	c.cache = make([]*GetCommittedData, 0)
	return c
}

func (p *GetCommittedDataCache) Get() *GetCommittedData {
	var t *GetCommittedData
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetCommittedData{}
	}
	return t
}
func (p *GetCommittedDataCache) Put(t *GetCommittedData) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetCommittedData) Marshal(wire io.Writer) {
	var b [20]byte
	var bs []byte
	bs = b[:20]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *GetCommittedData) Unmarshal(wire io.Reader) error {
	var b [20]byte
	var bs []byte
	bs = b[:20]
	if _, err := io.ReadAtLeast(wire, bs, 20); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	return nil
}

func (t *ReplicateEntries) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ReplicateEntriesCache struct {
	mu	sync.Mutex
	cache	[]*ReplicateEntries
}

func NewReplicateEntriesCache() *ReplicateEntriesCache {
	c := &ReplicateEntriesCache{}
	c.cache = make([]*ReplicateEntries, 0)
	return c
}

func (p *ReplicateEntriesCache) Get() *ReplicateEntries {
	var t *ReplicateEntries
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ReplicateEntries{}
	}
	return t
}
func (p *ReplicateEntriesCache) Put(t *ReplicateEntries) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ReplicateEntries) Marshal(wire io.Writer) {
	var b [44]byte
	var bs []byte
	bs = b[:44]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	tmp64 := t.LeaderTimestamp
	bs[20] = byte(tmp64)
	bs[21] = byte(tmp64 >> 8)
	bs[22] = byte(tmp64 >> 16)
	bs[23] = byte(tmp64 >> 24)
	bs[24] = byte(tmp64 >> 32)
	bs[25] = byte(tmp64 >> 40)
	bs[26] = byte(tmp64 >> 48)
	bs[27] = byte(tmp64 >> 56)
	tmp32 = t.PrevLogIndex
	bs[28] = byte(tmp32)
	bs[29] = byte(tmp32 >> 8)
	bs[30] = byte(tmp32 >> 16)
	bs[31] = byte(tmp32 >> 24)
	tmp32 = t.PrevLogSenderId
	bs[32] = byte(tmp32)
	bs[33] = byte(tmp32 >> 8)
	bs[34] = byte(tmp32 >> 16)
	bs[35] = byte(tmp32 >> 24)
	tmp64 = t.PrevLogTimestamp
	bs[36] = byte(tmp64)
	bs[37] = byte(tmp64 >> 8)
	bs[38] = byte(tmp64 >> 16)
	bs[39] = byte(tmp64 >> 24)
	bs[40] = byte(tmp64 >> 32)
	bs[41] = byte(tmp64 >> 40)
	bs[42] = byte(tmp64 >> 48)
	bs[43] = byte(tmp64 >> 56)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
}

func (t *ReplicateEntries) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [44]byte
	var bs []byte
	bs = b[:44]
	if _, err := io.ReadAtLeast(wire, bs, 44); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.LeaderTimestamp = int64((uint64(bs[20]) | (uint64(bs[21]) << 8) | (uint64(bs[22]) << 16) | (uint64(bs[23]) << 24) | (uint64(bs[24]) << 32) | (uint64(bs[25]) << 40) | (uint64(bs[26]) << 48) | (uint64(bs[27]) << 56)))
	t.PrevLogIndex = int32((uint32(bs[28]) | (uint32(bs[29]) << 8) | (uint32(bs[30]) << 16) | (uint32(bs[31]) << 24)))
	t.PrevLogSenderId = int32((uint32(bs[32]) | (uint32(bs[33]) << 8) | (uint32(bs[34]) << 16) | (uint32(bs[35]) << 24)))
	t.PrevLogTimestamp = int64((uint64(bs[36]) | (uint64(bs[37]) << 8) | (uint64(bs[38]) << 16) | (uint64(bs[39]) << 24) | (uint64(bs[40]) << 32) | (uint64(bs[41]) << 40) | (uint64(bs[42]) << 48) | (uint64(bs[43]) << 56)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	return nil
}

func (t *BenOrBroadcast) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type BenOrBroadcastCache struct {
	mu	sync.Mutex
	cache	[]*BenOrBroadcast
}

func NewBenOrBroadcastCache() *BenOrBroadcastCache {
	c := &BenOrBroadcastCache{}
	c.cache = make([]*BenOrBroadcast, 0)
	return c
}

func (p *BenOrBroadcastCache) Get() *BenOrBroadcast {
	var t *BenOrBroadcast
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BenOrBroadcast{}
	}
	return t
}
func (p *BenOrBroadcastCache) Put(t *BenOrBroadcast) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BenOrBroadcast) Marshal(wire io.Writer) {
	var b [24]byte
	var bs []byte
	bs = b[:24]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	tmp32 = t.Iteration
	bs[20] = byte(tmp32)
	bs[21] = byte(tmp32 >> 8)
	bs[22] = byte(tmp32 >> 16)
	bs[23] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.BroadcastEntry.Marshal(wire)
	bs = b[:4]
	tmp32 = t.StartIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *BenOrBroadcast) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [24]byte
	var bs []byte
	bs = b[:24]
	if _, err := io.ReadAtLeast(wire, bs, 24); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.Iteration = int32((uint32(bs[20]) | (uint32(bs[21]) << 8) | (uint32(bs[22]) << 16) | (uint32(bs[23]) << 24)))
	t.BroadcastEntry.Unmarshal(wire)
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.StartIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}

func (t *BenOrConsensusReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type BenOrConsensusReplyCache struct {
	mu	sync.Mutex
	cache	[]*BenOrConsensusReply
}

func NewBenOrConsensusReplyCache() *BenOrConsensusReplyCache {
	c := &BenOrConsensusReplyCache{}
	c.cache = make([]*BenOrConsensusReply, 0)
	return c
}

func (p *BenOrConsensusReplyCache) Get() *BenOrConsensusReply {
	var t *BenOrConsensusReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BenOrConsensusReply{}
	}
	return t
}
func (p *BenOrConsensusReplyCache) Put(t *BenOrConsensusReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BenOrConsensusReply) Marshal(wire io.Writer) {
	var b [37]byte
	var bs []byte
	bs = b[:37]
	tmp32 := t.SenderId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.CommitIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.LogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.LogLength
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	bs[20] = byte(t.BenOrMsgValid)
	tmp32 = t.Iteration
	bs[21] = byte(tmp32)
	bs[22] = byte(tmp32 >> 8)
	bs[23] = byte(tmp32 >> 16)
	bs[24] = byte(tmp32 >> 24)
	tmp32 = t.Phase
	bs[25] = byte(tmp32)
	bs[26] = byte(tmp32 >> 8)
	bs[27] = byte(tmp32 >> 16)
	bs[28] = byte(tmp32 >> 24)
	tmp32 = t.Stage
	bs[29] = byte(tmp32)
	bs[30] = byte(tmp32 >> 8)
	bs[31] = byte(tmp32 >> 16)
	bs[32] = byte(tmp32 >> 24)
	tmp32 = t.Vote
	bs[33] = byte(tmp32)
	bs[34] = byte(tmp32 >> 8)
	bs[35] = byte(tmp32 >> 16)
	bs[36] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.MajRequest.Marshal(wire)
	bs = b[:4]
	tmp32 = t.StartIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:]
	alen2 := int64(len(t.PQEntries))
	if wlen := binary.PutVarint(bs, alen2); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Marshal(wire)
	}
}

func (t *BenOrConsensusReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [37]byte
	var bs []byte
	bs = b[:37]
	if _, err := io.ReadAtLeast(wire, bs, 37); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CommitIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.LogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.LogLength = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.BenOrMsgValid = uint8(bs[20])
	t.Iteration = int32((uint32(bs[21]) | (uint32(bs[22]) << 8) | (uint32(bs[23]) << 16) | (uint32(bs[24]) << 24)))
	t.Phase = int32((uint32(bs[25]) | (uint32(bs[26]) << 8) | (uint32(bs[27]) << 16) | (uint32(bs[28]) << 24)))
	t.Stage = int32((uint32(bs[29]) | (uint32(bs[30]) << 8) | (uint32(bs[31]) << 16) | (uint32(bs[32]) << 24)))
	t.Vote = int32((uint32(bs[33]) | (uint32(bs[34]) << 8) | (uint32(bs[35]) << 16) | (uint32(bs[36]) << 24)))
	t.MajRequest.Unmarshal(wire)
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.StartIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	alen2, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.PQEntries = make([]Entry, alen2)
	for i := int64(0); i < alen2; i++ {
		t.PQEntries[i].Unmarshal(wire)
	}
	return nil
}
