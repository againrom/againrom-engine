package game

import (
	"image"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// buildRoomPage is a tavern or inn page as the room's data lays it out: the
// first n seats of its seat grid, the first hires of them a hire actor's,
// each drawing its back by the page art pic answers, and its buttons' hit
// rectangles. Both games' pages are built here and painted by
// ui.DrawRoomPage.
func buildRoomPage(r *town.RoomSpec, pic func(string) image.Image, n, hires int) ui.RoomPage {
	var p ui.RoomPage
	if r == nil {
		return p
	}
	if s := r.Seats; s != nil {
		for i := 0; i < n && i < s.Count(); i++ {
			back := s.Back
			if i < hires && s.HireBack != "" {
				back = s.HireBack
			}
			p.Seats = append(p.Seats, ui.RoomSeat{Rect: s.Seat(i), Back: pagePicture(pic, back)})
		}
	}
	for _, b := range r.Buttons {
		p.Buttons = append(p.Buttons, b.Rect.Rectangle())
	}
	return p
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

// TavernPageLayout is the first game's tavern page of n seats, the first
// hires a mercenary's, as its tavern room lays it out over the loaded art.
func (in *InstallResources) TavernPageLayout(n, hires int) ui.RoomPage {
	var pic func(string) image.Image
	if a := in.TownTavernArt.Value(); a != nil {
		pic = a.Picture
	}
	return buildRoomPage(ROM1TownDescription().Room("tavern"), pic, n, hires)
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
	v.Page = buildRoomPage(t.tavernRoom(), t.tavernPicture(), n, hires)
	v.Page.Person = t.tavernPerson(v.Candidate)
	v.Page.Person.Picture, v.Page.Person.Text = v.CandidatePixels, v.CandidateText
}

// renderTavernPerson draws candidate c once over the tavern's own art; nil
// without it.
func (t *townScreen) renderTavernPerson(c ui.TownCharacterView) *image.RGBA {
	if t.tavernArt() == nil || !c.HasSubject && c.Figure == nil {
		return nil
	}
	return ui.RenderRoomPerson(t.tavernPerson(c))
}
