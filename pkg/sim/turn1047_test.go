package sim

import (
	"bytes"
	"testing"
)

func turnActor(id EntityID, x, y int32, facing uint8, rate int32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, Facing: facing,
		DesiredFacing: facing, RotationSpeed: rate, Reach: 1, ActorState: actorStateGuard}
}

func TestTurnDurationUsesTheShortestByteArc(t *testing.T) {
	for _, tc := range []struct {
		name       string
		from, to   uint8
		rate       int32
		wantArc    int32
		wantRemain uint8
		wantFacing uint8
	}{
		{"wrap clockwise", 224, 0, 16, 32, 1, 0},
		{"wrap counter-clockwise", 0, 224, 16, 32, 1, 224},
		{"two directions", 0, 64, 24, 64, 3, 0},
		{"half circle", 64, 192, 16, 128, 8, 64},
		{"very large positive rate", 64, 192, 2147483647, 128, 1, 64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := facingArc(tc.from, tc.to); got != tc.wantArc {
				t.Fatalf("arc = %d, want %d", got, tc.wantArc)
			}
			e := turnActor(1, 1, 1, tc.from, tc.rate)
			wait := e.requestFacing(tc.to)
			if !wait || e.TurnRemaining != tc.wantRemain || e.TurnTotal != tc.wantRemain || e.Facing != tc.wantFacing || e.DesiredFacing != tc.to {
				t.Errorf("turn = facing %d desired %d remaining/total %d/%d wait %t, want %d/%d/%d/%d/true",
					e.Facing, e.DesiredFacing, e.TurnRemaining, e.TurnTotal, wait, tc.wantFacing, tc.to, tc.wantRemain, tc.wantRemain)
			}
		})
	}

	for _, rate := range []int32{0, -1} {
		e := turnActor(1, 1, 1, 0, rate)
		wait := e.requestFacing(128)
		if wait || e.Facing != 128 || e.DesiredFacing != 128 || e.Turning() {
			t.Errorf("rate %d compatibility turn = %+v, wait=%t", rate, e, wait)
		}
	}
}

func TestMovementWaitsForItsTurnAndRetainsRoutes(t *testing.T) {
	b := Bounds{Width: 7, Height: 7}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{
		turnActor(1, 3, 3, facingOfDir(2), 16),
	})
	Step(w, []Command{{Entity: 1, X: 1, Y: 3}})
	e := w.entities[0]
	if e.X != 3 || e.Y != 3 || e.Facing != facingOfDir(2) || e.DesiredFacing != facingOfDir(6) || e.TurnRemaining != 8 {
		t.Fatalf("initial reverse = %+v, want stationary east-to-west turn with 8 ticks", e)
	}
	if len(w.routes[0]) == 0 {
		t.Fatal("large turn discarded its admitted route")
	}
	for remaining := uint8(7); remaining > 0; remaining-- {
		Step(w, nil)
		e = w.entities[0]
		if e.X != 3 || e.Y != 3 || e.TurnRemaining != remaining {
			t.Fatalf("remaining %d: actor = %+v", remaining, e)
		}
	}
	Step(w, nil)
	e = w.entities[0]
	if e.X != 2 || e.Y != 3 || e.Turning() || e.Facing != facingOfDir(6) || e.DesiredFacing != facingOfDir(6) {
		t.Errorf("completed reverse = %+v, want first westward step", e)
	}
}

func TestTurnReplacementRestartsButARepeatedOrderPreservesProgress(t *testing.T) {
	b := Bounds{Width: 9, Height: 9}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{
		turnActor(1, 4, 4, facingOfDir(2), 16),
	})
	west := Command{Entity: 1, X: 2, Y: 4}
	Step(w, []Command{west})
	Step(w, []Command{west})
	if got := w.entities[0].TurnRemaining; got != 7 {
		t.Fatalf("repeated destination restarted turn at %d, want 7", got)
	}
	Step(w, []Command{{Entity: 1, X: 4, Y: 2}})
	e := w.entities[0]
	if e.DesiredFacing != facingOfDir(0) || e.TurnRemaining != 4 || e.Facing != facingOfDir(2) {
		t.Errorf("replacement destination left %+v, want a fresh four-tick north turn", e)
	}
}

func TestAttackWaitsUntilTheAttackerFacesItsVictim(t *testing.T) {
	a := turnActor(1, 4, 4, facingOfDir(6), 32)
	a.AttackCharge, a.AttackRelax, a.AlwaysHits, a.DamageBase = 1, 0, true, 7
	v := turnActor(2, 5, 4, facingOfDir(0), 0)
	w := mustWorld(t, 1, Bounds{Width: 10, Height: 10}, []Entity{a, v})
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	if got := w.entities[0]; got.TurnRemaining != 4 || got.AttackPhase != AttackReady {
		t.Fatalf("attack admission = %+v, want four-tick turn and ready cycle", got)
	}
	for n := 0; n < 3; n++ {
		Step(w, nil)
		if w.entities[1].HP != 100 || w.entities[0].AttackPhase != AttackReady {
			t.Fatalf("tick %d resolved attack during turn: attacker=%+v victim=%+v", n, w.entities[0], w.entities[1])
		}
	}
	Step(w, nil)
	if got := w.entities[1].HP; got != 93 {
		t.Errorf("first faced attack left victim at %d, want 93", got)
	}
	if got := w.entities[0]; got.Turning() || got.Facing != facingOfDir(2) {
		t.Errorf("attacker struck before settling east: %+v", got)
	}
}

func TestWithdrawalTurnsAwayBeforeMovingAndTurnsBackBeforeAttacking(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 10)
	self.Withdraw, self.Facing, self.DesiredFacing, self.RotationSpeed = 10, facingOfDir(2), facingOfDir(2), 32
	hostile := withdrawalFighter(2, 3, 21, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), self, hostile)
	w.withdrawalPass()

	// The retreat is west, opposite the hostile and the actor's original east
	// facing. Movement admission owns the first physical reverse.
	Step(w, nil)
	if got := w.entities[0]; got.X != 20 || got.DesiredFacing != facingOfDir(6) || got.TurnRemaining != 4 {
		t.Fatalf("retreat did not begin with a west turn: %+v", got)
	}
	for w.entities[0].Turning() {
		Step(w, nil)
	}
	if w.entities[0].X >= 20 {
		t.Fatalf("retreater did not move after facing west: %+v", w.entities[0])
	}

	// Re-engaging the eastern hostile replaces the retreat. Approach owns the
	// second physical reverse and the attack cycle remains ready until it ends.
	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})
	got := w.entities[0]
	if got.DesiredFacing != facingOfDir(2) || !got.Turning() || got.AttackPhase != AttackReady {
		t.Fatalf("re-engage did not begin with an east turn: %+v", got)
	}
}

func TestBookWindupDoesNotAdvanceWhileTheCasterTurns(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 8,
		DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 4, 4, 40, 100, 100, 1<<1)
	caster.Facing, caster.DesiredFacing, caster.RotationSpeed, caster.AttackCharge = facingOfDir(6), facingOfDir(6), 32, 4
	target := spEnt(2, 5, 4)
	w := spWorld(t, 1, []SpellRule{spell}, caster, target)
	Step(w, []Command{spCast(1, 2, 1)})
	// castWindupTicks floors at castPeriod (8): the caster's AttackCharge of 4
	// is below that floor, so wind-up starts at 8 while the turn starts at the
	// independently derived 4-tick arc.
	if len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != 8 || w.entities[0].TurnRemaining != 4 {
		t.Fatalf("admitted cast = entity %+v cast %+v", w.entities[0], w.bookCasts)
	}
	for n := 0; n < 3; n++ {
		Step(w, nil)
		if w.bookCasts[0].Remaining != 8 {
			t.Fatalf("turn tick %d advanced wind-up to %d", n, w.bookCasts[0].Remaining)
		}
	}
	Step(w, nil)
	if w.entities[0].Turning() || w.bookCasts[0].Remaining != 7 {
		t.Errorf("first faced cast tick = entity %+v cast %+v", w.entities[0], w.bookCasts[0])
	}
}

func TestActiveTurnRoundTripsAndChangesTheDigest(t *testing.T) {
	active := turnActor(1, 2, 2, facingOfDir(0), 16)
	active.DesiredFacing, active.TurnRemaining, active.TurnTotal = facingOfDir(4), 8, 8
	w := mustWorld(t, 9, Bounds{Width: 6, Height: 6}, []Entity{active})
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if got := back.Entities()[0]; got.DesiredFacing != active.DesiredFacing || got.TurnRemaining != active.TurnRemaining || got.TurnTotal != active.TurnTotal {
		t.Errorf("active turn came back as %+v", got)
	}
	settled := append([]byte(nil), form...)
	// Entity 0's record starts after the header and the three always-materialised
	// planes (binary.go's encode, and binary_test.go's own
	// strippedWorldOfTurnProgress uses the same base).
	recordBase := headerLen + 3*int(gridCells(w.bounds))
	settled[recordBase+291], settled[recordBase+292], settled[recordBase+293] = settled[recordBase+91], 0, 0
	var other World
	if err := other.UnmarshalBinary(settled); err != nil {
		t.Fatalf("settled UnmarshalBinary: %v", err)
	}
	if bytes.Equal(form, settled) || w.Hash() == other.Hash() {
		t.Error("active and settled turn states have the same form or digest")
	}
}

// TestInactiveResidueIsNormalisedByTheConstructorAndRefusedByTheDecoder covers
// the "inactive mismatch" shape by itself: NewWorld normalises a caller-built
// entity with no active turn to the current facing (world.go's own "a caller
// that supplies no active turn supplies no second facing either" rule, on the
// Reach/HealthHundredths precedent beside it), so it cannot be the boundary
// that refuses this shape. The byte form is: turnFault runs there strictly,
// and a saved record with zero remaining and a mismatched desired facing is
// residue no producer in this tree can write.
func TestInactiveResidueIsNormalisedByTheConstructorAndRefusedByTheDecoder(t *testing.T) {
	e := turnActor(1, 1, 1, 0, 16)
	e.DesiredFacing = 32
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if got := w.Entities()[0]; got.DesiredFacing != got.Facing {
		t.Fatalf("constructor kept mismatched residue = %+v", got)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	recordBase := headerLen + 3*int(gridCells(w.bounds))
	form[recordBase+291] = 32 // DesiredFacing, with TurnRemaining left at 0
	var back World
	if err := back.UnmarshalBinary(form); err == nil {
		t.Fatal("UnmarshalBinary accepted inactive residue")
	}
}

func TestTurnFaultRejectsStatesNoProducerCanMake(t *testing.T) {
	base := turnActor(1, 1, 1, 0, 16)
	for _, tc := range []struct {
		name string
		edit func(*Entity)
	}{
		{"nonpositive active rate", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal, e.RotationSpeed = 32, 1, 1, 0 }},
		{"missing total", func(e *Entity) { e.DesiredFacing, e.TurnRemaining = 32, 1 }},
		{"remainder above total", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 32, 2, 1 }},
		{"non-direction desired", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 31, 1, 1 }},
		{"overlong remainder", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 128, 129, 129 }},
		{"equal long turn", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 0, 2, 2 }},
		{"equal facing with stale total", func(e *Entity) { e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 0, 1, 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := base
			tc.edit(&e)
			if _, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{e}); err == nil {
				t.Fatal("NewWorld accepted malformed turn")
			}
		})
	}
}

// TestARaisedGhostInheritsAnInactiveTurnAtItsSourcesFacing is a regression
// test for two production defects found on the same composite literal,
// raisedGhost (pkg/sim/spell.go), across two rounds of this story.
//
// Round 1: the literal set Facing from the corpse but left DesiredFacing at
// its uint8 zero value, building a malformed inactive-turn residue (contract
// point 2 — desired facing must equal current facing whenever TurnRemaining
// is 0) for any corpse whose Facing is not already 0. TestControlSpiritCon-
// sumesBonesAndCreatesANewOwnedGhost does not catch this because its corpse
// fixture carries the default zero facing, the one value at which the defect
// is invisible.
//
// Round 3 (adversarial pass 2, P1): sim.GhostTemplate carried no
// RotationSpeed field at all, so pkg/mapload's ghostTemplate could not read
// the shipped Ghost row's own column and the literal named none, leaving
// every raised ghost at RotationSpeed 0 — the pkg/sim/facing.go compatibility
// arm, on shipped content. This test now asserts all four turn fields the
// literal names, so a future field dropped from the same literal fails here
// rather than needing a fourth review to find.
func TestARaisedGhostInheritsAnInactiveTurnAtItsSourcesFacing(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	corpse := spEnt(2, 2, 1)
	corpse.Facing = 96
	w := hlGhostWorld(t, 32, []SpellRule{{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}}, hlGhostTemplate(), caster, corpse)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones

	spRunCast(w, Command{Kind: KindCast, Entity: 1, X: 2, Y: 25})
	if len(w.entities) != 2 {
		t.Fatalf("control spirit did not raise a ghost: %+v", w.entities)
	}
	got := w.entities[1]
	if got.Facing != 96 || got.DesiredFacing != 96 || got.TurnRemaining != 0 || got.TurnTotal != 0 {
		t.Fatalf("raised ghost turn state = facing %d, desired %d, remaining/total %d/%d; want 96, 96, 0/0",
			got.Facing, got.DesiredFacing, got.TurnRemaining, got.TurnTotal)
	}
	if got.RotationSpeed != 16 {
		t.Fatalf("raised ghost RotationSpeed = %d; want 16 (hlGhostTemplate's value, carried from the template's own field)",
			got.RotationSpeed)
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatalf("a ghost raised from a facing corpse is not persistent: %v", err)
	}
}

// TestDeathDuringAnActiveTurnClearsItWithoutCompletingIt is round 2's own
// fixture for remaining-surface item 9 (return-brief section 7): a valid
// death mid-turn, through the same KindKill production path corpseloot_test.go
// and combat_test.go use, rather than the invalid HP=-5-at-construction probe
// pass 1's review reported (HP<0 makes Dead() true, but clearFelled only runs
// from the production sites that observe a fresh transition — a fixture that
// starts already dead reaches none of them).
//
// The actor is given an eight-tick reversal (an east start, a west order, rate
// 16: shortest arc 128, ceil(128/16) = 8) and is killed after the turn is
// under way but before it completes. clearFelled calls clearTurn
// (step.go:1203, facing.go:140), which sets DesiredFacing to the CURRENT
// Facing and zeroes TurnRemaining — it does not advance Facing to the desired
// direction the turn was heading toward. A corpse therefore keeps the facing
// it held at the moment of death, not the direction it never finished turning
// to.
func TestDeathDuringAnActiveTurnClearsItWithoutCompletingIt(t *testing.T) {
	b := Bounds{Width: 7, Height: 7}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{
		turnActor(1, 3, 3, facingOfDir(2), 16),
	})
	Step(w, []Command{{Entity: 1, X: 1, Y: 3}})
	if e := w.entities[0]; e.Facing != facingOfDir(2) || e.DesiredFacing != facingOfDir(6) || e.TurnRemaining != 8 {
		t.Fatalf("setup: turn did not start as expected: %+v", e)
	}
	Step(w, nil)
	Step(w, nil)
	if e := w.entities[0]; !e.Turning() || e.TurnRemaining != 6 {
		t.Fatalf("setup: turn did not stay open two ticks in: %+v", e)
	}

	Step(w, []Command{{Kind: KindKill, Entity: 1}})

	e := w.entities[0]
	if e.Alive() {
		t.Fatalf("KindKill left the actor alive: %+v", e)
	}
	if e.Turning() {
		t.Fatalf("death left an open turn: %+v", e)
	}
	if e.TurnRemaining != 0 {
		t.Fatalf("death left TurnRemaining = %d, want 0", e.TurnRemaining)
	}
	if e.TurnTotal != 0 {
		t.Fatalf("death left TurnTotal = %d, want 0", e.TurnTotal)
	}
	if e.Facing != facingOfDir(2) {
		t.Fatalf("death changed Facing to %d, want the pre-death facing %d (a corpse does not finish its turn)",
			e.Facing, facingOfDir(2))
	}
	if e.DesiredFacing != e.Facing {
		t.Fatalf("death left DesiredFacing %d != Facing %d, which turnFault refuses on a non-turning actor",
			e.DesiredFacing, e.Facing)
	}
	if err := turnFault(e); err != nil {
		t.Fatalf("the felled actor fails its own turn invariant: %v", err)
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatalf("a felled actor mid-turn is not persistent: %v", err)
	}
}
