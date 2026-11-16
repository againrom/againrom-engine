package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// standAside drags the open object and list popups by their titles, through
// the editor's own drag, from the middle of the window to the corners of the
// map area they stood at before every popup opened in the middle: the
// object popup at the top right, the list at the top left. A test that works
// on the map behind them clicks there after; the drag is kept, as a map
// maker's is. A popup docked in a panel, covered at its title or dragged
// already stays.
func standAside(t *testing.T, e *MapEditor) {
	t.Helper()
	if err := moveAside(e); err != nil {
		t.Fatal(err)
	}
}

// moveAside is standAside's drag; an error when a popup did not move.
func moveAside(e *MapEditor) error {
	for _, c := range []struct {
		id    editorPopupID
		right bool
	}{{popObject, true}, {popList, false}} {
		s, ok := e.movableSpec(c.id)
		if !ok {
			continue
		}
		r := layoutPopup(s, e.font, e.bodyFont()).Rect
		p := r.Min.Add(image.Pt(10, popupTopBand+2))
		if e.chromeHit(p) != c.id || e.places[e.storeRow(c.id)].slack {
			continue
		}
		room := e.mapArea()
		// The object popup's corner is 420 wide in from the right.
		to := image.Pt(room.Min.X+8, e.toolbarRect().Max.Y+8)
		if c.right {
			to.X = room.Max.X - 8 - 420
		}
		to.X = max(room.Min.X, min(to.X, room.Max.X-r.Dx()))
		to.Y = max(room.Min.Y, min(to.Y, room.Max.Y-r.Dy()))
		if to == r.Min {
			continue
		}
		q := p.Add(to.Sub(r.Min))
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
		e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
		if s, ok := e.movableSpec(c.id); ok && layoutPopup(s, e.font, e.bodyFont()).Rect.Min == r.Min {
			return fmt.Errorf("popup %d did not move aside from %v", c.id, r.Min)
		}
	}
	return nil
}
