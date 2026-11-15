package sim

import "testing"

// moveOntoEnemyWorld is an open field with the player's hero at (4,5) and a
// hostile at (12,5), in the game the profile names.
func moveOntoEnemyWorld(t *testing.T, second bool, foeOwner uint32, foeX, foeY int32) *World {
	t.Helper()
	hero := engFighter(1, SelfSlot, 4, 5)
	hero.HP, hero.MaxHP = 2000, 2000
	foe := engFighter(2, foeOwner, foeX, foeY)
	foe.HP, foe.MaxHP = 2000, 2000
	foe.DamageBase = 0
	foe.ScanRange = 0
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 1})
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, foe}, s, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	wakeGroups(w)
	return w
}

// walkToHeldGoal orders the hero to (9,5) and sends the unit at (9,6) onto it,
// then steps w for ticks and reports the hero's and the unit's state after each.
func walkToHeldGoal(t *testing.T, w *World, ticks int, each func(k int, hero, other Entity)) {
	t.Helper()
	Step(w, []Command{MoveTo(1, CellPoint{X: 9, Y: 5})})
	Step(w, []Command{MoveTo(2, CellPoint{X: 9, Y: 5})})
	for k := 0; k < ticks; k++ {
		Step(w, nil)
		each(k, laEnt(t, w, 1), laEnt(t, w, 2))
	}
}

// A player's walk whose goal an enemy takes on the way ends beside the enemy in
// the stand order, and the hero strikes it (the move order is gone).
func TestWalkGoalTakenByEnemyBecomesStand(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := moveOntoEnemyWorld(t, second, 3, 9, 6)
		stood, struck := -1, -1
		walkToHeldGoal(t, w, 60, func(k int, h, f Entity) {
			if f.X != 9 || f.Y != 5 {
				return
			}
			if stood < 0 && !h.HasTarget {
				stood = k
				if h.ActorState != actorStateAcquire || h.X != 8 || h.Y != 5 {
					t.Fatalf("second=%v tick %d: hero at (%d,%d) state %#x; want the stand beside the enemy at (8,5)", second, k, h.X, h.Y, h.ActorState)
				}
			}
			if struck < 0 && f.HP < 2000 {
				struck = k
			}
		})
		if stood < 0 || stood > 5 {
			t.Errorf("second=%v: the move order ended at tick %d; want within 5 ticks of the unit's arrival", second, stood)
		}
		if struck < 0 || struck > 40 {
			t.Errorf("second=%v: the hero struck at tick %d; want within the stand order's own reach rules", second, struck)
		}
	}
}

// A goal a friendly unit holds keeps the walk as it was: the hero holds the
// order and never strikes.
func TestWalkGoalHeldByFriendKeepsWalking(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := moveOntoEnemyWorld(t, second, SelfSlot, 9, 6)
		walkToHeldGoal(t, w, 30, func(k int, h, f Entity) {
			if f.X == 9 && f.Y == 5 && (k > 3 && k < 12 && !h.HasTarget || h.HasAttackTarget || f.HP < 2000) {
				t.Fatalf("second=%v tick %d: hero target %v victim %v; a friendly goal must not start a fight or end the walk before the stall count does", second, k, h.HasTarget, h.HasAttackTarget)
			}
		})
	}
}

// A hero not yet beside the held cell keeps walking toward it.
func TestWalkGoalHeldFarAwayKeepsWalking(t *testing.T) {
	for _, second := range []bool{false, true} {
		w := moveOntoEnemyWorld(t, second, 3, 12, 5)
		Step(w, []Command{MoveTo(1, CellPoint{X: 12, Y: 5})})
		for k := 0; k < 2; k++ {
			Step(w, nil)
		}
		h := laEnt(t, w, 1)
		if !h.HasTarget || h.X <= 4 || h.HasAttackTarget {
			t.Errorf("second=%v: hero at (%d,%d) target %v victim %v; want it still walking", second, h.X, h.Y, h.HasTarget, h.HasAttackTarget)
		}
	}
}
