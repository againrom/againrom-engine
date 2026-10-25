package game

import (
	"image"
	"image/draw"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/screenart"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondInnPatron is one inn seat's person as the page shows it: the
// statistics card's subject, the full figure and the frames of the seat.
// A person whose Humans row does not resolve has no card and no figure.
type secondInnPatron struct {
	known   bool
	subject ui.PanelSubject
	figure  *image.RGBA
	frames  []image.Image
}

// secondInnPatronKey is a seated person of one page.
type secondInnPatronKey struct {
	room *town.RoomSpec
	npc  int
}

// innPatron is the person of inn NPC key npc, built once per screen: the
// Humans row whose server id is the key, built as a party member by the one
// character builder, carded by the one party projection and drawn by the
// one figure composer (R2-SESSION-203, R2-SESSION-213).
func (t *secondCampaignScreen) innPatron(r *town.RoomSpec, page *screenart.Set, npc int) *secondInnPatron {
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
	tbl := t.install.Table
	if tbl == nil || tbl.Humans == nil {
		return p
	}
	member, ok := mapload.CharacterMember(tbl, mapload.CharacterSpec{Row: data.FindHumanByServerID(tbl.Humans, int32(npc))}, nil)
	if !ok {
		return p
	}
	if seats.Patron != "" {
		if src := t.install.Archives.artSource(); src != nil {
			if frames, err := screenArtOf(src).Bind(r.Name, r.Art, seats.Patron, member.Class); err == nil && len(frames) > 0 {
				p.frames = frames
			}
		}
	}
	// A patron is no player character: the card withholds its weight and
	// its experience, and its figure draws without the hero's backdrop.
	member.PlayerCharacter = false
	p.known = true
	p.subject = partyPanelSubject(member, tbl, t.install.Words)
	p.subject.Char.ExperienceWithheld = true
	p.subject.Unplaced = true
	// The card names the class by its unitname entry and, above it, the
	// person by its npcnames entry when the two differ.
	p.subject.Name, p.subject.Char.Name = "", ""
	class, _ := ui.PanelSubjectName(p.subject)
	if role := secondInnPersonName(tbl, npc); role != class {
		p.subject.Role = role
	}
	fig := memberFigureID(member)
	if t.install.Archives != nil {
		p.figure, _ = composeUnitFigure(t.install.Archives.Containers, equipmentFromSlots(member.Worn), fig)
	}
	return p
}

// secondInnPersonName is the npcnames entry inn NPC key npc names: a key
// below 1000 names the entry of its last two digits, any other key its own
// entry.
func secondInnPersonName(tbl *mapload.Table, npc int) string {
	if npc > 0 && npc < 1000 {
		npc %= 100
	}
	if npc < 0 || npc >= len(tbl.NPCNames) {
		return ""
	}
	return tbl.NPCNames[npc]
}

// selectedPatron is the selected seat's person, nil without a selection.
func (v secondTownView) selectedPatron(r *town.RoomSpec, page *screenart.Set) *secondInnPatron {
	list, _ := v.t.state().innSeats()
	seat := v.t.composed.seat
	if seat < 0 || seat >= len(list) {
		return nil
	}
	return v.t.innPatron(r, page, list[seat].npc)
}

// paintPatron draws the selected seat's statistics card and full figure.
func (v secondTownView) paintPatron(dst *image.RGBA, r *town.RoomSpec, page *screenart.Set) {
	seats := r.Seats
	p := v.selectedPatron(r, page)
	if p == nil || !p.known {
		return
	}
	if c := seats.Card; c != nil {
		if a := v.t.install.ChargenAssets; a != nil && a.Presentation != nil {
			// The card board must be an RGBA picture; a nil one stays a nil image.
			var body image.Image
			if pic := rgbaOf(page.Picture(c.Art)); pic != nil {
				body = pic
			}
			view := a.Presentation.CardView(p.subject, c.Rect.Rectangle(), c.Offset.Pt(), body, page.Picture(c.Seam))
			ui.DrawCharacterPaneBody(dst, view)
		}
	}
	if f := seats.Figure; f != nil && p.figure != nil {
		clip := f.Rect.Rectangle()
		at := clip.Min.Add(f.Offset.Pt())
		b := p.figure.Bounds()
		dr := image.Rectangle{Min: at, Max: at.Add(b.Size())}.Intersect(clip)
		draw.Draw(dst, dr, p.figure, b.Min.Add(dr.Min.Sub(at)), draw.Over)
	}
}
