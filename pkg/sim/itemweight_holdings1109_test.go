package sim

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestInstanceWeightOriginalStockLateResidueIsAtomic(t *testing.T) {
	for _, slot := range []string{"carried", "held", "worn"} {
		t.Run(slot, func(t *testing.T) {
			w := mustStockedWorld(t, 1, []Entity{{ID: 1, HP: 10, MaxHP: 10}, {ID: 2, HP: 10, MaxHP: 10}},
				[]Stock{{ID: 1, Items: []uint16{0x0e06}}, {ID: 2, Items: []uint16{0x0e07}}})
			before, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			hash := w.Hash()
			first := OriginalActorStock{ID: 1, Carried: []ItemStack{{Code: 0x0e06, Count: 3, Weight: -5, WeightPresent: true}}}
			last := OriginalActorStock{ID: 2}
			bad := ItemInstance{Code: 0x0101, Weight: -7} // absent, nonzero residue
			switch slot {
			case "carried":
				last.Carried = []ItemStack{StackItem(bad, 1)}
			case "held":
				last.Equipped[0] = bad
			case "worn":
				bad.Code = 0x0c01
				last.Equipped[11] = bad
			}
			err = w.ImportOriginalActorStock([]OriginalActorStock{first, last})
			if err == nil || !strings.Contains(err.Error(), "absent item weight") {
				t.Fatalf("late invalid weight was admitted: %v", err)
			}
			after, err := w.MarshalBinary()
			if err != nil || hash != w.Hash() || !bytes.Equal(before, after) {
				t.Fatalf("late refusal changed earlier stock/cache/load: %v", err)
			}
		})
	}
}

func TestInstanceWeightOriginalStockKeepsKindPriceAndPresence(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, HP: 10, MaxHP: 10}}, nil)
	// Literal cells differ by exactly one retained field. None may disappear
	// in ImportOriginalActorStock, nor in the post-weight native canonical fold.
	want := []ItemStack{
		{Code: 0x0701, Kind: 1, Price: 50, Count: 2, Weight: 0, WeightPresent: true},
		{Code: 0x0701, Kind: 2, Price: 50, Count: 3, Weight: 0, WeightPresent: true},
		{Code: 0x0701, Kind: 1, Price: 51, Count: 4, Weight: 0, WeightPresent: true},
		{Code: 0x0701, Kind: 1, Price: 50, Count: 5, Weight: -7, WeightPresent: true},
		{Code: 0x0701, Kind: 1, Price: 50, Count: 6, Weight: 9, WeightPresent: true},
		{Code: 0x0701, Kind: 1, Price: 50, Count: 7},
	}
	if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1, Carried: want}}); err != nil {
		t.Fatal(err)
	}
	assert := func(w *World) {
		t.Helper()
		got, _ := w.CarriedStacks(1)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("distinct cell state folded: %+v", got)
		}
	}
	assert(w)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	assert(&fresh)
	if fresh.Hash() != w.Hash() {
		t.Fatal("native reload changed weight identity")
	}
}
