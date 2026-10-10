package ui

import (
	"image"
	"math/rand"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
)

type ambientBank struct {
	samples map[int]audio.Sample
	reads   []int
}

func (b *ambientBank) Sample(slot int) (audio.Sample, bool) {
	b.reads = append(b.reads, slot)
	s, ok := b.samples[slot]
	return s, ok
}

type ambientOneShot struct {
	samples    []audio.Sample
	placements []audio.Placement
}

func (p *ambientOneShot) Play(sample audio.Sample, placement audio.Placement) {
	p.samples = append(p.samples, sample)
	p.placements = append(p.placements, placement)
}

func (p *ambientOneShot) RequestSample(sample audio.Sample, request audio.Request) audio.Voice {
	p.Play(sample, request.Placement)
	return nil
}

type ambientLoopEvent struct {
	kind      AmbientLoop
	placement audio.Placement
}

type ambientLoopRecorder struct {
	starts   []ambientLoopEvent
	moves    []ambientLoopEvent
	stops    []AmbientLoop
	stopAll  int
	settings []audio.Settings
	voices   [ambientLoopCount]*ambientRecordedVoice
}

type ambientRecordedVoice struct{ playing bool }

func (v *ambientRecordedVoice) Playing() bool { return v.playing }
func (v *ambientRecordedVoice) Stop()         { v.playing = false }

func (r *ambientLoopRecorder) RequestLoop(kind AmbientLoop, sample audio.Sample, request audio.Request) audio.Voice {
	r.StartLoop(kind, sample, request.Placement)
	v := &ambientRecordedVoice{playing: true}
	r.voices[kind] = v
	return v
}

func (r *ambientLoopRecorder) StartLoop(kind AmbientLoop, _ audio.Sample, placement audio.Placement) {
	r.starts = append(r.starts, ambientLoopEvent{kind: kind, placement: placement})
}
func (r *ambientLoopRecorder) MoveLoop(kind AmbientLoop, placement audio.Placement) {
	r.moves = append(r.moves, ambientLoopEvent{kind: kind, placement: placement})
}
func (r *ambientLoopRecorder) StopLoop(kind AmbientLoop) {
	r.stops = append(r.stops, kind)
	if v := r.voices[kind]; v != nil {
		v.Stop()
	}
}
func (r *ambientLoopRecorder) Stop() {
	r.stopAll++
	for _, v := range r.voices {
		if v != nil {
			v.Stop()
		}
	}
}
func (r *ambientLoopRecorder) SetSettings(st audio.Settings) {
	r.settings = append(r.settings, st)
}

func ambientSample(value int16) audio.Sample {
	return audio.Sample{Rate: audio.DeviceRate, PCM: []int16{value, -value}}
}

func TestAmbientSnapshotUsesInclusiveExpandedRecoveredScan(t *testing.T) {
	const width, height = 64, 48
	grid := terrain.Grid{Width: width, Height: height, Tiles: make([]uint16, width*height), Overlay: make([]uint8, width*height)}
	grid.Tiles[8*width+12] = 8 << 6
	grid.Tiles[16*width+20] = 11 << 6
	grid.Tiles[32*width+36] = 9 << 6
	grid.Tiles[18*width+22] = 7 << 6
	for _, cell := range []image.Point{image.Pt(11, 8), image.Pt(37, 32), image.Pt(20, 7), image.Pt(20, 33)} {
		grid.Tiles[cell.Y*width+cell.X] = 8 << 6
	}
	grid.Overlay[9*width+12] = 1
	grid.Overlay[31*width+36] = 2
	grid.Overlay[9*width+11] = 1
	grid.Overlay[31*width+37] = 2
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{FireObject: -2}
	set.Classes[2] = &terrain.StaticClass{FireObject: 0}

	v, err := NewViewerWithStatics("ambient", grid, &terrain.Tileset{}, set, false, false, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	v.cam.X, v.cam.Y, v.cam.Zoom = 20*32, 16*32, 1
	v.cam.ViewW, v.cam.ViewH = 8*32, 8*32
	v.SetAmbientWallFire([]image.Point{image.Pt(12, 10), image.Pt(12, 10), image.Pt(36, 30), image.Pt(11, 10), image.Pt(37, 30)})
	s := v.ambientSnapshot()
	if !s.hasView || s.geometry != (audio.ViewGeometry{Origin: image.Pt(20, 16), Span: image.Pt(8, 8)}) || s.listener != image.Pt(24, 20) {
		t.Fatal("the constructed viewer lost its explicit geometry or integer listener", s)
	}
	if got, want := s.river, []image.Point{image.Pt(12, 8), image.Pt(20, 16), image.Pt(36, 32)}; !pointsEqual(got, want) {
		t.Fatalf("river sources = %v, want %v", got, want)
	}
	if got, want := s.wallFire, []image.Point{image.Pt(12, 10), image.Pt(36, 30)}; !pointsEqual(got, want) {
		t.Fatalf("Wall of Fire sources = %v, want %v", got, want)
	}
	if got, want := s.crows, []image.Point{image.Pt(12, 9)}; !pointsEqual(got, want) {
		t.Fatalf("crow sources = %v, want %v", got, want)
	}
	if got, want := s.birds, []image.Point{image.Pt(36, 31)}; !pointsEqual(got, want) {
		t.Fatalf("bird sources = %v, want %v", got, want)
	}
	t.Logf("inclusive expanded scan=[12,8]-[36,32] river=%v fire=%v crow=%v bird=%v", s.river, s.wallFire, s.crows, s.birds)
}

func TestAmbientSnapshotClampsScanAtEightCellMargins(t *testing.T) {
	const width, height = 64, 48
	grid := terrain.Grid{Width: width, Height: height, Tiles: make([]uint16, width*height)}
	for _, cell := range []image.Point{image.Pt(8, 8), image.Pt(56, 40), image.Pt(7, 8), image.Pt(57, 40), image.Pt(8, 7), image.Pt(56, 41)} {
		grid.Tiles[cell.Y*width+cell.X] = 8 << 6
	}
	v, err := NewViewerWithStatics("ambient margins", grid, &terrain.Tileset{}, nil, false, false, false, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	v.cam.X, v.cam.Y, v.cam.Zoom = 8*32, 8*32, 1
	v.cam.ViewW, v.cam.ViewH = 32*32, 32*32
	s := v.ambientSnapshot()
	if !s.hasView || s.geometry != (audio.ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(32, 32)}) {
		t.Fatal("the margin witness lost its explicit geometry", s)
	}
	if want := []image.Point{image.Pt(8, 8), image.Pt(56, 40)}; !pointsEqual(s.river, want) {
		t.Fatalf("inclusive margin-clamped scan river=%v, want %v", s.river, want)
	}
	t.Logf("inclusive eight-cell margins=[8,8]-[56,40] river=%v", s.river)
}

func TestAmbientLoopsCoexistMoveWithoutRestartAndStopIndependently(t *testing.T) {
	bank := &ambientBank{samples: map[int]audio.Sample{
		ambientRiverSlot:    ambientSample(50),
		ambientWallFireSlot: ambientSample(90),
	}}
	device := &ambientLoopRecorder{}
	c := newAmbientController(bank, &ambientOneShot{}, device, random.NewStream(1071))
	now := time.Unix(10, 0)
	s := ambientSnapshot{
		geometry: audio.ViewGeometry{Span: image.Pt(10, 10)},
		listener: image.Pt(5, 5), hasView: true,
		river: []image.Point{image.Pt(3, 5)}, wallFire: []image.Point{image.Pt(7, 5)},
	}
	c.update(now, s)
	if len(device.starts) != 2 || device.starts[0].kind != AmbientRiver || device.starts[1].kind != AmbientWallOfFire {
		t.Fatalf("loop starts = %+v, want river and Wall of Fire", device.starts)
	}
	c.update(now.Add(time.Second), s)
	if len(device.starts) != 2 || len(device.moves) != 0 {
		t.Fatalf("unchanged sources restarted or moved loops: starts=%v moves=%v", device.starts, device.moves)
	}

	s.river = []image.Point{image.Pt(1, 5)}
	s.wallFire = []image.Point{image.Pt(9, 5)}
	s.geometry.Origin, s.listener = image.Pt(1, 0), image.Pt(6, 5)
	c.update(now.Add(2*time.Second), s)
	if len(device.moves) != 2 || device.moves[0].kind != AmbientRiver || device.moves[1].kind != AmbientWallOfFire {
		t.Fatalf("placement moves = %+v, want both loops once", device.moves)
	}
	if len(device.starts) != 2 {
		t.Fatalf("placement change restarted a loop: %v", device.starts)
	}

	s.river = nil
	s.geometry.Origin, s.listener = image.Pt(2, 0), image.Pt(7, 5)
	c.update(now.Add(3*time.Second), s)
	if len(device.stops) != 1 || device.stops[0] != AmbientRiver {
		t.Fatalf("source disappearance stops = %v, want river only", device.stops)
	}
	if !c.loop[AmbientWallOfFire].active {
		t.Fatal("stopping the river also stopped Wall of Fire")
	}
	c.stop()
	if device.stopAll != 1 {
		t.Fatalf("teardown stop count = %d, want 1", device.stopAll)
	}
}

func TestAmbientDeadlineUsesDecodedCadenceAndMissingSampleIsQuiet(t *testing.T) {
	const seed = int64(71)
	now := time.Unix(20, 0)
	bank := &ambientBank{samples: map[int]audio.Sample{ambientCrowSlot: ambientSample(70)}}
	ones := &ambientOneShot{}
	c := newAmbientController(bank, ones, &ambientLoopRecorder{}, random.NewStream(seed))
	c.next = now
	s := ambientSnapshot{geometry: audio.ViewGeometry{Span: image.Pt(20, 20)}, listener: image.Pt(10, 10), hasView: true, crows: []image.Point{image.Pt(12, 10)}}
	c.update(now, s)
	if len(ones.samples) != 0 || len(bank.reads) != 0 || !c.next.Equal(now) {
		t.Fatal("deadline equality played a source or changed the deadline", len(ones.samples), bank.reads, c.next)
	}
	due := now.Add(time.Millisecond)
	c.update(due, s)
	if len(ones.samples) != 1 || len(bank.reads) != 1 || bank.reads[0] != ambientCrowSlot {
		t.Fatalf("deadline play = samples %d reads %v, want one slot-70 play", len(ones.samples), bank.reads)
	}
	wantRNG := rand.New(rand.NewSource(seed))
	_ = wantRNG.Intn(0x8000) // the bird-or-crow draw of one crow
	wantNext := due.Add(time.Duration(10000+wantRNG.Intn(0x8000)/2) * time.Millisecond)
	if !c.next.Equal(wantNext) {
		t.Fatalf("next deadline = %v, want %v", c.next, wantNext)
	}

	// A due source advances the deadline even when its registered leaf is
	// absent. This is the production path slot 62 takes in both shipped roots.
	quiet := newAmbientController(&ambientBank{samples: map[int]audio.Sample{}}, ones, nil, random.NewStream(seed))
	quiet.next = now
	before := len(ones.samples)
	quiet.update(due, s)
	if len(ones.samples) != before || !quiet.next.After(due) {
		t.Fatalf("missing sample played or failed to reschedule: plays=%d next=%v", len(ones.samples)-before, quiet.next)
	}
}

func TestAmbientScanUsesOriginMovementOrStrictDeadline(t *testing.T) {
	bank := &ambientBank{samples: map[int]audio.Sample{
		ambientRiverSlot: ambientSample(50),
		ambientCrowSlot:  ambientSample(70),
	}}
	ones, device := &ambientOneShot{}, &ambientLoopRecorder{}
	c := newAmbientController(bank, ones, device, random.NewStream(71))
	now := time.Unix(20, 0)
	deadline := now.Add(time.Second)
	c.next = deadline
	s := ambientSnapshot{
		geometry: audio.ViewGeometry{Origin: image.Pt(8, 8), Span: image.Pt(15, 15)},
		listener: image.Pt(15, 15), hasView: true,
		river: []image.Point{image.Pt(22, 15)}, crows: []image.Point{image.Pt(15, 15)},
	}
	c.update(now, s)
	if len(device.starts) != 1 || len(ones.samples) != 0 || !c.next.Equal(deadline) || c.loop[AmbientRiver].request.Pan != 933 {
		t.Fatal("origin movement did not scan loops alone", device.starts, len(ones.samples), c.next, c.loop[AmbientRiver].request)
	}
	origin := c.origin
	s.geometry.Span, s.listener = image.Pt(16, 17), image.Pt(16, 16)
	for _, at := range []time.Time{now.Add(time.Millisecond), deadline} {
		c.update(at, s)
		if len(device.starts) != 1 || len(device.moves) != 0 || len(ones.samples) != 0 || !c.next.Equal(deadline) || c.origin != origin || c.loop[AmbientRiver].request.Pan != 933 {
			t.Fatal("extent-only change or deadline equality scanned, moved or rescheduled", at, device, len(ones.samples), c.next, c.origin, c.loop[AmbientRiver].request)
		}
	}
	s.geometry.Origin, s.listener = image.Pt(9, 8), image.Pt(17, 16)
	c.update(deadline, s)
	if len(device.starts) != 1 || len(device.moves) != 1 || len(ones.samples) != 0 || !c.next.Equal(deadline) || c.origin != s.geometry.Origin || c.loop[AmbientRiver].request.Pan != 625 {
		t.Fatal("movement at deadline equality did not update only the retained loop", device, len(ones.samples), c.next, c.origin, c.loop[AmbientRiver].request)
	}
	c.update(deadline.Add(time.Millisecond), s)
	if len(device.starts) != 1 || len(device.moves) != 1 || len(ones.samples) != 1 || !c.next.After(deadline) || c.origin != s.geometry.Origin {
		t.Fatal("strictly past deadline did not request a crow without restarting loops", device, len(ones.samples), c.next, c.origin)
	}
	next := c.next
	s.river, s.crows = nil, nil
	c.update(next.Add(time.Millisecond), s)
	if len(device.stops) != 1 || device.stops[0] != AmbientRiver || len(ones.samples) != 1 || !c.next.Equal(next) {
		t.Fatal("due empty populations did not stop their loop and retain the deadline", device.stops, len(ones.samples), c.next, next)
	}
	t.Logf("extent/equality unchanged; moved equality pan=625; strictly due crow calls=%d; empty due deadline=%v", len(ones.samples), c.next)
}

func TestAmbientStaticSelectorCarriesNegativeTwoAndPotentialNonNegative(t *testing.T) {
	g := terrain.Grid{Width: 3, Height: 1, Tiles: make([]uint16, 3), Overlay: []uint8{1, 2, 3}}
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{FireObject: -2}
	set.Classes[2] = &terrain.StaticClass{FireObject: -1}
	set.Classes[3] = &terrain.StaticClass{FireObject: 4}
	got := ambientStaticCells(g, set)
	if len(got) != 2 || got[0].fireObject != -2 || got[1].fireObject != 4 {
		t.Fatalf("ambient static cells = %+v, want -2 and non-negative while -1 is absent", got)
	}
}

func TestExitToMainStopsActiveAmbientWithoutReleasingSession(t *testing.T) {
	bank := &ambientBank{samples: map[int]audio.Sample{ambientRiverSlot: ambientSample(50)}}
	device := &ambientLoopRecorder{}
	viewer := &Viewer{ambient: newAmbientController(bank, &ambientOneShot{}, device, random.NewStream(1071))}
	viewer.ambient.update(time.Unix(30, 0), ambientSnapshot{
		geometry: audio.ViewGeometry{Span: image.Pt(10, 10)},
		listener: image.Pt(5, 5),
		hasView:  true,
		river:    []image.Point{image.Pt(6, 5)},
	})
	if len(device.starts) != 1 || device.starts[0].kind != AmbientRiver {
		t.Fatalf("active loop starts = %+v, want one river", device.starts)
	}

	f := openMissionMenu(t)
	f.viewer = viewer
	f.tick = func() {}
	f.rebuildGameMenu(gameMenuEndQuestConfirmation, 0)
	selectGameMenuAction(f, gameMenuExitMain)
	f.chooseGameMenu()

	if f.screen != ScreenMenu {
		t.Fatalf("Exit to Main Menu reached %v, want main menu", f.screen)
	}
	if f.viewer != viewer || f.tick == nil {
		t.Fatal("Exit to Main Menu released the retained viewer/session seams")
	}
	if device.stopAll != 1 || viewer.ambient != nil {
		t.Fatalf("departed map ambience = stop calls %d controller %v, want one stop and no controller",
			device.stopAll, viewer.ambient)
	}
}

func pointsEqual(a, b []image.Point) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
