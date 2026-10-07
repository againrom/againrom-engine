package sim

import "testing"

// TestSavedProjectilesRoundTripsAndDetaches locks SavedProjectiles/
// ImportOriginalProjectiles' own contract: a fresh detached copy on read,
// duplicate ids refused, and SetSavedProjectiles replacing outright — the
// priming role the two staging round trips use it for
// (pkg/game/originalholdings.go, pkg/sim/originalliving.go; resumeWorld,
// pkg/game/resume.go, deliberately calls neither —
// TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming below), on
// SavedCellRecords/ImportOriginalCellRecords' own precedent
// (savedcellrecord_test.go).
func TestSavedProjectilesRoundTripsAndDetaches(t *testing.T) {
	w := &World{}
	in := SavedProjectiles{
		FreeIndex: 267,
		IDs:       []uint16{266, 7, 266},
		Items: []SavedProjectile{
			{ID: 266, X: 1, ActionSpell: 9},
			{ID: 7, X: 2},
		},
	}
	if err := w.ImportOriginalProjectiles(in); err != nil {
		t.Fatalf("ImportOriginalProjectiles: %v", err)
	}
	got := w.SavedProjectiles()
	if got.FreeIndex != 267 {
		t.Fatalf("FreeIndex = %#x, want 0x10b", got.FreeIndex)
	}
	if len(got.IDs) != 3 || got.IDs[0] != 266 || got.IDs[1] != 7 || got.IDs[2] != 266 {
		t.Fatalf("IDs = %v, want [266 7 266] (wire order and multiplicity preserved)", got.IDs)
	}
	if len(got.Items) != 2 {
		t.Fatalf("Items = %+v, want 2 entries", got.Items)
	}
	got.Items[0].X = 99 // Mutating the result must not reach the world.
	got.IDs[0] = 99
	if again := w.SavedProjectiles(); again.Items[0].X == 99 || again.IDs[0] == 99 {
		t.Fatalf("SavedProjectiles leaked its backing arrays: %+v", again)
	}

	// A duplicate Items id refuses outright, and must not disturb prior state.
	dup := SavedProjectiles{Items: []SavedProjectile{{ID: 5}, {ID: 5}}}
	if err := w.ImportOriginalProjectiles(dup); err == nil {
		t.Fatal("want an error for a duplicate projectile id, got none")
	}
	if got := w.SavedProjectiles(); len(got.Items) != 2 {
		t.Fatalf("refused import changed prior state: %+v", got)
	}

	// SetSavedProjectiles replaces outright, duplicate ids included: it is a
	// priming primitive for a staging round trip's fresh receiver, not a
	// validated overlay writer.
	w.SetSavedProjectiles(SavedProjectiles{FreeIndex: 1, IDs: []uint16{5, 5}})
	if got := w.SavedProjectiles(); got.FreeIndex != 1 || len(got.IDs) != 2 || len(got.Items) != 0 {
		t.Fatalf("SetSavedProjectiles did not replace outright: %+v", got)
	}
}

// TestImportOriginalLivingActorsStagingCarriesSavedProjectiles is
// TestImportOriginalLivingActorsStagingCarriesSavedCellRecords' own
// regression shape applied to this story's carried field: a world whose
// projectile store was already restored (applyOriginalProjectiles, which
// like applyOriginalCellRecords runs before actor admission in both original
// LOAD doors) must not come out of ImportOriginalLivingActors' own staging
// round trip silently emptied.
func TestImportOriginalLivingActorsStagingCarriesSavedProjectiles(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := w.ImportOriginalProjectiles(SavedProjectiles{
		FreeIndex: 267, IDs: []uint16{266}, Items: []SavedProjectile{{ID: 266, X: 5, ActionSpell: 42}},
	}); err != nil {
		t.Fatalf("ImportOriginalProjectiles: %v", err)
	}
	if err := w.ImportOriginalLivingActors(nil); err != nil {
		t.Fatalf("ImportOriginalLivingActors: %v", err)
	}
	got := w.SavedProjectiles()
	if got.FreeIndex != 267 || len(got.Items) != 1 || got.Items[0].X != 5 || got.Items[0].ActionSpell != 42 {
		t.Fatalf("saved projectiles did not survive ImportOriginalLivingActors' own staging: %+v", got)
	}
}

// TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming answers, by
// direct evidence rather than inspection alone, the question DIV-932 leaves
// open for savedCellRecords and this story's own docs/1133/story.md answers
// for savedProjectiles: resumeWorld (pkg/game/resume.go) calls
// ms.World.UnmarshalBinary(form) directly on ms.World itself, never a
// separate staging receiver. UnmarshalBinary's own composite literal reads
// w.savedProjectiles — w being that SAME receiver — before *w is replaced,
// so the value already survives this call with no SetSavedProjectiles priming
// at the call site: priming here would read and rewrite the identical field
// on the identical object, which cannot change its value under any input.
func TestUnmarshalBinaryOntoTheSameReceiverNeedsNoProjectilePriming(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := w.ImportOriginalProjectiles(SavedProjectiles{
		FreeIndex: 267, IDs: []uint16{266}, Items: []SavedProjectile{{ID: 266, X: 5}},
	}); err != nil {
		t.Fatalf("ImportOriginalProjectiles: %v", err)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := w.UnmarshalBinary(form); err != nil { // resumeWorld's own exact shape.
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got := w.SavedProjectiles()
	if got.FreeIndex != 267 || len(got.Items) != 1 || got.Items[0].X != 5 {
		t.Fatalf("a same-receiver UnmarshalBinary call lost the carried store with no priming call: %+v", got)
	}
}
