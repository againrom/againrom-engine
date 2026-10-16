package ui

import (
	"strings"
	"testing"
)

type namedRecordHost struct{ *modesFull }

func (h *namedRecordHost) RecordChoices(kind string, index int, key string) []InspectionChoice {
	return []InspectionChoice{{Value: 10, Label: "10: First named record"}, {Value: 20, Label: "20: Second named record"}}
}

func TestMapEditorNewRecordNamesAreVisibleBeforeCommit(t *testing.T) {
	for _, key := range []string{"structure", "unit", "sack"} {
		t.Run(key, func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			host := &namedRecordHost{e.doc.Edits.(*modesFull)}
			host.self, e.doc.Edits = host, host
			_ = e.Key(key)
			modesAct(e, "ui:click-cell")
			_ = e.Key("down")
			if title := modesTitle(e); !strings.Contains(title, "First named record / Up, Down") || len(calls.lines) != 0 {
				t.Fatal("the name is not visible before adding the record", title, calls.lines)
			}
			_ = e.Key("up")
			if title := modesTitle(e); !strings.Contains(title, "Second named record / Up, Down") {
				t.Fatal("the named list did not wrap", title)
			}
			if err := e.Key("enter"); err != nil || len(calls.lines) != 1 || !strings.HasSuffix(calls.lines[0], " 20") {
				t.Fatal("the choice did not reach the existing constructor", err, calls.lines)
			}
			_ = e.Key(key)
			modesAct(e, "ui:click-cell")
			e.TextInput("900")
			if err := e.Key("enter"); err != nil || len(calls.lines) != 2 || !strings.HasSuffix(calls.lines[1], " 900") {
				t.Fatal("manual numeric input lost its constructor seam", err, calls.lines)
			}
		})
	}
}
