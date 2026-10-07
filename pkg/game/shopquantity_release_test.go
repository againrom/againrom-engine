package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
)

type quantityVariant struct {
	dx, dy   int
	noShadow bool
	ink      *color.RGBA
	shadow   *color.RGBA
	font     *priceOracle
	// rightAligned stands the figure at the cell's right edge less 6, the price
	// run's flag 1, instead of the quantity run's flag 0.
	rightAligned bool
}

func (o priceOracle) paintQuantity(pic *image.RGBA, cell image.Rectangle, count uint32, v quantityVariant) {
	if v.font != nil {
		o = *v.font
	}
	s := groupedFigure(int32(count))
	x0, y0 := cell.Min.X+10+v.dx, cell.Max.Y-15+v.dy
	if v.rightAligned {
		width := 0
		for i := 0; i < len(s); i++ {
			width += o.advances[int(s[i])-32] + 2
		}
		x0 = cell.Max.X - 6 - width + v.dx
	}
	pass := func(offset int, colour func(level uint8) color.RGBA) {
		pen := 0
		for i := 0; i < len(s); i++ {
			g := o.frames[int(s[i])-32]
			for row := 0; row < g.Height; row++ {
				for col := 0; col < g.Width; col++ {
					if p := g.Pixels[row*g.Width+col]; p.Painted {
						pic.SetRGBA(x0+pen+col+offset, y0+row+offset, colour(p.Value))
					}
				}
			}
			pen += o.advances[int(s[i])-32] + 2
		}
	}
	if !v.noShadow {
		flat := color.RGBA{R: 8, G: 8, B: 8, A: 255}
		if v.shadow != nil {
			flat = *v.shadow
		}
		pass(1, func(uint8) color.RGBA { return flat })
	}
	pass(0, func(level uint8) color.RGBA {
		k := int(level)
		ink := color.RGBA{R: 185, G: 159, B: 73}
		if v.ink != nil {
			ink = *v.ink
		}
		return color.RGBA{R: uint8(int(ink.R) * k / 15), G: uint8(int(ink.G) * k / 15), B: uint8(int(ink.B) * k / 15), A: 255}
	})
}

func quantityDrawn(c ui.ShopCell) bool { return c.Count > 1 && (c.Money || c.Occupied()) }

func quantityStripDifferences(got, bare *image.RGBA, slots []priceSlot, o priceOracle, v quantityVariant) (int, string) {
	want := image.NewRGBA(bare.Bounds())
	copy(want.Pix, bare.Pix)
	for _, s := range slots {
		if quantityDrawn(s.cell) {
			o.paintQuantity(want, s.rect, s.cell.Count, v)
		}
	}
	bad, first := 0, ""
	for _, s := range slots {
		if !quantityDrawn(s.cell) {
			continue
		}
		r := image.Rect(s.rect.Min.X, s.rect.Max.Y-16, s.rect.Max.X, s.rect.Max.Y)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if got.RGBAAt(x, y) != want.RGBAAt(x, y) {
					if bad == 0 {
						first = fmt.Sprintf("%s %d at (%d,%d): got %+v, want %+v", s.grid, s.index, x, y, got.RGBAAt(x, y), want.RGBAAt(x, y))
					}
					bad++
				}
			}
		}
	}
	return bad, first
}

// The grid quantity and the purse draw left-aligned in font2, in the ramp over
// a flat shadow (SHOP-106, MISSION-MSGLINE-056).
func TestReleaseShopGridQuantityAndPurseAreRampedWithAShadow(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	s.CloseTip()
	oracle := newPriceOracle(t, f, "font2")
	wrongFont := newPriceOracle(t, f, "font1")
	cream := color.RGBA{R: 0xe8, G: 0xdc, B: 0xc0, A: 0xff}
	black := color.RGBA{A: 255}
	controls := map[string]quantityVariant{
		"one pixel left":  {dx: -1},
		"one pixel up":    {dy: -1},
		"no shadow":       {noShadow: true},
		"the shop's ink":  {ink: &cream},
		"a black shadow":  {shadow: &black},
		"font1 glyphs":    {font: &wrongFont},
		"right-aligned":   {rightAligned: true},
		"one pixel right": {dx: 1},
	}
	verify := func(name string, got *image.RGBA, view ui.ShopScreenView) int {
		t.Helper()
		bareView := view
		bareView.Font, bareView.PriceFont, bareView.Character.Font = nil, nil, nil
		bare := ui.ComposeShopScreen(bareView, image.Point{}, false, nil, false)
		slots := priceSlots(view)
		n := 0
		for _, slot := range slots {
			if quantityDrawn(slot.cell) {
				n++
			}
		}
		if bad, first := quantityStripDifferences(got, bare, slots, oracle, quantityVariant{}); bad != 0 {
			t.Fatalf("%s: the screen differs from the claimed numbers at %d pixel(s), first %s", name, bad, first)
		}
		for variant, v := range controls {
			if n > 0 {
				if bad, _ := quantityStripDifferences(got, bare, slots, oracle, v); bad == 0 {
					t.Errorf("%s: the check cannot tell the claimed numbers from %s", name, variant)
				}
			}
		}
		return n
	}

	ix, iy, err := app.HeadlessShopIdlePoint()
	if err != nil {
		t.Fatal(err)
	}
	stacks, purses := 0, 0
	for pick := range shopRoomShelves {
		x, y, err := app.HeadlessShopPoint("shelf_pick", pick)
		if err != nil {
			t.Fatal(err)
		}
		for _, edge := range []string{"press", "release"} {
			if err := app.HeadlessPointer(edge, x, y); err != nil {
				t.Fatal(err)
			}
		}
		if err := app.HeadlessPointer("hover", ix, iy); err != nil {
			t.Fatal(err)
		}
		frame, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		view := s.ShopScreen()
		verify(fmt.Sprintf("shelf %d", pick), frame, view)
		for _, slot := range priceSlots(view) {
			switch {
			case !quantityDrawn(slot.cell):
			case slot.cell.Money:
				purses++
			default:
				stacks++
			}
		}
	}
	if stacks == 0 || purses == 0 {
		t.Fatalf("the App's frames carried %d stack numbers and %d purse numbers, want some of each", stacks, purses)
	}

	art, font := f.shopArt(), f.Font.Value()
	if art == nil || font == nil {
		t.Fatal("shop art or font did not resolve")
	}
	counts := []uint32{2, 9, 10, 999, 1000, 12345, 1000000, 9999999}
	v := ui.ShopScreenView{Chosen: -1, Art: art, Font: font, PriceFont: f.tipFont()}
	for i := 0; i < 5; i++ {
		cell := ui.ShopCell{Back: ui.ShopBackItem, Count: counts[i], Price: 1}
		v.Shelf[i], v.Table[i] = cell, cell
		if i < 3 {
			v.Pack[i] = cell
		}
	}
	v.Pack[3] = ui.ShopCell{Money: true, Count: counts[7]}
	v.Pack[4] = ui.ShopCell{Money: true, Count: counts[5]}
	if verify("population of counts", ui.ComposeShopScreen(v, image.Point{}, false, nil, false), v) != 15 {
		t.Fatal("the population did not carry its 15 numbers")
	}
}
