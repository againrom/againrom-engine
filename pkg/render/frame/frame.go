// Package frame places a virtual frame of ANY size inside a window of any size:
// one uniform scale, centred, letterboxed, nothing cropped and no aspect
// distortion.
//
// THE FRAME SIZE IS A VALUE, NOT A PACKAGE CONSTANT (1026 B1). It holds two
// now: the town family composes at 640x480 and the mission screen at
// 1024x768. Every mapping below reads the size its own Placement was fitted
// with, so a Placement is self-describing and two frames of different sizes
// cannot come to share one mapping. ExpandedWidth lets callers retain a
// logical height while widening that frame to a modern window aspect before
// fitting it.
//
// The scaling rule is the project's own design, not a decoded game behaviour —
// the original ran at a fixed resolution and nothing about windowed presentation
// is established by the research. What *is* load-bearing is that the mapping be
// exact: a cursor position decides which brooch button the hit mask is sampled
// at, so a mapping that loses a pixel row makes a strip of a button unclickable.
//
// The scale is therefore kept as an exact rational num/den and every decision is
// taken in integer arithmetic; float64 appears only in Scale and Origin, which
// exist to build a draw transform and are never used to decide anything. Under
// that formulation the three properties the frame owes hold by construction
// rather than within a tolerance:
//
//   - the whole frame lies inside the window for any window at least 1x1;
//   - a window position maps to at most one frame pixel, and a position in the
//     letterbox maps to none;
//   - WindowToFrame after FrameToWindow is the identity wherever FrameToWindow
//     reports ok, and ok is false exactly for the frame pixels no window pixel
//     maps to (which happens only when the frame is scaled below 1:1).
//
// The package is stdlib-only and engine-free, so all of that is unit-testable
// without opening a window.
package frame

import (
	"image"
	"math/bits"
)

// W and H are the TOWN FAMILY's minimum/native frame size in frame pixels: the menu,
// the map picker, the chargen pages, the town screens and the in-game menu panel
// originate at this size. A wide window retains H and expands W through
// ExpandedWidth; native install art remains at its original size inside it.
//
// They are no longer "the" frame size. The mission screen composes at its own
// larger frame (pkg/ui's MissionFrameW and MissionFrameH), and nothing in this
// package reads these two: they are a caller's argument to Fit like any other.
const (
	W = 640
	H = 480
)

// ExpandedWidth keeps logicalHeight fixed and grows minWidth just enough to
// cover the window's aspect ratio. The ceiling is deliberate: the fitted frame
// may leave a subpixel horizontal band, but it never leaves a vertical band.
// Narrow, tall, or temporarily invalid windows retain the minimum width.
func ExpandedWidth(minWidth, logicalHeight, winW, winH int) int {
	if minWidth <= 0 || logicalHeight <= 0 {
		return 0
	}
	if winW <= 0 || winH <= 0 {
		return minWidth
	}

	hi, lo := bits.Mul64(uint64(winW), uint64(logicalHeight))
	den := uint64(winH)
	maxInt := uint64(^uint(0) >> 1)
	if hi >= den {
		return int(maxInt)
	}
	want, rem := bits.Div64(hi, lo, den)
	if rem != 0 {
		if want == ^uint64(0) {
			return int(maxInt)
		}
		want++
	}
	if want > maxInt {
		want = maxInt
	}
	if want < uint64(minWidth) {
		return minWidth
	}
	return int(want)
}

// Placement is one window's fitted frame: the frame size, the window size, and
// the scale held as the exact rational num/den.
//
// IT CARRIES ITS OWN FRAME SIZE (1026 B1). Every mapping is a function of that
// size, so a Placement fitted for the town's 640x480 and one fitted for the
// mission's 1024x768 each map into the frame it was built for and into no other.
//
// The zero Placement is not usable and reports Valid() == false; construct one
// with Fit. Keeping the scale rational rather than as a float64 is what makes
// WindowToFrame and FrameToWindow exact inverses on the frame pixels that have a
// window pixel at all.
type Placement struct {
	frameW, frameH int64
	winW, winH     int64
	num, den       int64 // scale = num/den, exactly; num == 0 means "not usable"
}

// Fit computes the placement of a frameW x frameH virtual frame in a winW x winH
// window.
//
// The scale is min(winW/frameW, winH/frameH) as an exact rational: whichever
// axis binds contributes its own window extent as the numerator and the frame
// extent as the denominator. A window OR A FRAME with a non-positive dimension
// yields an unusable placement rather than an error, so a caller need not
// special-case the moment before a window has a size; every mapping on it then
// refuses, and no division is ever executed.
func Fit(frameW, frameH, winW, winH int) Placement {
	if frameW <= 0 || frameH <= 0 || winW <= 0 || winH <= 0 {
		return Placement{}
	}
	fw, fh := int64(frameW), int64(frameH)
	w, h := int64(winW), int64(winH)
	if w*fh <= h*fw {
		return Placement{frameW: fw, frameH: fh, winW: w, winH: h, num: w, den: fw} // width-limited
	}
	return Placement{frameW: fw, frameH: fh, winW: w, winH: h, num: h, den: fh} // height-limited
}

// FitDown centres a native frame and shrinks it only when it would not fit.
func FitDown(frameW, frameH, winW, winH int) Placement {
	p := Fit(frameW, frameH, winW, winH)
	if p.num > p.den {
		p.num = p.den
	}
	return p
}

// FrameSize is the frame size this placement was fitted for, in frame pixels.
// An unusable placement reports the zero point.
func (p Placement) FrameSize() image.Point {
	if !p.Valid() {
		return image.Point{}
	}
	return image.Point{X: int(p.frameW), Y: int(p.frameH)}
}

// WindowSize is the window size this placement was fitted into, in window
// pixels. An unusable placement reports the zero point.
//
// It exists so a caller that must decide something about the WINDOW's own edge
// -- the mission viewer's edge-scroll band is the one -- reads the size from
// the placement it already holds instead of keeping a second copy that nothing
// makes agree with this one.
func (p Placement) WindowSize() image.Point {
	if !p.Valid() {
		return image.Point{}
	}
	return image.Point{X: int(p.winW), Y: int(p.winH)}
}

// Valid reports whether the placement came from a window with a positive size.
// Every mapping on an invalid placement reports "no frame pixel".
func (p Placement) Valid() bool { return p.num > 0 && p.den > 0 }

// Scale is the uniform scale factor, for building a draw transform only. No
// decision in this package is taken on it.
func (p Placement) Scale() float64 {
	if !p.Valid() {
		return 0
	}
	return float64(p.num) / float64(p.den)
}

// Origin is the top-left corner of the scaled frame within the window, in window
// pixels, for building a draw transform only.
//
// It is non-negative and origin+scale*frame never exceeds the window, both
// exactly: each reduces to frameW*num <= winW*den (and frameH*num <= winH*den),
// which is what Fit selects the binding axis for.
//
// The numerator and denominator are formed in integers and divided once, rather
// than derived from the rounded Scale(). That is not fussiness: (win - f*Scale())/2
// rounds the scale first and then subtracts two nearly equal quantities, which
// drives the result a few ulp negative on a large family of window sizes (700x500
// and 642x481 among them) and breaks the non-negativity above. One rounding, on
// the final quotient, cannot.
func (p Placement) Origin() (x, y float64) {
	if !p.Valid() {
		return 0, 0
	}
	return float64(p.winW*p.den-p.frameW*p.num) / float64(2*p.den),
		float64(p.winH*p.den-p.frameH*p.num) / float64(2*p.den)
}

// WindowToFrame maps a window position to the frame pixel under it, reporting
// false when the position lies in the letterbox, outside the window, or on an
// unusable placement.
//
// Frame pixel f owns the half-open window interval [ox + f*s, ox + (f+1)*s),
// where ox = (win*den - frame*num) / (2*den) and s = num/den. Solving
// f = floor((x - ox)/s) and multiplying through by 2*den*num clears every
// fraction, leaving one floored integer division per axis.
func (p Placement) WindowToFrame(x, y int) (image.Point, bool) {
	pt, ok := p.WindowToFrameExtended(x, y)
	if !ok || pt.X < 0 || pt.X >= int(p.frameW) || pt.Y < 0 || pt.Y >= int(p.frameH) {
		return image.Point{}, false
	}
	return pt, true
}

// WindowToFrameExtended maps a window position to a frame position, continuing
// the frame's own pixel lattice OUTSIDE the frame instead of reporting a miss.
// It reports false only for an unusable placement.
//
// IT EXISTS FOR A CONTINUOUS CURSOR RATHER THAN FOR A BUTTON (1026 B4). A hit
// test wants WindowToFrame's miss: a press in the letterbox belongs to no
// control. A cursor does not. The mission screen's edge-scroll asks whether the
// cursor is inside the view, and its drag asks how far the cursor moved; both
// need a position for a cursor that has left the frame. Folding those onto the
// frame's edge would edge-scroll forever off the left border and would eat a
// drag's travel outside the picture. Returning the extended lattice lets the
// caller's own "is it inside" test answer as it always did.
//
// The returned coordinate is exact in the same sense WindowToFrame's is: it is
// the floor of (x - origin) / scale evaluated in integer arithmetic, so it agrees
// with WindowToFrame wherever WindowToFrame answers at all.
func (p Placement) WindowToFrameExtended(x, y int) (image.Point, bool) {
	if !p.Valid() {
		return image.Point{}, false
	}
	fx := floorDiv(2*p.den*int64(x)-p.den*p.winW+p.frameW*p.num, 2*p.num)
	fy := floorDiv(2*p.den*int64(y)-p.den*p.winH+p.frameH*p.num, 2*p.num)
	return image.Point{X: int(fx), Y: int(fy)}, true
}

// FrameToWindow maps a frame pixel to the first window position that maps back
// to it, reporting false when no window position does.
//
// That happens only below 1:1, where a frame pixel's window interval can be
// shorter than one pixel and contain no integer. Reporting it rather than
// returning a nearby position keeps the pair exact inverses: ok is precisely
// membership in WindowToFrame's image, so a caller never gets a coordinate that
// silently belongs to a different frame pixel.
func (p Placement) FrameToWindow(pt image.Point) (x, y int, ok bool) {
	if !p.Valid() || pt.X < 0 || pt.X >= int(p.frameW) || pt.Y < 0 || pt.Y >= int(p.frameH) {
		return 0, 0, false
	}
	fx, fy := int64(pt.X), int64(pt.Y)

	// The interval's lower bound, rounded up to the first integer at or above it.
	wx := ceilDiv(p.winW*p.den-p.frameW*p.num+2*fx*p.num, 2*p.den)
	wy := ceilDiv(p.winH*p.den-p.frameH*p.num+2*fy*p.num, 2*p.den)

	// ...which is inside the interval only if it is strictly below the upper
	// bound. Both sides are multiplied by 2*den so the comparison stays exact.
	if 2*p.den*wx >= p.winW*p.den-p.frameW*p.num+2*(fx+1)*p.num {
		return 0, 0, false
	}
	if 2*p.den*wy >= p.winH*p.den-p.frameH*p.num+2*(fy+1)*p.num {
		return 0, 0, false
	}
	return int(wx), int(wy), true
}

// floorDiv divides rounding towards negative infinity. Go's / truncates towards
// zero, which would fold the window positions just left of the frame onto frame
// column 0 instead of rejecting them.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// ceilDiv divides rounding towards positive infinity.
func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) == (b < 0) {
		q++
	}
	return q
}
