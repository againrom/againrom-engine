package game

import (
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// secondFrom places a record on cell of the second-game map m as a copy of
// the template record, which keeps every byte the format notes do not read.
// The copy takes its own id, sack and bound elements from the section row.
func (s *mapInspectionEdits) secondFrom(m *mapedit.Editor, k mapedit.Kind, template int, cell image.Point) (int, error) {
	x, y, err := m.Position(k, template)
	if err != nil {
		return -1, err
	}
	nx, ny := placeWords(m, cell, x, y)
	return m.CloneRecord(k, template, nx, ny)
}

// secondUnitFrom adds a unit who is id: a copy of a unit of the map who is
// id, owned by the map's second roster slot when it has one.
func (s *mapInspectionEdits) secondUnitFrom(m *mapedit.Editor, cell image.Point, id mapload.Identity) (int, error) {
	view, err := m.Map()
	if err != nil {
		return -1, err
	}
	for i, u := range view.Units {
		if got, _, ok := mapload.IdentifyAt(u, s.inspector.table, s.mission()); !ok || got != id {
			continue
		}
		at, err := s.secondFrom(m, mapedit.KindUnit, i, cell)
		if err != nil {
			return -1, err
		}
		owner := int64(min(2, max(1, len(view.Groups))))
		return at, m.SetField(mapedit.KindUnit, at, "owner", owner)
	}
	return -1, wordErr("editor.map.secondtemplate")
}

// secondStructureFrom adds a structure of kind: a copy of a structure of the
// map of that kind.
func (s *mapInspectionEdits) secondStructureFrom(m *mapedit.Editor, cell image.Point, kind int64) (int, error) {
	view, err := m.Map()
	if err != nil {
		return -1, err
	}
	for i, o := range view.Objects {
		if int64(uint8(o.Kind)) == kind {
			return s.secondFrom(m, mapedit.KindStructure, i, cell)
		}
	}
	return -1, wordErr("editor.map.secondtemplate")
}

// addable is the kinds the second game's palettes place: units, structures
// and ground sacks.
func (c secondEdits) addable(k ui.RecordKind) bool {
	return k == ui.KindUnit || k == ui.KindStructure || k == ui.KindSack
}
