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
func (t *townScreen) secondPaneParty() []mapload.PartyMember {
	return t.sess.Carried
}

// heroFace is the shown member's face, composed on first use after a room
// entry; nil without a party.
func (t *townScreen) heroFace() *secondHeroFace {
	party := t.secondPaneParty()
	if len(party) == 0 {
		return nil
	}
	i := townPickerIndex(party, t.shopMember)
	if f := t.composed.hero; f != nil && f.member == i {
		return f
	}
	member := party[i]
	f := &secondHeroFace{member: i, subject: t.in.composeMemberFace(&member, uint32(i+1), nil)}
	f.info = t.in.wornItemLines(member, nil)
	t.composed.hero = f
	return f
}

// heroFaceView is the shown member's face as the room panes draw it: his
// figure, its slot map and his worn item text; nothing without a party.
func (t *townScreen) heroFaceView(int) (*image.RGBA, *ui.SlotMask, [12][]string) {
	f := t.heroFace()
	if f == nil {
		return nil, nil, [12][]string{}
	}
	return f.subject.Figure, f.subject.SlotMask, f.info
}

// TownSquarePane acts on a click on a room pane's corner: the mode corner
// swaps the figure and the statistics card, the picker arrows step the shown
// member.
func (t *townScreen) TownSquarePane(c ui.CharacterPaneCorner) {
	switch c {
	case ui.CharacterPaneMode:
		t.townStats = !t.townStats
	case ui.CharacterPanePrev:
		t.stepMemberOver(t.secondPaneParty(), -1)
	case ui.CharacterPaneNext:
		t.stepMemberOver(t.secondPaneParty(), +1)
	}
}

// paneSquareView fills the room pane of v from the room spec r and the
// selected patron's figure as a doll.
func (t *townScreen) paneSquareView(v *ui.TownSquareView, r *town.RoomSpec, page *townRoomArt) {
	if r == nil {
		return
	}
	if r.Pane != nil {
		v.Pane = true
		v.Hero, v.HeroSlotMask, v.HeroSlotInfo = t.heroPane(r.Pane)
	}
	if r.Seats == nil || page == nil {
		return
	}
	view := townRoomView{t: t, room: r.Name}
	p := view.selectedPatron(r, page.page)
	if p == nil || !p.known || p.figure == nil || p.mask == nil {
		return
	}
	f := placeRoomPerson(r.Seats, nil, ui.TownCharacterView{}, p.figure).Figure
	if f.Rect.Empty() {
		return
	}
	v.Dolls = append(v.Dolls, ui.TownSquareDoll{At: f.Origin(image.Pt(p.mask.W, p.mask.H)), Clip: f.Rect, Mask: p.mask, Info: p.info})
}
