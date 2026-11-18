package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

func triggerRows(n int) []PickerRow {
	rows := make([]PickerRow, n)
	for i := range rows {
		rows[i] = PickerRow{Text: fmt.Sprint("trigger ", i), Choosable: true}
	}
	return rows
}

func TestOpenDropdownChoosesARowAboveItsAnchor(t *testing.T) {
	e := NewMapEditor(nil, nil, nil)
	e.SetBodyFont(wordsFont("en"))
	e.Layout(1920, 1080)
	e.openPrompt(editorPrompt{kind: promptAreaTrigger, area: &areaPrompt{box: image.Rect(1, 1, 4, 4), choice: NewPicker(triggerRows(40)), open: true}})
	titleDragBy(t, e, image.Pt(0, 430))
	d, ok := e.promptDropdown()
	if !ok || !d.Open {
		t.Fatal("the area prompt shows no open trigger list")
	}
	f := e.bodyFont()
	frame := d.listFrame(f)
	if !frame.In(image.Rectangle{Max: e.window}) || frame.Max.Y > d.Rect.Min.Y {
		t.Fatalf("list frame %v is not inside the window above %v", frame, d.Rect)
	}
	top, _ := d.List.Visible()
	row := top + 1
	p := center(d.listBox(f).Row(row - top))
	editorClickAt(e, p, time.Unix(1, 0))
	after, _ := e.promptDropdown()
	if after.Open || after.List.Selection() != row {
		t.Fatalf("row %d left the list open=%v with selection %d", row, after.Open, after.List.Selection())
	}
	if part, _ := after.hit(f, center(after.Rect)); part != dropdownToggle {
		t.Fatalf("closed choice hit %v", part)
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
