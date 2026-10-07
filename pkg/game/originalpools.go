package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// Both original-load doors share the ordering: stock can Rearm and replace
// maxima, so saved pools must be the last actor-state write before publication.
func restoreOriginalActors(ms *Mission, holdings []sav.ActorHoldings, pools []sav.ActorPools,
	books []sav.ActorSpellbook, table *mapload.Table, report *OriginalSaveResume, profiles []sav.ActorCurrent) error {
	if err := restoreOriginalActorStock(ms, holdings, table, report); err != nil {
		return err
	}
	if err := applyOriginalActorSpellbooks(ms, books, report); err != nil {
		return err
	}
	if err := applyOriginalActorProfiles(ms, profiles, report); err != nil {
		return err
	}
	return applyOriginalActorPools(ms, pools, report)
}

// applyOriginalActorPools is deliberately independent of the heuristic head
// position and stock joins. All eligible source and destination identities are
// counted before the sim batch can write. Missing actors are an existing import
// boundary, not permission to create actors or refuse every ordinary save.
func applyOriginalActorPools(ms *Mission, source []sav.ActorPools, r *OriginalSaveResume) error {
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original actor pools: mission has no map/world")
	}
	party := make(map[sim.EntityID]bool, len(ms.Start.IDs))
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	byMapID := make(map[uint16][]sim.Entity)
	for _, entity := range ms.World.Entities() {
		if entity.MapUnitID != 0 {
			byMapID[entity.MapUnitID] = append(byMapID[entity.MapUnitID], entity)
		}
	}
	var batch []sim.OriginalActorPools
	var current []sim.EntityID
	seen := make(map[uint16]bool)
	for _, saved := range source {
		var target sim.Entity
		if ms.actorRegistry != nil {
			bound, err := registryTarget(ms, saved.Off)
			if err != nil {
				return err
			}
			if bound == nil {
				r.PoolsExcluded++
				continue
			}
			target = *bound
			if binding, ok := ms.actorRegistry.actor(saved.Off); ok && binding.Source.CurrentEntity {
				current = append(current, target.ID)
			}
		} else {
			if saved.MapUnitID != 0 && originalPartyCarriesMapUnit(ms.Party, saved.MapUnitID) {
				r.PoolsParty++
				continue
			}
			if saved.Stage != 0 || int16(saved.HP) <= 0 || saved.MapUnitID == 0 ||
				int(saved.Cell&255) >= int(ms.Map.Width) || int(saved.Cell>>8) >= int(ms.Map.Height) {
				r.PoolsExcluded++
				continue
			}
			targets := byMapID[saved.MapUnitID]
			if len(targets) == 0 {
				r.PoolsUnmatched++
				continue
			}
			if len(targets) != 1 {
				return fmt.Errorf("original actor pools: ambiguous MapUnitID %d at saved offset %d", saved.MapUnitID, saved.Off)
			}
			target = targets[0]
			if party[target.ID] {
				r.PoolsParty++
				continue
			}
			if !target.Alive() || target.OffMap {
				r.PoolsExcluded++
				continue
			}
			if seen[saved.MapUnitID] {
				return fmt.Errorf("original actor pools: ambiguous MapUnitID %d at saved offset %d", saved.MapUnitID, saved.Off)
			}
			seen[saved.MapUnitID] = true
		}
		batch = append(batch, sim.OriginalActorPools{ID: target.ID,
			HP: int32(int16(saved.HP)), MaxHP: int32(saved.MaxHP),
			Mana: int32(saved.Mana), MaxMana: int32(saved.MaxMana)})
	}
	if err := ms.World.ImportOriginalActorPools(batch, current...); err != nil {
		return err
	}
	r.PoolsRestored = len(batch)
	return nil
}
