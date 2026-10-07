package game

import "againrom/pkg/render/terrain"

func (mw *mapWorld) pushSavedStructureRoster() {
	source, _, present := mw.world.SavedStructures()
	if !present {
		return
	}
	structures := mw.world.Structures()
	records := make([]terrain.StructureRecord, len(source))
	for i, s := range source {
		st := structures[i]
		records[i] = terrain.StructureRecord{ID: uint32(st.ID), X: uint32(st.Col) << 8,
			Y: uint32(st.Row) << 8, Key: uint32(s.Kind)}
		if s.Kind == 0x21 {
			records[i].VariableWidth, records[i].VariableHeight = int(st.Width), int(st.Height)
		}
	}
	mw.view.SetStructureRecords(records)
}
