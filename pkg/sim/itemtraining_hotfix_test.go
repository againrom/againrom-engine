package sim

import "testing"

func itemTrainingCaster(spell uint16, school int) Entity {
	e := wpnCaster(1, 0, 0, spell, 30, 1, 0)
	e.Owner, e.GainsXP, e.TypeID, e.Mind = 2, true, HumanTypeID, 60
	e.Skill[school] = 10
	e.SkillXP[school] = skillXPFor(10)
	return e
}

// Deleting the pseudo-damage call in ordinaryEffect leaves the Stone Curse
// attached and therefore kills the visible effect control while this school
// assertion alone catches the missing staff-training mechanism.
func TestStaffStoneCurseRaisesEarthFromPseudoDamageOnly(t *testing.T) {
	rule := SpellRule{ID: 20, School: 4, TargetsUnit: true, MaxRange: 5,
		EffectKind: EffectAbsorption, EffectMagnitude: 5, EffectMode: EffectDuration, SpellDuration: 10}
	caster := itemTrainingCaster(20, 4)
	victim := spEnt(2, 1, 0)
	victim.Owner, victim.XPValue = 3, 100
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	got := spAt(t, w, 1)
	if got.Skill[4] != 11 || got.SkillXP[4] != skillXPFor(10)+4 {
		t.Fatalf("Earth result = level %d XP %d, want 11/%d from the 3%% pseudo-damage award",
			got.Skill[4], got.SkillXP[4], skillXPFor(10)+4)
	}
	if got.Mana != caster.Mana {
		t.Fatalf("staff Stone Curse spent mana: %d -> %d", caster.Mana, got.Mana)
	}
}

func TestItemAreaTrainsPerDamageEventWithoutHalfManaAward(t *testing.T) {
	rule := SpellRule{ID: 2, School: 1, ManaCost: 100, TargetsUnit: true, MaxRange: 5,
		Area: true, Radius: 1, Damaging: true, DamageMin: 1, DamageMax: 1}
	caster := itemTrainingCaster(2, 1)
	a, b := spEnt(2, 2, 2), spEnt(3, 3, 2)
	a.Owner, a.XPValue, b.Owner, b.XPValue = 3, 100, 4, 100
	w := spWorld(t, 1, []SpellRule{rule}, caster, a, b)

	Step(w, []Command{cbOrder(1, 2)})

	got := spAt(t, w, 1)
	if got.SkillXP[1] != skillXPFor(10)+8 {
		t.Fatalf("item Fire Ball banked %d, want two four-XP damage events and no half-mana award",
			got.SkillXP[1])
	}
}

func TestPoisonTicksCarrySignedAwardsAndClearOnlyBelowZeroCasterHealth(t *testing.T) {
	rule := SpellRule{ID: 8, School: 2, EffectKind: EffectHealth, EffectMode: EffectContinuous}
	caster := itemTrainingCaster(8, 2)
	caster.HP = 0 // exactly zero retains attribution
	target := spEnt(2, 1, 0)
	target.Owner, target.XPValue, target.HP = 3, 100, 50
	w := spWorld(t, 1, []SpellRule{rule}, caster, target)

	if !w.attachEffect(2, 1, rule, EffectHealth, 4, 8, EffectContinuous) {
		t.Fatal("positive health delta did not attach as a custom negative-power Poison control")
	}
	if got := spAt(t, w, 1).SkillXP[2]; got != skillXPFor(10)-2 {
		t.Fatalf("healing Poison tick banked %d, want signed reduction to %d", got, skillXPFor(10)-2)
	}
	if !w.attached[0].HasCaster {
		t.Fatal("caster at exactly zero health was cleared")
	}

	w.entities[0].HP = -1
	Step(w, nil) // old remaining8 -> 7: a continuous tick whose caster is now below zero
	if w.attached[0].HasCaster {
		t.Fatal("negative-health caster remained attributed to Poison")
	}
	if got := spAt(t, w, 1).SkillXP[2]; got != skillXPFor(10)-2 {
		t.Fatalf("cleared Poison caster received another award: %d", got)
	}
	w.entities[0].HP, w.entities[0].Decay, w.entities[0].Dwell = 1, DecayNone, 0

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary signed XP: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary signed XP: %v", err)
	}
	if got := spAt(t, &back, 1).SkillXP[2]; got != skillXPFor(10)-2 {
		t.Fatalf("signed XP after save/load = %d, want %d", got, skillXPFor(10)-2)
	}
}

func TestLowTypeHumanCannotTrainFromStaffDamage(t *testing.T) {
	rule := wpnRule(2, 2, 5)
	rule.School = 1
	caster := itemTrainingCaster(1, 1)
	caster.TypeID = 0x17
	victim := spEnt(2, 1, 0)
	victim.Owner, victim.XPValue = 3, 100
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if got := spAt(t, w, 1).SkillXP[1]; got != skillXPFor(10) {
		t.Fatalf("low-TypeID Human trained to %d, want unchanged %d", got, skillXPFor(10))
	}
}
