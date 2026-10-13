package sim

import (
	"bytes"
	"testing"
)

func TestSightOfStampsOnlyTheNamedBodiesOfTheOwner(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10, Owner: SelfSlot, ScanRange: 3},
		{ID: 2, X: 20, Y: 20, HP: 10, MaxHP: 10, Owner: SelfSlot, ScanRange: 3},
		{ID: 3, X: 30, Y: 30, HP: 10, MaxHP: 10, Owner: 2, ScanRange: 3},
	}
	w, err := NewTerrainWorld(1, Bounds{Width: 45, Height: 45}, ModeCanonical, Terrain{}, ents, nil)
	if err != nil {
		t.Fatal(err)
	}
	both := w.SightOf(SelfSlot, []EntityID{1, 2, 3})
	if !bytes.Equal(both, w.Sight(SelfSlot)) {
		t.Fatal("SightOf over every body is not Sight")
	}
	one := w.SightOf(SelfSlot, []EntityID{1})
	n, other := 0, 0
	for i, v := range one {
		if v != 0 {
			n++
			if both[i] == 0 {
				t.Fatal("SightOf lit a cell Sight does not")
			}
		}
	}
	for _, v := range w.SightOf(SelfSlot, []EntityID{2}) {
		if v != 0 {
			other++
		}
	}
	total := 0
	for _, v := range both {
		if v != 0 {
			total++
		}
	}
	if n == 0 || other == 0 || n+other != total {
		t.Fatalf("SightOf bodies 1 and 2 light %d and %d cells, Sight lights %d", n, other, total)
	}
	for _, v := range w.SightOf(SelfSlot, []EntityID{3}) {
		if v != 0 {
			t.Fatal("SightOf stamped a body of another owner")
		}
	}
}
