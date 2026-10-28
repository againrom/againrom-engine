package game

import (
	"againrom/pkg/formats/alm"
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
	if mw.mission != nil && mw.mission.state != nil {
		restorePointerSlots(mw.world, mw.mission.state.Map, table)
	}
}

// restorePointerSlots gives a loaded Pointer the action slot its map placement
// derives, matched by ID, kind and cell, when the loaded form carried none.
func restorePointerSlots(w *sim.World, m *alm.Map, table *mapload.Table) {
	placed := mapload.Structures(m, table)
	fresh := w.Structures()
	for i, s := range fresh {
		id := int(s.ID)
		if sim.PointerKind(s.Kind) && id < len(placed) && placed[id].Kind == s.Kind && placed[id].Col == s.Col && placed[id].Row == s.Row {
			fresh[i].UseAmount = placed[id].UseAmount
		}
	}
	w.RestoreStructureUseMetadata(fresh)
}
