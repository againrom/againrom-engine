package mapload

import "againrom/pkg/formats/alm"

// Cell is a whole-tile position: a column and a row of a map's grid.
//
// It is a type of this package rather than a pair of ints because a drop cell,
// a party member's cell and the report that carries both are one thing said
// three times otherwise, and the two coordinates are the one pair a caller must
// not get the wrong way round.
type Cell struct {
	X, Y int32
}

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
// The nodes come through the format's one walk of the type-7 framing, which
// REQUIRES THE FRAMING TO TILE THE PAYLOAD EXACTLY; a payload it refuses
// answers no cell at all. That the payload is three counted arrays is a
// reading taken from a corpus rather than from a consuming instruction, and
// the exact tiling is what makes it falsifiable per file at load. A map it
// does not fit contributes nothing, rather than a cell read at an offset a
// walk drifted to.
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
	sc, err := m.Triggers.Decode()
	if err != nil {
		return nil
	}
	var out []Cell
	for _, n := range sc.Actions {
		if n.Opcode != dropOpcode {
			continue
		}
		out = append(out, Cell{X: int32(byte(n.Value[0])), Y: int32(byte(n.Value[1]))})
	}
	return out
}
