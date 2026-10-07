package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseTwoSavePointsMissionAndCity(t *testing.T) {
	missionSave := func(f *FrontEnd, mission int) {
		t.Helper()
		if f.Town.Open() {
			t.Fatal("city exists before mission30")
		}
		if _, _, err := f.Snapshot(false); err == nil {
			t.Fatal("pre-city off-map Snapshot succeeded")
		}
		store := SaveStore{Dir: t.TempDir()}
		save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
		name, err := save(true)
		if err != nil {
			t.Fatal("mission SAVE", mission, err)
		}
		raw, err := store.Read(name)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil || int(doc.Head.Mission) != mission || doc.World == nil {
			t.Fatal("mission wrote another save point", mission, err)
		}
	}
	f := loadedMission20(t, func(front *FrontEnd, _ sim.EntityID) { missionSave(front, 10) })
	missionSave(f, 20)
	if _, _, err := f.LiveCompleteCampaign(); err != nil {
		t.Fatal(err)
	}
	if !f.Town.Open() || f.Town.Chapter() != 30 {
		t.Fatal("first city did not begin at mission30")
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal("initial city SAVE", err)
	}
	app := f.App("two save points")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("city LOAD", err)
	}
	screen := f.TownScreen().(*townScreen)
	screen.beginWorldMapReturn(20)
	if screen.worldMap == nil || screen.worldMap.returnMission != 20 {
		t.Fatal("installed return route is absent")
	}
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("pending F2 opened SAVE", err, app.Screen())
	}
	if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenLoad {
		t.Fatal("pending return lost F3", err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("LOAD did not return to menu")
	}
	if err := app.HeadlessGameMenuAction("save"); err == nil {
		t.Fatal("pending return enabled menu SAVE")
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	gold := f.Town.Gold()
	screen.Back()
	if !screen.AtTownSquare() || f.Town.Gold() != gold {
		t.Fatal("ordinary return completion changed purse")
	}
	for i, room := range []townRoom{roomSquare, roomShop, roomTavern, roomSchool, roomGates} {
		screen.room = room
		if room == roomGates {
			screen.enterWorldMap()
		}
		if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
			t.Fatal("resolved city room lost F2", room, err)
		}
		label := "city-point-" + string(rune('a'+i))
		if err := app.HeadlessSaveEdit(store.Dir, label, ui.SaveSAV); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessSaveAction("save"); err != nil {
			t.Fatal("resolved city SAVE", room, err)
		}
		raw, err := os.ReadFile(filepath.Join(store.Dir, label+".sav"))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil || doc.Head.Mission != 0 || doc.World != nil {
			t.Fatal("city room wrote another save point", room, err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a.WorldMapReturn != nil {
			t.Fatal("city encoded travel", room, err)
		}
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	accepted := false
	for _, offer := range f.Town.Offers(TownTavern) {
		if offer.Mission > 0 {
			_, accepted = f.Town.Take(TownTavern, offer.Index)
			break
		}
	}
	if !accepted {
		t.Fatal("city has no acceptable tavern mission")
	}
	screen.enterWorldMap()
	selected := -1
	for i, mission := range screen.worldMap.missions {
		if mission.Enabled {
			selected = i
			break
		}
	}
	if selected < 0 {
		t.Fatal("city has no outward mission")
	}
	screen.selectWorldMission(selected)
	if len(screen.worldMap.route) == 0 {
		t.Fatal("outward route is absent")
	}
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("outward city lost F2", err)
	}
	if err := app.HeadlessSaveEdit(store.Dir, "outward-city", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, "outward-city.sav"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.Head.Mission != 0 || doc.World != nil {
		t.Fatal("outward route wrote a third save point", err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a.WorldMapReturn != nil {
		t.Fatal("outward route encoded travel", err)
	}
	control := releaseFront(t)
	if _, city, err := control.RestoreOriginal(raw); err != nil || !city {
		t.Fatal("outward city LOAD", city, err)
	}
	want, _, err := control.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	legacy := savePointLegacySAV(t, raw, 20)
	legacyStore := SaveStore{Dir: t.TempDir()}
	legacyName := filepath.Join(legacyStore.Dir, "legacy-return.sav")
	if err := os.WriteFile(legacyName, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	cold := releaseFront(t)
	coldApp := cold.App("legacy city arrival")
	cold.ConfigureSaveSeams(coldApp, legacyStore, OriginalStore{}, nil)
	if err := headlessOpenLoad(coldApp); err != nil {
		t.Fatal(err)
	}
	if err := coldApp.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	settled := cold.TownScreen().(*townScreen)
	if coldApp.Screen() != ui.ScreenTown || !settled.AtTownSquare() {
		t.Fatal("legacy LOAD did not settle in city")
	}
	after, _, err := cold.Snapshot(false)
	if err != nil || after.WorldMapReturn != nil {
		t.Fatal("legacy LOAD retained travel", err)
	}
	if !reflect.DeepEqual(after.Party, want.Party) || after.Gold != want.Gold || !reflect.DeepEqual(after.Campaign, want.Campaign) || !reflect.DeepEqual(after.Documents, want.Documents) {
		t.Fatal("legacy LOAD changed party, purse, journal or campaign")
	}
	for range 3 {
		settled.WorldMapTick()
	}
	if cold.Town.Gold() != want.Gold {
		t.Fatal("legacy return repeated reward")
	}
	resolvedStore := SaveStore{Dir: t.TempDir()}
	if err := coldApp.HeadlessKey("f2"); err != nil || coldApp.Screen() != ui.ScreenSave {
		t.Fatal("resolved legacy F2", err)
	}
	if err := coldApp.HeadlessSaveEdit(resolvedStore.Dir, "resolved", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := coldApp.HeadlessSaveAction("save"); err != nil {
		t.Fatal("resolved legacy SAVE", err)
	}
	resumed := releaseFront(t)
	resumedApp := resumed.App("resolved city")
	resumed.ConfigureSaveSeams(resumedApp, resolvedStore, OriginalStore{}, nil)
	if err := headlessOpenLoad(resumedApp); err != nil {
		t.Fatal(err)
	}
	if err := resumedApp.HeadlessActivate("@first"); err != nil || resumedApp.Screen() != ui.ScreenTown {
		t.Fatal("resolved city cold LOAD", err)
	}
	if !resumed.TownScreen().(*townScreen).AtTownSquare() || resumed.Town.Gold() != want.Gold {
		t.Fatal("second LOAD resumed travel or changed purse")
	}
	if err := resumedApp.HeadlessActivate("GATES"); err != nil {
		t.Fatal("next city action", err)
	}
	view := resumed.TownScreen().(*townScreen).WorldMapView()
	if !view.AtHome || view.Returning {
		t.Fatal("next city action restarted legacy return")
	}
	t.Log("mission10/20 SAVE; pending F2/menu denied; F3 retained; city square/shop/tavern/school/idle and outward map SAVE; legacy pending LOAD resolves without party, purse, journal or campaign loss")
	if _, err := store.Read(name); err != nil {
		t.Fatal("original city slot disappeared", err)
	}
}
