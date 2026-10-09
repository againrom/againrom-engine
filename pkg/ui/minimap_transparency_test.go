package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestCrystalBlackPixelsPreserveTheMissionUnderlay(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)
	v.Layout(MissionFrameW, MissionFrameH)
	v.DeferPointer(true)
	black := color.RGBA{A: 255}
	bevel := color.RGBA{G: 180, B: 30, A: 255}
	nearBlack := color.RGBA{G: 1, A: 255}
	texture := color.RGBA{G: 60, B: 7, A: 255}
	seam := solidPic(16, 158, black)
	crystal := solidPic(160, 158, black)
	seam.SetRGBA(8, 60, bevel)
	crystal.SetRGBA(40, 4, bevel)
	crystal.SetRGBA(120, 4, nearBlack)
	crystal.SetRGBA(72, 79, texture)
	v.SetDialogFrame(&DialogFrame{Minimap: crystal, MinimapSeam: seam})
	v.SetFog(make([]byte, cliffW*cliffH), cliffW, cliffH)
	pic, at, ok := v.minimapPresent()
	if !ok {
		t.Fatal("missing crystal")
	}
	world := color.RGBA{R: 91, G: 123, B: 57, A: 255}
	frame := image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
	draw.Draw(frame, frame.Bounds(), image.NewUniform(world), image.Point{}, draw.Src)
	screen := ebiten.NewImage(v.frameW, v.frameH)
	defer screen.Dispose()
	v.Draw(screen)
	if len(v.canvasLog.ops) == 0 || v.canvasLog.ops[0].kind != pixelUnknown || v.canvasLog.ops[0].rect != frame.Bounds() {
		t.Fatal("mission draw did not retain its world underlay")
	}
	v.canvasLog.ops[0] = pixelOp{kind: pixelReplace, rect: frame.Bounds(), pic: frame}
	if pic.Bounds() != image.Rect(0, 0, 176, 158) || at != image.Pt(v.frameW-176, 0) {
		t.Fatal("crystal placement changed", pic.Bounds(), at)
	}
	for _, tc := range []struct {
		name string
		p    image.Point
		want color.RGBA
	}{
		{"left of bevel", image.Pt(1, 60), world},
		{"above crystal", image.Pt(10, 1), world},
		{"column background", image.Pt(40, 1), invFill},
		{"left art", image.Pt(8, 60), bevel},
		{"right art", image.Pt(56, 4), bevel},
		{"near black art", image.Pt(136, 4), nearBlack},
		{"unseen cell texture", image.Pt(88, 79), texture},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := at.Add(tc.p)
			if got, known := v.canvasLog.value(len(v.canvasLog.ops), p.X, p.Y); !known || got != tc.want {
				t.Fatalf("mission pixel %v = %v (known %v), want %v", p, got, known, tc.want)
			}
		})
	}
}
