package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestWorldMarkerHistoryRoundTripsAndReplacesThePreviousGame(t *testing.T) {
	campaign := saveCampaign()
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: saveTown(t)}}
	screen := f.TownScreen().(*townScreen)
	screen.worldSelectedOnce = map[int]bool{20: true, 10: true, 99: false}

	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{10, 20}; !reflect.DeepEqual(snapshot.WorldSelectedOnce, want) {
		t.Fatalf("snapshot marker history = %v, want sorted %v", snapshot.WorldSelectedOnce, want)
	}
	encoded, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(encoded)
	if err != nil {
		t.Fatal(err)
	}

	g := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: NewTown(campaign)}}
	g.TownScreen()
	g.townUI.worldSelectedOnce = map[int]bool{999: true}
	open, town, err := g.Restore(decoded)
	if err != nil || open != nil || !town {
		t.Fatalf("Restore = opener %v town %v err %v", open != nil, town, err)
	}
	if want := map[int]bool{10: true, 20: true}; !reflect.DeepEqual(g.townUI.worldSelectedOnce, want) {
		t.Fatalf("restored marker history = %v, want %v", g.townUI.worldSelectedOnce, want)
	}
	decoded.WorldSelectedOnce[0] = 77
	if g.townUI.worldSelectedOnce[77] || !g.townUI.worldSelectedOnce[10] {
		t.Fatal("installed marker history aliases the caller's snapshot slice")
	}

	g.townUI.resetForNewGame()
	if len(g.townUI.worldSelectedOnce) != 0 {
		t.Fatalf("new-game reset retained marker history %v", g.townUI.worldSelectedOnce)
	}
}

func TestMissionSnapshotCapturesWorldMarkerHistory(t *testing.T) {
	campaign := saveCampaign()
	world, err := sim.NewWorld(7, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, make([]byte, 4), nil)
	if err != nil {
		t.Fatal(err)
	}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: saveTown(t), live: &mapWorld{world: world}, liveMission: 10}}
	f.TownScreen().(*townScreen).worldSelectedOnce = map[int]bool{20: true}
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Mission != 10 || !reflect.DeepEqual(snapshot.WorldSelectedOnce, []int{20}) {
		t.Fatalf("mission snapshot = mission %d marker history %v", snapshot.Mission, snapshot.WorldSelectedOnce)
	}
}

func TestMissionCandidateCommitRestoresWorldMarkerHistory(t *testing.T) {
	campaign := saveCampaign()
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: NewTown(campaign)}}
	f.TownScreen()
	f.townUI.worldSelectedOnce = map[int]bool{999: true}
	activated := false
	f.installCandidate(&restoreCandidate{
		town:              NewTown(campaign),
		carried:           []mapload.PartyMember{{Body: "loaded"}},
		worldSelectedOnce: map[int]bool{20: true},
		activate:          func() { activated = true },
	})
	if !activated {
		t.Fatal("mission candidate did not activate")
	}
	if want := map[int]bool{20: true}; !reflect.DeepEqual(f.townUI.worldSelectedOnce, want) {
		t.Fatalf("mission marker history = %v, want %v", f.townUI.worldSelectedOnce, want)
	}
}

func TestMalformedWorldMarkerHistoryRefusesBeforeCommit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		missions []int
	}{
		{"zero", []int{0}},
		{"negative", []int{-1}},
		{"duplicate", []int{10, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			campaign := saveCampaign()
			oldTown := saveTown(t)
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(campaign, nil)}, CampaignSession: CampaignSession{Town: oldTown}}
			f.TownScreen()
			f.townUI.worldSelectedOnce = map[int]bool{99: true}
			if _, _, err := f.Restore(Snapshot{Open: true, WorldSelectedOnce: tc.missions}); err == nil {
				t.Fatal("malformed marker history was accepted")
			}
			if f.Town != oldTown || !f.townUI.worldSelectedOnce[99] {
				t.Fatal("refused marker history changed the active game")
			}
		})
	}
}
