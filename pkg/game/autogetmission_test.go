package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
)

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
	snapshotTown(town, &first, firstCampaignRules{})
	if first.Campaign.AutoGetMission != 20 || first.Campaign.Main.Announced {
		t.Fatalf("mission 10 writes AutoGetMission %#x, announced %v; want 0x14, false",
			first.Campaign.AutoGetMission, first.Campaign.Main.Announced)
	}

	if side, ok := town.Won(10); side || !ok {
		t.Fatalf("Won(10) = side %v, ok %v", side, ok)
	}
	var second Snapshot
	snapshotTown(town, &second, firstCampaignRules{})
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
	snapshotTown(town, &home, firstCampaignRules{})
	if home.Campaign.Main.Mission != 30 || home.Campaign.AutoGetMission != 0xFFFFFFFF || home.Campaign.Main.Announced {
		t.Fatalf("after the mission-20 win the campaign writes main %d, AutoGetMission %#x, announced %v; want 30, 0xffffffff, false",
			home.Campaign.Main.Mission, home.Campaign.AutoGetMission, home.Campaign.Main.Announced)
	}
}

// A LOAD keeps the file's AutoGetMission and LastMission: the original's
// reader stores both raw, and only a main record loaded above the current one
// replaces them (SAV-1330). The next SAVE writes them unchanged. A completion
// loads the next main record, which writes that record's own values, and the
// announce latch it sets follows the won record's own declaration (DIV-3800).
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
		snapshotTown(town, &s, firstCampaignRules{})
		if s.Campaign.AutoGetMission != tc.stored || s.Campaign.LastMission != tc.last {
			t.Errorf("main %d stored %#x/%d writes AutoGetMission %#x, LastMission %d", tc.main, tc.stored, tc.last, s.Campaign.AutoGetMission, s.Campaign.LastMission)
		}
		if side, ok := town.Won(tc.main); side || !ok {
			t.Fatalf("Won(%d) = side %v, ok %v", tc.main, side, ok)
		}
		var won Snapshot
		snapshotTown(town, &won, firstCampaignRules{})
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
	snapshotTown(f.Town, &s, f.campaign())
	projection, err := nativeCampaignProjectionForChapter(f, s, f.Town.Chapter())
	if err != nil {
		t.Fatal(err)
	}
	if projection.Main.Mission != 20 || projection.AutoGetMission != 0xFFFFFFFF || !projection.Main.Announced {
		t.Fatalf("fresh mission 20 writes main %d, AutoGetMission %#x, announced %v; want 20, 0xffffffff, true",
			projection.Main.Mission, projection.AutoGetMission, projection.Main.Announced)
	}
}

// A loaded AutoGetMission other than -1 and not above the current main
// mission takes the travel arm at a mission end (SAV-1329): the main record is
// announced and opens again, nothing is completed or paid, and the town is not
// reached. -1 completes into the town and 20 at mission 10 loads mission 20,
// as before.
func TestALoadedAutoGetMissionAtOrBelowTheMainTravelsToTheMain(t *testing.T) {
	c := townCampaign(t)
	for _, tc := range []struct {
		name           string
		main, won      int
		stored         uint32
		next           int
		done, arrived  bool
		gold, wantMain int
	}{
		{"0 at main 10", 10, 10, 0, 10, false, false, 0, 10},
		{"20 at main 20", 20, 20, 20, 20, false, false, 0, 20},
		{"0 at main 30", 30, 30, 0, 30, false, false, 0, 30},
		{"0 at main 30, side 31 won", 30, 31, 0, 30, false, false, 0, 30},
		{"-2 at main 30", 30, 30, 0xFFFFFFFE, 30, false, false, 0, 30},
		{"20 at main 10", 10, 10, 20, 20, true, false, 0, 20},
		{"-1 at main 30", 30, 30, 0xFFFFFFFF, 0, true, true, 1000, 40},
	} {
		projection := campaignProjectionAt(c, tc.main)
		projection.AutoGetMission = tc.stored
		if tc.won != tc.main {
			ch := c.Chapters[tc.won]
			projection.Children = []sav.CampaignRecord{{Mission: uint32(tc.won), MapObject: uint32(c.MapObjects[tc.won]),
				Payment: uint32(ch.Payment), Announced: true}}
			projection.SelectedMission = uint32(tc.won)
		}
		progress, err := campaignProgressFromSAV(c, projection)
		if err != nil {
			t.Fatal(tc.name, err)
		}
		s := CampaignSession{Town: newTownFromCampaignProgress(c, progress), Offered: 5}
		s.Town.gold = 0
		arrived := false
		next, msg := s.recordWin(c, tc.won, nil, nil, nil, nil,
			townArrival{arrive: func() { arrived = true }, welcome: func(int) {}})
		var snap Snapshot
		snapshotTown(s.Town, &snap, firstCampaignRules{})
		travel := !tc.done
		if next != tc.next || s.Town.Done(tc.won) != tc.done || arrived != tc.arrived || s.Town.Gold() != tc.gold ||
			int(snap.Campaign.Main.Mission) != tc.wantMain || s.Offered != 0 && travel {
			t.Errorf("%s: next %d done %v arrived %v gold %d main %d offered %d (%s); want next %d done %v arrived %v gold %d main %d",
				tc.name, next, s.Town.Done(tc.won), arrived, s.Town.Gold(), snap.Campaign.Main.Mission, s.Offered, msg,
				tc.next, tc.done, tc.arrived, tc.gold, tc.wantMain)
			continue
		}
		if !travel {
			continue
		}
		if !snap.Campaign.Main.Announced || snap.Campaign.SelectedMission != uint32(tc.main) || snap.Campaign.AutoGetMission != tc.stored {
			t.Errorf("%s: travel writes main %+v selected %d AutoGetMission %#x; want main %d announced, selected, %#x kept",
				tc.name, snap.Campaign.Main, snap.Campaign.SelectedMission, snap.Campaign.AutoGetMission, tc.main, tc.stored)
		}
		if tc.won != tc.main && len(snap.Campaign.Children) != 1 {
			t.Errorf("%s: the won side record was removed: children %+v", tc.name, snap.Campaign.Children)
		}
		if want := fmt.Sprintf("mission %d won - the party goes back to mission %d's map point; mission %d opens again", tc.won, tc.main, tc.main); msg != want {
			t.Errorf("%s: message %q, want %q", tc.name, msg, want)
		}
	}
	// The campaign's ending comes before the arm: a won terminal mission
	// completes although its loaded word is 0 (SAV-1336).
	terminal := townCampaign(t)
	terminal.Last = map[int]bool{40: true}
	projection := campaignProjectionAt(terminal, 40)
	projection.AutoGetMission = 0
	progress, err := campaignProgressFromSAV(terminal, projection)
	if err != nil {
		t.Fatal(err)
	}
	s := CampaignSession{Town: newTownFromCampaignProgress(terminal, progress)}
	next, msg := s.recordWin(terminal, 40, nil, nil, nil, nil, townArrival{arrive: func() {}, welcome: func(int) {}})
	if !s.Town.Done(40) || next == 40 {
		t.Errorf("terminal 0 at main 40: next %d done %v (%s); want the mission completed and no travel", next, s.Town.Done(40), msg)
	}
}
