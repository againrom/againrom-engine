package sim

import "testing"

// A mage's spell kill: the resolver writes the damage kind 2, then the point
// or area tail runs. The first game's tail writes the spell id 20; the second
// game's writes none, so the award reads kind 2 as a spell id and trains the
// School of spell 2 (R2-ENGINE-1826).
func TestSecondGameSpellTailKeepsTheDamageKind(t *testing.T) {
	kind, cast := SpellRule{ID: 2, School: 3}, SpellRule{ID: 20, School: 4}
	for _, c := range []struct {
		name   string
		second bool
		area   bool
		spell  int8
		slot   int
	}{
		{"first game point", false, false, 20, 4},
		{"first game area", false, true, 20, 4},
		{"second game point", true, false, 2, 3},
		{"second game area", true, true, 2, 3},
	} {
		w := spWorld(t, 1, []SpellRule{kind, cast}, killCreditMage(1, 4), killCreditVictim(2, 10))
		setTestGame(w, c.second)
		w.resolveDamageAttribution(0, 1, 2)
		if c.area {
			w.areaAttribution(0, 1, cast)
		} else {
			w.pointAttribution(0, 1, cast)
		}
		if got := w.entities[1].KillCreditSpell; got != c.spell {
			t.Fatalf("%s: kill spell %d, want %d", c.name, got, c.spell)
		}
		before := w.entities[0].SkillXP
		w.processKillCredit(1)
		for slot := range before {
			if rose := w.entities[0].SkillXP[slot] > before[slot]; rose != (slot == c.slot) {
				t.Errorf("%s: skill %d rose %v", c.name, slot, rose)
			}
		}
	}
}
