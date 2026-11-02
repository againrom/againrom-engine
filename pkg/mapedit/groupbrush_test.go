package mapedit_test

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// A mod group's brush paints whole cells of its words; the mask names the
// group; a later terrain stroke beside it leaves its cells alone; the saved
// bytes reopen with the same words and mask.
func TestModGroupBrushPaintsWholeCellsAndSaves(t *testing.T) {
	ed, err := mapedit.New(brushMap(0x14, 40))
	if err != nil {
		t.Fatal(err)
	}
	snow := []uint16{13<<6 | 0x00, 13<<6 | 0x01, 13<<6 | 0x10}
	ice := []uint16{31<<6 | 0x21}
	paint := func(b mapedit.Brush, x, y int) {
		t.Helper()
		if _, err := ed.Paint(b, []mapedit.Cell{{X: x, Y: y}}, testClass); err != nil {
			t.Fatal(err)
		}
	}
	paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: mapedit.GroupBrush(13), Words: snow, Radius: 1, Shape: mapedit.BrushSquare}, 12, 12)
	paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: mapedit.GroupBrush(31), Words: ice}, 20, 20)
	before, _, _ := planes(t, ed.Bytes())
	paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassSand)}, 14, 12)
	tiles, _, mask := planes(t, ed.Bytes())
	for y := 11; y <= 13; y++ {
		for x := 11; x <= 13; x++ {
			w := tiles[y*brushSide+x]
			if !slices.Contains(snow, w) || w != before[y*brushSide+x] {
				t.Errorf("cell (%d,%d) = %#04x, want its snow word %#04x kept", x, y, w, before[y*brushSide+x])
			}
		}
	}
	if tiles[20*brushSide+20] != ice[0] {
		t.Errorf("ice cell %#04x", tiles[20*brushSide+20])
	}
	if mask&(1<<13) == 0 || mask&(1<<31) == 0 || mask != alm.TileGroupMask(tiles) {
		t.Errorf("mask %#x, want bits 13 and 31 (%#x)", mask, alm.TileGroupMask(tiles))
	}
	saved, err := ed.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	again, err := mapedit.New(saved)
	if err != nil {
		t.Fatal(err)
	}
	m, err := again.Map()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.Tiles, tiles) || m.Meta.Bitmask != mask {
		t.Error("the reopened map's words or mask differ")
	}
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: mapedit.GroupBrush(13), Words: ice}, []mapedit.Cell{{X: 5, Y: 5}}, testClass); err == nil {
		t.Error("a word of group 31 under group 13's brush was painted")
	}
	if g, ok := mapedit.BrushGroup(mapedit.GroupBrush(31)); !ok || g != 31 {
		t.Error("BrushGroup does not invert GroupBrush")
	}
	if _, ok := mapedit.BrushGroup(int(mapedit.ClassRoad)); ok {
		t.Error("a terrain class reads as a mod group")
	}
}

// Random terrain leaves every cell of a mod group as it is, whatever its
// group's low bits read as in the install's table, and draws the install's
// cells word for word as on a map without the mod cells.
func TestRandomTerrainKeepsModGroupCells(t *testing.T) {
	last := func(n int) int { return n - 1 }
	groups := []int{13, 16, 20, 24, 28, 31}
	paintMap := func(withMod bool) *mapedit.Editor {
		t.Helper()
		ed, err := mapedit.New(brushMap(0x14, 40))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassSand)}, []mapedit.Cell{{X: 4, Y: 18}}, testClass); err != nil {
			t.Fatal(err)
		}
		if !withMod {
			return ed
		}
		for n, g := range groups {
			words := []uint16{uint16(g<<6 | 0x00), uint16(g<<6 | 0x01), uint16(g<<6 | 0x10), uint16(g<<6 | 0x21)}
			b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: mapedit.GroupBrush(g), Words: words}
			if _, err := ed.Paint(b, []mapedit.Cell{{X: 3 + 3*n, Y: 12}}, testClass); err != nil {
				t.Fatal(err)
			}
		}
		return ed
	}
	ed, twin := paintMap(true), paintMap(false)
	before, _, _ := planes(t, ed.Bytes())
	if _, err := ed.RandomizeTerrain(last); err != nil {
		t.Fatal(err)
	}
	if _, err := twin.RandomizeTerrain(last); err != nil {
		t.Fatal(err)
	}
	after, _, _ := planes(t, ed.Bytes())
	base, _, _ := planes(t, twin.Bytes())
	for n, g := range groups {
		i := 12*brushSide + 3 + 3*n
		if alm.TileGroup(before[i]) != g || after[i] != before[i] {
			t.Errorf("group %d cell: before %#04x after %#04x", g, before[i], after[i])
		}
	}
	changed := false
	for i, w := range after {
		if alm.TileGroup(before[i]) >= alm.InstallTileGroups {
			continue
		}
		if w != base[i] {
			t.Errorf("install cell %d: %#04x, the map without mod cells gives %#04x", i, w, base[i])
		}
		changed = changed || w != before[i]
	}
	if sand := 18*brushSide + 4; !changed || after[sand] != base[sand] {
		t.Errorf("the install's cells were not drawn as without the mod cells (sand %#04x, want %#04x)", after[sand], base[sand])
	}
}
