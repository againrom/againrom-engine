package text

import (
	"image"
	"image/color"
	"testing"
)

// A composer that paints its text on a transparent layer and lays the layer
// over its background records the empty layer under each glyph. Underlay
// replaces the recorded cells, from the first glyph of the window it names,
// with the background's own, so the overlay puts back the cell the page shows.
func TestUnderlayReplacesTheRecordedCellsWithTheBackdrop(t *testing.T) {
	f := levelFont()
	layer := image.NewRGBA(image.Rect(0, 0, 32, 4))
	backdrop := fill(layer.Bounds(), backColor)
	ResetCapture()
	t.Cleanup(ResetCapture)
	SetCapture(false)
	f.Draw(layer, "A", 0, 0, textColor)
	start := CapturedLen()
	f.Draw(layer, "AA", 8, 1, textColor)
	Underlay(start, backdrop)
	StopCapture()

	calls := Captured()
	if len(calls) != 3 {
		t.Fatalf("captured %d glyphs, want 3", len(calls))
	}
	for n, p := range calls[0].Glyph.Pixels {
		if p.Painted && calls[0].Under[n] != (color.RGBA{}) {
			t.Errorf("a glyph before start had its cell %d replaced with %v", n, calls[0].Under[n])
		}
	}
	for i, c := range calls[1:] {
		for n, p := range c.Glyph.Pixels {
			want := color.RGBA{}
			if p.Painted {
				want = backColor
			}
			if c.Under[n] != want {
				t.Errorf("glyph %d cell %d holds %v under it, want %v", i+1, n, c.Under[n], want)
			}
		}
	}
}

// A cell outside the backdrop is left as recorded, and nothing happens outside
// a capture window or without a backdrop.
func TestUnderlayLeavesCellsOutsideTheBackdropAndOutsideAWindow(t *testing.T) {
	f := levelFont()
	layer := image.NewRGBA(image.Rect(0, 0, 32, 4))
	ResetCapture()
	t.Cleanup(ResetCapture)
	SetCapture(false)
	f.Draw(layer, "A", 0, 0, textColor)
	Underlay(0, fill(image.Rect(2, 0, 32, 4), backColor))
	c := Captured()[0]
	for n, p := range c.Glyph.Pixels {
		if !p.Painted {
			continue
		}
		x := n % c.Glyph.Width
		want := color.RGBA{}
		if x >= 2 {
			want = backColor
		}
		if c.Under[n] != want {
			t.Errorf("cell %d (column %d) holds %v, want %v", n, x, c.Under[n], want)
		}
	}
	Underlay(0, nil)
	StopCapture()
	Underlay(0, fill(layer.Bounds(), color.RGBA{R: 1, A: 255}))
	for n, p := range Captured()[0].Glyph.Pixels {
		if p.Painted && Captured()[0].Under[n] == (color.RGBA{R: 1, A: 255}) {
			t.Fatalf("Underlay outside a window changed cell %d", n)
		}
	}
}

// A glyph drawn into a sub-image is clipped to it, and shifting or appending a
// glyph moves that clip with it: the overlay cuts a glyph at the clip's edge
// wherever the glyph is finally pasted.
func TestClipTravelsWithAShiftedOrAppendedGlyph(t *testing.T) {
	f := levelFont()
	dst := fill(image.Rect(0, 0, 32, 8), backColor)
	sub := dst.SubImage(image.Rect(4, 2, 12, 6)).(*image.RGBA)
	ResetCapture()
	t.Cleanup(ResetCapture)
	SetCapture(false)
	f.Draw(sub, "A", 10, 3, textColor)
	if got := Captured()[0].Clip; got != sub.Bounds() {
		t.Fatalf("a glyph drawn into a sub-image recorded clip %v, want %v", got, sub.Bounds())
	}
	ShiftCaptured(0, 100, 50)
	if got, want := Captured()[0].Clip, sub.Bounds().Add(image.Pt(100, 50)); got != want {
		t.Fatalf("ShiftCaptured left the clip at %v, want %v", got, want)
	}
	StopCapture()

	own := Record(func() { f.Draw(sub, "A", 10, 3, textColor) })
	SetCapture(false)
	Append(own, -4, -2)
	if got, want := Captured()[1].Clip, sub.Bounds().Add(image.Pt(-4, -2)); got != want {
		t.Fatalf("Append left the clip at %v, want %v", got, want)
	}
	StopCapture()
}

// A glyph drawn into an image with no area records nothing, as it paints
// nothing.
func TestDrawIntoAnEmptyImageRecordsNothing(t *testing.T) {
	f := levelFont()
	ResetCapture()
	t.Cleanup(ResetCapture)
	SetCapture(false)
	f.Draw(image.NewRGBA(image.Rectangle{}), "AA", 0, 0, textColor)
	f.DrawFlat(image.NewRGBA(image.Rectangle{}), "AA", 0, 0, textColor)
	StopCapture()
	if n := CapturedLen(); n != 0 {
		t.Fatalf("drawing into an empty image recorded %d glyphs", n)
	}
}

// MarkErased flags every call it is given and answers the same slice.
func TestMarkErasedFlagsEveryCall(t *testing.T) {
	calls := []DrawCall{{X: 1}, {X: 2}, {X: 3, Erased: true}}
	got := MarkErased(calls)
	if len(got) != 3 || &got[0] != &calls[0] {
		t.Fatalf("MarkErased answered a different slice")
	}
	for i, c := range got {
		if !c.Erased {
			t.Errorf("call %d was not marked erased", i)
		}
	}
	if len(MarkErased(nil)) != 0 {
		t.Fatal("MarkErased(nil) answered calls")
	}
}

// An audit counts the glyphs a window recorded and the glyphs painted with no
// window open, naming the caller of each unrecorded one, and skips a glyph that
// paints no cell.
func TestAuditCountsRecordedAndUnrecordedGlyphs(t *testing.T) {
	f := levelFont()
	dst := fill(image.Rect(0, 0, 32, 4), backColor)
	ResetCapture()
	t.Cleanup(ResetCapture)
	if got := StopAudit(); got.Captured != 0 || got.Uncaptured != 0 {
		t.Fatalf("StopAudit with none started answered %+v", got)
	}
	StartAudit()
	SetCapture(false)
	f.Draw(dst, "AA", 0, 0, textColor)
	f.DrawFlat(dst, "A", 12, 0, textColor)
	StopCapture()
	f.Draw(dst, "A A", 16, 0, textColor)
	f.DrawFlat(dst, "A", 28, 0, textColor)
	got := StopAudit()
	if got.Captured != 3 {
		t.Errorf("the audit counted %d recorded glyphs, want 3", got.Captured)
	}
	if got.Uncaptured != 3 {
		t.Errorf("the audit counted %d unrecorded glyphs, want 3 (the blank between the two drawn is not one)", got.Uncaptured)
	}
	sites := 0
	for _, n := range got.Sites {
		sites += n
	}
	if sites != got.Uncaptured {
		t.Errorf("sites count %d glyphs, want %d", sites, got.Uncaptured)
	}
	f.Draw(dst, "A", 0, 0, textColor)
	if got = StopAudit(); got.Captured != 0 || got.Uncaptured != 0 {
		t.Errorf("a glyph drawn after StopAudit was counted: %+v", got)
	}
}
