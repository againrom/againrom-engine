package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestPartyClonePreservesCompleteItemInstances(t *testing.T) {
	carried := sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: 250,
		Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	worn := sim.ItemInstance{Code: 0x914e, Kind: 2, Price: 812,
		Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(13) | uint32(uint16(15))<<16}}}
	member := mapload.PartyMember{ID: "item-gob", Carry: &mapload.Carry{
		Items: []uint16{carried.Code}, ItemInstances: []sim.ItemInstance{carried},
		Equipped:      [sim.EquipSlots]uint16{worn.Code},
		EquippedItems: [sim.EquipSlots]sim.ItemInstance{worn},
	}}
	back := mapload.CloneParty([]mapload.PartyMember{member})
	if len(back) != 1 || back[0].Carry == nil {
		t.Fatalf("cloned party = %+v", back)
	}
	got := back[0].Carry
	if !reflect.DeepEqual(got.ItemInstances, []sim.ItemInstance{carried}) ||
		!reflect.DeepEqual(got.EquippedItems[0], worn) ||
		!reflect.DeepEqual(got.Items, []uint16{carried.Code}) || got.Equipped[0] != worn.Code {
		t.Fatalf("party clone item state = %+v, want carried %+v worn %+v", got, carried, worn)
	}
}
