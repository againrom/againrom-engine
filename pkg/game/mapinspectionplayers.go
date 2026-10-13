package game

import (
	"fmt"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) AddPlayer() (*ui.InspectionDocument, error) {
	return s.change(func(m *mapedit.Editor) (bool, error) {
		rec, err := m.NewPlayerRecord()
		if err != nil {
			return false, err
		}
		_, err = m.Place(mapedit.KindPlayer, rec)
		return err == nil, err
	})
}

func (s *mapInspectionEdits) SetTextField(kind string, index int, key, value string) (*ui.InspectionDocument, error) {
	k, err := editKind(kind)
	if err != nil {
		return nil, err
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		old, err := m.TextFieldValue(k, index, key)
		if err != nil || old == value {
			return false, err
		}
		err = m.SetTextField(k, index, key, value)
		return err == nil, err
	})
}

func inspectionPlayer(model *mapedit.Editor, index int, name string) ui.InspectionRecord {
	r := ui.InspectionRecord{Kind: "Player", Section: 5, Index: index, Label: fmt.Sprintf("%d: %s", index+1, name)}
	fields, err := model.Fields(mapedit.KindPlayer, index)
	if err != nil {
		r.Warning = err.Error()
		return r
	}
	for _, f := range fields {
		if f.Text {
			continue
		}
		v, err := model.FieldValue(mapedit.KindPlayer, index, f.Key)
		if err != nil {
			r.Warning = err.Error()
			continue
		}
		r.Details = append(r.Details, fmt.Sprintf("%s: %d", f.Label, v))
	}
	return r
}
