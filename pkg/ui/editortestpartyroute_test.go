package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// partyBoxPress presses the middle of every box the frame draws, through the
// editor's pointer route, and fails unless the press switches that card's box
// and no other.
func partyBoxPress(t *testing.T, e *MapEditor, label string) {
	t.Helper()
	if !e.TestPartyOpen() {
		t.Fatalf("%s: the window is closed", label)
	}
	cards := e.prompt.party.cards
	for row := range testPartyRows {
		for col, c := range cards {
			s, ok := e.promptSpec()
			if !ok || s.Grid == nil {
				t.Fatalf("%s: no grid", label)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			n := row*len(cards) + col
			if n >= len(l.GridCells) {
				t.Fatalf("%s: the frame draws %d boxes of %d", label, len(l.GridCells), len(s.Grid.Cells))
			}
			if l.GridInert[n] {
				continue
			}
			r := l.GridCells[n]
			before := e.TestPartyChoice()
			editorClickAt(e, r.Min.Add(r.Size().Div(2)), time.Now())
			if !e.TestPartyOpen() {
				t.Fatalf("%s: row %d card %s closed the window", label, row, c.Name)
			}
			after := e.TestPartyChoice()
			if len(after) != len(before) {
				t.Fatalf("%s: the cards changed from %d to %d", label, len(before), len(after))
			}
			for i := range after {
				want := before[i]
				if i == col {
					switch row {
					case testPartyRowPlays:
						want.On = !want.On
					case testPartyRowSkills:
						want.Skills = !want.Skills
					default:
						want.Spells = !want.Spells
					}
				}
				if after[i] != want {
					t.Fatalf("%s: press row %d card %d (%s): card %d holds %+v, want %+v", label, row, col, c.Name, i, after[i], want)
				}
			}
		}
	}
}

func partyDragTitle(e *MapEditor, d image.Point) {
	s, _ := e.promptSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	from := image.Pt(l.Title.Min.X+20, l.Title.Min.Y+l.Title.Dy()/2)
	now := time.Now()
	e.Pointer(Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, now)
	for k := 1; k <= 4; k++ {
		q := from.Add(d.Mul(k).Div(4))
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, now)
	}
	q := from.Add(d)
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, now)
}

func partyDragCorner(e *MapEditor, d image.Point) {
	s, _ := e.promptSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	from := l.Rect.Max.Sub(image.Pt(2, 2))
	now := time.Now()
	e.Pointer(Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, now)
	for k := 1; k <= 4; k++ {
		q := from.Add(d.Mul(k).Div(4))
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, now)
	}
	q := from.Add(d)
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, now)
}

// Every box the frame draws takes its press through the editor's pointer route
// after the window was opened on Standard first, moved, sized wider or
// narrower than its grid, or opened a second time, at both window sizes: the
// press switches that card's box and no other.
func TestThePartyWindowBoxesTakePressesAfterMovesAndSizes(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{1024, 768}, {640, 480}} {
			for _, route := range []string{"fresh", "standardFirst", "moved", "movedStd", "sizedStd", "sizedCustom", "reopen"} {
				name := fmt.Sprintf("%s%d/%s", lang, size.X, route)
				t.Run(name, func(t *testing.T) {
					d := &declaredEdits{partyEdits{mapPanelEdits: mapPanelEdits{facts: playable()}, byHero: herdCards}}
					e := partyEditor(t, &d.partyEdits, lang)
					e.doc.Edits = d
					e.SetPanelProfile(nil, nil, func([]byte) error { return nil })
					e.Layout(size.X, size.Y)
					_ = e.Key("testplay")
					switch route {
					case "fresh":
						partyPressMode(e)
					case "standardFirst":
						partyPressMode(e)
						partyBoxPress(t, e, "custom")
					case "moved":
						partyPressMode(e)
						partyDragTitle(e, image.Pt(-37, 21))
					case "movedStd":
						partyDragTitle(e, image.Pt(-37, 21))
						partyPressMode(e)
					case "sizedStd":
						partyDragCorner(e, image.Pt(60, 40))
						partyPressMode(e)
					case "sizedCustom":
						partyPressMode(e)
						partyDragCorner(e, image.Pt(-90, 30))
					case "reopen":
						partyPressMode(e)
						partyBoxPress(t, e, "first")
						_ = e.Key("escape")
						_ = e.Key("testplay")
					}
					for k := 0; k < 5; k++ {
						partyBoxPress(t, e, fmt.Sprintf("kind %d", e.TestPartyHero()))
						partyNextKind(e)
					}
				})
			}
		}
	}
}
