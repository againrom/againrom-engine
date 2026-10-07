package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The fixture map's extent. SEVENTY-TWO ON BOTH AXES, and the size is not a
// round number: a derived passability plane blocks the outer eight rings of
// every map, and mapload.Schedule aims a leg at the extent and caps it at 24
// cells toward the roomier side — so a full leg from an interior start clears
// the ring on the far side only once the extent reaches 64. Below that a capped
// leg lands in the border, the unit holds its cell, and every test here that
// asserts motion fails for a reason unrelated to what it measures.
//
// So the legs below are FULL LENGTH rather than capped by the room on their own
// axis, which is the arm of mapload.Schedule's rule this file now leans on.
const (
	worldFixtureW = 72
	worldFixtureH = 72
)

// worldFixtureUnits are the fixture's four placed units, at hand-written
// fixed-point anchors. Three of them sit off a cell boundary on at least one
// axis, and unit 2's low byte is 0xff — the value a derivation that ROUNDED
// rather than truncated would put on the next cell over, and the one AC-8 asks
// for by name.
//
// All four stand INSIDE the interior — clear of the eight-cell ring on every
// side — and each fractional part is the one it always carried.
//
// Each carries a class key the hand-assembled bundle below resolves (0022
// AC-8: a synthetic map whose units resolve). The ids are sparse and
// non-adjacent, mirroring the contract's own domain, so a resolution keyed by
// slice index instead of ID answers nothing at any of them.
var worldFixtureUnits = []alm.Unit{
	{X: 0x1580, Y: 0x1700, ClassID: 3},  // 21.5, 23.0
	{X: 0x1900, Y: 0x1480, ClassID: 7},  // 25.0, 20.5
	{X: 0x16ff, Y: 0x1bc0, ClassID: 21}, // 22.996…, 27.75
	{X: 0x1f40, Y: 0x1840, ClassID: 34}, // 31.25, 24.25
}

// cellTicks is how many ticks one straight cell costs a unit these fixtures
// place ON LEVEL GROUND, diagonalCellTicks the same for a diagonal one, and the
// two uphill counts beside them what the SAME steps cost climbing this fixture's
// map.
//
// All four are written out rather than computed. Every placement in this
// package's maps resolves to no definition, so each takes mapload.DefaultSpeed —
// the base constructor's own 10 — and every tile word in them is zero, which
// derives to the default cost of 8, so the rate law composes 10 and the sub-cell
// grid of 256 divides by it to 26 ticks a cell; a diagonal step is 10 times the
// diagonal factor, truncated to 7, and 256 over 7 rounds up to 37. pkg/sim keeps
// its law unexported, and a test that re-derived it here would be asserting our
// own arithmetic twice.
//
// THE UPHILL PAIR ARRIVED WITH THE HEIGHT PLANE (0076). This fixture's map ramps
// three per column eastward, so every EASTWARD leg in this file climbs: the tilt
// takes the rate from 10 to 9, which is 29 ticks a straight cell and 43 a
// diagonal one. A leg along a column crosses no height at all and still costs
// the level pair, which is why both survive. Before the planes there was one
// pair and it was the level one; a count that changed here changed for that
// reason and no other.
//
// Before the rate itself existed every one of them was 1, which is what every
// count in this file was written against.
const (
	cellTicks               = 26
	diagonalCellTicks       = 37
	uphillCellTicks         = 29
	uphillDiagonalCellTicks = 43
)

// worldFixtureCells is the cell each of those units stands on, written out by
// hand and not computed: the whole part of each anchor, the fraction dropped.
// Nothing in this file derives these from the anchors above.
var worldFixtureCells = []image.Point{
	{X: 21, Y: 23},
	{X: 25, Y: 20},
	{X: 22, Y: 27},
	{X: 31, Y: 24},
}

// worldFixtureAltitudes is a relief that rises to the east, three native pixels
// per column. It exists so displaced mode is actually displaced AND so the lift
// DIFFERS from cell to cell: a uniform relief would let a marker carrying the
// wrong cell's lift pass unnoticed.
//
// Three and not four: the plane is a byte per cell and the extent is now 72
// columns, so a gradient of four would wrap at column 64 and hand two columns
// the same height.
func worldFixtureAltitudes() []uint8 {
	alts := make([]uint8, worldFixtureW*worldFixtureH)
	for row := 0; row < worldFixtureH; row++ {
		for col := 0; col < worldFixtureW; col++ {
			alts[row*worldFixtureW+col] = uint8(3 * col)
		}
	}
	return alts
}

// worldFixtureMap is the decoded map itself, built as a literal: no bytes,
// no alm.Open, no builder.
func worldFixtureMap() *alm.Map {
	return &alm.Map{
		Width:     worldFixtureW,
		Height:    worldFixtureH,
		Name:      "Fixture",
		Tiles:     make([]uint16, worldFixtureW*worldFixtureH),
		Altitudes: worldFixtureAltitudes(),
		Units:     append([]alm.Unit(nil), worldFixtureUnits...),
	}
}

// worldFixtureViewer is a viewer over that map's grid: the same layers
// LoadMapViewer hands one, assembled here so this file needs no map stream.
func worldFixtureViewer(t *testing.T, m *alm.Map) *ui.Viewer {
	t.Helper()
	v, err := ui.NewViewer(m.Name, terrain.Grid{
		Width:     m.Width,
		Height:    m.Height,
		Tiles:     m.Tiles,
		Altitudes: m.Altitudes,
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return v
}

// worldFixtureArt hand-assembles one drawable unit class: the class canvas
// and ground-touching pixel beside frames frames of the given size, every
// frame pixel opaque. Plain render-tier data, no archive, no decode. The
// frames are all the same size — placement reads the DRAWN frame's own
// size, so a selection moving between them moves no rectangle — but each
// is its OWN pointer, which is what lets a test say WHICH frame the seam
// selected without re-deriving an index.
func worldFixtureArt(canvasW, canvasH, cx, cy, frameW, frameH, frames int) *terrain.UnitClass {
	c := &terrain.UnitClass{Width: canvasW, Height: canvasH, CenterX: cx, CenterY: cy,
		Frames: make([]*terrain.StaticFrame, frames)}
	for n := range c.Frames {
		f := &terrain.StaticFrame{Width: frameW, Height: frameH, Pixels: make([]terrain.StaticPixel, frameW*frameH)}
		for i := range f.Pixels {
			f.Pixels[i] = terrain.StaticPixel{Index: 1, Opaque: true}
		}
		c.Frames[n] = f
	}
	return c
}

// worldFixtureMoverAnim is a Flip-1 walker's descriptor, every field a
// hand-written literal off the spec's sheet contract — never a call into
// pkg/data — for the class MB 1, MV 2, AT 0, DY 0, BN 0, ID 0 at (S, D) =
// (9, 5): MoveBase 9, every later base 9 + 5*(1+2) = 24, MoveSlot 3,
// MoveWind 1, predicted total 24; move pair Time [2 1], Frame [0 1] expands
// to the track [0 0 1], period 3.
func worldFixtureMoverAnim() terrain.UnitAnim {
	return terrain.UnitAnim{S: 9, D: 5,
		MoveBase: 9, AttackBase: 24, DyingBase: 24, TailBase: 24,
		MoveSlot: 3, MoveWind: 1, Total: 24,
		MoveTrack: []int{0, 0, 1}, MoveOK: true}
}

func worldFixtureIdleAnim() terrain.UnitAnim {
	return terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 16, AttackBase: 16, DyingBase: 16, TailBase: 16,
		IdleSlot: 2, Total: 32,
		IdleTrack: []int{0, 1}, IdleOK: true}
}

// worldFixtureUnitSet is the bundle under which every fixture unit resolves:
// one drawable class per ClassID above, each with a canvas UNEQUAL to its
// frame and a centre off the canvas centre, so an anchor read off the frame
// alone, or the canvas alone, moves the placements the instrument below
// stands on. The anchors, by the spec's formula (every /2 truncating):
//
//	class 3:  canvas 64x64, centre (32,60), frame 10x6 → anchor (5, 31)
//	class 7:  canvas 31x31, centre (15,29), frame  4x4 → anchor (2, 16)
//	class 21: canvas 48x56, centre (24,50), frame  3x2 → anchor (1, 23)
//	class 34: canvas 16x16, centre  (8,14), frame  5x4 → anchor (2,  8)
//
// A fresh value per call, like worldFixtureMap: no test can leak a mutation
// into the next.
func worldFixtureUnitSet() *terrain.UnitSet {
	mover := worldFixtureArt(64, 64, 32, 60, 10, 6, 24)
	mover.Anim = worldFixtureMoverAnim()
	idler := worldFixtureArt(48, 56, 24, 50, 3, 2, 32)
	idler.Anim = worldFixtureIdleAnim()
	return &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		3:  mover,
		7:  worldFixtureArt(31, 31, 15, 29, 4, 4, 1),
		21: idler,
		34: worldFixtureArt(16, 16, 8, 14, 5, 4, 1),
	}}
}

// seamDraws runs the production seam derivation once over a bare world and
// bundle, at the state a freshly opened screen holds — scene 0, an empty
// facing memory and an empty death clock, exactly newMapWorld's before its
// tick-0 push. The struct literal is deliberate: these call sites read a
// derivation, not a screen, so no viewer and no push belong here, and the
// seam's own method stays the ONE spelling of resolution this file
// exercises. Every memory the derivation WRITES has to be present, which is
// why the list grows when the seam gains one.
func seamDraws(w *sim.World, set *terrain.UnitSet) []ui.MapEntity {
	return (&mapWorld{world: w, units: set,
		swing: make(map[sim.EntityID]int),
		died:  make(map[sim.EntityID]int)}).entityDraws()
}

// entityCells reads a world's entities back as bare cells, through the
// production derivation itself — the seam under no bundle — so this file
// holds no second copy of the entity-to-cell convention: the expected values
// are the literals above, and the read path is the one the screen uses.
func entityCells(w *sim.World) []image.Point {
	draws := seamDraws(w, nil)
	cells := make([]image.Point, len(draws))
	for i, d := range draws {
		cells[i] = d.Cell
	}
	return cells
}

// assertCells compares a cell list against expected values, naming the label of
// whatever produced it.
func assertCells(t *testing.T, label string, got, want []image.Point) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d cells %v, want %d %v", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: cell %d = %v, want %v (all cells: %v, want %v)", label, i, got[i], want[i], got, want)
		}
	}
}

// someEntityHasMoved reports whether any entity stands somewhere other than the
// cell it started on. Every digest comparison below calls it, because a
// comparison made before any entity has moved is a comparison of two untouched
// worlds and would pass over a driver that applied no command at all.
func someEntityHasMoved(start, now []image.Point) bool {
	for i := range now {
		if i < len(start) && now[i] != start[i] {
			return true
		}
	}
	return false
}

// TestOpenMapWorldStandsAtTickZeroOverTheMapsOwnUnits — 0020 SC-2 (AC-1):
// opening a map builds one world at tick 0, one entity per placed unit,
// bounded by the map's own extent, each entity on its unit's cell — and
// the viewer ALREADY HOLDS those cells before any tick runs.
func TestOpenMapWorldStandsAtTickZeroOverTheMapsOwnUnits(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)

	// The bundle rides the production route: the tick-0 push resolves against
	// it, exactly as a front-end's does.
	mw := mustOpenMapWorld(t, m, nil, worldFixtureUnitSet(), v)

	if got := mw.world.Tick(); got != 0 {
		t.Errorf("a freshly opened world stands at tick %d, want 0", got)
	}
	if got, want := len(mw.world.Entities()), len(m.Units); got != want {
		t.Errorf("the world holds %d entities, want one per placed unit (%d)", got, want)
	}
	if got, want := mw.world.Bounds(), (sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}); got != want {
		t.Errorf("bounds = %+v, want the map's extent %+v", got, want)
	}
	assertCells(t, "the opened world's entity cells", entityCells(mw.world), worldFixtureCells)

	if got, want := v.EntityMarkers(), len(m.Units); got != want {
		t.Errorf("the viewer holds %d entity cells before the first tick, want %d — the tick-0 push is what "+
			"makes the frame a map opens on show its entities at all (DD-2)", got, want)
	}
	// THE OPENED WORLD CARRIES NO SCHEDULE, and this assertion is the reverse
	// of the one it replaces. A schedule arriving here again is that
	// regression, so it is pinned rather than merely absent.
	if got := len(mw.sched); got != 0 {
		t.Errorf("the opened world carries a %d-entry schedule; the placeholder walkabout is out of the game", got)
	}
}

// TestOpeningTheSameMapAgainBuildsAFreshWorld — 0020 SC-2 (AC-4): a map
// opened, ticked and opened again yields a SECOND world at tick 0 holding
// the entities the first one started from, and the first world is untouched
// by the second opening.
//
// The screen half of this — that leaving by Esc drops the viewer and the tick
// together, and that choosing again calls the loader a second time — is the UI
// tier's, pinned in its own flow tests over a stub loader (SC-1). What is here
// is what that tier cannot see: what the second call actually builds.
func TestOpeningTheSameMapAgainBuildsAFreshWorld(t *testing.T) {
	m := worldFixtureMap()

	first := mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m))
	// The walkabout is test support now (schedule_test.go) and no longer
	// something opening a map does. It is handed in here because THIS test needs
	// the first world to have moved: what it compares is a second opening
	// against the first world's START cells, and two worlds that both stand
	// still compare equal for the wrong reason.
	first.sched = placeholderSchedule(m)
	start := entityCells(first.world) // taken BEFORE the first world is ticked
	assertCells(t, "the first world at tick 0", start, worldFixtureCells)

	const ticks = 12
	for i := 0; i < ticks; i++ {
		first.tick()
	}
	if got := first.world.Tick(); got != ticks {
		t.Fatalf("the first world stands at tick %d after %d drives, want %d", got, ticks, ticks)
	}
	if !someEntityHasMoved(start, entityCells(first.world)) {
		t.Fatalf("no entity moved in %d ticks (%v) — the fixture makes the comparison below vacuous; pick another count",
			ticks, entityCells(first.world))
	}

	second := mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m))

	if second.world == first.world {
		t.Fatal("opening again handed back the SAME world; FR-1 requires a fresh one")
	}
	if got := second.world.Tick(); got != 0 {
		t.Errorf("the second world stands at tick %d, want 0", got)
	}
	assertCells(t, "the second world's entity cells", entityCells(second.world), start)

	if got := first.world.Tick(); got != ticks {
		t.Errorf("opening a second world moved the first to tick %d, want it left at %d", got, ticks)
	}
}

// TestScheduleEntryIsAppliedAtItsOwnTickAndNothingPastTheEnd — 0020 SC-3
// (AC-2): over a schedule written out by hand, the entry for tick t is
// applied at t and at no other tick, and past the schedule's end no command
// is applied at all.
//
// The schedule's last entry is a RETURN LEG on purpose. Past the end a world
// that re-applied entries — wrapping round to the front, say — would send an
// entity out again from the cell it just came home to, which is exactly what
// the two ticks after the end assert does not happen. A schedule whose last
// entry merely repeated its own target would be idempotent and could not tell
// the two apart.
func TestScheduleEntryIsAppliedAtItsOwnTickAndNothingPastTheEnd(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)

	// Entities start at (21,23), (25,20), (22,27), (31,24). The schedule's four
	// entries sit one CELL apart rather than one tick apart: a unit these maps
	// place resolves to no definition, takes the constructor's default speed and
	// crosses a straight cell in cellTicks ticks, and an order arriving inside a
	// crossing waits for it. Written a tick apart the last three entries would
	// all land on a mover that could not act on them, and this test would be
	// asserting that a schedule reaches a world that is standing still.
	sched := make([][]sim.Command, 3*cellTicks+1)
	sched[0*cellTicks] = []sim.Command{{Entity: 0, X: 24, Y: 23}} // entity 0 sets out east
	sched[1*cellTicks] = []sim.Command{{Entity: 3, X: 28, Y: 24}} // entity 3 sets out west
	sched[2*cellTicks] = nil                                      // nothing
	sched[3*cellTicks] = []sim.Command{{Entity: 3, X: 31, Y: 24}} // entity 3 turns for home

	mw := newMapWorld(mapload.FromALM(m), sched, nil, v)

	// One row per CELL, hand-walked from Step's contract: the block's first tick
	// applies that block's commands and every entity free to move takes one cell
	// toward its target, and the rest of the block is the crossing it owes. The
	// last two rows are past the schedule's end.
	want := [][]image.Point{
		{{X: 22, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 31, Y: 24}}, // entry 0 applied
		{{X: 23, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 30, Y: 24}}, // entry 1 applied
		{{X: 24, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 29, Y: 24}}, // entry 2 names none; both walk on
		{{X: 24, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 30, Y: 24}}, // entry 3 applied; entity 0 stands, arrived
		{{X: 24, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 31, Y: 24}}, // past the end: entity 3 walks on to its target
		{{X: 24, Y: 23}, {X: 25, Y: 20}, {X: 22, Y: 27}, {X: 31, Y: 24}}, // past the end: everything stands
	}

	for block, cells := range want {
		for i := 0; i < cellTicks; i++ {
			mw.tick()
		}
		if got, want := mw.world.Tick(), uint64((block+1)*cellTicks); got != want {
			t.Fatalf("after driving %d times the world stands at tick %d", want, got)
		}
		assertCells(t, "after the cell indexed "+strconv.Itoa(block), entityCells(mw.world), cells)
	}
}

func TestDrivenWorldReachesTheDigestOfAHeadlessRun(t *testing.T) {
	m := worldFixtureMap()
	// The walkabout is TEST SUPPORT now and no longer something a map screen
	// does (schedule_test.go). This test still needs it: what it witnesses is
	// that a renderer changes no simulated state, and two worlds where nothing
	// moves witness that vacuously.
	sched := placeholderSchedule(m)

	// One tick past the last index the schedule fills, so the far side of
	// "no commands past the end" is exercised by the digest as well.
	pastTheEnd := len(sched) + 1
	if 10 >= len(sched) {
		t.Fatalf("the fixture schedule holds %d entries, so k=10 is not mid-schedule; pick another k", len(sched))
	}

	for _, k := range []int{10, pastTheEnd} {
		t.Run("k="+strconv.Itoa(k), func(t *testing.T) {
			v := worldFixtureViewer(t, m)
			mw := mustOpenMapWorld(t, m, nil, worldFixtureUnitSet(), v)
			// Handed in, where openMapWorld used to build one itself.
			mw.sched = sched

			// The headless side: a FRESH world of the same map, advanced in
			// lockstep by the package that owns the run loop — continuing an
			// interrupted run is expressed by slicing the schedule, which is
			// sim.Run's own contract — with nothing drawing it.
			fresh := mapload.FromALM(m)
			freshSched := placeholderSchedule(m)

			sawMirror := false
			idleFrames := make(map[*terrain.StaticFrame]bool)

			for i := 0; i < k; i++ {
				mw.tick()
				if got, want := v.EntityMarkers(), len(m.Units); got != want {
					t.Fatalf("after tick %d the viewer holds %d entity cells, want %d — the driven side must "+
						"carry a renderer for this comparison to witness P-1", i, got, want)
				}

				sim.Run(fresh, freshSched[i:], 1)
				if got, want := mw.world.Hash(), fresh.Hash(); got != want {
					t.Fatalf("driven digest %#x != headless digest %#x after tick %d — render-side state "+
						"reached the world (P-4)", got, want, i+1)
				}

				// What the viewer holds is the seam's own slice, one call
				// apart (the push adopted its twin this very tick; the
				// derivation is idempotent over the seam's state).
				draws := mw.entityDraws()
				if draws[0].Frame == nil || draws[2].Frame == nil {
					t.Fatalf("after tick %d the mover holds frame %p and the idler %p — a resolving bundle "+
						"must keep BOTH holding sprites mid-run, or this digest comparison is vacuous (DD-7)",
						i+1, draws[0].Frame, draws[2].Frame)
				}
				sawMirror = sawMirror || draws[0].Mirror
				idleFrames[draws[2].Frame] = true
			}

			if !sawMirror {
				t.Errorf("the Flip-1 mover never crossed mirrored in %d ticks — selection is not demonstrably "+
					"live over this run (DD-7)", k)
			}
			if len(idleFrames) < 2 {
				t.Errorf("the idle-cycle entity held %d distinct frame(s) over %d ticks, want at least 2 — "+
					"a dead scene clock would look exactly like this (DD-7)", len(idleFrames), k)
			}

			if got, want := mw.world.Tick(), uint64(k); got != want {
				t.Fatalf("the driven world stands at tick %d, want %d", got, want)
			}
			if got, want := fresh.Tick(), uint64(k); got != want {
				t.Fatalf("the headless world stands at tick %d, want %d", got, want)
			}
			if got, want := mw.world.Hash(), fresh.Hash(); got != want {
				t.Fatalf("driven digest %#x != headless digest %#x after %d ticks; driven cells %v, headless cells %v",
					got, want, k, entityCells(mw.world), entityCells(fresh))
			}

			// Non-vacuity, both halves: somebody moved, and the digest is not
			// the one an untouched world of this map carries.
			if !someEntityHasMoved(worldFixtureCells, entityCells(mw.world)) {
				t.Fatalf("no entity has left its start cell after %d ticks (%v): at this k the comparison above "+
					"is between two untouched worlds and says nothing — pick another k", k, entityCells(mw.world))
			}
			if untouched := mapload.FromALM(m); mw.world.Hash() == untouched.Hash() {
				t.Fatalf("the driven digest equals a never-stepped world's, so the comparison discriminates nothing")
			}
		})
	}
}

// TestEntityZeroWalksItsFirstLeg — 0020 SC-4's "a k by which an entity has
// moved", stated as the cell rather than as an inequality.
//
// Entity 0 starts on (21,23) and its first order — the only one the schedule
// names at tick 0 — sends it east to (45,23), a leg of the full 24 cells with
// room to spare on the roomier side. Ten CELLS is ten times what a cell costs
// this fixture's units, so this names where it must be, and a driver that
// applied the schedule one tick late, or not at all, lands somewhere else.
func TestEntityZeroWalksItsFirstLeg(t *testing.T) {
	m := worldFixtureMap()
	// The map's OWN schedule gives entity 0 exactly this order at tick 0, and it
	// is written out here rather than taken from it because the schedule's legs
	// are spaced 24 ticks apart — the ticks a full leg took while a cell cost
	// one. A unit now takes its next corner before finishing a leg, so a
	// ten-cell walk driven by the placeholder would be measuring the placeholder
	// rather than the driver. What this test is about is that the entry is
	// applied at tick 0 and walked, which one entry says as well as the script.
	sched := [][]sim.Command{{{Entity: 0, X: 45, Y: 23}}}
	mw := newMapWorld(mapload.FromALM(m), sched, nil, worldFixtureViewer(t, m))

	const legTurnTicks = 4
	for i := 0; i < legTurnTicks+9*uphillCellTicks+1; i++ {
		mw.tick()
	}

	cells := entityCells(mw.world)
	if want := (image.Point{X: 31, Y: 23}); cells[0] != want {
		t.Fatalf("entity 0 stands on %v after 10 cells' worth of ticks, want %v (it starts on %v)",
			cells[0], want, worldFixtureCells[0])
	}
}

// TestEntityStandsOnItsOwnUnitsMarkerCellAtTickZero — 0020 SC-8 (AC-8): at
// tick 0 the cell of the entity built from a placed unit equals the cell
// that unit's diagnostic cross is drawn on, and on screen the cross's arm
// centre falls INSIDE the entity's square — in flat and displaced mode, at
// two camera positions.
//
// The two cells reach this test from packages that do not import each other:
// the entity's through mapload, the cross's through the render tier's
// AnchorCell, and both are compared against the literals at the top of this
// file rather than against each other. Nothing in pkg/game shifts an anchor.
func TestEntityStandsOnItsOwnUnitsMarkerCellAtTickZero(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; the lift and geometry literals below are stated over 32-pixel cells", terrain.CellSize)
	}

	m := worldFixtureMap()

	// The two derivations, each against the literals and never against each
	// other.
	world := mapload.FromALM(m)
	entities := entityCells(world)
	_, units := MarkerCells(m)
	assertCells(t, "the world's entity cells", entities, worldFixtureCells)
	assertCells(t, "the unit overlay's marker cells", units, worldFixtureCells)

	// An independent projection over a COPY of the fixture's altitude bytes, so
	// the expected lift shares nothing with the viewer's own.
	proj := terrain.Project(worldFixtureAltitudes(), worldFixtureW, worldFixtureH)

	lifted := 0
	for _, c := range worldFixtureCells {
		if -proj.AnchorHeight(c.X, c.Y)-proj.MinV != 0 {
			lifted++
		}
	}
	if lifted == 0 {
		t.Fatal("every fixture cell lifts by zero, so displaced mode below is indistinguishable from flat")
	}

	modes := []struct {
		name string
		flat bool
	}{{"displaced", false}, {"flat", true}}
	positions := []struct {
		name       string
		zoom       float64
		panX, panY float64
	}{
		{"origin at native zoom", 1, 0, 0},
		{"panned and zoomed in", 2, 137, 91},
	}

	for _, mode := range modes {
		for _, pos := range positions {
			t.Run(mode.name+", "+pos.name, func(t *testing.T) {
				v := worldFixtureViewer(t, m)
				v.Layout(200, 150)
				if mode.flat {
					v.SetFlat(true)
				}
				if got, want := v.Mode() == ui.ModeFlat, mode.flat; got != want {
					t.Fatalf("viewer is in flat mode = %v, want %v", got, want)
				}

				cam := v.Camera()
				cam.SetZoom(pos.zoom)
				cam.Pan(pos.panX, pos.panY)
				if cam.Zoom != pos.zoom || cam.X != pos.panX || cam.Y != pos.panY {
					t.Fatalf("camera is at (%v,%v) zoom %v, want (%v,%v) zoom %v — the clamp moved it; pick another position",
						cam.X, cam.Y, cam.Zoom, pos.panX, pos.panY, pos.zoom)
				}

				// The viewer really holds both layers: the count is all this tier can
				// read back, and it is what says the cells arrived.
				v.SetUnits(true, units)
				draws := make([]ui.MapEntity, len(entities))
				for i, c := range entities {
					draws[i] = ui.MapEntity{Cell: c}
				}
				v.SetEntities(draws)
				if on, n := v.UnitOverlay(); !on || n != len(units) {
					t.Fatalf("unit overlay = (on %v, %d cells), want (on true, %d cells)", on, n, len(units))
				}
				if n := v.EntityMarkers(); n != len(entities) {
					t.Fatalf("the viewer holds %d entity cells, want %d", n, len(entities))
				}

				for i := range worldFixtureCells {
					e, u := entities[i], units[i]

					// Each side carries the lift of ITS OWN cell, so a
					// disagreement about the cell shows here too and does not
					// cancel out of the comparison.
					eLift, uLift := 0, 0
					if !mode.flat {
						eLift = -proj.AnchorHeight(e.X, e.Y) - proj.MinV
						uLift = -proj.AnchorHeight(u.X, u.Y) - proj.MinV
					}

					sq := terrain.EntityMarkerRects(e.X, e.Y, m.Width, m.Height, terrain.CellSize)
					if len(sq) != 1 {
						t.Fatalf("entity %d on %v yields %d squares, want exactly one", i, e, len(sq))
					}
					sx, sy := cam.WorldToScreen(float64(sq[0].Min.X), float64(sq[0].Min.Y+eLift))
					sw, sh := float64(sq[0].Dx())*cam.Zoom, float64(sq[0].Dy())*cam.Zoom

					cx, cy := terrain.MarkerAnchor(u.X, u.Y, terrain.CellSize, 0, uLift)
					px, py := cam.WorldToScreen(float64(cx), float64(cy))

					if px < sx || px >= sx+sw || py < sy || py >= sy+sh {
						t.Errorf("unit %d's cross centre (%v,%v) lies outside entity %d's square [%v,%v)x[%v,%v): "+
							"the entity stands away from its own cross (FR-8)",
							i, px, py, i, sx, sx+sw, sy, sy+sh)
					}
				}
			})
		}
	}
}

// TestEntityDrawsHandArtExactlyWhereTheBundleHoldsAFrame — 0022 SC-7's
// push clauses and 0024 SC-5's square half (AC-4): what push hands the
// viewer IS the seam's slice, one call apart, so this pins the resolution at
// its seam. Art is the bundle's OWN ENTRY — pointer identity, the one the
// texture cache downstream keys by — exactly where that entry holds a
// frame; a frameless entry (an excluded class — AC-4's class with no art),
// an id naming no class (AC-4's other square) and a nil bundle all cross as
// nil art AND nil frame, because at this seam every one of them draws the
// same square; the cells ride beside unchanged; and resolving reads the
// world without touching it.
//
// Since 0024 T5 the frame beside the art is the LIVE tick-0 selection, and
// the expected indices are hand-walked from the spec, never captured: a
// freshly opened screen stands at scene 0 with an empty facing memory, so
// every entity is idle at octant 0. Classes 3 (S 9, no idle cycle), 7 and 34
// (zero descriptors) stand — g = 2*0 = 0, frame 0, plain. Class 21 plays its
// idle cycle: TailBase 16 + slot 0*2 + IdleTrack[(scene 0 + id 2) mod 2 = 0]
// = 16, plain. The 16 is the discriminating value — the T4 interim wired
// Frames[0] for every entity, and this is the assertion that buried it.
func TestEntityDrawsHandArtExactlyWhereTheBundleHoldsAFrame(t *testing.T) {
	m := worldFixtureMap()
	world := mapload.FromALM(m)
	before := world.Hash()

	full := worldFixtureUnitSet()
	draws := seamDraws(world, full)
	if len(draws) != len(worldFixtureUnits) {
		t.Fatalf("the seam handed %d entities, want one per placed unit (%d)", len(draws), len(worldFixtureUnits))
	}
	// Hand-walked above, one per entity, at octant 4 — NORTH, which is what a unit
	// that has never turned faces since 0081; these read octant 0, south, before
	// it, and every one of them moved. Class 3 stands at S 9: g = 2*4 = 8, which
	// is not past 8, so frame 8 plain. Classes 7 and 34 hold ONE frame each, so
	// the standing index 8 is outside their own sheets and each selector's guard
	// answers frame 0 — the same square they drew before, reached by the guard
	// rather than by the arithmetic. Class 21 plays its idle cycle: TailBase 16 +
	// slot 4*2 + IdleTrack[(scene 0 + id 2) mod 2 = 0] = 24.
	tickZeroFrames := []int{8, 0, 24, 0}
	for i, d := range draws {
		if d.Cell != worldFixtureCells[i] {
			t.Errorf("entity %d crossed on cell %v, want %v — resolution may add art and change nothing else", i, d.Cell, worldFixtureCells[i])
		}
		want := full.Classes[int32(worldFixtureUnits[i].ClassID)]
		if d.Art != want {
			t.Errorf("entity %d's art = %p, want the bundle's own entry %p for class %d",
				i, d.Art, want, worldFixtureUnits[i].ClassID)
		}
		if wantFrame := want.Frames[tickZeroFrames[i]]; d.Frame != wantFrame {
			t.Errorf("entity %d's frame = %p, want the entry's own Frames[%d] %p — the tick-0 idle selection "+
				"at octant 4 (AC-4, FR-4)", i, d.Frame, tickZeroFrames[i], wantFrame)
		}
		if d.Mirror {
			t.Errorf("entity %d crossed mirrored; a D-8 layout never mirrors", i)
		}
	}

	// The partial bundle: entity 0 keeps its frame; entity 1's class holds a
	// FRAMELESS entry — the excluded class; entities 2 and 3 name no class in
	// this bundle at all. The last three all cross as nil.
	partial := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{
		3: full.Classes[3],
		7: {Width: 31, Height: 31, CenterX: 15, CenterY: 29}, // no Frames
	}}
	draws = seamDraws(world, partial)
	if want := partial.Classes[3]; draws[0].Art != want {
		t.Errorf("entity 0's art = %p under the partial bundle, want %p", draws[0].Art, want)
	}
	for i, label := range map[int]string{1: "a frameless entry", 2: "an id naming no class", 3: "an id naming no class"} {
		if draws[i].Art != nil || draws[i].Frame != nil {
			t.Errorf("entity %d (%s) crossed with art %p frame %p, want both nil — the square is its draw (FR-4)",
				i, label, draws[i].Art, draws[i].Frame)
		}
	}

	// A nil bundle — the hand-assembled front-end — pushes every entity
	// art-less: byte for byte the screen this story inherits.
	for i, d := range seamDraws(world, nil) {
		if d.Art != nil {
			t.Errorf("entity %d crossed with art %p under a nil bundle, want every entity art-less", i, d.Art)
		}
		if d.Cell != worldFixtureCells[i] {
			t.Errorf("entity %d crossed on cell %v under a nil bundle, want %v", i, d.Cell, worldFixtureCells[i])
		}
	}

	if got := world.Hash(); got != before {
		t.Errorf("resolving moved the world's digest %#x -> %#x; resolution reads a world and never writes one (FR-6)",
			before, got)
	}
}

// TestEntityIdsCrossTheSeamAsTheirOwnIds — 0028 T2's id witness: the id
// the window tier receives is the SIMULATION'S OWN, carried across
// entityDraws unchanged, and never the position the entity happened to
// occupy in the snapshot.
//
// It is witnessed AT THE SEAM — the production derivation over a real world —
// because a hand-built []ui.MapEntity would only witness the test's own literal.
// The world is built here rather than out of the fixture map deliberately: the
// map's own convention makes an entity's id its unit-slice index, so on that
// fixture id and position agree at every entity and a derivation that answered
// with either would pass. These ids are SPARSE, NON-ADJACENT and given out of
// order, so an index disagrees at all three, and one of them is past 255, so a
// narrower conversion would show as well.
func TestEntityIdsCrossTheSeamAsTheirOwnIds(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 9, X: 1, Y: 1, Class: 3},
		{ID: 4, X: 2, Y: 2, Class: 7},
		{ID: 258, X: 3, Y: 3, Class: 21},
	})
	if err != nil {
		t.Fatalf("sim.NewWorld: %v", err)
	}

	draws := seamDraws(w, worldFixtureUnitSet())

	// Entities() hands them back in ascending id, so this is the order the
	// window tier sees them in — and at no position does it equal the index.
	want := []uint32{4, 9, 258}
	if len(draws) != len(want) {
		t.Fatalf("the seam handed %d entities, want %d", len(draws), len(want))
	}
	for i, d := range draws {
		if d.ID != want[i] {
			t.Errorf("entity at position %d crossed with id %d, want %d (the index there is %d)",
				i, d.ID, want[i], i)
		}
	}
}

func TestSeamDrawsFrameZeroUnmirroredWhenTheSheetFallsShort(t *testing.T) {
	m := worldFixtureMap()
	short := worldFixtureArt(64, 64, 32, 60, 10, 6, 3)
	short.Anim = worldFixtureMoverAnim()
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{3: short}}

	// One order, at tick 0: entity 0 (class 3) east across the map — a leg it
	// cannot finish in one tick, so the push after that tick observes it
	// moving, target still standing.
	sched := [][]sim.Command{{{Entity: 0, X: 31, Y: 23}}}
	mw := newMapWorld(mapload.FromALM(m), sched, set, worldFixtureViewer(t, m))
	mw.tick()

	draws := mw.entityDraws()
	if draws[0].Art != short {
		t.Fatalf("entity 0's art = %p, want the short-sheet entry %p — the guard degrades the frame, never the sprite (AC-4)",
			draws[0].Art, short)
	}
	if draws[0].Frame != short.Frames[0] {
		t.Errorf("entity 0's frame = %p, want the entry's own Frames[0] %p — an out-of-sheet selection draws "+
			"sheet frame 0 (AC-4, FR-6)", draws[0].Frame, short.Frames[0])
	}
	if draws[0].Mirror {
		t.Errorf("entity 0 crossed mirrored; the guard resets the mirror bit with the frame it refuses (AC-4)")
	}
}

// TestIdleEntityKeepsItsLastOctantPerId — 0024 SC-5's memory half (AC-5):
// an entity that walked and stopped keeps its last octant while idle; one
// that never moved faces octant 0; and the memories are independent per id
// — two walkers stopped facing different ways hold different octants while
// the never-moved beside them still hold 0.
//
// THE OCTANT IS READ THROUGH THE SEAM, off which frame crosses it: every
// class in this bundle carries an octant-legible idle cycle — S 16, D 8,
// every phase scalar 0 but ID 1, so every base is 16, IdleSlot is 1 and the
// idle track is [0] — under which an idle entity's frame is Frames[16 + oct]
// by the spec's idle formula, tick-insensitive (any tick mod period 1 steps
// to 0, so the id de-sync cancels out of this test by construction) and
// never mirrored (D 8). The sheets hold the predicted 16 + 8*1 = 24 frames.
//
// The walks: entity 0 to (24,20) — dx +3, dy -3 from (21,23), the NE
// sign-octant 5; entity 1 to (23,22) — dx -2, dy +2 from (25,20), SW, octant 1.
// Both legs span MORE than one tick, deliberately: memory is written only from an
// observed mover, and an arrival clears the target inside the very step that
// reaches it, so a one-tick leg would never be seen moving at all. Entities
// 2 and 3 are never ordered. The read comes three idle pushes after the
// later arrival: an idle push that erased or rewrote memory would surface
// here as octant 0, and a memory shared across ids as one octant everywhere.
func TestIdleEntityKeepsItsLastOctantPerId(t *testing.T) {
	octAnim := terrain.UnitAnim{S: 16, D: 8,
		MoveBase: 16, AttackBase: 16, DyingBase: 16, TailBase: 16,
		IdleSlot: 1, Total: 24, IdleTrack: []int{0}, IdleOK: true}
	classes := map[int32]*terrain.UnitClass{}
	for _, id := range []int32{3, 7, 21, 34} {
		c := worldFixtureArt(16, 16, 8, 14, 5, 4, 24)
		c.Anim = octAnim
		classes[id] = c
	}

	m := worldFixtureMap()
	sched := [][]sim.Command{{
		{Entity: 0, X: 24, Y: 20}, // NE, arrives after 3 cells
		{Entity: 1, X: 23, Y: 22}, // SW, arrives after 2 cells
	}}
	mw := newMapWorld(mapload.FromALM(m), sched, &terrain.UnitSet{Classes: classes},
		worldFixtureViewer(t, m))

	// Both legs are diagonal, so each cell costs the longer of the two counts.
	// Five cells' worth carries the longer leg home with two spare, which is
	// what the idle reads below need.
	for i := 0; i < 5*diagonalCellTicks; i++ {
		mw.tick()
	}

	// Both walkers stand on their own targets, which is what says they
	// arrived and were released: an entity on its target is idle by the
	// spec's classification, target cleared by the step that landed it.
	cells := entityCells(mw.world)
	if cells[0] != (image.Point{X: 24, Y: 20}) || cells[1] != (image.Point{X: 23, Y: 22}) {
		t.Fatalf("the walkers stand on %v and %v after five cells' worth of ticks, want (24,20) and "+
			"(23,22) — the reads below assume both legs are over", cells[0], cells[1])
	}

	draws := mw.entityDraws()
	// The two walkers keep the octant their last leg went in; the two that never
	// moved face NORTH, octant 4, where they faced south before 0081.
	wantOct := []int{5, 1, 4, 4}
	for i, d := range draws {
		cls := classes[int32(worldFixtureUnits[i].ClassID)]
		if want := cls.Frames[16+wantOct[i]]; d.Frame != want {
			t.Errorf("entity %d holds frame %p, want Frames[%d] %p — its idle facing must be octant %d (AC-5)",
				i, d.Frame, 16+wantOct[i], want, wantOct[i])
		}
		if d.Mirror {
			t.Errorf("entity %d crossed mirrored; a D-8 idle cycle never mirrors", i)
		}
	}
}

// TestSpriteGroundPointLandsOnItsUnitCrossAnchorAtTickZero — 0022 SC-6
// (AC-8): at tick 0, for every resolved entity, the sprite's ground point
// — top-left plus its own anchor, UnitPlace's Ground() — lands on the
// world point the unit's diagnostic cross is centred on, exactly, in screen
// pixels, both geometries, several cameras.
//
// THE TWO SIDES ARE THE TWO PRODUCTION DERIVATIONS, sharing only the camera
// transform. The sprite side: the entity's cell and art through the seam —
// mapload's world, the bundle's class geometry — placed by UnitPlace at
// the lift terms the entity layer passes, AnchorHeight and MinV displaced
// and 0 and 0 flat, both unnegated. The cross side: the unit record's cell
// through MarkerCells, centred by MarkerAnchor and lifted by the marker
// path's own -AnchorHeight - MinV, no class field, no frame size. The
// projection is an INDEPENDENT terrain.Project over a copy of the fixture's
// altitude bytes — never a viewer's own — and each side carries the lift
// of ITS OWN cell, so a wrong cell mapping does not cancel out of the
// comparison. The sprite's originY is MinV, so the two lift expressions
// cancel identically and the equality is exact through one WorldToScreen —
// no tolerance, no rounding of ours.
func TestSpriteGroundPointLandsOnItsUnitCrossAnchorAtTickZero(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; the fixture's geometry literals are stated over 32-pixel cells", terrain.CellSize)
	}

	m := worldFixtureMap()
	draws := seamDraws(mapload.FromALM(m), worldFixtureUnitSet())
	_, units := MarkerCells(m)
	if len(draws) != len(units) {
		t.Fatalf("%d entities against %d unit cells; the comparison below pairs them by index", len(draws), len(units))
	}
	for i, d := range draws {
		if d.Art == nil {
			t.Fatalf("entity %d does not resolve — AC-8 is stated over a map whose units resolve", i)
		}
	}

	// The independent projection, over a COPY of the fixture's altitude bytes,
	// and the non-vacuity guard: displaced must actually displace, unevenly.
	proj := terrain.Project(worldFixtureAltitudes(), worldFixtureW, worldFixtureH)
	lifted := 0
	for _, c := range worldFixtureCells {
		if -proj.AnchorHeight(c.X, c.Y)-proj.MinV != 0 {
			lifted++
		}
	}
	if lifted == 0 {
		t.Fatal("every fixture cell lifts by zero, so displaced mode below is indistinguishable from flat")
	}

	modes := []struct {
		name string
		flat bool
	}{{"displaced", false}, {"flat", true}}
	positions := []struct {
		name       string
		zoom       float64
		panX, panY float64
	}{
		{"origin at native zoom", 1, 0, 0},
		{"panned and zoomed in", 2, 137, 91},
		{"panned at a wheel-step zoom", 1.2, 60, 40},
	}

	for _, mode := range modes {
		for _, pos := range positions {
			t.Run(mode.name+", "+pos.name, func(t *testing.T) {
				cam := camera.New(worldFixtureW, worldFixtureH, 200, 150)
				cam.SetZoom(pos.zoom)
				cam.Pan(pos.panX, pos.panY)
				if cam.Zoom != pos.zoom || cam.X != pos.panX || cam.Y != pos.panY {
					t.Fatalf("camera is at (%v,%v) zoom %v, want (%v,%v) zoom %v — the clamp moved it; pick another position",
						cam.X, cam.Y, cam.Zoom, pos.panX, pos.panY, pos.zoom)
				}

				for i := range draws {
					e, u := draws[i].Cell, units[i]

					eLift, eOrigin, uLift := 0, 0, 0
					if !mode.flat {
						eLift = proj.AnchorHeight(e.X, e.Y)
						eOrigin = proj.MinV
						uLift = -proj.AnchorHeight(u.X, u.Y) - proj.MinV
					}

					p, ok := terrain.UnitPlace(e.X, e.Y, draws[i].Art, draws[i].Frame, draws[i].Mirror, eLift, eOrigin)
					if !ok {
						t.Fatalf("entity %d: UnitPlace refused a resolved entity's art", i)
					}
					g := p.Ground()
					sx, sy := cam.WorldToScreen(float64(g.X), float64(g.Y))

					cx, cy := terrain.MarkerAnchor(u.X, u.Y, terrain.CellSize, 0, uLift)
					px, py := cam.WorldToScreen(float64(cx), float64(cy))

					if sx != px || sy != py {
						t.Errorf("entity %d: sprite ground point (%v,%v) != its unit cross's anchor (%v,%v) — "+
							"the art stands away from its own cross (FR-5)", i, sx, sy, px, py)
					}
				}
			})
		}
	}
}

// TestPerturbingTheClassCentreMovesOnlyTheSprite — 0022 SC-6's
// perturbation half: with the class's centre perturbed, the drawn art MOVES
// — its placed top-left, on both axes — while the cross stands still and
// the two compared ground points still coincide with it.
//
// That last clause is the instrument's stated bound, asserted rather than
// left implicit: the anchor terms cancel out of Ground(), so the tick-0
// equality catches a wrong cell mapping, lift, sign or origin — those move
// one compared side alone — and CANNOT catch a wrong class geometry, which
// moves only the art.
func TestPerturbingTheClassCentreMovesOnlyTheSprite(t *testing.T) {
	m := worldFixtureMap()
	draws := seamDraws(mapload.FromALM(m), worldFixtureUnitSet())
	_, units := MarkerCells(m)
	proj := terrain.Project(worldFixtureAltitudes(), worldFixtureW, worldFixtureH)

	// Entity 0: cell (1,3), class 3 — and a camera away from the identity, so
	// "in screen pixels" is not "in world pixels" by accident.
	const i = 0
	e, u := draws[i].Cell, units[i]
	base := draws[i].Art
	cam := camera.New(worldFixtureW, worldFixtureH, 200, 150)
	cam.SetZoom(2)
	cam.Pan(137, 91)
	if cam.Zoom != 2 || cam.X != 137 || cam.Y != 91 {
		t.Fatalf("camera is at (%v,%v) zoom %v, want (137,91) zoom 2 — the clamp moved it", cam.X, cam.Y, cam.Zoom)
	}

	// The same class with its centre moved by (+3,+2) — asymmetric, so each
	// axis is observed on its own. Nothing else differs, frame included.
	perturbed := *base
	perturbed.CenterX += 3
	perturbed.CenterY += 2

	eLift, eOrigin := proj.AnchorHeight(e.X, e.Y), proj.MinV
	uLift := -proj.AnchorHeight(u.X, u.Y) - proj.MinV

	p0, ok0 := terrain.UnitPlace(e.X, e.Y, base, draws[i].Frame, false, eLift, eOrigin)
	p1, ok1 := terrain.UnitPlace(e.X, e.Y, &perturbed, draws[i].Frame, false, eLift, eOrigin)
	if !ok0 || !ok1 {
		t.Fatalf("UnitPlace refused a drawable class (base %v, perturbed %v)", ok0, ok1)
	}

	// The art moved, by exactly the perturbation, opposite in sign: anchor
	// +(3,2), top-left -(3,2). In screen pixels the drawn rectangle moved on
	// both axes.
	if got, want := p1.Anchor.Sub(p0.Anchor), image.Pt(3, 2); got != want {
		t.Errorf("the perturbed anchor moved by %v, want %v", got, want)
	}
	if got, want := p1.TopLeft.Sub(p0.TopLeft), image.Pt(-3, -2); got != want {
		t.Errorf("the perturbed top-left moved by %v, want %v — the art must stand away from its cross", got, want)
	}
	bx, by := cam.WorldToScreen(float64(p0.TopLeft.X), float64(p0.TopLeft.Y))
	qx, qy := cam.WorldToScreen(float64(p1.TopLeft.X), float64(p1.TopLeft.Y))
	if qx == bx || qy == by {
		t.Errorf("on screen the perturbed sprite stands at (%v,%v) against (%v,%v), want it moved on both axes",
			qx, qy, bx, by)
	}

	// The cross did not move — MarkerAnchor took no class field to perturb —
	// and BOTH ground points still land on it: the bound, stated exactly.
	cx, cy := terrain.MarkerAnchor(u.X, u.Y, terrain.CellSize, 0, uLift)
	px, py := cam.WorldToScreen(float64(cx), float64(cy))
	for _, side := range []struct {
		label string
		p     terrain.StaticPlacement
	}{{"base", p0}, {"perturbed", p1}} {
		g := side.p.Ground()
		sx, sy := cam.WorldToScreen(float64(g.X), float64(g.Y))
		if sx != px || sy != py {
			t.Errorf("%s: ground point (%v,%v) left the cross anchor (%v,%v) — the centre must cancel out of "+
				"Ground(), or the instrument would blame the class for what only the eye can judge",
				side.label, sx, sy, px, py)
		}
	}
}

// TestOnlyTheGameFrontEndBuildsAWorld — 0020 SC-9 (AC-10): the shared load
// path both entry points come through builds NO world — a viewer straight
// out of it reports no entity cells at all — while the game's own picker
// loader builds one per opening and hands back the tick that advances it.
//
// The fixture map carries units, which is what makes the first half say
// something: a viewer holding no entity cells over a map with three placed
// units cannot be one that quietly built a world and ignored it.
//
// This is the one test in this file over a map STREAM rather than a literal,
// because a viewer out of LoadMapViewer is what SC-9 asks for and that path
// takes bytes. The stream is the synthetic builder the other load tests use;
// nothing here reads a game install.
func TestOnlyTheGameFrontEndBuildsAWorld(t *testing.T) {
	units := []synth.ALMUnit{{X: 0x0500, Y: 0x0080}, {X: 0x0180, Y: 0x0300}, {X: 0x02ff, Y: 0x0000}}
	data := synth.ALM(synth.ALMOptions{Width: 6, Height: 5, Name: "Loaded", Units: units})

	t.Run("the shared load path owns no world", func(t *testing.T) {
		mv, err := LoadMapViewer(&terrain.Tileset{}, data, "x", Markers{Units: true}, StaticLayer{}, StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		if got := len(mv.Map.Units); got != len(units) {
			t.Fatalf("the fixture decoded %d units, want %d — with none the check below is vacuous", got, len(units))
		}
		if got := mv.Viewer.EntityMarkers(); got != 0 {
			t.Errorf("a viewer out of LoadMapViewer reports %d entity cells, want 0: the standalone viewer owns "+
				"no world, and a world it ignored would still be one it owned (FR-3, DD-2)", got)
		}
	})

	t.Run("the front-end's loader builds one per opening", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "a.alm"), data, 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Loose: looseOver(t, dir)}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "a.alm", Name: "Loaded"}}}, Presentation: Presentation{Markers: Markers{Units: true}}}

		v1, tick1, order1, cadence1, _, _, _, _, _, _, err := f.loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if tick1 == nil {
			t.Fatal("loadMap handed back no tick, so the map screen would advance nothing (FR-2)")
		}
		if order1 == nil {
			t.Fatal("loadMap handed back no order seam, so the map screen could issue nothing (0028 FR-4, DD-5)")
		}
		if cadence1 == nil {
			t.Fatal("loadMap handed back no cadence seam, so the map screen could re-rate nothing (0041 FR-5, DD-6)")
		}
		if got := v1.EntityMarkers(); got != len(units) {
			t.Fatalf("the opened viewer holds %d entity cells, want %d", got, len(units))
		}
		for i := 0; i < 10; i++ {
			tick1()
		}
		if got := v1.EntityMarkers(); got != len(units) {
			t.Errorf("after ten ticks the viewer holds %d entity cells, want %d every tick", got, len(units))
		}

		v2, tick2, order2, cadence2, _, _, _, _, _, _, err := f.loadMap(0)
		if err != nil {
			t.Fatalf("second loadMap: %v", err)
		}
		if tick2 == nil {
			t.Fatal("the second opening handed back no tick")
		}
		if order2 == nil {
			t.Fatal("the second opening handed back no order seam")
		}
		if cadence2 == nil {
			t.Fatal("the second opening handed back no cadence seam")
		}
		if v2 == v1 {
			t.Error("the second opening handed back the SAME viewer; each opening is a fresh map screen (FR-1)")
		}
		if got := v2.EntityMarkers(); got != len(units) {
			t.Errorf("the second viewer holds %d entity cells, want %d", got, len(units))
		}
	})

	t.Run("a row that will not decode builds nothing", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.alm"), data[:40], 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Loose: looseOver(t, dir)}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "bad.alm", Name: "Broken"}}}}
		v, tick, order, cadence, _, _, _, _, _, _, err := f.loadMap(0)
		if err == nil {
			t.Fatal("a truncated map loaded without error")
		}
		if v != nil || tick != nil || order != nil || cadence != nil {
			t.Errorf("a failed load handed back viewer %v, tick non-nil %v, order non-nil %v and cadence "+
				"non-nil %v, want none (P-4; 0028 DD-5; 0041 DD-6)", v, tick != nil, order != nil, cadence != nil)
		}
	})
}

// TestPacedAdvanceRunsAtTheDecodedLogicRate — 0024 SC-10 (AC-12): the
// advance fires at the decoded logic rate rather than at the front-end's
// frame rate, the catch-up bound holds, and a tick is the same tick whatever
// paced it.
//
// NO TEST HERE READS A CLOCK. paceTo takes its instant as an argument, so every
// elapsed run below is a literal and the whole cadence is decidable without a
// window — which is the reason the wall clock is read on one line (paced) that
// this file does not drive.
//
// The tick length is terrain's decoded value at the speed index map load
// selects, and the expectations are hand-computed from it in the comments
// beside them. It is asserted once, at the top, so a literal 62 below has a
// stated provenance rather than an assumed one.
//
// paceTo's SECOND result — how many pending orders the advance applied —
// is discarded throughout: nothing here enqueues one, and what this test is
// about is when a tick fires, not what it applies. The count's own witnesses
// are 0028's, further down this file.
func TestPacedAdvanceRunsAtTheDecodedLogicRate(t *testing.T) {
	const tickMs = 62

	if got := terrain.TickMillis(terrain.DefaultSpeedIndex); got != tickMs {
		t.Fatalf("the decoded tick is %d ms at the default speed index, not the %d ms every "+
			"expectation below is computed from", got, tickMs)
	}
	if got := terrain.TicksPerSecond(terrain.DefaultSpeedIndex); got != 16 {
		t.Fatalf("the decoded rate is %d ticks/s, not the 16 this criterion is stated over", got)
	}

	base := time.Unix(1_700_000_000, 0)
	ms := func(n int) time.Time { return base.Add(time.Duration(n) * time.Millisecond) }

	opened := func(t *testing.T) *mapWorld {
		t.Helper()
		m := worldFixtureMap()
		return mustOpenMapWorld(t, m, nil, worldFixtureUnitSet(), worldFixtureViewer(t, m))
	}

	// AC-12: the first call advances nothing. A map that took a second to open
	// must not spend that second's ticks on the frame it opened.
	t.Run("the first call only takes the baseline", func(t *testing.T) {
		mw := opened(t)
		if got, _ := mw.paceTo(base); got != 0 {
			t.Errorf("the first paced call ran %d ticks, want 0", got)
		}
		if mw.world.Tick() != 0 || mw.scene != 0 {
			t.Errorf("world tick %d, scene %d after the baseline call, want 0 and 0", mw.world.Tick(), mw.scene)
		}
		// ...and the second call is measured from that baseline, not from zero.
		if got, _ := mw.paceTo(ms(tickMs)); got != 1 {
			t.Errorf("one tick's worth of elapsed time ran %d ticks, want 1", got)
		}
	})

	// AC-12: under, at and over one tick, with the remainder carried.
	t.Run("elapsed under one tick fires nothing and is not lost", func(t *testing.T) {
		mw := opened(t)
		mw.paceTo(base)

		for _, at := range []int{30, 60} { // 30 ms, then 60 ms: still under 62
			if got, _ := mw.paceTo(ms(at)); got != 0 {
				t.Fatalf("at %d ms elapsed the pacing ran %d ticks, want 0", at, got)
			}
		}
		if got, _ := mw.paceTo(ms(62)); got != 1 { // the 2 ms that crosses it
			t.Errorf("the call crossing 62 ms ran %d ticks, want 1 — the earlier 60 ms were carried, not dropped", got)
		}
		if mw.world.Tick() != 1 || mw.scene != 1 {
			t.Errorf("world tick %d, scene %d, want 1 and 1", mw.world.Tick(), mw.scene)
		}
	})

	// AC-12 / SC-10: the RATE. Ten calls 100 ms apart are one second of
	// wall-clock time, and one second is 1000/62 = 16 whole ticks — not the ten
	// a per-call advance would run, and not the 60 a 60 fps front-end would.
	t.Run("one second of calls is one second of ticks", func(t *testing.T) {
		mw := opened(t)
		mw.paceTo(base)

		total := 0
		bound := maxCatchUp(mw.clock.Period())
		for i := 1; i <= 10; i++ {
			n, _ := mw.paceTo(ms(100 * i))
			if n > bound {
				t.Fatalf("call %d ran %d ticks, past the bound of %d", i, n, bound)
			}
			total += n
		}
		if total != 16 {
			t.Errorf("ten 100 ms calls ran %d ticks, want 16 = 1000/62 — the decoded rate, not the call rate", total)
		}
		// The scene clock counts TICKS, not calls: it is what the animation
		// selects on, and it is 16 here because 16 ticks ran, not 10 because 10
		// calls did.
		if mw.scene != total || mw.world.Tick() != uint64(total) {
			t.Errorf("scene %d and world tick %d after %d ticks in 10 calls, want both %d",
				mw.scene, mw.world.Tick(), total, total)
		}
	})

	// AC-12: a stall fires the bound and no more, and the time past it is
	// dropped rather than repaid on the calls that follow.
	t.Run("a stall fires the bound and repays nothing", func(t *testing.T) {
		mw := opened(t)
		mw.paceTo(base)

		bound := maxCatchUp(mw.clock.Period())
		if bound != 4 {
			t.Fatalf("the bound at the map-load period is %d ticks, want the shipped 4", bound)
		}
		if got, _ := mw.paceTo(ms(10_000)); got != bound {
			t.Errorf("a 10 s stall ran %d ticks, want the bound %d (unbounded it is 161)", got, bound)
		}
		// 10000 ms is 161 whole ticks with 18 ms left over. The next 100 ms call
		// sees 118 ms — one tick — and not the 157 a queued debt would hand back.
		if got, _ := mw.paceTo(ms(10_100)); got != 1 {
			t.Errorf("the call after the stall ran %d ticks, want 1 — dropped ticks are dropped, not queued", got)
		}
		if want := uint64(bound + 1); mw.world.Tick() != want {
			t.Errorf("the world stands at tick %d after the stall and one call, want %d", mw.world.Tick(), want)
		}
	})

	// SC-10: no elapsed time is lost below a millisecond either. A 60 fps
	// front-end calls every 16.67 ms, and truncating each call to 16 would run
	// the whole game 4% slow with nothing to point at.
	t.Run("sub-millisecond elapsed is carried, not truncated away", func(t *testing.T) {
		mw := opened(t)
		mw.paceTo(base)

		total := 0
		for i := 1; i <= 124; i++ { // 124 calls of 500 us = 62 ms = exactly one tick
			n, _ := mw.paceTo(base.Add(time.Duration(i) * 500 * time.Microsecond))
			total += n
		}
		if total != 1 {
			t.Errorf("124 calls of 500 us ran %d ticks, want 1 — each call truncates to 0 ms, and the "+
				"tail must stay on the clock", total)
		}
	})

	// AC-12's last clause, and the determinism wall: a paced run and a directly
	// driven one of the SAME tick count reach the same world, byte for byte.
	// Wall-clock decided when the ticks fired — two of them in one call, twice
	// over — and changed nothing about what any of them did.
	t.Run("k paced ticks reach the digest of k direct ticks", func(t *testing.T) {
		paced := opened(t)
		paced.paceTo(base)

		k := 0
		for i := 1; i <= 10; i++ {
			n, _ := paced.paceTo(ms(100 * i))
			k += n
		}
		if k != 16 {
			t.Fatalf("the paced run fired %d ticks; this comparison is written over 16", k)
		}

		direct := opened(t)
		untouched := direct.world.Hash()
		for i := 0; i < k; i++ {
			direct.tick()
		}

		if got, want := paced.world.Hash(), direct.world.Hash(); got != want {
			t.Errorf("paced digest %#x != directly driven digest %#x after %d ticks — wall-clock reached "+
				"what a tick DOES, not only when it fires (P-4)", got, want, k)
		}
		if direct.world.Hash() == untouched {
			t.Fatal("both digests equal a never-stepped world's, so the comparison discriminates nothing")
		}
		if paced.scene != direct.scene {
			t.Errorf("paced scene %d != driven scene %d", paced.scene, direct.scene)
		}
		assertCells(t, "the paced world's cells", entityCells(paced.world), entityCells(direct.world))
	})

	// A clock that went backwards must not rewind the cadence or fire on the
	// way back up. Unreachable through time.Now(), whose monotonic reading
	// cannot decrease — which is exactly why it is stated here instead.
	t.Run("a backwards clock advances nothing", func(t *testing.T) {
		mw := opened(t)
		mw.paceTo(ms(1000))
		if got, _ := mw.paceTo(ms(500)); got != 0 {
			t.Errorf("a backwards call ran %d ticks, want 0", got)
		}
		if got, _ := mw.paceTo(ms(1000 + tickMs)); got != 1 {
			t.Errorf("after the backwards call one tick's elapsed ran %d ticks, want 1 — the baseline "+
				"must not have been rewound", got)
		}
	})
}

// cadenceWorld is a map screen's world over the fixture's own placeholder
// schedule, re-rated to the given period and with its baseline already taken.
func cadenceWorld(t *testing.T, periodUS int, at time.Time) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))
	mw.setCadence(periodUS, false)
	if n, applied := mw.paceTo(at); n != 0 || applied != 0 {
		t.Fatalf("the baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}
	return mw
}

// TestAStoppedWorldRunsNoTickAndResumesWithExactlyOne — 0041 SC-3 (AC-3):
// over a stopped span of many frames and many seconds no tick fires and the
// tick count, the scene clock, the canonical byte form and the digest are
// identical at every one of those frames; clearing the stop changes no rate
// and the first advance after it runs ONE tick, not the backlog the span was
// worth.
//
// A SECOND LEG, never stopped, is driven over the very same instants. Without it
// "no tick fired" is a statement about a driver that might not have fired one
// anyway, and every equality above would be an equality between two untouched
// worlds.
func TestAStoppedWorldRunsNoTickAndResumesWithExactlyOne(t *testing.T) {
	const (
		periodUS = 62_500 // our rate model's 16/s — 1000000/16, not the decoded 62000
		frameUS  = 16_667 // a 60 fps front-end's own call gap
		frames   = 600    // 600 x 16667 us = 10.0002 s stopped
		// Six ticks before the stop, so the state held still below is one the
		// world WALKED to. Not eight: the script's four-tick laps put both
		// scripted entities back on their start cells at tick 8, and a stopped
		// comparison anchored there would be anchored on the constructor's own
		// picture after all.
		warmUp = 6
	)
	base := time.Unix(1_700_000_000, 0)
	period := time.Duration(periodUS) * time.Microsecond

	mw := cadenceWorld(t, periodUS, base)
	run := cadenceWorld(t, periodUS, base)
	if got := mw.clock.Period(); got != periodUS {
		t.Fatalf("the clock stands at %d us, want the %d every count below is computed from", got, periodUS)
	}

	now := base
	for i := 0; i < warmUp; i++ {
		now = now.Add(period)
		if n, _ := mw.paceTo(now); n != 1 {
			t.Fatalf("warm-up call %d ran %d ticks, want 1", i+1, n)
		}
		run.paceTo(now)
	}
	if !someEntityHasMoved(worldFixtureCells, entityCells(mw.world)) {
		t.Fatalf("no entity moved in the %d warm-up ticks, so the state held still below is the constructor's: %v",
			warmUp, entityCells(mw.world))
	}

	beforeTick, beforeScene := mw.world.Tick(), mw.scene
	beforeBytes, beforeHash := marshalWorld(t, mw.world), mw.world.Hash()
	beforePeriod := mw.clock.Period()
	beforeCells := entityCells(mw.world)

	mw.setCadence(periodUS, true)
	stoppedAt := now
	for i := 1; i <= frames; i++ {
		now = stoppedAt.Add(time.Duration(i*frameUS) * time.Microsecond)
		n, applied := mw.paceTo(now)
		run.paceTo(now)

		if n != 0 || applied != 0 {
			t.Fatalf("stopped frame %d ran %d ticks and applied %d orders, want 0 and 0", i, n, applied)
		}
		if mw.world.Tick() != beforeTick || mw.scene != beforeScene {
			t.Fatalf("stopped frame %d: world tick %d, scene %d, want %d and %d",
				i, mw.world.Tick(), mw.scene, beforeTick, beforeScene)
		}
		if !bytes.Equal(marshalWorld(t, mw.world), beforeBytes) {
			t.Fatalf("stopped frame %d moved the canonical byte form (FR-3)", i)
		}
		if got := mw.world.Hash(); got != beforeHash {
			t.Fatalf("stopped frame %d moved the digest %#x -> %#x (FR-3, P-4)", i, beforeHash, got)
		}
		assertCells(t, "the stopped world's cells", entityCells(mw.world), beforeCells)
	}

	// The instants really were worth something: the leg that was not stopped ran
	// over exactly them and moved.
	if run.world.Tick() <= beforeTick || run.world.Hash() == beforeHash {
		t.Fatalf("the unstopped leg stands at tick %d with digest %#x after the same %d frames, so the stopped "+
			"leg's stillness says nothing", run.world.Tick(), run.world.Hash(), frames)
	}

	// Clearing it changes no rate...
	mw.setCadence(periodUS, false)
	if got := mw.clock.Period(); got != beforePeriod {
		t.Errorf("the clock stands at %d us after the toggle, want the %d it held before it", got, beforePeriod)
	}
	// ...and owes nothing: one period of elapsed time is one tick, not the 160
	// the stopped span was worth.
	now = now.Add(period)
	if n, applied := mw.paceTo(now); n != 1 || applied != 0 {
		t.Errorf("the first advance after the stop ran %d ticks and applied %d orders, want 1 and 0 — a resume "+
			"must not unwind the paused span", n, applied)
	}
	if got, want := mw.world.Tick(), beforeTick+1; got != want {
		t.Errorf("the world stands at tick %d after the resume, want %d", got, want)
	}
}

// TestOnePacedCallIsBoundedByASpanOfWorldTimeAndOwesNothing — 0041 SC-6
// (AC-6): one paced call runs the bound's worth of WORLD TIME and no more,
// the same span at every rate the bound is longer than a tick at; at least
// one tick fires where it is shorter; and the surplus is dropped rather than
// owed to the call that follows.
//
// Reading it as a span and not as a count is the whole point, and the two are
// told apart HERE: rates sixteen times apart run sixteen times as many ticks in
// one call, and those two calls cover the same world time. A fixed count would
// give the same number at both.
func TestOnePacedCallIsBoundedByASpanOfWorldTimeAndOwesNothing(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	const stall = 10 * time.Second

	cases := []struct {
		rate      int // what the period names, for the message alone
		periodUS  int // 1000000/rate, truncated — hand-computed
		wantTicks int // max(1, 250000/period) — hand-computed
		floored   bool
	}{
		{rate: 16, periodUS: 62_500, wantTicks: 4},                  // 250000/62500 = 4
		{rate: 256, periodUS: 3_906, wantTicks: 64},                 // 250000/3906  = 64.0
		{rate: 1, periodUS: 1_000_000, wantTicks: 1, floored: true}, // 250000/1000000 = 0, floored to 1
	}

	spans := make(map[int]int, len(cases))
	for _, c := range cases {
		mw := cadenceWorld(t, c.periodUS, base)
		if got := mw.clock.Period(); got != c.periodUS {
			t.Fatalf("rate %d left the clock at %d us, want %d", c.rate, got, c.periodUS)
		}

		n, _ := mw.paceTo(base.Add(stall))
		if n != c.wantTicks {
			t.Errorf("rate %d: a %v stall ran %d ticks in one call, want %d", c.rate, stall, n, c.wantTicks)
			continue
		}
		spans[c.rate] = n * c.periodUS

		// The surplus is DROPPED. One further period of elapsed time is one
		// tick — not the hundreds a queued debt would hand back.
		if got, _ := mw.paceTo(base.Add(stall + time.Duration(c.periodUS)*time.Microsecond)); got != 1 {
			t.Errorf("rate %d: the call after the stall ran %d ticks, want 1 — dropped time is not owed", c.rate, got)
		}
		if got, want := mw.world.Tick(), uint64(c.wantTicks+1); got != want {
			t.Errorf("rate %d: the world stands at tick %d after the stall and one call, want %d", c.rate, got, want)
		}

		switch {
		case c.floored:
			if spans[c.rate] <= maxCatchUpMicros {
				t.Errorf("rate %d ran %d us in one call, which is inside the %d us bound — this case exists "+
					"because the bound is SHORTER than one tick there", c.rate, spans[c.rate], maxCatchUpMicros)
			}
		case spans[c.rate] > maxCatchUpMicros || maxCatchUpMicros-spans[c.rate] >= c.periodUS:
			t.Errorf("rate %d ran %d us in one call, want the whole periods that fit in %d us",
				c.rate, spans[c.rate], maxCatchUpMicros)
		}
	}

	// The bound is a SPAN, not a count: sixteen times the rate is sixteen times
	// the ticks over the same world time.
	if spans[16] == 0 || spans[256] == 0 {
		t.Fatal("a rate case did not run, so the comparison below has nothing to compare")
	}
	if spans[16] == spans[256] && cases[0].wantTicks == cases[1].wantTicks {
		t.Fatal("the two rates ran the same number of ticks; the bound is still a count")
	}
	if diff := spans[16] - spans[256]; diff >= cases[1].periodUS || diff < 0 {
		t.Errorf("one call covers %d us at rate 16 and %d us at rate 256; the two must be the same span to "+
			"within the coarser period", spans[16], spans[256])
	}
}

// orderTickMs is the decoded logic tick this file paces its advances by. Every
// advance below is one tick apart, so an advance and a tick are the same event
// and the applied count is reported per advance without a second cadence to
// reason about.
func orderTickMs(t *testing.T) int {
	t.Helper()
	ms := terrain.TickMillis(terrain.DefaultSpeedIndex)
	if ms <= 1 {
		t.Fatalf("the decoded tick is %d ms, which leaves no room for a sub-tick probe", ms)
	}
	return ms
}

// orderWorld is a map screen's world over an EMPTY script, with a viewer taking
// its cells and no unit bundle: the world AC-5 is stated over.
func orderWorld(t *testing.T) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	if len(mw.sched) != 0 {
		t.Fatalf("fixture: the script holds %d entries, want an empty one", len(mw.sched))
	}
	if len(mw.pending) != 0 {
		t.Fatalf("fixture: a fresh map world already holds %d pending orders", len(mw.pending))
	}
	return mw
}

// TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun — 0028
// SC-5 (AC-5): one order, issued between two advances over an empty script,
// is applied by EXACTLY ONE advance and by no later one — witnessed at
// every tick against a headless run of that same single command at the tick
// it drained.
//
// The headless side is the discriminator, and it is compared at every tick on
// the way rather than only at arrival: an order applied one advance early or
// late reaches the same cell in the end, and only the tick-by-tick comparison
// says WHEN it was applied. A re-application at a LATER tick is deliberately not
// what this test measures — under last-write a repeated identical order is
// invisible in world state, which is why the count, and not the digest, is where
// the queue's truncation is witnessed (SC-6, and SC-9's mutant).
func TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun(t *testing.T) {
	const (
		orderedEntity        = 0
		targetCol, targetRow = 31, 28 // ten cells east and five south, well inside the interior
		drainAt              = 2      // the advance the order is issued just before
		// The two crossing lengths, named here so the walk below reads as one
		// cell per entry: this fixture's units cross a straight cell in one and
		// a diagonal in the other.
		straightTick    = uphillCellTicks
		diagonalTick    = uphillDiagonalCellTicks
		legTurnTicks    = 6
		cornerTurnTicks = 1
	)

	base := time.Unix(1_700_000_000, 0)
	tickMs := orderTickMs(t)

	mw := orderWorld(t)
	m := worldFixtureMap()
	fresh := mapload.FromALM(m) // the headless side: same map, no viewer, no queue

	if got, want := mw.world.Hash(), fresh.Hash(); got != want {
		t.Fatalf("the two sides start at %#x and %#x — they must begin equal", got, want)
	}
	if worldFixtureCells[orderedEntity] != (image.Point{X: 21, Y: 23}) {
		t.Fatalf("the fixture starts entity %d on %v; the walk below is written from (21,23)",
			orderedEntity, worldFixtureCells[orderedEntity])
	}

	// The first paced call only takes the baseline and advances nothing.
	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("the baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}

	// One entry per advance at which the ordered entity's CELL CHANGES,
	// hand-walked from Step's contract: the order sets the target at the advance
	// it drains and the entity takes a cell then, and one further cell each time
	// it has paid for the last — a diagonal costs diagonalCellTicks and a
	// straight one cellTicks. Between two of these its cell does not move, and
	// the loop below asserts that too: it carries the last named cell forward
	// and checks every advance against it, so the coverage is one row per
	// advance as it was, written as the changes instead of as the repeats.
	walkAt := map[int]image.Point{
		1:                                       {X: 21, Y: 23}, // nothing scripted, nothing pending — it stands
		drainAt + legTurnTicks:                  {X: 22, Y: 24}, // the order drains at drainAt, pays its turn, then moves
		drainAt + legTurnTicks + 1*diagonalTick: {X: 23, Y: 25},
		drainAt + legTurnTicks + 2*diagonalTick: {X: 24, Y: 26},
		drainAt + legTurnTicks + 3*diagonalTick: {X: 25, Y: 27},
		drainAt + legTurnTicks + 4*diagonalTick: {X: 26, Y: 28}, // the row axis is home; the column walks on alone
		// The fifth diagonal step is what the unit pays for next, so the first
		// straight cell begins a DIAGONAL crossing later and not a straight one:
		// a crossing costs what the step that entered it cost. It also pays the
		// corner's own turn, cornerTurnTicks, once, on top of that crossing —
		// the direction changes here and nowhere else in the walk.
		drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 0*straightTick: {X: 27, Y: 28},
		drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 1*straightTick: {X: 28, Y: 28},
		drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 2*straightTick: {X: 29, Y: 28},
		drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 3*straightTick: {X: 30, Y: 28},
		drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 4*straightTick: {X: 31, Y: 28}, // arrived, and the arrival clears the target
	}
	// Two advances past the arrival, so the run still witnesses the unit
	// standing with nothing left pending.
	advances := drainAt + legTurnTicks + 5*diagonalTick + cornerTurnTicks + 4*straightTick + 2
	// The other three entities are named by nothing at all here, so every
	// assertion below is that entity's cell beside these three, unmoved.
	still := []image.Point{worldFixtureCells[1], worldFixtureCells[2], worldFixtureCells[3]}
	last := image.Point{X: 21, Y: 23}

	for i := 1; i <= advances; i++ {
		if i == drainAt {
			before := mw.world.Hash()
			beforeCells := entityCells(mw.world)
			mw.enqueue(orderedEntity, targetCol, targetRow)
			if got := mw.world.Hash(); got != before {
				t.Fatalf("enqueueing with no advance moved the digest %#x -> %#x — an order reached the "+
					"world outside an advance (P-1)", before, got)
			}
			assertCells(t, "after enqueueing with no advance", entityCells(mw.world), beforeCells)
			if len(mw.pending) != 1 {
				t.Fatalf("the queue holds %d orders after one enqueue, want 1", len(mw.pending))
			}
		}

		if n, _ := mw.paceTo(base.Add(time.Duration(i*tickMs) * time.Millisecond)); n != 1 {
			t.Fatalf("advance %d ran %d ticks; this run is written over one tick per advance", i, n)
		}

		// The headless side in lockstep: the SINGLE command, at the tick the
		// driven side drained it, and nothing at any other tick.
		var cmds []sim.Command
		if i == drainAt {
			// A GROUP order of one, which is what the seam issues: the identity
			// on the destination, and the tag reaches no world field.
			cmds = grouped([]sim.Command{{Entity: orderedEntity, X: targetCol, Y: targetRow}})
		}
		sim.Step(fresh, cmds)

		if got, want := mw.world.Hash(), fresh.Hash(); got != want {
			t.Fatalf("driven digest %#x != headless digest %#x after advance %d — the order did not apply "+
				"at exactly the advance that drained it (FR-4, P-6)", got, want, i)
		}
		if p, ok := walkAt[i]; ok {
			last = p
		}
		assertCells(t, "after advance "+strconv.Itoa(i),
			entityCells(mw.world), append([]image.Point{last}, still...))
	}

	// Non-vacuity, both halves: the ordered unit actually left its start cell,
	// and the digest the two sides agree on is not an untouched world's.
	if !someEntityHasMoved(worldFixtureCells, entityCells(mw.world)) {
		t.Fatalf("no entity has moved after %d advances — the comparison above is between two untouched worlds",
			advances)
	}
	if untouched := mapload.FromALM(m); mw.world.Hash() == untouched.Hash() {
		t.Fatalf("the driven digest equals a never-stepped world's, so the comparison discriminates nothing")
	}
	if got := len(mw.pending); got != 0 {
		t.Errorf("the queue holds %d orders after the run, want 0 — one order, one advance", got)
	}
}

// TestAnAdvanceReportsHowManyOrdersItApplied — 0028 SC-6 (AC-8): what an
// advance applied is reportable without a window, orders pending at one
// advance apply in issue order so the last issued wins, and the count is the
// queue's length at the tick that drained it and zero at every other.
//
// THE COUNT IS THE ONLY WITNESS THE TRUNCATION HAS, and that is stated
// rather than incidental: while move-to is the only kind of command, a batch
// applied twice is indistinguishable from one applied once — the second
// application writes the target the first already wrote — so no digest and
// no cell can tell the two apart.
func TestAnAdvanceReportsHowManyOrdersItApplied(t *testing.T) {
	const orderedEntity = 0

	base := time.Unix(1_700_000_000, 0)
	tickMs := orderTickMs(t)
	mw := orderWorld(t)

	// Each advance is one tick further on, so an advance and a tick coincide and
	// the count below is per advance with no second cadence in the way.
	advance := func(i int) (ticks, applied int) {
		t.Helper()
		return mw.paceTo(base.Add(time.Duration(i*tickMs) * time.Millisecond))
	}

	if n, applied := mw.paceTo(base); n != 0 || applied != 0 {
		t.Fatalf("the baseline call ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}

	// AC-8: a selected unit right-clicked on two different in-extent cells
	// BETWEEN two advances.
	mw.enqueue(orderedEntity, 21, 28) // due south
	mw.enqueue(orderedEntity, 31, 23) // due east — issued second

	if n, applied := mw.paceTo(base.Add(time.Duration(tickMs/2) * time.Millisecond)); n != 0 || applied != 0 {
		t.Fatalf("a sub-tick advance ran %d ticks and applied %d orders, want 0 and 0", n, applied)
	}
	if got := len(mw.pending); got != 2 {
		t.Fatalf("a sub-tick advance left %d orders queued, want the 2 it was handed", got)
	}

	// The first advance reports TWO.
	if n, applied := advance(1); n != 1 || applied != 2 {
		t.Fatalf("the first advance ran %d ticks and reported %d orders applied, want 1 and 2", n, applied)
	}
	// ...in issue order, so the unit turns toward the SECOND cell: it has not
	// yet paid the four-tick turn east costs, so the cell itself is still
	// (21,23) and the discriminator is the facing the order admitted.
	if cells := entityCells(mw.world); cells[orderedEntity] != (image.Point{X: 21, Y: 23}) {
		t.Fatalf("after the first advance the unit already stands on %v; this advance is measured before "+
			"any cell changes", cells[orderedEntity])
	}
	e := mw.world.Entities()[orderedEntity]
	if want := uint8(64); e.DesiredFacing != want {
		t.Fatalf("after the first advance the unit's desired facing is %d, want %d (east) — under the "+
			"first order it would be 128 (south), so the two were not applied in issue order",
			e.DesiredFacing, want)
	}
	if !e.HasTarget || e.TargetX != 31 || e.TargetY != 23 {
		t.Fatalf("the unit heads for (%d,%d) hasTarget=%v, want (31,23) — the last order issued must win",
			e.TargetX, e.TargetY, e.HasTarget)
	}

	// The NEXT advance reports zero: the batch was drained, not re-applied.
	if n, applied := advance(2); n != 1 || applied != 0 {
		t.Fatalf("the advance after the drain ran %d ticks and reported %d orders applied, want 1 and 0 — "+
			"an order must apply at exactly one advance and no later one (P-6)", n, applied)
	}
	if got := len(mw.pending); got != 0 {
		t.Fatalf("the queue holds %d orders after the drain, want 0", got)
	}

	// SC-6: a third advance after ONE further order reports one, so the count is
	// no constant — not always the first batch's size, and not always zero.
	mw.enqueue(orderedEntity, 25, 26)
	if n, applied := advance(3); n != 1 || applied != 1 {
		t.Fatalf("the third advance ran %d ticks and reported %d orders applied, want 1 and 1", n, applied)
	}

	mw.enqueue(orderedEntity, 28, 28)
	if n, applied := advance(6); n != 3 || applied != 1 {
		t.Fatalf("an advance worth three ticks ran %d and reported %d orders applied, want 3 and 1", n, applied)
	}
	if got := len(mw.pending); got != 0 {
		t.Errorf("the queue holds %d orders after the run, want 0", got)
	}
}

// scriptedLaps is the hand-written script this section measures over: entities 0
// and 1 given a fresh target every four CELLS, four turns each, at ticks 0,
// 4, 8 and 12 times what a straight cell costs. Entities 2 and 3 are named by
// nothing, so what they do is what a world does under no command at all.
//
// The turns are a cell apart rather than a tick apart because an order arriving
// inside a crossing waits for it: written a tick apart, three of the four turns
// would land on a unit that could not act on them and the script would be
// measured against a world standing still.
//
// FOUR TURNS IS THE FIXTURE'S DISCRIMINATING POWER. An order issued after the
// first turn leaves THREE more scripted targets for the commanded unit — past
// AC-6's "two of them" — and each is a tick on which an exclusion that lasted
// one tick, or was cleared by the drain, hands the unit back to the script.
// Entity 1's legs are three cells against a four-cell turn, so it ARRIVES on
// every one of its own targets with a cell to spare, which is what lets "a
// second unit keeps reaching its own" be read off cells rather than off a target
// field.
//
// Each entry is its own slice, freshly made per call: two entries that aliased
// one another would let a write into the first hide as an agreement with the
// second, and the identity assertion below is stated over exactly these headers.
func scriptedLaps() [][]sim.Command {
	out := func() []sim.Command {
		return []sim.Command{{Entity: 0, X: 31, Y: 23}, {Entity: 1, X: 25, Y: 23}}
	}
	home := func() []sim.Command {
		return []sim.Command{{Entity: 0, X: 21, Y: 23}, {Entity: 1, X: 25, Y: 20}}
	}
	sched := make([][]sim.Command, 13*cellTicks)
	sched[0], sched[4*cellTicks] = out(), home()
	sched[8*cellTicks], sched[12*cellTicks] = out(), home()
	return sched
}

// sameSlice reports whether a and b are the SAME slice rather than two that
// compare equal: one data pointer, one length, one capacity is one slice
// value.
func sameSlice(a, b []sim.Command) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() &&
		len(a) == len(b) && cap(a) == cap(b)
}

// TestACommandedUnitTakesNoFurtherScriptedTarget — 0028 SC-7 (AC-6, C-4):
// from the first order naming a unit that unit takes no further command from
// the placeholder script, driven past three of its own scripted turns, while
// a second unit keeps reaching its own throughout.
//
// The order is issued AFTER the script's first turn, deliberately: right
// after tick 0, entity 0 already holds the script's own EAST target, which is
// what says the script reaches that entity at all on this fixture. A test
// that ordered before tick 0 could not tell an exclusion from a script that
// never named the unit.
//
// THE SCRIPT'S OWN EAST TURN NEVER BECOMES VISIBLE AT ALL, which is a real
// consequence and not an artifact of this test: the player's order lands one
// tick later, while entity 0's Facing is still 0 (a turn's Facing does not
// move until the turn completes — advanceTurns, pkg/sim/facing.go). Replacing
// the target cancels the in-progress east turn outright
// (cancelTurnForTargetChange), so the south-east turn that follows is
// computed fresh from the UNCHANGED facing 0, not from wherever the cancelled
// turn had progressed to. A one-tick-early exclusion could not be told apart
// from this fixture's own timing by cell position either, for the same
// reason: both land on the same tick, because Facing is 0 in both cases when
// the redirect happens. Measured directly against this fixture rather than
// assumed.
func TestACommandedUnitTakesNoFurtherScriptedTarget(t *testing.T) {
	const (
		orderedEntity        = 0
		targetCol, targetRow = 28, 28
	)

	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))

	// Entities start at (21,23), (25,20), (22,27), (31,24); 2 and 3 are named by
	// nothing here and so stand on their start cells throughout. Each entry is a
	// tick on which one of the two CHANGES CELL; between two of them neither
	// moves, and the loop carries the last pair forward and asserts it on every
	// tick, so the coverage is one assertion per tick written as the changes.
	//
	// Entity 0's ordered leg is south-east, so its cells cost the diagonal
	// crossing; ordered is the tick the order drains and orderedTurnTicks the
	// fresh south-east turn from facing 0 that follows it (ceil(96/16), the
	// same arc and rate as the corner turn below). cornerTurnTicks is the
	// diagonal-to-straight corner once the row axis is home, this fixture's
	// twin of the one in TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun.
	// Entity 0's steps all carry it EAST across the ramp, so both its scripted
	// straight one and its ordered diagonals are the uphill counts; entity 1
	// walks a column and stays on the level pair.
	const (
		ordered          = 1
		orderedTurnTicks = 6
		cornerTurnTicks  = 1
		// entity1TurnTicks is entity 1's own turn, north-to-south or the
		// reverse, at every one of its four scripted legs: ceil(128/16), the
		// same arc and the same RotationSpeed each time, so the same eight
		// ticks apply uniformly at every leg (measured directly against this
		// fixture).
		entity1TurnTicks = 8
	)
	e0At := map[int]image.Point{
		ordered + orderedTurnTicks + 0*uphillDiagonalCellTicks:                                       {X: 22, Y: 24}, // the order has drained and turned south-east; the script's own east step never became visible
		ordered + orderedTurnTicks + 1*uphillDiagonalCellTicks:                                       {X: 23, Y: 25},
		ordered + orderedTurnTicks + 2*uphillDiagonalCellTicks:                                       {X: 24, Y: 26},
		ordered + orderedTurnTicks + 3*uphillDiagonalCellTicks:                                       {X: 25, Y: 27},
		ordered + orderedTurnTicks + 4*uphillDiagonalCellTicks:                                       {X: 26, Y: 28}, // the row axis is home
		ordered + orderedTurnTicks + 5*uphillDiagonalCellTicks + cornerTurnTicks + 0*uphillCellTicks: {X: 27, Y: 28},
		ordered + orderedTurnTicks + 5*uphillDiagonalCellTicks + cornerTurnTicks + 1*uphillCellTicks: {X: 28, Y: 28}, // arrived, on the ORDERED cell
	}
	e1At := map[int]image.Point{
		0*cellTicks + entity1TurnTicks:  {X: 25, Y: 21}, // turn ONE, south
		1*cellTicks + entity1TurnTicks:  {X: 25, Y: 22},
		2*cellTicks + entity1TurnTicks:  {X: 25, Y: 23}, // reaches its first target
		4*cellTicks + entity1TurnTicks:  {X: 25, Y: 22}, // turn TWO — entity 0's is cut, entity 1's is not
		5*cellTicks + entity1TurnTicks:  {X: 25, Y: 21},
		6*cellTicks + entity1TurnTicks:  {X: 25, Y: 20}, // home
		8*cellTicks + entity1TurnTicks:  {X: 25, Y: 21}, // turn THREE — entity 0 stands, entity 1 sets out
		9*cellTicks + entity1TurnTicks:  {X: 25, Y: 22},
		10*cellTicks + entity1TurnTicks: {X: 25, Y: 23},
		12*cellTicks + entity1TurnTicks: {X: 25, Y: 22}, // turn FOUR — entity 0 still stands
	}

	e0, e1 := worldFixtureCells[0], worldFixtureCells[1]
	for i := 0; i < 13*cellTicks; i++ {
		if i == 1 {
			mw.enqueue(orderedEntity, targetCol, targetRow)
		}
		mw.tick()
		if i == 0 {
			// The script's own east order reached entity 0 on this very
			// tick, before the player's order overrides it on the next —
			// checked on the target it admitted, since the cell it will
			// eventually reach is the same either way (the paragraph above
			// this function).
			if e := mw.world.Entities()[orderedEntity]; !e.HasTarget || e.TargetX != 31 || e.TargetY != 23 {
				t.Fatalf("after tick 0 entity %d heads for (%d,%d) hasTarget=%v, want (31,23) — the script "+
					"never reached it", orderedEntity, e.TargetX, e.TargetY, e.HasTarget)
			}
		}
		if p, ok := e0At[i]; ok {
			e0 = p
		}
		if p, ok := e1At[i]; ok {
			e1 = p
		}
		want := []image.Point{e0, e1, worldFixtureCells[2], worldFixtureCells[3]}
		assertCells(t, "after the tick indexed "+strconv.Itoa(i), entityCells(mw.world), want)
	}

	// "No further command from the script" is also a statement about the target
	// field, not only about where the unit stopped: three scripted turns have
	// gone by since it arrived, and it is holding none of them.
	if e := mw.world.Entities()[orderedEntity]; e.HasTarget {
		t.Errorf("entity %d heads for (%d,%d) after four scripted turns, want no target at all — a commanded "+
			"unit leaves the script for good (FR-6, C-4)", orderedEntity, e.TargetX, e.TargetY)
	}
}

// TestTheAssembledStreamIsTheScriptsOwnEntryUntilAnOrderJoinsIt — 0028
// SC-8's identity clause and T5's direct splice assertion.
//
// TWO CLAIMS, AND NEITHER CAN BE MADE ABOUT A WORLD. With nothing enqueued and
// nothing commanded the assembled stream IS the schedule's entry, by identity —
// there is no other slice for the advance to run on, which is what makes
// "invents no command" structural rather than sampled. And with an order pending
// the stream is the script's commands FOLLOWED BY it: the exclusion has already
// removed the scripted command an order could have lost to under last-write, so
// no digest and no cell distinguishes the two splice orders, and this slice is
// the only witness the direction has.
//
// The order names an entity the tested entry does NOT name, on purpose. The
// exclusion and the splice would otherwise be observed on one assertion, and a
// missing exclusion would read here as a splice defect.
func TestTheAssembledStreamIsTheScriptsOwnEntryUntilAnOrderJoinsIt(t *testing.T) {
	const orderedEntity = 2 // named by no entry in scriptedLaps

	m := worldFixtureMap()
	sched := scriptedLaps()
	entry := sched[0]
	if len(entry) != 2 {
		t.Fatalf("the fixture's first entry holds %d commands, want the 2 the splice below is written over", len(entry))
	}
	for _, c := range entry {
		if c.Entity == orderedEntity {
			t.Fatalf("the fixture's first entry names entity %d, so the exclusion and the splice would be "+
				"observed on one assertion", orderedEntity)
		}
	}

	mw := newMapWorld(mapload.FromALM(m), sched, nil, worldFixtureViewer(t, m))
	if mw.world.Tick() != 0 {
		t.Fatalf("the world stands at tick %d, so commands() reads an entry other than sched[0]", mw.world.Tick())
	}

	got := mw.commands()
	if !sameSlice(got, entry) {
		t.Fatalf("with nothing enqueued and nothing commanded the assembled stream is %v (len %d cap %d), a "+
			"DIFFERENT slice from the schedule's own entry %v (len %d cap %d) — an advance with no order "+
			"pending must run on the schedule's own slice, not on a copy of it (FR-5, P-3)",
			got, len(got), cap(got), entry, len(entry), cap(entry))
	}

	// The splice: the script's commands, in the script's order, then the order.
	mw.enqueue(orderedEntity, 29, 21)
	got = mw.commands()
	// The order is a GROUP move-to under the first tag the queue opens, which is
	// the whole of what the seam now emits; the script's own two entries are
	// scripted commands and are untouched by it.
	want := []sim.Command{entry[0], entry[1],
		{Kind: sim.KindGroupMoveTo, Entity: orderedEntity, X: 29, Y: 21, Group: 1}}
	if len(got) != len(want) {
		t.Fatalf("the assembled stream holds %d commands %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("the assembled stream is %v, want %v — position %d differs. The pending order MUST come "+
				"after the tick's scripted commands: Step applies a slice in order with the last write "+
				"winning, and this slice is the only witness that direction has (FR-4, DD-7)", got, want, i)
		}
	}

	// Fresh, and the entry it was built from untouched: an append into a
	// schedule entry's spare capacity would corrupt a script a later tick has
	// still to apply.
	if sameSlice(got, entry) {
		t.Error("the spliced stream IS the schedule's entry — the splice wrote into the script itself")
	}
	if len(entry) != 2 || entry[0] != want[0] || entry[1] != want[1] {
		t.Errorf("the schedule's entry now reads %v, want it left as %v", entry, want[:2])
	}
}

// TestAnUnorderedRunFollowsTheScriptAndAnOrderedRunRepeats — 0028 SC-8's
// remaining clauses (AC-7).
//
// Its own non-vacuity is the last comparison: the ordered runs must not end on
// the digest the untouched run does, or two runs agreeing would be two runs of
// nothing.
func TestAnUnorderedRunFollowsTheScriptAndAnOrderedRunRepeats(t *testing.T) {
	const ticks = 13

	m := worldFixtureMap()

	quiet := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))
	fresh := mapload.FromALM(m)
	freshSched := scriptedLaps()
	if got, want := quiet.world.Hash(), fresh.Hash(); got != want {
		t.Fatalf("the two sides start at %#x and %#x — they must begin equal", got, want)
	}
	for i := 0; i < ticks; i++ {
		quiet.tick()
		sim.Run(fresh, freshSched[i:], 1)
		if got, want := quiet.world.Hash(), fresh.Hash(); got != want {
			t.Fatalf("with nothing ever enqueued the driven digest %#x != the headless digest %#x after tick "+
				"%d — an advance with no pending order runs on exactly the script's own commands, inventing "+
				"none and dropping none (FR-5, P-3)", got, want, i+1)
		}
	}
	quietDigest := quiet.world.Hash()
	if !someEntityHasMoved(worldFixtureCells, entityCells(quiet.world)) {
		t.Fatalf("no entity moved in %d ticks (%v) — the comparison above is between two untouched worlds",
			ticks, entityCells(quiet.world))
	}

	// One order script, run twice from equal worlds. Both entities the script
	// names are commanded, and both are commanded on a tick whose scripted entry
	// is empty, so the two runs differ from the quiet one by the exclusion and
	// the orders together.
	orders := []struct {
		at     int
		entity uint32
		x, y   int
	}{
		{1, 0, 28, 28},
		{6, 1, 29, 21},
	}
	run := func() []uint64 {
		mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, worldFixtureViewer(t, m))
		digests := make([]uint64, ticks)
		for i := 0; i < ticks; i++ {
			for _, o := range orders {
				if o.at == i {
					mw.enqueue(o.entity, o.x, o.y)
				}
			}
			mw.tick()
			digests[i] = mw.world.Hash()
		}
		return digests
	}

	first, second := run(), run()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("two runs of one order script hold %#x and %#x after tick %d — two worlds equal at the "+
				"start and advanced by equal combined streams are equal at every tick (P-4)",
				first[i], second[i], i+1)
		}
	}
	if first[ticks-1] == quietDigest {
		t.Fatalf("the ordered runs end on the digest the unordered one does (%#x), so their agreeing at every "+
			"tick says nothing about what an order did", quietDigest)
	}
}

// ---------------------------------------------------------------- 0033 AC-10

// TestTheSeamCarriesTheWorldsOwnLifeStateAndHealthPair is AC-10. One world holds
// all three states at once, and the snapshot is compared ENTRY BY ENTRY against
// what each unit's own health pair says it must be — an alive unit, one driven
// to exactly zero, one killed, and one with no health system at all.
//
// The expected life bytes are written out by hand as ui.LifeAlive/Downed/Dead
// against a health pair stated beside them, never read back off the entity's own
// predicates: read that way this table would agree with whatever the seam did,
// which is the whole failure a seam test exists to catch.
func TestTheSeamCarriesTheWorldsOwnLifeStateAndHealthPair(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: 70, MaxHP: 100},
		{ID: 1, X: 3, Y: 1, HP: 100, MaxHP: 100},
		{ID: 2, X: 5, Y: 1, HP: 100, MaxHP: 100},
		{ID: 3, X: 7, Y: 1},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	// The two states that cannot be built are reached the one way they exist:
	// through a blow.
	sim.Step(w, []sim.Command{
		{Kind: sim.KindDamage, Entity: 1, X: 100},
		{Kind: sim.KindKill, Entity: 2},
	})

	want := []struct {
		life       uint8
		hp, maxHP  int
		selectable bool
	}{
		{ui.LifeAlive, 70, 100, false},
		{ui.LifeDowned, 0, 100, false},
		{ui.LifeDead, -1, 100, true},
		{ui.LifeAlive, 0, 0, false},
	}

	draws := seamDraws(w, nil)
	if len(draws) != len(want) {
		t.Fatalf("the snapshot holds %d entries for %d entities", len(draws), len(want))
	}
	for i, d := range draws {
		if d.Life != want[i].life {
			t.Errorf("entry %d crossed as life %d, want %d", i, d.Life, want[i].life)
		}
		if d.HP != want[i].hp || d.MaxHP != want[i].maxHP {
			t.Errorf("entry %d crossed at %d/%d, want %d/%d", i, d.HP, d.MaxHP, want[i].hp, want[i].maxHP)
		}
		if d.Selectable != want[i].selectable {
			t.Errorf("entry %d selectable exception = %v, want %v", i, d.Selectable, want[i].selectable)
		}
		if d.ID != uint32(i) {
			t.Errorf("entry %d carries id %d — the snapshot is in the world's own order", i, d.ID)
		}
	}
}

func TestTheSeamCarriesTheWorldsOwnManaPool(t *testing.T) {
	b := sim.Bounds{Width: 12, Height: 12}
	w, err := sim.NewWorld(1, b, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 0, X: 1, Y: 1, HP: 100, MaxHP: 100, Mana: 12, MaxMana: 40},
		{ID: 1, X: 3, Y: 1, HP: 100, MaxHP: 100}, // no mana system
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	// A STEPPED world, not merely a built one — the whole point being tested
	// is that the pair still crosses off a world that has actually advanced.
	sim.Step(w, nil)

	draws := seamDraws(w, nil)
	if len(draws) != 2 {
		t.Fatalf("the snapshot holds %d entries for 2 entities", len(draws))
	}
	if draws[0].Mana != 12 || draws[0].MaxMana != 40 {
		t.Errorf("entity 0 crossed with mana %d/%d, want 12/40", draws[0].Mana, draws[0].MaxMana)
	}
	if draws[1].Mana != 0 || draws[1].MaxMana != 0 {
		t.Errorf("entity 1 (no mana system) crossed with %d/%d, want 0/0", draws[1].Mana, draws[1].MaxMana)
	}
}

func TestASnapshotPushMovesNoWorldFieldAndNoDigest(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), worldFixtureUnitSet(), worldFixtureViewer(t, m))
	for i := 0; i < 3; i++ {
		mw.tick()
	}
	sim.Step(mw.world, []sim.Command{
		{Kind: sim.KindDamage, Entity: 1, X: mapload.SpawnHP},
		{Kind: sim.KindKill, Entity: 2},
	})

	before, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	beforeHash := mw.world.Hash()

	seen := map[uint8]bool{}
	for i := 0; i < 4; i++ {
		mw.push()
		for _, d := range mw.entityDraws() {
			seen[d.Life] = true
		}
	}

	after, err := mw.world.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Errorf("the world's byte form moved under a push:\n % x\n % x", after, before)
	}
	if got := mw.world.Hash(); got != beforeHash {
		t.Errorf("the world hashes %#016x after a push and %#016x before it", got, beforeHash)
	}
	for _, l := range []uint8{ui.LifeAlive, ui.LifeDowned, ui.LifeDead} {
		if !seen[l] {
			t.Fatalf("life %d never crossed; the pushes above read a world in fewer than three states", l)
		}
	}
}

// TestTheTickZeroPushCarriesNoStepAndAsecondBuildRepeatsIt вЂ” 0047
// SC-1's two remaining clauses, over the fixture map's four units.
//
// The tick-0 push is the frame a map opens on, and every entity on it has taken
// NO step: the memory has not seen any of them, and presence is what answers,
// so no entity is handed a delta measured from the map's corner.
func TestTheTickZeroPushCarriesNoStepAndAsecondBuildRepeatsIt(t *testing.T) {
	m := worldFixtureMap()
	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), worldFixtureUnitSet(), worldFixtureViewer(t, m))

	for i, d := range mw.entityDraws() {
		if d.Step != (image.Point{}) {
			t.Errorf("at tick 0 entity %d is pushed with step %v, want none вЂ” nothing has advanced yet",
				i, d.Step)
		}
		if d.Cell != worldFixtureCells[i] {
			t.Errorf("at tick 0 entity %d stands on %v, want its unit record's own cell %v",
				i, d.Cell, worldFixtureCells[i])
		}
	}

	// Five ticks, so the run below reads a screen with somebody mid-walk on it
	// rather than four units that have never moved.
	for i := 0; i < 5; i++ {
		mw.tick()
	}

	first := mw.entityDraws()
	mw.push()
	second := mw.entityDraws()
	if !sameDraws(first, second) {
		t.Fatalf("a snapshot built twice with no advance between answers differently:\n %+v\n %+v",
			first, second)
	}

	movers := 0
	for _, d := range first {
		if d.Step != (image.Point{}) {
			movers++
		}
	}
	if movers == 0 {
		t.Fatal("no entity carried a step over those three ticks, so the repeat above compares two " +
			"snapshots in which nothing moved and says nothing about DD-1")
	}
}

// TestThePacedAdvancePushesItsOwnClocksPhaseAndAStopHoldsIt вЂ” 0047
// SC-6's clock half: the paced advance pushes where the frame stands inside
// the current tick, read from the accumulator that decides when a tick fires
// вЂ” and from no other instance of that type.
//
// THE VIEWER'S WATER CLOCK IS DELIBERATELY RE-RATED TO A DIFFERENT PERIOD before
// anything is measured. Those are two instances of the same type, re-rated
// through two separate call sites and each carrying its own remainder, so a
// phase read off the wrong one is the two-clock failure C-3 refuses вЂ” and here
// it shows as a period that is the water's and not the world's.
func TestThePacedAdvancePushesItsOwnClocksPhaseAndAStopHoldsIt(t *testing.T) {
	// Two distinct periods for the two clocks. Neither is a rung of the cadence
	// ladder, deliberately: the seam adopts a period verbatim, so a case about
	// telling two clocks apart must not be able to pass because both happened to
	// land on a rung.
	const worldPeriodUS, waterPeriodUS = 62_500, 3_906

	base := time.Unix(1_700_000_000, 0)
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)

	// A viewer nobody has told holds no phase at all вЂ” the standalone front-end,
	// and every caller written before this story.
	if e, p := v.Phase(); e != 0 || p != 0 {
		t.Fatalf("a viewer nobody paced reports phase (%d, %d), want (0, 0)", e, p)
	}

	mw := newMapWorld(mapload.FromALM(m), scriptedLaps(), nil, v)
	if e, p := v.Phase(); e != 0 || p != 0 {
		t.Fatalf("opening a map pushed phase (%d, %d); the constructor pushes entities, not a phase", e, p)
	}

	mw.setCadence(worldPeriodUS, false)
	if got := v.SetPeriod(waterPeriodUS); got != waterPeriodUS {
		t.Fatalf("the viewer's water clock adopted %d us, want %d", got, waterPeriodUS)
	}
	period := worldPeriodUS
	if waterPeriodUS == period {
		t.Fatalf("both clocks run at %d us; this test cannot tell them apart", period)
	}

	now := base
	if n, _ := mw.paceTo(now); n != 0 {
		t.Fatalf("the baseline call ran %d ticks, want 0", n)
	}
	// The baseline call returns before the elapsed span reaches the
	// accumulator, exactly as a stopped one does, so it pushes nothing either
	// — and until the second call the viewer still holds no phase, which is
	// the frame a map opens on and where nothing has stepped in any case.
	if e, p := v.Phase(); e != 0 || p != 0 {
		t.Fatalf("the baseline call pushed phase (%d, %d), want (0, 0)", e, p)
	}

	// Four frames a tick, so most of them fire nothing and every one of them is
	// a frame that has advanced WITHIN a tick.
	quarter := time.Duration(period/4) * time.Microsecond
	seen, ticks := map[int]bool{}, 0
	for i := 0; i < 9; i++ {
		now = now.Add(quarter)
		n, _ := mw.paceTo(now)
		ticks += n
		e, p := v.Phase()
		if p != period {
			t.Fatalf("frame %d was pushed period %d, want the WORLD clock's %d вЂ” the phase is read off the "+
				"pacing accumulator and never off the viewer's water instance (DD-4)", i, p, period)
		}
		if want := mw.clock.Remainder(); e != want {
			t.Fatalf("frame %d was pushed %d us, want the pacing clock's own remainder %d", i, e, want)
		}
		seen[e] = true
	}
	if len(seen) < 4 || ticks == 0 {
		t.Fatalf("those frames pushed %d distinct remainder(s) and fired %d tick(s); the run must both "+
			"advance within a tick and cross one", len(seen), ticks)
	}

	// Stopped, mid-stride: many frames, and the phase does not move.
	mw.setCadence(worldPeriodUS, true)
	heldE, heldP := v.Phase()
	for i := 0; i < 12; i++ {
		now = now.Add(quarter)
		if n, applied := mw.paceTo(now); n != 0 || applied != 0 {
			t.Fatalf("a stopped frame ran %d ticks and applied %d orders, want 0 and 0", n, applied)
		}
		if e, p := v.Phase(); e != heldE || p != heldP {
			t.Fatalf("stopped frame %d moved the phase from (%d, %d) to (%d, %d) вЂ” a stopped world's "+
				"picture must hold still (FR-6, P-2)", i, heldE, heldP, e, p)
		}
	}

	// And the stop really was holding back a clock that would otherwise have
	// moved: clearing it advances the phase again on the very next frame.
	mw.setCadence(worldPeriodUS, false)
	now = now.Add(quarter)
	mw.paceTo(now)
	if e, _ := v.Phase(); e == heldE {
		t.Errorf("the frame after the resume still reports %d us, so the stopped comparisons above are "+
			"comparisons over a clock that was never going to move", e)
	}
}

// mustOpenMapWorld is openMapWorld with its error turned into a fatal, so a case
// about what a world holds is not also a case about whether the table loaded.
//
// The table is a PARAMETER and not defaulted to nil inside, because a nil table
// and a table that resolves nothing are two different fixtures and every caller
// should say which one it means.
func mustOpenMapWorld(t *testing.T, m *alm.Map, tbl *mapload.Table, units *terrain.UnitSet, v *ui.Viewer) *mapWorld {
	t.Helper()
	mw, err := openMapWorld(m, tbl, units, v)
	if err != nil {
		t.Fatalf("openMapWorld: %v", err)
	}
	return mw
}

// unitsArmKey is a class key at or above the class-key floor, so a placement
// carrying it takes the units arm — the only arm that can reach a stat block —
// whatever its flag word and its definition id hold. npcArmKey is below the
// floor, which is the only band from which the flag word diverts at all.
//
// The fixture map's own four keys are all below the floor and reach humans
// entries, which is why a case about the units table adds a placement of its own
// rather than re-keying one of theirs.
const (
	unitsArmKey = 0x40
	npcArmKey   = 0x09
)

// tableFixtureMap is the fixture map with one units-arm placement appended, at
// the cell the first fixture unit stands on. The appended entity is the LAST,
// so the four the rest of this file asserts over keep their ids.
func tableFixtureMap() *alm.Map {
	m := worldFixtureMap()
	m.Units = append(m.Units, alm.Unit{X: 0x1580, Y: 0x1700, ClassID: unitsArmKey})
	return m
}

// unitsRow is a units row keyed on unitsArmKey. Face is written as 0 and not
// left empty, because an empty cell takes the constructor's 1 while the
// placement carries no subkey at all.
func unitsRow(name string, slots map[int]int32) dbEntry {
	p := make([]int32, 38)
	for i := range p {
		p[i] = -1
	}
	p[0x1d], p[0x1e] = unitsArmKey, 0
	for k, v := range slots {
		p[k] = v
	}
	return dbEntry{name: name, params: p}
}

// dbCollection is a definition collection built in test code: index 0 is the
// reserved empty entry and every other row is written by slot.
type dbEntry struct {
	name    string
	params  []int32
	strings []string
}

type dbCollection []dbEntry

func (c dbCollection) Len() int                    { return len(c) }
func (c dbCollection) EntryName(i int) string      { return c[i].name }
func (c dbCollection) EntryParams(i int) []int32   { return c[i].params }
func (c dbCollection) EntryStrings(i int) []string { return c[i].strings }

func TestTheTableReachesTheWorldTheApplicationOpens(t *testing.T) {
	const health = 57
	if health == mapload.SpawnHP {
		t.Fatal("the fixture chose the provisional constant; it would prove nothing")
	}
	m := tableFixtureMap()
	last := len(m.Units) - 1

	tbl := &mapload.Table{Units: dbCollection{
		{},
		unitsRow("flyer", map[int]int32{0x04: health, 0x20: 3}),
	}}

	plain := mustOpenMapWorld(t, m, nil, nil, worldFixtureViewer(t, m))
	withTable := mustOpenMapWorld(t, m, tbl, nil, worldFixtureViewer(t, m))

	before := plain.world.Entities()[last]
	after := withTable.world.Entities()[last]

	if before.Domain != sim.DomainGround || before.HP != mapload.SpawnHP {
		t.Fatalf("with no table the placement is %+v; the fixture's baseline moved", before)
	}
	if after.Domain != sim.DomainAir {
		t.Errorf("with a table naming the air column the placement is in domain %d", after.Domain)
	}
	if after.HP != health || after.MaxHP != health {
		t.Errorf("with a table the placement is at %d/%d, want %d/%d",
			after.HP, after.MaxHP, health, health)
	}

	// The four placements that reach no units entry are untouched, so a table
	// changes what it resolves and nothing else.
	for i := 0; i < last; i++ {
		if plain.world.Entities()[i] != withTable.world.Entities()[i] {
			t.Errorf("placement %d moved though it resolves to no units entry", i)
		}
	}
	if plain.world.Hash() == withTable.world.Hash() {
		t.Error("the table changed no digest at all; nothing resolved")
	}
}

func TestAMapOpenRefusesATableTheLoaderRefuses(t *testing.T) {
	m := tableFixtureMap()
	// A row whose damage selector takes an arm the definition loader does not
	// model. It resolves, and is then refused by name.
	bad := &mapload.Table{Units: dbCollection{{}, unitsRow("Offender", map[int]int32{0x0d: 1})}}

	mw, err := openMapWorld(m, bad, nil, worldFixtureViewer(t, m))
	if err == nil {
		t.Fatal("openMapWorld accepted a table the loader refuses and built a world")
	}
	if mw != nil {
		t.Errorf("openMapWorld returned a world alongside its error")
	}
	if !strings.Contains(err.Error(), "Offender") {
		t.Errorf("error %q does not name the entry that was refused", err)
	}
}

// The mission driver: the notices a running mission produces, what an
// advance does with them, and that none of it reaches the simulation.
//
// These drive a REAL scripted world through pkg/sim rather than stubbing an
// outcome or a latch array, because the whole claim is about what a script pass
// writes and when. Every fixture is built here; nothing reads a game install and
// every event text below is ASCII this file wrote.

// missionVar is the register the scripts below keep their variable in. It is
// above every check they compile, because a variable and a check result share
// one array and an index below the check count is a cell a check overwrites
// every pass.
const missionVar int32 = 10

func missionChecks() []sim.ScriptCheck {
	return []sim.ScriptCheck{
		{Op: sim.ScriptCheckVariable, Register: 0, Args: [10]int32{missionVar}},
		{Op: sim.ScriptCheckConstant, Register: 1, Args: [10]int32{0}},
		{Op: sim.ScriptCheckConstant, Register: 2, Args: [10]int32{1}},
	}
}

// missionTrigger is one trigger comparing register 0 against register right,
// latching at the given map position, running the given instant subscripts.
func missionTrigger(latch, right int32, once bool, acts ...int32) sim.ScriptTrigger {
	var t sim.ScriptTrigger
	t.Pairs[0] = sim.ScriptPair{Left: 0, Right: right, Cmp: sim.ScriptCmpEQ, Used: true}
	t.Once, t.Latch = once, latch
	for i := range t.Instants {
		t.Instants[i] = sim.ScriptNone
	}
	for i, a := range acts {
		t.Instants[i] = a
	}
	return t
}

func missionWorld(t *testing.T, instants []sim.ScriptInstant, triggers []sim.ScriptTrigger) *sim.World {
	t.Helper()
	s, err := sim.NewScript(missionChecks(), instants, triggers)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), nil, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w
}

// missionSource is the archive set a mission's words are read out of: a map of
// address to bytes and nothing else, which is the whole of what the driver needs
// of a filesystem.
type missionSource map[string][]byte

func (s missionSource) ReadFile(name string) ([]byte, error) {
	if b, ok := s[name]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("no entry %q", name)
}

// missionEvent is the address mission n's event e is read at, composed by the
// same function the driver uses so a renamed container moves both together.
func missionEvent(t *testing.T, n, e int) string {
	t.Helper()
	addr, ok := EventTextPath(n, e)
	if !ok {
		t.Fatalf("EventTextPath(%d, %d) refused", n, e)
	}
	return addr
}

// missionFont is one record per byte, all alike: enough for the viewer to
// consider itself able to draw. Nothing here asserts about pixels — a viewer
// with no font reports every notice closed, so omitting it would make every
// assertion below vacuous.
func missionFont() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = text.Glyph{Width: 1, Height: 6, Pixels: make([]text.Pixel, 6), Advance: 1}
	}
	return f
}

// missionDriverFor builds the driver under a hand-built scripted world, with the
// fixture map's schedule and tiers and a viewer over its grid.
func missionDriverFor(t *testing.T, w *sim.World, raises []mapload.ScriptRaise,
	src missionSource) (*mapWorld, *ui.Viewer) {
	t.Helper()
	return missionDriverWithFaces(t, w, raises, src, nil)
}

// missionDriverWithFaces is missionDriverFor with a picture seam supplied. The
// SHIPPED path passes nil — nothing in this tree can resolve a speaker's face
// yet — so the plain helper above is what every assertion about shipped
// behaviour goes through, and this one exists to exercise the seam.
func missionDriverWithFaces(t *testing.T, w *sim.World, raises []mapload.ScriptRaise,
	src missionSource, faces FaceSource) (*mapWorld, *ui.Viewer) {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	v.SetFont(missionFont())
	mw := openMission(&Mission{Number: 7, Map: m, World: w, Raises: raises}, nil, nil, v, src, faces, nil)
	return mw, v
}

// missionSteps advances the driver over n whole script cycles, one tick at a
// time, the way paceTo does.
func missionSteps(mw *mapWorld, cycles int) {
	for i := 0; i < cycles*16; i++ {
		mw.tick()
	}
}

// AC-5: a fired raise-message action over a shipped event text shows that file's
// part 1.
func TestMissionShowsPartOneWhenAnAnnouncementFires(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte("<part=1>\r\nhello\r\n<part=2>\r\nagain")})

	missionSteps(mw, 2)
	got, kind, open := v.NoticeState()
	if !open || kind != ui.NoticeDialogue || got != "hello" {
		t.Fatalf("notice = %q/%v/%v, want %q as a dialogue", got, kind, open, "hello")
	}
}

// AC-6: a raise naming no shipped file produces nothing at all.
func TestMissionIsSilentForAnUnshippedEventText(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{9}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 9}}, missionSource{})

	missionSteps(mw, 3)
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a raise naming no file opened a notice")
	}
	if mw.mission.open {
		t.Fatal("the driver believes a notice is open for a file that does not ship")
	}
}

// A file that ships and yields no part 1 is the same silence.
func TestMissionIsSilentForAFileWithNoPartOne(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte("<part=2>\r\nonly the second")})

	missionSteps(mw, 3)
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a file with no part 1 opened a notice")
	}
	if mw.mission.open {
		t.Fatal("the driver believes a notice is open for a file with no part 1")
	}
}

// AC-7: advancing pages to part n+1, and advancing past the last part closes it.
func TestAdvancingADialoguePagesThenCloses(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte("<part=1>\r\none\r\n<part=2>\r\ntwo\r\n<part=3>\r\nthree")})

	missionSteps(mw, 2)
	for _, want := range []string{"one", "two", "three"} {
		got, _, open := v.NoticeState()
		if !open || got != want {
			t.Fatalf("notice = %q (open %v), want %q", got, open, want)
		}
		if dest, msg, _ := mw.advanceNotice(); dest != ui.NoticeStay || msg != "" {
			t.Fatalf("paging answered %v/%q, want NoticeStay and no message", dest, msg)
		}
	}
	if _, _, open := v.NoticeState(); open {
		t.Fatal("advancing past the last part left the notice open")
	}
	if mw.mission.open || mw.mission.part != 0 {
		t.Fatalf("the driver still holds %+v after the last part", mw.mission)
	}
}

// AC-8: a raise arriving while any notice is open is DISCARDED — not queued, not
// deferred, and not shown when the open notice closes.
func TestARaiseWhileANoticeIsOpenIsDiscarded(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{
			{Op: 2, Args: [10]int32{4}},
			{Op: 2, Args: [10]int32{5}},
		},
		[]sim.ScriptTrigger{
			missionTrigger(3, 1, true, 0),
			missionTrigger(5, 1, true, 1),
		})
	mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}, {Latch: 5, Event: 5}},
		missionSource{
			missionEvent(t, 7, 4): []byte("<part=1>\r\nfirst"),
			missionEvent(t, 7, 5): []byte("<part=1>\r\nsecond"),
		})

	missionSteps(mw, 2)
	if got, _, _ := v.NoticeState(); got != "first" {
		t.Fatalf("notice = %q, want the first raise's text", got)
	}
	mw.advanceNotice()
	if _, _, open := v.NoticeState(); open {
		t.Fatal("closing the first notice showed the discarded raise")
	}
	missionSteps(mw, 4)
	if _, _, open := v.NoticeState(); open {
		t.Fatal("the discarded raise appeared on a later step")
	}
}

// A dialogue raised by the action that decides the mission is shown to
// completion before the latched outcome. The outcome still appears exactly
// once and remains the only notice whose dismissal leaves the mission.
func TestTheOutcomeWaitsForEveryFinalDialoguePageAndIsShownOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   int32
		want string
		kind ui.NoticeKind
	}{
		{"won", sim.ScriptInstantWin, MissionWonText, ui.NoticeSuccess},
		{"lost", sim.ScriptInstantLose, MissionLostText, ui.NoticeFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := missionWorld(t,
				[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}, {Op: tc.op}},
				[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0, 1)})
			mw, v := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
				missionSource{missionEvent(t, 7, 4): []byte("<part=1>\r\nlast words\r\n<part=2>\r\nfarewell")})

			missionSteps(mw, 2)
			for _, words := range []string{"last words", "farewell"} {
				got, kind, open := v.NoticeState()
				if !open || kind != ui.NoticeDialogue || got != words {
					t.Fatalf("notice = %q/%v/%v, want final dialogue page %q", got, kind, open, words)
				}
				if dest, _, _ := mw.advanceNotice(); dest != ui.NoticeStay {
					t.Fatalf("advancing final dialogue returned %v, want NoticeStay", dest)
				}
			}
			got, kind, open := v.NoticeState()
			if !open || kind != tc.kind || got != tc.want {
				t.Fatalf("notice after final page = %q/%v/%v, want outcome %q", got, kind, open, tc.want)
			}
			mw.advanceNotice()
			missionSteps(mw, 4)
			if _, _, open := v.NoticeState(); open {
				t.Fatal("the outcome notice came back after being dismissed")
			}
		})
	}
}

// AC-15: THE DRIVER does not stop the world while a notice is up. Its tick
// advances and its script keeps being evaluated, so nothing in the notice path
// gates the world's own step.
//
// SINCE 0073 THAT IS NO LONGER THE SAME SENTENCE AS "the world keeps running
// behind an open box", and this test's wording said so until then. The world
// IS held still behind a notice now — by the FRONT-END, which declares the stop
// through the cadence seam and goes on asking for its advance, so the pacing
// arm above tick() returns without running one. This drives tick() directly,
// below that arm, so what it measures is unchanged and still worth measuring:
// the suspension is the front-end's alone, and no gate on a notice ever leaked
// into the driver.
func TestTheWorldKeepsRunningUnderAnOpenNotice(t *testing.T) {
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, _ := missionDriverFor(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte("<part=1>\r\nwords")})

	missionSteps(mw, 2)
	if !mw.mission.open {
		t.Fatal("setup: no notice open")
	}
	before := mw.world.Tick()
	missionSteps(mw, 2)
	if mw.world.Tick() <= before {
		t.Fatalf("tick %d after %d — the world stopped while a notice was up", mw.world.Tick(), before)
	}
	if !mw.mission.open {
		t.Error("the notice closed on its own; nothing closes one on a timer")
	}
}

func TestAdvancingTheOutcomeSelectsItsDestination(t *testing.T) {
	for _, tc := range []struct {
		name string
		op   int32
		dest ui.NoticeDest
		msg  string
	}{
		{"a lost mission returns to the menu", sim.ScriptInstantLose, ui.NoticeToMenu, ""},
		{"a won mission returns to the map list, saying so", sim.ScriptInstantWin,
			ui.NoticeToMapList, TownNotBuiltMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := missionWorld(t, []sim.ScriptInstant{{Op: tc.op}},
				[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
			mw, _ := missionDriverFor(t, w, nil, missionSource{})

			missionSteps(mw, 2)
			if !mw.mission.open {
				t.Fatal("setup: no outcome notice")
			}
			dest, msg, _ := mw.advanceNotice()
			if dest != tc.dest || msg != tc.msg {
				t.Fatalf("advance = %v/%q, want %v/%q", dest, msg, tc.dest, tc.msg)
			}
		})
	}
}

// A driver with no notice open answers the seam with "stay" and no message, so a
// stray press cannot navigate — and a map the picker opened carries no mission
// state at all, which is what keeps that whole path the one this story
// inherited.
func TestAdvancingNothingStays(t *testing.T) {
	w := missionWorld(t, nil, nil)
	mw, _ := missionDriverFor(t, w, nil, missionSource{})
	if dest, msg, _ := mw.advanceNotice(); dest != ui.NoticeStay || msg != "" {
		t.Fatalf("advance with nothing open = %v/%q", dest, msg)
	}

	m := worldFixtureMap()
	plain := newMapWorld(mapload.FromALM(m), nil, nil, worldFixtureViewer(t, m))
	if plain.mission != nil {
		t.Fatal("a picker-opened map carries mission state")
	}
	if dest, _, _ := plain.advanceNotice(); dest != ui.NoticeStay {
		t.Fatalf("a picker-opened map answered %v", dest)
	}
	plain.tick()
}

func TestNoticesDoNotReachTheWorld(t *testing.T) {
	mk := func() *sim.World {
		return missionWorld(t,
			[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
			[]sim.ScriptTrigger{missionTrigger(3, 1, false, 0)})
	}
	loud, _ := missionDriverFor(t, mk(), []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte("<part=1>\r\none\r\n<part=2>\r\ntwo")})
	quiet := newMapWorld(mk(), nil, nil, worldFixtureViewer(t, worldFixtureMap()))

	for i := 0; i < 64; i++ {
		loud.tick()
		quiet.tick()
		if loud.world.Tick() != quiet.world.Tick() {
			t.Fatalf("step %d: %d ticks with notices, %d without", i, loud.world.Tick(), quiet.world.Tick())
		}
		if loud.world.Hash() != quiet.world.Hash() {
			t.Fatalf("step %d: the notice path changed the world's digest", i)
		}
	}
	if !loud.mission.open {
		t.Fatal("setup: the loud driver never showed a notice, so this measured nothing")
	}
}

// ---------------------------------------------------------------------------
// The speaker the driver decides.
// ---------------------------------------------------------------------------

// missionFaces is a picture seam over a table of speakers. A "picture" here is a
// 1x1 image built in test code: nothing about the pixels is asserted below, only
// which picture arrived and whether one did.
//
// EVERY SPEAKER HERE STATES A WINDOW DERIVED FROM ITS OWN NUMBER, so a test can
// tell one speaker's crop from another's without a second table. What the driver
// does with it is carry it: the pane that cuts it lives a tier away.
type missionFaces map[int]*image.RGBA

func (m missionFaces) SpeakerFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	img, ok := m[speaker]
	if !ok {
		return nil, image.Rectangle{}, false
	}
	return img, ui.NoticeFaceWindow(speaker, speaker), true
}

func faceDot() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, 1, 1)) }

// missionDialogue drives the fixture to its one raise and returns the driver and
// the viewer with the window open on part 1.
func missionDialogue(t *testing.T, payload string, faces FaceSource) (*mapWorld, *ui.Viewer) {
	t.Helper()
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: 2, Args: [10]int32{4}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	mw, v := missionDriverWithFaces(t, w, []mapload.ScriptRaise{{Latch: 3, Event: 4}},
		missionSource{missionEvent(t, 7, 4): []byte(payload)}, faces)
	missionSteps(mw, 2)
	if _, _, open := v.NoticeState(); !open {
		t.Fatal("no notice opened")
	}
	return mw, v
}

// The window's shape comes from the WHOLE FILE and is carried across every page
// of it — including parts whose own tags name nobody, which is the case the two
// tests differ on.
func TestMissionSettlesTheShapeOnceFromTheFile(t *testing.T) {
	for _, tc := range []struct {
		name     string
		payload  string
		portrait bool
	}{
		{"a tag names a speaker", "<npc=21,part=1>\r\none\r\n<part=2>\r\ntwo", true},
		{"only the prose carries the letters", "<part=1>\r\nthe npc guild\r\n<part=2>\r\ntwo", true},
		{"neither", "<part=1>\r\none\r\n<part=2>\r\ntwo", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mw, v := missionDialogue(t, tc.payload, nil)
			for part := 1; part <= 2; part++ {
				if got, _ := v.NoticeSpeaker(); got != tc.portrait {
					t.Fatalf("part %d: shape = %v, want %v", part, got, tc.portrait)
				}
				if mw.mission.portrait != tc.portrait {
					t.Fatalf("part %d: the driver's own answer is %v", part, mw.mission.portrait)
				}
				mw.advanceNotice()
			}
		})
	}
}

// A part naming nobody LEAVES THE STANDING FACE; a part naming a different
// speaker replaces it; a named speaker the seam cannot resolve empties the pane.
func TestMissionPagesTheFaceThroughTheSeam(t *testing.T) {
	red, blue := faceDot(), faceDot()
	faces := missionFaces{21: red, 25: blue}
	mw, v := missionDialogue(t,
		"<npc=21,part=1>\r\none\r\n<part=2>\r\ntwo\r\n<npc=25,part=3>\r\nthree\r\n<npc=99,part=4>\r\nfour", faces)

	if _, got := v.NoticeSpeaker(); got != red {
		t.Fatal("part 1's speaker's picture did not arrive")
	}
	mw.advanceNotice()
	if _, got := v.NoticeSpeaker(); got != red {
		t.Fatal("part 2 names nobody and must leave the standing face")
	}
	mw.advanceNotice()
	if _, got := v.NoticeSpeaker(); got != blue {
		t.Fatal("part 3 names a different speaker and must replace the face")
	}
	mw.advanceNotice()
	if _, got := v.NoticeSpeaker(); got != nil {
		t.Fatal("part 4 names a speaker the seam cannot resolve; the pane must be emptied")
	}
}

// THE SHIPPED PATH HAS NO SEAM, and this is the behaviour the player sees today:
// the pane is drawn on every window the corpus produces, and no face is ever put
// in it. It is the story's disclosed open hop, asserted rather than assumed.
func TestMissionWithNoFaceSourceDrawsThePaneEmpty(t *testing.T) {
	mw, v := missionDialogue(t, "<npc=21,part=1>\r\none\r\n<npc=25,part=2>\r\ntwo", nil)
	for part := 1; part <= 2; part++ {
		pane, face := v.NoticeSpeaker()
		if !pane {
			t.Fatalf("part %d: the pane must still be drawn", part)
		}
		if face != nil {
			t.Fatalf("part %d: a face arrived with no source to resolve one", part)
		}
		mw.advanceNotice()
	}
}

// The outcome notice takes neither test and carries no pane, even where the
// dialogue that preceded it had one standing.
func TestMissionOutcomeDropsTheSpeaker(t *testing.T) {
	mw, v := missionDialogue(t, "<npc=21,part=1>\r\none", missionFaces{21: faceDot()})
	if pane, face := v.NoticeSpeaker(); !pane || face == nil {
		t.Fatal("the dialogue did not open with a face standing")
	}
	mw.mission.open, mw.mission.announced = false, false
	mw.view.SetNotice(MissionWonText, ui.NoticeOutcome)
	if pane, face := v.NoticeSpeaker(); pane || face != nil {
		t.Fatalf("the outcome notice kept pane=%v face=%v", pane, face != nil)
	}
}

// Closing a window drops the driver's own answer too, so the next one cannot
// inherit it.
func TestMissionCloseDropsTheShape(t *testing.T) {
	mw, _ := missionDialogue(t, "<npc=21,part=1>\r\none", nil)
	if !mw.mission.portrait {
		t.Fatal("the window did not open with a pane")
	}
	mw.advanceNotice() // part 2 does not exist, so this closes it
	if mw.mission.open {
		t.Fatal("the window did not close")
	}
	if mw.mission.portrait {
		t.Fatal("the driver kept the closed window's shape")
	}
}

// The pack refresh and the grab key (AC-11). packCode1 and countingSource
// are inventory_test.go's own, this package's shared fixtures.

// grabWorld opens a mission through openMission — the only path that ever
// sets invSubjectSet — with entity as the sole party member and the sole
// start id, buildInventorySubject's own pairing (inventory.go). It is
// invMission's shape with a caller-chosen id and world, rather than
// invMission's own fixed id 42, so a test here can put the entity on a
// chosen cell.
func grabWorld(t *testing.T, w *sim.World, entity sim.EntityID, src entrySource) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{PlayerCharacter: true, StartingHero: true}},
		Start: mapload.Start{IDs: []sim.EntityID{entity}}}
	return openMission(ms, nil, nil, v, src, nil, nil)
}

func TestRefreshPackReadsTheArchiveOnceOverAnUnchangedContainer(t *testing.T) {
	addr := graphicsPrefix + data.ItemIconPath(data.ItemCode(packCode1))
	reads := 0
	src := countingSource{missionSource{addr: packIconStream(0xff, 0, 0)}, &reads}

	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{packCode1}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}

	mw := grabWorld(t, w, 7, src)
	baseline := reads
	if baseline == 0 {
		t.Fatal("reads after openMission = 0, want at least one — the one carried code's icon")
	}

	mw.refreshPack()
	if reads != baseline {
		t.Errorf("reads after a refresh over an unchanged container = %d, want still %d — "+
			"DD-14's compare must skip the archive entirely", reads, baseline)
	}
}

func TestRefreshPackShowsAPurseChangeWithoutAnItemChange(t *testing.T) {
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3, Owner: sim.SelfSlot}}, nil, sim.Relations{}, nil,
		[]sim.Stock{{ID: 7, Items: []uint16{packCode1}}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	mw := grabWorld(t, w, 7, missionSource{})
	if !w.SetPurse(sim.SelfSlot, 500) {
		t.Fatal("SetPurse refused SelfSlot")
	}

	mw.refreshPack()

	if got := inventoryGold(mw.invSubject); got != 500 {
		t.Fatalf("inventoryGold = %d, want 500", got)
	}
	if got := mw.invSubject.PackCount; len(got) != 2 || got[0] != 1 || got[1] != 500 {
		t.Fatalf("PackCount = %v, want the item count followed by gold 500", got)
	}
}

func TestSwitchingInventoryHeroesTransfersNoCharacterState(t *testing.T) {
	const (
		danathID     sim.EntityID = 7
		reniestaID   sim.EntityID = 8
		danathItem                = uint16(0x1111)
		reniestaItem              = uint16(0x2222)
		danathWorn                = uint16(0x1331)
		reniestaWorn              = uint16(0x2442)
	)
	danath := sim.Entity{ID: danathID, X: 3, Y: 3, Owner: sim.SelfSlot,
		HP: 80, MaxHP: 90, Skill: [data.SkillSlots]int32{1, 11, 12, 13, 14, 15},
		SkillXP: [data.SkillSlots]int32{101, 102, 103, 104, 105, 106}}
	reniesta := sim.Entity{ID: reniestaID, X: 4, Y: 3, Owner: sim.SelfSlot,
		HP: 50, MaxHP: 60, MaxMana: 70, Mana: 65,
		Skill:   [data.SkillSlots]int32{2, 21, 22, 23, 24, 25},
		SkillXP: [data.SkillSlots]int32{201, 202, 203, 204, 205, 206}}
	var danathEq, reniestaEq [sim.EquipSlots]uint16
	danathEq[0], reniestaEq[7] = danathWorn, reniestaWorn
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{danath, reniesta}, nil, sim.Relations{}, nil,
		[]sim.Stock{
			{ID: danathID, Items: []uint16{danathItem, danathItem}, Equipped: danathEq},
			{ID: reniestaID, Items: []uint16{reniestaItem}, Equipped: reniestaEq},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	if !w.SetPurse(sim.SelfSlot, 345) {
		t.Fatal("SetPurse refused SelfSlot")
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Danath", PlayerCharacter: true, StartingHero: true,
				FigureDir: string(data.FigureDirManFighter), FigureFace: 1,
				Hero: data.Hero{Skill: danath.Skill}, Worn: danathEq},
			{Name: "Reniesta", PlayerCharacter: true, CompanionNPC: 22, Mage: true,
				FigureDir: string(data.FigureDirWomanMage), FigureFace: 1,
				Hero: data.Hero{Skill: reniesta.Skill}, Worn: reniestaEq},
		},
		Start: mapload.Start{IDs: []sim.EntityID{danathID, reniestaID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary before selection: %v", err)
	}

	mw.switchInventorySubject(uint32(reniestaID))
	mw.refreshEquipment()
	mw.refreshAppearance()
	mw.rearm()
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after selection: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("selecting Reniesta and running presentation refreshes changed the simulation byte form")
	}
	if mw.invSubject.ID != uint32(reniestaID) || !mw.invParty.mage || mw.invParty.figureDir != data.FigureDirWomanMage {
		t.Fatalf("selected inventory subject = id %d, mage %v, figure %q; want Reniesta's own presentation",
			mw.invSubject.ID, mw.invParty.mage, mw.invParty.figureDir)
	}
	if got := mw.invCodes; len(got) != 1 || !sim.StackStateEqual(got[0], sim.ItemStack{Code: reniestaItem, Count: 1}) {
		t.Fatalf("Reniesta inventory stacks = %v, want only her own %#04x", got, reniestaItem)
	}
	if got := inventoryGold(mw.invSubject); got != 0 {
		t.Fatalf("Reniesta inventory displays gold %d, want none: the purse belongs to the primary surface", got)
	}

	mw.switchInventorySubject(uint32(danathID))
	final, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after switching back: %v", err)
	}
	if !bytes.Equal(final, before) {
		t.Fatal("switching back to Danath changed the simulation byte form")
	}
	if got := inventoryGold(mw.invSubject); got != 345 {
		t.Fatalf("Danath inventory displays gold %d, want shared purse 345", got)
	}
	entities := w.Entities()
	if entities[0].Skill != danath.Skill || entities[0].SkillXP != danath.SkillXP {
		t.Fatalf("Danath state after both selections = skill %v xp %v; want %v and %v",
			entities[0].Skill, entities[0].SkillXP, danath.Skill, danath.SkillXP)
	}
	if entities[1].Skill != reniesta.Skill || entities[1].SkillXP != reniesta.SkillXP {
		t.Fatalf("Reniesta state after both selections = skill %v xp %v; want %v and %v",
			entities[1].Skill, entities[1].SkillXP, reniesta.Skill, reniesta.SkillXP)
	}
}

// TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure is
// counterexample 2's own witness (round-2 adversarial review): a companion
// whose starting weapon was never folded into the array — rosterTemplate's
// own shape (pkg/mapload/spawn.go), Worn[0] empty and Weapon set — answers
// SlotInfo for slot 1 exactly as buildInventorySubject already draws the
// figure wearing it. invFigureEquipment (refreshEquipment's own guard) reads
// the fallback; invEquipment (rearm's own guard) reads the raw array and
// must NOT, or Rearm's own everEquipped-gated fallback
// (pkg/mapload/loadout.go) would be disabled for exactly the member it
// exists to cover.
func TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure(t *testing.T) {
	const allyID sim.EntityID = 9
	fallbackWeapon := data.ItemCode(0x1551)
	var allyEq [sim.EquipSlots]uint16 // slot 1 (index 0) stays empty
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: allyID, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 40, MaxHP: 40}}, nil,
		sim.Relations{}, nil,
		[]sim.Stock{{ID: allyID, Equipped: allyEq}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Ally", PlayerCharacter: true, CompanionNPC: 30,
				FigureDir: string(data.FigureDirManFighter), FigureFace: 1,
				Weapon: &data.Weapon{Code: fallbackWeapon}, Worn: allyEq},
		},
		Start: mapload.Start{IDs: []sim.EntityID{allyID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	if !mw.invSubjectSet {
		t.Fatal("openMission set no inventory subject over a one-member party")
	}
	if len(mw.invSubject.SlotInfo[0]) == 0 {
		t.Error("openMission's own construction: SlotInfo[0] is empty over a fallback-worn slot 1")
	}
	// WeaponFallback (counterexample 4, round-2 adversarial review, third
	// pass): command.go reads this at press time to refuse arming a drag for
	// slot 1 while nothing in the live entity's own equipment array backs it.
	// openMission must set it true here, the same frame it widens SlotInfo
	// through the fallback.
	if !mw.invSubject.WeaponFallback {
		t.Error("openMission's own construction: WeaponFallback is false over a fallback-worn slot 1")
	}

	mw.switchInventorySubject(uint32(allyID))
	if len(mw.invSubject.SlotInfo[0]) == 0 {
		t.Error("SlotInfo[0] is empty over a fallback-worn slot 1")
	}
	if !mw.invSubject.WeaponFallback {
		t.Error("switchInventorySubject: WeaponFallback is false over a fallback-worn slot 1")
	}
	if occupied, _ := mw.invFigureEquipment.Occupied(1); !occupied {
		t.Error("invFigureEquipment reads slot 1 unoccupied despite the Weapon fallback")
	}
	if occupied, _ := mw.invEquipment.Occupied(1); occupied {
		t.Error("invEquipment (rearm's own tracker) widened past the raw array")
	}

	// refreshEquipment's own guard compares against the SAME fallback-aware
	// equipment invFigureEquipment was just seeded with. A guard reading the
	// raw array instead would see the two disagree (nothing changed, but
	// the compare would say otherwise) and recompose SlotInfo from the raw
	// array, erasing what switchInventorySubject just set.
	mw.refreshEquipment()
	if len(mw.invSubject.SlotInfo[0]) == 0 {
		t.Error("refreshEquipment erased SlotInfo[0] over an unchanged fallback-worn slot 1")
	}
	if !mw.invSubject.WeaponFallback {
		t.Error("refreshEquipment: WeaponFallback is false over an unchanged fallback-worn slot 1")
	}

	// Put the same code into the real slot through the simulation, then take
	// it off again. The figure's code is unchanged at the first transition
	// (fallback code to equal real code), so refreshEquipment must compare
	// WeaponFallback as well as the equipment value. Once the real slot has
	// been observed, the sticky history prevents the fallback from returning
	// after the unequip.
	if !w.ReplaceStock(sim.Stock{ID: allyID, Items: []uint16{uint16(fallbackWeapon)}}) {
		t.Fatal("ReplaceStock refused the live ally")
	}
	sim.Step(w, []sim.Command{{Kind: sim.KindEquip, Entity: allyID, X: 0, Y: 1}})
	mw.rearm()
	mw.refreshEquipment()
	if mw.invSubject.WeaponFallback {
		t.Error("WeaponFallback stayed true after slot 1 became real with the same code")
	}
	if !mw.invWeaponEverEquipped {
		t.Fatal("invWeaponEverEquipped stayed false after slot 1 became occupied")
	}
	// rearm's own write reaches ms.Party[0] itself, not only mw's own in-memory
	// tracker (round-2 adversarial review, fifth pass, counterexamples A and
	// B): mw.mission.party IS ms.Party, the same backing array FinishMission
	// later hands to mapload.CarryParty, so this is the write that has to
	// survive the mission boundary.
	if !ms.Party[0].WeaponMaterialized {
		t.Fatal("rearm's real occupation did not persist WeaponMaterialized onto ms.Party[0]")
	}

	sim.Step(w, []sim.Command{{Kind: sim.KindUnequip, Entity: allyID, X: 1}})
	mw.rearm()
	mw.refreshEquipment()
	if mw.invSubject.WeaponFallback {
		t.Error("WeaponFallback returned after a real slot-1 item was unequipped")
	}
	if occupied, _ := mw.invFigureEquipment.Occupied(1); occupied {
		t.Error("invFigureEquipment reintroduced the starting weapon after a real unequip")
	}
	if len(mw.invSubject.SlotInfo[0]) != 0 {
		t.Error("SlotInfo[0] still describes the starting weapon after a real unequip")
	}
}

// TestRefreshAppearanceAgreesWithTheDollOverAFallbackWeapon is
// counterexample I (round-2 adversarial review, fifth pass):
// refreshAppearance used to derive the world-sprite body from
// mw.currentEquipment, the raw array alone, while refreshEquipment already
// derived the doll from mw.currentFigureEquipment — the array widened by
// the starting weapon's own fallback for a member whose worn set never folds
// it in (rosterTemplate's own shape, pkg/mapload/spawn.go, DIV-070's
// population). The two surfaces disagreed: the doll drew him holding the
// weapon, the figure standing on the map drew him bare-handed.
// mw.bodyEquipment is refreshAppearance's own comparison tracker and is
// checked directly, rather than through data.HeroAppearance's resolved body
// name, because this fixture supplies no body archive (mw.units is nil) —
// refreshAppearance still writes bodyEquipment before that nil guard, so the
// tracker alone witnesses which equipment the derivation actually ran on.
func TestRefreshAppearanceAgreesWithTheDollOverAFallbackWeapon(t *testing.T) {
	const allyID sim.EntityID = 9
	fallbackWeapon := data.ItemCode(0x1552)
	var allyEq [sim.EquipSlots]uint16 // slot 1 (index 0) stays empty
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: allyID, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 40, MaxHP: 40}}, nil,
		sim.Relations{}, nil,
		[]sim.Stock{{ID: allyID, Equipped: allyEq}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Ally", PlayerCharacter: true, CompanionNPC: 30,
				FigureDir: string(data.FigureDirManFighter), FigureFace: 1,
				Weapon: &data.Weapon{Code: fallbackWeapon}, Worn: allyEq},
		},
		Start: mapload.Start{IDs: []sim.EntityID{allyID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	if !mw.invSubjectSet {
		t.Fatal("openMission set no inventory subject over a one-member party")
	}
	if occupied, _ := mw.invFigureEquipment.Occupied(1); !occupied {
		t.Fatal("setup: invFigureEquipment (the doll's own source) does not carry the fallback weapon")
	}

	mw.refreshAppearance()

	occupied, ok := mw.bodyEquipment.Occupied(1)
	if !occupied || !ok {
		t.Fatal("refreshAppearance's own bodyEquipment does not carry the fallback weapon; the world sprite would draw the member bare-handed while the doll shows him armed")
	}
	code, ok := mw.bodyEquipment.Code(1)
	if !ok || code != fallbackWeapon {
		t.Errorf("bodyEquipment slot 1 = %v, %v, want the fallback weapon's own code %d, true", code, ok, fallbackWeapon)
	}
	if mw.bodyEquipment != mw.invFigureEquipment {
		t.Errorf("bodyEquipment = %+v, invFigureEquipment (the doll's own source) = %+v, want them equal", mw.bodyEquipment, mw.invFigureEquipment)
	}
}

// TestRefreshDollDragEquipmentSourceAgreesWithTheOrdinaryFigure is
// counterexample 3 (round-2 adversarial review, third pass): refreshDollDrag
// (world.go) composes the suppressed doll from mw.currentFigureEquipment,
// the same fallback-widened source refreshEquipment already composes the
// ordinary doll from — not mw.currentEquipment, the raw array alone. For a
// member whose starting weapon is carried only through PartyMember.Weapon
// (rosterTemplate's own shape, slot 1 never folded into the array),
// suppressing a DIFFERENT slot (2) must still show the slot-1 weapon in the
// recomposed figure and mask, matching the ordinary picture — DIV-085's "a
// hit test and a drawn pixel can never disagree," restated for the
// suppressed doll against the ordinary one rather than against itself.
//
// This calls refreshDollDrag's own composition seam over a live mapWorld.
// The pointer state remains private to package ui; the source choice and the
// resulting picture/mask pair are exercised here rather than reproduced in
// the test.
func TestRefreshDollDragEquipmentSourceAgreesWithTheOrdinaryFigure(t *testing.T) {
	const allyID sim.EntityID = 9
	fallbackWeapon := data.ItemCode(0x1551)
	wornCode := invWornCode(2)
	var allyEq [sim.EquipSlots]uint16
	allyEq[1] = uint16(wornCode) // slot 2, one-based; slot 1 stays empty in the array

	src := missionSource{
		graphicsPrefix + data.ItemFigureBasePath(data.FigureDirManFighter, 1):               invBaseSheet(),
		graphicsPrefix + data.ItemFigureLayerPath(data.FigureDirManFighter, fallbackWeapon): invSparseSheet(color.RGBA{B: 0xff, A: 0xff}, 0), // slot 1's own pixel, (0,0)
		graphicsPrefix + data.ItemFigureLayerPath(data.FigureDirManFighter, wornCode):       invSparseSheet(color.RGBA{G: 0xff, A: 0xff}, 1), // slot 2's own pixel, (1,0)
	}
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: allyID, X: 3, Y: 3, Owner: sim.SelfSlot, HP: 40, MaxHP: 40}}, nil,
		sim.Relations{}, nil,
		[]sim.Stock{{ID: allyID, Equipped: allyEq}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Ally", PlayerCharacter: true, CompanionNPC: 30,
				FigureDir: string(data.FigureDirManFighter), FigureFace: 1,
				Weapon: &data.Weapon{Code: fallbackWeapon}, Worn: allyEq},
		},
		Start: mapload.Start{IDs: []sim.EntityID{allyID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), src, nil, nil)
	if !mw.invSubjectSet {
		t.Fatal("openMission set no inventory subject over a one-member party")
	}

	// Prove the fixture distinguishes the raw source from the required one.
	raw := mw.currentEquipment()
	raw.SetCode(2, 0)
	rawComposed, _ := composeInventorySubject(mw.mission.src, mw.invSubject.ID, raw, mw.invParty.figureDir, mw.invParty.figureFace)
	if _, ok := rawComposed.SlotMask.At(0, 0); ok {
		t.Fatal("setup: currentEquipment (raw) already names a slot at (0,0) — the fixture's own weapon pixel does not isolate the fallback")
	}

	suppressed := mw.suppressedDollSubject(2)
	if n, ok := suppressed.SlotMask.At(0, 0); !ok || n != 1 {
		t.Errorf("suppressedDollSubject(2) mask.At(0,0) = (%d,%v), want (1,true): slot 1's fallback weapon must remain while slot 2 is suppressed", n, ok)
	}
	if px := suppressed.Figure.RGBAAt(0, 0); px.B != 0xff {
		t.Errorf("suppressedDollSubject(2) figure (0,0) = %+v, want slot 1's own blue — the fallback weapon layer", px)
	}
}

// Inventory presentation follows every party member whose own stock can be
// shown. Persistence is not the boundary: campaign purse/documents are
// primary-only, but a temporary ally or mercenary still owns their doll,
// worn slots and pack while selected.
func TestEntityDrawCarriesTheLiveWeaponSpellIntervalToTheSheet(t *testing.T) {
	w, err := sim.NewSpelledWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 100, MaxHP: 100, MaxMana: 50, Mana: 50,
			DamageBase: 2, WeaponSpell: 1, WeaponSpellLevel: 10}}, nil,
		[]sim.SpellRule{{ID: 1, DamageMin: 15, DamageMax: 18, MaxRange: 5,
			TargetsUnit: true, Damaging: true}})
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	m := worldFixtureMap()
	mw := newMapWorld(w, nil, nil, worldFixtureViewer(t, m))
	draws := mw.entityDraws()
	if len(draws) != 1 {
		t.Fatalf("entityDraws length = %d, want 1", len(draws))
	}
	c := draws[0].Combat
	if c.DamageBase != 2 || c.DamageSpread != 0 {
		t.Fatalf("physical damage = %d + %d, want staff's own 2 + 0", c.DamageBase, c.DamageSpread)
	}
	if !c.WeaponSpellKnown || !c.WeaponSpellDamageKnown || c.SpellDamageBase != 20 || c.SpellDamageSpread != 4 {
		t.Fatalf("weapon-spell damage = %d + %d, known %v; want live interval 20 + 4, true",
			c.SpellDamageBase, c.SpellDamageSpread, c.WeaponSpellDamageKnown)
	}
}

func TestEntityDrawSuppressesStoneCursePhysicalDamage(t *testing.T) {
	rule := sim.SpellRule{ID: 20, MaxRange: 5, TargetsUnit: true, SpellDuration: 10,
		EffectKind: sim.EffectAbsorption, EffectMode: sim.EffectDuration}
	w, err := sim.NewSpelledWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 7, X: 3, Y: 3, HP: 100, MaxHP: 100, MaxMana: 50, Mana: 50,
			DamageBase: 20, WeaponSpell: 20, WeaponSpellLevel: 1}}, nil, []sim.SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, nil, worldFixtureViewer(t, worldFixtureMap()))
	before := w.Hash()
	draws := mw.entityDraws()
	if len(draws) != 1 {
		t.Fatalf("entityDraws length = %d, want 1", len(draws))
	}
	c := draws[0].Combat
	if !c.WeaponSpellKnown || c.WeaponSpellDamageKnown {
		t.Fatalf("Stone Curse combat = %+v; want spell release without damage", c)
	}
	if got := w.Hash(); got != before {
		t.Fatalf("panel projection changed World hash: %#x, want %#x", got, before)
	}
}

func TestTownPanelCarriesStoneCurseWithoutPhysicalDamage(t *testing.T) {
	table, weapon := staffTooltipTable(t, 20, "Stone Curse", 1, 5, 0, 0, 10)
	member := mapload.PartyMember{Mage: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true},
		Hero:   data.Hero{Body: 30, Reaction: 22, Mind: 51, Spirit: 36},
		Weapon: weapon}
	member.Worn[0] = uint16(weapon.Code)
	c := partyPanelSubject(member, table).Combat
	if !c.WeaponSpellKnown || c.WeaponSpellDamageKnown {
		t.Fatalf("town Stone Curse combat = %+v; want a non-damaging release", c)
	}
}

func TestEntityDrawCarriesTheOriginalConditionalPanelOperands(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, nil, []sim.Entity{
			{ID: 7, X: 3, Y: 3, HP: 10, MaxHP: 10, TypeID: 0x49, XPValue: 81},
			{ID: 8, X: 4, Y: 3, HP: 10, MaxHP: 10, TypeID: 0x17, XPValue: 999,
				SkillXP: [6]int32{9000, 2700}},
		})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	draws := seamDraws(w, nil)
	if len(draws) != 2 {
		t.Fatalf("entityDraws length = %d, want 2", len(draws))
	}
	if got, want := draws[0].OriginalPanel,
		(ui.OriginalPanelActor{Known: true, XPValue: 81, Byte14A: 2}); got != want {
		t.Errorf("type 0x49 actor projection = %+v, want %+v", got, want)
	}
	if got, want := draws[1].OriginalPanel,
		(ui.OriginalPanelActor{Known: true, Flags: 0x10, XPValue: 27}); got != want {
		t.Errorf("type 0x17 humanoid projection = %+v, want %+v", got, want)
	}

	player := originalPanelActor(true, 0x49, 81)
	if player.Flags != 0x1 || player.Byte14A != 2 || player.XPValue != 81 {
		t.Errorf("player projection = %+v, want bit 0 plus the independently carried values", player)
	}
	params := make([]int32, 38)
	for i := range params {
		params[i] = -1
	}
	params[0x1d], params[37] = 0x49, 81
	table := &mapload.Table{Units: originalPanelCollection{{}, {name: "type49", params: params}}}
	if got := partyOriginalPanelXPValue(mapload.PartyMember{Class: 0x49}, table, 0); got != 81 {
		t.Errorf("town flat-unit XP projection = %d, want Units-row 81", got)
	}
}

type originalPanelEntry struct {
	name   string
	params []int32
}

type originalPanelCollection []originalPanelEntry

func (c originalPanelCollection) Len() int                  { return len(c) }
func (c originalPanelCollection) EntryName(i int) string    { return c[i].name }
func (c originalPanelCollection) EntryParams(i int) []int32 { return c[i].params }
func (c originalPanelCollection) EntryStrings(int) []string { return nil }

func TestTemporaryPartyMembersKeepTheirOwnInventorySubjects(t *testing.T) {
	const (
		allyItem = uint16(0x1111)
		mercItem = uint16(0x2222)
		allyWorn = uint16(0x1331)
		mercWorn = uint16(0x2442)
	)
	var allyEquipment, mercEquipment [sim.EquipSlots]uint16
	allyEquipment[6], mercEquipment[7] = allyWorn, mercWorn
	w, err := sim.NewStockedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, sim.Terrain{}, []sim.Entity{{ID: 7}, {ID: 8}, {ID: 9}}, nil,
		sim.Relations{}, nil, []sim.Stock{
			{ID: 8, Items: []uint16{allyItem}, Equipped: allyEquipment},
			{ID: 9, Items: []uint16{mercItem}, Equipped: mercEquipment},
		})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Danath", PlayerCharacter: true, StartingHero: true},
			{Name: "Sarindar", Worn: allyEquipment},
			{Name: "Mercenary", MercenaryType: 1, Worn: mercEquipment},
		},
		Start: mapload.Start{IDs: []sim.EntityID{7, 8, 9}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary before temporary selections: %v", err)
	}
	for _, tc := range []struct {
		id        uint32
		item      uint16
		equipment [sim.EquipSlots]uint16
	}{{8, allyItem, allyEquipment}} {
		id := tc.id
		mw.switchInventorySubject(id)
		if got := mw.invSubject.ID; got != id {
			t.Fatalf("selecting temporary party id %d left inventory subject at %d", id, got)
		}
		if mw.invParty.primary {
			t.Fatalf("temporary party id %d was marked campaign-primary", id)
		}
		if got := mw.invCodes; len(got) != 1 || !sim.StackStateEqual(got[0], sim.ItemStack{Code: tc.item, Count: 1}) {
			t.Fatalf("temporary party id %d pack = %v, want only %#04x", id, got, tc.item)
		}
		if got := mw.currentEquipment(); got != equipmentFromSlots(tc.equipment) {
			t.Fatalf("temporary party id %d worn = %v, want %v", id, got, tc.equipment)
		}
		if got := inventoryGold(mw.invSubject); got != 0 {
			t.Fatalf("temporary party id %d displays campaign gold %d", id, got)
		}
	}
	// The hired man is the same fixture entry the loop above used to walk, and
	// selecting him must leave the subject on the ally the loop last selected.
	mw.switchInventorySubject(9)
	if got := mw.invSubject.ID; got != 8 {
		t.Fatalf("selecting hired party id 9 moved the inventory subject to %d, want it left at 8", got)
	}
	if got := mw.invCodes; len(got) != 1 || !sim.StackStateEqual(got[0], sim.ItemStack{Code: allyItem, Count: 1}) {
		t.Fatalf("selecting hired party id 9 left pack %v, want the ally's %#04x", got, allyItem)
	}
	if got := mw.currentEquipment(); got != equipmentFromSlots(allyEquipment) {
		t.Fatalf("selecting hired party id 9 left worn %v, want the ally's %v", got, allyEquipment)
	}
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after temporary selections: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("temporary party selection changed simulation state")
	}
}

// Gold is Player-level state, and quest documents are campaign state presented
// through the primary hero. The actor taking the sack can therefore be the
// selected companion while both surfaces still resolve through Danath.
func TestACompanionPickupCreditsThePrimaryPurseAndDocumentSurface(t *testing.T) {
	const ordinary = uint16(0x1111)
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{
			{ID: 7, X: 2, Y: 2, Owner: sim.SelfSlot},
			{ID: 8, X: 3, Y: 3, Owner: sim.SelfSlot},
		}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 40,
			Items: []uint16{ordinary, uint16(data.QuestDocumentCode), uint16(data.QuestDocumentCode)}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{
			{Name: "Danath", PlayerCharacter: true, StartingHero: true},
			{Name: "Reniesta", PlayerCharacter: true, CompanionNPC: 22, Mage: true},
		},
		Start: mapload.Start{IDs: []sim.EntityID{7, 8}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	mw.switchInventorySubject(8)
	mw.grab(0, 0, 0, false)
	mw.refreshPack()

	if got := w.Purse(sim.SelfSlot); got != 40 {
		t.Fatalf("shared purse after Reniesta pickup = %d, want 40", got)
	}
	if got := inventoryGold(mw.invSubject); got != 0 {
		t.Fatalf("Reniesta inventory displays gold %d after her pickup, want none", got)
	}
	companion, _ := w.Carried(8)
	if !reflect.DeepEqual(companion, []uint16{ordinary}) {
		t.Fatalf("Reniesta carried = %#v, want only ordinary item %#04x", companion, ordinary)
	}
	primary, _ := w.Carried(7)
	wantDocuments := []uint16{uint16(data.QuestDocumentCode), uint16(data.QuestDocumentCode)}
	if !reflect.DeepEqual(primary, wantDocuments) {
		t.Fatalf("Danath carried = %#v, want campaign document stack %#v", primary, wantDocuments)
	}

	mw.switchInventorySubject(7)
	if got := inventoryGold(mw.invSubject); got != 40 {
		t.Fatalf("Danath inventory displays gold %d after companion pickup, want 40", got)
	}
	if got := mw.invCodes; len(got) != 1 || !sim.StackStateEqual(got[0], sim.ItemStack{Code: uint16(data.QuestDocumentCode), Count: 2}) {
		t.Fatalf("Danath inventory stacks = %v, want one document element at count 2", got)
	}
}

func TestDocumentPickupWithoutAPrimaryHeroIsAtomic(t *testing.T) {
	doc := uint16(data.QuestDocumentCode)
	w, err := sim.NewStockedWorld(1,
		sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 8, X: 3, Y: 3, Owner: sim.SelfSlot}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{0x1111, doc}}}, nil)
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{Name: "Reniesta", PlayerCharacter: true, CompanionNPC: 22}},
		Start: mapload.Start{IDs: []sim.EntityID{8}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)
	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary before pickup: %v", err)
	}

	mw.grab(0, 0, 0, false)
	after, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after pickup: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("document pickup without a valid primary surface changed canonical state")
	}
	if got := w.Purse(sim.SelfSlot); got != 0 {
		t.Errorf("refused pickup credited purse %d, want 0", got)
	}
	if sacks := w.Sacks(); len(sacks) != 1 || len(sacks[0].Items) != 2 {
		t.Fatalf("refused pickup partially moved the sack: %+v", sacks)
	}
}

func TestMapWorldGrabTakesTheSackUnderTheSubjectEntity(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 7, X: 3, Y: 3}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 40, Items: []uint16{9, 5}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	mw := grabWorld(t, w, 7, missionSource{})

	mw.grab(0, 0, 0, false)

	codes, ok := w.Carried(7)
	if !ok || len(codes) != 2 || codes[0] != 9 || codes[1] != 5 {
		t.Fatalf("Carried(7) = %v, %v, want [9 5], true — sack order preserved", codes, ok)
	}
	if got := w.Sacks(); len(got) != 0 {
		t.Errorf("Sacks() = %v, want none — the grab took the only one on the map", got)
	}
}

// AC-11: the key is inert — not an error, not a change of any kind — when
// the subject entity stands on no sack.
func TestMapWorldGrabIsInertWhenTheSubjectStandsOnNoSack(t *testing.T) {
	w, err := sim.NewWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 7, X: 3, Y: 3}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	before := w.Hash()

	mw := grabWorld(t, w, 7, missionSource{})
	mw.grab(0, 0, 0, false)

	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d — grab must be a no-op with no sack under the subject",
			before, got)
	}
	if codes, ok := w.Carried(7); !ok || len(codes) != 0 {
		t.Errorf("Carried(7) = %v, %v, want none, true", codes, ok)
	}
}

// Spec: guard the empty-party case. Entity 0 is a REAL entity that may carry
// something, and a mission with no party leaves invSubjectSet false rather
// than a subject whose id happens to be 0 — so grab must not reach entity 0
// through that zero value even when entity 0 exists and stands on a sack.
func TestMapWorldGrabIsInertWithNoPartyEvenWhenEntityZeroStandsOnASack(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: 0, X: 3, Y: 3}}, nil, sim.Relations{},
		[]sim.Sack{{X: 3, Y: 3, Gold: 5, Items: []uint16{9}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	before := w.Hash()

	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w} // no Party, no Start.IDs
	mw := openMission(ms, nil, nil, v, missionSource{}, nil, nil)

	mw.grab(0, 0, 0, false)

	if got := w.Hash(); got != before {
		t.Errorf("World.Hash() changed from %d to %d — a party-less mission must not grab through entity 0",
			before, got)
	}
}

// The hero rule. A mission with no living hero is lost, ahead of both script
// counters — and a mission with no recorded hero is untouched by any of
// it.

// heroWorld is missionWorld with entities: the same checks, the same trigger
// shape, and the same script construction, so every fixture already proven by
// the outcome-notice tests above answers here too. It exists because the hero
// rule is the one part of this file that needs both a script AND a placed
// entity in the same world, which missionWorld alone cannot build.
func heroWorld(t *testing.T, instants []sim.ScriptInstant, triggers []sim.ScriptTrigger, ents []sim.Entity) *sim.World {
	t.Helper()
	s, err := sim.NewScript(missionChecks(), instants, triggers)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH},
		sim.ModeCanonical, make([]byte, worldFixtureW*worldFixtureH), ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w
}

// AC-7: the world no longer holding the recorded hero's entity is a loss.
func TestMissionWithTheHeroRemovedFromTheWorldReportsALoss(t *testing.T) {
	w := heroWorld(t, nil, nil, nil) // no script, no entities: 99 was never here
	mw := grabWorld(t, w, 99, missionSource{})

	mw.tick()
	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeLost {
		t.Fatalf("announced=%v outcome=%v, want true/OutcomeLost — the hero's entity is gone",
			mw.mission.announced, mw.mission.outcome)
	}
}

// AC-8: a hero at zero health with a health system is DOWNED, not dead. Heal
// can still raise him, so he does not lose the mission.
func TestMissionWithTheHeroDownedIsNotLostWhileHeCanBeHealed(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{{ID: 99, Owner: sim.SelfSlot, HP: 0, MaxHP: 10}})
	mw := grabWorld(t, w, 99, missionSource{})

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced=true outcome=%v, want no report — a downed hero can still be healed",
			mw.mission.outcome)
	}
}

// AC-9: a living hero does not trip the rule, and the winning counter decides
// the mission exactly as it always has.
func TestMissionWithTheHeroAliveAndTheWinningCounterSatisfiedIsWon(t *testing.T) {
	w := heroWorld(t,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)},
		[]sim.Entity{{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10}})
	mw := grabWorld(t, w, 99, missionSource{})

	missionSteps(mw, 2)
	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeWon {
		t.Fatalf("announced=%v outcome=%v, want true/OutcomeWon — the hero rule must not fire on a living hero",
			mw.mission.announced, mw.mission.outcome)
	}
}

// AC-10: the hero test runs BEFORE Outcome() is read, so a world that has
// already latched Won reports the loss anyway when its recorded hero is dead
// at the moment settleNotices first looks. The world is advanced directly,
// outside of any driver, so "already reached" means what it says: the win is
// real world state before openMission or a single tick of mw.tick() exists.
func TestMissionWithTheHeroDeadAndAWinAlreadyReachedReportsTheLoss(t *testing.T) {
	w := heroWorld(t,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantWin}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)},
		[]sim.Entity{{ID: 99, Owner: sim.SelfSlot, HP: -10, MaxHP: 10}})
	for i := 0; i < 16 && w.Outcome() == sim.OutcomeUndecided; i++ {
		sim.Step(w, nil)
	}
	if w.Outcome() != sim.OutcomeWon {
		t.Fatalf("setup: world.Outcome() = %v, want OutcomeWon before the driver ever sees this world", w.Outcome())
	}

	mw := grabWorld(t, w, 99, missionSource{})
	mw.tick()
	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeLost {
		t.Fatalf("announced=%v outcome=%v, want true/OutcomeLost — the hero test runs before Outcome() is read",
			mw.mission.announced, mw.mission.outcome)
	}
}

func TestMissionWithNoPartyReportsNothingEvenWithADeadEntityZero(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{{ID: 0, HP: -1, MaxHP: 10}})
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w} // no Party, no Start.IDs
	mw := openMission(ms, nil, nil, v, missionSource{}, nil, nil)

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced = true with outcome %v, want no report — no party means no hero to test",
			mw.mission.outcome)
	}
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a party-less mission opened a notice")
	}
}

func partyWorld(t *testing.T, w *sim.World, party []mapload.PartyMember, ids []sim.EntityID) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	ms := &Mission{Number: 1, Map: m, World: w, Party: party, Start: mapload.Start{IDs: ids}}
	return openMission(ms, nil, nil, v, missionSource{}, nil, nil)
}

// townCompanionParty is the shape the campaign builds in town: the starting
// hero, then npc 22 as a full character who is not the starting hero
// (frontend.go's addChapterCompanions).
func townCompanionParty() []mapload.PartyMember {
	return []mapload.PartyMember{
		{ID: "hero", PlayerCharacter: true, StartingHero: true},
		{ID: "npc:22", PlayerCharacter: true},
	}
}

// A dead town companion loses the mission while the starting hero lives.
func TestMissionWithADeadTownCompanionReportsALoss(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{
		{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
		{ID: 100, Owner: sim.SelfSlot, HP: -10, MaxHP: 10},
	})
	mw := partyWorld(t, w, townCompanionParty(), []sim.EntityID{99, 100})

	mw.tick()
	if !mw.mission.announced || mw.mission.outcome != sim.OutcomeLost {
		t.Fatalf("announced=%v outcome=%v, want true/OutcomeLost — a player character other than "+
			"Start.IDs[0] died", mw.mission.announced, mw.mission.outcome)
	}
}

// The control for the test above: both alive is not a loss. Without it the
// test above would pass against an implementation that loses every mission
// with a second party member.
func TestMissionWithALivingTownCompanionIsNotLost(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{
		{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
		{ID: 100, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
	})
	mw := partyWorld(t, w, townCompanionParty(), []sim.EntityID{99, 100})

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced = true with outcome %v, want no report — every player character is alive",
			mw.mission.outcome)
	}
}

// A hired mercenary is a unit and not a character: its death is not the
// mission's. This is the AUTHORED half of the rule (world.go's guardedEntities),
// and it is asserted rather than left implicit because the party model's two
// fields are written by different constructors.
func TestMissionWithADeadHiredMercenaryIsNotLost(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{
		{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
		{ID: 100, Owner: sim.SelfSlot, HP: -1, MaxHP: 10},
	})
	party := []mapload.PartyMember{
		{ID: "hero", PlayerCharacter: true, StartingHero: true},
		{ID: "hire:1", Temporary: true, MercenaryType: 3},
	}
	mw := partyWorld(t, w, party, []sim.EntityID{99, 100})

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced = true with outcome %v, want no report — a hire is not a character",
			mw.mission.outcome)
	}
}

// The mercenary-type half of the rule, witnessed on its own.
//
// NO CONSTRUCTOR IN THIS TREE WRITES THIS SHAPE TODAY: the tavern writes
// MercenaryType and leaves PlayerCharacter clear, so the test above passes
// against an implementation that reads PlayerCharacter alone and its mutation
// does not redden. That is what this case is for — the second condition exists
// against a future constructor that sets both, and a condition no test can see
// is one a later reader may delete as dead.
func TestMissionWithADeadHireCarryingBothFlagsIsNotLost(t *testing.T) {
	w := heroWorld(t, nil, nil, []sim.Entity{
		{ID: 99, Owner: sim.SelfSlot, HP: 10, MaxHP: 10},
		{ID: 100, Owner: sim.SelfSlot, HP: -1, MaxHP: 10},
	})
	party := []mapload.PartyMember{
		{ID: "hero", PlayerCharacter: true, StartingHero: true},
		{ID: "hire:1", PlayerCharacter: true, Temporary: true, MercenaryType: 3},
	}
	mw := partyWorld(t, w, party, []sim.EntityID{99, 100})

	mw.tick()
	if mw.mission.announced {
		t.Fatalf("announced = true with outcome %v, want no report — MercenaryType decides against "+
			"PlayerCharacter", mw.mission.outcome)
	}
}

func TestAMapOpenedAsAMapOpensNoNoticeAfterManySteps(t *testing.T) {
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	mw := mustOpenMapWorld(t, m, nil, nil, v)

	missionSteps(mw, 20)
	if mw.mission != nil {
		t.Fatal("openMapWorld built a mission driver; FR-11 says a map gains no ending")
	}
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a plain map opened a notice after many steps")
	}
}

func TestTheHeroLossBannerOpensOnceAndDismissingGoesToTheMenu(t *testing.T) {
	w := heroWorld(t, nil, nil, nil) // the recorded hero is never placed at all
	mw := grabWorld(t, w, 99, missionSource{})

	mw.tick()
	if got, kind, open := mw.view.NoticeState(); !open || kind != ui.NoticeFailure || got != MissionLostText {
		t.Fatalf("notice = %q/%v/%v, want %q as an outcome", got, kind, open, MissionLostText)
	}

	missionSteps(mw, 3)
	if got, _, open := mw.view.NoticeState(); !open || got != MissionLostText {
		t.Fatalf("notice after further steps = %q (open %v), want the same banner still open", got, open)
	}

	dest, msg, _ := mw.advanceNotice()
	if dest != ui.NoticeToMenu || msg != "" {
		t.Fatalf("advance = %v/%q, want NoticeToMenu and no message", dest, msg)
	}
	if _, _, open := mw.view.NoticeState(); open {
		t.Fatal("dismissing the loss banner left it open")
	}
}
