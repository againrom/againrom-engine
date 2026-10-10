package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func tipClaimFrame() *DialogFrame {
	a := &DialogFrame{}
	for i, size := range []image.Point{{48, 32}, {32, 32}, {48, 32}, {32, 32}, {32, 32}, {32, 32}, {32, 32}, {48, 32}, {32, 32}} {
		a.Pieces[i] = image.NewRGBA(image.Rectangle{Max: size})
		draw.Draw(a.Pieces[i], a.Pieces[i].Bounds(), image.NewUniform(color.RGBA{uint8(40 + i*10), 80, 70, 255}), image.Point{}, draw.Src)
	}
	return a
}

func TestTipFramePiecesFillAndPackedShadow(t *testing.T) {
	r := image.Rect(20, 20, 332, 220)
	a := tipClaimFrame()
	background := color.RGBA{248, 252, 248, 255}
	dst := image.NewRGBA(image.Rect(0, 0, 360, 240))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	ComposeTipPanel(dst, TipPanelView{Rect: r, Text: "x", CloseLabel: "X", ToggleLabel: "Y", Font: shopTipTestFont(), Art: &TipPanelArt{Frame: a}})
	for _, tc := range []struct {
		at    image.Point
		piece int
	}{
		{image.Pt(20, 20), 1}, {image.Pt(292, 20), 3},
		{image.Pt(20, 180), 6}, {image.Pt(292, 180), 8},
		{image.Pt(60, 20), 2}, {image.Pt(20, 70), 4},
		{image.Pt(300, 70), 5}, {image.Pt(60, 195), 7},
		{image.Pt(150, 100), 0},
	} {
		if got, want := dst.RGBAAt(tc.at.X, tc.at.Y), a.Pieces[tc.piece].RGBAAt(0, 0); got != want {
			t.Errorf("piece %d at %v = %v, want %v", tc.piece, tc.at, got, want)
		}
	}
	for _, at := range []image.Point{{328, 100}, {150, 216}, {328, 216}} {
		if got, want := dst.RGBAAt(at.X, at.Y), (color.RGBA{156, 157, 156, 255}); got != want {
			t.Errorf("shadow at %v = %v, want packed level six %v", at, got, want)
		}
	}
	if got := dst.RGBAAt(332, 100); got != background {
		t.Errorf("outside panel changed to %v", got)
	}
}

func TestTipChildrenMatchPublishedRoomOffsets(t *testing.T) {
	r := image.Rect(328, 0, 640, 200)
	for _, tc := range []struct {
		name      string
		got, want image.Rectangle
	}{
		{"list", TipPanelListRect(r), image.Rect(348, 24, 612, 164)},
		{"Close", TipPanelCloseRect(r), image.Rect(520, 160, 600, 178)},
		{"checkbox", TipPanelToggleRect(r), image.Rect(368, 160, 516, 176)},
		{"text", TipPanelTextRect(r), image.Rect(348, 24, 612, 164)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
		if !tc.got.In(image.Rect(328, 0, 632, 192)) {
			t.Errorf("%s outside frame body", tc.name)
		}
	}
}

func TestTipFitHeightKeepsOnlyTheRequiredFrameTileRows(t *testing.T) {
	f := shopTipTestFont()
	if got, want := TipPanelFitHeight(f, "one", 312), 104; got != want {
		t.Errorf("one line height = %d, want %d", got, want)
	}
	r := image.Rect(328, 0, 640, 373)
	if got, want := TipPanelShrinkRect(r, f, "one"), image.Rect(328, 0, 640, 104); got != want {
		t.Errorf("tight panel = %v, want %v", got, want)
	}
}
