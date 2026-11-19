package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// A window taller than its 640x480 bounds gives way in the layout pass, in
// order, and stays inside them: its body pages with every row in reach at
// some page, its list shows fewer rows down to its least, and its note lines
// fold behind the title's (?) mark.
func TestEditorWindowGivesWayInsideItsBounds(t *testing.T) {
	f := panelFont()
	bounds := image.Rect(0, 26, 640, 480)
	var rows []popupRow
	var notes []string
	for i := range 60 {
		rows = append(rows, popupRow{Kind: popupText, Label: fmt.Sprint("row ", i), Value: "value"})
		notes = append(notes, fmt.Sprint("note line ", i))
	}
	list := NewPicker(make([]PickerRow, 40))
	for _, c := range []struct {
		name string
		s    popupSpec
	}{
		{"rows", popupSpec{Title: "Rows", Width: 400, Bounds: bounds, Rows: rows, Buttons: []string{"OK"}}},
		{"list", popupSpec{Title: "List", Width: 400, Bounds: bounds, Rows: rows[:2], List: list, ListRows: 40, Buttons: []string{"OK"}}},
		{"notes", popupSpec{Title: "Notes", Width: 400, Bounds: bounds, Rows: rows[:2], Notes: notes, Buttons: []string{"OK"}}},
	} {
		l := layoutPopup(c.s, f, f)
		w := l.window(c.s)
		if !l.Rect.In(bounds) {
			t.Errorf("%s: %v leaves the bounds %v", c.name, l.Rect, bounds)
		}
		switch c.name {
		case "rows":
			if l.RowBar.Empty() || w.Body.MaxRows < popupMinRows {
				t.Fatalf("rows: no paged body: bar %v, %d rows", l.RowBar, w.Body.MaxRows)
			}
			seen := map[int]bool{}
			for page := 0; page <= len(rows); page++ {
				c.s.Page = page
				for _, rr := range layoutPopup(c.s, f, f).Rows {
					seen[rr.Index] = true
				}
			}
			if len(seen) != len(rows) {
				t.Errorf("rows: %d of %d rows in reach", len(seen), len(rows))
			}
		case "list":
			if l.ListRows < popupMinRows || l.ListRows >= 40 || !l.RowBar.Empty() {
				t.Errorf("list: %d list rows, bar %v", l.ListRows, l.RowBar)
			}
		case "notes":
			if len(l.Notes) != 0 || l.TitleHelp.Empty() || !strings.Contains(w.Title.Help, "note line 59") {
				t.Errorf("notes: %d lines stand, mark %v, help %q", len(l.Notes), l.TitleHelp, w.Title.Help)
			}
		}
	}
}

// The map panel's lines and the notice before test play keep inside a
// 640x480 editor: the panel's body pages and the wheel over it and its bar
// reach the last line; the notice's warnings fold behind its (?) mark.
func TestEditorPagedWindowsKeepInsideA640x480Editor(t *testing.T) {
	f, rows := endless()
	for i := range 30 {
		rows = append(rows, InspectionFinding{Rule: RuleVictory, Text: fmt.Sprint("finding ", i)})
	}
	f.Wins = []string{"one", "two", "three", "four", "five", "six", "seven", "eight"}
	p := &mapPanelEdits{facts: f, rows: rows}
	e, _ := mapPanelEditor(t, p, nil)
	e.Layout(640, 480)
	s, ok := e.listSpec()
	if !ok || e.listKind() != "panel" {
		t.Fatal("no map panel")
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if !l.Rect.In(e.popupBounds()) {
		t.Fatalf("the map panel %v leaves %v", l.Rect, e.popupBounds())
	}
	if !l.RowBar.Empty() {
		last := len(s.Rows) - 1
		at := center(l.RowsViewport)
		for range len(s.Rows) {
			e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, time.Unix(1, 0))
		}
		s, _ = e.listSpec()
		shown := false
		for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
			shown = shown || rr.Index == last
		}
		if !shown {
			t.Errorf("the wheel does not reach the panel's last line: page %d", s.Page)
		}
	}
	_ = e.Key("testplay")
	if e.prompt.kind != promptNotice {
		t.Fatal("no notice")
	}
	n, _ := e.promptSpec()
	nl := layoutPopup(n, e.font, e.bodyFont())
	if !nl.Rect.In(e.popupBounds()) {
		t.Errorf("the notice %v leaves %v", nl.Rect, e.popupBounds())
	}
	if w := nl.window(n); len(n.Notes) > len(w.Notes.Lines) && !strings.Contains(w.Title.Help, "finding 29") {
		t.Errorf("the notice's folded warnings are not behind its mark: %q", w.Title.Help)
	}
	// The notice's own warnings are capped below a fold, so a notice with
	// many more notes is laid out to prove the fold is reached.
	for i := range 60 {
		n.Notes = append(n.Notes, fmt.Sprint("extra warning ", i))
	}
	fl := layoutPopup(n, e.font, e.bodyFont())
	if w := fl.window(n); len(w.Notes.Lines) != 0 || fl.TitleHelp.Empty() || !strings.Contains(w.Title.Help, "extra warning 59") || !fl.Rect.In(e.popupBounds()) {
		t.Errorf("the notice with 60 more notes does not fold inside %v: %d notes stand, mark %v, window %v", e.popupBounds(), len(w.Notes.Lines), fl.TitleHelp, fl.Rect)
	}
}

func TestEditorBrushPaletteKeysClamp(t *testing.T) {
	for _, mode := range []string{"terrain", "height", "objects"} {
		t.Run(mode, func(t *testing.T) {
			e, _ := brushEditor(t)
			if err := e.Key(mode); err != nil {
				t.Fatal(err)
			}
			n := len(e.brushRows())
			if n < 2 {
				t.Fatalf("need multiple palette rows, got %d", n)
			}
			e.brushRowClick(0)
			_ = e.Key("up")
			if got := e.brush.sel[e.brush.mode]; got != 0 {
				t.Errorf("Up at first row selected %d of %d", got, n)
			}
			e.brushRowClick(0)
			for i := 0; i < n-1; i++ {
				_ = e.Key("down")
				if got := e.brush.sel[e.brush.mode]; got != i+1 {
					t.Fatalf("Down selected %d, want %d", got, i+1)
				}
			}
			_ = e.Key("down")
			if got := e.brush.sel[e.brush.mode]; got != n-1 {
				t.Errorf("Down at last row selected %d of %d", got, n)
			}
		})
	}
}

func TestEditorLegacyFieldChoicesClamp(t *testing.T) {
	e, h := popupEditor(t)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.Select(2, false)
	e.openPrompt(editorPrompt{kind: promptField, field: 1, path: "1"})
	for _, c := range []struct {
		delta int
		want  string
	}{{-1, "1"}, {1, "2"}, {1, "3"}, {1, "3"}, {-1, "2"}, {-1, "1"}, {-1, "1"}} {
		if !e.stepChoice(c.delta) || e.prompt.path != c.want {
			t.Fatalf("stepChoice(%d) selected %q, want %q", c.delta, e.prompt.path, c.want)
		}
	}
	if e.Dirty() || len(h.calls) != 0 {
		t.Fatal("choice navigation wrote the map")
	}
}

// A field prompt's choice row opens its list on a press on the box; a row
// chooses and closes it, the bar keeps it open and the box closes it.
func TestFieldPromptListOpensChoosesAndCloses(t *testing.T) {
	e, _ := popupEditor(t)
	e.SetBodyFont(wordsFont("en"))
	e.Layout(1920, 1080)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.Select(2, false)
	e.openPrompt(editorPrompt{kind: promptField, field: 1, path: "1"})
	s, _ := e.promptSpec()
	var box image.Rectangle
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if s.Rows[rr.Index].Kind == popupChoice {
			box = rr.Control
		}
	}
	if box.Empty() {
		t.Fatal("the field prompt shows no choice row")
	}
	chromeClick(e, center(box))
	d, ok := e.promptDropdown()
	if !ok || !d.Open || e.prompt.list == nil {
		t.Fatal("a press on the box opened no list")
	}
	f := e.bodyFont()
	chromeClick(e, center(d.listBox(f).Bar()))
	if e.prompt.list == nil {
		t.Fatal("a press on the bar closed the list")
	}
	top, _ := d.List.Visible()
	chromeClick(e, center(d.listBox(f).Row(1-top)))
	if e.prompt.list != nil || e.prompt.path != "2" {
		t.Fatalf("choosing the second row left list %v and path %q", e.prompt.list != nil, e.prompt.path)
	}
	chromeClick(e, center(box))
	chromeClick(e, center(box))
	if e.prompt.list != nil {
		t.Fatal("a second press on the box left the list open")
	}
}
