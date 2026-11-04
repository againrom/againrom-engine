package game

import (
	"fmt"
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) checkTrapArea(r ui.InspectionNewRecord) error {
	kind, ok := spellTemplateKind(s.model, recordValue(r, "template"))
	view, err := s.model.Map()
	if err != nil {
		return err
	}
	if r.Kind != ui.KindEnchantment || !ok || kind >= 4 {
		return fmt.Errorf("a trap area requires a trap template")
	}
	if !r.Area.In(image.Rect(0, 0, min(view.Width, 256), min(view.Height, 256))) || image.Pt(0, 0).In(r.Area) || r.Cell != r.Area.Min {
		return fmt.Errorf("trap area must fit the nonzero byte-coordinate map")
	}
	return nil
}

func (s *mapInspectionEdits) addTrapArea(m *mapedit.Editor, k newRecordKind, r ui.InspectionNewRecord, index *int) (bool, error) {
	if err := s.checkTrapArea(r); err != nil {
		return false, err
	}
	mark := m.Mark()
	var indices []int
	for y := r.Area.Min.Y; y < r.Area.Max.Y; y++ {
		for x := r.Area.Min.X; x < r.Area.Max.X; x++ {
			r.Cell = image.Pt(x, y)
			i, err := s.addOn(m, k, r)
			if err != nil {
				return false, err
			}
			indices = append(indices, i)
			*index = i
		}
	}
	if err := m.GroupSpellRecords(indices); err != nil {
		return false, err
	}
	m.Squash(mark)
	return true, nil
}
