package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// A record's key names a type-4 object by its id word; the caster takes the
// building's low owner word and its cell word. A key naming no object builds
// none and the load goes on. A world built from the map has no health gate: a
// building whose health is zero there still builds its caster. The base does
// not compile this test.
func TestBuildingCastersKeyOwnerAndCellWord(t *testing.T) {
	m := &alm.Map{
		Objects: []alm.Object{{Field0E: 0x00070002, Field12: 42}, {Field0E: 3, Field12: 43}},
		Enchantments: []alm.Enchantment{
			{Tag: 4, X: 9, Y: 10, A: 4, B: 42, SpellRaw: 2 | 99<<16},
			{Tag: 5, X: 11, Y: 12, A: 4, B: 77, SpellRaw: 3 | 40<<16},
			{Tag: 6, X: 13, Y: 14, A: 4, B: 43, SpellRaw: 4 | 50<<16},
		},
	}
	w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 20, Height: 20}, sim.ModeCanonical, sim.Terrain{}, nil, nil, sim.Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.DeclareStructures([]sim.Structure{
		{ID: 0, Col: 8, Row: 9, Width: 1, Height: 1, MaxHealth: 100, Field42: 100},
		{ID: 1, Col: 3, Row: 4, Width: 1, Height: 1, MaxHealth: 100, Field42: 0},
	})
	got := BuildingCasters(m, w)
	want := []sim.BuildingCaster{
		{X: 9, Y: 10, Spell: 2, Power: 99, Reach: 4, Owner: 2, Cell: 9<<8 | 8},
		{X: 13, Y: 14, Spell: 4, Power: 50, Reach: 6, Owner: 3, Cell: 4<<8 | 3},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %+v, want %+v: no object 77, object 43's zero health is no gate", got, want)
	}
}
