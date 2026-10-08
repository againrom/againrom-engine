package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func nativeItemRecord(v sim.SavedItemObject) sim.NativeItemRecord {
	token := v.Token
	token.T1C = 0 // Price has its own current item field.
	return sim.NativeItemRecord{Class: v.Value.SourceEquipment.Class, Token: token, F45: v.F45, F46: v.F46, F47: v.F47, F48: v.F48}
}

type nativeItemEmission struct {
	value sim.ItemStack
	index uint16
}

func nativeItemRecordAnchor(v sim.NativeItemRecord) [32]byte {
	b, _ := binary.Append(nil, binary.LittleEndian, v)
	return sha256.Sum256(b)
}

func nativeItemPlayerReference(doc *sav.DocumentData, reference uint32) uint32 {
	if reference == 0 {
		return 0
	}
	var object uint16
	for i, r := range doc.Objects {
		for _, v := range r.Values {
			if (v.Name == "Identity" || v.Name == "This") && v.Value == reference {
				if object != 0 {
					return 0
				}
				object = uint16(i + 1)
				break
			}
		}
	}
	if object == 0 || doc.Objects[object-1].Class != "Player" {
		return 0
	}
	var ordinal uint32
	for i, root := range doc.Players {
		if root == object {
			if ordinal != 0 {
				return 0
			}
			ordinal = uint32(i + 1)
		}
	}
	return ordinal
}

func nativeItemAbsenceAnchor(doc *sav.DocumentData, v sim.NativeItemRecord, version uint8) [32]byte {
	if version == 0 {
		return nativeItemRecordAnchor(v)
	}
	b := []byte{1, 0}
	if ordinal := nativeItemPlayerReference(doc, v.Token.Reference); ordinal != 0 {
		v.Token.Reference = 0
		b[1] = 1
		b = binary.LittleEndian.AppendUint32(b, ordinal)
	}
	b, _ = binary.Append(b, binary.LittleEndian, v)
	return sha256.Sum256(b)
}

func captureNativeItemRecordAnchor(doc *sav.DocumentData, row *currentOwnedObject) error {
	row.NativeRecordAnchor = nil
	row.NativeRecordAnchorVersion = 0
	if row.Kind != 1 || row.ID != 0 || row.Object == 0 || row.NativeRecordKnown {
		return nil
	}
	v, err := savedItemRecord(doc, row.Object)
	if err != nil {
		return err
	}
	row.NativeRecordAnchorVersion = 1
	anchor := nativeItemAbsenceAnchor(doc, nativeItemRecord(v), row.NativeRecordAnchorVersion)
	row.NativeRecordAnchor = &anchor
	return nil
}

func validateNativeItemRecordPolicy(row currentOwnedObject) error {
	if row.NativeRecordAnchorVersion > 1 || row.NativeRecordAnchorVersion != 0 && row.NativeRecordAnchor == nil {
		return fmt.Errorf("current native Item record anchor version is unsupported or unbound")
	}
	if (row.NativeRecordKnown || row.NativeRecordAnchor != nil) && (row.Kind != 1 || row.ID != 0 || row.Object == 0) {
		return fmt.Errorf("current native Item record has no unregistered ordinary owner")
	}
	if row.NativeRecordKnown && row.NativeRecordAnchor != nil || row.NativeRecordAnchor != nil && *row.NativeRecordAnchor == ([32]byte{}) {
		return fmt.Errorf("current native Item record availability conflicts")
	}
	return nil
}
