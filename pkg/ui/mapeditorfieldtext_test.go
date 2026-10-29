package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/render/text"
)

// outsideGlyphs are the recorded glyphs whose cells lie wholly left or right
// of the image they were drawn into: characters a field places where no
// pixel of them shows.
func outsideGlyphs(calls []text.DrawCall) []int {
	var out []int
	for _, c := range calls {
		if c.Glyph == nil || c.Clip.Empty() {
			continue
		}
		if c.X+c.Glyph.Width <= c.Clip.Min.X || c.X >= c.Clip.Max.X {
			out = append(out, c.X)
		}
	}
	return out
}

// An edit or pick field whose value is longer than the field shows it to
// its end: unfocused, the value ends in an ellipsis inside the field; focused
// with the caret at the end, the text moves left and only the characters the
// field shows are drawn. Fails when a field draws the whole value into its
// clip, leaving characters wholly outside it.
func TestEditorFieldDrawsNoCharacterOutsideIt(t *testing.T) {
	f := panelFont()
	long := strings.Repeat("88,167  1x1 ", 4)
	s := popupFixture()
	s.Rows[2].Focus = false
	s.Rows[2].Value = long
	s.Rows[4].Value = long
	l := layoutPopup(s, f, f)
	pic := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	calls := text.Record(func() {
		drawPopup(pic, f, f, nil, s, l, popupInk{Ground: editorPanel, Edge: editorGold, TitleGround: editorInk, Title: editorGold, Label: editorPaper, Value: editorLight}, image.Point{}, false)
	})
	if out := outsideGlyphs(calls); len(out) != 0 {
		t.Errorf("unfocused edit and pick fields draw %d characters outside them, at x %v", len(out), out)
	}
	box := image.Rect(10, 10, 90, 30)
	v := &editorTextView{text: long, caret: len([]rune(long)), anchor: len([]rune(long)), span: editorTextSpan{0, len([]rune(long))}, encode: func(s string) string { return s }}
	row := popupRow{Kind: popupEdit, Value: long, Focus: true, Edit: v}
	calls = text.Record(func() {
		drawEditorInput(image.NewRGBA(image.Rect(0, 0, 100, 40)), f, row, box, editorPopupInk, editorLight)
	})
	if len(calls) == 0 {
		t.Fatal("the focused field drew no character")
	}
	if out := outsideGlyphs(calls); len(out) != 0 {
		t.Errorf("the focused field draws %d characters outside it, at x %v", len(out), out)
	}
}

// A focused one-line field draws only the characters that lie wholly inside
// it, whatever offset its text moves left by. Fails when it draws a
// character cut at its left edge, of which no pixel may show, as the W
// window's Working folder field did with some folder names.
func TestEditorFocusedFieldCutsNoCharacterAtItsEdges(t *testing.T) {
	f := panelFont()
	box := image.Rect(10, 10, 90, 30)
	for n := 10; n < 40; n++ {
		long := strings.Repeat("\\02 guest-chain", 3)[:n]
		v := &editorTextView{text: long, caret: n, anchor: n, span: editorTextSpan{0, n}, encode: func(s string) string { return s }}
		row := popupRow{Kind: popupEdit, Value: long, Focus: true, Edit: v}
		calls := text.Record(func() {
			drawEditorInput(image.NewRGBA(image.Rect(0, 0, 100, 40)), f, row, box, editorPopupInk, editorLight)
		})
		for _, c := range calls {
			if c.Glyph != nil && (c.X < c.Clip.Min.X || c.X+c.Glyph.Advance > c.Clip.Max.X) {
				t.Errorf("%d characters: a character at x %d crosses the field's clip %v", n, c.X, c.Clip)
			}
		}
	}
}

// At the editor's smallest window the person window ends above the status
// strip, so the status line stays in view. Fails when the window's height
// rule takes the room under the bar down to the window's bottom.
func TestEditorPersonWindowLeavesTheStatusLine(t *testing.T) {
	e, host := popupEditor(t)
	_ = e.Open("popups")
	e.doc.Edits = &personPortraitProbe{&personSettingsProbe{&settingsPreviewProbe{popupHost: host}}}
	e.Layout(640, 480)
	e.Select(0, false)
	s, _, ok := e.objectSpec()
	if !ok || !e.nativeUnitWindow() {
		t.Fatal("the person window is not open")
	}
	if r := layoutPopup(s, e.font, e.bodyFont()).Rect; r.Max.Y > e.statusRect().Min.Y {
		t.Fatalf("the person window %v reaches into the status strip %v", r, e.statusRect())
	}
}
