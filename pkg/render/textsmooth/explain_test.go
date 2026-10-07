package textsmooth

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

var (
	explainBack  = color.RGBA{R: 10, G: 20, B: 30, A: 255}
	explainInk   = color.RGBA{R: 200, G: 180, B: 120, A: 255}
	explainDark  = color.RGBA{R: 5, G: 5, B: 5, A: 255}
	explainCover = color.RGBA{R: 1, G: 2, B: 3, A: 255}
)

// barFont's glyphs are 4x3 cells with the middle row painted and the rows above
// and below it empty, so a glyph has painted cells and unpainted neighbours.
func barFont() *text.Font {
	f := &text.Font{Glyphs: make([]text.Glyph, 224)}
	for i := range f.Glyphs {
		g := text.Glyph{Width: 4, Height: 3, Advance: 5, Pixels: make([]text.Pixel, 12)}
		for x := 0; x < 4; x++ {
			g.Pixels[4+x] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[i] = g
	}
	return f
}

func explainFrame(w, h int) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(frame.Pix); i += 4 {
		copy(frame.Pix[i:], []uint8{explainBack.R, explainBack.G, explainBack.B, explainBack.A})
	}
	return frame
}

// frameOracle answers Explain from the frame itself.
func frameOracle(frame *image.RGBA) Oracle {
	return Oracle{Cell: func(x, y int, want color.RGBA) Verdict {
		if frame.RGBAAt(x, y) == want {
			return Matches
		}
		return Differs
	}}
}

func explainFates(out []Outcome) []Fate {
	fates := make([]Fate, len(out))
	for i, o := range out {
		fates[i] = o.Fate
	}
	return fates
}

func wantFates(t *testing.T, out []Outcome, want ...Fate) {
	t.Helper()
	got := explainFates(out)
	if len(got) != len(want) {
		t.Fatalf("fates %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("fates %v, want %v", got, want)
		}
	}
}

func coverCell(frame *image.RGBA, x, y int) { frame.SetRGBA(x, y, explainCover) }

// A glyph the frame shows is kept whole, and the same glyph drawn again at the
// same place in the same colour is a repeat.
func TestExplainKeepsAShownGlyphAndNamesARepeat(t *testing.T) {
	frame := explainFrame(12, 3)
	font := barFont()
	calls := text.Record(func() {
		font.Draw(frame, "A", 0, 0, explainInk)
		font.Draw(frame, "A", 0, 0, explainInk)
	})
	out, certain := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Kept, Repeat)
	if !certain || out[0].Call.Mask != nil {
		t.Fatalf("certain %v, mask %v; want a certain frame and no mask on a whole glyph", certain, out[0].Call.Mask)
	}
}

// A glyph that paints nothing is blank, and a call that holds no record of the
// cells it replaced cannot be put back and is unrecorded.
func TestExplainNamesBlankAndUnrecordedCalls(t *testing.T) {
	frame := explainFrame(12, 3)
	empty := &text.Glyph{Width: 2, Height: 2, Pixels: make([]text.Pixel, 4)}
	inked := &barFont().Glyphs[0]
	calls := []text.DrawCall{
		{Glyph: empty, Color: explainInk, Under: make([]color.RGBA, 4)},
		{Glyph: nil, Color: explainInk},
		{Glyph: inked, X: 0, Color: explainInk},
	}
	out, _ := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Blank, Unrecorded, Unrecorded)
}

// A glyph a later picture covers wholly, or that lies wholly outside the
// frame, is hidden and left to the picture.
func TestExplainHidesAGlyphNoCellOfWhichShows(t *testing.T) {
	frame := explainFrame(12, 3)
	calls := text.Record(func() {
		barFont().Draw(frame, "A", 0, 0, explainInk)
		barFont().Draw(frame, "A", 20, 0, explainInk)
	})
	for x := 0; x < 4; x++ {
		coverCell(frame, x, 1)
	}
	out, certain := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Hidden, Hidden)
	if !certain {
		t.Fatal("a covered glyph left the frame uncertain")
	}
}

// A glyph a later picture covers in part is kept with a mask: the cells that
// show are drawn, the covered cells are not, and an empty cell beside a covered
// one is not, so the resampled edge of the glyph does not bleed onto the
// picture that covers it.
func TestExplainKeepsAPartlyCoveredGlyphWithAMask(t *testing.T) {
	frame := explainFrame(12, 3)
	calls := text.Record(func() { barFont().Draw(frame, "A", 0, 0, explainInk) })
	coverCell(frame, 2, 1)
	coverCell(frame, 3, 1)
	out, certain := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Kept)
	want := []bool{
		true, false, false, false,
		true, true, false, false,
		true, false, false, false,
	}
	got := out[0].Call.Mask
	if !certain || len(got) != len(want) {
		t.Fatalf("certain %v, mask %v; want a certain frame and a mask of %d cells", certain, got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mask %v, want %v", got, want)
		}
	}
}

// A cell the oracle cannot tell, on a glyph a later picture covers in part, is
// left out of the mask; only a glyph nothing is known about waits for the frame.
func TestExplainLeavesAnUnknownCellOutOfTheMaskOfACoveredGlyph(t *testing.T) {
	frame := explainFrame(12, 3)
	calls := text.Record(func() { barFont().Draw(frame, "A", 0, 0, explainInk) })
	o := Oracle{Cell: func(x, y int, want color.RGBA) Verdict {
		switch {
		case x < 2:
			return Matches
		case x == 2:
			return Differs
		}
		return Unknown
	}}
	out, certain := Explain(calls, frame.Bounds(), o)
	wantFates(t, out, Kept)
	mask := out[0].Call.Mask
	if !certain || mask == nil || !mask[4] || !mask[5] || mask[6] || mask[7] {
		t.Fatalf("certain %v, mask %v; want the two shown cells only", certain, mask)
	}

	unknown := Oracle{Cell: func(x, y int, want color.RGBA) Verdict { return Unknown }}
	out, certain = Explain(calls, frame.Bounds(), unknown)
	wantFates(t, out, Undecided)
	if certain {
		t.Fatal("a glyph nothing is known about left the frame certain")
	}
}

// A glyph drawn over an earlier glyph's cells hides them, and the earlier glyph
// still shows through the later one's own colour: the flat shadow under a face,
// and a neighbour that overlaps. Both are kept, whole, and erasing them puts the
// background back.
func TestExplainCountsCellsALaterKeptGlyphPaintedOver(t *testing.T) {
	frame := explainFrame(12, 3)
	original := image.NewRGBA(frame.Rect)
	copy(original.Pix, frame.Pix)
	font := barFont()
	calls := text.Record(func() {
		font.DrawFlat(frame, "A", 1, 0, explainDark)
		font.Draw(frame, "A", 0, 0, explainInk)
	})
	if !calls[0].Flat || calls[1].Flat {
		t.Fatalf("recorded flat %v and %v, want the shadow flat and the face not", calls[0].Flat, calls[1].Flat)
	}
	out, certain := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Kept, Kept)
	if !certain || out[0].Call.Mask != nil || out[1].Call.Mask != nil {
		t.Fatalf("certain %v, masks %v and %v; want whole glyphs", certain, out[0].Call.Mask, out[1].Call.Mask)
	}
	kept, _ := DecideWith(calls, frame.Bounds(), frameOracle(frame))
	Erase(frame, kept)
	if string(frame.Pix) != string(original.Pix) {
		t.Fatal("erasing the shadow and the face left the frame different from its background")
	}
}

// A glyph drawn into a clipped image paints the cells inside the clip only. The
// cells outside it are not judged, are not erased and are not drawn, and a glyph
// wholly outside the clip is hidden.
func TestExplainAndEraseStopAtTheClip(t *testing.T) {
	frame := explainFrame(12, 3)
	clip := image.Rect(0, 0, 3, 3)
	sub := frame.SubImage(clip).(*image.RGBA)
	calls := text.Record(func() {
		barFont().Draw(sub, "A", 0, 0, explainInk)
		barFont().Draw(sub, "A", 6, 0, explainInk)
	})
	out, certain := Explain(calls, frame.Bounds(), frameOracle(frame))
	wantFates(t, out, Kept, Hidden)
	if !certain || out[0].Call.Mask != nil {
		t.Fatalf("certain %v, mask %v; want a whole glyph, its clipped cell not counted as covered", certain, out[0].Call.Mask)
	}
	marker := color.RGBA{R: 77, G: 88, B: 99, A: 255}
	frame.SetRGBA(3, 1, marker)
	Erase(frame, []text.DrawCall{out[0].Call})
	if got := frame.RGBAAt(3, 1); got != marker {
		t.Fatalf("Erase wrote %v into a cell outside the clip", got)
	}
	if got := frame.RGBAAt(0, 1); got != explainBack {
		t.Fatalf("Erase left %v inside the clip, want the background", got)
	}
}

// A glyph drawn onto a translucent cell can be put back only when the colour
// beneath is known: the kept call then carries the cell resolved over it.
func TestExplainResolvesAGlyphOnATranslucentCell(t *testing.T) {
	g := &text.Glyph{Width: 2, Height: 1, Pixels: []text.Pixel{
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true}}}
	translucent := color.RGBA{R: 10, G: 10, B: 10, A: 128}
	beneath := color.RGBA{A: 255}
	call := text.DrawCall{Glyph: g, X: 0, Y: 0, Color: explainInk, Under: []color.RGBA{translucent, explainBack}}
	shown := Oracle{Cell: func(x, y int, want color.RGBA) Verdict { return Matches }}

	out, _ := Explain([]text.DrawCall{call}, image.Rect(0, 0, 2, 1), shown)
	wantFates(t, out, Translucent)

	shown.Below = func(x, y int, want color.RGBA) (color.RGBA, bool) { return beneath, true }
	out, _ = Explain([]text.DrawCall{call}, image.Rect(0, 0, 2, 1), shown)
	wantFates(t, out, Kept)
	if got, want := out[0].Call.Under[0], (color.RGBA{R: 10, G: 10, B: 10, A: 255}); got != want {
		t.Fatalf("the translucent cell resolved to %v, want %v", got, want)
	}
	if got := out[0].Call.Under[1]; got != explainBack {
		t.Fatalf("the opaque cell resolved to %v, want it unchanged", got)
	}
	if call.Under[0] != translucent {
		t.Fatal("Explain changed the caller's record")
	}

	shown.Below = func(x, y int, want color.RGBA) (color.RGBA, bool) { return color.RGBA{}, false }
	out, _ = Explain([]text.DrawCall{call}, image.Rect(0, 0, 2, 1), shown)
	wantFates(t, out, Translucent)
}

// A glyph a later translucent draw dimmed is kept with the dim as its tint, and
// the cells it replaced dimmed the same way: the frame holds the dimmed colour
// and the overlay draws it.
func TestExplainKeepsADimmedGlyphWithItsTint(t *testing.T) {
	frame := explainFrame(12, 3)
	calls := text.Record(func() { barFont().Draw(frame, "A", 0, 0, explainInk) })
	dim := color.RGBA{A: 0x90}
	o := Oracle{
		Cell: func(x, y int, want color.RGBA) Verdict { return Differs },
		Tint: func(x, y int, want color.RGBA) (color.RGBA, Verdict) {
			if want == explainInk {
				return dim, Matches
			}
			return color.RGBA{}, Differs
		},
	}
	out, certain := Explain(calls, frame.Bounds(), o)
	wantFates(t, out, Kept)
	kept := out[0].Call
	if !certain || kept.Tint != dim {
		t.Fatalf("certain %v, tint %v; want the dim %v", certain, kept.Tint, dim)
	}
	for n, p := range kept.Glyph.Pixels {
		if p.Painted && kept.Under[n] != text.Tinted(dim, explainBack) {
			t.Fatalf("cell %d replaced %v, want the background dimmed to %v", n, kept.Under[n], text.Tinted(dim, explainBack))
		}
	}
	// A tint another cell contradicts is a covered cell, not a dim.
	o.Tint = func(x, y int, want color.RGBA) (color.RGBA, Verdict) {
		if x == 0 {
			return dim, Matches
		}
		return color.RGBA{A: 0x40}, Matches
	}
	out, _ = Explain(calls, frame.Bounds(), o)
	wantFates(t, out, Kept)
	if mask := out[0].Call.Mask; mask == nil || !mask[4] || mask[5] || mask[6] || mask[7] {
		t.Fatalf("mask %v, want only the cell that agrees with the dim", mask)
	}
}

// A glyph drawn on a picture that is presented without it has no raster in the
// frame: a cell nothing is known to cover counts as showing, and one a later
// picture covers is masked.
func TestExplainKeepsAnErasedGlyphWhereNothingCoversIt(t *testing.T) {
	g := &barFont().Glyphs[0]
	call := text.DrawCall{Glyph: g, X: 0, Y: 0, Color: explainInk, Erased: true}
	bounds := image.Rect(0, 0, 12, 3)
	unknown := Oracle{Cell: func(x, y int, want color.RGBA) Verdict { return Unknown }}
	out, certain := Explain([]text.DrawCall{call}, bounds, unknown)
	wantFates(t, out, Kept)
	if !certain || out[0].Call.Mask != nil {
		t.Fatalf("certain %v, mask %v; want a whole glyph", certain, out[0].Call.Mask)
	}
	partly := Oracle{Cell: func(x, y int, want color.RGBA) Verdict {
		if x == 2 {
			return Differs
		}
		return Unknown
	}}
	out, certain = Explain([]text.DrawCall{call}, bounds, partly)
	wantFates(t, out, Kept)
	if mask := out[0].Call.Mask; !certain || mask == nil || !mask[4] || !mask[5] || mask[6] || !mask[7] {
		t.Fatalf("certain %v, mask %v; want the covered cell alone masked", certain, mask)
	}
	covered := Oracle{Cell: func(x, y int, want color.RGBA) Verdict { return Differs }}
	out, _ = Explain([]text.DrawCall{call}, bounds, covered)
	wantFates(t, out, Hidden)
}

// Erase puts back the cells a kept glyph replaced, except the cells its mask
// leaves to a covering picture and the cells of a glyph that never had a raster.
func TestEraseSkipsMaskedCellsAndErasedCalls(t *testing.T) {
	frame := explainFrame(12, 3)
	calls := text.Record(func() { barFont().Draw(frame, "A", 0, 0, explainInk) })
	kept := calls[0]
	kept.Mask = []bool{
		true, true, true, true,
		true, true, false, true,
		true, true, true, true,
	}
	Erase(frame, []text.DrawCall{kept})
	for x := 0; x < 4; x++ {
		want := explainBack
		if x == 2 {
			want = explainInk
		}
		if got := frame.RGBAAt(x, 1); got != want {
			t.Errorf("cell (%d,1) holds %v after Erase, want %v", x, got, want)
		}
	}
	erased := calls[0]
	erased.Erased = true
	erased.Under = nil
	before := string(frame.Pix)
	Erase(frame, []text.DrawCall{erased})
	if string(frame.Pix) != before {
		t.Fatal("Erase changed the frame for a glyph that never had a raster")
	}
}

// The overlay stops at a call's clip and leaves the cells its mask names to the
// picture that covers them.
func TestCompositeStopsAtTheClipAndTheMask(t *testing.T) {
	g := &text.Glyph{Width: 4, Height: 1, Pixels: make([]text.Pixel, 4)}
	for i := range g.Pixels {
		g.Pixels[i] = text.Pixel{Level: text.MaxLevel, Painted: true}
	}
	const scale = 4
	painted := func(dst *image.RGBA, x0, x1 int) (in, out int) {
		for y := 0; y < dst.Rect.Dy(); y++ {
			for x := 0; x < dst.Rect.Dx(); x++ {
				if dst.RGBAAt(x, y).A == 0 {
					continue
				}
				if x >= x0 && x < x1 {
					in++
				} else {
					out++
				}
			}
		}
		return in, out
	}
	masked := image.NewRGBA(image.Rect(0, 0, 16, 4))
	Composite(masked, []text.DrawCall{{Glyph: g, Color: explainInk, Mask: []bool{true, true, false, false}}}, scale, 0, 0)
	if in, out := painted(masked, 0, 8); in != 8*4 || out != 0 {
		t.Fatalf("a mask over the last two cells painted %d cells inside the first two and %d outside", in, out)
	}
	clipped := image.NewRGBA(image.Rect(0, 0, 16, 4))
	Composite(clipped, []text.DrawCall{{Glyph: g, Color: explainInk, Clip: image.Rect(0, 0, 2, 1)}}, scale, 0, 0)
	if in, out := painted(clipped, 0, 8); in != 8*4 || out != 0 {
		t.Fatalf("a clip over the first two columns painted %d cells inside and %d outside", in, out)
	}
	whole := image.NewRGBA(image.Rect(0, 0, 16, 4))
	Composite(whole, []text.DrawCall{{Glyph: g, Color: explainInk}}, scale, 0, 0)
	if in, out := painted(whole, 0, 16); in != 16*4 || out != 0 {
		t.Fatalf("an unclipped glyph painted %d of 64 cells", in)
	}
}

// A tinted glyph composites in the dimmed colour it shows in the frame.
func TestCompositeDrawsATintedGlyphDimmed(t *testing.T) {
	g := &text.Glyph{Width: 1, Height: 1, Pixels: []text.Pixel{{Level: text.MaxLevel, Painted: true}}}
	dim := color.RGBA{A: 0x90}
	dst := image.NewRGBA(image.Rect(0, 0, 3, 3))
	Composite(dst, []text.DrawCall{{Glyph: g, Color: explainInk, Tint: dim}}, 3, 0, 0)
	want := text.Tinted(dim, explainInk)
	if got := dst.RGBAAt(1, 1); got != want {
		t.Fatalf("the middle of a dimmed glyph is %v, want %v", got, want)
	}
}
