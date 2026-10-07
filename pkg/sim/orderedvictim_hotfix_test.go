package sim

import "testing"

// orderedVictimWorld puts the human participant's warrior beside one hostile
// and lets two more hostiles of the same group walk in behind it, one after the
// other. The warrior stands its ground; the three hostiles guard. The first
// hostile has the highest id, so an ordinary decision prefers each newcomer.
func orderedVictimWorld(t *testing.T) *World {
	t.Helper()
	warrior := engFighter(1, SelfSlot, 10, 10)
	warrior.HP, warrior.MaxHP, warrior.DamageBase = 2000, 2000, 6
	first := engFighter(4, 9, 11, 10)
	first.HP, first.MaxHP = 60, 60
	second := engFighter(2, 9, 10, 14)
	third := engFighter(3, 9, 10, 17)
	return engWorld(t, engRel(t, [3]uint32{SelfSlot, 9, 1}, [3]uint32{9, SelfSlot, 1}),
		warrior, second, third, first)
}

// adjacentStrikers counts the hostiles beside the warrior that hold it as their
// victim.
func adjacentStrikers(w *World) int {
	warrior := w.entities[indexOfEntity(w.entities, 1)]
	n := 0
	for _, e := range w.entities {
		if e.Owner == 9 && e.HasAttackTarget && e.AttackTarget == warrior.ID &&
			cellOf(e).chebyshevTo(cellOf(warrior)) <= 1 {
			n++
		}
	}
	return n
}

// TestAnOrderedVictimIsKeptWhileOtherEnemiesStrike is the tester's fight: a
// warrior ordered onto the first of three hostiles that reach him one after the
// other keeps that hostile until it falls. AI-CMD-054 gives the player's attack
// order a group at order 0 with the named target, and AI-RETAL-056 says a blow
// received issues no order and produces no target.
func TestAnOrderedVictimIsKeptWhileOtherEnemiesStrike(t *testing.T) {
	t.Parallel()

	t.Run("control: without the order the warrior changes to a newcomer", func(t *testing.T) {
		t.Parallel()
		w := orderedVictimWorld(t)
		changed := false
		for range 120 {
			Step(w, nil)
			if e := cmdEntity(t, w, 1); e.HasAttackTarget && e.AttackTarget != 4 {
				changed = true
			}
		}
		if !changed {
			t.Fatal("an unordered warrior never left the first hostile: the fixture does not " +
				"produce the fight the order is measured against")
		}
	})

	t.Run("ordered: the warrior stays on the first hostile until it falls", func(t *testing.T) {
		t.Parallel()
		w := orderedVictimWorld(t)
		Step(w, []Command{Attack(1, 4)})
		crowded, fallen := 0, false
		for n := 0; n < 400 && !fallen; n++ {
			if got := cmdEntity(t, w, 1); !got.HasAttackTarget || got.AttackTarget != 4 {
				if cmdEntity(t, w, 4).OrdinaryTargetable() {
					t.Fatalf("tick %d: the warrior ordered onto entity 4 holds victim %v/%d while it stands",
						w.Tick(), got.HasAttackTarget, got.AttackTarget)
				}
				fallen = true
				break
			}
			crowded = max(crowded, adjacentStrikers(w))
			Step(w, nil)
		}
		if !fallen {
			t.Fatal("the warrior never brought the first hostile down in 400 ticks")
		}
		if crowded < 3 {
			t.Fatalf("at most %d hostiles struck the warrior at once, want all 3 before the first fell",
				crowded)
		}
	})
}

// TestAWarriorTakesTheStanceBackWhenItsOrderedVictimIsGone shows the order is
// not a permanent exemption: once the named hostile is down, the group decision
// gives the warrior the next hostile beside it and reads that victim as its own
// choice, so a later newcomer can replace it.
func TestAWarriorTakesTheStanceBackWhenItsOrderedVictimIsGone(t *testing.T) {
	t.Parallel()

	w := orderedVictimWorld(t)
	Step(w, []Command{Attack(1, 4)})
	for n := 0; n < 400 && cmdEntity(t, w, 4).OrdinaryTargetable(); n++ {
		Step(w, nil)
	}
	if cmdEntity(t, w, 4).OrdinaryTargetable() {
		t.Fatal("the warrior never brought the first hostile down in 400 ticks")
	}
	for range 2 * scriptCycle {
		Step(w, nil)
	}
	got := cmdEntity(t, w, 1)
	if !got.HasAttackTarget || (got.AttackTarget != 2 && got.AttackTarget != 3) {
		t.Fatalf("after the ordered hostile fell the warrior holds victim %v/%d, want a hostile still beside it",
			got.HasAttackTarget, got.AttackTarget)
	}
	if got.ActorState != actorStateGuard {
		t.Errorf("the warrior's actor state is %d, want guard (%d) once the order is over",
			got.ActorState, actorStateGuard)
	}
}
