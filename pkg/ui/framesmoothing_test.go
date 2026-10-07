package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// finalBlit is one recorded call of the three paths drawFinalFrame chooses.
type finalBlit struct {
	path  string
	src   *ebiten.Image
	scale float64
}

// recordFinalBlits replaces the three paths for one test. None forwards to a
// draw: ebitengine refuses a readback before the game loop, so the call is
// the observation.
func recordFinalBlits(t *testing.T) *[]finalBlit {
	t.Helper()
	var calls []finalBlit
	prevSharp, prevCR, prevNearest := sharpBilinearBlit, catmullRomBlit, nearestBlit
	sharpBilinearBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, buf **ebiten.Image) {
		calls = append(calls, finalBlit{"sharp", src, op.GeoM.Element(0, 0)})
	}
	catmullRomBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) bool {
		calls = append(calls, finalBlit{"catmullrom", src, op.GeoM.Element(0, 0)})
		return true
	}
	nearestBlit = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions) {
		calls = append(calls, finalBlit{"nearest", src, op.GeoM.Element(0, 0)})
	}
	t.Cleanup(func() { sharpBilinearBlit, catmullRomBlit, nearestBlit = prevSharp, prevCR, prevNearest })
	return &calls
}

func pathOf(calls []finalBlit, match func(finalBlit) bool) (string, float64) {
	for _, c := range calls {
		if match(c) {
			return c.path, c.scale
		}
	}
	return "", 0
}

func isTiny(c finalBlit) bool { return c.src.Bounds().Size() == image.Pt(1, 1) }

// drawSites draws each of the six final blits once with the named site's
// placement at scale s and answers the path and scale each reached.
func drawSites(t *testing.T, s float64, on bool) map[string][2]any {
	t.Helper()
	got := map[string][2]any{}
	put := func(site, path string, scale float64) { got[site] = [2]any{path, scale} }

	// The town family: the root menu composite, the cursor and the cutscene.
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetFrameSmoothing(on)
	a.SetCursorRegistry(mcRegistry())
	a.Layout(int(640*s), int(480*s))
	calls := recordFinalBlits(t)
	a.Draw(ebiten.NewImage(a.winW, a.winH))
	p, sc := pathOf(*calls, func(c finalBlit) bool { return c.src.Bounds().Dx() >= 640 })
	put("townComposite", p, sc)
	p, sc = pathOf(*calls, isTiny)
	put("cursor", p, sc)
	*calls = nil
	a.drawCutscene(ebiten.NewImage(a.winW, a.winH))
	p, sc = pathOf(*calls, func(c finalBlit) bool { return c.src == a.cutsceneCanvas })
	put("cutscene", p, sc)

	// The mission frame and its deferred pointer, at the viewer's placement.
	a, v, _ := atOnMap(t)
	a.SetFrameSmoothing(on)
	a.SetCursorRegistry(mcRegistry())
	a.Layout(int(1024*s), int(768*s))
	a.step(atFrame(v.frameW/2, v.frameH/2), atAt)
	calls = recordFinalBlits(t)
	a.Draw(ebiten.NewImage(a.winW, a.winH))
	p, sc = pathOf(*calls, func(c finalBlit) bool { return c.src == v.canvas })
	put("missionFrame", p, sc)
	p, sc = pathOf(*calls, isTiny)
	put("pointer", p, sc)

	// The in-game menu over the map, at the town family's placement.
	a, v, _ = atOnMap(t)
	a.SetFrameSmoothing(on)
	a.SetCursorRegistry(mcRegistry())
	esc := atFrame(v.frameW/2, v.frameH/2)
	esc.Escape = true
	a.step(esc, atAt)
	if a.Screen() != ScreenGameMenu {
		t.Fatalf("setup: Escape landed on %v, want the in-game menu", a.Screen())
	}
	a.Layout(int(640*s), int(480*s))
	calls = recordFinalBlits(t)
	a.Draw(ebiten.NewImage(a.winW, a.winH))
	p, sc = pathOf(*calls, func(c finalBlit) bool { return c.src == a.menuCanvas })
	put("menuOverMap", p, sc)
	return got
}

var frameSites = []string{"townComposite", "cursor", "cutscene", "missionFrame", "pointer", "menuOverMap"}

// With FrameSmoothing on, the default, every site reaches the Catmull-Rom
// path at the shipped non-unit scales, the integer 3.0 included, and the
// single nearest draw at exactly 1. With it off every site reaches method B.
func TestFrameSmoothingDispatchAtEverySite(t *testing.T) {
	for _, s := range []float64{1, 1.40625, 1.875, 2.25, 3} {
		for _, on := range []bool{true, false} {
			got := drawSites(t, s, on)
			want := "catmullrom"
			switch {
			case !on:
				want = "sharp"
			case s == 1:
				want = "nearest"
			}
			for _, site := range frameSites {
				if g := got[site]; g[0] != want || g[1] != s {
					t.Errorf("scale %v smoothing %v: %s reached %v at scale %v, want %s at %v", s, on, site, g[0], g[1], want, s)
				}
			}
		}
	}
}

// A new App and a new Viewer use the Catmull-Rom scaler until told otherwise.
func TestFrameSmoothingDefaultsOn(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	if !a.FrameSmoothing() {
		t.Fatal("NewApp: FrameSmoothing off")
	}
	if v := geometryViewer(t); v.frameSmoothingOff {
		t.Fatal("NewViewer: FrameSmoothing off")
	}
}

// The Kage source compiles without a game loop.
func TestCatmullRomShaderCompiles(t *testing.T) {
	if catmullRomProgram() == nil {
		t.Fatalf("the Catmull-Rom shader did not compile: %v", catmullRomShaderErr)
	}
}

// The production path draws a 2560x1440 town and mission frame without a
// panic, and so does its compile-failure fallback.
func TestDrawAtWindowScaleRuns(t *testing.T) {
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetCursorRegistry(mcRegistry())
	a.Layout(2560, 1440)
	a.Draw(ebiten.NewImage(2560, 1440))
	a, _, _ = atOnMap(t)
	a.Layout(2560, 1440)
	a.Draw(ebiten.NewImage(2560, 1440))

	prev := catmullRomShader
	catmullRomShader = nil
	t.Cleanup(func() { catmullRomShader = prev })
	var buf *ebiten.Image
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(1.875, 1.875)
	drawFinalFrame(ebiten.NewImage(64, 64), ebiten.NewImage(8, 8), &op, &buf, false)
	if buf == nil {
		t.Fatal("with no shader the draw did not fall back to method B")
	}
}
