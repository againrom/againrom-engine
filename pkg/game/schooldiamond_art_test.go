package game

import (
	"fmt"
	"image/color"
	"strings"
	"testing"
)

func addSchoolDiamondFixture(src chargenSource) {
	for i := 0; i < 9; i++ {
		name := fmt.Sprintf("graphics/interface/training/diamond/on%04d.bmp", i)
		src[name] = synthBMPBlack(80, 76, color.RGBA{R: uint8(i + 1), G: 23, A: 255}, 31)
	}
}

func TestLoadSchoolDiamondFramesInArchiveOrderAndOpaque(t *testing.T) {
	src := townSchoolSource()
	addSchoolDiamondFixture(src)
	art, err := LoadTownSchoolArt(ROM1TownDescription(), src)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		pic := art.Scene["diamond"][i]
		if pic == nil || pic.Bounds().Dx() != 80 || pic.Bounds().Dy() != 76 {
			t.Fatalf("frame %d: missing or not 80x76: %v", i, pic)
		}
		if got := color.RGBAModel.Convert(pic.At(40, 40)); got != (color.RGBA{R: uint8(i + 1), G: 23, A: 255}) {
			t.Fatalf("frame %d resolves the wrong archive address: %v", i, got)
		}
		black := 0
		for y := 0; y < 76; y++ {
			for x := 0; x < 80; x++ {
				c := color.RGBAModel.Convert(pic.At(x, y)).(color.RGBA)
				if c.A != 255 {
					t.Fatalf("frame %d keyed pixel at %d,%d: %v", i, x, y, c)
				}
				if c.R == 0 && c.G == 0 && c.B == 0 {
					black++
				}
			}
		}
		if black != 31 {
			t.Fatalf("frame %d lost pure-black pixels: %d, want 31", i, black)
		}
	}
}

func TestLoadSchoolDiamondRejectsMissingAndWrongSizeFrames(t *testing.T) {
	for i := 0; i < 9; i++ {
		name := fmt.Sprintf("graphics/interface/training/diamond/on%04d.bmp", i)
		for _, badSize := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/size=%t", i, badSize), func(t *testing.T) {
				src := townSchoolSource()
				addSchoolDiamondFixture(src)
				if badSize {
					src[name] = synthBMP(79, 76, color.RGBA{A: 255})
				} else {
					delete(src, name)
				}
				art, err := LoadTownSchoolArt(ROM1TownDescription(), src)
				if art != nil || err == nil || !strings.Contains(err.Error(), name) {
					t.Fatalf("bad frame: art=%v error=%v, want nil art and path %q", art, err, name)
				}
			})
		}
	}
}
