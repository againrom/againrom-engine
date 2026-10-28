package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// TestDiaryEntryConversionRoundTrips locks savDiaryEntriesToSaved/
// savedDiaryEntriesToSav as exact inverses, including the nil/empty case
// applyOriginalDiaries and exportOriginalDiaries both rely on: an
// all-default Diary (no departing entries) converts to a nil slice, not an
// empty non-nil one, on sav.DiaryEntry's own sparse-projection contract
// (diary.go).
func TestDiaryEntryConversionRoundTrips(t *testing.T) {
	if got := savDiaryEntriesToSaved(nil); got != nil {
		t.Fatalf("savDiaryEntriesToSaved(nil) = %v, want nil", got)
	}
	if got := savedDiaryEntriesToSav(nil); got != nil {
		t.Fatalf("savedDiaryEntriesToSav(nil) = %v, want nil", got)
	}
	in := []sav.DiaryEntry{{Index: 3, Count: 7, Remaining: 1017}, {Index: 41, Count: 1, Remaining: 0}}
	saved := savDiaryEntriesToSaved(in)
	want := []sim.SavedDiaryEntry{{Index: 3, Count: 7, Remaining: 1017}, {Index: 41, Count: 1, Remaining: 0}}
	if len(saved) != len(want) || saved[0] != want[0] || saved[1] != want[1] {
		t.Fatalf("savDiaryEntriesToSaved(%+v) = %+v, want %+v", in, saved, want)
	}
	back := savedDiaryEntriesToSav(saved)
	if len(back) != len(in) || back[0] != in[0] || back[1] != in[1] {
		t.Fatalf("round trip = %+v, want %+v", back, in)
	}
}

// TestApplyOriginalDiariesRefusesAMissionWithNoWorld is applyOriginalProjectiles'
// own nil-guard precedent (originalprojectiles.go), repeated for the Diary
// importer: a construction bug that reaches this call with no world must
// fail loud, not silently skip diary restoration.
func TestApplyOriginalDiariesRefusesAMissionWithNoWorld(t *testing.T) {
	if err := applyOriginalDiaries(&Mission{}, nil, nil, nil); err == nil {
		t.Fatal("want an error for a mission with no world")
	}
	if err := applyOriginalDiaries(nil, nil, nil, nil); err == nil {
		t.Fatal("want an error for a nil mission")
	}
}

// TestExportOriginalDiariesRefusesANilOrCitySave is exportOriginalProjectiles'
// own nil-guard/city-save precedent, repeated for the Diary exporter: a city
// save has no world half (exportOriginalProjectiles' own rule), and a nil
// world or file must refuse rather than patch nothing silently.
func TestExportOriginalDiariesRefusesANilOrCitySave(t *testing.T) {
	if err := exportOriginalDiaries(nil, &sim.World{}); err == nil {
		t.Fatal("want an error for a nil file")
	}
	if err := exportOriginalDiaries(&sav.File{}, &sim.World{}); err == nil {
		t.Fatal("want an error for a save with no world half")
	}
}

// The export writes the Diary of a Human another Player owns as well as the
// walked Player's own. LOAD carries both (importSavedDiaries), so an export
// that reached only the walked Player's actor list refused such a save.
func TestExportDiariesWritesEveryPlayersActorDiary(t *testing.T) {
	diary := func(index int, count uint32) *sav.Diary {
		return &sav.Diary{Length: 4, Entries: []sav.DiaryEntry{{Index: index, Count: count, Remaining: 1000}}}
	}
	mine := &poolFixtureActor{human: true, cell: 0x0605, hp: 10, maxHP: 20, diary: diary(1, 3)}
	theirs := &poolFixtureActor{human: true, cell: 0x0807, hp: 10, maxHP: 20, diary: diary(2, 5)}
	plain := &poolFixtureActor{human: true, cell: 0x0909, hp: 10, maxHP: 20}
	raw := savedContainer(poolFixtureBody([]*poolFixturePlayer{
		{groups: [][]*poolFixtureActor{{mine}}},
		{groups: [][]*poolFixtureActor{{theirs, plain}}},
	}, nil))
	open := func(raw []byte) (*sav.File, map[uint16]sav.ActorRecord) {
		t.Helper()
		f, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		graph, err := f.ActorGraph()
		if err != nil {
			t.Fatal(err)
		}
		byCell := map[uint16]sav.ActorRecord{}
		for _, a := range graph.Actors {
			byCell[a.Cell] = a
		}
		return f, byCell
	}
	f, actors := open(raw)
	if !actors[0x0605].Character.HasDiary || !actors[0x0807].Character.HasDiary || actors[0x0909].Character.HasDiary {
		t.Fatalf("fixture lost its shape: %+v", actors)
	}
	walked, _, err := f.PartyWalk()
	if err != nil || len(walked) != 1 || walked[0].Cell != 0x0605 {
		t.Fatalf("the walked Player's actors are %+v, %v; want the first Player's one Human", walked, err)
	}
	index := map[sim.EntityID]uint16{1: actors[0x0605].ArchiveIndex, 2: actors[0x0807].ArchiveIndex, 3: actors[0x0909].ArchiveIndex}
	carried := []sim.SavedDiary{
		{Owner: sim.SavedDiaryOwner{Actor: 1}, Length: 4, Entries: []sim.SavedDiaryEntry{{Index: 0, Count: 7, Remaining: 9}}},
		{Owner: sim.SavedDiaryOwner{Actor: 2}, Length: 4, Entries: []sim.SavedDiaryEntry{{Index: 3, Count: 11, Remaining: 13}}},
	}
	if err := exportDiaries(f, carried, index); err != nil {
		t.Fatalf("exportDiaries: %v", err)
	}
	_, written := open(f.Marshal())
	for cell, want := range map[uint16]sav.DiaryEntry{0x0605: {Index: 0, Count: 7, Remaining: 9}, 0x0807: {Index: 3, Count: 11, Remaining: 13}} {
		got := written[cell].Character.Diary
		if got.Length != 4 || len(got.Entries) != 1 || got.Entries[0] != want {
			t.Errorf("actor at %#04x: written Diary %+v, want the one entry %+v", cell, got, want)
		}
	}
	// Controls: an owner with a null Diary reference, and one with no archive
	// index, still refuse.
	for _, owner := range []sim.EntityID{3, 4} {
		f, _ := open(raw)
		if err := exportDiaries(f, []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Actor: owner}, Length: 4}}, index); err == nil {
			t.Errorf("owner %d: want a refusal", owner)
		}
	}
}
