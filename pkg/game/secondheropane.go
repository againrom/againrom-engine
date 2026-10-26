package game

import (
	"image"

	"againrom/pkg/mapload"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondHeroFace is the shown party member as the second game's room panes
// show him: his figure, its slot map and his worn item text, composed once
// per room entry.
type secondHeroFace struct {
	member  int
	subject ui.InventorySubject
	info    [12][]string
}

// secondPaneParty is the party the panes show: the campaign's party, the one
// the mission takes. The shop's bag is its own member (DIV-3043).
func (t *secondCampaignScreen) secondPaneParty() []mapload.PartyMember {
	return t.session.Carried
}

// heroFace is the shown member's face, composed on first use after a room
// entry; nil without a party.
func (t *secondCampaignScreen) heroFace() *secondHeroFace {
	party := t.secondPaneParty()
	if len(party) == 0 {
		return nil
	}
	i := townPickerIndex(party, t.shopMember)
	if f := t.composed.hero; f != nil && f.member == i {
		return f
	}
	member := party[i]
	f := &secondHeroFace{member: i, subject: t.install.composeMemberFace(&member, uint32(i+1), nil)}
	f.info = t.install.wornItemLines(member, nil)
	t.composed.hero = f
	return f
}

// heroPane is the character pane spec p shows: the shown member's figure or
// statistics card, the card built as the inn's patron card is, with the
// picker and mode corners and the book corner when p has one.
func (t *secondCampaignScreen) heroPane(p *town.PaneSpec) (ui.TownCharacterView, *ui.SlotMask, [12][]string) {
	var info [12][]string
	var panes TownCharacterPaneArt
	var corners *ui.CharacterPaneCornerArt
	if t.paneArt != nil {
		panes, corners = t.paneArt()
	}
	var presentation *ui.ChargenPresentation
	if a := t.install.ChargenAssets; a != nil {
		presentation = a.Presentation
	}
	party := t.secondPaneParty()
	var subject ui.PanelSubject
	i := townPickerIndex(party, t.shopMember)
	if len(party) > 0 {
		subject = partyPanelSubject(party[i], t.install.Table, t.install.Words)
		subject.Unplaced, subject.Selected = true, 1
	}
	v := presentation.CardView(subject, p.Rect.Rectangle(), image.Point{}, panes.Stats.Body, panes.Stats.Seam)
	v.HasSubject = len(party) > 0
	v.Statistics = t.composed.heroStats
	v.FigurePane, v.CornerArt, v.Session = panes.Figure, corners, ui.CharacterPaneShop
	v.NoBook, v.BookOpen = !p.Book, p.Book && t.shopBook
	v.Member, v.MemberCount = townPickerPosition(party, i)
	if presentation == nil {
		v.Font, v.CardFont = t.install.Font.Value(), t.install.Font.Value()
	}
	var mask *ui.SlotMask
	if f := t.heroFace(); f != nil {
		v.Figure, mask, info = f.subject.Figure, f.subject.SlotMask, f.info
	}
	return v, mask, info
}

// TownSquarePane acts on a click on a room pane's corner: the mode corner
// swaps the figure and the statistics card, the picker arrows step the shown
// member.
func (t *secondCampaignScreen) TownSquarePane(c ui.CharacterPaneCorner) {
	switch c {
	case ui.CharacterPaneMode:
		t.composed.heroStats = !t.composed.heroStats
	case ui.CharacterPanePrev:
		t.shopStepMember(-1)
	case ui.CharacterPaneNext:
		t.shopStepMember(+1)
	}
}

// paneSquareView fills the room pane of v from the room spec r and the
// selected patron's figure as a doll.
func (t *secondCampaignScreen) paneSquareView(v *ui.TownSquareView, r *town.RoomSpec, page *secondTownArt) {
	if r == nil {
		return
	}
	if r.Pane != nil {
		v.Pane = true
		v.Hero, v.HeroSlotMask, v.HeroSlotInfo = t.heroPane(r.Pane)
	}
	if r.Seats == nil || r.Seats.Figure == nil || page == nil {
		return
	}
	view := secondTownView{t: t, room: r.Name}
	p := view.selectedPatron(r, page.page)
	if p == nil || !p.known || p.figure == nil || p.mask == nil {
		return
	}
	f := r.Seats.Figure
	clip := f.Rect.Rectangle()
	v.Dolls = append(v.Dolls, ui.TownSquareDoll{At: clip.Min.Add(f.Offset.Pt()), Clip: clip, Mask: p.mask, Info: p.info})
}
