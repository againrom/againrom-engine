package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// overBox is a point on a row of d's open list that is also over d's own box,
// and that row's index among the shown rows; ok false when the list does not
// cover its box.
func overBox(e *MapEditor, d dropdown) (image.Point, int, bool) {
	lb := d.listBox(e.bodyFont())
	for y := d.Rect.Min.Y; y < d.Rect.Max.Y; y++ {
		p := image.Pt(d.Rect.Min.X+40, y)
		if row, ok := lb.RowAt(p); ok && row >= 1 {
			return p, row, true
		}
	}
	return image.Point{}, 0, false
}

func triggerRows(n int) []PickerRow {
	rows := make([]PickerRow, n)
	for i := range rows {
		rows[i] = PickerRow{Text: fmt.Sprint("trigger ", i), Choosable: true}
	}
	return rows
}

// An open list is painted over its own box, so a press on a row drawn there
// chooses that row: the area prompt dragged low in a 640x480 editor with 40
// triggers, and every drop-down a choice row builds.
func TestOpenDropdownListIsHitBeforeItsBox(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.SetBodyFont(wordsFont("en"))
	e.Layout(640, 480)
	e.openPrompt(editorPrompt{kind: promptAreaTrigger, area: &areaPrompt{box: image.Rect(1, 1, 4, 4), choice: NewPicker(triggerRows(40)), open: true}})
	titleDragBy(t, e, image.Pt(0, 120))
	d, ok := e.areaDropdown()
	if !ok || !d.Open {
		t.Fatal("the area prompt shows no open trigger list")
	}
	p, row, ok := overBox(e, d)
	if !ok {
		t.Fatalf("the list %v does not cover its box %v", d.listBox(e.bodyFont()).Rect, d.Rect)
	}
	top, _ := d.List.Visible()
	editorClickAt(e, p, time.Unix(1, 0))
	after, _ := e.areaDropdown()
	if after.Open || after.List.Selection() != top+row {
		t.Errorf("a press at %v on list row %d: open %v, selection %d, want closed and %d", p, row, after.Open, after.List.Selection(), top+row)
	}

	// The object window's choice is built by the same function.
	box := image.Rect(100, 440, 400, 462)
	od := e.rowDropdown(box, "x", NewPicker(triggerRows(40)), 10, true)
	op, orow, ok := overBox(e, od)
	if !ok {
		t.Fatalf("the list %v does not cover its box %v", od.listBox(e.bodyFont()).Rect, box)
	}
	if part, got := od.hit(e.bodyFont(), op); part != dropdownRow || got != orow {
		t.Errorf("a press at %v on list row %d over the box hits part %d row %d", op, orow, part, got)
	}
	od.Open = false
	if part, _ := od.hit(e.bodyFont(), op); part != dropdownToggle {
		t.Errorf("a press on the closed box hits part %d", part)
	}
}

// Every note a window folds behind its title's (?) mark can be read inside
// the editor: the tip is kept inside it and shows a window of its lines that
// scrolls to the last.
func TestFoldedNotesTipReadsInsideTheEditor(t *testing.T) {
	f := panelFont()
	for _, size := range []image.Point{{640, 480}, {1920, 1080}} {
		for _, count := range []int{30, 60} {
			bounds := image.Rect(0, 26, size.X, size.Y)
			var notes []string
			for i := range count {
				notes = append(notes, fmt.Sprint("note line ", i, strings.Repeat(" and some more words", 6)))
			}
			s := popupSpec{Title: "Notes", Width: 400, Bounds: bounds, Rows: []popupRow{{Kind: popupText, Label: "a", Value: "b"}}, Notes: notes, Buttons: []string{"OK"}}
			l := layoutPopup(s, f, f)
			name := fmt.Sprintf("%dx%d with %d notes", size.X, size.Y, count)
			if l.TitleHelp.Empty() {
				if size.X == 640 || len(l.Notes) != count {
					t.Errorf("%s: no fold, %d notes stand", name, len(l.Notes))
				}
				continue
			}
			tip, ok := popupTipAt(image.Rect(0, 0, size.X, size.Y), f, f, s, l, center(l.TitleHelp), true)
			if !ok || !tip.Help || !tip.Rect.In(image.Rect(0, 0, size.X, size.Y)) {
				t.Fatalf("%s: tip %v ok %v leaves the editor", name, tip.Rect, ok)
			}
			read := map[string]bool{}
			for top := 0; ; {
				for _, line := range tip.scrolled(top, 0).shown() {
					read[line] = true
				}
				if tip.Rows == 0 || top >= len(tip.Lines)-tip.Rows {
					break
				}
				top = tip.scrolled(top, 3).Top
			}
			for _, line := range tip.Lines {
				if !read[line] {
					t.Errorf("%s: line %q is never shown", name, line)
				}
			}
			if !strings.HasSuffix(tip.Lines[len(tip.Lines)-1], "more words") {
				t.Errorf("%s: the tip lacks the last note", name)
			}
			if h := len(tip.shown())*fontHeight(f, 15) + 2*tipPad; h > tip.Rect.Dy() {
				t.Errorf("%s: shown lines are %d high in a box %v", name, h, tip.Rect)
			}
		}
	}
}
