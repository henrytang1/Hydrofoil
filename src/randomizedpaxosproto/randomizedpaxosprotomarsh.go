package randomizedpaxosproto

import (
	"io"
	"sync"
	"bufio"
	"encoding/binary"
)

type byteReader interface {
	io.Reader
	ReadByte() (c byte, err error)
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
	var b [12]byte
	var bs []byte
	t.Data.Marshal(wire)
	bs = b[:12]
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
	wire.Write(bs)
	t.BenOrActive.Marshal(wire)
	bs = b[:8]
	tmp64 := t.Timestamp
	bs[0] = byte(tmp64)
	bs[1] = byte(tmp64 >> 8)
	bs[2] = byte(tmp64 >> 16)
	bs[3] = byte(tmp64 >> 24)
	bs[4] = byte(tmp64 >> 32)
	bs[5] = byte(tmp64 >> 40)
	bs[6] = byte(tmp64 >> 48)
	bs[7] = byte(tmp64 >> 56)
	wire.Write(bs)
}

func (t *Entry) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	t.Data.Unmarshal(wire)
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Index = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.BenOrActive.Unmarshal(wire)
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.Timestamp = int64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
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
	var b [16]byte
	var bs []byte
	bs = b[:16]
	tmp32 := t.ReplicaId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.ReplicaBenOrIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.ReplicaPreparedIndex
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.ReplicaEntries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.ReplicaEntries[i].Marshal(wire)
	}
	bs = b[:8]
	tmp32 = t.PrevLogIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.RequestedIndex
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Success.Marshal(wire)
}

func (t *ReplicateEntriesReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [16]byte
	var bs []byte
	bs = b[:16]
	if _, err := io.ReadAtLeast(wire, bs, 16); err != nil {
		return err
	}
	t.ReplicaId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.ReplicaBenOrIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.ReplicaPreparedIndex = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.ReplicaEntries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.ReplicaEntries[i].Unmarshal(wire)
	}
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.PrevLogIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.RequestedIndex = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Success.Unmarshal(wire)
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
	var b [16]byte
	var bs []byte
	bs = b[:12]
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
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.ClientReq.Marshal(wire)
	bs = b[:16]
	tmp64 := t.Timestamp
	bs[0] = byte(tmp64)
	bs[1] = byte(tmp64 >> 8)
	bs[2] = byte(tmp64 >> 16)
	bs[3] = byte(tmp64 >> 24)
	bs[4] = byte(tmp64 >> 32)
	bs[5] = byte(tmp64 >> 40)
	bs[6] = byte(tmp64 >> 48)
	bs[7] = byte(tmp64 >> 56)
	tmp32 = t.RequestTerm
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.Index
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *BenOrBroadcast) Unmarshal(wire io.Reader) error {
	var b [16]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.ClientReq.Unmarshal(wire)
	bs = b[:16]
	if _, err := io.ReadAtLeast(wire, bs, 16); err != nil {
		return err
	}
	t.Timestamp = int64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
	t.RequestTerm = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.Index = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
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
	var b [12]byte
	var bs []byte
	bs = b[:12]
	tmp32 := t.ReplicaId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.CommittedEntry.Marshal(wire)
}

func (t *BenOrBroadcastReply) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.ReplicaId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.CommittedEntry.Unmarshal(wire)
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
	var b [12]byte
	var bs []byte
	bs = b[:12]
	tmp32 := t.ReplicaId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.CommittedEntry.Marshal(wire)
}

func (t *BenOrConsensusReply) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.ReplicaId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.CommittedEntry.Unmarshal(wire)
	return nil
}

func (t *InfoBroadcastReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 12, true
}

type InfoBroadcastReplyCache struct {
	mu	sync.Mutex
	cache	[]*InfoBroadcastReply
}

func NewInfoBroadcastReplyCache() *InfoBroadcastReplyCache {
	c := &InfoBroadcastReplyCache{}
	c.cache = make([]*InfoBroadcastReply, 0)
	return c
}

func (p *InfoBroadcastReplyCache) Get() *InfoBroadcastReply {
	var t *InfoBroadcastReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &InfoBroadcastReply{}
	}
	return t
}
func (p *InfoBroadcastReplyCache) Put(t *InfoBroadcastReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *InfoBroadcastReply) Marshal(wire io.Writer) {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	tmp32 := t.ReplicaId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *InfoBroadcastReply) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.ReplicaId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
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
	var b [16]byte
	var bs []byte
	bs = b[:16]
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
	tmp32 = t.PrevLogIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.PrevLogTerm
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.Entries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Marshal(wire)
	}
	bs = b[:8]
	tmp32 = t.LeaderBenOrIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.LeaderPreparedIndex
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *ReplicateEntries) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [16]byte
	var bs []byte
	bs = b[:16]
	if _, err := io.ReadAtLeast(wire, bs, 16); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.PrevLogIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.PrevLogTerm = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.Entries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.Entries[i].Unmarshal(wire)
	}
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.LeaderBenOrIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.LeaderPreparedIndex = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	return nil
}

func (t *RequestVote) BinarySize() (nbytes int, sizeKnown bool) {
	return 12, true
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
	var b [12]byte
	var bs []byte
	bs = b[:12]
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
	tmp32 = t.CandidateBenOrIndex
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *RequestVote) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.CandidateBenOrIndex = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	return nil
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
	var b [10]byte
	var bs []byte
	bs = b[:8]
	tmp32 := t.ReplicaId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.Term
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.VoteGranted.Marshal(wire)
	tmp32 = t.ReplicaBenOrIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	tmp32 = t.ReplicaPreparedIndex
	bs[4] = byte(tmp32)
	bs[5] = byte(tmp32 >> 8)
	bs[6] = byte(tmp32 >> 16)
	bs[7] = byte(tmp32 >> 24)
	wire.Write(bs)
	bs = b[:]
	alen1 := int64(len(t.ReplicaEntries))
	if wlen := binary.PutVarint(bs, alen1); wlen >= 0 {
		wire.Write(b[0:wlen])
	}
	for i := int64(0); i < alen1; i++ {
		t.ReplicaEntries[i].Marshal(wire)
	}
	bs = b[:4]
	tmp32 = t.CandidateBenOrIndex
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.EntryAtCandidateBenOrIndex.Marshal(wire)
}

func (t *RequestVoteReply) Unmarshal(rr io.Reader) error {
	var wire byteReader
	var ok bool
	if wire, ok = rr.(byteReader); !ok {
		wire = bufio.NewReader(rr)
	}
	var b [10]byte
	var bs []byte
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.ReplicaId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.VoteGranted.Unmarshal(wire)
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.ReplicaBenOrIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.ReplicaPreparedIndex = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	alen1, err := binary.ReadVarint(wire)
	if err != nil {
		return err
	}
	t.ReplicaEntries = make([]Entry, alen1)
	for i := int64(0); i < alen1; i++ {
		t.ReplicaEntries[i].Unmarshal(wire)
	}
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.CandidateBenOrIndex = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.EntryAtCandidateBenOrIndex.Unmarshal(wire)
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
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	tmp32 = t.Phase
	bs[12] = byte(tmp32)
	bs[13] = byte(tmp32 >> 8)
	bs[14] = byte(tmp32 >> 16)
	bs[15] = byte(tmp32 >> 24)
	tmp32 = t.Vote
	bs[16] = byte(tmp32)
	bs[17] = byte(tmp32 >> 8)
	bs[18] = byte(tmp32 >> 16)
	bs[19] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.MajRequest.Marshal(wire)
	t.LeaderRequest.Marshal(wire)
	bs = b[:4]
	tmp32 = t.EntryType
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *BenOrConsensus) Unmarshal(wire io.Reader) error {
	var b [20]byte
	var bs []byte
	bs = b[:20]
	if _, err := io.ReadAtLeast(wire, bs, 20); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.Phase = int32((uint32(bs[12]) | (uint32(bs[13]) << 8) | (uint32(bs[14]) << 16) | (uint32(bs[15]) << 24)))
	t.Vote = int32((uint32(bs[16]) | (uint32(bs[17]) << 8) | (uint32(bs[18]) << 16) | (uint32(bs[19]) << 24)))
	t.MajRequest.Unmarshal(wire)
	t.LeaderRequest.Unmarshal(wire)
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.EntryType = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	return nil
}

func (t *InfoBroadcast) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type InfoBroadcastCache struct {
	mu	sync.Mutex
	cache	[]*InfoBroadcast
}

func NewInfoBroadcastCache() *InfoBroadcastCache {
	c := &InfoBroadcastCache{}
	c.cache = make([]*InfoBroadcast, 0)
	return c
}

func (p *InfoBroadcastCache) Get() *InfoBroadcast {
	var t *InfoBroadcast
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &InfoBroadcast{}
	}
	return t
}
func (p *InfoBroadcastCache) Put(t *InfoBroadcast) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *InfoBroadcast) Marshal(wire io.Writer) {
	var b [12]byte
	var bs []byte
	bs = b[:12]
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
	tmp32 = t.Counter
	bs[8] = byte(tmp32)
	bs[9] = byte(tmp32 >> 8)
	bs[10] = byte(tmp32 >> 16)
	bs[11] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.ClientReq.Marshal(wire)
}

func (t *InfoBroadcast) Unmarshal(wire io.Reader) error {
	var b [12]byte
	var bs []byte
	bs = b[:12]
	if _, err := io.ReadAtLeast(wire, bs, 12); err != nil {
		return err
	}
	t.SenderId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Term = int32((uint32(bs[4]) | (uint32(bs[5]) << 8) | (uint32(bs[6]) << 16) | (uint32(bs[7]) << 24)))
	t.Counter = int32((uint32(bs[8]) | (uint32(bs[9]) << 8) | (uint32(bs[10]) << 16) | (uint32(bs[11]) << 24)))
	t.ClientReq.Unmarshal(wire)
	return nil
}
