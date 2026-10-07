package game

import (
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
)

// Current-state persistence owns this no-resurrection rule. SAV-RECON-268/269
// and SAV-CELLLOAD-108 establish graph-before-terrain loading, not this exact
// absent-placement join. Only a complete living/dead graph and unique authored
// map IDs authorize it; incomplete documents retain the existing load policy.
func retainSavedActorPlacements(m *alm.Map, graph sav.SavedActorGraph, dead []sav.DeadActor, state *SnapshotSAVDocument) {
	if m == nil || state == nil || state.Document == nil || state.Document.World == nil {
		return
	}
	if graph.CurrentPopulation {
		present := make(map[uint16]bool)
		current, placements := make(map[uint16]int), make(map[uint16]int)
		for _, actor := range graph.Actors {
			if actor.CurrentEntity && actor.MapUnitID != 0 {
				present[actor.MapUnitID] = true
				current[actor.MapUnitID]++
			}
		}
		for _, actor := range dead {
			if actor.MapUnitID != 0 {
				present[actor.MapUnitID] = true
			}
		}
		for _, unit := range m.Units {
			placements[unit.UnitID]++
		}
		m.Units = slices.DeleteFunc(slices.Clone(m.Units), func(unit alm.Unit) bool {
			// Exact current objects own ambiguous native map IDs separately.
			// Their ordinary records construct actors without selecting an ALM
			// duplicate or leaving the unselected placement alive beside them.
			return !present[unit.UnitID] || current[unit.UnitID] != 0 && (current[unit.UnitID] != 1 || placements[unit.UnitID] != 1)
		})
		return
	}
	keys := make(map[uint32]uint16)
	for i, record := range state.Document.Objects {
		if record.Class == "Unit" || record.Class == "Human" || record.Class == "Humanoid" {
			key, err := savedStructureValue(&record, "Identity")
			if err != nil || key == 0 || keys[key] != 0 {
				return
			}
			keys[key] = uint16(i + 1)
		}
	}
	seen := make(map[uint32]bool)
	present := make(map[uint16]bool)
	for _, actor := range graph.Actors {
		if keys[actor.Identity] == 0 {
			return
		}
		seen[actor.Identity] = true
		present[actor.MapUnitID] = true
	}
	for _, actor := range dead {
		if keys[actor.Identity] == 0 {
			return
		}
		seen[actor.Identity] = true
		present[actor.MapUnitID] = true
	}
	if len(keys) == 0 || len(keys) != len(seen) {
		return
	}
	counts := make(map[uint16]int)
	for _, unit := range m.Units {
		counts[unit.UnitID]++
	}
	m.Units = slices.DeleteFunc(slices.Clone(m.Units), func(unit alm.Unit) bool {
		return unit.UnitID != 0 && counts[unit.UnitID] == 1 && !present[unit.UnitID]
	})
}
