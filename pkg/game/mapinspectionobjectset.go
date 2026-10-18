package game

import (
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) CheckObjectCell(kind string, index int, cell image.Point) error {
	trial := *s
	trial.model = s.model.Clone()
	trial.batch = trial.model
	_, err := trial.Move(kind, index, cell)
	return err
}

func (s *mapInspectionEdits) SetObject(kind string, index int, values []ui.InspectionFieldValue, cell *image.Point) (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		trial := *s
		trial.model, trial.batch = m, m
		mark := m.Mark()
		if cell != nil {
			if _, err := trial.Move(kind, index, *cell); err != nil {
				return false, err
			}
		}
		if _, err := trial.SetFields(kind, index, values); err != nil {
			return false, err
		}
		changed := m.Mark() != mark
		m.Squash(mark)
		return changed, nil
	})
}
