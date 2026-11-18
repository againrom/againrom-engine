package ui

import (
	"fmt"
	"testing"
)

type namedRecordHost struct{ *modesFull }

func (h *namedRecordHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "identity" || key == "kind" || key == "item" {
		return InspectionNewStep{Choices: []InspectionChoice{{Value: 10, Label: "10: First named record"}, {Value: 20, Label: "20: Second named record"}}, Value: 10}
	}
	return h.modesFull.NewRecordStep(r, key)
}

// CheckNewRecord refuses a named value off the list, as a host's table does.
func (h *namedRecordHost) CheckNewRecord(r InspectionNewRecord, key string) error {
	for _, v := range r.Values {
		if v.Key == key && (key == "identity" || key == "kind" || key == "item") && v.Value != 10 && v.Value != 20 {
			return fmt.Errorf("no record %d", v.Value)
		}
	}
	return nil
}

func TestMapEditorNewRecordNamesAreVisibleBeforeCommit(t *testing.T) {
	for _, key := range []string{"structure", "unit", "sack"} {
		t.Run(key, func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			host := &namedRecordHost{e.doc.Edits.(*modesFull)}
			host.self, e.doc.Edits = host, host
			_ = e.Key(key)
			r, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: 1})
			if !ok {
				t.Fatal("palette has no second choice")
			}
			cell, _, _ := e.gridCellAt(center(r))
			if cell.Tip != "20: Second named record" {
				t.Fatal("palette does not name the second choice", cell.Tip)
			}
			chromeClick(e, center(r))
			if !e.addPicking() || len(calls.lines) != 0 {
				t.Fatal("selecting a choice did not arm placement without writing")
			}
		})
	}
}
