package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

// SAVE allocates no runtime id: a record's id is the one its World object
// carries, zero included, even where another record carries the same id
// (SAV-1371). The document projection writes only the edges.
func TestCurrentRuntimeProjectionAllocatesNoID(t *testing.T) {
	classes := []string{"Weapon", "Human", "Unit", "Building", "Sack", "Unit"}
	ids := []uint32{1, 1, 0, 1, 1, 0x10004}
	doc := sav.DocumentData{}
	for i, class := range classes {
		r := mustNewRecord(class)
		mustSetValue(&r, "RuntimeID", ids[i])
		doc.Objects = append(doc.Objects, r)
	}
	if err := projectCurrentRuntimeIDs(&doc, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		if got, err := savedStructureValue(&doc.Objects[i], "RuntimeID"); err != nil || got != ids[i] {
			t.Fatalf("%s RuntimeID %d, want %d (%v)", classes[i], got, ids[i], err)
		}
	}
}
