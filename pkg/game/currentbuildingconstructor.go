package game

import (
	"fmt"

	"againrom/pkg/mod"
	"againrom/pkg/sim"
)

func constructCurrentBuildings(ms *Mission, src entrySource, groups *mod.TerrainData) error {
	if ms == nil || ms.World == nil || ms.Map == nil {
		return fmt.Errorf("current Building constructor lacks a mission")
	}
	w := ms.World
	live := w.Structures()
	if len(live) == 0 {
		return nil
	}
	planes, err := currentSpatialPlanes(w, ms.Map, src, groups)
	if err != nil {
		return err
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		return err
	}
	b := generatedDocumentBuilder{nextKey: 0x51000000}
	b.reserveCurrentForm(raw)
	terrain := b.identity()
	sources := make([]sim.SavedStructure, len(live))
	for i, st := range live {
		if st.Col < 0 || st.Col > 255 || st.Row < 0 || st.Row > 255 {
			return fmt.Errorf("current Building constructor position exceeds cell bytes")
		}
		s := sim.SavedStructure{ID: st.ID, Class: sim.GeneratedBuilding, SourceKey: b.identity(),
			Kind: uint8(st.Kind), Token0E: st.Kind, Token18: 2, Blocking: st.Blocking}
		copy(s.Position[:], constructedPositionBlock(st.Col, st.Row, terrain))
		// Base52 keeps the Building constructor's zero record (SAV-1385).
		if uint64(st.ID) < uint64(len(ms.Map.Objects)) {
			if object := ms.Map.Objects[st.ID]; object.Field12 != 0 {
				s.AuthoredID, s.AuthoredIndex, s.HasAuthored = uint32(object.Field12), uint32(st.ID), true
			}
		}
		sources[i] = s
	}
	return w.ConstructSavedStructures(sources, currentBuildingCells(w, planes))
}

// currentBuildingCells is the cell row of every cell a structure of w's
// roster occupies, with the baselines planes holds. The constructor and the
// LOAD of a collapsed variable extent (restoreVariableExtents) build their
// rows here.
func currentBuildingCells(w *sim.World, planes *sim.SavedCellPlanes) []sim.SavedStructureCell {
	occupied := w.StructureOccupancy()
	cells := make([]sim.SavedStructureCell, len(occupied))
	for i, cell := range occupied {
		cells[i] = sim.SavedStructureCell{Cell: cell.Cell, ID: cell.ID, HasStructure: true,
			BaselineCost: planes.Cost[cell.Cell], BaselineStatic: planes.Static[cell.Cell]}
	}
	return cells
}
