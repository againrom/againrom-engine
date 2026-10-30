package mapedit

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"math"
	"slices"

	"againrom/pkg/formats/alm"
)

// Plane names the grid layer a brush writes.
type Plane uint8

const (
	PlaneTile       Plane = iota + 1 // type 1, 16-bit tile words
	PlaneHeight                      // type 2, one height byte per cell
	PlaneObject                      // type 3, one static-object class code per cell
	PlaneImpassable                  // type 1, bit 13 of each tile word alone
)

// BrushShape names the cells a brush covers around a centre.
type BrushShape uint8

const (
	BrushCircle BrushShape = iota // the cells within the radius of the centre
	BrushSquare                   // the cells within the radius on both axes: a square of side 2*radius+1
)

// Falloff names how a brush's effect fades from its centre to its rim.
type Falloff uint8

const (
	FalloffNone   Falloff = iota // every cell of the brush takes the full effect
	FalloffCosine                // the effect follows a quarter cosine wave to zero one cell past the radius
)

// Brush is the one operation the grid planes share, as data. On the tile plane
// Value is the terrain class the brush cells take (a Palette entry's Class);
// on the height plane Value is the signed height change at the centre; on the
// object plane Value is the class code every brush cell takes, 0 clearing it;
// on the impassable plane Value 1 sets and 0 clears the tile word's bit 13.
// Radius and Shape give the cells around each centre (walk). A stroke is a
// Brush and the centres the pointer visited.
type Brush struct {
	Plane   Plane
	Value   int
	Radius  int
	Shape   BrushShape
	Falloff Falloff
	// Fill chooses the words of the tile plane's centre cells; the zero value
	// is the position pattern.
	Fill Fill
	// Scatter, when it names classes, makes an object stroke plant or thin a
	// mix instead of writing Value.
	Scatter Scatter
	// Set, on the height plane, makes the stroke set a height instead of
	// adding one: Value is the height (0..127) the first centre takes, and the
	// cells within the radius blend toward it by cosine from their own height
	// (TERR-251). The brush is a circle of radius at least 1.
	Set bool
	// Flat, with Set, makes every cell the brush covers around every centre
	// take Value, with no blend, in the brush's own shape and radius; the row
	// step is then bounded as an added height's is.
	Flat bool
	// Relative, with Set and Flat, makes Value the change from the height of
	// the first centre's cell: the stroke levels its path at that height
	// plus Value, clamped to 0..127.
	Relative bool
	// Once, on the height plane without Set, adds the change to each cell
	// the stroke covers once, however many centres cover it.
	Once bool
}

// Scatter is the forest form of an object stroke. Each covered cell is drawn
// once: a planting stroke gives an open cell one of Classes, each equally
// likely, at Share per cent; a thinning stroke (Thin) clears a cell holding
// one of Classes at Share per cent and leaves every other cell. A cell is open
// when its type-3 byte is 0, its tile index is not in the water range
// (TERR-PASS-049), the classifier does not call it Road (TERR-PASS-050) and
// Fenced, the host's per-cell structure footprint (TERR-STRUCT-068), does not
// hold it. Draw answers a number in 0..n-1; the caller owns the source.
type Scatter struct {
	Classes []int
	Share   int
	Thin    bool
	Draw    func(n int) int
	Fenced  []bool
}

// Cell is a grid position.
type Cell struct{ X, Y int }

// Limits of a stroke. Heights are the 0..alm.MaxHeight the format holds; a row step
// above rowStep between a cell and the one below it is bounded after a height
// stroke; MaxRadius and maxCentres keep one call's work bounded.
const (
	rowStep    = 31
	MaxRadius  = 16
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

// effective is b as it covers cells: a height-setting brush that blends is a
// circle of radius at least 1 (TERR-251).
func (b Brush) effective() Brush {
	if b.Set && !b.Flat && b.Plane == PlaneHeight {
		b.Shape, b.Radius = BrushCircle, max(b.Radius, 1)
	}
	return b
}

// walk calls fn for each cell the brush covers around c that lies on the
// grid, in row order, with its squared distance from c: the Euclidean one for
// a circle, the larger axis distance for a square. The fall-off reads that
// distance, so a square brush fades by rings of squares.
func (b Brush) walk(c Cell, w, h int, fn func(x, y, d2 int)) {
	b = b.effective()
	r := b.Radius
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			d2 := dx*dx + dy*dy
			if b.Shape == BrushSquare {
				d2 = max(dx*dx, dy*dy)
			}
			if x, y := c.X+dx, c.Y+dy; d2 <= r*r && x >= 0 && y >= 0 && x < w && y < h {
				fn(x, y, d2)
			}
		}
	}
}

// GridSize is the map's width and height in cells.
func (e *Editor) GridSize() (w, h int) {
	meta := e.locate().required(metaTypeID)
	return int(e.u32(meta.abs(metaW))), int(e.u32(meta.abs(metaH)))
}

// Footprint is every cell of a w x h grid a stroke of b through the centres
// at covers, each once, in row order. A height-setting brush that blends
// covers its effective circle around the first centre alone, as heights
// writes it.
func (b Brush) Footprint(at []Cell, w, h int) []Cell {
	b = b.effective()
	if b.Set && !b.Flat && b.Plane == PlaneHeight && len(at) > 1 {
		at = at[:1]
	}
	in := map[Cell]bool{}
	var out []Cell
	for _, c := range at {
		b.walk(c, w, h, func(x, y, _ int) {
			if cell := (Cell{x, y}); !in[cell] {
				in[cell] = true
				out = append(out, cell)
			}
		})
	}
	slices.SortFunc(out, func(a, b Cell) int { return cmp.Or(a.Y-b.Y, a.X-b.X) })
	return out
}

// Paint applies one stroke as one edit and reports whether it changed a byte.
// A stroke that changes nothing records nothing; a rejected one writes nothing.
//
// A tile stroke gives every brush cell the class and rewrites the cells whose
// corners that moves (TileFix); a cell still showing its corners keeps its
// word. The plane then takes the save normalisation (ALM-SAVE-238): the top
// eight rows take the last cell's word when the stroke touched either, and
// the type-0 tile-group mask is recomputed. A height stroke adds the brush's change, faded by its
// fall-off, clamps to 0..127 and bounds the row step at 31 (TERR-245). class
// is the classifier a tile stroke reads classes with.
func (e *Editor) Paint(b Brush, at []Cell, class func(uint16) uint8) (bool, error) {
	f := e.locate()
	meta := f.required(metaTypeID)
	w, h := int(e.u32(meta.abs(metaW))), int(e.u32(meta.abs(metaH)))
	switch {
	case len(at) == 0 || len(at) > maxCentres:
		return false, fmt.Errorf("mapedit: a stroke needs between 1 and %d cells", maxCentres)
	case b.Radius < 0 || b.Radius > MaxRadius:
		return false, fmt.Errorf("mapedit: brush radius %d is outside 0..%d", b.Radius, MaxRadius)
	case b.Shape != BrushCircle && b.Shape != BrushSquare:
		return false, fmt.Errorf("mapedit: no brush shape %d", b.Shape)
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
	case PlaneObject:
		var err error
		if len(b.Scatter.Classes) > 0 {
			if class == nil {
				return false, fmt.Errorf("mapedit: a forest stroke needs a classifier")
			}
			ed, err = e.scatterStroke(f, b, at, w, h, class)
		} else {
			ed, err = e.objectStroke(f, b, at, w, h)
		}
		if err != nil {
			return false, err
		}
	case PlaneImpassable:
		var err error
		if ed, err = e.impassableStroke(f, b, at, w, h); err != nil {
			return false, err
		}
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
	span := f.required(tileTypeID)
	old := e.tileWords(span, w*h)
	painted := make([]bool, len(old))
	var cells []int
	for _, cell := range at {
		b.walk(cell, w, h, func(x, y, _ int) {
			if i := y*w + x; !painted[i] {
				painted[i] = true
				cells = append(cells, i)
			}
		})
	}
	next := append([]uint16(nil), old...)
	fix := NewTileFix(old, w, h, class)
	fix.Fill = b.Fill
	for _, ch := range fix.Paint(cells, c) {
		next[ch.Cell] = ch.Word
	}
	return e.tileEdit(f, w, old, next, painted), nil
}

// tileEdit is the one writer of the tile plane: the edit that turns old into
// next after the save normalisation (ALM-SAVE-238). When next differs from
// old in the top eight rows or the last cell, the top eight rows take the
// last cell's word, each keeping its own bit 13 unless cleared marks it (nil
// marks none); then the type-0 tile-group mask is recomputed. next is
// normalised in place.
func (e *Editor) tileEdit(f frame, w int, old, next []uint16, cleared []bool) edit {
	span, meta := f.required(tileTypeID), f.required(metaTypeID)
	last := len(next) - 1
	touched := next[last] != old[last]
	for i := 0; i < 8*w && i < len(next) && !touched; i++ {
		touched = next[i] != old[i]
	}
	if touched {
		for i := 0; i < 8*w && i < len(next); i++ {
			next[i] = next[last] &^ blockedBit
			if cleared == nil || !cleared[i] {
				next[i] |= old[i] & blockedBit
			}
		}
	}
	ed := e.wordRuns(span, old, next)
	if len(ed) == 0 {
		return nil
	}
	mask := alm.TileGroupMask(next)
	if at := meta.abs(metaBitmask); e.u32(at) != mask {
		var img [4]byte
		binary.LittleEndian.PutUint32(img[:], mask)
		ed = append(ed, replace(at, e.data[at:at+4], img[:]))
	}
	return ed
}

// tileWords is the tile plane of n cells as it stands.
func (e *Editor) tileWords(span span, n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(e.data[span.abs(2*i):])
	}
	return out
}

// wordRuns is the edit that turns the 16-bit plane at span from old into
// next: one part per run of changed cells.
func (e *Editor) wordRuns(span span, old, next []uint16) edit {
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
	return ed
}

// byteRuns is the edit that turns the byte plane at span from old into next:
// one part per run of changed cells.
func byteRuns(span span, old, next []byte) edit {
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

// blockedBit is the tile word's impassable flag, kept on unpainted cells.
const blockedBit = 0x2000

func (e *Editor) heightStroke(f frame, b Brush, at []Cell, w, h int) edit {
	span := f.required(altitudeTypeID)
	old := append([]byte(nil), e.data[span.payloadOff:span.payloadOff+w*h]...)
	return byteRuns(span, old, b.heights(old, at, w, h))
}

// heights is the height plane old after a stroke of b through the centres at:
// the change, faded by the fall-off, clamped to 0..127, then the row step
// bounded at 31 (TERR-245). old is not changed.
func (b Brush) heights(old []byte, at []Cell, w, h int) []byte {
	b = b.effective()
	next := append([]byte(nil), old...)
	in := make([]bool, len(old))
	if b.Set && b.Flat {
		v := b.Value
		if b.Relative {
			v += int(old[at[0].Y*w+at[0].X])
		}
		target := byte(min(max(v, 0), alm.MaxHeight))
		for _, c := range at {
			b.walk(c, w, h, func(x, y, _ int) { next[y*w+x], in[y*w+x] = target, true })
		}
		boundRows(next, in, at[0].Y, w)
		return next
	}
	if b.Set {
		// The first centre takes the target; each other cell within the radius
		// blends from its own height toward it by the cosine of its distance
		// (TERR-251), truncated to a byte.
		target := min(max(b.Value, 0), alm.MaxHeight)
		b.walk(at[0], w, h, func(x, y, d2 int) {
			i := y*w + x
			in[i] = true
			if d2 == 0 {
				next[i] = byte(target)
				return
			}
			d := math.Sqrt(float64(d2))
			v := float64(old[i]) + float64(target-int(old[i]))*math.Cos(d*3.1415926/float64(2*b.Radius))
			next[i] = byte(min(max(int(v), 0), alm.MaxHeight))
		})
		boundSetRows(next, in, at[0], w)
		return next
	}
	b.addHeights(next, in, at, w, h)
	boundRows(next, in, at[0].Y, w)
	return next
}

// boundSetRows bounds the row step at 31 around the centre c of a
// height-setting stroke (TERR-251): only the cells with |dx| < |dy| are
// changed, so the centre keeps its target. Above c the upper cell rises to
// the cell below it less 31; below c the lower cell falls to the cell above
// it plus 31; both run away from c.
func boundSetRows(next []byte, in []bool, c Cell, w int) {
	for y := c.Y - 1; y >= 0; y-- {
		for x := 0; x < w; x++ {
			if i := y*w + x; in[i] && abs(x-c.X) < c.Y-y && int(next[i+w])-int(next[i]) > rowStep {
				next[i] = next[i+w] - rowStep
			}
		}
	}
	for y := c.Y + 1; (y+1)*w <= len(next); y++ {
		for x := 0; x < w; x++ {
			if i := y*w + x; in[i] && abs(x-c.X) < y-c.Y && int(next[i])-int(next[i-w]) > rowStep {
				next[i] = next[i-w] + rowStep
			}
		}
	}
}

func abs(v int) int { return max(v, -v) }

// addHeights adds the brush's change, faded by its fall-off, to next at every
// cell the brush covers around the centres at, and marks those cells in in.
// With Once a cell already marked takes no further change.
func (b Brush) addHeights(next []byte, in []bool, at []Cell, w, h int) {
	for _, cell := range at {
		b.walk(cell, w, h, func(x, y, d2 int) {
			i := y*w + x
			if b.Once && in[i] {
				return
			}
			in[i] = true
			v := int(next[i]) + b.Value*weight(b.Falloff, d2, b.Radius)/64
			next[i] = byte(min(max(v, 0), alm.MaxHeight))
		})
	}
}

// boundRows bounds the row step at 31 in next among the cells marked in in
// (TERR-245), for a stroke whose first cell is in row anchor.
func boundRows(next []byte, in []bool, anchor, w int) {
	// Above the stroke's first cell the step is bounded by raising the upper
	// cell, bottom to top; every remaining step is bounded by lowering the
	// lower cell, top to bottom (TERR-245).
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
}

// objectStroke writes the class code (0..255) into every brush cell.
// Which codes name a class is the host's knowledge (the object registry); the
// model refuses a code a byte cannot hold and a map with no type-3 record.
func (e *Editor) objectStroke(f frame, b Brush, at []Cell, w, h int) (edit, error) {
	if b.Value < 0 || b.Value > 255 {
		return nil, fmt.Errorf("mapedit: %d is not an object class code", b.Value)
	}
	span, err := f.optional("paint objects", overlayTypeID)
	if err != nil {
		return nil, err
	}
	old := append([]byte(nil), e.data[span.payloadOff:span.payloadOff+w*h]...)
	next := append([]byte(nil), old...)
	for _, cell := range at {
		b.walk(cell, w, h, func(x, y, _ int) { next[y*w+x] = byte(b.Value) })
	}
	return byteRuns(span, old, next), nil
}

// waterIndex is the tile index range the block plane's water arm tests
// (TERR-PASS-049 (d)).
func waterIndex(word uint16) bool { return alm.TileIndex(word)&0x300 == 0x200 }

// scatterStroke plants or thins Scatter.Classes over the cells the brush
// covers, each cell once and in row order, so one source gives one result.
func (e *Editor) scatterStroke(f frame, b Brush, at []Cell, w, h int, class func(uint16) uint8) (edit, error) {
	sc := b.Scatter
	switch {
	case sc.Share < 1 || sc.Share > 100:
		return nil, fmt.Errorf("mapedit: a forest share of %d%% is outside 1..100", sc.Share)
	case sc.Draw == nil:
		return nil, fmt.Errorf("mapedit: a forest stroke needs a random source")
	case sc.Fenced != nil && len(sc.Fenced) != w*h:
		return nil, fmt.Errorf("mapedit: a structure plane of %d cells for a %dx%d grid", len(sc.Fenced), w, h)
	}
	kinds := map[int]bool{}
	for _, c := range sc.Classes {
		if c < 1 || c > 255 {
			return nil, fmt.Errorf("mapedit: %d is not an object class code a forest plants", c)
		}
		kinds[c] = true
	}
	span, err := f.optional("paint objects", overlayTypeID)
	if err != nil {
		return nil, err
	}
	tiles := f.required(tileTypeID)
	old := append([]byte(nil), e.data[span.payloadOff:span.payloadOff+w*h]...)
	next := append([]byte(nil), old...)
	for _, c := range b.Footprint(at, w, h) {
		i := c.Y*w + c.X
		if sc.Thin {
			if kinds[int(old[i])] && sc.Draw(100) < sc.Share {
				next[i] = 0
			}
			continue
		}
		word := binary.LittleEndian.Uint16(e.data[tiles.abs(2*i):])
		if old[i] != 0 || waterIndex(word) || class(word) == ClassRoad || sc.Fenced != nil && sc.Fenced[i] {
			continue
		}
		if sc.Draw(100) < sc.Share {
			next[i] = byte(sc.Classes[sc.Draw(len(sc.Classes))])
		}
	}
	return byteRuns(span, old, next), nil
}

// impassableStroke sets (Value 1) or clears (Value 0) bit 13 of the tile word
// of every brush cell (ALM-GRID-012) and writes no other bit, cell or
// record: the tile-group mask reads bits 0-9 alone, so it cannot move.
func (e *Editor) impassableStroke(f frame, b Brush, at []Cell, w, h int) (edit, error) {
	if b.Value != 0 && b.Value != 1 {
		return nil, fmt.Errorf("mapedit: %d is not an impassable flag value", b.Value)
	}
	span := f.required(tileTypeID)
	old := e.tileWords(span, w*h)
	next := append([]uint16(nil), old...)
	for _, cell := range at {
		b.walk(cell, w, h, func(x, y, _ int) {
			i := y*w + x
			next[i] = old[i] &^ blockedBit
			if b.Value == 1 {
				next[i] |= blockedBit
			}
		})
	}
	return e.wordRuns(span, old, next), nil
}
