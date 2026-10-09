package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestNativePackRepairProtectsRemappedPendingCellIndices(t *testing.T) {
	for _, pending := range []bool{false, true} {
		a := sim.ItemInstance{Code: 0x1626, Kind: 1, Price: 85, NativeRecord: &sim.NativeItemRecord{Class: sim.SourceArmor, Token: sim.SavedObjectToken{Identity: 101, RuntimeID: 41}}}
		b := a.Clone()
		b.NativeRecord.Token.Identity, b.NativeRecord.Token.RuntimeID = 102, 42
		w, err := sim.NewStockedWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{{ID: 7, HP: 10, MaxHP: 10}}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 7, ItemInstances: []sim.ItemInstance{a, b}, OrderedStacks: []sim.ItemStack{sim.StackItem(a, 1), sim.StackItem(b, 1)}}})
		if err != nil {
			t.Fatal(err)
		}
		data := &currentActionData{Party: []currentPartyMember{{Entity: 7}}}
		if pending {
			data.Pending = &currentPendingQueue{Commands: []currentPendingCommand{{Command: sim.DropCarried(70, 1, sim.CellPoint{X: 1, Y: 1})}}}
		}
		repairCurrentNativePacks(&Mission{World: w}, data, map[sim.EntityID]sim.EntityID{70: 7})
		stacks, _ := w.CarriedStacks(7)
		if pending && (len(stacks) != 2 || stacks[1].NativeRecord.Token.Identity != 102) {
			t.Fatal("repair shifted a pending cell after actor remapping")
		}
		if !pending && (len(stacks) != 1 || stacks[0].Count != 2) {
			t.Fatal("repair did not join independent current pack cells")
		}
	}
}
