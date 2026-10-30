package game

import (
	"fmt"
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) AddStructure(cell image.Point, kind int64) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if err := insideMap(m, cell, "structure cell", "Structure"); err != nil {
			return false, err
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
		return true, s.checkPlacement(mapedit.KindStructure, m, i, image.Rectangle{})
	})
}

func (s *mapInspectionEdits) AddStock(unit int) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		_, err := m.AddUnitStock(unit)
		return err == nil, err
	})
}

func (s *mapInspectionEdits) AddItemRecipe(kind string, index int, item int64) (*ui.InspectionDocument, error) {
	if kind != "Sack" && kind != "Stock" {
		return nil, fmt.Errorf("select a sack or stock before creating an item recipe")
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		if item < 1 || item > int64(^uint(0)>>1) {
			return false, fmt.Errorf("item number must be positive")
		}
		_, err := m.AddItemRecipe(index, int(item))
		return err == nil, err
	})
}
