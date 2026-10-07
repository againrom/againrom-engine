package sim

import (
	"reflect"
	"testing"
)

func TestReplaceGroundSacksIsAtomicCanonicalAndDetached(t *testing.T) {
	w := mustLootWorld(t, 1, sfBounds, []Sack{{X: 9, Y: 9, Items: []uint16{99}}})
	before := w.Hash()
	if err := w.ReplaceGroundSacks([]Sack{{X: 1, Y: 1}, {X: 10, Y: 1}}); err == nil || w.Hash() != before {
		t.Fatal("invalid late sack changed the world")
	}
	item := ItemInstance{Code: 12, Kind: 3, Price: -10, Effects: []ItemEffect{{Kind: 11, Mode: 1, Operand: 3}, {Kind: 11, Operand: 5}}}
	in := []Sack{{X: 3, Y: 2, Gold: 7, ItemInstances: []ItemInstance{item}}, {X: 1, Y: 1, Gold: 4}, {X: 3, Y: 2, Gold: 9, Items: []uint16{13}}}
	if err := w.ReplaceGroundSacks(in); err != nil {
		t.Fatal(err)
	}
	in[0].ItemInstances[0].Effects[0].Operand = 999
	got := w.Sacks()
	if len(got) != 2 || got[0].X != 1 || got[1].Gold != 16 || !reflect.DeepEqual(got[1].Items, []uint16{12, 13}) || got[1].ItemInstances[0].Effects[0].Operand != 3 {
		t.Fatalf("replacement = %+v", got)
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil || back.Hash() != w.Hash() || !reflect.DeepEqual(back.Sacks(), w.Sacks()) {
		t.Fatalf("native roundtrip: %v", err)
	}
	if err := w.ReplaceGroundSacks(nil); err != nil || len(w.Sacks()) != 0 {
		t.Fatal("empty saved list did not clear fresh loot")
	}
}
