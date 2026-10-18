package game

import (
	"fmt"

	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) PreviewPlacement(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	k, err := newRecordKindOf(r.Kind)
	if err != nil {
		return nil, err
	}
	if !k.complete(r) {
		return nil, fmt.Errorf("incomplete placement")
	}
	m, _, err := s.trialAdd(k, r)
	if err != nil {
		return nil, err
	}
	var pages map[string][]byte
	if s.workspace != nil {
		pages = s.workspace.Pages
	}
	return s.inspector.inspectModel(m, s.source, pages)
}
