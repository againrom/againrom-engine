package game

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"math/bits"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func (x *MapInspector) npcChoices(maxID int64, roles bool) []ui.InspectionChoice {
	var choices []ui.InspectionChoice
	if x.table == nil {
		return choices
	}
	for _, e := range x.table.NPCCatalog.Records() {
		id := int64(e.ID)
		if id < 0 || id > maxID && e.Mod == "" {
			continue
		}
		choices = append(choices, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(x.npcName(e, roles)))})
	}
	slices.SortFunc(choices, func(a, b ui.InspectionChoice) int { return int(a.Value - b.Value) })
	return choices
}

// editorSpeakers is the map at mission start as a dialogue's speaker
// resolution reads it, kept while the map bytes and the table stay the same:
// the map the mission opens (border placements withdrawn), the world built
// from it at Normal difficulty, the cast missionSpeakers makes over that
// world's placements and liveness, each creature's tier, the speaker records,
// and the speaker list's faces.
type editorSpeakers struct {
	from  []byte
	table *mapload.Table
	// m is the map as the mission opens it; placed[i] is the index in m of the
	// authored placement i, or -1 for a placement the border withdrew.
	m       *alm.Map
	placed  []int
	world   *sim.World
	tiers   map[sim.EntityID]int
	records map[int32]data.NPCFace
	cast    speakerCast
	faces   map[int64]*image.RGBA
	// heroFaces are the faces tried for a speaker with a planted starting
	// hero, nil for none.
	heroFaces map[heroFaceKey]*image.RGBA
}

// heroFaceKey is a speaker and a planted starting hero, a bit of
// ui.HeroAudience.
type heroFaceKey struct {
	speaker int64
	hero    int
}

// missionStart is the map's editorSpeakers, built again when the map's
// bytes or the table changed.
func (s *mapInspectionEdits) missionStart() *editorSpeakers {
	b, t := s.model.Bytes(), s.inspector.table
	if sp := s.speakers; sp != nil && sp.table == t && bytes.Equal(sp.from, b) {
		return sp
	}
	sp := &editorSpeakers{from: b, table: t, faces: map[int64]*image.RGBA{}, heroFaces: map[heroFaceKey]*image.RGBA{}}
	s.speakers = sp
	authored, err := s.model.Map()
	if err != nil || t == nil {
		return sp
	}
	// The mission opens the map with its border placements withdrawn, and the
	// world build is free to change the map it is given.
	m, err := s.model.Map()
	if err != nil {
		return sp
	}
	sp.placed = make([]int, len(authored.Units))
	kept := 0
	for i := range authored.Units {
		sp.placed[i] = -1
		if !mapload.BorderWithdrawn(authored, i) {
			sp.placed[i] = kept
			kept++
		}
	}
	mapload.WithdrawBorderPlacements(m)
	sp.m, sp.tiers, sp.records = m, entityTiers(m, t), speakerFaces(t)
	world, roster, err := mapload.FromALMRoster(m, t, mapload.DifficultyNormal)
	if err != nil || world == nil {
		return sp
	}
	sp.world = world
	// The mission driver's own questions of a world: who is alive and what each
	// wears.
	mw := &mapWorld{world: world}
	sp.cast = newSpeakerCast(editorHero(), &speakerWorld{
		actors: missionSpeakers(m, t, roster, nil, nil, placedEntities(m, world.Entities(), roster, nil)),
		alive:  mw.entityAlive, worn: mw.equipmentOf,
	})
	return sp
}

// speakerResolver resolves speakers over sp, through the inspector's picture
// caches.
func (s *mapInspectionEdits) speakerResolver(sp *editorSpeakers) *speakerResolver {
	x := s.inspector
	if x.speakerPictures == nil {
		x.speakerPictures = map[string]*image.RGBA{}
		x.speakerFigures = map[figureCacheKey]*image.RGBA{}
	}
	var src entrySource
	if x.archives != nil {
		src = x.archives.artSource()
	}
	return &speakerResolver{src: src, units: x.units, npcFaces: sp.records,
		portraits: x.speakerPictures, figurePics: x.speakerFigures, cast: sp.cast}
}

// npcPortraits are the faces of the NPC records the choices name, by choice
// value: each the picture a dialogue on this map shows at mission start when
// that NPC speaks, cut as the dialogue pane cuts it. A choice whose record
// has no face has no entry.
func (s *mapInspectionEdits) npcPortraits(choices []ui.InspectionChoice) map[int64]*image.RGBA {
	x := s.inspector
	if x == nil || x.table == nil || x.archives == nil {
		return nil
	}
	sp := s.missionStart()
	var r *speakerResolver
	out := map[int64]*image.RGBA{}
	for _, c := range choices {
		pic, tried := sp.faces[c.Value]
		if !tried {
			if r == nil {
				r = s.speakerResolver(sp)
			}
			if c.Value >= 0 && c.Value <= math.MaxInt32 {
				if face, window, ok := r.SpeakerFace(int(c.Value)); ok {
					pic = ui.SpeakerPortrait(face, window)
				}
			}
			sp.faces[c.Value] = pic
		}
		if pic != nil {
			out[c.Value] = pic
		}
	}
	return out
}

// UnitSpeaker is the speaker that shows placement unit at mission start. A
// person takes its own NPC record when that record resolves to it, else the
// lowest speaker that resolves to it. A creature takes the lowest speaker
// whose picture is the one its info window shows. When no speaker shows the
// unit, the miss says why.
func (s *mapInspectionEdits) UnitSpeaker(unit int) (int64, ui.SpeakerMiss) {
	x := s.inspector
	sp := s.missionStart()
	if sp.m == nil || sp.world == nil || unit < 0 || unit >= len(sp.placed) {
		return 0, ui.SpeakerMiss{Cause: ui.SpeakerNoMission}
	}
	if sp.placed[unit] < 0 {
		return 0, ui.SpeakerMiss{Cause: ui.SpeakerBorder}
	}
	if id, ok := s.boundSpeaker(unit); ok {
		return id, ui.SpeakerMiss{}
	}
	unit = sp.placed[unit]
	var ids []int64
	for _, c := range x.npcChoices(255, true) {
		ids = append(ids, c.Value)
	}
	u := sp.m.Units[unit]
	r := mapload.Resolve(u, x.table)
	// The mission start here has no starting hero, so it builds no unit for
	// an npc record the start composes from the hero.
	if _, rows, _ := mapload.IdentifyAt(u, x.table, s.mission()); len(rows) > 0 {
		return 0, ui.SpeakerMiss{Cause: ui.SpeakerByHero}
	}
	if r.Arm == mapload.ArmUnits {
		return s.creatureSpeaker(sp, unit, ids)
	}
	shows := func(id int64) bool {
		rec, ok := sp.records[int32(id)]
		if !ok {
			return false
		}
		a, ok := sp.cast.resolveNPC(int32(id), rec)
		return ok && a.id == sim.EntityID(unit)
	}
	if own := int64(r.NPC); r.Arm == mapload.ArmNPC && slices.Contains(ids, own) && shows(own) {
		return own, ui.SpeakerMiss{}
	}
	for _, id := range ids {
		if shows(id) {
			return id, ui.SpeakerMiss{}
		}
	}
	return 0, s.personMiss(sp, unit, ids)
}

// personMiss is why no speaker shows person unit of the mission start: the
// placement makes no live person, the person starts down, no record's terms
// fit it, or a record fits it and shows another unit first. For the last, the
// record is the one with most terms (the lowest on a tie) and the cell the
// first unit that record shows.
func (s *mapInspectionEdits) personMiss(sp *editorSpeakers, unit int, ids []int64) ui.SpeakerMiss {
	actor, ok := sp.cast.actor(sim.EntityID(unit))
	if !ok {
		return ui.SpeakerMiss{Cause: ui.SpeakerNoPlacement}
	}
	if !sp.cast.alive(actor.id) {
		return ui.SpeakerMiss{Cause: ui.SpeakerDown}
	}
	best, terms := int64(-1), -1
	for _, id := range ids {
		rec, ok := sp.records[int32(id)]
		if !ok || !sp.cast.matches(rec, actor) {
			continue
		}
		if n := bits.OnesCount32(uint32(rec.Tokens)); n > terms {
			best, terms = id, n
		}
	}
	if best < 0 {
		return ui.SpeakerMiss{Cause: ui.SpeakerNoFit}
	}
	first, ok := sp.cast.resolveNPC(int32(best), sp.records[int32(best)])
	if !ok || int(first.id) >= len(sp.m.Units) {
		return ui.SpeakerMiss{Cause: ui.SpeakerNoFit}
	}
	at := sp.m.Units[first.id]
	return ui.SpeakerMiss{Cause: ui.SpeakerNotFirst, Record: best, Cell: image.Pt(int(at.X>>8), int(at.Y>>8)), Unit: s.inspector.unitLabel(at, s.mission())}
}

// creatureSpeaker is the lowest of ids whose dialogue picture is the one the
// info window shows for creature unit at mission start. No speaker carries
// the picture of a creature of a class the speaker list does not name, or of
// a unit whose class composes a figure.
func (s *mapInspectionEdits) creatureSpeaker(sp *editorSpeakers, unit int, ids []int64) (int64, ui.SpeakerMiss) {
	e, ok := sp.world.Entity(sim.EntityID(unit))
	if !ok {
		return 0, ui.SpeakerMiss{Cause: ui.SpeakerNoPlacement}
	}
	miss := ui.SpeakerMiss{Cause: ui.SpeakerNoPicture}
	if data.ComposesFigure(e.Class) {
		return 0, miss
	}
	r := s.speakerResolver(sp)
	want := infoPortrait(r.src, r.units, e.Class, sp.tiers[e.ID], r.portraits)
	if want == nil {
		return 0, miss
	}
	for _, id := range ids {
		if face, _, ok := r.SpeakerFace(int(id)); ok && face == want {
			return id, ui.SpeakerMiss{}
		}
	}
	return 0, miss
}

// npcName is NPC record e's name in UTF-8. With roles, as the message popup
// names a speaker, a speaker whose person depends on the first hero is its
// role's word; otherwise the record's name, its npcnames.txt line, or its
// section. The npcnames.txt line is the one the game writes as a placed
// person's name: line n, counted from 1, for record n.
func (x *MapInspector) npcName(e data.NPCEntry, roles bool) string {
	id := int64(e.ID)
	if roles {
		if role, ok := x.speakerRole(id); ok {
			return role
		}
	}
	name := e.Name
	if line := id - 1; name == "" && line >= 0 && line < int64(len(x.table.NPCNames)) {
		name = x.table.NPCNames[line]
	}
	if name == "" {
		name = e.Section
	}
	return x.messageUTF8([]byte(name))
}

// speakerLabel is speaker id's label in UTF-8 as the message popup builds
// it, false when the install has no such speaker.
func (x *MapInspector) speakerLabel(id int64) (string, bool) {
	if role, ok := x.speakerRole(id); ok {
		return role, true
	}
	if x.table == nil || x.table.NPCCatalog == nil || id < 0 || id > math.MaxInt32 {
		return "", false
	}
	e, ok := x.table.NPCCatalog.Entry(int32(id))
	if !ok {
		return "", false
	}
	return x.npcName(e, true), true
}

// heroLabels builds the hero band's labels from the words format and
// roleFormat, each text passed through encode. On a first-game install heroes
// 1 to 6 name speakers 21 to 26 (TRIG-HEROTPL-076), and each carries that
// speaker's label; on a second-game install the label stays a number.
func (x *MapInspector) heroLabels(format, roleFormat string, encode func(string) string) ui.HeroLabels {
	h := ui.HeroLabels{Format: encode(format), RoleFormat: encode(roleFormat)}
	if x == nil {
		return h
	}
	var profile base.Profile
	if x.archives != nil {
		profile = x.archives.Base.Profile
	}
	if !profile.HasFirstData() {
		return h
	}
	h.Roles = map[int64]string{}
	for n := int64(1); n <= 6; n++ {
		if label, ok := x.speakerLabel(20 + n); ok {
			h.Roles[n] = encode(label)
		}
	}
	return h
}

// editorHeroLabels are the hero band's labels in the editor's words and the
// font's bytes.
func (x *MapInspector) editorHeroLabels() ui.HeroLabels {
	return x.heroLabels(x.word("editor.param.hero"), x.word("editor.param.herorole"), x.inspectionUTF8)
}

// editorHero is the player's hero the editor's speakers are built with: a
// man, a fighter, the first face. A map has no player yet, and a speaker whose
// face the game takes from the player's hero is shown as it is for a man.
func editorHero() []mapload.PartyMember { return plantedHero(herokind.MaleFighter) }

// plantedHero is a party of one starting hero of that sex and class, the
// first face: the hero the editor reads a dialogue's branch for.
func plantedHero(k herokind.Kind) []mapload.PartyMember {
	return []mapload.PartyMember{{
		PlayerCharacter: true, StartingHero: true,
		FigureDir: string(data.FigureDirFor(k.Mage(), k.Female())), Mage: k.Mage(), FigureFace: 1,
	}}
}

// eventTags are the tag rules of the game the editor's install is.
func (x *MapInspector) eventTags() eventTags {
	if x == nil || x.archives == nil {
		return firstEventTags{}
	}
	return campaignOf(x.archives.Game()).eventTags()
}
