package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"testing"
)

func TestInstanceWeightAbsentResidueRefusesBeforeNativeRestore(t *testing.T) {
	bad := sim.ItemInstance{Code: 0x311, Weight: 7}
	for _, p := range []mapload.PartyMember{
		{ID: "carried", CarriedItems: []sim.ItemInstance{bad}},
		{ID: "worn", WornItems: [sim.EquipSlots]sim.ItemInstance{bad}},
		{ID: "carry", Carry: &mapload.Carry{ItemInstances: []sim.ItemInstance{bad}}},
	} {
		s := Snapshot{Party: []mapload.PartyMember{p}}
		if _, e := EncodeSave(s, "bad weight"); e == nil {
			t.Fatal("writer accepted residue", p.ID)
		}
		f := &FrontEnd{}
		if _, e := f.prepareRestore(s); e == nil {
			t.Fatal("restore accepted residue", p.ID)
		}
	}
}
