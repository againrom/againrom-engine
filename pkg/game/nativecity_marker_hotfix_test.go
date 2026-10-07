package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCityCampaignMarkerPathsRepairsOnlyKnownBareNames(t *testing.T) {
	f := &FrontEnd{Presentation: Presentation{worldMapCache: resolved(&worldMapAssets{data: &globalMapData{Missions: map[int]int{100: 0}, Objects: []globalMapObject{{Valid: true, Picture: "onmap13"}}}}, nil)}}
	input := sav.CampaignProjection{Markers: []sav.CampaignMarker{
		{Value: 100, Picture: "onmap13", Field0: 17, Field1: 23},
		{Value: 100, Picture: `main\graphics\Global.Map\onmap13.256`, Field0: 29, Field1: 31},
		{Value: 100, Picture: "custom.256"},
		{Value: 999, Picture: "onmap13"},
	}}
	original := append([]sav.CampaignMarker(nil), input.Markers...)
	got := f.cityCampaignMarkerPaths(input)
	want := append([]sav.CampaignMarker(nil), original...)
	want[0].Picture = `main\graphics\Global.Map\onmap13.256`
	if !reflect.DeepEqual(got.Markers, want) {
		t.Fatalf("marker repair = %+v, want %+v", got.Markers, want)
	}
	if !reflect.DeepEqual(input.Markers, original) {
		t.Fatal("SAV export changed the captured campaign")
	}
	if again := f.cityCampaignMarkerPaths(got); !reflect.DeepEqual(again, got) {
		t.Fatal("qualified marker changed on second export")
	}
}
