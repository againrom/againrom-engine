package ui

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/render/terrain"
)

func treeFrame() *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: 40, Height: 40, Pixels: make([]terrain.StaticPixel, 1600)}
	f.Palette[1] = color.RGBA{255, 0, 255, 255}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: 1, Opaque: true}
	}
	return f
}

func countColour(pic *image.RGBA, r image.Rectangle, c color.RGBA) int {
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if pic.RGBAAt(x, y) == c {
				n++
			}
		}
	}
	return n
}

func TestModWindowDrawsNoPictureOfASelectedRecord(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	host := e.doc.Edits.(*modesWorkspace)
	host.overview, host.graph = modesOverview, modesGraph()
	e.selected = 0
	e.doc.Records[0].Preview = treeFrame()
	_ = e.Key("package")
	_ = e.Key("package")
	if !e.modWindow || e.modPage != modPageGraph {
		t.Fatalf("W is on page %d, open %v", e.modPage, e.modWindow)
	}
	s, _, ok := e.modSpec()
	if !ok {
		t.Fatal("W has no spec")
	}
	rect := layoutPopup(s, e.font, e.bodyFont()).Rect
	pic := e.chromeCompose(false)
	if file := os.Getenv("AGAINROM_MOD_WINDOW_CROP"); filepath.IsAbs(file) {
		witness.PNG(t, file, pic.SubImage(rect.Intersect(pic.Bounds())))
	}
	if n := countColour(pic, rect, color.RGBA{255, 0, 255, 255}); n != 0 {
		t.Fatalf("W draws %d pixels of the selected record's picture", n)
	}
}

func TestObjectWindowDrawsTheSelectedRecordsPicture(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.selected = 0
	e.doc.Records[0].Preview = treeFrame()
	e.openObjectRecord(0, false)
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("the object window has no spec")
	}
	pic := image.NewRGBA(image.Rect(0, 0, e.window.X, e.window.Y))
	e.drawWindow(pic, s)
	e.drawObjectPreview(pic, s)
	if n := countColour(pic, pic.Bounds(), color.RGBA{255, 0, 255, 255}); n == 0 {
		t.Fatal("the object window draws no picture of its record")
	}
}
