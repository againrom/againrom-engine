package ui

import (
	"fmt"
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
	// Action 6 with a constant 3 is a raw number in the second game, a
	// group command word in the first.
	types, values := [10]uint32{scriptConst}, [10]uint32{3}
	second := e.nodeParams(false, 6, types, values)
	e.doc.Script = &InspectionScriptModel{}
	first := e.nodeParams(false, 6, types, values)
	if len(second) != 1 || !strings.HasSuffix(second[0], " 3") || len(first) != 1 || strings.HasSuffix(first[0], " 3") {
		t.Errorf("action 6 slot 0: second %q, first %q", second, first)
	}
	if p := e.doc.Script.Profile(); p != scriptprofile.First {
		t.Errorf("default profile %+v", p)
	}
}

// On a second-game map only the label and the parameter fields of a decoded
// operation are editable; the operation is shown, so is a type-9 slot, and
// every field of an operation no claim decodes is shown.
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
	}{{3, [10]uint32{1, 5, 6}, []bool{false, true, false, false}}, {7, [10]uint32{1, 5, 6}, []bool{true, true, true, true}},
		// The record a type-9 slot names is unread: that slot alone stays shown.
		{3, [10]uint32{9, 5, 6}, []bool{false, true, true, false}}} {
		v := scriptView{node: &InspectionScriptNode{Opcode: c.opcode, Types: c.types}}
		e.obj.types = c.types
		for i, f := range e.secondNodeFields(v, fields()) {
			if f.ReadOnly != c.want[i] {
				t.Errorf("operation %d field %s: read-only %v, want %v", c.opcode, f.Key, f.ReadOnly, c.want[i])
			}
		}
	}
}
