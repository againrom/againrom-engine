package ui

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// smoothingTestColor and smoothingTestFont are a tiny, hand-built font used
// only to place one recognizable glyph inside a composed sub-picture, so a
// capture test can tell where the ENGINE placed it apart from where the
// PICTURE was pasted.
var smoothingTestColor = color.RGBA{R: 10, G: 20, B: 30, A: 255}

// smoothingTestFont sizes its record slice exactly as text.FirstChar and the
// byte 'A' require (text.Font.index: record c-FirstChar with the default,
// non-converting selector) so byte 'A' actually selects the inked record and
// not the blank fallback a too-short slice would silently substitute.
func smoothingTestFont() *text.Font {
	blank := text.Glyph{Width: 2, Height: 2, Advance: 2}
	glyphs := make([]text.Glyph, 'A'-text.FirstChar+1)
	for i := range glyphs {
		glyphs[i] = blank
	}
	glyphs['A'-text.FirstChar] = text.Glyph{Width: 2, Height: 2, Advance: 2, Pixels: []text.Pixel{
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
		{Level: text.MaxLevel, Painted: true}, {Level: text.MaxLevel, Painted: true},
	}}
	return &text.Font{Glyphs: glyphs}
}

// TestMarkCaptureAndShiftCaptureMoveOnlyEntriesFromStart is the App-side
// helper pair's own contract (DIV-1385): a call inside an ALREADY-OPEN outer
// capture window records where a sub-picture's own glyphs begin, composes
// the picture, and shifts only those glyphs by the same offset the picture
// itself is pasted at — leaving glyphs captured before the mark untouched.
func TestMarkCaptureAndShiftCaptureMoveOnlyEntriesFromStart(t *testing.T) {
	text.ResetCapture()
	text.SetCapture(true)
	defer text.StopCapture()

	f := smoothingTestFont()
	outer := image.NewRGBA(image.Rect(0, 0, 20, 20))
	f.Draw(outer, "A", 0, 0, smoothingTestColor) // outer-window glyph, must not move

	start := markCapture()
	sub := image.NewRGBA(image.Rect(0, 0, 4, 4))
	f.Draw(sub, "A", 1, 1, smoothingTestColor) // the sub-picture's own local coordinates
	shiftCapture(start, image.Pt(50, 60))

	calls := text.Captured()
	if len(calls) != 2 {
		t.Fatalf("Captured() len = %d, want 2", len(calls))
	}
	if calls[0].X != 0 || calls[0].Y != 0 {
		t.Fatalf("the outer glyph moved: %+v", calls[0])
	}
	if calls[1].X != 51 || calls[1].Y != 61 {
		t.Fatalf("the sub-picture glyph = %+v, want X=51 Y=61 (1+50, 1+60)", calls[1])
	}
}

// TestBeginEndTextCaptureOpensAndClosesItsOwnWindow is the Viewer-side
// helper pair's own contract: unlike markCapture/shiftCapture, this pair
// opens and closes an INDEPENDENT text.SetCapture window of its own, for a
// compose call that stands alone rather than nesting inside one the caller
// already opened. Disabled answers -1 and both calls become no-ops.
func TestBeginEndTextCaptureOpensAndClosesItsOwnWindow(t *testing.T) {
	text.ResetCapture()
	f := smoothingTestFont()

	if start := beginTextCapture(false); start != -1 {
		t.Fatalf("beginTextCapture(false) = %d, want -1", start)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	f.Draw(dst, "A", 0, 0, smoothingTestColor)
	endTextCapture(-1, image.Pt(9, 9))
	if n := text.CapturedLen(); n != 0 {
		t.Fatalf("disabled window recorded %d glyphs, want 0", n)
	}

	start := beginTextCapture(true)
	pic := image.NewRGBA(image.Rect(0, 0, 4, 4))
	f.Draw(pic, "A", 2, 3, smoothingTestColor)
	endTextCapture(start, image.Pt(100, 200))

	calls := text.Captured()
	if len(calls) != 1 {
		t.Fatalf("Captured() len = %d, want 1", len(calls))
	}
	if calls[0].X != 102 || calls[0].Y != 203 {
		t.Fatalf("captured glyph = %+v, want X=102 Y=203", calls[0])
	}

	// The window is closed: a Draw after endTextCapture is not recorded.
	f.Draw(pic, "A", 0, 0, smoothingTestColor)
	if n := text.CapturedLen(); n != 1 {
		t.Fatalf("Captured() len after the window closed = %d, want still 1", n)
	}
}

// lazyDialogueTown composes its own dialogue picture fresh on every
// TownDialogue() call, the shape a real TownScreen uses (townscreen.go's
// TownDialogue composes via ui.RenderNotice on every call rather than
// caching a picture built before the caller's own capture window opened) —
// unlike fakeTownDialogue, whose fixed pic field is built once, outside any
// window, by every other test in this package.
type lazyDialogueTown struct {
	fakeTown
	font *text.Font
}

func (f *lazyDialogueTown) TownDialogue() (*image.RGBA, bool) {
	pic := image.NewRGBA(image.Rect(0, 0, 40, 30))
	draw.Draw(pic, pic.Rect, image.NewUniform(color.RGBA{A: 0xff}), image.Point{}, draw.Src)
	f.font.Draw(pic, "A", 3, 5, smoothingTestColor)
	return pic, true
}
func (f *lazyDialogueTown) TownDialogueButton() (image.Rectangle, bool) {
	return image.Rect(0, 0, 40, 30), true
}
func (f *lazyDialogueTown) AdvanceTownDialogue() TownAction { return TownAction{} }

// TestOverlayTownDialogueShiftsCapturedGlyphsIntoFrameSpace exercises the
// exact coordinate-space fix round-1's own prototype found necessary for
// this site (app.go's overlayTownDialogue doc comment): a dialogue composes
// its own small picture, so a glyph Draw places while composing it carries
// THAT picture's local coordinates. Called inside an outer capture window —
// the shape App.Draw itself opens around its whole switch — the glyph must
// land at dialogueOrigin's own offset in the destination frame, the same
// offset the picture's own pixels are pasted at.
func TestOverlayTownDialogueShiftsCapturedGlyphsIntoFrameSpace(t *testing.T) {
	town := &lazyDialogueTown{font: smoothingTestFont()}
	pix := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))

	// SetCapture(false): the raster is not suppressed, so the pasted
	// picture's own pixels can be read back and compared against the
	// captured glyph's own position below. App.Draw's own window always
	// requests skip-raster (skipRaster, capture.go's own contract); that
	// behaviour is capture_test.go's own TestCaptureSkipRasterRecordsWithoutPainting,
	// not this site's coordinate-space claim.
	text.ResetCapture()
	text.SetCapture(false)
	defer text.StopCapture()

	overlayTownDialogue(pix, town)
	calls := text.Captured()

	// dialogueOrigin depends only on the picture's own fixed size (40x30,
	// above), so this is safe to compute from a freshly built rectangle
	// rather than calling TownDialogue() again, which would compose (and
	// capture) a second picture while the deferred StopCapture is still
	// pending.
	ox, oy := dialogueOrigin(image.NewRGBA(image.Rect(0, 0, 40, 30)))
	if len(calls) != 1 {
		t.Fatalf("Captured() len = %d, want 1", len(calls))
	}
	if calls[0].X != 3+ox || calls[0].Y != 5+oy {
		t.Fatalf("captured glyph = %+v, want X=%d Y=%d (3+%d, 5+%d)", calls[0], 3+ox, 5+oy, ox, oy)
	}

	// The picture's own pixels landed at the identical offset, so the
	// captured glyph and the pasted art agree about where the dialogue sits.
	if got := pix.RGBAAt(3+ox, 5+oy); got != smoothingTestColor {
		t.Fatalf("pasted pixel at the captured glyph's own position = %v, want %v", got, smoothingTestColor)
	}
}

// TestDrawCharacterPaneBodyShiftsStatisticsCardGlyphs is
// missionCardPresent's own site: the statistics card composes on
// RenderCharacterPanel's own small canvas and is pasted onto the pane's
// destination at r.Min.Sub(b.Min) (drawCharacterPaneBody, townshell.go). A
// capture window open around the whole call must see the card's captured
// glyph land at the SAME position as the pasted pixel it belongs to, the
// same proof TestOverlayTownDialogueShiftsCapturedGlyphsIntoFrameSpace
// already gives the town dialogue's own nesting. A non-empty PaneRect is
// required for this test to be able to catch a missing shift at all: at
// PaneRect's zero value the paste offset is itself zero and an unshifted
// bug would read identically to a fixed one.
func TestDrawCharacterPaneBodyShiftsStatisticsCardGlyphs(t *testing.T) {
	text.ResetCapture()
	text.SetCapture(false)
	defer text.StopCapture()

	f := smoothingTestFont()
	dst := image.NewRGBA(image.Rect(0, 0, 200, 200))
	view := TownCharacterView{
		HasSubject: true,
		Statistics: true,
		PaneRect:   image.Rect(70, 90, 230, 332),
		Font:       f,
		CardFont:   f,
		Subject:    PanelSubject{Name: "A"},
	}
	drawCharacterPaneBody(dst, view)

	// The card's other rows draw plenty of captured calls of their own
	// (numeric fields, headings) through smoothingTestFont's UNINKED
	// records, which paint nothing at all; only the 'A' glyph identifies
	// calls this test can check a real painted pixel against. Each call's
	// OWN Color is the right comparison, not one fixed colour: composePanel
	// colours a row's text by field (CompactPanelLayout gives PanelFieldName
	// its own LabelColor rather than the caller's ValueColor), so the
	// card's several 'A' occurrences legitimately paint in more than one
	// colour.
	aGlyph := &f.Glyphs['A'-text.FirstChar]
	found := false
	for _, c := range text.Captured() {
		if c.Glyph != aGlyph {
			continue
		}
		found = true
		if got := dst.RGBAAt(c.X, c.Y); got != c.Color {
			t.Fatalf("pasted pixel at captured glyph %+v = %v, want %v (the call's own recorded "+
				"colour): a captured position that does not land on its own painted pixel is still "+
				"in the card's local, pre-paste space", c, got, c.Color)
		}
	}
	if !found {
		t.Fatal("Captured() holds no call for the 'A' glyph: the subject's Name never reached the card")
	}
}

// TestTextSmoothingOffOpensNoCaptureWindow proves the option's off state at
// the mechanism level: Draw with SetTextSmoothing(false) never touches the
// text package's capture state at all (no ResetCapture/SetCapture/
// StopCapture call), so text.Font.Draw's own contract — capturing off draws
// exactly the pixels it always has, and records nothing — applies to the
// WHOLE frame unconditionally. textOverlay.draw is correspondingly never
// invoked: the retained overlay texture stays nil rather than being
// allocated and left blank.
func TestTextSmoothingOffOpensNoCaptureWindow(t *testing.T) {
	town := &lazyDialogueTown{font: smoothingTestFont()}

	a := newTestApp(t, appRows(3), okLoader(t))
	a.SetTown(town)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the dialogue town")
	}
	a.canvas = ebiten.NewImage(frame.W, frame.H)

	text.ResetCapture()
	a.SetTextSmoothing(false)
	a.Draw(ebiten.NewImage(frame.W, frame.H))

	if n := text.CapturedLen(); n != 0 {
		t.Fatalf("Captured() len after a smoothing-off Draw = %d, want 0: the switch must open no window at all", n)
	}
	if a.textOverlay.tex != nil {
		t.Fatal("textOverlay.tex was allocated with smoothing off, want it left nil")
	}

	a.SetTextSmoothing(true)
	a.Draw(ebiten.NewImage(frame.W, frame.H))
	if a.textOverlay.tex == nil {
		t.Fatal("textOverlay.tex stayed nil with smoothing on and a real dialogue glyph present, want it allocated")
	}
}
