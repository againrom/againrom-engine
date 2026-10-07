package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"testing"

	"againrom/pkg/render/terrain"
)

// sbCheckBar compares one native bar rectangle of the composed frame with the
// original's picture at that rectangle's own width: the measured cap at both
// ends, the interior between them filled for fill columns from the left
// (half blended when faded) and ground past the fill, and ground in the
// column either side of the bar. at is the screen pixel of the bar's native
// top-left. It returns one line per native row that differs.
func sbCheckBar(img *image.RGBA, at image.Point, zoom, width int, rows [4]uint32, fill int, faded bool, ground color.RGBA) []string {
	var bad []string
	for y := range 4 {
		n, first := 0, ""
		for x := -1; x <= width; x++ {
			want, blend := ground, false
			switch {
			case x < 0 || x >= width:
			case x < 4:
				if c := sbCap[y][x]; c != sbClear {
					want = sbRGB(c)
				}
			case x >= width-4:
				if c := sbCap[y][x-width+4]; c != sbClear {
					want = sbRGB(c)
				}
			case x-4 < fill && !faded:
				want = sbRGB(rows[y])
			case x-4 < fill:
				c := sbRGB(rows[y])
				half := func(a, b uint8) uint8 { return uint8((uint32(a) + uint32(b)) / 2) }
				want, blend = color.RGBA{half(c.R, ground.R), half(c.G, ground.G), half(c.B, ground.B), 0xff}, true
			}
			tol := 0
			if blend {
				tol = 1
			}
			var got color.RGBA
			differs := false
			for sy := 0; sy < zoom && !differs; sy++ {
				for sx := 0; sx < zoom && !differs; sx++ {
					got = img.RGBAAt(at.X+x*zoom+sx, at.Y+y*zoom+sy)
					differs = !sbNear(got, want, tol)
				}
			}
			if differs {
				if n == 0 {
					first = fmt.Sprintf("(%d,%d) got #%02x%02x%02x want #%02x%02x%02x",
						x, y, got.R, got.G, got.B, want.R, want.G, want.B)
				}
				n++
			}
		}
		if n > 0 {
			bad = append(bad, fmt.Sprintf("row %d: %d of %d native pixels differ, first %s", y, n, width+2, first))
		}
	}
	return bad
}

// rectWidths is the width of each rectangle, in order.
func rectWidths(rs []image.Rectangle) []int {
	out := make([]int, len(rs))
	for i, r := range rs {
		out[i] = r.Dx()
	}
	return out
}

// TestStatusBarsSpanAWideUnitsSelectionBox composes the frame's overlay passes
// for an unselected, wounded dragon and a selected mage, each drawn with its
// installed class geometry: canvas centre, TileSize and units.reg selection
// box. The owner's capture of the original shows a dragon's bars three times a
// small unit's width over the dragon's body.
//
// The dragon stands on cell (3,4), so its body's class centre lands on that
// cell's centre (112,144). Its bars span its 96-column box, x 64 to 159, with
// health 2 rows above the box's top edge (row 94) and mana under it; each has
// the measured cap at both ends and an 88-column interior, filled 88*45/60 =
// 66 and 88*10/40 = 22 columns and half blended. The mage's bars keep the
// one-cell rule, 32 by 4 over its cell (DIV-1460).
func TestStatusBarsSpanAWideUnitsSelectionBox(t *testing.T) {
	dragon := &terrain.UnitClass{Width: 160, Height: 160, CenterX: 80, CenterY: 80, TileSize: 3,
		Selection: image.Rect(32, 32, 128, 128)}
	mage := &terrain.UnitClass{Width: 128, Height: 128, CenterX: 64, CenterY: 78, TileSize: 1,
		Selection: image.Rect(48, 48, 80, 90)}
	for _, zoom := range []int{1, 2} {
		t.Run(fmt.Sprintf("zoom %d", zoom), func(t *testing.T) {
			v := commandViewer(t)
			v.SetEntities([]MapEntity{
				{ID: 1, Cell: image.Pt(3, 4), TokenSize: 3, Art: dragon, Life: LifeAlive, HP: 45, MaxHP: 60, Mana: 10, MaxMana: 40},
				{ID: 2, Cell: image.Pt(6, 1), TokenSize: 1, Art: mage, Life: LifeAlive, HP: 60, MaxHP: 60, Mana: 40, MaxMana: 40},
			})
			v.sel = selection{2}
			if zoom != 1 {
				v.Camera().X, v.Camera().Y, v.Camera().Zoom = 0, 0, float64(zoom)
			}
			bars := map[image.Point][]image.Rectangle{}
			for _, b := range v.statusBars() {
				bars[b.Cell] = append(bars[b.Cell], b.Rect)
			}
			for _, u := range []struct {
				name string
				cell image.Point
				want []image.Rectangle
			}{
				{"dragon", image.Pt(3, 4), []image.Rectangle{image.Rect(64, 94, 160, 98), image.Rect(64, 98, 160, 102)}},
				{"mage", image.Pt(6, 1), []image.Rectangle{image.Rect(192, 16, 224, 20), image.Rect(192, 20, 224, 24)}},
			} {
				if got := bars[u.cell]; !slices.Equal(got, u.want) {
					t.Errorf("the %s's health and mana bars are %v, %v columns wide; want %v, %d wide",
						u.name, got, rectWidths(got), u.want, u.want[0].Dx())
				}
			}
			img := sbCompose(v, 800, 600, sbGround)
			for _, b := range []struct {
				name  string
				cell  image.Point
				bar   image.Rectangle
				rows  [4]uint32
				fill  int
				faded bool
			}{
				{"unselected dragon's health", image.Pt(3, 4), image.Rect(64, 94, 160, 98), sbHealthRows, 66, true},
				{"unselected dragon's mana", image.Pt(3, 4), image.Rect(64, 98, 160, 102), sbManaRows, 22, true},
				{"selected mage's health", image.Pt(6, 1), image.Rect(192, 16, 224, 20), sbHealthRows, 24, false},
				{"selected mage's mana", image.Pt(6, 1), image.Rect(192, 20, 224, 24), sbManaRows, 24, false},
			} {
				r, ok := v.placeArm(b.cell, b.bar)
				if !ok {
					t.Fatalf("%s: the bar %v is outside the view", b.name, b.bar)
				}
				at := image.Pt(int(math.Round(r.X)), int(math.Round(r.Y)))
				if bad := sbCheckBar(img, at, zoom, b.bar.Dx(), b.rows, b.fill, b.faded, sbGround); len(bad) > 0 {
					t.Errorf("%s: the %d-column bar at native %v differs from the original's picture, in native pixels from its top-left:\n  %s",
						b.name, b.bar.Dx(), b.bar.Min, joinLines(bad))
				}
			}
		})
	}
}
