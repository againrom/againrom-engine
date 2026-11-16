package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The chain follows the Main line from a fresh mission 10 to the end of the
// campaign and enters each side mission a town offers from a cold LOAD of that
// town. Wins use the reachability objective actions and App frames; it is not a
// combat playthrough. The registry is read through installedCampaign.

// chainPayment is a mission's scenario payment in the installed registry.
func chainPayment(t *testing.T, f *FrontEnd, mission int) int {
	t.Helper()
	return installedCampaign(t, f).Chapters[mission].Payment
}

func chainCarriedNPCs(f *FrontEnd) []int {
	var npcs []int
	for _, p := range f.Carried {
		npcs = append(npcs, p.CompanionNPC)
	}
	return npcs
}

func chainMissionNPCs(f *FrontEnd) []int {
	var npcs []int
	for _, p := range f.live.mission.party {
		npcs = append(npcs, p.CompanionNPC)
	}
	return npcs
}

// chainEnterByWorldMap accepts the offer and travels into the mission by its
// world-map scroll.
func chainEnterByWorldMap(t *testing.T, f *FrontEnd, app *ui.App, mission int) {
	t.Helper()
	carried := chainSorted(chainCarriedNPCs(f))
	takeCampaignOffer(t, f, mission)
	if !slices.Contains(f.Town.Available(), mission) {
		t.Fatalf("accepted mission %d is not available: %v", mission, f.Town.Available())
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal("gates", err)
	}
	screen := f.TownScreen().(*townScreen)
	view := screen.WorldMapView()
	if !screen.AtWorldMap() || !view.AtHome || view.HideScrolls {
		t.Fatal("gates did not open the world map at home with usable scrolls")
	}
	var scrolls []int
	for _, m := range view.Missions {
		scrolls = append(scrolls, m.Number)
		if m.Payment != chainPayment(t, f, m.Number) {
			t.Fatalf("world-map scroll %d shows payment %d, installed %d", m.Number, m.Payment, chainPayment(t, f, m.Number))
		}
	}
	slices.Sort(scrolls)
	if !slices.Equal(scrolls, f.Town.Available()) {
		t.Fatalf("world-map scrolls %v, accepted missions %v", scrolls, f.Town.Available())
	}
	selected := false
	for range view.Missions {
		screen.WorldMapMove(1)
		view = screen.WorldMapView()
		if view.Selected >= 0 && view.Selected < len(view.Missions) && view.Missions[view.Selected].Number == mission {
			selected = view.Missions[view.Selected].Enabled
			break
		}
	}
	if !selected {
		t.Fatalf("world map offers no enabled scroll for mission %d", mission)
	}
	var action ui.TownAction
	for i := 0; i < 4000 && action.Open == nil && screen.AtWorldMap(); i++ {
		action = screen.WorldMapTick()
	}
	if action.Open == nil {
		t.Fatalf("world map travel to mission %d has no opener", mission)
	}
	if err := app.OpenMission(action.Open); err != nil || f.liveMission != mission {
		t.Fatalf("mission %d entry from the world map: %v live %d", mission, err, f.liveMission)
	}
	if got := chainSorted(chainMissionNPCs(f)); !slices.Equal(got, carried) {
		t.Fatalf("mission %d party companions %v, carried %v", mission, got, carried)
	}
}

func chainSorted(npcs []int) []int {
	out := slices.Clone(npcs)
	slices.Sort(out)
	return out
}

// chainScriptMoney is the Give Money the chain's objective actions fire in a
// mission; chainGain requires the installed script to hold it. Mission 30 and
// mission 131 hold a Give Money of 1000 and 10000 that those actions do not fire.
var chainScriptMoney = map[int]int{141: 1500000}

// chainEarnedPurse is the gold missions 20 and 40 add without a script grant. The
// installed data does not state it; it is pinned and equal on EN and RU.
var chainEarnedPurse = map[int]int{20: 500, 40: 783}

func chainPurse(f *FrontEnd) int { return int(f.live.world.Purse(sim.SelfSlot)) }

// chainRequirePurse requires the live mission's purse to hold want.
func chainRequirePurse(t *testing.T, f *FrontEnd, want int, where string) {
	t.Helper()
	if err := chainPurseError(f, want); err != nil {
		t.Fatalf("%s: %v", where, err)
	}
}

func chainPurseError(f *FrontEnd, want int) error {
	if got := chainPurse(f); got != want {
		return fmt.Errorf("mission %d purse %d, want %d", f.liveMission, got, want)
	}
	return nil
}

// chainGain is the purse a mission adds from entry to Victory under the
// chain's objective actions: its fired Give Money, which must be one the
// installed script holds, plus the pinned earnings.
func chainGain(t *testing.T, f *FrontEnd, n int) int {
	t.Helper()
	if money := chainScriptMoney[n]; money != 0 {
		src, err := f.live.mission.state.Map.Script()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, a := range src.Actions {
			found = found || a.Opcode == 23 && int(a.Value[1]) == money
		}
		if !found {
			t.Fatalf("mission %d script holds no Give Money of %d", n, money)
		}
	}
	return chainScriptMoney[n] + chainEarnedPurse[n]
}

// chainWinFrom wins mission n entered with the purse entry and returns the purse
// held at victory, which is entry plus the mission's gain.
func chainWinFrom(t *testing.T, f *FrontEnd, app *ui.App, n, entry int) int {
	t.Helper()
	if f.liveMission != n {
		t.Fatalf("live mission %d, want %d", f.liveMission, n)
	}
	if n == 130 && slices.Contains(chainMissionNPCs(f), shippedTownCompanion) {
		chainBetrayal(t, f, app)
	} else {
		documentsObjective(t, f, app, n)
	}
	if f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("mission %d not won", n)
	}
	chainRequirePurse(t, f, entry+chainGain(t, f, n), "purse at Victory")
	return chainPurse(f)
}

// chainBetrayal wins mission 130 with the carried companion: it walks to
// Veglud, the installed Give Unit hands it to player 8, and Veglud's side is
// defeated. Scenario 1060-130's wait wins only without the companion.
func chainBetrayal(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
	veglud, ok := refs[135]
	if !ok {
		t.Fatal("installed mission 130 lacks Veglud unit135")
	}
	var companion sim.EntityID
	for i, p := range f.live.mission.party {
		if p.CompanionNPC == shippedTownCompanion {
			companion = f.live.mission.ids[i]
		}
	}
	v, _ := f.live.entity(veglud)
	livePlaceAndWalk(t, f.live, companion, v.X+1, v.Y)
	for i := 0; i < 48; i++ {
		documentsFrame(t, f, app)
	}
	if e, _ := f.live.entity(companion); e.Owner != 8 {
		t.Fatalf("the installed Give Unit did not hand the companion to player 8: owner %d", e.Owner)
	}
	if count, err := f.live.world.HeadlessKillPlayer(8); err != nil || count == 0 {
		t.Fatal("Veglud's side", count, err)
	}
	for i := 0; i < 300; i++ {
		if _, kind, up := f.LiveNotice(); up && kind == ui.NoticeSuccess {
			return
		}
		documentsFrame(t, f, app)
	}
	t.Fatal("defeating Veglud's side showed no Victory notice")
}

// chainWin wins mission n and returns the purse held at victory.
func chainWin(t *testing.T, f *FrontEnd, app *ui.App, n int) int {
	t.Helper()
	return chainWinFrom(t, f, app, n, chainPurse(f))
}
