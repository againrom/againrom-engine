package game

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// townSquareArtPrefix is where the ROM1 description keys the square's
// bitmaps.
const townSquareArtPrefix = graphicsPrefix + "interface/town/"

// squareMaskCodes is the description's required mask bytes.
func squareMaskCodes() []uint8 {
	var out []uint8
	for _, b := range ROM1TownDescription().Mask.RequiredBytes {
		out = append(out, uint8(b))
	}
	return out
}

func townSquareSource() chargenSource {
	src := chargenSource{
		townSquareArtPrefix + "townmain.bmp": synthBMP(640, 480, color.RGBA{R: 0x22, A: 0xff}),
		townSquareArtPrefix + "town_add.bmp": synthBMP(552, 92, color.RGBA{G: 0x22, A: 0xff}),
		townSquareArtPrefix + "townmask.bmp": synthBMP8(640, 480, squareMaskCodes()...),
		townSquareArtPrefix + "shop_l.bmp":   synthBMP(52, 76, color.RGBA{B: 0x22, A: 0xff}),
		townSquareArtPrefix + "tavern_l.bmp": synthBMP(28, 64, color.RGBA{B: 0x33, A: 0xff}),
		townSquareArtPrefix + "trener_l.bmp": synthBMP(140, 116, color.RGBA{B: 0x44, A: 0xff}),
	}
	return src
}

func TestLoadTownSquareArtReadsTheCompletePicture(t *testing.T) {
	src := townSquareSource()
	got, err := LoadTownSquareArt(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Pictures("base")) != 1 || len(got.Pictures("overlay")) != 1 || got.Mask == nil {
		t.Fatal("the base picture, its overlay strip or its mask was not retained")
	}
	if w, h := got.Pictures("base")[0].Bounds().Dx(), got.Pictures("base")[0].Bounds().Dy(); w != 640 || h != 480 {
		t.Fatalf("background is %dx%d, want 640x480", w, h)
	}
	if w, h := got.Pictures("overlay")[0].Bounds().Dx(), got.Pictures("overlay")[0].Bounds().Dy(); w != 552 || h != 92 {
		t.Fatalf("add is %dx%d, want 552x92", w, h)
	}
	wantSizes := [3][2]int{{52, 76}, {28, 64}, {140, 116}}
	for i, want := range wantSizes {
		var lbl image.Image
		if pics := got.Pictures([3]string{"label-shop", "label-tavern", "label-school"}[i]); len(pics) == 1 {
			lbl = pics[0]
		}
		if lbl == nil {
			t.Fatalf("label %d was not retained", i)
		}
		if w, h := lbl.Bounds().Dx(), lbl.Bounds().Dy(); w != want[0] || h != want[1] {
			t.Fatalf("label %d is %dx%d, want %dx%d", i, w, h, want[0], want[1])
		}
	}
}

// A missing node is an address-bearing error beside a nil result, so
// NewFrontEnd can keep a broken install playable with the row-button
// fallback (LoadTownSchoolArt's own rule).
func TestLoadTownSquareArtMissingNodeIsAddressBearing(t *testing.T) {
	src := townSquareSource()
	delete(src, townSquareArtPrefix+"trener_l.bmp")
	if _, err := LoadTownSquareArt(src); err == nil || !strings.Contains(err.Error(), "trener_l.bmp") {
		t.Fatalf("missing label error = %v, want an error naming trener_l.bmp", err)
	}
}

// A mis-sized node is refused rather than silently stretched or cropped.
func TestLoadTownSquareArtMisSizedNodeIsRefused(t *testing.T) {
	src := townSquareSource()
	src[townSquareArtPrefix+"townmain.bmp"] = synthBMP(640, 400, color.RGBA{A: 0xff})
	if _, err := LoadTownSquareArt(src); err == nil || !strings.Contains(err.Error(), "townmain.bmp") ||
		!strings.Contains(err.Error(), "640x400") {
		t.Fatalf("mis-sized background error = %v, want an error naming townmain.bmp and its wrong size", err)
	}
}

// A mask missing one of the five significant codes loads and leaves the
// whole square inert at runtime (chargenMaskCodes' own rule for the school),
// so LoadTownSquareArt refuses it instead.
func TestLoadTownSquareArtRefusesAMaskMissingASignificantCode(t *testing.T) {
	src := townSquareSource()
	src[townSquareArtPrefix+"townmask.bmp"] = synthBMP8(640, 480, squareMaskCodes()[1:]...)
	if _, err := LoadTownSquareArt(src); err == nil || !strings.Contains(err.Error(), "missing required mask index") {
		t.Fatalf("incomplete mask error = %v, want a missing-index error", err)
	}
}
