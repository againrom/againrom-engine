package sim

import "testing"

func TestHumanExperienceValueUsesLiveClassSkillsAndKeepsUnitTemplates(t *testing.T) {
	xp := [skillSlots]int32{999999, 100, 200, 300, 400, 599}
	for _, e := range []Entity{
		{Humanoid: true, SkillXP: xp},
		{Humanoid: true, SkillXP: xp, XPValue: 9876},
		{SkillXP: xp, ActorLoad: ActorLoad{Source: SourceActor{Class: 2}}},
	} {
		if got := e.ExperienceValue(); got != 15 {
			t.Fatalf("Human value=%d, want floor(1599/100)=15, excluding General", got)
		}
		e.SkillXP[5]++
		if e.ExperienceValue() != 16 {
			t.Fatal("live skill progress left a stale Human value")
		}
	}
	if got := (Entity{XPValue: 4986, SkillXP: xp}).ExperienceValue(); got != 4986 {
		t.Fatalf("flat Unit value=%d, want installed4986", got)
	}
}

func TestHumanTargetExperienceScalesHitsAndSpellsAfterOldSaveLoad(t *testing.T) {
	// Literal target values from two Human populations in the owner's native
	// mission90 save. Their old stored XPValue is zero in both populations.
	for _, tc := range []struct{ xp, hp, want int32 }{
		{303400, 261, 132}, {2047400, 397, 580},
	} {
		for _, spell := range []bool{false, true} {
			a, target := xpAttacker(1), xpTarget(2)
			a.Skill[3], a.SkillXP[3] = 30, skillXPFor(30)
			if spell {
				a.MaxMana = 100
			}
			target.Humanoid, target.XPValue = true, 0
			target.SkillXP[1], target.HP, target.MaxHP = tc.xp, tc.hp, tc.hp
			w := cbWorld(t, 1, a, target)
			current := mustMarshal(t, w)
			var cold World
			if err := cold.UnmarshalBinary(current); err != nil {
				t.Fatal(err)
			}
			before := cold.entities[0].SkillXP[3]
			if spell {
				cold.awardSpellDamage(0, 1, SpellRule{School: 3}, 10)
			} else {
				cold.payExperience(0, 1, 10, true)
			}
			if gain := cold.entities[0].SkillXP[3] - before; gain != tc.want {
				t.Fatalf("Human XP=%d HP=%d spell=%v: gain=%d, want%d", tc.xp, tc.hp, spell, gain, tc.want)
			}
		}
	}
}

func TestHumanTargetKillCreditUsesTheSameExperienceValue(t *testing.T) {
	a, target := xpAttacker(1), xpTarget(2)
	a.Skill[3], a.SkillXP[3] = 30, skillXPFor(30)
	target.Humanoid, target.XPValue, target.SkillXP[2] = true, 0, 20000
	w := cbWorld(t, 1, a, target)
	w.entities[1].HP = -1
	w.entities[1].HasKillCredit, w.entities[1].KillCreditSource = true, a.ID
	before := w.entities[0].SkillXP[3]
	w.processKillCredit(1)
	if got := w.entities[0].SkillXP[3] - before; got != 225 {
		t.Fatalf("Human kill award=%d, want (200/2)*2.25=225", got)
	}
}
