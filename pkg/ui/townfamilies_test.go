package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/frame"
)

var (
	familyHorse   = color.RGBA{R: 0xff, A: 0xff}
	familyBaba    = color.RGBA{G: 0xff, A: 0xff}
	familyDervish = color.RGBA{B: 0xff, A: 0xff}
	familyBase    = color.RGBA{R: 7, G: 7, B: 7, A: 255}
)

func familyArt(horse, baba, dervish image.Rectangle) *TownSquareArt {
	art := &TownSquareArt{
		Background: townAmbientPatch(image.Rect(0, 0, 640, 480), familyBase),
		Exterior:   &TownExteriorArt{},
	}
	for p := range art.Exterior.Horse {
		for v := range art.Exterior.Horse[p] {
			art.Exterior.Horse[p][v] = []image.Image{townAmbientPatch(horse, familyHorse), townAmbientPatch(horse, color.RGBA{R: 0x80, A: 0xff})}
		}
	}
	for p := range art.Exterior.Baba {
		for v := range art.Exterior.Baba[p] {
			art.Exterior.Baba[p][v] = []image.Image{townAmbientPatch(baba, familyBaba)}
		}
	}
	for p := range art.Exterior.Dervish {
		art.Exterior.Dervish[p] = []image.Image{townAmbientPatch(dervish, familyDervish)}
	}
	return art
}

// The three families draw at their table rectangles, horse first, baba next
// and dervish last, over the static picture (TOWN-489, TOWN-490).
func TestTownFamiliesComposeAtTableOriginsInPainterOrder(t *testing.T) {
	art := familyArt(image.Rect(0, 0, 100, 100), image.Rect(0, 0, 100, 100), image.Rect(0, 0, 10, 10))
	frame := &TownExteriorFrame{
		Horse:   TownFamilyFrame{Visible: true, Position: 3, Sheet: 1},
		Baba:    TownFamilyFrame{Visible: true, Position: 2, Sheet: 0},
		Dervish: TownFamilyFrame{Visible: true, Position: 2, Sheet: 0},
	}
	pix := ComposeTownSquare(TownSquareView{Art: art, Exterior: frame})
	for _, c := range []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{"horse origin (448,400)", 448, 400, familyHorse},
		{"horse before its origin", 447, 400, familyBase},
		{"baba origin (384,424)", 384, 424, familyBaba},
		{"baba over horse", 460, 440, familyBaba},
		{"dervish origin (392,420)", 392, 420, familyDervish},
		{"dervish over baba", 395, 425, familyDervish},
		{"baba beside dervish", 402, 425, familyBaba},
	} {
		if got := pix.RGBAAt(c.x, c.y); got != c.want {
			t.Errorf("%s: %v want %v", c.name, got, c.want)
		}
	}
}

// Every position has its own rectangle (TOWN-490).
func TestTownFamilyOriginsAreTheCompiledTables(t *testing.T) {
	art := familyArt(image.Rect(0, 0, 4, 4), image.Rect(0, 0, 4, 4), image.Rect(0, 0, 4, 4))
	horse := [5][2]int{{104, 404}, {104, 404}, {256, 344}, {448, 400}, {140, 400}}
	baba := [4][2]int{{216, 364}, {308, 424}, {384, 424}, {580, 384}}
	dervish := [4][2]int{{224, 364}, {324, 424}, {392, 420}, {592, 388}}
	for p, o := range horse {
		pix := ComposeTownSquare(TownSquareView{Art: art, Exterior: &TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: p}}})
		if pix.RGBAAt(o[0], o[1]) != familyHorse || pix.RGBAAt(o[0]-1, o[1]) != familyBase || pix.RGBAAt(o[0]+3, o[1]+3) != familyHorse {
			t.Errorf("horse position %d not at %v", p+1, o)
		}
	}
	for p, o := range baba {
		pix := ComposeTownSquare(TownSquareView{Art: art, Exterior: &TownExteriorFrame{Baba: TownFamilyFrame{Visible: true, Position: p}}})
		if pix.RGBAAt(o[0], o[1]) != familyBaba || pix.RGBAAt(o[0], o[1]-1) != familyBase {
			t.Errorf("baba position %d not at %v", p+1, o)
		}
	}
	for p, o := range dervish {
		pix := ComposeTownSquare(TownSquareView{Art: art, Exterior: &TownExteriorFrame{Dervish: TownFamilyFrame{Visible: true, Position: p}}})
		if pix.RGBAAt(o[0], o[1]) != familyDervish || pix.RGBAAt(o[0], o[1]-1) != familyBase {
			t.Errorf("dervish position %d not at %v", p+1, o)
		}
	}
}

// A zero-valued descriptor, an absent sheet and an out-of-range frame draw
// nothing; frame 0 is a real picture and the selected frame is drawn.
func TestTownFamiliesDrawNothingWithoutVisibleSheetFrame(t *testing.T) {
	art := familyArt(image.Rect(0, 0, 8, 8), image.Rect(0, 0, 8, 8), image.Rect(0, 0, 8, 8))
	at := func(f *TownExteriorFrame, a *TownSquareArt) color.RGBA {
		return ComposeTownSquare(TownSquareView{Art: a, Exterior: f}).RGBAAt(104, 404)
	}
	if got := at(&TownExteriorFrame{}, art); got != familyBase {
		t.Fatalf("zero descriptor painted %v", got)
	}
	if got := at(&TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: 0, Sheet: 0, Frame: 0}}, art); got != familyHorse {
		t.Fatalf("frame 0 = %v", got)
	}
	if got := at(&TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: 0, Sheet: 0, Frame: 1}}, art); got != (color.RGBA{R: 0x80, A: 0xff}) {
		t.Fatalf("frame 1 = %v", got)
	}
	if got := at(&TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: 0, Sheet: 0, Frame: 2}}, art); got != familyBase {
		t.Fatalf("frame past the sheet painted %v", got)
	}
	art.Exterior.Horse[0][0] = nil
	if got := at(&TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: 0, Sheet: 0}}, art); got != familyBase {
		t.Fatalf("absent sheet painted %v", got)
	}
	if got := at(&TownExteriorFrame{Horse: TownFamilyFrame{Visible: true, Position: 9}}, art); got != familyBase {
		t.Fatalf("out-of-range position painted %v", got)
	}
}

// The view origin is half the excess over 640x480 and equals the native fit's
// centring (TOWN-503).
func TestTownViewOriginIsHalfTheExcessOver640x480(t *testing.T) {
	for _, c := range []struct{ w, h, x, y int }{
		{640, 480, 0, 0}, {800, 600, 80, 60}, {1024, 768, 192, 144},
	} {
		want := image.Pt(c.x, c.y)
		if got := TownViewOrigin(c.w, c.h); got != want {
			t.Errorf("%dx%d origin %v want %v", c.w, c.h, got, want)
		}
		x, y, ok := frame.FitDown(640, 480, c.w, c.h).FrameToWindow(image.Pt(0, 0))
		if !ok || image.Pt(x, y) != want {
			t.Errorf("%dx%d native fit places the frame at (%d,%d) ok=%v", c.w, c.h, x, y, ok)
		}
	}
}
