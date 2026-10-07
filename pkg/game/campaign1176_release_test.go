package game

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// prepareAcceptedCampaignMission uses controlled prior victories and a real
// installed offer. The caller owns entry and victory.
func prepareAcceptedCampaignMission(t *testing.T, f *FrontEnd, mission int) {
	t.Helper()
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Campaign witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.arriveInTown()
	chapter := mission / 10 * 10
	for _, n := range f.Campaign.Value().Main {
		if n < chapter {
			f.Town.Won(n)
		}
	}
	if f.Town.Chapter() != chapter {
		t.Fatalf("controlled prerequisite chapter=%d, want %d", f.Town.Chapter(), chapter)
	}
	takeCampaignOffer(t, f, mission)
}

func takeCampaignOffer(t *testing.T, f *FrontEnd, mission int) {
	t.Helper()
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission == mission {
				if got, ok := f.Town.Take(building, offer.Index); !ok || got != mission {
					t.Fatal("actual mission offer refused", got, ok)
				}
				f.TownScreen().(*townScreen).markWorldSelected(mission)
				return
			}
		}
	}
	t.Fatalf("mission %d has no actual offer at chapter %d", mission, f.Town.Chapter())
}

func campaignWin1176(t *testing.T, f *FrontEnd, app *ui.App, mission int) {
	t.Helper()
	w := f.live.world
	if w.Outcome() != sim.OutcomeUndecided {
		t.Fatal("mission was already terminal before its objective action")
	}
	if mission == 150 {
		refs := mapload.ScriptUnits(f.live.mission.state.Map, f.live.mission.party)
		id, ok := refs[188]
		if !ok {
			t.Fatal("installed final mission lacks script unit 188")
		}
		if err := w.HeadlessKill(id); err != nil {
			t.Fatal(err)
		}
	} else {
		for _, player := range []uint32{3, 4} {
			if count, err := w.HeadlessKillPlayer(player); err != nil || count == 0 {
				t.Fatal("side-mission objective action", player, count, err)
			}
		}
	}
	for i := 0; i < 300; i++ {
		if _, kind, up := f.LiveNotice(); up {
			if kind == ui.NoticeSuccess {
				if w.Outcome() != sim.OutcomeWon {
					t.Fatal("Victory notice lacks script victory")
				}
				return
			}
			if err := app.HeadlessActivate("notice"); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("mission %d never displayed script Victory: outcome=%v", mission, w.Outcome())
}

// SaveDialogSeams is also used while the Victory modal holds input: that
// checkpoint measures pending-outcome persistence, not permission to open the
// player dialog over a modal. Cold LOAD below always uses the actual App.
func campaignSave(t *testing.T, f *FrontEnd, onMap bool) (SaveStore, Snapshot, []byte) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	before, _, err := f.Snapshot(onMap)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: store.Dir, Name: "Campaign checkpoint", Format: ui.SaveSAV, OnMap: onMap,
	})
	if err != nil {
		t.Fatal("prepare checkpoint", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal("publish checkpoint", paths, err)
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	cold, _ := campaignCold(t, store)
	written, _, err := cold.Snapshot(onMap)
	worldEqual := bytes.Equal(before.World, written.World)
	if !worldEqual {
		worldEqual = bytes.Equal(worldBytesWithConstructedCurrentSessionHead1115(t, before.World), written.World)
	}
	if err != nil || !worldEqual || !reflect.DeepEqual(before.Documents, written.Documents) || before.Gold != written.Gold {
		t.Logf("checkpoint components: World equal=%t, Documents equal=%t, Gold=%d -> %d", worldEqual, reflect.DeepEqual(before.Documents, written.Documents), before.Gold, written.Gold)
		if !worldEqual {
			var left, right sim.World
			leftErr, rightErr := left.UnmarshalBinary(before.World), right.UnmarshalBinary(written.World)
			if leftErr == nil && rightErr == nil {
				currentMenuWorldDiagnostics(t, &left, &right)
			} else {
				t.Log("checkpoint World diagnostic decode", leftErr, rightErr)
			}
		}
		currentItemFieldDiagnostics(t, "Documents", reflect.ValueOf(before.Documents), reflect.ValueOf(written.Documents))
		t.Fatal("written SAV differs from the current world, documents or purse", err)
	}
	return store, before, raw
}

func campaignCold(t *testing.T, store SaveStore) (*FrontEnd, *ui.App) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("1176 cold LOAD")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if legacy, _ := listAGS(store); len(legacy) != 0 {
		save, list, load := agsSaveSeams(f, store, OriginalStore{}, nil)
		app.SetSaveSeams(save, list, load)
	}
	if err := app.HeadlessActivate("load game"); err != nil {
		t.Fatal("cold main-menu LOAD", err)
	}
	rows := app.HeadlessRows()
	if app.Screen() != ui.ScreenLoad || len(rows) != 1 {
		t.Fatal("cold LOAD must see exactly the written file", app.Screen(), rows)
	}
	if err := app.HeadlessActivate(rows[0].Text); err != nil {
		t.Fatal("cold selected LOAD", err)
	}
	if app.Screen() == ui.ScreenLoad {
		t.Fatal("cold LOAD refused the written SAV", app.HeadlessMessage())
	}
	return f, app
}

func campaignMenu1176(t *testing.T, app *ui.App) {
	t.Helper()
	if app.Screen() == ui.ScreenEnding || app.Screen() == ui.ScreenCredits {
		for i := 0; i < 3 && (app.Screen() == ui.ScreenEnding || app.Screen() == ui.ScreenCredits); i++ {
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
		}
		if app.Screen() != ui.ScreenMenu {
			t.Fatal("ending main menu", app.Screen())
		}
		return
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatal("recovery game menu", app.Screen(), err)
	}
	if err := app.HeadlessGameMenuAction("abort"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("confirm-abort"); err != nil || app.Screen() != ui.ScreenMenu {
		t.Fatal("recovery main menu", app.Screen(), err)
	}
}

func campaignReturn(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal("Victory acknowledgment", err)
	}
	if f.completedCampaign() {
		if app.Screen() != ui.ScreenCredits {
			t.Fatal("terminal Victory did not show the ending credits", app.Screen())
		}
		return
	}
	for i := 0; i < 4000 && (app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare()); i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal("return frame", err)
		}
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
		t.Fatalf("Victory has no usable recovery: screen=%s town=%v", app.Screen(), f.townUI.AtTownSquare())
	}
}

func TestReleaseCampaign1176TerminalAndSideRecovery(t *testing.T) {
	for _, mission := range []int{150, 151} {
		t.Run(fmt.Sprint(mission), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			// Read the installed routing key independently of Campaign's
			// successor logic. Only 150 satisfies SAV-890's two predicates.
			raw, err := f.Archives.Containers.ReadFile(ScenarioRegistry)
			if err != nil {
				t.Fatal(err)
			}
			registry, err := reg.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			last, _ := registry.GetInt(fmt.Sprintf("Mission%d", mission), "LastMission")
			if (last != 0 && mission%10 == 0) != (mission == 150) {
				t.Fatal("installed terminal predicate changed", mission, last)
			}
			prepareAcceptedCampaignMission(t, f, mission)
			app := f.App("1176 campaign recovery")
			if err := app.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
				t.Fatal("offered mission entry", err)
			}
			campaignWin1176(t, f, app, mission)
			purse := int(f.live.world.Purse(sim.SelfSlot))
			pendingStore, _, _ := campaignSave(t, f, true)
			cold, coldApp := campaignCold(t, pendingStore)
			for i := 0; i < 5; i++ {
				if _, kind, up := cold.LiveNotice(); up && kind == ui.NoticeSuccess {
					break
				}
				if err := coldApp.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
			}
			if _, kind, up := cold.LiveNotice(); !up || kind != ui.NoticeSuccess || cold.Town.Done(mission) {
				t.Fatal("cold pending Victory was lost or paid before acknowledgment", up, kind)
			}
			campaignReturn(t, f, app)
			campaignReturn(t, cold, coldApp)
			if !f.Town.Done(mission) {
				t.Fatal("acknowledgment lost the completed mission")
			}
			// Side151 remains ordinary: independently read the installed
			// Payment, and require unfinished main150 to stay accessible.
			if mission == 151 {
				payment, _ := registry.GetInt("Mission151", "Payment")
				if f.Town.Gold() != purse+int(payment) || f.Town.Chapter() != 150 || f.Town.Done(150) {
					t.Fatal("side151 changed ordinary payout or main150", f.Town.Gold(), purse, payment, f.Town.Chapter())
				}
				takeCampaignOffer(t, f, 150)
				takeCampaignOffer(t, cold, 150)
				if !slices.Contains(f.Town.Available(), 150) {
					t.Fatal("ordinary side151 removed unfinished main150")
				}
			}
			gold := f.Town.Gold()
			presses := 3
			if mission == 150 {
				presses = 1 // one key ends the credits; more would press the hall's button
			}
			for i := 0; i < presses; i++ {
				if err := app.HeadlessKey("enter"); err != nil {
					t.Fatal(err)
				}
			}
			if f.Town.Gold() != gold || cold.Town.Gold() != gold || cold.Town.Chapter() != f.Town.Chapter() {
				t.Fatal("repeated acknowledgment or cold transition changed campaign payment", gold, f.Town.Gold(), cold.Town.Gold())
			}
			var reopened *FrontEnd
			var reopenedApp *ui.App
			if mission == 150 {
				if snapshot, _, err := f.Snapshot(false); err != nil {
					t.Fatal(err)
				} else if _, err := f.ExportCurrentSave(snapshot, "Completed"); err == nil {
					t.Fatal("the completed campaign was written as a SAV")
				}
				if app.Screen() != ui.ScreenEnding || !f.Town.Done(mission) {
					t.Fatal("completed campaign lost its hall", app.Screen())
				}
			} else {
				settledStore, settled, _ := campaignSave(t, f, false)
				reopened, reopenedApp = campaignCold(t, settledStore)
				if reopenedApp.Screen() != ui.ScreenTown || !reopened.Town.Done(mission) || reopened.Town.Gold() != gold || !reflect.DeepEqual(reopened.Town.Available(), f.Town.Available()) {
					t.Fatal("cold settled SAV changed completion, purse or availability", reopenedApp.Screen())
				}
				if settled.Mission != 0 || settled.World != nil {
					t.Fatal("town checkpoint accidentally retained completed World")
				}
			}
			campaignMenu1176(t, app)
			if reopenedApp != nil {
				campaignMenu1176(t, reopenedApp)
			}
			if mission == 151 {
				if err := reopenedApp.OpenMission(reopened.MissionOpenerWith(150, reopened.NextParty())); err != nil || reopened.liveMission != 150 {
					t.Fatal("cold side151 completion cannot enter offered main150", err)
				}
			}
			t.Logf("mission %d: source LastMission=%d; script Victory, pending cold SAV, one transition, ending or town and menu usable; purse=%d", mission, last, gold)
		})
	}
}
