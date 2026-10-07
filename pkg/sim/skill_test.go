package sim

import (
	"math"
	"testing"
)

// skAwarder is one awarding entity for the awardSkill tests below: a roster
// slot, a class that gains, a Mind of 60 and XPSlot 3 — a non-carrier
// (MaxMana 0) by default, so a case that wants a carrier sets MaxMana
// itself and a case that wants a different weapon skill sets XPSlot itself.
// Slot 3 rather than 0, on xpAttacker's own reason (experiencepay_test.go):
// a refusal that left slot 0 alone by accident could not pass by hiding
// behind General's own zero.
func skAwarder(id EntityID, owner uint32) Entity {
	e := cbEnt(id, 0, 0)
	e.Owner, e.GainsXP, e.Mind, e.XPSlot, e.TypeID = owner, true, 60, 3, HumanTypeID
	return e
}

func skSource(id EntityID, owner uint32) Entity {
	e := cbEnt(id, 1, 0)
	e.Owner = owner
	return e
}

func skPay(t *testing.T, a, s Entity, named int32, amount int64) (beforeA, beforeS, afterA, afterS Entity, raised bool) {
	t.Helper()
	w := cbWorld(t, 1, a, s)
	ai, si := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, s.ID)
	beforeA, beforeS = cbAt(t, w, a.ID), cbAt(t, w, s.ID)
	raised = w.awardSkill(ai, named, amount, si)
	afterA, afterS = cbAt(t, w, a.ID), cbAt(t, w, s.ID)
	return
}

func TestAwardSkillRefusesANonGainingClass(t *testing.T) {
	a := skAwarder(1, 2)
	a.GainsXP = false
	s := skSource(2, 3)

	beforeA, beforeS, afterA, afterS, raised := skPay(t, a, s, 0, 1000)
	if raised {
		t.Error("a non-gaining class raised a level")
	}
	if afterA != beforeA {
		t.Errorf("the awarder changed: got %+v, want the untouched %+v", afterA, beforeA)
	}
	if afterS != beforeS {
		t.Errorf("the source changed: got %+v, want the untouched %+v", afterS, beforeS)
	}
}

func TestAwardSkillRefusesTheSameOwnerSlot(t *testing.T) {
	a := skAwarder(1, 5)
	s := skSource(2, 5) // same owner slot as a

	beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1000)
	if raised {
		t.Error("sharing an owner slot raised a level")
	}
	if afterA != beforeA {
		t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
	}
}

func TestAwardSkillRefusesALockedRelation(t *testing.T) {
	a := skAwarder(1, 2)
	s := skSource(2, 3)
	rel := engRel(t, [3]uint32{2, 3, relationLocked})
	w, err := NewRelatedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{a, s}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	ai, si := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, s.ID)
	before := cbAt(t, w, a.ID)

	if raised := w.awardSkill(ai, 0, 1000, si); raised {
		t.Error("a locked relation raised a level")
	}
	if got := cbAt(t, w, a.ID); got != before {
		t.Errorf("the awarder changed: got %+v, want %+v", got, before)
	}
}

func TestAwardSkillDoesNotCareWhetherTheSourceIsAlive(t *testing.T) {
	a := skAwarder(1, 2)
	s := skSource(2, 3)
	s.HP = -50 // already dead

	if _, _, _, _, raised := skPay(t, a, s, 0, 1000); !raised {
		t.Error("an award naming an already-dead source was refused; FR-5.2 does not test aliveness")
	}
}

// TestAwardSkillWithNoSourceSkipsFR52Entirely is the sink's own contract for
// srcIdx < 0 (tasks.md): an award naming no source at all is compared
// against no owner slot and no relation, because there is nothing to compare
// it against.
func TestAwardSkillWithNoSourceSkipsFR52Entirely(t *testing.T) {
	a := skAwarder(1, 2)
	w := cbWorld(t, 1, a)
	ai := indexOfEntity(w.entities, a.ID)

	if raised := w.awardSkill(ai, 0, 1000, -1); !raised {
		t.Error("an award naming no source was refused; want it paid")
	}
}

func TestAwardSkillCarrierTakesTheNamedSlotAndRefusesSlotZero(t *testing.T) {
	a := skAwarder(1, 2)
	a.MaxMana = 50 //
	s := skSource(2, 3)

	t.Run("named slot 0 is refused", func(t *testing.T) {
		beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1000)
		if raised {
			t.Error("a carrier's award naming slot 0 raised a level")
		}
		if afterA != beforeA {
			t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
		}
	})

	t.Run("named slot outside the six is refused", func(t *testing.T) {
		beforeA, _, afterA, _, raised := skPay(t, a, s, 6, 1000)
		if raised {
			t.Error("a carrier's award naming slot 6 raised a level")
		}
		if afterA != beforeA {
			t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
		}
	})

	t.Run("named slot 3 credits slot 3 and nothing else", func(t *testing.T) {
		_, _, afterA, _, raised := skPay(t, a, s, 3, 1000)
		if !raised {
			t.Fatal("a carrier's award naming slot 3 did not raise it")
		}
		for i, lvl := range afterA.Skill {
			if i == 3 {
				if lvl != 1 {
					t.Errorf("slot 3 holds level %d, want 1", lvl)
				}
				continue
			}
			if lvl != 0 {
				t.Errorf("slot %d holds level %d, want untouched", i, lvl)
			}
		}
	})
}

func TestAwardSkillNonCarrierRefusesAnyNamedSlotButZero(t *testing.T) {
	a := skAwarder(1, 2) // MaxMana 0: not a carrier; XPSlot 3
	s := skSource(2, 3)

	beforeA, _, afterA, _, raised := skPay(t, a, s, 3, 1000)
	if raised {
		t.Error("a non-carrier's award naming slot 3 raised a level")
	}
	if afterA != beforeA {
		t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
	}
}

func TestAwardSkillNonCarrierCreditsXPSlotAndRefusesAnEmptyHand(t *testing.T) {
	s := skSource(2, 3)

	t.Run("XPSlot 3 credits slot 3", func(t *testing.T) {
		a := skAwarder(1, 2) // XPSlot 3
		_, _, afterA, _, raised := skPay(t, a, s, 0, 1000)
		if !raised {
			t.Fatal("did not raise slot 3")
		}
		if afterA.Skill[3] != 1 {
			t.Errorf("slot 3 holds level %d, want 1", afterA.Skill[3])
		}
	})

	t.Run("XPSlot 0 (an empty hand) is refused", func(t *testing.T) {
		a := skAwarder(1, 2)
		a.XPSlot = 0
		beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1000)
		if raised {
			t.Error("an empty-handed non-carrier's award raised a level")
		}
		if afterA != beforeA {
			t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
		}
	})
}

func TestAwardSkillRefusesASlotAlreadyAtLevel100(t *testing.T) {
	s := skSource(2, 3)

	t.Run("an ordinary positive amount", func(t *testing.T) {
		a := skAwarder(1, 2)
		a.Skill[3], a.SkillXP[3] = 100, skillXPFor(100)

		beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1_000_000)
		if raised {
			t.Error("a slot at level 100 raised a level")
		}
		if afterA != beforeA {
			t.Errorf("the awarder changed: got %+v, want %+v", afterA, beforeA)
		}
	})

	t.Run("a negative Mind that would otherwise erode the slot", func(t *testing.T) {
		a := skAwarder(1, 2)
		a.Mind = -100
		a.Skill[3], a.SkillXP[3] = 100, skillXPFor(100)

		beforeA, _, afterA, _, raised := skPay(t, a, s, 0, 1000)
		if raised {
			t.Error("a slot at level 100 raised a level")
		}
		if afterA.SkillXP[3] != beforeA.SkillXP[3] {
			t.Errorf("slot 3's experience moved from %d to %d — FR-5.4 must refuse before FR-6's cap "+
				"can let a negative gain through", beforeA.SkillXP[3], afterA.SkillXP[3])
		}
	})
}

func TestAwardSkillScalesByMindBeforeCapping(t *testing.T) {
	// Started 50 below S(10), a buffer neither gain below is anywhere near
	// crossing — what is measured is the amount BANKED, not a raise.
	const startBelow = 50
	build := func(mind int32) Entity {
		a := skAwarder(1, 2)
		a.Mind = mind
		a.Skill[3], a.SkillXP[3] = 10, skillXPFor(10)-startBelow
		return a
	}
	s := skSource(2, 3)

	_, _, lowMind, _, raised1 := skPay(t, build(0), s, 0, 10)
	_, _, highMind, _, raised2 := skPay(t, build(60), s, 0, 10)
	if raised1 || raised2 {
		t.Fatal("fixture: neither case is meant to cross a level boundary")
	}

	lowGain := lowMind.SkillXP[3] - (skillXPFor(10) - startBelow)
	highGain := highMind.SkillXP[3] - (skillXPFor(10) - startBelow)
	if lowGain != 2 {
		t.Errorf("Mind 0 banked %d, want 2 (xpGain(10, 0))", lowGain)
	}
	if highGain != 22 {
		t.Errorf("Mind 60 banked %d, want 22 (xpGain(10, 60))", highGain)
	}
	if lowGain == highGain {
		t.Error("Mind made no difference to the amount banked — FR-6's scaling did not run")
	}
}

func TestAwardSkillCapsAtOneLevelsWorth(t *testing.T) {
	a := skAwarder(1, 2)
	a.Mind = 400 // scaling alone would blow well past any level's cap
	a.Skill[3], a.SkillXP[3] = 20, skillXPFor(20)
	s := skSource(2, 3)

	_, _, afterA, _, raised := skPay(t, a, s, 0, 1_000_000)
	if !raised {
		t.Fatal("an amount this large did not even raise the level")
	}
	want := skillXPFor(21) - skillXPFor(20)
	if bank := afterA.SkillXP[3] - skillXPFor(20); bank != want {
		t.Errorf("banked %d, want exactly %d (S(21)-S(20), the cap)", bank, want)
	}
	if afterA.Skill[3] != 21 {
		t.Errorf("level is %d, want exactly 21 — one raise, not more", afterA.Skill[3])
	}
}

func TestAwardSkillRaisesOnlyWhenStrictlyAboveTheThreshold(t *testing.T) {
	s := skSource(2, 3)
	build := func() Entity {
		a := skAwarder(1, 2)
		a.Mind = 0
		a.Skill[3] = 10
		a.SkillXP[3] = skillXPFor(10) - 30
		return a
	}

	t.Run("exactly at the threshold: no raise", func(t *testing.T) {
		// xpGain(120, 0) = 120*30/120 = 30 exactly, landing SkillXP at
		// precisely S(10).
		_, _, afterA, _, raised := skPay(t, build(), s, 0, 120)
		if afterA.SkillXP[3] != skillXPFor(10) {
			t.Fatalf("fixture: banked to %d, want exactly S(10)=%d", afterA.SkillXP[3], skillXPFor(10))
		}
		if raised {
			t.Error("landing exactly at S(10) raised the level; FR-8 wants strictly above")
		}
		if afterA.Skill[3] != 10 {
			t.Errorf("level is %d, want the unraised 10", afterA.Skill[3])
		}
	})

	t.Run("one point above the threshold: raises", func(t *testing.T) {
		// xpGain(124, 0) = 124*30/120 = 31, one above S(10)'s own gap.
		_, _, afterA, _, raised := skPay(t, build(), s, 0, 124)
		if !raised {
			t.Fatal("one point above the threshold did not raise the level")
		}
		if afterA.Skill[3] != 11 {
			t.Errorf("level is %d, want 11", afterA.Skill[3])
		}
	})
}

func TestP3NoSingleAwardRaisesASlotByMoreThanOne(t *testing.T) {
	s := skSource(2, 3)
	amounts := []int64{1, 100, 1000, 1_000_000, 1 << 40}
	for level := int32(0); level < 100; level += 7 {
		for _, amount := range amounts {
			a := skAwarder(1, 2)
			a.Mind = 100
			a.Skill[3], a.SkillXP[3] = level, skillXPFor(level)
			_, _, afterA, _, _ := skPay(t, a, s, 0, amount)
			if delta := afterA.Skill[3] - level; delta > 1 {
				t.Errorf("level %d amount %d: level moved by %d, want at most 1", level, amount, delta)
			}
		}
	}
}

// ---------------------------------------------------------------------- AC-1, AC-3, AC-4

// TestAC1ANonCarrierBlowRaisesExactlyTheCreditedSlotsLevel is AC-1, driven
// through the whole attack cycle: a non-carrier that gains, holding a
// weapon whose skill is not slot 0, lands one blow on a hostile of another
// owner and ends the tick with that slot's level exactly one higher and
// every other slot untouched.
func TestAC1ANonCarrierBlowRaisesExactlyTheCreditedSlotsLevel(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 3, HumanTypeID
	a.Skill[3], a.SkillXP[3] = 10, skillXPFor(10)
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 3, 10
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})

	got := cbAt(t, w, 1)
	for i, lvl := range got.Skill {
		if i == int(a.XPSlot) {
			if lvl != 11 {
				t.Errorf("credited slot %d holds level %d, want exactly 11", i, lvl)
			}
			continue
		}
		if lvl != 0 {
			t.Errorf("slot %d holds level %d, want the starting zero", i, lvl)
		}
	}
}

func TestAC3ACarrierBlowChangesNoLevelAndNoExperience(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 3, HumanTypeID
	a.MaxMana = 50 // a carrier
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 3, 10
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})

	got := cbAt(t, w, 1)
	if got.Skill != a.Skill {
		t.Errorf("levels moved: got %v, want %v", got.Skill, a.Skill)
	}
	if got.SkillXP != a.SkillXP {
		t.Errorf("experiences moved: got %v, want %v", got.SkillXP, a.SkillXP)
	}
}

// TestAC4ANonCarrierWithWeaponSkillZeroGainsNothingFromABlow is AC-4's
// first clause, driven through the whole attack cycle rather than through
// the sink alone (TestAwardSkillNonCarrierCreditsXPSlotAndRefusesAnEmptyHand
// covers the sink): XPSlot 0 — an empty hand, or a weapon whose kind names
// no skill — earns nothing from a landed blow.
func TestAC4ANonCarrierWithWeaponSkillZeroGainsNothingFromABlow(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 0, HumanTypeID
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 3, 10
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})

	got := cbAt(t, w, 1)
	if got.Skill != a.Skill || got.SkillXP != a.SkillXP {
		t.Errorf("slot 0 gained something: Skill %v SkillXP %v", got.Skill, got.SkillXP)
	}
}

func TestSpellAwardAmountMatchesTheRealValuedForm(t *testing.T) {
	for manaCost := int32(0); manaCost <= 2000; manaCost++ {
		got := (int64(manaCost) + 1) / 2
		want := int64(math.Trunc(float64(manaCost)*0.5 + 0.5))
		if got != want {
			t.Fatalf("manaCost %d: (manaCost+1)/2 = %d, want %d (ftol(manaCost*0.5+0.5))", manaCost, got, want)
		}
	}
}

func TestAC7ACastRaisesTheSpellsOwnSchool(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 2, MaxRange: 5, DamageMin: 30, DamageMax: 60, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	caster.Owner, caster.GainsXP, caster.TypeID = 2, true, HumanTypeID
	caster.Skill[2], caster.SkillXP[2] = 10, skillXPFor(10)
	victim := spEnt(2, 3, 0)
	victim.Owner = 3
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)
	before := SpellCharacteristicsFor(Rules{}, caster, spell)

	spRunCast(w, spCast(1, 2, 1))

	got := spAt(t, w, 1)
	after := SpellCharacteristicsFor(Rules{}, got, spell)
	if got.Skill[2] != 11 {
		t.Errorf("school 2's level is %d, want 11", got.Skill[2])
	}
	for i, lvl := range got.Skill {
		if i != 2 && lvl != 0 {
			t.Errorf("slot %d moved to %d, want untouched", i, lvl)
		}
	}
	if after.Power <= before.Power || after.RecordPower <= before.RecordPower || after.Range < before.Range {
		t.Errorf("live popup projection did not follow level-up: before %+v after %+v", before, after)
	}
	if after.ManaCost != before.ManaCost {
		t.Errorf("flat mana cost moved from %d to %d", before.ManaCost, after.ManaCost)
	}
}

func TestSpellCharacteristicsAreAProjectionOfTheCurrentActor(t *testing.T) {
	t.Parallel()

	rule := SpellRule{ID: 6, ManaCost: 10, School: 2, MaxRange: 6,
		DamageMin: 10, DamageMax: 20, Restorative: true, TargetsUnit: true}
	a := spMage(1, 0, 0, 30, 50, 50, 1<<6)
	a.Skill[2], a.SkillXP[2] = 4, 90
	b := a
	b.Mind, b.Skill[2], b.SkillXP[2] = 60, 12, 240
	pa, pb := SpellCharacteristicsFor(Rules{}, a, rule), SpellCharacteristicsFor(Rules{}, b, rule)
	if pa.SkillLevel != 4 || pa.SkillXP != 90 || pb.SkillLevel != 12 || pb.SkillXP != 240 {
		t.Fatalf("hero switch leaked projection state: A %+v B %+v", pa, pb)
	}
	if pb.Power <= pa.Power || pb.RecordPower <= pa.RecordPower || pb.Range < pa.Range {
		t.Errorf("Mind/skill change did not move canonical characteristics: A %+v B %+v", pa, pb)
	}
	if again := SpellCharacteristicsFor(Rules{}, a, rule); again != pa {
		t.Errorf("switching back to A returned %+v, want current A %+v", again, pa)
	}
}

func TestACastAtAHigherSchoolLevelTakesMoreHealth(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 2, MaxRange: 5, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}
	cast := func(level int32) int32 {
		t.Helper()
		caster := spMage(1, 0, 0, 30, 50, 20, 1<<1)
		caster.Owner = 2
		caster.Skill[2] = level
		victim := spEnt(2, 3, 0)
		victim.Owner = 3
		w := spWorld(t, 42, []SpellRule{spell}, caster, victim)
		spRunCast(w, spCast(1, 2, 1))
		return spAt(t, w, 2).HP
	}
	low, high := cast(0), cast(40)
	if !(high < low) {
		t.Errorf("a caster at school level 40 left the victim at %d and one at level 0 left it at %d, "+
			"want the higher level to take strictly more", high, low)
	}
}

func TestAC7DamageRisesWithTheSchoolsLevel(t *testing.T) {
	const mind = 0
	lowPower := spellPower(0, mind)
	highPower := spellPower(40, mind)
	if !(highPower > lowPower) {
		t.Fatalf("fixture: power did not rise with level (%d vs %d)", lowPower, highPower)
	}

	lowBase, lowSpread := spellDamage(4, 8, lowPower)
	highBase, highSpread := spellDamage(4, 8, highPower)
	if !(highBase >= lowBase && highBase+highSpread > lowBase+lowSpread) {
		t.Errorf("damage did not rise with the school's level: low [%d,%d] high [%d,%d]",
			lowBase, lowBase+lowSpread, highBase, highBase+highSpread)
	}
}
