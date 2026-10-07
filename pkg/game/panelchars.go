package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// PartyCharacters is what each party member's sheet states, keyed by the
// entity id the mission's start minted for him — the exported face of the
// map the front end already builds for the unit panel.
//
// IT EXISTS FOR THE MEASUREMENT TOOL AND FOR NOTHING ELSE. A developer tool
// that composes the panel against a lawful install has to state the same
// character the running game does, and the recompute that produces one —
// statistics, experience, the two five-wide families, the trained slot, the
// weapon's name — is a rule of this package. A tool holding its own copy
// would measure a window the game never draws, which is the one failure a
// measurement story cannot afford.
//
// It is a thin wrapper rather than a rename because the unexported name is what
// the front end's own call sites read, and moving them would put this story's
// diff inside a file another lane is editing.
func PartyCharacters(ms *Mission) map[sim.EntityID]ui.UnitCharacter {
	return partyCharacters(ms)
}

// MissionCharacters is PartyCharacters' successor for the measurement tool,
// and it exists for exactly the reason that one does: a tool composing the
// panel against a lawful install must state the character the running game
// states. Since 0137 the running game states one for every placement that
// resolved to a definition entry, so a tool still calling PartyCharacters
// alone would compose a placed unit's panel out of the zero character and
// measure a window nobody sees — the one failure a measurement instrument
// cannot afford, and the reason this is exported rather than left to the
// driver.
//
// PartyCharacters is KEPT beside it rather than replaced: "what does the
// party alone state" is still a question a caller may want answered, and it
// is the half of this map that owes nothing to a table.
func MissionCharacters(ms *Mission, t *mapload.Table) map[sim.EntityID]ui.UnitCharacter {
	if ms == nil {
		return nil
	}
	return missionCharacters(ms.Map, t, ms)
}

// characterBand turns pkg/mapload's own two-valued Band into the window's
// three-valued CharacterBand. The map- loading tier states no third value of
// its own: a placement Resolve could not match is simply absent from
// PlacedSheets, never a Sheet carrying an "unknown" band, so every Sheet
// this function is handed already answers creature-or-person and the switch
// below is total over it. The window's own third value,
// CharacterBandUnknown, is reached only by a caller that never runs a Sheet
// through this function at all — a UnitCharacter's own zero value —
// which is why nothing here ever returns it.
func characterBand(b mapload.Band) ui.CharacterBand {
	if b == mapload.BandCreature {
		return ui.CharacterBandCreature
	}
	return ui.CharacterBandPerson
}

// sheetCharacter is one placement's mapload.Sheet, copied field by field
// into the window's own UnitCharacter. IT IS THE ONE PLACE THE TWO
// VOCABULARIES MEET: pkg/mapload sits below pkg/ui in the import DAG and
// must not name UnitCharacter, so the tier below states a plain Sheet and
// this function performs the mechanical widening — the same kind of copy
// partyCharacters already performs out of a Recompute's own derived set,
// applied here to the other source of a character.
//
// A creature's Skill[0] and Experience are UNSTATED at the source (Sheet's
// own doc says so, and Band is what a reader must check before trusting
// them); this function copies them across regardless, deriving nothing and
// suppressing nothing — Band alone still says which fields of the result may
// be trusted, exactly as it does on the Sheet this was copied from.
func sheetCharacter(s mapload.Sheet) ui.UnitCharacter {
	c := ui.UnitCharacter{
		Known: true,
		Band:  characterBand(s.Band),
		Mage:  s.Mage,
		Body:  int(s.Body), Reaction: int(s.Reaction), Mind: int(s.Mind), Spirit: int(s.Spirit),
		Experience: int(s.Experience),
		Sight:      int(s.Sight),
	}
	for i, lvl := range s.Skill {
		c.Skills[i] = int(lvl)
	}
	for i := range c.Protection {
		c.Protection[i] = int(s.Elemental[i])
		c.Resistance[i] = int(s.WeaponKind[i])
	}
	return c
}

// placedCharacters is every placed entity's character, converted from the map-
// loading tier's own PlacedSheets and keyed by the entity its placement became
// (placedEntities). PlacedSheets keys a sheet by the placement's index, which
// is the entity id of a fresh map and of nothing else: a SAV load keeps the
// saved ids and withdraws the placements of actors that are gone, so an entity
// takes the sheet of its own placement and never that of the one at its id. A
// placement PlacedSheets states nothing for — no table, no resolved entry —
// states no character here either: sheetCharacter above is never asked to
// widen a Sheet nothing produced, so "no zero character" holds by
// construction and not by a second check in this function.
func placedCharacters(m *alm.Map, t *mapload.Table, placed []placedEntity) map[sim.EntityID]ui.UnitCharacter {
	sheets := mapload.PlacedSheets(m, t)
	if len(sheets) == 0 {
		return nil
	}
	out := make(map[sim.EntityID]ui.UnitCharacter, len(sheets))
	for _, pe := range placed {
		if s, found := sheets[sim.EntityID(pe.index)]; found && pe.index >= 0 {
			out[pe.id] = sheetCharacter(s)
		}
	}
	return out
}

// missionCharacters is the ONE character map a mission's driver builds:
// every placement's character first, the party's own second. The party's ids
// are minted after every placement's — mapload's own id convention, which
// PlacedSheets' own doc already states this package leans on — so the
// second pass can never overwrite one placement's character with another
// unit's; it runs second because that is the order the two lookups' own ids
// already guarantee, not because this function arbitrates a collision. Every
// existing reader of the merged map — the draw path, rearm's own
// weapon-name write — is untouched: both already read one map keyed by
// entity id, and this changes only what fills it before the mission starts.
func missionCharacters(m *alm.Map, t *mapload.Table, ms *Mission) map[sim.EntityID]ui.UnitCharacter {
	var entities []sim.Entity
	if ms.World != nil {
		entities = ms.World.Entities()
	}
	out := placedCharacters(m, t, placedEntities(m, entities, ms.Start.Roster, ms.savedDocument))
	// A scenario person that can be handed to the player has a richer,
	// identity-bearing roster template than the static placement sheet. This is
	// also the only source for composed npc21..24, whose row depends on the
	// entering hero and therefore cannot be resolved from (map, table) alone.
	for id, p := range ms.Start.Roster {
		if out == nil {
			out = make(map[sim.EntityID]ui.UnitCharacter, len(ms.Start.Roster))
		}
		out[id] = partyPanelSubject(p, t).Char
	}
	party := partyCharacters(ms, t)
	if len(party) == 0 {
		return out
	}
	if out == nil {
		out = make(map[sim.EntityID]ui.UnitCharacter, len(party))
	}
	for id, c := range party {
		out[id] = c
	}
	return out
}
