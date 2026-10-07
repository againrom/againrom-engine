package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

// uniformImage is a fixed-colour synthetic body or seam, sized so a wrong
// source origin or a wrong destination rect is visible in the result rather
// than accidentally correct by symmetry.
func uniformImage(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return img
}

func fillDst(w, h int, c color.RGBA) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return dst
}

func everyPixelIs(t *testing.T, dst *image.RGBA, r image.Rectangle, want color.RGBA) {
	t.Helper()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if got := dst.RGBAAt(x, y); got != want {
				t.Fatalf("pixel (%d,%d) = %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestDrawTownPaneBlitsBodyOpaque(t *testing.T) {
	bodyC := color.RGBA{R: 10, G: 20, B: 30, A: 128}
	sentinel := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	dst := fillDst(20, 20, sentinel)
	bodyRect := image.Rect(0, 0, 20, 20)
	p := TownPane{Body: uniformImage(20, 20, bodyC)}

	drawTownPane(dst, p, bodyRect, image.Rectangle{})

	everyPixelIs(t, dst, bodyRect, bodyC)
}

func TestDrawTownPaneBlitsSeamKeyed(t *testing.T) {
	before := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	opaque := color.RGBA{R: 200, G: 100, B: 50, A: 255}

	dst := fillDst(20, 20, before)
	seamRect := image.Rect(0, 0, 20, 20)
	seam := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(seam, image.Rect(0, 0, 10, 20), &image.Uniform{C: opaque}, image.Point{}, draw.Src)
	// The right half stays the zero value: R=G=B=A=0, keyBlack's own output
	// on a pure-black source pixel.
	p := TownPane{Seam: seam}

	drawTownPane(dst, p, image.Rectangle{}, seamRect)

	everyPixelIs(t, dst, image.Rect(0, 0, 10, 20), opaque)
	everyPixelIs(t, dst, image.Rect(10, 0, 20, 20), before)
}

// TestDrawTownPaneNilSeamLeavesSeamRectUntouched is a TownPane whose
// neighbouring content already covers its own 16 columns (the tavern's own
// two left slots, 1021 spec B1): a nil Seam must draw nothing at seamRect,
// leaving whatever the caller composed there first exactly as it was.
func TestDrawTownPaneNilSeamLeavesSeamRectUntouched(t *testing.T) {
	bodyC := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	sentinel := color.RGBA{R: 1, G: 2, B: 3, A: 255}

	dst := fillDst(40, 20, sentinel)
	bodyRect := image.Rect(0, 0, 20, 20)
	seamRect := image.Rect(20, 0, 40, 20)
	p := TownPane{Body: uniformImage(20, 20, bodyC)}

	drawTownPane(dst, p, bodyRect, seamRect)

	everyPixelIs(t, dst, bodyRect, bodyC)
	everyPixelIs(t, dst, seamRect, sentinel)
}

// TestDrawTownPaneNilBodyLeavesBodyRectForCallersFallback is the same rule
// the other way: a nil Body (a caller predating 1021, or a load failure)
// draws nothing at all, so DrawTownCharacterRegion's own authored-fill
// fallback (tested below) is what a reader actually sees, not a half-drawn
// pane.
func TestDrawTownPaneNilBodyLeavesBodyRectForCallersFallback(t *testing.T) {
	sentinel := color.RGBA{R: 9, G: 9, B: 9, A: 255}
	dst := fillDst(20, 20, sentinel)
	bodyRect := image.Rect(0, 0, 20, 20)

	drawTownPane(dst, TownPane{}, bodyRect, image.Rectangle{})

	everyPixelIs(t, dst, bodyRect, sentinel)
}

func TestDrawTownCharacterRegionFallsBackToAuthoredFillWithNilPane(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 640, 480))
	DrawTownCharacterRegion(dst, TownCharacterView{})
	// The fallback fill's own interior, away from the 1px outline and away
	// from the always-drawn chevrons/mode box/name row this function also
	// paints over the region: a single interior pixel is enough to tell the
	// authored fill from a shipped body, since no shipped body in this
	// story's own art is a flat single colour there (drawTownPane's own
	// opaque-blit test above already covers the shipped-body path).
	p := image.Pt(500, 250)
	if !p.In(TownCharacterRegion) {
		t.Fatalf("test point %v is not inside TownCharacterRegion %v", p, TownCharacterRegion)
	}
	if got, want := dst.RGBAAt(p.X, p.Y), townShellPanel; got != want {
		t.Fatalf("DrawTownCharacterRegion with a zero Pane painted %v at %v, want the authored fill %v", got, p, want)
	}
}
