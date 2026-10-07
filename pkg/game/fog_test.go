package game

import (
	"bytes"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var fogTestBounds = sim.Bounds{Width: 45, Height: 45}

// fogTestWorld builds a synthetic world over fogTestBounds holding ents,
// through the same exported constructor pkg/sim's own sight tests use
// (sightGoldenWorld and its neighbours) — never a decode, and never a field
// this package could not have reached from outside pkg/sim.
func fogTestWorld(t *testing.T, ents []sim.Entity) *sim.World {
	t.Helper()
	w, err := sim.NewTerrainWorld(1, fogTestBounds, sim.ModeCanonical, sim.Terrain{}, ents, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w
}

// fogTestViewer is a bare viewer over fogTestBounds' own extent — worldFixtureViewer's
// shape, sized for this file's own bounds rather than the package's 72x72
// fixture.
func fogTestViewer(t *testing.T) *ui.Viewer {
	t.Helper()
	w, h := int(fogTestBounds.Width), int(fogTestBounds.Height)
	v, err := ui.NewViewer("fog fixture", terrain.Grid{
		Width:  w,
		Height: h,
		Tiles:  make([]uint16, w*h),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	return v
}

// countState counts how many bytes of plane equal want.
func countState(plane []byte, want uint8) int {
	n := 0
	for _, v := range plane {
		if v == want {
			n++
		}
	}
	return n
}

func TestFreshFogPlaneIsEntirelyUnseen(t *testing.T) {
	const cols, rows = 6, 4
	p := newFogPlane(cols, rows)
	proj := p.project()
	if got, want := len(proj), cols*rows; got != want {
		t.Fatalf("project() returned %d byte(s), want %d (cols*rows)", got, want)
	}
	if n := countState(proj, ui.FogUnseen); n != cols*rows {
		t.Fatalf("a fresh plane answers FogUnseen at %d of %d cells, want all of them", n, cols*rows)
	}
	for i, v := range p.visible {
		if v != 0 {
			t.Fatalf("visible[%d] = %d on a fresh plane, want 0", i, v)
		}
	}
	for i, v := range p.explored {
		if v != 0 {
			t.Fatalf("explored[%d] = %d on a fresh plane, want 0", i, v)
		}
	}
}

// TestOneRefreshLightsSomeCellsAndLeavesOthersUnseen is AC-6, measured
// directly on fogPlane.refresh: a world holding one living entity owned by
// sim.SelfSlot, refreshed once, lights at least one cell and leaves at least
// one unseen — the flat-ground counts AC-2 gives for a range-5 entity (105 of
// 2025 cells) make both sides of that comparison non-vacuous on this file's
// own bounds.
func TestOneRefreshLightsSomeCellsAndLeavesOthersUnseen(t *testing.T) {
	w := fogTestWorld(t, []sim.Entity{
		{ID: 1, X: 22, Y: 22, Owner: sim.SelfSlot, ScanRange: 5},
	})
	p := newFogPlane(int(fogTestBounds.Width), int(fogTestBounds.Height))
	p.refresh(w, sim.SelfSlot)

	proj := p.project()
	visible := countState(proj, ui.FogVisible)
	unseen := countState(proj, ui.FogUnseen)
	if visible == 0 {
		t.Error("one refresh over a world with one owned entity lit no cell at all")
	}
	if unseen == 0 {
		t.Error("one refresh lit the whole plane, leaving nothing unseen — this fixture's bounds are meant to leave most of the map dark")
	}
	if got, want := visible, 105; got != want {
		t.Errorf("a range-5 entity on flat ground lights %d cell(s) through fogPlane.refresh, want %d (AC-2's own count)", got, want)
	}
}

// TestExploredNeverShrinksAndVacatedCellsStayExplored is AC-7: refreshing
// after the owned entity has moved away leaves its old cell FogExplored, not
// FogUnseen, and the explored set — measured over several further refreshes,
// alternating the entity's position — never shrinks.
//
// The move is modelled as two independent synthetic worlds, "before" and
// "after", refreshed on the SAME plane in turn: fogPlane.refresh does not
// care where its world came from, only what w.Sight(owner) answers, so this
// is the accumulation logic in isolation, with pkg/sim's own movement
// mechanics — which are that package's business and not this file's —
// nowhere in the loop.
func TestExploredNeverShrinksAndVacatedCellsStayExplored(t *testing.T) {
	const width = int(45)
	before := fogTestWorld(t, []sim.Entity{{ID: 1, X: 15, Y: 22, Owner: sim.SelfSlot, ScanRange: 4}})
	after := fogTestWorld(t, []sim.Entity{{ID: 1, X: 30, Y: 22, Owner: sim.SelfSlot, ScanRange: 4}})
	vacated := 22*width + 15 // the observer's own starting cell (marched unconditionally — sight.go's marchSight)

	p := newFogPlane(int(fogTestBounds.Width), int(fogTestBounds.Height))

	p.refresh(before, sim.SelfSlot)
	if p.visible[vacated] == 0 {
		t.Fatalf("the observer's own starting cell is not visible right after its own refresh")
	}
	counts := []int{countState(p.explored, 1)}

	p.refresh(after, sim.SelfSlot)
	if p.visible[vacated] != 0 {
		t.Errorf("the vacated cell is still FogVisible after the entity marched 15 cells away")
	}
	if p.explored[vacated] == 0 {
		t.Errorf("the vacated cell fell to unseen, want it to stay FogExplored (AC-7)")
	}
	counts = append(counts, countState(p.explored, 1))

	// Several more refreshes, alternating the two worlds, so the visible
	// layer keeps moving back and forth while explored is only ever asked
	// to grow.
	worlds := []*sim.World{before, after, before, after}
	for _, w := range worlds {
		p.refresh(w, sim.SelfSlot)
		counts = append(counts, countState(p.explored, 1))
	}

	for i := 1; i < len(counts); i++ {
		if counts[i] < counts[i-1] {
			t.Fatalf("the explored count fell from %d to %d at refresh %d — the explored set must never shrink (AC-7)",
				counts[i-1], counts[i], i)
		}
	}
	if counts[len(counts)-1] <= counts[0] {
		t.Errorf("the explored count never grew past its first refresh's %d over %d further refreshes — "+
			"this measurement is vacuous unless the set actually grows somewhere", counts[0], len(counts)-1)
	}
}

func TestProjectMapsToTheThreeUIFogConstants(t *testing.T) {
	p := newFogPlane(3, 1)
	// cell 0: neither layer — FogUnseen.
	// cell 1: explored only — FogExplored.
	p.explored[1] = 1
	// cell 2: visible (and, as any refresh would leave it, explored too) — FogVisible.
	p.explored[2] = 1
	p.visible[2] = 1

	got := p.project()
	want := []byte{ui.FogUnseen, ui.FogExplored, ui.FogVisible}
	if !bytes.Equal(got, want) {
		t.Fatalf("project() = %v, want %v", got, want)
	}
}

// TestAMissionOpensClosedAtTickZero is AC-6 read through the actual
// construction path rather than through fogPlane directly: newMapWorld
// builds and refreshes the fog plane BEFORE its own tick-0 push, so a
// mapWorld fresh off the constructor already shows the owned entity's
// surroundings and nothing else — the screen a mission opens on is closed
// by construction.
func TestAMissionOpensClosedAtTickZero(t *testing.T) {
	w := fogTestWorld(t, []sim.Entity{
		{ID: 1, X: 22, Y: 22, Owner: sim.SelfSlot, ScanRange: 5},
	})
	mw := newMapWorld(w, nil, nil, fogTestViewer(t))

	proj := mw.fog.project()
	visible := countState(proj, ui.FogVisible)
	unseen := countState(proj, ui.FogUnseen)
	if visible == 0 {
		t.Fatal("a freshly constructed mapWorld shows nothing visible at tick 0 (AC-6)")
	}
	if unseen == 0 {
		t.Fatal("a freshly constructed mapWorld shows the whole map at tick 0 — the screen did not open closed (AC-6, FR-10)")
	}
}

func TestFogRefreshesOnlyEveryFogPeriodWorldTicks(t *testing.T) {
	w := fogTestWorld(t, []sim.Entity{
		{ID: 1, X: 22, Y: 22, Owner: sim.SelfSlot, ScanRange: 3},
	})
	mw := newMapWorld(w, nil, nil, fogTestViewer(t))

	const farCell = 1*45 + 1 // (1,1): far outside a range-3 march from (22,22)
	if mw.fog.visible[farCell] != 0 {
		t.Fatalf("cell (1,1) is visible right after construction; this test needs a cell no refresh ever lights")
	}

	for i := 1; i < fogPeriod; i++ {
		mw.fog.visible[farCell] = 7
		mw.tick()
		if got := mw.fog.visible[farCell]; got != 7 {
			t.Fatalf("world tick %d (not a multiple of fogPeriod=%d) overwrote the sentinel — "+
				"a refresh ran on a tick it should not have", mw.world.Tick(), fogPeriod)
		}
	}

	mw.fog.visible[farCell] = 7
	mw.tick()
	if got := mw.world.Tick(); got != fogPeriod {
		t.Fatalf("this test drove the world to tick %d, want exactly %d — its own count is off", got, fogPeriod)
	}
	if got := mw.fog.visible[farCell]; got != 0 {
		t.Fatalf("world tick %d (a multiple of fogPeriod) left the sentinel in place — refresh did not run", fogPeriod)
	}
}
