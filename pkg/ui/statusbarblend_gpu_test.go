//go:build statusbargpu && !dialoguegpu && !terrainseamgpu

package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

type statusBarGPUProbe struct {
	done  bool
	err   error
	cases int
}

func (g *statusBarGPUProbe) Layout(_, _ int) (int, int) { return 32, 32 }
func (g *statusBarGPUProbe) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}

func (g *statusBarGPUProbe) Draw(_ *ebiten.Image) {
	if g.done {
		return
	}
	g.done = true
	base := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for word := 0; word < 65536; word++ {
		base.SetRGBA(word%256, word/256, color.RGBA{uint8(word / 2048 * 255 / 31), uint8(word / 32 % 64 * 255 / 63), uint8(word % 32 * 255 / 31), 255})
	}
	var v Viewer
	defer func() {
		if v.statusBarScratch != nil {
			v.statusBarScratch.Dispose()
		}
	}()
	for _, source := range []color.RGBA{{128, 0, 0, 255}, {255, 0, 0, 255}, {192, 0, 0, 255}, {128, 128, 0, 255}, {255, 255, 0, 255}, {192, 192, 0, 255}, {0, 128, 0, 255}, {0, 255, 0, 255}, {0, 192, 0, 255}, {0, 0, 128, 255}, {0, 0, 255, 255}, {0, 0, 192, 255}} {
		actual := ebiten.NewImageFromImage(base)
		v.drawStatusBarHalfRect(actual, screenRect{W: 256, H: 256}, source)
		pixels := make([]byte, len(base.Pix))
		actual.ReadPixels(pixels)
		for word := 0; word < 65536; word++ {
			r, gc, b := word/2048, word/32%64, word%32
			want := [4]byte{uint8((r/2 + int(source.R)/8/2) * 255 / 31), uint8((gc/2 + int(source.G)/4/2) * 255 / 63), uint8((b/2 + int(source.B)/8/2) * 255 / 31), 255}
			for channel := range 4 {
				if pixels[word*4+channel] != want[channel] && g.err == nil {
					g.err = fmt.Errorf("source=%v word=%04x got=%v want=%v", source, word, pixels[word*4:word*4+4], want)
				}
			}
		}
		actual.Dispose()
		g.cases++
	}
	actual := ebiten.NewImageFromImage(base)
	expected := image.NewRGBA(base.Bounds())
	copy(expected.Pix, base.Pix)
	for _, r := range []screenRect{{X: 7.25, Y: 5.75, W: 19.5, H: 9.5}, {X: 12, Y: 8, W: 7, H: 4}, {X: -3, Y: 250, W: 12, H: 20}} {
		source := color.RGBA{255, 255, 0, 255}
		v.drawStatusBarHalfRect(actual, r, source)
		box := image.Rect(int(r.X+0.5), int(r.Y+0.5), int(r.X+r.W+0.5), int(r.Y+r.H+0.5)).Intersect(base.Bounds())
		// The fractional inputs avoid equality at pixel centres.
		if r.X < 0 {
			box.Min.X = 0
		}
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				c := expected.RGBAAt(x, y)
				expected.SetRGBA(x, y, sbPackedBlend(source, c))
			}
		}
	}
	pixels := make([]byte, len(base.Pix))
	actual.ReadPixels(pixels)
	actual.Dispose()
	for i, want := range expected.Pix {
		if pixels[i] != want && g.err == nil {
			g.err = fmt.Errorf("overlap/clipping byte %d got=%d want=%d", i, pixels[i], want)
		}
	}
	g.cases++
}

var statusBarGPUResult statusBarGPUProbe

func TestMain(m *testing.M) {
	ebiten.SetWindowSize(32, 32)
	ebiten.SetWindowPosition(-32000, -32000)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGameWithOptions(&statusBarGPUResult, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true}); err != nil && err != ebiten.Termination {
		statusBarGPUResult.err = err
	}
	os.Exit(m.Run())
}

func TestStatusBarGPUReadback(t *testing.T) {
	if statusBarGPUResult.err != nil {
		t.Fatal(statusBarGPUResult.err)
	}
	if statusBarGPUResult.cases != 13 {
		t.Fatalf("GPU cases=%d want13", statusBarGPUResult.cases)
	}
	t.Log("12 source colours over all 65536 RGB565 destinations and overlap/clipping match exact pixels")
}
