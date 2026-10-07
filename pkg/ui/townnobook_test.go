package ui

import "testing"

type fakeNoBookTown struct{ fakeStatsSurfaceTown }

func (f *fakeNoBookTown) TownSurface() TownSurfaceView {
	v := f.fakeStatsSurfaceTown.TownSurface()
	v.Hero.NoBook = true
	return v
}

// A room with no spellbook has no book corner and no book key; the mode key
// still works there.
func TestTownRoomWithoutASpellbookHasNoBookControl(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.Layout(640, 480)
	town := &fakeNoBookTown{}
	a.SetTown(town)
	a.flow.showTown("")
	for _, key := range []string{"tab", "b", "q"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if len(town.surfaceClicks) != 1 || town.surfaceClicks[0].Kind != TownSurfaceControlMode {
		t.Fatalf("keys reached %+v, want only the mode control", town.surfaceClicks)
	}
	v := town.TownSurface()
	box := CharacterPaneCornerRect(v.Hero.paneRect(), CharacterPaneBook)
	if c, ok := TownSurfaceControlAt(v, box.Min.Add(box.Size().Div(2))); ok && c.Kind == TownSurfaceControlBook {
		t.Error("the book corner still answers a press")
	}
	v.Hero.NoBook = false
	if c, ok := TownSurfaceControlAt(v, box.Min.Add(box.Size().Div(2))); !ok || c.Kind != TownSurfaceControlBook {
		t.Errorf("control %+v,%v at the book corner with a book, want the book control: the check is vacuous", c, ok)
	}
}
