package terrain

// The engine margin, read at the tier that DRAWS it.
//
// Every map's playable region is smaller than its tile grid: a fixed margin
// surrounds it that no ALM record carries (TERR-PASS-049). This file answers
// "is this cell in that margin" for a caller holding a derived block plane, and
// it answers it WITHOUT a depth of its own — see BorderCell.

// blockAir is bit 1 of a derived block byte: the bit that stops an air mover.
//
// It is spelled here, at a tier that reads such a byte, for the reason
// pkg/mapload spells it at the tier that writes one — the import DAG puts a
// shared constant out of reach of both, and a bare 2 at the read site would
// leave the bit's meaning written down nowhere on this side of the seam. What
// is duplicated is a bit position in a byte layout, which is what a tier
// boundary costs; the margin's WIDTH is not duplicated anywhere.
const blockAir uint8 = 1 << 1

// BorderCell reports whether cell (col,row) lies in the fixed engine margin
// that surrounds the map's playable region.
//
// It is answered from the plane's bit 1 and from NO depth constant of this
// tier's own, which is the whole design. Bit 1 is the margin's only writer
// — decoded in the game's own ingest, where the border stamp is the one
// arm that sets it (TERR-PASS-073), and true by construction of the plane
// this field actually carries, whose builder sets bit 1 if and only if the
// cell is in the margin. So the depth lives in exactly one place in this
// program, at the tier that derives the plane, and moving it moves what this
// function answers with no edit here at all. A second constant would be a
// second place for it to be wrong.
//
// It is TOTAL, and deliberately permissive rather than strict: a nil plane, a
// plane shorter than the cell asked for, a non-positive dimension or a cell
// outside the grid all answer false. False is the safe answer for every one of
// them — a caller that supplied no plane gets the picture it drew before the
// plane existed, which is what keeps this field's absence a non-event for every
// front-end and test that does not set it.
func (g Grid) BorderCell(col, row int) bool {
	if g.Width <= 0 || g.Height <= 0 {
		return false
	}
	if col < 0 || row < 0 || col >= g.Width || row >= g.Height {
		return false
	}
	i := row*g.Width + col
	if i >= len(g.Block) {
		return false
	}
	return g.Block[i]&blockAir != 0
}
