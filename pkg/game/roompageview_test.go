package game

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

func roomPageTestPicture(w, h int, c color.RGBA) *image.RGBA {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(m, m.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return m
}

// roomPageTestRoom is a page of three 10x20 seats a row from first, two
// rows climbing by 20, a card at card and a figure at the corner of
// (0,40,30,80) moved by figure.
func roomPageTestRoom(first town.Rect, card town.Rect, figure town.Point) *town.RoomSpec {
	return &town.RoomSpec{
		Seats: &town.SeatSpec{First: first, Step: town.Point{10, 0}, RowStep: town.Point{0, -20}, Columns: 3, Rows: 2,
			Back: "talk-back", HireBack: "hire-back",
			Card:   &town.SeatCard{Rect: card, Art: "board"},
			Figure: &town.SeatFigure{Rect: town.Rect{0, 40, 30, 80}, Offset: figure}},
		Buttons: []town.ButtonSpec{{Name: "exit", Rect: town.Rect{100, 0, 120, 10}}},
	}
}

var (
	roomPageHireInk = color.RGBA{R: 0x30, A: 0xff}
	roomPageTalkInk = color.RGBA{G: 0x30, A: 0xff}
	roomPageFigInk  = color.RGBA{B: 0x90, A: 0xff}
)

func roomPageTestArt(name string) image.Image {
	switch name {
	case "hire-back":
		return roomPageTestPicture(10, 20, roomPageHireInk)
	case "talk-back":
		return roomPageTestPicture(10, 20, roomPageTalkInk)
	}
	return nil
}

func paintTestRoomPage(r *town.RoomSpec) (*image.RGBA, ui.RoomPage) {
	p := buildRoomPage(r, roomPageTestArt, 4, 1)
	p.Person = placeRoomPerson(r.Seats, roomPageTestArt, ui.TownCharacterView{}, roomPageTestPicture(4, 4, roomPageFigInk))
	dst := image.NewRGBA(image.Rect(0, 0, 160, 120))
	ui.DrawRoomPage(dst, p)
	return dst, p
}

// TestRoomPageMovesWithItsRoomData moves the first seat and the figure
// offset in the room data: the painted seats and figure and the hit test
// follow, and the hire seat draws the hire back.
func TestRoomPageMovesWithItsRoomData(t *testing.T) {
	a, pa := paintTestRoomPage(roomPageTestRoom(town.Rect{40, 60, 50, 80}, town.Rect{0, 0, 30, 40}, town.Point{2, 3}))
	b, pb := paintTestRoomPage(roomPageTestRoom(town.Rect{60, 90, 70, 110}, town.Rect{0, 0, 30, 40}, town.Point{6, 7}))
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("moving the seats and the figure in the room data left the page unchanged")
	}
	if a.RGBAAt(40, 60) != roomPageHireInk || a.RGBAAt(50, 60) != roomPageTalkInk || a.RGBAAt(40, 40) != roomPageTalkInk {
		t.Fatalf("seats 0, 1 and 3 at (40,60) (50,60) (40,40) = %v %v %v", a.RGBAAt(40, 60), a.RGBAAt(50, 60), a.RGBAAt(40, 40))
	}
	if b.RGBAAt(40, 60) == roomPageHireInk || b.RGBAAt(60, 90) != roomPageHireInk {
		t.Fatalf("moved seat 0 at (40,60) %v, at (60,90) %v", b.RGBAAt(40, 60), b.RGBAAt(60, 90))
	}
	if a.RGBAAt(2, 43) != roomPageFigInk || b.RGBAAt(2, 43) == roomPageFigInk || b.RGBAAt(6, 47) != roomPageFigInk {
		t.Fatalf("figure corner (2,43) %v / %v, moved (6,47) %v", a.RGBAAt(2, 43), b.RGBAAt(2, 43), b.RGBAAt(6, 47))
	}
	if got := pa.SeatAt(image.Pt(45, 45)); got != 3 {
		t.Fatalf("seat at (45,45) = %d, want 3", got)
	}
	if got := pb.SeatAt(image.Pt(45, 45)); got != -1 {
		t.Fatalf("moved page's seat at (45,45) = %d, want none", got)
	}
	if got := pb.SeatAt(image.Pt(65, 75)); got != 3 {
		t.Fatalf("moved page's seat at (65,75) = %d, want 3", got)
	}
	if len(pa.Seats) != 4 {
		t.Fatalf("page holds %d seats, want the 4 seated", len(pa.Seats))
	}
	if got := pa.ButtonAt(image.Pt(110, 5)); got != 0 {
		t.Fatalf("button at (110,5) = %d, want 0", got)
	}
}

// TestRoomPersonCardFollowsItsRoomData moves the card rectangle and offset.
func TestRoomPersonCardFollowsItsRoomData(t *testing.T) {
	r := roomPageTestRoom(town.Rect{40, 60, 50, 80}, town.Rect{0, 0, 30, 40}, town.Point{})
	r.Seats.Card.Offset = town.Point{12, 0}
	card := ui.TownCharacterView{HasSubject: true}
	p := placeRoomPerson(r.Seats, roomPageTestArt, card, nil)
	if p.Card == nil || p.Card.PaneRect != image.Rect(0, 0, 30, 40) || p.Card.CardOffset != image.Pt(12, 0) || !p.Card.Statistics {
		t.Fatalf("placed card = %+v", p.Card)
	}
	r.Seats.Card.Rect, r.Seats.Card.Offset = town.Rect{5, 6, 35, 46}, town.Point{3, 1}
	if q := placeRoomPerson(r.Seats, roomPageTestArt, card, nil); q.Card.PaneRect != image.Rect(5, 6, 35, 46) || q.Card.CardOffset != image.Pt(3, 1) {
		t.Fatalf("moved card = %v %v", q.Card.PaneRect, q.Card.CardOffset)
	}
	if q := placeRoomPerson(r.Seats, roomPageTestArt, ui.TownCharacterView{}, nil); q.Card != nil {
		t.Fatal("a person without a subject shows a card")
	}
}

// TestFirstTavernRoomCarriesItsPageGeometry pins the first game's tavern
// room data: 48x64 seats six a row from the bottom-left, the card moved
// right by 12 over its board and the figure centred and moved by (8,1).
func TestFirstTavernRoomCarriesItsPageGeometry(t *testing.T) {
	r := ROM1TownDescription().Room("tavern")
	if r == nil || r.Seats == nil {
		t.Fatal("the first game's tavern room lays out no seats")
	}
	p := buildRoomPage(r, nil, 18, 0)
	for i, want := range map[int]image.Rectangle{0: image.Rect(176, 416, 224, 480), 5: image.Rect(416, 416, 464, 480),
		6: image.Rect(176, 352, 224, 416), 17: image.Rect(416, 288, 464, 352)} {
		if got := p.Seats[i].Rect; got != want {
			t.Errorf("seat %d = %v, want %v", i, got, want)
		}
	}
	if n := len(buildRoomPage(r, nil, 30, 0).Seats); n != 18 {
		t.Errorf("the tavern lays out %d seats, want TOWN-467's 18", n)
	}
	person := placeRoomPerson(r.Seats, nil, ui.TownCharacterView{HasSubject: true}, nil)
	if c := person.Card; c == nil || c.PaneRect != image.Rect(0, 0, 160, 238) || c.CardOffset != image.Pt(12, 0) {
		t.Fatalf("tavern card = %+v", c)
	}
	f := person.Figure
	if f.Rect != image.Rect(0, 238, 160, 480) || !f.Centre || f.Origin(image.Pt(100, 200)) != image.Pt(30+8, 259+1) {
		t.Fatalf("tavern figure = %+v, origin of 100x200 %v", f, f.Origin(image.Pt(100, 200)))
	}
	if r.Seats.Back != "man-back-talk" || r.Seats.HireBack != "man-back" || r.Seats.Card.Art != "left-stats" || r.Seats.Card.Seam != "left-stats-seam" {
		t.Fatalf("tavern seat art = %q %q %q %q", r.Seats.Back, r.Seats.HireBack, r.Seats.Card.Art, r.Seats.Card.Seam)
	}
}
