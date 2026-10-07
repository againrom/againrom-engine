package game

import (
	"testing"

	"againrom/pkg/sim"
)

// Hold Position pressed on the command panel while a warrior's blow is loaded
// and a pickup waits behind it. The Stand Ground setter stores the pending order
// 0 and no progress: the loaded blow lands on its victim, and the pickup the
// setter replaced never runs (AI-351, AI-352, AI-354, AI-356). The mission is
// saved through F2 after the press and loaded cold through the main menu; the
// loaded world matches the live one tick for tick, the sack stays on the
// ground and the purse is unchanged. The control, in
// TestReleaseMidStrikePickupCurrentSAVAppColdLoad, is the same setup without the
// press: the pickup transfers.
func TestReleaseHoldStrikeReplacesPickupSAVAppColdLoad(t *testing.T) {
	arena := openLoadedCycleArena(t, 1)
	hero, victim := arena.heroes[0], arena.east
	arena.front.ConfigureSaveSeams(arena.app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	arena.selectFirstHero(t)
	arena.orderAttack(t, victim)
	arena.untilLoaded(t, hero, victim)
	if err := arena.live.world.ReplaceGroundSacks([]sim.Sack{{X: arena.cx, Y: arena.cy, Gold: 500}}); err != nil {
		t.Fatal(err)
	}
	arena.live.orderPickup(hero, arena.cx, arena.cy)
	arena.live.tick()
	if h := arena.get(hero); h.PendingOrder.Kind != sim.PendingPickup || !h.HasAttackTarget || h.AttackPhase != sim.AttackCharging {
		t.Fatal("fixture: the pickup is not waiting behind the loaded blow")
	}
	held := arena.get(hero)
	holdPress(t, arena, hero)
	h := arena.get(hero)
	if h.PendingOrder.Kind != sim.PendingNone || !h.HasAttackTarget || h.AttackTarget != victim || h.AttackPhase == sim.AttackReady {
		t.Fatal("Hold Position did not replace the pickup and keep the loaded blow")
	}
	if h.AttackCountdown > held.AttackCountdown {
		t.Fatal("Hold Position restarted the loaded blow")
	}
	dir, _ := castOrderF2Save(t, arena.front, arena.app, "hold strike pickup")
	cold, _ := castOrderSession(t, dir)
	got := pendingVictimEntity(t, cold.live.world, hero)
	if got.PendingOrder.Kind != sim.PendingNone || got.AttackTarget != victim || got.AttackPhase != h.AttackPhase || got.AttackCountdown != h.AttackCountdown {
		t.Fatal("App cold LOAD changed the held warrior's blow or restored the pickup")
	}
	beforeHP := arena.get(victim).HP
	gold := arena.live.world.Purse(sim.SelfSlot)
	for range 240 {
		arena.live.tick()
		cold.live.tick()
		if !midStrikeArenaSame(arena.live.world, cold.live.world) {
			t.Fatal("installed cold next action differs")
		}
	}
	if len(arena.live.world.Sacks()) != 1 || arena.live.world.Purse(sim.SelfSlot) != gold || arena.get(victim).HP >= beforeHP {
		t.Fatal("the pickup ran after Hold Position or the loaded blow did not land")
	}
	t.Logf("App Hold, F2, cold LOAD: victim=%d phase=%d countdown=%d; blow landed, pickup replaced", victim, h.AttackPhase, h.AttackCountdown)
}
