package game

import (
	"fmt"
	"image"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/text"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// tavernTalkObject is the object a talk-only cell stands for: the live actor
// the npc record's `Flags` terms admit, or the synthesised one
// (TAVERN-TALKPIC-016). Its fields are the `+0x18c` bits and the `+0x15b` byte
// the cell's sheet and the left panel read.
type tavernTalkObject struct {
	// live is the matched party member's index in shopParty, or -1 for a
	// synthesised object.
	live int
	// hero, mage and human are `+0x18c` bits 0, 1 and 0x10.
	hero, mage, human bool
	// sheet is `+0x15b`: the npc id on the synthesised arm, the mercenary type
	// byte on a live hired or stock unit.
	sheet int
	// stock is the mercenary type of a matched stock unit, else 0.
	stock int
}

// statistics reports `+0x18c & 0x40` clear, which TAVERN-TALKSTATS-017 gates
// the statistics panel on. The synthesiser stores 0x48 and no later store
// clears bit 0x40; a live actor's class setter keeps only bit 0x80.
func (o tavernTalkObject) statistics() bool { return o.live >= 0 }

// composed reports `+0x18c & 0x11`, which chooses the composed figure over the
// infowindow picture below the panel.
func (o tavernTalkObject) composed() bool { return o.hero || o.human }

// tavernStockActor is one stock mercenary unit the original builds at tavern
// entry: types 1..15 from their stage-level templates (TAVERN-ORDER-015,
// SAV-926). Only the Human types 3..15 can answer a talk record.
type tavernStockActor struct {
	typ   int
	actor speakerActor
}

// tavernTalkCast is the live actor population a talk record resolves over:
// the carried party in party order, then the stock mercenary units in the
// stock walk. The original walks one actor map holding both
// (DLG-SPEAKER-023); where the party falls in that walk is not decoded, so
// party first is this build's order. The town has no world, so every actor is
// alive.
func tavernTalkCast(party []mapload.PartyMember, stock []tavernStockActor) speakerCast {
	c := speakerCast{alive: func(sim.EntityID) bool { return true }}
	for i, p := range party {
		dir, face := memberFigure(p)
		c.actors = append(c.actors, speakerActor{
			id: sim.EntityID(i), me: primaryPlayerHero(p), hero: p.PlayerCharacter && p.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: p.Class,
		})
		if primaryPlayerHero(p) && !c.hasPlayer {
			c.playerDir, c.hasPlayer = dir, true
		}
	}
	for k, a := range stock {
		a.actor.id = sim.EntityID(len(party) + k)
		c.actors = append(c.actors, a.actor)
	}
	return c
}

// tavernStockActors builds the Human stock units of the current stage from
// the Humans rows `NPC%02d_<level>`, in the stock walk. The Face term reads
// the row's own `Face` column, and the Picture term its `TypeID`.
func (t *townScreen) tavernStockActors() []tavernStockActor {
	if t == nil || t.sess == nil || t.in.Table == nil || t.sess.Town == nil {
		return nil
	}
	var out []tavernStockActor
	for _, typ := range tavernStockWalk() {
		if typ <= 2 {
			continue
		}
		name := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(t.sess.Town.Chapter()))
		i := data.FindHumanByName(t.in.Table.Humans, name)
		if i == data.NotFound {
			continue
		}
		def, err := data.NewHumanDef(name, t.in.Table.Humans.EntryParams(i))
		if err != nil {
			continue
		}
		dir, face := data.FigureFor(def.TypeID, def.Face, def.Gender)
		out = append(out, tavernStockActor{typ: typ, actor: speakerActor{
			fig: figureID{Dir: dir, Face: face}, face: int32(def.Face), typeID: def.TypeID}})
	}
	return out
}

// tavernSpeakerCast is the cast a tavern conversation's speakers resolve over:
// the stock units, each wearing the set its generated candidate wears, so a
// speaker one of them answers is drawn as that unit is in the inspection panel
// (DLG-SPEAKER-023, TAVERN-TALKPIC-016). The party is not in it. A party
// member's dialogue figure is the town's own composition of him, and a Hero
// record never admits a unit. The player's figure still answers the `MySex` and
// `MyClass` terms.
func (t *townScreen) tavernSpeakerCast() speakerCast {
	party, stock := t.shopParty(), t.tavernStockActors()
	c := tavernTalkCast(party, stock)
	c.actors = c.actors[len(party):]
	worn := make(map[sim.EntityID]data.Equipment, len(stock))
	for k, unit := range stock {
		if squad, ok := t.buildMercenarySquad(unit.typ, 1); ok && len(squad) == 1 {
			worn[c.actors[k].id] = mapload.EquipmentFromParty(squad[0])
		}
	}
	c.worn = func(id sim.EntityID) data.Equipment { return worn[id] }
	return c
}

// resolveTavernTalk answers which object InnNPC value npc stands for.
//
// A record carrying `Platoon` is the mercenary section of type npc. It stands
// for a carried unit of that type, else for that type's stock unit, as the
// talk picture does (platoonTalkPicture). The term's predicate is Unknown
// (TAVERN-TALKSTATS-017); the owner saw the original show the type-2 stock
// unit's statistics for npc2 (DIV-2778).
func resolveTavernTalk(npc int, rec data.NPCFace, known bool, party []mapload.PartyMember, stock []tavernStockActor) tavernTalkObject {
	if known && rec.Tokens.Has(data.NPCTokenPlatoon) && npc >= 1 && npc <= tavernMercenaryTypes {
		return platoonTalkObject(npc, party)
	}
	cast := tavernTalkCast(party, stock)
	if known && !rec.Tokens.Has(data.NPCTokenPlatoon) {
		if a, ok := cast.resolve(rec); ok {
			o := tavernTalkObject{live: int(a.id), hero: a.hero, mage: a.fig.Dir.Mage(), human: !a.hero}
			if i := int(a.id); i < len(party) {
				o.sheet = int(party[i].MercenaryType)
			} else {
				o.stock = stock[i-len(party)].typ
				o.sheet = o.stock
			}
			return o
		}
	}
	o := tavernTalkObject{live: -1, sheet: npc}
	if !known {
		return o
	}
	t := rec.Tokens
	o.hero, o.human = t.Has(data.NPCTokenHero), t.Has(data.NPCTokenHuman)
	o.mage = t.Has(data.NPCTokenMage)
	if t.Has(data.NPCTokenMyClass) || t.Has(data.NPCTokenNotMyClass) {
		if cast.hasPlayer {
			o.mage = cast.playerDir.Mage() == t.Has(data.NPCTokenMyClass)
		}
	}
	return o
}

// tavernMercenaryTypes is the count of tavern mercenary types, 1..15
// (TAVERN-ORDER-015).
const tavernMercenaryTypes = 15

// platoonTalkObject is the object a Platoon record of mercenary type typ
// stands for: the first carried member of that type, else the stock unit of
// that type, which counts as live after the party.
func platoonTalkObject(typ int, party []mapload.PartyMember) tavernTalkObject {
	o := tavernTalkObject{live: len(party), sheet: typ, stock: typ, human: typ > 2}
	for i, p := range party {
		if int(p.MercenaryType) == typ {
			o.live, o.stock = i, 0
			break
		}
	}
	return o
}

// tavernTalkFrames is the sheet a talk cell draws: HeroMage or HeroFighter for
// a Hero object, else Unit<+0x15b>. An unshipped sheet answers nil; the
// original aborts there (DIV-1408).
func tavernTalkFrames(art *ui.TownTavernArt, o tavernTalkObject) []image.Image {
	if art == nil {
		return nil
	}
	if o.hero {
		if o.mage {
			return art.HeroFrames[1]
		}
		return art.HeroFrames[0]
	}
	if o.sheet <= 0 || o.sheet >= len(art.UnitFrames) {
		return nil
	}
	return art.UnitFrames[o.sheet]
}

func (t *townScreen) tavernTalkObject(npc int) tavernTalkObject {
	rec, known := t.in.NPCFaces[int32(npc)]
	return resolveTavernTalk(npc, rec, known, t.shopParty(), t.tavernStockActors())
}

// tavernTalkDetail is the left panel for a selected talk cell
// (TAVERN-TALKSTATS-017): a live member's statistics card and figure, or, for
// a synthesised object, no statistics and either its composed face figure or
// its class's infowindow picture.
func (t *townScreen) tavernTalkDetail(npc int) (ui.TownCharacterView, bool) {
	o := t.tavernTalkObject(npc)
	panes := t.art.characterPanes()
	v := ui.TownCharacterView{Font: t.in.Font.Value(), CardFont: t.in.tipFont(),
		FigurePane: panes.Figure, StatsPane: panes.Stats}
	if o.stock > 0 {
		detail, _, _, ok := t.tavernCandidateDetail(o.stock)
		return detail, ok
	}
	if o.statistics() {
		party := t.shopParty()
		if len(t.shopFigures) != len(party) {
			t.composeShopFaces()
		}
		m := party[o.live]
		v.Subject = partyPanelSubject(m, t.in.Table, t.in.Words)
		v.Subject.Unplaced, v.Subject.Selected = true, 1
		v.HasSubject = true
		if o.live < len(t.shopFigures) {
			v.Figure = t.shopFigures[o.live]
		}
		return v, true
	}
	if o.composed() {
		pic, _, ok := t.resolver.SpeakerFace(npc)
		v.Figure = pic
		return v, ok && pic != nil
	}
	// Neither Hero nor Human: the infowindow picture of the class the object
	// holds, which the synthesiser sets only under a `Picture` token.
	rec := t.in.NPCFaces[int32(npc)]
	if !rec.Tokens.Has(data.NPCTokenPicture) || !rec.HasClass || t.in.Units == nil {
		return v, false
	}
	var src entrySource
	if t.in.Archives != nil {
		src = t.in.Archives.Containers
	}
	if t.resolver.portraits == nil {
		t.resolver.portraits = make(map[string]*image.RGBA)
	}
	v.Figure = classPortrait(src, t.in.Units.Classes[rec.Class], rec.Face, t.resolver.portraits)
	return v, v.Figure != nil
}

// renderTavernTalkDetail sets the selected talk cell's panel. The result is
// cached in the tavern detail fields under the key -npc, which no mercenary
// type takes; a selection change clears it (clearTavernDetail).
func (t *townScreen) renderTavernTalkDetail(v *ui.TownSurfaceView, npc int) {
	key := -npc
	if npc <= 0 || t.tavernDetailType != key {
		detail, ok := t.tavernTalkDetail(npc)
		t.clearTavernDetail()
		t.tavernDetailType = key
		if ok {
			t.tavernDetail = detail
			t.tavernDetailText = text.Record(func() {
				t.tavernDetailPixels = ui.RenderTownCandidateInspection(detail, t.in.TownTavernArt.Value())
			})
		}
	}
	v.Candidate, v.CandidatePixels, v.CandidateText = t.tavernDetail, t.tavernDetailPixels, t.tavernDetailText
}
