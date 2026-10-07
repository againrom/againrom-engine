package data_test

import (
	"testing"

	"againrom/pkg/data"
)

// TestAC2UnequippedProtectionsAreCappedSpiritHalved is AC-2: a recompute of
// an unequipped character reads the same number on all five elemental
// protections, and that number is his capped Spirit halved.
func TestAC2UnequippedProtectionsAreCappedSpiritHalved(t *testing.T) {
	h := data.Hero{Spirit: 41}
	d := h.Recompute(data.Profile{}, data.Loadout{})

	want := int32(41 / 2)
	for i, p := range d.Protection {
		if p != want {
			t.Errorf("Protection[%d] = %d, want %d", i, p, want)
		}
	}

	// A stat above the cap must read the CAPPED Spirit, not the raw one.
	capped := data.Hero{Spirit: 999}
	dc := capped.Recompute(data.Profile{}, data.Loadout{})
	wantCapped := data.StatCap / 2
	for i, p := range dc.Protection {
		if p != wantCapped {
			t.Errorf("capped Protection[%d] = %d, want %d", i, p, wantCapped)
		}
	}
}

// TestAC3RecomputeIsIdempotentAndDoesNotAccumulate is AC-3: recomputing
// twice in a row over the same inputs yields the same derived set, and the
// cleared block does not accumulate across two recomputes.
func TestAC3RecomputeIsIdempotentAndDoesNotAccumulate(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30, Mind: 20, Spirit: 20}
	h.Skill[data.SkillBlade] = 15
	p := data.Profile{Fighter: true, HealthColumn: true, ManaColumn: true}
	l := data.Loadout{Mod: data.EquipMod{Defence: 5, Protection: [5]int32{1, 2, 3, 4, 5}}}

	a := h.Recompute(p, l)
	b := h.Recompute(p, l)
	if a != b {
		t.Fatalf("two recomputes over the same inputs differ:\n%+v\n%+v", a, b)
	}
}

// TestAC5ExperienceIsAFunctionOfSkillLevels is AC-5: a character whose skill
// slots are all zero has experience 0; one skill at level 10 accounts for
// 1593 and nothing else contributes.
func TestAC5ExperienceIsAFunctionOfSkillLevels(t *testing.T) {
	zero := data.Hero{}
	d := zero.Recompute(data.Profile{}, data.Loadout{})
	if d.Experience != 0 {
		t.Errorf("all-zero skills: experience = %d, want 0", d.Experience)
	}
	for i, v := range d.SkillXP {
		if v != 0 {
			t.Errorf("all-zero skills: SkillXP[%d] = %d, want 0", i, v)
		}
	}

	trained := data.Hero{}
	trained.Skill[data.SkillBlade] = 10
	dt := trained.Recompute(data.Profile{}, data.Loadout{})
	if dt.Experience != 1593 {
		t.Errorf("one slot at level 10: experience = %d, want 1593", dt.Experience)
	}
	for i, v := range dt.SkillXP {
		if i == data.SkillBlade {
			if v != 1593 {
				t.Errorf("SkillXP[%d] = %d, want 1593", i, v)
			}
			continue
		}
		if v != 0 {
			t.Errorf("SkillXP[%d] = %d, want 0 (nothing else trained)", i, v)
		}
	}
}

// TestAC3HealthColumnDoesNotMoveTheMaximum is 0133-person-health's AC-3: two
// profiles differing only in the column flag, over one character, derive the
// SAME maximum. The health arm's own gate is the product of the capped Body
// and the class multiplier (recompute.go step 3); HealthColumn plays no part
// in it at either end.
func TestAC3HealthColumnDoesNotMoveTheMaximum(t *testing.T) {
	h := data.Hero{Body: 25}
	h.Skill[data.SkillBlade] = 10 // gives Experience 1593, so the log term is nonzero too

	without := h.Recompute(data.Profile{HealthColumn: false}, data.Loadout{})
	with := h.Recompute(data.Profile{HealthColumn: true}, data.Loadout{})

	if without.HealthMax != with.HealthMax {
		t.Fatalf("HealthColumn false gave HealthMax %d, true gave %d, want the same number on "+
			"both -- the column plays no part in the arm's gate", without.HealthMax, with.HealthMax)
	}
	// A CONCRETE NUMBER: Body 25, Experience 1593, no class flag, recomputes
	// to 29 on both sides. Reverting recompute.go's product gate to a column
	// check on the arm's first term alone would answer 2 with the column
	// clear and leave the equality above failing before this line runs.
	if without.HealthMax != 29 {
		t.Fatalf("the fixture's own figure moved: Body 25, Experience 1593 now recomputes to %d, "+
			"want 29 -- update this comment before trusting the rest of this test", without.HealthMax)
	}
}

// TestAC2ZeroBodyDerivesNoHealth is 0133-person-health's AC-2: a character
// whose capped Body is 0 derives a health maximum of 0 -- the WHOLE arm
// yields nothing, not merely the term the column used to guard. The fixture
// trains a skill so Experience is nonzero: an untrained Body-0 character
// would answer 0 whichever gate recompute.go used, and would witness
// nothing about which one is in effect.
func TestAC2ZeroBodyDerivesNoHealth(t *testing.T) {
	h := data.Hero{}
	h.Skill[data.SkillBlade] = 10 // Experience 1593, nonzero, and still reaches nothing below

	d := h.Recompute(data.Profile{}, data.Loadout{})
	if d.HealthMax != 0 {
		t.Fatalf("Body 0, trained: HealthMax = %d, want 0 -- the arm's gate is the PRODUCT of "+
			"Body and the class multiplier, so a zero Body zeroes the WHOLE arm, experience term "+
			"included", d.HealthMax)
	}
	// Reverting recompute.go's step-3 gate back to a column check leaves
	// this failing: a trained Body-0 character would then answer HealthMax
	// 2 from the experience term alone, whatever the column said.
}

// TestAC4NonzeroBodyDerivesPositiveWithNoHealthColumn is
// 0133-person-health's AC-4: a character with a nonzero Body and a health
// column of 0 still derives a POSITIVE maximum -- the arm's gate no longer
// needs the column at all. The fixture is untrained (Experience 0) on
// purpose, so the whole positive answer comes from the Body term and the
// growth term alone, with nothing from experience to blur which term is
// doing the work.
func TestAC4NonzeroBodyDerivesPositiveWithNoHealthColumn(t *testing.T) {
	h := data.Hero{Body: 25}
	d := h.Recompute(data.Profile{HealthColumn: false}, data.Loadout{})
	if d.HealthMax <= 0 {
		t.Fatalf("Body 25, HealthColumn false: HealthMax = %d, want a positive figure -- a row "+
			"stating no health column no longer means a character built from it has none",
			d.HealthMax)
	}
	// A CONCRETE NUMBER: Body 25, no skill trained, no class flag, no health
	// column, recomputes to 27. Reverting recompute.go's gate to a column
	// check would answer 0 here instead: the old gate skipped the arm's
	// first term entirely when the column was absent and left nothing else
	// to contribute at zero Experience.
	if d.HealthMax != 27 {
		t.Fatalf("the fixture's own figure moved: Body 25, HealthColumn false now recomputes to "+
			"%d, want 27 -- update this comment before trusting the rest of this test", d.HealthMax)
	}
}

func TestAC11DerivedMaximumIsIdempotent(t *testing.T) {
	d := data.HumanDef{Body: 30, Reaction: 20, Mind: 15, Spirit: 15}
	d.Skill[data.SkillBlade] = 20

	a := d.DerivedMaximum()
	b := d.DerivedMaximum()
	if a != b {
		t.Fatalf("two derivations of the same definition differ: %d then %d, want the same "+
			"number both times", a, b)
	}
	if a <= 0 {
		t.Fatalf("the fixture's own derivation answered %d, want a positive figure -- this test "+
			"would pass vacuously comparing two zeroes", a)
	}

	if want := d.Hero().Recompute(d.Profile(), data.Loadout{}).HealthMax; a != want {
		t.Errorf("DerivedMaximum answered %d, want Recompute's own %d over the same Hero and Profile",
			a, want)
	}
}

// TestAC7ClampPools is AC-7: a live health above the maximum clamps to it;
// a character with ManaColumn false ends at zero mana.
func TestAC7ClampPools(t *testing.T) {
	// HE HAS TRAINED, so his experience term is nonzero: a character with
	// no skills would reach ManaMax 0 through the logarithm of 1 whatever
	// the column gate did, and would witness nothing about the gate.
	h := data.Hero{Body: 25, Spirit: 25, Skill: [data.SkillSlots]int32{data.SkillBlade: 10}}
	d := h.Recompute(data.Profile{HealthColumn: true, ManaColumn: false}, data.Loadout{})
	if armed := h.Recompute(data.Profile{HealthColumn: true, ManaColumn: true}, data.Loadout{}); armed.ManaMax == 0 {
		t.Fatalf("a set mana column gave ManaMax 0, so the gate below witnesses nothing")
	}

	gotHealth, gotMana := d.ClampPools(d.HealthMax+1000, 500)
	if gotHealth != d.HealthMax {
		t.Errorf("health clamped to %d, want the maximum %d", gotHealth, d.HealthMax)
	}
	if d.ManaMax != 0 {
		t.Fatalf("ManaColumn false gave ManaMax %d, want 0 (setup assumption broken)", d.ManaMax)
	}
	if gotMana != 0 {
		t.Errorf("mana clamped to %d, want 0 through a zero ManaMax", gotMana)
	}
}

// TestAC8ResistancesAreZeroWithOnlyAWeapon is AC-8: the five damage-kind
// resistances of a character carrying only a weapon are all zero.
func TestAC8ResistancesAreZeroWithOnlyAWeapon(t *testing.T) {
	w := data.Weapon{Name: "W", AttackType: data.SkillBlade, DamageBase: 5, DamageSpread: 3, Range: 1}
	h := data.Hero{Body: 30, Reaction: 30, Spirit: 30}
	d := h.Recompute(data.Profile{}, data.Loadout{Weapon: &w})

	for i, r := range d.Resistance {
		if r != 0 {
			t.Errorf("Resistance[%d] = %d, want 0", i, r)
		}
	}
}

// TestRangedRecomputeUsesLiveGeneralAndThenAppliesTheElementalEffect covers
// the generated-person and live Rearm producer. Weapon::Equip writes its
// built-in ranged component before walking item effects, so the later effect
// replaces the whole triple; accuracy still uses the Hero's live General
// rather than the weapon's ToHit or a General-bonus effect.
func TestRangedRecomputeUsesLiveGeneralAndThenAppliesTheElementalEffect(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30}
	h.Skill[data.SkillGeneral] = 17
	w := data.Weapon{
		AttackType: 11, DamageBase: 20, DamageSpread: 10,
		ToHit: 90, Defence: 80, Range: 8,
	}
	mod := data.EquipMod{
		SecondaryDamage:    data.SecondaryDamage{Base: 250, Spread: 251, Selector: 4},
		HasSecondaryDamage: true,
	}
	mod.SkillBonus[data.SkillGeneral] = 23

	bare := h.Recompute(data.Profile{}, data.Loadout{Mod: mod})
	got := h.Recompute(data.Profile{}, data.Loadout{Mod: mod, Weapon: &w})

	if got.Combat.DamageBase != bare.Combat.DamageBase ||
		got.Combat.DamageSpread != bare.Combat.DamageSpread ||
		got.Combat.Defence != bare.Combat.Defence {
		t.Errorf("ranged weapon moved physical/defence fields: got %+v, bare %+v", got.Combat, bare.Combat)
	}
	if got.Combat.ToHit != bare.Combat.ToHit+17 {
		t.Errorf("ToHit = %d, want bare %d + live General 17; weapon ToHit=%d and General bonus=23 are not sources",
			got.Combat.ToHit, bare.Combat.ToHit, w.ToHit)
	}
	if want := (data.SecondaryDamage{Base: 250, Spread: 251, Selector: 4}); got.SecondaryDamage != want {
		t.Errorf("SecondaryDamage = %+v, want later item effect to replace weapon Fire %+v", got.SecondaryDamage, want)
	}
	if got.Combat.Reach != 8 || got.Combat.SkillSlot != data.SkillGeneral {
		t.Errorf("Reach/SkillSlot = %d/%d, want 8/General", got.Combat.Reach, got.Combat.SkillSlot)
	}

	zeroEffect := h.Recompute(data.Profile{}, data.Loadout{
		Weapon: &w, Mod: data.EquipMod{HasSecondaryDamage: true},
	})
	if zeroEffect.SecondaryDamage != (data.SecondaryDamage{}) {
		t.Errorf("explicit zero elemental effect left weapon component %+v, want zero replacement",
			zeroEffect.SecondaryDamage)
	}
}

// TestAC9EquipModDefenceAndDamageAreIndependent is AC-9: a modifier block
// carrying a defence term moves defence and no damage number; one carrying
// a damage term moves the pair and not defence.
func TestAC9EquipModDefenceAndDamageAreIndependent(t *testing.T) {
	h := data.Hero{Body: 30, Reaction: 30}
	base := h.Recompute(data.Profile{}, data.Loadout{})

	withDefence := h.Recompute(data.Profile{}, data.Loadout{Mod: data.EquipMod{Defence: 7}})
	if withDefence.Combat.Defence-base.Combat.Defence != 7 {
		t.Errorf("defence moved by %d, want 7", withDefence.Combat.Defence-base.Combat.Defence)
	}
	if withDefence.Combat.DamageBase != base.Combat.DamageBase ||
		withDefence.Combat.DamageSpread != base.Combat.DamageSpread {
		t.Errorf("a defence-only EquipMod moved the damage pair: %+v vs %+v",
			withDefence.Combat, base.Combat)
	}

	withDamage := h.Recompute(data.Profile{}, data.Loadout{Mod: data.EquipMod{DamageBase: 4, DamageSpread: 2}})
	if withDamage.Combat.DamageBase-base.Combat.DamageBase != 4 {
		t.Errorf("damage base moved by %d, want 4", withDamage.Combat.DamageBase-base.Combat.DamageBase)
	}
	if withDamage.Combat.DamageSpread-base.Combat.DamageSpread != 2 {
		t.Errorf("damage spread moved by %d, want 2", withDamage.Combat.DamageSpread-base.Combat.DamageSpread)
	}
	if withDamage.Combat.Defence != base.Combat.Defence {
		t.Errorf("a damage-only EquipMod moved defence: %d vs %d",
			withDamage.Combat.Defence, base.Combat.Defence)
	}
}

// TestAC10ProtectionClamp is AC-10: a protection term over the clamp is
// bounded by min(Spirit/2+70, 100); a negative one is bounded at 0.
func TestAC10ProtectionClamp(t *testing.T) {
	h := data.Hero{Spirit: 30} // capped Spirit/2 = 15, ceiling = 15+70 = 85
	over := h.Recompute(data.Profile{}, data.Loadout{Mod: data.EquipMod{Protection: [5]int32{1000, 0, 0, 0, 0}}})
	wantCeiling := int32(30/2 + 70)
	if over.Protection[0] != wantCeiling {
		t.Errorf("over-clamp protection = %d, want %d", over.Protection[0], wantCeiling)
	}

	under := h.Recompute(data.Profile{}, data.Loadout{Mod: data.EquipMod{Protection: [5]int32{-1000, 0, 0, 0, 0}}})
	if under.Protection[0] != 0 {
		t.Errorf("negative protection = %d, want 0", under.Protection[0])
	}
}

// TestAC5SlotZeroIsInTheSum is the second half of AC-5: slot 0 -- the one the
// sheet does not show, and the one the restore and the clamp both skip -- is
// nonetheless inside the experience sum. The asymmetry is the original's; a
// build that dropped slot 0 from the sum would lose experience silently.
func TestAC5SlotZeroIsInTheSum(t *testing.T) {
	h := data.Hero{}
	h.Skill[data.SkillGeneral] = 10
	d := h.Recompute(data.Profile{}, data.Loadout{})
	if d.SkillXP[data.SkillGeneral] != 1593 || d.Experience != 1593 {
		t.Errorf("slot 0 at level 10: SkillXP[0] = %d, Experience = %d, want 1593 and 1593",
			d.SkillXP[data.SkillGeneral], d.Experience)
	}
}

// TestAC13TheSkillRestore is AC-13: a per-slot bonus raises a level and the
// raised level moves to-hit and the damage base; the same bonus on slot 0 moves
// nothing; and the clamp bounds a level at 100 above and 0 below.
func TestAC13TheSkillRestore(t *testing.T) {
	w := data.Weapon{Name: "W", AttackType: data.SkillBlade, Range: 1}
	h := data.Hero{Body: 30, Reaction: 30}
	h.Skill[data.SkillBlade] = 10

	bare := h.Recompute(data.Profile{}, data.Loadout{Weapon: &w})

	var mod data.EquipMod
	mod.SkillBonus[data.SkillBlade] = 20
	raised := h.Recompute(data.Profile{}, data.Loadout{Weapon: &w, Mod: mod})

	if raised.Skill[data.SkillBlade] != 30 {
		t.Errorf("a bonus of 20 on a level of 10 restored to %d, want 30",
			raised.Skill[data.SkillBlade])
	}
	// Three times the level to to-hit, a fifth of it to the base alone.
	if got, want := raised.Combat.ToHit-bare.Combat.ToHit, int32(3*20); got != want {
		t.Errorf("the raised level moved to-hit by %d, want %d", got, want)
	}
	if got, want := raised.Combat.DamageBase-bare.Combat.DamageBase, int32(30/5-10/5); got != want {
		t.Errorf("the raised level moved the damage base by %d, want %d", got, want)
	}
	if raised.Combat.DamageSpread != bare.Combat.DamageSpread {
		t.Errorf("the raised level moved the spread to %d, want %d unchanged",
			raised.Combat.DamageSpread, bare.Combat.DamageSpread)
	}

	// SLOT 0 is the shared general-skill modifier. It is restored before the
	// class-specific five slots and is not clamped.
	var zeroMod data.EquipMod
	zeroMod.SkillBonus[data.SkillGeneral] = 40
	g := data.Hero{Body: 30, Reaction: 30}
	g.Skill[data.SkillGeneral] = 200
	dg := g.Recompute(data.Profile{}, data.Loadout{Weapon: &w, Mod: zeroMod})
	if dg.Skill[data.SkillGeneral] != 240 {
		t.Errorf("slot 0 restored to %d, want incoming 200 plus modifier 40", dg.Skill[data.SkillGeneral])
	}

	// The clamp, both ends. The original clamps base plus bonus to [0,100]
	// (HERO-SKILL-009); this tree lifts the ceiling to the effective bound,
	// 255 as a literal on purpose, and keeps the floor at 0. The trained base
	// still holds to the training cap, 100 by default, before the bonus.
	var big data.EquipMod
	big.SkillBonus[data.SkillBlade] = 500
	if got := h.Recompute(data.Profile{}, data.Loadout{Mod: big}).Skill[data.SkillBlade]; got != 255 {
		t.Errorf("a bonus past the bound restored to %d, want the effective bound 255", got)
	}
	var neg data.EquipMod
	neg.SkillBonus[data.SkillBlade] = -500
	if got := h.Recompute(data.Profile{}, data.Loadout{Mod: neg}).Skill[data.SkillBlade]; got != 0 {
		t.Errorf("a bonus past the floor restored to %d, want the decoded floor 0", got)
	}

	// The constants themselves carry the decoded bounds, checked here rather
	// than assumed by the two assertions above.
	if data.SkillCap != 100 || data.SkillFloor != 0 {
		t.Errorf("SkillFloor/SkillCap are %d/%d, want the decoded 0/100 (HERO-SKILL-009)",
			data.SkillFloor, data.SkillCap)
	}
}

// skillCurveOf is S(n) itself, read off Recompute rather than reimplemented:
// a hero with exactly one slot -- SkillBlade -- trained to n, and that slot's
// own SkillXP afterwards. The two tests below compare SkillLevelFor against
// this, not against a second copy of the formula.
func skillCurveOf(n int32) int32 {
	h := data.Hero{}
	h.Skill[data.SkillBlade] = n
	return h.Recompute(data.Profile{}, data.Loadout{}).SkillXP[data.SkillBlade]
}

func TestP1SkillLevelForIsTheExactInverse(t *testing.T) {
	prev := skillCurveOf(0)
	for n := int32(0); n <= 100; n++ {
		s := skillCurveOf(n)
		if got := data.SkillLevelFor(s); got != n {
			t.Errorf("SkillLevelFor(S(%d)=%d) = %d, want %d", n, s, got, n)
		}
		if n > 0 && s > prev {
			if got := data.SkillLevelFor(s - 1); got >= n {
				t.Errorf("SkillLevelFor(S(%d)-1=%d) = %d, want below %d", n, s-1, got, n)
			}
		}
		prev = s
	}
}

func TestSkillCurveNeverGoesFlat(t *testing.T) {
	prev := skillCurveOf(0)
	for n := int32(1); n <= 100; n++ {
		s := skillCurveOf(n)
		if s == prev {
			t.Errorf("S(%d) == S(%d) == %d: the curve is flat here, and the inverse is not injective at this step", n, n-1, s)
		}
		prev = s
	}
}
