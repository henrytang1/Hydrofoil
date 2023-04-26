package genericsmrproto

import (
	"io"
	"sync"
)

func (t *BeTheLeaderArgs) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type BeTheLeaderArgsCache struct {
	mu	sync.Mutex
	cache	[]*BeTheLeaderArgs
}

func NewBeTheLeaderArgsCache() *BeTheLeaderArgsCache {
	c := &BeTheLeaderArgsCache{}
	c.cache = make([]*BeTheLeaderArgs, 0)
	return c
}

func (p *BeTheLeaderArgsCache) Get() *BeTheLeaderArgs {
	var t *BeTheLeaderArgs
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BeTheLeaderArgs{}
	}
	return t
}
func (p *BeTheLeaderArgsCache) Put(t *BeTheLeaderArgs) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BeTheLeaderArgs) Marshal(wire io.Writer) {
}

func (t *BeTheLeaderArgs) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *GetState) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type GetStateCache struct {
	mu	sync.Mutex
	cache	[]*GetState
}

func NewGetStateCache() *GetStateCache {
	c := &GetStateCache{}
	c.cache = make([]*GetState, 0)
	return c
}

func (p *GetStateCache) Get() *GetState {
	var t *GetState
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetState{}
	}
	return t
}
func (p *GetStateCache) Put(t *GetState) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetState) Marshal(wire io.Writer) {
}

func (t *GetState) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *Slowdown) BinarySize() (nbytes int, sizeKnown bool) {
	return 4, true
}

type SlowdownCache struct {
	mu	sync.Mutex
	cache	[]*Slowdown
}

func NewSlowdownCache() *SlowdownCache {
	c := &SlowdownCache{}
	c.cache = make([]*Slowdown, 0)
	return c
}

func (p *SlowdownCache) Get() *Slowdown {
	var t *Slowdown
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Slowdown{}
	}
	return t
}
func (p *SlowdownCache) Put(t *Slowdown) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Slowdown) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.TimeInMs
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *Slowdown) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.TimeInMs = uint32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	return nil
}

func (t *Connect) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type ConnectCache struct {
	mu	sync.Mutex
	cache	[]*Connect
}

func NewConnectCache() *ConnectCache {
	c := &ConnectCache{}
	c.cache = make([]*Connect, 0)
	return c
}

func (p *ConnectCache) Get() *Connect {
	var t *Connect
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Connect{}
	}
	return t
}
func (p *ConnectCache) Put(t *Connect) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Connect) Marshal(wire io.Writer) {
}

func (t *Connect) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *Disconnect) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type DisconnectCache struct {
	mu	sync.Mutex
	cache	[]*Disconnect
}

func NewDisconnectCache() *DisconnectCache {
	c := &DisconnectCache{}
	c.cache = make([]*Disconnect, 0)
	return c
}

func (p *DisconnectCache) Get() *Disconnect {
	var t *Disconnect
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Disconnect{}
	}
	return t
}
func (p *DisconnectCache) Put(t *Disconnect) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Disconnect) Marshal(wire io.Writer) {
}

func (t *Disconnect) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *ProposeAndRead) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ProposeAndReadCache struct {
	mu	sync.Mutex
	cache	[]*ProposeAndRead
}

func NewProposeAndReadCache() *ProposeAndReadCache {
	c := &ProposeAndReadCache{}
	c.cache = make([]*ProposeAndRead, 0)
	return c
}

func (p *ProposeAndReadCache) Get() *ProposeAndRead {
	var t *ProposeAndRead
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ProposeAndRead{}
	}
	return t
}
func (p *ProposeAndReadCache) Put(t *ProposeAndRead) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ProposeAndRead) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.CommandId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Command.Marshal(wire)
	t.Key.Marshal(wire)
}

func (t *ProposeAndRead) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.CommandId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Command.Unmarshal(wire)
	t.Key.Unmarshal(wire)
	return nil
}

func (t *ProposeAndReadReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ProposeAndReadReplyCache struct {
	mu	sync.Mutex
	cache	[]*ProposeAndReadReply
}

func NewProposeAndReadReplyCache() *ProposeAndReadReplyCache {
	c := &ProposeAndReadReplyCache{}
	c.cache = make([]*ProposeAndReadReply, 0)
	return c
}

func (p *ProposeAndReadReplyCache) Get() *ProposeAndReadReply {
	var t *ProposeAndReadReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ProposeAndReadReply{}
	}
	return t
}
func (p *ProposeAndReadReplyCache) Put(t *ProposeAndReadReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ProposeAndReadReply) Marshal(wire io.Writer) {
	var b [5]byte
	var bs []byte
	bs = b[:5]
	bs[0] = byte(t.OK)
	tmp32 := t.CommandId
	bs[1] = byte(tmp32)
	bs[2] = byte(tmp32 >> 8)
	bs[3] = byte(tmp32 >> 16)
	bs[4] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Value.Marshal(wire)
}

func (t *ProposeAndReadReply) Unmarshal(wire io.Reader) error {
	var b [5]byte
	var bs []byte
	bs = b[:5]
	if _, err := io.ReadAtLeast(wire, bs, 5); err != nil {
		return err
	}
	t.OK = uint8(bs[0])
	t.CommandId = int32((uint32(bs[1]) | (uint32(bs[2]) << 8) | (uint32(bs[3]) << 16) | (uint32(bs[4]) << 24)))
	t.Value.Unmarshal(wire)
	return nil
}

func (t *PingReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type PingReplyCache struct {
	mu	sync.Mutex
	cache	[]*PingReply
}

func NewPingReplyCache() *PingReplyCache {
	c := &PingReplyCache{}
	c.cache = make([]*PingReply, 0)
	return c
}

func (p *PingReplyCache) Get() *PingReply {
	var t *PingReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &PingReply{}
	}
	return t
}
func (p *PingReplyCache) Put(t *PingReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *PingReply) Marshal(wire io.Writer) {
}

func (t *PingReply) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *DisconnectReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type DisconnectReplyCache struct {
	mu	sync.Mutex
	cache	[]*DisconnectReply
}

func NewDisconnectReplyCache() *DisconnectReplyCache {
	c := &DisconnectReplyCache{}
	c.cache = make([]*DisconnectReply, 0)
	return c
}

func (p *DisconnectReplyCache) Get() *DisconnectReply {
	var t *DisconnectReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &DisconnectReply{}
	}
	return t
}
func (p *DisconnectReplyCache) Put(t *DisconnectReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *DisconnectReply) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.Success)
	wire.Write(bs)
}

func (t *DisconnectReply) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.Success = uint8(bs[0])
	return nil
}

func (t *ConnectReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type ConnectReplyCache struct {
	mu	sync.Mutex
	cache	[]*ConnectReply
}

func NewConnectReplyCache() *ConnectReplyCache {
	c := &ConnectReplyCache{}
	c.cache = make([]*ConnectReply, 0)
	return c
}

func (p *ConnectReplyCache) Get() *ConnectReply {
	var t *ConnectReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ConnectReply{}
	}
	return t
}
func (p *ConnectReplyCache) Put(t *ConnectReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ConnectReply) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.Success)
	wire.Write(bs)
}

func (t *ConnectReply) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.Success = uint8(bs[0])
	return nil
}

func (t *ReadReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ReadReplyCache struct {
	mu	sync.Mutex
	cache	[]*ReadReply
}

func NewReadReplyCache() *ReadReplyCache {
	c := &ReadReplyCache{}
	c.cache = make([]*ReadReply, 0)
	return c
}

func (p *ReadReplyCache) Get() *ReadReply {
	var t *ReadReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ReadReply{}
	}
	return t
}
func (p *ReadReplyCache) Put(t *ReadReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ReadReply) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.CommandId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Value.Marshal(wire)
}

func (t *ReadReply) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.CommandId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Value.Unmarshal(wire)
	return nil
}

func (t *RegisterClientIdArgs) BinarySize() (nbytes int, sizeKnown bool) {
	return 4, true
}

type RegisterClientIdArgsCache struct {
	mu	sync.Mutex
	cache	[]*RegisterClientIdArgs
}

func NewRegisterClientIdArgsCache() *RegisterClientIdArgsCache {
	c := &RegisterClientIdArgsCache{}
	c.cache = make([]*RegisterClientIdArgs, 0)
	return c
}

func (p *RegisterClientIdArgsCache) Get() *RegisterClientIdArgs {
	var t *RegisterClientIdArgs
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &RegisterClientIdArgs{}
	}
	return t
}
func (p *RegisterClientIdArgsCache) Put(t *RegisterClientIdArgs) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *RegisterClientIdArgs) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.ClientId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *RegisterClientIdArgs) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.ClientId = uint32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	return nil
}

func (t *GetStateReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type GetStateReplyCache struct {
	mu	sync.Mutex
	cache	[]*GetStateReply
}

func NewGetStateReplyCache() *GetStateReplyCache {
	c := &GetStateReplyCache{}
	c.cache = make([]*GetStateReply, 0)
	return c
}

func (p *GetStateReplyCache) Get() *GetStateReply {
	var t *GetStateReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetStateReply{}
	}
	return t
}
func (p *GetStateReplyCache) Put(t *GetStateReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetStateReply) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.IsLeader)
	wire.Write(bs)
}

func (t *GetStateReply) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.IsLeader = uint8(bs[0])
	return nil
}

func (t *BeTheLeaderReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type BeTheLeaderReplyCache struct {
	mu	sync.Mutex
	cache	[]*BeTheLeaderReply
}

func NewBeTheLeaderReplyCache() *BeTheLeaderReplyCache {
	c := &BeTheLeaderReplyCache{}
	c.cache = make([]*BeTheLeaderReply, 0)
	return c
}

func (p *BeTheLeaderReplyCache) Get() *BeTheLeaderReply {
	var t *BeTheLeaderReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BeTheLeaderReply{}
	}
	return t
}
func (p *BeTheLeaderReplyCache) Put(t *BeTheLeaderReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BeTheLeaderReply) Marshal(wire io.Writer) {
}

func (t *BeTheLeaderReply) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *GetView) BinarySize() (nbytes int, sizeKnown bool) {
	return 4, true
}

type GetViewCache struct {
	mu	sync.Mutex
	cache	[]*GetView
}

func NewGetViewCache() *GetViewCache {
	c := &GetViewCache{}
	c.cache = make([]*GetView, 0)
	return c
}

func (p *GetViewCache) Get() *GetView {
	var t *GetView
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetView{}
	}
	return t
}
func (p *GetViewCache) Put(t *GetView) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetView) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.PilotId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *GetView) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.PilotId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	return nil
}

func (t *GetViewReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 13, true
}

type GetViewReplyCache struct {
	mu	sync.Mutex
	cache	[]*GetViewReply
}

func NewGetViewReplyCache() *GetViewReplyCache {
	c := &GetViewReplyCache{}
	c.cache = make([]*GetViewReply, 0)
	return c
}

func (p *GetViewReplyCache) Get() *GetViewReply {
	var t *GetViewReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetViewReply{}
	}
	return t
}
func (p *GetViewReplyCache) Put(t *GetViewReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetViewReply) Marshal(wire io.Writer) {
	var b [13]byte
	var bs []byte
	bs = b[:13]
	bs[0] = byte(t.OK)
	tmp32 := t.ViewId
	bs[1] = byte(tmp32)
	bs[2] = byte(tmp32 >> 8)
	bs[3] = byte(tmp32 >> 16)
	bs[4] = byte(tmp32 >> 24)
	tmp32 = t.PilotId
	bs[5] = byte(tmp32)
	bs[6] = byte(tmp32 >> 8)
	bs[7] = byte(tmp32 >> 16)
	bs[8] = byte(tmp32 >> 24)
	tmp32 = t.ReplicaId
	bs[9] = byte(tmp32)
	bs[10] = byte(tmp32 >> 8)
	bs[11] = byte(tmp32 >> 16)
	bs[12] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *GetViewReply) Unmarshal(wire io.Reader) error {
	var b [13]byte
	var bs []byte
	bs = b[:13]
	if _, err := io.ReadAtLeast(wire, bs, 13); err != nil {
		return err
	}
	t.OK = uint8(bs[0])
	t.ViewId = int32((uint32(bs[1]) | (uint32(bs[2]) << 8) | (uint32(bs[3]) << 16) | (uint32(bs[4]) << 24)))
	t.PilotId = int32((uint32(bs[5]) | (uint32(bs[6]) << 8) | (uint32(bs[7]) << 16) | (uint32(bs[8]) << 24)))
	t.ReplicaId = int32((uint32(bs[9]) | (uint32(bs[10]) << 8) | (uint32(bs[11]) << 16) | (uint32(bs[12]) << 24)))
	return nil
}

func (t *SlowdownReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type SlowdownReplyCache struct {
	mu	sync.Mutex
	cache	[]*SlowdownReply
}

func NewSlowdownReplyCache() *SlowdownReplyCache {
	c := &SlowdownReplyCache{}
	c.cache = make([]*SlowdownReply, 0)
	return c
}

func (p *SlowdownReplyCache) Get() *SlowdownReply {
	var t *SlowdownReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &SlowdownReply{}
	}
	return t
}
func (p *SlowdownReplyCache) Put(t *SlowdownReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *SlowdownReply) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.Success)
	wire.Write(bs)
}

func (t *SlowdownReply) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.Success = uint8(bs[0])
	return nil
}

func (t *ProposeReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 5, true
}

type ProposeReplyCache struct {
	mu	sync.Mutex
	cache	[]*ProposeReply
}

func NewProposeReplyCache() *ProposeReplyCache {
	c := &ProposeReplyCache{}
	c.cache = make([]*ProposeReply, 0)
	return c
}

func (p *ProposeReplyCache) Get() *ProposeReply {
	var t *ProposeReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ProposeReply{}
	}
	return t
}
func (p *ProposeReplyCache) Put(t *ProposeReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ProposeReply) Marshal(wire io.Writer) {
	var b [5]byte
	var bs []byte
	bs = b[:5]
	bs[0] = byte(t.OK)
	tmp32 := t.CommandId
	bs[1] = byte(tmp32)
	bs[2] = byte(tmp32 >> 8)
	bs[3] = byte(tmp32 >> 16)
	bs[4] = byte(tmp32 >> 24)
	wire.Write(bs)
}

func (t *ProposeReply) Unmarshal(wire io.Reader) error {
	var b [5]byte
	var bs []byte
	bs = b[:5]
	if _, err := io.ReadAtLeast(wire, bs, 5); err != nil {
		return err
	}
	t.OK = uint8(bs[0])
	t.CommandId = int32((uint32(bs[1]) | (uint32(bs[2]) << 8) | (uint32(bs[3]) << 16) | (uint32(bs[4]) << 24)))
	return nil
}

func (t *ProposeReplyTS) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ProposeReplyTSCache struct {
	mu	sync.Mutex
	cache	[]*ProposeReplyTS
}

func NewProposeReplyTSCache() *ProposeReplyTSCache {
	c := &ProposeReplyTSCache{}
	c.cache = make([]*ProposeReplyTS, 0)
	return c
}

func (p *ProposeReplyTSCache) Get() *ProposeReplyTS {
	var t *ProposeReplyTS
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &ProposeReplyTS{}
	}
	return t
}
func (p *ProposeReplyTSCache) Put(t *ProposeReplyTS) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *ProposeReplyTS) Marshal(wire io.Writer) {
	var b [8]byte
	var bs []byte
	bs = b[:5]
	bs[0] = byte(t.OK)
	tmp32 := t.CommandId
	bs[1] = byte(tmp32)
	bs[2] = byte(tmp32 >> 8)
	bs[3] = byte(tmp32 >> 16)
	bs[4] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Value.Marshal(wire)
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

func (t *ProposeReplyTS) Unmarshal(wire io.Reader) error {
	var b [8]byte
	var bs []byte
	bs = b[:5]
	if _, err := io.ReadAtLeast(wire, bs, 5); err != nil {
		return err
	}
	t.OK = uint8(bs[0])
	t.CommandId = int32((uint32(bs[1]) | (uint32(bs[2]) << 8) | (uint32(bs[3]) << 16) | (uint32(bs[4]) << 24)))
	t.Value.Unmarshal(wire)
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.Timestamp = int64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
	return nil
}

func (t *PingArgs) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type PingArgsCache struct {
	mu	sync.Mutex
	cache	[]*PingArgs
}

func NewPingArgsCache() *PingArgsCache {
	c := &PingArgsCache{}
	c.cache = make([]*PingArgs, 0)
	return c
}

func (p *PingArgsCache) Get() *PingArgs {
	var t *PingArgs
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &PingArgs{}
	}
	return t
}
func (p *PingArgsCache) Put(t *PingArgs) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *PingArgs) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.ActAsLeader)
	wire.Write(bs)
}

func (t *PingArgs) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.ActAsLeader = uint8(bs[0])
	return nil
}

func (t *BeaconReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 8, true
}

type BeaconReplyCache struct {
	mu	sync.Mutex
	cache	[]*BeaconReply
}

func NewBeaconReplyCache() *BeaconReplyCache {
	c := &BeaconReplyCache{}
	c.cache = make([]*BeaconReply, 0)
	return c
}

func (p *BeaconReplyCache) Get() *BeaconReply {
	var t *BeaconReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &BeaconReply{}
	}
	return t
}
func (p *BeaconReplyCache) Put(t *BeaconReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *BeaconReply) Marshal(wire io.Writer) {
	var b [8]byte
	var bs []byte
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

func (t *BeaconReply) Unmarshal(wire io.Reader) error {
	var b [8]byte
	var bs []byte
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.Timestamp = uint64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
	return nil
}

func (t *GetStatus) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, true
}

type GetStatusCache struct {
	mu	sync.Mutex
	cache	[]*GetStatus
}

func NewGetStatusCache() *GetStatusCache {
	c := &GetStatusCache{}
	c.cache = make([]*GetStatus, 0)
	return c
}

func (p *GetStatusCache) Get() *GetStatus {
	var t *GetStatus
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &GetStatus{}
	}
	return t
}
func (p *GetStatusCache) Put(t *GetStatus) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *GetStatus) Marshal(wire io.Writer) {
}

func (t *GetStatus) Unmarshal(wire io.Reader) error {
	return nil
}

func (t *RegisterClientIdReply) BinarySize() (nbytes int, sizeKnown bool) {
	return 1, true
}

type RegisterClientIdReplyCache struct {
	mu	sync.Mutex
	cache	[]*RegisterClientIdReply
}

func NewRegisterClientIdReplyCache() *RegisterClientIdReplyCache {
	c := &RegisterClientIdReplyCache{}
	c.cache = make([]*RegisterClientIdReply, 0)
	return c
}

func (p *RegisterClientIdReplyCache) Get() *RegisterClientIdReply {
	var t *RegisterClientIdReply
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &RegisterClientIdReply{}
	}
	return t
}
func (p *RegisterClientIdReplyCache) Put(t *RegisterClientIdReply) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *RegisterClientIdReply) Marshal(wire io.Writer) {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	bs[0] = byte(t.OK)
	wire.Write(bs)
}

func (t *RegisterClientIdReply) Unmarshal(wire io.Reader) error {
	var b [1]byte
	var bs []byte
	bs = b[:1]
	if _, err := io.ReadAtLeast(wire, bs, 1); err != nil {
		return err
	}
	t.OK = uint8(bs[0])
	return nil
}

func (t *Propose) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ProposeCache struct {
	mu	sync.Mutex
	cache	[]*Propose
}

func NewProposeCache() *ProposeCache {
	c := &ProposeCache{}
	c.cache = make([]*Propose, 0)
	return c
}

func (p *ProposeCache) Get() *Propose {
	var t *Propose
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Propose{}
	}
	return t
}
func (p *ProposeCache) Put(t *Propose) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Propose) Marshal(wire io.Writer) {
	var b [8]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.CommandId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Command.Marshal(wire)
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

func (t *Propose) Unmarshal(wire io.Reader) error {
	var b [8]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.CommandId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Command.Unmarshal(wire)
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.Timestamp = int64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
	return nil
}

func (t *Read) BinarySize() (nbytes int, sizeKnown bool) {
	return 0, false
}

type ReadCache struct {
	mu	sync.Mutex
	cache	[]*Read
}

func NewReadCache() *ReadCache {
	c := &ReadCache{}
	c.cache = make([]*Read, 0)
	return c
}

func (p *ReadCache) Get() *Read {
	var t *Read
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Read{}
	}
	return t
}
func (p *ReadCache) Put(t *Read) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Read) Marshal(wire io.Writer) {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	tmp32 := t.CommandId
	bs[0] = byte(tmp32)
	bs[1] = byte(tmp32 >> 8)
	bs[2] = byte(tmp32 >> 16)
	bs[3] = byte(tmp32 >> 24)
	wire.Write(bs)
	t.Key.Marshal(wire)
}

func (t *Read) Unmarshal(wire io.Reader) error {
	var b [4]byte
	var bs []byte
	bs = b[:4]
	if _, err := io.ReadAtLeast(wire, bs, 4); err != nil {
		return err
	}
	t.CommandId = int32((uint32(bs[0]) | (uint32(bs[1]) << 8) | (uint32(bs[2]) << 16) | (uint32(bs[3]) << 24)))
	t.Key.Unmarshal(wire)
	return nil
}

func (t *Beacon) BinarySize() (nbytes int, sizeKnown bool) {
	return 8, true
}

type BeaconCache struct {
	mu	sync.Mutex
	cache	[]*Beacon
}

func NewBeaconCache() *BeaconCache {
	c := &BeaconCache{}
	c.cache = make([]*Beacon, 0)
	return c
}

func (p *BeaconCache) Get() *Beacon {
	var t *Beacon
	p.mu.Lock()
	if len(p.cache) > 0 {
		t = p.cache[len(p.cache)-1]
		p.cache = p.cache[0:(len(p.cache) - 1)]
	}
	p.mu.Unlock()
	if t == nil {
		t = &Beacon{}
	}
	return t
}
func (p *BeaconCache) Put(t *Beacon) {
	p.mu.Lock()
	p.cache = append(p.cache, t)
	p.mu.Unlock()
}
func (t *Beacon) Marshal(wire io.Writer) {
	var b [8]byte
	var bs []byte
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

func (t *Beacon) Unmarshal(wire io.Reader) error {
	var b [8]byte
	var bs []byte
	bs = b[:8]
	if _, err := io.ReadAtLeast(wire, bs, 8); err != nil {
		return err
	}
	t.Timestamp = uint64((uint64(bs[0]) | (uint64(bs[1]) << 8) | (uint64(bs[2]) << 16) | (uint64(bs[3]) << 24) | (uint64(bs[4]) << 32) | (uint64(bs[5]) << 40) | (uint64(bs[6]) << 48) | (uint64(bs[7]) << 56)))
	return nil
}
