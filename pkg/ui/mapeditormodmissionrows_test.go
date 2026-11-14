package ui

import (
	"fmt"
	"strings"
	"testing"
)

// The missions page has a row for every mission the host lists, however
// many, and its scroll reaches the last one.
func TestModWindowMissionsPageHasARowForEveryMission(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	ws := e.doc.Edits.(*modesWorkspace)
	var choices []InspectionChoice
	for n := 1; n <= 9; n++ {
		choices = append(choices, InspectionChoice{Value: int64(n), Label: fmt.Sprintf("row%d (%d)", n, 20+n)})
	}
	for i := range ws.fields {
		if ws.fields[i].Key == "source" {
			ws.fields[i].Choices, ws.fields[i].Value = choices, 7
		}
	}
	_ = e.Key("package")
	if e.modPage != modPageFields {
		t.Fatalf("W shows page %d", e.modPage)
	}
	s, _, _ := e.modSpec()
	got := 0
	for _, r := range s.Rows {
		if strings.HasPrefix(r.Label, "row") && len(r.Buttons) == 1 {
			got++
		}
	}
	if got != len(choices) {
		t.Fatalf("the page has %d mission rows of %d", got, len(choices))
	}
}
