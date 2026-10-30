package game

import (
	"image"

	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondInnPatron is one inn seat's person as the page shows it: the
// statistics card's subject, the full figure and the frames of the seat.
// A person whose campaign template does not resolve has no card or figure.
type secondInnPatron struct {
	known   bool
	subject ui.PanelSubject
	figure  *image.RGBA
	frames  []image.Image
	// mask is the figure's slot map and info each worn slot's item text.
	mask *ui.SlotMask
	info [12][]string
}

// secondInnPatronKey is a seated person of one page.
type secondInnPatronKey struct {
	room *town.RoomSpec
	npc  int
}

// innPatron projects the resolved campaign actor through the shared card,
// seat art and member portrait composers (R2-SESSION-203, R2-SESSION-213).
func (t *townScreen) innPatron(r *town.RoomSpec, page *screenart.Set, npc int) *secondInnPatron {
	seats := r.Seats
	key := secondInnPatronKey{room: r, npc: npc}
	if p := t.composed.patrons[key]; p != nil {
		return p
	}
	p := &secondInnPatron{frames: page.Pictures(seats.Actor)}
	if t.composed.patrons == nil {
		t.composed.patrons = map[secondInnPatronKey]*secondInnPatron{}
	}
	t.composed.patrons[key] = p
	tbl := t.in.Table
	if tbl == nil {
		return p
	}
	member, ok := t.campaignActor(npc)
	if !ok {
		return p
	}
	if seats.Patron != "" {
		if src := t.in.Archives.artSource(); src != nil {
			if frames, err := screenArtOf(src).Bind(r.Name, r.Art, seats.Patron, member.Class); err == nil && len(frames) > 0 {
				p.frames = frames
			}
		}
	}
	// A patron is no player character: the card withholds its weight and
	// its experience, and its figure draws without the hero's backdrop.
	member.PlayerCharacter = false
	p.known = true
	p.subject = partyPanelSubject(member, tbl, t.in.Words)
	p.subject.Char.ExperienceWithheld = true
	p.subject.Unplaced = true
	// The sheet's heading (R2-ENGINE-581): the person's own name when it has
	// one, else Mercenary above the type's unitname entry for an actor of
	// flags 0x11, else the type's entry alone.
	p.subject.Name, p.subject.Char.Name = "", ""
	if name := secondInnPersonName(t.in, npc, member.Name); name != "" {
		p.subject.Name, p.subject.Char.Name = name, name
	} else if p.subject.OriginalPanel.Flags&0x11 != 0 {
		p.subject.Role = t.in.Words.PanelCaptions[mainSlotMercenary]
	}
	if t.in.Archives != nil {
		portrait := composeMemberPortrait(t.in.Archives.Containers, t.in.Units, 0, equipmentFromSlots(member.Worn), member)
		p.figure, p.mask = portrait.Figure, portrait.SlotMask
	}
	p.info = t.in.wornItemLines(member, nil)
	return p
}

// secondInnPersonName names a controlled actor from its resolved member, or
// an ordinary key of 1000 or more from npcnames (DIV-4087, DIV-4199).
func secondInnPersonName(in *InstallResources, npc int, resolved ...string) string {
	if _, controlled := tableEdition(in.Table).PartyIdentity(npc); controlled {
		if len(resolved) > 0 && resolved[0] != "" {
			return resolved[0]
		}
		return in.localizedNPCName(npc-1, "")
	}
	if npc < 1000 {
		return ""
	}
	return in.localizedNPCName(npc-1, "")
}

// selectedPatron is the selected seat's person, nil without a selection.
func (v townRoomView) selectedPatron(r *town.RoomSpec, page *screenart.Set) *secondInnPatron {
	list, _ := v.t.state().innSeats()
	seat := v.t.composed.seat
	if seat < 0 || seat >= len(list) {
		return nil
	}
	return v.t.innPatron(r, page, list[seat].npc)
}

// patron is the selected seat's person placed on the page: its statistics
// card in the generator's font and its full figure.
func (v townRoomView) patron(r *town.RoomSpec, page *screenart.Set) ui.RoomPerson {
	p := v.selectedPatron(r, page)
	if p == nil || !p.known {
		return ui.RoomPerson{}
	}
	var card ui.TownCharacterView
	if a := v.t.in.ChargenAssets; a != nil && a.Presentation != nil {
		card = a.Presentation.CardView(p.subject, image.Rectangle{}, image.Point{}, nil, nil)
	}
	return placeRoomPerson(r.Seats, page.Picture, card, p.figure)
}
