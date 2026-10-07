package mapedit

import (
	"encoding/binary"
	"fmt"
)

// The three grid layers and the two type-0 words that size them. Each layer's
// payload is pure cells, row-major from its first byte, so cell (x,y) begins at
// (y*W+x)*elem with elem that layer's own cell width — two bytes for a type-1
// tile, one for a type-2 altitude or a type-3 overlay.
const (
	tileTypeID     = 1
	altitudeTypeID = 2
	overlayTypeID  = 3

	metaW = 0x00
	metaH = 0x04
)

// SetTile sets the type-1 cell at (x,y) to v as a whole 16-bit word. The cell's
// flag bits are the caller's: this package writes the value given and asserts
// no meaning for any bit of it.
func (e *Editor) SetTile(x, y int, v uint16) error {
	var cell [2]byte
	binary.LittleEndian.PutUint16(cell[:], v)
	f := e.locate()
	return e.setCell(f, f.required(tileTypeID), "tile", x, y, cell[:])
}

// SetAltitude sets the type-2 altitude cell at (x,y) to v.
func (e *Editor) SetAltitude(x, y int, v uint8) error {
	f := e.locate()
	return e.setCell(f, f.required(altitudeTypeID), "altitude", x, y, []byte{v})
}

// SetOverlay sets the type-3 overlay cell at (x,y) to v, on a map that carries
// a type-3 record — and is rejected on one that does not.
//
// It is the only grid layer with that answer, and the asymmetry is the format's
// rather than this package's: alm refuses a stream without type-1 or type-2,
// and manufactures a W*H zero plane when type-3 is missing so the decoded view
// always has one. That plane is the reader's, not the file's. Painting a cell
// of it would mean adding the record, which this story does not do — so the
// call fails instead of inventing bytes the input never had.
func (e *Editor) SetOverlay(x, y int, v uint8) error {
	f := e.locate()
	grid, err := f.optional("set an overlay cell", overlayTypeID)
	if err != nil {
		return err
	}
	return e.setCell(f, grid, "overlay", x, y, []byte{v})
}

// setCell is the one write path of the three grid setters, entered with the
// layer's record already located: whether that layer is in the file at all is
// the caller's question, and it is a different one from whether (x,y) is on the
// grid. It resolves (x,y) against W and H read out of the current bytes, and it
// finishes rejecting before an edit exists — an out-of-range cell has written
// nothing and recorded nothing, because there was nothing to write or record
// yet. The cell's width is the length of the image handed in, so the two cannot
// disagree.
//
// No second bounds check against the layer's payload is needed. alm's
// acceptance ties each layer's payload length to W and H — 2*W*H for the tiles,
// W*H for the altitudes, and W*H for the overlay wherever that record is
// present at all — so a cell inside the grid is inside its layer's payload for
// as long as the buffer is a stream alm accepted, which is every state this
// model has. W and H are read as unsigned and compared as such, so a hostile
// width cannot fold a rejection into an accepted index.
func (e *Editor) setCell(f frame, grid span, layer string, x, y int, cell []byte) error {
	meta := f.required(metaTypeID)
	w := e.u32(meta.abs(metaW))
	h := e.u32(meta.abs(metaH))
	if x < 0 || y < 0 || uint64(x) >= uint64(w) || uint64(y) >= uint64(h) {
		return fmt.Errorf("mapedit: %s cell (%d,%d) is outside the %dx%d grid", layer, x, y, w, h)
	}

	at := grid.abs((y*int(w) + x) * len(cell))
	e.apply(edit{replace(at, e.data[at:at+len(cell)], cell)})
	return nil
}
