package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// presentation is what the library does with a game in one window: the size
// the game's layout answers, the frame the game draws (the library's
// offscreen), the device pixels of the window, and the final stage that
// brings the frame to the device pixels with its scale and filter.
type presentation struct {
	layout      [2]float64
	frame       image.Point
	device      image.Point
	scale       float64
	filter      ebiten.Filter
	finalStages int
	// cursor is the pointer the game reads for a device pixel.
	cursor func(dx, dy float64) (float64, float64)
}

// finalScreenRecorder is a FinalScreen that records the draws the library's
// default final stage makes.
type finalScreenRecorder struct {
	ebiten.FinalScreen
	bounds image.Rectangle
	draws  []ebiten.DrawImageOptions
}

func (r *finalScreenRecorder) Bounds() image.Rectangle { return r.bounds }
func (r *finalScreenRecorder) DrawImage(_ *ebiten.Image, op *ebiten.DrawImageOptions) {
	r.draws = append(r.draws, *op)
}
func (r *finalScreenRecorder) Clear()             {}
func (r *finalScreenRecorder) Fill(_ color.Color) {}

// libraryPresentation replays the library's frame for a window whose client
// area is device pixels at device scale s, from the library's own code
// (ebiten v2.10.5):
//   - internal/ui/ui_glfw.go outsideSizeInDIP: the layout gets the client
//     size divided by s, or the requested whole size in points;
//   - gameforui.go Layout: LayoutF when the game has it, else Layout with the
//     sizes truncated to whole points;
//   - internal/ui/context.go layoutGame: the offscreen is the answer rounded up;
//   - context.go letterboxScaleAndOffsets: the final stage's scale;
//   - gameforui.go DefaultDrawFinalScreen, run here on a recording screen:
//     the final stage's filter;
//   - context.go clientPositionToLogicalPosition: the cursor in frame pixels.
func libraryPresentation(t *testing.T, g ebiten.Game, device image.Point, s float64) presentation {
	t.Helper()
	ow, oh := float64(device.X)/s, float64(device.Y)/s
	if w, h := math.Round(ow), math.Round(oh); math.Round(w*s) == float64(device.X) && math.Round(h*s) == float64(device.Y) {
		ow, oh = w, h
	}
	var lw, lh float64
	if l, ok := g.(ebiten.LayoutFer); ok {
		lw, lh = l.LayoutF(ow, oh)
	} else {
		w, h := g.Layout(int(max(ow, 1)), int(max(oh, 1)))
		lw, lh = float64(w), float64(h)
	}
	p := presentation{layout: [2]float64{lw, lh}, device: device}
	p.frame = image.Pt(int(math.Ceil(lw)), int(math.Ceil(lh)))
	p.scale = min(float64(device.X)/lw, float64(device.Y)/lh)
	ox, oy := (float64(device.X)-lw*p.scale)/2, (float64(device.Y)-lh*p.scale)/2
	var geoM ebiten.GeoM
	geoM.Scale(p.scale, p.scale)
	geoM.Translate(ox, oy)
	rec := &finalScreenRecorder{bounds: image.Rectangle{Max: device}}
	if _, ok := g.(ebiten.FinalScreenDrawer); ok {
		t.Fatal("the game draws its own final screen; the replay knows only the library's")
	}
	ebiten.DefaultDrawFinalScreen(rec, nil, geoM)
	if len(rec.draws) != 1 {
		t.Fatalf("the final stage made %d draws, want 1", len(rec.draws))
	}
	p.filter = rec.draws[0].Filter
	if p.scale != 1 || ox != 0 || oy != 0 {
		p.finalStages = 1
	}
	p.cursor = func(dx, dy float64) (float64, float64) {
		return (dx - ox) / p.scale, (dy - oy) / p.scale
	}
	return p
}

func filterName(f ebiten.Filter) string {
	switch f {
	case ebiten.FilterNearest:
		return "nearest"
	case ebiten.FilterLinear:
		return "linear"
	case ebiten.FilterPixelated:
		return "pixelated"
	}
	return fmt.Sprintf("filter %d", f)
}

// deviceScales are the Windows display scales the story measures.
var deviceScales = []float64{1, 1.25, 1.5, 2}

// atDeviceScale feeds the monitor's device scale for the test.
func atDeviceScale(t *testing.T, s float64) {
	t.Helper()
	old := monitorDeviceScale
	t.Cleanup(func() { monitorDeviceScale = old })
	monitorDeviceScale = func() float64 { return s }
}

// The measurement of the frame each game draws at each device scale, on a
// 2560 by 1440 client area: the layout answers the device pixels, the frame
// is drawn in them, and the library's final stage is an exact copy.
func TestDisplayScaleMeasurement(t *testing.T) {
	device := image.Pt(2560, 1440)
	for _, s := range deviceScales {
		atDeviceScale(t, s)
		v, err := NewViewer("scale", grid(60, 60), zoomTileset(false))
		if err != nil {
			t.Fatal(err)
		}
		games := []struct {
			name string
			game ebiten.Game
		}{
			{"front end", NewApp("scale", nil, nil, nil)},
			{"frame log", &frameLog{app: NewApp("scale", nil, nil, nil)}},
			{"viewer", v},
			{"map editor", &frameGuard{game: NewMapEditor(nil, nil, nil)}},
		}
		for _, g := range games {
			p := libraryPresentation(t, g.game, device, s)
			t.Logf("%s scale %.2f: layout answers %.2fx%.2f, frame %v, device %v, final stage x%.4f %s (stages %d)",
				g.name, s, p.layout[0], p.layout[1], p.frame, p.device, p.scale, filterName(p.filter), p.finalStages)
			if p.frame != device || p.finalStages != 0 || p.filter != ebiten.FilterNearest {
				t.Fatalf("%s scale %v: frame %v, %d library stages by %s; want the device %v copied",
					g.name, s, p.frame, p.finalStages, filterName(p.filter), device)
			}
			if x, y := p.cursor(1919, 1079); x != 1919 || y != 1079 {
				t.Fatalf("%s scale %v: device pixel (1919,1079) reads as pointer (%v,%v)", g.name, s, x, y)
			}
		}
	}
}

// The device size is the points times the scale, rounded as the library
// rounds the window it makes; the points are truncated as for Layout.
func TestWindowPixelsRoundsAsTheLibrary(t *testing.T) {
	for _, c := range []struct {
		w, h, s        float64
		device, points image.Point
	}{
		{2560, 1440, 1, image.Pt(2560, 1440), image.Pt(2560, 1440)},
		{2048, 1152, 1.25, image.Pt(2560, 1440), image.Pt(2048, 1152)},
		{2560 / 1.5, 960, 1.5, image.Pt(2560, 1440), image.Pt(1706, 960)},
		{1707, 960, 1.5, image.Pt(2561, 1440), image.Pt(1707, 960)},
		{1280, 720, 2, image.Pt(2560, 1440), image.Pt(1280, 720)},
		{1024, 768, 0, image.Pt(1024, 768), image.Pt(1024, 768)},
		{0.2, 0.2, 1, image.Pt(1, 1), image.Pt(1, 1)},
	} {
		w := windowPixels(c.w, c.h, c.s)
		if w.device != c.device || w.points != c.points {
			t.Fatalf("%v x%v at %v: device %v points %v, want %v %v", c.w, c.h, c.s, w.device, w.points, c.device, c.points)
		}
	}
}

// pointsWindow is the window the pointer tests use, in points.
var pointsWindow = image.Pt(1280, 720)

// pointsAt calls each point of the points window on a step at which the
// point times the scale is a whole device pixel.
func pointsAt(s float64, call func(p, d image.Point)) {
	for y := 0; y < pointsWindow.Y; y += 4 {
		for x := 0; x < pointsWindow.X; x += 4 {
			call(image.Pt(x, y), image.Pt(int(float64(x)*s), int(float64(y)*s)))
		}
	}
}

// At every device scale a front-end screen's frame is placed in the device
// pixels and scaled to them once, and a pointer on the device pixel under a
// point lands on the same frame pixel and the same menu button as at scale 1.
func TestFrontEndFrameAndPointerAtEveryDeviceScale(t *testing.T) {
	atDeviceScale(t, 1)
	one := newTestApp(t, appRows(1), nil)
	one.LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
	buttons := 0
	for _, s := range deviceScales {
		atDeviceScale(t, s)
		a := newTestApp(t, appRows(1), nil)
		a.LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
		device := image.Pt(int(float64(pointsWindow.X)*s), int(float64(pointsWindow.Y)*s))
		pix, extras, err := a.HeadlessPicture(HeadlessRequest{Part: HeadlessPartFrame, Scaled: true})
		if err != nil {
			t.Fatal(err)
		}
		if pix.Rect.Size() != device || strings.Count(extras.Note, "->") != 1 {
			t.Fatalf("scale %v: frame %v with %q, want %v in one stage", s, pix.Rect.Size(), extras.Note, device)
		}
		if math.Abs(a.place.Scale()-s*one.place.Scale()) > 1e-12 {
			t.Fatalf("scale %v: placement x%v, want x%v", s, a.place.Scale(), s*one.place.Scale())
		}
		pointsAt(s, func(p, d image.Point) {
			want, wok := one.windowToNativeFrame(p.X, p.Y)
			got, gok := a.windowToNativeFrame(d.X, d.Y)
			if want != got || wok != gok {
				t.Fatalf("scale %v: point %v is frame %v %v, device %v is %v %v", s, p, want, wok, d, got, gok)
			}
			wb, gb := one.hoverButtonAt(p.X, p.Y), a.hoverButtonAt(d.X, d.Y)
			if wb != gb {
				t.Fatalf("scale %v: point %v is button %d, device %v is %d", s, p, wb, d, gb)
			}
			if gb != 0 {
				buttons++
			}
		})
	}
	if buttons == 0 {
		t.Fatal("no point fell on a menu button")
	}
}

// At every device scale the mission frame is drawn in the device pixels:
// each picture is scaled once straight onto the window, the placement is the
// device scale times that of scale 1, and a pointer on the device pixel under
// a point lands on the same frame pixel and the same map cell.
func TestMissionFrameAndPointerAtEveryDeviceScale(t *testing.T) {
	got := recordScaledBlits(t)
	build := func(s float64) *Viewer {
		atDeviceScale(t, s)
		v, err := NewViewer("scale", grid(60, 60), zoomTileset(false))
		if err != nil {
			t.Fatal(err)
		}
		v.SetGraphicsOptions(GraphicsOptions{Smoothing: true})
		v.LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
		v.cam.SetZoom(1.2)
		v.cam.X, v.cam.Y = 300.4, 280.7
		return v
	}
	one := build(1)
	cells := 0
	for _, s := range deviceScales {
		v := build(s)
		device := image.Pt(int(float64(pointsWindow.X)*s), int(float64(pointsWindow.Y)*s))
		if v.place.WindowSize() != device || math.Abs(v.place.Scale()-s*one.place.Scale()) > 1e-12 {
			t.Fatalf("scale %v: placed in %v at x%v", s, v.place.WindowSize(), v.place.Scale())
		}
		*got = nil
		screen := ebiten.NewImage(device.X, device.Y)
		v.Draw(screen)
		want := map[*ebiten.Image]float64{v.smoothMap: 1.2 * v.place.Scale()}
		if v.place.Scale() != 1 {
			want[v.canvas] = v.place.Scale()
		}
		if len(*got) != len(want) {
			t.Fatalf("scale %v: %d scaled blits %v, want %d", s, len(*got), *got, len(want))
		}
		for _, b := range *got {
			if f, ok := want[b.src]; !ok || b.dst != screen || math.Abs(b.scale-f) > 1e-9 {
				t.Fatalf("scale %v: blit %+v, want one per picture onto the window at %v", s, b, want)
			}
		}
		pointsAt(s, func(p, d image.Point) {
			wx, wy := one.windowToFrame(p.X, p.Y)
			gx, gy := v.windowToFrame(d.X, d.Y)
			if wx != gx || wy != gy {
				t.Fatalf("scale %v: point %v is frame (%d,%d), device %v is (%d,%d)", s, p, wx, wy, d, gx, gy)
			}
			wc, wr, win := one.groundCellAt(float64(wx), float64(wy))
			gc, gr, gin := v.groundCellAt(float64(gx), float64(gy))
			if wc != gc || wr != gr || win != gin {
				t.Fatalf("scale %v: point %v is cell (%d,%d,%v), device %v is (%d,%d,%v)", s, p, wc, wr, win, d, gc, gr, gin)
			}
			if gin {
				cells++
			}
		})
	}
	if cells == 0 {
		t.Fatal("no point fell on a map cell")
	}
}

// At every device scale the map editor lays out in points, draws its frame
// in points and scales it once onto the device pixels, and its input door
// brings a device pixel back to the point it lies in.
func TestMapEditorFrameAndPointerAtEveryDeviceScale(t *testing.T) {
	got := recordScaledBlits(t)
	for _, s := range deviceScales {
		atDeviceScale(t, s)
		e := NewMapEditor(nil, nil, nil)
		lw, lh := (&frameGuard{game: e}).LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
		device := image.Pt(int(float64(pointsWindow.X)*s), int(float64(pointsWindow.Y)*s))
		if e.window != pointsWindow || image.Pt(int(lw), int(lh)) != device {
			t.Fatalf("scale %v: editor window %v, layout %vx%v; want %v in %v", s, e.window, lw, lh, pointsWindow, device)
		}
		*got = nil
		screen := ebiten.NewImage(device.X, device.Y)
		e.Draw(screen)
		if s == 1 {
			if len(*got) != 0 || e.pointsFrame != nil {
				t.Fatalf("scale 1: %d scaled blits, points frame %v", len(*got), e.pointsFrame)
			}
		} else if len(*got) != 1 || (*got)[0].src != e.pointsFrame || (*got)[0].dst != screen || (*got)[0].scale != s {
			t.Fatalf("scale %v: scaled blits %+v, want the points frame once onto the window", s, *got)
		}
		old := engineEditorInput
		var at image.Point
		engineEditorInput = func() editorNativeInput {
			return editorNativeInput{canvas: Input{CursorX: at.X, CursorY: at.Y}}
		}
		for y := 0; y < device.Y; y += 7 {
			for x := 0; x < device.X; x += 7 {
				at = image.Pt(x, y)
				c := e.readNative().canvas
				p := image.Pt(c.CursorX, c.CursorY)
				if float64(p.X)*s > float64(x) || float64(p.X+1)*s <= float64(x) ||
					float64(p.Y)*s > float64(y) || float64(p.Y+1)*s <= float64(y) {
					t.Fatalf("scale %v: device %v read as point %v, which is drawn elsewhere", s, at, p)
				}
			}
		}
		engineEditorInput = old
	}
}

// The edge-scroll strip is EdgeMargin points wide at every device scale.
func TestEdgeScrollStripKeepsItsWidthAtEveryDeviceScale(t *testing.T) {
	const fromEdge = 30
	lone := func(s float64) *Viewer {
		v, err := NewViewer("scale", grid(60, 60), zoomTileset(false))
		if err != nil {
			t.Fatal(err)
		}
		v.LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
		return v
	}
	inApp := func(s float64) *Viewer {
		a, v, _ := atOnMap(t)
		a.LayoutF(float64(pointsWindow.X), float64(pointsWindow.Y))
		return v
	}
	for name, build := range map[string]func(float64) *Viewer{"viewer": lone, "front end": inApp} {
		for _, s := range deviceScales {
			atDeviceScale(t, s)
			v := build(s)
			if want := int(math.Round(EdgeMargin * s)); v.edgeMargin() != want {
				t.Fatalf("%s scale %v: the strip is %d device pixels, want %d", name, s, v.edgeMargin(), want)
			}
			v.cam.X, v.cam.Y = 300, 300
			v.cam.Clamp()
			x0 := v.cam.X
			if x0 < PanSpeed {
				t.Fatalf("%s scale %v: setup: the camera at %v has no room to scroll left", name, s, x0)
			}
			v.step(Input{CursorX: fromEdge, CursorY: v.place.WindowSize().Y / 2}, atAt)
			scrolled, want := v.cam.X != x0, float64(EdgeMargin)*s > fromEdge
			if scrolled != want {
				t.Errorf("%s scale %v: pointer %d device pixels from the edge scrolled=%v, want %v", name, s, fromEdge, scrolled, want)
			}
		}
	}
}
