package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// TestReleaseMilestone2MapReopen1140 is the "map reopen" row's release
// witness (pipeline/SAV-COMPLETION.md, docs/1140/story.md). No earlier story
// registered one asserting that resuming an original mid-mission save opens
// the map the file itself names: existing release tests exercise session,
// clock, cell-record and actor state on real preserved saves, but none pin
// RestoreOriginal's own Mission.Number/Address against the file's Mission
// and MapName. It drives the same production door
// (originalsave.go RestoreOriginal) TestMilestone2MapReopen's opt-in
// corpus audit exercises independently, over one lawful owner save, in the
// default release gate.
func TestReleaseMilestone2MapReopen1140(t *testing.T) {
	_, payload := groundCorpusFile(t, "2026-08-24/game0021.sav", "7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c")
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	if source.Head.Mission == 0 || source.World == nil {
		t.Fatal("fixture is not a mid-mission save with Mission != 0")
	}

	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	ms, _, err := loadOriginalMission(f, payload)
	if err != nil {
		t.Fatalf("RestoreOriginal: %v", err)
	}
	if uint32(ms.Number) != source.Head.Mission {
		t.Fatalf("live Mission.Number %d, file Mission %d", ms.Number, source.Head.Mission)
	}
	wantSuffix := "/" + strings.ToLower(source.Head.MapName)
	if !strings.HasSuffix(strings.ToLower(ms.Address), wantSuffix) {
		t.Fatalf("live Address %q does not name file map %q", ms.Address, source.Head.MapName)
	}
	t.Logf("Mission %d, MapName %q, live Address %q", source.Head.Mission, source.Head.MapName, ms.Address)
}
