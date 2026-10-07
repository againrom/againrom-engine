package ui

import (
	"encoding/binary"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"againrom/pkg/audio"
)

type DeliveryDevicePlayer interface {
	Play()
	IsPlaying() bool
	Pause()
	Seek(time.Duration) error
	Position() time.Duration
	SetVolume(float64)
	Volume() float64
	Close() error
}

type DeliveryPlayerFactory func(io.ReadSeeker) (DeliveryDevicePlayer, error)

var deliveryFactory struct {
	sync.Mutex
	value DeliveryPlayerFactory
}

func SetDeliveryPlayerFactory(factory DeliveryPlayerFactory) func() {
	deliveryFactory.Lock()
	previous := deliveryFactory.value
	deliveryFactory.value = factory
	deliveryFactory.Unlock()
	return func() {
		deliveryFactory.Lock()
		deliveryFactory.value = previous
		deliveryFactory.Unlock()
	}
}

type DeliveryBufferReceipt struct {
	Sequence uint64
	Buffer   uint64
	Action   string
	Phase    int64
	Gain     float64
}

type DeliveryBufferState struct {
	ID         uint64
	Playing    bool
	Repeat     bool
	Phase      int64
	Placement  audio.Placement
	Gain       float64
	Group      audio.Channel
	DeviceGain float64
	Physical   bool
}

type DeliveryBackendState struct {
	Created, Destroyed uint64
	Buffers            []DeliveryBufferState
	Receipts           []DeliveryBufferReceipt
}

type SharedAudio struct {
	Service       *audio.Service
	mu            sync.Mutex
	factory       DeliveryPlayerFactory
	next          uint64
	closed        uint64
	seq           uint64
	buffers       map[uint64]*deliveryBuffer
	log           []DeliveryBufferReceipt
	st            [audio.ChannelCount]audio.Settings
	globalScope   audio.ScopeID
	globalSamples map[string]audio.SampleID
	sampleMu      sync.Mutex
}

func OpenSharedAudio(effects, speech audio.Settings) (*SharedAudio, error) {
	ctx, err := openAudioContext()
	if err != nil {
		return nil, err
	}
	deliveryFactory.Lock()
	factory := deliveryFactory.value
	deliveryFactory.Unlock()
	if factory == nil {
		factory = func(reader io.ReadSeeker) (DeliveryDevicePlayer, error) {
			player, err := ctx.NewPlayer(reader)
			if err != nil {
				return nil, err
			}
			return newDevicePlayer(player), nil
		}
	}
	s := &SharedAudio{factory: factory, buffers: make(map[uint64]*deliveryBuffer)}
	s.Service = audio.NewDelivery(audio.Limits{}, s.newBuffer)
	s.globalScope = s.Service.NewScope()
	s.globalSamples = make(map[string]audio.SampleID)
	s.SetSettings(audio.EffectsChannel, effects)
	s.SetSettings(audio.SpeechChannel, speech)
	return s, nil
}

type AudioScope struct {
	shared  *SharedAudio
	id      audio.ScopeID
	mu      sync.Mutex
	samples map[string]audio.SampleID
	closed  bool
}

func (s *AudioScope) Owner() *SharedAudio {
	if s == nil {
		return nil
	}
	return s.shared
}

func (s *SharedAudio) NewScope() *AudioScope {
	if s == nil {
		return nil
	}
	return &AudioScope{shared: s, id: s.Service.NewScope(), samples: make(map[string]audio.SampleID)}
}

func (s *AudioScope) Player(group audio.Channel) audio.Player {
	if s == nil {
		return nil
	}
	return &deliveryPlayer{scope: s, group: group}
}

func (s *AudioScope) Ambient() AmbientDevice {
	if s == nil {
		return nil
	}
	return &deliveryAmbient{scope: s}
}

func (s *AudioScope) Destroy() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.samples = nil
	s.mu.Unlock()
	if s.shared != nil {
		s.shared.Service.DestroyScope(s.id)
	}
}

func (s *AudioScope) sample(selector string, sample audio.Sample) audio.SampleID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || selector == "" {
		return 0
	}
	if strings.HasPrefix(selector, "registry:") || strings.HasPrefix(selector, "voice:") {
		return s.shared.processSample(selector, sample)
	}
	if id := s.samples[selector]; id != 0 {
		return id
	}
	id := s.shared.Service.RegisterSample(s.id, selector, sample)
	s.samples[selector] = id
	return id
}

func (s *SharedAudio) processSample(selector string, sample audio.Sample) audio.SampleID {
	s.sampleMu.Lock()
	defer s.sampleMu.Unlock()
	if id := s.globalSamples[selector]; id != 0 {
		return id
	}
	id := s.Service.RegisterSample(s.globalScope, selector, sample)
	s.globalSamples[selector] = id
	return id
}

func (s *AudioScope) destroySample(id audio.SampleID) {
	s.mu.Lock()
	for selector, sample := range s.samples {
		if sample == id {
			delete(s.samples, selector)
		}
	}
	s.mu.Unlock()
	s.shared.sampleMu.Lock()
	for selector, sample := range s.shared.globalSamples {
		if sample == id {
			delete(s.shared.globalSamples, selector)
		}
	}
	s.shared.sampleMu.Unlock()
	s.shared.Service.DestroySample(id)
}

type deliveryPlayer struct {
	scope *AudioScope
	group audio.Channel
}

func (p *deliveryPlayer) Play(audio.Sample, audio.Placement)                   {}
func (p *deliveryPlayer) StartVoice(audio.Sample, audio.Placement) audio.Voice { return nil }

func (p *deliveryPlayer) RequestSample(sample audio.Sample, request audio.Request) audio.Voice {
	if p == nil || p.scope == nil || request.Source == "" || request.Recipe == "" {
		return nil
	}
	request.Sample = p.scope.sample(request.Selector, sample)
	request.Scope = p.scope.id
	voice := p.scope.shared.Service.Request(request)
	if voice == nil {
		return nil
	}
	return &deliveryVoice{DeliveryVoice: voice, scope: p.scope}
}

func (p *deliveryPlayer) SetSettings(st audio.Settings) { p.scope.shared.SetSettings(p.group, st) }

func (p *deliveryPlayer) Close() { p.scope.shared.Service.Close() }

type deliveryVoice struct {
	*audio.DeliveryVoice
	scope *AudioScope
}

func (v *deliveryVoice) Destroy() {
	if v != nil {
		v.scope.destroySample(v.ID().Sample)
	}
}

func DeliveryOwner(player audio.Player) *SharedAudio {
	if p, ok := player.(*deliveryPlayer); ok && p != nil && p.scope != nil {
		return p.scope.shared
	}
	return nil
}

func NewAudioScope(player audio.Player) *AudioScope {
	if owner := DeliveryOwner(player); owner != nil {
		return owner.NewScope()
	}
	return nil
}

func DestroyAudioScope(player audio.Player) {
	if p, ok := player.(*deliveryPlayer); ok && p != nil {
		p.scope.Destroy()
	}
}

func (v *Viewer) SetScopedAudio(effects, speech audio.Player, bank SoundBank, ambient AmbientDevice, seed int64) {
	v.DestroyAudio()
	v.audioScope = NewAudioScope(effects)
	if v.audioScope != nil {
		effects, speech, ambient = v.audioScope.Player(audio.EffectsChannel), v.audioScope.Player(audio.SpeechChannel), v.audioScope.Ambient()
	}
	v.SetAudio(effects, bank)
	v.SetSpeechAudio(speech)
	v.SetAmbientAudio(effects, bank, ambient, seed)
}

func (v *Viewer) DestroyAudio() {
	if v == nil {
		return
	}
	v.StopAmbient()
	v.audioScope.Destroy()
	v.audioScope = nil
}

func sharedDeviceState(owner *SharedAudio, group audio.Channel) (audio.Settings, []float64, bool) {
	if owner == nil {
		return audio.Settings{}, nil, false
	}
	owner.mu.Lock()
	settings := owner.st[group]
	owner.mu.Unlock()
	var gains []float64
	for _, buffer := range owner.BackendState().Buffers {
		if buffer.Playing && buffer.Group == group {
			gains = append(gains, buffer.Gain)
		}
	}
	return settings, gains, true
}

func (s *SharedAudio) SetSettings(group audio.Channel, st audio.Settings) {
	if s == nil || group >= audio.ChannelCount {
		return
	}
	s.mu.Lock()
	s.st[group] = st
	s.mu.Unlock()
	s.Service.SetSettings(group, st)
}

func (s *SharedAudio) record(buffer uint64, action string, phase int64, gain float64) {
	s.mu.Lock()
	s.seq++
	s.log = append(s.log, DeliveryBufferReceipt{Sequence: s.seq, Buffer: buffer, Action: action, Phase: phase, Gain: gain})
	if len(s.log) > 1024 {
		copy(s.log, s.log[len(s.log)-1024:])
		s.log = s.log[:1024]
	}
	s.mu.Unlock()
}

func (s *SharedAudio) BackendState() DeliveryBackendState {
	if s == nil {
		return DeliveryBackendState{}
	}
	s.mu.Lock()
	out := DeliveryBackendState{Created: s.next, Destroyed: s.closed, Receipts: append([]DeliveryBufferReceipt(nil), s.log...)}
	var buffers []*deliveryBuffer
	for _, buffer := range s.buffers {
		buffers = append(buffers, buffer)
	}
	s.mu.Unlock()
	for _, buffer := range buffers {
		buffer.mu.Lock()
		if !buffer.destroyed {
			buffer.reader.mu.Lock()
			repeat, placement := buffer.reader.repeat, buffer.reader.placement
			buffer.reader.mu.Unlock()
			state := DeliveryBufferState{ID: buffer.id, Playing: buffer.player.IsPlaying(), Repeat: repeat, Phase: buffer.phaseLocked(), Placement: placement, Gain: buffer.player.Volume(), Group: buffer.group}
			if player, ok := buffer.player.(interface{ DeviceGain() float64 }); ok {
				state.Physical, state.DeviceGain = true, player.DeviceGain()
			}
			out.Buffers = append(out.Buffers, state)
		}
		buffer.mu.Unlock()
	}
	sort.Slice(out.Buffers, func(i, j int) bool { return out.Buffers[i].ID < out.Buffers[j].ID })
	return out
}

func (s *SharedAudio) newBuffer(sample audio.Sample, repeat bool, placement audio.Placement) audio.Buffer {
	if sample.Rate != audio.DeviceRate || len(sample.PCM) == 0 {
		return nil
	}
	reader := &deliveryReader{sample: sample, placement: placement, repeat: repeat}
	player, err := s.factory(reader)
	if err != nil || player == nil {
		if player != nil {
			_ = player.Close()
		}
		return nil
	}
	s.mu.Lock()
	s.next++
	buffer := &deliveryBuffer{shared: s, id: s.next, reader: reader, player: player}
	s.buffers[buffer.id] = buffer
	s.mu.Unlock()
	s.record(buffer.id, "create", 0, player.Volume())
	return buffer
}

type deliveryBuffer struct {
	shared    *SharedAudio
	id        uint64
	mu        sync.Mutex
	reader    *deliveryReader
	player    DeliveryDevicePlayer
	destroyed bool
	group     audio.Channel
}

func (b *deliveryBuffer) SetGroup(group audio.Channel) {
	b.mu.Lock()
	b.group = group
	b.mu.Unlock()
}

func (b *deliveryBuffer) phaseLocked() int64 {
	if b.destroyed || len(b.reader.sample.PCM) == 0 {
		return 0
	}
	frames := int64(b.player.Position()) * int64(b.reader.sample.Rate) / int64(time.Second)
	return frames % int64(len(b.reader.sample.PCM))
}

func (b *deliveryBuffer) Play() {
	b.mu.Lock()
	if !b.destroyed {
		b.player.Play()
		b.shared.record(b.id, "play", b.phaseLocked(), b.player.Volume())
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) Playing() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return !b.destroyed && b.player.IsPlaying()
}

func (b *deliveryBuffer) Stop() {
	b.mu.Lock()
	if !b.destroyed {
		b.player.Pause()
		b.shared.record(b.id, "stop", b.phaseLocked(), b.player.Volume())
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) Rewind() {
	b.mu.Lock()
	if !b.destroyed {
		_ = b.player.Seek(0)
		b.shared.record(b.id, "rewind", b.phaseLocked(), b.player.Volume())
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) Phase() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.phaseLocked()
}

func (b *deliveryBuffer) Move(placement audio.Placement) {
	b.mu.Lock()
	if !b.destroyed {
		b.reader.mu.Lock()
		b.reader.placement = placement
		b.reader.mu.Unlock()
		b.shared.record(b.id, "move", b.phaseLocked(), b.player.Volume())
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) SetSettings(st audio.Settings) {
	b.mu.Lock()
	if !b.destroyed {
		b.player.SetVolume(musicVolume(st))
		b.shared.record(b.id, "settings", b.phaseLocked(), b.player.Volume())
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) SetRepeat(repeat bool) {
	b.mu.Lock()
	if !b.destroyed {
		b.reader.mu.Lock()
		b.reader.repeat = repeat
		b.reader.mu.Unlock()
	}
	b.mu.Unlock()
}

func (b *deliveryBuffer) Destroy() {
	b.mu.Lock()
	if !b.destroyed {
		phase, gain := b.phaseLocked(), b.player.Volume()
		_ = b.player.Close()
		b.destroyed = true
		b.shared.mu.Lock()
		delete(b.shared.buffers, b.id)
		b.shared.closed++
		b.shared.mu.Unlock()
		b.shared.record(b.id, "destroy", phase, gain)
	}
	b.mu.Unlock()
}

type deliveryReader struct {
	mu        sync.Mutex
	sample    audio.Sample
	placement audio.Placement
	repeat    bool
	position  int64
}

func (r *deliveryReader) Read(out []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sample.PCM) == 0 {
		return 0, io.EOF
	}
	n := 0
	for n+4 <= len(out) {
		if r.position >= int64(len(r.sample.PCM))*4 {
			if !r.repeat {
				if n == 0 {
					return 0, io.EOF
				}
				return n, nil
			}
			r.position = 0
		}
		value := int64(r.sample.PCM[r.position/4])
		gain := func(gain int) uint16 {
			return uint16(int16(max(int64(-32768), min(int64(32767), value*int64(gain)/audio.GainUnit))))
		}
		binary.LittleEndian.PutUint16(out[n:], gain(r.placement.Left))
		binary.LittleEndian.PutUint16(out[n+2:], gain(r.placement.Right))
		r.position += 4
		n += 4
	}
	return n, nil
}

func (r *deliveryReader) Seek(offset int64, whence int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	position := offset
	switch whence {
	case io.SeekCurrent:
		position += r.position
	case io.SeekEnd:
		position += int64(len(r.sample.PCM)) * 4
	}
	r.position = max(int64(0), position-position%4)
	return r.position, nil
}

type deliveryAmbient struct {
	scope  *AudioScope
	voices [ambientLoopCount]audio.Voice
}

func (d *deliveryAmbient) StartLoop(AmbientLoop, audio.Sample, audio.Placement) {}

func (d *deliveryAmbient) RequestLoop(kind AmbientLoop, sample audio.Sample, request audio.Request) audio.Voice {
	if !validAmbientLoop(kind) {
		return nil
	}
	if voice := d.voices[kind]; voice != nil && voice.Playing() {
		return voice
	}
	voice := audio.Dispatch(d.scope.Player(audio.EffectsChannel), sample, request)
	d.voices[kind] = voice
	return voice
}

func (d *deliveryAmbient) MoveLoop(kind AmbientLoop, placement audio.Placement) {
	if validAmbientLoop(kind) {
		if voice, ok := d.voices[kind].(interface{ Move(audio.Placement) }); ok {
			voice.Move(placement)
		}
	}
}

func (d *deliveryAmbient) MoveLoopRequest(kind AmbientLoop, request audio.Request) {
	if validAmbientLoop(kind) {
		if voice, ok := d.voices[kind].(interface{ MoveRequest(audio.Request) }); ok {
			voice.MoveRequest(request)
		}
	}
}

func (d *deliveryAmbient) StopLoop(kind AmbientLoop) {
	if validAmbientLoop(kind) {
		audio.StopReset(d.voices[kind])
		d.voices[kind] = nil
	}
}

func (d *deliveryAmbient) Stop() {
	for kind := AmbientLoop(0); kind < ambientLoopCount; kind++ {
		d.StopLoop(kind)
	}
}

func (d *deliveryAmbient) SetSettings(st audio.Settings) {
	d.scope.shared.SetSettings(audio.EffectsChannel, st)
}

func RequestAmbient(device AmbientDevice, kind AmbientLoop, sample audio.Sample, request audio.Request) audio.Voice {
	if typed, ok := device.(interface {
		RequestLoop(AmbientLoop, audio.Sample, audio.Request) audio.Voice
	}); ok {
		return typed.RequestLoop(kind, sample, request)
	}
	return nil
}
