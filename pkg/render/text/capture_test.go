package text

import (
	"image"
	"testing"
)

// TestCaptureOffRecordsNothing is the default-path guarantee the
// presentation-layer overlay (DIV-1385) depends on: a Draw call outside any
// capture window neither records nor changes what it paints.
func TestCaptureOffRecordsNothing(t *testing.T) {
	ResetCapture()
	f := levelFont()
	dst := fill(image.Rect(0, 0, 8, 4), backColor)
	f.Draw(dst, "A", 0, 0, textColor)
	if n := CapturedLen(); n != 0 {
		t.Fatalf("CapturedLen() = %d outside any capture window, want 0", n)
	}
	if got := dst.RGBAAt(0, 0); got != textColor {
		t.Fatalf("Draw outside capture painted %v, want the level-15 colour %v", got, textColor)
	}
}

// TestCaptureRecordsPositionAndColour: within a capture window, every glyph
// Draw places is recorded with the same destination position and caller
// colour Draw itself used to paint it — the overlay's own contract, that
// what it resamples is exactly what Font.Draw would have painted.
func TestCaptureRecordsPositionAndColour(t *testing.T) {
	ResetCapture()
	SetCapture(false)
	f := levelFont()
	dst := fill(image.Rect(0, 0, 32, 4), backColor)
	f.Draw(dst, "AA", 3, 1, textColor)
	StopCapture()

	calls := Captured()
	if len(calls) != 2 {
		t.Fatalf("Captured() len = %d, want 2 glyphs for \"AA\"", len(calls))
	}
	aGlyph := f.GlyphFor('A')
	step := f.advance(f.index('A'), aGlyph)
	for i, want := range []struct{ x, y int }{{3, 1}, {3 + step, 1}} {
		if calls[i].X != want.x || calls[i].Y != want.y {
			t.Fatalf("call %d position = (%d,%d), want (%d,%d)", i, calls[i].X, calls[i].Y, want.x, want.y)
		}
		if calls[i].Color != textColor {
			t.Fatalf("call %d colour = %v, want %v", i, calls[i].Color, textColor)
		}
		if calls[i].Glyph != aGlyph {
			t.Fatalf("call %d glyph pointer = %p, want the font's own record %p", i, calls[i].Glyph, aGlyph)
		}
	}

	// Rasterisation still happened: skip was not requested.
	if got := dst.RGBAAt(3, 1); got != textColor {
		t.Fatalf("Draw with SetCapture(false) painted %v at (3,1), want %v", got, textColor)
	}
}

// TestCaptureSkipRasterRecordsWithoutPainting: SetCapture(true) is the
// recorder/deferral hook the presentation layer opens around a composition
// it means to smooth — Draw still records every glyph's exact placement and
// colour, but paints nothing, so the CPU buffer comes back with the text
// removed and every other pixel unchanged.
func TestCaptureSkipRasterRecordsWithoutPainting(t *testing.T) {
	ResetCapture()
	SetCapture(true)
	f := levelFont()
	dst := fill(image.Rect(0, 0, 8, 4), backColor)
	f.Draw(dst, "A", 0, 0, textColor)
	StopCapture()

	if n := CapturedLen(); n != 1 {
		t.Fatalf("CapturedLen() = %d, want 1", n)
	}
	if calls := Captured(); calls[0].X != 0 || calls[0].Y != 0 || calls[0].Color != textColor {
		t.Fatalf("captured call = %+v, want X=0 Y=0 Color=%v", calls[0], textColor)
	}
	for x := 0; x < 4; x++ {
		for y := 0; y < 2; y++ {
			if got := dst.RGBAAt(x, y); got != backColor {
				t.Fatalf("pixel (%d,%d) = %v under skip-raster, want the untouched background %v", x, y, got, backColor)
			}
		}
	}
}

// TestResetCaptureClearsAcrossWindows: ResetCapture is the once-per-frame
// clear; SetCapture/StopCapture toggling within a frame does not lose an
// earlier window's own glyphs, which is what lets two independently
// composed sub-pictures land in one ordered list (capture.go's own
// documented contract).
func TestResetCaptureClearsAcrossWindows(t *testing.T) {
	ResetCapture()
	f := levelFont()
	dst := fill(image.Rect(0, 0, 32, 4), backColor)

	SetCapture(true)
	f.Draw(dst, "A", 0, 0, textColor)
	StopCapture()
	if n := CapturedLen(); n != 1 {
		t.Fatalf("after first window CapturedLen() = %d, want 1", n)
	}

	SetCapture(true)
	f.Draw(dst, "A", 10, 0, textColor)
	StopCapture()
	calls := Captured()
	if len(calls) != 2 {
		t.Fatalf("after second window CapturedLen() = %d, want 2 (both windows kept)", len(calls))
	}
	if calls[0].X != 0 || calls[1].X != 10 {
		t.Fatalf("captured order/positions = %+v, want X=0 then X=10", calls)
	}

	ResetCapture()
	if n := CapturedLen(); n != 0 {
		t.Fatalf("ResetCapture left CapturedLen() = %d, want 0", n)
	}
	if capturing || skipRaster {
		t.Fatalf("ResetCapture left capturing=%v skipRaster=%v, want both false", capturing, skipRaster)
	}
}

// TestShiftCapturedMovesOnlyFromStart mirrors the offset-paste sites this
// mechanism exists for: a caller composes a sub-picture (recording start :=
// CapturedLen() first), then shifts only the glyphs recorded from that index
// onward by the same offset it pastes the picture's own pixels at, leaving
// every earlier entry untouched.
func TestShiftCapturedMovesOnlyFromStart(t *testing.T) {
	ResetCapture()
	SetCapture(true)
	f := levelFont()
	dst := fill(image.Rect(0, 0, 32, 4), backColor)

	f.Draw(dst, "A", 0, 0, textColor) // outer-window glyph, must not move
	start := CapturedLen()
	f.Draw(dst, "A", 5, 5, textColor) // sub-picture's own local coordinates
	ShiftCaptured(start, 100, 200)
	StopCapture()

	calls := Captured()
	if calls[0].X != 0 || calls[0].Y != 0 {
		t.Fatalf("entry before start moved: %+v", calls[0])
	}
	if calls[1].X != 105 || calls[1].Y != 205 {
		t.Fatalf("entry from start = %+v, want X=105 Y=205", calls[1])
	}
}

// A negative start clamps to 0 rather than panicking or silently skipping
// the shift, matching ShiftCaptured's own documented clamp.
func TestShiftCapturedClampsNegativeStart(t *testing.T) {
	ResetCapture()
	SetCapture(true)
	f := levelFont()
	dst := fill(image.Rect(0, 0, 8, 4), backColor)
	f.Draw(dst, "A", 0, 0, textColor)
	StopCapture()

	ShiftCaptured(-5, 1, 1)
	if calls := Captured(); calls[0].X != 1 || calls[0].Y != 1 {
		t.Fatalf("negative start = %+v, want X=1 Y=1 (clamped to 0)", calls[0])
	}
}

// TestRecordKeepsItsOwnGlyphsAndTheOuterWindow: a cached picture's composer
// records its glyphs, with the cells they replaced, without disturbing the
// frame's open window, and Append re-captures them only while one is open.
func TestRecordKeepsItsOwnGlyphsAndTheOuterWindow(t *testing.T) {
	ResetCapture()
	SetCapture(false)
	f := levelFont()
	f.Draw(fill(image.Rect(0, 0, 8, 4), backColor), "A", 0, 0, textColor)
	cached := fill(image.Rect(0, 0, 32, 4), backColor)
	own := Record(func() { f.Draw(cached, "AA", 0, 0, textColor) })
	if len(own) != 2 || CapturedLen() != 1 || !Capturing() {
		t.Fatalf("Record returned %d calls, outer window holds %d, capturing %v; want 2, 1, true", len(own), CapturedLen(), Capturing())
	}
	if own[0].Under == nil || own[0].Under[0] != backColor {
		t.Fatalf("recorded Under = %v, want the background %v under the painted cell", own[0].Under, backColor)
	}
	if got := cached.RGBAAt(0, 0); got != textColor {
		t.Fatalf("Record painted %v, want the rasterised colour %v", got, textColor)
	}
	Append(own, 5, 1)
	if CapturedLen() != 3 || Captured()[1].X != own[0].X+5 || Captured()[1].Y != own[0].Y+1 {
		t.Fatalf("Append left %d calls, first at (%d,%d); want 3 shifted by (5,1)", CapturedLen(), Captured()[1].X, Captured()[1].Y)
	}
	StopCapture()
	Append(own, 0, 0)
	if CapturedLen() != 3 {
		t.Fatalf("Append outside a window grew the capture to %d", CapturedLen())
	}
	ResetCapture()
}
