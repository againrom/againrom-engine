package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

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

func TestReleaseCityRosterMultipleHiredSquadsF2SAV(t *testing.T) {
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
			t.Fatal("live/cold mission continuation differs", tick)
		}
		sim.Step(worlds[0], nil)
		sim.Step(worlds[1], nil)
	}
	t.Logf("%s chapter %d: offered squads %v counts %v produce %d members; two F2 SAV cycles, return/rehire and next mission preserve current state", os.Getenv("AGAINROM_ASSETS"), live.Town.Chapter(), types, counts, len(wantParty))
}
