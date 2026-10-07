package game

import (
	"cmp"
	"slices"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// WHICH LIVE ACTOR A DIALOGUE IS ABOUT.
//
// `DLG-SPEAKER-022` splits the dialogue's speaker into two arms: the original
// searches the client actor list for one matching the section, and only when
// that search finds nothing does it synthesise a drawable whose twelve
// equipment slots are cleared and never written. This file is the first arm.
// The second is the record's own bare figure (speakers.go).
//
// `DLG-SPEAKER-023` reads the search: it walks the actor list, filters by
// runtime class, and combines ONE TERM PER TOKEN the section's `Flags`
// states, over seventeen tokens, plus the record's `Face` and `Picture`
// comparisons and a state gate; the FIRST survivor is returned. The token
// set is carried on the record (pkg/data/npcface.go) and the terms are
// evaluated here, because every one of them reads a live actor and pkg/data
// holds none.

// speakerActor is one candidate: a person the current mission holds, and the
// things the predicate compares against.
//
// IT IS IDENTITY AND NOT APPEARANCE. What the candidate is WEARING is
// deliberately absent: `DLG-SPEAKER-022` states that a live speaker's slots
// change by re-send and that its figure follows, so the worn set is read
// from the world when the picture is asked for and never cached here.
type speakerActor struct {
	id   sim.EntityID
	me   bool
	hero bool

	// fig is the figure directory and sheet this person draws with, the same
	// pair entityFigures resolves for the doll — so a speaker's dialogue
	// picture and his world picture are one figure by construction.
	fig figureID

	// face is the face number the person's drawable carries, the value
	// `DLG-SPEAKER-023`'s `Face` term compares the record's `Face` against
	// (`actor+0x24`, `REG-NPC-088`). It is fig.Face: the row's column for a
	// person whose spawner writes no face byte, the low seven bits of the byte
	// it writes otherwise. Mission 40's `npc25` states `Face = 1`.
	face int32

	// typeID is the row's own `TypeID` column, the right-hand side of the
	// `Picture` term (`actor+0x20`, `REG-NPC-088`). It is the DRAWN class id of
	// a zero-mode person — 3, 14, 24 on the four shipped archetype rows — and
	// not the hero-range id a Hero-mode placement's constructor writes there
	// (`ALM-CLS-054`).
	typeID int32
}

// entitySpeakers is every placed person of m, in ascending entity id: the
// candidate list the predicate walks.
//
// IT IS entityFigures' OWN WALK over the same placements and the same table
// (figures.go), and it is a second walk rather than a widening of that one for
// that function's own stated reason: a placement resolves down one of two
// bands, and a creature contributes no figure and no candidate. The row's
// `TypeID` column, which this list needs, reaches no figure, so folding it into
// figureID would widen the composed-picture cache key with a value the composer
// never reads.
//
// THE ORDER IS ASCENDING ENTITY ID AND IT IS OURS. The original returns the
// first survivor of a walk over the client actor list at `this+0x9b8`; that
// list's order is not decoded (0160 provenance). Ascending id is the order the
// map states its placements in, and it puts a map's own placed person ahead of
// anything minted later.
//
// It is TOTAL and reports nothing: a row this build cannot parse contributes no
// candidate, exactly as it contributes no figure.
func entitySpeakers(m *alm.Map, t *mapload.Table) []speakerActor {
	if m == nil || t == nil || t.Humans == nil {
		return nil
	}
	out := make([]speakerActor, 0, len(m.Units))
	for i, u := range m.Units {
		r := mapload.Resolve(u, t)
		if r.Arm == mapload.ArmUnits || !r.Found() {
			continue
		}
		d, err := data.NewHumanDef(t.Humans.EntryName(r.Index), t.Humans.EntryParams(r.Index))
		if err != nil {
			continue
		}
		dir, face, _ := mapload.PlacedFigure(u, t)
		out = append(out, speakerActor{
			id: sim.EntityID(i), hero: r.Arm == mapload.ArmNPC && t.NPC.Hero(int32(u.ClassSubID)),
			fig:  figureID{Dir: dir, Face: face},
			face: int32(face), typeID: d.TypeID,
		})
	}
	return out
}

// placedEntity is one placement of the mission's map and the entity it became.
// index is -1 for a roster person whose placement the load did not keep.
type placedEntity struct {
	index  int
	id     sim.EntityID
	typeID int32
}

// placedEntities pairs the placements of m with their entities, in ascending
// entity id.
//
// A FRESH MAP MINTS PLACEMENT i AS ENTITY i, so the index is the pairing. A SAV
// LOAD KEEPS THE SAVED IDS BUT WITHDRAWS THE PLACEMENTS OF ACTORS THAT ARE GONE
// (retainSavedActorPlacements): after one withdrawal every later index names
// the next entity, and the highest ids name no placement at all. With a saved
// document an entity pairs with the placement at its own index only while that
// placement carries the entity's MapUnitID, and otherwise with the one
// placement that carries it; a MapUnitID the map holds twice pairs nothing, as
// in savedActorFigures. A roster person left unpaired is still a live person
// and keeps its native drawing class.
func placedEntities(m *alm.Map, entities []sim.Entity, roster map[sim.EntityID]mapload.PartyMember,
	state *SnapshotSAVDocument) []placedEntity {
	if m == nil {
		return nil
	}
	out := make([]placedEntity, 0, len(m.Units))
	if state == nil {
		for i := range m.Units {
			out = append(out, placedEntity{index: i, id: sim.EntityID(i)})
		}
		return out
	}
	byMapID := make(map[uint16]int, len(m.Units))
	for i, u := range m.Units {
		if u.UnitID == 0 {
			continue
		}
		if _, found := byMapID[u.UnitID]; found {
			byMapID[u.UnitID] = -1
		} else {
			byMapID[u.UnitID] = i
		}
	}
	paired := make(map[sim.EntityID]bool, len(entities))
	types := make(map[sim.EntityID]int32, len(entities))
	for _, e := range entities {
		types[e.ID] = e.TypeID
		if e.MapUnitID == 0 {
			continue
		}
		i, found := byMapID[e.MapUnitID]
		if int(e.ID) < len(m.Units) && m.Units[e.ID].UnitID == e.MapUnitID {
			i, found = int(e.ID), true
		}
		if !found || i < 0 {
			continue
		}
		out = append(out, placedEntity{index: i, id: e.ID, typeID: e.TypeID})
		paired[e.ID] = true
	}
	for id := range roster {
		if !paired[id] {
			out = append(out, placedEntity{index: -1, id: id, typeID: types[id]})
		}
	}
	slices.SortFunc(out, func(a, b placedEntity) int { return cmp.Compare(a.id, b.id) })
	return out
}

// missionSpeakers is the complete live client-actor population available to a
// mission dialogue. Map placements retain their ascending entity-id order;
// party actors follow in start order, paired with the ids the start minted for
// them. The latter are essential after a companion crosses a mission boundary:
// Brian in mission 70, for example, is no longer a placement in that map but is
// still a live actor and must not fall through to the bare synthetic arm.
func missionSpeakers(m *alm.Map, t *mapload.Table, roster map[sim.EntityID]mapload.PartyMember,
	party []mapload.PartyMember, ids []sim.EntityID, placed []placedEntity) []speakerActor {
	persons := entitySpeakers(m, t)
	byIndex := make(map[int]speakerActor, len(persons))
	for _, a := range persons {
		byIndex[int(a.id)] = a
	}
	out := make([]speakerActor, 0, len(placed)+len(party))
	for _, pe := range placed {
		if p, ok := roster[pe.id]; ok {
			hero := pe.index < 0 && data.FigureIsHero(pe.typeID)
			if m != nil && pe.index >= 0 && pe.index < len(m.Units) {
				u := m.Units[pe.index]
				hero = u.Flags&1 != 0 && t != nil && t.NPC.Hero(int32(u.ClassSubID))
			}
			dir, face := memberFigure(p)
			// A scenario-placed human's drawn class is his row's own TypeID
			// (UNIT-APPEAR-030). The roster template's class is not: the
			// equipment law rewrites it to a body class when the mission opens
			// and a SAV load replaces it with the class the entity was placed
			// with, so a horse read from it followed the SAVE and the LOAD. A
			// Hero-mode placement, built in the player-character class range
			// (ALM-CLS-054), and a person no placement pairs keep the
			// template's class.
			class := p.Class
			if placement, paired := byIndex[pe.index]; paired && !hero {
				class = placement.typeID
			}
			out = append(out, speakerActor{
				id: pe.id, hero: hero,
				fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: class,
			})
			continue
		}
		if a, ok := byIndex[pe.index]; ok {
			a.id = pe.id
			out = append(out, a)
		}
	}
	n := len(party)
	if len(ids) < n {
		n = len(ids)
	}
	for i := 0; i < n; i++ {
		p := party[i]
		dir, face := memberFigure(p)
		out = append(out, speakerActor{
			id: ids[i], me: primaryPlayerHero(p), hero: p.PlayerCharacter && p.MercenaryType == 0,
			fig: figureID{Dir: dir, Face: face}, face: int32(face), typeID: p.Class,
		})
	}
	return out
}

// speakerCast is the live half of speaker resolution: the candidates, the two
// world questions the predicate and the composition ask of each, and the
// player's own figure for the two terms that compare against it.
//
// A ZERO CAST RESOLVES NOTHING, which is the town screen's own state outside
// the tavern: it holds a party and no world, so a shop or school speaker takes
// the original's bare synthesised arm. The tavern's cast is its stock units
// (tavernSpeakerCast).
type speakerCast struct {
	actors []speakerActor

	// alive is `DLG-SPEAKER-023`'s state gate, in the terms this tree has:
	// the check at L03478 rejects a candidate whose state byte at
	// `actor+0x15a` is above 1, and what that byte counts is not decoded. A dead
	// person is the one state this build can name, and a corpse is not who a
	// dialogue is about.
	alive func(sim.EntityID) bool

	// worn is what the world says a candidate is wearing right now.
	worn func(sim.EntityID) data.Equipment

	// playerDir is the player's own figure directory, the right-hand side of
	// the `MySex` and `MyClass` terms — `[this+0x3f54]+0x18c` bits 2 and 1 in
	// `DLG-SPEAKER-023`, which are the two axes a figure directory IS
	// (`HERO-DOLL-078`). hasPlayer is false for a driver with no party
	// subject, and both terms then reject rather than guess.
	playerDir data.FigureDir
	hasPlayer bool
}

// resolve is the first live candidate the record's own predicate admits, or no
// speaker at all.
//
// NO LIVENESS TEST IS NO WORLD. A cast that cannot say whether a candidate is
// alive is not holding a world, and it resolves nobody rather than admitting a
// candidate it cannot gate.
func (c speakerCast) resolve(rec data.NPCFace) (speakerActor, bool) {
	if c.alive == nil {
		return speakerActor{}, false
	}
	for _, a := range c.actors {
		if !c.alive(a.id) {
			continue
		}
		if c.matches(rec, a) {
			return a, true
		}
	}
	return speakerActor{}, false
}

// matches is `DLG-SPEAKER-023`'s conjunction over one candidate: one term per
// token the record states, and nothing at all for a token it does not. A record
// stating no token narrows nothing and the first live candidate answers it.
//
// `Human` is satisfied by every candidate that is not a hero: each is either a
// placement resolved down the humans band or a party member minted from one. `Hero` is the
// registry's exact Hero-mode bit for a placement and the persistent-player mode
// for a party member. This distinction is load-bearing in mission 70: npc23's
// Hero token must skip ordinary female fighters before it reaches Naira.
//
// `Me` is the party member explicitly marked StartingHero. It is identity and
// not party order: a joined companion is still a live candidate, but never the
// player's own character merely because another transition moved it in the
// slice.
//
// `Platoon` IS NOT EVALUATED and states no constraint (0160 spec SC-3).
// Nothing in this tree holds a platoon to compare.
func (c speakerCast) matches(rec data.NPCFace, a speakerActor) bool {
	t := rec.Tokens
	if t.Has(data.NPCTokenHero) && !a.hero {
		return false
	}
	if t.Has(data.NPCTokenNotHero) && a.hero {
		return false
	}
	if t.Has(data.NPCTokenNotHuman) {
		return false
	}
	// A hero's class setter stores 9, a human class 0x18, so the Human bit
	// 0x10 is never set on a hero (TAVERN-TALKSTATS-017).
	if t.Has(data.NPCTokenHuman) && a.hero {
		return false
	}
	if t.Has(data.NPCTokenMe) && !a.me {
		return false
	}
	if t.Has(data.NPCTokenNotMe) && a.me {
		return false
	}
	if t.Has(data.NPCTokenMage) && !a.fig.Dir.Mage() {
		return false
	}
	if t.Has(data.NPCTokenNotMage) && a.fig.Dir.Mage() {
		return false
	}
	if t.Has(data.NPCTokenFemale) && !a.fig.Dir.Female() {
		return false
	}
	if t.Has(data.NPCTokenNotFemale) && a.fig.Dir.Female() {
		return false
	}
	if t.Has(data.NPCTokenMySex) || t.Has(data.NPCTokenNotMySex) {
		if !c.hasPlayer {
			return false
		}
		if same := a.fig.Dir.Female() == c.playerDir.Female(); same != t.Has(data.NPCTokenMySex) {
			return false
		}
	}
	if t.Has(data.NPCTokenMyClass) || t.Has(data.NPCTokenNotMyClass) {
		if !c.hasPlayer {
			return false
		}
		if same := a.fig.Dir.Mage() == c.playerDir.Mage(); same != t.Has(data.NPCTokenMyClass) {
			return false
		}
	}
	if t.Has(data.NPCTokenFace) && int32(rec.Face) != a.face {
		return false
	}
	if t.Has(data.NPCTokenPicture) && (!rec.HasClass || rec.Class != a.typeID) {
		return false
	}
	return true
}
