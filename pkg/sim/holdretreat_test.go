package sim

import "testing"

// retreatRun is a warrior of the local participant beside a hostile that never
// acts, slowed to eight ticks a cell so that a Retreat is a walk of several
// ticks. The relation locks the hostile's side to not hostile, so the world
// holds only what a test orders. The world stands on a decision tick, so the
// Retreat's first away decision falls on the tick the command is applied.
func retreatRun(t *testing.T, saved bool) *World {
	t.Helper()
	walker := withdrawalFighter(1, SelfSlot, 20, 20, 100)
	walker.Speed = 32
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2})
	w := engWorld(t, rel, walker, withdrawalFighter(2, 3, 22, 20, 100))
	if saved {
		savedTacticalRegistry(t, w, true)
	}
	w.tick = scriptPassPhase
	return w
}

// retreatingMidStep orders Retreat and steps until the warrior is in the middle
// of a step of its flight with cells still to walk.
func retreatingMidStep(t *testing.T, w *World) Entity {
	t.Helper()
	Step(w, []Command{GroupRetreat(1, SelfSlot, 1)})
	for range 40 {
		e := entityAt(t, w, 1)
		if e.ActorState == actorStateRetreat && e.HasTarget && e.Transit > 0 && e.X < 20 && e.X > e.TargetX {
			return e
		}
		Step(w, nil)
	}
	e := entityAt(t, w, 1)
	t.Fatalf("fixture: the warrior is not mid-step in a Retreat: state %#x at (%d,%d), destination %v (%d,%d), transit %d",
		e.ActorState, e.X, e.Y, e.HasTarget, e.TargetX, e.TargetY, e.Transit)
	return e
}

// TestHoldPressedDuringARetreatEndsTheRetreat: both Stand Ground setters store
// the actor state 0xc for every member (AI-351), so an explicit Retreat, the
// actor state 0x16 that recomputes its flight at every decision (AI-RETREAT-273),
// does not outlast the press. The step under way is paid out (AI-352) and the
// warrior stands; the same world without the press goes on fleeing.
//
// The native world and a world loaded from a SAV registry take different setter
// bodies (commandGroup, commandSavedGroup), so each is run.
func TestHoldPressedDuringARetreatEndsTheRetreat(t *testing.T) {
	t.Parallel()

	for _, saved := range []bool{false, true} {
		name := "native group"
		if saved {
			name = "saved group"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			w := retreatRun(t, saved)
			going := retreatingMidStep(t, w)
			control := worldRoundTripForTest(t, w)

			Step(w, []Command{GroupStance(1, OrderStandGround, 2)})
			held := entityAt(t, w, 1)
			if held.ActorState != actorStateGuard {
				t.Fatalf("after Hold the warrior holds state %#x, want guard %#x", held.ActorState, actorStateGuard)
			}
			if held.HasTarget || held.HasAttackTarget {
				t.Fatalf("after Hold the warrior holds destination %v (%d,%d), victim %v", held.HasTarget, held.TargetX, held.TargetY, held.HasAttackTarget)
			}
			var order uint8
			if saved {
				g := w.savedGroupFor(1)
				if g == nil {
					t.Fatal("after Hold the warrior stands in no saved group")
				}
				order = g.AI[0x20]
				if o := w.savedOrder(1); o == nil || o.State != uint32(actorStateGuard) {
					t.Fatalf("after Hold the saved order record holds %+v, want guard %#x", o, actorStateGuard)
				}
			} else {
				order, _, _ = w.groupState(held.Owner, effectiveGroup(held))
			}
			if order != orderStandGround {
				t.Fatalf("after Hold the warrior's group stands at order %d, want Stand Ground (%d)", order, orderStandGround)
			}
			// The step under way is paid out to the next cell centre.
			for range 20 {
				if entityAt(t, w, 1).Transit == 0 {
					break
				}
				Step(w, nil)
			}
			rest := entityAt(t, w, 1)
			if rest.Transit != 0 || rest.X != going.X || rest.Y != going.Y {
				t.Fatalf("the step under way was not paid out: transit %d at (%d,%d), it was crossing into (%d,%d)", rest.Transit, rest.X, rest.Y, going.X, going.Y)
			}
			for n := range 200 {
				Step(w, nil)
				if e := entityAt(t, w, 1); e.X != rest.X || e.Y != rest.Y || e.HasTarget || e.HasAttackTarget || e.ActorState == actorStateRetreat {
					t.Fatalf("%d ticks after Hold the warrior is at (%d,%d), state %#x, destination %v, victim %v; it stood at (%d,%d)",
						n, e.X, e.Y, e.ActorState, e.HasTarget, e.HasAttackTarget, rest.X, rest.Y)
				}
			}

			for range 220 {
				Step(control, nil)
			}
			ends := entityAt(t, control, 1)
			if ends.ActorState != actorStateRetreat || ends.X >= rest.X {
				t.Fatalf("control: without the press the warrior is in state %#x at (%d,%d); the held warrior stood at (%d,%d)",
					ends.ActorState, ends.X, ends.Y, rest.X, rest.Y)
			}
			t.Logf("Hold pressed with the warrior in state %#x mid-step at (%d,%d) walking to (%d,%d); it left state %#x and stood at (%d,%d); the control fled on to (%d,%d)",
				going.ActorState, going.X, going.Y, going.TargetX, going.TargetY, held.ActorState, rest.X, rest.Y, ends.X, ends.Y)
		})
	}
}

// TestHoldPressedOnARetreatingAndAnIdleMemberLeavesTheIdleOneAsItWas: one press
// of Hold reaches a member in Retreat and a member holding nothing. The second
// keeps its state, its cell and its empty order, and both stand in one group at
// Stand Ground.
func TestHoldPressedOnARetreatingAndAnIdleMemberLeavesTheIdleOneAsItWas(t *testing.T) {
	t.Parallel()

	for _, saved := range []bool{false, true} {
		name := "native group"
		if saved {
			name = "saved group"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			walker := withdrawalFighter(1, SelfSlot, 20, 20, 100)
			walker.Speed = 32
			idle := withdrawalFighter(3, SelfSlot, 20, 30, 100)
			rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2})
			w := engWorld(t, rel, walker, withdrawalFighter(2, 3, 22, 20, 100), idle)
			if saved {
				savedTacticalRegistry(t, w, true)
			}
			w.tick = scriptPassPhase
			retreatingMidStep(t, w)
			before := entityAt(t, w, 3)
			if before.ActorState != actorStateGuard || before.HasTarget || before.HasAttackTarget {
				t.Fatalf("fixture: the second warrior holds state %#x, destination %v, victim %v", before.ActorState, before.HasTarget, before.HasAttackTarget)
			}

			Step(w, []Command{GroupStance(1, OrderStandGround, 2), GroupStance(3, OrderStandGround, 2)})
			first, second := entityAt(t, w, 1), entityAt(t, w, 3)
			if first.ActorState != actorStateGuard || first.HasTarget {
				t.Fatalf("after Hold the retreating warrior holds state %#x, destination %v", first.ActorState, first.HasTarget)
			}
			if second.ActorState != before.ActorState || second.X != before.X || second.Y != before.Y || second.HasTarget || second.HasAttackTarget {
				t.Fatalf("after Hold the idle warrior holds state %#x at (%d,%d), destination %v, victim %v; it held state %#x at (%d,%d)",
					second.ActorState, second.X, second.Y, second.HasTarget, second.HasAttackTarget, before.ActorState, before.X, before.Y)
			}
			if first.CommandGroup == 0 || first.CommandGroup != second.CommandGroup {
				t.Fatalf("the two warriors stand in command groups %d and %d, want one", first.CommandGroup, second.CommandGroup)
			}
		})
	}
}
