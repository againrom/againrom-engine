package ui

// Behavioural tests for the one camera step both entry points drive.
//
// Before the input snapshot existed, none of this was reachable: Update read
// engine globals directly, so no test in the repo could drive a pan, an
// edge-scroll or a zoom. Every expectation below is computed from the camera
// contract and the viewer's published UX constants, never by calling the
// code under test.

import (
	"math"
	"testing"
	"time"

	"againrom/pkg/render/terrain"
)

// stepViewer builds a viewer over a world far larger than the view, positioned
// away from every clamp bound so a movement cannot be masked by a clamp.
func stepViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("step", grid(400, 400), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 800, 600)
	v.Camera().X, v.Camera().Y = 2000, 2000
	v.Camera().Clamp()
	return v
}

// centre is a cursor position in the middle of an 800x600 view: inside the
// window, but far from every edge margin, so it contributes no edge-scroll and
// key pans can be measured on their own.
var centre = struct{ x, y int }{400, 300}

func TestViewerStep(t *testing.T) {
	// A timestamp far enough apart to be unambiguous, but the animation subtest
	// is the only one that depends on it; the rest reuse one instant so elapsed
	// time is zero and cannot perturb the camera.
	frozen := time.Unix(1_700_000_000, 0)

	t.Run("each pan key moves the camera by PanSpeed in world pixels", func(t *testing.T) {
		cases := []struct {
			name   string
			in     Input
			dx, dy float64
		}{
			{"left", Input{PanLeft: true}, -PanSpeed, 0},
			{"right", Input{PanRight: true}, +PanSpeed, 0},
			{"up", Input{PanUp: true}, 0, -PanSpeed},
			{"down", Input{PanDown: true}, 0, +PanSpeed},
			{"diagonal", Input{PanRight: true, PanDown: true}, +PanSpeed, +PanSpeed},
			{"opposing keys cancel", Input{PanLeft: true, PanRight: true}, 0, 0},
			{"nothing pressed", Input{}, 0, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				v := stepViewer(t)
				in := tc.in
				in.CursorX, in.CursorY = centre.x, centre.y
				x0, y0 := v.Camera().X, v.Camera().Y

				v.step(in, frozen)

				if got, want := v.Camera().X, x0+tc.dx; got != want {
					t.Errorf("camera X = %v, want %v", got, want)
				}
				if got, want := v.Camera().Y, y0+tc.dy; got != want {
					t.Errorf("camera Y = %v, want %v", got, want)
				}
			})
		}
	})

	t.Run("edge-scroll fires on the right axis and sign at each edge", func(t *testing.T) {
		const winW, winH = 800 + MissionPanelW, 600
		cases := []struct {
			name   string
			x, y   int
			dx, dy float64
		}{
			{"left edge", 0, winH / 2, -PanSpeed, 0},
			{"just inside the left margin", EdgeMargin - 1, winH / 2, -PanSpeed, 0},
			{"just past the left margin", EdgeMargin, winH / 2, 0, 0},
			{"right edge", winW - 1, winH / 2, +PanSpeed, 0},
			{"just inside the right margin", winW - EdgeMargin, winH / 2, +PanSpeed, 0},
			{"just past the right margin", winW - EdgeMargin - 1, winH / 2, 0, 0},
			{"the map viewport's own right border is not an edge", 800 - 1, winH / 2, 0, 0},
			{"top edge", winW / 2, 0, 0, -PanSpeed},
			{"bottom edge", winW / 2, winH - 1, 0, +PanSpeed},
			{"top-left corner scrolls both axes", 0, 0, -PanSpeed, -PanSpeed},
			{"centre scrolls nothing", centre.x, centre.y, 0, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				v := stepViewer(t)
				x0, y0 := v.Camera().X, v.Camera().Y

				v.step(Input{CursorX: tc.x, CursorY: tc.y}, frozen)

				if got, want := v.Camera().X, x0+tc.dx; got != want {
					t.Errorf("camera X = %v, want %v", got, want)
				}
				if got, want := v.Camera().Y, y0+tc.dy; got != want {
					t.Errorf("camera Y = %v, want %v", got, want)
				}
			})
		}
	})

	t.Run("a cursor outside the window edge-scrolls nothing", func(t *testing.T) {
		// A cursor that has left the window still reports a position. Treating it
		// as an edge would scroll the map forever while the pointer sits on another
		// window. The bounds are the WINDOW's, which stepViewer lays out at
		// 800+MissionPanelW by 600.
		const winW, winH = 800 + MissionPanelW, 600
		for _, p := range [][2]int{{-1, 300}, {400, -1}, {winW, 300}, {400, winH}, {-50, -50}, {5000, 5000}} {
			v := stepViewer(t)
			x0, y0 := v.Camera().X, v.Camera().Y
			v.step(Input{CursorX: p[0], CursorY: p[1]}, frozen)
			if v.Camera().X != x0 || v.Camera().Y != y0 {
				t.Errorf("cursor (%d,%d): camera moved to (%v,%v), want it still at (%v,%v)",
					p[0], p[1], v.Camera().X, v.Camera().Y, x0, y0)
			}
		}
	})

	t.Run("a wide window reaches both horizontal edges and edge-scrolls there", func(t *testing.T) {
		const winW, winH = 1920, 1080
		cases := []struct {
			name   string
			x, y   int
			dx, dy float64
		}{
			{"the window's left edge", 0, winH / 2, -PanSpeed, 0},
			{"the window's right edge", winW - 1, winH / 2, +PanSpeed, 0},
			{"the window's top edge", winW / 2, 0, 0, -PanSpeed},
			{"the window's bottom edge", winW / 2, winH - 1, 0, +PanSpeed},
			{"the middle of the frame scrolls nothing", winW / 2, winH / 2, 0, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				v, err := NewViewer("wide", grid(400, 400), &terrain.Tileset{})
				if err != nil {
					t.Fatalf("NewViewer: %v", err)
				}
				v.Layout(winW, winH)
				v.Camera().X, v.Camera().Y = 2000, 2000
				v.Camera().Clamp()

				// Both side edges belong to the expanded frame. This is the direct
				// regression against the old 240-pixel pillarboxes.
				for _, x := range []int{0, winW - 1} {
					if _, ok := v.place.WindowToFrame(x, winH/2); !ok {
						t.Fatalf("window side x=%d is outside the expanded frame", x)
					}
				}

				x0, y0 := v.Camera().X, v.Camera().Y
				v.step(Input{CursorX: tc.x, CursorY: tc.y}, frozen)

				if got, want := v.Camera().X, x0+tc.dx; got != want {
					t.Errorf("camera X = %v, want %v", got, want)
				}
				if got, want := v.Camera().Y, y0+tc.dy; got != want {
					t.Errorf("camera Y = %v, want %v", got, want)
				}
			})
		}
	})

	t.Run("a wheel notch zooms about the view centre", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			wheel  float64
			factor float64
		}{
			{"in", +1, WheelZoomStep},
			{"out", -1, 1 / WheelZoomStep},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := stepViewer(t)
				z0 := v.Camera().Zoom
				v.syncMapViewport()
				sx, sy := float64(v.Camera().ViewW)/2, float64(v.Camera().ViewH)/2
				wx, wy := v.Camera().ScreenToWorld(sx, sy)

				v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: tc.wheel}, frozen)

				if got, want := v.Camera().Zoom, z0*tc.factor; math.Abs(got-want) > 1e-12 {
					t.Errorf("Zoom = %v, want %v", got, want)
				}
				gx, gy := v.Camera().ScreenToWorld(sx, sy)
				if math.Abs(gx-wx) > 1e-9 || math.Abs(gy-wy) > 1e-9 {
					t.Errorf("view centre moved from (%v,%v) to (%v,%v)", wx, wy, gx, gy)
				}
			})
		}
	})

	t.Run("zoom stays within its limits under repeated notches", func(t *testing.T) {
		for _, dir := range []float64{+1, -1} {
			v := stepViewer(t)
			for i := 0; i < 200; i++ {
				v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: dir}, frozen)
				if z := v.Camera().Zoom; z < 0.125 || z > 8 {
					t.Fatalf("Zoom = %v after %d notches, outside the limits", z, i+1)
				}
			}
		}
	})

	t.Run("the camera clamps at the world edges under any sequence", func(t *testing.T) {
		v := stepViewer(t)
		v.Camera().X, v.Camera().Y = 0, 0
		worldW, worldH := v.Camera().WorldW(), v.Camera().WorldH()

		inputs := []Input{
			{PanLeft: true, CursorX: centre.x, CursorY: centre.y},
			{PanUp: true, CursorX: centre.x, CursorY: centre.y},
			{PanRight: true, CursorX: centre.x, CursorY: centre.y},
			{PanDown: true, CursorX: centre.x, CursorY: centre.y},
			{CursorX: 0, CursorY: 0},
			{CursorX: 799, CursorY: 599},
			{CursorX: centre.x, CursorY: centre.y, WheelY: +1},
			{CursorX: centre.x, CursorY: centre.y, WheelY: -1},
		}
		for round := 0; round < 60; round++ {
			v.step(inputs[round%len(inputs)], frozen)
			c := v.Camera()
			vw := float64(c.ViewW) / c.Zoom
			vh := float64(c.ViewH) / c.Zoom
			if c.X < -1e-9 || c.X > worldW-vw+1e-9 {
				t.Fatalf("round %d: camera X = %v outside [0, %v]", round, c.X, worldW-vw)
			}
			if c.Y < -1e-9 || c.Y > worldH-vh+1e-9 {
				t.Fatalf("round %d: camera Y = %v outside [0, %v]", round, c.Y, worldH-vh)
			}
		}
	})

	t.Run("the water counter advances from the injected timestamp", func(t *testing.T) {
		v := stepViewer(t)
		in := Input{CursorX: centre.x, CursorY: centre.y}

		// The first call only takes the baseline: a slow startup must not fire a
		// burst of ticks.
		v.step(in, frozen)
		if got := v.AnimationCounter(); got != 0 {
			t.Fatalf("counter = %d after the baseline call, want 0", got)
		}

		_, speed := v.Animation()
		ms := terrain.TickMillis(speed)
		if ms <= 0 {
			t.Fatalf("tick length is %d ms; the fixture cannot measure ticks", ms)
		}

		// Exactly five ticks' worth of elapsed time.
		v.step(in, frozen.Add(time.Duration(5*ms)*time.Millisecond))
		if got := v.AnimationCounter(); got != 5 {
			t.Errorf("counter = %d after %d ms, want 5", got, 5*ms)
		}

		// Less than one tick advances nothing.
		v.step(in, frozen.Add(time.Duration(5*ms)*time.Millisecond+time.Millisecond))
		if got := v.AnimationCounter(); got != 5 {
			t.Errorf("counter = %d after a sub-tick gap, want it still 5", got)
		}
	})

	t.Run("a static viewer ignores elapsed time", func(t *testing.T) {
		v := stepViewer(t)
		v.SetAnimated(false)
		in := Input{CursorX: centre.x, CursorY: centre.y}
		v.step(in, frozen)
		v.step(in, frozen.Add(time.Hour))
		if got := v.AnimationCounter(); got != 0 {
			t.Errorf("counter = %d with animation off, want 0", got)
		}
	})

	t.Run("a primary-button drag pans by the screen delta divided by the zoom, opposite the cursor", func(t *testing.T) {
		for _, zoom := range []float64{1, 2} {
			t.Run(map[float64]string{1: "zoom 1", 2: "zoom 2"}[zoom], func(t *testing.T) {
				v := stepViewer(t)
				v.Camera().SetZoom(zoom)
				x0, y0 := v.Camera().X, v.Camera().Y

				// Press: the anchoring tick. It has no previous position to
				// difference against, so it pans zero however far the cursor
				// is from anywhere else.
				v.step(Input{PrimaryDown: true, CursorX: 400, CursorY: 300}, frozen)
				if got, want := v.Camera().X, x0; got != want {
					t.Fatalf("anchor tick: camera X = %v, want %v (unmoved)", got, want)
				}
				if got, want := v.Camera().Y, y0; got != want {
					t.Fatalf("anchor tick: camera Y = %v, want %v (unmoved)", got, want)
				}

				// Move: the screen delta is (50,20); the camera moves opposite
				// that, divided by the zoom.
				v.step(Input{PrimaryDown: true, CursorX: 450, CursorY: 320}, frozen)
				if got, want := v.Camera().X, x0-50/zoom; got != want {
					t.Errorf("first move: camera X = %v, want %v", got, want)
				}
				if got, want := v.Camera().Y, y0-20/zoom; got != want {
					t.Errorf("first move: camera Y = %v, want %v", got, want)
				}

				// Move again: the delta is measured from the PREVIOUS tick's
				// cursor (450,320), not from the press — (420,290) is a
				// screen delta of (-30,-30) from there.
				v.step(Input{PrimaryDown: true, CursorX: 420, CursorY: 290}, frozen)
				wantX := x0 - 50/zoom + 30/zoom
				wantY := y0 - 20/zoom + 30/zoom
				if got := v.Camera().X; got != wantX {
					t.Errorf("second move: camera X = %v, want %v", got, wantX)
				}
				if got := v.Camera().Y; got != wantY {
					t.Errorf("second move: camera Y = %v, want %v", got, wantY)
				}

				// Release at the same cursor position: no drag movement, and
				// no state should let this tick pan anything.
				preX, preY := v.Camera().X, v.Camera().Y
				v.step(Input{PrimaryDown: false, CursorX: 420, CursorY: 290}, frozen)
				if got := v.Camera().X; got != preX {
					t.Errorf("release: camera X = %v, want %v (unchanged)", got, preX)
				}
				if got := v.Camera().Y; got != preY {
					t.Errorf("release: camera Y = %v, want %v (unchanged)", got, preY)
				}
			})
		}
	})

	t.Run("a button already held on the very first tick anchors and pans zero", func(t *testing.T) {
		// Simulates a button held before the map opened: step has never run
		// before, so v.dragging starts at its zero value (false), and this
		// must still be treated as an anchor, not a jump to an arbitrary
		// cursor far from the view's centre.
		v := stepViewer(t)
		x0, y0 := v.Camera().X, v.Camera().Y
		v.step(Input{PrimaryDown: true, CursorX: 777, CursorY: 555}, frozen)
		if got, want := v.Camera().X, x0; got != want {
			t.Errorf("camera X = %v, want %v (unmoved)", got, want)
		}
		if got, want := v.Camera().Y, y0; got != want {
			t.Errorf("camera Y = %v, want %v (unmoved)", got, want)
		}
	})

	t.Run("a zero-move drag pans zero, at any zoom", func(t *testing.T) {
		for _, zoom := range []float64{1, 2, 0.5} {
			v := stepViewer(t)
			v.Camera().SetZoom(zoom)
			x0, y0 := v.Camera().X, v.Camera().Y

			v.step(Input{PrimaryDown: true, CursorX: centre.x, CursorY: centre.y}, frozen)
			v.step(Input{PrimaryDown: true, CursorX: centre.x, CursorY: centre.y}, frozen)

			if got, want := v.Camera().X, x0; got != want {
				t.Errorf("zoom %v: camera X = %v, want %v", zoom, got, want)
			}
			if got, want := v.Camera().Y, y0; got != want {
				t.Errorf("zoom %v: camera Y = %v, want %v", zoom, got, want)
			}
		}
	})

	// 0014 SC-15.
	t.Run("a drag inside the edge margin pans exactly once, not twice", func(t *testing.T) {
		v := stepViewer(t)
		x0, _ := v.Camera().X, v.Camera().Y

		v.step(Input{PrimaryDown: true, CursorX: centre.x, CursorY: centre.y}, frozen) // anchor, away from any margin
		v.step(Input{PrimaryDown: true, CursorX: 10, CursorY: centre.y}, frozen)       // move into the left margin

		// Drag alone: screen delta (10-400) = -390, so camera X moves by +390.
		// If edge-scroll also fired here (the bug this test exists to catch),
		// the left-margin term would add another -PanSpeed on top.
		if got, want := v.Camera().X, x0+390; got != want {
			t.Errorf("camera X = %v, want %v (drag alone; got %v extra, want 0 from edge-scroll)",
				got, want, got-want)
		}
	})

	t.Run("the same cursor position with the button up still edge-scrolls", func(t *testing.T) {
		v := stepViewer(t)
		x0, _ := v.Camera().X, v.Camera().Y

		v.step(Input{CursorX: 10, CursorY: centre.y}, frozen) // PrimaryDown zero value: button up

		if got, want := v.Camera().X, x0-PanSpeed; got != want {
			t.Errorf("camera X = %v, want %v (edge-scroll fires with the button up)", got, want)
		}
	})

	t.Run("the keyboard keeps working during a drag tick", func(t *testing.T) {
		v := stepViewer(t)
		x0, y0 := v.Camera().X, v.Camera().Y

		v.step(Input{PrimaryDown: true, CursorX: centre.x, CursorY: centre.y}, frozen) // anchor
		v.step(Input{PrimaryDown: true, PanRight: true, CursorX: 450, CursorY: 300}, frozen)

		// Keyboard (+PanSpeed) and drag (screen delta 50, opposite, zoom 1: -50)
		// both apply to this one tick.
		if got, want := v.Camera().X, x0+PanSpeed-50; got != want {
			t.Errorf("camera X = %v, want %v (keyboard + drag)", got, want)
		}
		if got, want := v.Camera().Y, y0; got != want {
			t.Errorf("camera Y = %v, want %v (no vertical component)", got, want)
		}
	})

	t.Run("a held drag raises the slop accumulator by the path the cursor travelled", func(t *testing.T) {
		v := stepViewer(t)
		v.dragMoved = 99 // residue from an earlier gesture

		v.step(Input{PrimaryDown: true, CursorX: 400, CursorY: 300}, frozen) // anchor
		if got := v.dragMoved; got != 0 {
			t.Fatalf("after the anchor tick the accumulator is %d, want 0 — a gesture starts at zero", got)
		}

		v.step(Input{PrimaryDown: true, CursorX: 410, CursorY: 305}, frozen) // |10| + |5|
		if got := v.dragMoved; got != 15 {
			t.Fatalf("accumulator = %d, want 15 (|10| + |5|)", got)
		}

		// Back to the anchor position: the DISPLACEMENT is now zero and the
		// path is not. A press that wanders out and back panned that far and is
		// a drag (R-1).
		v.step(Input{PrimaryDown: true, CursorX: 400, CursorY: 300}, frozen) // |-10| + |-5|
		if got := v.dragMoved; got != 30 {
			t.Errorf("accumulator = %d, want 30 — it is a path length, not a displacement", got)
		}
	})

	t.Run("a drag cannot pan the camera past the clamp", func(t *testing.T) {
		v := stepViewer(t)
		v.Camera().X, v.Camera().Y = 0, 0
		v.Camera().Clamp()

		v.step(Input{PrimaryDown: true, CursorX: centre.x, CursorY: centre.y}, frozen)
		// A cursor delta far larger than the world could ever need.
		v.step(Input{PrimaryDown: true, CursorX: centre.x + 5000, CursorY: centre.y + 5000}, frozen)

		if v.Camera().X < 0 {
			t.Errorf("camera X = %v, want >= 0 (clamp)", v.Camera().X)
		}
		if v.Camera().Y < 0 {
			t.Errorf("camera Y = %v, want >= 0 (clamp)", v.Camera().Y)
		}
	})
}

// dragGesture is the one three-tick gesture every latch case below runs, at the
// same positions the shipped drag test uses: an anchor at (400,300), then
// (450,320) — a screen delta of (50,20) — then (420,290), a delta of (-30,-30)
// from THERE and not from the press.
//
// At zoom 1 the unmodified drag therefore pans by (-50+30, -20+30) = (-20,+10)
// world pixels, and the accumulator rises by (50+20) + (30+30) = 130 whichever
// gesture the press latched. Both numbers are the camera contract's, written out
// here rather than read back off the code under test.
const dragGesturePan = 130

func dragGesture(t *testing.T, commandMode bool, shift []bool) (v *Viewer, dx, dy float64) {
	t.Helper()
	v = stepViewer(t)
	v.commandMode = commandMode
	x0, y0 := v.Camera().X, v.Camera().Y
	if got := v.Camera().Zoom; got != 1 {
		t.Fatalf("fixture zoom = %v, want 1 — every delta below is stated at native zoom", got)
	}
	for i, p := range [3][2]int{{400, 300}, {450, 320}, {420, 290}} {
		v.step(Input{PrimaryDown: true, Shift: shift[i], CursorX: p[0], CursorY: p[1]}, dragFrozen)
	}
	return v, v.Camera().X - x0, v.Camera().Y - y0
}

var dragFrozen = time.Unix(1_700_000_000, 0)

// held3 is one modifier level held for all three ticks of dragGesture.
func held3(down bool) []bool { return []bool{down, down, down} }

func TestTheLeftDragMarqueesOnTheMapAndPansOnlyInTheStandaloneViewer(t *testing.T) {
	for _, tc := range []struct {
		name        string
		commandMode bool
		shift       bool
		dx, dy      float64
	}{
		{"the standalone viewer: a plain drag pans", false, false, -20, +10},
		{"the standalone viewer: no modifier alters it", false, true, -20, +10},
		{"the map screen: a plain drag pans by zero", true, false, 0, 0},
		{"the map screen: Shift pans by zero too", true, true, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, dx, dy := dragGesture(t, tc.commandMode, held3(tc.shift))

			if dx != tc.dx || dy != tc.dy {
				t.Errorf("the camera moved by (%v,%v), want (%v,%v)", dx, dy, tc.dx, tc.dy)
			}
			if got := v.dragMoved; got != dragGesturePan {
				t.Errorf("the accumulator rose by %d, want %d — the slop measures the same path length "+
					"for both gestures", got, dragGesturePan)
			}
		})
	}
}

// TestTheMapGestureIsFixedAtThePressAndNotReadAgain — 0030 SC-1 (AC-8): a
// modifier pressed or released mid-drag does not change what the press is.
//
// THE STANDALONE VIEWER IS WHERE THE CLAIM IS STILL FALSIFIABLE, so the two
// rows below run there: it pans on every left drag, whatever the modifier does
// mid-gesture, and a build that had started reading the modifier live would
// answer one of the two rows differently.
func TestTheMapGestureIsFixedAtThePressAndNotReadAgain(t *testing.T) {
	for _, tc := range []struct {
		name        string
		commandMode bool
		shift       []bool
		dx, dy      float64
	}{
		{"the map screen, Shift pressed mid-drag: still a marquee, still pans zero",
			true, []bool{false, true, true}, 0, 0},
		{"the map screen, Shift released mid-drag: still a marquee, still pans zero",
			true, []bool{true, false, false}, 0, 0},
		{"the standalone viewer, Shift pressed mid-drag: still the whole pan",
			false, []bool{false, true, true}, -20, +10},
		{"the standalone viewer, Shift released mid-drag: still the whole pan",
			false, []bool{true, false, false}, -20, +10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, dx, dy := dragGesture(t, tc.commandMode, tc.shift)

			if dx != tc.dx || dy != tc.dy {
				t.Errorf("the camera moved by (%v,%v), want (%v,%v)", dx, dy, tc.dx, tc.dy)
			}
			if got := v.dragMoved; got != dragGesturePan {
				t.Errorf("the accumulator rose by %d, want %d", got, dragGesturePan)
			}
		})
	}
}

// TestTheRightDragPansTheMissionCamera is `AI-INPUT-127`'s capture-to-pan
// half: right down captures and stores the origin, each delivered move while
// the button is down pans, and right up releases capture.
//
// IT IS THE SAME THREE-TICK GESTURE the left cases above run, on the other
// button, so the two are directly comparable: the map screen answers (0,0) to
// the left drag and the whole (-20,+10) to the right one.
//
// THE MARK IS ASSERTED TOO. A drag that moved must mark itself, because
// `AI-INPUT-127` makes the mark the difference between a right release that
// cancels and one that does not.
func TestTheRightDragPansTheMissionCamera(t *testing.T) {
	v := stepViewer(t)
	v.commandMode = true
	x0, y0 := v.Camera().X, v.Camera().Y
	if got := v.Camera().Zoom; got != 1 {
		t.Fatalf("fixture zoom = %v, want 1 — every delta below is stated at native zoom", got)
	}
	for _, p := range [3][2]int{{400, 300}, {450, 320}, {420, 290}} {
		v.step(Input{SecondaryDown: true, CursorX: p[0], CursorY: p[1]}, dragFrozen)
	}
	if dx, dy := v.Camera().X-x0, v.Camera().Y-y0; dx != -20 || dy != +10 {
		t.Errorf("the camera moved by (%v,%v), want (-20,10)", dx, dy)
	}
	if !v.rightPanned {
		t.Error("the right drag did not mark itself; a marked drag is what suppresses the cancel")
	}
	if got := v.dragMoved; got != 0 {
		t.Errorf("the right drag raised the LEFT accumulator to %d, want 0 — the two gestures are "+
			"measured apart", got)
	}

	// THE MARK SURVIVES THE RELEASE TICK ITSELF and is spent on the next one.
	// command reads it on the tick carrying SecondaryReleased, and step runs
	// before command, so a mark cleared on that tick would make every drag
	// release read as a click.
	v.step(Input{CursorX: 420, CursorY: 290}, dragFrozen)
	if !v.rightPanned {
		t.Error("the mark was cleared on the release tick, before command could read it")
	}
	v.step(Input{CursorX: 420, CursorY: 290}, dragFrozen)
	if v.rightPanned {
		t.Error("the mark outlived the release by more than one tick; the NEXT right click would not cancel")
	}
}
