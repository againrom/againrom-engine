package sim

import (
	"reflect"
	"testing"
)

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

// Another Player's Diary is carried through the World's byte form beside the
// local Player's and an actor's, and follows its Player's identity rename.
func TestAnotherPlayerDiaryCarriesAndFollowsIdentity(t *testing.T) {
	w := participantWorld(t)
	if err := w.RestorePlayerParticipants([]PlayerParticipant{{101, 0}, {205, 1}}, true); err != nil {
		t.Fatal(err)
	}
	in := []SavedDiary{
		{Owner: SavedDiaryOwner{Player: true}, Length: 119, Entries: []SavedDiaryEntry{{Index: 64, Count: 2, Remaining: 1022}}},
		{Owner: SavedDiaryOwner{PlayerID: 205}, Length: 119, Entries: []SavedDiaryEntry{{Index: 72, Count: 1, Remaining: 1023}}},
		{Owner: SavedDiaryOwner{Actor: 42}, Length: 119},
	}
	if err := w.ImportOriginalDiaries(in); err != nil {
		t.Fatal(err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold.SavedDiaries(), in) {
		t.Fatalf("byte form changed the Diaries: %+v", cold.SavedDiaries())
	}
	if err := cold.RestoreCurrentPlayerIdentities(map[uint32]uint32{101: 501, 205: 605}); err != nil {
		t.Fatal(err)
	}
	want := cold.SavedDiaries()
	if want[1].Owner != (SavedDiaryOwner{PlayerID: 605}) || want[0].Owner != (SavedDiaryOwner{Player: true}) || want[2].Owner != (SavedDiaryOwner{Actor: 42}) {
		t.Fatalf("identity rename did not move only the other Player's Diary: %+v", want)
	}
	// Another Player's construction-default Diary is not carried; the local
	// Player's is.
	w.SetDiaryUnits(119, nil)
	if err := w.ImportOriginalDiaries([]SavedDiary{{Owner: SavedDiaryOwner{Player: true}, Length: 119}, {Owner: SavedDiaryOwner{PlayerID: 205}, Length: 119}, {Owner: SavedDiaryOwner{PlayerID: 101}, Length: 118}}); err != nil {
		t.Fatal(err)
	}
	if got := w.SavedDiaries(); len(got) != 2 || !got[0].Owner.Player || got[1].Owner.PlayerID != 101 {
		t.Fatalf("construction-default Diary of another Player carried: %+v", got)
	}
	for _, bad := range [][]SavedDiary{
		{{Owner: SavedDiaryOwner{PlayerID: 205}}, {Owner: SavedDiaryOwner{PlayerID: 205}}},
		{{Owner: SavedDiaryOwner{PlayerID: 205, Actor: 3}}},
		{{Owner: SavedDiaryOwner{Player: true, PlayerID: 205}}},
	} {
		if w.ImportOriginalDiaries(bad) == nil {
			t.Fatalf("invalid owners admitted: %+v", bad)
		}
	}
}
