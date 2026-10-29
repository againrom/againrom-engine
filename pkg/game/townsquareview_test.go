package game

import (
	"bytes"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// No shipped town square carries a pane: the square of either game shows
// none, and the second game's inn and shop carry theirs.
func TestShippedSquaresCarryNoPane(t *testing.T) {
	descriptions := []*town.Description{ROM1TownDescription()}
	for id := 1; id <= 3; id++ {
		descriptions = append(descriptions, secondTownDescription(id))
	}
	for _, d := range descriptions {
		if d == nil {
			t.Fatal("a shipped town has no description")
		}
		if p := d.Pane(d.Square.Name); p != nil {
			t.Errorf("town %s: square pane %+v", d.Town, p)
		}
	}
	for id := 1; id <= 3; id++ {
		d := secondTownDescription(id)
		if d.Pane("inn") != d.Room("inn").Pane || d.Pane("inn") == nil || d.Pane("shop") != d.Room("shop").Pane {
			t.Errorf("town %d: the room panes do not answer through the description", id)
		}
	}
}

// squarePane is a first-game mod edit that gives the square a pane.
func squarePane(pane map[string]any) func(d map[string]any) {
	return func(d map[string]any) { d["square"].(map[string]any)["pane"] = pane }
}

// A pane the first game's town data adds to its square appears on the
// square's view and in its frame; without it the square has none. A square
// pane with the spellbook corner is refused where the mod layers it.
func TestFirstSquarePaneIsTheSquareData(t *testing.T) {
	region := []any{480, 238, 640, 480}
	frames := map[bool][]byte{}
	for _, withPane := range []bool{false, true} {
		var edit func(d map[string]any)
		if withPane {
			edit = squarePane(map[string]any{"rect": region, "font": "install"})
		}
		f, _, err := layeredFront(t, minimalEastMods(t, edit), "east-town")
		if err != nil {
			t.Fatalf("pane %v: %v", withPane, err)
		}
		if err := checkEastTown(f, false); err != nil {
			t.Fatalf("pane %v: %v", withPane, err)
		}
		s := f.TownScreen().(*townScreen)
		v := s.TownSquareView()
		if v.Scene == nil {
			t.Fatalf("pane %v: the square has no scene", withPane)
		}
		if v.Pane != withPane || withPane && (v.Hero.PaneRect != ui.TownCharacterRegion || !v.Hero.HasSubject) {
			t.Errorf("pane %v: view pane %v rect %v subject %v", withPane, v.Pane, v.Hero.PaneRect, v.Hero.HasSubject)
		}
		if len(v.Dolls) != 0 {
			t.Errorf("pane %v: the square shows %d dolls", withPane, len(v.Dolls))
		}
		frames[withPane] = ui.ComposeTownSquare(v).Pix
	}
	if bytes.Equal(frames[false], frames[true]) {
		t.Error("the square's pane draws the same frame as no pane")
	}
	want := "square pane has the spellbook corner or a book_offset; the square has no spellbook"
	for name, pane := range map[string]map[string]any{
		"book":        {"rect": region, "book": true},
		"book offset": {"rect": region, "book_offset": []any{0, 2}},
	} {
		if _, _, err := layeredFront(t, minimalEastMods(t, squarePane(pane)), "east-town"); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v\nwant %q", name, err, want)
		}
	}
	moved := map[string]any{"rect": []any{0, 238, 160, 480}}
	if _, _, err := layeredFront(t, minimalEastMods(t, squarePane(moved)), "east-town"); err == nil ||
		!strings.Contains(err.Error(), "square pane rectangle") {
		t.Errorf("moved square pane: %v", err)
	}
	if _, _, err := layeredFront(t, minimalEastMods(t, squarePane(map[string]any{"rect": []any{0, 0, 0, 0}})), "east-town"); err == nil ||
		!strings.Contains(err.Error(), "square: the pane rectangle") {
		t.Errorf("empty square pane: %v", err)
	}
}

// withRoom is a copy of d whose named room edit changed.
func withRoom(d *town.Description, room string, edit func(r *town.RoomSpec)) *town.Description {
	c := *d
	c.Rooms = slices.Clone(d.Rooms)
	edit(c.Room(room))
	return &c
}

// The second game's inn shows the pane and the selected patron's doll its
// room data names: a pane removed from the data leaves the doll and drops
// the pane, the seats removed drop the doll.
func TestSecondSquareViewFollowsTheRoomData(t *testing.T) {
	f, app, screen := secondCampaignFixtureFiles(t, false)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2})
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok {
		t.Fatal("no composed town 2")
	}
	screen.enterSeats()
	seats, _ := c.innSeats()
	if len(seats) == 0 || screen.composed.seat != 0 {
		t.Fatalf("seats %v selected %d", seats, screen.composed.seat)
	}
	plant := func(d *town.Description) {
		if screen.composed.patrons == nil {
			screen.composed.patrons = map[secondInnPatronKey]*secondInnPatron{}
		}
		screen.composed.patrons[secondInnPatronKey{room: d.Room("inn"), npc: seats[0].npc}] = &secondInnPatron{known: true,
			figure: image.NewRGBA(image.Rect(0, 0, 4, 4)), mask: &ui.SlotMask{W: 4, H: 4, Slot: make([]uint8, 16)}}
	}
	plant(d)
	if v := screen.TownSquareView(); !v.Pane || len(v.Dolls) != 1 || v.Hero.PaneRect != d.Room("inn").Pane.Rect.Rectangle() {
		t.Fatalf("shipped inn: pane %v rect %v dolls %d", v.Pane, v.Hero.PaneRect, len(v.Dolls))
	}
	for _, c := range []struct {
		name  string
		edit  func(r *town.RoomSpec)
		pane  bool
		dolls int
	}{
		{"shipped", func(*town.RoomSpec) {}, true, 1},
		{"no pane", func(r *town.RoomSpec) { r.Pane = nil }, false, 1},
		{"no seats", func(r *town.RoomSpec) { r.Seats = nil }, true, 0},
	} {
		dd := withRoom(d, "inn", c.edit)
		plant(dd)
		var v ui.TownSquareView
		screen.paneSquareView(&v, dd, "inn")
		if v.Pane != c.pane || len(v.Dolls) != c.dolls {
			t.Errorf("%s: pane %v dolls %d, want %v and %d", c.name, v.Pane, len(v.Dolls), c.pane, c.dolls)
		}
	}
}
