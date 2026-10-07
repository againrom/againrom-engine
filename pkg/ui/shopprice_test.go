package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

// The price figure a shop grid draws on a plaque (ITEM-PRICETAG-144,
// MISSION-MSGLINE-056). Every expectation below is built from the claim's own
// numbers and from the font's glyph data, spelt out here rather than read back
// out of the code under test.

// priceTestFont is a font whose digit and comma glyphs each carry their own
// pattern of painted pixels and levels, so a figure drawn from the wrong glyph,
// at the wrong pen or in the wrong ramp differs in some pixel. Its digits advance
// 5 and its comma 2, and the letter spacing is 2.
func priceTestFont() *text.Font {
	font := &text.Font{Spacing: 2, Glyphs: make([]text.Glyph, 224)}
	for _, c := range []byte("0123456789,") {
		g := text.Glyph{Width: 8, Height: 10, Advance: 5, Pixels: make([]text.Pixel, 80)}
		if c == ',' {
			g.Advance = 2
		}
		for row := 0; row < g.Height; row++ {
			for col := 0; col < g.Advance; col++ {
				if (col*7+row*3+int(c))%4 == 0 {
					continue
				}
				g.Pixels[row*g.Width+col] = text.Pixel{Level: uint8(1 + (col*5+row*11+int(c))%15), Painted: true}
			}
		}
		font.Glyphs[int(c)-text.FirstChar] = g
	}
	return font
}

// priceTestArt is the flat-colour art set of the composition tests with a
// plaque of its own colour and shape for each digit count on each side: 60x10
// like the installed files, opaque only in the right of the canvas and four
// columns wider for each digit.
func priceTestArt() *ShopScreenArt {
	art := testArt()
	for side := range art.Plaque {
		for i := range art.Plaque[side] {
			pic := image.NewRGBA(image.Rect(0, 0, 60, 10))
			for y := 0; y < 10; y++ {
				for x := 60 - 20 - 4*i; x < 60; x++ {
					pic.SetRGBA(x, y, color.RGBA{R: uint8(100 + 20*side), G: uint8(40 + 30*i), B: 200, A: 255})
				}
			}
			art.Plaque[side][i] = pic
		}
	}
	return art
}

// priceTestCell is one priced cell of the composed screen: its grid, its
// rectangle and the figure the grid must draw on it, written out by hand.
type priceTestCell struct {
	grid string
	rect image.Rectangle
	text string
}

// priceTestView fills every shelf, table and pack place with an item, so that
// the screen carries figures of one to seven digits, on both sides of the deal
// and in all three grids.
func priceTestView(font *text.Font) (ShopScreenView, []priceTestCell) {
	v := ShopScreenView{Chosen: -1, Art: priceTestArt(), PriceFont: font}
	var cells []priceTestCell
	for i, c := range []struct {
		price int32
		text  string
	}{{7, "7"}, {99, "99"}, {1250, "1,250"}, {99999, "99,999"}, {1234567, "1,234,567"}, {100, "100"}} {
		v.Shelf[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price}
		cells = append(cells, priceTestCell{"shelf", ShopShelfCellRect(i), c.text})
	}
	for i, c := range []struct {
		price int32
		mine  bool
		text  string
	}{{12000, false, "12,000"}, {500, true, "500"}, {9999999, false, "9,999,999"}, {1000000, true, "1,000,000"}, {40, false, "40"}} {
		v.Table[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, Mine: c.mine}
		cells = append(cells, priceTestCell{"table", ShopTableCellRect(i), c.text})
	}
	for i, c := range []struct {
		price int32
		text  string
	}{{625, "625"}, {50000, "50,000"}, {5000000, "5,000,000"}, {3, "3"}, {123456, "123,456"}} {
		v.Pack[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, Mine: true}
		cells = append(cells, priceTestCell{"pack", ShopPackCellRect(i), c.text})
	}
	return v, cells
}

// priceFigureVariant states one way of drawing a figure: how far it stands from
// the claimed placement, whether its shadow is drawn and the colours its shadow
// and its ramp take. The zero variant is the placement, ramp and shadow the
// claim gives.
type priceFigureVariant struct {
	dx, dy   int
	noShadow bool
	ink      *color.RGBA
	shadow   *color.RGBA
}

// paintPriceFigure paints on pic the figure the claim gives for s in the cell:
// the text's pen ends 6 pixels left of the cell's right edge, its glyph cells
// start one pixel below the cell's top, and a shadow of the same text one pixel
// right and down is painted first in the flat colour 8 8 8. A glyph pixel of
// level k takes the ramp entry (185k/15, 159k/15, 73k/15).
func paintPriceFigure(pic *image.RGBA, font *text.Font, cell image.Rectangle, s string, o priceFigureVariant) {
	width := 0
	for i := 0; i < len(s); i++ {
		width += font.Glyphs[int(s[i])-text.FirstChar].Advance + font.Spacing
	}
	x0, y0 := cell.Max.X-6-width+o.dx, cell.Min.Y+1+o.dy
	pass := func(offset int, colour func(level uint8) color.RGBA) {
		pen := 0
		for i := 0; i < len(s); i++ {
			g := font.Glyphs[int(s[i])-text.FirstChar]
			for row := 0; row < g.Height; row++ {
				for col := 0; col < g.Width; col++ {
					if p := g.Pixels[row*g.Width+col]; p.Painted {
						pic.SetRGBA(x0+pen+col+offset, y0+row+offset, colour(p.Level))
					}
				}
			}
			pen += g.Advance + font.Spacing
		}
	}
	if !o.noShadow {
		flat := color.RGBA{R: 8, G: 8, B: 8, A: 255}
		if o.shadow != nil {
			flat = *o.shadow
		}
		pass(1, func(uint8) color.RGBA { return flat })
	}
	pass(0, func(level uint8) color.RGBA {
		k := int(level)
		if o.ink != nil {
			return color.RGBA{R: uint8(int(o.ink.R) * k / 15), G: uint8(int(o.ink.G) * k / 15), B: uint8(int(o.ink.B) * k / 15), A: 255}
		}
		return color.RGBA{R: uint8(185 * k / 15), G: uint8(159 * k / 15), B: uint8(73 * k / 15), A: 255}
	})
}

// firstScreenDifference is the first pixel, in reading order, at which two pictures
// of one size differ, and whether there is one.
func firstScreenDifference(a, b *image.RGBA) (image.Point, bool) {
	if a.Bounds() != b.Bounds() {
		return a.Bounds().Min, true
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return image.Pt(x, y), true
			}
		}
	}
	return image.Point{}, false
}

// Every shop grid draws a priced cell's grouped figure with its pen ending 6
// pixels left of the cell's right edge, its glyph cells one pixel below the
// cell's top, over the plaque, in the ramp (185k/15, 159k/15, 73k/15) with a
// flat 8 8 8 shadow one pixel right and down (ITEM-PRICETAG-144,
// MISSION-MSGLINE-056). The whole 640x480 screen equals the same screen
// without the figures with the claim's figures painted on it. Each way of drawing
// them differently (a pixel aside in each direction, no shadow, a shadow or a
// ramp of another colour) leaves the screen different from that expectation.
func TestShopGridPriceFigureIsRightAlignedWithAShadow(t *testing.T) {
	font := priceTestFont()
	v, cells := priceTestView(font)
	if len(cells) != shopShelfN+2*shopStripN {
		t.Fatalf("the view holds %d priced cells, want %d", len(cells), shopShelfN+2*shopStripN)
	}
	with := ComposeShopScreen(v, image.Point{}, false, nil, false)
	bare := v
	bare.PriceFont = nil
	base := ComposeShopScreen(bare, image.Point{}, false, nil, false)

	expected := func(o priceFigureVariant) *image.RGBA {
		want := image.NewRGBA(base.Bounds())
		copy(want.Pix, base.Pix)
		for _, c := range cells {
			paintPriceFigure(want, font, c.rect, c.text, o)
		}
		return want
	}
	claimed := expected(priceFigureVariant{})
	if at, differs := firstScreenDifference(with, claimed); differs {
		t.Fatalf("the composed screen differs from the claimed figures first at %v: got %+v, want %+v",
			at, with.RGBAAt(at.X, at.Y), claimed.RGBAAt(at.X, at.Y))
	}
	if _, differs := firstScreenDifference(with, base); !differs {
		t.Fatal("the screen carries no figure at all")
	}

	cream, black := shopTextColor, color.RGBA{A: 255}
	for name, o := range map[string]priceFigureVariant{
		"one pixel left":                   {dx: -1},
		"one pixel right":                  {dx: 1},
		"one pixel up":                     {dy: -1},
		"one pixel down":                   {dy: 1},
		"no shadow":                        {noShadow: true},
		"a black shadow":                   {shadow: &black},
		"a shadow one level off in red":    {shadow: &color.RGBA{R: 9, G: 8, B: 8, A: 255}},
		"the shop's own text ramp":         {ink: &cream},
		"a ramp one level off in blue":     {ink: &color.RGBA{R: 185, G: 159, B: 74, A: 255}},
		"a ramp one level off in red":      {ink: &color.RGBA{R: 186, G: 159, B: 73, A: 255}},
		"a ramp one level off in green":    {ink: &color.RGBA{R: 185, G: 160, B: 73, A: 255}},
		"the ramp's channels in BGR order": {ink: &color.RGBA{R: 73, G: 159, B: 185, A: 255}},
	} {
		if _, differs := firstScreenDifference(with, expected(o)); !differs {
			t.Errorf("the check cannot tell the claimed figures from %s", name)
		}
	}
}
