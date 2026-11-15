package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// variableNameMap holds a condition that reads register 99 and, at index 99,
// a preset node with the given label.
func variableNameMap(t *testing.T, label string) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "read", opcode: 19, id: 1, values: []uint32{99}, tags: []uint32{kInt}},
		{label: "zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{kInt}},
	}
	for i := 2; i < 99; i++ {
		conditions = append(conditions, scriptNode{label: "fill", opcode: 0x10002, id: uint32(i + 1), values: []uint32{0}, tags: []uint32{kInt}})
	}
	conditions = append(conditions, scriptNode{label: label, opcode: 0x10002, id: 100, values: []uint32{0}, tags: []uint32{kInt}})
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Read", left: 1, right: 2, actions: []uint32{1}},
	})}
	return built(t, o, nil, nil).Bytes()
}

// A variable whose preset node carries no label or the label the editor
// writes by default is named by its register; a label an author wrote names
// it. The register and the node bytes survive a save and a reopen. EN and RU.
func TestScriptVariableNameIgnoresDefaultPresetLabel(t *testing.T) {
	defaultLabel := mapload.ScriptOperationName(true, 0x10002)
	for _, c := range []struct {
		name, label string
		authored    bool
	}{{"default", defaultLabel, false}, {"empty", "", false}, {"author", "Gold", true}} {
		b := variableNameMap(t, c.label)
		before := wizScript(t, b)
		for _, l := range scriptAddLangs {
			e, x, w := scriptAddEditor(t, l, b)
			want := w("editor.scripts.variable", 99)
			if c.authored {
				want = x.inspectionUTF8("Gold")
			}
			found := false
			for _, tg := range e.Document().Script.Targets[ui.ScriptVariableParameter] {
				if tg.Value == 99 && tg.Record != -1 {
					found = true
					if tg.Name != want {
						t.Fatalf("%s %s: the variable choice is %q, want %q", l.name, c.name, tg.Name, want)
					}
				}
			}
			if !found {
				t.Fatalf("%s %s: no choice names register 99", l.name, c.name)
			}
			title := w("editor.scripts.variable", 99)
			if c.authored {
				title += "  Gold"
			}
			if !treeHas(e, title) {
				t.Fatalf("%s %s: the list lacks %q: %q", l.name, c.name, title, e.TreeLines())
			}
			path := filepath.Join(t.TempDir(), "saved.alm")
			if _, err := e.Document().Edits.SaveAs(path); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			after := wizScript(t, saved)
			if !bytes.Equal(saved, b) || !reflect.DeepEqual(after.Conditions, before.Conditions) {
				t.Fatalf("%s %s: the save changed the map", l.name, c.name)
			}
			if fmt.Sprint(after.Conditions[0].Value[0]) != "99" {
				t.Fatalf("%s %s: the condition reads %d", l.name, c.name, after.Conditions[0].Value[0])
			}
		}
	}
}
