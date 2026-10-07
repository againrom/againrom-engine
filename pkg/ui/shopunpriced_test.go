package ui

import (
	"image"
	"image/draw"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// The plaque and figure of a cell whose price is 0 or less (ITEM-PRICETAG-144,
// SHOP-SCREEN-037). The grid skips only the money element and a quantity of 0,
// so such a cell draws the first plaque of its side with its grouped number on
// it, in every grid. Every expectation is written out from the claim's numbers
// and the synthetic art and font of shopprice_test.go, not read back from the
// code under test. A plaque is chosen from the stored price a cell carries, which
// on the player's side is not the figure printed on it.

// unpricedTestFont is priceTestFont with a minus glyph of its own pattern, so a
// figure drawn without its sign differs in pixels and not only in advance.
func unpricedTestFont() *text.Font {
	font := priceTestFont()
	g := text.Glyph{Width: 8, Height: 10, Advance: 5, Pixels: make([]text.Pixel, 80)}
	for row := 0; row < g.Height; row++ {
		for col := 0; col < g.Advance; col++ {
			if (col*3+row*5+int('-'))%3 == 0 {
				continue
			}
			g.Pixels[row*g.Width+col] = text.Pixel{Level: uint8(1 + (col*7+row*3)%15), Painted: true}
		}
	}
	font.Glyphs[int('-')-text.FirstChar] = g
	return font
}

// unpricedTestCell is one occupied cell of the composed screen: where it stands,
// which side's plaque it takes, the stored price the plaque is chosen from and
// the number the claim has it print. An entry with an empty text stands for a
// place that draws no plaque and no figure.
type unpricedTestCell struct {
	grid   string
	rect   image.Rectangle
	mine   bool
	stored int32
	text   string
}

// unpricedTestView fills the shelf, the table and the pack with occupied cells
// priced 0 or below on both sides of the deal and with two priced cells, and
// leaves an empty shelf place, an empty table place and the money element among
// them. The text is what SHOP-SCREEN-037's arithmetic and TOWN-469's grouping
// give: the shelf and the merchant's table print the price, the player's side
// the halved figure the view already carries, of a stored price twice as large.
func unpricedTestView(font *text.Font) (ShopScreenView, []unpricedTestCell) {
	v := ShopScreenView{Chosen: -1, Art: priceTestArt(), PriceFont: font}
	var cells []unpricedTestCell
	for i, c := range []struct {
		price int32
		text  string
	}{{0, "0"}, {-1, "-1"}, {-1250, "-1,250"}, {-999, "-999"}, {1250, "1,250"}} {
		v.Shelf[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, PlaquePrice: c.price}
		cells = append(cells, unpricedTestCell{"shelf", ShopShelfCellRect(i), false, c.price, c.text})
	}
	v.Shelf[5] = ShopCell{Back: ShopBackEmpty}
	cells = append(cells, unpricedTestCell{"shelf", ShopShelfCellRect(5), false, 0, ""})
	for i, c := range []struct {
		price int32
		mine  bool
		text  string
	}{{0, false, "0"}, {0, true, "0"}, {-1, false, "-1"}, {-1, true, "-1"}} {
		v.Table[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, PlaquePrice: c.price, Mine: c.mine}
		cells = append(cells, unpricedTestCell{"table", ShopTableCellRect(i), c.mine, c.price, c.text})
	}
	v.Table[4] = ShopCell{Back: ShopBackEmpty}
	cells = append(cells, unpricedTestCell{"table", ShopTableCellRect(4), false, 0, ""})
	v.Pack[0] = ShopCell{Money: true, Count: 1} // one coin prints no quantity (SHOP-106)
	cells = append(cells, unpricedTestCell{"pack", ShopPackCellRect(0), true, 0, ""})
	for i, c := range []struct {
		stored, price int32
		text          string
	}{{0, 0, "0"}, {-1, -1, "-1"}, {-1250, -624, "-624"}, {1250, 625, "625"}} {
		v.Pack[i+1] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, PlaquePrice: c.stored, Mine: true}
		cells = append(cells, unpricedTestCell{"pack", ShopPackCellRect(i + 1), true, c.stored, c.text})
	}
	return v, cells
}

// unpricedVariant states one way of drawing the expectation differently from
// the claim. The zero variant is the claim.
type unpricedVariant struct {
	plaque   int  // the plaque index the oracle blits for a cell whose stored price is 0 or less
	other    bool // the other side's plaque
	noPlaque bool
	noFigure bool
	unsigned bool // the figure without its minus sign
}

// expectedUnpriced paints on base what the claim gives for the cells: the plaque
// of the cell's side, right-aligned to the cell's right edge and one pixel below
// its top (index 0 for a stored price of 0 or less, the digit count of the stored
// price minus one for a positive one), then the figure.
func expectedUnpriced(base *image.RGBA, art *ShopScreenArt, font *text.Font, cells []unpricedTestCell, o unpricedVariant) *image.RGBA {
	want := image.NewRGBA(base.Bounds())
	copy(want.Pix, base.Pix)
	for _, c := range cells {
		if c.text == "" {
			continue
		}
		side := 0
		if c.mine != o.other {
			side = 1
		}
		index := o.plaque
		if c.stored > 0 {
			index = len(strconv.Itoa(int(c.stored))) - 1
		}
		if !o.noPlaque {
			pic := art.Plaque[side][index]
			at := image.Pt(c.rect.Max.X-pic.Bounds().Dx(), c.rect.Min.Y+1)
			draw.Draw(want, image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}, pic, pic.Bounds().Min, draw.Over)
		}
		if o.noFigure {
			continue
		}
		s := c.text
		if o.unsigned {
			s = strings.TrimPrefix(s, "-")
		}
		paintPriceFigure(want, font, c.rect, s, priceFigureVariant{})
	}
	return want
}

// A cell priced 0 or -1 draws the first plaque of its side and its figure in the
// shelf, the table and the pack, exactly as a cell of any other price does; only
// an empty place and the money element draw neither. The whole 640x480 screen
// equals the same screen without plaques or figures with the claim's plaques and
// figures painted on it. Each way of drawing them differently (no plaque, no
// figure, the second plaque, the other side's plaque, the figure without its
// sign, a pixel aside) leaves the screen different from that expectation.
func TestShopGridDrawsThePlaqueAndFigureOfAnUnpricedCell(t *testing.T) {
	font := unpricedTestFont()
	v, cells := unpricedTestView(font)
	art := v.Art
	with := ComposeShopScreen(v, image.Point{}, false, nil, false)

	bare := v
	bareArt := *art
	bareArt.Plaque = [2][7]*image.RGBA{}
	bare.Art, bare.PriceFont = &bareArt, nil
	base := ComposeShopScreen(bare, image.Point{}, false, nil, false)

	claimed := expectedUnpriced(base, art, font, cells, unpricedVariant{})
	if at, differs := firstScreenDifference(with, claimed); differs {
		t.Fatalf("the composed screen differs from the claimed plaques and figures first at %v: got %+v, want %+v",
			at, with.RGBAAt(at.X, at.Y), claimed.RGBAAt(at.X, at.Y))
	}
	if _, differs := firstScreenDifference(with, base); !differs {
		t.Fatal("the screen carries no plaque and no figure at all")
	}

	for name, o := range map[string]unpricedVariant{
		"no plaque":               {noPlaque: true},
		"no figure":               {noFigure: true},
		"the second plaque":       {plaque: 1},
		"the other side's plaque": {other: true},
		"the figure unsigned":     {unsigned: true},
	} {
		if _, differs := firstScreenDifference(with, expectedUnpriced(base, art, font, cells, o)); !differs {
			t.Errorf("the check cannot tell the claimed screen from %s", name)
		}
	}
	// A figure a pixel aside is a different screen too: paint the claimed
	// screen's figures again one pixel to the left over the bare plaques.
	plaques := expectedUnpriced(base, art, font, cells, unpricedVariant{noFigure: true})
	for _, c := range cells {
		if c.text != "" {
			paintPriceFigure(plaques, font, c.rect, c.text, priceFigureVariant{dx: -1})
		}
	}
	if _, differs := firstScreenDifference(with, plaques); !differs {
		t.Error("the check cannot tell the claimed screen from figures one pixel left")
	}
}
