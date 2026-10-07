package game

import (
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestCampaignMapObjectProjectionRetainsOtherState(t *testing.T) {
	f := &FrontEnd{Presentation: Presentation{worldMapCache: resolved(&worldMapAssets{data: &globalMapData{
		Missions: map[int]int{30: 9, 31: 33, 41: 22, 777: -1, 778: 99}, Objects: make([]globalMapObject, 34),
	}}, nil)}}
	input := sav.CampaignProjection{
		Main: sav.CampaignRecord{Mission: 30, MapObject: 123, Payment: 57, Announced: true},
		Children: []sav.CampaignRecord{
			{Mission: 31, MapObject: 124, Age: 1},
			{Mission: 41, MapObject: 125, Announced: true},
			{Mission: 999, MapObject: 61},
			{Mission: 777, MapObject: 62},
			{Mission: 778, MapObject: 63},
		},
		SelectedMission: 41, FirstMapPoint: false, InnNPC: []uint16{22}, InnMission: []uint16{30},
	}
	before := input
	before.Children = append([]sav.CampaignRecord(nil), input.Children...)
	want := before
	want.Children = append([]sav.CampaignRecord(nil), before.Children...)
	want.Main.MapObject, want.Children[0].MapObject, want.Children[1].MapObject = 9, 33, 22
	got := f.campaignMapObjects(input)
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(input, before) {
		t.Fatal("projection differs or mutated retained input", got, want)
	}
	if again := f.campaignMapObjects(got); !reflect.DeepEqual(again, got) {
		t.Fatal("repeated projection changed campaign state")
	}
	for _, missing := range []*FrontEnd{nil, {}} {
		if got := missing.campaignMapObjects(input); !reflect.DeepEqual(got, input) {
			t.Fatal("missing map metadata changed retained records")
		}
	}
}

func TestCurrentCampaignUsesCapturedTown(t *testing.T) {
	campaign := saveCampaign()
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}}
	f.Town = NewTown(campaign)
	f.Town.open = true
	f.Town.won[10] = true
	f.Town.activateMission(20)
	f.Town.announceMission(20)
	f.Town.documents = []Document{{Value: 91, Kind: 2}}
	var captured Snapshot
	snapshotTown(f.Town, &captured)
	want, err := nativeCampaignProjectionForChapter(f, captured, 20)
	if err != nil || len(want.Documents) != 1 || want.Documents[0].Value != 91 {
		t.Fatal("captured campaign", want, err)
	}
	f.Town = nil
	again, err := nativeCampaignProjectionForChapter(f, captured, 20)
	if err != nil || !reflect.DeepEqual(want, again) {
		t.Fatal("SAVE reread the live Town", again, err)
	}
}

func TestCurrentCampaignAddsSelectedMarkersWithoutReplacingOrdinaryRecords(t *testing.T) {
	f := &FrontEnd{Presentation: Presentation{worldMapCache: resolved(&worldMapAssets{data: &globalMapData{
		Missions: map[int]int{30: 0, 31: 1}, Objects: []globalMapObject{{Valid: true, Picture: "old"}, {Valid: true, Picture: "new"}},
	}}, nil)}}
	saved := sav.CampaignMarker{Value: 30, Picture: "retained-path", Field0: 17, Field1: 23}
	source := sav.CampaignProjection{Markers: []sav.CampaignMarker{saved}, SelectedMission: 31}
	got := f.currentCampaignMarkers(source, []int{30, 31, 31, 999})
	if len(got.Markers) != 3 || got.Markers[0] != saved || got.Markers[1].Value != 31 || got.Markers[1].Picture != originalCityMarkerPath("new") || got.Markers[2].Value != 999 {
		t.Fatal("current marker history changed", got.Markers)
	}
	if len(source.Markers) != 1 || source.Markers[0] != saved {
		t.Fatal("capture was mutated")
	}
	got = f.currentCampaignMarkers(got, []int{31})
	if len(got.Markers) != 3 {
		t.Fatal("repeated capture duplicated a marker")
	}
}

func TestReleaseCampaignSAVMapObjectsMatchInstalledMapping(t *testing.T) {
	f := releaseFront(t)
	party := f.ChargenParty(ui.ChargenResult{Name: "Map witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	app := f.App("map witness")
	if err := app.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(generatedMissionSave(t, f, app))
	if err != nil {
		t.Fatal(err)
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	campaign, present, err := file.Campaign()
	if err != nil || !present || campaign.Main.Mission != 20 || campaign.Main.MapObject != 19 {
		t.Fatal("ordinary mission20 SAVE map object", campaign.Main, err)
	}
	state, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	data := f.worldMapAssets().data
	if data == nil || len(data.Missions) != 28 {
		t.Fatal("installed map population is missing")
	}
	for mission, object := range data.Missions {
		p, err := nativeCampaignProjectionForChapter(f, state, mission)
		if err != nil || p.Main.MapObject != uint32(object) {
			t.Errorf("mission%d object=%d want=%d error=%v", mission, p.Main.MapObject, object, err)
		}
		for _, child := range p.Children {
			if expected, ok := data.Missions[int(child.Mission)]; ok && child.MapObject != uint32(expected) {
				t.Errorf("mission%d child%d object=%d want=%d", mission, child.Mission, child.MapObject, expected)
			}
		}
	}
}
