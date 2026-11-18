package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// fileDesc is a two-page window: a file row on the shown second page, a
// paragraph in the right column and a sheet below.
const fileDesc = `{
  "window": "mod",
  "title": "editor.mod.window",
  "width": 900,
  "columns": 2,
  "tabs": ["editor.mod.missions", "editor.mod.overview"],
  "tab": 1,
  "rows": [
    {"kind": "file", "label": "editor.mod.output"},
    {"kind": "text", "label": "editor.mod.overview.empty", "wrap": true, "column": 1}
  ],
  "canvas": {"picture": "record:1"}
}`

// A description's file row is a pick row whose button and empty value say
// what every file row says; its tab is the shown one; its wrapped paragraph
// is one text row a line in the column's width; its canvas is the named
// picture below the rows.
func TestDescriptionFileRowTabParagraphAndCanvas(t *testing.T) {
	m, err := newEditorMockup([]byte(fileDesc), "en", image.Pt(1920, 1080), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	word := words.For("en").Text
	file := m.plain.Rows[0]
	if file.Kind != popupPick || len(file.Buttons) != 1 || file.Buttons[0] != word("editor.pick.choose") || file.Hint != word("editor.pick.none") {
		t.Fatalf("file row: kind %v buttons %q hint %q", file.Kind, file.Buttons, file.Hint)
	}
	if m.plain.ActiveTab != 1 || !m.plain.TabSwitch {
		t.Errorf("active tab %d, switch %v; want 1, true", m.plain.ActiveTab, m.plain.TabSwitch)
	}
	var lines []string
	for _, r := range m.plain.Rows[1:] {
		if r.Column != 1 || r.Kind != popupText {
			t.Fatalf("paragraph row %+v", r)
		}
		lines = append(lines, strings.TrimSpace(r.Label))
	}
	if len(lines) < 2 || strings.Join(lines, " ") != word("editor.mod.overview.empty") {
		t.Errorf("paragraph lines %q", lines)
	}
	for _, r := range m.layout.Rows {
		if r.Index == 0 {
			continue
		}
		if w := m.body.Advance(m.paint.Rows[r.Index].Label); w > r.Rect.Dx() {
			t.Errorf("paragraph line %d is %d wide in a row of %d", r.Index, w, r.Rect.Dx())
		}
	}
	if m.plain.Canvas == nil || m.layout.Canvas.Empty() {
		t.Errorf("canvas %v at %v", m.plain.Canvas, m.layout.Canvas)
	}
}
