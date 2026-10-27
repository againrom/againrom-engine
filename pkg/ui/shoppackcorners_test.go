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

// A strip with its own picture shows it under the places, and an empty
// place shows the strip instead of the empty cell's back; without one the
// empty place keeps the back.
func TestShopPackStripPictureShowsUnderEmptyPlaces(t *testing.T) {
	art := testArt()
	art.Back[ShopBackEmpty] = flat(80, 80, color.RGBA{R: 40, A: 0xff})
	place := func(pic *image.RGBA) uint8 {
		r := firstShopLayout.PackCellRect(2)
		return pic.RGBAAt(r.Min.X+40, r.Min.Y+40).R
	}
	v := ShopScreenView{Chosen: -1, Art: art}
	if got := place(ComposeShopScreen(v, image.Point{}, false, nil, false)); got != 40 {
		t.Errorf("no strip picture: empty place R=%d, want the back's 40", got)
	}
	art.Pack = flat(480, 90, color.RGBA{R: 50, A: 0xff})
	pic := ComposeShopScreen(v, image.Point{}, false, nil, false)
	if got := place(pic); got != 50 {
		t.Errorf("strip picture: empty place R=%d, want the strip's 50", got)
	}
	if got := pic.RGBAAt(shopPackRegion.Min.X+1, shopPackRegion.Min.Y+1).R; got != 50 {
		t.Errorf("strip picture: strip corner R=%d, want 50", got)
	}
}

// A layout's label offset moves the label line of a two-line plaque and
// leaves the value line in its band.
func TestShopLabelOffsetMovesTheLabelLine(t *testing.T) {
	r := shopButtonRects[1]
	plain := panelCaptions(r, []string{"Buy", "0"}, true, image.Point{}, image.Point{})
	moved := panelCaptions(r, []string{"Buy", "0"}, true, image.Pt(1, 4), image.Point{})
	if moved[0].Rect != plain[0].Rect.Add(image.Pt(1, 4)) || moved[1].Rect != plain[1].Rect {
		t.Fatalf("label %v value %v, from %v %v", moved[0].Rect, moved[1].Rect, plain[0].Rect, plain[1].Rect)
	}
	l := firstShopLayout
	l.LabelOffset[1] = image.Pt(1, 4)
	panel := shopButtonPanel(ShopScreenView{Layout: &l}, image.Point{}, false)
	face := panel.buttons[1].Face
	if face.Captions[0].Rect != plain[0].Rect.Add(image.Pt(1, 4)) {
		t.Fatalf("the layout's offset did not reach the plaque: %v", face.Captions[0].Rect)
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
