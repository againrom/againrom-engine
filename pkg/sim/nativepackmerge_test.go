package sim

import "testing"

func nativePackItem(identity, runtime uint32) ItemInstance {
	i := ItemInstance{Code: 0x1626, Kind: 1, Price: 85}
	i.NativeRecord = &NativeItemRecord{Class: SourceArmor, Token: SavedObjectToken{Identity: identity, RuntimeID: runtime, T0C: 6, T0E: 33, Reference: 17}, F45: 1, F46: 1, F48: 6}
	return i
}

func TestNativePackRepairKeepsCursorNullsAliasesAndBlockedActions(t *testing.T) {
	a, b := nativePackItem(101, 41), nativePackItem(102, 42)
	c := PlainItem(0x0202)
	for _, control := range []string{"merge", "blocked", "alias", "scroll"} {
		w := packMergeWorld(t, false, false, []ItemStack{StackItem(a, 1), StackItem(b, 1), {}, StackItem(c, 1)}, nil)
		w.entities[0].ActorLoad = ActorLoad{Present: true, ContainerPresent: true, InsertIndex: 2, Accumulator: 17}
		var blocked []EntityID
		switch control {
		case "blocked":
			blocked = []EntityID{7}
		case "alias":
			w.equipment[1][0] = a.Clone()
		case "scroll":
			w.scrollCasts = []ScrollCast{{Caster: 7, Index: 2, Item: PlainItem(0x0e01)}}
		}
		before := w.Hash()
		got := w.RepairNativePackCells([]EntityID{7}, blocked)
		if control != "merge" {
			if got != 0 || w.Hash() != before {
				t.Fatalf("%s changed protected cells", control)
			}
			continue
		}
		if got != 1 || len(w.carried[0]) != 3 || w.carried[0][0].Count != 2 || !StackStateEqual(w.carried[0][1], ItemStack{}) || w.entities[0].ActorLoad.InsertIndex != 1 || w.entities[0].ActorLoad.Accumulator != 17 || *w.carried[0][0].NativeRecord != *a.NativeRecord {
			t.Fatal("repair lost count, cursor, null slot, bookkeeping or destination record")
		}
		after := w.Hash()
		if w.RepairNativePackCells([]EntityID{7}, nil) != 0 || w.Hash() != after {
			t.Fatal("repair is not idempotent")
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != after {
			t.Fatal("repaired pack lost cold identity", err)
		}
	}
}

func TestNativeItemsWithDistinctObjectKeysJoinThePack(t *testing.T) {
	a, b := nativePackItem(101, 41), nativePackItem(102, 42)
	if StackStateEqual(StackItem(a, 1), StackItem(b, 1)) {
		t.Fatal("distinct records lost exact state identity")
	}
	w := packMergeWorld(t, false, false, nil, nil, a)
	if err := w.TakeSack(7, 3, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.ReplaceGroundSacks([]Sack{makeSack(3, 1, 0, []ItemInstance{b})}); err != nil {
		t.Fatal(err)
	}
	if err := w.TakeSack(7, 3, 1); err != nil {
		t.Fatal(err)
	}
	stacks, _ := w.CarriedStacks(7)
	if len(stacks) != 1 || stacks[0].Count != 2 || *stacks[0].NativeRecord != *a.NativeRecord {
		t.Fatalf("pickup leaves %d cells, want one x2", len(stacks))
	}
	stacks = FoldItems([]ItemInstance{a, b})
	if len(stacks) != 2 || stacks[0].Count != 1 || *stacks[0].NativeRecord != *a.NativeRecord {
		t.Fatal("generic fold changed native identity without an alias census")
	}
	for _, change := range []func(*ItemInstance){
		func(i *ItemInstance) { i.NativeRecord.Token.T08++ },
		func(i *ItemInstance) { i.NativeRecord.Token.T18++ },
		func(i *ItemInstance) { i.NativeRecord.Token.Position[0]++ },
		func(i *ItemInstance) { i.NativeRecord.Token.Reference++ },
		func(i *ItemInstance) { i.NativeRecord.F48++ },
		func(i *ItemInstance) { i.Price++ },
		func(i *ItemInstance) { i.WeightPresent = true; i.Weight = 1 },
		func(i *ItemInstance) { i.Effects = []ItemEffect{{Kind: 12, Operand: 1}} },
	} {
		other := b.Clone()
		change(&other)
		if CanMergeItemValues(a, other) {
			t.Fatal("distinct retained values merge")
		}
	}
}

func TestNativePackConstructionKeepsExternalAliases(t *testing.T) {
	a, b := nativePackItem(101, 41), nativePackItem(102, 42)
	w, err := NewStockedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, []Entity{{ID: 7, HP: 10, MaxHP: 10}}, nil, Relations{}, nil, []Stock{{ID: 7, ItemInstances: []ItemInstance{a, b}, EquippedItems: [EquipSlots]ItemInstance{a}}})
	if err != nil {
		t.Fatal(err)
	}
	stacks, _ := w.CarriedStacks(7)
	if len(stacks) != 2 || stacks[0].Count != 1 || stacks[1].Count != 1 || *stacks[0].NativeRecord != *a.NativeRecord || *w.equipment[0][0].NativeRecord != *a.NativeRecord {
		t.Fatal("construction changed an externally aliased pack object")
	}
}

func TestNativeItemPickupKeepsAliasedDestinationIndependent(t *testing.T) {
	a, b := nativePackItem(101, 41), nativePackItem(102, 42)
	for _, incoming := range []ItemInstance{a, b} {
		w := packMergeWorld(t, false, false, []ItemStack{StackItem(a, 1)}, nil, incoming)
		w.equipment[1][0] = a.Clone()
		if err := w.TakeSack(7, 3, 1); err != nil {
			t.Fatal(err)
		}
		stacks, _ := w.CarriedStacks(7)
		if len(stacks) != 2 || stacks[0].Count != 1 || stacks[1].Count != 1 || *stacks[0].NativeRecord != *a.NativeRecord || *w.equipment[1][0].NativeRecord != *a.NativeRecord {
			t.Fatal("pickup changed an aliased destination")
		}
	}
}
