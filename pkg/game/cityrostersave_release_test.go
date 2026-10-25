package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityRosterF2Save(t *testing.T, app *ui.App, store SaveStore, label string) []byte {
	t.Helper()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatal("city F2 SAVE", err, app.Screen())
	}
	if err := app.HeadlessSaveEdit(store.Dir, label, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, label+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func cityLoadApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "potion.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app := openLocalTownSAV(t, f, store.Dir, "potion.sav")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	return f, app, store
}

func cityShopPointer(t *testing.T, app *ui.App, target string, index int, edges ...string) {
	t.Helper()
	x, y, err := app.HeadlessShopPoint(target, index)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func cityMissionWorldsEqual(t *testing.T, live, cold *sim.World, tick int) {
	t.Helper()
	if live.Hash() == cold.Hash() {
		return
	}
	wantEntities, gotEntities := live.Entities(), cold.Entities()
	if len(wantEntities) == len(gotEntities) {
		for i, want := range wantEntities {
			wantValue, gotValue := reflect.ValueOf(want), reflect.ValueOf(gotEntities[i])
			for field := 0; field < wantValue.NumField(); field++ {
				if wantValue.Field(field).CanInterface() && !reflect.DeepEqual(wantValue.Field(field).Interface(), gotValue.Field(field).Interface()) {
					t.Logf("actor %d %s: live=%+v cold=%+v", want.ID, wantValue.Type().Field(field).Name, wantValue.Field(field).Interface(), gotValue.Field(field).Interface())
				}
			}
		}
	}
	t.Logf("live effects=%+v cold=%+v", live.ActiveEffects(), cold.ActiveEffects())
	for _, field := range []struct {
		name       string
		live, cold any
	}{
		{"stock", live.Stock(), cold.Stock()},
		{"actions", live.Actions(), cold.Actions()},
		{"policy", live.CurrentPolicy(), cold.CurrentPolicy()},
		{"saved objects", live.SavedObjects(), cold.SavedObjects()},
		{"sacks", live.Sacks(), cold.Sacks()},
		{"item weights", live.ItemWeights(), cold.ItemWeights()},
		{"script", live.Script(), cold.Script()},
		{"cells", live.SavedCellRecords(), cold.SavedCellRecords()},
	} {
		if !reflect.DeepEqual(field.live, field.cold) {
			a, b := fmt.Sprintf("%+v", field.live), fmt.Sprintf("%+v", field.cold)
			at := 0
			for at < len(a) && at < len(b) && a[at] == b[at] {
				at++
			}
			t.Logf("%s differ at %d (lengths %d/%d): live=%s cold=%s", field.name, at, len(a), len(b), a[max(0, at-80):min(len(a), at+400)], b[max(0, at-80):min(len(b), at+400)])
		}
	}
	a, aerr := live.MarshalBinary()
	b, berr := cold.MarshalBinary()
	if aerr != nil || berr != nil {
		t.Fatal("mismatched World encoding", aerr, berr)
	}
	at := 0
	for at < len(a) && at < len(b) && a[at] == b[at] {
		at++
	}
	t.Logf("world bytes differ at %d (lengths %d/%d): live=%x cold=%x", at, len(a), len(b), a[max(0, at-16):min(len(a), at+32)], b[max(0, at-16):min(len(b), at+32)])
	t.Fatalf("uninterrupted and cold next mission differ at tick %d", tick)
}

func TestReleaseCityRosterMultipleHiredSquadsF2SAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, _ := hireForOrderTest(t, "Complete Roster")
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	app := f.App("city roster SAVE")
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("city LOAD", err)
	}
	screen := f.TownScreen().(*townScreen)
	screen.room = roomTavern
	var types, counts []int
	beforeGold := f.Town.Gold()
	spent := 0
	for _, offer := range screen.tavernMercenaries() {
		if offer.Hired || !offer.Affordable {
			continue
		}
		screen.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: offer.Type}
		screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire}, false)
		if !f.Town.MercenaryHired(offer.Type) || screen.mercenaryPartyCount(offer.Type) != offer.Count {
			t.Fatal("installed tavern Hire omitted offered squad", offer)
		}
		types, counts, spent = append(types, offer.Type), append(counts, offer.Count), spent+offer.Price
		if len(f.Carried) > 12 {
			break
		}
	}
	if len(f.Carried) <= 12 || len(types) < 2 || f.Town.Gold() != beforeGold-spent {
		t.Fatal("lawful offered squad combination did not cross twelve", types, counts, len(f.Carried), f.Town.Gold())
	}
	live := f
	wantParty := mapload.CloneParty(f.Carried)
	var groups [][]string
	for cycle := 0; cycle < 2; cycle++ {
		raw := cityRosterF2Save(t, app, store, fmt.Sprintf("roster-%d", cycle))
		gotGroups := cityRosterOrdinary(t, raw, len(wantParty))
		if cycle == 0 {
			groups = gotGroups
		} else if !reflect.DeepEqual(groups, gotGroups) {
			t.Fatal("second city SAVE moved hired groups", groups, gotGroups)
		}
		cold := currentTownReload(t, raw)
		cityRosterSame(t, live, cold)
		f = cold
		store = SaveStore{Dir: t.TempDir()}
		app = f.App("cold city roster")
		f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
		seedSave, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
		if _, err := seedSave(false); err != nil {
			t.Fatal(err)
		}
		if err := headlessOpenLoad(app); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
			t.Fatal("cold city app LOAD", err)
		}
	}
	typ := types[len(types)-1]
	for _, target := range []*FrontEnd{live, f} {
		s := target.TownScreen().(*townScreen)
		s.room = roomTavern
		for _, hired := range []bool{false, true} {
			s.tavernSelection = tavernCandidateKey{kind: tavernCandidateMercenary, id: typ}
			s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: tavernButtonHire}, false)
			if target.Town.MercenaryHired(typ) != hired {
				t.Fatal("next return/rehire action failed", typ, hired)
			}
		}
	}
	cityRosterSame(t, live, f)
	var worlds []*sim.World
	for _, target := range []*FrontEnd{live, f} {
		mission := target.App("complete roster mission")
		if err := mission.OpenMission(target.MissionOpener(target.Town.Chapter())); err != nil {
			t.Fatal("next mission", err)
		}
		if len(target.live.mission.ids) != len(wantParty) {
			t.Fatal("next mission truncated city roster", len(target.live.mission.ids), len(wantParty))
		}
		worlds = append(worlds, target.live.world)
	}
	for tick := 0; tick < 3; tick++ {
		if worlds[0].Hash() != worlds[1].Hash() {
			currentValueDiagnostics(t, "live/cold World", reflect.ValueOf(worlds[0]).Elem(), reflect.ValueOf(worlds[1]).Elem())
			t.Fatal("live/cold mission continuation differs", tick)
		}
		sim.Step(worlds[0], nil)
		sim.Step(worlds[1], nil)
	}
	t.Logf("%s chapter %d: offered squads %v counts %v produce %d members; two F2 SAV cycles, return/rehire and next mission preserve current state", os.Getenv("AGAINROM_ASSETS"), live.Town.Chapter(), types, counts, len(wantParty))
}
