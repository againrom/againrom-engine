package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// firstTavernPage fills a hand-built tavern view's page as the game builds
// it from the first game's tavern room data; a view of another kind, or one
// whose page is set, is answered unchanged.
func firstTavernPage(v TownSurfaceView) TownSurfaceView {
	if v.Kind != TownSurfaceTavern || len(v.Page.Seats) > 0 || v.Page.Person.Card != nil || v.Page.Person.Figure.Rect != (image.Rectangle{}) {
		return v
	}
	art := v.TavernArt
	if art == nil {
		art = &TownTavernArt{}
	}
	for i, c := range v.Cells {
		if i >= 18 {
			break
		}
		back := art.ManBack
		if c.TalkOnly {
			back = art.ManBackTalk
		}
		r := image.Rect(176, 416, 224, 480).Add(image.Pt(48*(i%6), -64*(i/6)))
		v.Page.Seats = append(v.Page.Seats, RoomSeat{Rect: r, Back: back})
	}
	if c := v.Candidate; c.HasSubject {
		c.Statistics, c.PaneRect, c.CardOffset = true, image.Rect(0, 0, 160, 238), image.Pt(12, 0)
		c.StatsPane = TownPane{Body: art.LeftStats, Seam: art.LeftStatsSeam}
		v.Page.Person.Card = &c
	}
	v.Page.Person.Figure = RoomFigure{Picture: v.Candidate.Figure, Rect: image.Rect(0, 238, 160, 480), Offset: image.Pt(8, 1), Centre: true}
	v.Page.Person.Picture, v.Page.Person.Text = v.CandidatePixels, v.CandidateText
	return v
}

// townSurfaceCellRectFor is cell i's rectangle: the tavern's seat i of its
// page, the school's skill cell. It is the one reader of a built page's seat
// rectangles outside DrawRoomPage and the hit test, and only a test calls it.
func townSurfaceCellRectFor(v TownSurfaceView, i int) image.Rectangle {
	if v.Kind == TownSurfaceTavern {
		if i < 0 || i >= len(v.Page.Seats) {
			return image.Rectangle{}
		}
		return v.Page.Seats[i].Rect
	}
	return townSurfaceCellRect(i)
}

// renderTavernCandidate is the tavern's cached drawing of candidate c.
func renderTavernCandidate(c TownCharacterView, art *TownTavernArt) *image.RGBA {
	return RenderRoomPerson(firstTavernPage(TownSurfaceView{Kind: TownSurfaceTavern, TavernArt: art, Candidate: c}).Page.Person)
}

// testRoomPage is a page of two seats a row of 10x20 from first, two rows
// climbing by 20, a figure rectangle at (0,0,30,40) and two buttons.
func testRoomPage(first image.Point, figure image.Point, centre bool) RoomPage {
	var p RoomPage
	for i := 0; i < 4; i++ {
		r := image.Rect(0, 0, 10, 20).Add(first).Add(image.Pt(10*(i%2), -20*(i/2)))
		p.Seats = append(p.Seats, RoomSeat{Rect: r, Back: uniform(10, 20, color.RGBA{B: 0x40, A: 0xff})})
	}
	p.Buttons = []image.Rectangle{image.Rect(100, 0, 120, 10), image.Rect(100, 10, 120, 20)}
	pic := uniform(4, 4, color.RGBA{R: 0x80, A: 0xff})
	p.Person.Figure = RoomFigure{Picture: pic, Rect: image.Rect(0, 30, 30, 70), Offset: figure, Centre: centre}
	return p
}

func paintRoomPage(p RoomPage) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, 160, 120))
	DrawRoomPage(dst, p)
	return dst
}

// TestRoomPageFollowsItsData moves a seat and the figure in the page data:
// the painted pixels and the hit test move with them.
func TestRoomPageFollowsItsData(t *testing.T) {
	base := testRoomPage(image.Pt(40, 60), image.Pt(2, 3), false)
	moved := testRoomPage(image.Pt(60, 80), image.Pt(7, 9), false)
	a, b := paintRoomPage(base), paintRoomPage(moved)
	if bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("moving the seats and the figure in the data left the page unchanged")
	}
	back := color.RGBA{B: 0x40, A: 0xff}
	if a.RGBAAt(40, 60) != back || b.RGBAAt(40, 60) == back || b.RGBAAt(60, 80) != back {
		t.Fatalf("seat 0 back at (40,60) %v / %v, moved (60,80) %v", a.RGBAAt(40, 60), b.RGBAAt(40, 60), b.RGBAAt(60, 80))
	}
	fig := color.RGBA{R: 0x80, A: 0xff}
	if a.RGBAAt(2, 33) != fig || b.RGBAAt(2, 33) == fig || b.RGBAAt(7, 39) != fig {
		t.Fatalf("figure corner (2,33) %v / %v, moved (7,39) %v", a.RGBAAt(2, 33), b.RGBAAt(2, 33), b.RGBAAt(7, 39))
	}
	if got := base.SeatAt(image.Pt(41, 61)); got != 0 {
		t.Fatalf("base seat at (41,61) = %d, want 0", got)
	}
	if got := moved.SeatAt(image.Pt(41, 61)); got != -1 {
		t.Fatalf("moved seat at (41,61) = %d, want none", got)
	}
	if got := moved.SeatAt(image.Pt(71, 61)); got != 3 {
		t.Fatalf("moved seat at (71,61) = %d, want 3 (second row, second column)", got)
	}
	if got := moved.ButtonAt(image.Pt(110, 15)); got != 1 {
		t.Fatalf("button at (110,15) = %d, want 1", got)
	}
	centred := testRoomPage(image.Pt(40, 60), image.Pt(2, 3), true)
	if got, want := centred.Person.Figure.Origin(image.Pt(4, 4)), image.Pt(13+2, 30+18+3); got != want {
		t.Fatalf("centred figure origin = %v, want %v", got, want)
	}
	if c := paintRoomPage(centred); c.RGBAAt(15, 51) != fig || c.RGBAAt(2, 33) == fig {
		t.Fatalf("centred figure corner (15,51) %v, top-left (2,33) %v", c.RGBAAt(15, 51), c.RGBAAt(2, 33))
	}
}

// TestRoomPageHiddenSeatIsHitButNotDrawn keeps a hidden seat out of the
// paint and in the hit test.
func TestRoomPageHiddenSeatIsHitButNotDrawn(t *testing.T) {
	p := testRoomPage(image.Pt(40, 60), image.Point{}, false)
	p.Seats[0].Hidden = true
	if got := paintRoomPage(p).RGBAAt(40, 60); got.A != 0 {
		t.Fatalf("hidden seat painted %v", got)
	}
	if got := p.SeatAt(image.Pt(40, 60)); got != 0 {
		t.Fatalf("hidden seat hit = %d, want 0", got)
	}
}
