package game

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// gapTriggerMap holds one trigger whose first pair is empty yet carries a
// comparison word, whose second pair is used, and whose actions sit in
// slots 1 and 3.
func gapTriggerMap(t *testing.T) []byte {
	t.Helper()
	conditions := []scriptNode{
		{label: "dead", opcode: 5, id: 1, values: []uint32{10}, tags: []uint32{4}},
		{label: "zero", opcode: 0x10002, id: 2, values: []uint32{0}, tags: []uint32{7}},
	}
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}, {label: "lose", opcode: 5, id: 2}}
	tr := goalTrigger{name: "Gap", cmp: 3, actions: []uint32{0, 1, 0, 2},
		raw: map[int]uint32{0x88: 1, 0x8c: 2, 0xac: 5}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{tr})}
	return built(t, o, nil, nil).Bytes()
}

// Apply with no edit writes nothing: the empty slots and the slots the
// trigger's rows sit in stay as the map has them.
func TestTriggerWizardApplyWithoutEditKeepsTheBytes(t *testing.T) {
	e, _ := openGoalMap(t, gapTriggerMap(t))
	before := append([]byte(nil), e.Document().SourceBytes...)
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizApply(t, e)
	if !bytes.Equal(before, e.Document().SourceBytes) {
		t.Fatalf("Apply without an edit changed the map:\nbefore %+v\nafter  %+v", wizScript(t, before).Triggers[0], wizScript(t, e.Document().SourceBytes).Triggers[0])
	}
}

// An edited row keeps the slot it was read from; the slots no row reaches
// keep their words.
func TestTriggerWizardEditKeepsSlotPositions(t *testing.T) {
	e, _ := openGoalMap(t, gapTriggerMap(t))
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Lose the mission")
	wizApply(t, e)
	tr := wizScript(t, e.Document().SourceBytes).Triggers[0]
	if tr.Left[0] != 0 || tr.Right[0] != 0 || tr.Cmp[0] != 3 || tr.Left[1] != 1 || tr.Right[1] != 2 || tr.Cmp[1] != 5 {
		t.Fatalf("the pairs moved: %+v", tr)
	}
	if tr.Acts[0] != 0 || tr.Acts[1] == 0 || tr.Acts[2] != 0 || tr.Acts[3] != 2 {
		t.Fatalf("the actions moved: %v", tr.Acts)
	}
}

// A number the map cannot hold in 32 bits is refused with the reason, not
// written as another value.
func TestTriggerWizardRefusesANumberOutsideThirtyTwoBits(t *testing.T) {
	for _, typed := range []string{"4294967796", "2147483648", "-2147483649"} {
		e, _ := openGoalMap(t, triggerMap(t))
		wizNewTrigger(t, e, "Bounds")
		wizPress(t, e, "+action", 0)
		wizType(t, e, "a0.p1", typed)
		before := append([]byte(nil), e.Document().SourceBytes...)
		r, ok := e.PopupButtonRect(0)
		trigClick(t, e, r, ok, "Apply")
		if !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatalf("%s: the map changed", typed)
		}
		if !strings.Contains(trigLines(e), "-2147483648 and 2147483647") {
			t.Fatalf("%s: no reason shown:\n%s", typed, trigLines(e))
		}
	}
	e, _ := openGoalMap(t, triggerMap(t))
	wizNewTrigger(t, e, "Bounds")
	wizPress(t, e, "+action", 0)
	wizType(t, e, "a0.p1", "2147483647")
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	tr := sc.Triggers[len(sc.Triggers)-1]
	if got := wizNode(t, sc.Actions, tr.Acts[0]).Value[1]; got != 2147483647 {
		t.Fatalf("the largest amount is written as %d", got)
	}
}

// A zone on a zero coordinate would start the hero on the random fallback
// cell, not on the cell the tool shows; the tool refuses it, and a zone from
// 1 holds the hero on the cell shown.
func TestDropZoneRefusesACellTheGameWouldMove(t *testing.T) {
	for _, c := range []image.Point{{X: 0, Y: 20}, {X: 20, Y: 0}} {
		e, _ := openGoalMap(t, emptyScriptMap(t, 160))
		e.Fit()
		_ = e.Key("drop")
		before := append([]byte(nil), e.Document().SourceBytes...)
		clickCell(e, c)
		if len(e.Document().Drops) != 0 || !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatalf("a zone on %v was written: %v", c, e.Document().Drops)
		}
	}
	e, _ := openGoalMap(t, emptyScriptMap(t, 160))
	e.Fit()
	_ = e.Key("drop")
	clickCell(e, image.Pt(1, 20))
	if len(e.Document().Drops) != 1 {
		t.Fatalf("a zone on 1,20 was not written")
	}
	z, ok := e.Document().Edits.(ui.InspectionDropZones)
	if !ok {
		t.Fatal("no zone host")
	}
	if _, err := z.MoveDropZone(0, image.Pt(0, 5)); err == nil {
		t.Fatal("a zone moved onto a zero coordinate")
	}
	m := startEdited(t, e.Document().SourceBytes)
	if m.Start.Fallback || m.Start.Cells[0].X != 1 || m.Start.Cells[0].Y != 20 {
		t.Fatalf("the hero starts at %v (fallback %v), the tool shows 1,20", m.Start.Cells[0], m.Start.Fallback)
	}
}
