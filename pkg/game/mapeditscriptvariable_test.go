package game

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func packedVariableWindow(t *testing.T) (*ui.MapEditor, string) {
	t.Helper()
	e, path := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	for range 3 {
		scriptWindowAdd(t, e, "Condition")
	}
	scriptWindowSet(t, e, "Condition", 0, "value0", "99")
	scriptWindowSet(t, e, "Condition", 0, "opcode", "19")
	scriptWindowSet(t, e, "Condition", 2, "value0", "7")
	scriptWindowAdd(t, e, "Action")
	for _, field := range []struct{ key, value string }{
		{"value0", "99"}, {"opcode", "3"}, {"type1", "1"}, {"value1", "98"},
		{"type2", "1"}, {"value2", "7"}, {"type0", "0"},
	} {
		scriptWindowSet(t, e, "Action", 0, field.key, field.value)
	}
	scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", 1, "opcode", "4")
	for range 2 {
		scriptWindowAdd(t, e, "Trigger")
	}
	for _, field := range []struct {
		index      int
		key, value string
	}{{0, "left0", "2"}, {0, "right0", "2"}, {0, "action0", "1"}, {1, "left0", "1"}, {1, "right0", "3"}, {1, "action0", "2"}} {
		scriptWindowSet(t, e, "Trigger", field.index, field.key, field.value)
	}
	return e, path
}

func packedVariableAttempt(t *testing.T, e *ui.MapEditor, value int64) error {
	t.Helper()
	if !e.Select(recordOf(e.Document(), "Action", 0), false) {
		t.Fatal("packed action is missing")
	}
	for i, field := range e.Document().Edits.Fields("Action", 0) {
		if field.Key != "value1" {
			continue
		}
		if err := e.Key("field"); err != nil {
			t.Fatal(err)
		}
		for range i {
			_ = e.Key("tab")
		}
		_ = e.Key("clear")
		e.TextInput(strconv.FormatInt(value, 10))
		err := e.Key("enter")
		if err != nil {
			_ = e.Key("escape")
		}
		return err
	}
	t.Fatal("packed target field is missing")
	return nil
}

func TestScriptPackedVariableRefusesCollisionWithoutChangingHistory(t *testing.T) {
	for _, leading := range []uint32{0, 88} {
		t.Run(strconv.FormatUint(uint64(leading), 10), func(t *testing.T) {
			var e *ui.MapEditor
			var path string
			if leading == 0 {
				e, path = packedVariableWindow(t)
			} else {
				actions := []scriptNode{
					{label: "Packed variable", opcode: 3, id: 1, values: []uint32{99, 98, 7}, tags: []uint32{leading, 1, 1}},
					{label: "Win", opcode: 4, id: 2},
				}
				conditions := []scriptNode{
					{label: "Read variable", opcode: 19, id: 1, values: []uint32{99}, tags: []uint32{1}},
					{label: "Zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{1}},
					{label: "Seven", opcode: 0x10002, id: 3, values: []uint32{7}, tags: []uint32{1}},
				}
				triggers := []goalTrigger{{name: "Set", left: 2, right: 2, actions: []uint32{1}}, {name: "Win", left: 1, right: 3, actions: []uint32{2}}}
				e, path = openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: goalScript(actions, conditions, triggers)}))
			}
			authored := bytes.Clone(e.Document().SourceBytes)
			if _, err := e.Document().Edits.SaveAs(path + ".saved.alm"); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path + ".saved.alm")
			if err != nil || !bytes.Equal(authored, saved) {
				t.Fatal("packed parameter bytes changed on save", err)
			}
			e, _ = openGoalMap(t, saved)
			m, _ := alm.Open(saved)
			sc, _ := m.Script()
			if sc.Actions[0].Type[0] != leading || sc.Actions[0].Value[0] != 99 || sc.Actions[0].Value[1] != 98 {
				t.Fatal("inactive/unknown slot was normalized", sc.Actions[0])
			}
			scriptWindowSet(t, e, "Condition", 2, "value0", "8")
			changed := bytes.Clone(e.Document().SourceBytes)
			if err := e.Key("undo"); err != nil || !bytes.Equal(saved, e.Document().SourceBytes) || e.Document().Edits.CanUndo() || !e.Document().Edits.CanRedo() {
				t.Fatal("redo control is not armed", err)
			}
			for _, route := range []string{"window", "public API"} {
				var err error
				if route == "window" {
					err = packedVariableAttempt(t, e, 0)
				} else {
					_, err = e.Document().Edits.SetField("Action", 0, "value1", 0)
				}
				if err == nil || !strings.Contains(err.Error(), "variable register 0 is overwritten by condition 1") {
					t.Fatalf("%s accepted a packed write to a check register: %v", route, err)
				}
				model := e.Document().Edits.(*mapInspectionEdits).model
				if !bytes.Equal(saved, e.Document().SourceBytes) || !bytes.Equal(saved, model.Bytes()) || e.Dirty() || e.Document().Edits.CanUndo() || !e.Document().Edits.CanRedo() {
					t.Fatalf("%s refusal changed bytes, checkpoint or history", route)
				}
			}
			if err := e.Key("redo"); err != nil || !bytes.Equal(changed, e.Document().SourceBytes) {
				t.Fatal("refusal consumed or replaced the redo edit", err)
			}
			if err := e.Key("undo"); err != nil || !bytes.Equal(saved, e.Document().SourceBytes) || e.Dirty() {
				t.Fatal("refusal added a history entry", err)
			}
			w := scriptAuthorWorld(t, saved)
			for range 100 {
				sim.Step(w, nil)
			}
			if w.Outcome() != sim.OutcomeUndecided || w.ScriptRegister(98) != 7 || w.ScriptRegister(0) != 0 || w.ScriptRegister(99) != 0 {
				t.Fatalf("free-register control: outcome=%v r0=%d r98=%d r99=%d", w.Outcome(), w.ScriptRegister(0), w.ScriptRegister(98), w.ScriptRegister(99))
			}
		})
	}
}
