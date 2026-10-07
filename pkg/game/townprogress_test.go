package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func preTownCampaign(t *testing.T, restored bool) (Campaign, *Town) {
	t.Helper()
	c := townCampaign(t)
	if !restored {
		return c, NewTown(c)
	}
	p, err := campaignProgressFromSAV(c, campaignProjectionAt(c, 10))
	if err != nil {
		t.Fatal(err)
	}
	return c, newTownFromCampaignProgress(c, p)
}

func requireClosedTown(t *testing.T, town *Town, main int) {
	t.Helper()
	if town.Open() || town.Chapter() != main || town.ChapterData().Mission != main {
		t.Fatalf("closed town open/main/data = %v/%d/%d, want false/%d/%d",
			town.Open(), town.Chapter(), town.ChapterData().Mission, main, main)
	}
	var before Snapshot
	snapshotTown(town, &before)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		if offers := town.Offers(building); len(offers) != 0 {
			t.Fatalf("closed building %d offers %+v", building, offers)
		}
		if mission, ok := town.Take(building, 0); ok || mission != 0 {
			t.Fatalf("closed building %d accepted %d/%v", building, mission, ok)
		}
	}
	var after Snapshot
	snapshotTown(town, &after)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("closed offer queries changed campaign state")
	}
}

func TestTownCampaignMainFollowsThePreTownMissionBoundary(t *testing.T) {
	for _, restored := range []bool{false, true} {
		name := "native"
		if restored {
			name = "ordinary"
		}
		t.Run(name, func(t *testing.T) {
			c, town := preTownCampaign(t, restored)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: town}}
			requireClosedTown(t, town, 10)
			if next, _ := f.FinishMissionWithRoster(10, nil, nil, nil, nil); next != 20 {
				t.Fatalf("mission 10 successor = %d, want 20", next)
			}
			requireClosedTown(t, town, 20)
			if next, _ := f.FinishMissionWithRoster(20, nil, nil, nil, nil); next != 0 || f.Offered != 30 {
				t.Fatalf("mission 20 successor/offered = %d/%d, want town/30", next, f.Offered)
			}
			if !town.Open() || town.Chapter() != 30 {
				t.Fatalf("first arrival open/main = %v/%d, want true/30", town.Open(), town.Chapter())
			}
			if got, want := town.Offers(TownTavern), []TownOffer{{Index: 0, Mission: 30, NPC: 22}, {Index: 1, NPC: 90}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("arrival tavern = %+v, want %+v", got, want)
			}
			if got, want := town.Offers(TownShop), []TownOffer{{Index: 0, Mission: 31}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("arrival shop = %+v, want %+v", got, want)
			}
			if len(town.Offers(TownSchool)) != 0 {
				t.Fatal("arrival exposed a later chapter's school")
			}
			f.liveDriver(nil, 31, nil)
			if town.Chapter() != 30 || !town.Open() {
				t.Fatalf("side entry replaced open main: %d/%v", town.Chapter(), town.Open())
			}
			town.Won(31)
			if town.Chapter() != 30 {
				t.Fatal("side completion advanced the main")
			}
		})
	}
}

func TestTownOffersWaitForArriveEvenWithCurrentBuildingRows(t *testing.T) {
	for _, restored := range []bool{false, true} {
		_, town := preTownCampaign(t, restored)
		town.Won(10)
		town.Won(20)
		if len(town.ChapterData().Shop) != 1 || len(town.ChapterData().Inn) != 2 {
			t.Fatal("control has no current building rows")
		}
		requireClosedTown(t, town, 30)
		town.Arrive()
		if len(town.Offers(TownShop)) != 1 || len(town.Offers(TownTavern)) != 2 {
			t.Fatal("closed Take consumed a row before arrival")
		}
	}
}

func TestTownArrivalUsesADeclaredMainWhenAnEarlierSideIsOffered(t *testing.T) {
	c := townCampaign(t)
	c.Offered = []int{21, 30, 31, 40, 41}
	town := NewTown(c)
	town.Arrive()
	if town.Chapter() != 30 {
		t.Fatalf("first arrival chose %d, want declared main 30", town.Chapter())
	}
	legacy := restoreTown(c, Snapshot{Open: true, Mission: 31})
	if legacy.Chapter() != 30 || !legacy.Open() {
		t.Fatalf("legacy side entry lost its reached main: %d/%v", legacy.Chapter(), legacy.Open())
	}
}

func TestTownMissionIdentityCommitsOnlyWithThePreparedWorld(t *testing.T) {
	f := advanceFront(t)
	f.Town = NewTown(f.Campaign.Value())
	var activate func()
	opener := f.missionOpenerMode(20, nil, nil, nil, nil, &activate, nil, f.Difficulty, f.Town)
	if _, _, _, _, _, _, _, _, _, _, err := opener(); err != nil {
		t.Fatal(err)
	}
	if activate == nil || f.live != nil || f.Town.Chapter() != 10 {
		t.Fatal("preparing a mission adopted its main before activation")
	}
	activate()
	if f.live == nil || f.liveMission != 20 || f.Town.Chapter() != 20 || f.Town.Open() {
		t.Fatalf("activated mission/main/open = %d/%d/%v", f.liveMission, f.Town.Chapter(), f.Town.Open())
	}
	var before Snapshot
	snapshotTown(f.Town, &before)
	world := f.live
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(30)(); err == nil {
		t.Fatal("missing map unexpectedly opened")
	}
	var after Snapshot
	snapshotTown(f.Town, &after)
	if f.live != world || f.liveMission != 20 || !reflect.DeepEqual(before, after) {
		t.Fatal("failed mission activation changed the live town or world")
	}
}

func TestTownLegacyMainAndOrdinaryPreTownProjectionAgree(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}}
	for _, mission := range []int{10, 20} {
		legacy := Snapshot{Mission: mission}
		town := restoreTown(c, legacy)
		requireClosedTown(t, town, mission)
		var current Snapshot
		snapshotTown(town, &current)
		current.Mission = mission
		if current.MainMission != mission || f.generatedCampaignChapter(current) != mission {
			t.Fatalf("mission %d capture lost current main", mission)
		}
		projection, err := nativeCampaignProjection(f, current)
		if err != nil {
			t.Fatal(err)
		}
		if projection.Main.Mission != uint32(mission) || len(projection.ShopMission) != 0 || len(projection.InnMission) != 0 || len(projection.TCMission) != 0 {
			t.Fatalf("pre-town ordinary record = %+v", projection)
		}
		progress, err := campaignProgressFromSAV(c, projection)
		if err != nil {
			t.Fatal(err)
		}
		requireClosedTown(t, newTownFromCampaignProgress(c, progress), mission)
	}
}

func TestTownConsumedOffersSurviveArrivalAndColdCampaignRecords(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	f.Town.Arrive()
	for _, building := range []TownBuilding{TownShop, TownTavern} {
		if _, ok := f.Town.Take(building, 0); !ok {
			t.Fatalf("building %d supplied no offer", building)
		}
	}
	assertConsumed := func(town *Town) {
		t.Helper()
		town.Arrive()
		if !town.Open() || town.Chapter() != 30 || !reflect.DeepEqual(town.Available(), []int{30, 31}) || len(town.Offers(TownShop)) != 0 {
			t.Fatalf("consumed history main/open/available/shop = %d/%v/%v/%v", town.Chapter(), town.Open(), town.Available(), town.Offers(TownShop))
		}
		if offers := town.Offers(TownTavern); len(offers) != 1 || offers[0].NPC != 90 || offers[0].Mission != 0 {
			t.Fatalf("consumed inn row reappeared: %+v", offers)
		}
	}
	assertConsumed(f.Town)
	for cycle := 0; cycle < 2; cycle++ {
		snapshot := Snapshot{Mission: 31}
		snapshotTown(f.Town, &snapshot)
		assertConsumed(restoreTown(c, snapshot))
		projection := snapshot.Campaign
		if !snapshot.CampaignState {
			var err error
			projection, err = nativeCampaignProjection(f, snapshot)
			if err != nil {
				t.Fatal(err)
			}
		}
		projection.SelectedMission = 31
		file, err := sav.Open(originalSaveWithCampaign(t, 31, projection))
		if err != nil {
			t.Fatal(err)
		}
		decoded, present, err := file.Campaign()
		if err != nil || !present {
			t.Fatalf("cold campaign record: present=%v error=%v", present, err)
		}
		if decoded.Main.Mission != 30 || len(decoded.ShopMission) != 0 || !reflect.DeepEqual(decoded.InnMission, []uint16{0}) || !reflect.DeepEqual(decoded.InnNPC, []uint16{90}) {
			t.Fatalf("ordinary consumed rows changed: %+v", decoded)
		}
		f.Town = restoreTown(c, Snapshot{Mission: 31, MainMission: 10, CampaignState: true, Campaign: decoded})
		assertConsumed(f.Town)
	}
}
