package terrain

// The margin read: BorderCell over a derived block plane (AC-1, AC-2). Every
// plane below is built in test code from byte literals — nothing here
// decodes a map, reads an install or imports the tier that derives a real
// plane.

import "testing"

// planeGrid is a w*h Grid whose block plane is exactly the bytes given. Tiles
// is filled to the same extent so the value is a well-formed Grid, though
// BorderCell reads none of it.
func planeGrid(w, h int, block []uint8) Grid {
	return Grid{Width: w, Height: h, Tiles: make([]uint16, w*h), Block: block}
}

// ringPlane is a w*h block plane with bit 1 set on exactly the cells within d
// of an edge, and bit 0 set on every one of them too — which is the shape a
// real derived plane has, since a margin cell blocks both movers. Bit 0 is set
// on one interior cell as well, standing for interior water: BorderCell must
// not be reading "blocked at all".
func ringPlane(w, h, d int) []uint8 {
	p := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < d || y < d || x >= w-d || y >= h-d {
				p[y*w+x] = 0x03
			}
		}
	}
	if w > 2*d && h > 2*d {
		p[(h/2)*w+w/2] = 0x01
	}
	return p
}

func TestBorderCellIsBitOne(t *testing.T) {
	const w, h, d = 20, 19, 8
	plane := ringPlane(w, h, d)
	g := planeGrid(w, h, plane)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			want := plane[y*w+x]&0x02 != 0
			if got := g.BorderCell(x, y); got != want {
				t.Fatalf("BorderCell(%d,%d) = %v, want %v (byte %#02x)", x, y, got, want, plane[y*w+x])
			}
		}
	}

	// The ring the fixture built is the depth the plane's builder uses, so the
	// two halves of AC-1 are checked against each other rather than against one
	// restatement: an interior cell exists and is not border, and a cell one
	// step outside the interior is.
	if g.BorderCell(d, d) {
		t.Errorf("BorderCell(%d,%d) is the first interior cell and must not be border", d, d)
	}
	if !g.BorderCell(d-1, d) {
		t.Errorf("BorderCell(%d,%d) is the last margin cell and must be border", d-1, d)
	}
	if !g.BorderCell(w-d, d) {
		t.Errorf("BorderCell(%d,%d) is the far margin's first cell and must be border", w-d, d)
	}
}

// TestBorderCellWholeMapMargin — AC-1: a map no wider than twice the depth is
// margin throughout, with no interior cell at all. The plane says so and
// BorderCell reads it; nothing here special-cases the degenerate map, which is
// the point — the answer arrives as data, so a shape the reader never
// anticipated is still answered correctly.
func TestBorderCellWholeMapMargin(t *testing.T) {
	const w, h, d = 12, 30, 8
	g := planeGrid(w, h, ringPlane(w, h, d))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if !g.BorderCell(x, y) {
				t.Fatalf("BorderCell(%d,%d) = false on a %dx%d map, want every cell in the margin", x, y, w, h)
			}
		}
	}
}

func TestBorderCellIsTotal(t *testing.T) {
	full := ringPlane(20, 19, 8)

	cases := []struct {
		name     string
		g        Grid
		col, row int
	}{
		{"nil plane", planeGrid(20, 19, nil), 0, 0},
		{"nil plane, interior cell", planeGrid(20, 19, nil), 10, 10},
		{"empty plane", planeGrid(20, 19, []uint8{}), 0, 0},
		{"plane shorter than the extent", planeGrid(20, 19, full[:5]), 19, 18},
		{"negative column", planeGrid(20, 19, full), -1, 0},
		{"negative row", planeGrid(20, 19, full), 0, -1},
		{"column past the width", planeGrid(20, 19, full), 20, 0},
		{"row past the height", planeGrid(20, 19, full), 0, 19},
		{"zero width", Grid{Width: 0, Height: 19, Block: full}, 0, 0},
		{"negative height", Grid{Width: 20, Height: -1, Block: full}, 0, 0},
		{"zero value", Grid{}, 0, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.g.BorderCell(c.col, c.row); got {
				t.Fatalf("BorderCell(%d,%d) = true, want false", c.col, c.row)
			}
		})
	}
}

func TestBorderCellReadsRowMajor(t *testing.T) {
	const w, h = 7, 3
	plane := make([]uint8, w*h)
	plane[1*w+5] = blockAir // col 5, row 1

	g := planeGrid(w, h, plane)
	if !g.BorderCell(5, 1) {
		t.Errorf("BorderCell(5,1) = false, want true — the marked cell")
	}
	if g.BorderCell(1, 5) {
		t.Errorf("BorderCell(1,5) = true: (1,5) is outside a %dx%d grid", w, h)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x == 5 && y == 1 {
				continue
			}
			if g.BorderCell(x, y) {
				t.Fatalf("BorderCell(%d,%d) = true, want only (5,1)", x, y)
			}
		}
	}
}
