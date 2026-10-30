package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
)

func campaignRecordFront(t *testing.T) (*FrontEnd, Snapshot) {
	t.Helper()
	// Side mission 13 is offered by no building: the chapter holds no child
	// record of it.
	c := saveCampaign()
	c.Side = append(c.Side, 13)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	var s Snapshot
	snapshotTown(f.Town, &s, f.campaign())
	s.Mission = 13
	return f, s
}

// The mission route over the chapter source writes the chapter's native
// projection, then the played side mission's child record, its selection,
// its successor and the mission's map object, then the tail word. A resumed
// mission's record takes the selection and the map object only; the SAVE's
// tail comes when it is written.
func TestTheChapterSourceTakesItsRouteOverlays(t *testing.T) {
	f, s := campaignRecordFront(t)
	c := f.Campaign.Value()
	native, err := nativeCampaignProjectionForChapter(f, s, f.Town.Chapter())
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range native.Children {
		if child.Mission == 13 {
			t.Fatal("the fixture's chapter already holds the played child record")
		}
	}
	want := native
	child := progressRecordFromCampaign(c, 13).savRecord(true)
	child.Age = 0
	want.Children = append(append([]sav.CampaignRecord(nil), native.Children...), child)
	want.SelectedMission, want.AutoGetMission, want.FirstMapPoint = 13, autoGetMissionValue(c, 13), false
	got, err := f.campaignRecord(s, missionRecord)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("mission record\n%+v\nwant\n%+v (%v)", got, want, err)
	}

	resumed := native
	resumed.SelectedMission, resumed.FirstMapPoint = 13, false
	if got, err := f.campaignRecord(s, resumedMissionRecord); err != nil || !reflect.DeepEqual(got, resumed) {
		t.Fatalf("resumed record\n%+v\nwant\n%+v (%v)", got, resumed, err)
	}

	// A carried record takes the tail word only.
	carried := s
	carried.CampaignState, carried.Campaign = true, sav.CampaignProjection{Main: sav.CampaignRecord{Mission: 10}, SelectedMission: 12}
	if got, err := f.campaignRecord(carried, missionRecord); err != nil || !reflect.DeepEqual(got, carried.Campaign) {
		t.Fatalf("carried mission record %+v, want %+v (%v)", got, carried.Campaign, err)
	}

	// The town route over the chapter source is the chapter's projection
	// with its map objects; the fixture has no world map.
	town := s
	town.Mission = 0
	wantTown, err := nativeCampaignProjection(f, town)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := f.campaignRecord(town, townRecord); err != nil || !reflect.DeepEqual(got, wantTown) {
		t.Fatalf("town record\n%+v\nwant\n%+v (%v)", got, wantTown, err)
	}
}

// The second game's bank states the record of a session that holds the
// destinations campaign: the mission record keeps its own home word, the
// town record sets it, and neither takes the tail.
func TestTheSecondGamesStatedRecordKeepsItsHomeWord(t *testing.T) {
	profile, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Base: base.Match{Profile: profile}}, Campaign: resolved(Campaign{}, nil)},
		CampaignSession: CampaignSession{Town: NewTown(Campaign{})}}
	s := Snapshot{Mission: 10, second: &currentSecondCampaign{}}
	stated := sav.CampaignProjection{Main: sav.CampaignRecord{Mission: 10}, SelectedMission: 10, AutoGetMission: ^uint32(0)}
	if got, err := f.campaignRecord(s, missionRecord); err != nil || !reflect.DeepEqual(got, stated) {
		t.Fatalf("stated mission record %+v, want %+v (%v)", got, stated, err)
	}
	stated.FirstMapPoint = true
	if got, err := f.campaignRecord(s, townRecord); err != nil || !reflect.DeepEqual(got, stated) {
		t.Fatalf("stated town record %+v, want %+v (%v)", got, stated, err)
	}
	// A carried record of a session without the destinations campaign is
	// the chapter shape's.
	s.second, s.CampaignState, s.Campaign = nil, true, sav.CampaignProjection{SelectedMission: 10, FirstMapPoint: true}
	if got, err := f.campaignRecord(s, missionRecord); err != nil || !reflect.DeepEqual(got, s.Campaign) {
		t.Fatalf("carried record %+v, want %+v (%v)", got, s.Campaign, err)
	}
}

// A town with no main list writes the labels LOAD restored under either
// profile; a town with a main list writes its current chapter offers under
// the chapter profile and the loaded labels under the second game's.
func TestTheOfferLabelSourceIsTheProfiles(t *testing.T) {
	loaded := &currentOfferLabels{Chapter: 10, Buildings: [3][]TownOffer{TownShop: {{Mission: 11, Index: 1}}}}
	for _, tc := range []struct {
		name     string
		camp     Campaign
		rules    campaignService
		asLoaded bool
	}{
		{"no main list, chapter profile", Campaign{}, firstCampaignRules{}, true},
		{"no main list, second profile", Campaign{}, secondCampaignRules{}, true},
		{"main list, chapter profile", saveCampaign(), firstCampaignRules{}, false},
		{"main list, second profile", saveCampaign(), secondCampaignRules{}, true},
	} {
		town := NewTown(tc.camp)
		town.offerLabels = cloneOfferLabels(loaded)
		want := loaded
		if !tc.asLoaded {
			want = snapshotOfferLabels(town)
			if reflect.DeepEqual(want, loaded) {
				t.Fatalf("%s: the town's own offers are the loaded labels", tc.name)
			}
		}
		var s Snapshot
		snapshotTown(town, &s, tc.rules)
		if !reflect.DeepEqual(s.OfferLabels, want) {
			t.Errorf("%s: labels %+v, want %+v", tc.name, s.OfferLabels, want)
		}
	}
}
