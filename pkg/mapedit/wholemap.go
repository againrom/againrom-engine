package mapedit

import "fmt"

// The two whole-map commands. Each is one edit over one plane, and each
// takes the caller's random source, Draw answering a number in 0..n-1, so one
// seed on one map gives one result.

// lookalikes lists the words that show what tile word w shows: its strip
// group and the corner state of the corner rule's table, every look-alike of
// that state in table order; nil for a word outside the table or a state with
// one word.
func lookalikes(w uint16) []uint16 {
	i := w & 0x3ff
	water := i&0x300 == 0x200
	g := i >> 6 & 15
	fg := g
	if water {
		fg = waterGroup
	}
	st, ok := stateOfWord[shapeKey{water, word{uint8(i >> 4 & 3), uint8(i & 15)}}]
	if _, known := groupFamily(fg); !ok || !known {
		return nil
	}
	k := 0
	if water {
		k = 1
	}
	ws := stateWords[k][st]
	if len(ws) < 2 {
		return nil
	}
	out := make([]uint16, len(ws))
	for n, x := range ws {
		out[n] = w&^0x3ff | compose(g, x)
	}
	return out
}

// draw is one answer of the caller's source, held inside 0..n-1.
func draw(src func(int) int, n int) int { return max(0, min(src(n), n-1)) }

// RandomizeTerrain draws every cell's tile word anew among the words that
// show the same corners in the same strip group, as the original's whole-map
// randomizing does (TERR-248): no cell changes its terrain, so no transition
// moves. Cells are drawn column by column, x outer and y inner, one draw per
// cell that has more than one look-alike; bit 13 and the word's other high
// bits stay. The plane then takes the save normalisation (tileEdit). Units,
// structures, objects and heights are not touched. It reports whether a byte
// changed.
func (e *Editor) RandomizeTerrain(src func(n int) int) (bool, error) {
	if src == nil {
		return false, fmt.Errorf("mapedit: random terrain needs a random source")
	}
	f := e.locate()
	w, h := e.GridSize()
	span := f.required(tileTypeID)
	old := e.tileWords(span, w*h)
	next := append([]uint16(nil), old...)
	for x := range w {
		for y := range h {
			i := y*w + x
			if ws := lookalikes(old[i]); ws != nil {
				next[i] = ws[draw(src, len(ws))]
			}
		}
	}
	ed := e.tileEdit(f, w, old, next, nil)
	if len(ed) == 0 {
		return false, nil
	}
	e.apply(ed)
	return true, nil
}

// The random height field: one hill or hollow per reliefArea cells, each a
// cosine height stroke of radius 3 to 8 and a change of 8 to 32, up or down.
const (
	reliefArea   = 64
	reliefRadius = 3
	reliefRadii  = 6
	reliefStep   = 8
	reliefSteps  = 25
)

// RandomizeHeights lays a random relief over the whole height plane: for
// each reliefArea cells one height stroke at a drawn cell, with a drawn
// radius and a drawn change up or down, each through the height brush's own
// rule (heights: cosine fall-off, clamp 0..127, row step 31), added to the
// heights the map has. The draws per stroke are x, y, radius, change, sign.
// Tiles, objects and every record are not touched. It reports whether a
// byte changed.
func (e *Editor) RandomizeHeights(src func(n int) int) (bool, error) {
	if src == nil {
		return false, fmt.Errorf("mapedit: random heights need a random source")
	}
	f := e.locate()
	w, h := e.GridSize()
	span := f.required(altitudeTypeID)
	old := append([]byte(nil), e.data[span.payloadOff:span.payloadOff+w*h]...)
	next := old
	for range max(1, w*h/reliefArea) {
		c := Cell{X: draw(src, w), Y: draw(src, h)}
		b := Brush{Plane: PlaneHeight, Radius: reliefRadius + draw(src, reliefRadii), Falloff: FalloffCosine}
		b.Value = reliefStep + draw(src, reliefSteps)
		if draw(src, 2) == 0 {
			b.Value = -b.Value
		}
		next = b.heights(next, []Cell{c}, w, h)
	}
	ed := byteRuns(span, old, next)
	if len(ed) == 0 {
		return false, nil
	}
	e.apply(ed)
	return true, nil
}
