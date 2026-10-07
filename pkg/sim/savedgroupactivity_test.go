package sim

import (
	"encoding/binary"
	"testing"
)

func TestSavedGroupActivityRebuildsAndKeepsCoarseCornersAsleep(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 128, Height: 128}, []Entity{
		{ID: 1, Owner: SelfSlot, X: 80, Y: 80, HP: 100, MaxHP: 100},
		{ID: 2, Owner: 2, X: 96, Y: 88, HP: 100, MaxHP: 100},
		{ID: 3, Owner: 2, X: 96, Y: 96, HP: 100, MaxHP: 100},
		{ID: 4, Owner: 2, X: 10, Y: 10, HP: 100, MaxHP: 100},
		{ID: 5, Owner: 2, X: 120, Y: 8, HP: 100, MaxHP: 100},
	})
	groups := []SavedGroup{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5, Authored: true}}
	for n := 0; n < 3; n++ {
		groups[n].Members = []SavedGroupMember{{Entity: EntityID(n + 2), Bound: true}}
		groups[n].AI[0x45] = 17
	}
	groups[4].Members = []SavedGroupMember{{Entity: 5, Bound: true}}
	groups[4].AI[0x45] = 1  // explicit native command lifetime is unchanged
	groups[3].AI[0x45] = 23 // no represented member: retain the raw byte
	binary.LittleEndian.PutUint32(groups[2].AI[0x48:], 0x100)
	if err := w.ImportSavedGroups(groups, nil); err != nil {
		t.Fatal(err)
	}
	check := func(want [5]byte) {
		t.Helper()
		w.refreshSavedGroupActivity()
		for i, g := range w.savedGroups.Groups {
			if g.AI[0x45] != want[i] {
				t.Fatalf("group%d activity %d, want%d", g.ID, g.AI[0x45], want[i])
			}
		}
	}
	check([5]byte{1, 0, 1, 23, 1})
	check([5]byte{1, 0, 1, 23, 1}) // no accumulation across ticks
	w.entities[0].X, w.entities[0].Y = 96, 96
	check([5]byte{1, 1, 1, 23, 1})
	w.entities[0].X, w.entities[0].Y = 8, 120
	check([5]byte{0, 0, 1, 23, 1})
	w.entities[0].HP = 0
	check([5]byte{0, 0, 1, 23, 1})
}
