package sim

import "testing"

func TestStagedAreaSavedBeforeItsFirstStageHoldsNoCells(t *testing.T) {
	type fixture struct {
		name string
		make func(t *testing.T, row SpellRule) *World
	}
	first := func(t *testing.T, row SpellRule) *World {
		w, err := NewSpelledWorld(7, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
			[]Entity{blizzardCaster(20, 12)}, nil, []SpellRule{row})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	second := func(t *testing.T, row SpellRule) *World {
		return secondWorld(t, 7, Bounds{Width: 40, Height: 40}, []SpellRule{secondRow(row)}, blizzardCaster(20, 12))
	}
	for _, f := range []fixture{{"first", first}, {"second", second}} {
		ids := []uint16{4, 9}
		if f.name == "second" {
			ids = []uint16{9}
		}
		for _, id := range ids {
			row := acidStreamStartRow()
			row.ID = id
			w := f.make(t, row)
			if !w.landArea(w.spells[0], 50, 1, true, 20, 12, 20, 20, nil) {
				t.Fatalf("%s spell %d: no area", f.name, id)
			}
			cold := currentAreaOrdinaryWorld(t, w, nil)
			if len(cold.effects) != 1 || len(cold.effects[0].Cells) != 0 {
				t.Fatalf("%s spell %d: loaded record holds cells: %+v", f.name, id, cold.effects)
			}
			if cold.Hash() != w.Hash() {
				t.Fatalf("%s spell %d: loaded world hashes differently", f.name, id)
			}
		}
	}
}
