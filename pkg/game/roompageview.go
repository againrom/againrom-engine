package game

import (
	"image"

	"againrom/pkg/render/text"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// buildRoomPage is a tavern or inn page as the room's data lays it out: the
// first n seats of its seat grid, the first hires of them a hire actor's,
// each drawing its back by the page art pic answers, and its buttons' hit
// rectangles. Both games' pages are built here and painted by
// ui.DrawRoomPage.
func buildRoomPage(r *town.RoomSpec, pic func(string) image.Image, n, hires int, contents ...roomPageContents) ui.RoomPage {
	var p ui.RoomPage
	var content roomPageContents
	if len(contents) > 0 {
		content = contents[0]
	}
	p.Person, p.Scene = content.person, content.scene
	p.Font = content.font
	if r == nil {
		return p
	}
	if s := r.Seats; s != nil {
		appearance := town.SeatAppearance{}
		if content.appearance != nil {
			appearance = *content.appearance
		}
		if s.Appearance != nil {
			appearance = *s.Appearance
		}
		for i := 0; i < n && i < s.Count(); i++ {
			back := s.Back
			if i < hires && s.HireBack != "" {
				back = s.HireBack
			}
			var row roomSeatInput
			if i < len(content.seats) {
				row = content.seats[i]
			}
			seat := ui.RoomSeat{Rect: s.Seat(i), Back: pagePicture(pic, back), Frames: row.frames, Frame: row.frame, Animated: appearance.Phase && row.selected, Hidden: row.hidden, Boxed: appearance.Boxed}
			captions := appearance.Captions
			if len(captions) == 0 {
				captions = []town.SeatCaption{{Value: "hired"}}
			}
			for _, caption := range captions {
				value := map[string]string{"price": row.price, "detail": row.detail}[caption.Value]
				if caption.Value == "hired" && row.hired {
					value = content.hired
				}
				seat.Captions = append(seat.Captions, ui.RoomCaption{Text: value, Font: content.font, Place: ui.RoomCaptionPlace(caption.Place)})
			}
			p.Seats = append(p.Seats, seat)
		}
	}
	for _, b := range r.Buttons {
		p.Buttons = append(p.Buttons, b.Rect.Rectangle())
	}
	p.Regions = map[string][]image.Rectangle{}
	for _, region := range r.Regions {
		for _, rect := range region.Rects {
			p.Regions[region.Name] = append(p.Regions[region.Name], rect.Rectangle())
		}
	}

	if r.Scene != nil {
		for _, layer := range r.Scene.Backdrop {
			p.Backdrop = append(p.Backdrop, ui.RoomBackdrop{Picture: pagePicture(pic, layer.Art), Rect: layer.Rect.Rectangle(), Scene: layer.Scene, Over: layer.Over})
		}
	}
	finishRoomPage(&p, content)
	return p
}

type roomPageContents struct {
	surface    *ui.TownRoomView
	buttons    []ui.RoomButton
	targets    []ui.RoomControl
	style      *ui.RoomButtonStyle
	seats      []roomSeatInput
	person     ui.RoomPerson
	scene      ui.RoomScene
	font       *text.Font
	hired      string
	appearance *town.SeatAppearance
}

type roomSeatInput struct {
	frames                  []image.Image
	frame                   int
	hidden, selected, hired bool
	price, detail           string
}

func surfaceRoomContents(v ui.TownSurfaceView) roomPageContents {
	c := roomPageContents{surface: &v}
	c.appearance = ROM1TownDescription().Room("tavern").Seats.Appearance
	font := v.CardFont
	if font == nil {
		font = v.Font
	}
	hired := v.HiredLabel
	if hired == "" {
		hired = ui.AuthoredWords().TavernHired
	}
	for _, cell := range v.Cells {
		s := roomSeatInput{hidden: v.RosterUnpainted, selected: cell.Selected, frames: cell.Frames, price: cell.Price, detail: cell.Detail, hired: cell.Hired}
		if len(s.frames) == 0 && cell.Picture != nil {
			s.frames = []image.Image{cell.Picture}
		}
		c.seats = append(c.seats, s)
	}
	c.font, c.hired = font, hired
	return c
}

// TownRoomButtonRect reads a retained room's control row for instruments.
func TownRoomButtonRect(kind ui.TownSurfaceKind, i int) image.Rectangle {
	name := map[ui.TownSurfaceKind]string{ui.TownSurfaceTavern: "tavern", ui.TownSurfaceSchool: "school"}[kind]
	r := ROM1TownDescription().Room(name)
	if r == nil || i < 0 || i >= len(r.Buttons) {
		return image.Rectangle{}
	}
	return r.Buttons[i].Rect.Rectangle()
}

func TownRoomButtonWell(kind ui.TownSurfaceKind, i int) image.Rectangle {
	r := TownRoomButtonRect(kind, i)
	if r.Empty() {
		return r
	}
	origin := ui.TownUpperRegion.Min
	if kind == ui.TownSurfaceTavern {
		origin = ui.TownWideUpperRegion.Min
	}
	return r.Sub(origin)
}

func TownSchoolRegion(name string, i int) image.Rectangle {
	return buildRoomPage(ROM1TownDescription().Room("school"), nil, 0, 0).Region(name, i)
}

// placeRoomPerson places the selected seat's person where the room's seat
// data puts it: card, a styled statistics card when it has a subject, over
// the card art, and figure in the figure rectangle.
func placeRoomPerson(s *town.SeatSpec, pic func(string) image.Image, card ui.TownCharacterView, figure *image.RGBA) ui.RoomPerson {
	var p ui.RoomPerson
	if s == nil {
		return p
	}
	if c := s.Card; c != nil && card.HasSubject {
		card.Statistics, card.PaneRect, card.CardOffset = true, c.Rect.Rectangle(), c.Offset.Pt()
		// The card board must be an RGBA picture; a nil one stays a nil image.
		card.StatsPane = ui.TownPane{Seam: pagePicture(pic, c.Seam)}
		if body := rgbaOf(pagePicture(pic, c.Art)); body != nil {
			card.StatsPane.Body = body
		}
		p.Card = &card
	}
	if f := s.Figure; f != nil {
		p.Figure = ui.RoomFigure{Picture: figure, Rect: f.Rect.Rectangle(), Offset: f.Offset.Pt(), Centre: f.Centre}
	}
	return p
}

// pagePicture is the page art of name, nil without a lookup or a name.
func pagePicture(pic func(string) image.Image, name string) image.Image {
	if pic == nil || name == "" {
		return nil
	}
	return pic(name)
}

// tavernRoom is the first game's tavern room: the town's own when it lays
// out seats, else the profile's, as for a mod town that names none.
func (t *townScreen) tavernRoom() *town.RoomSpec {
	if d := t.townDescription(); d != nil {
		if r := d.Room("tavern"); r != nil && r.Seats != nil {
			return r
		}
	}
	d := t.defaults
	if d == nil {
		d = ROM1TownDescription()
	}
	return d.Room("tavern")
}

// tavernButtons are the first game's tavern buttons in index order: the
// shipped first town's, whatever a mod tavern room's buttons name.
func (t *townScreen) tavernButtons() []town.ButtonSpec {
	if r := ROM1TownDescription().Room("tavern"); r != nil {
		return r.Buttons
	}
	return nil
}

// TavernPageLayout is the first game's tavern page of n seats, the first
// hires a mercenary's, as its tavern room lays it out over the loaded art.
func (in *InstallResources) TavernPageLayout(n, hires int) ui.RoomPage {
	seats := ROM1TownDescription().Room("tavern").Seats
	return in.SurfacePageLayout(ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, Cells: make([]ui.TownSurfaceCell, min(max(n, 0), seats.Count())), TavernArt: in.TownTavernArt.Value()}, hires)
}

func (in *InstallResources) SurfacePageLayout(v ui.TownSurfaceView, hires int) ui.RoomPage {
	var pic func(string) image.Image
	if a := in.TownTavernArt.Value(); a != nil {
		pic = a.Picture
	}
	room := map[ui.TownSurfaceKind]string{ui.TownSurfaceTavern: "tavern", ui.TownSurfaceSchool: "school"}[v.Kind]
	r := ROM1TownDescription().Room(room)
	c := surfaceRoomContents(v)
	if r != nil {
		c.person = placeRoomPerson(r.Seats, pic, v.Candidate, v.Candidate.Figure)
	}
	c.person.Picture, c.person.Text = v.CandidatePixels, v.CandidateText
	c.scene = v.Scene
	return buildRoomPage(r, pic, len(v.Cells), hires, c)
}

// tavernPicture is the tavern's page art lookup, nil without its art.
func (t *townScreen) tavernPicture() func(string) image.Image {
	if a := t.tavernArt(); a != nil {
		return a.Picture
	}
	return nil
}

// tavernPerson is candidate c placed on the tavern page.
func (t *townScreen) tavernPerson(c ui.TownCharacterView) ui.RoomPerson {
	var seats *town.SeatSpec
	if r := t.tavernRoom(); r != nil {
		seats = r.Seats
	}
	return placeRoomPerson(seats, t.tavernPicture(), c, c.Figure)
}

// setTavernPage sets v's page: the tavern's first n seats, the first hires
// of them a mercenary's, and v's candidate placed on it and drawn by the
// pixels and text v carries.
func (t *townScreen) setTavernPage(v *ui.TownSurfaceView, n, hires int) {
	c := surfaceRoomContents(*v)
	c.person = t.tavernPerson(v.Candidate)
	c.person.Picture, c.person.Text = v.CandidatePixels, v.CandidateText
	c.scene = t.roomView("tavern")
	r := *t.tavernRoom()
	r.Buttons = t.tavernButtons()
	if r.Scene == nil || r.Scene.Backdrop == nil {
		var scene town.SceneSpec
		if r.Scene != nil {
			scene = *r.Scene
		}
		scene.Backdrop = ROM1TownDescription().Room("tavern").Scene.Backdrop
		r.Scene = &scene
	}
	v.Page = buildRoomPage(&r, t.tavernPicture(), n, hires, c)
}

// renderTavernPerson draws candidate c once over the tavern's own art; nil
// without it.
func (t *townScreen) renderTavernPerson(c ui.TownCharacterView) *image.RGBA {
	if t.tavernArt() == nil || !c.HasSubject && c.Figure == nil {
		return nil
	}
	return ui.RenderRoomPerson(t.tavernPerson(c))
}

func finishRoomPage(p *ui.RoomPage, c roomPageContents) {
	if v := c.surface; v != nil {
		p.ViewInputs = true
		p.Scene = v.Scene
		p.Font, p.Cells = v.Font, v.Cells
		school := v.Kind == ui.TownSurfaceSchool && v.SchoolArt != nil
		tavern := v.Kind == ui.TownSurfaceTavern && v.TavernArt != nil
		command := tavern && len(v.Buttons) == 4 && v.TavernArt.CommandUpper != nil
		p.Layers = []ui.RoomLayer{{Kind: ui.RoomLayerFill, Ink: ui.TownRoomFill}}
		switch {
		case school && v.SchoolArt.Background != nil:
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerPicture, Rect: ui.TownContentRegion, Picture: v.SchoolArt.Background})
		case tavern && v.TavernArt.Center != nil:
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerBackdrop})
		default:
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerBox, Rect: ui.TownContentRegion})
		}
		if school {
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerScene, Scene: "movies"})
		}
		p.Panel = roomPagePanel(*p, *v, school, tavern, command)
		p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerPanel})
		if !school && !tavern {
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerTitle, Rect: ui.TownRoomTitleRegion, Text: v.Title, Ink: ui.TownRoomText})
		}
		if v.Kind == ui.TownSurfaceTavern {
			p.HidePerson = !tavern
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerContent})
		} else if !school {
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerCells})
		}
		if school {
			p.School = &ui.RoomSchool{Art: v.SchoolArt, Class: v.SchoolClass, ColumnFrame: v.SchoolColumnFrame,
				ColumnSet: v.SchoolColumnSet, IdleShine: v.SchoolIdleShine, IdleSlot: v.SchoolIdleSlot}
			p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerScene, Scene: "column"}, ui.RoomLayer{Kind: ui.RoomLayerSchool}, ui.RoomLayer{Kind: ui.RoomLayerScene, Scene: "diamond"})
		}
		p.Layers = append(p.Layers, ui.RoomLayer{Kind: ui.RoomLayerButtons})
		for i, b := range v.Buttons {
			if i < len(p.Buttons) {
				p.BeforePane = append(p.BeforePane, ui.RoomHit{Rect: p.Buttons[i], Enabled: b.Enabled,
					Control: ui.RoomControl{Surface: ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: i}}})
			}
		}
		if v.Kind == ui.TownSurfaceTavern {
			for i, s := range p.Seats {
				p.AfterPane = append(p.AfterPane, ui.RoomHit{Rect: s.Rect, Enabled: true,
					Control: ui.RoomControl{Surface: ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}}})
			}
		} else {
			for i, cell := range v.Cells {
				p.AfterPane = append(p.AfterPane, ui.RoomHit{Rect: p.Region("cells", i), Enabled: cell.Enabled,
					Control: ui.RoomControl{Surface: ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: i}}})
			}
		}
		return
	}
	p.Panel = ui.RoomPanel{Composition: ui.TownRoomEntryPanel, Style: c.style, TrackPointer: true, HoverInside: true}
	for _, b := range c.buttons {
		p.Panel.Composition.Plaques = append(p.Panel.Composition.Plaques, b.Rect)
		pair := [2]image.Image{b.Off, b.On}
		if b.On == nil {
			pair[1] = b.Off
		}
		p.Panel.Art.Plaques = append(p.Panel.Art.Plaques, pair)
		button := ui.PanelButton{}
		if b.Caption != "" {
			button.Captions = []string{b.Caption}
		}
		p.Panel.Buttons = append(p.Panel.Buttons, button)
	}
	for i, s := range p.Seats {
		p.AfterPane = append(p.AfterPane, ui.RoomHit{Rect: s.Rect, Enabled: true,
			Control: ui.RoomControl{Square: ui.TownSquareControl{Kind: ui.TownSquareControlSelect, Door: i + 1}}})
	}
	for i, target := range c.targets {
		if i < len(p.Buttons) {
			p.AfterPane = append(p.AfterPane, ui.RoomHit{Rect: p.Buttons[i], Enabled: target.Square.Kind != ui.TownSquareControlNone, Control: target})
		}
	}
}

func roomPagePanel(p ui.RoomPage, v ui.TownRoomView, school, tavern, command bool) ui.RoomPanel {
	panel := ui.RoomPanel{Composition: ui.TownRoomPanel, Style: v.ButtonStyle, TrackPointer: true, TwoLines: true, Values: v.Buttons}
	if v.Kind == ui.TownSurfaceSchool {
		panel.Composition.MissingBox = ui.TownWideUpperRegion
	}
	switch {
	case school && v.SchoolArt.Upper != nil:
		panel.Art.Body, panel.Art.Seam = v.SchoolArt.Upper, v.SchoolArt.UpperSeam
	case command:
		panel.Composition = ui.TownRoomShopPanel
		panel.Art.Body = v.TavernArt.CommandUpper
	case tavern && v.TavernArt.Upper != nil:
		panel.Art.Body, panel.Art.Seam = v.TavernArt.Upper, v.TavernArt.UpperSeam
	}
	for i := range v.Buttons {
		rect := image.Rectangle{}
		if i < len(p.Buttons) {
			rect = p.Buttons[i]
		}
		panel.Composition.Plaques = append(panel.Composition.Plaques, rect)
		var pair [2]image.Image
		pressedOnly := false
		switch {
		case school && i < len(v.SchoolArt.Buttons):
			pair = ui.WholeRoomPair(v.SchoolArt.Buttons[i])
		case command && i < len(v.TavernArt.CommandButtons):
			pair[1], pressedOnly = v.TavernArt.CommandButtons[i], true
		case tavern:
			artIndex := i
			if len(v.Buttons) == 4 && i > 0 {
				artIndex = i - 1
			}
			if artIndex < len(v.TavernArt.Buttons) {
				pair = ui.WholeRoomPair(v.TavernArt.Buttons[artIndex])
			}
		}
		panel.Art.Plaques = append(panel.Art.Plaques, pair)
		panel.PressedOnly = append(panel.PressedOnly, pressedOnly)
	}
	return panel
}
