package game

import (
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func (mw *mapWorld) useStructure(entity, structure uint32) {
	mw.cancelPickup(sim.EntityID(entity))
	mw.pending = append(mw.pending, sim.UseStructure(sim.EntityID(entity), sim.StructureID(structure)))
}

func (mw *mapWorld) restoreStructureUseMetadata(table *mapload.Table) {
	if sources, _, present := mw.world.SavedStructures(); present {
		fresh := mw.world.Structures()
		for i := range fresh {
			for _, source := range sources {
				if source.ID == fresh[i].ID {
					mapload.StructureUseMetadata(&fresh[i], uint16(source.Kind), table)
					break
				}
			}
		}
		mw.world.RestoreStructureUseMetadata(fresh)
	}
}
