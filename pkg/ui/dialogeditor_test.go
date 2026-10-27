package ui

import (
	"image"
	"testing"
	"time"
)

// A card dragged by its grip shows a drop marker while held and lands where
// the marker stood; the draft changes, nothing is written.
func TestDialogCardDragShowsTheDropMarker(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	e.doc.Edits = &dialogueInputEdits{edits}
	e.openMessage(1)
	_, grip, ok := e.CardRects(2)
	first, _, shown := e.CardRects(0)
	if !ok || !shown || grip.Empty() {
		t.Fatal("cards 1 and 3 are not shown with a grip")
	}
	at := grip.Min.Add(grip.Size().Div(2))
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: at.X, CursorY: first.Min.Y + 2, PrimaryDown: true}, time.Time{})
	s, _, _ := e.objectSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	if l.Drop != 0 || l.Marker.Empty() || l.Marker.Max.Y < first.Min.Y-3 || l.Marker.Min.Y > first.Min.Y+3 {
		t.Fatalf("the held card shows drop %d at %v, want 0 above %v", l.Drop, l.Marker, first)
	}
	pic := e.ChromeImage()
	ground := pic.RGBAAt(l.Marker.Min.X+40, l.Marker.Min.Y-6)
	if pic.RGBAAt(l.Marker.Min.X+40, l.Marker.Min.Y+1) == ground {
		t.Fatal("the drop marker is not painted")
	}
	e.Pointer(Input{CursorX: at.X, CursorY: first.Min.Y + 2}, time.Time{})
	var got []string
	for _, p := range e.obj.messageParts {
		got = append(got, p.Body)
	}
	if len(got) != 3 || got[0] != "Third" || got[1] != "First" || got[2] != "Second" || !e.obj.messageChanged {
		t.Fatalf("the drop left the cards %q", got)
	}
	if e.dragCard != 0 || e.drag != popNone {
		t.Fatal("the drag outlived the release")
	}
}

// A card held over its own place shows no marker and stays.
func TestDialogCardDropOnItsPlaceMovesNothing(t *testing.T) {
	e, edits := selectedTextEditor(t, panelFont(), image.Pt(1280, 800))
	e.doc.Edits = &dialogueInputEdits{edits}
	e.openMessage(1)
	_, grip, _ := e.CardRects(1)
	at := grip.Min.Add(grip.Size().Div(2))
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y + 1, PrimaryDown: true}, time.Time{})
	s, _, _ := e.objectSpec()
	if l := layoutPopup(s, e.font, e.bodyFont()); !l.Marker.Empty() {
		t.Fatalf("a card over its own place shows a marker at %v", l.Marker)
	}
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y + 1}, time.Time{})
	if e.obj.messageChanged || e.obj.messageParts[1].Body != "Second" {
		t.Fatal("a drop on the card's own place changed the draft")
	}
}
