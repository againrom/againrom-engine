package sim

import "testing"

// TestSavedCellRecordsRoundTripsAndDetaches locks SavedCellRecords/
// ImportOriginalCellRecords' own contract: ascending order regardless of
// input order, a fresh detached slice on read, and SetSavedCellRecords
// replacing outright (the priming role the two staging round trips use it
// for, pkg/game/originalholdings.go and pkg/sim/originalliving.go).
func TestSavedCellRecordsRoundTripsAndDetaches(t *testing.T) {
	w := &World{}
	in := []SavedCellRecord{
		{Cell: 0x0200, LayerCount: 2, Sack: 9},
		{Cell: 0x0100, LayerCount: 1, Ground: SavedCellActorSlot{Key: 7, Bound: true, Entity: 3}},
	}
	if err := w.ImportOriginalCellRecords(in); err != nil {
		t.Fatalf("ImportOriginalCellRecords: %v", err)
	}
	got := w.SavedCellRecords()
	if len(got) != 2 || got[0].Cell != 0x0100 || got[1].Cell != 0x0200 {
		t.Fatalf("SavedCellRecords not ascending: %+v", got)
	}
	if got[0].Ground != (SavedCellActorSlot{Key: 7, Bound: true, Entity: 3}) {
		t.Fatalf("Ground slot not preserved: %+v", got[0].Ground)
	}
	got[0].LayerCount = 99 // Mutating the result must not reach the world.
	if again := w.SavedCellRecords(); again[0].LayerCount != 1 {
		t.Fatalf("SavedCellRecords leaked its backing array: %+v", again)
	}

	// A duplicate key refuses outright: the caller (applyOriginalCellRecords)
	// is the one responsible for last-write-wins deduplication first, on
	// validateSavedStructures' own precedent for StructureCell.
	dup := []SavedCellRecord{{Cell: 5}, {Cell: 5}}
	if err := w.ImportOriginalCellRecords(dup); err == nil {
		t.Fatal("want an error for a duplicate cell key, got none")
	}
	// A refused import must not have disturbed the prior state.
	if got := w.SavedCellRecords(); len(got) != 2 {
		t.Fatalf("refused import changed prior state: %+v", got)
	}

	// SetSavedCellRecords replaces outright, unsorted input included: it is a
	// priming primitive for a staging round trip's fresh receiver, not a
	// validated overlay writer.
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x0300}})
	if got := w.SavedCellRecords(); len(got) != 1 || got[0].Cell != 0x0300 {
		t.Fatalf("SetSavedCellRecords did not replace outright: %+v", got)
	}
}

// TestImportOriginalLivingActorsStagingCarriesSavedCellRecords is the
// regression witness for the defect this story found: ImportOriginalLivingActors'
// own staging round trip (originalliving.go) used to build its "checked" receiver
// with only sourceDerive primed, so a world whose cell-record residue was
// already restored (applyOriginalCellRecords, which runs before actor
// admission in both original LOAD doors) came out the other side silently
// emptied. Confirmed to fail with that exact symptom when the fix
// (checked.SetSavedCellRecords) is reverted.
func TestImportOriginalLivingActorsStagingCarriesSavedCellRecords(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if err := w.ImportOriginalCellRecords([]SavedCellRecord{{Cell: 0x0100, LayerCount: 5, Sack: 42}}); err != nil {
		t.Fatalf("ImportOriginalCellRecords: %v", err)
	}
	if err := w.ImportOriginalLivingActors(nil); err != nil {
		t.Fatalf("ImportOriginalLivingActors: %v", err)
	}
	got := w.SavedCellRecords()
	if len(got) != 1 || got[0].Cell != 0x0100 || got[0].LayerCount != 5 || got[0].Sack != 42 {
		t.Fatalf("saved cell records did not survive ImportOriginalLivingActors' own staging: %+v", got)
	}
}
