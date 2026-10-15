package mapload_test

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestExplicitPartyPackRetainsNullOrderCountsAndNativeAbsence(t *testing.T) {
	first := sim.ItemInstance{Code: 0x101, ObjectID: 42, Price: 8, Effects: []sim.ItemEffect{{Kind: 44, Operand: 3}}}
	second := first.Clone()
	second.ObjectID = 0
	member := mapload.PartyMember{ID: "hero", Carry: &mapload.Carry{}}
	want := []sim.ItemStack{sim.StackItem(first, 2), {}, sim.StackItem(second, 1)}
	if err := mapload.UpdatePartyLoadOrdered(member, &member, nil, true, false, want); err != nil {
		t.Fatal(err)
	}
	if err := mapload.ValidatePartyLoad(member); err != nil {
		t.Fatal(err)
	}
	if member.Carry.LiveLoad != nil || !reflect.DeepEqual(member.Carry.OrderedStacks, want) || len(member.Carry.ItemInstances) != 3 || member.Carry.ItemInstances[2].ObjectID != 0 {
		t.Fatal("explicit current order synthesized load, folded nodes or lost units")
	}
	member.Carry.OrderedStacks[0].Effects[0].Operand++
	if want[0].Effects[0].Operand != 3 || first.Effects[0].Operand != 3 {
		t.Fatal("ordered candidate aliases source item children")
	}
}

func TestExplicitPartyPackRejectsMalformedCountsWithoutPrefix(t *testing.T) {
	for _, bad := range []sim.ItemStack{
		{Code: 0x101}, {Count: 1}, {Price: 7}, {Code: 0x101, Count: ^uint32(0)},
	} {
		member := mapload.PartyMember{ID: "hero", Carry: &mapload.Carry{Items: []uint16{0x101}, ItemInstances: []sim.ItemInstance{{Code: 0x101}}}}
		before := mapload.CloneParty([]mapload.PartyMember{member})[0]
		if err := mapload.UpdatePartyLoadOrdered(before, &member, nil, true, false, []sim.ItemStack{{Code: 0x102, Count: 1}, bad}); err == nil {
			t.Fatal("malformed explicit container accepted", bad)
		}
		if !reflect.DeepEqual(member, before) {
			t.Fatal("invalid tail committed a pack prefix")
		}
	}
}

func TestValuePackMutationReconcilesOrderWithoutRetainedLoad(t *testing.T) {
	first, second, added := sim.PlainItem(0x101), sim.PlainItem(0x102), sim.PlainItem(0x103)
	before := mapload.PartyMember{ID: "hero", Carried: []uint16{0x999}, Carry: &mapload.Carry{}}
	if err := mapload.UpdatePartyLoadOrdered(before, &before, nil, false, false, []sim.ItemStack{sim.StackItem(first, 2), sim.StackItem(second, 1)}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		items []sim.ItemInstance
		want  []sim.ItemStack
	}{
		{"consume", []sim.ItemInstance{first, second}, []sim.ItemStack{sim.StackItem(first, 1), sim.StackItem(second, 1)}},
		{"append", []sim.ItemInstance{first, second, added}, []sim.ItemStack{sim.StackItem(first, 1), sim.StackItem(second, 1), sim.StackItem(added, 1)}},
		{"empty", nil, []sim.ItemStack{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next := mapload.CloneParty([]mapload.PartyMember{before})[0]
			next.Carry.ItemInstances, next.Carry.Items = tc.items, nil
			for _, item := range tc.items {
				next.Carry.Items = append(next.Carry.Items, item.Code)
			}
			if err := mapload.UpdatePartyLoad(before, &next, nil, false, false); err != nil {
				t.Fatal(err)
			}
			if next.Carry.LiveLoad != nil || !reflect.DeepEqual(next.Carry.OrderedStacks, tc.want) || len(mapload.MemberCarriedItems(next, nil)) != len(tc.items) {
				t.Fatalf("ordered=%+v load=%+v flat=%+v", next.Carry.OrderedStacks, next.Carry.LiveLoad, mapload.MemberCarriedItems(next, nil))
			}
		})
	}
}
