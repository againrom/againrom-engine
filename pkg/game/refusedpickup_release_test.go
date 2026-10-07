package game

import (
	"testing"

	"againrom/pkg/sim"
)

// refusedPickupArena is the mission 20 arena with a wall across the map east of
// the hero and a sack beyond it.
func refusedPickupArena(t *testing.T) (*loadedCycleArena, sim.EntityID, sim.CellPoint) {
	t.Helper()
	loadedCycleTerrainHook = func(block []byte, bw, bh int, cx, cy int32) {
		for y := 0; y < bh; y++ {
			block[y*bw+int(cx)+2] |= 1
		}
	}
	t.Cleanup(func() { loadedCycleTerrainHook = nil })
	arena := openLoadedCycleArena(t, 1)
	loadedCycleTerrainHook = nil
	sack := sim.CellPoint{X: arena.cx + 9, Y: arena.cy}
	if err := arena.live.world.ReplaceGroundSacks([]sim.Sack{{X: sack.X, Y: sack.Y, Gold: 500}}); err != nil {
		t.Fatal(err)
	}
	arena.front.ConfigureSaveSeams(arena.app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	arena.selectFirstHero(t)
	return arena, arena.heroes[0], sack
}

// A refused pick-up walk takes the hostile beside the hero; F2 and cold LOAD
// follow tick for tick (AI-375, AI-350).
func TestReleaseARefusedPickupWalkTakesTheHostileBesideTheHero(t *testing.T) {
	for _, lead := range []int{0, 2, 4} {
		arena, hero, sack := refusedPickupArena(t)
		for range lead {
			arena.live.tick()
		}
		arena.live.orderPickup(hero, sack.X, sack.Y)
		arena.live.tick()
		h := arena.get(hero)
		if !h.AcquirePursuit || !h.HasAttackTarget || h.AttackTarget != arena.east && h.AttackTarget != arena.south || h.PendingOrder.Kind != sim.PendingNone || h.HasTarget {
			t.Fatalf("lead %d: the tick the pick-up was refused left victim %v/%d acquisition %v pending %d walk %v, want a pick",
				lead, h.HasAttackTarget, h.AttackTarget, h.AcquirePursuit, h.PendingOrder.Kind, h.HasTarget)
		}
		if lead != 0 {
			continue
		}
		dir, _ := castOrderF2Save(t, arena.front, arena.app, "refused pickup")
		restored, _ := castOrderSession(t, dir)
		got := pendingVictimEntity(t, restored.live.world, hero)
		want := arena.get(hero)
		if !got.AcquirePursuit || got.AttackTarget != want.AttackTarget || got.PendingOrder.Kind != sim.PendingNone {
			t.Fatalf("cold LOAD changed the pick: %+v", got)
		}
		beforeHP := arena.get(want.AttackTarget).HP
		for range 160 {
			arena.live.tick()
			restored.live.tick()
			if !midStrikeArenaSame(arena.live.world, restored.live.world) {
				t.Fatal("installed cold next action differs")
			}
		}
		if arena.get(want.AttackTarget).HP >= beforeHP {
			t.Fatal("the hero never struck his pick")
		}
	}
}
