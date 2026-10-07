package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// A marker selected on the world map after an original city LOAD survives SAVE
// and a cold LOAD.
func TestReleaseCitySAVKeepsAMarkerSelectedAfterTheLoad(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "source.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app := openLocalTownSAV(t, f, src, "source.sav")
	t.Cleanup(app.StopAudio)
	if f.originalCity == nil {
		t.Fatal("the loaded town carries no original city state")
	}
	// Reach a chapter whose main mission has marker art.
	if _, ok := f.Town.Won(30); !ok {
		t.Fatal("the loaded chapter's main mission refused its win")
	}
	takeCityOffers(t, f)
	if _, ok := f.Town.Won(40); !ok {
		t.Fatal("chapter 40 refused its win")
	}
	takeCityOffers(t, f)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	view := s.WorldMapView()
	data := f.worldMapAssets().data
	loaded := map[int]bool{}
	for _, m := range f.Town.selectedMarkerMissions() {
		loaded[m] = true
	}
	mission := 0
	for _, m := range view.Missions {
		if m.Enabled && !loaded[m.Number] && worldMapMarkerHasPicture(data, m.Number) {
			mission = m.Number
			break
		}
	}
	if mission == 0 {
		t.Fatal("no unselected picture-bearing mission on the gates")
	}
	index := worldMarkerMissionIndex(t, view, mission)
	if view.Missions[index].Marker != nil {
		t.Fatalf("mission %d shows a marker before it is selected", mission)
	}
	// Control: a SAVE before the selection loads without the marker.
	before := saveCityThroughDialog(t, f)
	g0 := releaseFront(t)
	c0 := openLocalTownSAV(t, g0, filepath.Dir(before), filepath.Base(before))
	t.Cleanup(c0.StopAudio)
	if err := c0.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	pre := g0.TownScreen().(*townScreen).WorldMapView()
	if pre.Missions[worldMarkerMissionIndex(t, pre, mission)].Marker != nil {
		t.Fatalf("control: mission %d shows a marker in a SAVE written before it was selected", mission)
	}
	x, y, err := app.HeadlessWorldMapMissionPoint(mission)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if s.WorldMapView().Selected != index || s.WorldMapView().Missions[index].Marker == nil {
		t.Fatal("the pointer selection did not show the mission's marker")
	}

	path := saveCityThroughDialog(t, f)
	dir := filepath.Dir(path)
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(written)
	if err != nil {
		t.Fatal(err)
	}
	campaign, _, err := file.Campaign()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range campaign.Markers {
		found = found || int(m.Value) == mission
	}
	if !found {
		t.Fatalf("SAV markers %+v omit the selected mission %d", campaign.Markers, mission)
	}

	g := releaseFront(t)
	cold := openLocalTownSAV(t, g, dir, filepath.Base(path))
	t.Cleanup(cold.StopAudio)
	if err := cold.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	back := g.TownScreen().(*townScreen).WorldMapView()
	if back.Missions[worldMarkerMissionIndex(t, back, mission)].Marker == nil {
		t.Fatalf("mission %d lost its marker across SAVE and cold LOAD", mission)
	}
	for m := range loaded {
		if worldMapMarkerHasPicture(g.worldMapAssets().data, m) && back.Missions[worldMarkerMissionIndex(t, back, m)].Marker == nil {
			t.Fatalf("loaded marker %d lost across SAVE and cold LOAD", m)
		}
	}
}

func takeCityOffers(t *testing.T, f *FrontEnd) {
	t.Helper()
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission != 0 {
				if _, ok := f.Town.Take(building, offer.Index); !ok {
					t.Fatalf("offer for mission %d refused", offer.Mission)
				}
			}
		}
	}
}

// saveCityThroughDialog writes the town as SAV through the save dialog.
func saveCityThroughDialog(t *testing.T, f *FrontEnd) string {
	t.Helper()
	dir := t.TempDir()
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: dir, Name: "Marker city", Format: ui.SaveSAV,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 {
		t.Fatal(paths, err)
	}
	return paths[0]
}
