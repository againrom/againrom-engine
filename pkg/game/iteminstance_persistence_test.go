package game

import (
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestPartyGobRoundTripPreservesCompleteItemInstances(t *testing.T) {
	carried := sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: 250,
		Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	worn := sim.ItemInstance{Code: 0x914e, Kind: 2, Price: 812,
		Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(13) | uint32(uint16(15))<<16}}}
	member := mapload.PartyMember{ID: "item-gob", Carry: &mapload.Carry{
		Items: []uint16{carried.Code}, ItemInstances: []sim.ItemInstance{carried},
		Equipped:      [sim.EquipSlots]uint16{worn.Code},
		EquippedItems: [sim.EquipSlots]sim.ItemInstance{worn},
	}}
	raw, err := EncodeSave(Snapshot{Party: []mapload.PartyMember{member}}, "item-gob")
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}
	back, label, err := DecodeSave(raw)
	if err != nil {
		t.Fatalf("DecodeSave: %v", err)
	}
	if label != "item-gob" || len(back.Party) != 1 || back.Party[0].Carry == nil {
		t.Fatalf("decoded party/label = %+v/%q", back.Party, label)
	}
	got := back.Party[0].Carry
	if !reflect.DeepEqual(got.ItemInstances, []sim.ItemInstance{carried}) ||
		!reflect.DeepEqual(got.EquippedItems[0], worn) ||
		!reflect.DeepEqual(got.Items, []uint16{carried.Code}) || got.Equipped[0] != worn.Code {
		t.Fatalf("party gob item state = %+v, want carried %+v worn %+v", got, carried, worn)
	}
}
