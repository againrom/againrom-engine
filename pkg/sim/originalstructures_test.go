package sim

import (
	"reflect"
	"testing"
)

func TestOriginalStructureHealthBatchIsAtomicAndNative(t *testing.T) {
	w := destructibleWorld(t, []Structure{
		{ID: 2, Field42: 100, MaxHealth: 100, Col: 5, Row: 6, Width: 2, Height: 2, Attach: 15},
		{ID: 7, Field42: 90, MaxHealth: 90, Col: 8, Row: 7, Width: 1, Height: 1, Attach: 1},
	})
	before := w.Hash()
	shape := w.Structures()
	good := OriginalStructureHealth{ID: 2, Health: 7, MaxHealth: 31}
	for _, bad := range []OriginalStructureHealth{{ID: 2, Health: 3}, {ID: 99, Health: 3}} {
		if err := w.ImportOriginalStructureHealth([]OriginalStructureHealth{good, bad}); err == nil || w.Hash() != before {
			t.Fatalf("invalid batch mutated world: %+v %v", bad, err)
		}
	}
	for _, hp := range []uint16{7, 0, 0xffff} {
		if err := w.ImportOriginalStructureHealth([]OriginalStructureHealth{good, {ID: 7, Health: hp, MaxHealth: 0}}); err != nil {
			t.Fatal(err)
		}
		if w.Hash() == before || w.Tick() != 0 {
			t.Fatal("health import did not move hash or advanced time")
		}
		shape[0].Field42, shape[0].MaxHealth = 7, 31
		shape[1].Field42, shape[1].MaxHealth = hp, 0
		if !reflect.DeepEqual(w.Structures(), shape) {
			t.Fatal("health import changed identity/shape/order")
		}
		if len(w.structureSlots) != 5 {
			t.Fatal("zero/negative HP detached occupied cells")
		}
		b, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var back World
		if err := back.UnmarshalBinary(b); err != nil || back.Hash() != w.Hash() || !reflect.DeepEqual(back.Structures(), shape) {
			t.Fatalf("native structure round trip: %v", err)
		}
	}
	var nilWorld *World
	if err := nilWorld.ImportOriginalStructureHealth(nil); err == nil {
		t.Fatal("accepted nil world")
	}
}

func TestOriginalStructureHealthControlsStrikeAndNativeContinuation(t *testing.T) {
	a, s := structureCombatActor(), structureCombatTarget()
	a.Owner = SelfSlot
	w := structureCombatWorld(t, 0, a, s)
	if err := w.ImportOriginalStructureHealth([]OriginalStructureHealth{{ID: 0, Health: 45, MaxHealth: 0}}); err != nil {
		t.Fatal(err)
	}
	// Zero saved maximum makes the physical resolver return without damage.
	// Fresh maximum100 would let the same attack damage it.
	Step(w, []Command{{Kind: KindAttackStructure, Entity: 0, X: 0}})
	for range 32 {
		Step(w, nil)
	}
	if w.Structures()[0].Field42 != 45 {
		t.Fatal("saved maximum did not reach damage resolver")
	}
	if err := w.ImportOriginalStructureHealth([]OriginalStructureHealth{{ID: 0, Health: 45, MaxHealth: 100}}); err != nil {
		t.Fatal(err)
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	for range 64 {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("strike continuation differs at tick%d", w.Tick())
		}
	}
	if int16(w.Structures()[0].Field42) > 0 || w.Entities()[0].HasAttackTarget {
		t.Fatal("saved health did not reach strike/ruin stop")
	}
}
