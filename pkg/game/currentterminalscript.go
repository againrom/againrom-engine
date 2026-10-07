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
func restoreTerminalScriptBindings(ms *Mission) error {
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
	roles, _, err := mapload.CompileScript(ms.Map, mapload.ScriptRefs{Units: units, Structures: mapload.ScriptStructures(ms.Map)})
	if err != nil {
		return err
	}
	return restoreCurrentScriptRoles(ms.World, roles)
}
