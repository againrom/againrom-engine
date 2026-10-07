package ui

// Where the front-end's own cursor picture stands. With no window to draw into,
// the picture's place is read from cursorPlacement, which drawCursor draws from,
// and the position it uses is the one the latest step was given: the tick that
// acts on the pointer and the frame that draws it share one sample.

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/frame"
)

// menuCursorWindows covers scale 1, an integer scale, fractional scales, a wide
// window that pillarboxes the frame and a tall one that letterboxes it.
var menuCursorWindows = []image.Point{
	{640, 480}, {1280, 960}, {1024, 768}, {1440, 1080}, {1920, 1080}, {1000, 700}, {2560, 1440}, {700, 1000},
}

// menuCursorApp is the boot menu over a registry whose `select` hotspot is the
// shipped (3,4), in a window of the given size.
func menuCursorApp(t *testing.T, win image.Point) *App {
	t.Helper()
	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetCursorRegistry(clRegistry())
	a.Layout(win.X, win.Y)
	if a.Screen() != ScreenMenu {
		t.Fatalf("setup: screen = %v, want the menu", a.Screen())
	}
	return a
}

// TestTheMenuCursorStandsOnTheFramePixelUnderIt fails when the picture is
// placed by window pixels. The original blits its cursor at an integer position
// of its 640x480 surface and hit-tests the same surface (AI-CURSOR-218,
// MENU-MASK-004), so the pixel under the drawn hotspot is the pixel a click
// names, and the picture moves by whole frame pixels.
//
// Every assertion is about one step's own pointer: the placement is read after
// the step, with nothing drawn in between.
func TestTheMenuCursorStandsOnTheFramePixelUnderIt(t *testing.T) {
	const eps = 1e-9
	native := image.Rect(0, 0, frame.W, frame.H)
	for _, win := range menuCursorWindows {
		a := menuCursorApp(t, win)
		place := frame.Fit(frame.W, frame.H, win.X, win.Y)
		ox, oy := place.Origin()
		scale := place.Scale()

		probe := func(x, y int) {
			t.Helper()
			a.step(appInput{CursorX: x, CursorY: y}, atAt)
			c, ok := a.cursorPlacement()
			if !ok {
				t.Fatalf("window %v, pointer (%d,%d): the menu draws no cursor picture", win, x, y)
			}
			if c.Hot != image.Pt(3, 4) {
				t.Fatalf("window %v, pointer (%d,%d): hotspot %v, want the registered (3,4)", win, x, y, c.Hot)
			}
			if c.Scale != scale {
				t.Fatalf("window %v, pointer (%d,%d): picture scale %v, want the placement's %v", win, x, y, c.Scale, scale)
			}

			// The hotspot is on the pixel the hit tests resolve for this pointer,
			// or outside the frame exactly where they resolve none.
			if p, in := a.windowToNativeFrame(x, y); in {
				if c.Tip != p {
					t.Fatalf("window %v, pointer (%d,%d): the hotspot is on frame pixel %v, the hit test resolves %v", win, x, y, c.Tip, p)
				}
			} else if c.Tip.In(native) {
				t.Fatalf("window %v, pointer (%d,%d): no frame pixel there, yet the hotspot is on %v inside the frame", win, x, y, c.Tip)
			}

			// The picture's top-left is on the frame's lattice: the placement's
			// origin plus a whole number of frame pixels.
			fx := (c.X - ox) / scale
			fy := (c.Y - oy) / scale
			wantX, wantY := float64(c.Tip.X-c.Hot.X), float64(c.Tip.Y-c.Hot.Y)
			if math.Abs(fx-wantX) > eps || math.Abs(fy-wantY) > eps {
				t.Fatalf("window %v, pointer (%d,%d): picture top-left (%.4f,%.4f) is frame pixel (%.4f,%.4f), want the whole pixel (%v,%v)",
					win, x, y, c.X, c.Y, fx, fy, wantX, wantY)
			}

			// The pointer is inside the frame pixel the hotspot stands on.
			left := c.X + float64(c.Hot.X)*c.Scale
			top := c.Y + float64(c.Hot.Y)*c.Scale
			if float64(x) < left-eps || float64(x) >= left+c.Scale+eps || float64(y) < top-eps || float64(y) >= top+c.Scale+eps {
				t.Fatalf("window %v, pointer (%d,%d): outside the hotspot's frame pixel [%.4f,%.4f)x[%.4f,%.4f)",
					win, x, y, left, left+c.Scale, top, top+c.Scale)
			}
		}

		for y := -3; y < win.Y+3; y += 29 {
			for x := -3; x < win.X+3; x += 37 {
				probe(x, y)
			}
		}
		// Either side of frame pixel boundaries, where a placement by window
		// pixels and one by frame pixels part.
		for _, f := range []image.Point{{0, 0}, {1, 1}, {317, 239}, {318, 240}, {639, 479}} {
			wx, wy, ok := place.FrameToWindow(f)
			if !ok {
				t.Fatalf("setup: window %v has no window pixel for frame pixel %v", win, f)
			}
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					probe(wx+dx, wy+dy)
				}
			}
		}
	}
}

// TestTheMenuCursorMovesByWholeFramePixels is the same fact as the player sees
// it: a pointer that stays within one frame pixel leaves the picture where it
// is, and one that crosses into the next moves it by exactly the placement's
// scale. At a 2.25 scale that is a step every two or three window pixels; a
// picture placed by window pixels moves on every one.
func TestTheMenuCursorMovesByWholeFramePixels(t *testing.T) {
	win := image.Pt(1920, 1080)
	a := menuCursorApp(t, win)
	place := frame.Fit(frame.W, frame.H, win.X, win.Y)
	scale := place.Scale()

	wx, wy, ok := place.FrameToWindow(image.Pt(200, 200))
	if !ok {
		t.Fatal("setup: no window pixel for frame pixel (200,200)")
	}
	var last CursorPlacement
	var haveLast bool
	crossings := 0
	for x := wx - 6; x <= wx+12; x++ {
		a.step(appInput{CursorX: x, CursorY: wy}, atAt)
		c, ok := a.cursorPlacement()
		if !ok {
			t.Fatalf("pointer x=%d: no cursor picture", x)
		}
		if haveLast {
			switch {
			case c.Tip == last.Tip && (c.X != last.X || c.Y != last.Y):
				t.Errorf("pointer x=%d stays on frame pixel %v and the picture moved from (%v,%v) to (%v,%v)",
					x, c.Tip, last.X, last.Y, c.X, c.Y)
			case c.Tip.X == last.Tip.X+1 && c.Tip.Y == last.Tip.Y:
				crossings++
				if math.Abs(c.X-last.X-scale) > 1e-9 || c.Y != last.Y {
					t.Errorf("pointer x=%d crosses to frame pixel %v: the picture moved by (%v,%v), want (%v,0)",
						x, c.Tip, c.X-last.X, c.Y-last.Y, scale)
				}
			case c.Tip != last.Tip:
				t.Errorf("pointer x=%d: the tip went from %v to %v, want the same pixel or the next one", x, last.Tip, c.Tip)
			}
		}
		last, haveLast = c, true
	}
	if crossings < 4 {
		t.Fatalf("the sweep crossed %d frame pixels, want at least 4 for the comparison to mean anything", crossings)
	}
}

// tickDelay is how many ticks earlier than tick i the pointer stood on frame
// pixel tip: 0 when the picture is where this tick's own pointer is, -1 when no
// tick of the trace stood there.
func tickDelay(pointerFrame []image.Point, i int, tip image.Point) int {
	for d := 0; d <= i; d++ {
		if pointerFrame[i-d] == tip {
			return d
		}
	}
	return -1
}

// TestTheCursorPictureIsPlacedFromThisTicksPointerInTheMenuAndInAMission
// measures the delay, in ticks, between a pointer move and the picture that
// follows it: the position the next frame draws is the position of the tick
// that just acted on the pointer, on the main menu and on a mission map alike.
// The trace is logged.
func TestTheCursorPictureIsPlacedFromThisTicksPointerInTheMenuAndInAMission(t *testing.T) {
	t.Run("main menu", func(t *testing.T) {
		win := image.Pt(1920, 1080)
		a := menuCursorApp(t, win)
		var frames []image.Point
		for i := 0; i < 12; i++ {
			x, y := 300+91*i, 200+53*i
			a.step(appInput{CursorX: x, CursorY: y}, atAt)
			p, in := a.windowToNativeFrame(x, y)
			if !in {
				t.Fatalf("setup: pointer (%d,%d) is off the frame", x, y)
			}
			frames = append(frames, p)
			c, ok := a.cursorPlacement()
			if !ok {
				t.Fatalf("tick %d: no cursor picture", i)
			}
			d := tickDelay(frames, i, c.Tip)
			t.Logf("menu    tick %2d: pointer (%4d,%4d) frame %v, picture tip %v, delay %d ticks", i, x, y, p, c.Tip, d)
			if d != 0 {
				t.Fatalf("tick %d: the picture's hotspot is on the pixel the pointer stood on %d ticks earlier, want 0", i, d)
			}
		}
	})

	t.Run("mission", func(t *testing.T) {
		a, v, _ := atOnMap(t)
		a.SetCursorRegistry(clRegistry())
		var frames []image.Point
		for i, cell := range []image.Point{{1, 1}, {2, 1}, {3, 2}, {4, 1}, {5, 2}, {6, 1}, {2, 3}, {1, 4}} {
			x, y := cellPoint(v, cell.X, cell.Y)
			a.step(atFrame(x, y), atAt)
			fx, fy := v.windowToFrame(x, y)
			frames = append(frames, image.Pt(fx, fy))
			_, hot, ok := a.flow.cursor.Current()
			if !ok {
				t.Fatalf("tick %d: the manager holds no cursor", i)
			}
			_, at, ok := v.mapCursorPresent()
			if !ok {
				t.Fatalf("tick %d: the map draws no cursor picture", i)
			}
			tip := at.Add(hot)
			d := tickDelay(frames, i, tip)
			t.Logf("mission tick %2d: pointer (%4d,%4d) frame (%d,%d), picture tip %v, delay %d ticks", i, x, y, fx, fy, tip, d)
			if d != 0 {
				t.Fatalf("tick %d: the picture's hotspot is on the pixel the pointer stood on %d ticks earlier, want 0", i, d)
			}
		}
	})
}

// TestTheCursorKeepsWindowPixelsWithoutAPlacement pins the one case the frame
// has nothing to place against: before any window size is known, the window
// pixel is the frame pixel at scale 1, where the picture stood before there was
// a placement.
func TestTheCursorKeepsWindowPixelsWithoutAPlacement(t *testing.T) {
	a := menuCursorApp(t, image.Pt(640, 480))
	a.place = frame.Placement{}
	a.step(appInput{CursorX: 40, CursorY: 30}, atAt)
	c, ok := a.cursorPlacement()
	if !ok {
		t.Fatal("no cursor picture without a placement")
	}
	if c.Tip != image.Pt(40, 30) || c.Scale != 1 || c.X != 37 || c.Y != 26 {
		t.Errorf("placement without a window = tip %v scale %v top-left (%v,%v), want tip (40,30) scale 1 top-left (37,26)",
			c.Tip, c.Scale, c.X, c.Y)
	}
}
