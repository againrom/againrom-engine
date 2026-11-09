package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The arm's index is the pack cell, a stack's: a scroll behind a stack of two
// is the arm's own item, and a scroll with a native record keeps its identity
// across copies, so the release orders the cast in both cases.
func TestAScrollBehindAStackOrWithANativeRecordCasts(t *testing.T) {
	table := scrollUseTable()
	scroll := mapload.ItemInstanceFromCode(scrollUseOrdinary, table)
	potion := sim.ItemInstance{Code: 0x0e01, Kind: 3, Price: 10, Effects: []sim.ItemEffect{{Kind: 50, Operand: 5}}}
	native := scroll.Clone()
	native.NativeRecord = &sim.NativeItemRecord{Class: 1}
	for _, tc := range []struct {
		name  string
		items []sim.ItemInstance
	}{{"stack", []sim.ItemInstance{potion, potion, scroll}}, {"native", []sim.ItemInstance{native}}} {
		t.Run(tc.name, func(t *testing.T) {
			mw := scrollUseMission(t, true, tc.items...)
			stacks, _ := mw.world.CarriedStacks(7)
			last := len(stacks) - 1
			mw.useScroll(7, last, itemCastKey(stacks[last].Instance()), 8, 0, 0, false)
			if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScroll {
				t.Fatalf("the release queued %+v, want one scroll use", mw.pending)
			}
		})
	}
}
