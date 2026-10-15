package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"testing"
)

func TestInstanceWeightAbsentResidueFailsCurrentItemValidation(t *testing.T) {
	bad := sim.ItemInstance{Code: 0x311, Weight: 7}
	if err := bad.ValidateWeight(); err == nil {
		t.Fatal("absent weight accepted nonzero residue")
	}
	for _, location := range []string{"carried", "worn", "carry"} {
		t.Run(location, func(t *testing.T) {
			f, _ := city1095Front(t)
			s, _, err := f.Snapshot(false)
			if err != nil {
				t.Fatal(err)
			}
			p := &s.Party[0]
			p.OriginalHuman = nil
			switch location {
			case "carried":
				p.Carry = nil
				p.CarriedItems = []sim.ItemInstance{bad}
			case "worn":
				p.Carry = nil
				p.WornItems = [sim.EquipSlots]sim.ItemInstance{bad}
			case "carry":
				p.Carry = &mapload.Carry{ItemInstances: []sim.ItemInstance{bad}}
			}
			var selected sim.ItemInstance
			if location == "worn" {
				selected = mapload.MemberItemEquipment(*p, nil)[0]
			} else {
				items := mapload.MemberCarriedItems(*p, nil)
				if len(items) != 1 {
					t.Fatal("current holding selector lost malformed operand", location)
				}
				selected = items[0]
			}
			if selected.Code != bad.Code || selected.WeightPresent || selected.Weight != 7 {
				t.Fatal("current holding selector changed malformed operand", location)
			}
			if err := selected.ValidateWeight(); err == nil {
				t.Fatal("current item validation accepted absent weight residue", location)
			}
		})
	}
}
