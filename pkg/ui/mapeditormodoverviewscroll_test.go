package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// longOverview is an overview page of more lines than the window shows.
func longOverview() ModOverviewPage {
	var lines []string
	for n := 0; n < 60; n++ {
		lines = append(lines, fmt.Sprintf("file %02d.alm", n))
	}
	return ModOverviewPage{First: true, Sections: []ModOverviewSection{{Heading: "Files", Lines: lines}}}
}

// The overview page scrolls inside its own lines: the wheel moves the shown
// rows, stops at the last, and a page change after it answers as before.
func TestModWindowOverviewScrollsAndStopsAtItsContent(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits.(*modesWorkspace).overview = longOverview()
	_ = e.Key("package")
	if e.modPage != modPageOverview {
		t.Fatalf("W shows page %d", e.modPage)
	}
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	first := l.Rows[0].Index
	e.ScrollDetails(2)
	s, _, _ = e.modSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	if l.Rows[0].Index == first {
		t.Fatalf("the wheel left the overview at row %d (top %d of %d rows, %d fit)", first, s.Top, len(s.Rows), s.MaxRows)
	}
	e.ScrollDetails(1000)
	s, _, _ = e.modSpec()
	l = layoutPopup(s, e.font, e.bodyFont())
	last := l.Rows[len(l.Rows)-1].Index
	if last != len(s.Rows)-1 || e.objectTop > len(s.Rows)-s.MaxRows {
		t.Fatalf("the overview ends at row %d of %d, top %d, %d fit", last, len(s.Rows), e.objectTop, s.MaxRows)
	}
	for turn := 0; turn < 4; turn++ {
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
		if !e.modWindow {
			t.Fatalf("turn %d: the window closed", turn)
		}
		if _, _, ok := e.modSpec(); !ok {
			t.Fatalf("turn %d: no window spec", turn)
		}
	}
}

// Every label of W's graph page is drawn once through the host's encoding:
// no row, button or note of it turns into question marks in Russian, and the
// edit button reads the word book's word.
func TestModWindowGraphPageLabelsKeepTheirLetters(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := modesWordsEditor(t, "workspace", lang, image.Pt(1280, 1080))
			ws := e.doc.Edits.(*modesWorkspace)
			ws.graph = modesHandoutGraph()
			ws.graphFields = modesCampaignFields()
			_ = e.Key("package")
			_ = e.Key("package")
			if e.modPage != modPageGraph {
				t.Fatalf("W shows page %d", e.modPage)
			}
			var lines []string
			shown := func() {
				s, _, _ := e.modSpec()
				for _, r := range s.Rows {
					lines = append(lines, r.Label, r.Value)
					lines = append(lines, r.Buttons...)
				}
				lines = append(lines, s.Notes...)
				lines = append(lines, e.PopupLines()...)
			}
			shown()
			want := e.word("editor.graph.edit")
			if !slices.Contains(lines, want) {
				t.Errorf("the edit button does not read %q:\n%q", want, lines)
			}
			e.graphEdit, e.graphMission = true, 10
			e.obj.doc = nil
			shown()
			e.graphLink = true
			e.obj.doc = nil
			shown()
			for _, key := range []string{"editor.graph.openmaps", "editor.graph.link", "editor.graph.unlink"} {
				if w := e.word(key); !slices.Contains(lines, w) {
					t.Errorf("no button reads %q:\n%q", w, lines)
				}
			}
			for _, l := range lines {
				if lang == "ru" && strings.Contains(l, "??") {
					t.Errorf("a label lost its letters: %q", l)
				}
			}
			if lang == "ru" {
				if w := e.word("editor.graph.mission"); strings.Contains(w, "?") {
					t.Errorf("the word %q has no glyphs", w)
				}
			}
		})
	}
}
