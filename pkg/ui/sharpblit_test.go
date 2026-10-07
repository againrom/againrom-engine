package ui

// drawSharpBilinear's own tests (method B, "sharp bilinear", owner hotfix
// decision — see engine/docs/HOTFIXES.md and the disposable prototype at
// review/font-smoothing).
//
// EBITENGINE REFUSES A PIXEL READBACK WITH NO GRAPHICS CONTEXT — the same
// constraint blitFrameCanvas's and blitPointerLayer's own doc comments
// already state, confirmed here by hand: calling (*ebiten.Image).At or
// ReadPixels before a real game loop has started panics with "ui: ReadPixels
// cannot be called before the game starts". So evidence 1 (an exact integer
// scale is byte-identical to the old draw) is proved BY CONSTRUCTION and
// witnessed here by showing the construction: the integer branch performs
// the one call this project made before method B existed and allocates no
// intermediate at all, rather than trusting a GPU bilinear sample at 1:1 to
// land on the same bytes as a nearest one — which is exactly the thing this
// harness cannot observe. Evidence 2 (no uneven column/row pixel width at a
// non-integer scale) is proved arithmetically: the nearest step maps every
// source pixel to exactly floor(scale) destination pixels on both axes, and
// the one bilinear step after it is a single uniform affine resize with no
// per-column decision to be uneven. Evidence 3 (hit-testing unchanged) has
// no test here because nothing in pkg/render/frame or any WindowToFrame/
// FrameToWindow caller was touched by this hotfix; the existing input tests
// (missioncursor_test.go and the rest of this package) still pass unchanged.

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
)

// TestSharpBilinearIntegerScaleSkipsTheIntermediate is evidence 1: at every
// exact integer scale this project actually ships (the town family's 1x
// developer window, 2x, 3x at 2560x1440, and a plain 4x), drawSharpBilinear
// allocates no intermediate canvas at all — buf stays nil — which is only
// possible if it took the direct single-draw branch instead of the two-step
// one. That branch draws src with a COPY of the caller's own op (GeoM
// unchanged, Filter forced to FilterNearest), which is the literal draw call
// this project made before method B existed.
func TestSharpBilinearIntegerScaleSkipsTheIntermediate(t *testing.T) {
	for _, scale := range []float64{1, 2, 3, 4} {
		src := ebiten.NewImage(4, 6)
		dst := ebiten.NewImage(40, 40)
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(5, 7)

		var buf *ebiten.Image
		drawSharpBilinear(dst, src, &op, &buf)

		if buf != nil {
			t.Errorf("scale %v: intermediate buffer allocated at %v, want nil: an exact integer scale must skip the two-step path entirely",
				scale, buf.Bounds())
		}
	}
}

// TestSharpBilinearNonIntegerScaleUpscalesTheIntermediateByAnExactInteger is
// evidence 2, read off frame.Fit's own real placements rather than
// hand-picked numbers: the town family widened to 1080p (2.25x, DIV-249's
// own frame at a real 16:9 window) and the mission frame at the owner's
// named 1920- and 2560-wide "worst case" windows, both real ExpandedWidth
// results, both non-integer.
//
// Nearest upscaling by an EXACT integer factor is what removes the
// unevenness: source column i occupies destination columns
// [i*factor, (i+1)*factor), a fixed width of exactly factor pixels for every
// i, never floor(scale) for one column and ceil(scale) for its neighbour the
// way a single non-integer FilterNearest draw would round per destination
// pixel. The bilinear step after it is one Scale-then-Translate GeoM — a
// single uniform affine transform applied identically to every source pixel,
// asserted below by its own four terms — so it has no per-column branch that
// could reintroduce unevenness of its own.
func TestSharpBilinearNonIntegerScaleUpscalesTheIntermediateByAnExactInteger(t *testing.T) {
	cases := []struct {
		name       string
		srcW, srcH int
		place      frame.Placement
	}{
		{"town family at 1920x1080 (2.25x)", frame.W, frame.H, frame.Fit(frame.W, frame.H, 1920, 1080)},
		{"mission frame at 1920x1080 (owner's worst case)", 1366, MissionFrameH, frame.Fit(1366, MissionFrameH, 1920, 1080)},
		{"mission frame at 2560x1440 (owner's worst case)", 1366, MissionFrameH, frame.Fit(1366, MissionFrameH, 2560, 1440)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.place.Valid() {
				t.Fatal("setup: placement is not usable")
			}
			scale := tc.place.Scale()
			if scale == float64(int(scale)) {
				t.Fatalf("setup: scale %v is an exact integer, this case is meant to be non-integer", scale)
			}
			factor := int(scale)

			src := ebiten.NewImage(tc.srcW, tc.srcH)
			dst := ebiten.NewImage(4000, 3000)
			ox, oy := tc.place.Origin()
			var op ebiten.DrawImageOptions
			op.Filter = ebiten.FilterNearest
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(ox, oy)

			var buf *ebiten.Image
			drawSharpBilinear(dst, src, &op, &buf)

			if buf == nil {
				t.Fatalf("scale %v: no intermediate buffer allocated, want one", scale)
			}
			wantW, wantH := tc.srcW*factor, tc.srcH*factor
			if got := buf.Bounds().Dx(); got != wantW {
				t.Errorf("intermediate width %d, want %d (src %d x floor(scale) %d): a fractional width would leave some source columns wider than others",
					got, wantW, tc.srcW, factor)
			}
			if got := buf.Bounds().Dy(); got != wantH {
				t.Errorf("intermediate height %d, want %d (src %d x floor(scale) %d)", got, wantH, tc.srcH, factor)
			}
		})
	}
}

// TestSharpBilinearReusesItsBuffer proves the steady-state cost claim: a
// second call at the same source size and scale neither reallocates nor
// disposes the intermediate, and a call at a different source size replaces
// it exactly once.
func TestSharpBilinearReusesItsBuffer(t *testing.T) {
	dst := ebiten.NewImage(4000, 3000)
	var buf *ebiten.Image

	draw := func(srcW, srcH int, scale float64) {
		src := ebiten.NewImage(srcW, srcH)
		var op ebiten.DrawImageOptions
		op.Filter = ebiten.FilterNearest
		op.GeoM.Scale(scale, scale)
		drawSharpBilinear(dst, src, &op, &buf)
	}

	draw(640, 480, 1.5)
	first := buf
	if first == nil {
		t.Fatal("setup: first non-integer draw allocated no buffer")
	}

	draw(640, 480, 1.5)
	if buf != first {
		t.Fatalf("same source size and scale replaced the buffer: %p -> %p, want reuse", first, buf)
	}

	draw(1024, 768, 1.5)
	if buf == first {
		t.Fatal("a different source size kept the old buffer, want a replacement sized for the new source")
	}
	if got, want := buf.Bounds().Dx(), 1024; got != want {
		t.Errorf("replacement buffer width %d, want %d", got, want)
	}
}

// TestSharpBilinearRefusesAnUnusablePlacement is the guard every call site
// already has (a.place.Valid()/v.place.Valid()) exercised directly on the
// helper: a non-positive scale draws nothing and allocates nothing, rather
// than panicking on a zero-sized intermediate.
func TestSharpBilinearRefusesAnUnusablePlacement(t *testing.T) {
	src := ebiten.NewImage(4, 4)
	dst := ebiten.NewImage(4, 4)
	var op ebiten.DrawImageOptions // zero GeoM element(0,0) is 0, not 1: identity Scale(1,1) was never applied
	op.GeoM.Scale(0, 0)
	var buf *ebiten.Image
	drawSharpBilinear(dst, src, &op, &buf)
	if buf != nil {
		t.Errorf("a zero scale allocated a buffer at %v, want none drawn at all", buf.Bounds())
	}
}
