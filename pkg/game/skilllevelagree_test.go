package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func storedPartyAt(t *testing.T, f *FrontEnd, xp int32) [data.SkillSlots]int32 {
	t.Helper()
	c, err := storedFrom("Stored", make([]int, chargenChoiceCount), [4]uint32{31, 27, 22, 26})
	if err != nil {
		t.Fatal(err)
	}
	for s := data.SkillBlade; s < data.SkillSlots; s++ {
		c.skillXP[s] = xp
	}
	c.hasXP = true
	party := f.storedParty(c)
	if len(party) == 0 {
		t.Fatal("no party")
	}
	return party[0].Hero.Skill
}

func TestStoredPartyLevelRuleAgreesWithTheHeldLevel(t *testing.T) {
	var r sim.Rules
	cases := sessionCases()
	games := []struct {
		name    string
		dialect sim.ScriptDialect
		front   *FrontEnd
	}{{"first", sim.ScriptROM1, cases[0].front(t)}, {"second", sim.ScriptROM2, cases[1].front(t)}}
	for _, g := range games {
		for n := int32(0); n <= r.SkillCap(); n++ {
			for _, xp := range []int32{r.SkillXP(n) - 1, r.SkillXP(n), r.SkillXP(n) + 1} {
				if xp < 0 {
					continue
				}
				held := sim.HeldSkillLevel(g.dialect, r, xp)
				got := storedPartyAt(t, g.front, xp)
				for s := data.SkillBlade; s < data.SkillSlots; s++ {
					if got[s] != held {
						t.Errorf("%s game, boundary %d, experience %d, skill %d: stored party %d, held %d", g.name, n, xp, s, got[s], held)
					}
				}
			}
		}
	}
}

func TestFirstGameStoredPartyHoldsTheTrainedLevel(t *testing.T) {
	f := sessionCases()[0].front(t)
	for s, level := range storedPartyAt(t, f, 101) {
		if s >= int(data.SkillBlade) && level != 2 {
			t.Errorf("skill %d level %d, want 2", s, level)
		}
	}
}
