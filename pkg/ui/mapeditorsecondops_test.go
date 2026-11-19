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
