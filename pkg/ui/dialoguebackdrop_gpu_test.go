//go:build dialoguegpu && !terrainseamgpu

package ui

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"againrom/pkg/render/backdrop"
	"github.com/hajimehoshi/ebiten/v2"
)

type dialogueGPUProbe struct {
	done  bool
	err   error
	cases int
}

func (g *dialogueGPUProbe) Layout(_, _ int) (int, int) { return 64, 48 }
func (g *dialogueGPUProbe) Update() error {
	if g.done {
		return ebiten.Termination
	}
	return nil
}

func dialogueGPUOffsetTexture(pic *image.RGBA, at image.Point) *ebiten.Image {
	canvas := image.NewRGBA(image.Rect(0, 0, at.X+pic.Rect.Dx()+7, at.Y+pic.Rect.Dy()+9))
	for y := 0; y < pic.Rect.Dy(); y++ {
		for x := 0; x < pic.Rect.Dx(); x++ {
			canvas.SetRGBA(at.X+x, at.Y+y, pic.RGBAAt(pic.Rect.Min.X+x, pic.Rect.Min.Y+y))
		}
	}
	return ebiten.NewImageFromImage(canvas).SubImage(image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}).(*ebiten.Image)
}

func (g *dialogueGPUProbe) Draw(_ *ebiten.Image) {
	if g.done {
		return
	}
	g.done = true
	base := image.NewRGBA(image.Rect(0, 0, 64, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 64; x++ {
			base.SetRGBA(x, y, color.RGBA{uint8(96 + x*2), uint8(100 + y*3), uint8(240 - x), 255})
		}
	}
	check := func(name string, actual *ebiten.Image, expected *image.RGBA) {
		pixels := make([]byte, len(expected.Pix))
		actual.ReadPixels(pixels)
		g.cases++
		if !bytes.Equal(pixels, expected.Pix) && g.err == nil {
			for n := 0; n < len(pixels); n += 4 {
				if !bytes.Equal(pixels[n:n+4], expected.Pix[n:n+4]) {
					g.err = fmt.Errorf("%s actual GPU pixel %d,%d=%v, expected %v", name, n/4%64, n/4/64, pixels[n:n+4], expected.Pix[n:n+4])
					break
				}
			}
		}
	}
	for _, layout := range []backdrop.Layout{backdrop.RGB565, backdrop.RGB555} {
		for _, mode := range []backdrop.Mode{backdrop.Full, backdrop.Reduced} {
			for _, clipped := range []bool{false, true} {
				p := DialogueBackdrop{Layout: layout, Mode: mode, Clipped: clipped, Clip: image.Rect(9, 7, 51, 37), FrameClipped: clipped, FrameClip: image.Rect(13, 9, 45, 32)}
				for _, shows := range []uint64{1, 2} {
					s := dialogueBackdropState{policy: p}
					l := s.table()
					s.texture = dialogueGPUOffsetTexture(l.Texture(), image.Pt(37, 41))
					actual := ebiten.NewImageFromImage(base)
					var log pixelLog
					log.reset(base.Bounds())
					log.upload(base)
					s.drawWithCalls(actual, &log, shows, nil)
					expected := &image.RGBA{Pix: bytes.Clone(base.Pix), Stride: base.Stride, Rect: base.Rect}
					s.apply(expected, shows)
					check(fmt.Sprintf("backdrop layout=%d mode=%d clip=%v shows=%d", layout, mode, clipped, shows), actual, expected)
				}
				s := dialogueBackdropState{policy: p}
				art := &DialogFrame{}
				for i := range art.Pieces {
					pic := image.NewRGBA(image.Rect(0, 0, 4, 4))
					for y := 0; y < 4; y++ {
						for x := 0; x < 4; x++ {
							if (x+y)%3 != 0 {
								pic.SetRGBA(x, y, color.RGBA{1, 1, 1, 255})
							}
						}
					}
					art.Pieces[i] = pic
				}
				body := image.Rect(8, 6, 48, 34)
				mask := dialogueShadowMask(art, body, image.Point{}, 1, dialoguePolicyClip(p, base.Bounds()))
				s.shadowTexture = dialogueGPUOffsetTexture(s.shadowTable().Texture(), image.Pt(37, 41))
				s.shadowMaskTexture = dialogueGPUOffsetTexture(mask, image.Pt(7, 9))
				actual := ebiten.NewImageFromImage(base)
				var log pixelLog
				log.reset(base.Bounds())
				log.upload(base)
				s.drawFrameShadows(actual, &log, art, body, image.Point{}, 1, nil)
				expected := &image.RGBA{Pix: bytes.Clone(base.Pix), Stride: base.Stride, Rect: base.Rect}
				s.applyFrameShadows(expected, art, body, image.Point{}, nil)
				check(fmt.Sprintf("shadow layout=%d mode=%d clip=%v", layout, mode, clipped), actual, expected)
			}
		}
	}
}

var dialogueGPUResult *dialogueGPUProbe

func TestMain(m *testing.M) {
	if os.Getenv("AGAINROM_DIALOGUE_GPU") == "1" {
		dialogueGPUResult = &dialogueGPUProbe{}
		ebiten.SetWindowSize(64, 48)
		ebiten.SetWindowPosition(-32000, -32000)
		ebiten.SetRunnableOnUnfocused(true)
		if err := ebiten.RunGameWithOptions(dialogueGPUResult, &ebiten.RunGameOptions{InitUnfocused: true, SkipTaskbar: true}); err != nil && err != ebiten.Termination {
			dialogueGPUResult.err = err
		}
	}
	os.Exit(m.Run())
}

func TestDialogueGPUReadback(t *testing.T) {
	if dialogueGPUResult == nil {
		t.Skip("AGAINROM_DIALOGUE_GPU=1 required")
	}
	if dialogueGPUResult.err != nil {
		t.Fatal(dialogueGPUResult.err)
	}
	if dialogueGPUResult.cases != 24 {
		t.Fatalf("GPU cases=%d, want24", dialogueGPUResult.cases)
	}
	t.Log("24 actual GPU lookup/shadow cases match CPU pixels with displaced source textures")
}
