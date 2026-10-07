package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
)

// equipmentDefinitionRow keeps a nonzero row and derives an absent one from
// the code (ITEM-CODE-029, ITEM-NAMEKEY-037). The original terminates on LOAD
// of a row-0 Weapon, Armor or Shield (SAV-1089, SAV-1091).
func equipmentDefinitionRow(code uint16, row uint8) uint8 {
	if row != 0 {
		return row
	}
	return uint8(data.ItemCode(code).D())
}

func equipmentRecordClass(class string) bool {
	return class == "Weapon" || class == "Armor" || class == "Shield"
}

// repairEquipmentDefinitionRows derives every row-0 equipment record's row.
func repairEquipmentDefinitionRows(doc *sav.DocumentData) int {
	changed := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !equipmentRecordClass(r.Class) {
			continue
		}
		code, err := savedStructureValue(r, "F40")
		if err != nil {
			continue
		}
		for j := range r.Values {
			if r.Values[j].Name == "T0C" && r.Values[j].Value == 0 {
				r.Values[j].Value = uint32(equipmentDefinitionRow(uint16(code), 0))
				changed++
			}
		}
	}
	return changed
}

// repairLoadedEquipmentRows re-encodes only a document holding a row-0
// record, so every reader of the loaded bytes binds the derived row.
func repairLoadedEquipmentRows(saved []byte) []byte {
	doc, err := sav.DecodeDocumentData(saved)
	if err != nil || repairEquipmentDefinitionRows(&doc) == 0 {
		return saved
	}
	repaired, err := sav.EncodeDocumentData(doc)
	if err != nil {
		return saved
	}
	return repaired
}
