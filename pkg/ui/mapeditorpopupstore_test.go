package ui

import (
	"crypto/sha256"
	"fmt"
	"image"
	"os"
	"strings"
	"testing"
	"time"
)

// popupTranscriptHash and popupTranscriptLines pin the answers every movable
// editor popup gave before its placement became one store: where it opens, how
// a title drag, every border drag and a window resize move it step by step,
// and where it reopens.
const (
	popupTranscriptHash  = "8375661892fa7e8df23850de90e25938124d459daaf3ab00d6e134ee657fb72b"
	popupTranscriptLines = 2310
)

type popupTranscript struct {
	t   *testing.T
	out strings.Builder
}

func (r *popupTranscript) rec(e *MapEditor, label string) {
	rects := e.popupRects()
	title, _ := e.TitleRect()
	front, _ := e.PopupRect()
	fmt.Fprintf(&r.out, "%s front=%v title=%v all=%v\n", label, front, title, rects)
}

// path presses at from and walks the pointer by each step of steps, recording
// every stop, then lets go.
func (r *popupTranscript) path(e *MapEditor, label string, from image.Point, steps []image.Point) {
	e.Pointer(Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, time.Time{})
	at := from
	for i, d := range steps {
		at = at.Add(d)
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, time.Time{})
		r.rec(e, fmt.Sprintf("%s step %d", label, i))
	}
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y}, time.Time{})
	r.rec(e, label+" released")
}

var popupTranscriptSteps = []image.Point{{7, 3}, {20, -9}, {-300, 40}, {-5000, -5000}, {9000, 9000}, {-3, -3}, {-600, -200}}

type popupScenario struct {
	name   string
	open   func(t *testing.T) *MapEditor
	reopen func(e *MapEditor)
}

func popupScenarios() []popupScenario {
	groups := func(t *testing.T, act func(e *MapEditor)) *MapEditor {
		e, _ := groupsEditor(t)
		act(e)
		return e
	}
	return []popupScenario{
		{"brush list", func(t *testing.T) *MapEditor {
			e, _ := brushEditor(t)
			_ = e.Key("terrain")
			e.ShowCell(image.Pt(30, 30))
			return e
		}, func(e *MapEditor) { _ = e.Key("terrain"); _ = e.Key("terrain") }},
		{"unit window", func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			e.Select(2, true)
			return e
		}, func(e *MapEditor) { e.Select(3, true); e.Select(2, true) }},
		{"map panel", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			return e
		}, func(e *MapEditor) { _ = e.Key("testplay"); _ = e.Key("escape") }},
		{"notice", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			_ = e.Key("testplay")
			return e
		}, func(e *MapEditor) { _ = e.Key("escape"); _ = e.Key("testplay") }},
		{"group popup", func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			openGroup(t, e)
			return e
		}, func(e *MapEditor) { e.Select(0, false); e.Select(2, true) }},
		{"groups list", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { chromeClick(e, barPoint(t, e, "groups")) })
		}, func(e *MapEditor) { _ = e.Key("players"); _ = e.Key("help") }},
		{"unit own", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { e.Select(0, false) })
		}, func(e *MapEditor) { e.Select(5, false); e.Select(7, false) }},
		{"trigger", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { e.Select(7, false) })
		}, func(e *MapEditor) { e.Select(0, false); e.Select(7, false) }},
		{"players", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { _ = e.Key("players") })
		}, func(e *MapEditor) { _ = e.Key("help"); _ = e.Key("players") }},
		{"save as", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { _ = e.Key("saveas") })
		}, func(e *MapEditor) { _ = e.Key("escape"); _ = e.Key("saveas") }},
		{"help", func(t *testing.T) *MapEditor {
			return groups(t, func(e *MapEditor) { _ = e.Key("help") })
		}, func(e *MapEditor) { _ = e.Key("players") }},
	}
}

// popupTranscriptText runs every scenario at two window sizes.
func popupTranscriptText(t *testing.T) string {
	r := &popupTranscript{t: t}
	pickers := []struct {
		name string
		pick func(image.Rectangle) image.Point
	}{{"right", rightEdge}, {"bottom", bottomEdge}, {"left", leftEdge}, {"top", topEdge}, {"corner", bottomRight}, {"topleft", topLeft}}
	resizeSteps := []image.Point{{13, 7}, {-40, -30}, {2000, 2000}, {-2000, -2000}, {-5, 11}}
	for _, size := range []image.Point{{1280, 800}, {640, 480}} {
		for _, sc := range popupScenarios() {
			prefix := fmt.Sprintf("%dx%d %s", size.X, size.Y, sc.name)
			fresh := func() *MapEditor {
				e := sc.open(t)
				e.Layout(size.X, size.Y)
				return e
			}
			e := fresh()
			r.rec(e, prefix+" open")
			if title, ok := e.TitleRect(); ok {
				r.path(e, prefix+" title", title.Min.Add(image.Pt(10, 4)), popupTranscriptSteps)
			}
			for _, p := range pickers {
				e = fresh()
				if front, ok := e.PopupRect(); ok {
					r.path(e, prefix+" "+p.name, p.pick(front), resizeSteps)
					r.path(e, prefix+" "+p.name+" again", p.pick(func() image.Rectangle { f, _ := e.PopupRect(); return f }()), resizeSteps[:2])
					if title, ok := e.TitleRect(); ok {
						r.path(e, prefix+" "+p.name+" title", title.Min.Add(image.Pt(10, 4)), popupTranscriptSteps[:3])
					}
					sc.reopen(e)
					r.rec(e, prefix+" "+p.name+" reopened")
					e.Layout(size.Y+100, size.X/2)
					r.rec(e, prefix+" "+p.name+" relaid")
					e.Layout(size.X, size.Y)
					r.rec(e, prefix+" "+p.name+" back")
				}
			}
		}
	}
	return r.out.String()
}

// Every movable editor popup opens, moves, resizes and reopens as it did
// when each kept its own place and size.
func TestEditorPopupPlacementTranscript(t *testing.T) {
	text := popupTranscriptText(t)
	if path := os.Getenv("AGAINROM_POPUP_TRANSCRIPT"); path != "" {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	lines := strings.Count(text, "\n")
	if sum != popupTranscriptHash || lines != popupTranscriptLines {
		t.Fatalf("popup placement transcript: %d lines, sha256 %s", lines, sum)
	}
}
