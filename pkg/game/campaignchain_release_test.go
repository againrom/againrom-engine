package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
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

// chainOffers is each building's mission list in a chapter: the inn pairs its
// missions with its NPCs and its zero is an NPC with nothing to give.
func chainOffers(c Campaign, chapter int) map[TownBuilding][]int {
	out := map[TownBuilding][]int{}
	ch := c.Chapters[chapter]
	for i := 0; i < min(len(ch.Inn), len(ch.InnNPC)); i++ {
		if ch.Inn[i] != 0 {
			out[TownTavern] = append(out[TownTavern], ch.Inn[i])
		}
	}
	for b, list := range map[TownBuilding][]int{TownShop: ch.Shop, TownSchool: ch.School} {
		for _, m := range list {
			if m != 0 {
				out[b] = append(out[b], m)
			}
		}
	}
	return out
}

// chainSides is a chapter's offered missions other than the chapter itself.
func chainSides(offers map[TownBuilding][]int, chapter int) []int {
	var out []int
	for _, b := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, m := range offers[b] {
			if m != chapter {
				out = append(out, m)
			}
		}
	}
	slices.Sort(out)
	return out
}

// chainPages is the collection after the record loads of every section up to
// mission: text before picture, one append deduplicated on the pair.
func chainPages(c Campaign, mission int) []Document {
	var out []Document
	for n := 1; n <= mission; n++ {
		for _, d := range []struct {
			values []int
			kind   int
		}{{c.Chapters[n].TextDocuments, DocumentText}, {c.Chapters[n].PictureDocuments, DocumentPicture}} {
			for _, v := range d.values {
				if doc := (Document{v, d.kind}); !slices.Contains(out, doc) {
					out = append(out, doc)
				}
			}
		}
	}
	return out
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

// chainJoins is the companion NPC a Main line mission's objective actions bring
// into the party: mission 40 unit 6 is npc25, mission 100 unit 245 npc24 and
// mission 140 unit 443 npc26 (HERO-JOIN-120, HERO-JOIN-121). chainLeaves is the
// companion the mission-130 betrayal takes away.
var (
	chainJoins  = map[int]int{40: 25, 100: 24, 140: 26}
	chainLeaves = map[int]int{130: shippedTownCompanion}
)

// chainTown is one expected town between missions.
type chainTown struct {
	won, chapter, gold int
	offers             map[TownBuilding][]int
	pages              []Document
	companions         []int
}

// chainTownState names the first difference from the expected town.
func chainTownState(f *FrontEnd, want chainTown) error {
	if !f.Town.Open() || !f.Town.Done(want.won) {
		return fmt.Errorf("town open=%t done(%d)=%t", f.Town.Open(), want.won, f.Town.Done(want.won))
	}
	if f.Town.Chapter() != want.chapter || f.Offered != want.chapter {
		return fmt.Errorf("chapter=%d offered=%d, want %d", f.Town.Chapter(), f.Offered, want.chapter)
	}
	if f.Town.Gold() != want.gold {
		return fmt.Errorf("purse=%d, want %d", f.Town.Gold(), want.gold)
	}
	if got := f.Town.Documents(); !reflect.DeepEqual(got, want.pages) {
		return fmt.Errorf("collected pages=%v, want %v", got, want.pages)
	}
	if !reflect.DeepEqual(want.pages, documentPairs[:len(want.pages)]) {
		return fmt.Errorf("installed pages %v are not the witnessed physical-document pages", want.pages)
	}
	if err := documentsState(f, false, len(want.pages)); err != nil {
		return err
	}
	if got := chainSorted(chainCarriedNPCs(f)); !slices.Equal(got, want.companions) {
		return fmt.Errorf("carried companion NPCs %v, want %v", got, want.companions)
	}
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		var got []int
		for _, o := range f.Town.Offers(building) {
			if o.Mission != 0 {
				got = append(got, o.Mission)
			}
		}
		if !slices.Equal(got, want.offers[building]) {
			return fmt.Errorf("building %d offers missions %v, want %v", building, got, want.offers[building])
		}
	}
	return nil
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

func chainRequire(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// chainStartPurse is the purse a new campaign holds on entering mission 10.
// The installed data does not state it; it is pinned and equal on EN and RU.
const chainStartPurse = 100

// chainScriptMoney is the Give Money the chain's objective actions fire in a
// mission; chainGain requires the installed script to hold it. Mission 30 and
// mission 131 hold a Give Money of 1000 and 10000 that those actions do not fire.
var chainScriptMoney = map[int]int{141: 1500000}

// chainEarnedPurse is the gold missions 20 and 40 add without a script grant. The
// installed data does not state it; it is pinned and equal on EN and RU.
var chainEarnedPurse = map[int]int{20: 500, 40: 783}

// chainFinalPurse is the purse at the credits, pinned beside the derivation.
const chainFinalPurse = 192883

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

// chainMissionSave saves inside the live mission, cold-loads the SAV and
// requires the loaded mission to hold the purse the live one held.
func chainMissionSave(t *testing.T, f *FrontEnd, app *ui.App, pages int) (*FrontEnd, *ui.App) {
	t.Helper()
	before := chainPurse(f)
	g, ga, _ := documentsUISave(t, f, app, pages)
	chainRequirePurse(t, g, before, "purse across the mission SAVE and cold LOAD")
	return g, ga
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

// chainColdTown saves the town, loads it cold and requires the same state and
// availability.
func chainColdTown(t *testing.T, f *FrontEnd, want chainTown) (SaveStore, []byte, *FrontEnd, *ui.App) {
	t.Helper()
	store, _, raw := campaignSave(t, f, false)
	g, ga := campaignCold(t, store)
	if ga.Screen() != ui.ScreenTown {
		t.Fatal("cold town SAV did not load into the town", ga.Screen())
	}
	chainRequire(t, chainTownState(g, want))
	if !reflect.DeepEqual(g.Town.Available(), f.Town.Available()) {
		t.Fatalf("cold town availability %v, want %v", g.Town.Available(), f.Town.Available())
	}
	return store, raw, g, ga
}

// chainMercenaries names the first type a won mission's EnableMercenary list
// leaves locked.
func chainMercenaries(f *FrontEnd, c Campaign, mission int) error {
	for _, typ := range c.Chapters[mission].EnableMercenary {
		if !f.Town.MercenaryEnabled(typ) {
			return fmt.Errorf("mission %d left mercenary type %d locked", mission, typ)
		}
	}
	return nil
}

// chainSide enters a side mission from a cold LOAD of the town that offers it,
// wins it and requires the return: its payment, its mercenary unlocks, the
// chapter kept, its offer consumed and the chapter's own mission still open.
func chainSide(t *testing.T, store SaveStore, town chainTown, side int) {
	f, app := campaignCold(t, store)
	r := installedCampaign(t, f)
	for _, typ := range r.Chapters[side].EnableMercenary {
		if f.Town.MercenaryEnabled(typ) {
			t.Fatalf("mercenary type %d is unlocked before side mission %d", typ, side)
		}
	}
	chainEnterByWorldMap(t, f, app, side)
	chainRequirePurse(t, f, town.gold, "purse on entering the side mission from the world map")
	requireDocuments(t, f, true, len(town.pages))
	purse := chainWinFrom(t, f, app, side, town.gold)
	campaignReturn(t, f, app)
	want := town
	want.won = side
	want.gold = purse + chainPayment(t, f, side)
	want.offers = map[TownBuilding][]int{}
	for b, list := range town.offers {
		want.offers[b] = slices.DeleteFunc(slices.Clone(list), func(m int) bool { return m == side })
		if len(want.offers[b]) == 0 {
			delete(want.offers, b)
		}
	}
	check := func(f *FrontEnd) error {
		if err := chainTownState(f, want); err != nil {
			return err
		}
		if f.Town.Done(town.chapter) {
			return fmt.Errorf("side mission %d completed main mission %d", side, town.chapter)
		}
		return chainMercenaries(f, r, side)
	}
	chainRequire(t, check(f))
	_, _, g, _ := chainColdTown(t, f, want)
	chainRequire(t, check(g))
	if side == 41 {
		t.Run("loss_controls", func(t *testing.T) {
			chainSideLossControls(t, g, r, check)
		})
	}
	takeCampaignOffer(t, g, town.chapter)
	if !slices.Contains(g.Town.Available(), town.chapter) {
		t.Fatalf("side mission %d removed main mission %d: %v", side, town.chapter, g.Town.Available())
	}
	t.Logf("side %d from chapter %d: purse %d, payment %d", side, town.chapter, want.gold, chainPayment(t, f, side))
}

// chainEnding requires the campaign end after the terminal Victory: the
// credits, no save point, the hall of fame and the main menu.
func chainEnding(t *testing.T, f *FrontEnd, app *ui.App, last, gold int) {
	t.Helper()
	chainRequire(t, chainEndingState(f, app, last, gold))
	t.Run("ending_loss_control", func(t *testing.T) {
		t.Run("terminal_completion_removed", func(t *testing.T) {
			delete(f.Town.won, last)
			defer func() { f.Town.won[last] = true }()
			if chainEndingState(f, app, last, gold) == nil {
				t.Fatal("a campaign without the terminal completion passed the ending assertions")
			}
		})
	})
	if err := app.HeadlessKey("enter"); err != nil || app.Screen() != ui.ScreenEnding {
		t.Fatal("one key does not end the credits on the hall of fame", app.Screen(), err)
	}
	campaignMenu(t, app)
}

func chainEndingState(f *FrontEnd, app *ui.App, last, gold int) error {
	if app.Screen() != ui.ScreenCredits {
		return fmt.Errorf("terminal Victory shows %s, not the credits", app.Screen())
	}
	if !f.Town.Done(last) || !f.completedCampaign() || f.Town.Chapter() != 0 {
		return fmt.Errorf("done(%d)=%t completed=%t chapter=%d", last, f.Town.Done(last), f.completedCampaign(), f.Town.Chapter())
	}
	if f.Town.Gold() != gold {
		return fmt.Errorf("final purse=%d, want %d", f.Town.Gold(), gold)
	}
	snapshot, _, err := f.Snapshot(false)
	if err == nil {
		if _, err = f.ExportCurrentSave(snapshot, "Completed"); err == nil {
			return fmt.Errorf("the completed campaign was written as a SAV")
		}
	}
	return nil
}

func TestReleaseCampaignChainFromMission10(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{}
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Chain traveler", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("campaign chain")
	r := installedCampaign(t, f)

	// Mission 10: fresh entry carries the collected pages and the document.
	if err := app.OpenMission(f.MissionOpenerWith(10, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	requireDocuments(t, f, true, 3)
	chainRequirePurse(t, f, chainStartPurse, "purse on entering mission 10")
	purse10 := chainWinFrom(t, f, app, 10, chainStartPurse)
	if err := app.HeadlessActivate("notice"); err != nil || f.liveMission != 20 {
		t.Fatal("mission 10 victory did not enter mission 20", err, f.liveMission)
	}
	purse := purse10 + chainPayment(t, f, 10)
	if !f.Town.Done(10) || f.Town.Gold() != purse {
		t.Fatalf("mission 10 purse=%d, want %d + payment %d", f.Town.Gold(), purse10, chainPayment(t, f, 10))
	}
	chainRequirePurse(t, f, purse, "purse on entering mission 20")
	if err := chainMercenaries(f, r, 10); err != nil {
		t.Fatal(err)
	}
	requireDocuments(t, f, true, 3)

	// Mission point: SAVE inside mission 20, cold LOAD, continue from it.
	f, app = chainMissionSave(t, f, app, 3)
	if f.liveMission != 20 || !f.Town.Done(10) || f.Town.Done(20) {
		t.Fatalf("cold mission SAV: live %d done10=%t done20=%t", f.liveMission, f.Town.Done(10), f.Town.Done(20))
	}
	purse20 := chainWinFrom(t, f, app, 20, purse)
	campaignReturn(t, f, app)
	// Mission 20 pays its scenario payment and the 500 transition reward.
	town := chainTown{won: 20, chapter: 30, gold: purse20 + chainPayment(t, f, 20) + 500, offers: chainOffers(r, 30), pages: chainPages(r, 20), companions: []int{0, shippedTownCompanion}}
	if !reflect.DeepEqual(town.offers, map[TownBuilding][]int{TownTavern: {30}, TownShop: {31}}) {
		t.Fatalf("installed chapter-30 offers %v", town.offers)
	}
	chainRequire(t, chainTownState(f, town))
	if rewards := f.Campaign.Value().TransitionRewards; !reflect.DeepEqual(rewards, map[int]int{20: 500}) {
		t.Fatal("the transition rewards are not the one 500 paid on completing mission 20", rewards)
	}

	var departed mapload.PartyMember
	// Each later mission enters from the town the previous win left, after a
	// town SAVE and cold LOAD; each side mission branches from that SAV.
	for n := 30; ; n += 10 {
		store, raw, g, ga := chainColdTown(t, f, town)
		if n == 30 {
			t.Run("loss_controls", func(t *testing.T) {
				chainLossControls(t, f, raw, town)
			})
		}
		if n == 70 {
			t.Run("picture_loss_control", func(t *testing.T) {
				chainPageLossControl(t, raw, town)
			})
		}
		for _, side := range chainSides(town.offers, n) {
			t.Run(fmt.Sprintf("side_%d", side), func(t *testing.T) {
				chainSide(t, store, town, side)
			})
		}
		t.Run(fmt.Sprintf("services_%d", n), func(t *testing.T) {
			chainServices(t, raw, town, n == chainServiceControlTown)
		})
		f, app = g, ga
		for _, p := range f.Carried {
			if p.CompanionNPC == chainLeaves[n] && chainLeaves[n] != 0 {
				departed = p
			}
		}
		chainEnterByWorldMap(t, f, app, n)
		chainRequirePurse(t, f, town.gold, "purse on entering the mission from the world map")
		pages := chainPages(r, n)
		requireDocuments(t, f, true, len(pages))
		if n >= 40 {
			// Mission point: SAVE inside the mission, cold LOAD, continue.
			f, app = chainMissionSave(t, f, app, len(pages))
			if n == 60 {
				t.Run("purse_loss_control", func(t *testing.T) {
					chainPurseLossControl(t, f, town.gold)
				})
			}
			if f.liveMission != n || !f.Town.Done(n-10) || f.Town.Done(n) || !slices.Equal(chainSorted(chainMissionNPCs(f)), town.companions) {
				t.Fatalf("cold mission %d SAV: live %d done%d=%t done=%t party %v", n, f.liveMission, n-10, f.Town.Done(n-10), f.Town.Done(n), chainMissionNPCs(f))
			}
		}
		purse := chainWinFrom(t, f, app, n, town.gold)
		gold := purse + chainPayment(t, f, n)
		if r.Last[n] {
			campaignReturn(t, f, app)
			if gold != chainFinalPurse {
				t.Fatalf("purse at the credits %d, want %d", gold, chainFinalPurse)
			}
			chainEnding(t, f, app, n, gold)
			t.Logf("campaign end after mission %d: credits, hall of fame, menu; purse %d", n, gold)
			return
		}
		campaignReturn(t, f, app)
		companions := slices.Clone(town.companions)
		if npc, ok := chainJoins[n]; ok {
			companions = chainSorted(append(companions, npc))
		}
		if npc, ok := chainLeaves[n]; ok {
			companions = slices.DeleteFunc(companions, func(c int) bool { return c == npc })
		}
		town = chainTown{won: n, chapter: n + 10, gold: gold, offers: chainOffers(r, n+10), pages: pages, companions: companions}
		t.Logf("town %d: purse %d carried NPCs %v pages %d", n+10, f.Town.Gold(), chainCarriedNPCs(f), len(f.Town.Documents()))
		chainRequire(t, chainTownState(f, town))
		chainRequire(t, chainMercenaries(f, r, n))
		if _, ok := chainLeaves[n]; ok {
			t.Run("departed_companion_kept", func(t *testing.T) {
				saved := f.Carried
				defer func() { f.Carried = saved }()
				f.Carried = append(slices.Clone(saved), departed)
				if departed.CompanionNPC != chainLeaves[n] || chainTownState(f, town) == nil {
					t.Fatal("a town still carrying the departed companion passed the chain assertions")
				}
			})
		}
		if n >= 150 {
			t.Fatal("the Main line ran past mission 150 without a LastMission section")
		}
	}
}

// chainPurseLossControl changes the loaded mission's purse by one gold either
// way; the purse check must fail each time.
func chainPurseLossControl(t *testing.T, f *FrontEnd, want int) {
	t.Helper()
	if err := chainPurseError(f, want); err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int{-1, 1} {
		if !f.live.world.SetPurse(sim.SelfSlot, uint32(want+delta)) {
			t.Fatal("the mission purse cannot be set")
		}
		failed := chainPurseError(f, want) != nil
		f.live.world.SetPurse(sim.SelfSlot, uint32(want))
		if !failed {
			t.Fatalf("a mission purse %+d from the town purse passed the purse check", delta)
		}
	}
}

// chainLossControls removes one carrier at a time; the town check must fail.
func chainLossControls(t *testing.T, f *FrontEnd, townRaw []byte, town chainTown) {
	t.Helper()
	t.Run("document_grant_removed", func(t *testing.T) {
		g := chainTrimmedPagesTown(t, townRaw, 2)
		if chainTownState(g, town) == nil {
			t.Fatal("a town without the third page passed the chain assertions")
		}
	})
	t.Run("companion_carry_removed", func(t *testing.T) {
		saved := f.Carried
		defer func() { f.Carried = saved }()
		f.Carried = slices.DeleteFunc(slices.Clone(f.Carried), func(p mapload.PartyMember) bool { return p.CompanionNPC == shippedTownCompanion })
		if len(f.Carried) == len(saved) || chainTownState(f, town) == nil {
			t.Fatal("a party without the companion passed the chain assertions")
		}
	})
	t.Run("payment_removed", func(t *testing.T) {
		f.Town.gold -= 500
		defer func() { f.Town.gold += 500 }()
		if chainTownState(f, town) == nil {
			t.Fatal("a purse without the transition reward passed the chain assertions")
		}
	})
	t.Run("offer_consumed", func(t *testing.T) {
		g := chainTrimmedPagesTown(t, townRaw, len(town.pages))
		takeCampaignOffer(t, g, 31)
		if chainTownState(g, town) == nil {
			t.Fatal("a town whose shop offer was consumed passed the chain assertions")
		}
	})
}

// chainPageLossControl drops the picture page from the town SAV.
func chainPageLossControl(t *testing.T, townRaw []byte, town chainTown) {
	g := chainTrimmedPagesTown(t, townRaw, len(town.pages)-1)
	if town.pages[len(town.pages)-1].Kind != DocumentPicture || chainTownState(g, town) == nil {
		t.Fatal("a town without the picture page passed the chain assertions")
	}
}

// chainTrimmedPagesTown cold-loads the town SAV with its collection cut to
// pages.
func chainTrimmedPagesTown(t *testing.T, townRaw []byte, pages int) *FrontEnd {
	t.Helper()
	source, err := sav.Open(townRaw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, _, err := source.Campaign()
	if err != nil || len(campaign.Documents) < pages {
		t.Fatal("control lacks the source pages", err)
	}
	if err := source.SetCampaignDocuments(campaign.Documents[:pages]); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Loss.sav"), source.Marshal(), 0600); err != nil {
		t.Fatal(err)
	}
	g, _ := campaignCold(t, SaveStore{Dir: dir})
	return g
}

// chainSideLossControls removes the side completion and the unlock it pays.
func chainSideLossControls(t *testing.T, f *FrontEnd, r Campaign, check func(*FrontEnd) error) {
	t.Run("side_completion_removed", func(t *testing.T) {
		delete(f.Town.won, 41)
		defer func() { f.Town.won[41] = true }()
		if check(f) == nil {
			t.Fatal("a town without the side completion passed the side assertions")
		}
	})
	t.Run("mercenary_unlock_removed", func(t *testing.T) {
		unlocks := f.Town.mercEnabled
		defer func() { f.Town.mercEnabled = unlocks }()
		for _, typ := range r.Chapters[41].EnableMercenary {
			f.Town.mercEnabled[typ] = false
		}
		if check(f) == nil {
			t.Fatal("a town without the side mission's mercenary unlock passed the side assertions")
		}
	})
}

// chainWin wins mission n and returns the purse held at victory.
func chainWin(t *testing.T, f *FrontEnd, app *ui.App, n int) int {
	t.Helper()
	return chainWinFrom(t, f, app, n, chainPurse(f))
}
