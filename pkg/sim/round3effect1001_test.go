package sim

import "testing"

// R3-A1 fix witnesses (docs/1001-spell-effects/round3-review.md, docs/1001-
// spell-effects/round3-effect.md). round3-review.md's own measurement, on
// Haste (+7), Slow (-7) and Freezing Cloud's inner effect (-7) — the shipped
// magnitudes at power 100 — standing together on a base-10 actor: expiry in
// duration order left the actor at 15, and SetDerived (pkg/game/rearm.go's
// recomputeRaisedSkills, called on every skill level that rises) carried no
// floor at all and could hand a negative Entity.Speed to moverSpeed/rated
// (world.go) and groupMinSpeed (group.go).

// TestSpeedEffectsReverseExactlyAcrossOverlappingClamps is R3-A1's own
// scenario, expired in both duration orders.
//
// TO CONFIRM "positive expires first" WITNESSES THE FIX, undo the
// redistribution removeAttachedAt performs (pkg/sim/effect.go) — go back to
// discarding applyEffectDelta's return — and rerun: the actor ends at 15,
// not the base 10 ("speed after every effect expired = 15, want the base
// 10").
func TestSpeedEffectsReverseExactlyAcrossOverlappingClamps(t *testing.T) {
	const base = 10
	for _, tc := range []struct {
		name                         string
		hasteDur, slowDur, freezeDur uint16
	}{
		// Haste — the only positive term — expires first, while Slow and
		// Freezing Cloud together already hold the actor at the floor. This
		// is the order round3-review.md measured broken.
		{"positive expires first", 10, 40, 70},
		// Both negative terms expire before Haste, so no removal here ever
		// needs the floor: each one removes into a state no lower than the
		// base alone. Kept as a control that the fix does not regress the
		// order that was never broken.
		{"positive expires last", 70, 10, 25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			victim := spEnt(2, 3, 2)
			victim.Speed, victim.TokenSize = base, 1
			w := hlWorld(t, 0x1101, Relations{}, nil, effectMage(1, 2, 2, 0), victim)

			w.attachEffect(2, 1, SpellRule{ID: 24}, EffectSpeed, +7, tc.hasteDur, EffectDuration)
			w.attachEffect(2, 1, SpellRule{ID: 28}, EffectSpeed, -7, tc.slowDur, EffectDuration)
			w.attachEffect(2, 1, SpellRule{ID: 7}, EffectSpeed, -7, tc.freezeDur, EffectDuration)
			if got := spAt(t, w, 2).Speed; got != 3 {
				t.Fatalf("all three attached: speed %d, want 3 — the setup itself changed", got)
			}

			for i := 0; i < 200 && len(w.attached) > 0; i++ {
				Step(w, nil)
			}
			if len(w.attached) != 0 {
				t.Fatalf("%d effects still attached after 200 ticks", len(w.attached))
			}
			if got := spAt(t, w, 2).Speed; got != base {
				t.Errorf("speed after every effect expired = %d, want the base %d", got, base)
			}
		})
	}
}

// TestSetDerivedRecomputesExactlyAfterAClampedRemoval goes past "does it
// floor" (round3-review.md's own SetDerived probe only checked that) to "is
// the number right": once redistributeShortfall has moved a removal's
// discarded remainder onto a still-active sibling, a fresh base combined
// with that sibling's own effectDelta must land on the exact value the
// invariant predicts — base plus the active total, clamped once.
func TestSetDerivedRecomputesExactlyAfterAClampedRemoval(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.Speed, victim.TokenSize = 10, 1
	w := hlWorld(t, 0x1102, Relations{}, nil, effectMage(1, 2, 2, 0), victim)

	w.attachEffect(2, 1, SpellRule{ID: 24}, EffectSpeed, +7, 5, EffectDuration)    // Haste, short
	w.attachEffect(2, 1, SpellRule{ID: 28}, EffectSpeed, -7, 5000, EffectDuration) // Slow
	w.attachEffect(2, 1, SpellRule{ID: 7}, EffectSpeed, -7, 5000, EffectDuration)  // Freezing Cloud
	for i := 0; i < 8; i++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 2).Speed; got != 1 {
		t.Fatalf("after Haste expired speed is %d, want the floor 1 — the setup itself changed", got)
	}
	delta := w.effectDelta(2, EffectSpeed)
	if delta != -9 {
		t.Fatalf("the active total is %d, want -9 (Slow's redistributed -2 plus Freezing Cloud's -7)"+
			" — the setup itself changed", delta)
	}

	if !w.SetDerived(2, DerivedBlock{MaxHP: 100, Speed: 20, ScanRange: 8,
		Reaction: 10, Mind: 10, Spirit: 10, Combat: CombatBlock{XPSlot: 0}}) {
		t.Fatal("SetDerived refused")
	}
	if got, want := spAt(t, w, 2).Speed, int32(20)+delta; got != want {
		t.Errorf("speed after SetDerived is %d, want the fresh base plus the active total, %d", got, want)
	}
}

// TestSetDerivedFloorsWhenTheFreshBaseStillCannotClearIt is R3-A1's own
// "no floor at all" half of the finding: SetDerived must floor exactly like
// every other writer of Entity.Speed.
//
// TO CONFIRM IT WITNESSES THE FIX, remove the `if speed < minEffectSpeed`
// block around SetDerived's Speed line (pkg/sim/rearm.go) and rerun: speed
// goes to -4 and rated reports the actor unrated — moverSpeed's own doc
// (world.go) names that the FASTEST cadence the engine has.
func TestSetDerivedFloorsWhenTheFreshBaseStillCannotClearIt(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.Speed, victim.TokenSize = 10, 1
	w := hlWorld(t, 0x1103, Relations{}, nil, effectMage(1, 2, 2, 0), victim)

	w.attachEffect(2, 1, SpellRule{ID: 24}, EffectSpeed, +7, 5, EffectDuration)
	w.attachEffect(2, 1, SpellRule{ID: 28}, EffectSpeed, -7, 5000, EffectDuration)
	w.attachEffect(2, 1, SpellRule{ID: 7}, EffectSpeed, -7, 5000, EffectDuration)
	for i := 0; i < 8; i++ {
		Step(w, nil)
	}

	// Base 5 combined with the active total of -9 (see the recompute test
	// above) is -4: still short of the floor even off a fresh base.
	if !w.SetDerived(2, DerivedBlock{MaxHP: 100, Speed: 5, ScanRange: 8,
		Reaction: 10, Mind: 10, Spirit: 10, Combat: CombatBlock{XPSlot: 0}}) {
		t.Fatal("SetDerived refused")
	}
	e := spAt(t, w, 2)
	if e.Speed != minEffectSpeed {
		t.Fatalf("speed after SetDerived is %d, want the floor %d exactly", e.Speed, minEffectSpeed)
	}
	if !rated(e) {
		t.Error("SetDerived left the actor unrated")
	}
	if got := moverSpeed(e); got != minEffectSpeed {
		t.Errorf("moverSpeed reads %d, want the floor %d", got, minEffectSpeed)
	}
}

// TestGroupSpeedReadingDoesNotWrapAfterASpeedFloorMiss is R3-A1's
// groupMinSpeed symptom. TestTheGroupTermIsTheMinimumInTheOriginalsOwnWidths
// (formation_test.go) documents a negative Speed reading as its unsigned low
// byte (-1 -> 255) as an INTENTIONAL, decoded narrowing, reachable only by
// customisation past the shipped speed column. SetDerived handing
// groupMinSpeed a negative Speed from an ORDINARY effect stack is a
// different thing: an input that narrowing rule was never meant to see
// arriving anyway. This test's only claim is that the floor keeps it from
// arriving.
//
// TO CONFIRM IT WITNESSES THE FIX, remove the `if speed < minEffectSpeed`
// block in SetDerived (pkg/sim/rearm.go) and rerun: the floored member's
// speed becomes -4 and groupMinSpeed reports 252 (-4's wrapped low byte,
// round3-review.md's own measured number), not the floor's 1.
func TestGroupSpeedReadingDoesNotWrapAfterASpeedFloorMiss(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.Speed, victim.TokenSize = 10, 1
	w := hlWorld(t, 0x1104, Relations{}, nil, effectMage(1, 2, 2, 0), victim)

	w.attachEffect(2, 1, SpellRule{ID: 24}, EffectSpeed, +7, 5, EffectDuration)
	w.attachEffect(2, 1, SpellRule{ID: 28}, EffectSpeed, -7, 5000, EffectDuration)
	w.attachEffect(2, 1, SpellRule{ID: 7}, EffectSpeed, -7, 5000, EffectDuration)
	for i := 0; i < 8; i++ {
		Step(w, nil)
	}
	if !w.SetDerived(2, DerivedBlock{MaxHP: 100, Speed: 5, ScanRange: 8,
		Reaction: 10, Mind: 10, Spirit: 10, Combat: CombatBlock{XPSlot: 0}}) {
		t.Fatal("SetDerived refused")
	}

	fast := Entity{ID: 9, Speed: 50}
	slow := spAt(t, w, 2)
	if got := groupMinSpeed([]Entity{fast, slow}, []int{0, 1}); got != uint8(minEffectSpeed) {
		t.Errorf("groupMinSpeed reads %d, want the floored member's own %d"+
			" — 252 would be -4's wrapped low byte", got, minEffectSpeed)
	}
}

// TestProtectionSurvivesARecomputeWhileClampedThenExpiresToTheNewBase is
// round3-review.md's minor finding, named the same family as R3-A1: a
// Protection effect that clamps at attach, then a recompute — a level rise,
// an armour swap, anything that calls SetCombat — that lands while the
// effect is still active, then the effect's own expiry, must not leave the
// actor below whatever base that recompute established.
//
// TO CONFIRM IT WITNESSES THE FIX, remove the `if hasEffect { ... }`
// redistribution block in SetCombat's protection loop (pkg/sim/rearm.go) and
// rerun: the actor ends at 80, the ATTACH-TIME base, instead of the
// recompute's own 95.
func TestProtectionSurvivesARecomputeWhileClampedThenExpiresToTheNewBase(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.Protection[0] = 80
	w := hlWorld(t, 0x1105, Relations{}, nil, victim)

	if !w.attachEffect(2, 0, SpellRule{ID: 5}, EffectProtectionFire, 45, 4, EffectDuration) {
		t.Fatal("attach refused")
	}
	if got := spAt(t, w, 2).Protection[0]; got != 100 {
		t.Fatalf("fire protection after attach is %d, want the clamped 100 — the setup itself changed", got)
	}

	if !w.SetCombat(2, CombatBlock{Protection: [5]int32{95, 0, 0, 0, 0}}) {
		t.Fatal("SetCombat refused")
	}
	if got := spAt(t, w, 2).Protection[0]; got != 100 {
		t.Fatalf("fire protection after the recompute is %d, want the still-clamped 100"+
			" — the setup itself changed", got)
	}

	for i := 0; i < 10 && len(w.attached) > 0; i++ {
		Step(w, nil)
	}
	if len(w.attached) != 0 {
		t.Fatalf("%d effects still attached after 10 ticks", len(w.attached))
	}
	if got := spAt(t, w, 2).Protection[0]; got != 95 {
		t.Errorf("fire protection after expiry is %d, want the recompute's own base 95", got)
	}
}

// TestHealthEffectRemovalDoesNotCorruptASiblingsOwnMagnitude confirms the
// `e.Kind != EffectHealth` guard in removeAttachedAt's redistribution
// (effect.go): Health's own clamp is the MaxHP ceiling, reachable by
// ordinary healing that has nothing to do with any effect, so there is no
// effect-free base to redistribute a ceiling-clamped shortfall against the
// way there is for Speed, ScanRange and Protection. A ceiling-clamped health
// effect's removal must not adjust a DIFFERENT, unrelated health effect's
// own stored magnitude.
//
// TO CONFIRM IT WITNESSES THE FIX, drop the `&& e.Kind != EffectHealth`
// half of removeAttachedAt's guard (pkg/sim/effect.go) and rerun: B's own
// removal gives back 40 instead of its nominal 20, landing the actor at 190
// instead of 170.
func TestHealthEffectRemovalDoesNotCorruptASiblingsOwnMagnitude(t *testing.T) {
	victim := spEnt(2, 3, 2)
	victim.MaxHP = 200
	w := hlWorld(t, 0x1106, Relations{}, nil, victim)

	// A and B: two duration-mode damage effects, neither clamping on attach.
	if !w.attachEffect(2, 0, SpellRule{ID: 1}, EffectHealth, -30, 3, EffectDuration) {
		t.Fatal("attach A refused")
	}
	if !w.attachEffect(2, 0, SpellRule{ID: 2}, EffectHealth, -20, 6, EffectDuration) {
		t.Fatal("attach B refused")
	}

	// Ordinary combat healing, unrelated to either effect, pushes the actor
	// to the ceiling before A expires.
	i := indexOfEntity(w.entities, 2)
	if i < 0 {
		t.Fatal("world holds no entity 2")
	}
	w.entities[i].HP = 190

	for j := 0; j < 3; j++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 2).HP; got != 200 {
		t.Fatalf("HP after A's ceiling-clamped removal is %d, want the clamped 200"+
			" — the setup itself changed", got)
	}

	// More ordinary combat, unrelated to B, before B expires.
	i = indexOfEntity(w.entities, 2)
	w.entities[i].HP = 150

	for j := 0; j < 3; j++ {
		Step(w, nil)
	}
	if got := spAt(t, w, 2).HP; got != 170 {
		t.Errorf("HP after B's own removal is %d, want 170 (150 plus B's nominal 20)", got)
	}
}
