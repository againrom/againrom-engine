package sim

import (
	"bytes"
	"reflect"
	"testing"
)

const legacyLinkedUnderflow = uint32(0x0000fb05)

func legacyRepairWorld(t *testing.T) *World {
	t.Helper()
	carried := ItemInstance{Code: 0x1111, Kind: 1, Price: 1, Effects: []ItemEffect{
		{Kind: 12, Operand: legacyLinkedUnderflow},
		{Kind: 44, Operand: legacyLinkedUnderflow},
	}}
	correctCarried := carried.Clone()
	correctCarried.Price = 101
	correctCarried.Effects[0].Operand = 5
	equipped := ItemInstance{Code: 0x2222, Kind: 1, Price: 2, Effects: []ItemEffect{
		{Kind: 21, Operand: legacyLinkedUnderflow},
		{Kind: 41, Operand: legacyLinkedUnderflow},
	}}
	ground := ItemInstance{Code: 0x3333, Kind: 1, Price: 3, Effects: []ItemEffect{
		{Kind: 49, Operand: legacyLinkedUnderflow},
		{Kind: 42, Operand: legacyLinkedUnderflow},
	}}
	negative := ItemInstance{Code: 0x4444, Kind: 1, Price: 77, Effects: []ItemEffect{
		{Kind: 44, Operand: legacyLinkedUnderflow},
		{Kind: 41, Operand: legacyLinkedUnderflow},
		{Kind: 42, Operand: legacyLinkedUnderflow},
		{Kind: 12, Mode: 8, Operand: legacyLinkedUnderflow},
		{Kind: 12, Operand: 0x0001fb05},
		{Kind: 12, Operand: 0x0000fa05},
		{Kind: 12, Operand: 5},
		{Kind: 0, Operand: legacyLinkedUnderflow},
	}}
	var worn [EquipSlots]ItemInstance
	worn[2] = equipped
	w, err := NewStockedWorld(91, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{},
		[]Entity{{ID: 7, X: 2, Y: 3}}, nil, Relations{},
		[]Sack{{X: 4, Y: 5, ItemInstances: []ItemInstance{ground}}},
		[]Stock{{ID: 7, ItemInstances: []ItemInstance{carried, carried, correctCarried, negative}, EquippedItems: worn}})
	if err != nil {
		t.Fatalf("NewStockedWorld: %v", err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(raw); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	return &back
}

func TestLegacyLinkedItemRepairCoversEveryWorldContainerAndIsIdempotent(t *testing.T) {
	w := legacyRepairWorld(t)
	beforeInspect, _ := w.MarshalBinary()
	candidates := InspectLegacyLinkedItems(w)
	afterInspect, _ := w.MarshalBinary()
	if !bytes.Equal(beforeInspect, afterInspect) {
		t.Fatal("InspectLegacyLinkedItems mutated the world")
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d, want 3: %+v", len(candidates), candidates)
	}
	wantPopulations := []LegacyLinkedItemPopulation{
		LegacyLinkedItemCarried, LegacyLinkedItemEquipped, LegacyLinkedItemSack,
	}
	wantKinds := []uint8{12, 21, 49}
	for i, candidate := range candidates {
		if candidate.Population != wantPopulations[i] || candidate.Kind != wantKinds[i] ||
			candidate.Before != legacyLinkedUnderflow || candidate.After != 5 {
			t.Errorf("candidate %d = %+v", i, candidate)
		}
	}
	if candidates[0].Entity != 7 || candidates[0].Count != 2 ||
		candidates[1].Entity != 7 || candidates[1].Position != 2 ||
		candidates[2].SackX != 4 || candidates[2].SackY != 5 {
		t.Fatalf("candidate locations = %+v", candidates)
	}

	calls := 0
	repairs := RepairLegacyLinkedItems(w, func(item ItemInstance) int32 {
		calls++
		switch item.Code {
		case 0x1111:
			if got := item.Effects; !reflect.DeepEqual(got, []ItemEffect{
				{Kind: 12, Operand: 5}, {Kind: 44, Operand: legacyLinkedUnderflow},
			}) {
				t.Errorf("carried repricing effects = %+v", got)
			}
			return 101
		case 0x2222:
			if got := item.Effects; !reflect.DeepEqual(got, []ItemEffect{
				{Kind: 21, Operand: 5}, {Kind: 41, Operand: legacyLinkedUnderflow},
			}) {
				t.Errorf("equipment repricing effects = %+v", got)
			}
			return 202
		case 0x3333:
			if got := item.Effects; !reflect.DeepEqual(got, []ItemEffect{
				{Kind: 49, Operand: 5}, {Kind: 42, Operand: legacyLinkedUnderflow},
			}) {
				t.Errorf("sack repricing effects = %+v", got)
			}
			return 303
		default:
			t.Errorf("repriced untouched item %#04x", item.Code)
			return item.Price
		}
	})
	if len(repairs) != 3 || calls != 3 {
		t.Fatalf("repairs/callbacks = %d/%d, want 3/3", len(repairs), calls)
	}

	stacks, _ := w.CarriedStacks(7)
	if len(stacks) != 2 || stacks[0].Code != 0x1111 || stacks[0].Count != 3 ||
		stacks[0].Price != 101 || stacks[0].Effects[0].Operand != 5 {
		t.Fatalf("repaired carried stacks = %+v", stacks)
	}
	negative := stacks[1].Instance()
	wantNegative := legacyRepairWorld(t)
	wantStacks, _ := wantNegative.CarriedStacks(7)
	if !reflect.DeepEqual(negative, wantStacks[2].Instance()) {
		t.Fatalf("negative cases changed: %+v", negative)
	}
	worn, _ := w.EquippedItems(7)
	if worn[2].Price != 202 || worn[2].Effects[0].Operand != 5 ||
		worn[2].Effects[1].Operand != legacyLinkedUnderflow {
		t.Fatalf("repaired equipment = %+v", worn[2])
	}
	sacks := w.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 ||
		sacks[0].ItemInstances[0].Price != 303 || sacks[0].ItemInstances[0].Effects[0].Operand != 5 ||
		sacks[0].ItemInstances[0].Effects[1].Operand != legacyLinkedUnderflow {
		t.Fatalf("repaired sacks = %+v", sacks)
	}

	once, _ := w.MarshalBinary()
	calls = 0
	if second := RepairLegacyLinkedItems(w, func(ItemInstance) int32 { calls++; return -1 }); len(second) != 0 {
		t.Fatalf("second repair = %+v, want none", second)
	}
	twice, _ := w.MarshalBinary()
	if calls != 0 || !bytes.Equal(once, twice) {
		t.Fatalf("second repair called repricer %d time(s) or changed the world", calls)
	}
}
