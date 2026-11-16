package bmp

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// A paletted picture of an odd width written by EncodePaletted reads back
// through DecodePaletted with the same indices and table; a pixel past the
// table is refused.
func TestEncodePalettedReadsBackThroughTheDecoder(t *testing.T) {
	palette := color.Palette{color.RGBA{1, 2, 3, 0xff}, color.RGBA{200, 100, 50, 0xff}, color.RGBA{0, 0xff, 0, 0xff}}
	img := image.NewPaletted(image.Rect(0, 0, 5, 3), palette)
	for i := range img.Pix {
		img.Pix[i] = uint8(i % len(palette))
	}
	data, err := EncodePaletted(img)
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodePaletted(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Bounds() != img.Bounds() || !bytes.Equal(back.Pix, img.Pix) || len(back.Palette) != len(palette) {
		t.Fatalf("read back %v, %d entries", back.Bounds(), len(back.Palette))
	}
	for i, c := range palette {
		if back.Palette[i] != c {
			t.Errorf("entry %d: %v, want %v", i, back.Palette[i], c)
		}
	}
	img.Pix[4] = 7
	if _, err := EncodePaletted(img); err == nil {
		t.Fatal("a pixel past the table was written")
	}
}
