package game

import (
	"encoding/binary"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// This is an installed-content implementation witness, not an original-runtime
// observation. The unmodified M10 recording enters both original LOAD doors;
// no script substitution, relocation, command injection or callback replay is
// used. Unknown original actor callbacks remain outside this witness.
func TestReleaseOriginalCellPlanes1115NativeContinuation(t *testing.T) {
	f := releaseFront(t)
	path, raw := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if source.Head.Mission != 10 || source.World == nil || binary.LittleEndian.Uint32(source.Body[:4]) != 372 || binary.LittleEndian.Uint32(source.Body[4:8]) != 23 {
		t.Fatal("natural M10 source anchors changed")
	}
	want, sourceCells, width, height := originalCellPlaneReference1115(t, f, source)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, f.Bodies)
	if err != nil {
		t.Fatal(err)
	}
	if ms.Number != 10 || ms.Map.Width != width || ms.Map.Height != height {
		t.Fatal("diagnostic LOAD did not use the installed M10 dimensions")
	}
	diagnosticPlanes := originalCellPlanesCheck1115(t, "ResumeOriginalSave", ms.World, want)
	diagnosticCells := originalCellNodesCheck1115(t, "ResumeOriginalSave", ms.World, sourceCells)

	f.SetDeterministicFrames(true)
	app := f.App("1115 natural raw cell planes")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, filepath.Base(path))
	if f.live.world.Bounds() != (sim.Bounds{Width: int32(width), Height: int32(height)}) || f.live.world.Tick() != 372 {
		t.Fatal("App original LOAD changed map dimensions or advanced the source")
	}
	warmPlanes := originalCellPlanesCheck1115(t, "App original LOAD", f.live.world, diagnosticPlanes)
	warmCells := originalCellNodesCheck1115(t, "App original LOAD", f.live.world, sourceCells)
	if !reflect.DeepEqual(warmCells, diagnosticCells) {
		t.Fatal("the original LOAD doors disagree on complete cells or exact actor bindings")
	}
	if app.HeadlessNoticeOpen() {
		t.Fatal("natural source unexpectedly starts in a notice")
	}

	warmHash := f.live.world.Hash()
	cold, coldApp := holdingsNativeFresh(t, f, app, store, nil)
	originalCellPlanesCheck1115(t, "fresh native LOAD", cold.live.world, warmPlanes)
	_, coldCells, _, present := cold.live.world.SavedActorMotions()
	if !present || !reflect.DeepEqual(coldCells, warmCells) || f.live.world.Hash() != warmHash || cold.live.world.Hash() != warmHash {
		t.Fatal("ordinary menu SAVE/fresh native LOAD lost cells or changed the world hash")
	}
	// The menu SAVE leaves the warm App in its menu. Restore the same screen
	// state, then exercise actual driver ticks rather than UI-frame counters.
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	lastPlanes, lastCells := warmPlanes, warmCells
	planeChanges, cellChanges := 0, 0
	for tick := 1; tick <= 20; tick++ {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Tick() != uint64(372+tick) || cold.live.world.Tick() != uint64(372+tick) || f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("native continuation differs at actual simulation tick %d", 372+tick)
		}
		current, ok := f.live.world.SavedCellPlanes()
		if !ok || current == nil {
			t.Fatal("actual continuation discarded raw plane authority", tick)
		}
		originalCellPlanesCheck1115(t, "native continuation", cold.live.world, current)
		_, currentCells, _, currentOK := f.live.world.SavedActorMotions()
		_, resumedCells, _, resumedOK := cold.live.world.SavedActorMotions()
		if !currentOK || !resumedOK || !reflect.DeepEqual(currentCells, resumedCells) {
			t.Fatal("native continuation changed complete cell semantics", tick)
		}
		if app.HeadlessNoticeOpen() != coldApp.HeadlessNoticeOpen() {
			t.Fatal("native continuation changed natural notice timing", tick)
		}
		if *current != *lastPlanes {
			planeChanges++
		}
		if !reflect.DeepEqual(currentCells, lastCells) {
			cellChanges++
		}
		lastPlanes, lastCells = current, currentCells
	}
	if planeChanges == 0 || cellChanges == 0 {
		t.Fatal("natural20-tick continuation no longer exercises changing raw planes and cells", planeChanges, cellChanges)
	}
	t.Logf("natural M10 sha7acf1d56: %dx%d, source Blocks%d Cells%d/current%d; installed Costs%v; ordinary menu SAVE/fresh LOAD then20 actual ticks; observed plane-change steps%d cell-change steps%d; final hash%016x; original callbacks not claimed",
		width, height, len(source.World.Blocks), source.World.CellRecCount, len(warmCells), want.Costs, planeChanges, cellChanges, cold.live.world.Hash())
}

// Read the installed registry and ALM through format readers only. Expected
// planes never call the terrain producer, native passability/grid or document
// projector. ALM-TERR-043 supplies key order; TERR-PASS-049/050 supplies the
// constructor fill and the following independently expressed weighted table.
func originalCellPlaneReference1115(t *testing.T, f *FrontEnd, source *sav.File) (*sim.SavedCellPlanes, map[uint16][52]byte, int, int) {
	t.Helper()
	rawREG, err := f.Archives.Containers.ReadFile("world/data/map.reg")
	if err != nil {
		t.Fatal("installed map.reg", err)
	}
	r, err := reg.Parse(rawREG)
	if err != nil {
		t.Fatal(err)
	}
	want := &sim.SavedCellPlanes{}
	want.Costs[0] = 255
	for i, key := range []string{"CostLand", "CostGrass", "CostFlowers", "CostSand", "CostCracked", "CostStones", "CostSavanna", "CostMountain", "CostWater", "CostRoad"} {
		value, found := r.GetInt("Terrain", key)
		if !found {
			t.Fatalf("installed map.reg lacks %s; this witness needs the actual installed parameters", key)
		}
		want.Costs[i+1] = byte(value)
	}
	// These shipped values differ from the executable's missing-key defaults
	// at Flowers and Savanna, so quietly manufacturing defaults is observable.
	if want.Costs != ([11]byte{255, 8, 8, 8, 14, 6, 12, 8, 16, 8, 6}) {
		t.Fatal("promoted installed Terrain parameter anchor changed", want.Costs)
	}
	rawALM, err := f.Archives.Containers.ReadFile("scenario/10.alm")
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(rawALM)
	if err != nil {
		t.Fatal(err)
	}
	if m.Width != 80 || m.Height != 80 {
		t.Fatalf("installed natural M10 dimensions=%dx%d, want80x80", m.Width, m.Height)
	}
	// The numeric class pairs and selector rows are promoted TERR-PASS-050
	// literals. Weight0..4 on the primary is an independent interpolation of
	// the five arms; there is no call to the implementation's classifier.
	pairs := [13][2]int{0: {2, 1}, 1: {5, 1}, 2: {4, 1}, 3: {7, 1}, 4: {6, 1}, 5: {5, 6}, 6: {3, 7}, 7: {8, 6}, 12: {10, 1}}
	levels := [4]string{"23243422224444", "35331324224244", "23243424224244", "55555522224444"}
	for cell := range want.Cost {
		want.Cost[cell], want.CostKnown[cell] = 1, 1
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			at, cell := y*m.Width+x, y*256+x
			word := m.Tiles[at]
			index := int(word & 1023)
			cost, mountain := byte(255), false
			if index >= 512 && index < 768 {
				if index%16 < 8 {
					cost = 8 // Entry literal, independent of CostWater.
				}
			} else if index%16 < 14 {
				group := index / 64
				if group >= 13 || pairs[group][0] == 0 {
					t.Fatalf("installed ALM reaches uninitialized class pair at%04x", cell)
				}
				weight := int(levels[index/16%4][index%16] - '1')
				primary, secondary := pairs[group][0], pairs[group][1]
				cost = byte((weight*int(want.Costs[primary]) + (4-weight)*int(want.Costs[secondary])) / 4)
				mountain = group == 7 && weight >= 2
			}
			want.Cost[cell], want.Height[cell] = cost, m.Altitudes[at]
			if word&0x2000 != 0 || mountain || index >= 512 && index < 768 {
				want.Static[cell] = 1
			}
			if m.Overlay[at] != 0 {
				want.Static[cell] = 5
			}
			// This selected ALM is square: border stores are confined to the
			// 80-square, not a fabricated ring around the full256-array.
			if x < 8 || y < 8 || x >= 72 || y >= 72 {
				want.Static[cell] = 31
			}
		}
	}
	want.Dynamic = want.Static
	// SAV-LOAD-057/TERR-PASS-053: four-byte literal records assign Static and
	// Dynamic only. The decoder's typed Blocks and a SAVE projector are not
	// used to manufacture these expected plane bytes.
	for row := 0; row < len(source.World.Blocks); row++ {
		off := source.World.BlocksDataOff + 4*row
		block := source.Body[off : off+4]
		cell := binary.LittleEndian.Uint16(block[2:])
		want.Static[cell], want.Dynamic[cell] = block[0], block[1]
	}
	// SAV-CELLLOAD-109/110: final per-key payload wins. A record's Cost byte
	// remains its creation baseline; it does NOT overwrite the load Cost plane.
	cells := make(map[uint16][52]byte)
	for row := 0; row < source.World.CellRecCount; row++ {
		off := source.World.CellRecDataOff + 54*row
		var payload [52]byte
		copy(payload[:], source.Body[off+2:off+54])
		cells[binary.LittleEndian.Uint16(source.Body[off:])] = payload
	}
	if len(cells) == 0 || len(source.World.Blocks) == 0 {
		t.Fatal("natural source no longer witnesses saved cells and block assignments")
	}
	return want, cells, m.Width, m.Height
}

func originalCellPlanesCheck1115(t *testing.T, door string, w *sim.World, want *sim.SavedCellPlanes) *sim.SavedCellPlanes {
	t.Helper()
	got, present := w.SavedCellPlanes()
	if !present || got == nil {
		t.Fatalf("%s did not establish raw original cell-plane authority", door)
	}
	if got.Costs != want.Costs {
		t.Fatalf("%s retained Costs%v want installed%v", door, got.Costs, want.Costs)
	}
	for _, plane := range []struct {
		name      string
		got, want *[65536]byte
	}{
		{"Cost", &got.Cost, &want.Cost}, {"Static", &got.Static, &want.Static},
		{"Dynamic", &got.Dynamic, &want.Dynamic}, {"Height", &got.Height, &want.Height},
		{"CostKnown", &got.CostKnown, &want.CostKnown},
	} {
		for cell, expected := range plane.want {
			if plane.got[cell] != expected {
				t.Fatalf("%s %s[%04x]=%02x want%02x", door, plane.name, cell, plane.got[cell], expected)
			}
		}
	}
	for cell, known := range got.CostKnown {
		if known != 1 {
			t.Fatalf("%s dropped cost authority at%04x", door, cell)
		}
	}
	return got
}

func originalCellNodesCheck1115(t *testing.T, door string, w *sim.World, want map[uint16][52]byte) []sim.SavedActorCell {
	t.Helper()
	_, cells, _, present := w.SavedActorMotions()
	if !present || len(cells) != len(want) {
		t.Fatalf("%s current cells%d want%d, present%t", door, len(cells), len(want), present)
	}
	seen := make(map[uint16]bool, len(cells))
	for _, cell := range cells {
		payload, exists := want[cell.Cell]
		if !exists || seen[cell.Cell] || cell.Payload != payload || cell.Ground.Key != binary.LittleEndian.Uint32(payload[4:]) || cell.Air.Key != binary.LittleEndian.Uint32(payload[8:]) {
			t.Fatalf("%s cell%04x lost its final52-byte payload or exact occupant keys", door, cell.Cell)
		}
		seen[cell.Cell] = true
	}
	return cells
}
