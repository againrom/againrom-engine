package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

func TestReadoutReplaysTheGlyphsOfItsCachedPicture(t *testing.T) {
	v := readoutViewer(t)
	v.SetReadout(Readout{PeriodUS: 62000, Tick: 10, Digest: 99})
	t.Cleanup(text.ResetCapture)
	var first *image.RGBA
	var count int
	for frame := range 2 {
		text.ResetCapture()
		start := beginTextCapture(true)
		pic, at, ok := v.readoutPresent(60)
		endTextCapture(start, at)
		if !ok {
			t.Fatal("the visible readout has no picture")
		}
		calls := text.Captured()
		if len(calls) == 0 {
			t.Fatalf("frame %d: the readout paints text but captured no glyph", frame)
		}
		if frame == 0 {
			first, count = pic, len(calls)
		} else if pic != first || v.readoutBuilds != 1 || len(calls) != count {
			t.Fatalf("cached frame: same picture %v, builds %d, glyphs %d, want %d", pic == first, v.readoutBuilds, len(calls), count)
		}
		for _, c := range calls {
			if !image.Pt(c.X, c.Y).In(pic.Bounds().Add(at)) {
				t.Fatalf("glyph at (%d,%d) lies outside its presented readout", c.X, c.Y)
			}
		}
	}
}

func TestSpellbookFallbackLabelsAndBindingsAreCapturedByTheDrawnFrame(t *testing.T) {
	a := spellbookTestApp(t)
	a.Layout(1280, 960)
	a.SetTextSmoothing(true)
	v := a.flow.viewer
	v.ShowReadout(false)
	v.SetEntities(sbEntities())
	v.sel = selection{sbUnitID}
	v.SetSpellbook(sbUnitID, sbBook())
	v.SetQuickSpells(&[4]uint32{1})
	if _, _, ok := v.spellbookBar(); !ok {
		if err := a.HeadlessKey("book"); err != nil {
			t.Fatal(err)
		}
	}
	bar, _, ok := v.spellbookBar()
	if !ok {
		t.Fatal("the book did not open")
	}
	text.StartAudit()
	t.Cleanup(func() { text.StopAudit(); text.ResetCapture() })
	a.Draw(ebiten.NewImage(1280, 960))
	audit := text.StopAudit()
	if audit.Uncaptured != 0 {
		t.Fatalf("%d fallback glyphs painted outside capture: %v", audit.Uncaptured, audit.Sites)
	}
	kept := 0
	for _, ft := range a.TextFates() {
		if image.Pt(ft.Call.X, ft.Call.Y).In(bar) && ft.Fate == textsmooth.Kept {
			kept++
		}
	}
	if kept != 7 {
		t.Fatalf("the drawn book kept %d glyphs, want five abbreviation glyphs and F5", kept)
	}
}

// frameOracle answers Explain from a composed frame.
func frameOracle(frame *image.RGBA) textsmooth.Oracle {
	return textsmooth.Oracle{Cell: func(x, y int, want color.RGBA) textsmooth.Verdict {
		if frame.RGBAAt(x, y) == want {
			return textsmooth.Matches
		}
		return textsmooth.Differs
	}}
}

// The detailed page pastes its character card, composed on a canvas of its own,
// into the page. Every glyph the card captured lies where the page shows it and
// over an opaque cell of the card's background, so the overlay keeps each one and
// can put that cell back under it.
func TestDetailedPageCardGlyphsAreCapturedWhereThePageShowsThem(t *testing.T) {
	font := chargenTestFont()
	bg := logTestPic(image.Rect(0, 0, 160, 242), color.RGBA{R: 40, G: 30, B: 20, A: 255})
	art := &ChargenPresentation{Layout: testGenerator(), Plate: image.NewRGBA(image.Rect(0, 0, 160, 238)), CardBackground: bg, Font: font}
	c := NewChargen(ChargenSetup{
		PreCreate: &ChargenPreCreate{Art: art},
		Choices: []ChargenChoice{
			{Options: []string{"m"}, Parent: -1},
			{Options: []string{"fighter"}, Parent: -1},
			{Options: []string{"blade", "axe", "bludgeon", "pike", "shooting"}, Parent: -1},
		},
		Stats: []ChargenStat{{Floor: 0, Ceiling: 50, Start: 25}}, Cost: triangular(50), Budget: 2000,
	})
	c.Forward()

	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	text.SetCapture(false)
	frame := composeChargenDetailedPage(c, chargenNone, chargenNone)
	text.StopCapture()
	calls := append([]text.DrawCall(nil), text.Captured()...)

	outcomes, _ := textsmooth.Explain(calls, frame.Bounds(), frameOracle(frame))
	inCard, kept := 0, 0
	for i, o := range outcomes {
		call := calls[i]
		if !image.Pt(call.X, call.Y).In(chargenCardBox) {
			continue
		}
		inCard++
		switch o.Fate {
		case textsmooth.Kept:
			kept++
			for n, p := range call.Glyph.Pixels {
				if p.Painted && o.Call.Under[n].A != 255 {
					t.Fatalf("glyph %d at (%d,%d) replaced the translucent cell %v: the card's background is not what lies under it", i, call.X, call.Y, o.Call.Under[n])
				}
			}
		case textsmooth.Blank, textsmooth.Repeat:
		default:
			t.Errorf("glyph %d at (%d,%d) is %s: it would stay baked", i, call.X, call.Y, o.Fate)
		}
	}
	if inCard == 0 || kept == 0 {
		t.Fatalf("%d glyphs captured inside the card's box, %d kept", inCard, kept)
	}
}

// pixelLogTextScene holds a background, a picture that puts one glyph colour on
// cell (3,1) and translucent dims laid over it, each as the log records them.
func pixelLogTextScene(dims ...color.RGBA) (*pixelLog, color.RGBA, color.RGBA) {
	ink := color.RGBA{R: 200, G: 180, B: 120, A: 255}
	back := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	pic := image.NewRGBA(image.Rect(0, 0, 4, 4))
	pic.SetRGBA(1, 1, ink)
	var l pixelLog
	l.reset(image.Rect(0, 0, 8, 4))
	l.fill(l.bounds, back)
	l.over(pic, image.Pt(2, 0))
	for _, d := range dims {
		l.overSolid(l.bounds, d)
	}
	l.end()
	return &l, ink, back
}

// below is the colour a cell held before the draw that put the glyph there, and
// only while nothing was drawn over the glyph since.
func TestPixelLogBelowIsWhatLayUnderTheDrawThatPutTheGlyphThere(t *testing.T) {
	l, ink, back := pixelLogTextScene()
	if got, ok := l.below(3, 1, ink); !ok || got != back {
		t.Fatalf("below = %v, %v; want %v under the glyph", got, ok, back)
	}
	if _, ok := l.below(3, 1, back); ok {
		t.Fatal("below answered for a colour the last draw there did not put")
	}
	if _, ok := l.below(0, 0, back); ok {
		t.Fatal("below answered for a cell with nothing drawn beneath the fill")
	}
	dimmed, _, _ := pixelLogTextScene(color.RGBA{A: 0x90})
	if _, ok := dimmed.below(3, 1, ink); ok {
		t.Fatal("below answered for a glyph a dim was laid over")
	}
	var unknown pixelLog
	unknown.reset(image.Rect(0, 0, 8, 4))
	unknown.end()
	if _, ok := unknown.below(3, 1, ink); ok {
		t.Fatal("below answered for a cell nothing is known about")
	}
}

// tint is the colour of the translucent draws laid over a glyph after it, in the
// order they were drawn, and tells a glyph a picture covered from one a dim lies
// over.
func TestPixelLogTintNamesTheDimsDrawnAfterTheGlyph(t *testing.T) {
	first, second := color.RGBA{A: 0x90}, color.RGBA{R: 4, G: 4, B: 4, A: 0x40}
	l, ink, back := pixelLogTextScene()
	if got, v := l.tint(3, 1, ink); v != textsmooth.Matches || got != (color.RGBA{}) {
		t.Fatalf("tint with no dim = %v, %v; want none, Matches", got, v)
	}
	l, ink, _ = pixelLogTextScene(first)
	if got, v := l.tint(3, 1, ink); v != textsmooth.Matches || got != first {
		t.Fatalf("tint under one dim = %v, %v; want %v, Matches", got, v, first)
	}
	l, ink, _ = pixelLogTextScene(first, second)
	if got, v := l.tint(3, 1, ink); v != textsmooth.Matches || got != text.Tinted(second, first) {
		t.Fatalf("tint under two dims = %v, %v; want %v, Matches", got, v, text.Tinted(second, first))
	}
	if _, v := l.tint(3, 1, back); v != textsmooth.Differs {
		t.Fatalf("tint for a colour the last opaque draw is not = %v, want Differs", v)
	}
	many := make([]color.RGBA, 5)
	for i := range many {
		many[i] = color.RGBA{A: 0x10}
	}
	l, ink, _ = pixelLogTextScene(many...)
	if _, v := l.tint(3, 1, ink); v != textsmooth.Unknown {
		t.Fatalf("tint under five dims = %v, want Unknown", v)
	}
	l, ink, _ = pixelLogTextScene()
	l.begin(l.bounds)
	l.unknown(image.Rect(3, 1, 4, 2))
	l.end()
	if _, v := l.tint(3, 1, ink); v != textsmooth.Unknown {
		t.Fatalf("tint over a draw the log cannot model = %v, want Unknown", v)
	}
	oracle := l.oracle()
	if oracle.Cell == nil || oracle.Below == nil || oracle.Tint == nil {
		t.Fatal("the log's oracle answers only some of the questions")
	}
}

// A composer that paints text on a picture laid over content the frame does not
// model hands back the picture with its glyphs, a copy to blit without them and
// the glyphs, marked erased. With the overlay off it hands back the picture
// alone, and the composer's window is left as it was.
func TestGlyphPictureHandsBackTheGlyphsAndAPictureWithoutThem(t *testing.T) {
	font := smoothingTestFont()
	compose := func() *image.RGBA {
		pic := image.NewRGBA(image.Rect(0, 0, 8, 4))
		font.Draw(pic, "A", 2, 1, smoothingTestColor)
		return pic
	}
	pic, blit, calls := glyphPicture(false, compose)
	if blit != nil || calls != nil || pic.RGBAAt(2, 1) != smoothingTestColor {
		t.Fatalf("with the overlay off: blit %v, calls %v, glyph cell %v", blit != nil, calls, pic.RGBAAt(2, 1))
	}

	text.ResetCapture()
	t.Cleanup(text.ResetCapture)
	text.SetCapture(false)
	pic, blit, calls = glyphPicture(true, compose)
	if text.CapturedLen() != 0 || !text.Capturing() {
		t.Fatalf("the composer's window holds %d glyphs and capturing %v; want it untouched", text.CapturedLen(), text.Capturing())
	}
	text.StopCapture()
	if len(calls) != 1 || !calls[0].Erased || calls[0].X != 2 || calls[0].Y != 1 {
		t.Fatalf("calls %+v, want the one glyph at (2,1), marked erased", calls)
	}
	if pic.RGBAAt(2, 1) != smoothingTestColor {
		t.Fatal("the picture lost its glyph")
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			if got := blit.RGBAAt(x, y); got != (color.RGBA{}) {
				t.Fatalf("the blit holds %v at (%d,%d), want a picture without the glyph", got, x, y)
			}
		}
	}
	if pic, blit, calls = glyphPicture(true, func() *image.RGBA { return nil }); pic != nil || blit != nil || calls != nil {
		t.Fatal("a composer with nothing to show handed back a picture")
	}
}

// sameCalls tells two lists of calls apart by every property the overlay draws
// or erases with.
func TestSameCallsSeesEveryPropertyTheOverlayDrawsWith(t *testing.T) {
	g := &text.Glyph{Width: 1, Height: 1, Pixels: []text.Pixel{{Level: text.MaxLevel, Painted: true}}}
	base := text.DrawCall{Glyph: g, X: 1, Y: 2, Color: smoothingTestColor, Under: []color.RGBA{{A: 255}}, Mask: []bool{true}}
	same := func(a, b text.DrawCall, under bool) bool {
		return sameCalls([]text.DrawCall{a}, []text.DrawCall{b}, under)
	}
	if !same(base, base, true) {
		t.Fatal("a call differs from itself")
	}
	for name, mutate := range map[string]func(*text.DrawCall){
		"x":      func(c *text.DrawCall) { c.X++ },
		"colour": func(c *text.DrawCall) { c.Color.R++ },
		"flat":   func(c *text.DrawCall) { c.Flat = true },
		"tint":   func(c *text.DrawCall) { c.Tint = color.RGBA{A: 0x90} },
		"clip":   func(c *text.DrawCall) { c.Clip = image.Rect(0, 0, 3, 3) },
		"mask":   func(c *text.DrawCall) { c.Mask = []bool{false} },
		"erased": func(c *text.DrawCall) { c.Erased = true },
	} {
		other := base
		mutate(&other)
		if same(base, other, true) {
			t.Errorf("calls that differ in %s compare equal", name)
		}
	}
	other := base
	other.Under = []color.RGBA{{R: 1, A: 255}}
	if same(base, other, true) || !same(base, other, false) {
		t.Error("the cells a call replaced should count only when asked for")
	}
	if sameCalls([]text.DrawCall{base}, nil, true) {
		t.Error("lists of different length compare equal")
	}
}

// A damage figure composed while the overlay is on carries its glyphs and a
// picture without them, and is placed with both; with the overlay off it is
// placed as the picture alone.
func TestASmoothedFigureCarriesItsGlyphsAndAPictureWithoutThem(t *testing.T) {
	for _, on := range []bool{true, false} {
		v := litViewer(t)
		v.SetTextSmoothing(on)
		v.SetLocalOwner(1)
		push(v, at0, numeralUnit(1, 1, 40))
		push(v, at0, numeralUnit(1, 1, 33))
		placed := v.numeralPlacements()
		if len(placed) != 1 || placed[0].Pic == nil {
			t.Fatalf("overlay %v: placed %d figures", on, len(placed))
		}
		p := placed[0]
		if !on {
			if p.Blit != nil || p.Calls != nil {
				t.Errorf("overlay off: the figure carries a blit and %d glyphs", len(p.Calls))
			}
			continue
		}
		if len(p.Calls) != 2 {
			t.Fatalf("overlay on: the figure carries %d glyphs, want its shadow and its face", len(p.Calls))
		}
		for i, c := range p.Calls {
			if !c.Erased {
				t.Errorf("glyph %d is not marked erased", i)
			}
		}
		for y := p.Pic.Rect.Min.Y; y < p.Pic.Rect.Max.Y; y++ {
			for x := p.Pic.Rect.Min.X; x < p.Pic.Rect.Max.X; x++ {
				if got := p.Blit.RGBAAt(x, y); got != (color.RGBA{}) {
					t.Fatalf("the blit holds %v at (%d,%d), want no glyph raster", got, x, y)
				}
			}
		}
	}
}
