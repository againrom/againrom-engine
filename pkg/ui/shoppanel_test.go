package ui

import (
	"image"
	"image/color"
	"testing"
)

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
