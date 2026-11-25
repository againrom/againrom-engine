package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// TestVocabularyRulesReachTheirBodies guards that each operation vocabulary's
// rule values index a body in the game's tables, so no route reads a rule
// that has no body.
func TestVocabularyRulesReachTheirBodies(t *testing.T) {
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		v := scriptprofile.VocabularyOf(ops)
		kind := kindRuleBodies[v.Kinds]
		if kind.copyPage == nil || kind.parameterKind == nil {
			t.Errorf("ops %d: kind rule %d has no body", ops, v.Kinds)
		}
		if operationNameBodies[v.Naming] == nil {
			t.Errorf("ops %d: naming rule %d has no body", ops, v.Naming)
		}
	}
}

func TestInspectScriptShapesAndNotesFollowOpRows(t *testing.T) {
	m := &alm.Map{Width: 20, Height: 20}
	sc := alm.Script{
		Conditions: []alm.ScriptNode{
			{ID: 1, Opcode: 2, Type: [10]uint32{mapload.ScriptUnit, mapload.ScriptX, mapload.ScriptY, mapload.ScriptX, mapload.ScriptY}, Value: [10]uint32{1, 3, 4, 8, 6}},
			{ID: 2, Opcode: 3, Type: [10]uint32{mapload.ScriptUnit, mapload.ScriptX, mapload.ScriptY, mapload.ScriptInt}, Value: [10]uint32{1, 5, 5, 2}},
			{ID: 3, Opcode: mapload.ScriptConstant, Type: [10]uint32{mapload.ScriptInt}, Value: [10]uint32{1}},
		},
		Actions: []alm.ScriptNode{
			{ID: 1, Opcode: 2, Type: [10]uint32{mapload.ScriptInt}, Value: [10]uint32{1}},
		},
		Triggers: []alm.ScriptTrigger{{Left: []uint32{1, 2, 0}, Right: []uint32{3, 3, 0}, Cmp: []uint32{0, 0, 0}, Acts: []uint32{1, 0, 0, 0}}},
	}
	inspect := func(ops scriptprofile.Ops) *ui.InspectionDocument {
		d := &ui.InspectionDocument{}
		inspectScript(d, m, sc, scriptprofile.For(ops, false), func(s string) string { return s })
		return d
	}
	hasLine := func(r ui.InspectionRecord, sub string) bool {
		for _, l := range r.Lines {
			if strings.Contains(l.Text, sub) {
				return true
			}
		}
		return false
	}
	hasRef := func(r ui.InspectionRecord, sub string) bool {
		for _, ref := range r.References {
			if strings.Contains(ref.Label, sub) {
				return true
			}
		}
		return false
	}
	first := inspect(scriptprofile.OpsFirst)
	var firstShapes [3]int
	for _, a := range first.Script.Areas {
		firstShapes[a.Shape]++
	}
	if firstShapes[ui.AreaBox] != 1 || firstShapes[ui.AreaSquare] != 1 {
		t.Fatalf("first game shapes %v, want 1 box and 1 square", firstShapes)
	}
	boxRec, sqRec, actRec := first.Records[first.Script.Conditions[0].Record], first.Records[first.Script.Conditions[1].Record], first.Records[first.Script.Actions[0].Record]
	if !hasLine(boxRec, "Box endpoints authored; edge rules unknown") {
		t.Errorf("first game check 2 missing box edge note: %+v", boxRec.Lines)
	}
	if !hasRef(sqRec, "Distance square: 5,5 radius 2") {
		t.Errorf("first game check 3 missing distance square ref: %+v", sqRec.References)
	}
	if hasLine(actRec, "Box endpoints authored; edge rules unknown") {
		t.Errorf("action 2 drew box edge note: %+v", actRec.Lines)
	}

	second := inspect(scriptprofile.OpsSecond)
	for _, a := range second.Script.Areas {
		if a.Shape != ui.AreaCell {
			t.Errorf("second game drew non-cell shape %+v", a)
		}
	}
	if len(second.Script.Areas) != 3 {
		t.Errorf("second game cell areas %d, want 3", len(second.Script.Areas))
	}
	if hasLine(second.Records[second.Script.Conditions[0].Record], "Box endpoints authored") {
		t.Errorf("second game check 2 emitted box edge note")
	}
	if hasRef(second.Records[second.Script.Conditions[1].Record], "Distance square:") {
		t.Errorf("second game check 3 emitted distance square ref")
	}
}
