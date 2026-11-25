package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/scriptprofile"
)

// On a second-game map an operation shows by the catalog's word where a claim
// reads it and as its raw id, "not decoded", where none does; the first
// game's tables are not consulted.
func TestSecondGameOperationNames(t *testing.T) {
	e := newMapEditorFixture(t)
	e.doc.Script = &InspectionScriptModel{Limits: scriptprofile.Second, Catalog: InspectionScriptCatalog{
		Checks:  []InspectionScriptOperation{{Opcode: 19, Named: true}},
		Actions: []InspectionScriptOperation{{Opcode: 65538, Named: true}},
	}}
	if p := e.doc.Script.Profile(); p.Registers != 1000 || p.Ops != scriptprofile.OpsSecond {
		t.Fatalf("profile %+v", p)
	}
	if got, want := e.opName(true, 19), e.word("editor.second.check.19"); got != want {
		t.Errorf("check 19 = %q, want %q", got, want)
	}
	if got := e.opName(false, 65538); !strings.Contains(got, e.word("editor.second.action.65538")) {
		t.Errorf("action 65538 = %q", got)
	}
	// Check 1 is named in the first game and unread in the second.
	if got, want := e.opName(true, 1), fmt.Sprintf(e.word("editor.op.undecoded"), 1); got != want {
		t.Errorf("check 1 = %q, want %q", got, want)
	}
	// Action 6 with a constant 3 names the second game's arm 3 in its words
	// and the first game's group command 3 in the command panel's.
	types, values := [10]uint32{scriptConst}, [10]uint32{3}
	second := e.nodeParams(false, 6, types, values)
	e.doc.Script = &InspectionScriptModel{Limits: scriptprofile.First}
	first := e.nodeParams(false, 6, types, values)
	if len(second) != 1 || !strings.HasSuffix(second[0], e.word("editor.second.command.3")) || len(first) != 1 || strings.HasSuffix(first[0], " 3") || first[0] == second[0] {
		t.Errorf("action 6 slot 0: second %q, first %q", second, first)
	}
	if p := e.doc.Script.Profile(); p != scriptprofile.First {
		t.Errorf("default profile %+v", p)
	}
}

// On a second-game map the label, the operation and the parameter fields of
// a decoded operation are editable, a type-9 slot among them; of an
// operation no claim decodes only the label is (R2-ENGINE-1681).
func TestSecondGameNodeFieldsEditOnlyDecodedParameters(t *testing.T) {
	e := newMapEditorFixture(t)
	e.doc.Script = &InspectionScriptModel{Limits: scriptprofile.Second, Catalog: InspectionScriptCatalog{
		Actions: []InspectionScriptOperation{{Opcode: 3, Named: true}},
	}}
	fields := func() []InspectionField {
		return []InspectionField{{Key: "name"}, {Key: "opcode"}, {Key: "value0"}, {Key: "cell1"}}
	}
	for _, c := range []struct {
		opcode uint32
		types  [10]uint32
		want   []bool
	}{{3, [10]uint32{1, 5, 6}, []bool{false, false, false, false}}, {7, [10]uint32{1, 5, 6}, []bool{false, true, true, true}},
		// A type-9 slot names a type-4 object of the map: it edits.
		{3, [10]uint32{9, 5, 6}, []bool{false, false, false, false}}} {
		v := scriptView{node: &InspectionScriptNode{Opcode: c.opcode, Types: c.types}}
		e.obj.types = c.types
		for i, f := range e.secondNodeFields(v, fields()) {
			if f.ReadOnly != c.want[i] {
				t.Errorf("operation %d field %s: read-only %v, want %v", c.opcode, f.Key, f.ReadOnly, c.want[i])
			}
		}
	}
}

// On a second-game map a node's operation list holds only the operations
// whose slots the table states, and the node's own: check 27 and check
// 65538 are not offered to a check 21, and a check 27 keeps its own.
func TestSecondGameOperationListHoldsOnlyStatedSignatures(t *testing.T) {
	e := newMapEditorFixture(t)
	e.doc.Script = &InspectionScriptModel{Limits: scriptprofile.Second, Catalog: secondCatalog()}
	choices := func(opcode uint32) []int64 {
		fields := []InspectionField{{Key: "opcode", Choices: []InspectionChoice{{Value: 2}, {Value: 21}, {Value: 27}, {Value: 0x10002}}}}
		var out []int64
		for _, c := range e.secondNodeFields(scriptView{condition: true, node: &InspectionScriptNode{Opcode: opcode}}, fields)[0].Choices {
			out = append(out, c.Value)
		}
		return out
	}
	if got := choices(21); !slices.Equal(got, []int64{2, 21}) {
		t.Errorf("a check 21 is offered %v, want [2 21]", got)
	}
	if got := choices(27); !slices.Equal(got, []int64{2, 21, 27}) {
		t.Errorf("a check 27 is offered %v, want [2 21 27]", got)
	}
}

// On a second-game map the trigger window changes a decoded action's
// operation: the list holds the second game's rows, the slots become action
// 24's signature (R2-ENGINE-1742) and Apply sends the slot types, the values
// and the operation last.
func TestSecondGameNodeOperationChangeRetypesToTheSignature(t *testing.T) {
	e, _ := wizardChoiceEditor(t)
	e.Layout(1920, 1080)
	e.doc.Script.Limits = scriptprofile.Second
	e.doc.Script.Catalog = secondCatalog()
	e.Select(4, false) // action 11, operation 4
	e.loadObject()
	v, ok := e.scriptPopup()
	if !ok || v.node == nil || v.node.ID != 11 {
		t.Fatal("no action popup")
	}
	field := func(key string) int {
		for i, f := range e.obj.fields {
			if f.Key == key {
				return i
			}
		}
		return -1
	}
	op := field("opcode")
	if op < 0 || e.obj.fields[op].ReadOnly || !slices.ContainsFunc(e.obj.fields[op].Choices, func(c InspectionChoice) bool { return c.Value == 24 }) {
		t.Fatalf("the operation field %+v", e.obj.fields)
	}
	e.obj.buf[op] = "24"
	e.retypeNode()
	if e.obj.types != [10]uint32{5, 6, 4, 1, 1} {
		t.Fatalf("action 24's slots %v", e.obj.types)
	}
	for key, text := range map[string]string{"cell0": "3,4", "value2": "10001", "value3": "6", "value4": "100"} {
		f := field(key)
		if f < 0 || e.obj.fields[f].ReadOnly {
			t.Fatalf("field %s is not edited: %+v", key, e.obj.fields)
		}
		e.obj.buf[f] = text
	}
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	want := []InspectionFieldValue{{Key: "type0", Value: 5}, {Key: "type1", Value: 6}, {Key: "type2", Value: 4}, {Key: "type3", Value: 1}, {Key: "type4", Value: 1},
		{Key: "value0", Value: 3}, {Key: "value1", Value: 4}, {Key: "value2", Value: 10001}, {Key: "value3", Value: 6}, {Key: "value4", Value: 100},
		{Key: "opcode", Value: 24}}
	if !slices.Equal(got, want) {
		t.Fatalf("Apply sends %+v, want %+v", got, want)
	}
}
