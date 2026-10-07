package ui

// The status bars as the frame composes them, measured against the owner's
// screenshot of the original. The expected picture is written out here from
// the measurements, not read from the render tier, so the render tier cannot
// agree with itself by accident.

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
)

// sbClear marks a transparent cap pixel in the tables below.
const sbClear = 1 << 24

// The measured bar: a 4x4 end cap, 24 interior columns and the same cap again,
// unmirrored; rows top to bottom.
var (
	sbCap = [4][4]uint32{
		{sbClear, 0x6b4129, 0x4a2c18, sbClear},
		{0x9c6542, 0xce926b, 0x6b4129, 0x422810},
		{0x6b4129, 0x9c6542, 0x4a2810, 0x392410},
		{0x000400, 0x392410, 0x211408, sbClear},
	}
	sbHealthRows = [4]uint32{0x008200, 0x00ff00, 0x00c300, 0x008200}
	sbManaRows   = [4]uint32{0x000084, 0x0000ff, 0x0000c6, 0x000084}
	// sbGround is the terrain colour the screenshot's faded bar stands on.
	sbGround = color.RGBA{R: 0x31, G: 0x34, B: 0x21, A: 0xff}
)

func sbRGB(v uint32) color.RGBA {
	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

// sbCompose paints the frame's overlay passes over a uniform ground the way
// Draw submits them: a rectangle covers the pixels whose centres it contains,
// and its premultiplied colour is blended source-over.
func sbCompose(v *Viewer, w, h int, ground color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = ground.R, ground.G, ground.B, 0xff
	}
	for _, p := range v.overlayPasses() {
		for _, r := range p.Rects {
			x0, x1 := int(math.Ceil(r.X-0.5)), int(math.Ceil(r.X+r.W-0.5))
			y0, y1 := int(math.Ceil(r.Y-0.5)), int(math.Ceil(r.Y+r.H-0.5))
			for y := max(y0, 0); y < min(y1, h); y++ {
				for x := max(x0, 0); x < min(x1, w); x++ {
					old := img.RGBAAt(x, y)
					keep := 255 - uint32(p.Color.A)
					over := func(c, d uint8) uint8 { return uint8(uint32(c) + (uint32(d)*keep+127)/255) }
					img.SetRGBA(x, y, color.RGBA{over(p.Color.R, old.R), over(p.Color.G, old.G), over(p.Color.B, old.B), 0xff})
				}
			}
		}
	}
	return img
}

// sbBar is one bar the frame must show: its rows, how many interior columns
// are filled, whether the unit is unselected, and its native top-left relative
// to the top-left of the unit's cell.
type sbBar struct {
	rows  [4]uint32
	fill  int
	faded bool
	at    image.Point
}

// sbExpect is the colour one native bar pixel must read over ground, and a
// tolerance: the caps are opaque and exact, a selected unit's interior is
// opaque and exact, an unselected unit's interior is half the row colour and
// half the ground, and an unfilled interior column is the ground itself.
func sbExpect(b sbBar, x, y int, ground color.RGBA) (want color.RGBA, blend bool) {
	switch {
	case x < 4:
		if c := sbCap[y][x]; c != sbClear {
			return sbRGB(c), false
		}
		return ground, false
	case x >= 28:
		if c := sbCap[y][x-28]; c != sbClear {
			return sbRGB(c), false
		}
		return ground, false
	case x-4 >= b.fill:
		return ground, false
	case !b.faded:
		return sbRGB(b.rows[y]), false
	}
	c := sbRGB(b.rows[y])
	half := func(a, b uint8) uint8 { return uint8((uint32(a) + uint32(b)) / 2) }
	return color.RGBA{half(c.R, ground.R), half(c.G, ground.G), half(c.B, ground.B), 0xff}, true
}

func sbNear(got, want color.RGBA, tol int) bool {
	d := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return d(got.R, want.R) <= tol && d(got.G, want.G) <= tol && d(got.B, want.B) <= tol
}

// sbCheckBand compares the band above one unit's cell — 24 native rows up and
// four columns either side of the cell — against the bars that band must hold
// and the bare ground everywhere else. A native pixel differs when any of the
// zoom x zoom screen pixels it covers differs. It returns one line per native
// row that differs, naming the row's first differing pixel.
func sbCheckBand(img *image.RGBA, cellTL image.Point, zoom int, bars []sbBar, ground color.RGBA) []string {
	var bad []string
	for ny := -24; ny < 0; ny++ {
		n, first := 0, ""
		for nx := -4; nx < 36; nx++ {
			want, blend := ground, false
			for _, b := range bars {
				if bx, by := nx-b.at.X, ny-b.at.Y; bx >= 0 && bx < 32 && by >= 0 && by < 4 {
					want, blend = sbExpect(b, bx, by, ground)
				}
			}
			tol := 0
			if blend {
				tol = 1
			}
			differs := false
			var got color.RGBA
			for sy := 0; sy < zoom && !differs; sy++ {
				for sx := 0; sx < zoom && !differs; sx++ {
					got = img.RGBAAt(cellTL.X+nx*zoom+sx, cellTL.Y+ny*zoom+sy)
					differs = !sbNear(got, want, tol)
				}
			}
			if differs {
				if n == 0 {
					first = fmt.Sprintf("(%d,%d) got #%02x%02x%02x want #%02x%02x%02x",
						nx, ny, got.R, got.G, got.B, want.R, want.G, want.B)
				}
				n++
			}
		}
		if n > 0 {
			bad = append(bad, fmt.Sprintf("row %d: %d of 40 native pixels differ, first %s", ny, n, first))
		}
	}
	return bad
}

// TestStatusBarsComposeTheOriginalsPicture composes the frame's own overlay
// passes over the screenshot's ground colour and reads the band above four
// units pixel by pixel: a selected full mage, an unselected full fighter, a
// selected wounded mage and an unselected wounded fighter.
//
// Every bar is 32 by 4 native pixels with the measured cap at both ends. A
// selected unit's rows are opaque; an unselected unit's caps are opaque and
// its interior is a 50 percent blend with the ground. Health stands 16 to 13
// rows above the cell's top edge and mana directly under it, 12 to 9 rows
// above, both centred on the cell (DIV-1460). A wounded bar fills 24*value/max
// interior columns from the left and leaves the rest of the interior
// undrawn, with both caps kept (DIV-1459). The rest of the band is bare ground.
func TestStatusBarsComposeTheOriginalsPicture(t *testing.T) {
	for _, zoom := range []int{1, 2} {
		t.Run(fmt.Sprintf("zoom %d", zoom), func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{
				{ID: 1, Cell: image.Pt(1, 2), Life: LifeAlive, HP: 90, MaxHP: 90},
				{ID: 2, Cell: image.Pt(4, 2), Life: LifeAlive, HP: 60, MaxHP: 60, Mana: 40, MaxMana: 40},
				{ID: 3, Cell: image.Pt(1, 5), Life: LifeAlive, HP: 30, MaxHP: 60, Mana: 10, MaxMana: 40},
				{ID: 4, Cell: image.Pt(4, 5), Life: LifeAlive, HP: 45, MaxHP: 60},
			})
			v.sel = selection{2, 3}
			if zoom != 1 {
				v.Camera().X, v.Camera().Y, v.Camera().Zoom = 0, 0, float64(zoom)
			}
			img := sbCompose(v, 800, 600, sbGround)

			health := func(fill int, faded bool) sbBar {
				return sbBar{rows: sbHealthRows, fill: fill, faded: faded, at: image.Pt(0, -16)}
			}
			mana := func(fill int, faded bool) sbBar {
				return sbBar{rows: sbManaRows, fill: fill, faded: faded, at: image.Pt(0, -12)}
			}
			for _, u := range []struct {
				name string
				cell image.Point
				bars []sbBar
			}{
				{"unselected full fighter", image.Pt(1, 2), []sbBar{health(24, true)}},
				{"selected full mage", image.Pt(4, 2), []sbBar{health(24, false), mana(24, false)}},
				{"selected wounded mage", image.Pt(1, 5), []sbBar{health(12, false), mana(6, false)}},
				{"unselected wounded fighter", image.Pt(4, 5), []sbBar{health(18, true)}},
			} {
				r, ok := v.placeArm(u.cell, image.Rect(u.cell.X*32, u.cell.Y*32, u.cell.X*32+32, u.cell.Y*32+32))
				if !ok {
					t.Fatalf("%s: the cell is outside the view", u.name)
				}
				tl := image.Pt(int(math.Round(r.X)), int(math.Round(r.Y)))
				if bad := sbCheckBand(img, tl, zoom, u.bars, sbGround); len(bad) > 0 {
					t.Errorf("%s: the band above the cell differs from the original's bar, in native pixels from the cell's top-left:\n  %s",
						u.name, joinLines(bad))
				}
			}

			// The screenshot's own sample: the unselected fighter's brightest
			// row, #00ff00 over #313421, reads #189610 in the original. This
			// build blends at 8-bit precision, the original at 16-bit, so the
			// check allows the four units the original's halving loses in green.
			r, _ := v.placeArm(image.Pt(1, 2), image.Rect(32, 64, 64, 96))
			got := img.RGBAAt(int(math.Round(r.X))+10*zoom, int(math.Round(r.Y))-15*zoom)
			if !sbNear(got, sbRGB(0x189610), 4) {
				t.Errorf("unselected #00ff00 over #313421 reads #%02x%02x%02x; the original reads #189610", got.R, got.G, got.B)
			}
		})
	}
}

func joinLines(s []string) string {
	out := ""
	for i, l := range s {
		if i > 0 {
			out += "\n  "
		}
		out += l
	}
	return out
}
