package game

import "testing"

// Every original mission-10 SAV holds AutoGetMission 20 and every original
// mission-20 SAV holds -1 with its main record announced (REG-SCN-063). A
// restored campaign that wins mission 10 writes the second mission's own
// value: mission 10's 20 sends the original's party back to mission 20's map
// object at its end, and the town is never reached.
func TestAWonMissionTenWritesTheSecondMissionsOwnRouting(t *testing.T) {
	c := townCampaign(t)
	projection := campaignProjectionAt(c, 10)
	projection.AutoGetMission = 20
	progress, err := campaignProgressFromSAV(c, projection)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(c, progress)
	var first Snapshot
	snapshotTown(town, &first)
	if first.Campaign.AutoGetMission != 20 || first.Campaign.Main.Announced {
		t.Fatalf("mission 10 writes AutoGetMission %#x, announced %v; want 0x14, false",
			first.Campaign.AutoGetMission, first.Campaign.Main.Announced)
	}

	if side, ok := town.Won(10); side || !ok {
		t.Fatalf("Won(10) = side %v, ok %v", side, ok)
	}
	var second Snapshot
	snapshotTown(town, &second)
	if second.Campaign.Main.Mission != 20 || second.Campaign.AutoGetMission != 0xFFFFFFFF || !second.Campaign.Main.Announced {
		t.Fatalf("after the mission-10 win the campaign writes main %d, AutoGetMission %#x, announced %v; want 20, 0xffffffff, true",
			second.Campaign.Main.Mission, second.Campaign.AutoGetMission, second.Campaign.Main.Announced)
	}

	// Mission 20's own -1 sends the party home: the next record loads
	// unannounced, as the original's city save after mission 20 holds it.
	if side, ok := town.Won(20); side || !ok {
		t.Fatalf("Won(20) = side %v, ok %v", side, ok)
	}
	var home Snapshot
	snapshotTown(town, &home)
	if home.Campaign.Main.Mission != 30 || home.Campaign.AutoGetMission != 0xFFFFFFFF || home.Campaign.Main.Announced {
		t.Fatalf("after the mission-20 win the campaign writes main %d, AutoGetMission %#x, announced %v; want 30, 0xffffffff, false",
			home.Campaign.Main.Mission, home.Campaign.AutoGetMission, home.Campaign.Main.Announced)
	}
}

// A LOAD keeps the file's AutoGetMission and LastMission: the original's
// reader stores both raw, and only a main record loaded above the current one
// replaces them (SAV-1330). The next SAVE writes them unchanged. A mission end
// loads the next main record, which writes that record's own values, and the
// announce latch it sets follows the won record's own declaration, so a
// carried value does not change the route (DIV-1497).
func TestALoadedSAVKeepsItsAutoGetMissionUntilAMissionEnd(t *testing.T) {
	c := townCampaign(t)
	for _, tc := range []struct {
		main   int
		stored uint32
		last   uint32
	}{
		{10, 20, 0},
		{10, 0, 0},
		{10, 0xFFFFFFFF, 7},
		{20, 0xFFFFFFFF, 0},
		{20, 20, 0},
		{20, 0, 1},
		{30, 20, 0},
		{30, 0, 0},
	} {
		projection := campaignProjectionAt(c, tc.main)
		projection.AutoGetMission, projection.LastMission = tc.stored, tc.last
		progress, err := campaignProgressFromSAV(c, projection)
		if err != nil {
			t.Fatal(err)
		}
		town := newTownFromCampaignProgress(c, progress)
		var s Snapshot
		snapshotTown(town, &s)
		if s.Campaign.AutoGetMission != tc.stored || s.Campaign.LastMission != tc.last {
			t.Errorf("main %d stored %#x/%d writes AutoGetMission %#x, LastMission %d", tc.main, tc.stored, tc.last, s.Campaign.AutoGetMission, s.Campaign.LastMission)
		}
		if side, ok := town.Won(tc.main); side || !ok {
			t.Fatalf("Won(%d) = side %v, ok %v", tc.main, side, ok)
		}
		var won Snapshot
		snapshotTown(town, &won)
		next := tc.main + mainMissionStride
		if won.Campaign.Main.Mission != uint32(next) || won.Campaign.AutoGetMission != autoGetMissionValue(c, next) ||
			won.Campaign.LastMission != lastMissionValue(c, next) || won.Campaign.Main.Announced != (tc.main == 10) {
			t.Errorf("main %d stored %#x: after the win main %d, AutoGetMission %#x, LastMission %d, announced %v",
				tc.main, tc.stored, won.Campaign.Main.Mission, won.Campaign.AutoGetMission, won.Campaign.LastMission, won.Campaign.Main.Announced)
		}
	}
}

// A fresh campaign that wins mission 10 writes mission 20's record announced,
// as the restored route does.
func TestAFreshMissionTenWinAnnouncesTheSecondMission(t *testing.T) {
	c := townCampaign(t)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}}
	if side, ok := f.Town.Won(10); side || !ok {
		t.Fatalf("Won(10) = side %v, ok %v", side, ok)
	}
	var s Snapshot
	snapshotTown(f.Town, &s)
	projection, err := nativeCampaignProjectionForChapter(f, s, f.Town.Chapter())
	if err != nil {
		t.Fatal(err)
	}
	if projection.Main.Mission != 20 || projection.AutoGetMission != 0xFFFFFFFF || !projection.Main.Announced {
		t.Fatalf("fresh mission 20 writes main %d, AutoGetMission %#x, announced %v; want 20, 0xffffffff, true",
			projection.Main.Mission, projection.AutoGetMission, projection.Main.Announced)
	}
}
