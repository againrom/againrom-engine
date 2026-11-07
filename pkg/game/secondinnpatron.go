package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/rules"
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
	if d := t.townDescription(); d != nil && seats.Patron != "" {
		if frames := t.screenArtBundle(d).patron(r.Name, member.Class); len(frames) > 0 {
			p.frames = frames
		}
	}
	person := t.buildRoomPatron(member, npc, roomPersonProfiles["campaign"], nil)
	person.frames = p.frames
	*p = *person
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
func (v roomPageView) selectedPatron(r *town.RoomSpec, page *screenart.Set) *secondInnPatron {
	list, _ := v.t.state().innSeats()
	seat := v.t.composed.seat
	if seat < 0 || seat >= len(list) {
		return nil
	}
	return v.t.innPatron(r, page, list[seat].npc)
}

// patron is the selected seat's person placed on the page: its statistics
// card in the generator's font and its full figure.
func (v roomPageView) patron(r *town.RoomSpec, page *screenart.Set) ui.RoomPerson {
	p := v.selectedPatron(r, page)
	if p == nil || !p.known {
		return ui.RoomPerson{}
	}
	card := v.t.roomPersonCard(p.subject, p.figure, true)
	return placeRoomPerson(r.Seats, page.Picture, card, p.figure)
}

type roomPersonProfile struct {
	name                                                                                        string
	selected                                                                                    int
	fullSight, withheld, noPlayer, requireArchives, hoverMask, currentItems, keyFigure, compose bool
}

var roomPersonProfiles = map[string]roomPersonProfile{
	"campaign": {name: "actor", withheld: true, noPlayer: true, requireArchives: true, compose: true},
	"stock":    {name: "template", selected: 1, fullSight: true, hoverMask: true, currentItems: true, keyFigure: true, compose: true},
	"live":     {selected: 1},
}

type roomPersonHeading struct {
	name, role string
	index      int
	hasIndex   bool
}

var roomPersonNames = map[string]func(*townScreen, mapload.PartyMember, int, ui.PanelSubject) roomPersonHeading{
	"actor": func(t *townScreen, m mapload.PartyMember, key int, s ui.PanelSubject) roomPersonHeading {
		h := roomPersonHeading{name: secondInnPersonName(t.in, key, m.Name)}
		if h.name == "" && s.OriginalPanel.Flags&0x11 != 0 {
			h.role = t.in.Words.PanelCaptions[mainSlotMercenary]
		}
		return h
	},
	"template": func(t *townScreen, m mapload.PartyMember, key int, s ui.PanelSubject) roomPersonHeading {
		index := int(m.Class)
		if key > 2 {
			template := fmt.Sprintf("NPC%02d_%d", key, mercenaryLevel(t.sess.Town.Chapter()))
			if i := data.FindHumanByName(t.in.Table.Humans, template); i != data.NotFound {
				if d, err := data.NewHumanDef(template, t.in.Table.Humans.EntryParams(i)); err == nil {
					index = int(d.TypeID)
				}
			}
		}
		return roomPersonHeading{name: t.in.definitionName(m, index, &s.Words, ""), role: t.in.Words.PanelCaptions[mainSlotMercenary], index: index, hasIndex: true}
	},
}

// buildRoomPatron projects every seated member through the same subject,
// portrait and item producers; the source's profile preserves its card rules.
func (t *townScreen) buildRoomPatron(m mapload.PartyMember, key int, profile roomPersonProfile, figure *image.RGBA) *secondInnPatron {
	if profile.noPlayer {
		m.PlayerCharacter = false
	}
	p := &secondInnPatron{known: true, figure: figure}
	p.subject = partyPanelSubject(m, t.in.Table, t.in.Words)
	p.subject.Unplaced, p.subject.Selected = true, profile.selected
	if profile.withheld {
		p.subject.Char.ExperienceWithheld = true
	}
	if names := roomPersonNames[profile.name]; names != nil {
		h := names(t, m, key, p.subject)
		p.subject.Name, p.subject.Char.Name, p.subject.Role = h.name, h.name, h.role
		if h.hasIndex {
			p.subject.UnitNameIndex, p.subject.Char.UnitNameIndex = h.index, h.index
		}
	}
	if c := p.subject.Char; profile.fullSight && c.Mind+c.Reaction > 0 {
		if word, ok := rules.SightWordIn(int32(c.Mind), int32(c.Reaction), int32(c.Sight)); ok {
			p.subject.Char.Sight256 = word
		}
	}
	if profile.compose && (!profile.requireArchives || t.in.Archives != nil) {
		var src entrySource
		if t.in.Archives != nil {
			src = t.in.Archives.Containers
		}
		eq, id := equipmentFromSlots(m.Worn), uint32(0)
		if profile.currentItems {
			eq = mapload.EquipmentFromParty(m)
		}
		if profile.keyFigure {
			id = uint32(key)
		}
		portrait := composeMemberPortrait(src, t.in.Units, id, eq, m)
		p.figure, p.mask = portrait.Figure, portrait.SlotMask
		if profile.hoverMask && portrait.HoverSlotMask != nil {
			p.mask = portrait.HoverSlotMask
		}
	}
	if profile.currentItems {
		for i, item := range mapload.MemberItemEquipment(m, t.in.Table) {
			if item.Code != 0 {
				p.info[i] = itemInstanceInfoLines(item, t.in.Table, t.in.Words)
			}
		}
	} else {
		p.info = t.in.wornItemLines(m, nil)
	}
	return p
}

func (t *townScreen) roomPersonCard(subject ui.PanelSubject, figure *image.RGBA, known bool) ui.TownCharacterView {
	if t.in.roomCardFromGenerator() {
		if a := t.in.ChargenAssets; a != nil && a.Presentation != nil {
			return a.Presentation.CardView(subject, image.Rectangle{}, image.Point{}, nil, nil)
		}
		return ui.TownCharacterView{}
	}
	panes := t.art.characterPanes()
	return ui.TownCharacterView{Subject: subject, HasSubject: known, Font: t.in.Font.Value(), CardFont: t.in.tipFont(), FigurePane: panes.Figure, StatsPane: panes.Stats, Figure: figure}
}
