package ui

import (
	"image"
	"image/color"
	"testing"
)

// Each corner of the shop's bottom strip always shows a picture: the resting
// cap while the strip cannot turn that way, the arrow while it can.
func TestShopPackCornersShowTheCapUntilTheStripCanTurn(t *testing.T) {
	art := testArt()
	resting := color.RGBA{R: 20, A: 0xff}
	arrow := color.RGBA{R: 30, A: 0xff}
	for i := range art.PackCap {
		art.PackCap[i] = flat(32, 88, resting)
		art.PackArrow[i] = flat(32, 88, arrow)
	}
	at := func(pic *image.RGBA, r image.Rectangle) uint8 {
		return pic.RGBAAt(r.Min.X+16, r.Min.Y+40).R
	}
	for _, tc := range []struct {
		name                string
		back, forward       bool
		wantLeft, wantRight uint8
	}{
		{"neither way", false, false, resting.R, resting.R},
		{"back only", true, false, arrow.R, resting.R},
		{"forward only", false, true, resting.R, arrow.R},
		{"both ways", true, true, arrow.R, arrow.R},
	} {
		v := ShopScreenView{Chosen: -1, Art: art, PackBack: tc.back, PackForward: tc.forward}
		pic := ComposeShopScreen(v, image.Point{}, false, nil, false)
		if got := at(pic, shopPackLeftRect); got != tc.wantLeft {
			t.Errorf("%s: left corner R=%d, want %d", tc.name, got, tc.wantLeft)
		}
		if got := at(pic, shopPackRightRect); got != tc.wantRight {
			t.Errorf("%s: right corner R=%d, want %d", tc.name, got, tc.wantRight)
		}
	}
}

func flat(w, h int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i+3 < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = c.R, c.G, c.B, c.A
	}
	return pic
}

func TestShopPackCornerHoverShowsThePointerInsideArrow(t *testing.T) {
	art := testArt()
	for i := range art.PackArrow {
		art.PackArrow[i] = flat(32, 88, color.RGBA{R: uint8(30 + i), A: 0xff})
		art.PackCap[i%2] = flat(32, 88, color.RGBA{R: 20, A: 0xff})
	}
	at := func(pic *image.RGBA, r image.Rectangle) uint8 { return pic.RGBAAt(r.Min.X+16, r.Min.Y+40).R }
	v := ShopScreenView{Chosen: -1, Art: art, PackBack: true, PackForward: true}
	pic := ComposeShopScreen(v, shopPackLeftRect.Min.Add(image.Pt(5, 5)), true, nil, false)
	if got := at(pic, shopPackLeftRect); got != 32 {
		t.Errorf("hovered left arrow R=%d, want 32", got)
	}
	if got := at(pic, shopPackRightRect); got != 31 {
		t.Errorf("unhovered right arrow R=%d, want 31", got)
	}
	v.PackBack = false
	pic = ComposeShopScreen(v, shopPackLeftRect.Min.Add(image.Pt(5, 5)), true, nil, false)
	if got := at(pic, shopPackLeftRect); got != 20 {
		t.Errorf("hovered disabled corner R=%d, want the cap 20", got)
	}
}
