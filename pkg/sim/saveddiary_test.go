package sim

import "testing"

// TestSavedDiariesRoundTripsAndDetaches locks SavedDiaries/
// ImportOriginalDiaries' own contract: a fresh detached copy on read, a
// duplicate owner refused, and SetSavedDiaries replacing outright — the
// priming role the two staging round trips use it for
// (pkg/game/originalholdings.go, pkg/sim/originalliving.go; resumeWorld,
// pkg/game/resume.go, deliberately calls neither —
// TestUnmarshalBinaryOntoTheSameReceiverNeedsNoDiaryPriming below), on
// SavedProjectiles/ImportOriginalProjectiles' own precedent
// (savedprojectile_test.go).
func TestSavedDiariesRoundTripsAndDetaches(t *testing.T) {
	w := &World{}
	in := []SavedDiary{
		{Owner: SavedDiaryOwner{Player: true}, Length: 5, Entries: []SavedDiaryEntry{{Index: 1, Count: 7, Remaining: 1017}}},
		{Owner: SavedDiaryOwner{Actor: 42}, Length: 3, Entries: nil},
	}
	if err := w.ImportOriginalDiaries(in); err != nil {
		t.Fatalf("ImportOriginalDiaries: %v", err)
	}
	got := w.SavedDiaries()
	if len(got) != 2 {
		t.Fatalf("SavedDiaries = %+v, want 2 entries", got)
	}
	if !got[0].Owner.Player || got[0].Length != 5 || len(got[0].Entries) != 1 || got[0].Entries[0].Count != 7 {
		t.Fatalf("player diary = %+v", got[0])
	}
	if got[1].Owner.Player || got[1].Owner.Actor != 42 || got[1].Length != 3 || len(got[1].Entries) != 0 {
		t.Fatalf("actor diary = %+v", got[1])
	}
	got[0].Entries[0].Count = 99 // Mutating the result must not reach the world.
	if again := w.SavedDiaries(); again[0].Entries[0].Count == 99 {
		t.Fatalf("SavedDiaries leaked its backing arrays: %+v", again)
	}

	// Two Player-owned diaries, or two diaries naming the same actor, both
	// refuse outright without disturbing prior state.
	for _, dup := range [][]SavedDiary{
		{{Owner: SavedDiaryOwner{Player: true}}, {Owner: SavedDiaryOwner{Player: true}}},
		{{Owner: SavedDiaryOwner{Actor: 1}}, {Owner: SavedDiaryOwner{Actor: 1}}},
	} {
		if err := w.ImportOriginalDiaries(dup); err == nil {
			t.Fatalf("want an error for duplicate diary owners %+v, got none", dup)
		}
		if got := w.SavedDiaries(); len(got) != 2 {
			t.Fatalf("refused import changed prior state: %+v", got)
		}
	}

	// SetSavedDiaries replaces outright, duplicate owners included: it is a
	// priming primitive for a staging round trip's fresh receiver, not a
	// validated overlay writer.
	dup := []SavedDiary{{Owner: SavedDiaryOwner{Actor: 9}}, {Owner: SavedDiaryOwner{Actor: 9}}}
	w.SetSavedDiaries(dup)
	if got := w.SavedDiaries(); len(got) != 2 {
		t.Fatalf("SetSavedDiaries did not replace outright: %+v", got)
	}
}

// TestUnmarshalBinaryOntoTheSameReceiverNeedsNoDiaryPriming is
// TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming's own proof,
// repeated for savedDiaries: resumeWorld (pkg/game/resume.go) decodes onto
// the SAME receiver ImportOriginalDiaries set, so the composite literal in
// binary.go carries the field across with no priming call.
func TestUnmarshalBinaryOntoTheSameReceiverNeedsNoDiaryPriming(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := w.ImportOriginalDiaries([]SavedDiary{
		{Owner: SavedDiaryOwner{Player: true}, Length: 2, Entries: []SavedDiaryEntry{{Index: 0, Count: 3, Remaining: 1021}}},
	}); err != nil {
		t.Fatalf("ImportOriginalDiaries: %v", err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := w.UnmarshalBinary(form); err != nil { // resumeWorld's own exact shape.
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got := w.SavedDiaries()
	if len(got) != 1 || got[0].Length != 2 || len(got[0].Entries) != 1 || got[0].Entries[0].Count != 3 {
		t.Fatalf("a same-receiver UnmarshalBinary call lost the carried diaries with no priming call: %+v", got)
	}
}
