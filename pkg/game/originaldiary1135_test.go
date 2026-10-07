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
