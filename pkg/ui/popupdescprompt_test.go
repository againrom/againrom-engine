package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

// promptDesc is a prompt window: a typed field, the keys line after the
// title help, a warning note longer than the window and a picture grid of
// two cells, the first chosen.
const promptDesc = `{
  "window": "probe",
  "title": "editor.prompt.recipe",
  "titleHelp": "editor.prompt.recipe.help",
  "titleHelpMore": ["editor.prompt.keys"],
  "width": 300,
  "labelWidth": 1,
  "rows": [{"kind": "edit", "value": "text", "focus": true}],
  "grid": {"each": "items", "picture": "icon", "tip": "name", "chosen": "chosen", "rows": 1},
  "notes": [{"word": "editor.path.dialogfailed", "args": ["cause"]}],
  "noteInk": "warning",
  "buttons": [{"word": "editor.button.ok"}],
  "sample": {
    "values": {"text": "12", "cause": "the dialog program was not found on this computer"},
    "lists": {"items": [
      {"values": {"icon": "record:1", "name": "first"}, "flags": ["chosen"]},
      {"values": {"icon": "item:2", "name": "second"}}
    ]}
  }
}`

func TestDescriptionPromptHelpNotesGridAndFocus(t *testing.T) {
	m, err := newEditorMockup([]byte(promptDesc), "en", image.Pt(800, 600), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	if want := "Choose the item of this sack that gets a magic recipe. Enter: OK. Esc: cancel."; m.plain.TitleHelp != want {
		t.Errorf("title help %q, not %q", m.plain.TitleHelp, want)
	}
	whole := "The system file dialog failed: the dialog program was not found on this computer."
	if len(m.plain.Notes) < 2 || strings.Join(m.plain.Notes, "") != whole || m.plain.NoteInk != editorWarning {
		t.Errorf("notes %q in %v: the long note is not wrapped in the warning ink", m.plain.Notes, m.plain.NoteInk)
	}
	row := m.paint.Rows[0]
	if !row.Focus || row.Caret <= 0 {
		t.Errorf("edit row focus %v caret %d", row.Focus, row.Caret)
	}
	g := m.plain.Grid
	if g == nil || len(g.Cells) != 2 || !g.Cells[0].Pressed || g.Cells[1].Pressed || g.Cells[0].Tip != "first" {
		t.Fatalf("grid %+v", g)
	}
	if g.Cells[0].Icon == nil || g.Cells[1].Icon != nil {
		t.Errorf("cell pictures %v %v: only record:1 resolves", g.Cells[0].Icon != nil, g.Cells[1].Icon != nil)
	}
	if len(m.layout.GridCells) != 2 || len(m.layout.Notes) != len(m.plain.Notes) {
		t.Fatalf("layout holds %d cells and %d notes", len(m.layout.GridCells), len(m.layout.Notes))
	}
	for _, r := range append(append([]image.Rectangle{}, m.layout.GridCells...), m.layout.Notes...) {
		if !r.In(m.layout.Rect) {
			t.Errorf("%v lies outside the window %v", r, m.layout.Rect)
		}
	}
	page := m.Page()
	if n := bytes.Count(page, []byte(`data-part="cell"`)); n != 2 {
		t.Errorf("the page holds %d cells, not 2", n)
	}
	for _, want := range []string{`title="first"`, `title="second"`, `data-part="note"`} {
		if !bytes.Contains(page, []byte(want)) {
			t.Errorf("the page holds no %s", want)
		}
	}
}
