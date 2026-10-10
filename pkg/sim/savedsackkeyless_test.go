package sim

import "testing"

// A keyless Sack binding joins a cell with no record; an exact binding still
// requires the cell record naming the Sack, and a keyless one refuses a cell
// whose Sack slot names it.
func TestSavedSackKeylessBindingNeedsACellNamingNoSack(t *testing.T) {
	w := goldPickupSavedWorld(t, 31)
	r := w.SavedObjects()
	w.savedObjects = nil
	w.sacks[0].ObjectID = 0
	b := []SavedSackBinding{{ID: r.Sacks[0].ID, X: 10, Y: 20, Gold: 31}}
	b[0].Keyless = true
	if err := w.ImportSavedObjects(r, b); err == nil {
		t.Fatal("keyless binding accepted a cell whose Sack slot names the Sack")
	}
	w.savedMotion.Cells, w.savedStructureCells = nil, nil
	b[0].Keyless = false
	if err := w.ImportSavedObjects(r, b); err == nil {
		t.Fatal("exact binding accepted a cell with no record")
	}
	b[0].Keyless = true
	n := w
	if err := n.ImportSavedObjects(r, b); err != nil {
		t.Fatal(err)
	}
	if got := n.Sacks(); len(got) != 1 || got[0].ObjectID != b[0].ID {
		t.Fatal("keyless binding did not bind the native Sack", got)
	}
}
