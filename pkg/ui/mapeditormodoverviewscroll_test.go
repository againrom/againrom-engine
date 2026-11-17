package ui

import (
	"fmt"
	"image"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// pageTipBox finds a page part's box, with the tip a (?) mark carries as its
// title between the part's name and its box.
var pageTipBox = regexp.MustCompile(`data-part="([a-z-]+)"(?: data-row="\d+")?(?: title="[^"]*")? data-box="(-?\d+) (-?\d+) (\d+) (\d+)"`)

// modPageSizes are the window sizes the focused layout properties run at.
var modPageSizes = []image.Point{{640, 480}, {800, 600}, {1920, 1080}}

// modPageProperties checks a mod page at each paged top: every row, label,
// control, button and (?) mark lies inside the screen; at 1920x1080 every
// label is whole; hit answers each label, control, (?) mark, row button,
// tab, close and window button at its rectangle with its index; the HTML
// page places every part at the layout's rectangle.
func modPageProperties(t *testing.T, e *MapEditor, screen image.Rectangle, tag string) {
	t.Helper()
	for top := 0; top < 14; top++ {
		e.objectTop = top
		s, _, ok := e.modSpec()
		if !ok {
			t.Fatalf("%s top %d: no window spec", tag, top)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		if !l.Rect.In(screen) {
			t.Errorf("%s top %d: window %v lies outside %v", tag, top, l.Rect, screen)
		}
		for _, r := range l.Rows {
			for _, rect := range []image.Rectangle{r.Rect, r.Label, r.Control, r.Button, r.Help} {
				if !rect.Empty() && !rect.In(screen) {
					t.Errorf("%s top %d: row %d element %v lies outside %v", tag, top, r.Index, rect, screen)
				}
			}
			for _, b := range r.Buttons {
				if !b.In(screen) {
					t.Errorf("%s top %d: row %d button %v lies outside %v", tag, top, r.Index, b, screen)
				}
			}
			if screen.Dx() == 1920 && !r.Label.Empty() {
				label := s.Rows[r.Index].Label
				lines := editorTextLines(e.bodyFont(), label, r.Label.Dx())
				if h := len(lines) * e.bodyFont().Height(); h > r.Label.Dy() {
					t.Errorf("%s top %d: row %d label %q needs %d px in %v", tag, top, r.Index, label, h, r.Label)
				}
			}
			if !r.Label.Empty() {
				if p, at := l.hit(boxMidpoint(r.Label)); p != popupRowLine || at != r.Index {
					t.Errorf("%s top %d: row %d label hit %v %d", tag, top, r.Index, p, at)
				}
			}
			if !r.Control.Empty() {
				if p, at := l.hit(boxMidpoint(r.Control)); p != popupRowControl || at != r.Index {
					t.Errorf("%s top %d: row %d control hit %v %d", tag, top, r.Index, p, at)
				}
			}
			if !r.Help.Empty() {
				if p, at := l.hit(boxMidpoint(r.Help)); p != popupRowHelp || at != r.Index {
					t.Errorf("%s top %d: row %d help hit %v %d", tag, top, r.Index, p, at)
				}
			}
			for _, b := range r.Buttons {
				if p, at := l.hit(boxMidpoint(b)); p != popupRowButton || at != r.Index {
					t.Errorf("%s top %d: row %d button hit %v %d", tag, top, r.Index, p, at)
				}
			}
		}
		for n, b := range l.Buttons {
			if !b.In(screen) {
				t.Errorf("%s top %d: window button %d %v lies outside %v", tag, top, n, b, screen)
			}
			if p, at := l.hit(boxMidpoint(b)); p != popupButton || at != n {
				t.Errorf("%s top %d: window button %d hit %v %d", tag, top, n, p, at)
			}
		}
		for n, b := range l.Tabs {
			if !b.In(screen) {
				t.Errorf("%s top %d: tab %d %v lies outside %v", tag, top, n, b, screen)
			}
			if p, at := l.hit(boxMidpoint(b)); p != popupTab || at != n {
				t.Errorf("%s top %d: tab %d hit %v %d", tag, top, n, p, at)
			}
		}
		if p, _ := l.hit(boxMidpoint(l.Close)); p != popupClose {
			t.Errorf("%s top %d: close hit %v", tag, top, p)
		}
		page := popupPage(popupDesc{Window: "mod"}, specWindow(s), l, image.Rectangle{}, "en", screen.Max)
		var got []pagePart
		for _, p := range pageTipBox.FindAllStringSubmatch(string(page), -1) {
			n := func(i int) int { v, _ := strconv.Atoi(p[i]); return v }
			got = append(got, pagePart{p[1], image.Rect(n(2), n(3), n(2)+n(4), n(3)+n(5))})
		}
		if want := layoutParts(s, l); !reflect.DeepEqual(got, want) {
			t.Errorf("%s top %d: page parts\n%v\nlayout\n%v", tag, top, got, want)
		}
	}
}

// boxMidpoint is the middle point of r, where hit is asked.
func boxMidpoint(r image.Rectangle) image.Point {
	return r.Min.Add(r.Size().Div(2))
}

// The overview page keeps its rows, labels and buttons inside the window at
// each size, with longOverview's lines paging through it.
func TestModOverviewPageHasFocusedLayoutProperties(t *testing.T) {
	for _, size := range modPageSizes {
		e := modesWordsEditor(t, "workspace", "en", size)
		host := e.doc.Edits.(*modesWorkspace)
		host.overview = longOverview()
		_ = e.Key("package")
		if e.modPage != modPageOverview {
			t.Fatalf("%v: W shows page %d, not the overview", size, e.modPage)
		}
		modPageProperties(t, e, image.Rect(0, 0, size.X, size.Y), fmt.Sprint(size))
	}
}

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
