package ui

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"sync"
	"testing"
)

// firstTavernPage fills a hand-built tavern view's page as the game builds
// it from the first game's tavern room data; a view of another kind, or one
// whose page is set, is answered unchanged.
func firstTavernPage(v TownSurfaceView) TownSurfaceView {
	if v.Page.Regions == nil {
		v.Page.Regions = map[string][]image.Rectangle{}
		for _, r := range testSurfaceDescription().Rooms {
			if r.Name != "school" {
				continue
			}
			for _, region := range r.Regions {
				for _, rect := range region.Rects {
					v.Page.Regions[region.Name] = append(v.Page.Regions[region.Name], image.Rect(rect[0], rect[1], rect[2], rect[3]))
				}
			}
		}
	}
	if len(v.Page.Buttons) == 0 {
		for i := range v.Buttons {
			v.Page.Buttons = append(v.Page.Buttons, TownSurfaceButtonRect(v.Kind, i))
		}
	}
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
		font := v.CardFont
		if font == nil {
			font = v.Font
		}
		frames := c.Frames
		if len(frames) == 0 && c.Picture != nil {
			frames = []image.Image{c.Picture}
		}
		seat := RoomSeat{Rect: r, Back: back, Boxed: true, Animated: c.Selected, Frames: frames, Hidden: v.RosterUnpainted,
			Captions: []RoomCaption{{Text: c.Price, Font: font, Place: RoomCaptionTopRight}, {Text: c.Detail, Font: font, Place: RoomCaptionBottomLeft}}}
		hired := v.HiredLabel
		if hired == "" {
			hired = AuthoredWords().TavernHired
		}
		if c.Hired {
			seat.Captions = append(seat.Captions, RoomCaption{Text: hired, Font: font, Place: RoomCaptionMark})
		}
		v.Page.Seats = append(v.Page.Seats, seat)
	}
	if c := v.Candidate; c.HasSubject {
		c.Statistics, c.PaneRect, c.CardOffset = true, image.Rect(0, 0, 160, 238), image.Pt(12, 0)
		c.StatsPane = TownPane{Body: art.LeftStats, Seam: art.LeftStatsSeam}
		v.Page.Person.Card = &c
	}
	v.Page.Person.Figure = RoomFigure{Picture: v.Candidate.Figure, Rect: image.Rect(0, 238, 160, 480), Offset: image.Pt(8, 1), Centre: true}
	v.Page.Person.Picture, v.Page.Person.Text = v.CandidatePixels, v.CandidateText
	v.Page.Scene = v.Scene
	for _, layer := range testSurfaceDescription().Rooms[0].Scene.Backdrop {
		pictures := map[string]image.Image{"center": art.Center, "left-stats": art.LeftStats, "left-stats-seam": art.LeftStatsSeam, "left-picture": art.LeftPicture, "left-picture-seam": art.LeftPictureSeam}
		v.Page.Backdrop = append(v.Page.Backdrop, RoomBackdrop{Picture: pictures[layer.Art], Rect: image.Rect(layer.Rect[0], layer.Rect[1], layer.Rect[2], layer.Rect[3]), Scene: layer.Scene, Over: layer.Over})
	}
	return v
}

type testSurfaceData struct {
	Rooms []struct {
		Name    string
		Regions []struct {
			Name  string
			Rects [][4]int
		}
		Buttons []struct{ Rect [4]int }
		Scene   struct {
			Backdrop []struct {
				Art, Scene string
				Rect       [4]int
				Over       bool
			}
		}
	}
}

var surfaceDataOnce sync.Once
var surfaceData testSurfaceData

var townSurfaceButtonWells = map[TownSurfaceKind][]image.Rectangle{TownSurfaceSchool: {TownSurfaceButtonWell(TownSurfaceSchool, 0), TownSurfaceButtonWell(TownSurfaceSchool, 1)}}

var schoolPanelRects = [2]image.Rectangle{SchoolPanelRect(0), SchoolPanelRect(1)}

func testSurfaceDescription() testSurfaceData {
	surfaceDataOnce.Do(func() {
		raw, err := os.ReadFile("../game/towns/rom1.json")
		if err != nil {
			panic(err)
		}
		if err := json.Unmarshal(raw, &surfaceData); err != nil {
			panic(err)
		}
	})
	return surfaceData
}

func TownSurfaceButtonRect(kind TownSurfaceKind, i int) image.Rectangle {
	name := map[TownSurfaceKind]string{TownSurfaceTavern: "tavern", TownSurfaceSchool: "school"}[kind]
	for _, r := range testSurfaceDescription().Rooms {
		if r.Name == name && i >= 0 && i < len(r.Buttons) {
			b := r.Buttons[i].Rect
			return image.Rect(b[0], b[1], b[2], b[3])
		}
	}
	return image.Rectangle{}
}

func townSurfaceButtonRect(kind TownSurfaceKind, i int) image.Rectangle {
	return TownSurfaceButtonRect(kind, i)
}
func TownSurfaceButtonWell(kind TownSurfaceKind, i int) image.Rectangle {
	origin := TownUpperRegion.Min
	if kind == TownSurfaceTavern {
		origin = TownWideUpperRegion.Min
	}
	return TownSurfaceButtonRect(kind, i).Sub(origin)
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

func SchoolSkillRect(class, slot int) image.Rectangle {
	if class < 0 || class > 1 || slot < 0 || slot > 4 {
		return image.Rectangle{}
	}
	return firstTavernPage(TownSurfaceView{Kind: TownSurfaceSchool}).Page.Region([]string{"fighter", "mage"}[class], slot)
}
func SchoolPanelRect(class int) image.Rectangle {
	return firstTavernPage(TownSurfaceView{Kind: TownSurfaceSchool}).Page.Region("panels", class)
}
func townSurfaceCellRect(i int) image.Rectangle {
	return firstTavernPage(TownSurfaceView{Kind: TownSurfaceSchool}).Page.Region("cells", i)
}
