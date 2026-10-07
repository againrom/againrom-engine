package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// retreatingState is the actor state an explicit Retreat installs.
const retreatingState = 0x16

// Hold Position pressed while a warrior flees from an explicit Retreat ends the
// Retreat. Both Stand Ground setters store the actor state 0xc for every member
// (AI-351), and the state 0x16 arm, which recomputes the flight at each
// decision and has no end of its own, is dispatched only for a member the group
// leaves to its own state (AI-RETREAT-273, AI-RETREAT-275). The step under way
// is paid out (AI-352) and the warrior then stands.
//
// The arena is mission 20's own ground with the creatures of the loaded-cycle
// arena, which hold the participant as a locked ally: they never strike and
// never walk. The first warrior stands between two of them. He is selected and
// ordered to Retreat with the R key, and once he is on his flight, two cells
// from where he began and out of reach of both, the whole party is selected
// with the E key and Hold Position is pressed on the command panel. The second
// warrior stands idle far to the north. The control is the same world bytes
// stepped with no press: it goes on fleeing.
func TestReleaseHoldPositionEndsARetreat(t *testing.T) {
	arena := openLoadedCycleArena(t, 2)
	runner, idle := arena.heroes[0], arena.heroes[1]
	start := arena.get(runner)
	waiting := arena.get(idle)
	if waiting.ActorState == retreatingState || waiting.HasTarget || waiting.HasAttackTarget {
		t.Fatalf("fixture: the idle warrior holds state %#x, destination %v, victim %v", waiting.ActorState, waiting.HasTarget, waiting.HasAttackTarget)
	}

	holdSelect(t, arena, runner)
	if err := arena.app.HeadlessKey("r"); err != nil {
		t.Fatal(err)
	}
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindGroupRetreat || got[0].Entity != runner || got[0].Player != sim.SelfSlot {
		t.Fatalf("the R key queued %+v, want Retreat for warrior %d", got, runner)
	}
	ordered := arena.now()
	holdUntil(t, arena, runner, "in a Retreat, mid-step, two cells from where he began", 400, func(h sim.Entity) bool {
		return h.ActorState == retreatingState && h.HasTarget && h.Transit > 0 && cellDistance(h, start.X, start.Y) >= 2
	})
	going := arena.get(runner)
	for _, id := range []sim.EntityID{arena.east, arena.south} {
		if c := arena.get(id); cellDistance(going, c.X, c.Y) < 2 {
			t.Fatalf("the warrior is within a cell of creature %d at (%d,%d): the press would come too early to prove anything", id, c.X, c.Y)
		}
	}
	if err := arena.app.HeadlessKey("e"); err != nil {
		t.Fatal(err)
	}
	if got := arena.app.HeadlessSelection(); len(got) != 2 || !slices.Contains(got, uint32(runner)) || !slices.Contains(got, uint32(idle)) {
		t.Fatalf("the E key selected %v, want both warriors %d and %d", got, runner, idle)
	}
	control := holdControl(t, arena)
	holdPress(t, arena, runner)
	pressed := arena.now()
	if order, known := holdGroupOrder(arena.live.world, arena.get(idle)); !known || order != uint8(sim.OrderStandGround) {
		t.Fatalf("the second warrior stands in a group at order %d (known %v), want Stand Ground", order, known)
	}
	after := arena.get(runner)
	if after.ActorState != 0xb || after.HasTarget || after.HasAttackTarget {
		t.Fatalf("after Hold Position the fleeing warrior holds state %#x, destination %v (%d,%d), victim %v",
			after.ActorState, after.HasTarget, after.TargetX, after.TargetY, after.HasAttackTarget)
	}
	for range 20 {
		if arena.get(runner).Transit == 0 {
			break
		}
		arena.live.tick()
	}
	rest, quiet := arena.get(runner), arena.get(idle)
	if rest.Transit != 0 {
		t.Fatalf("the step under way was not paid out: transit %d", rest.Transit)
	}
	for k := range 200 {
		arena.live.tick()
		if h := arena.get(runner); h.X != rest.X || h.Y != rest.Y || h.HasTarget || h.HasAttackTarget || h.ActorState == retreatingState {
			t.Fatalf("%d ticks after Hold Position the fleeing warrior is at (%d,%d), state %#x, destination %v, victim %v; he stood at (%d,%d)",
				k+1, h.X, h.Y, h.ActorState, h.HasTarget, h.HasAttackTarget, rest.X, rest.Y)
		}
		if h := arena.get(idle); h.X != quiet.X || h.Y != quiet.Y || h.HasTarget || h.HasAttackTarget || h.ActorState != waiting.ActorState {
			t.Fatalf("%d ticks after Hold Position the idle warrior is at (%d,%d), state %#x, destination %v, victim %v; he held state %#x at (%d,%d)",
				k+1, h.X, h.Y, h.ActorState, h.HasTarget, h.HasAttackTarget, waiting.ActorState, quiet.X, quiet.Y)
		}
	}
	for range 220 {
		sim.Step(control, nil)
	}
	var ends sim.Entity
	for _, e := range control.Entities() {
		if e.ID == runner {
			ends = e
		}
	}
	east := arena.get(arena.east)
	if ends.ActorState != retreatingState || cellDistance(ends, east.X, east.Y) <= cellDistance(rest, east.X, east.Y) {
		t.Fatalf("control: without the press the warrior is in state %#x at (%d,%d), %d cells from the creature at (%d,%d); the held warrior stood %d cells from it",
			ends.ActorState, ends.X, ends.Y, cellDistance(ends, east.X, east.Y), east.X, east.Y, cellDistance(rest, east.X, east.Y))
	}
	t.Logf("Retreat ordered at tick %d from (%d,%d); Hold Position pressed at tick %d with the warrior in state %#x at (%d,%d) walking to (%d,%d); after it he held state %#x and stood at (%d,%d) for 200 ticks; the control fled on to (%d,%d) in state %#x",
		ordered, start.X, start.Y, pressed, going.ActorState, going.X, going.Y, going.TargetX, going.TargetY,
		after.ActorState, rest.X, rest.Y, ends.X, ends.Y, ends.ActorState)
}
