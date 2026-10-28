package game

import (
	"fmt"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// PreviewPlacement is the map as it would stand after r: the Add window's
// record added, or for a paste brush's clip, the clip pasted with its origin
// on r.Cell by PasteRecords on a copy of the model.
func (s *mapInspectionEdits) PreviewPlacement(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	var m *mapedit.Editor
	if r.Clip != nil {
		trial := *s
		trial.model = s.model.Clone()
		trial.batch = trial.model
		if _, err := trial.PasteRecords(r.Clip, r.Cell); err != nil {
			return nil, err
		}
		m = trial.model
	} else {
		k, err := newRecordKindOf(r.Kind)
		if err != nil {
			return nil, err
		}
		if !k.complete(r) {
			return nil, fmt.Errorf("incomplete placement")
		}
		if m, _, err = s.trialAdd(k, r); err != nil {
			return nil, err
		}
	}
	var pages map[string][]byte
	if s.workspace != nil {
		pages = s.workspace.Pages
	}
	return s.inspector.inspectModel(m, s.source, pages)
}
