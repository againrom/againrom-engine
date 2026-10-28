package game

import (
	"image"
	"os"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// orderMap is triggerMap's four units and one trigger, "Mad", whose action
// orders group 21 to Swarm 2 to cell 30,25 with the slots of a shipped group
// order: the command first, the cell after it and the group in the last slot.
func orderMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{{label: "one", opcode: 0x10002, id: 1, values: []uint32{1}, tags: []uint32{1}}}
	actions := []scriptNode{{label: "mad", opcode: 6, id: 1,
		values: []uint32{5, 30, 25, 0, 0, 0, 0, 0, 0, 21}, tags: []uint32{7, 5, 6, 0, 0, 0, 0, 0, 0, 2}}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{
		{name: "Mad", left: 1, right: 1, actions: []uint32{1}},
	})}
	for i := 0; i < 4; i++ {
		o.Units = append(o.Units, synth.ALMUnit{X: uint32(12+3*i)*256 + 128, Y: 14*256 + 128, ClassID: 40, ClassSubID: 2})
	}
	return built(t, o, nil, nil).Bytes()
}

// A group order built in the wizard by choices and a click on the map, with
// no number typed, holds the shipped order's node slot for slot after Save As
// and a fresh load; the shipped order reads as the same wizard row.
func TestTriggerWizardOrdersAGroupToAPointAsTheShippedOrder(t *testing.T) {
	e, path := openGoalMap(t, orderMap(t))
	e.Fit()
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	shipped := trigLines(e)
	for _, want := range []string{"Action Order a group to a point", "Group Group 21", "Command Swarm 2", "Point 30,25"} {
		if !strings.Contains(shipped, want) {
			t.Fatalf("the shipped order reads as no wizard row (%q):\n%s", want, shipped)
		}
	}
	wizNewTrigger(t, e, "Mine")
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Order a group")
	trigChoose(t, e, wizField(t, e, "a0.p0"), "Group 21")
	trigChoose(t, e, wizField(t, e, "a0.p1"), "Swarm 2")
	wizPress(t, e, "a0.p2", 0) // On map
	clickCell(e, image.Pt(30, 25))
	wizApply(t, e)
	out := path + ".out.alm"
	if _, err := e.Document().Edits.SaveAs(out); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	sc := wizScript(t, saved)
	if len(sc.Triggers) != 2 || sc.Triggers[1].Name != "Mine" {
		t.Fatalf("the saved map's triggers %+v", sc.Triggers)
	}
	orig, made := wizNode(t, sc.Actions, 1), wizNode(t, sc.Actions, sc.Triggers[1].Acts[0])
	if made.ID == orig.ID || made.Opcode != orig.Opcode || made.Type != orig.Type || made.Value != orig.Value {
		t.Fatalf("the built order %+v, the shipped %+v", made, orig)
	}
	reopened, _ := openGoalMap(t, saved)
	reopened.Select(trigRecord(t, reopened, "Trigger", 1), true)
	got := trigLines(reopened)
	for _, want := range []string{"Action Order a group to a point", "Group Group 21", "Command Swarm 2", "Point 30,25"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the reloaded order lacks %q:\n%s", want, got)
		}
	}
}
