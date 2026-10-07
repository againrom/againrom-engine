package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// Expected pixels come from raw BMP bytes and explicit .256 leaves, without
// the portrait converter, figure composer or pane geometry under test.
func TestReleasePortraitPaneTransparencyAndFeet(t *testing.T) {
	f := releaseFront(t)
	readBMP := func(addr string) *image.RGBA {
		b, err := f.Archives.Containers.ReadFile(addr)
		if err != nil {
			t.Fatal(err)
		}
		return inspectionBMP(t, b)
	}
	background := readBMP("graphics/interface/humanbackr.bmp")
	for _, tc := range []struct {
		name      string
		got, want *image.RGBA
		key       bool
	}{
		{"goblin", loadPortrait(f.Archives.Containers, "graphics/infowindow/Goblin.bmp"), readBMP("graphics/infowindow/Goblin.bmp"), true},
		{"orc", loadPortrait(f.Archives.Containers, "graphics/infowindow/Orc.bmp"), readBMP("graphics/infowindow/Orc.bmp"), true},
		{"human", composePortraitWitnessHuman(f), inspectionSprite(t, f, "graphics/equipment/mfighter/17.256"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == nil || tc.got.Bounds() != image.Rect(0, 0, 160, 240) || tc.want.Bounds() != tc.got.Bounds() {
				t.Fatal("missing or mis-sized figure")
			}
			view := ui.TownCharacterView{HasSubject: true, Figure: tc.got,
				PaneRect: image.Rect(16, 0, 176, 242), FigurePane: f.characterPanes().Figure}
			out := image.NewRGBA(image.Rect(0, 0, 176, 242))
			ui.DrawCharacterPaneBody(out, view)
			transparent, feet, mismatches := 0, 0, 0
			for y := 0; y < 240; y++ {
				for x := 0; x < 160; x++ {
					want := tc.want.RGBAAt(x, y)
					if want.A == 0 || tc.key && want.R|want.G|want.B == 0 {
						transparent++
						want = background.RGBAAt(x, y+2)
					} else if y >= 200 {
						feet++
					}
					if got := out.RGBAAt(x+16, y+2); got != want {
						if mismatches == 0 {
							t.Errorf("source %d,%d: composed %v, want %v", x, y, got, want)
						}
						mismatches++
					}
				}
			}
			if transparent < 1000 || feet < 30 || mismatches != 0 {
				t.Fatalf("background pixels=%d, opaque bottom pixels=%d, mismatches=%d", transparent, feet, mismatches)
			}
			t.Logf("all 38400 source pixels checked; transparent=%d, bottom=%d", transparent, feet)
		})
	}
}

func composePortraitWitnessHuman(f *FrontEnd) *image.RGBA {
	pic, _ := composeUnitFigure(f.Archives.Containers, data.Equipment{}, figureID{Dir: data.FigureDirManFighter, Face: 17})
	return pic
}

// A 24-bit source has no alpha channel. The portrait blit nevertheless uses
// black as the transparent colour; retain the independent BMP reader for
// opaque interface backgrounds as well.
func inspectionPortraitBMP(t *testing.T, b []byte) *image.RGBA {
	pic := inspectionBMP(t, b)
	for y := 0; y < pic.Bounds().Dy(); y++ {
		for x := 0; x < pic.Bounds().Dx(); x++ {
			if pic.RGBAAt(x, y) == (color.RGBA{A: 255}) {
				pic.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return pic
}
