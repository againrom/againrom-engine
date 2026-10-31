package secondshop

import (
	"testing"

	"againrom/pkg/random"
)

// effectRow is a group B row of an effect of cost, min and max, usable by
// both classes, with the cumulative weight weights[c] in column c and
// otherwise fill.
func effectRow(cost, lo, hi, fill int32, weights map[int]int32) []int32 {
	p := []int32{cost, lo, hi, 3}
	for c := 0; c < 24; c++ {
		w, ok := weights[c]
		if !ok {
			w = fill
		}
		p = append(p, w)
	}
	return p
}

// enchantTables add to sortTables a fighter's weapon of part 1, a group B
// table whose effect 5 weighs 100 in every column and whose castSpell row
// adds 30 in the fighter's weapon column and 100 in the mage's, as the
// shipped table does, and the fighter's cast spells 18 and 26.
func enchantTables() Tables {
	tb := sortTables()
	word := []byte{1 << 1, 0}
	tb.Weapons = fixtureRows{n: 2, params: map[int][]int32{1: equipmentRow(300, 1, 1)}, raw: map[int][]byte{1: word}}
	tb.Magic = fixtureRows{n: 50, params: map[int][]int32{
		5:            effectRow(1, 1, 10, 100, nil),
		tagCastSpell: effectRow(1, 1, 1, 100, map[int]int32{0: 130, 12: 200}),
	}}
	spells := map[int][]int32{}
	for _, id := range []int{18, 26} {
		p := make([]int32, 22)
		p[colEnchant] = 10
		spells[id] = p
	}
	tb.Spells = fixtureRows{n: 30, params: spells}
	return tb
}

// An enchanted armour or shield category gives each item one to three
// effects by its part's weight column, and none of them casts a spell; the
// fighter's weapon column of the same table does cast (R2-ENGINE-711).
func TestSecondShopArmourAndShieldsCastNoSpell(t *testing.T) {
	tb := enchantTables()
	worn := Category{Min: 0, Max: 100000, Draws: 50, Mask: kindArmor | kindShield | bitEnchanted | 1<<classShift | 1<<1}
	held := Category{Min: 0, Max: 100000, Draws: 50, Mask: kindWeapon | bitEnchanted | 1<<classShift | 1<<1}
	casts := 0
	for seed := uint32(1); seed <= 20; seed++ {
		g := &random.MSVC{State: seed}
		s := tb.Fill(Record{worn, held}, func() int { return int(g.Rand()) })
		if len(s[0]) == 0 {
			t.Fatalf("seed %d: no enchanted armour or shield", seed)
		}
		for _, it := range s[0] {
			if len(it.Effects) < 1 || len(it.Effects) > 3 {
				t.Fatalf("seed %d: %+v holds %d effects", seed, it, len(it.Effects))
			}
			for _, e := range it.Effects {
				if e.Kind == tagCastSpell {
					t.Fatalf("seed %d: %+v casts a spell", seed, it)
				}
			}
		}
		for _, it := range s[1] {
			for _, e := range it.Effects {
				if e.Kind == tagCastSpell {
					casts++
				}
			}
		}
	}
	if casts == 0 {
		t.Fatal("the fighter's weapon column never cast: the fixture proves nothing")
	}
}
