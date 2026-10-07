package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// An equipment record names its code's row even when its value or its token
// carries none (SAV-1089).
func TestCurrentEquipmentRecordNeverNamesDefinitionRowZero(t *testing.T) {
	for _, c := range []struct {
		name      string
		code      uint16
		class     uint8
		valueRow  uint8
		tokenRow  uint8
		wantClass string
		wantRow   uint32
	}{
		{"weapon value without a row", 0x912d, sim.SourceWeapon, 0, 0, "Weapon", 13},
		{"weapon on an unclassed token", 0x110f, sim.SourceWeapon, 15, 0, "Weapon", 15},
		{"armor value without a row", 0xb72f, sim.SourceArmor, 0, 0, "Armor", 15},
		{"shield value without a row", 0x1202, sim.SourceShield, 0, 0, "Shield", 2},
		{"retained nonzero row", 0x110f, sim.SourceWeapon, 13, 13, "Weapon", 13},
	} {
		row := sim.SavedItemObject{ID: 1, Token: sim.SavedObjectToken{Identity: 0x62000010, RuntimeID: 1, T0C: c.tokenRow},
			Value: sim.ItemStack{ObjectID: 1, Code: c.code, Count: 1, Kind: 1, WeightPresent: true, Weight: 1,
				SourceEquipment: sim.SourceEquipment{Class: c.class, DefinitionRow: c.valueRow}}}
		r, err := savedCurrentItemRecord(row, map[sim.SavedObjectID]uint16{})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got, err := savedStructureValue(&r, "T0C")
		if err != nil || r.Class != c.wantClass || got != c.wantRow {
			t.Errorf("%s: %s row %d (%v); want %s row %d", c.name, r.Class, got, err, c.wantClass, c.wantRow)
		}
	}
	item := sim.SavedItemObject{ID: 1, Token: sim.SavedObjectToken{Identity: 0x62000010, T0C: 0},
		Value: sim.ItemStack{ObjectID: 1, Code: 0x912d, Count: 1, WeightPresent: true, Weight: 1}}
	r, err := savedCurrentItemRecord(item, map[sim.SavedObjectID]uint16{})
	if got, _ := savedStructureValue(&r, "T0C"); err != nil || r.Class != "Item" || got != 0 {
		t.Errorf("unclassed Item: %s row %d (%v); want Item row 0", r.Class, got, err)
	}
}

func TestRepairEquipmentDefinitionRowsChangesOnlyRowZero(t *testing.T) {
	record := func(class string, code, row uint32) sav.DocumentRecordData {
		return sav.DocumentRecordData{Class: class, Values: []sav.DocumentValueData{{Name: "F40", Value: code}, {Name: "T0C", Value: row}}}
	}
	doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
		record("Weapon", 0x912d, 0), record("Armor", 0xb72f, 0), record("Shield", 0x1202, 0),
		record("Weapon", 0x110f, 13), record("Item", 0xe02, 0),
	}}
	if n := repairEquipmentDefinitionRows(&doc); n != 3 {
		t.Fatalf("changed %d records, want 3", n)
	}
	for i, want := range []uint32{13, 15, 2, 13, 0} {
		if got, _ := savedStructureValue(&doc.Objects[i], "T0C"); got != want {
			t.Errorf("object %d %s row %d, want %d", i+1, doc.Objects[i].Class, got, want)
		}
	}
}
