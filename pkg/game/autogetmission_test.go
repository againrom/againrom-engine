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

// A LOAD takes AutoGetMission from the loaded main record's own section, not
// from the file: an engine SAV written before this rule holds mission 10's 20
// at mission 20, and saves descended from older engine documents hold 0
// (DIV-1497).
func TestALoadedSAVWritesTheMainRecordsOwnAutoGetMission(t *testing.T) {
	c := townCampaign(t)
	for _, tc := range []struct {
		main         int
		stored, want uint32
	}{
		{10, 20, 20},
		{10, 0, 20},
		{10, 0xFFFFFFFF, 20},
		{20, 0xFFFFFFFF, 0xFFFFFFFF},
		{20, 20, 0xFFFFFFFF},
		{20, 0, 0xFFFFFFFF},
		{30, 20, 0xFFFFFFFF},
		{30, 0, 0xFFFFFFFF},
	} {
		projection := campaignProjectionAt(c, tc.main)
		projection.AutoGetMission = tc.stored
		progress, err := campaignProgressFromSAV(c, projection)
		if err != nil {
			t.Fatal(err)
		}
		var s Snapshot
		snapshotTown(newTownFromCampaignProgress(c, progress), &s)
		if s.Campaign.AutoGetMission != tc.want {
			t.Errorf("main %d stored %#x writes AutoGetMission %#x, want %#x", tc.main, tc.stored, s.Campaign.AutoGetMission, tc.want)
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
