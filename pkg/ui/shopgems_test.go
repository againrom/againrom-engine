package ui

import (
	"image"
	"image/color"
	"testing"
)

// The shelf's two scroll gems follow the grid when the page asks for it: a gem
// whose way the shelf cannot turn is drawn from the dark picture. Without the
// request both gems stay lit.
func TestShelfGemsGoDarkWhereTheShelfCannotTurn(t *testing.T) {
	art := &ShopScreenArt{}
	for i := range art.Arrow {
		pic := image.NewRGBA(image.Rect(0, 0, 72, 32))
		for p := 0; p+3 < len(pic.Pix); p += 4 {
			pic.Pix[p], pic.Pix[p+1], pic.Pix[p+2], pic.Pix[p+3] = uint8(10*(i+1)), 0, 0, 0xff
		}
		art.Arrow[i] = pic
	}
	up, down := shopArrowUpRect.Min.Add(image.Pt(36, 16)), shopArrowDownRect.Min.Add(image.Pt(36, 16))
	for _, tc := range []struct {
		name         string
		gems, bk, fw bool
		wantUp       uint8
		wantDown     uint8
	}{
		{"first page", true, false, true, 30, 20},
		{"middle page", true, true, true, 10, 20},
		{"last page", true, true, false, 10, 40},
		{"gems not requested", false, false, false, 10, 20},
	} {
		v := ShopScreenView{Chosen: -1, Art: art, ShelfGems: tc.gems, ShelfBack: tc.bk, ShelfForward: tc.fw}
		pic := ComposeShopScreen(v, image.Point{}, false, nil, false)
		if got := pic.RGBAAt(up.X, up.Y); got != (color.RGBA{R: tc.wantUp, A: 0xff}) {
			t.Errorf("%s: top gem %v, want red %d", tc.name, got, tc.wantUp)
		}
		if got := pic.RGBAAt(down.X, down.Y); got != (color.RGBA{R: tc.wantDown, A: 0xff}) {
			t.Errorf("%s: bottom gem %v, want red %d", tc.name, got, tc.wantDown)
		}
	}
}
