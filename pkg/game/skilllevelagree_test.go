package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestStoredPartyLevelRuleAgreesWithTheHeldLevel(t *testing.T) {
	var r sim.Rules
	disagree := 0
	games := []struct {
		name    string
		dialect sim.ScriptDialect
	}{{"first", sim.ScriptROM1}, {"second", sim.ScriptROM2}}
	for _, g := range games {
		for n := int32(0); n <= r.SkillCap(); n++ {
			for _, xp := range []int32{r.SkillXP(n) - 1, r.SkillXP(n), r.SkillXP(n) + 1} {
				if xp < 0 {
					continue
				}
				if got, held := data.SkillLevelFor(xp), sim.HeldSkillLevel(g.dialect, r, xp); got != held {
					if g.dialect == sim.ScriptROM1 {
						disagree++
						continue
					}
					t.Errorf("%s game, boundary %d, experience %d: SkillLevelFor %d, HeldSkillLevel %d", g.name, n, xp, got, held)
				}
			}
		}
	}
	if disagree != 200 {
		t.Errorf("first game: %d disagreeing inputs, recorded 200", disagree)
	}
}
