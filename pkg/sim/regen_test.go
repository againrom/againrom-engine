package sim

// The regeneration pass (0109 T2): the two arms, their cadence, their gates
// and their inertness. The record and the byte form are T1's and are not
// re-tested here; the placement (T3) and the panel (T4) are later tasks'.

import "testing"

// rgWorld builds an eight-by-eight canonical world holding ents alone.
func rgWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	return mustWorld(t, 3, Bounds{Width: 8, Height: 8}, ents)
}

func TestTheCadenceConstantsAreTheDecodedOnes(t *testing.T) {
	t.Parallel()
	const wantPhase, wantManaCycle, wantHealthCycle = 12, 16, 64
	if regenPhase != wantPhase || manaRegenCycle != wantManaCycle || healthRegenCycle != wantHealthCycle {
		t.Fatalf("phase %d, mana cycle %d, health cycle %d — want %d, %d, %d",
			regenPhase, manaRegenCycle, healthRegenCycle, wantPhase, wantManaCycle, wantHealthCycle)
	}
}

// stepToPhase steps w one sub-tick at a time until the sub-tick it is about
// to process is the first one at or after w.Tick() whose value is congruent
// to regenPhase modulo cycle, and returns after that qualifying Step call.
func stepToPhase(t *testing.T, w *World, cycle uint64) {
	t.Helper()
	for w.Tick()%cycle != regenPhase {
		Step(w, nil)
	}
	Step(w, nil)
}

// TestAQualifyingTickAddsExactlyTheDecodedHundredths is AC-1: the exact
// hundredths each arm's formula computes on its first qualifying tick, and
// nothing at all before it. MaxMana 50 at period 200 gives a mana gain of
// 50*1*100*1/200 = 25; MaxHP 50 at period 400 gives a health gain of
// 50*2*100*1/400 = 25 — chosen so both arms move by the same amount on the
// same tick (tick 12 qualifies for both at once) and a mixed-up arm would
// still be caught by which field moved.
func TestAQualifyingTickAddsExactlyTheDecodedHundredths(t *testing.T) {
	t.Parallel()
	e := Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 50, HealthRegenPeriod: 400,
		Mana: 4, MaxMana: 50, ManaRegenPeriod: 200}
	w := rgWorld(t, e)

	for w.Tick()%manaRegenCycle != regenPhase {
		Step(w, nil)
		got := occEntity(t, w, 1)
		if got.HP != 10 || got.HealthHundredths != 0 || got.Mana != 4 || got.ManaHundredths != 0 {
			t.Fatalf("tick %d (non-qualifying): got %+v, want the pool untouched", w.Tick(), got)
		}
	}
	Step(w, nil) // the qualifying tick itself
	got := occEntity(t, w, 1)
	if got.HP != 10 || got.HealthHundredths != 25 {
		t.Errorf("health arm: got %d/%d hundredths, want 10 unmoved with 25 hundredths carried",
			got.HP, got.HealthHundredths)
	}
	if got.Mana != 4 || got.ManaHundredths != 25 {
		t.Errorf("mana arm: got %d/%d hundredths, want 4 unmoved with 25 hundredths carried",
			got.Mana, got.ManaHundredths)
	}
}

// TestTheHealthArmMovesOneFullTickInFourAndManaOnAllFour is AC-1's cadence
// half. Over one full health period — four mana-qualifying ticks — the mana
// arm's gain (25 hundredths a tick, MaxMana 50 at period 200) crosses a
// whole point exactly once, on the fourth, and the health arm (MaxHP 50 at
// period 25, gain 400 hundredths — four whole points) moves on the first of
// the four and stands still on the other three.
func TestTheHealthArmMovesOneFullTickInFourAndManaOnAllFour(t *testing.T) {
	t.Parallel()
	e := Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 50, HealthRegenPeriod: 25,
		Mana: 0, MaxMana: 50, ManaRegenPeriod: 200}
	w := rgWorld(t, e)

	healthMoves, manaMoves := 0, 0
	prev := occEntity(t, w, 1)
	for k := 0; k < 4; k++ {
		stepToPhase(t, w, manaRegenCycle)
		got := occEntity(t, w, 1)
		if got.HP != prev.HP {
			healthMoves++
		}
		if got.Mana != prev.Mana {
			manaMoves++
		}
		prev = got
	}
	if healthMoves != 1 {
		t.Errorf("the health arm moved %d of 4 qualifying mana ticks, want 1 in 4", healthMoves)
	}
	if manaMoves != 1 {
		// The mana pool crosses a whole point only once in four (100/25 =
		// 4 ticks of 25 hundredths each) — asserted so this case is not
		// mistaken for a mana arm that also gates on the four-tick filter.
		t.Errorf("the mana pool crossed a whole point %d of 4 times, want exactly 1", manaMoves)
	}
	if got := occEntity(t, w, 1); got.Mana != 1 || got.ManaHundredths != 0 {
		t.Errorf("after four qualifying ticks mana is %d/%d, want 1 with no remainder", got.Mana, got.ManaHundredths)
	}
}

// TestAGainUnderOnePointStillReachesTheMaximum is AC-2, transcribed: a unit
// of maximum 45 at a period of 100 gains 90 hundredths per qualifying tick
// (45*2*100*1/100), starts at 40 with an empty remainder, and takes its first
// whole point on the second qualifying tick. The third and fourth use the
// idle rate3, adding270 hundredths each and capping at45 on the fourth. Later
// full-pool ticks preserve the remainder from the capping store.
func TestAGainUnderOnePointStillReachesTheMaximum(t *testing.T) {
	t.Parallel()
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 40, MaxHP: 45, HealthRegenPeriod: 100})

	// Ticks12/76 use rate1; ticks140 onward follow more than80 idle ticks.
	wantHP := []int32{40, 41, 44, 45, 45, 45}
	wantRest := []uint8{90, 80, 50, 20, 20, 20}
	for k := 0; k < 6; k++ {
		stepToPhase(t, w, healthRegenCycle)
		got := occEntity(t, w, 1)
		if got.HP != wantHP[k] || got.HealthHundredths != wantRest[k] {
			t.Fatalf("qualifying tick %d: got %d/%d, want %d/%d",
				k+1, got.HP, got.HealthHundredths, wantHP[k], wantRest[k])
		}
	}

	// And it does not overshoot: one qualifying tick further, at the maximum
	// already, gains nothing more (AC-3, over this same unit).
	stepToPhase(t, w, healthRegenCycle)
	if got := occEntity(t, w, 1); got.HP != 45 || got.HealthHundredths != 20 {
		t.Errorf("a tick past the maximum: got %d/%d, want 45/20 unmoved", got.HP, got.HealthHundredths)
	}
}

// TestNeitherPoolExceedsItsMaximum is AC-3's cap, on both arms, from a gain
// that overshoots the maximum in a single qualifying tick: MaxMana 10 at
// period 1 gains 1000 hundredths (10 whole points) a tick, and MaxHP 10 at
// period 1 gains 2000. Both pools start one point short of full.
func TestNeitherPoolExceedsItsMaximum(t *testing.T) {
	t.Parallel()
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 9, MaxHP: 10, HealthRegenPeriod: 1,
		Mana: 9, MaxMana: 10, ManaRegenPeriod: 1})
	stepToPhase(t, w, healthRegenCycle) // qualifies for both arms
	got := occEntity(t, w, 1)
	if got.HP != 10 {
		t.Errorf("health arm: got %d, want capped at 10", got.HP)
	}
	if got.Mana != 10 {
		t.Errorf("mana arm: got %d, want capped at 10", got.Mana)
	}
	// And staying there over further qualifying ticks (AC-3 "at any period,
	// from any starting value" — the arrival value included).
	for k := 0; k < 3; k++ {
		stepToPhase(t, w, healthRegenCycle)
	}
	got = occEntity(t, w, 1)
	if got.HP != 10 || got.Mana != 10 {
		t.Errorf("after further qualifying ticks: %d hp / %d mana, want both to stay at 10", got.HP, got.Mana)
	}
}

// TestANonPositivePeriodRegeneratesNothingOnItsOwnArm is AC-4: a period of
// zero and a negative period, on each arm in turn, leaving the OTHER arm
// working — so a period gate that accidentally closed both arms together is
// caught by the arm that should have kept moving.
func TestANonPositivePeriodRegeneratesNothingOnItsOwnArm(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                     string
		healthPeriod, manaPeriod int32
	}{
		{"health period zero", 0, 20},
		{"health period negative", -3, 20},
		{"mana period zero", 20, 0},
		{"mana period negative", 20, -7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 100, HealthRegenPeriod: tc.healthPeriod,
				Mana: 10, MaxMana: 100, ManaRegenPeriod: tc.manaPeriod})
			for k := 0; k < 5; k++ {
				stepToPhase(t, w, healthRegenCycle)
			}
			got := occEntity(t, w, 1)
			healthMoved := got.HP != 10 || got.HealthHundredths != 0
			manaMoved := got.Mana != 10 || got.ManaHundredths != 0
			wantHealthMoves := tc.healthPeriod > 0
			wantManaMoves := tc.manaPeriod > 0
			if healthMoved != wantHealthMoves {
				t.Errorf("health period %d: moved %v, want %v (got %d/%d)",
					tc.healthPeriod, healthMoved, wantHealthMoves, got.HP, got.HealthHundredths)
			}
			if manaMoved != wantManaMoves {
				t.Errorf("mana period %d: moved %v, want %v (got %d/%d)",
					tc.manaPeriod, manaMoved, wantManaMoves, got.Mana, got.ManaHundredths)
			}
		})
	}
}

// TestANotAliveEntityGainsNothing is AC-5: a dead and a downed entity, each
// carrying periods that would otherwise regenerate briskly, unchanged after
// many qualifying ticks. DyingTime is set far past this test's tick count so
// the decay ladder's own walk — a different mechanism, reached through the
// same HP field — never starts and cannot be mistaken for a regen move.
func TestANotAliveEntityGainsNothing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		hp, maxHP int32
	}{
		{"dead", -50, 100},
		{"downed", 0, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Entity{ID: 1, X: 1, Y: 1, HP: tc.hp, MaxHP: tc.maxHP, DyingTime: 10000,
				HealthRegenPeriod: 1, Mana: 20, MaxMana: 100, ManaRegenPeriod: 1}
			w := rgWorld(t, e)
			if got := occEntity(t, w, 1); got.Alive() {
				t.Fatalf("fixture: entity %q is alive, want not alive", tc.name)
			}
			for k := 0; k < 300; k++ {
				Step(w, nil)
			}
			got := occEntity(t, w, 1)
			if got.HP != tc.hp || got.HealthHundredths != 0 {
				t.Errorf("%s: HP %d/%d hundredths after 300 ticks, want %d/0 unchanged",
					tc.name, got.HP, got.HealthHundredths, tc.hp)
			}
			if got.Mana != 20 || got.ManaHundredths != 0 {
				t.Errorf("%s: mana %d/%d hundredths after 300 ticks, want 20/0 unchanged",
					tc.name, got.Mana, got.ManaHundredths)
			}
		})
	}
}

func TestANegativeMaximumRegeneratesNothingOnEitherArm(t *testing.T) {
	t.Parallel()

	t.Run("mana, alive entity, negative max, pool below it", func(t *testing.T) {
		w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 50, MaxHP: 100,
			Mana: -25, MaxMana: -20, ManaRegenPeriod: 3})
		if got := occEntity(t, w, 1); got.Mana >= got.MaxMana {
			t.Fatalf("fixture: mana %d is not below its maximum %d", got.Mana, got.MaxMana)
		}
		for k := 0; k < 10; k++ {
			stepToPhase(t, w, manaRegenCycle)
		}
		if got := occEntity(t, w, 1); got.Mana != -25 || got.ManaHundredths != 0 {
			t.Errorf("mana %d/%d hundredths after ten qualifying ticks against a maximum of -20, "+
				"want -25/0 unchanged", got.Mana, got.ManaHundredths)
		}
	})

	// The health case: the ONLY way an entity can be Alive() with a
	// non-positive MaxHP is HP exactly 0 (Downed() also requires MaxHP > 0,
	// so it does not fire) — the {0,-3,lifeAlive} row health_test.go already
	// carries. HP 0 is never below a non-positive maximum, so this case is
	// masked by BOTH Alive() and *cur >= max; it stays here as coverage of
	// the outcome, not as the gate's witness — the mana case above is that.
	t.Run("health, alive entity, negative max", func(t *testing.T) {
		w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 0, MaxHP: -15, HealthRegenPeriod: 4})
		if got := occEntity(t, w, 1); !got.Alive() {
			t.Fatalf("fixture: HP 0 against MaxHP -15 is not alive, want it to be")
		}
		for k := 0; k < 10; k++ {
			stepToPhase(t, w, healthRegenCycle)
		}
		if got := occEntity(t, w, 1); got.HP != 0 || got.HealthHundredths != 0 {
			t.Errorf("HP %d/%d hundredths after ten qualifying ticks against a maximum of -15, "+
				"want 0/0 unchanged", got.HP, got.HealthHundredths)
		}
	})
}

func TestAWorldWithNoPeriodMovesNoPinnedDigest(t *testing.T) {
	t.Parallel()
	ents := []Entity{{ID: 1, X: 2, Y: 3, HP: 37, MaxHP: 80,
		Mana: 5, MaxMana: 12, HealthHundredths: 44, ManaHundredths: 9}}
	w := rgWorld(t, ents...)
	ref := rgWorld(t, ents...)

	const n = 2*healthRegenCycle + 1
	for k := 0; k < n; k++ {
		Step(w, nil)
	}
	ref.tick = uint64(n)

	if got, want := w.Hash(), ref.Hash(); got != want {
		t.Errorf("after %d idle ticks with no period the world hashes %#016x, "+
			"want %#016x — the pass moved a digest a period-less world pins", n, got, want)
	}
	stepped, still := occEntity(t, w, 1), occEntity(t, ref, 1)
	if stepped != still {
		t.Errorf("the entity came out at %+v, want it exactly as constructed: %+v", stepped, still)
	}
}

func TestRegenerationDrawsNoRandomness(t *testing.T) {
	t.Parallel()
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 100, HealthRegenPeriod: 3,
		Mana: 0, MaxMana: 100, ManaRegenPeriod: 5})
	before := w.rng.state
	for k := 0; k < 300; k++ {
		Step(w, nil)
	}
	if got := w.rng.state; got != before {
		t.Errorf("the generator's state moved from %#016x to %#016x over 300 regenerating ticks, "+
			"want it unmoved — the pass reads no random source (P-1)", before, got)
	}
}

func TestACappingTickStillStoresItsRemainder(t *testing.T) {
	t.Parallel()
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20,
		Mana: 44, MaxMana: 45, ManaRegenPeriod: 20, ManaHundredths: 7})
	stepToPhase(t, w, manaRegenCycle)

	got := occEntity(t, w, 1)
	if got.Mana != 45 {
		t.Errorf("mana %d after the capping tick, want it held at its maximum of 45", got.Mana)
	}
	if got.ManaHundredths != 32 {
		t.Errorf("the remainder is %d after a capping tick, want 32 — the fraction this tick "+
			"computed, not the 7 it started with", got.ManaHundredths)
	}
}

func TestANegativePoolLeavesALegalRemainder(t *testing.T) {
	t.Parallel()
	w := rgWorld(t, Entity{ID: 1, X: 1, Y: 1, HP: 20, MaxHP: 20,
		Mana: -5, MaxMana: 45, ManaRegenPeriod: 100})
	stepToPhase(t, w, manaRegenCycle)

	got := occEntity(t, w, 1)
	if got.ManaHundredths > 99 {
		t.Errorf("the remainder is %d after a tick on a negative pool, want a value the byte and "+
			"the decoder can both hold (0..99)", got.ManaHundredths)
	}
	if got.Mana != -5 || got.ManaHundredths != 45 {
		t.Errorf("mana %d remainder %d, want -5 and 45 — the accumulator's own value, taken so "+
			"that the quotient and the remainder come from one adjustment", got.Mana, got.ManaHundredths)
	}

	// The statement that matters: the world still crosses its own form. A
	// remainder of 201 is refused by regenFault on decode, so this round trip
	// is the assertion that the pass cannot reach a state the decoder rejects.
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatalf("the stepped world does not survive its own round trip: %v", err)
	}
}

// An actor a script took off the map is not visited by the pass: neither pool
// moves while it is off, and both resume once it returns.
func TestAnOffMapEntityRegeneratesNothingUntilItReturns(t *testing.T) {
	t.Parallel()
	e := Entity{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 50, HealthRegenPeriod: 400,
		Mana: 4, MaxMana: 50, ManaRegenPeriod: 200}
	on := e
	on.ID, on.X = 2, 3
	w := rgWorld(t, e, on)
	w.entities[0].OffMap = true
	for range 256 {
		Step(w, nil)
	}
	if got := occEntity(t, w, 1); got.HP != 10 || got.Mana != 4 || got.HealthHundredths != 0 || got.ManaHundredths != 0 {
		t.Fatalf("off-map entity regenerated: %+v", got)
	}
	if got := occEntity(t, w, 2); got.HP == 10 || got.Mana == 4 {
		t.Fatalf("control entity on the map did not regenerate: %+v", got)
	}
	w.entities[0].OffMap = false
	for range 256 {
		Step(w, nil)
	}
	if got := occEntity(t, w, 1); got.HP == 10 || got.Mana == 4 {
		t.Fatalf("returned entity did not resume regeneration: %+v", got)
	}
}
