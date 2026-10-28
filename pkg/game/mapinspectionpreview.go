package game

import (
	"fmt"
	"image"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// PreviewPlacement is the map as it would stand after r: the Add window's
// record added, or for a paste brush's clip, the clip pasted with its origin
// on r.Cell by PasteRecords on a copy of the model. An Add whose footprint
// checkPlacement refuses answers the map with the refused record on it and
// the refusal; a paste answers the records it refuses in Refused.
func (s *mapInspectionEdits) PreviewPlacement(r ui.InspectionNewRecord) (*ui.InspectionDocument, error) {
	var m *mapedit.Editor
	var refused []ui.SelectionRefusal
	var placement error
	if r.Clip != nil {
		trial := *s
		trial.model = s.model.Clone()
		trial.batch = trial.model
		edit, err := trial.PasteRecords(r.Clip, r.Cell)
		if err != nil {
			return nil, err
		}
		m, refused = trial.model, edit.Refused
	} else {
		k, err := newRecordKindOf(r.Kind)
		if err != nil {
			return nil, err
		}
		if !k.complete(r) {
			return nil, fmt.Errorf("incomplete placement")
		}
		if m, _, err = s.trialAdd(k, r); m == nil {
			return nil, err
		}
		placement = err
	}
	var pages map[string][]byte
	if s.workspace != nil {
		pages = s.workspace.Pages
	}
	d, err := s.inspector.inspectModel(m, s.source, pages)
	if err != nil {
		return nil, err
	}
	d.Refused = refused
	return d, placement
}

// CheckMove is the records of refs that moving them by offset as one group
// edit would refuse: MoveRecords, or MoveArea when box is not empty, run on
// a copy of the model.
func (s *mapInspectionEdits) CheckMove(refs []ui.RecordRef, box image.Rectangle, offset image.Point) []ui.RecordRef {
	trial := *s
	trial.model = s.model.Clone()
	trial.batch = trial.model
	var edit ui.SelectionEdit
	var err error
	if box.Empty() {
		edit, err = trial.MoveRecords(refs, offset)
	} else {
		edit, err = trial.MoveArea(refs, box, offset)
	}
	if err != nil {
		return refs
	}
	var out []ui.RecordRef
	for _, r := range edit.Refused {
		out = append(out, r.Ref)
	}
	return out
}
