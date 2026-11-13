package game

import (
	"encoding/binary"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// secondBuildingCellBlocking is the fixture building's Blocking word over
// its 3x2 footprint, row-major: cells 0, 1, 3 and 5 close, 2 and 4 open.
const secondBuildingCellBlocking = 0b101011

// secondBuildingCellMission is the mage mission with one building of kind 1
// whose footprint starts at 30/30.
func secondBuildingCellMission() secondMissionContent {
	c := secondMageMission()
	c.objects = append(c.objects, alm.Object{X: 30<<8 | 0x80, Y: 30<<8 | 0x80, Kind: 1, Field0C: 1, Field12: 7})
	return c
}

func secondBuildingCellTable(f *FrontEnd) {
	f.Table.Buildings = dbCollection{{}, {name: "Hut", params: []int32{3, 2, 0, 100, secondBuildingCellBlocking, -1}}}
}

// secondKeyList is raw's key list by cell, and each terrain entry's
// building key and occupants by cell.
func secondKeyList(t *testing.T, raw []byte) (map[uint16]sav.BlockRecord, map[uint16][3]uint32) {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[uint16]sav.BlockRecord{}
	for _, b := range sf.World.Blocks {
		keys[b.Cell] = b
	}
	entries := map[uint16][3]uint32{}
	for i := range sf.World.CellRecCount {
		e, err := sf.CellRecord(i)
		if err != nil {
			t.Fatal(err)
		}
		p := e[2:]
		entries[binary.LittleEndian.Uint16(e)] = [3]uint32{binary.LittleEndian.Uint32(p[12:]), binary.LittleEndian.Uint32(p[4:]), binary.LittleEndian.Uint32(p[8:])}
	}
	return keys, entries
}

// secondBuildingCellLoad LOADs raw into a fresh front of the building
// fixture.
func secondBuildingCellLoad(t *testing.T, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondMissionFrontOf(t, secondBuildingCellMission())
	secondMageItems(f)
	secondBuildingCellTable(f)
	app := f.App("second-mission")
	app.Layout(1024, 768)
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game0001.sav")
	if f.live == nil {
		t.Fatal("the file does not LOAD")
	}
	return f, app
}

// A second-game mission SAVE gives a building cell's two key-list bytes
// bits 0x04 and 0x01 where the building's Blocking word closes the cell and
// neither where it opens it, as the original's refresh of an entry cell
// does (R2-ENGINE-758): 0x25 and 0x20 on a building that stands alone. A
// unit's cell and a cell with no entry keep their bytes. The file LOADs to
// the entered World, and the SAVE of that LOAD holds the same key list and
// entries. Fails while the bytes come from the native grid alone.
func TestSecondMissionSaveBuildingCellsCarryTheBlockingBits(t *testing.T) {
	f, app, screen, _ := secondBagFrontOf(t, nil, secondBuildingCellMission())
	secondMageItems(f)
	secondBuildingCellTable(f)
	secondBagEnter(t, f, app, screen)
	raw := secondPageSave(t, f, app)
	checkSecondMissionFile(t, raw, 10)

	keys, entries := secondKeyList(t, raw)
	var building uint32
	for at := 0; at < 6; at++ {
		cell := uint16(30+at/3)<<8 | uint16(30+at%3)
		e, ok := entries[cell]
		if !ok || e[0] == 0 || (building != 0 && e[0] != building) {
			t.Fatalf("entry %#04x names building %#x, want the one building of the footprint (%#x)", cell, e[0], building)
		}
		building = e[0]
		want := byte(0x20)
		if secondBuildingCellBlocking>>at&1 != 0 {
			want = 0x25
		}
		if k := keys[cell]; k.Static != want || k.Dyn != want {
			t.Errorf("building cell %#04x (footprint bit %d): key-list bytes %#02x and %#02x, want %#02x twice", cell, at, k.Dyn, k.Static, want)
		}
	}
	units := 0
	for cell, e := range entries {
		if e[0] != 0 {
			continue
		}
		k := keys[cell]
		if e[1] != 0 && e[2] == 0 {
			units++
			if k.Dyn != 0x60 || k.Static != 0x20 {
				t.Errorf("unit cell %#04x: key-list bytes %#02x and %#02x, want 0x60 and 0x20", cell, k.Dyn, k.Static)
			}
		}
	}
	if units < 7 {
		t.Fatalf("the file holds %d ground unit cells, want the hero, two knights and four creatures", units)
	}
	for cell, k := range keys {
		if _, ok := entries[cell]; !ok {
			t.Errorf("cell %#04x has no entry and key-list bytes %#02x and %#02x", cell, k.Dyn, k.Static)
		}
	}
	if _, listed := keys[uint16(29)<<8|30]; listed {
		t.Error("the open cell above the building is in the key list")
	}

	loaded, loadedApp := secondBuildingCellLoad(t, raw)
	if got, want := loaded.live.world.Hash(), f.live.world.Hash(); got != want {
		t.Fatalf("the LOAD's World hash is %x, the entered World's %x", got, want)
	}
	if got, want := loaded.live.world.CurrentPolicy().Terrain.Block, f.live.world.CurrentPolicy().Terrain.Block; !slices.Equal(got, want) {
		t.Fatal("the LOAD's native grid differs from the entered World's")
	}
	again := secondPageSave(t, loaded, loadedApp)
	keysAgain, entriesAgain := secondKeyList(t, again)
	if !maps.Equal(keysAgain, keys) || !maps.Equal(entriesAgain, entries) {
		t.Fatalf("the SAVE of the LOADed file holds key list %v and entries %v, want %v and %v", keysAgain, entriesAgain, keys, entries)
	}
}
