package mapedit

import "fmt"

// Patch is a rectangle of the three cell planes, row by row: W x H tile
// words, heights and object class bytes. Objects is nil when the map it was
// copied from carries no type-3 record.
type Patch struct {
	W, H    int
	Tiles   []uint16
	Heights []byte
	Objects []byte
}

// CopyPatch copies the w x h rectangle of cells whose top-left cell is (x, y)
// out of the three cell planes (ALM-EDITDISP-279: the original's copy works
// on a cell rectangle of tiles, altitudes and object bytes). The rectangle
// must lie on the grid.
func (e *Editor) CopyPatch(x, y, w, h int) (Patch, error) {
	gw, gh := e.GridSize()
	if w < 1 || h < 1 || x < 0 || y < 0 || x+w > gw || y+h > gh {
		return Patch{}, fmt.Errorf("mapedit: a %dx%d patch at (%d,%d) is not on the %dx%d grid", w, h, x, y, gw, gh)
	}
	f := e.locate()
	tiles, heights := e.tileWords(f.required(tileTypeID), gw*gh), f.required(altitudeTypeID)
	p := Patch{W: w, H: h, Tiles: make([]uint16, 0, w*h), Heights: make([]byte, 0, w*h)}
	objects, err := f.optional("copy objects", overlayTypeID)
	if err == nil {
		p.Objects = make([]byte, 0, w*h)
	}
	for row := y; row < y+h; row++ {
		at := row*gw + x
		p.Tiles = append(p.Tiles, tiles[at:at+w]...)
		p.Heights = append(p.Heights, e.data[heights.abs(at):heights.abs(at+w)]...)
		if p.Objects != nil {
			p.Objects = append(p.Objects, e.data[objects.abs(at):objects.abs(at+w)]...)
		}
	}
	return p, nil
}

// PastePatch writes p with its top-left cell on at, as one edit: the tile
// words, the heights and the object bytes as they were copied, raw, as the
// original's paste writes them (ALM-EDITPASTE-300), each cell of p that falls
// off the grid left out. A patch object byte of 0 clears the cell. The tile
// plane takes the save normalisation (tileEdit). A patch with objects on a map
// with no type-3 record is refused, as is a patch whose planes do not match
// its size. It reports whether a byte changed.
func (e *Editor) PastePatch(p Patch, at Cell) (bool, error) {
	n := p.W * p.H
	if p.W < 1 || p.H < 1 || len(p.Tiles) != n || len(p.Heights) != n || p.Objects != nil && len(p.Objects) != n {
		return false, fmt.Errorf("mapedit: a %dx%d patch with %d tiles, %d heights and %d objects", p.W, p.H, len(p.Tiles), len(p.Heights), len(p.Objects))
	}
	f := e.locate()
	gw, gh := e.GridSize()
	tiles, heights := f.required(tileTypeID), f.required(altitudeTypeID)
	objects, err := f.optional("paste objects", overlayTypeID)
	if err != nil && p.Objects != nil {
		for _, b := range p.Objects {
			if b != 0 {
				return false, err
			}
		}
	}
	oldT := e.tileWords(tiles, gw*gh)
	nextT := append([]uint16(nil), oldT...)
	oldH := append([]byte(nil), e.data[heights.payloadOff:heights.payloadOff+gw*gh]...)
	nextH := append([]byte(nil), oldH...)
	var oldO, nextO []byte
	if err == nil && p.Objects != nil {
		oldO = append([]byte(nil), e.data[objects.payloadOff:objects.payloadOff+gw*gh]...)
		nextO = append([]byte(nil), oldO...)
	}
	for py := range p.H {
		for px := range p.W {
			x, y := at.X+px, at.Y+py
			if x < 0 || y < 0 || x >= gw || y >= gh {
				continue
			}
			i, k := y*gw+x, py*p.W+px
			nextT[i], nextH[i] = p.Tiles[k], p.Heights[k]
			if nextO != nil {
				nextO[i] = p.Objects[k]
			}
		}
	}
	ed := append(e.tileEdit(f, gw, oldT, nextT, nil), byteRuns(heights, oldH, nextH)...)
	if nextO != nil {
		ed = append(ed, byteRuns(objects, oldO, nextO)...)
	}
	if len(ed) == 0 {
		return false, nil
	}
	e.apply(ed)
	return true, nil
}
