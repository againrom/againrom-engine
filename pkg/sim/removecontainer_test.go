package sim

// remove's own compaction, repaired to carry a third parallel slice (0123
// T1, AC-6, SC-3).
//
// Before this fix, remove (step.go) rebuilt entities and routes together and
// left carried at its pre-removal length, so index i in carried no longer
// named the same entity as index i in entities once a removal had shifted
// the survivors down — and Stock() (carry.go), which pairs w.carried[i] with
// w.entities[i], panicked once the shorter of the two slices ran out first.
// It reaches w.remove, w.entities and w.carried directly, being internal to
// this package.

import "testing"

// TestRemoveKeepsTheContainerListAlignedWithTheEntityList is AC-6: after a
// world removes an entity, a survivor's container reads back as its own, the
// container list and the entity list are the same length, and the whole-world
// stock read answers without panicking. It panics on master.
//
// EQUIPMENT JOINS THE SAME CHECK (0124 T2): remove was widened to keep a
// fourth slice, w.equipment, aligned on the identical ground carried
// already stands on — a felled entity's slot 6 is set before the removal
// below so a misaligned index would read another survivor's equipment
// rather than merely a zero value either way.
func TestRemoveKeepsTheContainerListAlignedWithTheEntityList(t *testing.T) {
	w := mustStockedWorld(t, 1,
		[]Entity{{ID: 1, X: 1, Y: 1}, {ID: 2, X: 2, Y: 2}},
		[]Stock{{ID: 2, Items: []uint16{0x101}}})
	w.equipment[1][5] = PlainItem(0x201) // entity 2's own slot 6, before entity 1 is removed

	w.remove([]EntityID{1})

	if len(w.entities) != len(w.carried) {
		t.Fatalf("%d entit(y/ies) but %d container(s), want the same length",
			len(w.entities), len(w.carried))
	}
	if len(w.entities) != len(w.equipment) {
		t.Fatalf("%d entit(y/ies) but %d equipment record(s), want the same length",
			len(w.entities), len(w.equipment))
	}

	got, ok := w.Carried(2)
	if !ok {
		t.Fatal("Carried(2) answered not-ok for a surviving entity")
	}
	if !equalCodes(got, []uint16{0x101}) {
		t.Errorf("Carried(2) = %v after removing entity 1, want [0x101] — its own, undisturbed", got)
	}

	gotEquip, ok := w.Equipped(2)
	if !ok {
		t.Fatal("Equipped(2) answered not-ok for a surviving entity")
	}
	if gotEquip[5] != 0x201 {
		t.Errorf("Equipped(2) slot 6 is %#x after removing entity 1, want 0x201 — its own, undisturbed",
			gotEquip[5])
	}

	// The panic this story found: Stock() pairs w.carried[i] with
	// w.entities[i], and a carried left at its old length ran past the
	// shorter, compacted entity slice.
	stock := w.Stock()
	if len(stock) != 1 || stock[0].ID != 2 || !equalCodes(stock[0].Items, []uint16{0x101}) {
		t.Errorf("Stock() = %+v, want one entry for entity 2 holding [0x101]", stock)
	}
}
