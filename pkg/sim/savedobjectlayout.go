package sim

import "encoding/binary"

// Field-by-field forms of the fixed saved-object values the suffix writes and
// reads on every world encode, hash and decode. Each produces exactly the
// little-endian bytes encoding/binary produces for its type; any other value
// still goes through encoding/binary. savedobjectlayout_test.go holds them to
// encoding/binary over every field.

func appendSavedObjectLayout(dst []byte, value any) ([]byte, bool) {
	le := binary.LittleEndian
	switch v := value.(type) {
	case bool:
		return append(dst, layoutBool(v)), true
	case SavedObjectOrigin:
		return le.AppendUint64(append(dst, byte(v.Kind)), uint64(v.Parent)), true
	case SavedObjectToken:
		dst = append(dst, v.Position[:]...)
		dst = le.AppendUint32(dst, v.RuntimeID)
		dst = le.AppendUint16(append(dst, v.T0C), v.T0E)
		dst = le.AppendUint32(dst, v.T08)
		dst = le.AppendUint16(dst, v.T18)
		dst = le.AppendUint32(dst, v.T1C)
		dst = le.AppendUint32(dst, v.Identity)
		return le.AppendUint32(dst, v.Reference), true
	case SavedObjectOwner:
		dst = le.AppendUint32(append(dst, byte(v.Kind)), uint32(v.Entity))
		dst = le.AppendUint64(dst, uint64(v.Object))
		dst = le.AppendUint32(dst, v.Slot)
		return le.AppendUint64(dst, v.SessionHandle), true
	case ItemEffect:
		return le.AppendUint32(append(dst, v.Kind, v.Mode), v.Operand), true
	case SourceItemSpell:
		return le.AppendUint16(append(dst, layoutBool(v.Present), v.ID, v.Range, v.Defensive), v.ManaCost), true
	case SourceEquipment:
		n := len(dst)
		dst = append(dst, make([]byte, sourceEquipmentLen)...)
		putSourceEquipment(dst[n:], &v)
		return dst, true
	case SavedBookRoot:
		dst = le.AppendUint32(dst, uint32(v.Entity))
		for _, s := range v.Slots {
			dst = le.AppendUint64(dst, uint64(s))
		}
		return dst, true
	}
	return dst, false
}

// readSavedObjectLayout fills dst from the reader when dst is one of the
// layouts above or a fixed-width integer, reporting false for any other type.
func (r *savedObjectReader) readSavedObjectLayout(dst any) bool {
	le := binary.LittleEndian
	switch v := dst.(type) {
	case *uint8:
		if b := r.take(1); b != nil {
			*v = b[0]
		}
	case *uint16:
		if b := r.take(2); b != nil {
			*v = le.Uint16(b)
		}
	case *uint32:
		if b := r.take(4); b != nil {
			*v = le.Uint32(b)
		}
	case *uint64:
		if b := r.take(8); b != nil {
			*v = le.Uint64(b)
		}
	case *SavedObjectOrigin:
		if b := r.take(9); b != nil {
			*v = SavedObjectOrigin{Kind: SavedObjectOriginKind(b[0]), Parent: SavedObjectID(le.Uint64(b[1:]))}
		}
	case *SavedObjectToken:
		if b := r.take(37); b != nil {
			*v = SavedObjectToken{RuntimeID: le.Uint32(b[12:]), T0C: b[16], T0E: le.Uint16(b[17:]), T08: le.Uint32(b[19:]),
				T18: le.Uint16(b[23:]), T1C: le.Uint32(b[25:]), Identity: le.Uint32(b[29:]), Reference: le.Uint32(b[33:])}
			copy(v.Position[:], b[:12])
		}
	case *SavedObjectOwner:
		if b := r.take(25); b != nil {
			*v = SavedObjectOwner{Kind: SavedOwnerKind(b[0]), Entity: EntityID(le.Uint32(b[1:])), Object: SavedObjectID(le.Uint64(b[5:])),
				Slot: le.Uint32(b[13:]), SessionHandle: le.Uint64(b[17:])}
		}
	case *ItemEffect:
		if b := r.take(6); b != nil {
			*v = ItemEffect{Kind: b[0], Mode: b[1], Operand: le.Uint32(b[2:])}
		}
	case *SourceItemSpell:
		if b := r.take(6); b != nil {
			*v = SourceItemSpell{Present: b[0] != 0, ID: b[1], Range: b[2], Defensive: b[3], ManaCost: le.Uint16(b[4:])}
		}
	case *SourceEquipment:
		if b := r.take(sourceEquipmentLen); b != nil {
			getSourceEquipment(b, v)
		}
	case *SavedBookRoot:
		if b := r.take(228); b != nil {
			v.Entity = EntityID(le.Uint32(b))
			for i := range v.Slots {
				v.Slots[i] = SavedObjectID(le.Uint64(b[4+8*i:]))
			}
		}
	default:
		return false
	}
	return true
}
