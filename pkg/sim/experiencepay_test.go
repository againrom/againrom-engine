package sim

import (
	"math"
	"testing"
)

// zeroXP is the all-zero SkillXP array every fixture below starts from, so a
// refusal test can compare against it by name rather than by a literal
// repeated at every call site.
var zeroXP [skillSlots]int32

// xpAttacker is one attacker for the payExperience tests below: a roster
// slot, a class that gains, a Mind of 60 and a credited slot of 3 — not 0,
// so a refusal that left slot 0 alone by accident could not pass by hiding
// behind General's own zero (experienceEntity's own reason, T2).
func xpAttacker(owner uint32) Entity {
	e := cbEnt(1, 0, 0)
	e.Owner, e.GainsXP, e.Mind, e.XPSlot, e.TypeID = owner, true, 60, 3, HumanTypeID
	return e
}

// xpTarget is one target for the payExperience tests below: a roster slot
// and an experience value of 10, over cbEnt's own 100/100 health.
func xpTarget(owner uint32) Entity {
	e := cbEnt(2, 1, 0)
	e.Owner, e.XPValue = owner, 10
	return e
}

func xpPay(t *testing.T, a, v Entity, removed int64, aliveBefore bool) [skillSlots]int32 {
	t.Helper()
	w := cbWorld(t, 1, a, v)
	ai, ti := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, v.ID)
	w.payExperience(ai, ti, removed, aliveBefore)
	return w.entities[ai].SkillXP
}

func TestPayExperienceRefusesANonPositiveRemoval(t *testing.T) {
	t.Parallel()

	for _, removed := range []int64{0, -1, -1000} {
		if got := xpPay(t, xpAttacker(2), xpTarget(3), removed, true); got != zeroXP {
			t.Errorf("removed %d paid %v, want all six unchanged", removed, got)
		}
	}
}

// TestPayExperienceRefusesATargetThatWasAlreadyDead is AC-5: a blow whose
// target was already dead before it landed pays nothing, whatever the blow
// removed.
func TestPayExperienceRefusesATargetThatWasAlreadyDead(t *testing.T) {
	t.Parallel()

	if got := xpPay(t, xpAttacker(2), xpTarget(3), 10, false); got != zeroXP {
		t.Errorf("an already-dead target paid %v, want all six unchanged", got)
	}
}

func TestPayExperienceRefusesAClassThatDoesNotGain(t *testing.T) {
	t.Parallel()

	a := xpAttacker(2)
	a.GainsXP = false
	if got := xpPay(t, a, xpTarget(3), 10, true); got != zeroXP {
		t.Errorf("a non-gaining class paid %v, want all six unchanged", got)
	}
}

func TestPayExperienceRefusesTheSameOwnerSlotAndALockedRelation(t *testing.T) {
	t.Parallel()

	t.Run("same owner slot", func(t *testing.T) {
		if got := xpPay(t, xpAttacker(5), xpTarget(5), 10, true); got != zeroXP {
			t.Errorf("sharing an owner slot paid %v, want all six unchanged", got)
		}
	})

	t.Run("relation locked", func(t *testing.T) {
		a, v := xpAttacker(2), xpTarget(3)
		rel := engRel(t, [3]uint32{2, 3, relationLocked})
		w, err := NewRelatedWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
			[]Entity{a, v}, nil, rel)
		if err != nil {
			t.Fatalf("NewRelatedWorld: %v", err)
		}
		ai, ti := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, v.ID)
		w.payExperience(ai, ti, 10, true)
		if got := w.entities[ai].SkillXP; got != zeroXP {
			t.Errorf("a locked relation paid %v, want all six unchanged", got)
		}
	})
}

// TestAGainingAttackerEarnsInExactlyOneSlot is AC-1, driven through the
// whole attack cycle rather than called directly: an attacker whose class
// gains, credited to slot 3, landing one deterministic blow (AlwaysHits, no
// spread) on a target outside its own owner slot ends the tick with more
// experience in slot 3 and the same experience — zero — in the other five.
// Nothing on the target moves but its health.
func TestAGainingAttackerEarnsInExactlyOneSlot(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 3, HumanTypeID
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 3, 10
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})

	got := cbAt(t, w, 1)
	for i, xp := range got.SkillXP {
		if i == int(a.XPSlot) {
			if xp <= 0 {
				t.Errorf("credited slot %d holds %d, want more than the starting zero", i, xp)
			}
			continue
		}
		if xp != 0 {
			t.Errorf("slot %d holds %d, want the starting zero — only slot %d should move", i, xp, a.XPSlot)
		}
	}
	if tv := cbAt(t, w, 2); tv.SkillXP != v.SkillXP {
		t.Errorf("the target's own experience moved: %v", tv.SkillXP)
	}
}

func TestAKillingBlowStillPaysExperience(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 3, HumanTypeID
	a.DamageBase = 105 // lethal but above the physical award's post-hit -10 gate
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue = 3, 10
	w := cbWorld(t, 200, a, v)
	Step(w, []Command{cbOrder(1, 2)})

	if tv := cbAt(t, w, 2); !tv.Dead() {
		t.Fatalf("the fixture is meant to kill outright; the target is at %d/%d", tv.HP, tv.MaxHP)
	}
	if got := cbAt(t, w, 1).SkillXP[3]; got <= 0 {
		t.Errorf("the killing blow paid slot 3 %d, want more than the starting zero", got)
	}
}

func TestPayingOrRefusingExperienceDrawsNothingEither(t *testing.T) {
	build := func(t *testing.T, gains bool) *World {
		t.Helper()
		a := cbFighter(1, 0, 0, 1, 9) // relax 9: relaxing whatever the jitter draws
		a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, gains, 60, 3, HumanTypeID
		v := cbEnt(2, 1, 0)
		v.Owner, v.XPValue = 3, 10
		return cbWorld(t, 909, a, v)
	}
	paid, refused := build(t, true), build(t, false)
	beforePaid, beforeRefused := paid.rng.state, refused.rng.state
	Step(paid, []Command{cbOrder(1, 2)})
	Step(refused, []Command{cbOrder(1, 2)})

	np, nr := cbDraws(t, beforePaid, paid.rng.state), cbDraws(t, beforeRefused, refused.rng.state)
	if np != nr {
		t.Errorf("a paid advance drew %d time(s), a refused one %d — experience must draw nothing", np, nr)
	}
	if got := cbAt(t, paid, 1).SkillXP[3]; got <= 0 {
		t.Fatalf("the fixture is meant to pay; slot 3 is %d", got)
	}
	if got := cbAt(t, refused, 1).SkillXP[3]; got != 0 {
		t.Fatalf("the fixture is meant to be refused; slot 3 is %d", got)
	}
}

func TestFR6PaysHalfPlusOneAndSplitsAcrossBlows(t *testing.T) {
	t.Run("outright: half plus one", func(t *testing.T) {
		const xpValue, maxHP = 101, 100
		if got, want := xpRaw(xpValue, maxHP, maxHP), int64(xpValue/2+1); got != want {
			t.Errorf("a blow removing the whole of maxHP paid raw %d, want %d", got, want)
		}
	})

	t.Run("two equal blows dividing evenly: no shortfall", func(t *testing.T) {
		const xpValue, maxHP = 100, 100
		half := int64(maxHP / 2)
		sum := xpRaw(xpValue, half, maxHP) + xpRaw(xpValue, half, maxHP)
		if want := int64(xpValue/2) + 2; sum != want {
			t.Errorf("two blows of %d each summed to %d, want %d (half %d plus one per blow)",
				half, sum, want, xpValue/2)
		}
	})

	t.Run("three unequal blows: truncation loses a fraction of the halved part", func(t *testing.T) {
		const xpValue, maxHP = 101, 100
		sum := xpRaw(xpValue, 33, maxHP) + xpRaw(xpValue, 33, maxHP) + xpRaw(xpValue, 34, maxHP)
		bound := int64(xpValue/2) + 3
		if sum > bound {
			t.Errorf("three split blows (33+33+34=%d) summed to %d, want at most %d (half %d plus one per blow)",
				33+33+34, sum, bound, xpValue/2)
		}
		if sum == bound {
			t.Errorf("three split blows summed to the bound %d exactly — the fixture is meant to demonstrate "+
				"a shortfall, not an even split; adjust the removed amounts", bound)
		}
	})
}

func TestXPRawMatchesTheRealValuedForm(t *testing.T) {
	xpValues := []int32{0, 1, 2, 3, 7, 25, 100, 999, 1 << 20, 2147483647}
	maxHPs := []int32{1, 2, 3, 5, 10, 37, 100, 1000, 2000}

	divergences := 0
	for _, xpValue := range xpValues {
		for _, maxHP := range maxHPs {
			for removed := int64(1); removed <= int64(maxHP)*3; removed++ {
				got := xpRaw(xpValue, removed, maxHP)
				want := int64(math.Trunc(float64(xpValue)*0.5*float64(removed)/float64(maxHP) + 1))
				if got != want {
					divergences++
					if divergences <= 5 {
						t.Errorf("xpValue %d removed %d maxHP %d: integer form %d, real-valued form %d",
							xpValue, removed, maxHP, got, want)
					}
				}
			}
		}
	}
	if divergences != 0 {
		t.Errorf("%d divergence(s) over the swept range (xpValue in %v, maxHP in %v, removed 1..3*maxHP) — "+
			"P-2 claims none", divergences, xpValues, maxHPs)
	}
}

func TestXPGainAgainstTheRealValuedForm(t *testing.T) {
	type pair struct {
		mind   int32
		amount int64
	}
	var first pair
	found := false
	divergences := 0

	for mind := int32(0); mind <= 400; mind++ {
		for amount := int64(0); amount <= 2000; amount++ {
			got := xpGain(amount, mind)
			want := int64(math.Trunc(float64(amount) * (float64(mind)/30 + 0.25)))
			if got != want {
				divergences++
				if !found {
					first, found = pair{mind, amount}, true
				}
			}
		}
	}

	t.Logf("P-3: %d divergence(s) over the swept range, first at mind=%d amount=%d "+
		"(integer form %d, real-valued form %d)",
		divergences, first.mind, first.amount,
		xpGain(first.amount, first.mind),
		int64(math.Trunc(float64(first.amount)*(float64(first.mind)/30+0.25))))

	if !found {
		t.Error("no divergence found over the swept range — P-3 expects mind/30's binary inexactness " +
			"to surface at least one boundary; widen the sweep before trusting this measurement")
	}
}

func TestFR7MindScalesTheGainToTheseExactNumbers(t *testing.T) {
	for _, c := range []struct {
		raw  int64
		mind int32
		want int64
	}{
		{raw: 1, mind: 0, want: 0},
		{raw: 1, mind: 60, want: 2},
		{raw: 6, mind: 0, want: 1},
		{raw: 6, mind: 25, want: 6},
		{raw: 6, mind: 60, want: 13},
		{raw: 100, mind: 100, want: 358},
	} {
		if got := xpGain(c.raw, c.mind); got != c.want {
			t.Errorf("xpGain(%d, mind %d) = %d, want %d", c.raw, c.mind, got, c.want)
		}
	}
}

func TestFR7TwoAttackersDifferingOnlyInMindEarnDifferentAmounts(t *testing.T) {
	for _, c := range []struct {
		mind int32
		want int32
	}{
		{mind: 0, want: 1},
		{mind: 25, want: 6},
		{mind: 60, want: 13},
	} {
		a := xpAttacker(2)
		a.Mind = c.mind
		got := xpPay(t, a, xpTarget(3), 100, true)
		if got[a.XPSlot] != c.want {
			t.Errorf("a whole target's health removed at Mind %d credited %d, want %d",
				c.mind, got[a.XPSlot], c.want)
		}
	}
}

func TestResolveBlowReadsTheLivenessBeforeItSubtracts(t *testing.T) {
	a := cbFighter(1, 0, 0, 1, 0)
	a.Owner, a.GainsXP, a.Mind, a.XPSlot, a.TypeID = 2, true, 60, 3, HumanTypeID
	v := cbEnt(2, 1, 0)
	v.Owner, v.XPValue, v.HP = 3, 10, -5
	w := cbWorld(t, 1, a, v)

	ai, ti := indexOfEntity(w.entities, a.ID), indexOfEntity(w.entities, v.ID)
	before := w.entities[ti].HP
	w.resolveBlow(ai, ti, nil)

	if w.entities[ti].HP >= before {
		t.Fatalf("the blow removed nothing (%d -> %d); this case only means "+
			"something while the strike still lands", before, w.entities[ti].HP)
	}
	if got := w.entities[ai].SkillXP; got != zeroXP {
		t.Errorf("a blow on an already-dead target paid %v, want %v", got, zeroXP)
	}
}
