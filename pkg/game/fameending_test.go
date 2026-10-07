package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/fame"
	"againrom/pkg/mapload"
)

func completedFameFront(t *testing.T) *FrontEnd {
	t.Helper()
	c := Campaign{Main: []int{10, 150}, Last: map[int]bool{150: true}}
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}, CampaignSession: CampaignSession{Town: NewTown(c)}, PersistenceContext: PersistenceContext{hallStore: fameStore{Dir: t.TempDir()}}}
	f.Town.Won(10)
	f.Town.Won(150)
	f.Town.open = true // This unit fixture controls the already-completed town.
	return f
}

func TestEarnedEndingWritesOnceAndRestoresRecordedResult(t *testing.T) {
	f := completedFameFront(t)
	f.fame = SnapshotFame{Known: true, Time: 99, Events: 7, Result: &FameResult{Name: "Finisher", Score: 4321}}
	for i := 0; i < 3; i++ {
		view, complete := f.campaignEnding()
		if !complete || !view.HasScore || !view.Recorded || view.Score != 4321 || len(view.Hall) != 1 {
			t.Fatal("ending lost or duplicated result", view, complete)
		}
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(s, "complete")
	if err != nil {
		t.Fatal(err)
	}
	s, _, err = DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	cold := completedFameFront(t)
	cold.hallStore = f.hallStore
	cold.fame = fameFromSnapshot(s)
	view, _ := cold.campaignEnding()
	if len(view.Hall) != 1 || !view.Recorded {
		t.Fatal("cold view duplicated result", view)
	}
}

func TestEarnedEndingFailedPublicationRetriesWithoutLosingOldHall(t *testing.T) {
	f := completedFameFront(t)
	seed, _ := fame.Marshal([]fame.Record{{Name: "old", Score: 100}})
	path := filepath.Join(f.hallStore.Dir, fameFileName)
	if err := os.WriteFile(path, seed, 0600); err != nil {
		t.Fatal(err)
	}
	f.hallStore.files = failFameReplace{}
	f.fame = SnapshotFame{Known: true, Result: &FameResult{Name: "new", Score: 200}}
	view, _ := f.campaignEnding()
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(seed, actual) || !view.HasScore || view.Recorded || f.fame.Result.Recorded {
		t.Fatal("failed publication changed old hall or consumed result", err, view)
	}
	f.hallStore.files = nil
	view, _ = f.campaignEnding()
	if !view.Recorded || len(view.Hall) != 2 || view.Hall[0].Name != "new" {
		t.Fatal(view)
	}
	view, _ = f.campaignEnding()
	if len(view.Hall) != 2 {
		t.Fatal("retry repeated successful write")
	}
}

func TestLegacyCompletedEndingDoesNotInventScoreOrCreateHall(t *testing.T) {
	f := completedFameFront(t)
	view, _ := f.campaignEnding()
	if view.HasScore || f.fame.Result != nil {
		t.Fatal("invented result", view)
	}
	if _, err := os.Stat(filepath.Join(f.hallStore.Dir, fameFileName)); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// The terminal route resets the campaign once the hall holds the result; a
// result the store has not accepted keeps the campaign for a retry.
func TestLeaveCampaignEndingResetsOnlyARecordedCampaign(t *testing.T) {
	f := completedFameFront(t)
	f.fame = SnapshotFame{Known: true, Time: 9, Events: 3, Result: &FameResult{Name: "Finisher", Score: 5}}
	f.leaveCampaignEnding()
	if !f.completedCampaign() || f.fame.Result == nil {
		t.Fatal("an unrecorded result must keep the completed campaign")
	}
	f.fame.Result.Recorded = true
	f.leaveCampaignEnding()
	if f.completedCampaign() || f.fame.Known || f.fame.Result != nil || f.fame.Events != 0 {
		t.Fatal("the terminal route did not reset the campaign", f.fame)
	}
}

// The terminal route and every new-game route reset the campaign through one
// routine, so the fields they drop are the same (FAME-033, SAV-1159).
func TestTerminalResetDropsTheNewGameSessionFields(t *testing.T) {
	terminal, shared := nonZeroFrontEnd(), nonZeroFrontEnd()
	terminal.fame.Result, shared.fame.Result = nil, nil
	terminal.leaveCampaignEnding()
	shared.resetSessionForNewGame()
	type session struct {
		Carried     []mapload.PartyMember
		Offered     int
		Difficulty  mapload.Difficulty
		QuickSpells [4]uint32
		Fame        SnapshotFame
		Live        bool
		LiveMission int
		Shop        bool
	}
	read := func(f *FrontEnd) session {
		return session{f.Carried, f.Offered, f.Difficulty, f.quickSpells, f.fame, f.live != nil, f.liveMission, f.Shop != nil}
	}
	if got, want := read(terminal), read(shared); !reflect.DeepEqual(got, want) {
		t.Fatalf("terminal reset %+v, shared reset %+v", got, want)
	}
	if terminal.Town == nil || terminal.originalCity != nil || terminal.liveParty != nil {
		t.Fatal("the terminal reset left session state behind")
	}
}
