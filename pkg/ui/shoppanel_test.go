package ui

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

func TestShopBuyAndSellTotalsDrawBelowCaptions(t *testing.T) {
	font := townShellRosterTestFont()
	labels := [4]string{"Undo", "Buy", "Sell", "Exit"}
	for _, amount := range []int32{0, 12345, 1026633} {
		for pressed := -1; pressed < 4; pressed++ {
			t.Run(fmt.Sprintf("amount-%d/pressed-%d", amount, pressed), func(t *testing.T) {
				v := ShopScreenView{Font: font, Live: [4]bool{true, true, true, true},
					Words: Words{ShopUndo: labels[0], ShopBuy: labels[1], ShopSell: labels[2], ShopExit: labels[3]},
					Purse: 4567, Buy: amount, Sell: amount, Total: -7654}
				var hover image.Point
				if pressed >= 0 {
					r := shopButtonRects[pressed]
					hover = r.Min.Add(r.Max).Div(2)
					v.Press = ShopControl{Kind: ShopControlButton, Index: pressed}
				}
				calls := text.Record(func() { ComposeShopScreen(v, hover, pressed >= 0, nil, false) })
				values := [4]string{"4,567", GroupDigits(int64(amount)), GroupDigits(int64(amount)), "-7,654"}
				for i, r := range shopButtonRects {
					bands := [2]image.Rectangle{
						image.Rect(r.Min.X, r.Min.Y+2, r.Max.X, r.Min.Y+24),
						image.Rect(r.Min.X, r.Min.Y+23, r.Max.X, r.Max.Y-2),
					}
					for row, want := range []string{labels[i], values[i]} {
						w, h := font.Measure(want)
						x, y := bands[row].Min.X+(bands[row].Dx()-w)/2, bands[row].Min.Y+(bands[row].Dy()-h)/2
						if pressed == i {
							y++
						}
						var line []text.DrawCall
						for _, call := range calls {
							if !call.Flat && call.Y == y && image.Pt(call.X, call.Y).In(r) {
								line = append(line, call)
							}
						}
						if got := recordedText(font, line); got != want {
							t.Fatalf("button %d row %d at y=%d drew %q, want %q", i, row, y, got, want)
						}
						if line[0].X != x {
							t.Fatalf("button %d row %d starts at x=%d, want centered x=%d", i, row, line[0].X, x)
						}
					}
				}
			})
		}
	}
}

// TestShopPanelDrawsButtonBitmapOnlyWhilePressedAndHovered pins TOWN-260:
// ShopMenu.bmp is the panel at rest, and a ShopButton bitmap appears only at
// its own rect while that button is pressed and under the pointer.
func TestShopPanelDrawsButtonBitmapOnlyWhilePressedAndHovered(t *testing.T) {
	panel := color.RGBA{R: 0x40, G: 0x20, B: 0x10, A: 0xff}
	colors := [4]color.RGBA{
		{R: 0xd1, A: 0xff}, {G: 0xd1, A: 0xff}, {B: 0xd1, A: 0xff}, {R: 0xd1, G: 0xd1, A: 0xff},
	}
	art := &ShopScreenArt{Menu: uniform(176, 238, panel)}
	for i, r := range shopButtonRects {
		art.Button[i] = uniform(r.Dx(), r.Dy(), colors[i])
	}
	base := ShopScreenView{Art: art, Live: [4]bool{true, true, true, true}}
	centre := func(i int) image.Point { return shopButtonRects[i].Min.Add(shopButtonRects[i].Max).Div(2) }
	check := func(name string, pic *image.RGBA, drawn int) {
		t.Helper()
		for i, r := range shopButtonRects {
			want := panel
			if i == drawn {
				want = colors[i]
			}
			for y := r.Min.Y; y < r.Max.Y; y++ {
				for x := r.Min.X; x < r.Max.X; x++ {
					if got := pic.RGBAAt(x, y); got != want {
						t.Fatalf("%s: button %d pixel (%d,%d) = %#v, want %#v", name, i, x, y, got, want)
					}
				}
			}
		}
	}
	check("rest", ComposeShopScreen(base, image.Point{}, false, nil, false), -1)
	for i := range shopButtonRects {
		check("hovered only", ComposeShopScreen(base, centre(i), true, nil, false), -1)
		pressed := base
		pressed.Press = ShopControl{Kind: ShopControlButton, Index: i}
		check("pressed and hovered", ComposeShopScreen(pressed, centre(i), true, nil, false), i)
		check("pressed elsewhere", ComposeShopScreen(pressed, centre((i+1)%4), true, nil, false), -1)
	}
}
