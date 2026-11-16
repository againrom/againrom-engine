package ui

import (
	"image"
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

// The mod window opens from its home in the middle of the editor window, at
// two window sizes; a dragged one reopens where it was dragged.
func TestModWindowHomeOpensInTheMiddle(t *testing.T) {
	for _, size := range centreSizes {
		e := newMapEditorFixture(t)
		e.Layout(size.X, size.Y)
		s := popupSpec{Title: "mod", At: e.personCorner(homeMod), Width: 620, Bounds: e.popupBounds(),
			Rows: []popupRow{{Kind: popupEdit, Label: "a"}, {Kind: popupEdit, Label: "b"}}}
		e.openCentred(popObject, &s)
		if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !nearCentre(r, size) {
			t.Errorf("%dx%d: the mod window stands at %v", size.X, size.Y, r)
		}
		dragged := image.Pt(40, 90)
		e.places[e.storeRow(popObject)] = popupStanding{at: dragged, placed: true, slack: true}
		s.At = e.personCorner(homeMod)
		e.openCentred(popObject, &s)
		if s.At != dragged {
			t.Errorf("%dx%d: a dragged mod window reopens at %v, want %v", size.X, size.Y, s.At, dragged)
		}
	}
}

// Every home a list or an object window opens from stands it in the middle of
// the editor window at two window sizes, through keepInRoom, the route every
// home popup's spec takes.
func TestEveryPopupHomeOpensInTheMiddle(t *testing.T) {
	homes := []struct {
		name string
		id   editorPopupID
		home popupHome
	}{
		{"list on the map", popList, homeListMap}, {"list in the window", popList, homeListWindow},
		{"object on the map", popObject, homeObjectMap}, {"object in the window", popObject, homeObjectWindow},
		{"message", popObject, homeMessage}, {"trigger", popObject, homeTrigger},
		{"person", popObject, homePerson}, {"mod", popObject, homeMod},
	}
	for _, size := range centreSizes {
		for _, h := range homes {
			e := newMapEditorFixture(t)
			e.Layout(size.X, size.Y)
			s := popupSpec{Title: h.name, At: e.placedCorner(h.id, h.home), Width: 420, Bounds: e.popupBounds(),
				Rows: []popupRow{{Kind: popupEdit, Label: "a"}, {Kind: popupEdit, Label: "b"}, {Kind: popupEdit, Label: "c"}}}
			corner := s.At
			e.keepInRoom(h.id, &s)
			if r := layoutPopup(s, e.font, e.bodyFont()).Rect; !nearCentre(r, size) {
				t.Errorf("%dx%d %s: the window from home corner %v stands at %v", size.X, size.Y, h.name, corner, r)
			}
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
	p := title.Min.Add(image.Pt(10, 4))
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
// Players list opens in the middle, the player chosen from it opens at the
// list's corner plus the title row's height and popupPad, so the list's
// title bar stays uncovered and takes a press. A lone window stays in the
// middle and a dragged one keeps its place.
func TestWindowOpenedFromAnotherOpensOffsetFromIt(t *testing.T) {
	for _, size := range centreSizes {
		e, _ := newModesEditor(t, "full")
		e.Layout(size.X, size.Y)
		// The fixture parks both windows off the screen; these stand as opened.
		e.places[popList], e.places[popObject] = popupStanding{}, popupStanding{}
		if err := e.Key("players"); err != nil {
			t.Fatal(err)
		}
		ls, ok := e.listSpec()
		if !ok {
			t.Fatalf("%dx%d: the Players list is not open", size.X, size.Y)
		}
		os, _, ok := e.objectSpec()
		if !ok {
			t.Fatalf("%dx%d: the player window is not open", size.X, size.Y)
		}
		list := layoutPopup(ls, e.font, e.bodyFont())
		player := layoutPopup(os, e.font, e.bodyFont())
		step := list.Title.Dy() + popupPad
		if want := list.Rect.Min.Add(image.Pt(step, step)); player.Rect.Min != want {
			t.Errorf("%dx%d: the player window's corner is %v, want the list's %v plus %d", size.X, size.Y, player.Rect.Min, list.Rect.Min, step)
		}
		if player.Rect.Min.Y < list.Title.Max.Y {
			t.Errorf("%dx%d: the player window %v covers the list's title bar %v", size.X, size.Y, player.Rect, list.Title)
		}
		if at := list.Title.Min.Add(image.Pt(4, 4)); e.chromeHit(at) != popList {
			t.Errorf("%dx%d: a press at %v on the list's title bar reaches %v, not the list", size.X, size.Y, at, e.chromeHit(at))
		}
		if !nearCentre(list.Rect, size) {
			t.Errorf("%dx%d: the first window stands at %v, not in the middle", size.X, size.Y, list.Rect)
		}

		// The player window closed and opened again stands at the same offset.
		_ = e.Key("players")
		_ = e.Key("players")
		if os2, _, ok := e.objectSpec(); !ok || layoutPopup(os2, e.font, e.bodyFont()).Rect.Min != player.Rect.Min {
			t.Errorf("%dx%d: the reopened player window is not at %v", size.X, size.Y, player.Rect.Min)
		}

		// A dragged player window keeps its place beside the list.
		dragged := image.Pt(40, 90)
		e.places[popObject] = popupStanding{at: dragged, placed: true, slack: true}
		os, _, _ = e.objectSpec()
		if os.At != dragged {
			t.Errorf("%dx%d: a dragged player window stands at %v, want %v", size.X, size.Y, os.At, dragged)
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
