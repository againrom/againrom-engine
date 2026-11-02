package mapload

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
)

// ModTerrain is the terrain groups t's mods add; nil for none or a nil t.
func (t *Table) ModTerrain() *mod.TerrainData {
	if t == nil || t.Mods.Terrain.Empty() {
		return nil
	}
	return &t.Mods.Terrain
}

// ModTerrainGroup is the row of tile group g t's mods add; false for none.
func (t *Table) ModTerrainGroup(g int) (mod.TerrainGroup, bool) {
	if groups := t.ModTerrain(); groups != nil {
		return groups.Group(g)
	}
	return mod.TerrainGroup{}, false
}

// CheckTileGroups refuses a map that names a tile group with no picture: a
// +0x18 mask bit or a cell word of a group past the install's that no row of
// t's mods adds, a cell word past the loader's 32 groups, or a mod group's
// cell naming a frame its column's strip does not hold. The original stops
// with a fatal error on a set mask bit whose strip is absent (TERR-TILE-292);
// this refuses the map with the group and the map named instead.
func CheckTileGroups(m *alm.Map, t *Table) error {
	if m == nil {
		return nil
	}
	name, row := "the map", t.ModTerrainGroup
	if m.Name != "" {
		name = fmt.Sprintf("map %q", m.Name)
	}
	for g := alm.InstallTileGroups; g < alm.TileGroupCount; g++ {
		if m.Meta.Bitmask>>g&1 == 1 {
			if _, ok := row(g); !ok {
				return fmt.Errorf("%s: its group mask names terrain group %d, which no active mod adds a picture for", name, g)
			}
		}
	}
	for i, w := range m.Tiles {
		g := alm.TileGroup(w)
		if g < alm.InstallTileGroups {
			continue
		}
		x, y := i, 0
		if m.Width > 0 {
			x, y = i%m.Width, i/m.Width
		}
		r, ok := row(g)
		if !ok {
			return fmt.Errorf("%s: cell (%d,%d) names terrain group %d, which no active mod adds a picture for", name, x, y, g)
		}
		col, frame := int(w>>4&3), int(w&15)
		if frame >= r.Frames[col] {
			return fmt.Errorf("%s: cell (%d,%d) names terrain group %d column %d frame %d; mod %q's strip for that column holds %d frames", name, x, y, g, col, frame, r.Mod, r.Frames[col])
		}
	}
	return nil
}
