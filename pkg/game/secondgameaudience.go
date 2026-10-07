package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// DIV-2381: selected definition words bridge the unmeasured packet producer.
func secondGameNPCKeys(ms *Mission, table *mapload.Table) map[sim.EntityID]uint16 {
	if ms == nil || ms.Map == nil || ms.World == nil || table == nil {
		return nil
	}
	keys := make(map[sim.EntityID]uint16)
	for _, placed := range placedEntities(ms.Map, ms.World.Entities(), ms.Start.Roster, ms.savedDocument) {
		if placed.index < 0 || placed.index >= len(ms.Map.Units) {
			continue
		}
		resolved := mapload.Resolve(ms.Map.Units[placed.index], table)
		if !resolved.Found() {
			continue
		}
		var rows data.Collection = table.Units
		slot := 55
		human := resolved.Arm == mapload.ArmServerID
		if human {
			rows, slot = table.Humans, 24
		}
		if rows == nil {
			continue
		}
		params := rows.EntryParams(resolved.Index)
		if len(params) <= slot {
			continue
		}
		key := uint16(params[slot])
		if human && key > 10000 {
			key = (key / 10) % 1000
		}
		keys[placed.id] = key
	}
	return keys
}
