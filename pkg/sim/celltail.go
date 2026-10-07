package sim

import (
	"fmt"
	"sort"
)

// The CELL-RECORD TAIL instant 25 writes.
//
// `TRIG-CELLTAIL-035` reads the arm end to end: it takes `(spell, power, x, y)`,
// packs the addressed cell, and hands a helper six bytes that the helper stores
// at the cell's own dynamic record, `+0x2c..+0x31`. The helper creates the record
// where the cell has none. UNIT-M10ENTRY-055 adds the ground-footprint reader:
// spell bytes other than 0 and 26 request a cast at the entering actor. The
// separate relocation operation for spell 26 remains unimplemented.
//
// IT IS ITS OWN RECORD AND NOT AN AREA EFFECT (D-5). celleffect.go holds the
// six-slot area-effect array instants 21, 24 and 29 work on. This is the dynamic
// cell record, a different object in the thing being reconstructed, and the two
// key a cell by two different expressions — see tailKey below. Folding them into
// one would make that difference a special case instead of a consequence.

// cellTailLen is how many bytes of the cell record this package models: the six
// the instant-25 helper stores. The record is 52 bytes in the original and the
// rest of it is not written by any arm this build has.
const cellTailLen = 6

// cellTail is one cell's tail: which cell, and the six bytes standing on it.
//
// Key is the packed cell and is the ONLY position this record carries, on
// cellEffect's own rule: a second pair of coordinates beside it could come to
// disagree with the key the arm actually addresses.
type cellTail struct {
	Key   uint16
	Bytes [cellTailLen]byte
}

// tailKey is instant 25's own cell key: `(u8(y) << 8) OR u8(x)`.
//
// IT IS NOT cellKey (D-6). Instant 29's key is a 16-bit ADD over 16-bit
// truncations (`TRIG-CELLEFFECT-045`), so an x above 255 carries into the y
// byte; this one truncates BOTH operands to a byte first and ors them, so an x
// above 255 is masked and cannot carry. The two are written as two functions
// with this comment between them rather than shared, because a shared helper
// would have to be one of the two and would be wrong for the other.
func tailKey(x, y int32) uint16 {
	return uint16(uint8(y))<<8 | uint16(uint8(x))
}

// CellTail is one cell tail as a consumer sees it: where it stands, and its
// bytes.
type CellTail struct {
	X, Y  int32
	Bytes [cellTailLen]byte
}

// CellTails returns every cell tail standing on the world, in ascending cell
// key. It is a fresh slice, on Entities', Sacks' and CellEffects' own rule:
// mutating the result cannot reach the world it came from.
//
// The cell is read back through keyCell, which is celleffect.go's own reader of
// the high and low bytes of a packed cell. That is correct for both keys: the
// two combiners differ in what they do with an x above 255, and both leave x in
// the low byte and y in the high one.
func (w *World) CellTails() []CellTail {
	out := make([]CellTail, len(w.cellTails))
	for i, t := range w.cellTails {
		x, y := keyCell(t.Key)
		out[i] = CellTail{X: x, Y: y, Bytes: t.Bytes}
	}
	return out
}

// ImportOriginalCellTails overlays the saved projection after map construction.
// SAV-CELLLOAD-109 copies each saved payload onto its key or inserts a new key;
// construction-only records survive and duplicate saved keys are last-write-wins.
// This is not DeclareCellTails: saved keys are not subject to the fresh writer's
// block-plane gate or current map bounds. All six bytes survive, including zero,
// operation 26 and unknown operations. Nothing attaches or casts during restore.
func (w *World) ImportOriginalCellTails(tails []CellTail) error {
	for _, t := range tails {
		if t.X < 0 || t.X > 255 || t.Y < 0 || t.Y > 255 {
			return fmt.Errorf("sim: saved cell-tail position (%d,%d) exceeds byte coordinates", t.X, t.Y)
		}
	}
	for _, t := range tails {
		w.writeCellTail(tailKey(t.X, t.Y), t.Bytes)
	}
	return nil
}

// setCellTail is instant 25's whole arm.
//
// THE SIX BYTES ARE `{spell, power, 0, y, 0, y}`, each narrowed to a byte, and
// the authored x IS NOT AMONG THEM. That is what the helper is handed and it is
// not a simplification: the arm packs x into the key and then passes y twice.
//
// ONE TAIL PER CELL, OVERWRITTEN IN PLACE. The helper updates the cell's record
// where one exists and creates it where one does not, so a cell written twice
// carries the second node's bytes and not two records.
//
// THE SLICE IS KEPT SORTED BY KEY, which is placeCellEffect's own reason:
// canonical order is what makes the byte form a function of the logical
// world rather than of the order the writes happened in, and the digest is
// taken over exactly the bytes that form produces.
//
// NO VALUE IS REFUSED. There is no test in front of the store — not on the
// spell, not on the power, and not on a cell that lies outside this world's own
// bounds, because the key is a packed byte pair and every packing names some
// cell of the 256x256 space the key can address. Runtime footprint entry reads
// only byte-addressable cells inside the world. DeclareCellTails applies the
// initial installed-data writer's separate block-plane gate.
func (w *World) setCellTail(spell, power, x, y int32) {
	key := tailKey(x, y)
	bytes := [cellTailLen]byte{
		byte(spell), byte(power), 0, byte(y), 0, byte(y),
	}
	w.writeCellTail(key, bytes)
}

func (w *World) writeCellTail(key uint16, bytes [cellTailLen]byte) {
	i := sort.Search(len(w.cellTails), func(i int) bool { return w.cellTails[i].Key >= key })
	if i < len(w.cellTails) && w.cellTails[i].Key == key {
		w.cellTails[i].Bytes = bytes
		w.syncSavedCellTail(key, bytes)
		return
	}
	w.cellTails = append(w.cellTails, cellTail{})
	copy(w.cellTails[i+1:], w.cellTails[i:])
	w.cellTails[i] = cellTail{Key: key, Bytes: bytes}
	w.syncSavedCellTail(key, bytes)
}
