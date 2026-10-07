package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestOriginalActorLoad1138DoesNotRecomputeFromContainer(t *testing.T) {
	source := sav.ActorLoadState{Present: true, OwnWeight: 178, Load: 181, Capacity: 300, Speed: 20}
	got := originalActorLoad(source, 20)
	if got == nil {
		t.Fatal("originalActorLoad returned nil for a present source")
	}
	if got.Load != 181 {
		t.Fatalf("Load = %d, want 181 (the stored value, not a recompute)", got.Load)
	}
	if got.Inventory.OwnWeight != 178 {
		t.Fatalf("OwnWeight = %d, want 178", got.Inventory.OwnWeight)
	}
	// Sanity: an empty-container recompute of this same OwnWeight gives 178,
	// not 181 — confirming 181 is a value only round-tripping the file
	// produces, exactly the disagreement SAV-794 reports.
	if recomputed := got.Inventory.CurrentLoad(); recomputed != 178 {
		t.Fatalf("sanity CurrentLoad() = %d, want 178", recomputed)
	}
}

// TestOriginalActorLoad1138AbsentSourceIsNil confirms an unread SAV load
// state (Present: false) restores no snapshot rather than a zeroed one, so a
// caller cannot mistake "this record had no load words" for "this actor
// carries nothing."
func TestOriginalActorLoad1138AbsentSourceIsNil(t *testing.T) {
	if got := originalActorLoad(sav.ActorLoadState{}, 20); got != nil {
		t.Fatalf("got %+v, want nil for an absent source", got)
	}
}
