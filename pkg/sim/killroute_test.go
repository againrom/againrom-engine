package sim

import "testing"

// killRouteWorld is a fighter at Mind 24 trained at 20 in blades (index 0)
// and a computer player's creature of value 100 (index 1) under game d.
func killRouteWorld(t *testing.T, d ScriptDialect) *World {
	t.Helper()
	v := awardVictim(3)
	v.ID, v.X = 2, 1
	w := cbWorld(t, 1, awardHero(true, 24, 1, chosenLevels(1)), v)
	w.game = newGameProfile(d)
	w.entities[1].SourceBinding = SourceBinding{Class: GeneratedUnitBinding, Identity: 2, TokenRow: 10}
	return w
}

// The second game's kill route calls the award only for a killer whose health
// is not below zero, and for a non-mage only with spell 0; a call it does not
// make pays nothing and counts nothing (R2-ENGINE-1158). The first game's
// fighter is paid for a kill credited to a spell.
func TestKillRouteCallsTheAwardForTheClaimedKillers(t *testing.T) {
	for _, c := range []struct {
		name    string
		dialect ScriptDialect
		hp      int32
		spell   int8
		want    int32
		counted bool
	}{
		{"second, fighter, spell 0", ScriptROM2, 10, 0, 52, true},
		{"second, fighter credited to a spell", ScriptROM2, 10, 3, 0, false},
		{"second, fighter below zero health", ScriptROM2, -1, 0, 0, false},
		{"second, fighter at zero health", ScriptROM2, 0, 0, 52, true},
		{"first, fighter credited to a spell", ScriptROM1, 10, 3, 52, false},
		{"first, fighter below zero health", ScriptROM1, -1, 0, 52, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := killRouteWorld(t, c.dialect)
			w.entities[0].HP = c.hp
			v := &w.entities[1]
			v.KillCreditSpell = c.spell
			v.KillCreditSource, v.HasKillCredit = w.entities[0].ID, true
			before := w.entities[0].SkillXP[1]
			w.processKillCredit(1)
			if got := w.entities[0].SkillXP[1] - before; got != c.want {
				t.Errorf("the kill paid %d, want %d", got, c.want)
			}
			if counted := w.entities[0].victimKinds.n != 0; counted != c.counted {
				t.Errorf("the kind counted %v, want %v", counted, c.counted)
			}
		})
	}
}
