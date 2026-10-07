package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// fallenHeroParty names the loaded mission's primary hero, the first living
// party member who knows a restorative spell, and that spell.
func fallenHeroParty(t *testing.T, f *FrontEnd) (hero, healer sim.EntityID, heal sim.SpellRule) {
	t.Helper()
	ids := f.live.mission.ids
	for _, rule := range f.live.world.Spells() {
		if !rule.Restorative {
			continue
		}
		for _, id := range ids[1:] {
			e, ok := f.live.entity(id)
			if ok && e.Alive() && e.Book.State != sim.BookAbsent && e.KnownSpells&(uint32(1)<<rule.ID) != 0 {
				return ids[0], id, rule
			}
		}
	}
	t.Fatal("the party holds no living member who knows a restorative spell")
	return 0, 0, sim.SpellRule{}
}

func requireFallenHeroNotLost(t *testing.T, f *FrontEnd, hero sim.EntityID, tick int) {
	t.Helper()
	if f.live.mission.announced {
		e, _ := f.live.entity(hero)
		t.Fatalf("settleNotices -> guardedCharacterLost announced outcome %v on tick %d while the hero lay at health %d with dwell %d: Heal can still raise him",
			f.live.mission.outcome, tick, e.HP, e.Dwell)
	}
}

// fallenHeroWalk orders walker to (x, y) through the map's move seam and ticks
// until he stands there, with the mission unannounced throughout.
func fallenHeroWalk(t *testing.T, f *FrontEnd, hero, walker sim.EntityID, x, y int32) {
	t.Helper()
	f.live.enqueue(uint32(walker), int(x), int(y))
	for tick := 1; tick <= 512; tick++ {
		f.live.tick()
		requireFallenHeroNotLost(t, f, hero, tick)
		if e, _ := f.live.entity(walker); e.Alive() && e.X == x && e.Y == y {
			return
		}
	}
	e, _ := f.live.entity(walker)
	t.Fatalf("character %d stands at (%d,%d) after 512 ticks, ordered to (%d,%d)", walker, e.X, e.Y, x, y)
}

// fallenHeroHealed ticks until hero is alive again, at most limit ticks, and
// reports whether a Heal cast by one of casters restored his health.
func fallenHeroHealed(t *testing.T, f *FrontEnd, hero sim.EntityID, heal sim.SpellRule, casters map[sim.EntityID]bool, limit int) (tick int, healed bool) {
	t.Helper()
	for tick = 1; tick <= limit; tick++ {
		f.live.tickWithCastSink(func(events []sim.CastEvent) {
			for _, ev := range events {
				healed = healed || casters[ev.Caster] && ev.Target == hero && ev.Spell == uint16(heal.ID) && ev.HealthRestored > 0
			}
		})
		requireFallenHeroNotLost(t, f, hero, tick)
		if e, _ := f.live.entity(hero); e.Alive() {
			return tick, healed
		}
	}
	e, _ := f.live.entity(hero)
	t.Fatalf("hero still fallen at health %d after %d ticks", e.HP, limit)
	return 0, false
}

// requireFallenHeroRecord requires the SAV's Human record for e to hold e's
// health at stage 1, outside the DeadActors root.
func requireFallenHeroRecord(t *testing.T, doc sav.DocumentData, e sim.Entity) {
	t.Helper()
	for i := range doc.Objects {
		r := &doc.Objects[i]
		runtime, _ := savedStructureValue(r, "RuntimeID")
		if r.Class != "Human" || runtime != e.SourceBinding.RuntimeID {
			continue
		}
		health, _ := savedStructureValue(r, "Health")
		stage, _ := savedStructureValue(r, "Stage")
		identity, _ := savedStructureValue(r, "Identity")
		if health != uint32(uint16(e.HP)) || stage != 1 || slices.Contains(roodRootKeys(t, doc), identity) {
			t.Fatalf("SAV hero Health=%#x Stage=%d rooted=%v, want Health %#x at Stage 1 outside DeadActors",
				health, stage, slices.Contains(roodRootKeys(t, doc), identity), uint16(e.HP))
		}
		return
	}
	t.Fatal("SAV holds no Human record for the hero")
}

// fallenHeroSAV walks the hero twelve cells north, out of the healer's scan
// range, fells him with the K key's command, lets him lie 128 ticks past his
// dying window and writes SAV through the ordinary producer. It returns that
// SAV and the health the hero lay at.
func fallenHeroSAV(t *testing.T, source []byte) ([]byte, int32) {
	t.Helper()
	f := loadRoodMission(t, source)
	hero, _, _ := fallenHeroParty(t, f)
	fallenHeroWalk(t, f, hero, hero, 15, 100)
	f.live.affect(uint32(hero), true)
	closed := 0
	for tick := 1; closed == 0 || tick <= closed+128; tick++ {
		f.live.tick()
		requireFallenHeroNotLost(t, f, hero, tick)
		e, _ := f.live.entity(hero)
		if e.Alive() || e.HP <= -10 || tick > 512 {
			t.Fatalf("hero at health %d on tick %d after K, want a body Heal can raise", e.HP, tick)
		}
		if closed == 0 && !e.Dying() {
			closed = tick
		}
	}
	e, _ := f.live.entity(hero)
	if e.Decay != sim.DecayFallen || e.Dwell != 0 {
		t.Fatalf("hero stage %d dwell %d, want stage 1 past the dying window", e.Decay, e.Dwell)
	}
	raw := saveRoodMission(t, f)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	requireFallenHeroRecord(t, doc, e)
	return raw, e.HP
}

// TestReleaseFallenHeroIsHealedInsteadOfLost loads the original mission-140
// SAV and fells the primary hero with the K key's command, sim.Kill. Beside
// his party, the healer's idle Heal raises him after his dying window closed.
// Out of the healer's reach, he lies past the window, survives SAVE and a cold
// LOAD, and the healer's ordered Heal raises him. Each time he walks on and
// the mission is never announced.
func TestReleaseFallenHeroIsHealedInsteadOfLost(t *testing.T) {
	source, _ := roodStageHealSource(t)
	t.Run("idle_heal", func(t *testing.T) {
		f := loadRoodMission(t, source)
		hero, _, heal := fallenHeroParty(t, f)
		party := map[sim.EntityID]bool{}
		for _, id := range f.live.mission.ids[1:] {
			party[id] = true
		}
		f.live.affect(uint32(hero), true)
		f.live.tick()
		fallen, _ := f.live.entity(hero)
		if fallen.Alive() || !fallen.Dying() {
			t.Fatalf("K left the hero at health %d dying=%v", fallen.HP, fallen.Dying())
		}
		window := 1 + int(fallen.Dwell)
		raised, healed := fallenHeroHealed(t, f, hero, heal, party, 256)
		raised++
		if !healed || raised <= window {
			t.Fatalf("hero raised on tick %d, dying window closed on tick %d, party Heal landed=%v: want a party Heal after the window", raised, window, healed)
		}
		e, _ := f.live.entity(hero)
		fallenHeroWalk(t, f, hero, hero, e.X, e.Y+2)
		t.Logf("K on tick 1, dying window closed on tick %d, party Heal raised him to health %d on tick %d", window, e.HP, raised)
	})
	t.Run("SAV_LOAD_ordered_heal", func(t *testing.T) {
		raw, hp := fallenHeroSAV(t, source)
		f := loadRoodMission(t, raw)
		hero, healer, heal := fallenHeroParty(t, f)
		if e, _ := f.live.entity(hero); e.Alive() || e.HP != hp || e.Decay != sim.DecayFallen {
			t.Fatalf("cold LOAD hero health %d stage %d, want the saved health %d at stage 1", e.HP, e.Decay, hp)
		}
		for tick := 1; tick <= 64; tick++ {
			f.live.tick()
			requireFallenHeroNotLost(t, f, hero, tick)
		}
		// A manual cast refuses a target beyond the spell's range, so the
		// healer first walks within it and then takes the Heal order.
		fallenHeroWalk(t, f, hero, healer, 16, 104)
		f.live.attackOrCast(uint32(healer), uint32(hero), uint32(heal.ID), 0, 0, false)
		raised, healed := fallenHeroHealed(t, f, hero, heal, map[sim.EntityID]bool{healer: true}, 256)
		e, _ := f.live.entity(hero)
		if !healed || e.Decay != sim.DecayNone {
			t.Fatalf("ordered Heal landed=%v, hero stage %d", healed, e.Decay)
		}
		fallenHeroWalk(t, f, hero, hero, e.X, e.Y+2)
		t.Logf("SAV at health %d; after cold LOAD the healer walked in and his Heal raised the hero to %d, %d ticks after the order", hp, e.HP, raised)
	})
}

// TestReleaseFallenHeroLosesAtRealDeath is the control on the same SAV: the
// loss still comes once Heal can no longer raise the hero. Unhealed after a
// cold LOAD of his fallen SAV, the corpse walk reaches -10 and that tick
// loses. Walked alone into the monster at (48,116), its blows fell him past
// -10 and the loss comes when his dying window closes.
func TestReleaseFallenHeroLosesAtRealDeath(t *testing.T) {
	source, _ := roodStageHealSource(t)
	t.Run("corpse_walk", func(t *testing.T) {
		raw, hp := fallenHeroSAV(t, source)
		f := loadRoodMission(t, raw)
		hero, _, _ := fallenHeroParty(t, f)
		for tick := 1; tick <= 1024; tick++ {
			f.live.tick()
			e, ok := f.live.entity(hero)
			if ok && e.HP > -10 {
				requireFallenHeroNotLost(t, f, hero, tick)
				if e.Alive() {
					t.Fatalf("hero raised on tick %d; the control needs him unhealed", tick)
				}
				continue
			}
			if !f.live.mission.announced || f.live.mission.outcome != sim.OutcomeLost {
				t.Fatalf("hero reached health %d on tick %d: announced=%v outcome=%v, want the loss on that tick",
					e.HP, tick, f.live.mission.announced, f.live.mission.outcome)
			}
			t.Logf("SAV at health %d; the corpse walk reached %d and lost on tick %d after LOAD", hp, e.HP, tick)
			return
		}
		t.Fatal("the corpse walk did not reach -10 within 1024 ticks")
	})
	t.Run("enemy_blows", func(t *testing.T) {
		f := loadRoodMission(t, source)
		hero, _, _ := fallenHeroParty(t, f)
		f.live.enqueue(uint32(hero), 46, 116)
		fell, struck := 0, false
		for tick := 1; tick <= 2048; tick++ {
			f.live.tick()
			for _, e := range f.live.world.Entities() {
				struck = struck || e.HasAttackTarget && e.AttackTarget == hero && e.Owner != sim.SelfSlot
			}
			e, ok := f.live.entity(hero)
			if fell == 0 && ok && !e.Alive() {
				fell = tick
			}
			if !f.live.mission.announced {
				continue
			}
			if f.live.mission.outcome != sim.OutcomeLost || !struck || fell == 0 || ok && e.HP > -10 {
				t.Fatalf("announced outcome %v on tick %d with the hero at health %d, fallen on tick %d, struck=%v: want the loss past -10 after enemy blows",
					f.live.mission.outcome, tick, e.HP, fell, struck)
			}
			t.Logf("enemy blows felled the hero on tick %d; lost on tick %d at health %d", fell, tick, e.HP)
			return
		}
		t.Fatal("the walk into the monster did not end the mission within 2048 ticks")
	})
}
