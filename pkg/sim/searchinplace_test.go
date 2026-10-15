package sim

import (
	"math/rand/v2"
	"slices"
	"testing"
)

// searchInPlace must answer exactly as slices.BinarySearchFunc does, on
// sorted, duplicated and unsorted input alike, because both probe the same
// midpoints in the same order.
func TestSearchInPlaceAnswersAsBinarySearchFunc(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	compare := func(a, b SavedObjectID) int { return savedIDCompare(a, b) }
	for round := range 2000 {
		rows := make([]SavedSpellObject, rng.IntN(12))
		for i := range rows {
			rows[i].ID = SavedObjectID(rng.IntN(16))
		}
		if round%3 != 0 {
			slices.SortFunc(rows, func(a, b SavedSpellObject) int { return compare(a.ID, b.ID) })
		}
		for target := SavedObjectID(0); target < 17; target++ {
			wantAt, wantFound := slices.BinarySearchFunc(rows, target, func(v SavedSpellObject, id SavedObjectID) int { return compare(v.ID, id) })
			gotAt, gotFound := searchInPlace(rows, target, func(v *SavedSpellObject, id SavedObjectID) int { return compare(v.ID, id) })
			if gotAt != wantAt || gotFound != wantFound {
				t.Fatalf("rows %v target %d: got %d %v, want %d %v", rows, target, gotAt, gotFound, wantAt, wantFound)
			}
		}
	}
}
