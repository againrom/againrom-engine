package sim

import "testing"

// ---------------------------------------------------------------- fixtures

// wpnEnt is one entity for these tests: an id, a cell, a health pool wide
// enough that no release in this file fells it by accident, and a dwell
// long enough that a body built already fallen (none is) would stay in the
// world for the run — cbEnt's and spEnt's own reasons (combat_test.go,
// spell_test.go).
func wpnEnt(id EntityID, x, y int32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200}
}

func wpnCaster(id EntityID, x, y int32, spell uint16, level, charge, relax int32) Entity {
	e := wpnEnt(id, x, y)
	e.MaxMana, e.Mana = 100, 100
	e.ScanRange = sightRings
	e.WeaponSpell, e.WeaponSpellLevel = spell, level
	e.AttackCharge, e.AttackRelax = charge, relax
	e.Reach = 1
	return e
}

func wpnRule(dmin, dmax int32, maxRange uint8) SpellRule {
	return SpellRule{ID: 1, ManaCost: 5, MaxRange: maxRange, DamageMin: dmin, DamageMax: dmax,
		TargetsUnit: true, Damaging: true}
}

func wpnSameExceptCycle(t *testing.T, attacker EntityID, before, after []Entity) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("entity count moved: %d before, %d after", len(before), len(after))
	}
	for i := range before {
		b, a := before[i], after[i]
		if b.ID == attacker {
			b.AttackPhase, b.AttackCountdown = 0, 0
			a.AttackPhase, a.AttackCountdown = 0, 0
		}
		if b != a {
			t.Errorf("entity %d changed beyond the attacker's own cycle:\n before %+v\n after  %+v",
				before[i].ID, before[i], after[i])
		}
	}
}

func TestAWeaponBorneCastReplacesTheStrike(t *testing.T) {
	rule := wpnRule(6, 6, 5) // spread 0 at power 30: base = 6*(30+30)/30 = 12
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.DamageBase, caster.AlwaysHits = 900, true // what a blow would remove instead
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-12 {
		t.Errorf("the victim is at %d, want %d — a cast of exactly 12 (spread zero at power 30)", hp, 100-12)
	}
	if p := spAt(t, w, 1).AttackPhase; p != AttackRelaxing {
		t.Errorf("the caster's phase is %v, want AttackRelaxing after one release", p)
	}
}

// TestANonCasterOrANoSpellWeaponStrikesNormally is AC-5's "no spell" clause:
// a mana pool but a weapon carrying no spell strikes an ordinary blow and
// releases nothing — WeaponSpell's own zero, read alone inside both
// weaponSpellFor (the caster's own arm) and weaponRiderSpellFor (the
// fighter's rider, R3-B3).
//
// THE LINE THIS TEST WITNESSES is weaponSpellFor's own `e.WeaponSpell == 0`
// half of its guard (verified by hand): normaliseSpells (above) refuses id 0
// as a row out of the table's own construction, so findSpell(0) can never
// match and this half is redundant with that invariant rather than
// independently falsifiable here — kept in weaponSpellFor for the reason
// its own doc gives (WeaponSpell's field doc, world.go), not because this
// test can tell its absence apart from findSpell's.
func TestANonCasterOrANoSpellWeaponStrikesNormally(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 0, 0, 0, 0, 1, 0) // WeaponSpell 0: no spell at all
	caster.DamageBase, caster.AlwaysHits = 40, true
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-40 {
		t.Errorf("the victim is at %d, want %d — an ordinary blow of 40", hp, 100-40)
	}
	if fx := spAt(t, w, 2).SpellFX; fx != 0 {
		t.Errorf("the victim carries a spell effect mark (SpellFX=%d), want none — no row to release", fx)
	}
}

// TestANoManaPoolFighterStrikesAndTakesTheWeaponsOwnRider is R3-B3: an actor
// with no mana pool at all is never a caster (weaponSpellFor's own isMage
// half stays false), so its attack is never replaced and the physical blow
// lands ordinarily — but weaponRiderSpellFor's own EXACT NEGATION of isMage
// means the same landed blow also fires the weapon's own rider
// (MAGIC-ITEM-007). The two numbers are asserted on separate lines so a slip
// in either arm is visible on its own: the blow alone would leave the victim
// at 60, the rider alone at 88, and the two together (this fixture) at 48.
//
// THE LINE THIS TEST WITNESSES is resolveBlow's own tail call,
// `w.weaponRiderApply(ai, ti, obs)` (combat.go, verified by hand): delete it
// and the victim is left at 60 rather than 48 — the blow lands but the
// weapon's own enchant never fires.
func TestANoManaPoolFighterStrikesAndTakesTheWeaponsOwnRider(t *testing.T) {
	rule := wpnRule(6, 6, 5) // spread 0 at level 30: base = 6*(30+30)/30 = 12
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0 // FR-2a: not a mage
	caster.DamageBase, caster.AlwaysHits = 40, true
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-40-12 {
		t.Errorf("the victim is at %d, want %d — a blow of 40 and the weapon's own rider of 12", hp, 100-40-12)
	}
	if p := spAt(t, w, 1).AttackPhase; p != AttackRelaxing {
		t.Errorf("the fighter's phase is %v, want AttackRelaxing after one blow", p)
	}
	if fx := spAt(t, w, 2).SpellFX; fx == 0 {
		t.Error("the victim carries no spell effect mark, want one — the rider is a release and marks its victim")
	}
}

func TestAMageNeverTakesTheFightersRiderThroughItsOwnAttack(t *testing.T) {
	rule := wpnRule(6, 6, 5) // spread 0 at level 30: base = 6*60/30 = 12
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.DamageBase, caster.AlwaysHits = 900, true // what a blow would remove instead
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if p := spAt(t, w, 1).AttackPhase; p != AttackRelaxing {
		t.Fatalf("the caster's phase is %v, want AttackRelaxing after one release", p)
	}
	if hp := spAt(t, w, 2).HP; hp != 100-12 {
		t.Errorf("the victim is at %d, want %d — the release alone, never the release plus a rider", hp, 100-12)
	}
}

// TestWeaponRiderSpellForRefusesAMage witnesses weaponRiderSpellFor's own
// isMage guard directly, called rather than reached through a tick: the test
// above already shows resolveBlow is unreachable for a mage through the
// ordinary cycle, which means that path alone can never independently
// exercise this line.
//
// THE LINE THIS TEST WITNESSES is weaponRiderSpellFor's own `isMage(e)` half
// of its guard (verified by hand): drop it and this test's ok comes back
// true for a mage instead of false.
func TestWeaponRiderSpellForRefusesAMage(t *testing.T) {
	e := wpnCaster(1, 0, 0, 1, 30, 1, 0) // isMage: MaxMana 100
	if _, ok := weaponRiderSpellFor([]SpellRule{wpnRule(5, 5, 5)}, e); ok {
		t.Error("weaponRiderSpellFor found a row for a mage, want none")
	}
}

// ---------------------------------------------------------------- AC-6, FR-2b

// TestAWeaponSpellNamingNoLoadedRowStrikesNormally is AC-6: an actor whose
// weapon's spell id is in no loaded spell table strikes as if the weapon
// carried no spell at all — FR-2b's own refusal, weaponSpell's call to
// findSpell.
//
// THE LINE THIS TEST WITNESSES is weaponSpell's own `return
// w.findSpell(uint32(e.WeaponSpell))`. Replace it with an unconditional
// `return rule, true` naming any row and the victim falls by 12 (a cast at
// a row this weapon does not actually name) instead of 40 (a blow).
func TestAWeaponSpellNamingNoLoadedRowStrikesNormally(t *testing.T) {
	table := []SpellRule{wpnRule(6, 6, 5)}    // id 1
	caster := wpnCaster(1, 0, 0, 9, 30, 1, 0) // names id 9, absent from the table
	caster.DamageBase, caster.AlwaysHits = 40, true
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, table, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-40 {
		t.Errorf("the victim is at %d, want %d — a weapon spell naming no row strikes normally", hp, 100-40)
	}
}

func TestClosedOnIsTheCastAdmissionForAnEligibleCasterAndInReachOtherwise(t *testing.T) {
	rule := wpnRule(6, 6, 4)
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0) // reach 1, range 4
	striker := wpnCaster(2, 0, 5, 0, 0, 1, 0) // WeaponSpell 0: an ordinary striker, reach 1
	victimA := spEnt(3, 3, 0)                 // distance 3 from (0,0)
	victimB := spEnt(4, 3, 5)                 // distance 3 from (0,5)
	w := spWorld(t, 1, []SpellRule{rule}, caster, striker, victimA, victimB)

	if ic, iv := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 3); !w.closedOn(ic, iv) {
		t.Error("a caster at distance 3 with a spell range of 4 is not closedOn, want true")
	}
	if is, iv := indexOfEntity(w.entities, 2), indexOfEntity(w.entities, 4); w.closedOn(is, iv) {
		t.Error("a striker at distance 3 with a reach of 1 is closedOn, want false")
	}
}

// TestACasterStopsAtSpellRangeNotMeleeReach is AC-7 end to end: a caster
// whose victim stands beyond the weapon's own melee reach but within the
// spell's range stops walking there and casts, rather than closing all the
// way to reach 1.
func TestACasterStopsAtSpellRangeNotMeleeReach(t *testing.T) {
	rule := wpnRule(6, 6, 4)
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.Speed = 40
	victim := spEnt(2, 10, 0)
	w := spWorld(t, 5, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	for i := 0; i < 150; i++ {
		Step(w, nil)
	}

	a := spAt(t, w, 1)
	dist := (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: 10, y: 0})
	if dist > 4 {
		t.Fatalf("the caster stands %d cell(s) from its victim after 150 advances, want at most 4 (the spell's own range)", dist)
	}
	if dist <= 1 {
		t.Errorf("the caster stands %d cell(s) from its victim, want more than 1 — it walked all the way to melee reach", dist)
	}
	if hp := spAt(t, w, 2).HP; hp == 100 {
		t.Errorf("the victim's health is still 100 — the caster never released from range")
	}
}

func TestRepeatedReleasesKeepTheWeaponAndCastAtTheSamePower(t *testing.T) {
	rule := wpnRule(5, 5, 5) // spread 0 at power 30: base = 5*60/30 = 10
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 50)
	caster.DamageBase, caster.AlwaysHits = 900, true
	victim := spEnt(2, 1, 0)
	victim.HP, victim.MaxHP = 100000, 100000
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	afterFirst := 100000 - spAt(t, w, 2).HP
	if afterFirst != 10 {
		t.Fatalf("the first release removed %d, want exactly 10", afterFirst)
	}
	for i := 0; i < 60; i++ {
		Step(w, nil)
	}
	total := 100000 - spAt(t, w, 2).HP
	if total != 20 {
		t.Errorf("after two releases the victim lost %d in all, want exactly 20 (10 + 10, the same power twice)", total)
	}
	a := spAt(t, w, 1)
	if a.WeaponSpell != 1 || a.WeaponSpellLevel != 30 {
		t.Errorf("the caster's weapon is spell %d level %d after two releases, want 1 and 30 — the weapon survives",
			a.WeaponSpell, a.WeaponSpellLevel)
	}
}

func TestAnEmptyManaPoolAndAnUnknownSpellStillCast(t *testing.T) {
	rule := wpnRule(5, 5, 5) // spread 0 at power 30: base = 5*60/30 = 10
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.Mana = 0        //
	caster.KnownSpells = 0 //
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-10 {
		t.Errorf("the victim is at %d, want %d — an empty pool and an unlearned spell must still cast", hp, 100-10)
	}
	if mana := spAt(t, w, 1).Mana; mana != 0 {
		t.Errorf("the caster's mana is %d, want 0 unchanged — a weapon-borne cast charges nothing", mana)
	}
}

// ---------------------------------------------------------------- AC-10, FR-3a

// TestFR3aBothDirectionsOfEligibilityChangingMidWindUp is AC-10: an actor
// that becomes eligible to cast while charging toward a blow never lands
// that blow, and one that stops being eligible while winding up a cast
// releases nothing — advanceAttack's own live re-ask (combat.go).
//
// THE LINE THIS TEST WITNESSES is `if wasCharging == casting { ...divert...
// }` in advanceAttack. Delete the divert (fall straight through to
// resolveBlow/releaseWeaponSpell by the wind-up's ORIGINAL kind alone) and
// each subtest's victim moves on the diverting tick instead of staying put.
func TestFR3aBothDirectionsOfEligibilityChangingMidWindUp(t *testing.T) {
	rule := wpnRule(5, 5, 5)

	t.Run("becomes eligible mid-charge: the blow never lands", func(t *testing.T) {
		caster := wpnCaster(1, 0, 0, 0, 0, 4, 0) // WeaponSpell 0: not yet eligible
		caster.DamageBase, caster.AlwaysHits = 40, true
		victim := spEnt(2, 1, 0)
		w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

		Step(w, []Command{cbOrder(1, 2)})
		if p := spAt(t, w, 1).AttackPhase; p != AttackCharging {
			t.Fatalf("fixture: phase is %v after the order, want AttackCharging", p)
		}
		Step(w, nil)
		Step(w, nil)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].WeaponSpell, w.entities[i].WeaponSpellLevel = 1, 30
		if cd := spAt(t, w, 1).AttackCountdown; cd != 1 {
			t.Fatalf("fixture: %d tick(s) owed before diverting, want exactly 1", cd)
		}
		before := spAt(t, w, 2).HP
		beforeRNG := w.rng.state

		Step(w, nil) // the countdown reaches zero here: the divert, not the blow

		if hp := spAt(t, w, 2).HP; hp != before {
			t.Errorf("the victim's health moved from %d to %d — the blow this charge was loaded for landed anyway", before, hp)
		}
		a := spAt(t, w, 1)
		if a.AttackPhase != AttackReady || a.AttackCountdown != 0 {
			t.Errorf("the diverted actor is %v/%d, want AttackReady/0", a.AttackPhase, a.AttackCountdown)
		}
		// FR-3b: the divert still draws the site's own jitter, once, and
		// discards it — it does not skip the draw just because the
		// outcome changed.
		if n := cbDraws(t, beforeRNG, w.rng.state); n != 1 {
			t.Errorf("the diverting advance drew %d time(s), want exactly 1 (the discarded jitter)", n)
		}
	})

	t.Run("stops being eligible mid-cast: the release never fires", func(t *testing.T) {
		caster := wpnCaster(1, 0, 0, 1, 30, 4, 0)
		victim := spEnt(2, 1, 0)
		w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

		Step(w, []Command{cbOrder(1, 2)})
		if p := spAt(t, w, 1).AttackPhase; p != AttackCasting {
			t.Fatalf("fixture: phase is %v after the order, want AttackCasting", p)
		}
		Step(w, nil)
		Step(w, nil)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].MaxMana = 0 // FR-2a no longer holds: not a mage
		if cd := spAt(t, w, 1).AttackCountdown; cd != 1 {
			t.Fatalf("fixture: %d tick(s) owed before diverting, want exactly 1", cd)
		}
		before := spAt(t, w, 2).HP
		beforeRNG := w.rng.state

		Step(w, nil)

		if hp := spAt(t, w, 2).HP; hp != before {
			t.Errorf("the victim's health moved from %d to %d — a release fired anyway", before, hp)
		}
		a := spAt(t, w, 1)
		if a.AttackPhase != AttackReady || a.AttackCountdown != 0 {
			t.Errorf("the diverted actor is %v/%d, want AttackReady/0", a.AttackPhase, a.AttackCountdown)
		}
		if n := cbDraws(t, beforeRNG, w.rng.state); n != 1 {
			t.Errorf("the diverting advance drew %d time(s), want exactly 1 (the discarded jitter)", n)
		}
	})
}

// TestFR10sFiveRefusalsLeaveTheWorldUnchangedButForTheAttackersOwnCycle is
// AC-11: each of releaseWeaponSpell's refusals changes nothing about the
// world except the attacker's own AttackPhase and AttackCountdown, on
// wpnSameExceptCycle's own comparison. Every case here loads a charge of 2
// so the release attempt falls on a SEPARATE advance from the order, and
// "before" and "after" bracket exactly that one advance.
//
// R3-B1 REPLACED TWO OF THE ORIGINAL FIVE. `!rule.Damaging` and
// `!rule.TargetsUnit` are gone from releaseWeaponSpell itself: a
// non-damaging row is now carried to weaponSpellApply, which refuses it only
// when NO arm applies at all (spellApplicable), and TargetsUnit no longer
// gates a release at all (spec.md; TestANonDamagingRowNowReleasesThroughThe
// OrdinaryApply and TestTargetsUnitDoesNotGateAWeaponRelease, below, are the
// positive side of that same change).
//
// "an inapplicable row" witnesses weaponSpellApply's `!spellApplicable`;
// "id 25" is refused over two overlapping gates (its own note); the last
// three witness the target-boundary, self-target and range refusals. A
// caster item's id 14 is not a refusal: releaseWeaponSpell routes it to
// applyPrismaticItem first (TestACasterItemPrismaticSprayFiresAtItsAimedTarget);
// weaponSpellApply's id-14 cut is reached only by the fighter's rider
// (TestWeaponSpellApplyRefusesPrismaticSprayForTheRider). The whole-world
// half of AC-11 — that nothing else moves either — is
// TestARefusedReleaseTouchesNothingButTheAttackersOwnCycle below, on one
// clean geometry rather than six: a downed victim decays on its own dwell
// every tick regardless of what any release does, and a caster still
// approaching moves on its own too, so a strict whole-world diff on THESE
// fixtures would fail for reasons the guard being witnessed has nothing to
// do with.
func TestFR10sFiveRefusalsLeaveTheVictimUntouched(t *testing.T) {
	run := func(t *testing.T, table []SpellRule, ents []Entity, order EntityID, wantHP int32) {
		t.Helper()
		w := spWorld(t, 1, table, ents...)
		Step(w, []Command{cbOrder(1, order)})
		if p := spAt(t, w, 1).AttackPhase; p != AttackCasting {
			t.Fatalf("fixture: phase is %v after the order, want AttackCasting", p)
		}
		Step(w, nil) // the release attempt: refused
		if hp := spAt(t, w, order).HP; hp != wantHP {
			t.Errorf("the victim is at %d, want %d unchanged — the release was meant to be refused", hp, wantHP)
		}
	}

	t.Run("an inapplicable row (not damaging, no effect kind)", func(t *testing.T) {
		r := wpnRule(5, 5, 5)
		r.Damaging = false
		caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
		victim := spEnt(2, 1, 0)
		run(t, []SpellRule{r}, []Entity{caster, victim}, 2, 100)
	})

	// "the row is id 25, Control Spirit" pins that a weapon-borne Control
	// Spirit release against a LIVING victim is refused, but a living victim
	// is refused twice over here — weaponSpellApply's own id-25 cut, and
	// pointEffectRefusal's own "target is not a bones corpse" gate — so this
	// subtest alone cannot tell which one fired. The id-25 cut in isolation,
	// over the one fixture that WOULD otherwise raise a ghost, is
	// TestWeaponSpellApplyRefusesControlSpiritEvenOverAValidCorpse, below.
	t.Run("the row is id 25, Control Spirit", func(t *testing.T) {
		r := wpnRule(5, 5, 5)
		r.ID, r.Damaging, r.TargetsUnit = 25, false, false
		caster := wpnCaster(1, 0, 0, 25, 30, 2, 0)
		victim := spEnt(2, 1, 0)
		run(t, []SpellRule{r}, []Entity{caster, victim}, 2, 100)
	})

	t.Run("the victim crosses the -10 target boundary", func(t *testing.T) {
		caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
		victim := spEnt(2, 1, 0)
		victim.HP = -9
		w := spWorld(t, 1, []SpellRule{wpnRule(5, 5, 5)}, caster, victim)
		Step(w, []Command{cbOrder(1, 2)})
		vi := indexOfEntity(w.entities, 2)
		w.entities[vi].HP = -10
		Step(w, nil)
		if e := spAt(t, w, 1); e.HasAttackTarget {
			t.Fatalf("the stale weapon release kept target %d", e.AttackTarget)
		}
		if hp := spAt(t, w, 2).HP; hp != -10 {
			t.Errorf("the refused release moved target health to %d", hp)
		}
	})

	t.Run("the victim is the caster itself", func(t *testing.T) {
		caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
		w := spWorld(t, 1, []SpellRule{wpnRule(5, 5, 5)}, caster)
		i := indexOfEntity(w.entities, 1)
		// orderAttack itself refuses a victim naming its own attacker
		// (step.go's KindAttack arm), so the self-order is written
		// directly rather than commanded — the same technique
		// TestFR6BothArmsOfTheDiagonal (combat_test.go) already uses to
		// reach a shape the command layer will not construct.
		w.entities[i].AttackTarget, w.entities[i].HasAttackTarget = 1, true
		w.entities[i].AttackPhase, w.entities[i].AttackCountdown = AttackReady, 0

		Step(w, nil) // AttackReady is free: this call also takes the first tick off the charge
		if p := spAt(t, w, 1).AttackPhase; p != AttackCasting {
			t.Fatalf("fixture: phase is %v after loading, want AttackCasting", p)
		}
		Step(w, nil) // the release attempt: refused, the victim is the caster
		if hp := spAt(t, w, 1).HP; hp != 100 {
			t.Errorf("the caster is at %d, want 100 unchanged — a release on oneself is refused", hp)
		}
	})

	t.Run("the victim is one cell past MaxRange", func(t *testing.T) {
		caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
		victim := spEnt(2, 10, 0)
		w := spWorld(t, 1, []SpellRule{wpnRule(5, 5, 1)}, caster, victim)
		w.entities[0].AttackTarget, w.entities[0].HasAttackTarget = 2, true
		w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCasting, 1
		Step(w, nil) // release-time range revalidation
		if hp := spAt(t, w, 2).HP; hp != 100 {
			t.Errorf("the victim is at %d, want 100 unchanged — release moved outside MaxRange", hp)
		}
	})
}

// ---------------------------------------------------------------- R3-B1, positive cases

// TestANonDamagingRowNowReleasesThroughTheOrdinaryApply is R3-B1's own
// positive case, at the shape of the shipped carrier (mission 130 entity 55,
// M130_Veglud, an Elven Magic Wood Staff carrying Stone Curse): a weapon
// spell that is not Damaging is no longer refused outright. It reaches
// weaponSpellApply's point branch and attaches through ordinaryEffect
// exactly as a book cast of the same row would.
//
// THE LINE THIS TEST WITNESSES is releaseWeaponSpell's own former first
// refusal, `!rule.Damaging` (verified by hand, against the pre-fix source):
// restore it ahead of weaponSpellApply's own call and this release is
// refused before anything attaches — the victim's Absorption stays at 0
// rather than rising to 5, and this test goes red.
func TestANonDamagingRowNowReleasesThroughTheOrdinaryApply(t *testing.T) {
	rule := SpellRule{ID: 20, ManaCost: 40, MaxRange: 5, TargetsUnit: true,
		EffectKind: EffectAbsorption, EffectMode: EffectDuration, EffectMagnitude: 5, SpellDuration: 10}
	caster := wpnCaster(1, 0, 0, 20, 30, 1, 0)
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if a := spAt(t, w, 2).Absorption; a != 5 {
		t.Errorf("the victim's Absorption is %d, want 5 — a non-damaging weapon spell must still release", a)
	}
	if fx := spAt(t, w, 2).SpellFX; fx == 0 {
		t.Error("the victim carries no spell effect mark, want one")
	}
	if mana := spAt(t, w, 1).Mana; mana != 100 {
		t.Errorf("the caster's mana is %d, want 100 unchanged — a weapon-borne cast charges nothing", mana)
	}
}

// TestTargetsUnitDoesNotGateAWeaponRelease is R3-B1's other positive case:
// TargetsUnit is a different column from Area (SpellRule.Area's own doc,
// above) and gates a COMMANDED cast's choice of victim, not an APPLIED
// release. A weapon-borne row with TargetsUnit false still damages its
// victim.
//
// THE LINE THIS TEST WITNESSES is releaseWeaponSpell's own former second
// refusal, `!rule.TargetsUnit` (verified by hand, against the pre-fix
// source): restore it and this release is refused — the victim's health
// stays at 100 rather than falling by 10, and this test goes red.
func TestTargetsUnitDoesNotGateAWeaponRelease(t *testing.T) {
	rule := wpnRule(5, 5, 5) // spread 0 at level 30: base = 5*60/30 = 10
	rule.TargetsUnit = false
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100-10 {
		t.Errorf("the victim is at %d, want %d — TargetsUnit does not gate an applied release", hp, 100-10)
	}
}

// ---------------------------------------------------------------- R3-B2

// Prismatic Spray has an admitted caster-item fan in releaseWeaponSpell. Its
// item projection therefore reports the powered damage pair, the raw release
// range, and the owner-defined ray count rather than falling back to physical
// weapon statistics.
func TestWeaponSpellCharacteristicsForReportsPrismaticDamageRaysAndRange(t *testing.T) {
	rule := SpellRule{ID: 14, MaxRange: 7, DamageMin: 5, DamageMax: 15,
		TargetsUnit: true, Damaging: true}
	e := Entity{MaxMana: 1, WeaponSpell: 14, WeaponSpellLevel: 30}

	got, ok := WeaponSpellCharacteristicsFor(Rules{}, e, []SpellRule{rule})
	if !ok {
		t.Fatal("WeaponSpellCharacteristicsFor refused Prismatic Spray")
	}
	if got.SpellID != 14 || !got.HasDamage || got.DamageMin != 10 || got.DamageMax != 30 ||
		got.MaxRange != 7 || got.RayCount != 3 || got.HasDuration {
		t.Fatalf("Prismatic characteristics = %+v", got)
	}
	base, spread, damageOK := WeaponSpellDamageFor(Rules{}, e, []SpellRule{rule})
	if !damageOK || base != 10 || spread != 20 {
		t.Fatalf("WeaponSpellDamageFor = %d + %d, %v; want 10 + 20, true", base, spread, damageOK)
	}
}

func TestWeaponSpellCharacteristicsForReportsStoneCurseDurationInTicks(t *testing.T) {
	rule := SpellRule{ID: 20, MaxRange: 5, TargetsUnit: true, SpellDuration: 10,
		EffectKind: EffectAbsorption, EffectMode: EffectDuration}
	e := Entity{MaxMana: 1, WeaponSpell: 20, WeaponSpellLevel: 1}

	got, ok := WeaponSpellCharacteristicsFor(Rules{}, e, []SpellRule{rule})
	if !ok {
		t.Fatal("WeaponSpellCharacteristicsFor refused Stone Curse")
	}
	if got.SpellID != 20 || got.HasDamage || !got.HasDuration || got.DurationTicks != 163 ||
		got.MaxRange != 5 || got.RayCount != 0 {
		t.Fatalf("Stone Curse characteristics = %+v", got)
	}
}

// ---------------------------------------------------------------- R3-B1, ownership cut

func TestACasterItemPrismaticSprayFiresAtItsAimedTarget(t *testing.T) {
	r := wpnRule(5, 5, 5)
	r.ID = 14
	w := spWorld(t, 1, []SpellRule{r}, wpnCaster(1, 0, 0, 14, 30, 2, 0), spEnt(2, 1, 0))
	Step(w, []Command{cbOrder(1, 2)})
	Step(w, nil)
	if hp := spAt(t, w, 2).HP; hp != 90 {
		t.Errorf("the aimed non-hostile primary is at %d, want 90", hp)
	}
}

func TestWeaponSpellApplyRefusesPrismaticSprayForTheRider(t *testing.T) {
	r := wpnRule(5, 5, 5)
	r.ID = 14
	w := spWorld(t, 1, []SpellRule{r}, wpnCaster(1, 0, 0, 14, 30, 2, 0), spEnt(2, 1, 0))
	if w.weaponSpellApply(0, 1, r, 30, nil) || w.entities[1].HP != 100 || len(w.deliveries) != 0 {
		t.Error("weaponSpellApply applied id 14")
	}
}

// ---------------------------------------------------------------- R3-B3

// TestTheFightersRiderStopsWhenThePhysicalBlowCrossesMinusTen is R3-B3 at
// the shape of the shipped carrier (Boulder Thrower, castSpell=Fire_Ball,
// missions 90/111/140): a fighter's weapon carrying an AREA row lands it at
// the struck target's own cell (spec.md), and the rider used to fire after
// any killing blow. The owner-directed boundary now ends the whole ordinary
// action at -10, before the area rider can affect the bystander (`DIV-442`).
func TestTheFightersRiderStopsWhenThePhysicalBlowCrossesMinusTen(t *testing.T) {
	rule := SpellRule{ID: 2, Area: true, Radius: 1, School: 1, MaxRange: 5,
		DamageMin: 10, DamageMax: 10, Damaging: true}
	caster := wpnCaster(1, 0, 0, 2, 40, 1, 0) // level 40: base = 10*(40+30)/30 = 23
	caster.MaxMana, caster.Mana = 0, 0        // FR-2a: not a mage — the fighter's rider, not the caster's
	caster.DamageBase, caster.AlwaysHits = 900, true
	victim := spEnt(2, 1, 0)
	victim.HP = 5 // the physical blow alone fells it
	bystander := spEnt(3, 1, 1)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim, bystander)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp >= 0 {
		t.Fatalf("fixture: the primary target is at %d, want the physical blow alone to fell it (< 0)", hp)
	}
	if hp := spAt(t, w, 3).HP; hp != 100 {
		t.Errorf("the rider crossed the -10 boundary and damaged the bystander to %d", hp)
	}
}

// TestAMageNeverTakesTheFightersRiderThroughItsOwnAttack (above) already
// pins the mage side of R3-B3's own exclusion. TestTheFightersRiderTrains
// Nothing and TestAnAbsorbedBlowNeverFiresTheRider pin the fighter side: the
// rider fires only after a landed, damaging blow, and never trains.

func TestTheFightersRiderTrainsTheCurrentWeaponSkill(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	build := func(weaponSpell uint16) *World {
		caster := wpnCaster(1, 0, 0, weaponSpell, 30, 1, 0)
		caster.MaxMana, caster.Mana = 0, 0 // fighter, not caster: awardSkill's own non-mage
		caster.XPSlot = 4                  // path credits XPSlot, not the row's named school
		caster.Owner, caster.GainsXP, caster.Mind, caster.TypeID = 2, true, 60, HumanTypeID
		caster.DamageBase, caster.AlwaysHits = 10, true // survivable: the rider must still be live
		caster.Skill[4], caster.SkillXP[4] = 10, skillXPFor(10)
		victim := spEnt(2, 1, 0)
		victim.Owner = 3
		return spWorld(t, 1, []SpellRule{rule}, caster, victim)
	}

	withRider := build(1) // the weapon carries the row: the rider fires
	noSpell := build(0)   // the weapon carries nothing: only the ordinary blow

	Step(withRider, []Command{cbOrder(1, 2)})
	Step(noSpell, []Command{cbOrder(1, 2)})

	gotLvl, wantLvl := spAt(t, withRider, 1).Skill[4], spAt(t, noSpell, 1).Skill[4]
	gotXP, wantXP := spAt(t, withRider, 1).SkillXP[4], spAt(t, noSpell, 1).SkillXP[4]
	if gotLvl != wantLvl || gotXP != wantXP+2 {
		t.Errorf("slot 4 is level %d, XP %d with the rider firing; level %d, XP %d without it — "+
			"the rider must add its separate damage-event award", gotLvl, gotXP, wantLvl, wantXP)
	}
}

// TestAnAbsorbedBlowNeverFiresTheRider is R3-B3's own gate on WHEN the
// rider may run: "after the damage lands" (MAGIC-ITEM-007), which resolveBlow
// reads as past its own `dmg <= 0` return. An ordinary guaranteed hit that is
// fully absorbed must not fire the rider. AlwaysHits is deliberately false:
// HERO-DAMAGE-022 says that arm skips flat absorption.
//
// THE LINE THIS TEST WITNESSES is resolveBlow's own placement of
// `w.weaponRiderApply(ai, ti, obs)` AFTER the `if dmg <= 0 { return }` guard
// (verified by hand): move the call ahead of that guard and the victim's
// Absorption-cancelled blow still releases the weapon's own spell.
func TestAnAbsorbedBlowNeverFiresTheRider(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0 // fighter, not caster
	caster.DamageBase, caster.DamageSpread = 5, 0
	caster.AlwaysHits, caster.ToHit = false, 2147483647
	victim := spEnt(2, 1, 0)
	victim.Absorption = 100 // the blow lands but removes nothing
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100 {
		t.Errorf("the victim is at %d, want 100 unchanged — an absorbed-to-nothing blow must not fire the rider", hp)
	}
	if fx := spAt(t, w, 2).SpellFX; fx != 0 {
		t.Error("the victim carries a spell effect mark despite a fully absorbed blow")
	}
}

// TestAResistanceCancelledBlowNeverFiresTheRider is the same interaction at
// the new boundary: the hit and relation flip happen, but a Blade resistance
// of 100 leaves no positive physical result, so neither health nor the
// downstream weapon spell may move.
func TestAResistanceCancelledBlowNeverFiresTheRider(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0
	caster.DamageBase, caster.DamageSpread = 5, 0
	caster.AlwaysHits, caster.XPSlot = true, 1
	victim := spEnt(2, 1, 0)
	victim.Resistance[0] = 100
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if hp := spAt(t, w, 2).HP; hp != 100 {
		t.Errorf("the victim is at %d, want 100 unchanged", hp)
	}
	if fx := spAt(t, w, 2).SpellFX; fx != 0 {
		t.Error("the victim carries a spell effect mark despite a resistance-cancelled blow")
	}
}

// TestARefusedReleaseTouchesNothingButTheAttackersOwnCycle is AC-11's own
// whole-world claim, on one clean geometry: the caster starts already
// closed on its victim (no approach needed) and the victim is alive (no
// ambient decay), so the ONLY thing a refused release may move at all is
// the attacker's own AttackPhase and AttackCountdown — wpnSameExceptCycle's
// own comparison, over every field of every entity.
func TestARefusedReleaseTouchesNothingButTheAttackersOwnCycle(t *testing.T) {
	r := wpnRule(5, 5, 5)
	r.Damaging = false //
	caster := wpnCaster(1, 0, 0, 1, 30, 2, 0)
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{r}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	if p := spAt(t, w, 1).AttackPhase; p != AttackCasting {
		t.Fatalf("fixture: phase is %v after the order, want AttackCasting", p)
	}
	before := w.Entities()
	Step(w, nil) // the release attempt: refused
	after := w.Entities()
	wpnSameExceptCycle(t, 1, before, after)
}

// ---------------------------------------------------------------- AC-11a, FR-3b

// TestFR3bTheSitesOwnDrawIsIndependentOfEligibility is AC-11a: two worlds
// identical but for one actor's eligibility to cast, each refused at its
// own site (out of melee reach; beyond the spell's own range) on the same
// advance, draw the same number of random values — advanceAttack's own
// jitter draw does not depend on whether the actor is a caster.
//
// The geometry is TestAnOutOfReachOrderWaitsBeforeItsCycle's own
// (combat_test.go): distance 4 against an admission of 1, so one approach
// step this same tick leaves 3, still out of admission and still refused
// before either arm's own draws.
func TestFR3bTheSitesOwnDrawIsIndependentOfEligibility(t *testing.T) {
	rule := wpnRule(5, 5, 1)

	fighter := wpnCaster(1, 0, 0, 0, 0, 1, 9) // not eligible
	victimA := spEnt(2, 4, 0)
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 9) // eligible
	victimB := spEnt(2, 4, 0)

	wA := spWorld(t, 88, []SpellRule{rule}, fighter, victimA)
	wB := spWorld(t, 88, []SpellRule{rule}, caster, victimB)

	beforeA, beforeB := wA.rng.state, wB.rng.state
	Step(wA, []Command{cbOrder(1, 2)})
	Step(wB, []Command{cbOrder(1, 2)})

	if hp := spAt(t, wA, 2).HP; hp != 100 {
		t.Fatalf("fixture: the non-caster's blow connected, want it refused by reach")
	}
	if hp := spAt(t, wB, 2).HP; hp != 100 {
		t.Fatalf("fixture: the caster's release connected, want it refused by range")
	}
	nA := cbDraws(t, beforeA, wA.rng.state)
	nB := cbDraws(t, beforeB, wB.rng.state)
	if nA != nB {
		t.Errorf("the non-caster drew %d time(s) and the caster %d, want them equal", nA, nB)
	}
}

// TestAReleasedCastReproducesTheCommandedCastsArithmetic is AC-12: a
// weapon-borne release's damage is the same arithmetic a commanded cast
// reaches at the same power and the same row — applySpellDamage
// (spell.go), the one function both paths call. A spread of zero removes
// the generator from the comparison, so the two amounts are asserted
// EQUAL rather than merely both in range.
//
// THE LINE THIS TEST WITNESSES is releaseWeaponSpell's own tail call,
// `w.applySpellDamage(ti, rule, a.WeaponSpellLevel)` (verified by hand): a
// power off by 1 still truncates to the same 14 at this row's own
// DamageMin (7*(31+30)/30 floors to 14 exactly as 7*60/30 does), so this
// test does not catch every possible slip in that argument — but a wrong
// power off by more than a couple of points, or a different arithmetic
// entirely, moves the weapon path's own amount away from the commanded
// path's and this test catches that.
func TestAReleasedCastReproducesTheCommandedCastsArithmetic(t *testing.T) {
	rule := wpnRule(7, 7, 5) // spread 0 at power 30: base = 7*60/30 = 14

	weaponCaster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	weaponVictim := spEnt(2, 1, 0)
	wWeapon := spWorld(t, 1, []SpellRule{rule}, weaponCaster, weaponVictim)
	Step(wWeapon, []Command{cbOrder(1, 2)})

	commandedCaster := spMage(1, 0, 0, 60, 50, 20, 1<<1) // Mind 60: spellPower(60) = 30, the same power
	commandedVictim := spEnt(2, 1, 0)
	wCommanded := spWorld(t, 2, []SpellRule{rule}, commandedCaster, commandedVictim)
	spRunCast(wCommanded, spCast(1, 2, 1))

	weaponAmount := 100 - spAt(t, wWeapon, 2).HP
	commandedAmount := 100 - spAt(t, wCommanded, 2).HP
	if weaponAmount != 14 || commandedAmount != 14 {
		t.Fatalf("fixture: the weapon path removed %d and the commanded path removed %d, want 14 both",
			weaponAmount, commandedAmount)
	}
	if weaponAmount != commandedAmount {
		t.Errorf("the weapon-borne release removed %d and the commanded cast removed %d, want them equal",
			weaponAmount, commandedAmount)
	}
}

// The sheet-facing interval is not a parallel formula: it must name exactly
// what the weapon-spell release removes. A deliberately tiny physical staff
// roll makes an accidental fallback to DamageBase visible, while a zero spell
// spread makes the live release an exact comparison rather than a range test.
func TestWeaponSpellDamageReportsTheLiveReleasesDamageInterval(t *testing.T) {
	rule := wpnRule(15, 15, 5) // level 10: 15*(10+30)/30 = 20
	caster := wpnCaster(1, 0, 0, 1, 10, 1, 0)
	caster.DamageBase, caster.DamageSpread = 2, 0
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	base, spread, ok := w.WeaponSpellDamage(1)
	if !ok || base != 20 || spread != 0 {
		t.Fatalf("WeaponSpellDamage = %d + %d, %v; want spell release interval 20 + 0, true", base, spread, ok)
	}
	if base == int64(caster.DamageBase) {
		t.Fatal("WeaponSpellDamage fell back to the staff's unused physical 2-2 roll")
	}

	Step(w, []Command{cbOrder(1, 2)})
	if removed := int64(100 - spAt(t, w, 2).HP); removed != base {
		t.Fatalf("live weapon-spell release removed %d, want the displayed interval's exact %d", removed, base)
	}
}

// TestAWeaponBorneCastRoundTripsThroughTheByteForm is AC-14's first half:
// an actor mid-cast — AttackCasting, a weapon carrying a spell and a level
// — marshals, unmarshals and hashes identically.
//
// THE LINES THIS TEST WITNESSES are binary.go's own encode and decode of
// WeaponSpell and WeaponSpellLevel at the record's tail: drop either
// assignment on the write or the read side and the decoded caster comes
// back with a zero spell instead of 1 at level 30.
func TestAWeaponBorneCastRoundTripsThroughTheByteForm(t *testing.T) {
	rule := wpnRule(5, 5, 5)
	caster := wpnCaster(1, 0, 0, 1, 30, 6, 2)
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	Step(w, nil)
	if e := spAt(t, w, 1); e.AttackPhase != AttackCasting || e.AttackCountdown == 0 {
		t.Fatalf("fixture: the caster is meant to be caught mid-cast, phase %v countdown %d",
			e.AttackPhase, e.AttackCountdown)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got, want := back.Hash(), w.Hash(); got != want {
		t.Errorf("the decoded world hashes %#016x, the original %#016x", got, want)
	}
	got := back.entities[indexOfEntity(back.entities, 1)]
	if got.AttackPhase != AttackCasting || got.WeaponSpell != 1 || got.WeaponSpellLevel != 30 {
		t.Errorf("the decoded caster is %+v, want AttackCasting with weapon spell 1 at level 30", got)
	}
}

// TestACastingCycleNoTickCanLeaveIsRefused is AC-14's second half: a
// casting cycle whose countdown is at or above its own charge — a state no
// tick this package runs can produce — is refused rather than accepted.
// attackFault (combat.go) is the one function the constructor and the
// decoder both ask, so this exercises the same guard the decoder would.
//
// THE LINE THIS TEST WITNESSES is attackFault's own merged case: `(e
// .AttackPhase == AttackCharging || e.AttackPhase == AttackCasting) &&
// int64(e.AttackCountdown) >= chargeTicks(e)`. Narrow it back to
// AttackCharging alone — the pre-0139 line — and this fixture's
// AttackCasting entity is accepted instead of refused.
func TestACastingCycleNoTickCanLeaveIsRefused(t *testing.T) {
	caster := wpnCaster(1, 0, 0, 1, 30, 6, 2)
	caster.AttackTarget, caster.HasAttackTarget = 2, true
	caster.AttackPhase, caster.AttackCountdown = AttackCasting, 6 // == its own charge, which no tick can leave
	victim := spEnt(2, 1, 0)

	_, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
		[]Entity{caster, victim}, nil, []SpellRule{wpnRule(5, 5, 5)})
	if err == nil {
		t.Fatal("NewSpelledWorld accepted an AttackCasting countdown at its own charge")
	}
}

// TestAWeaponBorneReleaseTrainsFromDamageNotMana pins both halves of the
// item-cast rule: no immediate half-mana award is paid, but the accepted damage
// event credits the spell's actual school.
func TestAWeaponBorneReleaseTrainsFromDamageNotMana(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	rule.School = 2
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.Owner, caster.GainsXP, caster.Mind, caster.TypeID = 2, true, 60, HumanTypeID
	caster.XPSlot = 4 // the slot a BLOW would credit, which a release must not
	caster.Skill[2], caster.SkillXP[2] = 10, skillXPFor(10)
	victim := spEnt(2, 1, 0)
	victim.Owner = 3
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	got := spAt(t, w, 1)
	for i, lvl := range got.Skill {
		want := int32(0)
		if i == 2 {
			want = 11
		}
		if lvl != want {
			t.Errorf("slot %d moved to %d, want %d", i, lvl, want)
		}
	}
	if got.SkillXP[2] != skillXPFor(10)+2 {
		t.Errorf("school 2 banked %d, want one damage event (%d); a half-mana award would be larger",
			got.SkillXP[2], skillXPFor(10)+2)
	}
}

func TestARefusedReleaseTrainsNothing(t *testing.T) {
	rule := wpnRule(6, 6, 5)
	rule.School, rule.Damaging = 2, false
	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.Owner, caster.GainsXP, caster.Mind, caster.TypeID = 2, true, 60, HumanTypeID
	caster.Skill[2], caster.SkillXP[2] = 10, skillXPFor(10)
	victim := spEnt(2, 1, 0)
	victim.Owner = 3
	w := spWorld(t, 1, []SpellRule{rule}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})

	if got := spAt(t, w, 1); got.Skill[2] != 10 {
		t.Errorf("school 2 is at level %d after a REFUSED release, want the untouched 10", got.Skill[2])
	}
}
