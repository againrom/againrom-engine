package game

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
)

// TestReleaseFogFootprintIsTheClassTileSize reads the installed Units and
// Humans tables beside units.reg. The fog stamp's write centre adds half the
// body's token size; the original adds half the units.reg TileSize of the
// body's class (TERR-FOG-273). Every Units row's token size equals the TileSize
// of the class its TypeID names, every Humans row's is 1 and so is the TileSize
// of each class a Humans TypeID names, and every body of mission 81, which
// places a TileSize 2 and a TileSize 3 class, carries its class's TileSize.
func TestReleaseFogFootprintIsTheClassTileSize(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	classes, err := loadUnitClasses(f.Archives.Containers)
	if err != nil {
		t.Fatalf("units.reg: %v", err)
	}
	rows := 0
	for i := 1; i < f.Table.Units.Len(); i++ {
		if len(f.Table.Units.EntryParams(i)) == 0 {
			continue
		}
		def, err := data.NewUnitDef(f.Table.Units.EntryName(i), f.Table.Units.EntryParams(i))
		if err != nil {
			t.Fatalf("Units row %d: %v", i, err)
		}
		c, ok := classes.ByID(def.TypeID)
		if !ok {
			if def.TypeID != 0 {
				t.Errorf("Units row %d names TypeID %d, which units.reg lacks", i, def.TypeID)
			}
			continue
		}
		rows++
		if c.TileSize != def.TokenSize {
			t.Errorf("Units row %d %q: token size %d, class %d TileSize %d", i, f.Table.Units.EntryName(i), def.TokenSize, def.TypeID, c.TileSize)
		}
	}
	humans := 0
	for i := 1; i < f.Table.Humans.Len(); i++ {
		def, err := data.NewHumanDef(f.Table.Humans.EntryName(i), f.Table.Humans.EntryParams(i))
		if err != nil {
			continue
		}
		humans++
		if def.TokenSize != 1 {
			t.Errorf("Humans row %d %q: token size %d", i, f.Table.Humans.EntryName(i), def.TokenSize)
		}
		if c, ok := classes.ByID(def.TypeID); ok && c.TileSize != 1 {
			t.Errorf("Humans row %d %q: class %d TileSize %d", i, f.Table.Humans.EntryName(i), def.TypeID, c.TileSize)
		}
	}
	t.Logf("%d Units rows with a class, %d Humans rows", rows, humans)
	if rows == 0 || humans == 0 {
		t.Fatal("no Units or Humans rows read")
	}

	f.SetDeterministicFrames(true)
	if err := f.App("fog footprint").OpenMission(f.MissionOpener(81)); err != nil {
		t.Fatalf("open mission 81: %v", err)
	}
	wide := map[int32]int{}
	for _, e := range f.live.world.Entities() {
		c, ok := classes.ByID(e.Class)
		if !ok || e.Humanoid {
			continue
		}
		if int32(e.TokenSize) != c.TileSize {
			t.Errorf("mission 81 body %d class %d: token size %d, TileSize %d", e.ID, e.Class, e.TokenSize, c.TileSize)
		}
		if c.TileSize > 1 {
			wide[c.TileSize]++
		}
	}
	if wide[2] == 0 || wide[3] == 0 {
		t.Fatalf("mission 81 bodies by TileSize above 1: %v; want both 2 and 3", wide)
	}
}

// TestReleaseFogHeightPlaneIsTheParserGrid frames mission 150's type-2 grid
// as the original's light parser does (TERR-FOG-271): each record's payload
// starts at its head plus the head's own length word. The fresh World's height
// plane, which the fog walk reads, equals that grid byte for byte.
func TestReleaseFogHeightPlaneIsTheParserGrid(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	addr, _ := MissionMap(150)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	word := func(at int) int {
		if at+4 > len(raw) {
			t.Fatalf("%s: word at %d past %d bytes", addr, at, len(raw))
		}
		return int(binary.LittleEndian.Uint32(raw[at:]))
	}
	cursor, count := word(4), word(12)
	var grid []byte
	start := 0
	for i := 0; i < count; i++ {
		head, payload, kind := word(cursor+4), word(cursor+8), word(cursor+12)
		if kind == 2 {
			start = cursor + head
			grid = raw[start : start+payload]
		}
		cursor += head + payload
	}
	if grid == nil {
		t.Fatalf("%s carries no type-2 record", addr)
	}
	f.SetDeterministicFrames(true)
	if err := f.App("fog height plane").OpenMission(f.MissionOpener(150)); err != nil {
		t.Fatalf("open mission 150: %v", err)
	}
	height := f.live.world.CurrentPolicy().Terrain.Height
	t.Logf("%s: type-2 grid at bytes [%d, %d), World height plane %d cells", addr, start, start+len(grid), len(height))
	if len(height) != len(grid) {
		t.Fatalf("World height plane %d cells, type-2 grid %d bytes", len(height), len(grid))
	}
	for i := range grid {
		if height[i] != grid[i] {
			t.Fatalf("cell %d: World height %d, type-2 grid %d", i, height[i], grid[i])
		}
	}
	// Control: the slice 8 bytes earlier (TERR-FOG-267's) is another plane.
	if bytes.Equal(height, raw[start-8:start-8+len(grid)]) {
		t.Fatal("the height plane also equals the slice 8 bytes earlier; the frame is not discriminated")
	}
}
