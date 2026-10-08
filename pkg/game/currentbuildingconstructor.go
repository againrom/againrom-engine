package game

import (
	"encoding/binary"
	"fmt"

	"againrom/pkg/sim"
)

func constructCurrentBuildings(ms *Mission, src entrySource) error {
	if ms == nil || ms.World == nil || ms.Map == nil {
		return fmt.Errorf("current Building constructor lacks a mission")
	}
	w := ms.World
	live := w.Structures()
	if len(live) == 0 {
		return nil
	}
	planes, err := currentSpatialPlanes(w, ms.Map, src)
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
		s := sim.SavedStructure{ID: st.ID, Class: sim.GeneratedBuilding, SourceKey: b.identity(), RuntimeID: b.runtime(),
			Kind: uint8(st.Kind), Token0E: st.Kind, Token18: 2, Blocking: st.Blocking}
		copy(s.Position[:], constructedPositionBlock(st.Col, st.Row, terrain))
		s.Base52[14], s.Base52[15] = st.Width, st.Height
		binary.LittleEndian.PutUint32(s.Base52[18:], st.Blocking)
		if uint64(st.ID) < uint64(len(ms.Map.Objects)) {
			if object := ms.Map.Objects[st.ID]; object.Field12 != 0 {
				s.AuthoredID, s.AuthoredIndex, s.HasAuthored = uint32(object.Field12), uint32(st.ID), true
			}
		}
		sources[i] = s
	}
	occupied := w.StructureOccupancy()
	cells := make([]sim.SavedStructureCell, len(occupied))
	for i, cell := range occupied {
		cells[i] = sim.SavedStructureCell{Cell: cell.Cell, ID: cell.ID, HasStructure: true,
			BaselineCost: planes.Cost[cell.Cell], BaselineStatic: planes.Static[cell.Cell]}
	}
	return w.ConstructSavedStructures(sources, cells)
}
