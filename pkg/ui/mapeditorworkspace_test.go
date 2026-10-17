package ui

import (
	"image"
	"strings"
	"testing"
)

func TestMapEditorPopupsStayInsideTheWindow(t *testing.T) {
	for _, size := range []image.Point{image.Pt(640, 480), image.Pt(1280, 800), image.Pt(1920, 1080)} {
		for _, path := range []string{"list", "selected", "clone", "sack", "unit", "drop", "goals", "goals-pick", "terrain", "height", "objects", "players", "field", "open", "saveas"} {
			e, _ := newModesEditor(t, "full")
			e.Layout(size.X, size.Y)
			for _, step := range modesPathSteps(path) {
				modesAct(e, step)
			}
			window := image.Rect(0, e.barHeight(), size.X, size.Y)
			if s, ok := e.listSpec(); ok {
				if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !r.In(window) {
					t.Fatalf("%v/%s list popup %v leaves %v", size, path, r, window)
				}
			}
			if s, _, ok := e.objectSpec(); ok {
				if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !r.In(window) {
					t.Fatalf("%v/%s object popup %v leaves %v", size, path, r, window)
				}
			}
			if e.prompt.open() {
				e.prompt.path = strings.Repeat("path/", 160) + "tail"
				e.message = "refused input: keep the document and typed tail"
				s, l := promptLayout(t, e)
				if !l.Rect.In(window) {
					t.Fatalf("%v/%s prompt %v leaves %v", size, path, l.Rect, window)
				}
				last := s.Rows[len(s.Rows)-1]
				for _, r := range s.Rows {
					if r.Focus {
						last = r
					}
				}
				refusal := false
				for _, n := range s.Notes {
					refusal = refusal || strings.Contains(n, "refused input")
				}
				if !strings.HasSuffix(last.Value, "tail") || !last.Focus || !refusal {
					t.Fatalf("%v/%s lost caret or refusal: %+v %v", size, path, last, s.Notes)
				}
			}
			_ = e.ChromeImage()
		}
	}
}

func TestMapEditorAllKeysUseTheScrollableReport(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	before := e.doc
	if err := e.Key("help"); err != nil {
		t.Fatal(err)
	}
	rows, open := e.ReportOpen()
	if !open {
		t.Fatal("F1 did not expose every key row")
	}
	seen := map[string]bool{}
	for _, row := range rows {
		if seen[row.Text] {
			t.Fatalf("duplicate help %q", row.Text)
		}
		seen[row.Text] = true
	}
	for _, line := range append(EditorHelpLines(), e.saveHelp()) {
		if !seen[line] {
			t.Errorf("missing help %q", line)
		}
	}
	for _, line := range []string{"P: records / A: add / Del: delete", "T: off / Drag: paint / [ ]: size", "Enter: set / Tab: next / Esc: back", "Target: click a unit or", "structure / Esc: cancel"} {
		if !seen[line] {
			t.Errorf("missing contextual control %q", line)
		}
	}
	if !strings.Contains(strings.Join(e.helpLines(), " "), "P: players / F5: test play") {
		t.Fatal("P and F5 lack visible labels")
	}
	if e.doc != before || len(calls.lines) != 0 {
		t.Fatal("help changed the map")
	}
	for range rows {
		if err := e.Key("down"); err != nil {
			t.Fatal(err)
		}
	}
	if e.ReportSelected() != len(rows)-1 {
		t.Fatal("help did not scroll to the last row")
	}
	if err := e.Key("help"); err != nil {
		t.Fatal(err)
	}
	if _, open := e.ReportOpen(); open {
		t.Fatal("F1 did not close its report")
	}
}
