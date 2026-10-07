package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The chain follows the Main line from a fresh mission 10 through 20 and 30 to
// the entry of 40. Wins use the reachability objective actions and App frames;
// it is not a combat playthrough.

// chainPayment reads a payment straight from the installed registry.
func chainPayment(t *testing.T, f *FrontEnd, mission int) int {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(ScenarioRegistry)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := reg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	payment, _ := registry.GetInt(fmt.Sprintf("Mission%d", mission), "Payment")
	return int(payment)
}

func chainCarriedNPCs(f *FrontEnd) []int {
	var npcs []int
	for _, p := range f.Carried {
		npcs = append(npcs, p.CompanionNPC)
	}
	return npcs
}

// chainTownState names the first difference from the expected town.
func chainTownState(f *FrontEnd, won, chapter, gold int, offers map[TownBuilding][]int, companion bool) error {
	if !f.Town.Open() || !f.Town.Done(won) {
		return fmt.Errorf("town open=%t done(%d)=%t", f.Town.Open(), won, f.Town.Done(won))
	}
	if f.Town.Chapter() != chapter || f.Offered != chapter {
		return fmt.Errorf("chapter=%d offered=%d, want %d", f.Town.Chapter(), f.Offered, chapter)
	}
	if f.Town.Gold() != gold {
		return fmt.Errorf("purse=%d, want %d", f.Town.Gold(), gold)
	}
	if got := f.Town.Documents(); !reflect.DeepEqual(got, documentPairs[:3]) {
		return fmt.Errorf("collected pages=%v", got)
	}
	if got := slices.Contains(chainCarriedNPCs(f), townGrantedCompanion); got != companion {
		return fmt.Errorf("companion carried=%t, want %t (carried NPCs %v)", got, companion, chainCarriedNPCs(f))
	}
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		var got []int
		for _, o := range f.Town.Offers(building) {
			if o.Mission != 0 {
				got = append(got, o.Mission)
			}
		}
		if !slices.Equal(got, offers[building]) {
			return fmt.Errorf("building %d offers missions %v, want %v", building, got, offers[building])
		}
	}
	return nil
}

// chainEnterByWorldMap accepts the offer, selects its world-map scroll and
// travels into the mission.
func chainEnterByWorldMap(t *testing.T, f *FrontEnd, app *ui.App, mission int) {
	t.Helper()
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
}

func chainRequire(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// chainWin wins mission n and returns the purse held at victory.
func chainWin(t *testing.T, f *FrontEnd, app *ui.App, n int) int {
	t.Helper()
	if f.liveMission != n {
		t.Fatalf("live mission %d, want %d", f.liveMission, n)
	}
	documentsObjective(t, f, app, n)
	if f.live.world.Outcome() != sim.OutcomeWon {
		t.Fatalf("mission %d not won", n)
	}
	return int(f.live.world.Purse(sim.SelfSlot))
}

func TestReleaseCampaignChainFromMission10(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Chain traveler", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("campaign chain")

	// Mission 10: fresh entry carries the collected pages and the document.
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	requireDocuments(t, f, true, 3)
	purse10 := chainWin(t, f, app, 10)
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("mission 10 victory did not enter mission 20", err, f.liveMission)
	}
	if !f.Town.Done(10) || f.Town.Gold() != purse10+chainPayment(t, f, 10) {
		t.Fatalf("mission 10 purse=%d, want %d + payment %d", f.Town.Gold(), purse10, chainPayment(t, f, 10))
	}
	requireDocuments(t, f, true, 3)

	// Mission point: SAVE inside mission 20, cold LOAD, continue from it.
	f, app, _ = documentsUISave(t, f, app, 3)
	if f.liveMission != 20 || !f.Town.Done(10) || f.Town.Done(20) {
		t.Fatalf("cold mission SAV: live %d done10=%t done20=%t", f.liveMission, f.Town.Done(10), f.Town.Done(20))
	}
	purse20 := chainWin(t, f, app, 20)
	campaignReturn(t, f, app)
	// Mission 20 pays its scenario payment and the 500 transition reward.
	gold20 := purse20 + chainPayment(t, f, 20) + 500
	offers30 := map[TownBuilding][]int{TownTavern: {30}, TownShop: {31}}
	chainRequire(t, chainTownState(f, 20, 30, gold20, offers30, true))
	if f.Campaign.Value().TransitionRewards[10] != 0 || f.Campaign.Value().TransitionRewards[20] != 500 {
		t.Fatal("transition reward is not the 500 paid on completing mission 20")
	}

	// Town point: SAVE in the town, cold LOAD, continue from it.
	townStore, _, townRaw := campaignSave(t, f, false)
	t.Run("loss_controls", func(t *testing.T) {
		chainLossControls(t, f, townRaw, gold20, offers30)
	})
	g, ga := campaignCold(t, townStore)
	if ga.Screen() != ui.ScreenTown {
		t.Fatal("cold town SAV did not load into the town", ga.Screen())
	}
	chainRequire(t, chainTownState(g, 20, 30, gold20, offers30, true))
	if !reflect.DeepEqual(g.Town.Available(), f.Town.Available()) {
		t.Fatalf("cold town availability %v, want %v", g.Town.Available(), f.Town.Available())
	}
	f, app = g, ga

	// Mission 30: entered from the loaded town with the companion in its party.
	chainEnterByWorldMap(t, f, app, 30)
	var mission30 []int
	for _, p := range f.live.mission.party {
		mission30 = append(mission30, p.CompanionNPC)
	}
	if !slices.Contains(mission30, townGrantedCompanion) {
		t.Fatalf("mission 30 party lacks the carried companion: %v", mission30)
	}
	requireDocuments(t, f, true, 3)
	purse30 := chainWin(t, f, app, 30)
	campaignReturn(t, f, app)
	gold30 := purse30 + chainPayment(t, f, 30)
	if f.Town.Gold() != gold30 || !f.Town.Done(30) || f.Town.Chapter() != 40 || f.Offered != 40 {
		t.Fatalf("after mission 30: purse=%d want %d done=%t chapter=%d offered=%d", f.Town.Gold(), gold30, f.Town.Done(30), f.Town.Chapter(), f.Offered)
	}
	if !slices.Contains(chainCarriedNPCs(f), townGrantedCompanion) {
		t.Fatal("mission 30 return lost the companion", chainCarriedNPCs(f))
	}
	requireDocuments(t, f, false, 3)

	// Mission 40 enters from the town the previous win left.
	chainEnterByWorldMap(t, f, app, 40)
	requireDocuments(t, f, true, 3)
}

// chainLossControls removes one carrier at a time; the town check must fail.
func chainLossControls(t *testing.T, f *FrontEnd, townRaw []byte, gold int, offers map[TownBuilding][]int) {
	t.Helper()
	t.Run("document_grant_removed", func(t *testing.T) {
		source, err := sav.Open(townRaw)
		if err != nil {
			t.Fatal(err)
		}
		campaign, _, err := source.Campaign()
		if err != nil || len(campaign.Documents) != 3 {
			t.Fatal("control lacks the three source pages", err)
		}
		if err := source.SetCampaignDocuments(campaign.Documents[:2]); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "Loss.sav"), source.Marshal(), 0600); err != nil {
			t.Fatal(err)
		}
		g, _ := campaignCold(t, SaveStore{Dir: dir})
		if chainTownState(g, 20, 30, gold, offers, true) == nil {
			t.Fatal("a town without the third page passed the chain assertions")
		}
	})
	t.Run("companion_carry_removed", func(t *testing.T) {
		saved := f.Carried
		defer func() { f.Carried = saved }()
		f.Carried = slices.DeleteFunc(slices.Clone(f.Carried), func(p mapload.PartyMember) bool { return p.CompanionNPC == townGrantedCompanion })
		if len(f.Carried) == len(saved) || chainTownState(f, 20, 30, gold, offers, true) == nil {
			t.Fatal("a party without the companion passed the chain assertions")
		}
	})
	t.Run("payment_removed", func(t *testing.T) {
		f.Town.gold -= 500
		defer func() { f.Town.gold += 500 }()
		if chainTownState(f, 20, 30, gold, offers, true) == nil {
			t.Fatal("a purse without the transition reward passed the chain assertions")
		}
	})
}
