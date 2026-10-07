package frame_test

// Spec-derived acceptance test for the 640x480 virtual frame (SC-2).
//
// Every expectation is computed here, independently, from that contract:
// the scale is the exact rational min(winW/640, winH/480), the frame is
// centered, and frame pixel f owns the half-open window interval
// [origin + f*s, origin + (f+1)*s). The oracle evaluates that definition in
// exact integer arithmetic (multiplying through by 2*den), so nothing below is
// compared against a value produced by the code under test, and nothing is
// compared with a tolerance.

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/frame"
)

// The virtual frame's size, written as literals so the test carries its own
// idea of the contract rather than inheriting the package's.
const (
	frameW = 640
	frameH = 480
)

// otherFrames are frame sizes this file runs the whole contract at BESIDES
// the pair above (1026 B1). The mission screen's own 1024x768 is here,
// together with two sizes that share no common factor with it and one taller
// than it is wide.
var otherFrames = []struct {
	name string
	w, h int
}{
	{"1024x768, the mission frame", 1024, 768},
	{"800x600", 800, 600},
	{"320x200, not 4:3", 320, 200},
	{"97x131, taller than wide and coprime", 97, 131},
}

// ---------------------------------------------------------------------------
// oracle: the contract, transcribed in exact integer arithmetic
// ---------------------------------------------------------------------------

// ceilDiv returns ceil(a/b) for b > 0 and a of either sign. Go truncates
// toward zero, which is already ceil for a < 0.
func ceilDiv(a, b int64) int64 {
	q := a / b
	if a > 0 && a%b != 0 {
		q++
	}
	return q
}

// withinOneULP reports whether got is want or one of its two float64
// neighbours. A direct ulp comparison, not a tuned epsilon: the draw transform
// is allowed to round, and nothing more.
func withinOneULP(got, want float64) bool {
	return got == want ||
		got == math.Nextafter(want, math.Inf(1)) ||
		got == math.Nextafter(want, math.Inf(-1))
}

type oracle struct {
	frameW, frameH int64
	winW, winH     int
	valid          bool
	num, den       int64 // the uniform scale, exactly num/den = min(winW/640, winH/480)

	colOf, rowOf       []int // window pixel -> frame pixel, -1 when in the letterbox
	colFirst, rowFirst []int // frame pixel -> its first window pixel, -1 when it owns none
}

func newOracle(frameW, frameH, winW, winH int) *oracle {
	o := &oracle{frameW: int64(frameW), frameH: int64(frameH), winW: winW, winH: winH}
	if frameW <= 0 || frameH <= 0 || winW <= 0 || winH <= 0 {
		// A non-positive window has no placement at all.
		return o
	}
	o.valid = true

	// One uniform factor, the largest that fits: min(winW/640, winH/480), kept
	// as an exact rational and selected by an exact integer comparison so the
	// choice never depends on rounding. Ties (an exactly 4:3 window) take the
	// width-limited form; both forms are then the same real number.
	if int64(winW)*o.frameH <= int64(winH)*o.frameW {
		o.num, o.den = int64(winW), o.frameW
	} else {
		o.num, o.den = int64(winH), o.frameH
	}

	o.colOf, o.colFirst = axisOracle(int64(winW), o.frameW, o.num, o.den)
	o.rowOf, o.rowFirst = axisOracle(int64(winH), o.frameH, o.num, o.den)
	return o
}

// axisOracle evaluates, on one axis, "the frame is centered and frame pixel f
// covers the half-open window interval [origin + f*s, origin + (f+1)*s)", with
// s = num/den and origin = (win - size*s)/2. Multiplying every bound by 2*den
// (positive) removes the division: f covers the integers x with
//
//	a + 2*f*num <= 2*den*x < a + 2*(f+1)*num,  a = win*den - size*num = 2*den*origin
//
// i.e. ceil((a+2*f*num)/(2*den)) <= x < ceil((a+2*(f+1)*num)/(2*den)).
func axisOracle(win, size, num, den int64) (ownerOf, firstOf []int) {
	ownerOf = make([]int, win)
	for i := range ownerOf {
		ownerOf[i] = -1
	}
	firstOf = make([]int, size)

	a := win*den - size*num
	for f := int64(0); f < size; f++ {
		lo := ceilDiv(a+2*f*num, 2*den)
		hi := ceilDiv(a+2*(f+1)*num, 2*den)
		if lo >= hi {
			// The interval holds no whole window pixel: this frame pixel is not
			// in the image of the window->frame mapping (only possible below 1:1).
			firstOf[f] = -1
			continue
		}
		firstOf[f] = int(lo)
		for x := lo; x < hi; x++ {
			if x >= 0 && x < win {
				ownerOf[x] = int(f)
			}
		}
	}
	return ownerOf, firstOf
}

func (o *oracle) windowToFrame(x, y int) (image.Point, bool) {
	if !o.valid || x < 0 || y < 0 || x >= o.winW || y >= o.winH {
		return image.Point{}, false
	}
	fx, fy := o.colOf[x], o.rowOf[y]
	if fx < 0 || fy < 0 {
		return image.Point{}, false
	}
	return image.Pt(fx, fy), true
}

func (o *oracle) frameToWindow(fx, fy int) (int, int, bool) {
	if !o.valid || fx < 0 || fy < 0 || fx >= int(o.frameW) || fy >= int(o.frameH) {
		return 0, 0, false
	}
	x, y := o.colFirst[fx], o.rowFirst[fy]
	if x < 0 || y < 0 {
		return 0, 0, false
	}
	return x, y, true
}

// bands reports the letterbox extents on one axis: [0, before) and [after, win)
// hold no frame pixel.
func bands(win, size, num, den int64) (before, after int64) {
	a := win*den - size*num
	return ceilDiv(a, 2*den), ceilDiv(a+2*size*num, 2*den)
}

// ---------------------------------------------------------------------------
// the full check, run at every window size
// ---------------------------------------------------------------------------

// checkPlacement asserts the whole contract at one FRAME size and one window
// size (1026 B1: the frame size is a parameter, so this file can no longer pass
// against an implementation that reads a package constant instead of the size it
// was given). It asserts containment and
// a single undistorted scale, the letterbox and out-of-window refusals, that
// every window position maps to at most one in-range frame pixel, that the two
// mappings round-trip exactly in both directions, and that FrameToWindow's ok
// is exactly "some window pixel maps to this frame pixel".
func checkPlacement(t *testing.T, fw, fh, winW, winH int) {
	t.Helper()

	o := newOracle(fw, fh, winW, winH)
	p := frame.Fit(fw, fh, winW, winH)

	if got := p.Valid(); got != o.valid {
		t.Fatalf("Fit(%dx%d frame, %d,%d window).Valid() = %v, want %v", fw, fh, winW, winH, got, o.valid)
	}
	if !o.valid {
		return
	}

	wantScale := float64(o.num) / float64(o.den)
	if got := p.Scale(); got != wantScale {
		t.Fatalf("Fit(%dx%d frame, %d,%d window).Scale() = %v, want exactly %v (= %d/%d)",
			fw, fh, winW, winH, got, wantScale, o.num, o.den)
	}
	if p.Scale() <= 0 {
		t.Fatalf("Fit(%dx%d frame, %d,%d window).Scale() = %v, want a positive factor", fw, fh, winW, winH, p.Scale())
	}

	if int64(fw)*o.num > int64(winW)*o.den {
		t.Errorf("Fit(%dx%d frame, %d,%d window): scale %d/%d puts the frame's %d px width past the window: %d > %d",
			fw, fh, winW, winH, o.num, o.den, fw, int64(fw)*o.num, int64(winW)*o.den)
	}
	if int64(fh)*o.num > int64(winH)*o.den {
		t.Errorf("Fit(%dx%d frame, %d,%d window): scale %d/%d puts the frame's %d px height past the window: %d > %d",
			fw, fh, winW, winH, o.num, o.den, fh, int64(fh)*o.num, int64(winH)*o.den)
	}

	// The reported origin is non-negative exactly -- nothing is drawn off the
	// top or left edge -- and is the centred origin to within one ulp.
	// Reported with Errorf, not Fatalf, so a bad draw transform does not hide
	// the state of the integer mappings below.
	ox, oy := p.Origin()
	if !(ox >= 0) || !(oy >= 0) {
		t.Errorf("Fit(%dx%d frame, %d,%d window).Origin() = (%v,%v), want both >= 0", fw, fh, winW, winH, ox, oy)
	}
	wantOX := float64(int64(winW)*o.den-int64(fw)*o.num) / float64(2*o.den)
	wantOY := float64(int64(winH)*o.den-int64(fh)*o.num) / float64(2*o.den)
	if !withinOneULP(ox, wantOX) || !withinOneULP(oy, wantOY) {
		t.Errorf("Fit(%dx%d frame, %d,%d window).Origin() = (%v,%v), want the centred origin (%v,%v) to within one ulp",
			fw, fh, winW, winH, ox, oy, wantOX, wantOY)
	}

	// Sweep every window position. A function maps each position to at most one
	// frame pixel by construction, so the useful form is asserted instead: each
	// position either refuses or names an in-range frame pixel, and it is the
	// pixel whose half-open interval covers it.
	hit := make([]bool, fw*fh)
	for y := 0; y < winH; y++ {
		for x := 0; x < winW; x++ {
			got, ok := p.WindowToFrame(x, y)
			want, wantOK := o.windowToFrame(x, y)
			if ok != wantOK {
				t.Fatalf("%dx%d: WindowToFrame(%d,%d) ok = %v, want %v (frame pixel %v)",
					winW, winH, x, y, ok, wantOK, want)
			}
			if !ok {
				continue
			}
			if got != want {
				t.Fatalf("%dx%d: WindowToFrame(%d,%d) = %v, want %v", winW, winH, x, y, got, want)
			}
			if got.X < 0 || got.X >= fw || got.Y < 0 || got.Y >= fh {
				t.Fatalf("%dx%d: WindowToFrame(%d,%d) = %v, outside [0,%d)x[0,%d)",
					winW, winH, x, y, got, fw, fh)
			}
			hit[got.Y*fw+got.X] = true

			// Round trip window -> frame -> window: the frame pixel must report a
			// window position of its own, that position must not be past the one
			// we came from, and it must map back to the same frame pixel.
			bx, by, bok := p.FrameToWindow(got)
			if !bok {
				t.Fatalf("%dx%d: window (%d,%d) maps to frame %v but FrameToWindow(%v) reports no window pixel",
					winW, winH, x, y, got, got)
			}
			if bx > x || by > y {
				t.Fatalf("%dx%d: window (%d,%d) maps to frame %v, but FrameToWindow(%v) = (%d,%d), past it",
					winW, winH, x, y, got, got, bx, by)
			}
			if back, backOK := p.WindowToFrame(bx, by); !backOK || back != got {
				t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d), which maps to %v,%v, want %v,true",
					winW, winH, got, bx, by, back, backOK, got)
			}
		}
	}

	// Sweep every frame pixel: exact agreement with the first window pixel of
	// its interval, exact round trip, and ok == "the window sweep reached it".
	for fy := 0; fy < fh; fy++ {
		for fx := 0; fx < fw; fx++ {
			pt := image.Pt(fx, fy)
			x, y, ok := p.FrameToWindow(pt)
			wx, wy, wantOK := o.frameToWindow(fx, fy)
			if ok != wantOK {
				t.Fatalf("%dx%d: FrameToWindow(%v) ok = %v, want %v (window pixel (%d,%d))",
					winW, winH, pt, ok, wantOK, wx, wy)
			}
			if ok != hit[fy*fw+fx] {
				t.Fatalf("%dx%d: FrameToWindow(%v) ok = %v, but sweeping the whole window %s that frame pixel",
					winW, winH, pt, ok, map[bool]string{true: "does reach", false: "never reaches"}[hit[fy*fw+fx]])
			}
			if !ok {
				continue
			}
			if x != wx || y != wy {
				t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d), want the interval's first window pixel (%d,%d)",
					winW, winH, pt, x, y, wx, wy)
			}
			if x < 0 || x >= winW || y < 0 || y >= winH {
				t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d), outside the window", winW, winH, pt, x, y)
			}
			if got, gok := p.WindowToFrame(x, y); !gok || got != pt {
				t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d), which maps back to %v,%v, want %v,true",
					winW, winH, pt, x, y, got, gok, pt)
			}
		}
	}

	// Positions outside the window, on either axis and both, map to no frame pixel.
	outside := [][2]int{
		{-1, 0}, {0, -1}, {-1, -1},
		{-1, winH / 2}, {winW / 2, -1},
		{winW, 0}, {0, winH}, {winW, winH},
		{winW, winH / 2}, {winW / 2, winH},
		{winW + 7, winH + 3}, {-1000, 3}, {3, -1000},
		{1 << 20, 1 << 20}, {-(1 << 20), -(1 << 20)},
	}
	for _, q := range outside {
		if got, ok := p.WindowToFrame(q[0], q[1]); ok {
			t.Fatalf("%dx%d: WindowToFrame(%d,%d) is outside the window but reported frame pixel %v",
				winW, winH, q[0], q[1], got)
		}
	}

	// Points that are not frame pixels have no window pixel either.
	notFramePixels := []image.Point{
		{X: -1, Y: 0}, {X: 0, Y: -1}, {X: -1, Y: -1},
		{X: fw, Y: 0}, {X: 0, Y: fh}, {X: fw, Y: fh},
		{X: fw + 5, Y: 12}, {X: 12, Y: fh + 5},
	}
	for _, pt := range notFramePixels {
		if x, y, ok := p.FrameToWindow(pt); ok {
			t.Fatalf("%dx%d: FrameToWindow(%v) is outside [0,%d)x[0,%d) but reported window (%d,%d)",
				winW, winH, pt, fw, fh, x, y)
		}
	}
}

// ---------------------------------------------------------------------------
// the test
// ---------------------------------------------------------------------------

func TestFitAndMapping(t *testing.T) {
	t.Run("the town family's frame constants are what this file's default case uses", func(t *testing.T) {
		if frame.W != frameW || frame.H != frameH {
			t.Fatalf("frame.W, frame.H = %d, %d, want %d, %d", frame.W, frame.H, frameW, frameH)
		}
	})

	// 1026 B1. The same contract, at frame sizes other than the pair above and
	// at windows that letterbox on each axis in turn, scale above 1:1 and scale
	// below it. A mapping that still read a package constant would fail here at
	// its first assertion and pass everything else in this file.
	t.Run("the whole contract holds at other frame sizes", func(t *testing.T) {
		for _, f := range otherFrames {
			for _, win := range []struct {
				name string
				w, h int
			}{
				{"at its own size", f.w, f.h},
				{"in a wider window", f.w*2 + 37, f.h * 2},
				{"in a taller window", f.w, f.h*3/2 + 5},
				{"in a smaller window", f.w/3 + 1, f.h / 3},
			} {
				t.Run(f.name+" "+win.name, func(t *testing.T) {
					checkPlacement(t, f.w, f.h, win.w, win.h)
				})
			}
		}
	})

	// The placement reports the frame it was fitted for, so a caller holding two
	// of them can tell which is which.
	t.Run("a placement reports its own frame size", func(t *testing.T) {
		for _, f := range append([]struct {
			name string
			w, h int
		}{{"640x480", frameW, frameH}}, otherFrames...) {
			p := frame.Fit(f.w, f.h, 1000, 1000)
			if got := p.FrameSize(); got.X != f.w || got.Y != f.h {
				t.Fatalf("Fit(%d,%d,1000,1000).FrameSize() = %v, want (%d,%d)", f.w, f.h, got, f.w, f.h)
			}
		}
		if got := (frame.Placement{}).FrameSize(); got != (image.Point{}) {
			t.Fatalf("the zero Placement reports frame size %v, want the zero point", got)
		}
	})

	// A frame with a non-positive side has no placement, for the same reason a
	// window with one does not.
	t.Run("a non-positive frame is invalid and refuses every mapping", func(t *testing.T) {
		for _, f := range [][2]int{{0, 0}, {0, 480}, {640, 0}, {-1, 480}, {640, -9}} {
			p := frame.Fit(f[0], f[1], 800, 600)
			if p.Valid() {
				t.Fatalf("Fit(%d,%d,800,600).Valid() = true, want false", f[0], f[1])
			}
			if _, ok := p.WindowToFrame(10, 10); ok {
				t.Fatalf("Fit(%d,%d,800,600).WindowToFrame(10,10) reported a frame pixel", f[0], f[1])
			}
			if _, ok := p.WindowToFrameExtended(10, 10); ok {
				t.Fatalf("Fit(%d,%d,800,600).WindowToFrameExtended(10,10) reported a frame position", f[0], f[1])
			}
			if _, _, ok := p.FrameToWindow(image.Pt(0, 0)); ok {
				t.Fatalf("Fit(%d,%d,800,600).FrameToWindow(0,0) reported a window pixel", f[0], f[1])
			}
		}
	})

	// WindowToFrameExtended agrees with WindowToFrame wherever WindowToFrame
	// answers, and continues the same lattice where it refuses: a position one
	// window pixel outside the frame's own span lands one frame pixel outside
	// it, on the correct side. The expectation is built from the placement's own
	// reported ORIGIN and SCALE rather than from the mapping under test.
	t.Run("the extended mapping continues the lattice past the frame", func(t *testing.T) {
		for _, f := range append([]struct {
			name string
			w, h int
		}{{"640x480", frameW, frameH}}, otherFrames...) {
			for _, win := range [][2]int{{f.w, f.h}, {f.w * 3, f.h * 2}, {f.w*2 + 11, f.h * 2}} {
				p := frame.Fit(f.w, f.h, win[0], win[1])
				ox, oy := p.Origin()
				s := p.Scale()

				// Inside: the two mappings agree everywhere WindowToFrame answers.
				for y := 0; y < win[1]; y += 7 {
					for x := 0; x < win[0]; x += 7 {
						in, okIn := p.WindowToFrame(x, y)
						ext, okExt := p.WindowToFrameExtended(x, y)
						if !okExt {
							t.Fatalf("%s in %dx%d: WindowToFrameExtended(%d,%d) refused a usable placement", f.name, win[0], win[1], x, y)
						}
						if okIn && ext != in {
							t.Fatalf("%s in %dx%d: WindowToFrameExtended(%d,%d) = %v, WindowToFrame = %v",
								f.name, win[0], win[1], x, y, ext, in)
						}
						if !okIn && ext.X >= 0 && ext.X < f.w && ext.Y >= 0 && ext.Y < f.h {
							t.Fatalf("%s in %dx%d: (%d,%d) is in the letterbox but the extended mapping put it at %v, inside the frame",
								f.name, win[0], win[1], x, y, ext)
						}
					}
				}

				// Outside: one window pixel left of the frame's own first
				// window column is a negative frame x, and one past its last is
				// at least the frame width. Both bounds come from Origin() and
				// Scale(), not from the mapping.
				left := int(ox) - 1
				right := int(ox+float64(f.w)*s) + 1
				above := int(oy) - 1
				below := int(oy+float64(f.h)*s) + 1
				mid := int(oy + float64(f.h)*s/2)
				if got, _ := p.WindowToFrameExtended(left, mid); got.X >= 0 {
					t.Fatalf("%s in %dx%d: window x=%d is left of the frame, extended mapping gave x=%d, want negative",
						f.name, win[0], win[1], left, got.X)
				}
				if got, _ := p.WindowToFrameExtended(right, mid); got.X < f.w {
					t.Fatalf("%s in %dx%d: window x=%d is right of the frame, extended mapping gave x=%d, want at least %d",
						f.name, win[0], win[1], right, got.X, f.w)
				}
				midX := int(ox + float64(f.w)*s/2)
				if got, _ := p.WindowToFrameExtended(midX, above); got.Y >= 0 {
					t.Fatalf("%s in %dx%d: window y=%d is above the frame, extended mapping gave y=%d, want negative",
						f.name, win[0], win[1], above, got.Y)
				}
				if got, _ := p.WindowToFrameExtended(midX, below); got.Y < f.h {
					t.Fatalf("%s in %dx%d: window y=%d is below the frame, extended mapping gave y=%d, want at least %d",
						f.name, win[0], win[1], below, got.Y, f.h)
				}
			}
		}
	})

	sizes := []struct {
		name string
		w, h int
	}{
		{"1024x600 wider than 4:3", 1024, 600},
		{"800x480 wider than 4:3", 800, 480},
		{"700x500 wider than 4:3", 700, 500},
		{"600x800 taller than 4:3", 600, 800},
		{"640x600 taller than 4:3", 640, 600},
		{"500x700 taller than 4:3", 500, 700},
		{"800x600 exactly 4:3", 800, 600},
		{"1280x960 exactly 4:3", 1280, 960},
		{"640x480 exactly the frame", 640, 480},
		{"320x240 smaller, exactly 4:3", 320, 240},
		{"199x321 smaller, taller than 4:3", 199, 321},
		{"100x80 smaller, wider than 4:3", 100, 80},
		{"641x480 one pixel wider than the frame", 641, 480},
	}
	for _, tc := range sizes {
		t.Run("containment, mapping and round trip at "+tc.name, func(t *testing.T) {
			checkPlacement(t, frameW, frameH, tc.w, tc.h)
		})
	}

	// The sizes the plan names as the ones a float64 implementation gets wrong:
	// at these it round-trips a frame pixel off by one, or puts the origin
	// slightly below zero. Their own subtest so a failure points at the cause.
	t.Run("sizes where a float implementation provably fails", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			w, h int
		}{
			{"700x500", 700, 500},
			{"644x494", 644, 494},
			{"642x481", 642, 481},
		} {
			t.Run(tc.name, func(t *testing.T) {
				checkPlacement(t, frameW, frameH, tc.w, tc.h)
			})
		}
	})

	t.Run("letterbox bands map to no frame pixel", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			w, h int
		}{
			{"1024x600 has left and right bands", 1024, 600},
			{"640x600 has top and bottom bands", 640, 600},
			{"700x500", 700, 500},
			{"500x700", 500, 700},
			{"100x80", 100, 80},
			{"642x481", 642, 481},
			{"644x494", 644, 494},
		} {
			t.Run(tc.name, func(t *testing.T) {
				o := newOracle(frameW, frameH, tc.w, tc.h)
				p := frame.Fit(frameW, frameH, tc.w, tc.h)

				leftEnd, rightStart := bands(int64(tc.w), frameW, o.num, o.den)
				topEnd, bottomStart := bands(int64(tc.h), frameH, o.num, o.den)
				if leftEnd == 0 && int64(tc.w) == rightStart && topEnd == 0 && int64(tc.h) == bottomStart {
					t.Fatalf("%dx%d: no letterbox band at all, this case proves nothing", tc.w, tc.h)
				}

				for y := 0; y < tc.h; y++ {
					for _, x := range columnsIn(0, int(leftEnd), int(rightStart), tc.w) {
						if got, ok := p.WindowToFrame(x, y); ok {
							t.Fatalf("%dx%d: WindowToFrame(%d,%d) is in the left/right letterbox (frame occupies x in [%d,%d)) but reported %v",
								tc.w, tc.h, x, y, leftEnd, rightStart, got)
						}
					}
				}
				for x := 0; x < tc.w; x++ {
					for _, y := range columnsIn(0, int(topEnd), int(bottomStart), tc.h) {
						if got, ok := p.WindowToFrame(x, y); ok {
							t.Fatalf("%dx%d: WindowToFrame(%d,%d) is in the top/bottom letterbox (frame occupies y in [%d,%d)) but reported %v",
								tc.w, tc.h, x, y, topEnd, bottomStart, got)
						}
					}
				}
			})
		}
	})

	t.Run("ok from FrameToWindow means exactly one window pixel maps here", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			w, h int
		}{
			{"320x240 halves the frame", 320, 240},
			{"213x160 thirds the frame", 213, 160},
			{"100x80", 100, 80},
			{"199x321", 199, 321},
		} {
			t.Run(tc.name, func(t *testing.T) {
				p := frame.Fit(frameW, frameH, tc.w, tc.h)

				// Which frame pixels the whole window actually reaches.
				reached := make([]bool, frameW*frameH)
				for y := 0; y < tc.h; y++ {
					for x := 0; x < tc.w; x++ {
						if pt, ok := p.WindowToFrame(x, y); ok {
							reached[pt.Y*frameW+pt.X] = true
						}
					}
				}

				refused := 0
				for fy := 0; fy < frameH; fy++ {
					for fx := 0; fx < frameW; fx++ {
						pt := image.Pt(fx, fy)
						x, y, ok := p.FrameToWindow(pt)
						switch {
						case ok && !reached[fy*frameW+fx]:
							t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d),true but no window position maps to %v",
								tc.w, tc.h, pt, x, y, pt)
						case !ok && reached[fy*frameW+fx]:
							t.Fatalf("%dx%d: FrameToWindow(%v) reports no window pixel, but a window position maps to %v",
								tc.w, tc.h, pt, pt)
						case ok:
							if got, gok := p.WindowToFrame(x, y); !gok || got != pt {
								t.Fatalf("%dx%d: FrameToWindow(%v) = (%d,%d), which maps to %v,%v, want %v,true",
									tc.w, tc.h, pt, x, y, got, gok, pt)
							}
						default:
							refused++
						}
					}
				}
				if refused == 0 {
					t.Fatalf("%dx%d is scaled below 1:1, so some frame pixels must have no window pixel; none did", tc.w, tc.h)
				}
			})
		}
	})

	t.Run("a 1x1 window still contains the whole frame", func(t *testing.T) {
		p := frame.Fit(frameW, frameH, 1, 1)
		if !p.Valid() {
			t.Fatalf("Fit(1,1).Valid() = false, want true")
		}
		// min(1/640, 1/480) = 1/640.
		if want := 1.0 / 640.0; p.Scale() != want {
			t.Fatalf("Fit(1,1).Scale() = %v, want exactly %v", p.Scale(), want)
		}
		if p.Scale() <= 0 {
			t.Fatalf("Fit(1,1).Scale() = %v, want a positive factor", p.Scale())
		}
		ox, oy := p.Origin()
		if !(ox >= 0) || !(oy >= 0) {
			t.Fatalf("Fit(1,1).Origin() = (%v,%v), want both >= 0", ox, oy)
		}
		// Containment is left to checkPlacement, which asserts it in the same
		// exact-rational int64 form as every other size (scale 1/640 here, so the
		// 640 px width occupies exactly the 1 px window and the 480 px height
		// less). Repeating it in the float form would be the one place in this
		// file still asserting a property float64 cannot carry -- see DD5 and
		// checkPlacement's own note.
		checkPlacement(t, frameW, frameH, 1, 1)
	})

	// A window with a non-positive dimension has no placement: nothing is valid,
	// nothing maps, and nothing divides by zero.
	t.Run("a non-positive window is invalid and refuses every mapping", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			w, h int
		}{
			{"0x0", 0, 0},
			{"-3x9", -3, 9},
			{"5x0", 5, 0},
			{"0x480", 0, 480},
			{"640x-1", 640, -1},
			{"-7x-7", -7, -7},
		} {
			t.Run(tc.name, func(t *testing.T) {
				p := frame.Fit(frameW, frameH, tc.w, tc.h)
				if p.Valid() {
					t.Fatalf("Fit(%d,%d).Valid() = true, want false", tc.w, tc.h)
				}
				// Reading the draw transform must not divide by zero.
				s := p.Scale()
				ox, oy := p.Origin()
				if math.IsNaN(s) || math.IsInf(s, 0) {
					t.Fatalf("Fit(%d,%d).Scale() = %v, want a finite value", tc.w, tc.h, s)
				}
				if math.IsNaN(ox) || math.IsInf(ox, 0) || math.IsNaN(oy) || math.IsInf(oy, 0) {
					t.Fatalf("Fit(%d,%d).Origin() = (%v,%v), want finite values", tc.w, tc.h, ox, oy)
				}

				for _, q := range [][2]int{{0, 0}, {1, 1}, {-1, -1}, {5, 5}, {319, 239}, {639, 479}, {tc.w, tc.h}} {
					if got, ok := p.WindowToFrame(q[0], q[1]); ok {
						t.Fatalf("Fit(%d,%d): WindowToFrame(%d,%d) = %v,true, want no frame pixel",
							tc.w, tc.h, q[0], q[1], got)
					}
				}
				for _, pt := range []image.Point{{X: 0, Y: 0}, {X: 320, Y: 240}, {X: 639, Y: 479}, {X: -1, Y: 0}, {X: frameW, Y: frameH}} {
					if x, y, ok := p.FrameToWindow(pt); ok {
						t.Fatalf("Fit(%d,%d): FrameToWindow(%v) = (%d,%d),true, want no window pixel",
							tc.w, tc.h, pt, x, y)
					}
				}
			})
		}
	})

	t.Run("hand-computed placements", func(t *testing.T) {
		type wProbe struct {
			x, y   int
			want   image.Point
			wantOK bool
		}
		type fProbe struct {
			pt     image.Point
			x, y   int
			wantOK bool
		}
		cases := []struct {
			name      string
			w, h      int
			wantScale float64
			wProbes   []wProbe
			fProbes   []fProbe
		}{
			{
				// 800*480 == 600*640, so 4:3 exactly: s = 800/640 = 1.25, origin (0,0).
				name: "800x600 fills the window at 1.25",
				w:    800, h: 600, wantScale: 1.25,
				wProbes: []wProbe{
					{0, 0, image.Pt(0, 0), true},
					{1, 0, image.Pt(0, 0), true}, // frame col 0 covers [0,1.25)
					{2, 0, image.Pt(1, 0), true}, // frame col 1 covers [1.25,2.5)
					{799, 599, image.Pt(639, 479), true},
					{800, 0, image.Point{}, false},
					{-1, 0, image.Point{}, false},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 0, 0, true},
					{image.Pt(1, 0), 2, 0, true}, // ceil(1.25) = 2
					{image.Pt(639, 479), 799, 599, true},
				},
			},
			{
				// 1024*480 > 600*640, so height-limited: s = 600/480 = 1.25,
				// origin.x = (1024 - 800)/2 = 112, origin.y = 0.
				name: "1024x600 letterboxes 112px left and right",
				w:    1024, h: 600, wantScale: 1.25,
				wProbes: []wProbe{
					{111, 0, image.Point{}, false},
					{112, 0, image.Pt(0, 0), true},
					{911, 599, image.Pt(639, 479), true},
					{912, 300, image.Point{}, false},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 112, 0, true},
					{image.Pt(639, 479), 911, 599, true},
				},
			},
			{
				// 640*480 <= 600*640, so width-limited: s = 1, origin.y = (600-480)/2 = 60.
				name: "640x600 letterboxes 60px top and bottom",
				w:    640, h: 600, wantScale: 1,
				wProbes: []wProbe{
					{0, 59, image.Point{}, false},
					{0, 60, image.Pt(0, 0), true},
					{639, 539, image.Pt(639, 479), true},
					{320, 540, image.Point{}, false},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 0, 60, true},
					{image.Pt(639, 479), 639, 539, true},
				},
			},
			{
				// Exactly half size: frame col f covers [f/2, (f+1)/2), so odd
				// columns hold no whole window pixel.
				name: "320x240 halves the frame, odd frame pixels have no window pixel",
				w:    320, h: 240, wantScale: 0.5,
				wProbes: []wProbe{
					{0, 0, image.Pt(0, 0), true},
					{1, 0, image.Pt(2, 0), true},
					{319, 239, image.Pt(638, 478), true},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 0, 0, true},
					{image.Pt(1, 0), 0, 0, false},
					{image.Pt(2, 0), 1, 0, true},
					{image.Pt(639, 479), 0, 0, false},
				},
			},
			{
				// 644*480 = 309120 < 494*640 = 316160, so width-limited:
				// s = 644/640 = 1.00625, origin.y = (494 - 483)/2 = 5.5.
				name: "644x494 puts the origin on a half pixel",
				w:    644, h: 494, wantScale: 644.0 / 640.0,
				wProbes: []wProbe{
					{0, 5, image.Point{}, false},
					{0, 6, image.Pt(0, 0), true},
					{643, 488, image.Pt(639, 479), true},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 0, 6, true},
					{image.Pt(639, 479), 643, 488, true},
				},
			},
			{
				// 642*480 = 308160 > 481*640 = 307840, so height-limited:
				// s = 481/480, origin.x = 320/960 = 1/3, origin.y = 0.
				name: "642x481 puts the origin on a third of a pixel",
				w:    642, h: 481, wantScale: 481.0 / 480.0,
				wProbes: []wProbe{
					{0, 0, image.Point{}, false}, // the frame starts at x = 1/3
					{1, 0, image.Pt(0, 0), true},
					{641, 480, image.Pt(639, 479), true},
				},
				fProbes: []fProbe{
					{image.Pt(0, 0), 1, 0, true},
					{image.Pt(639, 479), 641, 480, true},
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := frame.Fit(frameW, frameH, tc.w, tc.h)
				if !p.Valid() {
					t.Fatalf("Fit(%d,%d).Valid() = false, want true", tc.w, tc.h)
				}
				if p.Scale() != tc.wantScale {
					t.Fatalf("Fit(%d,%d).Scale() = %v, want exactly %v", tc.w, tc.h, p.Scale(), tc.wantScale)
				}
				for _, q := range tc.wProbes {
					got, ok := p.WindowToFrame(q.x, q.y)
					if ok != q.wantOK || (ok && got != q.want) {
						t.Fatalf("Fit(%d,%d).WindowToFrame(%d,%d) = %v,%v, want %v,%v",
							tc.w, tc.h, q.x, q.y, got, ok, q.want, q.wantOK)
					}
				}
				for _, q := range tc.fProbes {
					x, y, ok := p.FrameToWindow(q.pt)
					if ok != q.wantOK {
						t.Fatalf("Fit(%d,%d).FrameToWindow(%v) ok = %v, want %v", tc.w, tc.h, q.pt, ok, q.wantOK)
					}
					if ok && (x != q.x || y != q.y) {
						t.Fatalf("Fit(%d,%d).FrameToWindow(%v) = (%d,%d), want (%d,%d)",
							tc.w, tc.h, q.pt, x, y, q.x, q.y)
					}
				}
			})
		}
	})
}

func TestExpandedWidthKeepsHeightAndRemovesVerticalBands(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	tests := []struct {
		name       string
		minimum    int
		height     int
		windowW    int
		windowH    int
		want       int
		checkEdges bool
	}{
		{"town 4:3", 640, 480, 1600, 1200, 640, true},
		{"town 16:9", 640, 480, 1920, 1080, 854, true},
		{"town 2560x1440", 640, 480, 2560, 1440, 854, true},
		{"town ultrawide", 640, 480, 3440, 1440, 1147, true},
		{"mission 16:9", 1024, 768, 1920, 1080, 1366, true},
		{"tall window retains minimum", 640, 480, 800, 1200, 640, true},
		{"zero window retains minimum", 640, 480, 0, 0, 640, false},
		{"invalid policy", 0, 480, 1920, 1080, 0, false},
		{"overflow clamps", 640, 480, maxInt, 1, maxInt, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := frame.ExpandedWidth(tc.minimum, tc.height, tc.windowW, tc.windowH)
			if got != tc.want {
				t.Fatalf("ExpandedWidth(%d,%d,%d,%d) = %d, want %d",
					tc.minimum, tc.height, tc.windowW, tc.windowH, got, tc.want)
			}
			if !tc.checkEdges {
				return
			}
			p := frame.Fit(got, tc.height, tc.windowW, tc.windowH)
			if _, ok := p.WindowToFrame(0, tc.windowH/2); !ok {
				t.Fatal("left window edge remained a vertical band")
			}
			if _, ok := p.WindowToFrame(tc.windowW-1, tc.windowH/2); !ok {
				t.Fatal("right window edge remained a vertical band")
			}
		})
	}
}

// columnsIn returns the indices in [lo,a) plus those in [b,hi) -- the two
// letterbox bands on one axis.
func columnsIn(lo, a, b, hi int) []int {
	var out []int
	for i := lo; i < a; i++ {
		out = append(out, i)
	}
	for i := b; i < hi; i++ {
		out = append(out, i)
	}
	return out
}
