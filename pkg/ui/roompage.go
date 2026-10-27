package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// RoomPage is a tavern or inn page as both games paint it: its seats, its
// button rectangles and the selected person. The game builds it from the
// room's data; DrawRoomPage paints it and SeatAt and ButtonAt hit-test it.
type RoomPage struct {
	Seats   []RoomSeat
	Buttons []image.Rectangle
	// Phase is the animation phase an Animated seat shows.
	Phase  int
	Person RoomPerson
}

// RoomSeat is one seat: its back drawn opaque, then its actor's frame, both
// at Rect's corner and clipped to Rect, then its captions. A Hidden seat is
// not drawn but is still hit. A Boxed seat without a back draws the town
// shell's box instead.
type RoomSeat struct {
	Rect     image.Rectangle
	Hidden   bool
	Back     image.Image
	Boxed    bool
	Frames   []image.Image
	Frame    int
	Animated bool
	Captions []RoomCaption
}

// RoomCaptionPlace is where a seat caption stands in its seat.
type RoomCaptionPlace uint8

const (
	// RoomCaptionFit is centred in the town shell's caption ink, cut to fit.
	RoomCaptionFit RoomCaptionPlace = iota
	// RoomCaptionTopRight and RoomCaptionBottomLeft are in the card ink.
	RoomCaptionTopRight
	RoomCaptionBottomLeft
	// RoomCaptionMark is centred in red, clipped to the seat.
	RoomCaptionMark
)

// RoomCaption is one text of a seat; a nil font or empty text draws none.
type RoomCaption struct {
	Text  string
	Font  *text.Font
	Place RoomCaptionPlace
}

// RoomPerson is the selected seat's person: its statistics card, placed,
// and its figure. Picture, when set, is the card and figure already drawn
// by RenderRoomPerson and pasted instead, Text the glyphs it was drawn with.
type RoomPerson struct {
	Card    *TownCharacterView
	Figure  RoomFigure
	Picture *image.RGBA
	Text    []text.DrawCall
}

// RoomFigure is a person's full figure clipped to Rect: centred in Rect when
// Centre is set, else at Rect's corner, then moved by Offset.
type RoomFigure struct {
	Picture *image.RGBA
	Rect    image.Rectangle
	Offset  image.Point
	Centre  bool
}

// Origin is where a figure of size stands: its top-left corner.
func (f RoomFigure) Origin(size image.Point) image.Point {
	at := f.Rect.Min
	if f.Centre {
		at = at.Add(f.Rect.Size().Sub(size).Div(2))
	}
	return at.Add(f.Offset)
}

func (f RoomFigure) draw(dst *image.RGBA) {
	if f.Picture == nil {
		return
	}
	if f.Centre {
		// The inventory doll's own composite, as every centred figure.
		drawInventoryPictureShifted(dst, f.Rect, f.Picture, f.Offset)
		return
	}
	b := f.Picture.Bounds()
	at := f.Origin(b.Size())
	dr := image.Rectangle{Min: at, Max: at.Add(b.Size())}.Intersect(f.Rect)
	draw.Draw(dst, dr, f.Picture, b.Min.Add(dr.Min.Sub(at)), draw.Over)
}

// bounds is the area the person's card and figure cover.
func (p RoomPerson) bounds() image.Rectangle {
	r := p.Figure.Rect
	if p.Card != nil {
		r = r.Union(p.Card.paneRect()).Union(p.Card.cardRect())
	}
	return r
}

// DrawRoomPerson draws the person's card and then its figure.
func DrawRoomPerson(dst *image.RGBA, p RoomPerson) {
	if p.Card != nil {
		DrawCharacterPaneBody(dst, *p.Card)
	}
	p.Figure.draw(dst)
}

// RenderRoomPerson draws the person once onto a transparent canvas of the
// area it covers; nil when it covers none.
func RenderRoomPerson(p RoomPerson) *image.RGBA {
	r := p.bounds()
	if r.Empty() {
		return nil
	}
	dst := image.NewRGBA(r)
	DrawRoomPerson(dst, p)
	return dst
}

// DrawRoomPage paints the seats in index order and then the person.
func DrawRoomPage(dst *image.RGBA, p RoomPage) {
	for _, s := range p.Seats {
		if !s.Hidden {
			drawRoomSeat(dst, s, p.Phase)
		}
	}
	if pic := p.Person.Picture; pic != nil {
		draw.Draw(dst, pic.Bounds(), pic, pic.Bounds().Min, draw.Over)
		text.Append(p.Person.Text, 0, 0)
		return
	}
	DrawRoomPerson(dst, p.Person)
}

func drawRoomSeat(dst *image.RGBA, s RoomSeat, phase int) {
	r := s.Rect
	switch {
	case s.Back != nil:
		draw.Draw(dst, r, s.Back, s.Back.Bounds().Min, draw.Src)
	case s.Boxed:
		drawTownShellBox(dst, r, false)
	}
	if n := len(s.Frames); n > 0 {
		i := s.Frame
		if s.Animated {
			i = phase
		}
		if i %= n; i < 0 {
			i += n
		}
		if pic := s.Frames[i]; pic != nil {
			b := pic.Bounds()
			placed := b.Add(r.Min.Sub(b.Min))
			if clip := r.Intersect(placed); !clip.Empty() {
				draw.Draw(dst, clip, pic, b.Min.Add(clip.Min.Sub(placed.Min)), draw.Over)
			}
		}
	}
	for _, c := range s.Captions {
		drawRoomCaption(dst, c, r)
	}
}

var roomCaptionMarkInk = color.RGBA{R: 255, A: 255}

func drawRoomCaption(dst *image.RGBA, c RoomCaption, r image.Rectangle) {
	if c.Font == nil || c.Text == "" {
		return
	}
	switch c.Place {
	case RoomCaptionFit:
		DrawTownCaption(dst, c.Font, c.Text, r)
	case RoomCaptionTopRight, RoomCaptionBottomLeft:
		drawTavernCardText(dst, c.Font, c.Text, r, c.Place == RoomCaptionTopRight, c.Place == RoomCaptionBottomLeft)
	case RoomCaptionMark:
		w, _ := c.Font.Measure(c.Text)
		c.Font.Draw(dst.SubImage(r).(*image.RGBA), c.Text, r.Min.X+(r.Dx()-w)/2, r.Min.Y+(r.Dy()-c.Font.Height())/2, roomCaptionMarkInk)
	}
}

// SeatAt answers the first seat whose rectangle holds pt, hidden or not; -1
// for none.
func (p RoomPage) SeatAt(pt image.Point) int {
	for i, s := range p.Seats {
		if pt.In(s.Rect) {
			return i
		}
	}
	return -1
}

// ButtonAt answers the first button whose rectangle holds pt, -1 for none.
func (p RoomPage) ButtonAt(pt image.Point) int {
	for i, r := range p.Buttons {
		if pt.In(r) {
			return i
		}
	}
	return -1
}
