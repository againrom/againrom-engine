package game

import (
	"math"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/render/camera"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// This release witness deliberately carries its own small projection oracle.
// It reads the lawful mission's raw altitude bytes and evaluates the published
// integer expressions directly. It does not call Projection, AnchorHeight,
// CellColumnBounds, groundCellAt, or another picker helper to obtain an
// expected cell. The synthetic tests in pkg/render/terrain pin the same
// arithmetic to literal values; this test pins the installed-data wiring.

func releaseGroundAltitude(m *alm.Map, col, row int) int {
	if col < 0 {
		col = 0
	} else if col >= m.Width {
		col = m.Width - 1
	}
	if row < 0 {
		row = 0
	} else if row >= m.Height {
		row = m.Height - 1
	}
	return int(int8(m.Altitudes[row*m.Width+col]))
}

func releaseGroundMinVertex(m *alm.Map) int {
	minV := -releaseGroundAltitude(m, 0, 0)
	for row := 0; row <= m.Height; row++ {
		for col := 0; col <= m.Width; col++ {
			v := row*camera.CellSize - releaseGroundAltitude(m, col, row)
			if v < minV {
				minV = v
			}
		}
	}
	return minV
}

func releaseCornerGroundCell(m *alm.Map, minV int, worldX, worldY float64) (col, row int, inside bool) {
	fc := math.Floor(worldX / camera.CellSize)
	if !(fc >= 0 && fc < float64(m.Width)) {
		return 0, 0, false
	}
	col = int(fc)
	local := int(math.Floor(worldX)) & (camera.CellSize - 1)
	for row = 0; row < m.Height; row++ {
		topLeft := row*camera.CellSize - releaseGroundAltitude(m, col, row) - minV
		topRight := row*camera.CellSize - releaseGroundAltitude(m, col+1, row) - minV
		bottomLeft := (row+1)*camera.CellSize - releaseGroundAltitude(m, col, row+1) - minV
		bottomRight := (row+1)*camera.CellSize - releaseGroundAltitude(m, col+1, row+1) - minV
		top := topLeft + (topRight-topLeft)*local/camera.CellSize
		bottom := bottomLeft + (bottomRight-bottomLeft)*local/camera.CellSize
		if float64(top) <= worldY && worldY <= float64(bottom) {
			return col, row, true
		}
	}
	return 0, 0, false
}

// releaseMeanGroundCell is the picker this hotfix replaces. Keeping it in the
// test gives the installed witness an explicit mutation discriminator; it is
// not used to calculate the expected corner-mesh answer.
func releaseMeanGroundCell(m *alm.Map, minV int, worldX, worldY float64) (col, row int, inside bool) {
	fc := math.Floor(worldX / camera.CellSize)
	if !(fc >= 0 && fc < float64(m.Width)) {
		return 0, 0, false
	}
	col = int(fc)
	found := false
	for r := 0; r < m.Height; r++ {
		h := releaseGroundAltitude(m, col, r) + releaseGroundAltitude(m, col+1, r) +
			releaseGroundAltitude(m, col, r+1) + releaseGroundAltitude(m, col+1, r+1)
		top := r*camera.CellSize - h/4 - minV
		if float64(top) <= worldY && worldY < float64(top+camera.CellSize) {
			row, found = r, true
		}
	}
	if !found {
		return 0, 0, false
	}
	return col, row, true
}

type releaseGroundWitness struct {
	x, y      int
	col, row  int
	oldCol    int
	oldRow    int
	oldInside bool
	worldX    float64
	worldY    float64
}

func releaseGroundWitnessPoint(t *testing.T, m *alm.Map, cam *camera.Camera, entities []sim.Entity,
	sacks []sim.Sack) releaseGroundWitness {
	t.Helper()
	if m.Width <= 0 || m.Height <= 0 || len(m.Altitudes) != m.Width*m.Height {
		t.Fatalf("mission ground grid = %dx%d with %d altitudes", m.Width, m.Height, len(m.Altitudes))
	}
	if !(cam.Zoom > 0) || math.IsNaN(cam.Zoom) || math.IsInf(cam.Zoom, 0) {
		t.Fatalf("mission camera zoom = %v", cam.Zoom)
	}

	minV := releaseGroundMinVertex(m)
	const safeMargin = 64
	for y := safeMargin; y < cam.ViewH-safeMargin; y++ {
		for x := safeMargin; x < cam.ViewW-safeMargin; x++ {
			worldX := float64(x)/cam.Zoom + cam.X
			worldY := float64(y)/cam.Zoom + cam.Y
			col, row, inside := releaseCornerGroundCell(m, minV, worldX, worldY)
			if !inside {
				continue
			}
			oldCol, oldRow, oldInside := releaseMeanGroundCell(m, minV, worldX, worldY)
			if oldInside && oldCol == col && oldRow == row {
				continue
			}

			// Keep the command witness on unoccupied ground. Four cells exceed
			// every ordinary mission-10 body footprint and keep a moving body's
			// placed rectangle away from the chosen pixel across the two input
			// frames.
			clear := true
			for _, e := range entities {
				if releaseAbs32(e.X-int32(col)) <= 4 && releaseAbs32(e.Y-int32(row)) <= 4 {
					clear = false
					break
				}
			}
			if !clear {
				continue
			}
			for _, s := range sacks {
				if s.X == int32(col) && s.Y == int32(row) {
					clear = false
					break
				}
			}
			if !clear {
				continue
			}

			return releaseGroundWitness{
				x: x, y: y, col: col, row: row,
				oldCol: oldCol, oldRow: oldRow, oldInside: oldInside,
				worldX: worldX, worldY: worldY,
			}
		}
	}
	t.Fatal("mission camera contains no unoccupied corner-mesh point that distinguishes the old mean picker")
	return releaseGroundWitness{}
}

func releaseAbs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestReleaseGroundPickerQueuesTheCornerMeshCellOnRealMissionContent starts at
// the production App input door and ends at mapWorld's simulation-command
// queue. The release gate runs it once against each lawful install. The point
// is selected only from a safe central viewport band and must distinguish the
// former mean-of-four picker before it is allowed to witness anything.
func openReleaseGroundMission(t *testing.T, label string) (*FrontEnd, *ui.App, *mapWorld, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App(label)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatalf("open mission 10: %v", err)
	}
	if f.live == nil || len(f.live.mission.ids) == 0 {
		t.Fatal("mission 10 opened without a live party")
	}
	live := f.live
	id := live.mission.ids[0]
	if got := live.view.Mode(); got != ui.ModeDisplaced {
		t.Fatalf("mission 10 viewer mode = %v, want displaced", got)
	}
	if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatalf("select %d through production input: %v", id, err)
	}
	return f, app, live, id
}

func TestReleaseGroundPickerQueuesTheCornerMeshCellOnRealMissionContent(t *testing.T) {
	f, app, live, id := openReleaseGroundMission(t, "div-044-ground-picker")
	_, controlApp, controlLive, controlID := openReleaseGroundMission(t, "div-044-ground-picker-control")
	if controlID != id {
		t.Fatalf("control party id = %d, want %d", controlID, id)
	}
	if got, want := controlLive.world.Hash(), live.world.Hash(); got != want {
		t.Fatalf("control digest before ground tap = %#016x, want %#016x", got, want)
	}

	addr, ok := MissionMap(10)
	if !ok {
		t.Fatal("mission 10 has no campaign map address")
	}
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatalf("decode %s for independent ground oracle: %v", addr, err)
	}
	w := releaseGroundWitnessPoint(t, m, live.view.Camera(), live.world.Entities(), live.world.Sacks())

	gotCol, gotRow, err := app.HeadlessDropCell(w.x, w.y)
	if err != nil {
		t.Fatalf("ground cell at frame point (%d,%d): %v", w.x, w.y, err)
	}
	if gotCol != w.col || gotRow != w.row {
		t.Fatalf("ground cell at frame (%d,%d), world (%.3f,%.3f) = (%d,%d), want independent corner-mesh cell (%d,%d); old mean = (%d,%d,%v)",
			w.x, w.y, w.worldX, w.worldY, gotCol, gotRow, w.col, w.row, w.oldCol, w.oldRow, w.oldInside)
	}

	beforePending := len(live.pending)
	controlPending := len(controlLive.pending)
	if err := app.HeadlessPointer("press", w.x, w.y); err != nil {
		t.Fatalf("press corner-mesh ground: %v", err)
	}
	if err := app.HeadlessPointer("release", w.x, w.y); err != nil {
		t.Fatalf("release corner-mesh ground: %v", err)
	}
	// Match the two production input frames on a control mission with no
	// pointer edge. The world advances on every App step, so comparing to the
	// pre-press digest would confuse ordinary ticks with command-queue state.
	// The paired drive isolates the only difference: one UI queue now holds the
	// target and the other does not.
	if err := controlApp.HeadlessStep(); err != nil {
		t.Fatalf("first control step: %v", err)
	}
	if err := controlApp.HeadlessStep(); err != nil {
		t.Fatalf("second control step: %v", err)
	}
	if len(live.pending) != beforePending+1 {
		t.Fatalf("corner-mesh tap queued %d commands, want 1", len(live.pending)-beforePending)
	}
	cmd := live.pending[len(live.pending)-1]
	if cmd.Entity != id || cmd.Kind != sim.KindGroupMoveTo || cmd.X != int32(w.col) || cmd.Y != int32(w.row) {
		t.Errorf("corner-mesh tap queued %+v, want entity %d KindGroupMoveTo at (%d,%d)", cmd, id, w.col, w.row)
	}
	if len(controlLive.pending) != controlPending {
		t.Fatalf("two no-input control frames queued %d commands, want 0", len(controlLive.pending)-controlPending)
	}
	if got, want := live.world.Hash(), controlLive.world.Hash(); got != want {
		t.Errorf("queued UI target and matched no-input control hash to %#016x and %#016x; the pending target reached canonical state before advance", got, want)
	}
	t.Logf("%s mission 10: frame (%d,%d), world (%.3f,%.3f), corner cell (%d,%d), old mean (%d,%d,%v)",
		filepath.Base(f.Archives.Root), w.x, w.y, w.worldX, w.worldY, w.col, w.row, w.oldCol, w.oldRow, w.oldInside)

	if got, ok := live.view.SelectedUnit(); !ok || got != uint32(id) {
		t.Errorf("selection after ground order = (%d,%v), want (%d,true)", got, ok, id)
	}
}
