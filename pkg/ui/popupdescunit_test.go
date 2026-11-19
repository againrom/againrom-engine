package ui

import (
	"bytes"
	"html"
	"image"
	"os"
	"testing"

	"againrom/pkg/words"
)

func testdataDesc(name string) (popupDesc, error) {
	data, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		return popupDesc{}, err
	}
	return parsePopupDesc(data)
}

// The unit window's compact detailed view stands its settings in three
// columns: who the unit is, then health, mana and attributes, then defence,
// speed and sight. Every override field explains itself behind a (?) and
// shows the value in game in grey while it inherits; weapon skills,
// protections and equipment are blocks that stay closed until opened. The
// whole window shows without a scroll bar in both languages at both sizes.
func TestUnitCompactViewGroupsFieldsInColumns(t *testing.T) {
	d, err := testdataDesc("unitcompact")
	if err != nil {
		t.Fatal(err)
	}
	column := map[string]int{
		"editor.unit.group.vitals": 1, "editor.unit.group.attributes": 1, "editor.unit.group.skills": 1,
		"editor.unit.group.defence": 2, "editor.unit.group.motion": 2, "editor.unit.group.protections": 2,
		"editor.unit.group.equipment": 0,
	}
	folds := []string{"editor.unit.group.skills", "editor.unit.group.protections", "editor.unit.group.equipment"}
	for _, lang := range []string{"en", "ru"} {
		word := words.For(lang).Text
		font := wordsFont(lang)
		for _, size := range []image.Point{{1920, 1080}, {1280, 720}} {
			m, err := NewEditorMockup(d.encode(), lang, size, font, font, nil)
			if err != nil {
				t.Fatal(err)
			}
			s, l := m.plain, m.layout
			if len(l.Rows) != len(s.Rows) || !l.RowBar.Empty() || !l.Rect.In(image.Rectangle{Max: size}) {
				t.Errorf("%s %v: %d of %d rows show in %v; the whole window shows", lang, size, len(l.Rows), len(s.Rows), l.Rect)
			}
			heads := map[string]popupRow{}
			for _, rr := range l.Rows {
				row := s.Rows[rr.Index]
				for id := range column {
					if row.Kind == popupHeading && row.Label == word(id) {
						heads[id] = row
					}
				}
				if row.Column > 0 && row.Kind == popupEdit && (rr.Help.Empty() || row.Value == "" && row.Hint == "") {
					t.Errorf("%s %v: %q shows no (?) or no grey value in game", lang, size, row.Label)
				}
			}
			left := map[int]int{}
			for _, rr := range l.Rows {
				c := s.Rows[rr.Index].Column
				if x, ok := left[c]; !ok || rr.Rect.Min.X < x {
					left[c] = rr.Rect.Min.X
				}
			}
			if size.X == 1920 && !(left[0] < left[1] && left[1] < left[2]) || !(left[0] < left[1]) {
				t.Errorf("%s %v: the columns start at %v; the field columns stand left to right", lang, size, left)
			}
			for id, c := range column {
				if row, ok := heads[id]; !ok || min(row.Column, s.Columns-1) != min(c, s.Columns-1) {
					t.Errorf("%s %v: the %s group is not a heading of column %d", lang, size, id, c)
				}
			}
			for _, id := range folds {
				if heads[id].Fold != FoldClosed {
					t.Errorf("%s %v: the %s block does not start closed", lang, size, id)
				}
			}
			help := ` title="` + html.EscapeString(word("editor.unit.zero.refused")) + `"`
			if page := m.Page(); !bytes.Contains(page, []byte(help)) {
				t.Errorf("%s %v: the page's (?) marks do not carry their explanation", lang, size)
			}
		}
	}
	open := d
	sample := *d.Sample
	sample.Flags = append([]string{"skills", "protections"}, sample.Flags...)
	open.Sample = &sample
	closed, _ := d.build(d.Sample, words.For("en").Text)
	opened, _ := open.build(open.Sample, words.For("en").Text)
	if got := len(opened.Rows) - len(closed.Rows); got != 10 {
		t.Errorf("opening skills and protections adds %d rows, not their 10 fields", got)
	}
}

// The compact view stands the person's card and figure in their own block
// right of the field columns, as the unit window does: the card the game
// draws at the top, the figure at the game's size under it, and the whole
// window inside the editor at both sizes, three field columns at 1920.
func TestUnitCompactViewStandsCardAndFigureBesideColumns(t *testing.T) {
	d, err := testdataDesc("unitcompact")
	if err != nil {
		t.Fatal(err)
	}
	doll := image.NewRGBA(image.Rect(0, 0, 96, 150))
	for _, lang := range []string{"en", "ru"} {
		font := wordsFont(lang)
		for _, size := range []image.Point{{1920, 1080}, {1280, 720}} {
			m, err := newEditorMockup(d.encode(), lang, size, font, font, nil, mockupPerson{doll: doll})
			if err != nil {
				t.Fatal(err)
			}
			s, l := m.paint, m.layout
			room := figureRoom(s, l, m.figure)
			window := image.Rectangle{Max: size}
			if s.Card == nil || s.Card.Name != "Grok" || s.Card.Char.Skills[2] != 30 || l.Card.Dx() != compactPanelW {
				t.Errorf("%s %v: the card %+v stands at %v; the sample's card shows", lang, size, s.Card, l.Card)
			}
			if m.figure != gameFigureSize(doll, size) || room.Dx() < m.figure.X || room.Dy() < m.figure.Y || room.Min.Y < l.Card.Max.Y {
				t.Errorf("%s %v: the figure %v has the room %v under the card %v", lang, size, m.figure, room, l.Card)
			}
			if !l.Rect.In(window) || !l.Card.In(l.Rect) || !room.In(l.Rect) || len(l.Rows) != len(s.Rows) || !l.RowBar.Empty() {
				t.Errorf("%s %v: the window %v shows %d of %d rows, card %v, figure %v", lang, size, l.Rect, len(l.Rows), len(s.Rows), l.Card, room)
			}
			for _, rr := range l.Rows {
				if rr.Rect.Max.X > min(l.Card.Min.X, room.Min.X) {
					t.Errorf("%s %v: row %q at %v reaches the card block", lang, size, s.Rows[rr.Index].Label, rr.Rect)
				}
			}
			if size.X == 1920 && s.Columns != 3 || s.Width > size.X {
				t.Errorf("%s %v: %d field columns in a window %d wide", lang, size, s.Columns, s.Width)
			}
			page := m.Page()
			if !bytes.Contains(page, []byte(`data-part="card"`)) || !bytes.Contains(page, []byte(`data-part="figure"`)) {
				t.Errorf("%s %v: the page has no card or figure part", lang, size)
			}
			if pic := m.Picture(); pic.Bounds().Size() != size {
				t.Errorf("%s %v: the picture is %v", lang, size, pic.Bounds())
			}
		}
	}
}
