package game

import (
	"testing"

	"againrom/pkg/sim"
)

// heroWalkState is the hero's order state word, its arrival latch
// (ord+0x50) and its Group's order byte.
func heroWalkState(w *sim.World, id sim.EntityID) (state uint32, latch byte, order uint8) {
	groups, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == id {
			state, latch = o.State, o.Raw[0x50]
		}
	}
	for _, g := range groups {
		for _, m := range g.Members {
			if m.Bound && m.Entity == id {
				order = g.AI[0x20]
			}
		}
	}
	return state, latch, order
}

// A hero sent by a Move onto the unreachable creature's cell is refused by
// the far search, the original's static search (AI-658). The refusal stands
// it down to state 0xc with its post on its cell and the arrival latch set,
// and takes no victim; the Move Group then acquires for it and does not send
// it again, so it takes the hostile beside it and strikes. SAVE and cold LOAD
// at the stand-down restore the same World, and the loaded hero takes the
// pick and strikes on the same ticks as the live one.
func TestReleaseARefusedMoveTakesTheHostileBesideTheHeroAfterItsStandDown(t *testing.T) {
	a, _ := openUnreachableMission(t, true, 1, 19, 64)
	foe := a.get(releaseUnreachableFoe)
	a.live.pending = append(a.live.pending, sim.GroupMoveTo(a.hero, sim.CellPoint{X: foe.X, Y: foe.Y}, 1))
	start := a.get(a.beside).HP
	var stood, picked, struck int
	var latch byte
	var cold *FrontEnd
	for n := 0; n < 600 && struck == 0; n++ {
		a.live.tick()
		h := a.get(a.hero)
		state, latched, order := heroWalkState(a.live.world, a.hero)
		justLoaded := false
		if stood == 0 && order == 4 && state == 0xc {
			stood, latch = n+1, latched
			if latch != 1 {
				t.Fatalf("tick %d: the far search's refusal left arrival latch %d, want 1", n, latch)
			}
			if h.HasAttackTarget || h.ActorState != 0xc || h.PostX != h.X || h.PostY != h.Y {
				t.Fatalf("tick %d: the stand-down left victim %v/%d, state %#x, post (%d,%d) at (%d,%d)",
					n, h.HasAttackTarget, h.AttackTarget, h.ActorState, h.PostX, h.PostY, h.X, h.Y)
			}
			cold = a.reloadCold(t)
			assertCurrentWorldEqual(t, a.live.world, cold.live.world, "LOAD of the stood-down Move member")
			justLoaded = true
		}
		if cold != nil && !justLoaded {
			cold.live.tick()
			assertEntitiesEqual(t, a.live.world, cold.live.world, []sim.EntityID{a.hero, releaseUnreachableFoe, a.beside}, "successor of the stood-down Move member")
		}
		if picked == 0 && h.HasAttackTarget && h.AttackTarget == a.beside {
			picked = n + 1
			if stood == 0 {
				t.Fatalf("tick %d: the hero took the pick before any stand-down", n)
			}
		}
		if b := a.get(a.beside); b.HP < start {
			struck = n + 1
		}
	}
	if stood == 0 || picked == 0 || struck == 0 {
		h := a.get(a.hero)
		state, _, order := heroWalkState(a.live.world, a.hero)
		t.Fatalf("stand-down tick %d, pick tick %d, first blow tick %d; hero at (%d,%d) state %#x/%#x order %d victim %v/%d",
			stood, picked, struck, h.X, h.Y, h.ActorState, state, order, h.HasAttackTarget, h.AttackTarget)
	}
	t.Logf("stand-down at tick %d with arrival latch %d, pick at tick %d, first blow on it at tick %d", stood, latch, picked, struck)
}
