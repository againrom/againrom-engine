package sim

import (
	"bytes"
	"testing"
)

func killCreditMage(id EntityID, school uint8) Entity {
	e := itemTrainingCaster(1, int(school))
	e.ID = id
	e.WeaponSpell = 0
	return e
}

func killCreditFighter(id EntityID, slot uint8) Entity {
	e := spEnt(id, 0, 0)
	e.Owner, e.TypeID, e.GainsXP, e.Mind, e.XPSlot = 2, HumanTypeID, true, 60, slot
	e.Skill[slot] = 10
	e.SkillXP[slot] = skillXPFor(10)
	return e
}

func killCreditVictim(id EntityID, hp int32) Entity {
	e := spEnt(id, 1, 0)
	e.Owner, e.TypeID, e.HP, e.XPValue = 3, 10, hp, 100
	return e
}

// The damaging resolver first records the damage kind, then the admitted
// PointEffect tail replaces it with the actual spell id. The death consumer
// must see that final id in the same tick and add the distinct half-XP award.
func TestDirectItemSpellKillUsesActualSpellForDelayedCredit(t *testing.T) {
	rule := wpnRule(1, 1, 5)
	rule.School = 1
	caster := itemTrainingCaster(rule.ID, int(rule.School))
	victim := killCreditVictim(2, 1)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	before := spAt(t, w, 1).SkillXP[1]
	Step(w, []Command{cbOrder(1, 2)})

	got := spAt(t, w, 1)
	if v := spAt(t, w, 2); !v.Dead() || !v.HasKillCredit || v.KillCreditSource != 1 || v.KillCreditSpell != int8(rule.ID) {
		t.Fatalf("lethal PointEffect attribution = dead %v, source %d/%v, spell %d", v.Dead(), v.KillCreditSource, v.HasKillCredit, v.KillCreditSpell)
	}
	// Power 30 turns the fixed one damage into two: four scaled XP for the
	// damage event, then 112 for XPValue/2 at Mind 60.
	if want := before + int32(xpGain(xpRaw(100, 2, 100), 60)+xpGain(50, 60)); got.SkillXP[1] != want {
		t.Fatalf("direct lethal item spell banked %d, want %d", got.SkillXP[1], want)
	}
}

func TestDelayedDeathConsumesSurvivingAttributionOnce(t *testing.T) {
	rule := SpellRule{ID: 20, School: 4}
	caster := killCreditMage(1, rule.School)
	victim := killCreditVictim(2, 10)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
	w.pointAttribution(0, 1, rule)
	before := spAt(t, w, 1).SkillXP[4]

	Step(w, []Command{{Kind: KindKill, Entity: 2}})
	if got, want := spAt(t, w, 1).SkillXP[4], before+int32(xpGain(50, 60)); got != want {
		t.Fatalf("delayed death banked %d, want %d", got, want)
	}
	Step(w, nil)
	if got, want := spAt(t, w, 1).SkillXP[4], before+int32(xpGain(50, 60)); got != want {
		t.Fatalf("dead victim paid again: %d, want %d", got, want)
	}
}

func TestZeroDwellFlyingDeathPaysBeforeTheBodyIsRemoved(t *testing.T) {
	rule := SpellRule{ID: 20, School: 4}
	caster := killCreditMage(1, rule.School)
	victim := killCreditVictim(2, 10)
	victim.Domain, victim.DyingTime = DomainAir, 1
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)
	w.pointAttribution(0, 1, rule)
	before := spAt(t, w, 1).SkillXP[4]

	Step(w, []Command{{Kind: KindKill, Entity: 2}})
	if indexOfEntity(w.entities, 2) >= 0 {
		t.Fatal("one-dwell flying body was not removed")
	}
	if got, want := spAt(t, w, 1).SkillXP[4], before+int32(xpGain(50, 60)); got != want {
		t.Fatalf("removed flying victim paid %d, want %d", got, want)
	}
}

func TestLaterPointAttributionRedirectsKillAndFighterUsesCurrentWeaponSkill(t *testing.T) {
	rules := []SpellRule{{ID: 1, School: 1}, {ID: 20, School: 4}}
	first := killCreditMage(1, 1)
	later := killCreditFighter(3, 5)
	victim := killCreditVictim(2, 10)
	w := spWorld(t, 1, rules, first, victim, later)
	w.pointAttribution(0, 1, rules[0])
	w.pointAttribution(2, 1, rules[1])
	firstBefore := spAt(t, w, 1).SkillXP[1]
	laterBefore := spAt(t, w, 3).SkillXP[5]

	Step(w, []Command{{Kind: KindKill, Entity: 2}})
	if got := spAt(t, w, 1).SkillXP[1]; got != firstBefore {
		t.Fatalf("overwritten mage received kill credit: %d -> %d", firstBefore, got)
	}
	if got, want := spAt(t, w, 3).SkillXP[5], laterBefore+int32(xpGain(50, 60)); got != want {
		t.Fatalf("fighter kill credit banked %d in current weapon slot, want %d", got, want)
	}
}

func TestAttributionClearAndRetainWritersAreShapeSpecific(t *testing.T) {
	rule := SpellRule{ID: 20, School: 4}
	definedOwnerless := killCreditMage(1, 4)
	definedOwnerless.Owner = 0
	nullDefinition := killCreditMage(3, 4)
	nullDefinition.TypeID = 0
	victim := killCreditVictim(2, 10)
	w := spWorld(t, 1, []SpellRule{rule}, definedOwnerless, victim, nullDefinition)
	seed := func() {
		w.entities[1].KillCreditSource = 9
		w.entities[1].HasKillCredit = true
		w.entities[1].KillCreditSpell = 7
	}
	kept := func(where string) {
		t.Helper()
		v := w.entities[1]
		if !v.HasKillCredit || v.KillCreditSource != 9 || v.KillCreditSpell != 7 {
			t.Fatalf("%s did not retain attribution: %+v", where, v)
		}
	}
	cleared := func(where string) {
		t.Helper()
		v := w.entities[1]
		if v.HasKillCredit || v.KillCreditSource != 0 {
			t.Fatalf("%s did not clear source pointer: %+v", where, v)
		}
		if v.KillCreditSpell != 7 {
			t.Fatalf("%s cleared spell history too: %d", where, v.KillCreditSpell)
		}
	}

	seed()
	w.resolveDamageAttribution(2, 1, 4) // null definition retains
	kept("damage resolver null definition")
	seed()
	w.resolveDamageAttribution(0, 1, 4) // defined source, null owner clears
	cleared("damage resolver null owner")
	seed()
	w.pointAttribution(0, 1, rule) // Point tail retains a null owner
	kept("point null owner")
	seed()
	w.pointAttribution(2, 1, rule) // Point tail clears a null definition
	cleared("point null definition")
	seed()
	w.areaAttribution(0, 1, rule) // Area tail clears either loss
	cleared("area null owner")
	seed()
	w.areaAttribution(2, 1, rule)
	cleared("area null definition")
}

func TestDrainAndRefusedEffectsDoNotRewritePriorAttribution(t *testing.T) {
	drain := SpellRule{ID: 11, School: 3, DamageMin: 1, DamageMax: 1}
	caster := killCreditMage(1, 3)
	caster.HP = 50
	victim := killCreditVictim(2, 100)
	w := spWorld(t, 1, []SpellRule{drain}, caster, victim)
	w.entities[1].KillCreditSource, w.entities[1].HasKillCredit, w.entities[1].KillCreditSpell = 9, true, 7

	if !w.ordinaryEffect(0, 1, drain, 30) {
		t.Fatal("Drain control was refused")
	}
	if v := w.entities[1]; !v.HasKillCredit || v.KillCreditSource != 9 || v.KillCreditSpell != 7 {
		t.Fatalf("Drain rewrote prior attribution: %+v", v)
	}
	w.entities[1].HP = -1
	if w.ordinaryEffect(0, 1, SpellRule{ID: 20, School: 4}, 30) {
		t.Fatal("ordinary effect applied to a dead target")
	}
	if v := w.entities[1]; !v.HasKillCredit || v.KillCreditSource != 9 || v.KillCreditSpell != 7 {
		t.Fatalf("refused effect rewrote prior attribution: %+v", v)
	}
}

func TestPoisonTickDoesNotRefreshOverwrittenAreaAttribution(t *testing.T) {
	poison := SpellRule{ID: 8, School: 2, EffectKind: EffectHealth, EffectMode: EffectContinuous,
		EffectMagnitude: -1, EffectDuration: 8}
	poisoner := killCreditMage(1, 2)
	later := killCreditMage(3, 4)
	victim := killCreditVictim(2, 1)
	w := spWorld(t, 1, []SpellRule{poison, SpellRule{ID: 20, School: 4}}, poisoner, victim, later)

	if !w.ordinaryAreaEffect(0, 1, poison, 0) {
		t.Fatal("Poison area did not attach")
	}
	if !w.entities[1].Downed() || w.entities[1].KillCreditSource != 1 || w.entities[1].KillCreditSpell != 8 {
		t.Fatalf("initial Poison result = %+v", w.entities[1])
	}
	w.pointAttribution(2, 1, SpellRule{ID: 20, School: 4})
	poisonBefore := spAt(t, w, 1).SkillXP[2]
	laterBefore := spAt(t, w, 3).SkillXP[4]

	Step(w, nil) // old8 -> 7: the continuous tick kills without refreshing id 8
	if v := spAt(t, w, 2); !v.Dead() || v.KillCreditSource != 3 || v.KillCreditSpell != 20 {
		t.Fatalf("Poison tick changed overwritten attribution: %+v", v)
	}
	if got, want := spAt(t, w, 1).SkillXP[2], poisonBefore+int32(xpGain(1, 60)); got != want {
		t.Fatalf("Poison tick damage award = %d, want %d", got, want)
	}
	if got, want := spAt(t, w, 3).SkillXP[4], laterBefore+int32(xpGain(50, 60)); got != want {
		t.Fatalf("stale Poison death credit = %d, want redirected %d", got, want)
	}
}

func TestKillAttributionRoundTripsHashesAndRejectsInvalidTails(t *testing.T) {
	caster := killCreditMage(1, 4)
	victim := killCreditVictim(2, 10)
	victim.KillCreditSource, victim.HasKillCredit, victim.KillCreditSpell = 1, true, -7
	w := spWorld(t, 1, []SpellRule{{ID: 20, School: 4}}, caster, victim)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if v := spAt(t, &back, 2); !v.HasKillCredit || v.KillCreditSource != 1 || v.KillCreditSpell != -7 {
		t.Fatalf("round-trip attribution = %+v", v)
	}

	plainVictim := victim
	plainVictim.KillCreditSource, plainVictim.HasKillCredit, plainVictim.KillCreditSpell = 0, false, 0
	plain := spWorld(t, 1, []SpellRule{{ID: 20, School: 4}}, caster, plainVictim)
	plainForm, err := plain.MarshalBinary()
	if err != nil {
		t.Fatalf("plain MarshalBinary: %v", err)
	}
	if bytes.Equal(form, plainForm) || w.Hash() == plain.Hash() {
		t.Fatal("attribution did not distinguish canonical form and hash")
	}

	base := headerLen + 3*16*16
	tail := base + entityLen + 261
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"presence byte 2", withByte(form, tail+4, 2)},
		{"absent pointer with source residue", withByte(form, tail+4, 0)},
		{"present pointer naming no entity", withU32(form, tail, 99)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got World
			if err := got.UnmarshalBinary(tc.data); err == nil {
				t.Fatal("invalid attribution tail was accepted")
			}
		})
	}
}
