package game

import (
	"image"
	"os"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseWorldMapMarkerHistorySurvivesNativeSaveLoad(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: world-map marker save/load needs a lawful install")
	}
	f, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	mission, noPictureMission := markerCurrentRecords(t, f)
	screen := f.TownScreen().(*townScreen)
	screen.Choose(3)
	index := worldMarkerMissionIndex(t, screen.WorldMapView(), mission)
	noPictureIndex := worldMarkerMissionIndex(t, screen.WorldMapView(), noPictureMission)
	if screen.WorldMapView().Missions[index].Marker != nil {
		t.Fatalf("mission %d painted its marker before selection", mission)
	}
	card := ui.WorldMapCardRect(index)
	centre := image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2)
	if action := screen.WorldMapClick(centre); action.Open != nil {
		t.Fatal("selection opened the mission before route completion")
	}
	if screen.WorldMapView().Missions[index].Marker == nil {
		t.Fatalf("mission %d did not paint its installed marker after selection", mission)
	}
	noPictureCard := ui.WorldMapCardRect(noPictureIndex)
	noPictureCentre := image.Pt((noPictureCard.Min.X+noPictureCard.Max.X)/2, (noPictureCard.Min.Y+noPictureCard.Max.Y)/2)
	if action := screen.WorldMapClick(noPictureCentre); action.Open != nil {
		t.Fatal("selection of the no-picture mission opened it before route completion")
	}
	if screen.WorldMapView().Missions[noPictureIndex].Marker != nil {
		t.Fatalf("mission %d without marker art painted a marker after selection", noPictureMission)
	}
	if !screen.worldSelectedOnce[mission] {
		t.Fatalf("picture-bearing mission %d was not recorded in marker history", mission)
	}
	if _, recorded := screen.worldSelectedOnce[noPictureMission]; recorded {
		t.Fatalf("mission %d without marker art entered marker history %v", noPictureMission, screen.worldSelectedOnce)
	}

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.WorldSelectedOnce) != 1 || decoded.WorldSelectedOnce[0] != mission {
		t.Fatalf("encoded marker history = %v, want [%d]", decoded.WorldSelectedOnce, mission)
	}

	restored, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	if open, town, err := restored.Restore(decoded); err != nil || open != nil || !town {
		t.Fatalf("Restore = opener %v town %v err %v", open != nil, town, err)
	}
	back := restored.TownScreen().(*townScreen)
	back.Choose(3)
	backIndex := worldMarkerMissionIndex(t, back.WorldMapView(), mission)
	if back.WorldMapView().Missions[backIndex].Marker == nil {
		t.Fatalf("mission %d lost its installed marker after save/load", mission)
	}

	fresh, err := NewFrontEnd(root)
	if err != nil {
		t.Fatal(err)
	}
	freshMission, freshNoPicture := markerCurrentRecords(t, fresh)
	if freshMission != mission || freshNoPicture != noPictureMission {
		t.Fatal("fresh marker fixture identities changed")
	}
	freshScreen := fresh.TownScreen().(*townScreen)
	freshScreen.Choose(3)
	freshIndex := worldMarkerMissionIndex(t, freshScreen.WorldMapView(), mission)
	if freshScreen.WorldMapView().Missions[freshIndex].Marker != nil {
		t.Fatalf("fresh game inherited mission %d marker history", mission)
	}
	t.Logf("mission %d keeps its installed world-map marker across native save/load; no-picture mission %d is excluded at the writer", mission, noPictureMission)
}

func worldMarkerMissionIndex(t *testing.T, view ui.WorldMapView, mission int) int {
	t.Helper()
	for i := range view.Missions {
		if view.Missions[i].Number == mission {
			return i
		}
	}
	t.Fatalf("mission %d is absent from world-map view %+v", mission, view.Missions)
	return -1
}

func markerCurrentRecords(t *testing.T, f *FrontEnd) (int, int) {
	t.Helper()
	assets := f.worldMapAssets()
	if assets == nil || assets.data == nil {
		t.Fatal("missing installed world map")
	}
	c := f.Campaign.Value()
	first, ok := c.TownBegins()
	if !ok {
		t.Fatal("campaign has no town")
	}
	for _, main := range c.Main {
		if main < first {
			continue
		}
		records := newCampaignRecords(c, main)
		values := append([]campaignProgressRecord{records.main}, records.children...)
		mission, noPictureMission := 0, 0
		for _, r := range values {
			object := r.mapObject
			if object < 0 || object >= len(assets.data.Objects) {
				continue
			}
			row := assets.data.Objects[object]
			if row.Valid && row.Picture != "" && !strings.EqualFold(row.Picture, "nothing") {
				mission = r.mission
				break
			}
		}
		if mission == 0 {
			continue
		}
		for _, r := range values {
			object := r.mapObject
			if r.mission == mission || object == records.record(mission).mapObject || object < 0 || object >= len(assets.data.Objects) || !assets.data.Objects[object].Valid {
				continue
			}
			noPictureMission = r.mission
			if assets.data.Objects[object].Picture != "" && !strings.EqualFold(assets.data.Objects[object].Picture, "nothing") {
				assets.data.Objects[object].Picture = "nothing"
				t.Logf("synthetic missing-marker control on live installed mission %d; source installs unchanged", r.mission)
			}
			break
		}
		if noPictureMission == 0 {
			continue
		}
		f.Town.main, f.Town.records, f.Town.open = main, records, true
		f.Town.announceMission(mission)
		f.Town.announceMission(noPictureMission)
		t.Logf("marker fixture current main=%d live accepted records=%d/%d", main, mission, noPictureMission)
		return mission, noPictureMission
	}
	t.Fatal("installed campaign has no live record pair with marker art")
	return 0, 0
}
