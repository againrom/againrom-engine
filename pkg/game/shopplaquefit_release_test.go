package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// paintedBounds is the smallest rectangle holding every pixel a draw of s in
// font at (x, y) paints, and the empty rectangle for a draw that paints none.
func paintedBounds(font *text.Font, s string, x, y int) image.Rectangle {
	w, h := font.Measure(s)
	var out image.Rectangle
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			if font.PaintedAt(s, x, y, image.Pt(px, py)) {
				out = out.Union(image.Rect(px, py, px+1, py+1))
			}
		}
	}
	return out
}

// TestReleaseShopPricesFitTheirPlaques measures, on the lawful art, the ink of
// the grouped price figure where ITEM-PRICETAG-144 puts it against the ink of the
// plaque SHOP-SCREEN-037 picks for the stored price's digit count, on both sides
// of the deal and in all three grids, for stored prices of every digit count
// from one to seven, at the edges of each decade and where the player's half
// keeps the digit count, and for a price of 0, which takes the first plaque.
// Every pixel the figure paints must lie inside its plaque's ink, and its shadow
// one pixel right and down inside its cell. The margins between the figure and
// the ink, and how far the shadow reaches past the ink, are logged for the shelf
// grid.
func TestReleaseShopPricesFitTheirPlaques(t *testing.T) {
	f := releaseFront(t)
	art := f.shopArt()
	font := f.tipFont()
	if art == nil || font == nil {
		t.Fatal("shop art or price font did not resolve")
	}
	cells := map[string]image.Rectangle{"shelf": ui.ShopShelfCellRect(1),
		"table": ui.ShopTableCellRect(4), "pack": ui.ShopPackCellRect(4)}
	for _, stored := range []int32{0, 1, 9, 10, 18, 19, 20, 99, 100, 198, 199, 200, 999, 1000, 1998, 1999, 2000, 9999, 12000, 19998, 20000, 99999,
		100000, 199998, 200000, 999999, 1000000, 1999998, 2000000, 9999999} {
		for _, mine := range []bool{false, true} {
			for grid, r := range cells {
				price := shopGridPrice(gridWeapon, stored, mine)
				cell := ui.ShopCell{Back: ui.ShopBackItem, Count: 1, Price: price, PlaquePrice: shopClientPrice(gridWeapon, stored), Mine: mine}
				ink, figure := ui.ShopPricePlacement(art, font, cell, r)
				name := fmt.Sprintf("%s stored price %d prints %d mine %v", grid, stored, price, mine)
				if ink.Empty() || figure.Empty() {
					t.Fatalf("%s: plaque ink %v figure %v", name, ink, figure)
				}
				figureInk := paintedBounds(font, ui.GroupDigits(int64(price)), figure.Min.X, figure.Min.Y)
				shadow := figureInk.Add(image.Pt(1, 1))
				if figureInk.Empty() || !figureInk.In(ink) || !shadow.In(r) {
					t.Fatalf("%s: figure ink %v (%d px) outside plaque ink %v (%d px), or its shadow %v outside cell %v",
						name, figureInk, figureInk.Dx(), ink, ink.Dx(), shadow, r)
				}
				if grid == "shelf" {
					t.Logf("%s: plaque ink %v, figure ink %v: margin left %d right %d top %d bottom %d, shadow past the ink right %d bottom %d",
						name, ink, figureInk, figureInk.Min.X-ink.Min.X, ink.Max.X-figureInk.Max.X,
						figureInk.Min.Y-ink.Min.Y, ink.Max.Y-figureInk.Max.Y, max(0, shadow.Max.X-ink.Max.X), max(0, shadow.Max.Y-ink.Max.Y))
				}
			}
		}
	}
}
