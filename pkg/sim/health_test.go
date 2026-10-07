package sim

// AC-1 and the constructor's half of the order rule: the three states read off
// two integers, and what a world does with a unit that is not alive and carries
// an order anyway.
//
// Nothing here touches a byte form, a digest or a step. Every expected state is
// written out by hand from the contract's own table — a case that asked the
// predicates what they said and then agreed with them would pass whatever they
// did.

import "testing"

// life is one of the three states as this file names them, so a case can state
// which one it expects and a failure can print it. It is the TEST's enum and
// deliberately not the package's: the contract fixes three predicates and no
// state value, and a production enum here would be a fourth thing to keep
// exhaustive.
type life int

const (
	lifeAlive life = iota
	lifeDowned
	lifeDead
)

func (l life) String() string {
	switch l {
	case lifeAlive:
		return "alive"
	case lifeDowned:
		return "downed"
	case lifeDead:
		return "dead"
	}
	return "no state"
}

// healthCases is AC-1's table, transcribed. Eight pairs: full health, a scratch,
// exactly zero with a health system and exactly zero without one, zero against a
// NEGATIVE maximum, and three below zero — one of them at a maximum of zero, so
// that "no health system" is shown to be killable, and one far below, where a
// later story's decay stages will read.
var healthCases = []struct {
	hp, maxHP int32
	want      life
}{
	{100, 100, lifeAlive},
	{1, 100, lifeAlive},
	{0, 100, lifeDowned},
	{0, 0, lifeAlive},
	{0, -3, lifeAlive},
	{-1, 100, lifeDead},
	{-1, 0, lifeDead},
	{-1000, 7, lifeDead},
}

// state is what the three predicates jointly say about e, and whether they said
// exactly one thing. It asks all three every time rather than stopping at the
// first true one, which is what lets the caller see two answers or none.
func state(e Entity) (life, int) {
	n, got := 0, lifeAlive
	if e.Alive() {
		n, got = n+1, lifeAlive
	}
	if e.Downed() {
		n, got = n+1, lifeDowned
	}
	if e.Dead() {
		n, got = n+1, lifeDead
	}
	return got, n
}

// TestEachPairIsExactlyOneOfTheThreeStates is AC-1: the eight pairs, the state
// each is in, and the count of predicates that held — so a pair answering twice
// or not at all fails here rather than passing as whichever answer was asked
// for first.
func TestEachPairIsExactlyOneOfTheThreeStates(t *testing.T) {
	for _, tc := range healthCases {
		e := Entity{ID: 1, HP: tc.hp, MaxHP: tc.maxHP}
		got, n := state(e)
		if n != 1 {
			t.Errorf("(%d,%d) satisfies %d of the three predicates — alive %v, downed %v, dead %v; "+
				"exactly one must hold", tc.hp, tc.maxHP, n, e.Alive(), e.Downed(), e.Dead())
			continue
		}
		if got != tc.want {
			t.Errorf("(%d,%d) is %s, want %s", tc.hp, tc.maxHP, got, tc.want)
		}
	}
}

func TestTheThreeStatesArePairwiseExclusiveAndJointlyTotal(t *testing.T) {
	values := []int32{-2147483648, -1000, -3, -1, 0, 1, 3, 1000, 2147483647}
	for _, hp := range values {
		for _, maxHP := range values {
			e := Entity{HP: hp, MaxHP: maxHP}
			if _, n := state(e); n != 1 {
				t.Errorf("(%d,%d) satisfies %d of the three predicates — alive %v, downed %v, dead %v",
					hp, maxHP, n, e.Alive(), e.Downed(), e.Dead())
			}
		}
	}
}

// TestTheConstructorClearsTheOrderOfAUnitThatIsNotAlive is the constructor's
// half of the rule: a unit that is not alive holds no target, no stall count and
// no route, and a world built with one has that order cleared rather than
// refused.
//
// The three units are given the SAME order, so what separates their outcomes is
// their health pair and nothing else; and the alive one is the control that
// keeps this from passing by clearing everything. The two health fields
// themselves are untouched by the clearing, which is checked with the rest: a
// rule that zeroed the pair while clearing the order would revive a corpse.
func TestTheConstructorClearsTheOrderOfAUnitThatIsNotAlive(t *testing.T) {
	order := Entity{X: 2, Y: 3, TargetX: 9, TargetY: 9, HasTarget: true, Stall: 5, ActorState: actorStateGuard, Reach: 1,
		PostX: 2, PostY: 3}

	alive := order
	alive.ID, alive.HP, alive.MaxHP = 1, 40, 100
	downed := order
	downed.ID, downed.HP, downed.MaxHP = 2, 0, 100
	dead := order
	dead.ID, dead.HP, dead.MaxHP = 3, -7, 100

	w := mustWorld(t, 1, Bounds{Width: 16, Height: 16}, []Entity{alive, downed, dead})

	cleared := order
	cleared.TargetX, cleared.TargetY, cleared.HasTarget, cleared.Stall = 0, 0, false, 0

	// And BOTH not-alive units come out on the decay ladder, which is the
	// constructor's other normalisation over the same predicate: a positive stage
	// and being not alive hold of the same entities, so a body handed over at no
	// stage at all is put where a death would have put it. Neither carries a
	// dwell, their dying times being none.
	wantDowned, wantDead := cleared, cleared
	wantDowned.ID, wantDowned.HP, wantDowned.MaxHP = 2, 0, 100
	wantDowned.Decay = DecayFallen
	wantDead.ID, wantDead.HP, wantDead.MaxHP = 3, -7, 100
	wantDead.Decay = DecayFallen

	for _, tc := range []struct {
		what string
		want Entity
	}{
		{"the alive unit keeps its order", alive},
		{"the downed unit's order is cleared", wantDowned},
		{"the dead unit's order is cleared", wantDead},
	} {
		if got := occEntity(t, w, tc.want.ID); got != tc.want {
			t.Errorf("%s: unit %d is %+v, want %+v", tc.what, tc.want.ID, got, tc.want)
		}
	}

	// The third field an order consists of. A world built here holds no route for
	// anybody, so this is the clearing's easy half — and it is asserted rather
	// than assumed, because "no target, no stall, no route" is one rule and a
	// reader of this test should be able to see all three of it.
	for i := range w.routes {
		if len(w.routes[i]) != 0 {
			t.Errorf("unit %d was built holding the route %s", w.entities[i].ID, fmtRoute(w.routes[i]))
		}
	}
}

// TestAWorldKeepsTheHealthPairItWasHandedAndDefaultsNeither is the other half of
// the constructor's contract: it invents no health. A unit built with neither
// field named comes back at 0/0 — alive, with no health system — which is what
// leaves every world assembled before these fields existed in exactly the state
// it had, and what makes 100/100 a property of the spawn path rather than of
// this constructor.
func TestAWorldKeepsTheHealthPairItWasHandedAndDefaultsNeither(t *testing.T) {
	given := []Entity{
		{ID: 1},
		{ID: 2, HP: 55, MaxHP: 70},
		{ID: 3, HP: -400, MaxHP: 100},
		{ID: 4, HP: 3, MaxHP: -1},
	}
	w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, given)
	for _, want := range given {
		got := occEntity(t, w, want.ID)
		if got.HP != want.HP || got.MaxHP != want.MaxHP {
			t.Errorf("unit %d was built at %d/%d, want the %d/%d it was handed",
				want.ID, got.HP, got.MaxHP, want.HP, want.MaxHP)
		}
	}
}
