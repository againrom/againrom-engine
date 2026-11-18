package ui

import (
	"image"
	"slices"
	"testing"
	"time"
)

var centreSizes = []image.Point{{1280, 720}, {2560, 1440}}

// nearCentre reports whether rectangle r stands in the middle of a window of
// size, to three pixels.
func nearCentre(r image.Rectangle, size image.Point) bool {
	d := r.Min.Add(r.Max).Div(2).Sub(size.Div(2))
	return d.X >= -3 && d.X <= 3 && d.Y >= -3 && d.Y <= 3
}

// The mod window first opens in the middle of the editor window, at two
// window sizes; a dragged one reopens where it was dragged.
func TestModWindowOpensInTheMiddle(t *testing.T) {
	for _, size := range centreSizes {
		e := newMapEditorFixture(t)
		e.Layout(size.X, size.Y)
		s := popupSpec{Title: "mod", At: e.keptCorner(popObject), Width: 620, Bounds: e.popupBounds(),
			Rows: []popupRow{{Kind: popupEdit, Label: "a"}, {Kind: popupEdit, Label: "b"}}}
		e.openCentred(popObject, &s)
		if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !nearCentre(r, size) {
			t.Errorf("%dx%d: the mod window stands at %v", size.X, size.Y, r)
		}
		dragged := image.Pt(40, 90)
		e.places[popObject] = popupStanding{at: dragged, slack: true}
		s.At = e.keptCorner(popObject)
		e.openCentred(popObject, &s)
		if s.At != dragged {
			t.Errorf("%dx%d: a dragged mod window reopens at %v, want %v", size.X, size.Y, s.At, dragged)
		}
	}
}

// The list and the object window first open in the middle of the editor
// window at two window sizes, through keepInRoom, the route both specs take.
func TestListAndObjectWindowsFirstOpenInTheMiddle(t *testing.T) {
	for _, size := range centreSizes {
		for _, id := range []editorPopupID{popList, popObject} {
			e := newMapEditorFixture(t)
			e.Layout(size.X, size.Y)
			s := popupSpec{Title: "window", At: e.keptCorner(id), Width: 420, Bounds: e.popupBounds(),
				Rows: []popupRow{{Kind: popupEdit, Label: "a"}, {Kind: popupEdit, Label: "b"}, {Kind: popupEdit, Label: "c"}}}
			e.keepInRoom(id, &s)
			if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !nearCentre(r, size) {
				t.Errorf("%dx%d popup %d: the window stands at %v", size.X, size.Y, id, r)
			}
		}
	}
}

// A prompt first opens in the middle of the editor window, as every window
// does, and the drag of its title moves it from there; the Add window states
// its own place, centred across with its top at an eighth of the height.
func TestPromptsFirstOpenInTheMiddle(t *testing.T) {
	for _, size := range centreSizes {
		f, rows := endless()
		e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
		e.Layout(size.X, size.Y)
		_ = e.Key("testplay")
		s, ok := e.promptSpec()
		if !ok || e.prompt.kind != promptNotice {
			t.Fatalf("%dx%d: no notice", size.X, size.Y)
		}
		if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !nearCentre(r, size) {
			t.Errorf("%dx%d: the notice stands at %v", size.X, size.Y, r)
		}
		after := titleDragBy(t, e, image.Pt(30, 20))
		s, _ = e.promptSpec()
		if r := layoutPopup(s, e.font, e.bodyFont()).Rect; r.Min.Y != after.Min.Y {
			t.Errorf("%dx%d: the dragged notice stands at %v, its title at %v", size.X, size.Y, r, after)
		}

	}
}

// titleDragBy drags the front popup by its title bar by d through the
// editor's pointer and returns the title bar after.
func titleDragBy(t *testing.T, e *MapEditor, d image.Point) image.Rectangle {
	t.Helper()
	title, ok := e.FindControl(ControlQuery{Role: ControlTitle})
	if !ok {
		t.Fatal("no popup title")
	}
	p := title.Min.Add(image.Pt(10, popupTopBand+2))
	q := p.Add(d)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	after, _ := e.FindControl(ControlQuery{Role: ControlTitle})
	if after == title {
		t.Fatalf("the title drag did not move the popup from %v", title)
	}
	return after
}

// A list and an object window open in the middle, and once dragged by the
// title they close and reopen where the drag left them, at two window sizes.
func TestDraggedPopupReopensWhereItWasDragged(t *testing.T) {
	for _, size := range centreSizes {
		e, _ := popupEditor(t)
		_ = e.Open("popups")
		e.Layout(size.X, size.Y)
		e.Select(2, true)
		if r, ok := e.PopupRect(); !ok || !nearCentre(r, size) && r.Min.Y != e.popupBounds().Min.Y {
			t.Fatalf("%dx%d: the unit window opens at %v", size.X, size.Y, r)
		}
		dragged := titleDragBy(t, e, image.Pt(-200, 40))
		s, _, _ := e.objectSpec()
		chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Close))
		if _, _, ok := e.objectSpec(); ok {
			t.Fatalf("%dx%d: the close control left the unit window open", size.X, size.Y)
		}
		e.Select(2, true)
		if title, _ := e.FindControl(ControlQuery{Role: ControlTitle}); title != dragged {
			t.Errorf("%dx%d: the dragged unit window reopens with its title at %v, want %v", size.X, size.Y, title, dragged)
		}

		b, _ := brushEditor(t)
		b.Layout(size.X, size.Y)
		_ = b.Key("terrain")
		if r, ok := b.PopupRect(); !ok || !nearCentre(r, size) {
			t.Fatalf("%dx%d: the brush list opens at %v", size.X, size.Y, r)
		}
		dragged = titleDragBy(t, b, image.Pt(-300, 60))
		_ = b.Key("terrain")
		if _, ok := b.PopupRect(); ok {
			t.Fatalf("%dx%d: the second Terrain key left the brush list open", size.X, size.Y)
		}
		_ = b.Key("terrain")
		if title, _ := b.FindControl(ControlQuery{Role: ControlTitle}); title != dragged {
			t.Errorf("%dx%d: the dragged brush list reopens with its title at %v, want %v", size.X, size.Y, title, dragged)
		}
	}
}

// A window opened from another stands one step down and right of it: the
// Units list opens in the middle, the unit chosen from it opens at the
// list's corner plus the title row's height and popupPad, so the list's
// title bar stays uncovered and takes a press. A lone window stays in the
// middle and a dragged one keeps its place.
func TestWindowOpenedFromAnotherOpensOffsetFromIt(t *testing.T) {
	for _, size := range centreSizes {
		e, _ := newModesEditor(t, "full")
		e.Layout(size.X, size.Y)
		// The fixture parks both windows off the screen; these stand as opened.
		e.places[popList], e.places[popObject] = popupStanding{}, popupStanding{}
		c, _ := categoryByKey("units")
		unit := slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == KindUnit })
		openUnit := func() {
			e.openRecords(c.filter)
			e.Select(unit, false)
		}
		openUnit()
		ls, ok := e.listSpec()
		if !ok {
			t.Fatalf("%dx%d: the Units list is not open", size.X, size.Y)
		}
		os, _, ok := e.objectSpec()
		if !ok {
			t.Fatalf("%dx%d: the unit window is not open", size.X, size.Y)
		}
		list := layoutPopup(ls, e.font, e.bodyFont())
		win := layoutPopup(os, e.font, e.bodyFont())
		step := list.Title.Dy() + popupPad
		if want := list.Rect.Min.Add(image.Pt(step, step)); win.Rect.Min != want {
			t.Errorf("%dx%d: the unit window's corner is %v, want the list's %v plus %d", size.X, size.Y, win.Rect.Min, list.Rect.Min, step)
		}
		if win.Rect.Min.Y < list.Title.Max.Y {
			t.Errorf("%dx%d: the unit window %v covers the list's title bar %v", size.X, size.Y, win.Rect, list.Title)
		}
		if at := list.Title.Min.Add(image.Pt(4, popupTopBand+2)); e.chromeHit(at) != popList {
			t.Errorf("%dx%d: a press at %v on the list's title bar reaches %v, not the list", size.X, size.Y, at, e.chromeHit(at))
		}
		if !nearCentre(list.Rect, size) {
			t.Errorf("%dx%d: the first window stands at %v, not in the middle", size.X, size.Y, list.Rect)
		}

		// The unit window closed and opened again stands at the same offset.
		e.closeBarWindow()
		openUnit()
		if os2, _, ok := e.objectSpec(); !ok || layoutPopup(os2, e.font, e.bodyFont()).Rect.Min != win.Rect.Min {
			t.Errorf("%dx%d: the reopened unit window is not at %v", size.X, size.Y, win.Rect.Min)
		}

		// A dragged unit window keeps its place beside the list.
		dragged := image.Pt(40, 90)
		e.places[popObject] = popupStanding{at: dragged, placed: true, slack: true}
		os, _, _ = e.objectSpec()
		if os.At != dragged {
			t.Errorf("%dx%d: a dragged unit window stands at %v, want %v", size.X, size.Y, os.At, dragged)
		}

		// A window that cannot move off the list without leaving the room keeps the middle.
		e.places[popObject] = popupStanding{}
		tall := popupSpec{Title: "tall", Width: 420, Bounds: e.popupBounds()}
		for i := 0; i < 80; i++ {
			tall.Rows = append(tall.Rows, popupRow{Kind: popupEdit, Label: "row"})
		}
		tall.MaxRows = 80
		e.openCentred(popObject, &tall)
		if r := layoutPopup(tall, e.font, e.bodyFont()).Rect; r.Min.X != (size.X-r.Dx())/2 {
			t.Errorf("%dx%d: the tall window stands at %v, not in the middle", size.X, size.Y, r)
		}
	}
}

// A window already standing stays where it is when the list window opens
// after it; only a window that opens while the list stands is offset.
func TestStandingWindowStaysWhenTheListOpensAfterIt(t *testing.T) {
	for _, size := range centreSizes {
		e, _ := popupEditor(t)
		_ = e.Open("popups")
		e.Layout(size.X, size.Y)
		e.Select(2, true)
		before, ok := e.PopupRect()
		if !ok || !nearCentre(before, size) {
			t.Fatalf("%dx%d: the unit window opens at %v", size.X, size.Y, before)
		}
		e.showRecords(0)
		if _, ok := e.listSpec(); !ok {
			t.Fatalf("%dx%d: the list did not open", size.X, size.Y)
		}
		if after, _ := e.PopupRect(); after != before {
			t.Errorf("%dx%d: the standing window moved from %v to %v when the list opened", size.X, size.Y, before, after)
		}
	}
}
