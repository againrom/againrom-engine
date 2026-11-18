package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"
)

// areaPos is the first shown position of area i of target, read from a fresh
// build of the areas.
func areaPos(e *MapEditor, target editorPopupID, i int) (int, scrollArea) {
	as := e.scrollAreas(target)
	if i >= len(as) {
		return -1, scrollArea{}
	}
	a := as[i]
	if a.List != nil {
		return max(a.List.Selection(), 0), a
	}
	return a.Bar.Pos, a
}

// walkScrollAreas drives every area of target that has a bar through the
// pointer: a thumb drag to the top, a wheel notch, a track click below the
// thumb and a thumb drag to the end; each must move the area's position.
// It reports how many areas it walked.
func walkScrollAreas(t *testing.T, e *MapEditor, target editorPopupID) int {
	t.Helper()
	now := time.Unix(1, 0)
	step := func() time.Time { now = now.Add(time.Second); return now }
	walked := 0
	for i := range e.scrollAreas(target) {
		_, a := areaPos(e, target, i)
		if a.Bar.Rect.Empty() || a.Bar.Count < 4 {
			continue
		}
		walked++
		name := fmt.Sprintf("popup %d area %d", target, i)
		// bar is area i's bar now, with its position and count.
		bar := func() vScrollBar {
			pos, a := areaPos(e, target, i)
			b := a.Bar
			b.Pos = pos
			if a.List != nil {
				b.Count = a.List.Len()
			}
			return b
		}
		drag := func(to int) {
			b := bar()
			from := center(b.Thumb())
			end := b
			end.Pos = to
			dest := center(end.Thumb())
			e.Pointer(Input{CursorX: from.X, CursorY: from.Y, PrimaryDown: true}, step())
			e.Pointer(Input{CursorX: dest.X, CursorY: dest.Y, PrimaryDown: true}, step())
			e.Pointer(Input{CursorX: dest.X, CursorY: dest.Y}, step())
		}
		drag(0)
		if pos, _ := areaPos(e, target, i); pos != 0 {
			t.Fatalf("%s: dragging the thumb to the top left it at %d", name, pos)
		}
		b := bar()
		at := center(b.Rect)
		e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, step())
		if pos, _ := areaPos(e, target, i); pos < 1 {
			t.Fatalf("%s: one wheel notch over the bar left it at %d", name, pos)
		}
		drag(0)
		wheeled := 0
		b = bar()
		track := image.Pt(center(b.Rect).X, b.Thumb().Max.Y+3)
		e.Pointer(Input{CursorX: track.X, CursorY: track.Y, PrimaryDown: true}, step())
		e.Pointer(Input{CursorX: track.X, CursorY: track.Y}, step())
		if pos, _ := areaPos(e, target, i); pos <= wheeled {
			t.Fatalf("%s: a click on the track below the thumb left it at %d", name, pos)
		}
		b = bar()
		drag(b.Count - 1)
		if pos, _ := areaPos(e, target, i); pos < b.Count-2 {
			t.Fatalf("%s: dragging the thumb to the end left it at %d of %d", name, pos, b.Count)
		}
	}
	return walked
}

// Every scroll area the editor builds answers the wheel, a track click and
// a thumb drag through the same route.
func TestEveryEditorScrollAreaAnswersWheelTrackAndThumb(t *testing.T) {
	cases := []struct {
		name   string
		target editorPopupID
		build  func(t *testing.T) *MapEditor
	}{
		{"item grid", popList, func(t *testing.T) *MapEditor {
			e, _ := bagGrid(t)
			return e
		}},
		{"record list", popList, func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			e.Layout(1920, 1080)
			_ = e.Open("popups")
			e.showRecords(0)
			for i := 0; i < 100; i++ {
				e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindUnit, Index: i + 10, Label: fmt.Sprint(i)})
			}
			e.showRecords(0)
			return e
		}},
		{"drop-down", popObject, func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			e.Layout(1920, 1080)
			_ = e.Open("popups")
			e.Select(0, false)
			e.objectSpec()
			for i := 0; i < 80; i++ {
				e.obj.fields[1].Choices = append(e.obj.fields[1].Choices, InspectionChoice{Value: int64(i + 100), Label: fmt.Sprint(i)})
			}
			e.objectRowPress(popupRowControl, e.fieldRow(1))
			return e
		}},
		{"paged body", popList, func(t *testing.T) *MapEditor {
			p := &mapPanelEdits{facts: playable()}
			p.facts.Description = strings.TrimSpace(strings.Repeat("Hold the northern bridge. ", 24))
			e, _ := mapPanelEditor(t, p, nil)
			e.Layout(1920, 1080)
			e.places[popList].key = e.windowKey(popList, "")
			e.places[popList].size = image.Pt(580, 120)
			return e
		}},
		{"details page", popObject, func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			e.Layout(1920, 1080)
			_ = e.Open("popups")
			e.Select(0, false)
			for i := 0; i < 100; i++ {
				e.doc.Records[0].Details = append(e.doc.Records[0].Details, fmt.Sprint(i))
			}
			return e
		}},
		{"message text", popObject, func(t *testing.T) *MapEditor {
			e := messageWindowEditor(t, "en", longMessageCases("en")["one of 60 lines"])
			e.Layout(1920, 1080)
			return e
		}},
		{"campaign graph", popObject, func(t *testing.T) *MapEditor {
			e, _ := graphEditor(t, modesGraph())
			e.Layout(1920, 1080)
			return e
		}},
	}
	walked := 0
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := walkScrollAreas(t, c.build(t), c.target)
			if n == 0 {
				t.Fatalf("%s: no scroll area with a bar to walk", c.name)
			}
			walked += n
		})
	}
	if walked < len(cases) {
		t.Fatalf("walked %d areas over %d cases", walked, len(cases))
	}
}
