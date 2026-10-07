package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestDrawNinePatchBorderTilesFourDistinctEdges(t *testing.T) {
	const cw, ch = 8, 4
	// The border source: each of its eight nine-patch regions is a distinct
	// solid colour so a destination pixel's colour alone says which region
	// the blit read from.
	src := image.NewRGBA(image.Rect(0, 0, 24, 12))
	tl := color.RGBA{R: 0x10, A: 0xff}
	tr := color.RGBA{R: 0x20, A: 0xff}
	bl := color.RGBA{R: 0x30, A: 0xff}
	br := color.RGBA{R: 0x40, A: 0xff}
	top := color.RGBA{R: 0x50, A: 0xff}
	bottom := color.RGBA{R: 0x60, A: 0xff}
	left := color.RGBA{R: 0x70, A: 0xff}
	right := color.RGBA{R: 0x80, A: 0xff}
	fill := func(r image.Rectangle, c color.RGBA) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				src.SetRGBA(x, y, c)
			}
		}
	}
	fill(image.Rect(0, 0, cw, ch), tl)
	fill(image.Rect(16, 0, 24, ch), tr)
	fill(image.Rect(0, 8, cw, 12), bl)
	fill(image.Rect(16, 8, 24, 12), br)
	fill(image.Rect(cw, 0, 16, ch), top)
	fill(image.Rect(cw, 8, 16, 12), bottom)
	fill(image.Rect(0, ch, cw, 8), left)
	fill(image.Rect(16, ch, 24, 8), right)

	// The panel: 30 wide, 20 tall, at a non-zero origin so a mutation that
	// used the border's own coordinates instead of the panel's would also be
	// caught. Interior rows/columns exist on every edge (30 > 2*8, 20 > 2*4).
	panel := image.Rect(5, 5, 35, 25)
	dst := image.NewRGBA(image.Rect(0, 0, 40, 30))
	drawNinePatchBorder(dst, panel, src, cw, ch)

	at := func(p image.Point) color.RGBA { return dst.RGBAAt(p.X, p.Y) }

	// Corners: unscaled, exactly cw x ch at each of the panel's own four
	// corners.
	if c := at(image.Pt(5, 5)); c != tl {
		t.Errorf("top-left corner (5,5) = %#v, want %#v", c, tl)
	}
	if c := at(image.Pt(34, 5)); c != tr {
		t.Errorf("top-right corner (34,5) = %#v, want %#v", c, tr)
	}
	if c := at(image.Pt(5, 24)); c != bl {
		t.Errorf("bottom-left corner (5,24) = %#v, want %#v", c, bl)
	}
	if c := at(image.Pt(34, 24)); c != br {
		t.Errorf("bottom-right corner (34,24) = %#v, want %#v", c, br)
	}

	// The left tile: THE MUTATION SITE (tippanel.go:497). Its destination is
	// panel.Min.X .. panel.Min.X+cw, panel.Min.Y+ch .. panel.Max.Y-ch — the
	// strip between the top-left and bottom-left corners. Boundary pixels on
	// both sides prove the exact extent, not just that some pixel inside is
	// right.
	if c := at(image.Pt(5, 8)); c != tl {
		t.Errorf("(5,8), one row above the left tile's own top edge, = %#v, want the corner colour %#v", c, tl)
	}
	if c := at(image.Pt(5, 9)); c != left {
		t.Errorf("(5,9), the left tile's own first row, = %#v, want %#v", c, left)
	}
	if c := at(image.Pt(5, 20)); c != left {
		t.Errorf("(5,20), the left tile's own last row, = %#v, want %#v", c, left)
	}
	if c := at(image.Pt(5, 21)); c != bl {
		t.Errorf("(5,21), one row below the left tile's own bottom edge, = %#v, want the corner colour %#v", c, bl)
	}
	if c := at(image.Pt(15, 15)); c != (color.RGBA{}) {
		t.Errorf("(15,15), inside the panel, past the left tile's own %d-wide column and short of the right tile, = %#v, want untouched (nine-patch draws no centre)", cw, c)
	}

	// The right tile, top and bottom tiles: same boundary shape, one
	// representative interior pixel each.
	if c := at(image.Pt(33, 15)); c != right {
		t.Errorf("right tile interior (33,15) = %#v, want %#v", c, right)
	}
	if c := at(image.Pt(20, 5)); c != top {
		t.Errorf("top tile interior (20,5) = %#v, want %#v", c, top)
	}
	if c := at(image.Pt(20, 24)); c != bottom {
		t.Errorf("bottom tile interior (20,24) = %#v, want %#v", c, bottom)
	}
}

// TestDrawNinePatchBorderRefusesADegenerateCornerOrPanel is the function's
// own early-return guard, exercised so a caller passing an unfit border or a
// panel too small for it is a documented no-op rather than an untested
// branch.
func TestDrawNinePatchBorderRefusesADegenerateCornerOrPanel(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	dst := image.NewRGBA(image.Rect(0, 0, 20, 20))
	before := append([]byte(nil), dst.Pix...)

	drawNinePatchBorder(dst, image.Rect(0, 0, 20, 20), nil, 2, 2)
	drawNinePatchBorder(dst, image.Rect(0, 0, 20, 20), src, 0, 2)
	drawNinePatchBorder(dst, image.Rect(0, 0, 1, 1), src, 2, 2)

	for i, b := range dst.Pix {
		if b != before[i] {
			t.Fatalf("a degenerate call painted pixel byte %d: got %d, want untouched %d", i, b, before[i])
		}
	}
}
