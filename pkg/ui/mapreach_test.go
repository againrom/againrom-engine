package ui

import (
	"image"
	"testing"
)

func TestMapBottomCellsRemainClickableAtSideLimits(t *testing.T) {
	const width, height = 80, 80
	g := grid(width, height)
	g.Block = make([]uint8, width*height)
	g.Altitudes = make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y >= 70 {
				g.Altitudes[y*width+x] = 180
			}
			if x < 8 || x >= width-8 || y < 8 || y >= height-8 {
				g.Block[y*width+x] = borderBit
			}
		}
	}
	v := newViewer(t, g)
	layoutViewport(v, 864, 593)
	for _, side := range []struct {
		name string
		pan  float64
		x    float64
		col  int
	}{{"left", -1e9, 80, 10}, {"right", 1e9, 848, 71}} {
		t.Run(side.name, func(t *testing.T) {
			v.cam.Pan(side.pan, 1e9)
			col, row, inside := v.groundCellAt(side.x, 592)
			if !inside || col != side.col || row != 70 {
				t.Fatalf("bottom point (%g,592) = (%d,%d,%v), want (%d,70,true); camera (%g,%g)", side.x, col, row, inside, side.col, v.cam.X, v.cam.Y)
			}
			r := &mapCoverageTarget{headlessShroudTarget{pixels: image.NewRGBA(image.Rect(0, 0, 864, 593))}}
			v.drawDisplaced(r)
			if r.err != nil {
				t.Fatal(r.err)
			}
			for y := 0; y < 593; y++ {
				for x := 0; x < 864; x++ {
					if r.pixels.RGBAAt(x, y).A != 255 {
						t.Fatalf("terrain leaves pixel (%d,%d) unpainted", x, y)
					}
				}
			}
		})
	}
}
