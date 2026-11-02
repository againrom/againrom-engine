package mapload_test

import (
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// modTerrainTable is a table whose mods add group 13 as grass, 26 as grass
// (its low ten bits fall in the raw water range) and 31 as mountain, each
// with two-frame strips.
func modTerrainTable() *mapload.Table {
	costs := originalTerrainCosts()
	row := func(g int, class uint8) mod.TerrainGroup {
		return mod.TerrainGroup{Mod: "snow-field", Group: g, Class: class, Name: "g", Frames: [4]int{2, 2, 2, 2}}
	}
	t := &mapload.Table{TerrainCosts: &costs}
	t.Mods.Terrain = mod.TerrainData{Groups: []mod.TerrainGroup{row(13, 2), row(26, 2), row(31, 8)}}
	return t
}

// A cell of a mod group takes its row's land kind in the cost plane, the
// block plane and the original constructor planes, whatever its low ten bits
// would classify as (DIV-4413).
func TestModGroupCellsTakeTheirRowsLandKind(t *testing.T) {
	m := passMap(passW, passH)
	m.Altitudes = make([]uint8, passW*passH)
	snow, water26, ice := groupWord(13, 1, 1), groupWord(26, 0, 0), groupWord(31, 0, 1)
	at := func(x, y int) int { return y*passW + x }
	m.Tiles[at(8, 8)], m.Tiles[at(9, 8)], m.Tiles[at(10, 8)] = snow, water26, ice
	table := modTerrainTable()
	planes := mapload.Planes(m, table)
	if planes.BaseFault != nil {
		t.Fatalf("constructor planes refused a mod map: %v", planes.BaseFault)
	}
	costs := *table.TerrainCosts
	for _, c := range []struct {
		name  string
		x     int
		block byte
		cost  byte
	}{{"snow (grass)", 8, 0, costs[2]}, {"group 26 (grass, raw water range)", 9, 0, costs[2]}, {"ice (mountain)", 10, 1, costs[8]}} {
		i := at(c.x, 8)
		if planes.Block[i]&1 != c.block || planes.Cost[i] != c.cost {
			t.Errorf("%s: block %d cost %d, want %d and %d", c.name, planes.Block[i]&1, planes.Cost[i], c.block, c.cost)
		}
		if got := planes.Base.Static[8*256+c.x]; got != c.block {
			t.Errorf("%s: constructor static %d, want %d", c.name, got, c.block)
		}
	}
	// Without the rows the same words read as the original's low ten bits.
	if got := mapload.Passability(m)[at(9, 8)]; got&1 != 1 {
		t.Errorf("group 26 without a row: block %d, want the raw water arm's 1", got)
	}
}

// The map load refuses a map naming a tile group no mod draws, naming the
// group and the map; with the rows it builds.
func TestMapNamingAGroupWithNoPictureIsRefused(t *testing.T) {
	m := passMap(passW, passH)
	m.Altitudes = make([]uint8, passW*passH)
	m.Name = "Snow Pass"
	m.Tiles[9*passW+9] = groupWord(13, 2, 1)
	m.Meta.Bitmask = 1 | 1<<13
	if _, err := mapload.FromALMWith(m, modTerrainTable(), mapload.DifficultyNormal); err != nil {
		t.Fatalf("with the rows: %v", err)
	}
	plain := &mapload.Table{}
	_, err := mapload.FromALMWith(m, plain, mapload.DifficultyNormal)
	if err == nil || !strings.Contains(err.Error(), "terrain group 13") || !strings.Contains(err.Error(), `"Snow Pass"`) {
		t.Fatalf("without the rows: %v, want a refusal naming group 13 and the map", err)
	}
	m.Tiles[9*passW+9] = groupWord(13, 2, 5)
	if _, err := mapload.FromALMWith(m, modTerrainTable(), mapload.DifficultyNormal); err == nil || !strings.Contains(err.Error(), "frame 5") {
		t.Fatalf("a frame past the strip: %v", err)
	}
	m.Tiles[9*passW+9] = 0
	m.Meta.Bitmask = 1 | 1<<20
	if _, err := mapload.FromALMWith(m, modTerrainTable(), mapload.DifficultyNormal); err == nil || !strings.Contains(err.Error(), "group mask names terrain group 20") {
		t.Fatalf("a mask bit with no row: %v", err)
	}
	m.Meta.Bitmask = 1
	m.Tiles[9*passW+9] = uint16(40 << 6)
	if err := mapload.CheckTileGroups(m, modTerrainTable()); err == nil || !strings.Contains(err.Error(), "terrain group 40") {
		t.Fatalf("a group past the loader: %v", err)
	}
	if err := mapload.CheckTileGroups(&alm.Map{Width: 1, Height: 1, Tiles: []uint16{0x11}}, nil); err != nil {
		t.Fatalf("an install word with no table: %v", err)
	}
}
