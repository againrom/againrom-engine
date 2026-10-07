package ui

import (
	"againrom/pkg/render/terrain"
	"slices"
)

// SetStructureRecords replaces the current roster, including with an empty
// roster. Rebuild both geometries and the inspection class/plane-order caches;
// updating health alone would keep drawing removed ALM placements.
func (v *Viewer) SetStructureRecords(records []terrain.StructureRecord) {
	if slices.Equal(v.grid.Structures, records) {
		return
	}
	v.grid.Structures = slices.Clone(records)
	v.structuresFlat, v.structureCounts, v.structureAnimFlat = terrain.StructurePlacements(v.grid, v.structureSet, nil, 0)
	v.structuresDisplaced, v.structureAnimDisplaced = nil, nil
	if v.proj != nil {
		v.structuresDisplaced, _, v.structureAnimDisplaced = terrain.StructurePlacements(v.grid, v.structureSet, v.proj.Altitude, v.proj.MinV)
	}
	v.structureInfo = make(map[uint32]*terrain.StructureClass, len(records))
	if v.structureSet != nil {
		for _, r := range records {
			v.structureInfo[r.ID] = v.structureSet.Classes[r.Key&0xff]
		}
	}
	v.planeOrder = terrain.PlaneOrder(v.structuresFlat, v.staticsFlat)
	v.depthOrder = nil
}
