package sim

// The stall count, and the tick that spends it.
//
// A unit whose NEAR search finds no route keeps everything it had and counts the
// tick. When the count reaches the limit the target is cleared inside that same
// tick, so the count a stored world carries is always below the limit — which is
// what makes the byte form's refusal of one at or above it a check on corruption
// rather than on this package's own output. That last sentence is asserted over
// the run below rather than argued: every tick's world is marshalled and read
// back, and a count the decoder would refuse shows up as an error there.
//
// The fixture is a unit boxed in BY OTHER UNITS, and it has to be. A wall of
// terrain no longer produces a stall at all: the far search reads terrain, it
// answers before the first tick is over, and the order ends there rather than
// sixteen ticks later. What a count measures now is a mover waiting for other
// movers, which is the only thing that can change while a world is advanced.

import "testing"

// The boxed fixture: a five by five of open ground, one unit at (2,2) ordered to
// (4,2), and eight units standing on every one of its neighbours holding no
// order of their own.
//
// The terrain says the order is servable, so the far search answers with a route
// and the unit keeps it. It is the near search that finds nothing, on this tick
// and on every tick after it, because the ring never moves.
var boxedBounds = Bounds{Width: 5, Height: 5}

var boxedUnit = Entity{ID: 1, X: 2, Y: 2}

const boxedTargetX, boxedTargetY = 4, 2

// boxedRing is the eight units around boxedUnit, ids ascending after its own.
func boxedRing() []Entity {
	out := make([]Entity, 0, 8)
	id := EntityID(2)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			out = append(out, Entity{ID: id, X: boxedUnit.X + dx, Y: boxedUnit.Y + dy})
			id++
		}
	}
	return out
}

// boxedWorld builds the fixture, so each case below starts from the same place.
func boxedWorld(t *testing.T) *World {
	t.Helper()
	return mustWorldGrid(t, 1, boxedBounds, ModeCanonical, nil, append([]Entity{boxedUnit}, boxedRing()...))
}

func boxedOrder() []Command {
	return []Command{{Entity: boxedUnit.ID, X: boxedTargetX, Y: boxedTargetY}}
}

// TestABoxedInOrderIsHeldFifteenTicksAndGivenUpOnTheSixteenth is AC-3's give-up
// field by field. The three target fields are compared one at a time rather than
// as a struct, because "the target is cleared" is three separate facts and a
// coordinate left behind as residue is exactly the one a struct comparison would
// still catch but a HasTarget check alone would not.
func TestABoxedInOrderIsHeldFifteenTicksAndGivenUpOnTheSixteenth(t *testing.T) {
	w := boxedWorld(t)

	for tick := 1; tick <= stallLimit; tick++ {
		if tick == 1 {
			Step(w, boxedOrder())
		} else {
			Step(w, nil)
		}
		e := w.Entities()[0]

		if e.X != boxedUnit.X || e.Y != boxedUnit.Y {
			t.Fatalf("tick %d: the unit is at (%d,%d), want (%d,%d) — a boxed-in mover moves nothing",
				tick, e.X, e.Y, boxedUnit.X, boxedUnit.Y)
		}

		wantX, wantY, wantHas := int32(boxedTargetX), int32(boxedTargetY), true
		wantStall := uint8(tick)
		if tick == stallLimit {
			wantX, wantY, wantHas, wantStall = 0, 0, false, 0
		}
		if e.TargetX != wantX {
			t.Errorf("tick %d: TargetX is %d, want %d", tick, e.TargetX, wantX)
		}
		if e.TargetY != wantY {
			t.Errorf("tick %d: TargetY is %d, want %d", tick, e.TargetY, wantY)
		}
		if e.HasTarget != wantHas {
			t.Errorf("tick %d: HasTarget is %v, want %v", tick, e.HasTarget, wantHas)
		}
		if e.Stall != wantStall {
			t.Errorf("tick %d: the stall count is %d, want %d", tick, e.Stall, wantStall)
		}

		// The count a world CARRIES is always below the limit, so the form it
		// writes is one it can read back. Asserted at every tick, the one that
		// reaches the limit included.
		if e.Stall >= stallLimit {
			t.Errorf("tick %d: the stored stall count is %d, and the byte form refuses %d or more",
				tick, e.Stall, stallLimit)
		}
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("tick %d: MarshalBinary: %v", tick, err)
		}
		var back World
		if err := back.UnmarshalBinary(form); err != nil {
			t.Fatalf("tick %d: the world this package produced does not decode: %v", tick, err)
		}
	}
}

func TestAStalledTickWritesTheCountAndNothingElse(t *testing.T) {
	w := boxedWorld(t)

	for tick := 1; tick < stallLimit; tick++ {
		before := w.Entities()[0]
		beforeRoute := append([]cell(nil), w.routes[0]...)
		if tick == 1 {
			Step(w, boxedOrder())
		} else {
			Step(w, nil)
		}
		after := w.Entities()[0]
		if tick > 1 {
			checkRoute(t, "the route across a stalled tick", w.routes[0], true, beforeRoute)
		}

		if after.Stall != before.Stall+1 {
			t.Fatalf("tick %d: the count went %d to %d, want a rise of one", tick, before.Stall, after.Stall)
		}
		// The order is given on tick 1, so the target it sets is the baseline
		// from tick 1 onward rather than before it.
		if tick == 1 {
			before.TargetX, before.TargetY, before.HasTarget = boxedTargetX, boxedTargetY, true
		}
		before.Stall = after.Stall
		if after != before {
			t.Errorf("tick %d: the entity is %+v, want %+v with the count alone moved", tick, after, before)
		}
	}
}

func TestEachTickHasExactlyOneOutcome(t *testing.T) {
	unservable := func(t *testing.T) *World {
		t.Helper()
		g := make([]byte, 25)
		for _, c := range []cell{{0, 1}, {1, 1}, {1, 2}, {1, 3}, {0, 3}} {
			g[c.y*5+c.x] = blockGround
		}
		return mustWorldGrid(t, 1, boxedBounds, ModeCanonical, g, []Entity{{ID: 1, X: 0, Y: 2}})
	}

	for _, tc := range []struct {
		what       string
		build      func(*testing.T) *World
		order      []Command
		wantHeld   int
		wantGaveUp int
		wantEnded  int
	}{
		{"a mover boxed in by units", boxedWorld, boxedOrder(), stallLimit - 1, 1, 0},
		{"an order the terrain does not serve", unservable,
			[]Command{{Entity: 1, X: 4, Y: 2}}, 0, 0, 1},
	} {
		w := tc.build(t)
		held, gaveUp, ended := 0, 0, 0

		for tick := 1; tick <= stallLimit; tick++ {
			before := w.Entities()[0]
			if tick == 1 {
				// The order is given on this tick and applies before the move, so
				// the state the outcome is classified against is the one the
				// command leaves — not the one before it arrived.
				before.TargetX, before.TargetY, before.HasTarget = tc.order[0].X, tc.order[0].Y, true
				Step(w, tc.order)
			} else {
				Step(w, nil)
			}
			after := w.Entities()[0]
			if !before.HasTarget {
				continue // the order is over; there is no outcome to classify
			}

			moved := after.X != before.X || after.Y != before.Y
			cleared := before.HasTarget && !after.HasTarget
			raised := after.Stall == before.Stall+1

			advanced := moved && after.Stall == 0 && !raised
			holding := !moved && !cleared && raised && after.HasTarget
			gave := !moved && cleared && after.Stall == 0 && before.Stall == stallLimit-1
			over := !moved && cleared && after.Stall == 0 && before.Stall < stallLimit-1

			n := 0
			for _, outcome := range []bool{advanced, holding, gave, over} {
				if outcome {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("%s tick %d: %d of the four outcomes hold (advanced=%v holding=%v gave-up=%v "+
					"ended=%v) for %+v -> %+v", tc.what, tick, n, advanced, holding, gave, over, before, after)
			}
			if holding {
				held++
			}
			if gave {
				gaveUp++
			}
			if over {
				ended++
			}
		}

		if held != tc.wantHeld || gaveUp != tc.wantGaveUp || ended != tc.wantEnded {
			t.Errorf("%s: %d hold(s), %d give-up(s), %d order(s) ended by a far failure; want %d, %d and %d",
				tc.what, held, gaveUp, ended, tc.wantHeld, tc.wantGaveUp, tc.wantEnded)
		}
	}
}

// TestAdvancingReturnsTheCountToZero is the reset, and the fixture is built so
// that the advance is NOT also an arrival: a unit that arrives has its target
// cleared, and a cleared target zeroes the count too, so an arriving unit cannot
// tell the two rules apart.
//
//	# # . # #        A corridor along row 1 with one pocket at (2,0). The unit
//	A B . . .        under test is A at (0,1), ordered to (4,1); B stands in the
//	# # # # #        one cell A must pass through, so A stalls until B is ordered
//	                 into the pocket and steps out of the corridor.
func TestAdvancingReturnsTheCountToZero(t *testing.T) {
	b := Bounds{Width: 5, Height: 3}
	grid := make([]byte, 15)
	for x := int32(0); x < 5; x++ {
		grid[0*5+x] = blockGround
		grid[2*5+x] = blockGround
	}
	grid[0*5+2] = 0 // the pocket

	w := mustWorldGrid(t, 1, b, ModeCanonical, grid, []Entity{
		{ID: 1, X: 0, Y: 1},
		{ID: 2, X: 1, Y: 1},
	})

	// Three ticks with B in the way: A counts them and moves nothing. B holds no
	// target at all, and an entity holding no target holds a zero count.
	for tick := 1; tick <= 3; tick++ {
		if tick == 1 {
			Step(w, []Command{{Entity: 1, X: 4, Y: 1}})
		} else {
			Step(w, nil)
		}
		ents := w.Entities()
		if got := (Entity{ID: 1, X: 0, Y: 1, TargetX: 4, TargetY: 1, HasTarget: true, Stall: uint8(tick), ActorState: actorStateGuard, Reach: 1,
			PostX: 0, PostY: 1}); ents[0] != got {
			t.Fatalf("tick %d: A is %+v, want %+v", tick, ents[0], got)
		}
		if ents[1] != (Entity{ID: 2, X: 1, Y: 1, ActorState: actorStateGuard, Reach: 1, PostX: 1, PostY: 1}) {
			t.Fatalf("tick %d: B is %+v, want it standing still with no target and no count", tick, ents[1])
		}
	}

	// Tick 4 orders B aside. A is resolved first — ids ascend — so it still sees
	// B where it was and counts a fourth tick; B then leaves the corridor.
	Step(w, []Command{{Entity: 2, X: 2, Y: 0}})
	ents := w.Entities()
	if ents[0].Stall != 4 || ents[0].X != 0 || ents[0].Y != 1 {
		t.Fatalf("tick 4: A is %+v, want it held at (0,1) with a count of 4", ents[0])
	}
	// B stepped from (1,1) to (2,0) — one cell north-east — and the facing
	// it arrived with is that step's, which is a fact about B and not residue
	// of the order it has just spent.
	if ents[1] != (Entity{ID: 2, X: 2, Y: 0, Facing: facingOfDir(1), DesiredFacing: facingOfDir(1), ActorState: actorStateGuard, Reach: 1, PostX: 1, PostY: 1}) {
		t.Fatalf("tick 4: B is %+v, want it in the pocket with its target cleared", ents[1])
	}

	// Tick 5: the corridor is clear, A advances one cell — and does NOT arrive,
	// so the zero below is the advance's own reset and not a clearing's.
	Step(w, nil)
	want := Entity{ID: 1, X: 1, Y: 1, TargetX: 4, TargetY: 1, HasTarget: true, Stall: 0, Facing: facingOfDir(2), DesiredFacing: facingOfDir(2), ActorState: actorStateGuard, Reach: 1,
		PostX: 0, PostY: 1}
	if got := w.Entities()[0]; got != want {
		t.Errorf("tick 5: A is %+v, want %+v", got, want)
	}
}

// TestAClearedTargetTakesTheCountWithIt closes the one path on which a count
// could outlive the target that gave it meaning: a unit that has been stalling is
// ordered onto the cell it is already standing on, and the already-there rule
// clears the target without a search. A count left behind there is a state the
// byte form refuses, so the world would marshal and then fail to read back.
func TestAClearedTargetTakesTheCountWithIt(t *testing.T) {
	w := boxedWorld(t)
	Step(w, boxedOrder())
	Step(w, nil)
	if got := w.Entities()[0].Stall; got != 2 {
		t.Fatalf("the unit's count is %d, want 2 before the order that clears it", got)
	}

	Step(w, []Command{{Entity: boxedUnit.ID, X: boxedUnit.X, Y: boxedUnit.Y}})
	if got := w.Entities()[0]; got != (Entity{ID: 1, X: boxedUnit.X, Y: boxedUnit.Y, ActorState: actorStateGuard, Reach: 1,
		PostX: boxedUnit.X, PostY: boxedUnit.Y}) {
		t.Errorf("the unit is %+v, want its target and its count both gone", got)
	}
	if len(w.routes[0]) != 0 {
		t.Errorf("the unit holds the route %s with no target", fmtRoute(w.routes[0]))
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Errorf("the world does not decode: %v", err)
	}
}
