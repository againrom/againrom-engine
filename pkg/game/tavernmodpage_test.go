package game

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

func tavernModFixture(t *testing.T, change func(r *town.RoomSpec)) (*townScreen, ui.TownSurfaceView, ui.TownSurfaceView) {
	t.Helper()
	f := shellFrontEnd()
	art, err := LoadTownTavernArt(ROM1TownDescription(), townTavernSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownTavernArt = resolved(art, nil)
	s := f.townUI
	shipped := s.TownSurface()
	book := s.townBook()
	d := *s.townDescription()
	d.Rooms = slices.Clone(d.Rooms)
	r := d.Room("tavern")
	copied := *r
	copied.Buttons = slices.Clone(r.Buttons)
	seats := *r.Seats
	copied.Seats = &seats
	change(&copied)
	for i := range d.Rooms {
		if d.Rooms[i].Name == "tavern" {
			d.Rooms[i] = copied
		}
	}
	f.Town.camp.Towns = &townBook{catalog: book.catalog, descriptions: map[string]*town.Description{"rom1": &d}}
	return s, shipped, s.TownSurface()
}

// A mod town's tavern room sets the page's buttons and seat appearance.
func TestModTavernPageTakesButtonsAndSeatAppearanceFromItsTown(t *testing.T) {
	s, shipped, got := tavernModFixture(t, func(r *town.RoomSpec) {
		r.Buttons = r.Buttons[:2]
		a := *r.Seats.Appearance
		a.Boxed = !a.Boxed
		r.Seats.Appearance = &a
	})
	if n := len(got.Page.Buttons); n != 2 {
		t.Errorf("page has %d buttons, want the mod room's 2", n)
	}
	if n := len(s.tavernButtons()); n != 2 {
		t.Errorf("tavernButtons has %d, want 2", n)
	}
	if len(shipped.Page.Buttons) <= 2 {
		t.Fatalf("shipped page has %d buttons", len(shipped.Page.Buttons))
	}
	if bytes.Equal(ui.ComposeTownSurface(shipped).Pix, ui.ComposeTownSurface(got).Pix) {
		t.Error("a mod seat appearance changed no pixel")
	}
	if !got.Page.Seats[0].Rect.Eq(shipped.Page.Seats[0].Rect) {
		t.Error("seat layout moved without the mod naming it")
	}
}

// A room naming no buttons or appearance keeps the game's own.
func TestModTavernPageFallsBackWhereTheTownNamesNothing(t *testing.T) {
	_, shipped, got := tavernModFixture(t, func(r *town.RoomSpec) {
		r.Buttons, r.Seats.Appearance = nil, nil
	})
	if !reflect.DeepEqual(shipped.Page.Buttons, got.Page.Buttons) {
		t.Errorf("buttons %v, want %v", got.Page.Buttons, shipped.Page.Buttons)
	}
	if !bytes.Equal(ui.ComposeTownSurface(shipped).Pix, ui.ComposeTownSurface(got).Pix) {
		t.Error("fallback page differs from the shipped one")
	}
}

// A town with no tavern room, as the second game's, reads the game's room.
func TestTavernRoomWithoutTheTownsOwnIsTheGamesRoom(t *testing.T) {
	f := shellFrontEnd()
	s := f.townUI
	want := ROM1TownDescription().Room("tavern")
	book := s.townBook()
	d := *s.townDescription()
	d.Rooms = slices.DeleteFunc(slices.Clone(d.Rooms), func(r town.RoomSpec) bool { return r.Name == "tavern" })
	f.Town.camp.Towns = &townBook{catalog: book.catalog, descriptions: map[string]*town.Description{"rom1": &d}}
	if got := s.tavernRoom(); got != want {
		t.Errorf("tavernRoom %p, want the game's %p", got, want)
	}
	s.defaults = townDescriptions["rom2-town1"]
	if got := s.tavernRoom(); !reflect.DeepEqual(got.Buttons, want.Buttons) {
		t.Error("second game's tavern buttons moved")
	}
}

// The shipped first town's page equals the one the instruments lay out.
func TestShippedTavernPageLayoutIsUnchanged(t *testing.T) {
	f := shellFrontEnd()
	art, err := LoadTownTavernArt(ROM1TownDescription(), townTavernSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownTavernArt = resolved(art, nil)
	v := f.townUI.TownSurface()
	n := len(v.Cells)
	want := f.InstallResources.TavernPageLayout(n, 0)
	if !reflect.DeepEqual(v.Page.Buttons, want.Buttons) || !reflect.DeepEqual(v.Page.Regions, want.Regions) {
		t.Error("shipped tavern page layout differs from the instrument layout")
	}
	for i := range v.Page.Seats {
		if i < len(want.Seats) && !v.Page.Seats[i].Rect.Eq(want.Seats[i].Rect) {
			t.Errorf("seat %d rect %v, want %v", i, v.Page.Seats[i].Rect, want.Seats[i].Rect)
		}
	}
}
