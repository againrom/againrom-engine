package mapedit_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// newHardCorners counts the inner map points below the top eight rows where
// after shows a hard corner (mapedit.HardCorner) and before did not; before
// nil counts every hard corner of after.
func newHardCorners(before, after []uint16, w, h int) int {
	n := 0
	for py := 9; py < h; py++ {
		for px := 1; px < w; px++ {
			if mapedit.HardCorner(after, w, px, py, testClass) && (before == nil || !mapedit.HardCorner(before, w, px, py, testClass)) {
				n++
			}
		}
	}
	return n
}

// hardCorners counts the hard corners of a plane below the top eight rows.
func hardCorners(tiles []uint16, w, h int) int { return newHardCorners(nil, tiles, w, h) }

// junctionStroke is one stroke on ed and what it leaves: the cells TileFix
// leaves with corners no word shows, the hard corners it adds, the farthest
// rewritten cell from the footprint and the number of rewritten cells.
type junctionStroke struct {
	seams, hard, reach, changed int
}

func strokeOn(t *testing.T, ed *mapedit.Editor, b mapedit.Brush, at []mapedit.Cell) junctionStroke {
	t.Helper()
	m, err := ed.Map()
	if err != nil {
		t.Fatal(err)
	}
	w, h := m.Width, m.Height
	fp := b.Footprint(at, w, h)
	var cells []int
	for _, c := range fp {
		cells = append(cells, c.Y*w+c.X)
	}
	fix := mapedit.NewTileFix(m.Tiles, w, h, testClass)
	fix.Paint(cells, uint8(b.Value))
	if _, err := ed.Paint(b, at, testClass); err != nil {
		t.Fatal(err)
	}
	after, err := ed.Map()
	if err != nil {
		t.Fatal(err)
	}
	out := junctionStroke{seams: fix.Seams, hard: newHardCorners(m.Tiles, after.Tiles, w, h)}
	for i := 8 * w; i < len(after.Tiles); i++ {
		if after.Tiles[i] == m.Tiles[i] {
			continue
		}
		out.changed++
		d := 1 << 30
		for _, c := range fp {
			d = min(d, max(abs(c.X-i%w), abs(c.Y-i/w)))
		}
		out.reach = max(out.reach, d)
	}
	return out
}

func square(c uint8, r int) mapedit.Brush {
	return mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(c), Radius: r, Shape: mapedit.BrushSquare}
}

func circle(c uint8, r int) mapedit.Brush {
	return mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(c), Radius: r}
}

// The owner's junctions, built by strokes on a Grass map: a Cracked area and
// a Stones area side by side, so the editor puts Land between each and the
// Grass and a Cracked-Stones strip between the two. Strokes of each meeting
// class across the junctions leave every cell a word for its corners and add
// no hard corner: Grass, Land, Cracked and Stones over the point where the
// Cracked-Stones seam meets the Land band, Land on the seam, and Cracked or
// Stones into the Grass beside it.
func TestStrokesAtTheOwnersJunctionsLeaveNoHardCorner(t *testing.T) {
	ed, err := mapedit.NewMap(alm.NewMapSpec{Width: 40, Height: 40, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	setup := []struct {
		b  mapedit.Brush
		at mapedit.Cell
	}{
		{square(mapedit.ClassCracked, 4), mapedit.Cell{X: 14, Y: 18}},
		{square(mapedit.ClassStones, 4), mapedit.Cell{X: 23, Y: 18}},
	}
	for _, s := range setup {
		if _, err := ed.Paint(s.b, []mapedit.Cell{s.at}, testClass); err != nil {
			t.Fatal(err)
		}
	}
	steps := []struct {
		what string
		b    mapedit.Brush
		at   []mapedit.Cell
	}{
		{"Grass across the seam's south end", circle(mapedit.ClassGrass, 2), []mapedit.Cell{{X: 18, Y: 24}}},
		{"Grass across the seam's north end", square(mapedit.ClassGrass, 1), []mapedit.Cell{{X: 19, Y: 13}}},
		{"Land on the seam", circle(mapedit.ClassLand, 0), []mapedit.Cell{{X: 18, Y: 18}}},
		{"Land at the band's corner", circle(mapedit.ClassLand, 1), []mapedit.Cell{{X: 10, Y: 23}}},
		{"Cracked into the Grass", circle(mapedit.ClassCracked, 1), []mapedit.Cell{{X: 12, Y: 26}}},
		{"Stones into the Grass", circle(mapedit.ClassStones, 2), []mapedit.Cell{{X: 26, Y: 25}}},
		{"Cracked into the Stones", circle(mapedit.ClassCracked, 1), []mapedit.Cell{{X: 22, Y: 15}}},
		{"Stones into the Cracked", circle(mapedit.ClassStones, 1), []mapedit.Cell{{X: 15, Y: 20}}},
		{"Grass along the seam", circle(mapedit.ClassGrass, 1), []mapedit.Cell{{X: 18, Y: 16}, {X: 18, Y: 19}, {X: 19, Y: 21}}},
		{"Land across the Grass band", square(mapedit.ClassLand, 2), []mapedit.Cell{{X: 17, Y: 11}}},
	}
	seams, hard := 0, 0
	for _, s := range steps {
		got := strokeOn(t, ed, s.b, s.at)
		t.Logf("%s: %d cells rewritten, reach %d, %d cells with no word, %d new hard corners", s.what, got.changed, got.reach, got.seams, got.hard)
		seams += got.seams
		hard += got.hard
	}
	if seams != 0 || hard != 0 {
		t.Errorf("the junction strokes leave %d cells with no word and %d new hard corners, want none", seams, hard)
	}
}

// The owner's bad edges as raw words, as a paste leaves them: pure Cracked
// words above a Grass-over-Land edge word, and pure Stones words beside pure
// Grass, meeting at one point. A stroke along each edge with one of the
// classes that meet there joins both sides: no cell is left without a word,
// no hard corner is added, and no hard corner stays on the stroke's own cells.
func TestStrokeOverTheOwnersBadEdgesRepairsThem(t *testing.T) {
	cracked := []uint16{0x074, 0x074, 0x051, 0x071, 0x071}
	stones := []uint16{0x133, 0x131, 0x111, 0x135, 0x134, 0x130}
	const side = 40
	plane := func() *mapedit.Editor {
		ed, err := mapedit.NewMap(alm.NewMapSpec{Width: side, Height: side, Tile: 0x11, Altitude: 64})
		if err != nil {
			t.Fatal(err)
		}
		for y := 8; y < side-1; y++ {
			for x := 1; x < side-1; x++ {
				var w uint16 = 0x11
				switch {
				case x >= 22:
					w = stones[(x+y)%len(stones)]
				case y < 20:
					w = cracked[(x+y)%len(cracked)]
				case y == 20:
					w = 0x012
				}
				if err := ed.SetTile(x, y, w); err != nil {
					t.Fatal(err)
				}
			}
		}
		return ed
	}
	var row, col []mapedit.Cell
	for x := 6; x <= 21; x++ {
		row = append(row, mapedit.Cell{X: x, Y: 20})
	}
	for y := 20; y <= 32; y++ {
		col = append(col, mapedit.Cell{X: 21, Y: y})
	}
	for _, c := range []uint8{mapedit.ClassGrass, mapedit.ClassLand, mapedit.ClassCracked, mapedit.ClassStones} {
		for _, path := range []struct {
			what string
			at   []mapedit.Cell
		}{{"the Cracked edge", row}, {"the Stones edge", col}} {
			for _, r := range []int{1, 2} {
				ed := plane()
				b := circle(c, r)
				before, _ := ed.Map()
				got := strokeOn(t, ed, b, path.at)
				after, _ := ed.Map()
				was, is := hardCorners(before.Tiles, side, side), hardCorners(after.Tiles, side, side)
				what := fmt.Sprintf("class %d along %s, size %d", c, path.what, r)
				t.Logf("%s: %d cells rewritten, reach %d, hard corners %d -> %d", what, got.changed, got.reach, was, is)
				if got.seams != 0 || got.hard != 0 || is >= was {
					t.Errorf("%s: %d cells with no word, %d new hard corners, hard corners %d -> %d", what, got.seams, got.hard, was, is)
				}
			}
		}
	}
}

// A seeded run of random strokes of every palette class, sizes 0 to 4, over a
// map that holds all of them leaves no cell without a word and adds no hard
// corner.
func TestRandomStrokesOverEveryClassLeaveNoHardCorner(t *testing.T) {
	const side = 72
	ed, err := mapedit.NewMap(alm.NewMapSpec{Width: side, Height: side, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	for k, p := range mapedit.Palette {
		at := mapedit.Cell{X: 14 + 11*(k%5), Y: 22 + 14*(k/5)}
		if _, err := ed.Paint(square(p.Class, 5), []mapedit.Cell{at}, testClass); err != nil {
			t.Fatal(err)
		}
	}
	rng := rand.New(rand.NewPCG(3, 17))
	seams, hard, seamStrokes, reach, most := 0, 0, 0, 0, 0
	for step := range 400 {
		c := mapedit.Palette[rng.IntN(len(mapedit.Palette))].Class
		b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(c), Radius: rng.IntN(5), Shape: mapedit.BrushShape(rng.IntN(2))}
		at := []mapedit.Cell{{X: 10 + rng.IntN(side-20), Y: 18 + rng.IntN(side-28)}}
		for range rng.IntN(3) {
			last := at[len(at)-1]
			at = append(at, mapedit.Cell{X: last.X + rng.IntN(5) - 2, Y: last.Y + rng.IntN(5) - 2})
		}
		got := strokeOn(t, ed, b, at)
		if got.seams > 0 || got.hard > 0 {
			seamStrokes++
			if seamStrokes <= 5 {
				t.Logf("stroke %d (class %d at %v, size %d, shape %d): %d cells with no word, %d new hard corners", step, c, at, b.Radius, b.Shape, got.seams, got.hard)
			}
		}
		seams += got.seams
		hard += got.hard
		reach = max(reach, got.reach)
		most = max(most, got.changed)
	}
	t.Logf("400 strokes: %d cells with no word and %d new hard corners in %d strokes; farthest rewrite %d cells, most cells rewritten %d", seams, hard, seamStrokes, reach, most)
	if seams != 0 || hard != 0 {
		t.Errorf("random strokes leave %d cells with no word and %d new hard corners in %d strokes, want none", seams, hard, seamStrokes)
	}
}

// A Grass stroke of size 8 over the top rows of a Land map (word 0x114):
// TileFix leaves no hard corner, then the save normalisation writes the top
// eight rows back to the last cell's word, so the hard corners lie on row
// 8's top edge and none below.
func TestStrokeOverTheTopRowsLeavesHardCornersOnlyAtTheSavedBand(t *testing.T) {
	const land = 0x114
	ed, err := mapedit.New(brushMap(land, 64))
	if err != nil {
		t.Fatal(err)
	}
	before, err := ed.Map()
	if err != nil {
		t.Fatal(err)
	}
	b := mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassGrass), Radius: 8}
	at := []mapedit.Cell{{X: 3, Y: 0}}
	var cells []int
	for _, c := range b.Footprint(at, brushSide, brushSide) {
		cells = append(cells, c.Y*brushSide+c.X)
	}
	fix := mapedit.NewTileFix(before.Tiles, brushSide, brushSide, testClass)
	fixed := append([]uint16(nil), before.Tiles...)
	for _, ch := range fix.Paint(cells, mapedit.ClassGrass) {
		fixed[ch.Cell] = ch.Word
	}
	if _, err := ed.Paint(b, at, testClass); err != nil {
		t.Fatal(err)
	}
	after, err := ed.Map()
	if err != nil {
		t.Fatal(err)
	}
	band, fixBand := 0, 0
	for px := 1; px < brushSide; px++ {
		band += b2i(mapedit.HardCorner(after.Tiles, brushSide, px, 8, testClass))
		fixBand += b2i(mapedit.HardCorner(fixed, brushSide, px, 8, testClass))
	}
	for i := range 8 * brushSide {
		if after.Tiles[i] != land {
			t.Fatalf("cell %d of the top rows holds %#x, want the last cell's word", i, after.Tiles[i])
		}
	}
	below, fixBelow := newHardCorners(before.Tiles, after.Tiles, brushSide, brushSide), newHardCorners(before.Tiles, fixed, brushSide, brushSide)
	t.Logf("hard corners at row 8: %d after the save normalisation, %d from TileFix alone; below: %d and %d", band, fixBand, below, fixBelow)
	if band == 0 || fixBand != 0 || below != 0 || fixBelow != 0 || fix.Seams != 0 {
		t.Errorf("row 8: %d hard corners after the stroke, %d from TileFix; below row 8: %d and %d; %d cells with no word", band, fixBand, below, fixBelow, fix.Seams)
	}
}
