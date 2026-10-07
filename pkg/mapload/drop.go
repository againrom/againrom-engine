package mapload

import (
	"encoding/binary"

	"againrom/pkg/formats/alm"
)

// Cell is a whole-tile position: a column and a row of a map's grid.
//
// It is a type of this package rather than a pair of ints because a drop cell,
// a party member's cell and the report that carries both are one thing said
// three times otherwise, and the two coordinates are the one pair a caller must
// not get the wrong way round.
type Cell struct {
	X, Y int32
}

// The trigger payload's shape, as the map's own script stores it.
//
// The payload is THREE counted arrays, not one: actions, then conditions, then
// triggers, each behind its own count word. The format tier decodes the first
// count word and hands the rest over raw, so what is walked here begins at the
// first action record.
//
// The two record sizes are the sizes; nothing here reads a field of a condition
// or of a trigger, and both arrays exist in this file only so the walk can prove
// they tile.
const (
	nodeSize    = 796
	triggerSize = 184
	countSize   = 4
)

// The two words a node is read for, at their offsets inside the record: the
// opcode, and the first of the ten stored values. Nothing else of the 796 bytes
// is read — not the author's label, not the id, not the ten type codes, not the
// ten parameter names.
const (
	nodeOpcode = 0x40
	nodeValues = 0x4c
)

// dropOpcode is the action opcode the loader consumes at load time instead of
// dispatching at runtime: the drop table. It is the ONE opcode this file gives a
// meaning to, and the reason a story about starting a mission can read a map's
// script without owning the script's runtime.
const dropOpcode = 0x10002

// DropCells is every cell the map's script authorises as a start position, in
// node order.
//
// A cell is the LOW BYTE of each of the node's first two stored values — the
// original packs them into one word as `(y << 8) | x` and unpacks them again, so
// the byte is the whole of the coordinate and a value above 255 loses its high
// half rather than reaching the grid.
//
// THE WALK REQUIRES THE FRAMING TO TILE THE PAYLOAD EXACTLY, and answers no
// cell at all when it does not. That the payload is three counted arrays of
// 796, 796 and 184 bytes is a reading taken from a corpus rather than from a
// consuming instruction, and this is the reading's own discriminator: it is
// what makes the framing falsifiable per file at load, instead of believed
// once. A map it does not fit contributes nothing, rather than contributing
// a cell read at an offset the walk drifted to.
//
// A map with no trigger record, one with a truncated payload, and one whose count
// words overrun it are the same answer by the same path: no cells. The caller
// tells "the script authorises none" from "there is no script" by asking the map,
// not by asking here.
//
// A nil map is no cells, for the reason every derivation in this package is
// total: the signature has no error a caller could act on.
func DropCells(m *alm.Map) []Cell {
	if m == nil {
		return nil
	}
	body := m.Triggers.Body
	nActions := int(m.Triggers.EntryCount)
	if !tiles(body, nActions) {
		return nil
	}
	var out []Cell
	for i := 0; i < nActions; i++ {
		rec := body[i*nodeSize:]
		if binary.LittleEndian.Uint32(rec[nodeOpcode:]) != dropOpcode {
			continue
		}
		out = append(out, Cell{
			X: int32(byte(binary.LittleEndian.Uint32(rec[nodeValues:]))),
			Y: int32(byte(binary.LittleEndian.Uint32(rec[nodeValues+4:]))),
		})
	}
	return out
}

// tiles reports whether body is exactly nActions action records followed by two
// more counted arrays and nothing else.
//
// Every step is bounds-checked before it is taken and every product is computed
// in int64, so a count word of four billion is refused rather than wrapped into a
// small offset that then reads inside the payload. That matters because these
// counts come out of a file: a walk that trusted them would be an arbitrary read
// driven by two words a map author controls.
//
// "Exactly" means no residue. A payload one byte longer than the three arrays
// need is not a payload this framing describes, and saying so is the whole value
// of the check.
func tiles(body []byte, nActions int) bool {
	off, ok := advance(0, int64(nActions), nodeSize, len(body))
	if !ok {
		return false
	}
	for _, size := range [...]int64{nodeSize, triggerSize} {
		if off+countSize > int64(len(body)) {
			return false
		}
		n := int64(binary.LittleEndian.Uint32(body[off:]))
		off, ok = advance(off+countSize, n, size, len(body))
		if !ok {
			return false
		}
	}
	return off == int64(len(body))
}

// advance is off plus n records of size, or not-representable.
//
// The bound is written as a DIVISION rather than as a multiply-then-compare, so
// the product that would overflow is never formed: `n > (length-off)/size` is
// false only for an n whose product fits, which is what lets every caller above
// index with the result.
func advance(off, n, size int64, length int) (int64, bool) {
	if n < 0 || n > (int64(length)-off)/size {
		return 0, false
	}
	return off + n*size, true
}
