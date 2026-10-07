package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// documentsTestArt builds the panel's eleven bitmaps at their shipped sizes,
// each filled with a DISTINCT colour.
//
// Distinguishable is the point, on panelFont's own reasoning: art whose pieces
// all looked alike would let a composer that blitted the wrong state, or one
// piece over another's rectangle, pass every assertion below.
func documentsTestArt() *DocumentPanelArt {
	fill := func(w, h int, c color.RGBA) *image.RGBA {
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		for i := 0; i < len(img.Pix); i += 4 {
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = c.R, c.G, c.B, c.A
		}
		return img
	}
	a := &DocumentPanelArt{Sheet: fill(640, 480, color.RGBA{0x10, 0x10, 0x10, 0xff})}
	for i := 0; i < 3; i++ {
		a.Left[i] = fill(56, 40, color.RGBA{uint8(0x40 + i), 0, 0, 0xff})
		a.Right[i] = fill(60, 40, color.RGBA{0, uint8(0x40 + i), 0, 0xff})
	}
	for i := 0; i < 4; i++ {
		a.OK[i] = fill(44, 32, color.RGBA{0, 0, uint8(0x40 + i), 0xff})
	}
	return a
}

// TestComposeScreenSelectsDocumentsComposer is the screen registry's own
// selection witness for ScreenDocuments: composeScreen dispatches to
// composeDocumentsPanel over the flow's own panel and to nothing else.
func TestComposeScreenSelectsDocumentsComposer(t *testing.T) {
	a := newTestApp(t, appRows(1), nil)
	a.flow.docPanel = newDocumentPanel(documentsTestArt(), nil, []DocumentPage{{Text: "x"}})
	a.flow.screen = ScreenDocuments

	got, err := a.composeScreen()
	if err != nil {
		t.Fatalf("composeScreen() on %v: %v", ScreenDocuments, err)
	}
	want := composeDocumentsPanel(a.flow.docPanel)
	if got.Bounds() != want.Bounds() || !bytes.Equal(got.Pix, want.Pix) {
		t.Error("composeScreen() on documents did not match composeDocumentsPanel(flow.docPanel)")
	}
}

// TestDocumentsPanelDrawsItsThreeControlsAtTheirOwnRects is this screen's
// geometry witness for the sheet and the three controls.
//
// THE EXPECTED FRAME IS BUILT HERE, from hand-written literals, and compared
// against the production composite over the whole 640x480 frame — not against a
// second call to composeDocumentsPanel, and not by scanning a region for any
// matching pixel. The four origins below are MENU-DOC-009's own rectangles
// written out, so a composer that moved any one of them by a pixel fails on
// both edges of the moved piece.
//
// Mutation-proved at the USE site (AGENTS.md rule 6): shifting the left
// arrow's own copyNative call in composeDocumentsPanel to
// docLeftRect.Min.Add(image.Pt(1, 0)) fails this test, and so does the same
// shift on the right arrow's and OK's calls; each reverted byte-identical.
func TestDocumentsPanelDrawsItsThreeControlsAtTheirOwnRects(t *testing.T) {
	art := documentsTestArt()
	p := newDocumentPanel(art, nil, nil)

	want := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	blit := func(src *image.RGBA, x, y int) {
		b := src.Bounds()
		for sy := 0; sy < b.Dy(); sy++ {
			for sx := 0; sx < b.Dx(); sx++ {
				want.Set(x+sx, y+sy, src.At(b.Min.X+sx, b.Min.Y+sy))
			}
		}
	}
	blit(art.Sheet, 0, 0)
	blit(art.Left[0], 0, 200)
	blit(art.Right[0], 576, 200)
	blit(art.OK[0], 560, 416)

	got := composeDocumentsPanel(p)
	if got.Bounds() != want.Bounds() {
		t.Fatalf("composed bounds = %v, want %v", got.Bounds(), want.Bounds())
	}
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("the composed panel does not match the sheet with the three idle controls " +
			"blitted at (0,200), (576,200) and (560,416)")
	}
}

// TestDocumentsPanelDrawsTheTextPageAtItsOwnOriginAndPitch is this screen's
// geometry witness for the text page: the first line's origin, the pitch
// between lines, and that a page holds at most 21 of them.
//
// THE EXPECTATION IS DRAWN HERE with direct font.Draw calls at hand-written
// literals — (92+docParagraphIndent, 72) and a pitch of the font's own height
// plus two — and compared over the whole frame. It is not a second call to
// composeDocumentsPanel and it does not scan for a matching pixel.
//
// EVERY FIXTURE LINE IS ITS OWN ONE-WORD PARAGRAPH (documentsTestBody has no
// space on any of its 25 source lines), so every wrapped line opens a
// paragraph — drawn docParagraphIndent right of the margin — and closes one —
// never justified, since a one-word line has no gap to stretch. That is why a
// plain font.Draw at a fixed x witnesses this fixture; a fixture with a
// justified line needs the dedicated wrap/justify tests instead.
//
// Mutation-proved at the USE site: changing the page loop's own y in
// composeDocumentsPanel to docContentAt.Y+i*pitch+1 fails this test, and so
// does changing the pitch term to p.font.Height()+3; each reverted
// byte-identical.
func TestDocumentsPanelDrawsTheTextPageAtItsOwnOriginAndPitch(t *testing.T) {
	font := documentsTestFont()
	p := newDocumentPanel(documentsTestArt(), font, []DocumentPage{{Text: documentsTestBody()}})
	lines := p.lines[0]
	if len(lines) <= docPageLines {
		t.Fatalf("the fixture wrapped to %d lines, want more than one page of %d — "+
			"a fixture that fits on one page cannot witness the page bound", len(lines), docPageLines)
	}
	for i, l := range lines {
		if !l.indent || l.justify {
			t.Fatalf("line %d: indent=%v justify=%v, want every fixture line indented and unjustified", i, l.indent, l.justify)
		}
	}

	want := composeDocumentsPanel(newDocumentPanel(documentsTestArt(), nil, nil))
	for i := 0; i < docPageLines; i++ {
		font.Draw(want, lines[i].text, 92+docParagraphIndent, 72+i*(font.Height()+2), docTextColor)
	}

	got := composeDocumentsPanel(p)
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("the composed text page does not match 21 lines drawn from (92+docParagraphIndent,72) " +
			"at a pitch of the font's own height plus two")
	}

	// The second page starts at line 21 and holds what is left, which is what
	// makes the last page a partial one.
	if !p.pageStep(+1) {
		t.Fatal("pageStep(+1) refused a document with more than one page")
	}
	want2 := composeDocumentsPanel(newDocumentPanel(documentsTestArt(), nil, nil))
	for i := docPageLines; i < len(lines); i++ {
		font.Draw(want2, lines[i].text, 92+docParagraphIndent, 72+(i-docPageLines)*(font.Height()+2), docTextColor)
	}
	if !bytes.Equal(composeDocumentsPanel(p).Pix, want2.Pix) {
		t.Fatal("the second page does not match the lines from 21 on, drawn from (92+docParagraphIndent,72)")
	}
}

// documentsTestBody is a 25-line text: more than one 21-line page, with a
// different byte on every line so a page that took the wrong slice moves
// painted pixels.
func documentsTestBody() string {
	var body []byte
	for i := 0; i < 25; i++ {
		if i > 0 {
			body = append(body, byte(10))
		}
		body = append(body, byte('a'+i))
	}
	return string(body)
}

// documentsTestFont is panelFont's shape at an 8x8 cell: 224 records, each
// painting one distinguishable pixel, record 0 blank.
func documentsTestFont() *text.Font {
	const cell = 8
	f := &text.Font{Spacing: 2, Glyphs: make([]text.Glyph, 224)}
	for k := range f.Glyphs {
		g := text.Glyph{Width: cell, Height: cell,
			Pixels: make([]text.Pixel, cell*cell), Advance: 6}
		if k != 0 {
			g.Pixels[(k%cell)*cell+(k/cell)%cell] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
		f.Glyphs[k] = g
	}
	return f
}

// TestDocumentsArrowPagesFirstAndChangesDocumentOnlyWhenThePageStepReturnsZero
// is MENU-DOC-009's own paging rule: one arrow pair walks pages first and
// documents second.
//
// A build that paged documents directly is a different screen, which is why
// this is asserted on the ORDER and not only on the endpoints.
func TestDocumentsArrowPagesFirstAndChangesDocumentOnlyWhenThePageStepReturnsZero(t *testing.T) {
	font := documentsTestFont()
	// Three elements: a two-page text, a picture (no lines at all), and a
	// one-page text.
	pages := []DocumentPage{
		{Text: documentsTestBody()},
		{Picture: image.NewRGBA(image.Rect(0, 0, 4, 4))},
		{Text: "one"},
	}
	p := newDocumentPanel(documentsTestArt(), font, pages)

	if p.doc != 0 || p.line != 0 {
		t.Fatalf("a fresh panel is at (doc %d, line %d), want (0, 0)", p.doc, p.line)
	}
	p.step(+1)
	if p.doc != 0 || p.line != docPageLines {
		t.Fatalf("after one right step: (doc %d, line %d), want (0, %d) — the arrow must page "+
			"the current document before it changes document", p.doc, p.line, docPageLines)
	}
	p.step(+1)
	if p.doc != 1 || p.line != 0 {
		t.Fatalf("after the second right step: (doc %d, line %d), want (1, 0) — the document "+
			"changes only when the page step returns zero", p.doc, p.line)
	}
	// A picture carries no lines, so its page step always refuses and one
	// press moves on.
	p.step(+1)
	if p.doc != 2 || p.line != 0 {
		t.Fatalf("after the third right step: (doc %d, line %d), want (2, 0) — a picture has no "+
			"lines, so an arrow over it moves to the next document", p.doc, p.line)
	}
	// The last element's last page refuses rather than wrapping.
	p.step(+1)
	if p.doc != 2 || p.line != 0 {
		t.Fatalf("a right step at the end moved to (doc %d, line %d), want it refused at (2, 0)",
			p.doc, p.line)
	}
	// Back, and a new element starts at its first line in this direction too.
	p.step(-1)
	if p.doc != 1 || p.line != 0 {
		t.Fatalf("after a left step: (doc %d, line %d), want (1, 0)", p.doc, p.line)
	}
	p.step(-1)
	if p.doc != 0 || p.line != 0 {
		t.Fatalf("after the second left step: (doc %d, line %d), want (0, 0) — DIV-300: a new "+
			"element starts at its first line in both directions", p.doc, p.line)
	}
	p.step(-1)
	if p.doc != 0 || p.line != 0 {
		t.Fatalf("a left step at the start moved to (doc %d, line %d), want it refused at (0, 0)",
			p.doc, p.line)
	}
}

// TestDocumentsControlStatesSelectTheOriginalsOwnBitmapIndices pins
// MENU-DOC-009's own selection: element 1 on hover, 2 on press, 3 for OK, 0 on
// leave — and that OK's element 2 is reached by no state (DIV-301).
func TestDocumentsControlStatesSelectTheOriginalsOwnBitmapIndices(t *testing.T) {
	for _, c := range []struct {
		control     int
		hover       int
		press       int
		wantArrowIx int
	}{
		{docControlLeft, docNoControl, docNoControl, 0},
		{docControlLeft, docControlLeft, docNoControl, 1},
		{docControlLeft, docControlLeft, docControlLeft, 2},
		{docControlLeft, docControlLeft, docControlRight, 0},
		{docControlRight, docControlRight, docNoControl, 1},
		{docControlRight, docControlLeft, docNoControl, 0},
	} {
		if got := docArrowFrame(c.control, c.hover, c.press); got != c.wantArrowIx {
			t.Errorf("docArrowFrame(%d, hover %d, press %d) = %d, want %d",
				c.control, c.hover, c.press, got, c.wantArrowIx)
		}
	}
	for _, c := range []struct{ hover, press, want int }{
		{docNoControl, docNoControl, 0},
		{docControlOK, docNoControl, 1},
		{docControlOK, docControlOK, 3},
		{docControlOK, docControlLeft, 0},
	} {
		if got := docOKFrame(c.hover, c.press); got != c.want {
			t.Errorf("docOKFrame(hover %d, press %d) = %d, want %d", c.hover, c.press, got, c.want)
		}
	}
	// Element 2, Ok_l_off, is reachable from no (hover, press) pair over the
	// four constants — the row's own index sentence read out.
	for _, hover := range []int{docNoControl, docControlLeft, docControlRight, docControlOK} {
		for _, press := range []int{docNoControl, docControlLeft, docControlRight, docControlOK} {
			if docOKFrame(hover, press) == 2 {
				t.Errorf("docOKFrame(hover %d, press %d) selected Ok_l_off, which no arm selects",
					hover, press)
			}
		}
	}
}

// TestDocumentControlAtAnswersTheThreeDecodedRectangles pins the hit test
// against MENU-DOC-009's own literals, including each rectangle's exclusive
// far edge.
func TestDocumentControlAtAnswersTheThreeDecodedRectangles(t *testing.T) {
	for _, c := range []struct {
		x, y int
		want int
		ok   bool
	}{
		{0, 200, docControlLeft, true},
		{55, 239, docControlLeft, true},
		{56, 200, docNoControl, false},
		{0, 199, docNoControl, false},
		{576, 200, docControlRight, true},
		{635, 239, docControlRight, true},
		{636, 200, docNoControl, false},
		{560, 416, docControlOK, true},
		{603, 447, docControlOK, true},
		{604, 416, docNoControl, false},
		{320, 240, docNoControl, false},
	} {
		got, ok := documentControlAt(c.x, c.y)
		if got != c.want || ok != c.ok {
			t.Errorf("documentControlAt(%d, %d) = (%d, %v), want (%d, %v)",
				c.x, c.y, got, ok, c.want, c.ok)
		}
	}
}

// TestOpenDocumentsRefusesAnEmptyCollectionAndRebuildsAtEveryOpen pins the two
// rules flow.openDocuments carries: nothing collected does not open, and the
// panel is built from what the source answers NOW.
func TestOpenDocumentsRefusesAnEmptyCollectionAndRebuildsAtEveryOpen(t *testing.T) {
	src := &fakeDocumentSource{}
	f := &flow{screen: ScreenMap, docSrc: src, docArt: documentsTestArt()}

	if f.openDocuments(ScreenMap) {
		t.Fatal("openDocuments opened over an empty collection")
	}
	if f.screen != ScreenMap || f.docPanel != nil {
		t.Fatalf("a refused open left screen %v panel %v", f.screen, f.docPanel)
	}

	src.pages = []DocumentPage{{Text: "one"}}
	if !f.openDocuments(ScreenMap) {
		t.Fatal("openDocuments refused a collection of one")
	}
	if f.screen != ScreenDocuments || f.docBack != ScreenMap || len(f.docPanel.pages) != 1 {
		t.Fatalf("open left screen %v back %v pages %d", f.screen, f.docBack, len(f.docPanel.pages))
	}

	f.closeDocuments()
	if f.screen != ScreenMap || f.docPanel != nil {
		t.Fatalf("close left screen %v panel %v", f.screen, f.docPanel)
	}

	// The collection grew between the two opens, and the second panel sees it.
	src.pages = append(src.pages, DocumentPage{Text: "two"})
	if !f.openDocuments(ScreenTown) {
		t.Fatal("openDocuments refused a collection of two")
	}
	if len(f.docPanel.pages) != 2 {
		t.Fatalf("the second open built %d pages, want 2 — the source is asked at every open",
			len(f.docPanel.pages))
	}
	if f.docBack != ScreenTown {
		t.Fatalf("docBack = %v, want the screen this open was armed from", f.docBack)
	}
}

type fakeDocumentSource struct{ pages []DocumentPage }

func (s *fakeDocumentSource) Documents() []DocumentPage { return s.pages }
