package game

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func checkInstalledDialoguePointer(t *testing.T, a *ui.App, route string, button image.Rectangle, frameSize image.Point, read func() *image.RGBA) {
	t.Helper()
	center := button.Min.Add(button.Max).Div(2)
	outside := button.Min.Sub(image.Pt(10, 10))
	edge := button.Min.Add(image.Pt(2, 0))
	pointer := func(action string, p image.Point) {
		t.Helper()
		if err := a.HeadlessPointer(action, p.X, p.Y); err != nil {
			t.Fatal(route, err)
		}
	}
	pixels := func() *image.RGBA {
		t.Helper()
		if read != nil {
			return read()
		}
		a.Draw(ebiten.NewImage(frameSize.X, frameSize.Y))
		out := image.NewRGBA(button)
		for y := button.Min.Y; y < button.Max.Y; y++ {
			for x := button.Min.X; x < button.Max.X; x++ {
				c, ok := a.DialogueBackdropPixel(x, y)
				if !ok {
					t.Fatalf("%s submission pixel missing %d,%d", route, x, y)
				}
				out.SetRGBA(x, y, c)
			}
		}
		return out
	}
	pointer("hover", outside)
	idle := pixels()
	light, dark := color.RGBA{41, 68, 57, 255}, color.RGBA{0, 12, 8, 255}
	if got := idle.RGBAAt(edge.X, edge.Y); got != light {
		t.Fatalf("%s idle bevel %v want%v", route, got, light)
	}
	pointer("hover", center)
	hover := pixels()
	differences := 0
	for y := button.Min.Y + 3; y < button.Max.Y-3; y++ {
		for x := button.Min.X + 3; x < button.Max.X-3; x++ {
			if idle.RGBAAt(x, y) != hover.RGBAAt(x, y) {
				differences++
			}
		}
	}
	if differences < 10 {
		t.Fatalf("%s missing reached hover ink: %d changed label cells", route, differences)
	}
	pointer("press", center)
	pressed := pixels()
	if got := pressed.RGBAAt(edge.X, edge.Y); got != dark {
		t.Fatalf("%s press inside bevel %v want%v", route, got, dark)
	}
	pointer("move", outside)
	dragged := pixels()
	if got := dragged.RGBAAt(edge.X, edge.Y); got != light {
		t.Fatalf("%s press outside bevel %v want%v", route, got, light)
	}
	pointer("release", outside)
	pointer("press", outside)
	pointer("move", center)
	unarmed := pixels()
	if got := unarmed.RGBAAt(edge.X, edge.Y); got != dark {
		t.Fatalf("%s independent press/membership bevel %v want%v", route, got, dark)
	}
	pointer("release", center)
	pointer("hover", outside)
	t.Logf("%s reached hover%d label cells, pressed inside/outside, unarmed release preserves page", route, differences)
}
