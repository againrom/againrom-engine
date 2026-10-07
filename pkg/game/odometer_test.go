package game

// The walk odometer at the seam (SC-4, SC-5, SC-7's second half, SC-9).
//
// The worlds here are built in this file rather than loaded, so nothing
// measured depends on what a loader gives a placement. Every expected number is
// worked out in the comment beside it from the contract's own two figures — 256
// sub-cell units to a cell, sixteen to a timeline step — and never read back off
// the odometer.
//
// SC-7's first half is not here and does not need to be: drawn_invariance_test.go
// already drives this seam against a HEADLESS sim.Step over the same command
// stream and compares digests, so a walk odometer that reached a world field
// would turn that file red rather than this one.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// odoSpeed is the rate the movers below carry and odoTicks what one straight
// cell then costs them, written out rather than derived: the rate law is
// pkg/sim's and unexported, and re-deriving it here would assert our own
// arithmetic twice. At this speed the law's multiplier and its substitute for an
// absent cost plane cancel, so 256 over the speed rounds up to the count.
const (
	odoSpeed = 16
	odoTicks = 16
)

// odoWalker is a walker whose selected frame IS its track's own value:
// Flip-0, every base 0, slot length 0 and no wind-up, so the direction term
// vanishes and what is left is the timeline. track's LENGTH is the walk
// cycle.
func odoWalker(track []int, idle bool) *terrain.UnitClass {
	c := worldFixtureArt(8, 8, 4, 7, 4, 4, 8)
	c.Anim = terrain.UnitAnim{S: 16, D: 8, Total: 8,
		MoveTrack: track, MoveOK: true}
	if idle {
		c.Anim.IdleTrack = []int{0}
		c.Anim.IdleOK = true
		c.Anim.IdleSlot = 1
	}
	return c
}

// The two shipped shapes: eight art frames held two timeline steps each, cycle
// 16 — one whole cycle to a cell — and seven held two, cycle 14, one of the five
// lengths the shipped registry carries that do NOT divide sixteen.
func odoAlignedTrack() []int   { return []int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7} }
func odoUnalignedTrack() []int { return []int{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6} }

// odoWorld is two rated movers of the given classes on an open plane, each
// ordered a long way east so the walk runs several cells without stopping.
func odoWorld(t *testing.T, classes map[int32]*terrain.UnitClass, speed int32) (*mapWorld, *ui.Viewer) {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{
			{ID: 0, X: 10, Y: 20, Class: 1, Speed: speed},
			{ID: 1, X: 10, Y: 24, Class: 2, Speed: speed},
		})
	if err != nil {
		t.Fatalf("building the odometer world: %v", err)
	}
	sched := [][]sim.Command{{
		{Entity: 0, X: 40, Y: 20},
		{Entity: 1, X: 40, Y: 24},
	}}
	return newMapWorld(w, sched, &terrain.UnitSet{Classes: classes}, v), v
}

// frameIndexOf finds which of a class's frames the seam selected, by pointer.
// The fixture gives every frame its own pointer, so this says WHICH frame was
// drawn without re-deriving an index from the descriptor.
func frameIndexOf(c *terrain.UnitClass, d ui.MapEntity) int {
	for i, f := range c.Frames {
		if f == d.Frame {
			return i
		}
	}
	return -1
}

func TestTheOdometerIsTheRunningTotalOfTheCrossings(t *testing.T) {
	classes := map[int32]*terrain.UnitClass{
		1: odoWalker(odoAlignedTrack(), false),
		2: odoWalker(odoUnalignedTrack(), false),
	}
	mw, _ := odoWorld(t, classes, odoSpeed)

	for cell := 1; cell <= 3; cell++ {
		for k := 1; k <= odoTicks; k++ {
			mw.tick()
			want := (cell-1)*256 + k*16
			for _, id := range []sim.EntityID{0, 1} {
				if got := mw.walk[id].dist; got != want {
					t.Fatalf("entity %d, cell %d tick %d: walked %d, want %d", id, cell, k, got, want)
				}
			}
		}
	}
	// Three cells walked: three times sixteen timeline steps, no residue.
	if got := terrain.WalkPhase(mw.walk[0].dist); got != 48 {
		t.Errorf("three straight cells advanced the timeline %d steps, want 48", got)
	}
}

// TestTwoSpeedsAdvanceTheSameSixteenSteps covers AC-1, the story's own headline
// put as the owner would see it: two units of different speeds cross the same
// ground and their feet keep pace with their strides rather than with the clock.
//
// Both cross one cell. The slow one takes 32 ticks and the fast one 8, both
// advance the timeline by exactly sixteen steps, and the slow one is DRAWN on
// more of those steps — 16, holding each twice — while the fast one is drawn on
// 8 and skips the rest. A crossing shows min(span, 16) of the sixteen, which is
// what a tick-driven walk gets backwards: on the tick both would show as many
// steps as they have ticks and the slow one would cycle four times over.
//
// The measure is DISTINCT TIMELINE STEPS and not distinct art frames: the
// shipped shape holds each art frame for two steps, so at eight ticks a cell the
// fast unit lands on all eight frames while visiting only half the steps.
func TestTwoSpeedsAdvanceTheSameSixteenSteps(t *testing.T) {
	slow, fast := odoWalker(odoAlignedTrack(), false), odoWalker(odoAlignedTrack(), false)
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{
			{ID: 0, X: 10, Y: 20, Class: 1, Speed: 8},  // 256/8  = 32 ticks a cell
			{ID: 1, X: 10, Y: 24, Class: 2, Speed: 32}, // 256/32 =  8 ticks a cell
		})
	if err != nil {
		t.Fatalf("building the two-speed world: %v", err)
	}
	sched := [][]sim.Command{{{Entity: 0, X: 11, Y: 20}, {Entity: 1, X: 11, Y: 24}}}
	mw := newMapWorld(w, sched, &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{1: slow, 2: fast}}, v)

	// Frames are collected only while a unit is still walking, and the peak count
	// is taken as it goes: the fast one arrives at tick 8 and, having no idle
	// cycle, resets on tick 9 while the slow one is still crossing.
	slowSteps, fastSteps := map[int]bool{}, map[int]bool{}
	peak := map[sim.EntityID]int{}
	for k := 0; k < 32; k++ {
		mw.tick()
		for i, d := range mw.entityDraws() {
			if d.Step == (image.Point{}) {
				continue
			}
			id := sim.EntityID(i)
			if mw.walk[id].dist > peak[id] {
				peak[id] = mw.walk[id].dist
			}
			// The step this frame was drawn at, taken through the same
			// conversion the selection uses, and the frame checked beside it so
			// the two cannot come apart.
			step := terrain.WalkPhase(mw.walk[id].dist)
			class := slow
			seen := slowSteps
			if id == 1 {
				class, seen = fast, fastSteps
			}
			seen[step] = true
			if want := odoAlignedTrack()[step%16]; frameIndexOf(class, d) != want {
				t.Fatalf("entity %d at step %d drew frame %d, want %d",
					id, step, frameIndexOf(class, d), want)
			}
		}
	}
	// One cell each, and the same sixteen timeline steps for both.
	for _, id := range []sim.EntityID{0, 1} {
		if peak[id] != 256 || terrain.WalkPhase(peak[id]) != 16 {
			t.Fatalf("entity %d walked %d units over its cell, %d timeline steps, want 256 and 16",
				id, peak[id], terrain.WalkPhase(peak[id]))
		}
	}
	// The slow unit pays 8 units a tick, so its step is k/2 over 32 ticks: every
	// one of steps 0..16, seventeen values — step 0 because its first tick has
	// not yet covered a sixteenth of the cell, and step 16 on arrival. The fast
	// unit pays 32 a tick, so its step is 2k: eight values, and it never lands on
	// an odd one.
	if len(slowSteps) != 17 {
		t.Errorf("the slow unit was drawn on %d timeline steps, want steps 0..16, seventeen values",
			len(slowSteps))
	}
	if len(fastSteps) != 8 {
		t.Errorf("the fast unit was drawn on %d timeline steps, want 8 — it skips half of them",
			len(fastSteps))
	}
	if len(slowSteps) <= len(fastSteps) {
		t.Errorf("the slow unit saw %d steps and the fast one %d; the slow one must see more",
			len(slowSteps), len(fastSteps))
	}
}

func TestTheWalkFrameFollowsTheOdometerAcrossCells(t *testing.T) {
	aligned := odoWalker(odoAlignedTrack(), false)
	unaligned := odoWalker(odoUnalignedTrack(), false)
	mw, _ := odoWorld(t, map[int32]*terrain.UnitClass{1: aligned, 2: unaligned}, odoSpeed)

	wantUnaligned := [3]int{1, 2, 3}
	var seen []int
	for cell := 0; cell < 3; cell++ {
		for k := 0; k < odoTicks; k++ {
			mw.tick()
		}
		draws := mw.entityDraws()
		if got := frameIndexOf(aligned, draws[0]); got != 0 {
			t.Errorf("cycle 16, boundary %d drew frame %d, want 0", cell+1, got)
		}
		got := frameIndexOf(unaligned, draws[1])
		if got != wantUnaligned[cell] {
			t.Errorf("cycle 14, boundary %d drew frame %d, want %d", cell+1, got, wantUnaligned[cell])
		}
		seen = append(seen, got)
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] == seen[i-1] {
			t.Errorf("cycle 14 drew frame %d at two consecutive boundaries", seen[i])
		}
	}
}

func TestAStopResetsOnlyAClassWithNoIdleCycle(t *testing.T) {
	classes := map[int32]*terrain.UnitClass{
		1: odoWalker(odoAlignedTrack(), false), // no idle cycle
		2: odoWalker(odoAlignedTrack(), true),  // an idle cycle
	}
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{
			{ID: 0, X: 10, Y: 20, Class: 1, Speed: odoSpeed},
			{ID: 1, X: 10, Y: 24, Class: 2, Speed: odoSpeed},
		})
	if err != nil {
		t.Fatalf("building the stopping world: %v", err)
	}
	sched := [][]sim.Command{{{Entity: 0, X: 11, Y: 20}, {Entity: 1, X: 11, Y: 24}}}
	mw := newMapWorld(w, sched, &terrain.UnitSet{Classes: classes}, v)

	for k := 0; k < odoTicks; k++ {
		mw.tick()
	}
	for _, id := range []sim.EntityID{0, 1} {
		if got := mw.walk[id].dist; got != 256 {
			t.Fatalf("entity %d walked %d over one cell, want 256", id, got)
		}
	}

	// The ARRIVAL tick is inside the crossing it completes, so the earliest tick
	// that can reset is the one after it.
	mw.tick()
	if got := mw.walk[0].dist; got != 0 {
		t.Errorf("a class with no idle cycle stood still and kept %d units, want 0", got)
	}
	if got := mw.walk[1].dist; got != 256 {
		t.Errorf("a class with an idle cycle stood still and kept %d units, want its own 256", got)
	}
}

func TestAnUnratedMoverPaysAWholeCellEveryTick(t *testing.T) {
	classes := map[int32]*terrain.UnitClass{1: odoWalker(odoAlignedTrack(), false)}
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{{ID: 0, X: 10, Y: 20, Class: 1, Speed: odoSpeed}})
	if err != nil {
		t.Fatalf("building the unrated world: %v", err)
	}
	mw := newMapWorld(w, [][]sim.Command{{{Entity: 0, X: 11, Y: 20}}}, &terrain.UnitSet{Classes: classes}, v)
	for k := 0; k < odoTicks; k++ {
		mw.tick()
	}
	if got := mw.walk[0].dist; got != 256 {
		t.Fatalf("the rated crossing walked %d, want 256", got)
	}
	if got := mw.world.Entities()[0].TransitTotal; got != odoTicks {
		t.Fatalf("the rated crossing left a total of %d, want %d", got, odoTicks)
	}

	// Take the speed away and order it on. The total is now stale at sixteen
	// while the mover crosses a cell a tick.
	ents := mw.world.Entities()
	ents[0].Speed = 0
	w2, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, ents)
	if err != nil {
		t.Fatalf("rebuilding without a speed: %v", err)
	}
	mw.world = w2
	mw.pending = append(mw.pending, sim.Command{Entity: 0, X: 20, Y: 20})
	before := mw.walk[0].dist
	mw.tick()
	if got := mw.walk[0].dist - before; got != 256 {
		t.Errorf("an unrated mover walked %d in one tick, want a whole cell, 256", got)
	}
}

// TestTheSubCellGridIsOneNumberAcrossTheWall covers SC-9 and holds R-4 down. The
// odometer's 256 lives in pkg/render/terrain and pkg/sim's lives behind an
// unexported name; the transit a mover at the RATE FLOOR is given is the only
// window onto the second, and the two must be one number or a cell's worth of
// travel and a cell's worth of ticks are measuring different cells.
func TestTheSubCellGridIsOneNumberAcrossTheWall(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{{ID: 0, X: 10, Y: 20, Speed: 1}})
	if err != nil {
		t.Fatalf("building the floor-rate world: %v", err)
	}
	mw := newMapWorld(w, [][]sim.Command{{{Entity: 0, X: 40, Y: 20}}}, nil, v)
	mw.tick()

	simGrid := int(mw.world.Entities()[0].TransitTotal)
	odoGrid := terrain.WalkAdvance(1, 0, 1, 0)
	if simGrid != odoGrid {
		t.Errorf("pkg/sim crosses a cell at the rate floor in %d ticks while the odometer pays %d "+
			"units for one cell; the sub-cell grid has drifted into two numbers", simGrid, odoGrid)
	}
}

func TestBuildingASnapshotTwiceSelectsTheSameFrames(t *testing.T) {
	classes := map[int32]*terrain.UnitClass{
		1: odoWalker(odoAlignedTrack(), false),
		2: odoWalker(odoUnalignedTrack(), true),
	}
	mw, _ := odoWorld(t, classes, odoSpeed)

	for k := 0; k < 3*odoTicks+5; k++ {
		mw.tick()
		before := map[sim.EntityID]walkClock{}
		for id, w := range mw.walk {
			before[id] = w
		}
		first := mw.entityDraws()
		second := mw.entityDraws()
		for i := range first {
			if first[i].Frame != second[i].Frame || first[i].Mirror != second[i].Mirror ||
				first[i].Step != (image.Point{}) && first[i].Step != second[i].Step {
				t.Fatalf("tick %d: two builds of one picture selected (%p, %t) then (%p, %t)",
					k, first[i].Frame, first[i].Mirror, second[i].Frame, second[i].Mirror)
			}
		}
		for id, w := range before {
			if mw.walk[id] != w {
				t.Fatalf("tick %d: drawing moved entity %d's odometer from %+v to %+v",
					k, id, w, mw.walk[id])
			}
		}
	}
}
