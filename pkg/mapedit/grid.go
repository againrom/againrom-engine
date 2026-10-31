package mapedit

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/formats/alm"
)

// GridLayer names one of the three grid layers SetCell writes.
type GridLayer uint8

const (
	// GridTile is the type-1 layer, a 16-bit word per cell.
	GridTile GridLayer = iota
	// GridAltitude is the type-2 layer, a byte per cell.
	GridAltitude
	// GridOverlay is the type-3 layer, a byte per cell, present only on a map
	// that carries the record.
	GridOverlay
)

// GridPolicy is how SetCell finishes a cell write.
type GridPolicy uint8

const (
	// GridRaw writes the value as the whole cell and nothing else; an equal
	// value is still recorded as an edit.
	GridRaw GridPolicy = iota
	// GridNormalized writes as a brush stroke does: an unchanged cell records
	// nothing, and a tile write takes the save normalisation (tileEdit).
	GridNormalized
)

// SetCell is the one entry that writes a single grid cell. v is the whole cell
// word: sixteen bits of a tile, a byte of an altitude or overlay cell (a larger
// v is rejected). A map without a type-3 record has no overlay to write: alm
// manufactures that plane for the view only, so the call fails.
func (e *Editor) SetCell(p GridPolicy, layer GridLayer, x, y int, v uint16) error {
	f := e.locate()
	var grid span
	var name string
	switch layer {
	case GridTile:
		grid, name = f.required(alm.TypeTiles), "tile"
	case GridAltitude:
		grid, name = f.required(alm.TypeAltitude), "altitude"
	case GridOverlay:
		var err error
		if grid, err = f.optional("set an overlay cell", alm.TypeOverlay); err != nil {
			return err
		}
		name = "overlay"
	default:
		return fmt.Errorf("mapedit: no grid layer %d", layer)
	}
	if layer != GridTile && v > 0xff {
		return fmt.Errorf("mapedit: %s value %d does not fit a byte", name, v)
	}
	switch p {
	case GridRaw:
		cell := []byte{byte(v)}
		if layer == GridTile {
			cell = binary.LittleEndian.AppendUint16(nil, v)
		}
		return e.setCell(f, grid, name, x, y, cell)
	case GridNormalized:
		return e.normalizeCell(f, grid, layer, name, x, y, v)
	}
	return fmt.Errorf("mapedit: no grid policy %d", p)
}

// gridIndex resolves (x,y) against W and H read as unsigned, so a hostile width
// cannot fold a rejection into an accepted index.
func (e *Editor) gridIndex(f frame, layer string, x, y int) (idx, w, cells int, err error) {
	meta := f.required(alm.TypeMeta)
	gw := e.u32(meta.abs(alm.MetaWidth))
	gh := e.u32(meta.abs(alm.MetaHeight))
	if x < 0 || y < 0 || uint64(x) >= uint64(gw) || uint64(y) >= uint64(gh) {
		return 0, 0, 0, fmt.Errorf("mapedit: %s cell (%d,%d) is outside the %dx%d grid", layer, x, y, gw, gh)
	}
	return y*int(gw) + x, int(gw), int(gw) * int(gh), nil
}

// setCell is the raw write of SetCell, entered with the layer's record located.
// An out-of-range cell is rejected before an edit exists. The cell's width is
// the length of the image handed in. alm ties each payload length to W and H,
// so a cell inside the grid is inside its payload.
func (e *Editor) setCell(f frame, grid span, layer string, x, y int, cell []byte) error {
	idx, _, _, err := e.gridIndex(f, layer, x, y)
	if err != nil {
		return err
	}
	at := grid.abs(idx * len(cell))
	e.apply(edit{replace(at, e.data[at:at+len(cell)], cell)})
	return nil
}

// normalizeCell is the normalized write of SetCell: the layer as it stands with
// one cell changed, turned into an edit by the same writers a stroke uses.
func (e *Editor) normalizeCell(f frame, grid span, layer GridLayer, name string, x, y int, v uint16) error {
	idx, w, cells, err := e.gridIndex(f, name, x, y)
	if err != nil {
		return err
	}
	var ed edit
	if layer == GridTile {
		old := e.tileWords(grid, cells)
		next := append([]uint16(nil), old...)
		next[idx] = v
		ed = e.tileEdit(f, w, old, next, nil)
	} else {
		old := append([]byte(nil), e.data[grid.payloadOff:grid.payloadOff+cells]...)
		next := append([]byte(nil), old...)
		next[idx] = byte(v)
		ed = byteRuns(grid, old, next)
	}
	if len(ed) != 0 {
		e.apply(ed)
	}
	return nil
}
