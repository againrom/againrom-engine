package terrain_test

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// blitLitAt is where the fixture frame is placed for every whole-destination
// comparison below: wholly inside the canvas, so a difference in coverage is a
// difference this oracle sees rather than one the clip hides.
var blitLitAt = image.Pt(4, 3)

// litRows are the three rows AC-5 names, and they are the three that matter: the
// brightest (gain 2.0, where the fixture saturates), the identity (gain 1.0) and
// the darkest (gain 0.125).
var litRows = []int{0, 8, 15}

// noTint is the sky tint every criterion here fixes at zero — the tint our own
// sun carries everywhere.
var noTint = [3]uint8{}

// blitLitDest returns a fresh background canvas with the fixture frame blitted
// onto it at blitLitAt, lit at row when lit is true and raw when it is false.
func blitLitDest(t *testing.T, f *terrain.StaticFrame, lit bool, tint [3]uint8, row int) *image.RGBA {
	t.Helper()
	dst := blitDest()
	if lit {
		terrain.BlitStaticLit(dst, f, blitLitAt.X, blitLitAt.Y, tint, row)
	} else {
		terrain.BlitStatic(dst, f, blitLitAt.X, blitLitAt.Y)
	}
	return dst
}

// TestLitBlitAtRowEightIsTheUnshadedBlit — AC-2, SC-2.
//
// Row 8 with a zero tint is the raw palette exactly, so the two destinations are
// byte-for-byte equal — not approximately, not on the painted rectangle alone,
// but over the whole canvas including every pixel neither blit touched. That is
// the assertion a one-pixel displacement or a single wrong palette entry fails,
// where a comparison of the painted rectangle would let a coverage change hide
// at the edge.
func TestLitBlitAtRowEightIsTheUnshadedBlit(t *testing.T) {
	f := blitTestFrame(t)

	lit := blitLitDest(t, f, true, noTint, 8)
	raw := blitLitDest(t, f, false, noTint, 0)

	if !bytes.Equal(lit.Pix, raw.Pix) {
		for i := range raw.Pix {
			if lit.Pix[i] != raw.Pix[i] {
				t.Fatalf("row 8 differs from the unshaded blit at byte %d (pixel %d,%d): %d vs %d — "+
					"gain 1.0 must reproduce the raw palette",
					i, (i/4)%blitDstW, (i/4)/blitDstW, lit.Pix[i], raw.Pix[i])
			}
		}
	}

	// And the fixture is not vacuously equal: some row must differ, or the
	// equality above would hold for a blit that ignored its row entirely.
	dark := blitLitDest(t, f, true, noTint, 15)
	if bytes.Equal(dark.Pix, raw.Pix) {
		t.Error("row 15 is byte-identical to the raw palette; the fixture does not discriminate a row")
	}
}

// TestLitBlitChangesColourAndNeverCoverage — AC-5, SC-4's first half.
//
// At each of the three rows, over the WHOLE destination: the pixels left
// untouched are exactly those the unshaded blit leaves, every painted pixel is
// fully opaque, and the palette entry carrying alpha 0 paints its SHADED colour
// rather than an invisible one. The untouched set is computed from the unshaded
// run rather than from the frame, so a lit blit that painted one extra pixel —
// or skipped one — fails here whatever colour it used.
func TestLitBlitChangesColourAndNeverCoverage(t *testing.T) {
	f := blitTestFrame(t)
	raw := blitLitDest(t, f, false, noTint, 0)
	before := blitDest()

	for _, row := range litRows {
		lit := blitLitDest(t, f, true, noTint, row)

		painted, alphaZeroSeen, bad := 0, 0, 0
		b := lit.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				sx, sy := x-blitLitAt.X, y-blitLitAt.Y
				covered := sx >= 0 && sy >= 0 && sx < f.Width && sy < f.Height &&
					f.Pixels[sy*f.Width+sx].Opaque
				got := lit.RGBAAt(x, y)

				if !covered {
					if got != before.RGBAAt(x, y) || got != raw.RGBAAt(x, y) {
						bad++
						if bad <= 5 {
							t.Errorf("row %d: pixel (%d,%d) = %v was written, but the unshaded blit leaves it at %v",
								row, x, y, got, before.RGBAAt(x, y))
						}
					}
					continue
				}

				painted++
				idx := f.Pixels[sy*f.Width+sx].Index
				want := terrain.SpriteRGBA(f.Palette[idx], noTint, row)
				if idx == blitAlphaZeroIndex {
					alphaZeroSeen++
					if f.Palette[idx].A != 0 {
						t.Fatalf("the fixture's alpha-0 entry carries alpha %d", f.Palette[idx].A)
					}
				}
				if got.A != 0xff {
					bad++
					if bad <= 5 {
						t.Errorf("row %d: painted pixel (%d,%d) has alpha %d, want 0xff", row, x, y, got.A)
					}
					continue
				}
				if got != want {
					bad++
					if bad <= 5 {
						t.Errorf("row %d: pixel (%d,%d) index %d = %v, want the shaded entry %v",
							row, x, y, idx, got, want)
					}
				}
			}
		}
		if bad > 5 {
			t.Errorf("row %d: %d pixels wrong in total", row, bad)
		}
		if painted == 0 || alphaZeroSeen == 0 {
			t.Fatalf("row %d: the fixture painted %d pixels and %d alpha-0 entries; both must be non-zero",
				row, painted, alphaZeroSeen)
		}
	}
}

func TestLitBlitLeavesTheFrameAlone(t *testing.T) {
	f := blitTestFrame(t)
	pixels := append([]terrain.StaticPixel(nil), f.Pixels...)
	palette := f.Palette

	for row := -2; row < 18; row++ {
		dst := blitDest()
		terrain.BlitStaticLit(dst, f, blitLitAt.X, blitLitAt.Y, [3]uint8{9, 40, 200}, row)
	}

	if f.Palette != palette {
		t.Error("a lit blit wrote the shaded colours back into the frame's own palette")
	}
	for i := range pixels {
		if f.Pixels[i] != pixels[i] {
			t.Fatalf("a lit blit changed frame pixel %d", i)
		}
	}
}

// TestRGBALitIsTheLitBlitOntoATransparentCanvas — AC-6, SC-5's raster
// half.
//
// One frame, one row, one tint, down both paths: the standalone image and
// the blit onto a transparent canvas of the same size must agree pixel for
// pixel, with transparent frame pixels reaching the image as transparency
// rather than as black at full alpha. This is what forbids the window's
// texture and the raster tool's pixels from diverging once shading enters.
//
// A non-zero tint is used here on purpose: it is the one parameter the raster
// and the window could disagree about by dropping it on one side, and no other
// criterion in this story carries one.
func TestRGBALitIsTheLitBlitOntoATransparentCanvas(t *testing.T) {
	f := blitTestFrame(t)
	tint := [3]uint8{3, 17, 44}

	for _, row := range litRows {
		canvas := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
		terrain.BlitStaticLit(canvas, f, 0, 0, tint, row)

		got := f.RGBALit(tint, row)
		if got.Bounds() != canvas.Bounds() {
			t.Fatalf("row %d: RGBALit bounds %v, want %v", row, got.Bounds(), canvas.Bounds())
		}
		if !bytes.Equal(got.Pix, canvas.Pix) {
			t.Fatalf("row %d: RGBALit is not the lit blit onto a transparent canvas", row)
		}

		holes := 0
		for y := 0; y < f.Height; y++ {
			for x := 0; x < f.Width; x++ {
				if f.Pixels[y*f.Width+x].Opaque {
					continue
				}
				holes++
				if c := got.RGBAAt(x, y); c != (color.RGBA{}) {
					t.Fatalf("row %d: hole (%d,%d) reached the image as %v, want fully transparent", row, x, y, c)
				}
			}
		}
		if holes == 0 {
			t.Fatal("the fixture frame carries no transparent pixel")
		}
	}

	if !bytes.Equal(f.RGBA().Pix, f.RGBALit(noTint, 8).Pix) {
		t.Error("RGBA() and RGBALit(0, row 8) disagree; row 8 is the raw palette")
	}
}

// TestRGBALitIsTotal — the standalone image's own refusals, at every row: a nil
// receiver, a zero dimension and an unwalkable pixel slice each yield an image
// rather than nil or a panic, exactly as RGBA does.
func TestRGBALitIsTotal(t *testing.T) {
	var nilFrame *terrain.StaticFrame
	for _, row := range []int{-1, 0, 8, 15, 99} {
		if got := nilFrame.RGBALit(noTint, row); got == nil || !got.Bounds().Empty() {
			t.Errorf("row %d: a nil frame gave %v, want an empty image", row, got)
		}
		zero := &terrain.StaticFrame{Width: 0, Height: 4}
		if got := zero.RGBALit(noTint, row); got == nil || !got.Bounds().Empty() {
			t.Errorf("row %d: a zero-width frame gave %v, want an empty image", row, got)
		}
		short := &terrain.StaticFrame{Width: 3, Height: 3, Pixels: make([]terrain.StaticPixel, 4)}
		got := short.RGBALit(noTint, row)
		if got.Bounds() != image.Rect(0, 0, 3, 3) {
			t.Errorf("row %d: a short frame gave bounds %v, want the size its header claims", row, got.Bounds())
		}
		if !bytes.Equal(got.Pix, make([]uint8, len(got.Pix))) {
			t.Errorf("row %d: a short frame drew pixels", row)
		}
	}
}

// TestLitBlitRefusesExactlyWhatTheUnshadedOneRefuses — AC-10, SC-4's
// second half.
//
// Each refusal on a case of its own, at three rows apiece, with the WHOLE
// destination compared before and after: nothing panics, nothing draws, and no
// partial row or partial frame is written. Then the two out-of-range rows, which
// are NOT refusals — they draw, at rows 0 and 15 — so the refusal set stays the
// four it was and gains no fifth.
func TestLitBlitRefusesExactlyWhatTheUnshadedOneRefuses(t *testing.T) {
	good := blitTestFrame(t)

	cases := []struct {
		name string
		dst  *image.RGBA
		f    *terrain.StaticFrame
	}{
		{"a nil destination", nil, good},
		{"a nil frame", blitDest(), nil},
		{"a zero-width frame", blitDest(), &terrain.StaticFrame{Width: 0, Height: 4, Palette: blitPalette()}},
		{"a zero-height frame", blitDest(), &terrain.StaticFrame{Width: 4, Height: 0, Palette: blitPalette()}},
		{"a negative dimension", blitDest(), &terrain.StaticFrame{Width: -3, Height: 4, Palette: blitPalette()}},
		{"a pixel slice shorter than the header claims", blitDest(),
			&terrain.StaticFrame{Width: 5, Height: 4, Pixels: make([]terrain.StaticPixel, 19), Palette: blitPalette()}},
	}

	for _, c := range cases {
		for _, row := range litRows {
			var before *image.RGBA
			if c.dst != nil {
				before = cloneRGBA(c.dst)
			}
			terrain.BlitStaticLit(c.dst, c.f, 1, 1, noTint, row)
			if c.dst != nil && !bytes.Equal(c.dst.Pix, before.Pix) {
				t.Errorf("%s at row %d: the destination was written", c.name, row)
			}
		}
	}

	// An out-of-range row is HELD, not refused: -1 draws row 0's pixels and 99
	// draws row 15's, over the whole destination.
	for _, c := range []struct{ given, held int }{{-1, 0}, {99, 15}} {
		out := blitLitDest(t, good, true, noTint, c.given)
		want := blitLitDest(t, good, true, noTint, c.held)
		if !bytes.Equal(out.Pix, want.Pix) {
			t.Errorf("row %d did not draw as row %d", c.given, c.held)
		}
		if bytes.Equal(out.Pix, blitDest().Pix) {
			t.Errorf("row %d drew nothing; an out-of-range row is held into range, not refused", c.given)
		}
	}
}
