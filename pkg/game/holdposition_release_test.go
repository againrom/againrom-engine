package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// holdPanelCell is the command panel's Stand Ground cell, the game's Hold
// Position: the seventh of the eight, in the panel's own row-major order.
const holdPanelCell = 6

// holdSelect selects one warrior with a click on him.
func holdSelect(t *testing.T, arena *loadedCycleArena, hero sim.EntityID) {
	t.Helper()
	if err := arena.app.HeadlessSelectEntity(uint32(hero)); err != nil {
		t.Fatal(err)
	}
	if got := arena.app.HeadlessSelection(); !slices.Equal(got, []uint32{uint32(hero)}) {
		t.Fatalf("the selection is %v, want warrior %d alone", got, hero)
	}
}

// holdChase arms the attack cursor and clicks the creature, and requires the
// one command the click queued.
func holdChase(t *testing.T, arena *loadedCycleArena, hero, victim sim.EntityID) {
	t.Helper()
	if err := arena.app.HeadlessKey("attack"); err != nil {
		t.Fatal(err)
	}
	x, y, err := arena.app.HeadlessEntityPoint(uint32(victim))
	if err != nil {
		t.Fatal(err)
	}
	arena.tap(t, x, y)
	if got := arena.live.pending; len(got) != 1 || got[0].Kind != sim.KindAttack || got[0].Entity != hero || got[0].X != int32(victim) {
		t.Fatalf("the click queued %+v, want warrior %d's attack on creature %d", got, hero, victim)
	}
}

// holdGroupOrder is the order of the group a warrior is decided for: his
// command group when he holds one, the group the map placed him in otherwise.
func holdGroupOrder(w *sim.World, e sim.Entity) (uint8, bool) {
	group := e.CommandGroup
	if group == 0 {
		group = e.Group
	}
	order, _, known := w.FrozenGroupAI(e.Owner, group)
	return order, known
}

// holdPress presses the panel's Hold Position cell with the selection standing.
// The press queues the stance and the frame's own tick applies it, so the proof
// that it arrived is the warrior's group standing at Stand Ground after a
// command group or a group order that was not there before.
func holdPress(t *testing.T, arena *loadedCycleArena, hero sim.EntityID) {
	t.Helper()
	x, y, err := arena.app.HeadlessCommandPoint(holdPanelCell)
	if err != nil {
		t.Fatal(err)
	}
	before := arena.get(hero)
	beforeOrder, _ := holdGroupOrder(arena.live.world, before)
	arena.tap(t, x, y)
	after := arena.get(hero)
	order, known := holdGroupOrder(arena.live.world, after)
	if !known || order != uint8(sim.OrderStandGround) || after.CommandGroup == before.CommandGroup && order == beforeOrder {
		t.Fatalf("the panel's Hold Position cell left warrior %d in command group %d at order %d (known %v); he was in %d at order %d",
			hero, after.CommandGroup, order, known, before.CommandGroup, beforeOrder)
	}
}

// holdUntil advances the arena until the warrior satisfies the condition.
func holdUntil(t *testing.T, arena *loadedCycleArena, hero sim.EntityID, what string, limit int, ok func(sim.Entity) bool) {
	t.Helper()
	for range limit {
		if ok(arena.get(hero)) {
			return
		}
		arena.live.tick()
	}
	h := arena.get(hero)
	t.Fatalf("within %d ticks the warrior never reached: %s; he stands at (%d,%d), victim %v, destination %v",
		limit, what, h.X, h.Y, h.HasAttackTarget, h.HasTarget)
}

// holdControl is a copy of the mission's world as it stands, decoded from its
// own bytes and stepped without any command: the same ticks without the press.
func holdControl(t *testing.T, arena *loadedCycleArena) *sim.World {
	t.Helper()
	form, err := arena.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var control sim.World
	if err := control.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	return &control
}

// holdStands requires the warrior to hold no order, straight after the press
// and for n ticks on, and to stay on his cell; n covers at least twelve group
// decisions.
func holdStands(t *testing.T, arena *loadedCycleArena, hero sim.EntityID, n int) sim.Entity {
	t.Helper()
	held := arena.get(hero)
	if held.HasAttackTarget || held.HasTarget {
		t.Fatalf("after Hold Position the warrior holds victim %v/%d, destination %v (%d,%d)",
			held.HasAttackTarget, held.AttackTarget, held.HasTarget, held.TargetX, held.TargetY)
	}
	for k := range n {
		arena.live.tick()
		if h := arena.get(hero); h.X != held.X || h.Y != held.Y || h.HasAttackTarget || h.HasTarget {
			t.Fatalf("%d ticks after Hold Position the warrior is at (%d,%d) with victim %v, destination %v; he stood at (%d,%d)",
				k+1, h.X, h.Y, h.HasAttackTarget, h.HasTarget, held.X, held.Y)
		}
	}
	return held
}

// Tester item 45: Hold Position pressed on a warrior that was chasing a creature
// or walking to a cell let him carry on, and a member of the participant kept
// following a victim that had walked out of reach. Both Stand Ground setters
// store the pending order 0 and no progress, the executor runs no arm for 0, and
// the participant's evaluation stores 0 for a member that scores nothing
// (AI-351, AI-352, AI-350, AI-349, AI-353).
//
// The arena is mission 20's own ground with the creatures of the loaded-cycle
// arena, which hold the participant as a locked ally: they never strike, never
// turn hostile and never walk, so the second warrior is alone with what he is
// told. He is ordered with the attack cursor and a click on a creature six
// cells off, or with a ground click, and Hold Position is pressed on the
// command panel while he is on his way. The control is the same world bytes
// stepped with no press: it finishes the chase or the walk.
func TestReleaseHoldPositionStopsAChaseAndAWalk(t *testing.T) {
	t.Run("chase", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 2)
		hero, foe := arena.heroes[1], arena.east
		start := arena.get(hero)
		holdSelect(t, arena, hero)
		holdChase(t, arena, hero, foe)
		holdUntil(t, arena, hero, "on his way to the creature, three cells short of it", 120, func(h sim.Entity) bool {
			f := arena.get(foe)
			return h.HasAttackTarget && h.AttackTarget == foe && h.HasTarget && (h.X != start.X || h.Y != start.Y) &&
				cellDistance(h, f.X, f.Y) >= 3
		})
		before := arena.get(hero)
		control := holdControl(t, arena)
		holdPress(t, arena, hero)
		pressed := arena.now()
		held := holdStands(t, arena, hero, 200)
		f := arena.get(foe)
		t.Logf("Hold Position pressed at tick %d with the warrior at (%d,%d), %d cells from the creature at (%d,%d); he stood at (%d,%d) for 200 ticks",
			pressed, before.X, before.Y, cellDistance(before, f.X, f.Y), f.X, f.Y, held.X, held.Y)
		if cellDistance(held, f.X, f.Y) < 2 {
			t.Fatalf("the warrior stopped beside the creature, (%d,%d) against (%d,%d): the press came too late to prove anything",
				held.X, held.Y, f.X, f.Y)
		}
		for range 400 {
			sim.Step(control, nil)
		}
		var ends sim.Entity
		for _, e := range control.Entities() {
			if e.ID == hero {
				ends = e
			}
		}
		if !ends.HasAttackTarget || cellDistance(ends, f.X, f.Y) > 1 {
			t.Fatalf("control: without the press the warrior stands at (%d,%d), victim %v; the creature is at (%d,%d)",
				ends.X, ends.Y, ends.HasAttackTarget, f.X, f.Y)
		}
	})

	t.Run("walk", func(t *testing.T) {
		arena := openLoadedCycleArena(t, 2)
		hero := arena.heroes[1]
		start := arena.get(hero)
		holdSelect(t, arena, hero)
		goalX, goalY := start.X+5, start.Y
		if cmds := arena.orderMove(t, goalX, goalY); len(cmds) != 1 || cmds[0].Entity != hero {
			t.Fatalf("the ground click queued %+v, want one move for warrior %d", cmds, hero)
		}
		holdUntil(t, arena, hero, "walking, two cells from where he began", 120, func(h sim.Entity) bool {
			return h.HasTarget && h.TargetX == goalX && h.TargetY == goalY && cellDistance(h, start.X, start.Y) >= 2
		})
		before := arena.get(hero)
		control := holdControl(t, arena)
		holdPress(t, arena, hero)
		pressed := arena.now()
		held := holdStands(t, arena, hero, 200)
		t.Logf("Hold Position pressed at tick %d with the warrior at (%d,%d) walking to (%d,%d); he stood at (%d,%d) for 200 ticks",
			pressed, before.X, before.Y, goalX, goalY, held.X, held.Y)
		if cellDistance(held, goalX, goalY) < 2 {
			t.Fatalf("the warrior stopped on the goal (%d,%d) at (%d,%d): the press came too late to prove anything", goalX, goalY, held.X, held.Y)
		}
		for range 200 {
			sim.Step(control, nil)
		}
		var ends sim.Entity
		for _, e := range control.Entities() {
			if e.ID == hero {
				ends = e
			}
		}
		if ends.HasTarget || cellDistance(ends, goalX, goalY) > 1 {
			t.Fatalf("control: without the press the warrior stands at (%d,%d), destination %v, and the goal is (%d,%d)",
				ends.X, ends.Y, ends.HasTarget, goalX, goalY)
		}
	})
}
