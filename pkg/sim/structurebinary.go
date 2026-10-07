package sim

import (
	"encoding/binary"
	"fmt"
)

// The STRUCTURE SECTION (version 58, 1033 B3): one Structure per placed
// type-4 record on the map the world was built from.
//
// It sits between the script-state section and the script section, on the
// script-state section's own reason: decodeScript consumes the rest of the
// buffer and returns no used count, so a section behind it would have to give
// it one.
//
// One structure record, 22 bytes:
//
//	+0    4      id, uint32
//	+4    2      Field42, uint16
//	+6    2      maximum health, uint16
//	+8    4      footprint column, int32
//	+12   4      footprint row, int32
//	+16   1      footprint width, uint8
//	+17   1      footprint height, uint8
//	+18   4      footprint attach mask, uint32
//
// No presence byte and no reference into any other section: a structure is
// not resolved against an entity, a group or a player, so there is nothing
// here for one to disagree with.
const structureCountLen = 4
const structureRecordLen = 22

// structureSectionLen is how many bytes w's structure list occupies.
func (w *World) structureSectionLen() int {
	return structureCountLen + structureRecordLen*len(w.structures)
}

// WithoutStructureAndItemStateSections returns the form-57 whole-section
// shape used by migration fixtures. Version 61 places item state immediately
// after structures, so both later sections must be removed before a caller
// labels the bytes as an older form.
func (w *World) WithoutStructureAndItemStateSections() ([]byte, error) {
	full, err := w.MarshalBinary()
	if err != nil {
		return nil, err
	}
	// Migration fixture output predates imported dead records entirely.
	suffixLen := len(full) - w.binaryBodyLen()
	end := len(full) - relationLen - suffixLen
	full = append(full[:end-w.originalDeadSectionLen()-w.instanceWeightSectionLen()-w.actorLoadSectionLen()], full[end:]...)
	tailLen := w.scriptSectionLen() + relationLen + suffixLen
	removed := w.structureSectionLen() + w.itemStateSectionLen() + w.scrollSectionLen()
	head := full[:len(full)-removed-tailLen]
	tail := full[len(full)-tailLen:]
	out := make([]byte, 0, len(head)+len(tail))
	out = append(out, head...)
	out = append(out, tail...)
	return out, nil
}

// encodeStructures writes the section into b at off and returns the offset
// past it.
//
// THE STRUCTURES ARE WRITTEN IN ASCENDING ID ORDER, which newWorld's own sort
// already leaves them in: nothing is sorted here, on the entity section's own
// reason — an order the constructor already produced is the canonical one.
func (w *World) encodeStructures(b []byte, off int) int {
	binary.LittleEndian.PutUint32(b[off:off+4], uint32(len(w.structures)))
	off += structureCountLen
	for _, st := range w.structures {
		binary.LittleEndian.PutUint32(b[off:off+4], uint32(st.ID))
		binary.LittleEndian.PutUint16(b[off+4:off+6], st.Field42)
		binary.LittleEndian.PutUint16(b[off+6:off+8], st.MaxHealth)
		binary.LittleEndian.PutUint32(b[off+8:off+12], uint32(st.Col))
		binary.LittleEndian.PutUint32(b[off+12:off+16], uint32(st.Row))
		b[off+16], b[off+17] = st.Width, st.Height
		binary.LittleEndian.PutUint32(b[off+18:off+22], st.Attach)
		off += structureRecordLen
	}
	return off
}

// decodeStructures reads the section and returns the structures and how many
// bytes it consumed.
//
// THE DECLARED COUNT IS BOUNDED AGAINST THE BUFFER BEFORE A SINGLE RECORD IS
// ALLOCATED, on decodeGroups' and decodeCasting's own ground: a declared
// count cannot ask for memory the form does not carry the bytes for.
//
// NON-ASCENDING OR DUPLICATE IDS ARE REFUSED, the entity section's own rule:
// the order is canonical, so a form carrying either encodes differently from
// the world it decodes to and the digest stops being a function of the
// logical world alone.
func decodeStructures(data []byte, savedMode ...bool) ([]Structure, int, error) {
	saved := len(savedMode) != 0 && savedMode[0]
	if len(data) < structureCountLen {
		return nil, 0, fmt.Errorf(
			"sim: byte form truncated: the structure count needs %d byte(s), %d left",
			structureCountLen, len(data))
	}
	n := binary.LittleEndian.Uint32(data[:structureCountLen])
	off := structureCountLen
	span := int64(n) * structureRecordLen
	if avail := int64(len(data) - off); span > avail {
		return nil, 0, fmt.Errorf(
			"sim: byte form declares %d structure(s), whose records are %d byte(s), and carries %d byte(s) after the count",
			n, span, avail)
	}
	out := make([]Structure, n)
	for i := range out {
		o := off + structureRecordLen*i
		out[i] = Structure{
			ID:        StructureID(binary.LittleEndian.Uint32(data[o : o+4])),
			Field42:   binary.LittleEndian.Uint16(data[o+4 : o+6]),
			MaxHealth: binary.LittleEndian.Uint16(data[o+6 : o+8]),
			Col:       int32(binary.LittleEndian.Uint32(data[o+8 : o+12])),
			Row:       int32(binary.LittleEndian.Uint32(data[o+12 : o+16])),
			Width:     data[o+16],
			Height:    data[o+17],
			Attach:    binary.LittleEndian.Uint32(data[o+18 : o+22]),
		}
		out[i].Blocking = out[i].Attach
		if !saved && (out[i].Width == 0) != (out[i].Height == 0) {
			return nil, 0, fmt.Errorf(
				"sim: structure %d has incomplete footprint %dx%d",
				i, out[i].Width, out[i].Height)
		}
		if i > 0 && out[i].ID <= out[i-1].ID {
			return nil, 0, fmt.Errorf(
				"sim: structure %d is at id %d, which is not past its predecessor's %d",
				i, out[i].ID, out[i-1].ID)
		}
	}
	return out, off + int(span), nil
}
