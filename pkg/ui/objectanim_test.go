package ui

// The window's half of the object cycle (0031 AC-8, SC-7).
//
// SEPARATE CONTEXT: no window is opened, no GPU is touched and no clock is read.
// The viewer is driven through step() with timestamps the test supplies, which
// is the same seam the cadence tests use, so what is measured is the counter and
// the frames a rendered frame would take from it.
//
// The expectations are computed from the render tier's OWN selector at the
// counter the viewer reports, not from the viewer's list: that is what makes
// this file evidence that the window reads ONE counter rather than evidence that
// it agrees with itself.

import (
	"fmt"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// objectAnimTimeline is the fixture cycle: [0 1 2], period 3, so three
// consecutive counters at one cell select three different frames.
var objectAnimTimeline = []int{0, 1, 2}

// objectAnimBundle is a bundle whose byte-1 class carries a three-frame sheet
// and that cycle, and whose byte-2 class carries the same sheet and NO cycle at
// an Index of 2 — the class that makes the switch-off arm visible, since frame 0
// is not the frame it draws.
func objectAnimBundle() *terrain.StaticSet {
	frames := []*terrain.StaticFrame{
		staticsFrame(10, 6, 7),
		staticsFrame(12, 9, 5),
		staticsFrame(4, 4, 9),
	}
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{
		Width: 64, Height: 64, CenterX: 32, CenterY: 60,
		Frame: frames[0], Frames: frames, Index: 0, Timeline: objectAnimTimeline,
	}
	set.Classes[2] = &terrain.StaticClass{
		Width: 31, Height: 31, CenterX: 15, CenterY: 29,
		Frame: frames[2], Frames: frames, Index: 2,
	}
	set.Classes[3] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 7}
	return set
}

// objectAnimViewerGrid is staticsGrid with every tile word carrying both gate
// bits, so the DECODED gate — and not the diagnostic — is what opens the cycles
// this file drives.
func objectAnimViewerGrid() terrain.Grid {
	g := staticsGrid()
	g.Tiles = make([]uint16, len(g.Tiles))
	for i := range g.Tiles {
		g.Tiles[i] = 0xc000
	}
	return g
}

func newObjectAnimViewer(t *testing.T, g terrain.Grid, gate bool) *Viewer {
	t.Helper()
	return newObjectAnimViewerWith(t, g, objectAnimBundle(), gate)
}

// newObjectAnimViewerWith takes the bundle, so two viewers can be built over ONE
// set: a placement carries pointers into the bundle it was built from, and two
// bundles would make every placement differ for a reason that is not the gate.
func newObjectAnimViewerWith(t *testing.T, g terrain.Grid, set *terrain.StaticSet, gate bool) *Viewer {
	t.Helper()
	v, err := NewViewerWithStatics("m", g, &terrain.Tileset{}, set, true, false, gate, nil, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	return v
}

// driveObjectAnim advances the viewer by n whole ticks of its current period,
// through the same step() the run loop calls, and returns the timestamp it left
// off at. The first call establishes the baseline, exactly as a real frame does.
func driveObjectAnim(v *Viewer, now time.Time, ticks int) time.Time {
	v.step(Input{}, now)
	for i := 0; i < ticks; i++ {
		now = now.Add(time.Duration(v.anim.Period()) * time.Microsecond)
		v.step(Input{}, now)
	}
	return now
}

// wantObjectFrames is what the object layer must draw at one counter, computed
// from the render tier's selector over the bundle and the built list — never
// from the viewer's own answer.
func wantObjectFrames(v *Viewer, counter uint32, animate bool) []*terrain.StaticFrame {
	places, animated := v.staticLists()
	open := make(map[int]bool, len(animated))
	for _, i := range animated {
		open[i] = true
	}
	out := make([]*terrain.StaticFrame, len(places))
	for i, p := range places {
		c := p.Class
		sel := terrain.SelectObjectFrame(c.Timeline, c.Index, len(c.Frames), p.Cell.X, p.Cell.Y, counter, animate, open[i])
		out[i] = c.Frames[sel]
	}
	return out
}

func checkObjectFrames(t *testing.T, v *Viewer, label string, animate bool) {
	t.Helper()
	counter := v.AnimationCounter()
	want := wantObjectFrames(v, counter, animate)
	got := v.staticPlacements()
	if len(got) != len(want) {
		t.Fatalf("%s: %d placements drawn, want %d", label, len(got), len(want))
	}
	for i := range got {
		if got[i].Frame != want[i] {
			t.Errorf("%s: placement %d at counter %d drew %v, want %v",
				label, i, counter, got[i].Frame, want[i])
		}
	}
}

// TestObjectsAndWaterReadOneCounter is AC-8's first clause and SC-7's: the
// counter the object layer steps on is the counter the water phase reads,
// compared AS ONE VALUE through both consumers, at one rate and then at the rate
// doubled.
//
// The cadence is doubled through SetPeriod, which is the only re-rating there
// is; the object layer gains no cadence, no stop and no switch of its own, so a
// second one could only appear as a disagreement between these two readings.
func TestObjectsAndWaterReadOneCounter(t *testing.T) {
	v := newObjectAnimViewer(t, objectAnimViewerGrid(), terrain.AnimGateTiles)
	if _, animated := v.staticLists(); len(animated) == 0 {
		t.Fatal("the fixture opens no cycle; this test would assert nothing")
	}

	// ONE unbroken timeline across the re-rate: the accumulator carries its
	// remainder, so a jump in the timestamps would fire a burst of ticks and
	// measure the jump instead of the rate.
	now := time.Unix(0, 0)
	v.step(Input{}, now)

	// Two periods a doubling apart, written out rather than divided here: 125000
	// us is eight ticks a second and 62500 is sixteen.
	for _, periodUS := range []int{125_000, 62_500} {
		if got := v.SetPeriod(periodUS); got != periodUS {
			t.Fatalf("SetPeriod(%d) adopted %d", periodUS, got)
		}
		before := v.AnimationCounter()
		for i := 0; i < 5; i++ {
			now = now.Add(time.Duration(v.anim.Period()) * time.Microsecond)
			v.step(Input{}, now)
		}
		if advanced := v.AnimationCounter() - before; advanced != 5 {
			t.Fatalf("%d us: the counter advanced %d over five of its own periods, want 5", periodUS, advanced)
		}
		// The water phase and the object step, at the counter the viewer
		// reports: one value, read twice.
		counter := v.AnimationCounter()
		if ref := v.resolveCell(v.tileWord(0, 0), 0, 0); ref != terrain.ResolveAnimated(v.tileWord(0, 0), 0, 0, counter) {
			t.Errorf("%d us: the water cell resolves at a counter other than %d", periodUS, counter)
		}
		checkObjectFrames(t, v, fmt.Sprintf("%d us", periodUS), true)
	}

	// The doubling is what the two readings were compared ACROSS: at 62500 us the
	// same wall-clock span carries twice the counter it carried at 125000, and
	// both consumers moved with it. Asserted against the rate model's own
	// quotients, so a fixture edited to two periods that are not a doubling stops
	// claiming to be one.
	if terrain.RatePeriod(8) != 125_000 || terrain.RatePeriod(16) != 62_500 {
		t.Fatalf("the fixture periods are not 8/s and 16/s: %d vs %d",
			terrain.RatePeriod(8), terrain.RatePeriod(16))
	}
}

func TestObjectsCycleAsTheCounterRises(t *testing.T) {
	v := newObjectAnimViewer(t, objectAnimViewerGrid(), terrain.AnimGateTiles)
	places, animated := v.staticLists()
	if len(animated) == 0 {
		t.Fatal("the fixture opens no cycle")
	}
	at := animated[0]

	now := time.Unix(0, 0)
	v.step(Input{}, now)

	seen := map[*terrain.StaticFrame]bool{}
	for i := 0; i < len(objectAnimTimeline); i++ {
		checkObjectFrames(t, v, "cycling", true)
		seen[v.staticPlacements()[at].Frame] = true
		now = now.Add(time.Duration(v.anim.Period()) * time.Microsecond)
		v.step(Input{}, now)
	}
	if len(seen) != len(objectAnimTimeline) {
		t.Errorf("%d distinct frames over a period of %d", len(seen), len(objectAnimTimeline))
	}

	// The BUILT list never moved: the pass writes into the viewer's own scratch
	// and turning a counter changes what is painted, never what was placed.
	for i, p := range places {
		if p.Frame != p.Class.Frames[p.Class.Index] {
			t.Errorf("built placement %d was written through by the per-counter pass", i)
		}
	}
}

func TestStaticObjectAnimationFollowsLiveFogWithoutViewerRebuild(t *testing.T) {
	v := newObjectAnimViewer(t, staticsGrid(), terrain.AnimGateAll)
	built, candidates := v.staticLists()
	if len(built) == 0 || len(candidates) != 2 {
		t.Fatalf("built %d placements with %d cycle candidates, want a non-empty list and 2 candidates",
			len(built), len(candidates))
	}
	builtFirst := &built[0]
	candidateFirst := &candidates[0]

	plane := make([]byte, 3*4)
	setTargetFog := func(state uint8) {
		for i := range plane {
			plane[i] = FogUnseen
		}
		plane[0] = state // animated class at cell (0,0)
		v.SetFog(plane, 3, 4)
	}
	findTarget := func(places []terrain.StaticPlacement) (terrain.StaticPlacement, bool) {
		for _, p := range places {
			if p.Cell.X == 0 && p.Cell.Y == 0 {
				return p, true
			}
		}
		return terrain.StaticPlacement{}, false
	}

	setTargetFog(FogUnseen)
	if drawn := v.staticPlacements(); len(drawn) != 0 {
		t.Fatalf("unseen plane drew %d objects, want none", len(drawn))
	}
	if v.staticScratch != nil {
		t.Fatal("an entirely unseen cycle population allocated an animation buffer")
	}

	setTargetFog(FogVisible)
	driveObjectAnim(v, time.Unix(0, 0), 1)
	visible, ok := findTarget(v.staticPlacements())
	if !ok {
		t.Fatal("visible target object was not drawn")
	}
	if got, want := visible.Frame, visible.Class.Frames[1]; got != want {
		t.Fatalf("visible target at counter %d drew %p, want cycle frame %p",
			v.AnimationCounter(), got, want)
	}

	setTargetFog(FogExplored)
	if got := v.AnimationCounter(); got != 1 {
		t.Fatalf("SetFog moved the animation counter to %d, want 1", got)
	}
	explored, ok := findTarget(v.staticPlacements())
	if !ok {
		t.Fatal("explored target object disappeared instead of remaining dimmed")
	}
	if got, want := explored.Frame, explored.Class.Frame; got != want {
		t.Errorf("explored-hidden target drew live frame %p, want built Index frame %p", got, want)
	}

	setTargetFog(FogVisible)
	visibleAgain, ok := findTarget(v.staticPlacements())
	if !ok {
		t.Fatal("target did not return when visibility returned")
	}
	if got, want := visibleAgain.Frame, visibleAgain.Class.Frames[1]; got != want {
		t.Errorf("visible-again target drew %p, want resumed counter-1 frame %p", got, want)
	}

	setTargetFog(FogUnseen)
	if drawn := v.staticPlacements(); len(drawn) != 0 {
		t.Errorf("second unseen transition drew %d objects, want none", len(drawn))
	}
	if got := v.AnimationCounter(); got != 1 {
		t.Errorf("fog transitions moved the animation counter to %d, want 1", got)
	}
	if current, currentCandidates := v.staticLists(); &current[0] != builtFirst || &currentCandidates[0] != candidateFirst {
		t.Fatal("SetFog rebuilt the object placement or cycle-candidate list")
	}
}

// TestAnimationOffHoldsTheCounterAndDrawsFrameZero is AC-8's remaining clauses:
// with animation off the counter HOLDS and every object draws sheet frame 0, and
// switching it on resumes from the held value.
//
// It is driven over a map whose gate opens NOTHING, because that is the shipped
// case and the one where the switch is the only thing that can move a frame: on
// such a map the object layer is otherwise identical to the pre-story one at
// every counter.
func TestAnimationOffHoldsTheCounterAndDrawsFrameZero(t *testing.T) {
	v := newObjectAnimViewer(t, staticsGrid(), terrain.AnimGateTiles)
	if _, animated := v.staticLists(); len(animated) != 0 {
		t.Fatal("the fixture opens a cycle; this test is about the shipped case, where none is open")
	}
	places, _ := v.staticLists()

	now := driveObjectAnim(v, time.Unix(0, 0), 4)
	running := v.AnimationCounter()
	if running == 0 {
		t.Fatal("the counter did not advance while animation was on")
	}
	// On this map, with the switch on, the drawn list IS the built list.
	if drawn := v.staticPlacements(); !samePlacements(drawn, places) {
		t.Error("a map that opens no cycle drew something other than the list the builder produced")
	}

	v.SetAnimated(false)
	for i := 0; i < 6; i++ {
		now = now.Add(time.Duration(v.anim.Period()) * time.Microsecond)
		v.step(Input{}, now)
	}
	if held := v.AnimationCounter(); held != running {
		t.Errorf("the counter moved from %d to %d with animation off", running, held)
	}
	drawn := v.staticPlacements()
	if len(drawn) != len(places) {
		t.Fatalf("%d placements with animation off, want %d", len(drawn), len(places))
	}
	for i, p := range drawn {
		if p.Frame != p.Class.Frames[0] {
			t.Errorf("placement %d drew %v with animation off, want its sheet's frame 0 %v",
				i, p.Frame, p.Class.Frames[0])
		}
		if p.Ground() != places[i].Ground() {
			t.Errorf("placement %d ground %v with animation off, want the build's %v",
				i, p.Ground(), places[i].Ground())
		}
	}
	checkObjectFrames(t, v, "animation off", false)

	// The water half of the same switch is unmoved: phase 0 rather than a
	// frozen current phase, which is what freezing the counter at a value the
	// resolve ignores gives.
	if ref := v.resolveCell(v.tileWord(0, 0), 0, 0); ref != terrain.Resolve(v.tileWord(0, 0)) {
		t.Error("water did not resolve statically with animation off")
	}

	v.SetAnimated(true)
	now = now.Add(time.Duration(v.anim.Period()) * time.Microsecond)
	v.step(Input{}, now)
	if got := v.AnimationCounter(); got != running+1 {
		t.Errorf("the counter resumed at %d, want %d — the held value plus one tick", got, running+1)
	}
	if drawn := v.staticPlacements(); !samePlacements(drawn, places) {
		t.Error("switching animation back on did not return the layer to the built list")
	}
}

func TestObjectAnimDiagnosticMovesOnlyTheSubset(t *testing.T) {
	g := staticsGrid() // no tile word sets either gate bit
	set := objectAnimBundle()
	plain := newObjectAnimViewerWith(t, g, set, terrain.AnimGateTiles)
	diag := newObjectAnimViewerWith(t, g, set, terrain.AnimGateAll)

	plainPlaces, plainAnim := plain.staticLists()
	diagPlaces, diagAnim := diag.staticLists()

	if len(plainAnim) != 0 {
		t.Fatalf("the decoded gate opened %d cycles on a map that sets no gate bit", len(plainAnim))
	}
	if len(diagAnim) == 0 {
		t.Fatal("the diagnostic opened no cycle at all")
	}
	if len(plainPlaces) != len(diagPlaces) {
		t.Fatalf("%d placements plain, %d under the diagnostic", len(plainPlaces), len(diagPlaces))
	}
	for i := range plainPlaces {
		if plainPlaces[i] != diagPlaces[i] {
			t.Errorf("the diagnostic moved built placement %d", i)
		}
	}

	_, plainCounts := plain.Statics()
	_, diagCounts := diag.Statics()
	if plainCounts.Animated != 0 || diagCounts.Animated != len(diagAnim) {
		t.Errorf("census Animated = %d plain, %d diagnostic, want 0 and %d",
			plainCounts.Animated, diagCounts.Animated, len(diagAnim))
	}
	plainCounts.Animated = diagCounts.Animated
	if plainCounts != diagCounts {
		t.Errorf("the diagnostic changed the census beyond Animated: %+v vs %+v", diagCounts, plainCounts)
	}

	// The classes with no cycle draw their own Index under the diagnostic too:
	// it opens a gate, it does not invent a cycle.
	drawn := diag.staticPlacements()
	for i, p := range drawn {
		if len(p.Class.Timeline) == 0 && p.Frame != p.Class.Frames[p.Class.Index] {
			t.Errorf("placement %d has no cycle but the diagnostic moved its frame", i)
		}
	}
}

func TestObjectAnimScratchIsNotTheBuiltList(t *testing.T) {
	still := newObjectAnimViewer(t, staticsGrid(), terrain.AnimGateTiles)
	for i := 0; i < 4; i++ {
		still.staticPlacements()
	}
	if still.staticScratch != nil {
		t.Errorf("a map that opens no cycle allocated a %d-entry scratch", len(still.staticScratch))
	}

	moving := newObjectAnimViewer(t, objectAnimViewerGrid(), terrain.AnimGateTiles)
	places, _ := moving.staticLists()
	moving.staticPlacements()
	if moving.staticScratch == nil {
		t.Fatal("a map that opens a cycle drew without a buffer")
	}
	if samePlacements(moving.staticScratch, places) {
		t.Fatal("the scratch aliases the list the builder produced")
	}
	first := &moving.staticScratch[0]
	for i := 0; i < 5; i++ {
		moving.staticPlacements()
		if &moving.staticScratch[0] != first {
			t.Fatalf("frame %d allocated a second buffer", i)
		}
	}
}
