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
