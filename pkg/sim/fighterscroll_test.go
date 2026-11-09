package sim

import "testing"

// A Shield scroll aimed at its reader lands for a unit without mana as for a
// mage, whether the row is a unit-target or a self-only row.
func TestAFighterReadsAShieldScrollOnHimself(t *testing.T) {
	shield := acbShield()
	shield.TargetsUnit = false
	for _, selfOnly := range []bool{false, true} {
		for _, mana := range []int32{0, 100} {
			sh := shield
			sh.TargetsUnit = !selfOnly
			e := spEnt(1, 4, 8)
			e.Owner = SelfSlot
			e.HP, e.MaxHP = 100, 100
			e.MaxMana, e.Mana = mana, mana
			w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, nil, Relations{}, nil,
				[]Stock{{ID: 1, ItemInstances: []ItemInstance{{Code: 0x0e06, Kind: 4, Price: 100, Effects: []ItemEffect{{Kind: 41, Operand: 18 | 50<<16}}}}}},
				[]SpellRule{sh})
			if err != nil {
				t.Fatal(err)
			}
			Step(w, []Command{UseScroll(1, 0, 1)})
			for range 100 {
				Step(w, nil)
			}
			_, on := effectIndex(w.attached, 1, 18)
			if !on || len(w.ScrollCasts()) != 0 {
				t.Errorf("selfOnly %v mana %d: shielded %v casts %d", selfOnly, mana, on, len(w.ScrollCasts()))
			}
		}
	}
}
