package game

import (
	"bytes"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

func TestScriptsListDeleteKeepsVariableRefusal(t *testing.T) {
	e, _ := openGoalMap(t, synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Type7Payload: make([]byte, 12)}))
	scriptWindowAdd(t, e, "Condition")
	scriptWindowAdd(t, e, "Action")
	scriptWindowSet(t, e, "Action", 0, "value0", "0")
	scriptWindowSet(t, e, "Action", 0, "opcode", "3")
	_ = e.Key("scripts")
	_ = e.Key("escape")
	r, ok := e.ListRowRect(2)
	trigClick(t, e, r, ok, "variable row")
	if e.Selected() >= -1 || len(e.PopupLines()) == 0 {
		t.Fatal("variable row did not open its read-only popup")
	}
	before := append([]byte(nil), e.Document().SourceBytes...)
	_ = e.Key("delete")
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("read-only variable popup changed the map")
	}
	e.Select(recordOf(e.Document(), "Condition", 0), false)
	if err := e.Key("delete"); err == nil {
		t.Fatal("Delete removed a variable whose register is in use")
	}
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatal("refusal changed the map")
	}
}

func TestScriptsListHostVariableRegisters(t *testing.T) {
	for _, opcode := range []uint32{3, 8, 19} {
		for _, tag := range []uint32{0, 1, 5, 6, 7} {
			n := alm.ScriptNode{ID: 10, Opcode: opcode, Type: [10]uint32{4, tag}, Value: [10]uint32{77, 67}}
			sc := alm.Script{Actions: []alm.ScriptNode{n}}
			if opcode == 19 {
				sc = alm.Script{Conditions: []alm.ScriptNode{n}}
			}
			d := &ui.InspectionDocument{}
			inspectScript(d, &alm.Map{}, sc, func(s string) string { return s })
			nodes := d.Script.Actions
			if opcode == 19 {
				nodes = d.Script.Conditions
			}
			want := uint32(67)
			if tag == 0 {
				want = 0
			}
			if !nodes[0].AddressesVariable || nodes[0].VariableRegister != want {
				t.Fatalf("opcode=%d tag=%d: %+v, want register %d", opcode, tag, nodes[0], want)
			}
		}
	}
}
