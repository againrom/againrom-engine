package game

import (
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// TestCreateChoiceGlobalMarkerIsRefusedByWord pins that a command and a host
// edit given the global create marker refuse it with the table's word, not
// write it on as a number.
func TestCreateChoiceGlobalMarkerIsRefusedByWord(t *testing.T) {
	const word = "a new global is added on the Globals page"
	e := parameterEditor(t)
	_, err := e.Document().Edits.SetField("Action", 0, "value0", ui.NewGlobalChoice)
	if err == nil || !strings.Contains(err.Error(), word) {
		t.Fatalf("host edit gave %v, want the refusal %q", err, word)
	}
	tool, _ := editorMCP(t, e)
	r := tool("set_field", map[string]any{"kind": "Action", "index": 0, "key": "value0", "value": ui.NewGlobalChoice})
	if !r.IsError || len(r.Content) == 0 || !strings.Contains(r.Content[0].Text, word) {
		t.Fatalf("command gave %+v, want the refusal %q", r, word)
	}
}

// TestCreateChoiceRefusalOnUnrelatedField pins that a create marker given to a
// field that does not admit it is refused by the table's word, and that a
// unit's identity choice is not taken for a marker.
func TestCreateChoiceRefusalOnUnrelatedField(t *testing.T) {
	_, s, _ := nativeIdentityEditor(t)
	for _, marker := range []int64{ui.NewGlobalChoice, ui.NewMessageChoice} {
		word, _ := ui.NewChoiceRefusal(marker)
		if _, err := s.SetField("Unit", 0, "level", marker); err == nil || !strings.Contains(err.Error(), word) {
			t.Fatalf("unit level marker %d gave %v, want %q", marker, err, word)
		}
		if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{{Key: "level", Value: marker}}); err == nil || !strings.Contains(err.Error(), word) {
			t.Fatalf("unit level marker %d by SetFields gave %v, want %q", marker, err, word)
		}
	}
}
