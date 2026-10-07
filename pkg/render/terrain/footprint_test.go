package terrain_test

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// The file's one answer to "where is this cell": the footprint itself, clipped,
// and rejected on the same two conditions every glyph builder in the family
// rejects on — and then the two rims that border it.
func TestCellFootprint(t *testing.T) {
	t.Parallel()

	const cols, rows = 4, 3

	t.Run("an in-map cell is its own footprint", func(t *testing.T) {
		got, ok := terrain.CellFootprint(1, 2, cols, rows, terrain.CellSize)
		if !ok {
			t.Fatal("an in-map cell was rejected")
		}
		want := image.Rect(1*terrain.CellSize, 2*terrain.CellSize,
			2*terrain.CellSize, 3*terrain.CellSize)
		if got != want {
			t.Errorf("rect %v, want %v", got, want)
		}
	})

	t.Run("it tiles the map exactly", func(t *testing.T) {
		// Every cell's rect is disjoint from every other's and their union is
		// the map rectangle — which is what makes a wash over a set of cells a
		// statement about exactly those cells.
		covered := map[image.Point]int{}
		for row := 0; row < rows; row++ {
			for col := 0; col < cols; col++ {
				r, ok := terrain.CellFootprint(col, row, cols, rows, terrain.CellSize)
				if !ok {
					t.Fatalf("cell (%d,%d) was rejected", col, row)
				}
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						covered[image.Pt(x, y)]++
					}
				}
			}
		}
		if len(covered) != cols*rows*terrain.CellSize*terrain.CellSize {
			t.Errorf("the rects cover %d pixels, want the whole map's %d",
				len(covered), cols*rows*terrain.CellSize*terrain.CellSize)
		}
		for p, n := range covered {
			if n != 1 {
				t.Fatalf("pixel %v is covered %d times", p, n)
			}
		}
	})

	t.Run("an off-map cell and an invalid scale are rejected", func(t *testing.T) {
		for _, tc := range []struct{ col, row, cellpx int }{
			{-1, 0, terrain.CellSize},
			{0, -1, terrain.CellSize},
			{cols, 0, terrain.CellSize},
			{0, rows, terrain.CellSize},
			{0, 0, 0},
			{0, 0, -3},
		} {
			if r, ok := terrain.CellFootprint(tc.col, tc.row, cols, rows, tc.cellpx); ok {
				t.Errorf("cell (%d,%d) at scale %d yielded %v", tc.col, tc.row, tc.cellpx, r)
			}
		}
	})
}

// The two washes are TRANSLUCENT and PREMULTIPLIED. Both halves are
// load-bearing: opaque, a wash would hide whatever it is drawn over, and a
// channel above the alpha is not a paler colour but an invalid one.
func TestTheWashesArePremultiplied(t *testing.T) {
	t.Parallel()

	for name, c := range map[string]color.RGBA{
		"BlockedCellColor": terrain.BlockedCellColor,
		"CellGridColor":    terrain.CellGridColor,
	} {
		if c.A == 0 || c.A == 0xff {
			t.Errorf("%s alpha %#02x — the wash must be translucent", name, c.A)
		}
		for ch, v := range map[string]uint8{"R": c.R, "G": c.G, "B": c.B} {
			if v > c.A {
				t.Errorf("%s channel %s is %#02x above the alpha %#02x — color.RGBA is "+
					"premultiplied, so this is not a paler colour, it is an invalid one",
					name, ch, v, c.A)
			}
		}
	}
}

func TestCellGridRects(t *testing.T) {
	t.Parallel()

	const cols, rows = 4, 3

	t.Run("it is the footprint's border at one native pixel", func(t *testing.T) {
		got := terrain.CellGridRects(1, 2, cols, rows, terrain.CellSize)
		x0, y0 := 1*terrain.CellSize, 2*terrain.CellSize
		x1, y1 := x0+terrain.CellSize, y0+terrain.CellSize
		want := []image.Rectangle{
			image.Rect(x0, y0, x1, y0+1),
			image.Rect(x0, y1-1, x1, y1),
			image.Rect(x0, y0+1, x0+1, y1-1),
			image.Rect(x1-1, y0+1, x1, y1-1),
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("outline = %v, want %v", got, want)
		}
	})

	t.Run("it stands on the very rectangle the rim stands on", func(t *testing.T) {
		// The union of each glyph's strips has the footprint's own bounds. If
		// the two ever came off different rectangles this is the assertion that
		// would break, and it compares the builders' output rather than
		// re-deriving either.
		for _, cellpx := range []int{terrain.CellSize, 2 * terrain.CellSize, 8, 3} {
			for row := 0; row < rows; row++ {
				for col := 0; col < cols; col++ {
					foot, ok := terrain.CellFootprint(col, row, cols, rows, cellpx)
					if !ok {
						t.Fatalf("cell (%d,%d) at %d was rejected", col, row, cellpx)
					}
					for name, rects := range map[string][]image.Rectangle{
						"grid": terrain.CellGridRects(col, row, cols, rows, cellpx),
						"rim":  terrain.SelectionMarkerRects(col, row, cols, rows, cellpx),
					} {
						if len(rects) == 0 {
							t.Fatalf("%s produced nothing on cell (%d,%d) at %d", name, col, row, cellpx)
						}
						union := rects[0]
						for _, r := range rects[1:] {
							union = union.Union(r)
						}
						if union != foot {
							t.Errorf("%s on cell (%d,%d) at %d spans %v, want the footprint %v",
								name, col, row, cellpx, union, foot)
						}
					}
				}
			}
		}
	})

	t.Run("the outline is a strict subset of the rim at every scale", func(t *testing.T) {
		// Which is what lets the lattice run first and the rim last without
		// either hiding the other: one line inside the other, never beside it.
		for _, cellpx := range []int{terrain.CellSize, 2 * terrain.CellSize, 64, 16} {
			rim := map[image.Point]bool{}
			for _, r := range terrain.SelectionMarkerRects(1, 1, cols, rows, cellpx) {
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						rim[image.Pt(x, y)] = true
					}
				}
			}
			for _, r := range terrain.CellGridRects(1, 1, cols, rows, cellpx) {
				for y := r.Min.Y; y < r.Max.Y; y++ {
					for x := r.Min.X; x < r.Max.X; x++ {
						if !rim[image.Pt(x, y)] {
							t.Fatalf("at %d the outline covers %v, which the rim does not",
								cellpx, image.Pt(x, y))
						}
					}
				}
			}
		}
	})

	t.Run("an off-map cell and an invalid scale draw nothing", func(t *testing.T) {
		for _, tc := range []struct{ col, row, cellpx int }{
			{-1, 0, terrain.CellSize},
			{0, -1, terrain.CellSize},
			{cols, 0, terrain.CellSize},
			{0, rows, terrain.CellSize},
			{0, 0, 0},
		} {
			if got := terrain.CellGridRects(tc.col, tc.row, cols, rows, tc.cellpx); got != nil {
				t.Errorf("cell (%d,%d) at scale %d yielded %v", tc.col, tc.row, tc.cellpx, got)
			}
		}
	})
}
