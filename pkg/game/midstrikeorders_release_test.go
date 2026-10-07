package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMidStrikePickupCurrentSAVAppColdLoad(t *testing.T) {
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
	h := arena.get(hero)
	if h.PendingOrder.Kind != sim.PendingPickup || !h.HasAttackTarget || h.AttackPhase != sim.AttackCharging || len(arena.live.world.Sacks()) != 1 {
		t.Fatal("installed pickup input discarded old body or transferred early")
	}
	dir, raw := castOrderF2Save(t, arena.front, arena.app, "mid strike pickup")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("SAV lacks pending actions", err)
	}
	cold, _ := castOrderSession(t, dir)
	got := pendingVictimEntity(t, cold.live.world, hero)
	if got.PendingOrder != h.PendingOrder || got.AttackTarget != victim || got.AttackPhase != h.AttackPhase || got.AttackCountdown != h.AttackCountdown {
		t.Fatal("App cold LOAD changed pending pickup or physical body")
	}
	beforeHP := arena.get(victim).HP
	initialGold := arena.live.world.Purse(sim.SelfSlot)
	for range 240 {
		arena.live.tick()
		cold.live.tick()
		if !midStrikeArenaSame(arena.live.world, cold.live.world) {
			t.Fatal("installed cold next action differs")
		}
	}
	if len(arena.live.world.Sacks()) != 0 || arena.live.world.Purse(sim.SelfSlot) != initialGold+500 || arena.get(victim).HP >= beforeHP {
		t.Fatal("installed loaded blow, transfer or completion did not occur")
	}
	t.Logf("App F2/COLD LOAD retained victim=%d phase=%d countdown=%d pickup=(%d,%d); next strike and pickup completed; gold +500", victim, h.AttackPhase, h.AttackCountdown, h.PendingOrder.X, h.PendingOrder.Y)
}

func TestReleaseMidStrikeManualCastCurrentSAVAppColdLoad(t *testing.T) {
	for _, cellCast := range []bool{false, true} {
		arena := openLoadedCycleArenaChoices(t, 1, -1, []int{0, 1, 3})
		hero, victim := arena.heroes[0], arena.east
		arena.front.ConfigureSaveSeams(arena.app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
		arena.selectFirstHero(t)
		arena.orderAttack(t, victim)
		untilUnappliedStrike(t, arena, hero, victim)
		if err := arena.live.world.HeadlessDamage(hero, 10); err != nil {
			t.Fatal(err)
		}
		spell := uint32(6)
		at := sim.CellPoint{X: arena.cx - 1, Y: arena.cy}
		if cellCast {
			spell = 26
		}
		if _, _, err := arena.app.HeadlessSpellPoint(spell); err != nil {
			if err := arena.app.HeadlessKey("book"); err != nil {
				t.Fatal(err)
			}
		}
		x, y, err := arena.app.HeadlessSpellPoint(spell)
		if err != nil {
			t.Fatal(err)
		}
		arena.tap(t, x, y)
		if cellCast {
			castOrderGroundClick(t, arena.app, at)
		} else {
			x, y, err := arena.app.HeadlessEntityPoint(uint32(hero))
			if err != nil {
				t.Fatal(err)
			}
			arena.tap(t, x, y)
		}
		arena.live.tick()
		h := arena.get(hero)
		if !h.HasAttackTarget || h.AttackTarget != victim || h.AttackPhase == sim.AttackReady || h.PendingOrder.Kind == sim.PendingNone || len(arena.live.world.Actions().Books) != 0 {
			t.Fatal("installed book/pointer route discarded running strike or admitted early")
		}
		dir, raw := castOrderF2Save(t, arena.front, arena.app, "mid strike cast")
		for k := 0; k < 3 && arena.app.Screen() != ui.ScreenMap; k++ {
			if err := arena.app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil {
			t.Fatal("pending SAV supplement", err)
		}
		r := castOrderRecord(t, doc, a, hero)
		order := savedRecordRawForTest(t, r, "U158")
		if savedRecordValueForTest(t, r, "U5C") != savedRecordValueForTest(t, castOrderRecord(t, doc, a, victim), "Identity") || order[9] != 1 || order[8] != 0 || binary.LittleEndian.Uint32(order[0x30:]) == 0 {
			t.Fatal("pending SAVE replaced active victim/progress with requested cast")
		}
		cold, _ := castOrderSession(t, dir)
		admitted := false
		for range 180 {
			arena.live.tick()
			cold.live.tick()
			if !midStrikeArenaSame(arena.live.world, cold.live.world) {
				t.Fatal("pending installed cast cold action differs")
			}
			if b, ok := castOrderBook(arena.live.world, hero); ok && b.Phase == 1 {
				admitted = true
				break
			}
		}
		if !admitted {
			t.Fatal("installed cast did not admit after recovery")
		}
		if !cellCast {
			dir, raw = castOrderF2Save(t, arena.front, arena.app, "admitted unit cast")
			castOrderWire(t, raw, hero, hero, false)
			cold, _ = castOrderSession(t, dir)
			if e := pendingVictimEntity(t, cold.live.world, hero); !e.HasAttackTarget || e.AttackTarget != victim {
				t.Fatal("admitted self cast SAVE replaced the separate resume endpoint")
			}
		}
		landed := false
		for range 180 {
			arena.live.tickWithCastSink(func(events []sim.CastEvent) {
				for _, event := range events {
					landed = landed || event.Caster == hero && event.Spell == uint16(spell)
				}
			})
			cold.live.tick()
			if !midStrikeArenaSame(arena.live.world, cold.live.world) {
				t.Fatal("admitted installed cast cold action differs")
			}
			if landed {
				break
			}
		}
		if !landed || cellCast && (arena.get(hero).X != at.X || arena.get(hero).Y != at.Y) {
			t.Fatal("installed next cast did not land")
		}
	}
}

func untilUnappliedStrike(t *testing.T, arena *loadedCycleArena, hero, victim sim.EntityID) {
	t.Helper()
	for range 120 {
		arena.live.tick()
		h := arena.get(hero)
		if h.HasAttackTarget && h.AttackTarget == victim && (h.AttackPhase == sim.AttackCharging || h.AttackPhase == sim.AttackCasting) && h.AttackCountdown >= 5 {
			return
		}
	}
	t.Fatalf("the hero never held an unapplied strike on creature %d in 120 ticks", victim)
}

func midStrikeArenaSame(a, b *sim.World) bool {
	return a.Tick() == b.Tick() && reflect.DeepEqual(a.Entities(), b.Entities()) && reflect.DeepEqual(a.Actions(), b.Actions()) && reflect.DeepEqual(a.Sacks(), b.Sacks()) && a.Purse(sim.SelfSlot) == b.Purse(sim.SelfSlot)
}
