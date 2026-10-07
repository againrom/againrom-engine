package game

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/formats/spr256"
	"againrom/pkg/ui"
)

func TestReleaseNativeCityMarkerResourcePaths(t *testing.T) {
	for _, onMap := range []bool{false, true} {
		name := "city"
		if onMap {
			name = "mission"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			_, screen := reachabilityWalkArrive(t, f, "Marker traveler")
			for mission := 10; mission < 100; mission += 10 {
				f.Town.CollectDocuments(mission)
				f.Town.Won(mission)
			}
			f.arriveInTown()
			f.addChapterCompanions(f.Town.Chapter())
			data := f.worldMapAssets().data
			if data == nil {
				t.Fatal("no installed world-map registry")
			}
			var missions []int
			for mission := range data.Missions {
				missions = append(missions, mission)
			}
			slices.Sort(missions)
			selected := map[string]bool{}
			for _, mission := range missions {
				row := data.Objects[data.Missions[mission]]
				if !row.Valid || row.Picture == "" || strings.EqualFold(row.Picture, "nothing") || selected[row.Picture] {
					continue
				}
				selected[row.Picture] = true
				screen.markWorldSelected(mission)
			}
			for _, picture := range []string{"onmap03", "onmap13", "onmap14", "onmap18", "onmap19"} {
				if !selected[picture] {
					t.Fatalf("missing installed marker %s", picture)
				}
			}
			if onMap {
				app := f.App("marker mission")
				if err := app.OpenMission(f.MissionOpenerWith(100, f.NextParty())); err != nil {
					t.Fatal(err)
				}
			}
			before, _, err := f.Snapshot(onMap)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
				Directory: dir, Name: "Marker city", Format: ui.SaveSAV, OnMap: onMap,
			})
			if err != nil {
				t.Fatal(err)
			}
			paths, err := prepared.Commit(false)
			if err != nil || len(paths) != 1 {
				t.Fatal(paths, err)
			}
			raw, err := os.ReadFile(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			after, _, err := f.Snapshot(onMap)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatal("SAVE changed the live snapshot", err)
			}
			checkCityMarkerResources(t, f, raw, len(selected))

			// Keep the actual city/mission route while repairing legacy paths.
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			for i := range doc.Campaign.Markers {
				m := &doc.Campaign.Markers[i]
				text := strings.TrimSuffix(strings.TrimPrefix(string(m.Text[:len(m.Text)-1]), `main\graphics\Global.Map\`), ".256")
				m.Text = append([]byte(text), 0)
			}
			legacy, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			g := releaseFront(t)
			opener, town, err := g.RestoreOriginal(legacy)
			if err != nil || town == onMap {
				t.Fatal("legacy SAVE location", town, err)
			}
			if onMap {
				app := g.App("legacy marker mission")
				if err := app.OpenMission(opener); err != nil {
					t.Fatal(err)
				}
			}
			captured, _, err := g.Snapshot(onMap)
			if err != nil {
				t.Fatal(err)
			}
			var repaired []byte
			if onMap {
				repaired, err = g.ExportCurrentWorldSave(captured, "Repaired markers")
			} else {
				repaired, _, err = g.playerCitySave(captured, "Repaired markers")
			}
			if err != nil {
				t.Fatal(err)
			}
			checkCityMarkerResources(t, g, repaired, len(selected))
		})
	}
}

func checkCityMarkerResources(t *testing.T, f *FrontEnd, raw []byte, count int) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, ok, err := file.Campaign()
	if err != nil || !ok || len(campaign.Markers) != count {
		t.Fatal("marker population", campaign.Markers, err)
	}
	for _, marker := range campaign.Markers {
		const prefix = `main\graphics\Global.Map\`
		if !strings.HasPrefix(marker.Picture, prefix) || !strings.HasSuffix(marker.Picture, ".256") {
			t.Fatalf("marker %d stores unloadable bare path %q", marker.Value, marker.Picture)
		}
		// Load the serialized path itself, without rebuilding it from REG.
		image, err := f.Archives.Containers.ReadFile(marker.Picture)
		if err != nil {
			t.Fatalf("saved path %q: %v", marker.Picture, err)
		}
		sheet, err := spr256.Decode(image)
		if err != nil || len(sheet.Frames) == 0 {
			t.Fatal("saved marker decode", err)
		}
		bare := strings.TrimSuffix(strings.TrimPrefix(marker.Picture, prefix), ".256")
		if _, err := f.Archives.Containers.ReadFile(bare); err == nil {
			t.Fatal("bare-path negative control resolved", bare)
		}
	}
	wantDocuments := []sav.CampaignDocument{{Value: 1, Kind: 1}, {Value: 2, Kind: 1}, {Value: 3, Kind: 1}, {Value: 4, Kind: 1}, {Value: 1, Kind: 0}}
	if !reflect.DeepEqual(campaign.Documents, wantDocuments) {
		t.Fatalf("SAV lost campaign documents: %+v", campaign.Documents)
	}
	party, err := file.Party()
	if err != nil {
		t.Fatal(err)
	}
	documents := 0
	for _, member := range party {
		for _, item := range member.Items {
			if item.Code == 0x0e1c {
				documents += int(item.Stack)
			}
		}
	}
	if documents != 1 {
		t.Fatalf("SAV has %d document access items, want 1", documents)
	}
}
