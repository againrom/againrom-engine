package ui

import (
	"fmt"
	"strings"
	"testing"
)

type namedRecordHost struct{ *modesFull }

func (h *namedRecordHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	if key == "class" || key == "kind" || key == "item" {
		return InspectionNewStep{Choices: []InspectionChoice{{Value: 10, Label: "10: First named record"}, {Value: 20, Label: "20: Second named record"}}, Value: 10}
	}
	return h.modesFull.NewRecordStep(r, key)
}

// CheckNewRecord refuses a named value off the list, as a host's table does.
func (h *namedRecordHost) CheckNewRecord(r InspectionNewRecord, key string) error {
	for _, v := range r.Values {
		if v.Key == key && (key == "class" || key == "kind" || key == "item") && v.Value != 10 && v.Value != 20 {
			return fmt.Errorf("no record %d", v.Value)
		}
	}
	return nil
}

// addFocus is the Add window's asked row.
func addFocus(t *testing.T, e *MapEditor) popupRow {
	t.Helper()
	s, _ := e.addSpecOK()
	for _, r := range s.Rows {
		if r.Focus {
			return r
		}
	}
	t.Fatal("the Add window asks no row")
	return popupRow{}
}

// The Add window names each choice before the record is added; Up and Down
// walk the list, and the chosen value reaches the host's add. A typed value
// the host's check refuses is refused with its reason and nothing is
// written.
func TestMapEditorNewRecordNamesAreVisibleBeforeCommit(t *testing.T) {
	for _, key := range []string{"structure", "unit", "sack"} {
		t.Run(key, func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			host := &namedRecordHost{e.doc.Edits.(*modesFull)}
			host.self, e.doc.Edits = host, host
			_ = e.Key(key)
			if key == "sack" {
				e.TextInput("25,20")
				_ = e.Key("enter")
			}
			r, _ := e.addRunning()
			if r.list == nil || r.list.Len() != 2 || r.list.Rows()[0].Text != "10: First named record" {
				t.Fatal("the list does not name the choices", r.list)
			}
			if key == "sack" { // the many step starts empty; Down takes the highlight
				_ = e.Key("down")
			}
			_ = e.Key("down")
			_ = e.Key("down")
			if e.prompt.path != "20" || len(calls.lines) != 0 {
				t.Fatal("Down did not clamp to the last choice", e.prompt.path, calls.lines)
			}
			_ = e.Key("up")
			_ = e.Key("down")
			if row := addFocus(t, e); key != "sack" && row.Value != "20: Second named record" {
				t.Fatal("the asked row does not name the choice", row.Value)
			}
			e.TextInput("0")
			_ = e.Key("enter")
			if r.note != "no record 200" || len(calls.lines) != 0 {
				t.Fatal("a value outside the choices was taken", r.note, calls.lines)
			}
			_ = e.Key("backspace")
			for _, k := range []string{"enter", "enter", "ui:text-cell", "enter"} {
				if k == "ui:text-cell" {
					if key != "sack" {
						e.TextInput("25,20")
					}
					continue
				}
				_ = e.Key(k)
			}
			if len(calls.lines) != 1 || !strings.Contains(calls.lines[0], " 20 ") {
				t.Fatal("the choice did not reach the add", calls.lines)
			}
		})
	}
}
