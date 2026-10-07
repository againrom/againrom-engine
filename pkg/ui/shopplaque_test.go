package ui

import (
	"image"
	"image/draw"
	"strconv"
	"testing"

	"againrom/pkg/render/text"
)

// The plaque a shop cell draws is chosen from the stored price the cell carries,
// on both sides of the deal, and the figure printed on it does not enter the
// choice: the player's side prints half of the stored price on the plaque of the
// whole (SHOP-SCREEN-037, ITEM-PRICETAG-144). Every expectation is written out
// from the claim's numbers and the synthetic art and font of shopprice_test.go.

// plaqueTestCell is one occupied cell of the composed screen: where it stands,
// which side it is on, the stored price its plaque is chosen from, the figure
// the claim has it print and the plaque index the claim gives.
type plaqueTestCell struct {
	rect   image.Rectangle
	mine   bool
	stored int32
	text   string
	plaque int
}

// plaqueTestView fills the shelf, the table and the pack with occupied cells
// whose stored prices sit at the edges of a decade, where half the price has one
// digit fewer than the price. The shelf and the merchant's places print the
// stored price; the player's places print (stored+1)/2.
func plaqueTestView(font *text.Font) (ShopScreenView, []plaqueTestCell) {
	v := ShopScreenView{Chosen: -1, Art: priceTestArt(), PriceFont: font}
	var cells []plaqueTestCell
	for i, c := range []struct {
		stored int32
		text   string
		plaque int
	}{{9, "9", 0}, {10, "10", 1}, {18, "18", 1}, {100, "100", 2}, {1250, "1,250", 3}, {9999999, "9,999,999", 6}} {
		v.Shelf[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.stored, PlaquePrice: c.stored}
		cells = append(cells, plaqueTestCell{ShopShelfCellRect(i), false, c.stored, c.text, c.plaque})
	}
	for i, c := range []struct {
		stored, price int32
		mine          bool
		text          string
		plaque        int
	}{
		{10, 5, true, "5", 1}, {19, 10, true, "10", 1}, {100, 100, false, "100", 2},
		{198, 99, true, "99", 2}, {1999998, 999999, true, "999,999", 6},
	} {
		v.Table[i] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, PlaquePrice: c.stored, Mine: c.mine}
		cells = append(cells, plaqueTestCell{ShopTableCellRect(i), c.mine, c.stored, c.text, c.plaque})
	}
	v.Pack[0] = ShopCell{Money: true, Count: 1} // one coin prints no quantity (SHOP-106)
	for i, c := range []struct {
		stored, price int32
		text          string
		plaque        int
	}{{9, 5, "5", 0}, {14, 7, "7", 1}, {150, 75, "75", 2}, {1250, 625, "625", 3}} {
		v.Pack[i+1] = ShopCell{Back: ShopBackItem, Count: 1, Price: c.price, PlaquePrice: c.stored, Mine: true}
		cells = append(cells, plaqueTestCell{ShopPackCellRect(i + 1), true, c.stored, c.text, c.plaque})
	}
	return v, cells
}

// plaqueVariant states one way of choosing the plaque differently from the claim.
// The zero variant is the claim.
type plaqueVariant struct {
	fromFigure bool // the digit count of the printed figure, the halved one on the player's side
	other      bool // the other side's plaque
	shift      int  // added to the index, clamped to the seven plaques
	noPlaque   bool
}

// expectedPlaques paints on base the plaque the claim gives each cell,
// right-aligned to the cell's right edge and one pixel below its top, then the
// cell's figure.
func expectedPlaques(base *image.RGBA, art *ShopScreenArt, font *text.Font, cells []plaqueTestCell, o plaqueVariant) *image.RGBA {
	want := image.NewRGBA(base.Bounds())
	copy(want.Pix, base.Pix)
	for _, c := range cells {
		side := 0
		if c.mine != o.other {
			side = 1
		}
		index := c.plaque
		if o.fromFigure {
			digits := 0
			for _, r := range c.text {
				if r >= '0' && r <= '9' {
					digits++
				}
			}
			index = digits - 1
		}
		index = min(max(index+o.shift, 0), 6)
		if !o.noPlaque {
			pic := art.Plaque[side][index]
			at := image.Pt(c.rect.Max.X-pic.Bounds().Dx(), c.rect.Min.Y+1)
			draw.Draw(want, image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}, pic, pic.Bounds().Min, draw.Over)
		}
		paintPriceFigure(want, font, c.rect, c.text, priceFigureVariant{})
	}
	return want
}

// Every grid draws the plaque of the stored price its cell carries: on the
// player's side 10 draws the second plaque and prints 5, 198 the third and
// prints 99, 1,999,998 the seventh and prints 999,999. The whole 640x480 screen
// equals the same screen without plaques and figures with the claimed plaques
// and figures painted on it. Choosing the plaque from the printed figure, from
// the other side's family, one plaque up or down, or not drawing it leaves the
// screen different from that expectation.
func TestShopGridPlaqueIsChosenFromThePlaquePrice(t *testing.T) {
	font := priceTestFont()
	v, cells := plaqueTestView(font)
	art := v.Art
	with := ComposeShopScreen(v, image.Point{}, false, nil, false)

	bare := v
	bareArt := *art
	bareArt.Plaque = [2][7]*image.RGBA{}
	bare.Art, bare.PriceFont = &bareArt, nil
	base := ComposeShopScreen(bare, image.Point{}, false, nil, false)

	claimed := expectedPlaques(base, art, font, cells, plaqueVariant{})
	if at, differs := firstScreenDifference(with, claimed); differs {
		t.Fatalf("the composed screen differs from the claimed plaques first at %v: got %+v, want %+v",
			at, with.RGBAAt(at.X, at.Y), claimed.RGBAAt(at.X, at.Y))
	}
	if _, differs := firstScreenDifference(with, base); !differs {
		t.Fatal("the screen carries no plaque and no figure at all")
	}
	for _, c := range cells {
		if digits := len(strconv.Itoa(int(c.stored))); c.plaque != digits-1 {
			t.Fatalf("the cell at %v names plaque %d for the stored price %d of %d digits", c.rect, c.plaque, c.stored, digits)
		}
	}

	for name, o := range map[string]plaqueVariant{
		"the plaque of the printed figure": {fromFigure: true},
		"the other side's plaque":          {other: true},
		"one plaque up":                    {shift: 1},
		"one plaque down":                  {shift: -1},
		"no plaque":                        {noPlaque: true},
	} {
		if _, differs := firstScreenDifference(with, expectedPlaques(base, art, font, cells, o)); !differs {
			t.Errorf("the check cannot tell the claimed screen from %s", name)
		}
	}
}
