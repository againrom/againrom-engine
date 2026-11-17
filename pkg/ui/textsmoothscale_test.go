package ui

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

// smoothBar is a row of glyphs, each a solid bar three cells wide and five
// tall, drawn on transparent ground.
func smoothBar() []text.DrawCall {
	g := &text.Glyph{Width: 3, Height: 5, CoverageLevels: true, Pixels: make([]text.Pixel, 15)}
	for n := range g.Pixels {
		g.Pixels[n] = text.Pixel{Level: text.MaxLevel, Painted: true}
	}
	calls := make([]text.DrawCall, 4)
	for i := range calls {
		calls[i] = text.DrawCall{Glyph: g, X: 2 + i*8, Y: 2, Color: color.RGBA{230, 200, 120, 255}}
	}
	return calls
}

// rampWidth is the most pixels in a row of dst that carry an alpha strictly
// between empty and full: a smoothed edge spreads over more than one pixel.
func rampWidth(dst *image.RGBA) int {
	best := 0
	for y := dst.Rect.Min.Y; y < dst.Rect.Max.Y; y++ {
		run := 0
		for x := dst.Rect.Min.X; x < dst.Rect.Max.X; x++ {
			if a := dst.RGBAAt(x, y).A; a > 0 && a < 255 {
				run++
				best = max(best, run)
			} else {
				run = 0
			}
		}
	}
	return best
}

// TestTextOverlayIsSmoothAtFractionalWindowScales checks that at the window
// scales of 1920x1080 and 1.5 a glyph edge fades over more than one window
// pixel, in the game overlay and in the map editor overlay.
func TestTextOverlayIsSmoothAtFractionalWindowScales(t *testing.T) {
	for _, s := range []float64{1.40625, 1.5} {
		size := image.Rect(0, 0, int(40*s)+2, int(10*s)+2)
		game := image.NewRGBA(size)
		textsmooth.Composite(game, smoothBar(), s, 0, 0)
		if w := rampWidth(game); w < 2 {
			t.Errorf("game overlay at scale %v: widest edge fade %d window pixels, want 2 or more", s, w)
		}
		editor := image.NewRGBA(size)
		NewMapEditor(nil, nil, nil).textOverlay.cache.Composite(editor, smoothBar(), s, 0, 0)
		if w := rampWidth(editor); w < 2 {
			t.Errorf("editor overlay at scale %v: widest edge fade %d window pixels, want 2 or more", s, w)
		}
	}
}

// TestWriteGameTextCrop writes the name page text at the 1920x1080 window
// scale to AGAINROM_CROP_DIR as the file named by AGAINROM_CROP_NAME.
func TestWriteGameTextCrop(t *testing.T) {
	dir, name := os.Getenv("AGAINROM_CROP_DIR"), os.Getenv("AGAINROM_CROP_NAME")
	if dir == "" || name == "" {
		t.Skip("AGAINROM_CROP_DIR or AGAINROM_CROP_NAME not set")
	}
	frame, calls := rimCapture(rimNamePage(false, rimFont(false, 5)))
	win, _, _ := rimWindow(frame, calls, 1.40625)
	witness.PNG(t, filepath.Join(dir, name), win)
}
