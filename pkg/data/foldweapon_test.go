package data_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
)

// baseCombat is a bearer's own combat block before any weapon folds onto it —
// every field this file cares about NONZERO, and none of them equal to what
// either fixture weapon below states, so a fold that leaked the wrong number
// or failed to move the right one would show.
func baseCombat() data.Combat {
	return data.Combat{
		DamageBase: 5, DamageSpread: 3, ToHit: 7, Defence: 2,
		AttackChargeTime: 9, AttackRelaxTime: 4, Reach: 1,
	}
}

// rangedFixture is an elemental ranged weapon carrying nonzero damage,
// weapon to-hit and defence. Its physical pair, weapon to-hit and defence
// must stay out of the melee fold; its byte-width pair instead joins the
// third damage component and its accuracy comes from the bearer's General.
func rangedFixture() data.Weapon {
	return data.Weapon{
		AttackType:   11,
		DamageBase:   100,
		DamageSpread: 60,
		ToHit:        40,
		Defence:      20,
		ChargeTime:   3,
		RelaxTime:    3,
		Range:        6,
	}
}

// meleeFixture is the same numbers on a row below meleeAttackTypes, so the
// two fixtures differ only in which arm of the fold they take.
func meleeFixture() data.Weapon {
	f := rangedFixture()
	f.AttackType = 1
	return f
}

// TestARangedWeaponFoldsGeneralAndFireOutsideThePhysicalPair is DIV-363's
// ranged half. Attack type 11 leaves the physical pair and Defence alone,
// ignores the weapon's own ToHit, adds General to live accuracy, and adds its
// byte pair to the existing third component before assigning Fire.
func TestARangedWeaponFoldsGeneralAndFireOutsideThePhysicalPair(t *testing.T) {
	base := baseCombat()
	base.SecondaryDamage = data.SecondaryDamage{Base: 200, Spread: 220, Selector: 4}
	w := rangedFixture()

	got := data.FoldWeapon(base, &w, 17)

	if got.DamageBase != base.DamageBase || got.DamageSpread != base.DamageSpread ||
		got.Defence != base.Defence {
		t.Fatalf("a ranged fold moved the physical pair/defence: got %+v, want the bearer's own %+v",
			got, base)
	}
	if got.ToHit != base.ToHit+17 {
		t.Errorf("ToHit = %d, want bearer %d + General 17 (weapon ToHit %d is not used)",
			got.ToHit, base.ToHit, w.ToHit)
	}
	if want := (data.SecondaryDamage{Base: 44, Spread: 24, Selector: 0}); got.SecondaryDamage != want {
		t.Errorf("SecondaryDamage = %+v, want wrapped byte adds and Fire selector %+v", got.SecondaryDamage, want)
	}
	if got.Reach != w.Range {
		t.Errorf("Reach = %d, want the weapon's own range %d", got.Reach, w.Range)
	}
	if got.AttackChargeTime != w.ChargeTime || got.AttackRelaxTime != w.RelaxTime {
		t.Errorf("cadence = %d/%d, want the weapon's own %d/%d",
			got.AttackChargeTime, got.AttackRelaxTime, w.ChargeTime, w.RelaxTime)
	}
	if got.SkillSlot != data.SkillGeneral {
		t.Errorf("SkillSlot = %d, want General 0 for a ranged arm", got.SkillSlot)
	}
}

// TestAMeleeWeaponFoldsAllFourByExactlyItsOwnAmount is AC-2's melee half: the
// same numbers, on a row below meleeAttackTypes, move every one of the four —
// each by exactly the weapon's own amount, addition and not assignment — on
// top of reach and cadence, which move exactly as they do on the ranged arm.
func TestAMeleeWeaponFoldsAllFourByExactlyItsOwnAmount(t *testing.T) {
	base := baseCombat()
	w := meleeFixture()

	got := data.FoldWeapon(base, &w, 999)

	if got.DamageBase != base.DamageBase+w.DamageBase {
		t.Errorf("DamageBase = %d, want %d+%d", got.DamageBase, base.DamageBase, w.DamageBase)
	}
	if got.DamageSpread != base.DamageSpread+w.DamageSpread {
		t.Errorf("DamageSpread = %d, want %d+%d", got.DamageSpread, base.DamageSpread, w.DamageSpread)
	}
	if got.ToHit != base.ToHit+w.ToHit {
		t.Errorf("ToHit = %d, want %d+%d", got.ToHit, base.ToHit, w.ToHit)
	}
	if got.Defence != base.Defence+w.Defence {
		t.Errorf("Defence = %d, want %d+%d", got.Defence, base.Defence, w.Defence)
	}
	if got.Reach != w.Range {
		t.Errorf("Reach = %d, want the weapon's own range %d", got.Reach, w.Range)
	}
	if got.AttackChargeTime != w.ChargeTime || got.AttackRelaxTime != w.RelaxTime {
		t.Errorf("cadence = %d/%d, want the weapon's own %d/%d",
			got.AttackChargeTime, got.AttackRelaxTime, w.ChargeTime, w.RelaxTime)
	}
	if got.SkillSlot != data.SkillBlade {
		t.Errorf("SkillSlot = %d, want the melee weapon kind %d", got.SkillSlot, data.SkillBlade)
	}
}

// TestRangedAttackTypesChooseOnlyTheirPublishedThirdComponent verifies the
// exact Weapon::Equip switch: 11 selects Fire, 12 selects Earth, while the
// neighbouring generic ranged type 10 still receives General accuracy but
// contributes no weapon damage triple.
func TestRangedAttackTypesChooseOnlyTheirPublishedThirdComponent(t *testing.T) {
	for _, tc := range []struct {
		attackType int32
		want       data.SecondaryDamage
	}{
		{10, data.SecondaryDamage{Base: 1, Spread: 2, Selector: 4}},
		{11, data.SecondaryDamage{Base: 101, Spread: 62, Selector: 0}},
		{12, data.SecondaryDamage{Base: 101, Spread: 62, Selector: 3}},
	} {
		w := rangedFixture()
		w.AttackType = tc.attackType
		base := baseCombat()
		base.SecondaryDamage = data.SecondaryDamage{Base: 1, Spread: 2, Selector: 4}

		got := data.FoldWeapon(base, &w, 9)
		if got.ToHit != base.ToHit+9 {
			t.Errorf("AttackType %d: ToHit = %d, want %d", tc.attackType, got.ToHit, base.ToHit+9)
		}
		if got.SecondaryDamage != tc.want {
			t.Errorf("AttackType %d: SecondaryDamage = %+v, want %+v", tc.attackType, got.SecondaryDamage, tc.want)
		}
	}
}

// TestFoldWeaponNormalisesTheCompleteAttackTypeBoundary is the producer-side
// indexing guard. Only supported melee kinds 1..5 may reach Entity.XPSlot;
// bare/general, negative, unsupported melee values and all ranged arms clear
// it to zero before uint8 narrowing.
func TestFoldWeaponNormalisesTheCompleteAttackTypeBoundary(t *testing.T) {
	for _, tc := range []struct {
		attackType int32
		want       int32
	}{
		{-1, 0},
		{0, 0},
		{1, 1},
		{5, 5},
		{6, 0},
		{9, 0},
		{10, 0},
		{11, 0},
		{12, 0},
	} {
		w := meleeFixture()
		w.AttackType = tc.attackType
		got := data.FoldWeapon(data.Combat{SkillSlot: data.SkillPike}, &w, 0)
		if got.SkillSlot != tc.want {
			t.Errorf("AttackType %d: SkillSlot = %d, want %d", tc.attackType, got.SkillSlot, tc.want)
		}
	}
}

// TestANilWeaponLeavesCombatUnchanged is FoldWeapon's own stated no-op: a
// bare bearer folds nothing, so w == nil returns c exactly as it arrived.
func TestANilWeaponLeavesCombatUnchanged(t *testing.T) {
	base := baseCombat()

	if got := data.FoldWeapon(base, nil, 99); got != base {
		t.Fatalf("FoldWeapon(c, nil) = %+v, want c unchanged %+v", got, base)
	}
}

// TestFoldWeaponAssignsTheSpellPairOnBothArms is 0139 plan D-3: the spell a
// weapon carries is ASSIGNED, not summed, and it reaches the bearer on the
// ranged arm exactly as it does on the melee one — above the `if
// !w.Ranged()` branch, alongside Reach and the cadence pair, and unlike the
// four additive terms below it that a ranged weapon skips.
func TestFoldWeaponAssignsTheSpellPairOnBothArms(t *testing.T) {
	for _, tc := range []struct {
		label string
		w     func() data.Weapon
	}{
		{"ranged", rangedFixture},
		{"melee", meleeFixture},
	} {
		t.Run(tc.label, func(t *testing.T) {
			base := baseCombat()
			w := tc.w()
			w.SpellName, w.SpellPower = "Fire_Arrow", 10

			got := data.FoldWeapon(base, &w, 0)
			if got.SpellName != "Fire_Arrow" || got.SpellPower != 10 {
				t.Fatalf("spell pair = (%q, %d), want (\"Fire_Arrow\", 10)", got.SpellName, got.SpellPower)
			}
		})
	}
}

// TestFoldWeaponClearsAPreviousSpellWhenTheNewWeaponCarriesNone is plan
// D-3's "two weapons cannot sum their spells": a bearer already carrying a
// spell from an earlier fold loses it the moment a spell-less weapon folds
// on top, because the pair is ASSIGNED, never added — unlike, say, the
// cadence pair, which an EMPTY cell leaves standing rather than zeroing.
func TestFoldWeaponClearsAPreviousSpellWhenTheNewWeaponCarriesNone(t *testing.T) {
	base := baseCombat()
	base.SpellName, base.SpellPower = "Fire_Ball", 70

	w := meleeFixture()
	got := data.FoldWeapon(base, &w, 0)
	if got.SpellName != "" || got.SpellPower != 0 {
		t.Fatalf("spell pair = (%q, %d), want (\"\", 0) — a new weapon with no spell clears the old one",
			got.SpellName, got.SpellPower)
	}
}

func TestAWeaponWithAnEmptyRangeCellFoldsToReachOne(t *testing.T) {
	s, m, weapons := tables(t, nil, nil,
		[]synth.DataBinRow{weaponRow("Blade", 4, 6, 0, 0, 1, -1, 8, 4)})

	w, err := data.ResolveWeapon("Blade", s, m, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}

	got := data.FoldWeapon(data.Combat{Reach: 0}, &w, 0)
	if got.Reach != 1 {
		t.Fatalf("Reach = %d, want 1 for an empty range cell", got.Reach)
	}
}
