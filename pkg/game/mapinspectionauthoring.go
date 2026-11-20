package game

import (
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) AddStructure(cell image.Point, kind int64) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := s.admitCell(m, cell, "structure cell", ui.KindStructure, admitAdd); err != nil {
			return false, err
		}
		if rowFor(m).templatePlacing {
			_, err := s.structureFromTemplate(m, cell, kind)
			return err == nil, err
		}
		if err := s.inspector.checkStructureKind(kind); err != nil {
			return false, err
		}
		view, err := m.Map()
		if err != nil {
			return false, err
		}
		if err := rosterSlotError(view, 1); err != nil {
			return false, err
		}
		x, y := cellWords(cell, newSackFraction, newSackFraction)
		rec, err := m.NewStructureRecord(x, y, uint32(kind), 1)
		if err != nil {
			return false, err
		}
		i, err := m.Place(mapedit.KindStructure, rec)
		if err != nil {
			return false, err
		}
		return true, s.checkPlacement(admissionOf(ui.KindStructure, admitAdd), mapedit.KindStructure, m, i, image.Rectangle{})
	})
}

func (s *mapInspectionEdits) AddStock(unit int) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	return s.change(func(m *mapedit.Editor) (bool, error) {
		_, err := m.AddUnitStock(unit)
		return err == nil, err
	})
}

func (s *mapInspectionEdits) AddItemRecipe(kind ui.RecordKind, index int, item int64) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	if kind != ui.KindSack && kind != ui.KindStock {
		return nil, wordErr("editor.err.recipepick")
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if item < 1 || item > int64(^uint(0)>>1) {
			return false, wordErr("editor.err.itemnumber")
		}
		_, err := m.AddItemRecipe(index, int(item))
		return err == nil, err
	})
}
