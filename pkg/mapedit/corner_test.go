package mapedit_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"againrom/pkg/mapedit"
)

func abs(v int) int { return max(v, -v) }

// quadOf reads the terrain class at the four corners of a tile word (NW, NE,
// SW, SE) from its shape alone: a pure word shows its class everywhere, an
// edge the strip's primary class on the far side from Dir, an inner corner
// everywhere but the corner Dir names, an outer corner (water) and a tip only
// at one corner. The two classes are the strip pair testClass reads, Water
// and Land for a water word.
func quadOf(w uint16) [4]uint8 {
	c := testClass(w)
	s, g, ok := mapedit.WordShape(w)
	if !ok || s.Kind == mapedit.ShapePure {
		return [4]uint8{c, c, c, c}
	}
	pairs := map[uint16][2]uint8{
		0: {mapedit.ClassGrass, mapedit.ClassLand}, 1: {mapedit.ClassCracked, mapedit.ClassLand},
		2: {mapedit.ClassSand, mapedit.ClassLand}, 3: {mapedit.ClassSavanna, mapedit.ClassLand},
		4: {mapedit.ClassStones, mapedit.ClassLand}, 5: {mapedit.ClassCracked, mapedit.ClassStones},
		6: {mapedit.ClassFlowers, mapedit.ClassSavanna}, 7: {mapedit.ClassMountain, mapedit.ClassStones},
		8: {mapedit.ClassWater, mapedit.ClassLand}, 12: {mapedit.ClassRoad, mapedit.ClassLand},
	}
	p := pairs[g]
	var primary [4]bool // corners NW NE SW SE
	switch s.Kind {
	case mapedit.ShapeEdge: // Dir N E S W names the side the other class lies on
		far := [4][2]int{{2, 3}, {0, 2}, {0, 1}, {1, 3}}[s.Dir]
		primary[far[0]], primary[far[1]] = true, true
	case mapedit.ShapeInner: // Dir NW NE SW SE names the other class's corner
		primary = [4]bool{true, true, true, true}
		primary[s.Dir] = false
	case mapedit.ShapeOuter: // water at the corner opposite Dir
		primary[3-s.Dir] = true
	case mapedit.ShapeTip: // the primary class at the corner Dir names
		primary[s.Dir] = true
	}
	var q [4]uint8
	for k := range q {
		q[k] = p[1]
		if primary[k] {
			q[k] = p[0]
		}
	}
	return q
}

// seams counts the corners two neighbouring cells of a side x side plane show
// in different classes: a hard edge on the drawn map.
func seams(tiles []uint16, side int) int {
	return seamsFrom(tiles, side, 0)
}

// seenQuad is quadOf with Flowers read as Grass: the installed maps join the
// two at a shared corner with no strip between them.
func seenQuad(w uint16) [4]uint8 {
	q := quadOf(w)
	for k := range q {
		if q[k] == mapedit.ClassFlowers {
			q[k] = mapedit.ClassGrass
		}
	}
	return q
}

// seamsFrom is seams over the rows from top down.
func seamsFrom(tiles []uint16, side, top int) int {
	n := 0
	for y := top; y < side; y++ {
		for x := 0; x < side; x++ {
			q := seenQuad(tiles[y*side+x])
			if x+1 < side {
				r := seenQuad(tiles[y*side+x+1])
				n += b2i(q[1] != r[0]) + b2i(q[3] != r[2])
			}
			if y+1 < side {
				d := seenQuad(tiles[(y+1)*side+x])
				n += b2i(q[2] != d[0]) + b2i(q[3] != d[1])
			}
			if x+1 < side && y+1 < side {
				n += b2i(q[3] != seenQuad(tiles[(y+1)*side+x+1])[0])
			}
			if x > 0 && y+1 < side {
				n += b2i(q[2] != seenQuad(tiles[(y+1)*side+x-1])[1])
			}
		}
	}
	return n
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func paletteWord(c uint8) uint16 {
	for _, p := range mapedit.Palette {
		if p.Class == c {
			return p.Word
		}
	}
	panic(fmt.Sprintf("no palette class %d", c))
}

// A stroke of every paintable class, at sizes 0, 1 and 3, as a circle and as
// a square, over a map of every other class leaves no corner on which two
// neighbouring cells disagree, and the painted cells show their class at all
// four corners. The stroke stays clear of the top eight rows, which the save
// normalisation rewrites whole.
func TestStrokeOfEachClassPairLeavesNoSeam(t *testing.T) {
	for _, fill := range mapedit.Palette {
		for _, paint := range mapedit.Palette {
			for _, r := range []int{0, 1, 3} {
				for _, shape := range []mapedit.BrushShape{mapedit.BrushCircle, mapedit.BrushSquare} {
					ed, _ := mapedit.New(brushMap(fill.Word, 40))
					b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(paint.Class), Radius: r, Shape: shape}
					if _, err := ed.Paint(b, []mapedit.Cell{{X: 12, Y: 15}}, testClass); err != nil {
						t.Fatal(err)
					}
					tiles, _, _ := planes(t, ed.Bytes())
					what := fmt.Sprintf("%s over %s, size %d, shape %d", paint.Name, fill.Name, r, shape)
					if n := seams(tiles, brushSide); n != 0 {
						t.Errorf("%s: %d corners disagree", what, n)
					}
					c := paint.Class
					for _, cell := range b.Footprint([]mapedit.Cell{{X: 12, Y: 15}}, brushSide, brushSide) {
						if q := quadOf(tiles[cell.Y*brushSide+cell.X]); q != [4]uint8{c, c, c, c} {
							t.Fatalf("%s: painted cell %v shows %v", what, cell, q)
						}
					}
				}
			}
		}
	}
}

// Where no strip joins the painted class to its neighbour's, the cells around
// the stroke take the inserted class (TERR-243 base: Stones beside Land and
// Mountain, Land beside every other class), and the classes beyond it join
// through their own strips. Flowers' base, Grass, shares no strip with
// Flowers in the installed tile set, so Savanna, Flowers' one strip partner,
// is inserted instead.
func TestStrokeInsertsTheBaseClassWhereNoStripJoins(t *testing.T) {
	for _, tc := range []struct {
		fill, paint, insert uint8
	}{
		{mapedit.ClassStones, mapedit.ClassSand, mapedit.ClassLand},
		{mapedit.ClassWater, mapedit.ClassSand, mapedit.ClassLand},
		{mapedit.ClassSand, mapedit.ClassWater, mapedit.ClassLand},
		{mapedit.ClassLand, mapedit.ClassMountain, mapedit.ClassStones},
		{mapedit.ClassMountain, mapedit.ClassLand, mapedit.ClassStones},
		{mapedit.ClassGrass, mapedit.ClassMountain, mapedit.ClassStones},
		{mapedit.ClassMountain, mapedit.ClassCracked, mapedit.ClassStones},
		{mapedit.ClassLand, mapedit.ClassFlowers, mapedit.ClassLand},
		{mapedit.ClassGrass, mapedit.ClassFlowers, mapedit.ClassGrass},
		{mapedit.ClassSand, mapedit.ClassGrass, mapedit.ClassLand},
	} {
		ed, _ := mapedit.New(brushMap(paletteWord(tc.fill), 40))
		if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(tc.paint)}, []mapedit.Cell{{X: 12, Y: 12}}, testClass); err != nil {
			t.Fatal(err)
		}
		tiles, _, _ := planes(t, ed.Bytes())
		// The east neighbour shows the painted class on its west corners and
		// the inserted class on its east corners.
		p, i := tc.paint, tc.insert
		if p == mapedit.ClassFlowers {
			p = mapedit.ClassGrass
		}
		if q := seenQuad(tiles[12*brushSide+13]); q != [4]uint8{p, i, p, i} {
			t.Errorf("class %d over %d: the east neighbour shows %v, want %v", p, tc.fill, q, [4]uint8{p, i, p, i})
		}
		if n := seams(tiles, brushSide); n != 0 {
			t.Errorf("class %d over %d: %d corners disagree", p, tc.fill, n)
		}
	}
}

// strokeOutcome paints b through at on ed and reports the tiles before and
// after, the cells TileFix leaves with the nearest word, and the farthest
// rewritten cell from the footprint below the top rows the save
// normalisation rewrites whole.
func strokeOutcome(t *testing.T, ed *mapedit.Editor, b mapedit.Brush, at []mapedit.Cell) (before, after []uint16, nearest, reach int) {
	t.Helper()
	before, _, _ = planes(t, ed.Bytes())
	fp := b.Footprint(at, brushSide, brushSide)
	var cells []int
	for _, c := range fp {
		cells = append(cells, c.Y*brushSide+c.X)
	}
	fix := mapedit.NewTileFix(before, brushSide, brushSide, testClass)
	fix.Paint(cells, uint8(b.Value))
	if _, err := ed.Paint(b, at, testClass); err != nil {
		t.Fatal(err)
	}
	after, _, _ = planes(t, ed.Bytes())
	for i := 8 * brushSide; i < len(after); i++ {
		if after[i] == before[i] {
			continue
		}
		d := 1 << 30
		for _, c := range fp {
			d = min(d, max(abs(c.X-i%brushSide), abs(c.Y-i/brushSide)))
		}
		reach = max(reach, d)
	}
	return before, after, fix.Seams, reach
}

// Strokes of three classes meeting at one place, and a run of random strokes,
// rewrite no cell more than Reach+1 cells from the brush. A stroke adds a
// hard corner below the top rows only where the bound stops the rule at a
// three-class junction, and TileFix counts each such cell; the scripted steps
// leave the junction of Sand, Stones and Cracked as two hard corners.
func TestStrokesAcrossThreeClassesStayNearTheBrush(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 40))
	steps := []struct {
		c    uint8
		r    int
		at   []mapedit.Cell
		what string
		hard int
	}{
		{mapedit.ClassSand, 3, []mapedit.Cell{{X: 9, Y: 12}}, "sand into land", 0},
		{mapedit.ClassStones, 3, []mapedit.Cell{{X: 15, Y: 12}}, "land into stones beside sand", 0},
		{mapedit.ClassMountain, 2, []mapedit.Cell{{X: 12, Y: 17}}, "sand into mountain", 0},
		{mapedit.ClassCracked, 1, []mapedit.Cell{{X: 12, Y: 12}}, "a three-class corner", 6},
	}
	for _, s := range steps {
		b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(s.c), Radius: s.r}
		_, tiles, nearest, reach := strokeOutcome(t, ed, b, s.at)
		if n := seams(tiles, brushSide); n != s.hard || reach > mapedit.Reach+1 || (nearest == 0) != (s.hard == 0) {
			t.Errorf("%s: %d corners disagree (want %d), %d cells take the nearest word, reach %d", s.what, n, s.hard, nearest, reach)
		}
	}
	rng := rand.New(rand.NewPCG(7, 11))
	nearestStrokes := 0
	for step := range 200 {
		c := mapedit.Palette[rng.IntN(len(mapedit.Palette))].Class
		b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(c), Radius: rng.IntN(3), Shape: mapedit.BrushShape(rng.IntN(2))}
		at := []mapedit.Cell{{X: 6 + rng.IntN(12), Y: 12 + rng.IntN(6)}}
		at = append(at, mapedit.Cell{X: at[0].X + rng.IntN(3) - 1, Y: at[0].Y + rng.IntN(3) - 1})
		before, after, nearest, reach := strokeOutcome(t, ed, b, at)
		if nearest > 0 {
			nearestStrokes++
		}
		was, is := seamsFrom(before, brushSide, 8), seamsFrom(after, brushSide, 8)
		if reach > mapedit.Reach+1 || nearest == 0 && is > was {
			t.Fatalf("random stroke %d (class %d at %v, size %d, shape %d): reach %d, corners disagreeing %d -> %d, %d cells take the nearest word",
				step, c, at, b.Radius, b.Shape, reach, was, is, nearest)
		}
	}
	t.Logf("random strokes leaving cells with the nearest word: %d of 200", nearestStrokes)
}

// A one-cell Cracked stroke on the edge of a Stones band that crosses a Land
// map meets three classes at one corner: Cracked joins both, Stones and Land
// join each other, and no word shows all three. Without the bound the rule
// would move one corner after another along the seam to the map's edge. The
// stroke rewrites no cell more than Reach+1 cells from the painted cell, and
// the cell where the bound stops the rule takes the nearest word.
func TestThreeClassJunctionStaysNearTheStroke(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 40))
	band := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassStones), Radius: 3, Shape: mapedit.BrushSquare}
	for _, x := range []int{3, 9, 15, 20} {
		if _, err := ed.Paint(band, []mapedit.Cell{{X: x, Y: 15}}, testClass); err != nil {
			t.Fatal(err)
		}
	}
	click := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassCracked)}
	before, after, nearest, reach := strokeOutcome(t, ed, click, []mapedit.Cell{{X: 6, Y: 12}})
	changed := 0
	for i := range after {
		if after[i] != before[i] {
			changed++
		}
	}
	if reach > mapedit.Reach+1 || nearest == 0 || testClass(after[12*brushSide+6]) != mapedit.ClassCracked {
		t.Errorf("Cracked on the Stones seam: %d cells rewritten, reach %d (bound %d), %d cells take the nearest word, painted cell shows %#x",
			changed, reach, mapedit.Reach+1, nearest, after[12*brushSide+6])
	}
}

// The brush's cells: a circle holds the cells within its radius of the
// centre, a square every cell within its radius on both axes, for sizes 0, 1,
// 3 and the largest; both are cut at the grid's edge.
func TestBrushShapeCells(t *testing.T) {
	const side = 64
	for _, r := range []int{0, 1, 3, mapedit.MaxRadius} {
		for _, shape := range []mapedit.BrushShape{mapedit.BrushCircle, mapedit.BrushSquare} {
			var want []mapedit.Cell
			for y := 32 - r; y <= 32+r; y++ {
				for x := 32 - r; x <= 32+r; x++ {
					dx, dy := x-32, y-32
					if shape == mapedit.BrushSquare || dx*dx+dy*dy <= r*r {
						want = append(want, mapedit.Cell{X: x, Y: y})
					}
				}
			}
			got := mapedit.Brush{Radius: r, Shape: shape}.Footprint([]mapedit.Cell{{X: 32, Y: 32}}, side, side)
			if !slices.Equal(got, want) {
				t.Errorf("size %d shape %d: %d cells, want %d", r, shape, len(got), len(want))
			}
		}
	}
	for r, want := range map[int][2]int{0: {1, 1}, 1: {5, 9}, 3: {29, 49}, mapedit.MaxRadius: {797, 1089}} {
		for shape, n := range want {
			if got := len(mapedit.Brush{Radius: r, Shape: mapedit.BrushShape(shape)}.Footprint([]mapedit.Cell{{X: 32, Y: 32}}, side, side)); got != n {
				t.Errorf("size %d shape %d: %d cells, want %d", r, shape, got, n)
			}
		}
	}
	corner := mapedit.Brush{Radius: 3, Shape: mapedit.BrushSquare}.Footprint([]mapedit.Cell{{X: 0, Y: 0}}, side, side)
	if len(corner) != 16 {
		t.Errorf("a square of size 3 at the grid's corner covers %d cells, want 16", len(corner))
	}
}

// A square height stroke fades by its square rings: a cell on a ring's
// corner rises as much as the cell at the middle of the ring's side. A
// circle stroke of the same size leaves the square's corner cells alone.
func TestSquareHeightStrokeFadesOnTheSquareDistance(t *testing.T) {
	stroke := func(shape mapedit.BrushShape) []byte {
		ed, _ := mapedit.New(brushMap(0x14, 20))
		b := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 64, Radius: 3, Shape: shape, Falloff: mapedit.FalloffCosine}
		if _, err := ed.Paint(b, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
			t.Fatal(err)
		}
		_, alts, _ := planes(t, ed.Bytes())
		return alts
	}
	at := func(alts []byte, dx, dy int) int { return int(alts[(12+dy)*brushSide+12+dx]) }
	sq := stroke(mapedit.BrushSquare)
	for k := 1; k <= 3; k++ {
		if at(sq, k, k) != at(sq, k, 0) || at(sq, -k, k) != at(sq, 0, -k) {
			t.Errorf("ring %d: corner %d, side %d", k, at(sq, k, k), at(sq, k, 0))
		}
	}
	if at(sq, 3, 3) <= 20 || at(sq, 4, 0) != 20 {
		t.Errorf("square: rim corner %d, one past the rim %d", at(sq, 3, 3), at(sq, 4, 0))
	}
	ci := stroke(mapedit.BrushCircle)
	if at(ci, 3, 3) != 20 || at(ci, 3, 0) != at(sq, 3, 0) || at(ci, 0, 0) != at(sq, 0, 0) {
		t.Errorf("circle: corner %d, rim %d, centre %d", at(ci, 3, 3), at(ci, 3, 0), at(ci, 0, 0))
	}
}

// A brush shape the model does not know is refused.
func TestUnknownBrushShapeIsRefused(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 40))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 4, Shape: 2}, []mapedit.Cell{{X: 1, Y: 1}}, nil); err == nil || ed.CanUndo() {
		t.Fatalf("shape 2: err %v", err)
	}
}
