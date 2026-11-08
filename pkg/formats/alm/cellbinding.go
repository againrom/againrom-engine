package alm

import "fmt"

// CellBinding is the type-9 cell arm's six-byte payload, not an item recipe
// or a building caster. Coordinates are narrowed by the builder (UNIT-M10CELL-054).
type CellBinding struct {
	X, Y             uint8
	Spell, Power     uint8
	SourceX, SourceY uint8
	LastX, LastY     uint8
}

// CellBindings selects the cell arm and preserves file order. The other type-9
// arms remain available through Enchantments. Missing tail words are refused;
// they must not become an invented source at (0,0).
func (m *Map) CellBindings() ([]CellBinding, error) {
	if m == nil {
		return nil, nil
	}
	if m.Present(TypeTileMarkers) && uint64(len(m.Enchantments)) != uint64(m.TileMarkers.Count) {
		return nil, fmt.Errorf("alm: type-9 records do not tile the cell-binding payload")
	}
	var out []CellBinding
	for i, r := range m.Enchantments {
		if r.X == 0 && r.Y == 0 || r.A >= 4 {
			continue
		}
		if len(r.Elements) < 2 {
			return nil, fmt.Errorf("alm: cell binding %d needs two tail records, has %d", i, len(r.Elements))
		}
		out = append(out, CellBinding{
			X: uint8(r.X), Y: uint8(r.Y), Spell: uint8(r.SpellRaw), Power: uint8(r.SpellRaw >> 16),
			SourceX: uint8(r.Elements[0].Kind), SourceY: uint8(r.Elements[0].Low),
			LastX: uint8(r.Elements[1].Kind), LastY: uint8(r.Elements[1].Low),
		})
	}
	return out, nil
}

// BuildingCasterRecord is the mode-4 arm of a type-9 record: a caster bound to
// the building its Key names (R2-ENGINE-1006, R2-ENGINE-1009).
type BuildingCasterRecord struct {
	Key          uint32
	X, Y         uint8
	Spell, Power uint8
	Reach        uint8
}

// BuildingCasters selects the records of mode mask 4 with a nonzero cell and no
// sack binding, in file order. A loot element's link to a record is the sack
// binding (R2-ASSET-230); whether the key names a building is the loader's
// test, which holds the buildings.
func (m *Map) BuildingCasters() []BuildingCasterRecord {
	if m == nil {
		return nil
	}
	bound := map[uint32]bool{}
	if loot, err := m.Loot(); err == nil {
		for _, r := range loot.Records {
			for _, e := range r.Elements {
				if e.TileMarkerIndex != 0 {
					bound[e.TileMarkerIndex-1] = true
				}
			}
		}
	}
	var out []BuildingCasterRecord
	for i, r := range m.Enchantments {
		if r.A&4 == 0 || r.X == 0 && r.Y == 0 || bound[uint32(i)] {
			continue
		}
		out = append(out, BuildingCasterRecord{Key: uint32(r.B) | uint32(r.C)<<16, X: uint8(r.X), Y: uint8(r.Y),
			Spell: uint8(r.SpellRaw), Power: uint8(r.SpellRaw >> 16), Reach: uint8(r.Tag)})
	}
	return out
}
