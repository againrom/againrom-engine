package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

// A held Item whose loaded owner key names no written object takes its
// holder's current Player key; a zero key and a resolving key are kept.
func TestCityHeldItemStaleOwnerTakesTheHolderPlayer(t *testing.T) {
	record := func(class string, values ...sav.DocumentValueData) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: class, Values: values}
	}
	value := func(name string, v uint32) sav.DocumentValueData { return sav.DocumentValueData{Name: name, Value: v} }
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
		record("Player", value("This", 0x1000001)),
		record("Human", value("Identity", 0x1000002), value("Reference", 0x1000001)),
		record("Armor", value("Identity", 0x1000003), value("Reference", 0x2c1ed70)),
		record("Armor", value("Identity", 0x1000004), value("Reference", 0)),
		record("Weapon", value("Identity", 0x1000005), value("Reference", 0x1000002)),
		record("Armor", value("Identity", 0x1000006), value("Reference", 0x2c1ed70)),
	}}
	rebindCityItemOwners(&doc, 2, []uint16{3, 0, 4, 5})
	for index, want := range map[int]uint32{3: 0x1000001, 4: 0, 5: 0x1000002, 6: 0x2c1ed70} {
		if got, _ := savedStructureValue(&doc.Objects[index-1], "Reference"); got != want {
			t.Errorf("object %d Reference %#x, want %#x", index, got, want)
		}
	}
}
