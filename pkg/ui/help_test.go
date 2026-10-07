package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// helpFixtureText is synthetic CRLF paragraph text; it carries no install text.
func helpFixtureText(paragraphs int) string {
	var b strings.Builder
	for i := 0; i < paragraphs; i++ {
		fmt.Fprintf(&b, "Paragraph %d of the fixture text with enough words to wrap across the body\r\n", i)
	}
	return b.String()
}

func helpApp(t *testing.T, paragraphs int) (*App, *noticeSeam) {
	t.Helper()
	a, seam := noticeApp(t)
	w := a.flow.viewer.Words()
	w.HelpText = helpFixtureText(paragraphs)
	a.flow.viewer.SetWords(w)
	return a, seam
}

// F1 opens the panel on the map, it holds the world through the popup answer,
// and OK, Return and Esc close it without asking the mission driver.
func TestHelpKeyOpensAndEveryDismissalCloses(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, tc := range []struct {
		name string
		in   appInput
	}{
		{"RETURN", appInput{Enter: true}},
		{"ESCAPE", appInput{Escape: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, seam := helpApp(t, 60)
			v := a.flow.viewer
			a.step(appInput{Help: true}, now)
			if !v.HelpOpen() || !a.flow.popupOpen() {
				t.Fatal("F1 did not open a help panel that holds the world")
			}
			if _, kind, _ := v.NoticeState(); kind != NoticeDialogue {
				t.Errorf("help notice kind = %v", kind)
			}
			l := v.noticeLayout()
			if l.Box != image.Rect(76, 60, 564, 420) {
				t.Errorf("help box = %v, want 488x360 at (76,60)", l.Box)
			}
			a.step(appInput{Help: true}, now)
			if _, _, open := v.NoticeState(); !open || !v.HelpOpen() {
				t.Fatal("F1 over the help panel closed or replaced it")
			}
			a.step(tc.in, now)
			if v.HelpOpen() || a.flow.popupOpen() {
				t.Fatal("the dismissal did not close the help panel")
			}
			if seam.advances != 0 {
				t.Errorf("%d advances through the MapAdvance seam, want 0", seam.advances)
			}
		})
	}
}

// F1 is ignored over every other popup and when the install states no help.
func TestHelpKeyIgnoredOverPopupsAndWithoutText(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, seam := helpApp(t, 60)
	seam.v.SetNotice("the mission's own words", NoticeDialogue)
	a.step(appInput{Help: true}, now)
	if text, _, _ := seam.v.NoticeState(); text != "the mission's own words" || seam.v.HelpOpen() || a.flow.selfNotice {
		t.Errorf("F1 replaced a standing notice: %q", text)
	}

	a, _ = helpApp(t, 60)
	a.flow.viewer.menuUp = true
	a.step(appInput{Help: true}, now)
	if a.flow.viewer.HelpOpen() {
		t.Error("F1 opened help over the in-game menu popup")
	}

	a, _ = helpApp(t, 0)
	a.step(appInput{Help: true}, now)
	if _, _, open := a.flow.viewer.NoticeState(); open {
		t.Error("an empty help text opened a panel")
	}
}

// The body scrolls by line and page, clamps at both ends, and repaints.
func TestHelpScrollsAndClamps(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 60)
	v := a.flow.viewer
	a.step(appInput{Help: true}, now)
	lines, visible := v.HelpLines()
	if lines <= visible || visible <= 0 {
		t.Fatalf("fixture does not overflow: %d lines, %d visible", lines, visible)
	}
	if v.noticeLayout().Scrollbar.Empty() {
		t.Fatal("overflowing text has no scroll bar")
	}
	if first, last := v.HelpScroll(); first != 0 || last != lines-visible {
		t.Fatalf("scroll %d of %d, want 0 of %d", first, last, lines-visible)
	}
	pic0, _, _, _ := v.noticePresent()
	before := append([]byte(nil), pic0.Pix...)

	a.step(appInput{Up: true}, now)
	if first, _ := v.HelpScroll(); first != 0 {
		t.Errorf("Up at the top moved to %d", first)
	}
	// The current line starts at -1: the first Down only resyncs to 0.
	a.step(appInput{Down: true}, now)
	if first, _ := v.HelpScroll(); first != 0 {
		t.Errorf("first Down moved to %d, want 0", first)
	}
	a.step(appInput{Down: true}, now)
	if first, _ := v.HelpScroll(); first != 1 {
		t.Errorf("second Down moved to %d, want 1", first)
	}
	pic1, _, _, _ := v.noticePresent()
	if string(pic1.Pix) == string(before) {
		t.Error("scrolling did not repaint the panel")
	}
	a.step(appInput{PageDown: true}, now)
	if first, _ := v.HelpScroll(); first != 1+visible-1 {
		t.Errorf("Page Down moved to %d, want %d", first, visible)
	}
	for i := 0; i < 20; i++ {
		a.step(appInput{PageDown: true}, now)
	}
	if first, last := v.HelpScroll(); first != last {
		t.Errorf("Page Down stopped at %d, want the last %d", first, last)
	}
	a.step(appInput{PageUp: true}, now)
	if first, last := v.HelpScroll(); first != last-visible {
		t.Errorf("Page Up from the end moved to %d, want %d", first, last-visible)
	}
	if lines, _ := v.HelpLines(); lines != v.helpGeo().lines {
		t.Error("line count changed while scrolling")
	}
}

// A press on the scroll bar's down arrow and track scrolls; a press elsewhere
// in the panel does not.
func TestHelpScrollbarPress(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 60)
	v := a.flow.viewer
	a.step(appInput{Help: true}, now)
	l := v.noticeLayout()
	_, at, scale, ok := v.noticePresent()
	if !ok {
		t.Fatal("no help picture")
	}
	lines, visible := v.HelpLines()
	_, down, track, _ := helpBarParts(l.Scrollbar, 0, lines, visible)
	window := func(r image.Rectangle) (int, int) {
		c := r.Min.Add(r.Max).Div(2)
		return at.X + int(float64(c.X)*scale), at.Y + int(float64(c.Y)*scale)
	}
	x, y := window(down)
	a.step(appInput{PrimaryPressed: true, CursorX: x, CursorY: y}, now)
	if first, _ := v.HelpScroll(); first != 1 {
		t.Errorf("down arrow moved to %d, want 1", first)
	}
	x, y = window(image.Rect(track.Min.X, track.Max.Y-2, track.Max.X, track.Max.Y))
	a.step(appInput{PrimaryPressed: true, CursorX: x, CursorY: y}, now)
	if first, _ := v.HelpScroll(); first != 1+visible {
		t.Errorf("track below the thumb moved to %d, want %d", first, 1+visible)
	}
	if !v.HelpOpen() {
		t.Error("a scroll bar press closed the panel")
	}
}

// Text that fits has no scroll bar and nothing to scroll (loss control).
func TestHelpShortTextHasNoScrollbar(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 2)
	v := a.flow.viewer
	a.step(appInput{Help: true}, now)
	if !v.HelpOpen() || !v.noticeLayout().Scrollbar.Empty() {
		t.Fatal("short help text drew a scroll bar")
	}
	a.step(appInput{PageDown: true}, now)
	if first, last := v.HelpScroll(); first != 0 || last != 0 {
		t.Errorf("short text scrolled: %d of %d", first, last)
	}
}

// C: empty selection is not consumed, a selection without a caster is consumed
// with no state change, a caster arms Cast mode and chooses no spell, and a
// foreign selection is not consumed.
func TestCastKeyFollowsTheSelection(t *testing.T) {
	_, v := mkOnMap(t)
	v.SetEntities([]MapEntity{
		{ID: 4, Cell: image.Pt(3, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true},
		{ID: 5, Cell: image.Pt(4, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
		{ID: 7, Cell: image.Pt(5, 3), Life: LifeAlive, Owner: 2, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
	})
	v.selectedSpell = 0

	v.sel = nil
	if v.castKey() || v.spellArmed {
		t.Error("an empty selection consumed C or armed Cast")
	}

	v.sel = selection{4}
	if !v.castKey() {
		t.Error("a nonempty selection with no caster did not consume C")
	}
	if v.spellArmed || v.missionMode() != modeNone || v.selectedSpell != 0 || v.cmdOverlayHidden {
		t.Error("the refused C changed state")
	}

	v.sel = selection{7}
	if v.castKey() || v.spellArmed {
		t.Error("a foreign selection consumed C or armed Cast")
	}

	v.sel = selection{4, 5}
	if !v.castKey() || v.spellArmed || v.missionMode() == modeCast {
		t.Fatal("C with no spell selected was not a no-op")
	}
	v.selectedSpell = 1
	if !v.castKey() || !v.spellArmed || v.missionMode() != modeCast {
		t.Fatal("a caster in the selection with a spell selected did not arm Cast")
	}
}

// A mode armed by C survives a closed book, where a spell chosen in the book
// does not (MENU-055: the mode is the key's, the spell is the book's).
func TestCastKeyArmedModeNeedsNoBook(t *testing.T) {
	_, v := mkOnMap(t)
	v.SetEntities([]MapEntity{
		{ID: 5, Cell: image.Pt(4, 3), Life: LifeAlive, Owner: 1, SpellStateKnown: true, KnownSpells: 1 << 1, CastCapable: true},
	})
	v.sel = selection{5}
	v.spellArmed, v.spellNeedsBook, v.selectedSpell = true, true, 1
	if v.hudShown(hudPanelBook) {
		v.toggleHudPanel(hudPanelBook)
	}
	if v.hudShown(hudPanelBook) || v.spellModeLive() {
		t.Fatal("control: a book-chosen spell is live with the book closed")
	}
	v.castKey()
	if !v.spellModeLive() {
		t.Error("a mode armed by C is not live with the book closed")
	}
}
