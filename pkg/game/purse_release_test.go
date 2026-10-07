package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The installed mission and its roster carry a queued purse request through an
// engine SAV: the save written while the request waits holds the unspent
// purse and the request, the cold LOAD applies it on its first tick, a second
// SAV holds the ground Sack, and the next pickup returns every coin.
func TestReleaseEnginePurseGoldDropSurvivesQueuedSaveLoadAndPickup(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Purse save", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("purse gold engine SAV")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	app.Layout(1024, 768)
	w := f.live.world
	subject := f.live.mission.ids[0]
	actor, ok := f.live.entity(subject)
	if !ok {
		t.Fatal("no subject actor")
	}
	before := w.Purse(sim.SelfSlot)
	w.SetPurse(sim.SelfSlot, before+2500)
	request := sim.DropGold(sim.SelfSlot, 1000, sim.CellPoint{X: actor.X, Y: actor.Y})
	f.live.pending = append(f.live.pending, request)

	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	first, err := save(true)
	if err != nil {
		t.Fatal("SAVE with a queued purse request", err)
	}
	if got := w.Purse(sim.SelfSlot); got != before+2500 || len(f.live.pending) != 1 {
		t.Fatalf("SAVE spent the request: purse %d pending %d", got, len(f.live.pending))
	}

	cold, _ := loadSAVWindow(t, store, first)
	cw := cold.live.world
	if cw.Purse(sim.SelfSlot) != before+2500 || len(cold.live.pending) != 1 || cold.live.pending[0] != request {
		t.Fatalf("cold LOAD purse %d pending %+v", cw.Purse(sim.SelfSlot), cold.live.pending)
	}
	_, sacksBefore := purseGround(t, cw)
	goldBefore, _ := purseGround(t, cw)
	cold.live.tick()
	if cw.Purse(sim.SelfSlot) != before+1500 || len(cold.live.pending) != 0 {
		t.Fatalf("first tick purse %d pending %d", cw.Purse(sim.SelfSlot), len(cold.live.pending))
	}
	if g, n := purseGround(t, cw); g != goldBefore+1000 || n < sacksBefore {
		t.Fatalf("first tick ground gold %d/%d, want +1000 over %d", g, n, goldBefore)
	}
	cold.live.tick()
	if cw.Purse(sim.SelfSlot) != before+1500 {
		t.Fatal("the second tick repeated the drop")
	}

	second, _, _ := cold.SaveSeams(store, OriginalStore{}, nil)
	name, err := second(true)
	if err != nil {
		t.Fatal("SAVE after the drop", err)
	}
	again, _ := loadSAVWindow(t, store, name)
	aw := again.live.world
	if aw.Purse(sim.SelfSlot) != before+1500 {
		t.Fatalf("second cold LOAD purse %d", aw.Purse(sim.SelfSlot))
	}
	if g, _ := purseGround(t, aw); g != goldBefore+1000 {
		t.Fatalf("second cold LOAD ground gold %d, want %d", g, goldBefore+1000)
	}
	if err := aw.TakeSack(subject, actor.X, actor.Y); err != nil {
		t.Fatal("pickup after LOAD", err)
	}
	if aw.Purse(sim.SelfSlot) < before+2500 {
		t.Fatalf("pickup returned the purse to %d, want at least %d", aw.Purse(sim.SelfSlot), before+2500)
	}
}
