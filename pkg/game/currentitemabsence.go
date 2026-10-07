package game

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentItemAbsenceAnchor(value sim.ItemStack, equipment bool) [32]byte {
	raw := binary.LittleEndian.AppendUint16(nil, value.Code)
	if equipment {
		b, _ := json.Marshal(value.SourceEquipment)
		raw = append(raw, b...)
	} else {
		raw = binary.LittleEndian.AppendUint16(raw, uint16(value.Weight))
	}
	return sha256.Sum256(raw)
}

func currentItemAnchorMatches(anchor *[32]byte, value sim.ItemStack, equipment bool) bool {
	return anchor != nil && *anchor == currentItemAbsenceAnchor(value, equipment)
}

func captureCurrentItemAbsence(doc *sav.DocumentData, row *currentOwnedObject) error {
	row.WeightAnchor, row.EquipmentAnchor = nil, nil
	if row.Kind != 1 || row.Object == 0 || row.WeightKnown && row.EquipmentKnown {
		return nil
	}
	v, err := savedItemRecord(doc, row.Object)
	if err != nil {
		return err
	}
	if !row.WeightKnown {
		anchor := currentItemAbsenceAnchor(v.Value, false)
		row.WeightAnchor = &anchor
	}
	if !row.EquipmentKnown {
		anchor := currentItemAbsenceAnchor(v.Value, true)
		row.EquipmentAnchor = &anchor
	}
	return nil
}

func validateCurrentItemAbsence(row currentOwnedObject) error {
	for _, field := range []struct {
		anchor *[32]byte
		known  bool
	}{{row.WeightAnchor, row.WeightKnown}, {row.EquipmentAnchor, row.EquipmentKnown}} {
		if field.anchor != nil && (field.known || row.Kind != 1 || row.Object == 0 || *field.anchor == ([32]byte{})) {
			return fmt.Errorf("current Item absence has a conflicting ordinary anchor")
		}
	}
	return nil
}
