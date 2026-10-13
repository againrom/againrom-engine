package mapedit

import (
	"encoding/binary"
	"fmt"
)

// Plane names the grid layer a brush writes.
type Plane uint8

const (
	PlaneTile   Plane = iota + 1 // type 1, 16-bit tile words
	PlaneHeight                  // type 2, one height byte per cell
)

// Falloff names how a brush's effect fades from its centre to its rim.
type Falloff uint8

const (
	FalloffNone   Falloff = iota // every cell of the disc takes the full effect
	FalloffCosine                // the effect follows a quarter cosine wave to zero one cell past the radius
)

// Brush is the one operation both grid planes share, as data. On the tile plane
// Value is the terrain class the disc takes (a Palette entry's Class); on the
// height plane Value is the signed height change at the centre. A stroke is a
// Brush and the centres the pointer visited.
type Brush struct {
	Plane   Plane
	Value   int
	Radius  int
	Falloff Falloff
}

// Cell is a grid position.
type Cell struct{ X, Y int }

// Limits of a stroke. Heights are the 0..127 the format holds; a row step
// above rowStep between a cell and the one below it is bounded after a height
// stroke; maxRadius and maxCentres keep one call's work bounded.
const (
	heightMax  = 127
	rowStep    = 31
	maxRadius  = 16
	maxCentres = 1 << 16
)

// cosineWeight is the fall-off in sixty-fourths at eight steps from the centre
// to one cell past the radius.
var cosineWeight = [9]int{64, 63, 59, 53, 45, 36, 24, 12, 0}

// weight is the share of the brush's effect, in sixty-fourths, at squared
// distance d2 from the centre of a brush of radius r.
func weight(f Falloff, d2, r int) int {
	if f != FalloffCosine {
		return 64
	}
	k := 0
	for scaled := 64 * d2 / ((r + 1) * (r + 1)); (k+1)*(k+1) <= scaled && k < 8; {
		k++
	}
	return cosineWeight[k]
}

// disc calls fn for each cell of the brush's disc around c that lies on the
// grid, with its squared distance from c.
func disc(c Cell, r, w, h int, fn func(x, y, d2 int)) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x, y := c.X+dx, c.Y+dy
			if d2 := dx*dx + dy*dy; d2 <= r*r && x >= 0 && y >= 0 && x < w && y < h {
				fn(x, y, d2)
			}
		}
	}
}

// Paint applies one stroke as one edit and reports whether it changed a byte.
// A stroke that changes nothing records nothing; a rejected one writes nothing.
//
// A tile stroke sets the class of every cell of the disc and re-derives the
// words of those cells and their eight-neighbourhoods (Derive); a neighbour
// already showing the shape it would get keeps its word. The plane then takes
// the save normalisation (ALM-SAVE-238): the top eight rows take the last
// cell's word when the stroke touched either, and the type-0 tile-group mask
// is recomputed. A height stroke adds the brush's change, faded by its
// fall-off, clamps to 0..127 and bounds the row step at 31 (TERR-245). class
// is the classifier a tile stroke reads classes with.
func (e *Editor) Paint(b Brush, at []Cell, class func(uint16) uint8) (bool, error) {
	f := e.locate()
	meta := f.required(metaTypeID)
	w, h := int(e.u32(meta.abs(metaW))), int(e.u32(meta.abs(metaH)))
	switch {
	case len(at) == 0 || len(at) > maxCentres:
		return false, fmt.Errorf("mapedit: a stroke needs between 1 and %d cells", maxCentres)
	case b.Radius < 0 || b.Radius > maxRadius:
		return false, fmt.Errorf("mapedit: brush radius %d is outside 0..%d", b.Radius, maxRadius)
	}
	for _, c := range at {
		if c.X < 0 || c.Y < 0 || c.X >= w || c.Y >= h {
			return false, fmt.Errorf("mapedit: stroke cell (%d,%d) is outside the %dx%d grid", c.X, c.Y, w, h)
		}
	}
	var ed edit
	switch b.Plane {
	case PlaneTile:
		if class == nil {
			return false, fmt.Errorf("mapedit: a tile stroke needs a classifier")
		}
		var err error
		if ed, err = e.tileStroke(f, b, at, w, h, class); err != nil {
			return false, err
		}
	case PlaneHeight:
		ed = e.heightStroke(f, b, at, w, h)
	default:
		return false, fmt.Errorf("mapedit: no grid plane %d", b.Plane)
	}
	if len(ed) == 0 {
		return false, nil
	}
	e.apply(ed)
	return true, nil
}

func (e *Editor) tileStroke(f frame, b Brush, at []Cell, w, h int, class func(uint16) uint8) (edit, error) {
	c := uint8(b.Value)
	known := false
	for _, p := range Palette {
		known = known || p.Class == c && b.Value == int(c)
	}
	if !known {
		return nil, fmt.Errorf("mapedit: %d is not a paintable terrain class", b.Value)
	}
	span, meta := f.required(tileTypeID), f.required(metaTypeID)
	old := make([]uint16, w*h)
	for i := range old {
		old[i] = binary.LittleEndian.Uint16(e.data[span.abs(2*i):])
	}
	cls := make([]uint8, len(old))
	for i, v := range old {
		cls[i] = class(v)
	}
	painted := make([]bool, len(old))
	for _, cell := range at {
		disc(cell, b.Radius, w, h, func(x, y, _ int) {
			painted[y*w+x] = true
			cls[y*w+x] = c
		})
	}
	reach := make([]bool, len(old))
	for i, p := range painted {
		if !p {
			continue
		}
		x, y := i%w, i/w
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if nx, ny := x+dx, y+dy; nx >= 0 && ny >= 0 && nx < w && ny < h {
					reach[ny*w+nx] = true
				}
			}
		}
	}
	next := append([]uint16(nil), old...)
	for i, r := range reach {
		if !r || cls[i] == 0 || int(cls[i]) >= ClassCount {
			continue
		}
		x, y := i%w, i/w
		var around [8]uint8
		for d, o := range neighbourOffset {
			if nx, ny := x+o[0], y+o[1]; nx >= 0 && ny >= 0 && nx < w && ny < h {
				around[d] = cls[ny*w+nx]
			}
		}
		derived := Derive(cls[i], around, x, y)
		if !painted[i] && keepsShape(old[i], derived) {
			continue
		}
		if !painted[i] {
			if _, _, ok := WordShape(old[i]); !ok {
				continue
			}
		}
		next[i] = derived
		if !painted[i] {
			next[i] |= old[i] & blockedBit
		}
	}
	last := len(next) - 1
	touched := next[last] != old[last]
	for i := 0; i < 8*w && i < len(next) && !touched; i++ {
		touched = next[i] != old[i]
	}
	if touched {
		for i := 0; i < 8*w && i < len(next); i++ {
			next[i] = next[last] &^ blockedBit
			if !painted[i] {
				next[i] |= old[i] & blockedBit
			}
		}
	}
	var ed edit
	var cell [2]byte
	for i := 0; i < len(next); {
		if next[i] == old[i] {
			i++
			continue
		}
		j := i
		var run []byte
		for ; j < len(next) && next[j] != old[j]; j++ {
			binary.LittleEndian.PutUint16(cell[:], next[j])
			run = append(run, cell[:]...)
		}
		at := span.abs(2 * i)
		ed = append(ed, replace(at, e.data[at:at+2*(j-i)], run))
		i = j
	}
	if len(ed) == 0 {
		return nil, nil
	}
	mask := uint32(0)
	for _, v := range next {
		mask |= 1 << (v & 0x3ff >> 6)
	}
	if mask > 0xff {
		mask |= 0xf00
	}
	if mask > 0x1fff {
		mask = 0x1fff
	}
	if at := meta.abs(metaBitmask); e.u32(at) != mask {
		var img [4]byte
		binary.LittleEndian.PutUint32(img[:], mask)
		ed = append(ed, replace(at, e.data[at:at+4], img[:]))
	}
	return ed, nil
}

// blockedBit is the tile word's impassable flag, kept on unpainted cells.
const blockedBit = 0x2000

// keepsShape reports whether the word a cell shows already has the shape and
// strip pairing of the word the rule would give it.
func keepsShape(cur, derived uint16) bool {
	s1, g1, ok1 := WordShape(cur)
	s2, g2, ok2 := WordShape(derived)
	return ok1 && ok2 && s1 == s2 && (s1.Kind == ShapePure || g1 == g2)
}

func (e *Editor) heightStroke(f frame, b Brush, at []Cell, w, h int) edit {
	span := f.required(altitudeTypeID)
	old := append([]byte(nil), e.data[span.payloadOff:span.payloadOff+w*h]...)
	next := append([]byte(nil), old...)
	in := make([]bool, len(old))
	for _, cell := range at {
		disc(cell, b.Radius, w, h, func(x, y, d2 int) {
			i := y*w + x
			in[i] = true
			v := int(next[i]) + b.Value*weight(b.Falloff, d2, b.Radius)/64
			next[i] = byte(min(max(v, 0), heightMax))
		})
	}
	// Above the stroke's first cell the step is bounded by raising the upper
	// cell, bottom to top; every remaining step is bounded by lowering the
	// lower cell, top to bottom (TERR-245).
	anchor := at[0].Y
	for i := len(next) - w - 1; i >= 0; i-- {
		if in[i] && i/w < anchor && int(next[i+w])-int(next[i]) > rowStep {
			next[i] = next[i+w] - rowStep
		}
	}
	for i := 0; i+w < len(next); i++ {
		if in[i+w] && int(next[i+w])-int(next[i]) > rowStep {
			next[i+w] = next[i] + rowStep
		}
	}
	var ed edit
	for i := 0; i < len(next); {
		if next[i] == old[i] {
			i++
			continue
		}
		j := i
		for ; j < len(next) && next[j] != old[j]; j++ {
		}
		ed = append(ed, replace(span.abs(i), old[i:j], next[i:j]))
		i = j
	}
	return ed
}
