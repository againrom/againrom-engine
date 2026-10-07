package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// applyOriginalFacings uses the exact source offsets that built the party,
// including zero-map-ID companions. Other actors keep the existing unique
// MapUnitID join. This repairs current facing, not full actor materialization
// or incoming mover/order state. City imports do not call this function.
func applyOriginalFacings(ms *Mission, source []sav.Actor, partyOffsets []int) error {
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original facing: mission has no world")
	}
	if ms.actorRegistry != nil {
		var batch []sim.OriginalActorFacing
		for _, binding := range ms.actorRegistry.actors {
			batch = append(batch, sim.OriginalActorFacing{ID: binding.ID, Facing: binding.Source.Facing})
		}
		return ms.World.ImportOriginalActorFacings(batch)
	}
	if len(partyOffsets) != 0 && len(partyOffsets) != len(ms.Start.IDs) {
		return fmt.Errorf("original facing: party source/binding count differs")
	}
	byOffset := make(map[int]sav.Actor, len(source))
	for _, a := range source {
		if _, exists := byOffset[a.Off]; exists {
			return fmt.Errorf("original facing: repeated source offset %d", a.Off)
		}
		byOffset[a.Off] = a
	}
	partyIDs := make(map[sim.EntityID]bool, len(ms.Start.IDs))
	for _, id := range ms.Start.IDs {
		partyIDs[id] = true
	}
	partySource := make(map[int]bool, len(partyOffsets))
	var batch []sim.OriginalActorFacing
	for i, off := range partyOffsets {
		a, ok := byOffset[off]
		if !ok || partySource[off] {
			return fmt.Errorf("original facing: missing or repeated party source offset %d", off)
		}
		partySource[off] = true
		if !a.Dead() {
			batch = append(batch, sim.OriginalActorFacing{ID: ms.Start.IDs[i], Facing: a.Facing})
		}
	}
	byMapID := make(map[uint16][]sim.Entity)
	for _, e := range ms.World.Entities() {
		if !partyIDs[e.ID] && e.MapUnitID != 0 && e.Alive() && !e.OffMap {
			byMapID[e.MapUnitID] = append(byMapID[e.MapUnitID], e)
		}
	}
	seen := make(map[uint16]bool)
	for _, a := range source {
		if partySource[a.Off] || a.Dead() || a.MapUnitID == 0 {
			continue
		}
		targets := byMapID[a.MapUnitID]
		if len(targets) == 0 {
			continue
		}
		if len(targets) != 1 || seen[a.MapUnitID] {
			return fmt.Errorf("original facing: ambiguous MapUnitID %d", a.MapUnitID)
		}
		seen[a.MapUnitID] = true
		batch = append(batch, sim.OriginalActorFacing{ID: targets[0].ID, Facing: a.Facing})
	}
	return ms.World.ImportOriginalActorFacings(batch)
}
