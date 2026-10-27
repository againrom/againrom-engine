package sim

import (
	"slices"
	"testing"
)

func TestSourceDropKeepsOtherSackIDs(t *testing.T) {
	item := ItemInstance{Code: 0x0311, Kind: 1, Price: 71}
	for _, class := range []uint8{0, 1} {
		w := mustStockedWorld(t, 21, []Entity{{ID: 1, X: 4, Y: 4}, {ID: 2, X: 8, Y: 8}}, []Stock{{ID: 2, ItemInstances: []ItemInstance{item}}})
		w.InsertActorRuntimeIDs([]EntityID{1, 2})
		w.pourSack(2, 2, 10, nil)
		w.pourSack(3, 2, 10, nil)
		before := w.SackRuntimeIDs()
		w.entities[1].ActorLoad.Source.Class = class
		Step(w, []Command{TerminalKill(2)})
		for n := 0; n < 400 && len(w.Sacks()) < 3; n++ {
			Step(w, nil)
		}
		if len(w.Sacks()) != 3 {
			t.Fatalf("class %d: %d sacks after the drop, want 3", class, len(w.Sacks()))
		}
		got := w.SackRuntimeIDs()
		if !slices.Equal(got[:2], before) && !slices.Equal(got[1:], before) {
			t.Errorf("class %d: ids %v after the drop, existing were %v", class, got, before)
		}
		seen := map[uint32]bool{}
		for _, id := range got {
			if id == 0 || seen[id] {
				t.Errorf("class %d: ids %v carry 0 or a repeat", class, got)
			}
			seen[id] = true
		}
		if class == 1 && !slices.Contains(got, before[len(before)-1]+1) {
			t.Errorf("class %d: ids %v lack the next free id %d", class, got, before[len(before)-1]+1)
		}
	}
}
