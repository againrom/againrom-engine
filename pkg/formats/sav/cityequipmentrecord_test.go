package sav

import (
	"fmt"
	"strings"
	"testing"
)

// TestCityEquipmentRecordSlotPositions records where the city document's
// worn-set writer puts each of the twelve slots on a Human: slot by slot, the
// Unit reference it fills, and the frozen references it leaves.
func TestCityEquipmentRecordSlotPositions(t *testing.T) {
	g := &CityEquipmentGraph{}
	for slot := range g.Slots {
		class := "Armor"
		switch slot {
		case 0:
			class = "Weapon"
		case 1:
			class = "Shield"
		}
		r, err := NewDocumentRecord(class)
		if err != nil {
			t.Fatal(err)
		}
		g.Objects = append(g.Objects, r)
		g.Slots[slot] = uint16(slot + 1)
	}
	frozen := func(n int) *cityObject { return &cityObject{class: fmt.Sprintf("frozen%d", n)} }
	u := &cityUnit{reference74: frozen(74), reference78: frozen(78)}
	for at := 0; at < 13; at++ {
		u.equipment = append(u.equipment, frozen(at))
	}
	d := &cityDocument{objects: map[uint16]*cityObject{}}
	if _, err := applyCityEquipment(d, &cityObject{class: "Human", unit: u}, g); err != nil {
		t.Fatal(err)
	}
	name := func(o *cityObject) string {
		if o == nil {
			return "nil"
		}
		if o.item != nil {
			return fmt.Sprintf("%s#%d", o.class, o.sourceIndex)
		}
		return o.class
	}
	var got strings.Builder
	fmt.Fprintf(&got, "reference74=%s reference78=%s\n", name(u.reference74), name(u.reference78))
	for at, o := range u.equipment {
		fmt.Fprintf(&got, "equipment %d=%s\n", at, name(o))
	}
	if got.String() != cityEquipmentRecord {
		t.Fatalf("city worn-set positions changed:\n%s", got.String())
	}
	if _, err := applyCityEquipment(d, &cityObject{class: "Unit", unit: &cityUnit{}}, g); err == nil {
		t.Fatal("a Unit without worn positions took a worn set")
	}
}

const cityEquipmentRecord = `reference74=Weapon#1 reference78=Shield#2
equipment 0=frozen0
equipment 1=frozen1
equipment 2=Armor#3
equipment 3=Armor#4
equipment 4=Armor#5
equipment 5=Armor#6
equipment 6=Armor#7
equipment 7=Armor#8
equipment 8=Armor#9
equipment 9=Armor#10
equipment 10=Armor#11
equipment 11=Armor#12
equipment 12=frozen12
`
