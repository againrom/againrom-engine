package game

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func TestReleaseShopBuyAndSellTotalsDrawBelowCaptions(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, app, s, _ := releaseQuietShopApp(t)
	raw := roomCaptionRaw(t, f, "main/text/main.txt")
	labels := [4]string{
		roomCaptionRawLine(t, raw, 72), roomCaptionRawLine(t, raw, 70),
		roomCaptionRawLine(t, raw, 71), roomCaptionRawLine(t, raw, 73),
	}
	oracle := firstGameButtonOracle(t, f)
	font := oracle.font
	comparisons := 0
	verify := func(name string, v ui.ShopScreenView, frame *image.RGBA) {
		t.Helper()
		values := [4]int32{v.Purse, v.Buy, v.Sell, v.Total}
		for pressed := -1; pressed < 4; pressed++ {
			var hover image.Point
			if pressed >= 0 {
				r := ui.ShopButtonRect(pressed)
				hover = r.Min.Add(r.Max).Div(2)
				v.Press = ui.ShopControl{Kind: ui.ShopControlButton, Index: pressed}
			}
			got := frame
			if pressed >= 0 || got == nil {
				got = ui.ComposeShopScreen(v, hover, pressed >= 0, nil, false)
			}
			if dir := os.Getenv("AGAINROM_SHOP_TOTALS_OUT"); dir != "" && pressed < 0 {
				writeFramePNG(t, filepath.Join(dir, name+".png"), got)
			}
			bare := v
			bare.Font = nil
			if bare.ButtonStyle != nil {
				style := *bare.ButtonStyle
				style.Font = nil
				bare.ButtonStyle = &style
			}
			want := ui.ComposeShopScreen(bare, hover, pressed >= 0, nil, false)
			for i := range labels {
				r := ui.ShopButtonRect(i)
				bands := [2]image.Rectangle{
					image.Rect(r.Min.X, r.Min.Y+2, r.Max.X, r.Min.Y+24).Add(oracle.label),
					image.Rect(r.Min.X, r.Min.Y+23, r.Max.X, r.Max.Y-2).Add(oracle.value),
				}
				captions := []string{labels[i], groupedFigure(values[i])}
				if i == 1 || i == 2 {
					// SHOP-050: Buy and Sell draw caption and total as one line.
					bands[0], captions = r, []string{labels[i] + " " + groupedFigure(values[i])}
				}
				// Lit: a button that cannot act keeps the rest ink.
				ink := oracle.rest
				if pressed == i {
					ink = oracle.hover
					bands[0], bands[1] = bands[0].Add(image.Pt(0, 1)), bands[1].Add(image.Pt(0, 1))
				}
				for row, caption := range captions {
					// A joined line wider than its plaque is cut at the right,
					// as every plaque caption is; the oracle cuts it the same way.
					if width, _ := font.Measure(caption); width > r.Dx()-6 && len(captions) == 2 {
						t.Fatalf("%s button %d row %d: %q width %d exceeds well width %d", name, i, row, caption, width, r.Dx()-6)
					}
					oracleTownButtonText(want, font, caption, bands[row], ink)
				}
				if n, points := diffWells(got, want, r); n != 0 {
					t.Fatalf("%s pressed %d button %d: %d pixel differences from centered caption/total rows, first %v", name, pressed, i, n, points)
				}
				comparisons++
			}
		}
	}
	frame := func() *image.RGBA {
		t.Helper()
		x, y, err := app.HeadlessShopIdlePoint()
		if err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessPointer("hover", x, y); err != nil {
			t.Fatal(err)
		}
		pic, _, err := app.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return pic
	}
	zero := s.ShopScreen()
	if zero.Buy != 0 || zero.Sell != 0 {
		t.Fatalf("empty table totals %d/%d, want 0/0", zero.Buy, zero.Sell)
	}
	verify("empty table", zero, frame())
	point := shopPointer(t, app)
	for room, shelf := range shopRoomShelves {
		found := false
		for at, item := range f.Shop.Shelf(shelf.shelf) {
			if item.Price > 0 {
				point("shelf", shopOrderShow(t, app, s, room, at), "press", "release")
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	point("doll", 1, "press", "release")
	for i, cell := range s.ShopScreen().Pack {
		if cell.Occupied() && cell.Price > 0 {
			point("pack", i, "press", "release")
			break
		}
	}
	live := s.ShopScreen()
	if live.Buy <= 0 || live.Sell <= 0 {
		t.Fatalf("App shelf and pack input totals %d/%d, want both positive", live.Buy, live.Sell)
	}
	verify("live table", live, frame())
	wide := live
	wide.Buy, wide.Sell = 1026633, 2147483647
	verify("wide totals", wide, nil)
	t.Logf("selector %d: zero, live %d/%d, and wide 1,026,633/2,147,483,647 totals; %d full-well comparisons", font.Selector, live.Buy, live.Sell, comparisons)
}
