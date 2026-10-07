package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestCurrentItemAbsenceRequiresMatchingOrdinaryFields(t *testing.T) {
	f := itemObjectsOpen(t, false)
	doc, _ := currentRootSAVDocument(t, f)
	var object uint16
	for i, record := range doc.Objects {
		if record.Class == "Weapon" {
			object = uint16(i + 1)
			break
		}
	}
	if object == 0 {
		t.Fatal("fixture has no ordinary Weapon")
	}
	policy := currentOwnedObject{Object: object, Kind: 1}
	if err := captureCurrentItemAbsence(&doc, &policy); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"unchanged", "weight", "equipment", "legacy"} {
		t.Run(field, func(t *testing.T) {
			row := policy
			r := &doc.Objects[object-1]
			weight, _ := savedStructureValue(r, "F4A")
			raw, err := savedObjectRaw(r, "W52", 24)
			if err != nil {
				t.Fatal(err)
			}
			attack := raw[0]
			defer func() { savedObjectSetValue(r, "F4A", weight); raw[0] = attack }()
			switch field {
			case "weight":
				savedObjectSetValue(r, "F4A", 123)
			case "equipment":
				raw[0] = 73
			case "legacy":
				row.WeightAnchor, row.EquipmentAnchor = nil, nil
			}
			got, err := readCurrentItem(&doc, row, f.Table)
			if err != nil {
				t.Fatal(err)
			}
			wantWeight := field == "weight" || field == "legacy"
			wantEquipment := field == "equipment" || field == "legacy"
			if got.Value.WeightPresent != wantWeight || (got.Value.SourceEquipment.Class != 0) != wantEquipment {
				t.Fatal("Item field authority did not follow its own anchor", got.Value)
			}
			if field == "weight" && got.Value.Weight != 123 || field == "equipment" && got.Value.SourceEquipment.Attack[0] != 73 {
				t.Fatal("ordinary Item edit was lost", got.Value)
			}
			if !wantEquipment && got.Value.SourceEquipment != (sim.SourceEquipment{}) {
				t.Fatal("absent equipment retained partial operands")
			}
		})
	}
	for _, kind := range []string{"known", "wrong class", "detached", "empty digest"} {
		bad := policy
		switch kind {
		case "known":
			bad.WeightKnown = true
		case "wrong class":
			bad.Kind = 4
		case "detached":
			bad.Object = 0
		case "empty digest":
			bad.WeightAnchor = &[32]byte{}
		}
		if err := validateCurrentItemAbsence(bad); err == nil {
			t.Fatal("conflicting absence policy accepted", kind)
		}
	}
}
