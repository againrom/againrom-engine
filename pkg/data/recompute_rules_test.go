package data_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/rules"
)

func TestRecomputeAddsABonusAboveOneHundredWhateverTheTrainingCap(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30, Mind: 30, Spirit: 30}
	h.Skill[1] = 100
	var l data.Loadout
	l.Mod.SkillBonus[1] = 10
	if got := h.Recompute(data.Profile{}, l).Skill[1]; got != 110 {
		t.Fatalf("without rules the level is %d, want 110", got)
	}
	l.Rules, _ = rules.New(rules.Params{SkillCap: 150})
	if got := h.Recompute(data.Profile{}, l).Skill[1]; got != 110 {
		t.Fatalf("under a training cap of 150 the level is %d, want 110", got)
	}
	l.Rules, _ = rules.New(rules.Params{SkillCap: 120})
	if got := h.Recompute(data.Profile{}, l).Skill[1]; got != 110 {
		t.Fatalf("under a training cap of 120 the level is %d, want 110", got)
	}
	if got := h.Recompute(data.Profile{}, data.Loadout{}).Skill[1]; got != 100 {
		t.Fatalf("a hero with no bonus holds %d, want 100", got)
	}
}

func TestRecomputeBoundsTheEffectiveLevelAndKeepsTheExperienceOnTheBase(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30, Mind: 30, Spirit: 30}
	h.Skill[2] = 100
	var l data.Loadout
	l.Mod.SkillBonus[2] = 900
	d := h.Recompute(data.Profile{}, l)
	if d.Skill[2] != rules.EffectiveSkillBound {
		t.Fatalf("level %d, want the bound %d", d.Skill[2], rules.EffectiveSkillBound)
	}
	if want := h.Recompute(data.Profile{}, data.Loadout{}).SkillXP[2]; d.SkillXP[2] != want {
		t.Fatalf("a bonus moved the slot experience: %d, want %d", d.SkillXP[2], want)
	}
	l.Mod.SkillBonus[2] = -300
	if got := h.Recompute(data.Profile{}, l).Skill[2]; got != 0 {
		t.Fatalf("level %d, want the floor 0", got)
	}
}

func TestRecomputeExperienceSumSaturatesInsteadOfWrapping(t *testing.T) {
	var live [data.SkillSlots]int32
	for i := range live {
		live[i] = 2_000_000_000
	}
	h := data.Hero{Body: 30, Reaction: 30, Mind: 30, Spirit: 30}
	if got := h.RecomputeWithSkillXP(data.Profile{}, data.Loadout{}, live).Experience; got != 2147483647 {
		t.Fatalf("experience %d, want the int32 maximum", got)
	}
}

func TestSkillXPForPastTheTableContinuesTheCurve(t *testing.T) {
	if data.SkillXPFor(101) <= data.SkillXPFor(100) {
		t.Fatal("the curve stops at 100")
	}
}

// The town-1 inn mage (BODY 35, AGILITY 40, MIND 45, SPIRIT 50, General,
// Water and Air at 99) reads HEALTH 158 and MANA 604 under the second
// game's pool rule and 163 and 623 under the first's (R2-ENGINE-579).
func TestRecomputeSecondGamePoolsLeaveGeneralOut(t *testing.T) {
	h := data.Hero{Body: 35, Reaction: 40, Mind: 45, Spirit: 50, Skill: [data.SkillSlots]int32{99, 0, 99, 99, 0, 0}}
	p := data.Profile{HealthColumn: true, ManaColumn: true}
	for _, c := range []struct {
		rules    rules.Rules
		hp, mana int32
	}{{rules.Rules{}, 163, 623}, {rules.Rules{}.WithSchoolPoolExperience(), 158, 604}} {
		d := h.Recompute(p, data.Loadout{Rules: c.rules})
		if d.HealthMax != c.hp || d.ManaMax != c.mana {
			t.Errorf("pools %d/%d, want %d/%d", d.HealthMax, d.ManaMax, c.hp, c.mana)
		}
	}
}
