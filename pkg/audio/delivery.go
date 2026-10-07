package audio

import (
	"image"
	"sync"
)

type ScopeID uint64
type SampleID uint64

type VoiceID struct {
	Sample     SampleID
	Duplicate  int
	Generation uint64
}

type Limits struct {
	Duplicates int
	Channels   int
	Receipts   int
}

type Request struct {
	Source       string
	Recipe       string
	VolumeTerm   string
	Selector     string
	Sample       SampleID
	Scope        ScopeID
	Placement    Placement
	Repeat       bool
	Group        Channel
	Priority     uint8
	Attenuation  int
	Pan          int
	Frequency    uint32
	SpatialKnown bool
	Geometry     ViewGeometry
	SourceFine   image.Point
}

type Buffer interface {
	Play()
	Playing() bool
	Stop()
	Rewind()
	Move(Placement)
	SetSettings(Settings)
	SetRepeat(bool)
	Phase() int64
	Destroy()
}

type BufferFactory func(Sample, bool, Placement) Buffer

type DeliveryReason string

const (
	DeliveryAdmitted          DeliveryReason = "admitted"
	DeliverySampleBusy        DeliveryReason = "sample-busy"
	DeliveryChannelsBusy      DeliveryReason = "channels-busy"
	DeliveryCompleted         DeliveryReason = "completed"
	DeliveryEvicted           DeliveryReason = "evicted"
	DeliveryStopped           DeliveryReason = "stopped"
	DeliveryResumed           DeliveryReason = "resumed"
	DeliveryRewound           DeliveryReason = "rewound"
	DeliveryBufferCreated     DeliveryReason = "buffer-created"
	DeliveryBufferDestroyed   DeliveryReason = "buffer-destroyed"
	DeliveryMoved             DeliveryReason = "moved"
	DeliverySettingsChanged   DeliveryReason = "settings"
	DeliverySampleDestroyed   DeliveryReason = "sample-destroyed"
	DeliveryScopeDestroyed    DeliveryReason = "scope-destroyed"
	DeliveryScopeCreated      DeliveryReason = "scope-created"
	DeliveryClosed            DeliveryReason = "closed"
	DeliverySampleUnavailable DeliveryReason = "sample-unavailable"
	DeliveryDeviceUnavailable DeliveryReason = "device-unavailable"
	DeliveryBufferUnavailable DeliveryReason = "buffer-unavailable"
	DeliveryInvalidGroup      DeliveryReason = "invalid-group"
)

type DeliveryReceipt struct {
	Sequence uint64
	Reason   DeliveryReason
	Request
	Scope     ScopeID
	Voice     VoiceID
	Duplicate int
	Channel   int
	Phase     int64
	Retained  bool
}

type DeliveryCounters struct {
	BuffersCreated   uint64
	BuffersDestroyed uint64
	Admitted         uint64
	Refused          uint64
	Evicted          uint64
}

type DeliveryDuplicateSnapshot struct {
	Index      int
	Generation uint64
	Retained   bool
	Playing    bool
	Paused     bool
	Phase      int64
	Channel    int
	Request    Request
}

type DeliverySampleSnapshot struct {
	ID         SampleID
	Scope      ScopeID
	Selector   string
	Duplicates []DeliveryDuplicateSnapshot
}

type DeliveryChannelSnapshot struct {
	Index   int
	Voice   VoiceID
	Scope   ScopeID
	Request Request
	Playing bool
	Phase   int64
}

type DeliverySnapshot struct {
	Closed   bool
	Limits   Limits
	Scopes   []ScopeID
	Samples  []DeliverySampleSnapshot
	Channels []DeliveryChannelSnapshot
	Receipts []DeliveryReceipt
	Counters DeliveryCounters
}

type deliveryDuplicate struct {
	buffer     Buffer
	generation uint64
	channel    int
	active     bool
	paused     bool
	rewound    bool
	request    Request
}

type deliverySample struct {
	id         SampleID
	scope      ScopeID
	selector   string
	data       Sample
	duplicates []deliveryDuplicate
}

type deliveryScope struct {
	id      ScopeID
	samples []SampleID
}

type deliveryChannel struct {
	sample    *deliverySample
	duplicate int
}

type Service struct {
	mu            sync.Mutex
	limits        Limits
	factory       BufferFactory
	closed        bool
	nextScope     ScopeID
	nextSample    SampleID
	scopes        map[ScopeID]*deliveryScope
	scopeOrder    []ScopeID
	samples       map[SampleID]*deliverySample
	sampleOrder   []SampleID
	channels      []deliveryChannel
	settings      [ChannelCount]Settings
	receipts      []DeliveryReceipt
	receiptNext   int
	receiptCount  int
	receiptSerial uint64
	counters      DeliveryCounters
}

type DeliveryVoice struct {
	service *Service
	id      VoiceID
}

func NewDelivery(limits Limits, factory BufferFactory) *Service {
	if limits.Duplicates <= 0 {
		limits.Duplicates = 16
	}
	if limits.Channels <= 0 {
		limits.Channels = 16
	}
	if limits.Receipts <= 0 {
		limits.Receipts = 1024
	}
	s := &Service{
		limits:   limits,
		factory:  factory,
		scopes:   make(map[ScopeID]*deliveryScope),
		samples:  make(map[SampleID]*deliverySample),
		channels: make([]deliveryChannel, limits.Channels),
		receipts: make([]DeliveryReceipt, limits.Receipts),
	}
	for i := range s.settings {
		s.settings[i] = DefaultSettings
	}
	return s
}

func (s *Service) NewScope() ScopeID {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0
	}
	s.nextScope++
	id := s.nextScope
	s.scopes[id] = &deliveryScope{id: id}
	s.scopeOrder = append(s.scopeOrder, id)
	s.record(DeliveryReceipt{Reason: DeliveryScopeCreated, Scope: id, Duplicate: -1, Channel: -1})
	return id
}

func (s *Service) RegisterSample(scope ScopeID, selector string, sample Sample) SampleID {
	s.mu.Lock()
	defer s.mu.Unlock()
	owner := s.scopes[scope]
	if s.closed || owner == nil || sample.Rate <= 0 || len(sample.PCM) == 0 {
		return 0
	}
	s.nextSample++
	id := s.nextSample
	object := &deliverySample{id: id, scope: scope, selector: selector, data: sample, duplicates: make([]deliveryDuplicate, s.limits.Duplicates)}
	for i := range object.duplicates {
		object.duplicates[i].channel = -1
	}
	s.samples[id] = object
	s.sampleOrder = append(s.sampleOrder, id)
	owner.samples = append(owner.samples, id)
	return id
}

func (s *Service) Request(req Request) *DeliveryVoice {
	s.mu.Lock()
	defer s.mu.Unlock()
	object := s.samples[req.Sample]
	if object != nil && req.Selector == "" {
		req.Selector = object.selector
	}
	if s.closed {
		return s.refuse(req, object, -1, DeliveryClosed)
	}
	if req.Group == MusicChannel || req.Group >= ChannelCount {
		return s.refuse(req, object, -1, DeliveryInvalidGroup)
	}
	if object == nil {
		return s.refuse(req, nil, -1, DeliverySampleUnavailable)
	}
	if req.Scope == 0 {
		req.Scope = object.scope
	}
	if s.scopes[req.Scope] == nil {
		return s.refuse(req, object, -1, DeliverySampleUnavailable)
	}
	if s.factory == nil {
		return s.refuse(req, object, -1, DeliveryDeviceUnavailable)
	}
	duplicate := -1
	for i := range object.duplicates {
		d := &object.duplicates[i]
		if d.buffer == nil || !d.paused && !d.buffer.Playing() {
			duplicate = i
			break
		}
	}
	if duplicate < 0 {
		return s.refuse(req, object, -1, DeliverySampleBusy)
	}
	d := &object.duplicates[duplicate]
	retained := d.buffer != nil
	if d.buffer == nil {
		d.buffer = s.factory(object.data, req.Repeat, req.Placement)
		if d.buffer == nil {
			return s.refuse(req, object, duplicate, DeliveryBufferUnavailable)
		}
		d.request = req
		s.counters.BuffersCreated++
		s.recordDuplicate(DeliveryBufferCreated, object, duplicate, -1)
	}
	s.normalize()
	channel := s.chooseChannel(req.Priority)
	if channel < 0 {
		return s.refuse(req, object, duplicate, DeliveryChannelsBusy)
	}
	if victim := s.channels[channel]; victim.sample != nil {
		s.stopDuplicate(victim.sample, victim.duplicate, true, DeliveryEvicted)
		s.counters.Evicted++
	}
	d.generation++
	d.channel, d.active, d.paused, d.rewound, d.request = channel, true, false, false, req
	s.channels[channel] = deliveryChannel{sample: object, duplicate: duplicate}
	d.buffer.SetRepeat(req.Repeat)
	d.buffer.Move(req.Placement)
	if tagged, ok := d.buffer.(interface{ SetGroup(Channel) }); ok {
		tagged.SetGroup(req.Group)
	}
	d.buffer.SetSettings(s.settings[req.Group])
	if retained {
		d.buffer.Rewind()
		s.recordDuplicate(DeliveryRewound, object, duplicate, channel)
	}
	d.buffer.Play()
	s.counters.Admitted++
	s.recordDuplicate(DeliveryAdmitted, object, duplicate, channel)
	return &DeliveryVoice{service: s, id: s.voiceID(object, duplicate)}
}

func (s *Service) refuse(req Request, object *deliverySample, duplicate int, reason DeliveryReason) *DeliveryVoice {
	r := DeliveryReceipt{Reason: reason, Request: req, Duplicate: duplicate, Channel: -1}
	if object != nil {
		r.Scope = req.Scope
		if duplicate >= 0 {
			r.Voice = s.voiceID(object, duplicate)
			if b := object.duplicates[duplicate].buffer; b != nil {
				r.Retained, r.Phase = true, b.Phase()
			}
		}
	}
	s.counters.Refused++
	s.record(r)
	return nil
}

func (s *Service) chooseChannel(priority uint8) int {
	for i, ch := range s.channels {
		if ch.sample == nil {
			return i
		}
	}
	chosen, lowest := -1, priority
	for i, ch := range s.channels {
		p := ch.sample.duplicates[ch.duplicate].request.Priority
		if p < lowest {
			chosen, lowest = i, p
		}
	}
	return chosen
}

func (s *Service) normalize() {
	for i, ch := range s.channels {
		if ch.sample == nil {
			continue
		}
		d := &ch.sample.duplicates[ch.duplicate]
		if d.buffer != nil && d.buffer.Playing() {
			continue
		}
		d.active, d.channel = false, -1
		s.recordDuplicate(DeliveryCompleted, ch.sample, ch.duplicate, i)
		s.channels[i] = deliveryChannel{}
	}
}

func (s *Service) stopDuplicate(object *deliverySample, duplicate int, rewind bool, reason DeliveryReason) {
	d := &object.duplicates[duplicate]
	d.paused = false
	if d.buffer == nil {
		return
	}
	channel := d.channel
	if d.active {
		d.buffer.Stop()
		d.active, d.channel = false, -1
		if channel >= 0 {
			s.channels[channel] = deliveryChannel{}
		}
		if !rewind {
			s.recordDuplicate(reason, object, duplicate, channel)
		}
	}
	if rewind && !d.rewound {
		d.buffer.Rewind()
		d.rewound = true
		s.recordDuplicate(reason, object, duplicate, channel)
	}
}

func (s *Service) DestroySample(id SampleID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destroySample(id)
}

func (s *Service) destroySample(id SampleID) {
	object := s.samples[id]
	if object == nil {
		return
	}
	for i := range object.duplicates {
		s.destroyDuplicate(object, i)
	}
	for i, ch := range s.channels {
		if ch.sample == object {
			s.channels[i] = deliveryChannel{}
		}
	}
	delete(s.samples, id)
	s.sampleOrder = removeDeliveryID(s.sampleOrder, id)
	if owner := s.scopes[object.scope]; owner != nil {
		owner.samples = removeDeliveryID(owner.samples, id)
	}
	object.data = Sample{}
	s.record(DeliveryReceipt{Reason: DeliverySampleDestroyed, Request: Request{Sample: id, Selector: object.selector}, Scope: object.scope, Duplicate: -1, Channel: -1})
}

func (s *Service) destroyDuplicate(object *deliverySample, index int) {
	d := &object.duplicates[index]
	if d.buffer == nil {
		return
	}
	channel := d.channel
	if d.buffer.Playing() {
		d.buffer.Stop()
		s.recordDuplicate(DeliveryStopped, object, index, channel)
	}
	phase := d.buffer.Phase()
	d.buffer.Destroy()
	s.counters.BuffersDestroyed++
	s.record(DeliveryReceipt{Reason: DeliveryBufferDestroyed, Request: d.request, Scope: d.request.Scope,
		Voice: s.voiceID(object, index), Duplicate: index, Channel: channel, Phase: phase})
	if channel >= 0 {
		s.channels[channel] = deliveryChannel{}
	}
	d.buffer, d.active, d.paused, d.channel, d.request = nil, false, false, -1, Request{}
}

func (s *Service) DestroyScope(id ScopeID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.destroyScope(id)
}

func (s *Service) destroyScope(id ScopeID) {
	owner := s.scopes[id]
	if owner == nil {
		return
	}
	for _, sampleID := range s.sampleOrder {
		object := s.samples[sampleID]
		for i := range object.duplicates {
			if object.duplicates[i].request.Scope == id {
				s.destroyDuplicate(object, i)
			}
		}
	}
	for len(owner.samples) > 0 {
		s.destroySample(owner.samples[0])
	}
	delete(s.scopes, id)
	s.scopeOrder = removeDeliveryID(s.scopeOrder, id)
	s.record(DeliveryReceipt{Reason: DeliveryScopeDestroyed, Scope: id, Duplicate: -1, Channel: -1})
}

func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	for len(s.scopeOrder) > 0 {
		s.destroyScope(s.scopeOrder[0])
	}
	s.record(DeliveryReceipt{Reason: DeliveryClosed, Duplicate: -1, Channel: -1})
}

func (s *Service) SetSettings(group Channel, settings Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || group == MusicChannel || group >= ChannelCount {
		return
	}
	s.settings[group] = settings
	for _, id := range s.sampleOrder {
		object := s.samples[id]
		for i := range object.duplicates {
			d := &object.duplicates[i]
			if d.buffer != nil && d.request.Group == group {
				d.buffer.SetSettings(settings)
				s.recordDuplicate(DeliverySettingsChanged, object, i, d.channel)
			}
		}
	}
}

func (s *Service) Snapshot() DeliverySnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.normalize()
	out := DeliverySnapshot{Closed: s.closed, Limits: s.limits, Scopes: append([]ScopeID(nil), s.scopeOrder...), Counters: s.counters}
	out.Channels = make([]DeliveryChannelSnapshot, len(s.channels))
	for i, ch := range s.channels {
		dst := &out.Channels[i]
		dst.Index = i
		if ch.sample != nil {
			d := &ch.sample.duplicates[ch.duplicate]
			dst.Voice, dst.Scope, dst.Request = s.voiceID(ch.sample, ch.duplicate), d.request.Scope, d.request
			dst.Playing, dst.Phase = d.buffer.Playing(), d.buffer.Phase()
		}
	}
	for _, id := range s.sampleOrder {
		object := s.samples[id]
		dst := DeliverySampleSnapshot{ID: id, Scope: object.scope, Selector: object.selector, Duplicates: make([]DeliveryDuplicateSnapshot, len(object.duplicates))}
		for i, d := range object.duplicates {
			dup := &dst.Duplicates[i]
			dup.Index, dup.Generation, dup.Channel, dup.Request = i, d.generation, d.channel, d.request
			dup.Paused = d.paused
			if d.buffer != nil {
				dup.Retained, dup.Playing, dup.Phase = true, d.buffer.Playing(), d.buffer.Phase()
			}
		}
		out.Samples = append(out.Samples, dst)
	}
	out.Receipts = make([]DeliveryReceipt, s.receiptCount)
	start := (s.receiptNext - s.receiptCount + len(s.receipts)) % len(s.receipts)
	for i := range out.Receipts {
		out.Receipts[i] = s.receipts[(start+i)%len(s.receipts)]
	}
	return out
}

func (s *Service) recordDuplicate(reason DeliveryReason, object *deliverySample, duplicate, channel int) {
	d := &object.duplicates[duplicate]
	r := DeliveryReceipt{Reason: reason, Request: d.request, Scope: d.request.Scope, Voice: s.voiceID(object, duplicate), Duplicate: duplicate, Channel: channel}
	if d.buffer != nil {
		r.Retained, r.Phase = true, d.buffer.Phase()
	}
	s.record(r)
}

func (s *Service) record(receipt DeliveryReceipt) {
	s.receiptSerial++
	receipt.Sequence = s.receiptSerial
	s.receipts[s.receiptNext] = receipt
	s.receiptNext = (s.receiptNext + 1) % len(s.receipts)
	if s.receiptCount < len(s.receipts) {
		s.receiptCount++
	}
}

func (s *Service) voiceID(object *deliverySample, duplicate int) VoiceID {
	return VoiceID{Sample: object.id, Duplicate: duplicate, Generation: object.duplicates[duplicate].generation}
}

func (v *DeliveryVoice) ID() VoiceID {
	if v == nil {
		return VoiceID{}
	}
	return v.id
}

func (v *DeliveryVoice) duplicate() (*deliverySample, *deliveryDuplicate) {
	object := v.service.samples[v.id.Sample]
	if object == nil || v.id.Duplicate < 0 || v.id.Duplicate >= len(object.duplicates) {
		return nil, nil
	}
	d := &object.duplicates[v.id.Duplicate]
	if d.generation != v.id.Generation || d.buffer == nil {
		return nil, nil
	}
	return object, d
}

func (v *DeliveryVoice) Playing() bool {
	if v == nil || v.service == nil {
		return false
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	v.service.normalize()
	_, d := v.duplicate()
	return d != nil && d.active && d.buffer.Playing()
}

func (v *DeliveryVoice) Stop() {
	if v == nil || v.service == nil {
		return
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	if object, d := v.duplicate(); d != nil {
		v.service.stopDuplicate(object, v.id.Duplicate, false, DeliveryStopped)
	}
}

func (v *DeliveryVoice) Pause() bool {
	if v == nil || v.service == nil {
		return false
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	object, d := v.duplicate()
	if d == nil {
		return false
	}
	if d.paused {
		return true
	}
	if !d.active || !d.buffer.Playing() {
		return false
	}
	v.service.stopDuplicate(object, v.id.Duplicate, false, DeliveryStopped)
	d.paused = true
	return true
}

func (v *DeliveryVoice) Resume() bool {
	if v == nil || v.service == nil {
		return false
	}
	s := v.service
	s.mu.Lock()
	defer s.mu.Unlock()
	object, d := v.duplicate()
	if s.closed || d == nil {
		return false
	}
	if !d.paused {
		return d.active && d.buffer.Playing()
	}
	s.normalize()
	channel := s.chooseChannel(d.request.Priority)
	if channel < 0 {
		s.refuse(d.request, object, v.id.Duplicate, DeliveryChannelsBusy)
		return false
	}
	if victim := s.channels[channel]; victim.sample != nil {
		s.stopDuplicate(victim.sample, victim.duplicate, true, DeliveryEvicted)
		s.counters.Evicted++
	}
	d.channel, d.active, d.paused = channel, true, false
	s.channels[channel] = deliveryChannel{sample: object, duplicate: v.id.Duplicate}
	d.buffer.Play()
	s.recordDuplicate(DeliveryResumed, object, v.id.Duplicate, channel)
	return true
}

func (v *DeliveryVoice) StopReset() {
	if v == nil || v.service == nil {
		return
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	if object, d := v.duplicate(); d != nil {
		v.service.stopDuplicate(object, v.id.Duplicate, true, DeliveryRewound)
	}
}

func (v *DeliveryVoice) Move(placement Placement) {
	if v == nil || v.service == nil {
		return
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	if object, d := v.duplicate(); d != nil {
		d.request.Placement = placement
		d.buffer.Move(placement)
		v.service.recordDuplicate(DeliveryMoved, object, v.id.Duplicate, d.channel)
	}
}

func (v *DeliveryVoice) MoveRequest(request Request) {
	if v == nil || v.service == nil {
		return
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	if object, d := v.duplicate(); d != nil {
		d.request.Placement = request.Placement
		d.request.Attenuation, d.request.Pan = request.Attenuation, request.Pan
		d.request.Geometry, d.request.SourceFine = request.Geometry, request.SourceFine
		d.buffer.Move(request.Placement)
		v.service.recordDuplicate(DeliveryMoved, object, v.id.Duplicate, d.channel)
	}
}

func (v *DeliveryVoice) Phase() int64 {
	if v == nil || v.service == nil {
		return 0
	}
	v.service.mu.Lock()
	defer v.service.mu.Unlock()
	if _, d := v.duplicate(); d != nil {
		return d.buffer.Phase()
	}
	return 0
}

func removeDeliveryID[T ~uint64](ids []T, id T) []T {
	for i, candidate := range ids {
		if candidate == id {
			copy(ids[i:], ids[i+1:])
			ids[len(ids)-1] = 0
			return ids[:len(ids)-1]
		}
	}
	return ids
}
