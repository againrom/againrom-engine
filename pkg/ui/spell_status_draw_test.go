package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

type statusDrawRecorder struct {
	alpha []float32
	imgs  []*ebiten.Image
}

func (r *statusDrawRecorder) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	r.imgs = append(r.imgs, img)
	r.alpha = append(r.alpha, op.ColorScale.A())
}

func TestStonePixelsUseTheSourcePaletteAndKeepHoles(t *testing.T) {
	frame := &terrain.StaticFrame{Width: 2, Height: 1, Grayscale: true,
		Pixels:  []terrain.StaticPixel{{Index: 1, Opaque: true}, {}},
		Palette: [256]color.RGBA{1: {R: 200, G: 100, B: 20}}}
	v := &Viewer{}
	got := v.spritePixels(frame, 8)
	if p := got.RGBAAt(0, 0); p != (color.RGBA{R: 106, G: 106, B: 106, A: 255}) {
		t.Fatalf("Stone pixel = %v, want palette average 106", p)
	}
	if p := got.RGBAAt(1, 0); p != (color.RGBA{}) {
		t.Fatalf("Stone hole = %v, want transparent", p)
	}
	key := v.spriteKey(frame, 8)
	if headlessSpritePixels(key).RGBAAt(0, 0) != got.RGBAAt(0, 0) {
		t.Fatal("headless texture uses another colour path")
	}
}

func TestDetectedInvisibleEntityDrawsAtHalfAlphaAndStoneUsesAnotherTexture(t *testing.T) {
	frame := &terrain.StaticFrame{Width: 1, Height: 1, Grayscale: true,
		Pixels:  []terrain.StaticPixel{{Index: 1, Opaque: true}},
		Palette: [256]color.RGBA{1: {R: 200, G: 100, B: 20, A: 255}}}
	art := &terrain.UnitClass{Width: 1, Height: 1, Frames: []*terrain.StaticFrame{frame}}
	v := overlayViewer(t, 3, 3, 96, 96)
	v.SetEntities([]MapEntity{{ID: 1, Cell: image.Pt(1, 1), Art: art, Frame: frame,
		Stone: true, Translucent: true}})
	var rec statusDrawRecorder
	v.drawPlane(&rec)
	if len(rec.alpha) != 1 || rec.alpha[0] != 0.5 {
		t.Fatalf("status sprite alpha calls=%v, want one at exactly 0.5", rec.alpha)
	}
	if len(rec.imgs) != 1 || rec.imgs[0] == v.staticImage(frame) {
		t.Fatal("Stone Curse reused the ordinary colour texture instead of its grayscale texture")
	}
}
