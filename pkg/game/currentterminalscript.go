package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// restoreTerminalScriptBindings rebinds script references to authored units
// whose actors had already left the world when the SAV was written. LOAD
// constructs no placement for such an actor, so its reference compiles
// absent; the terminal row names the placement, and the reference follows the
// row the dead list still answers for. Bound references, and a placement two
// rows name, are left as they are.
func restoreTerminalScriptBindings(ms *Mission, table *mapload.Table) error {
	if ms.Map == nil || ms.World == nil || ms.World.Script() == nil {
		return nil
	}
	units := map[uint16]sim.EntityID{}
	shared := map[uint16]bool{}
	for _, row := range ms.World.CurrentTerminalActors() {
		if row.MapUnitID == 0 {
			continue
		}
		if _, seen := units[row.MapUnitID]; seen {
			shared[row.MapUnitID] = true
		}
		units[row.MapUnitID] = row.ID
	}
	for unit := range shared {
		delete(units, unit)
	}
	if len(units) == 0 {
		return nil
	}
	// The party's roster decides the same unbuilt nodes the program omitted.
	var refs mapload.ScriptRefs
	if len(ms.Party) == len(ms.Start.IDs) {
		refs = campaignScriptPartyRefs(ms.Map, table, ms.Party, func(i int) sim.EntityID { return ms.Start.IDs[i] }, loadedPlacedHeroes(ms, table))
	}
	refs.Units, refs.Structures = units, mapload.ScriptStructures(ms.Map)
	roles, err := currentScriptRolesProgram(ms.World, ms.Map, refs, mapload.CompileScript)
	if err != nil {
		return err
	}
	return restoreCurrentScriptRoles(ms.World, roles)
}
