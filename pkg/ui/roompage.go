package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/text"
)

// RoomPage is a town room's ordered paint layers and controls. The game
// builds its data once; DrawRoomPage paints it and ControlAt hit-tests it.
type RoomPage struct {
	Layers                []RoomLayer
	Panel                 RoomPanel
	Cells                 []TownSurfaceCell
	Font                  *text.Font
	School                *RoomSchool
	BeforePane, AfterPane []RoomHit
	Pane                  *TownCharacterView
	Book                  bool
	HitScene, PaneBlocks  bool
	HidePerson            bool
	ViewInputs            bool
	Pointer               RoomPointer
	Seats                 []RoomSeat
	Buttons               []image.Rectangle
	Backdrop              []RoomBackdrop
	Scene                 RoomScene
	Regions               map[string][]image.Rectangle
	// Phase is the animation phase an Animated seat shows.
	Phase  int
	Person RoomPerson
}

func (p RoomPage) Region(name string, i int) image.Rectangle {
	if rows := p.Regions[name]; i >= 0 && i < len(rows) {
		return rows[i]
	}
	return image.Rectangle{}
}

// RoomBackdrop is a page picture or a named scene group in paint order.
type RoomBackdrop struct {
	Picture image.Image
	Rect    image.Rectangle
	Scene   string
	Over    bool
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

// DrawRoomPage paints the page's layers in order, using the shared widgets.
func DrawRoomPage(dst *image.RGBA, p RoomPage, groups ...string) {
	if len(groups) > 0 {
		if groups[0] == "backdrop" {
			drawRoomBackdrop(dst, p)
		} else {
			drawRoomContents(dst, p)
		}
		return
	}
	layers := p.Layers
	if layers == nil {
		layers = []RoomLayer{{Kind: RoomLayerBackdrop}, {Kind: RoomLayerContent}, {Kind: RoomLayerButtons}}
	}
	for _, layer := range layers {
		switch layer.Kind {
		case RoomLayerFill:
			draw.Draw(dst, dst.Bounds(), &image.Uniform{C: layer.Ink}, image.Point{}, draw.Src)
		case RoomLayerBox:
			drawTownShellBox(dst, layer.Rect, false)
		case RoomLayerPicture:
			if layer.Picture != nil {
				draw.Draw(dst, layer.Rect, layer.Picture, layer.Picture.Bounds().Min, draw.Src)
			}
		case RoomLayerBackdrop:
			drawRoomBackdrop(dst, p)
		case RoomLayerScene:
			if p.Scene != nil {
				p.Scene.Paint(dst, layer.Scene)
			}
		case RoomLayerPanel:
			p.Panel.resolve(p.Pointer).drawBody(dst)
		case RoomLayerTitle:
			drawTownShellText(dst, p.Font, layer.Text, layer.Rect, layer.Ink)
		case RoomLayerContent:
			drawRoomContents(dst, p)
		case RoomLayerCells:
			drawRoomCells(dst, p)
		case RoomLayerSchool:
			if p.School != nil {
				p.School.draw(dst, p)
			}
		case RoomLayerButtons:
			p.Panel.resolve(p.Pointer).drawButtons(dst, p.Panel.Style.font(p.Font))
		}
	}
}

func drawRoomBackdrop(dst *image.RGBA, p RoomPage) {
	if p.Scene != nil {
		p.Scene.Paint(dst, "back")
	}
	for _, layer := range p.Backdrop {
		if layer.Scene != "" && p.Scene != nil {
			p.Scene.Paint(dst, layer.Scene)
		}
		if layer.Picture != nil {
			op := draw.Src
			if layer.Over {
				op = draw.Over
			}
			draw.Draw(dst, layer.Rect, layer.Picture, layer.Picture.Bounds().Min, op)
		}
	}
}

func drawRoomContents(dst *image.RGBA, p RoomPage) {
	for _, s := range p.Seats {
		if !s.Hidden {
			drawRoomSeat(dst, s, p.Phase)
		}
	}
	if !p.HidePerson {
		if pic := p.Person.Picture; pic != nil {
			draw.Draw(dst, pic.Bounds(), pic, pic.Bounds().Min, draw.Over)
			text.Append(p.Person.Text, 0, 0)
		} else {
			DrawRoomPerson(dst, p.Person)
		}
	}
	if p.Scene != nil {
		p.Scene.Paint(dst, "front")
	}
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
