package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestReleaseFreshPartyBodyKeepsItsRowDwell starts a fresh mission with the
// chargen hero and a hired healer, writes SAV through the ordinary producer and
// loads it cold. Every member carries his Humans row's dying time, 12 or 8, in
// the fresh and the loaded World alike (DAT-SCHEMA-007). A killed member's body
// then lies for that dwell before teardown (HERO-DWELL-065).
func TestReleaseFreshPartyBodyKeepsItsRowDwell(t *testing.T) {
	const healer = 4
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	c := f.Campaign.Value()
	target := hurtVoiceHealerChapter(t, c, healer)
	town := NewTown(c)
	for mission := range c.Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Dwell", Choices: []int{0, 0, 4}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	f.Town.mercEnabled[healer] = true
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	s.composeShopFaces()
	if msg, ok := s.toggleMercenary(healer); !ok {
		t.Fatalf("hire type %d in chapter %d refused: %s", healer, target, msg)
	}
	app := f.App("fresh party dwell")
	app.SetCutscenes(nil)
	if err := app.OpenMission(f.MissionOpenerWith(target, f.Carried)); err != nil {
		t.Fatal(err)
	}
	ids := append([]sim.EntityID(nil), f.live.mission.ids...)
	if len(ids) < 2 {
		t.Fatalf("mission %d carries %d party members, want the hero and a hire", target, len(ids))
	}
	fresh := make([]int32, len(ids))
	for i, id := range ids {
		e, ok := f.live.entity(id)
		if !ok || e.DyingTime != 12 && e.DyingTime != 8 {
			t.Fatalf("fresh member %d (%s) dying time %d, want his row's 12 or 8", i, f.live.mission.party[i].Name, e.DyingTime)
		}
		fresh[i] = e.DyingTime
	}

	store, _, _ := campaignSave(t, f, true)
	cold, _ := campaignCold(t, store)
	for i, id := range cold.live.mission.ids {
		e, ok := cold.live.entity(id)
		if i >= len(fresh) || !ok || e.DyingTime != fresh[i] {
			t.Fatalf("loaded member %d dying time %d, fresh %v", i, e.DyingTime, fresh)
		}
	}

	victim := ids[len(ids)-1]
	dwell := int(fresh[len(ids)-1])
	f.live.pending = append(f.live.pending, sim.TerminalKill(victim))
	for tick := 1; ; tick++ {
		f.live.tick()
		e, ok := f.live.entity(victim)
		if !ok || e.Alive() {
			t.Fatalf("victim present=%v alive=%v on tick %d", ok, e.Alive(), tick)
		}
		if e.Decay != sim.DecayFallen {
			if tick != dwell {
				t.Fatalf("body torn down on tick %d, want after the row's dwell %d", tick, dwell)
			}
			t.Logf("mission %d: dying times %v; member %d torn down on tick %d", target, fresh, len(ids)-1, tick)
			return
		}
		if tick > dwell {
			t.Fatalf("body still at stage %d dwell %d on tick %d, past the row's dwell %d", e.Decay, e.Dwell, tick, dwell)
		}
	}
}
