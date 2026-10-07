package data

import (
	"strconv"
	"testing"

	"againrom/internal/synth"
)

// nameSortedIndices is the order a .reg node table actually holds these sections
// in: child lists are name-sorted, so Unit10 and Unit11 sit between Unit1 and
// Unit2. All() is numeric section order, which is therefore a real reordering of
// what the tree offers — a loader enumerating Root.Children hands back this
// order instead.
var nameSortedIndices = []int{0, 1, 10, 11, 2, 3, 4, 5, 6, 7, 8, 9}

// sparseID is the fixture's ID at section index i: ordered, sparse, and never
// equal to the index, so a loader that mistook one for the other is visible.
func sparseID(i int) int32 { return int32(3*i + 1) }

// SC-6 (AC-6). All() is numeric section order and its length is the [Global]
// count; ByID hits every ID and misses every non-ID; and the two are one
// list, so a class is reachable by ByID exactly when it appears in All().
func TestAllIsNumericOrderAndByIDIsSparse(t *testing.T) {
	sections := make([]synth.RegNode, 0, len(nameSortedIndices))
	for _, i := range nameSortedIndices {
		sections = append(sections, regDir("Unit"+strconv.Itoa(i),
			regInt("ID", sparseID(i)), regInt("File", 0)))
	}
	cs := loadUnits(t, unitsReg(t, int32(len(nameSortedIndices)), sections...))

	all := cs.All()
	if len(all) != len(nameSortedIndices) {
		t.Fatalf("len(All()) = %d, want %d — the [Global] count", len(all), len(nameSortedIndices))
	}
	for i, c := range all {
		if c.ID != sparseID(i) {
			t.Errorf("All()[%d].ID = %d, want %d — All() is numeric section order, "+
				"not the name-sorted order of the node table", i, c.ID, sparseID(i))
		}
	}

	for i := range nameSortedIndices {
		c, ok := cs.ByID(sparseID(i))
		if !ok {
			t.Errorf("ByID(%d) missed a class that is in All()", sparseID(i))
			continue
		}
		if c != all[i] {
			t.Errorf("ByID(%d) is not All()[%d] — the map must point into the one list",
				sparseID(i), i)
		}
	}

	// 0 is held by no class here, 2 and 3 fall in the gaps the sparse domain
	// leaves, and 35 is one past the maximum.
	for _, miss := range []int32{0, 2, 3, sparseID(len(nameSortedIndices)-1) + 1} {
		if c, ok := cs.ByID(miss); ok {
			t.Errorf("ByID(%d) hit %v, want a miss — no class holds that ID", miss, c)
		}
	}

	// All() hands out a copy of the pointer slice.
	all[0] = nil
	if again := cs.All(); again[0] == nil {
		t.Error("All() returned the collection's own slice; a caller can empty it")
	}
}
