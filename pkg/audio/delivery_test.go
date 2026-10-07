package audio

import (
	"reflect"
	"testing"
)

type deliveryBufferFake struct {
	playing   bool
	phase     int64
	placement Placement
	settings  Settings
	group     Channel
	repeat    bool
	plays     int
	stops     int
	rewinds   int
	destroys  int
}

func (b *deliveryBufferFake) Play()                  { b.plays++; b.playing = true }
func (b *deliveryBufferFake) Playing() bool          { return b.playing }
func (b *deliveryBufferFake) Stop()                  { b.stops++; b.playing = false }
func (b *deliveryBufferFake) Rewind()                { b.rewinds++; b.phase = 0 }
func (b *deliveryBufferFake) Move(p Placement)       { b.placement = p }
func (b *deliveryBufferFake) SetSettings(s Settings) { b.settings = s }
func (b *deliveryBufferFake) SetGroup(group Channel) { b.group = group }
func (b *deliveryBufferFake) SetRepeat(repeat bool)  { b.repeat = repeat }
func (b *deliveryBufferFake) Phase() int64           { return b.phase }
func (b *deliveryBufferFake) Destroy()               { b.destroys++; b.playing = false }

type deliveryRig struct {
	service *Service
	scope   ScopeID
	buffers []*deliveryBufferFake
}

func newDeliveryRig(limits Limits) *deliveryRig {
	r := &deliveryRig{}
	r.service = NewDelivery(limits, func(_ Sample, repeat bool, placement Placement) Buffer {
		b := &deliveryBufferFake{repeat: repeat, placement: placement}
		r.buffers = append(r.buffers, b)
		return b
	})
	r.scope = r.service.NewScope()
	return r
}

func (r *deliveryRig) sample(selector string) SampleID {
	return r.service.RegisterSample(r.scope, selector, Sample{Rate: 22050, PCM: []int16{1, 2, 3}})
}

func deliveryRequest(sample SampleID, priority uint8) Request {
	return Request{Sample: sample, Priority: priority, Group: EffectsChannel, Placement: Placement{GainUnit, GainUnit}}
}

func deliveryLastReceipt(t *testing.T, s *Service) DeliveryReceipt {
	t.Helper()
	got := s.Snapshot().Receipts
	if len(got) == 0 {
		t.Fatal("no delivery receipt")
	}
	return got[len(got)-1]
}

func TestDeliverySampleBeforeGlobalCompetition(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 2})
	defer r.service.Close()
	a, b := r.sample("a"), r.sample("b")
	victim := r.service.Request(deliveryRequest(a, 1))
	other := r.service.Request(deliveryRequest(b, 2))
	if victim == nil || other == nil {
		t.Fatal("initial requests refused")
	}
	if got := r.service.Request(deliveryRequest(a, 255)); got != nil {
		t.Fatal("full sample evicted a global voice")
	}
	if !victim.Playing() || !other.Playing() || r.buffers[0].stops != 0 || r.buffers[1].stops != 0 {
		t.Fatal("sample refusal changed global playback")
	}
	if got := deliveryLastReceipt(t, r.service).Reason; got != DeliverySampleBusy {
		t.Fatal(got)
	}
}

func TestDeliveryEffectsAndSpeechShareDefaultChannels(t *testing.T) {
	r := newDeliveryRig(Limits{})
	defer r.service.Close()
	voices := make([]*DeliveryVoice, 16)
	for i := range voices {
		req := deliveryRequest(r.sample("shared"), 128)
		if i%2 == 1 {
			req.Group = SpeechChannel
		}
		voices[i] = r.service.Request(req)
		if voices[i] == nil {
			t.Fatal("channel refused", i)
		}
	}
	if got := r.service.Request(deliveryRequest(r.sample("overflow"), 128)); got != nil {
		t.Fatal("effects and speech received separate pools")
	}
	for i, v := range voices {
		if !v.Playing() {
			t.Fatal("equal priority evicted", i)
		}
	}
	got := r.service.Snapshot()
	if got.Limits.Duplicates != 16 || got.Limits.Channels != 16 || got.Limits.Receipts != 1024 || len(got.Channels) != 16 {
		t.Fatal(got.Limits, len(got.Channels))
	}
	if got.Counters.Admitted != 16 || got.Counters.Refused != 1 || got.Counters.Evicted != 0 {
		t.Fatal(got.Counters)
	}
}

func TestDeliveryFirstIdleThenFirstLowestStrictPriority(t *testing.T) {
	r := newDeliveryRig(Limits{Channels: 3})
	defer r.service.Close()
	samples := []SampleID{r.sample("a"), r.sample("b"), r.sample("c"), r.sample("d")}
	voices := make([]*DeliveryVoice, 3)
	for i, priority := range []uint8{2, 1, 1} {
		voices[i] = r.service.Request(deliveryRequest(samples[i], priority))
		if got := deliveryLastReceipt(t, r.service).Channel; got != i {
			t.Fatal("first idle", got, i)
		}
	}
	if r.service.Request(deliveryRequest(samples[3], 1)) != nil {
		t.Fatal("equal priority evicted")
	}
	if r.service.Request(deliveryRequest(samples[3], 2)) == nil {
		t.Fatal("higher priority refused")
	}
	if got := deliveryLastReceipt(t, r.service).Channel; got != 1 {
		t.Fatal("tie did not select first minimum", got)
	}
	if !voices[0].Playing() || voices[1].Playing() || !voices[2].Playing() {
		t.Fatal("wrong priority victim")
	}
	voices[0].Stop()
	if r.service.Request(deliveryRequest(samples[0], 0)) == nil {
		t.Fatal("idle channel refused priority zero")
	}
	if got := deliveryLastReceipt(t, r.service).Channel; got != 0 {
		t.Fatal("idle channel order", got)
	}
}

func TestDeliveryEvictionRetainsDuplicateAndRewinds(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	defer r.service.Close()
	a, b := r.sample("a"), r.sample("b")
	old := r.service.Request(deliveryRequest(a, 1))
	r.buffers[0].phase = 19
	if r.service.Request(deliveryRequest(b, 2)) == nil {
		t.Fatal("eviction refused")
	}
	if old.Playing() || r.buffers[0].stops != 1 || r.buffers[0].rewinds != 1 || r.buffers[0].destroys != 0 || r.buffers[0].phase != 0 {
		t.Fatal("eviction released or failed to rewind", r.buffers[0])
	}
	if r.service.Request(deliveryRequest(a, 3)) == nil || len(r.buffers) != 2 || r.buffers[0].plays != 2 {
		t.Fatal("evicted duplicate was not reused")
	}
	if got := r.service.Snapshot().Counters; got.BuffersCreated != 2 || got.BuffersDestroyed != 0 || got.Evicted != 2 {
		t.Fatal(got)
	}
}

func TestDeliveryCompletionRetainsAndInvalidatesOldGeneration(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	defer r.service.Close()
	id := r.sample("complete")
	old := r.service.Request(deliveryRequest(id, 5))
	r.buffers[0].phase, r.buffers[0].playing = 3, false
	got := r.service.Snapshot()
	if got.Channels[0].Voice.Sample != 0 || !got.Samples[0].Duplicates[0].Retained || got.Samples[0].Duplicates[0].Playing {
		t.Fatal("completion did not detach retained duplicate", got)
	}
	if deliveryLastReceipt(t, r.service).Reason != DeliveryCompleted || old.Playing() {
		t.Fatal("completion was not observed")
	}
	req := deliveryRequest(id, 5)
	req.Repeat = true
	current := r.service.Request(req)
	if current == nil || current.ID().Generation <= old.ID().Generation || len(r.buffers) != 1 || !r.buffers[0].repeat {
		t.Fatal("reused duplicate has stale generation or repeat mode")
	}
	stops, rewinds := r.buffers[0].stops, r.buffers[0].rewinds
	old.Stop()
	old.StopReset()
	old.Move(Placement{})
	if !current.Playing() || r.buffers[0].stops != stops || r.buffers[0].rewinds != rewinds || r.buffers[0].placement != req.Placement || old.Phase() != 0 {
		t.Fatal("stale handle controlled reused duplicate")
	}
}

func TestDeliveryStaleEvictedHandleCannotStopNewUse(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	defer r.service.Close()
	a, b := r.sample("a"), r.sample("b")
	old := r.service.Request(deliveryRequest(a, 1))
	r.service.Request(deliveryRequest(b, 2))
	current := r.service.Request(deliveryRequest(a, 3))
	stops, rewinds := r.buffers[0].stops, r.buffers[0].rewinds
	old.StopReset()
	if current == nil || !current.Playing() || r.buffers[0].stops != stops || r.buffers[0].rewinds != rewinds {
		t.Fatal("evicted handle controlled reused duplicate")
	}
}

func TestDeliveryStopAndStopResetKeepSeparatePhase(t *testing.T) {
	r := newDeliveryRig(Limits{})
	defer r.service.Close()
	v := r.service.Request(deliveryRequest(r.sample("stop"), 1))
	r.buffers[0].phase = 7
	v.Stop()
	v.Stop()
	if v.Playing() || v.Phase() != 7 || r.buffers[0].stops != 1 || r.buffers[0].rewinds != 0 || r.buffers[0].destroys != 0 {
		t.Fatal("stop rewound or released duplicate")
	}
	v.StopReset()
	v.StopReset()
	if v.Phase() != 0 || r.buffers[0].rewinds != 1 || r.buffers[0].destroys != 0 {
		t.Fatal("stop-reset did not rewind once")
	}
}

func TestDeliveryPauseReservesDuplicateAndResumesExactHandle(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 2, Channels: 2})
	defer r.service.Close()
	id := r.sample("speech")
	v := r.service.Request(deliveryRequest(id, 128))
	first := r.buffers[0]
	first.phase = 374
	before := v.ID()
	if !v.Pause() || !v.Pause() || v.Playing() || v.Phase() != 374 {
		t.Fatal("pause changed phase or failed to hold voice")
	}
	other := r.service.Request(deliveryRequest(id, 128))
	if other == nil || other.ID().Duplicate == before.Duplicate || v.ID() != before {
		t.Fatal("same sample request stole paused duplicate")
	}
	if !v.Resume() || !v.Resume() || !v.Playing() || v.Phase() != 374 || v.ID() != before {
		t.Fatal("resume changed voice identity or phase")
	}
	if first.plays != 2 || first.stops != 1 || first.rewinds != 0 || first.destroys != 0 || len(r.buffers) != 2 {
		t.Fatal("pause/resume duplicated, rewound or destroyed the buffer", first)
	}
	if last := deliveryLastReceipt(t, r.service); last.Reason != DeliveryResumed || last.Voice != before || last.Phase != 374 {
		t.Fatal("missing ordered retained resume receipt", last)
	}
}

func TestDeliveryPausedResumeRefusesWithoutQueueOrPhaseLoss(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	defer r.service.Close()
	id := r.sample("speech")
	v := r.service.Request(deliveryRequest(id, 128))
	r.buffers[0].phase = 374
	if !v.Pause() {
		t.Fatal("voice did not pause")
	}
	blocker := r.service.Request(deliveryRequest(r.sample("blocker"), 128))
	if blocker == nil || v.Resume() || !blocker.Playing() || v.Phase() != 374 || r.buffers[0].plays != 1 {
		t.Fatal("equal priority resume changed channel or phase")
	}
	if other := r.service.Request(deliveryRequest(id, 255)); other != nil || !blocker.Playing() {
		t.Fatal("paused sample bypassed its reserved duplicate")
	}
	blocker.Stop()
	if v.Playing() || r.buffers[0].plays != 1 {
		t.Fatal("free channel automatically resumed a refused voice")
	}
	if !v.Resume() || v.Phase() != 374 || r.buffers[0].rewinds != 0 || r.buffers[0].plays != 2 {
		t.Fatal("caller resume did not retain buffer at phase")
	}
	v.Pause()
	v.StopReset()
	if v.Resume() || v.Phase() != 0 || r.buffers[0].rewinds != 1 {
		t.Fatal("stop-reset retained temporary pause ownership")
	}
}

func TestDeliveryCompletionAndDisposalCannotResume(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	id := r.sample("speech")
	v := r.service.Request(deliveryRequest(id, 128))
	r.buffers[0].playing = false
	if v.Pause() || v.Resume() {
		t.Fatal("naturally completed voice was treated as paused")
	}
	v = r.service.Request(deliveryRequest(id, 128))
	v.Pause()
	r.service.DestroyScope(r.scope)
	r.service.Close()
	r.service.Close()
	if v.Resume() || v.Phase() != 0 || r.buffers[0].destroys != 1 {
		t.Fatal("disposed paused voice resumed or closed twice")
	}
}

func TestDeliveryExplicitReadmissionRewindsRetainedPhase(t *testing.T) {
	for _, completed := range []bool{false, true} {
		t.Run(map[bool]string{false: "stopped", true: "completed"}[completed], func(t *testing.T) {
			r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
			defer r.service.Close()
			id := r.sample("retained")
			old := r.service.Request(deliveryRequest(id, 128))
			r.buffers[0].phase = 3
			if completed {
				r.buffers[0].playing = false
				r.service.Snapshot()
			} else {
				old.Stop()
			}
			if old.Phase() != 3 || r.buffers[0].rewinds != 0 {
				t.Fatal("terminal observation changed retained phase")
			}
			current := r.service.Request(deliveryRequest(id, 128))
			if current == nil || current.Phase() != 0 || r.buffers[0].rewinds != 1 || r.buffers[0].plays != 2 || len(r.buffers) != 1 {
				t.Fatal("explicit request resumed at retained EOF or phase", r.buffers[0])
			}
		})
	}
}

func TestDeliveryDestroySampleReleasesAllDuplicatesAndDetachesAllSlots(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 3, Channels: 4})
	defer r.service.Close()
	id := r.sample("many")
	voices := make([]*DeliveryVoice, 3)
	for i := range voices {
		voices[i] = r.service.Request(deliveryRequest(id, 1))
	}
	voices[1].Stop()
	r.service.DestroySample(id)
	r.service.DestroySample(id)
	for i, b := range r.buffers {
		if b.destroys != 1 || b.stops != 1 || b.rewinds != 0 || voices[i].Playing() || voices[i].Phase() != 0 {
			t.Fatal("sample did not release all duplicates", i, b)
		}
	}
	got := r.service.Snapshot()
	if len(got.Samples) != 0 || got.Counters.BuffersDestroyed != 3 {
		t.Fatal(got)
	}
	for _, ch := range got.Channels {
		if ch.Voice.Sample != 0 || ch.Playing || ch.Request.Sample != 0 {
			t.Fatal("matching channel remains attached", ch)
		}
	}
	if r.service.Request(deliveryRequest(id, 255)) != nil {
		t.Fatal("destroyed sample admitted")
	}
}

func TestDeliveryScopeOwnershipAndCloseReleaseOnce(t *testing.T) {
	r := newDeliveryRig(Limits{})
	scope2 := r.service.NewScope()
	a, b := r.sample("same"), r.sample("same")
	c := r.service.RegisterSample(scope2, "same", Sample{Rate: 22050, PCM: []int16{4}})
	if a == b || a == c || b == c {
		t.Fatal("registration merged distinct sample objects")
	}
	va := r.service.Request(deliveryRequest(a, 1))
	vb := r.service.Request(deliveryRequest(b, 1))
	vc := r.service.Request(deliveryRequest(c, 1))
	r.service.DestroyScope(r.scope)
	if va.Playing() || vb.Playing() || !vc.Playing() || r.buffers[0].destroys != 1 || r.buffers[1].destroys != 1 || r.buffers[2].destroys != 0 {
		t.Fatal("scope destroyed another owner's sample")
	}
	if got := r.service.Snapshot(); !reflect.DeepEqual(got.Scopes, []ScopeID{scope2}) || len(got.Samples) != 1 || got.Samples[0].ID != c {
		t.Fatal("scope snapshot", got)
	}
	r.service.Close()
	r.service.Close()
	for _, b := range r.buffers {
		if b.destroys != 1 {
			t.Fatal("close released twice", b)
		}
	}
	if vc.Playing() || !r.service.Snapshot().Closed || r.service.NewScope() != 0 || r.service.RegisterSample(scope2, "new", Sample{Rate: 1, PCM: []int16{1}}) != 0 || r.service.Request(deliveryRequest(c, 255)) != nil {
		t.Fatal("closed service accepted work")
	}
}

func TestDeliveryMoveSettingsAndLoopPhaseDoNotReadmit(t *testing.T) {
	r := newDeliveryRig(Limits{})
	defer r.service.Close()
	id := r.sample("loop")
	req := deliveryRequest(id, 220)
	req.Repeat = true
	v := r.service.Request(req)
	r.buffers[0].phase = 23
	before := r.service.Snapshot().Counters
	p := Placement{1234, 4321}
	v.Move(p)
	r.service.SetSettings(EffectsChannel, Settings{Master: 25, Muted: true})
	if v.Phase() != 23 || !v.Playing() || r.buffers[0].plays != 1 || r.buffers[0].placement != p || r.buffers[0].group != EffectsChannel || r.buffers[0].settings != (Settings{Master: 25, Muted: true}) {
		t.Fatal("live adjustment restarted or dropped playback", r.buffers[0])
	}
	if got := r.service.Snapshot(); got.Counters != before || got.Channels[0].Request.Placement != p || got.Channels[0].Phase != 23 || !got.Channels[0].Request.Repeat {
		t.Fatal("adjustment re-admitted or lost loop state", got)
	}
}

func TestDeliverySnapshotsAreDetachedAndReceiptsAreBoundedOrdered(t *testing.T) {
	r := newDeliveryRig(Limits{Channels: 1, Receipts: 4})
	defer r.service.Close()
	id := r.sample("selector")
	req := deliveryRequest(id, 128)
	req.Source, req.Recipe = "semantic", "recipe"
	req.Attenuation, req.Pan, req.Frequency, req.SpatialKnown = -17, 9, 0, false
	v := r.service.Request(req)
	for i := 0; i < 10; i++ {
		v.Move(Placement{i, i + 1})
	}
	got := r.service.Snapshot()
	if len(got.Receipts) != 4 || len(got.Samples) != 1 || got.Samples[0].Scope != r.scope {
		t.Fatal(got)
	}
	for i, receipt := range got.Receipts {
		if receipt.Source != "semantic" || receipt.Recipe != "recipe" || receipt.Selector != "selector" || receipt.Sample != id || receipt.Scope != r.scope || receipt.Group != EffectsChannel || receipt.Priority != 128 || receipt.Attenuation != -17 || receipt.Pan != 9 || receipt.Frequency != 0 || receipt.SpatialKnown || receipt.Duplicate != 0 || receipt.Channel != 0 || !receipt.Retained {
			t.Fatal("incomplete request metadata", receipt)
		}
		if i > 0 && receipt.Sequence != got.Receipts[i-1].Sequence+1 {
			t.Fatal("receipt sequence unordered", got.Receipts)
		}
	}
	want := r.service.Snapshot()
	got.Scopes[0] = 99
	got.Samples[0].Duplicates[0].Retained = false
	got.Samples[0].Selector = "changed"
	got.Channels[0].Request.Source = "changed"
	got.Receipts[0].Request.Recipe = "changed"
	if after := r.service.Snapshot(); !reflect.DeepEqual(after, want) {
		t.Fatal("snapshot aliases service storage", after)
	}
}

func TestDeliveryRefusalHasNoQueueOrAutomaticLoopRestart(t *testing.T) {
	r := newDeliveryRig(Limits{Duplicates: 1, Channels: 1})
	defer r.service.Close()
	a, b := r.sample("loop"), r.sample("equal")
	loop := deliveryRequest(a, 128)
	loop.Repeat = true
	v := r.service.Request(loop)
	if r.service.Request(deliveryRequest(b, 128)) != nil {
		t.Fatal("equal request admitted")
	}
	v.StopReset()
	r.service.Snapshot()
	if r.buffers[1].plays != 0 || r.buffers[0].plays != 1 {
		t.Fatal("refused request queued or loop restarted")
	}
	if r.service.Request(deliveryRequest(b, 128)) == nil || r.buffers[1].plays != 1 {
		t.Fatal("explicit later request did not use retained buffer")
	}
}

func TestDeliveryRefusesMusicMissingDeviceAndMissingBuffer(t *testing.T) {
	r := newDeliveryRig(Limits{})
	defer r.service.Close()
	req := deliveryRequest(r.sample("music"), 255)
	req.Group = MusicChannel
	if r.service.Request(req) != nil || deliveryLastReceipt(t, r.service).Reason != DeliveryInvalidGroup || len(r.buffers) != 0 {
		t.Fatal("music entered shared SFX service")
	}
	for _, tc := range []struct {
		name    string
		factory BufferFactory
		reason  DeliveryReason
	}{
		{"device", nil, DeliveryDeviceUnavailable},
		{"buffer", func(Sample, bool, Placement) Buffer { return nil }, DeliveryBufferUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewDelivery(Limits{}, tc.factory)
			defer s.Close()
			id := s.RegisterSample(s.NewScope(), "sample", Sample{Rate: 1, PCM: []int16{1}})
			if s.Request(deliveryRequest(id, 255)) != nil || deliveryLastReceipt(t, s).Reason != tc.reason || s.Snapshot().Counters.Refused != 1 {
				t.Fatal("unavailable playback admitted")
			}
		})
	}
}
