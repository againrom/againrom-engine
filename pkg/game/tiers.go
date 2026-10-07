package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// entityTiers is the tier every placement of m resolves to, keyed by the entity
// id the world builder mints for it, and holding an entry only for a placement
// that states one.
//
// THE KEY IS THE MINTED ID, and this function may say so because the world
// builder documents it: the i-th unit record takes id i, counting from zero,
// and both this lookup and that world are built from the same decoded map in
// the same call. A lookup by loop index that happened to agree would be the
// same numbers by guess, not contract.
//
// ONLY THE ARM THAT CARRIES STATS CARRIES A TIER. A placement diverted to the
// scenario npc table, one resolved through the overriding definition id, one
// naming a humans entry, and one whose key reaches no entry at all are all
// ABSENT from the map — and absent is the answer, not zero written down: a
// missing key reads as the zero tier, which is the render tier's "no tier
// stated" and draws the sheet's own colours. Writing a 1 for them instead would
// assert that every such placement is a first-tier creature, which the table
// does not say.
//
// It is TOTAL and reports nothing. A nil map, a nil table, a table with no
// units collection and an entry this build's definition contract refuses each
// contribute no entry rather than an error: the world beside it was already
// built from the same inputs and has already failed on anything fatal, and a
// placement with no tier draws, which is the point.
//
// It is a pure function of the map and the table. Called twice with neither
// changed it answers the same tiers, so a snapshot built twice cannot report
// a creature in two colours.
func entityTiers(m *alm.Map, t *mapload.Table) map[sim.EntityID]int {
	if m == nil || t == nil || t.Units == nil {
		return nil
	}
	out := make(map[sim.EntityID]int, len(m.Units))
	for i, u := range m.Units {
		r := mapload.Resolve(u, t)
		if r.Arm != mapload.ArmUnits || !r.Found() {
			continue
		}
		d, err := data.NewUnitDef(t.Units.EntryName(r.Index), t.Units.EntryParams(r.Index))
		if err != nil {
			continue
		}
		// The definition's own column, carried whole. It is not clamped, ranged
		// or defaulted here: how many tiers the class it names actually has is
		// the bundle's answer, given at the draw, and a second bound here would
		// be a second place for the two to disagree.
		out[sim.EntityID(i)] = int(d.Face)
	}
	return out
}

func savedActorTiers(m *alm.Map, placed map[sim.EntityID]int, entities []sim.Entity, state *SnapshotSAVDocument) map[sim.EntityID]int {
	if state == nil || state.Document == nil || len(state.Actors) == 0 {
		return placed
	}
	byID := make(map[sim.EntityID]sim.Entity, len(entities))
	tiers := make(map[sim.EntityID]int, len(placed))
	for _, e := range entities {
		byID[e.ID] = e
	}
	for _, binding := range state.Actors {
		if binding.Retired || binding.ObjectIndex == 0 || int(binding.ObjectIndex) > len(state.Document.Objects) {
			continue
		}
		e, found := byID[binding.EntityID]
		if !found || e.Humanoid || e.SourceBinding.ActorClass() == 1 {
			continue
		}
		record := &state.Document.Objects[binding.ObjectIndex-1]
		if record.Class != "Unit" {
			continue
		}
		mapID, mapErr := savedStructureValue(record, "T08")
		typeID, typeErr := savedStructureValue(record, "T0E")
		face, faceErr := savedStructureValue(record, "U4B")
		if mapErr != nil || typeErr != nil || faceErr != nil || mapID != uint32(e.MapUnitID) || typeID != uint32(e.TypeID) || face == 0 {
			continue
		}
		tiers[e.ID] = int(face)
	}
	for _, e := range entities {
		if e.SourceBinding.ActorClass() == 1 && e.SourceBinding.Face != 0 {
			tiers[e.ID] = int(e.SourceBinding.Face)
		}
	}
	if m == nil {
		return tiers
	}
	byMapID := make(map[uint16]int, len(m.Units))
	for i, unit := range m.Units {
		if unit.UnitID == 0 {
			continue
		}
		if _, exists := byMapID[unit.UnitID]; exists {
			byMapID[unit.UnitID] = -1
		} else {
			byMapID[unit.UnitID] = i
		}
	}
	for _, e := range entities {
		if _, found := tiers[e.ID]; found || e.MapUnitID == 0 || e.Humanoid {
			continue
		}
		i, found := byMapID[e.MapUnitID]
		if !found || i < 0 || int32(uint8(m.Units[i].ClassID)) != e.TypeID {
			continue
		}
		if tier, found := placed[sim.EntityID(i)]; found {
			tiers[e.ID] = tier
		}
	}
	return tiers
}
