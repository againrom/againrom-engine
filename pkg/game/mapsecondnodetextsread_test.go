package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// A second-game map's nodes read their ten text fields beside the name,
// each under its own key and the word that says no game reader is found
// (R2-ENGINE-1681).
func TestSecondMapNodeTextFieldsRead(t *testing.T) {
	b, _, _ := secondTextedNodes(t)
	x := secondTestInspector(t)
	d, err := x.inspect(b, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []struct {
		node ui.InspectionScriptNode
		name string
	}{{d.Script.Conditions[0], "check"}, {d.Script.Actions[0], "action"}} {
		if len(n.node.Texts) != alm.ScriptParams {
			t.Fatalf("%s node texts %+v", n.name, n.node.Texts)
		}
		for i, f := range n.node.Texts {
			want := ui.InspectionField{Key: scriptprofile.TextKey(i), Label: fmt.Sprintf(x.word("editor.node.text"), i+1), Text: true, TextValue: fmt.Sprintf("%s %d", n.name, i+1)}
			if f.Key != want.Key || f.Label != want.Label || !f.Text || f.TextValue != want.TextValue {
				t.Fatalf("%s text %d reads %+v, want %+v", n.name, i, f, want)
			}
		}
	}
}
