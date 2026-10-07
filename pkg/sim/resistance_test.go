package sim

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"testing"
)

// resistanceBlow takes one deterministic physical blow through resolveBlow,
// the production resolver. The target starts far enough above every damage in
// these cases that the health floor cannot disguise the exact result.
func resistanceBlow(t *testing.T, slot uint8, resistance [5]uint8, absorption, damage int32, always bool) int32 {
	t.Helper()
	a := cbEnt(1, 0, 0)
	a.DamageBase = damage
	a.DamageSpread = 0
	a.ToHit = 2147483647
	a.AlwaysHits = always
	a.XPSlot = slot
	v := cbEnt(2, 1, 0)
	v.HP, v.MaxHP = 1_000_000, 1_000_000
	v.Absorption = absorption
	v.Resistance = resistance
	w := cbWorld(t, 1039, a, v)
	w.resolveBlow(0, 1, nil)
	return 1_000_000 - cbAt(t, w, 2).HP
}

// TestEveryWeaponKindSelectsItsOwnResistanceByte is the five-kind indexing
// witness. Slot zero is the unfilled leading byte and bypasses the family;
// slots one through five select Blade, Axe, Bludgeon, Pike and Shooting in
// order. Construction and SetCombat already refuse slots above five, so the
// subtraction can never index outside this array in a valid world.
func TestEveryWeaponKindSelectsItsOwnResistanceByte(t *testing.T) {
	resistance := [5]uint8{10, 20, 30, 40, 50}
	for _, tc := range []struct {
		slot uint8
		want int32
	}{
		{0, 100},
		{1, 90},
		{2, 80},
		{3, 70},
		{4, 60},
		{5, 50},
	} {
		if got := resistanceBlow(t, tc.slot, resistance, 0, 100, false); got != tc.want {
			t.Errorf("slot %d removed %d, want %d", tc.slot, got, tc.want)
		}
	}
}

// TestResistanceUsesThePublishedRoundingAtEveryBoundary gives the named
// arithmetic cases independently of the implementation expression, then
// compares every legal byte at representative reachable damages with the
// published floating expression. Values above 100 produce a non-positive
// component and the caller's final clamp removes nothing.
func TestResistanceUsesThePublishedRoundingAtEveryBoundary(t *testing.T) {
	for _, tc := range []struct {
		damage     int64
		resistance uint8
		want       int64
	}{
		{1, 0, 1},
		{1, 75, 1},
		{1, 76, 0},
		{10, 25, 8},
		{10, 50, 5},
		{10, 75, 3},
		{99, 99, 1},
		{100, 99, 1},
		{100, 100, 0},
		{100, 101, 0},
		{200, 101, -1},
		{200, 255, -309},
	} {
		if got := resistPhysicalDamage(tc.damage, tc.resistance); got != tc.want {
			t.Errorf("damage %d resistance %d: got %d, want %d", tc.damage, tc.resistance, got, tc.want)
		}
	}

	// This is the largest pre-resistance component the three signed int32
	// combat fields can produce: max base + max spread - min absorption.
	const largestReachable = int64(2147483647) + int64(2147483647) - int64(-2147483648)
	for _, damage := range []int64{1, 2, 10, 99, 100, 101, 200, 2147483647, largestReachable} {
		for r := 0; r <= 255; r++ {
			want := int64(math.Trunc(float64(damage)*(100-float64(r))/100 + 0.75))
			if got := resistPhysicalDamage(damage, uint8(r)); got != want {
				t.Fatalf("damage %d resistance %d: got %d, published expression gives %d", damage, r, got, want)
			}
		}
	}
}

// TestAbsorptionClampAndResistanceKeepTheirOrder distinguishes all three
// arms: ordinary blows absorb before multiplying, AlwaysHits skips absorption,
// and slot zero bypasses resistance. A non-positive physical component stops
// before multiplication and a resistance result at or below zero stops before
// health, attribution, experience or a weapon rider can move.
func TestAbsorptionClampAndResistanceKeepTheirOrder(t *testing.T) {
	resistance := [5]uint8{50}
	if got := resistanceBlow(t, 1, resistance, 20, 100, false); got != 40 {
		t.Errorf("ordinary blow removed %d, want (100-20)*50%% rounded = 40", got)
	}
	if got := resistanceBlow(t, 1, resistance, 20, 100, true); got != 50 {
		t.Errorf("AlwaysHits blow removed %d, want 100*50%% = 50 with absorption skipped", got)
	}
	if got := resistanceBlow(t, 0, resistance, 20, 100, false); got != 80 {
		t.Errorf("slot-zero blow removed %d, want 80 with resistance bypassed", got)
	}
	if got := resistanceBlow(t, 1, [5]uint8{255}, 0, 200, false); got != 0 {
		t.Errorf("resistance 255 removed %d, want the negative component clamped to zero", got)
	}
	if got := resistanceBlow(t, 1, resistance, 100, 100, false); got != 0 {
		t.Errorf("fully absorbed blow removed %d, want the pre-resistance zero clamp", got)
	}
	if got := resistanceBlow(t, 1, resistance, 0, -1, false); got != 0 {
		t.Errorf("negative base damage removed %d, want the pre-resistance zero clamp", got)
	}
	if got := resistanceBlow(t, 1, resistance, -20, 100, false); got != 60 {
		t.Errorf("negative absorption removed %d, want (100-(-20))*50%% = 60", got)
	}
}

// TestGhostBearingConstructorsRefuseEveryUnrepresentableResistanceSelector
// covers both public constructors that can retain a raisable GhostTemplate.
// All ten public constructors reach newWorld, so these are the two nonzero
// template entrances to its shared selector boundary.
func TestGhostBearingConstructorsRefuseEveryUnrepresentableResistanceSelector(t *testing.T) {
	type constructor struct {
		name  string
		build func(GhostTemplate) (*World, error)
	}
	constructors := []constructor{
		{name: "NewSummoningWorld", build: func(g GhostTemplate) (*World, error) {
			return NewSummoningWorld(1039, Bounds{Width: 16, Height: 16}, ModeCanonical,
				Terrain{}, nil, nil, Relations{}, nil, nil, nil, g)
		}},
		{name: "NewStructuredWorld", build: func(g GhostTemplate) (*World, error) {
			return NewStructuredWorld(1039, Bounds{Width: 16, Height: 16}, ModeCanonical,
				Terrain{}, nil, nil, Relations{}, nil, nil, nil, g, nil)
		}},
	}

	for _, constructor := range constructors {
		for _, tc := range []struct {
			slot uint8
			ok   bool
		}{
			{slot: 0, ok: true},
			{slot: 5, ok: true},
			{slot: 6},
			{slot: 255},
		} {
			t.Run(constructor.name+"/slot-"+strconv.Itoa(int(tc.slot)), func(t *testing.T) {
				ghost := hlGhostTemplate()
				ghost.XPSlot = tc.slot
				w, err := constructor.build(ghost)
				if tc.ok {
					if err != nil {
						t.Fatalf("slot %d was refused: %v", tc.slot, err)
					}
					if got := w.Ghost().XPSlot; got != tc.slot {
						t.Fatalf("stored slot = %d, want %d", got, tc.slot)
					}
					return
				}
				if err == nil || w != nil {
					t.Fatalf("slot %d produced world=%v error=%v, want nil world and refusal", tc.slot, w != nil, err)
				}
				if !strings.Contains(err.Error(), "ghost template: experience slot") {
					t.Fatalf("slot %d refusal = %q, want ghost-template selector fault", tc.slot, err)
				}
			})
		}
	}
}

// TestResistanceResultOwnsAttributionExperienceAndRelationConsequences makes
// the post-resistance value observable at every downstream physical-blow
// consumer. A landed blow still flips both relations even when resistance
// cancels it; a cancelled component writes neither attribution nor XP, while
// a positive component pays from the reduced amount rather than the roll.
func TestResistanceResultOwnsAttributionExperienceAndRelationConsequences(t *testing.T) {
	attacker := cbEnt(1, 0, 0)
	attacker.Owner = 2
	attacker.TypeID = HumanTypeID
	attacker.GainsXP = true
	attacker.Mind = 30
	attacker.XPSlot = 1
	attacker.DamageBase = 100
	attacker.AlwaysHits = true

	t.Run("full cancellation", func(t *testing.T) {
		target := cbEnt(2, 1, 0)
		target.Owner = 3
		target.XPValue = 100
		target.Resistance[0] = 100
		w := cbWorld(t, 1039, attacker, target)
		w.resolveBlow(0, 1, nil)

		gotTarget := cbAt(t, w, 2)
		gotAttacker := cbAt(t, w, 1)
		if gotTarget.HP != target.HP || gotTarget.HasKillCredit || gotTarget.KillCreditSource != 0 {
			t.Fatalf("cancelled blow left target hp=%d credit=%v/%d, want %d and no credit",
				gotTarget.HP, gotTarget.HasKillCredit, gotTarget.KillCreditSource, target.HP)
		}
		if gotAttacker.SkillXP != ([skillSlots]int32{}) {
			t.Fatalf("cancelled blow paid skill XP %v, want none", gotAttacker.SkillXP)
		}
		if !w.Relations().Hostile(attacker.Owner, target.Owner) ||
			!w.Relations().Hostile(target.Owner, attacker.Owner) {
			t.Fatalf("cancelled landed blow relations %d->%d=%d %d->%d=%d, want both hostile",
				attacker.Owner, target.Owner, w.Relations().Byte(attacker.Owner, target.Owner),
				target.Owner, attacker.Owner, w.Relations().Byte(target.Owner, attacker.Owner))
		}
	})

	t.Run("positive reduction", func(t *testing.T) {
		target := cbEnt(2, 1, 0)
		target.Owner = 3
		target.XPValue = 100
		target.Resistance[0] = 50
		w := cbWorld(t, 1039, attacker, target)
		w.resolveBlow(0, 1, nil)

		gotTarget := cbAt(t, w, 2)
		gotAttacker := cbAt(t, w, 1)
		if gotTarget.HP != 50 || !gotTarget.HasKillCredit || gotTarget.KillCreditSource != attacker.ID {
			t.Fatalf("reduced blow left target hp=%d credit=%v/%d, want 50 and source %d",
				gotTarget.HP, gotTarget.HasKillCredit, gotTarget.KillCreditSource, attacker.ID)
		}
		want := int32(xpGain(xpRaw(target.XPValue, 50, target.MaxHP), attacker.Mind))
		full := int32(xpGain(xpRaw(target.XPValue, 100, target.MaxHP), attacker.Mind))
		if got := gotAttacker.SkillXP[attacker.XPSlot]; got != want || got == full {
			t.Fatalf("reduced blow paid %d XP, want reduced-amount %d and not full-roll %d", got, want, full)
		}
		for slot, got := range gotAttacker.SkillXP {
			if slot != int(attacker.XPSlot) && got != 0 {
				t.Fatalf("reduced blow paid slot %d = %d, want only slot %d", slot, got, attacker.XPSlot)
			}
		}
	})
}

// TestAControlSpiritGhostRoundTripsBeforeItsKindReducesAPhysicalBlow follows
// the production Control Spirit arm through the only runtime Entity append,
// then decodes its real byte form before using the raised actor in the shared
// physical resolver.
func TestAControlSpiritGhostRoundTripsBeforeItsKindReducesAPhysicalBlow(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 1)
	corpse.HP, corpse.Decay = -10, DecayBones
	target := spEnt(3, 3, 1)
	target.HP, target.MaxHP, target.Owner = 1_000, 1_000, 8
	target.Resistance[3] = 50

	ghost := hlGhostTemplate()
	ghost.XPSlot = 4
	ghost.DamageBase, ghost.DamageSpread, ghost.AlwaysHits = 100, 0, true
	w := hlGhostWorld(t, 1039,
		[]SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}},
		ghost, caster, corpse, target)
	spRunCast(w, Command{Kind: KindCast, Entity: caster.ID, X: int32(corpse.ID), Y: 25})

	if got := cbAt(t, w, 4); got.XPSlot != ghost.XPSlot || got.Resistance != ghost.Resistance {
		t.Fatalf("raised ghost selector/resistance = %d/%v, want %d/%v",
			got.XPSlot, got.Resistance, ghost.XPSlot, ghost.Resistance)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary after Control Spirit: %v", err)
	}
	var resisted, control World
	if err := resisted.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary resisted world: %v", err)
	}
	if err := control.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary control world: %v", err)
	}
	backForm, err := resisted.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary decoded Control Spirit world: %v", err)
	}
	if !bytes.Equal(backForm, form) || resisted.Hash() != w.Hash() {
		t.Fatalf("Control Spirit round-trip moved its canonical bytes or hash: bytes equal=%v hash %016x/%016x",
			bytes.Equal(backForm, form), resisted.Hash(), w.Hash())
	}
	for _, decoded := range []struct {
		label string
		world *World
	}{
		{label: "resisted", world: &resisted},
		{label: "control", world: &control},
	} {
		got := cbAt(t, decoded.world, 4)
		if got.XPSlot != 4 || got.DamageBase != 100 || got.DamageSpread != 0 || !got.AlwaysHits {
			t.Fatalf("%s decoded ghost = %+v, want deterministic kind-4 100-damage attacker", decoded.label, got)
		}
	}

	resistedGhost, resistedTarget := indexOfEntity(resisted.entities, 4), indexOfEntity(resisted.entities, target.ID)
	controlGhost, controlTarget := indexOfEntity(control.entities, 4), indexOfEntity(control.entities, target.ID)
	if resistedGhost < 0 || resistedTarget < 0 || controlGhost < 0 || controlTarget < 0 {
		t.Fatalf("decoded indices resisted=%d/%d control=%d/%d, want ghost and target in both worlds",
			resistedGhost, resistedTarget, controlGhost, controlTarget)
	}
	control.entities[controlTarget].Resistance = [5]uint8{}
	resisted.resolveBlow(resistedGhost, resistedTarget, nil)
	control.resolveBlow(controlGhost, controlTarget, nil)
	resistedDamage := int32(1_000) - cbAt(t, &resisted, target.ID).HP
	controlDamage := int32(1_000) - cbAt(t, &control, target.ID).HP
	if controlDamage != 100 || resistedDamage != 50 {
		t.Fatalf("decoded raised ghost dealt control/resisted %d/%d, want 100/50", controlDamage, resistedDamage)
	}
}
