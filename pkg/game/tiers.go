package game

import (
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// entityTiers keys each resolved Units row's tier by the world builder's
// minted entity ID: placement i becomes entity i in the same decoded map.
// Other resolution arms and invalid definitions contribute no entry. An
// absent tier draws the sheet's own colours. The lookup is a pure function
// of the map and table; nil inputs state no tiers.
func entityTiers(m *alm.Map, t *mapload.Table) map[sim.EntityID]int {
	if m == nil || t == nil || t.Units == nil {
		return nil
	}
	out := make(map[sim.EntityID]int, len(m.Units))
	for i, u := range m.Units {
		if tier, ok := placementTier(u, t); ok {
			out[sim.EntityID(i)] = tier
		}
	}
	return out
}

// placementTier reads the resolved Units row's face, leaving art bounds to TierFrames.
func placementTier(u alm.Unit, t *mapload.Table) (int, bool) {
	if t == nil || t.Units == nil {
		return 0, false
	}
	r := mapload.Resolve(u, t)
	if r.Arm != mapload.ArmUnits || !r.Found() {
		return 0, false
	}
	d, err := data.NewUnitDef(t.Units.EntryName(r.Index), t.Units.EntryParams(r.Index))
	return int(d.Face), err == nil
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
