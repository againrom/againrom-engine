package game

import (
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// A pickup whose sack went before arrival is saved with F2 and cold loaded;
// both sessions walk on and complete it without a transfer, tick for tick
// (AI-645).
func TestReleaseAPickupWhoseSackWentCompletesAfterColdLoad(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	arena := openLoadedCycleArena(t, 1)
	hero := arena.heroes[0]
	sack := sim.CellPoint{X: arena.cx - 4, Y: arena.cy}
	if err := arena.live.world.ReplaceGroundSacks([]sim.Sack{{X: sack.X, Y: sack.Y, Gold: 500}}); err != nil {
		t.Fatal(err)
	}
	arena.front.ConfigureSaveSeams(arena.app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	arena.selectFirstHero(t)
	arena.live.orderPickup(hero, sack.X, sack.Y)
	arena.live.tick()
	if err := arena.live.world.ReplaceGroundSacks(nil); err != nil {
		t.Fatal(err)
	}
	arena.live.tick()
	h := arena.get(hero)
	if h.PendingOrder.Kind != sim.PendingPickup || h.PendingOrder.X != sack.X || h.PendingOrder.Y != sack.Y || h.X == sack.X && h.Y == sack.Y {
		t.Fatalf("setup: want a pending pickup still on its walk after the sack went; pending %+v at (%d,%d)", h.PendingOrder, h.X, h.Y)
	}
	dir, _ := castOrderF2Save(t, arena.front, arena.app, "vanished sack pickup")
	cold, _ := castOrderSession(t, dir)
	if got := pendingVictimEntity(t, cold.live.world, hero); got.PendingOrder != h.PendingOrder {
		t.Fatalf("cold LOAD changed the pending pickup: %+v, want %+v", got.PendingOrder, h.PendingOrder)
	}
	gold := arena.live.world.Purse(sim.SelfSlot)
	completed := -1
	for n := range 240 {
		arena.live.tick()
		cold.live.tick()
		if !midStrikeArenaSame(arena.live.world, cold.live.world) {
			t.Fatalf("tick %d: the cold-loaded session differs from the live one", n)
		}
		if e := arena.get(hero); completed < 0 && e.ActorState == 2 {
			if e.X != sack.X || e.Y != sack.Y || e.PostX != sack.X || e.PostY != sack.Y || e.PendingOrder.Kind == sim.PendingPickup {
				t.Fatalf("the pickup completed at (%d,%d) post (%d,%d) pending %+v, want the sack's cell, post there, the order ended",
					e.X, e.Y, e.PostX, e.PostY, e.PendingOrder)
			}
			completed = n
		}
	}
	if completed < 0 {
		t.Fatal("the pickup whose sack went never completed")
	}
	if arena.live.world.Purse(sim.SelfSlot) != gold || cold.live.world.Purse(sim.SelfSlot) != gold {
		t.Fatal("a pickup with no sack transferred gold")
	}
	if arena.live.pickup.set || cold.live.pickup.set {
		t.Fatal("the standing order is still armed after it completed")
	}
	e := arena.get(hero)
	t.Logf("completed on tick %d after LOAD; end state %#x pending %d attack target %v", completed, e.ActorState, e.PendingOrder.Kind, e.HasAttackTarget)
}
