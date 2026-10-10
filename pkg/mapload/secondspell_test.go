package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// TestSecondGameRulesClassifyTheDamagePairByArm: the second game's Heal and
// Drain Life are rows 24 and 26, and its rows 6 and 11 are Poison Cloud and
// Prismatic Spray.
func TestSecondGameRulesClassifyTheDamagePairByArm(t *testing.T) {
	rows := make([]data.Spell, 26)
	for _, id := range []int{5, 6, 11, 24, 26} {
		rows[id-1].DamageMin, rows[id-1].DamageMax = 4, 8
		rows[id-1].Damaging, rows[id-1].Restorative = data.ClassifySpell(id, 4, 8)
	}
	first := spellRules(rows)
	if !first[23].Damaging || first[5].Damaging || !first[5].Restorative || first[10].Damaging {
		t.Fatal("the first-game classification changed")
	}
	second := spellRules(rows)
	SecondGameSpellArms(second)
	want := map[int][2]bool{5: {true, false}, 6: {true, false}, 11: {true, false}, 24: {false, true}, 26: {false, false}}
	for id, w := range want {
		r := second[id-1]
		if r.Damaging != w[0] || r.Restorative != w[1] || !r.Second {
			t.Errorf("second-game row %d: damaging=%v restorative=%v second=%v, want %v", id, r.Damaging, r.Restorative, r.Second, w)
		}
	}
	if second[4].Arm != sim.ArmNone || second[23].ArmID() != 6 {
		t.Errorf("arms: Ice Missile %d, Heal %d", second[4].Arm, second[23].ArmID())
	}
}
