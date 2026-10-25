package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// A mission-10 SAV of the corpus whose campaign holds AutoGetMission 0, where
// mission 10's own record gives 20. The two guards are dead and the witch
// walks with the party.
const (
	zeroAutoGetSAV  = "2026-09-23/exp0397-r2/game0002.sav"
	zeroAutoGetHash = "4787b1a38f7680249462ae6908ff84f805eb5b44fb96a26b6d282c23e376e939"
)

// TestReleaseLoadedZeroAutoGetMissionTravelsToMissionTen cold-LOADs that SAV
// and acknowledges mission 10's Victory. SAV-1329's travel arm takes the 0: the
// party goes to mission 10's map point, mission 10 is neither completed nor
// paid, mission 20 is not loaded and the town stays closed. Mission 10 opens
// again with the carried party. Its F2 SAVE keeps the 0 with mission 10's
// main record announced and selected, and a cold LOAD of that file and a
// second SAVE write the same campaign words.
func TestReleaseLoadedZeroAutoGetMissionTravelsToMissionTen(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	_, raw := groundCorpusFile(t, zeroAutoGetSAV, zeroAutoGetHash)
	if c := autoGetCampaign(t, raw); c.AutoGetMission != 0 || c.Main.Mission != 10 || c.SelectedMission != 10 || c.Main.Announced {
		t.Fatalf("corpus SAV holds AutoGetMission %#x, main %+v, selected %d", c.AutoGetMission, c.Main, c.SelectedMission)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(zeroAutoGetSAV)), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := autoGetSession(t, dir)
	if f.liveMission != 10 || f.live == nil {
		t.Fatalf("cold LOAD opened mission %d, want 10", f.liveMission)
	}
	won := f.live.world
	payment := f.Town.campaignPayment(10)
	// The client's own Victory, acknowledged through the map driver's
	// continuation as the Victory panel's button does: the hero's walk is
	// refused after this LOAD, so the escort cannot be played to the
	// script's win here.
	purse := int(won.Purse(sim.SelfSlot))
	f.live.mission.outcome = sim.OutcomeWon
	list := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) { return ui.NoticeToMapList, "", nil }
	dest, line, open := continueMission(frontTransitions{f}, f.liveMission, f.live.mission.state, list)()
	if dest != ui.NoticeToMission || open == nil {
		t.Fatalf("mission-10 Victory leads to %v (%q), want mission 10 opened again", dest, line)
	}
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != 10 || f.live == nil || f.live.world == won || app.Screen() != ui.ScreenMap {
		t.Fatalf("mission-10 Victory left mission %d on screen %s (same world %v); want mission 10 opened again",
			f.liveMission, app.Screen(), f.live != nil && f.live.world == won)
	}
	if f.live.world.Outcome() != sim.OutcomeUndecided || f.live.world.ScriptRegister(50) != 0 {
		t.Fatalf("reopened mission 10: outcome %v, escort counter %d; want a fresh start", f.live.world.Outcome(), f.live.world.ScriptRegister(50))
	}
	if f.Town.Done(10) || f.Town.Open() || f.Town.Chapter() != 10 || f.Town.Gold() != purse {
		t.Fatalf("after the Victory: mission 10 done %v, town open %v, chapter %d, gold %d (purse %d, payment %d); want no completion, no payment and no town",
			f.Town.Done(10), f.Town.Open(), f.Town.Chapter(), f.Town.Gold(), purse, payment)
	}

	saved, again := autoGetF2Save(t, f, app, "mission ten again")
	first := autoGetCampaign(t, again)
	if first.AutoGetMission != 0 || first.Main.Mission != 10 || !first.Main.Announced || first.SelectedMission != 10 {
		t.Fatalf("SAVE after the Victory holds AutoGetMission %#x, main %+v, selected %d; want 0 at mission 10 announced and selected",
			first.AutoGetMission, first.Main, first.SelectedMission)
	}
	g, gapp := autoGetSession(t, saved)
	if g.liveMission != 10 || g.live == nil {
		t.Fatalf("cold LOAD of the new SAV opened mission %d, want 10", g.liveMission)
	}
	_, twice := autoGetF2Save(t, g, gapp, "mission ten twice")
	second := autoGetCampaign(t, twice)
	if second.AutoGetMission != first.AutoGetMission || second.Main.Mission != first.Main.Mission ||
		second.Main.Announced != first.Main.Announced || second.SelectedMission != first.SelectedMission {
		t.Fatalf("second SAVE holds AutoGetMission %#x, main %+v, selected %d; first %#x, %+v, %d",
			second.AutoGetMission, second.Main, second.SelectedMission, first.AutoGetMission, first.Main, first.SelectedMission)
	}
	t.Logf("%s", line)
	t.Logf("Victory reopened mission 10 at %s with %d party members, gold %d, payment %d not paid; SAVE main %+v AutoGetMission %#x selected %d",
		f.live.mission.state.Address, len(f.live.mission.party), f.Town.Gold(), payment, first.Main, first.AutoGetMission, first.SelectedMission)
}
