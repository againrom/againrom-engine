package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// The Scenario palette at 1920x1080, EN and RU: its buttons are the drop
// zone switch and New trigger in one row inside the palette,
// Add is gone, and the hint names the keys that remain, not A.
func TestMapEditorScenarioPaletteButtonsAndHint(t *testing.T) {
	for _, lang := range []string{"english", "russian"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			book := words.For(lang)
			e.SetWords(book)
			e.Layout(1920, 1080)
			e.showRecords(editorScriptsFilter)
			s, ok := e.listSpec()
			if !ok {
				t.Fatal("the Scenario palette did not open")
			}
			want := []string{
				e.word("editor.tools.dropzones"), e.word("editor.tool.newtrigger"),
			}
			if len(s.Buttons) != 2 || !strings.Contains(s.Buttons[0], want[0]) || s.Buttons[1] != want[1] {
				t.Fatalf("buttons %q, want drop zones, %q", s.Buttons, want[1])
			}
			hint := e.word("editor.scripts.hint")
			if !slices.Contains(s.Notes, hint) || strings.Contains(book.Text("editor.scripts.hint"), "A:") {
				t.Fatalf("notes %q, hint %q", s.Notes, hint)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			if len(l.Buttons) != 2 || !l.Rect.In(image.Rect(0, 0, 1920, 1080)) {
				t.Fatalf("%d buttons in %v", len(l.Buttons), l.Rect)
			}
			for i, r := range l.Buttons {
				if !r.In(l.Rect) || i > 0 && (r.Min.X < l.Buttons[i-1].Max.X || r.Min.Y != l.Buttons[i-1].Min.Y) {
					t.Fatalf("button %d at %v in %v after %v", i, r, l.Rect, l.Buttons)
				}
			}
			for _, key := range []string{"drop", "newtrigger"} {
				if _, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: key}); !ok {
					t.Fatalf("no button key %s", key)
				}
			}
			if _, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "newvariable"}); ok {
				t.Fatal("the palette keeps New variable")
			}
			if _, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "add"}); ok {
				t.Fatal("the palette keeps Add")
			}
		})
	}
}
