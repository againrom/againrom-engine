package sim

import (
	"reflect"
	"testing"
)

func TestRearrangeCarriedKeepsSelectedDestination(t *testing.T) {
	for _, bound := range []bool{false, true} {
		for _, whole := range []bool{false, true} {
			w := effectiveMergeWorld(t, true, bound, []ItemStack{StackItem(packMergeBow, 3), StackItem(packMergeBuiltBow(false), 2), StackItem(packMergeBow, 4)}, nil)
			before := cloneStacks(w.carried[0])
			Step(w, []Command{RearrangeCarried(7, 1, 2, whole)})
			pack, _ := w.CarriedStacks(7)
			n := uint32(1)
			if whole {
				n = 2
			}
			target := 2
			if whole {
				target--
			}
			if !reflect.DeepEqual(pack[0], before[0]) || pack[target].Count != 4+n || pack[target].ObjectID != before[2].ObjectID || !ItemEqual(pack[target].Instance(), before[2].Instance()) {
				t.Fatalf("bound=%v whole=%v selected merge = %+v", bound, whole, pack)
			}
			if !whole && (pack[1].Count != 1 || pack[1].ObjectID != before[1].ObjectID) {
				t.Fatalf("partial source = %+v", pack[1])
			}
			if !w.savedMutationValid() {
				t.Fatal("invalid saved objects after merge")
			}
		}
	}
}

func TestRearrangeCarriedUnequalQuantityAndIdentity(t *testing.T) {
	for _, bound := range []bool{false, true} {
		for _, count := range []uint32{1, 3} {
			for _, whole := range []bool{false, true} {
				w := effectiveMergeWorld(t, true, bound, []ItemStack{StackItem(packMergeBow, count), StackItem(packMergeBlockBow(), 2)}, nil)
				before := cloneStacks(w.carried[0])
				Step(w, []Command{RearrangeCarried(7, 0, 1, whole)})
				want := cloneStacks(before)
				if whole || count == 1 {
					want[0], want[1] = want[1], want[0]
				}
				if !reflect.DeepEqual(w.carried[0], want) || !w.savedMutationValid() {
					t.Fatalf("bound=%v count=%d whole=%v swap = %+v want %+v", bound, count, whole, w.carried[0], want)
				}
			}
		}
	}
}

func TestDropCarriedStackPreservesAllUnits(t *testing.T) {
	for _, bound := range []bool{false, true} {
		w := effectiveMergeWorld(t, true, bound, []ItemStack{StackItem(packMergeBow, 4)}, nil)
		id := w.carried[0][0].ObjectID
		Step(w, []Command{DropCarriedStack(7, 0, CellPoint{X: 2, Y: 2})})
		if len(w.carried[0]) != 0 || len(w.sacks) != 1 || len(w.sacks[0].ItemInstances) != 4 || !w.savedMutationValid() {
			t.Fatalf("bound=%v pack=%+v sacks=%+v", bound, w.carried[0], w.sacks)
		}
		for _, item := range w.sacks[0].ItemInstances {
			if item.ObjectID != id || item.Code != packMergeBow.Code {
				t.Fatalf("whole drop changed identity: %+v", item)
			}
		}
	}
}

func TestRearrangeCarriedRefusesQuantityOverflow(t *testing.T) {
	w := effectiveMergeWorld(t, true, false, []ItemStack{StackItem(packMergeBow, 1), StackItem(packMergeBow, ^uint32(0))}, nil)
	before := cloneStacks(w.carried[0])
	Step(w, []Command{RearrangeCarried(7, 0, 1, true)})
	if !reflect.DeepEqual(w.carried[0], before) {
		t.Fatal("overflow changed carried quantities", w.carried[0])
	}
}
