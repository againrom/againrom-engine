package ui

import (
	"bytes"
	"image"
	"strings"
	"testing"
)

// questionDesc is a question with every part: a help, a row with its own
// help, a note per list item and a note whose number fills a %s verb, a
// list, a grid with a picture row, and two buttons.
const questionDesc = `{
  "window": "probe",
  "question": true,
  "title": "editor.newmap.title",
  "titleHelp": "editor.newmap.title.help",
  "rows": [{"kind": "edit", "label": "editor.newmap.height", "help": "editor.area.newtitle.help", "value": "typed", "focus": true}],
  "notes": [
    {"word": "editor.newmap.badheight", "args": ["typed"]},
    {"each": "warn", "bind": "line"}
  ],
  "noteInk": "warning",
  "list": {"each": "groups", "bind": "text", "chosen": "chosen", "rows": 2},
  "grid": {
    "each": "cards", "colHead": "name", "rowHeads": ["editor.testparty.plays", "editor.testparty.spells"],
    "column": [{"picture": "pic", "label": "name", "tip": "name", "chosen": "on"}, {"chosen": "on", "inert": "!caster"}],
    "side": 60, "high": 40
  },
  "buttons": [{"word": "editor.mode.single"}, {"word": "editor.button.cancel"}],
  "sample": {
    "values": {"typed": "130"},
    "lists": {
      "warn": [{"values": {"line": "Start point: cell 10,52: the party cannot land there"}}],
      "groups": [{"values": {"text": "New (group 10)"}}, {"values": {"text": "Group 1: 1 units"}, "flags": ["chosen"]}, {"values": {"text": "Group 7: 4 units"}}],
      "cards": [{"values": {"name": "Hero"}, "flags": ["on"]}, {"values": {"name": "Reniesta", "pic": "record:1"}, "flags": ["caster"]}]
    }
  }
}`

func TestQuestionDescriptionBuildsEveryPart(t *testing.T) {
	m, err := newEditorMockup([]byte(questionDesc), "en", image.Pt(1920, 1080), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	s := m.plain
	if !strings.HasSuffix(s.TitleHelp, "Enter: Single-player. Esc: cancel.") || !strings.HasPrefix(s.TitleHelp, "In a single-player map") {
		t.Fatalf("title help %q", s.TitleHelp)
	}
	notes := strings.Join(s.Notes, "")
	if !strings.HasPrefix(notes, "130 is not a height from 0 to 126") || !strings.Contains(notes, "the party cannot land there") || s.NoteInk != editorWarning {
		t.Fatalf("notes %q ink %v", s.Notes, s.NoteInk)
	}
	if s.List == nil || s.List.Len() != 3 || s.List.Selection() != 1 || s.ListRows != 2 {
		t.Fatalf("list %+v rows %d", s.List, s.ListRows)
	}
	g := s.Grid
	if g == nil || g.Cols != 2 || g.Rows != 2 || len(g.Cells) != 4 || g.Cells[0].Label != "Hero" || g.Cells[1].Icon == nil || !g.Cells[0].Ticked || !g.Cells[2].Inert || g.Cells[3].Inert {
		t.Fatalf("grid %+v", g)
	}
	if !s.Rows[0].Focus || m.paint.Rows[0].Caret == 0 {
		t.Fatalf("focused row %+v", m.paint.Rows[0])
	}
	page := m.Page()
	for part, want := range map[string]int{`data-part="list-row"`: 2, `data-part="cell"`: 4, `data-part="column-head"`: 2, `data-part="grid-head"`: 2, `data-part="row-help"`: 1} {
		if n := bytes.Count(page, []byte(part)); n != want {
			t.Errorf("%s: %d on the page, want %d", part, n, want)
		}
	}
}

func TestQuestionStandsWholeInTheMiddle(t *testing.T) {
	for _, window := range []image.Point{{1920, 1080}} {
		m, err := newEditorMockup([]byte(questionDesc), "en", window, wordsFont("en"), wordsFont("en"), nil, mockupPerson{})
		if err != nil {
			t.Fatal(err)
		}
		l := m.layout
		if !l.Rect.In(image.Rectangle{Max: window}) || l.Rect.Min.X != (window.X-l.Rect.Dx())/2 {
			t.Fatalf("%v: window %v not whole in the middle", window, l.Rect)
		}
		boxes := append([]image.Rectangle{l.Title, l.Close, l.TitleHelp, l.Grid}, l.Notes...)
		boxes = append(append(boxes, l.Buttons...), l.GridCells...)
		for _, rr := range l.Rows {
			boxes = append(boxes, rr.Label, rr.Control, rr.Help)
		}
		for _, b := range boxes {
			if !b.Empty() && !b.In(l.Rect) {
				t.Errorf("%v: %v leaves the window %v", window, b, l.Rect)
			}
		}
	}
}
