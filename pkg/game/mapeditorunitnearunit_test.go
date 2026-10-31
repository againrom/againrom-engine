package game

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The wizard's "Unit near unit" condition takes two units picked on the map
// and a radius, and writes check 6 (TRIG-DIST-014) compared with the radius
// by "is at most", as its other distance conditions do. The compiled
// trigger fires with the two units within the radius, not beyond it, and not
// when one of them is dead.
func TestTriggerWizardUnitNearUnitWritesTheDistanceCheck(t *testing.T) {
	e, _ := openGoalMap(t, triggerMap(t))
	e.Fit()
	wizNewTrigger(t, e, "Meet")
	wizPress(t, e, "", 0)
	trigChoose(t, e, wizField(t, e, "c0.item"), "Unit near unit")
	if got := trigLines(e); !strings.Contains(got, "A dead unit is never near.") || !strings.Contains(got, "Second unit") {
		t.Fatalf("the condition shows no hint or second unit:\n%s", got)
	}
	wizPress(t, e, "c0.p0", 0)
	clickCell(e, image.Pt(12, 14)) // unit 10
	wizPress(t, e, "c0.p1", 0)
	clickCell(e, image.Pt(15, 14)) // unit 11
	if !e.WizardShown() {
		t.Fatal("the unit pick did not return to the wizard")
	}
	wizType(t, e, "c0.p2", "4")
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a0.item"), "Win the mission")
	wizApply(t, e)

	sc := wizScript(t, e.Document().SourceBytes)
	tr := sc.Triggers[len(sc.Triggers)-1]
	left, right := wizNode(t, sc.Conditions, tr.Left[0]), wizNode(t, sc.Conditions, tr.Right[0])
	if tr.Name != "Meet" || tr.Cmp[0] != uint32(sim.ScriptCmpLE) {
		t.Fatalf("the trigger reads %+v", tr)
	}
	if left.Opcode != uint32(sim.ScriptCheckUnitDistance) || left.Type != [10]uint32{4, 4} || left.Value != [10]uint32{10, 11} {
		t.Fatalf("the distance check reads %+v", left)
	}
	if right.Opcode != mapload.ScriptConstant || right.Type != [10]uint32{1} || right.Value != [10]uint32{4} {
		t.Fatalf("the radius reads %+v", right)
	}
	reopened, _ := openGoalMap(t, e.Document().SourceBytes)
	reopened.Select(trigRecord(t, reopened, "Trigger", len(sc.Triggers)-1), true)
	if got := trigLines(reopened); !strings.Contains(got, "Unit near unit") || !strings.Contains(got, "Radius 4") {
		t.Fatalf("the reopened wizard does not read the condition:\n%s", got)
	}

	program, _, err := mapload.CompileScriptFrom(sc, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{10: 1, 11: 2}})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name    string
		x, hp   int32
		outcome sim.Outcome
	}{
		{"within the radius", 12, 100, sim.OutcomeWon},
		{"on the radius", 14, 100, sim.OutcomeWon},
		{"beyond the radius", 15, 100, sim.OutcomeUndecided},
		{"dead within the radius", 12, -1, sim.OutcomeUndecided},
	} {
		w, err := sim.NewScriptedWorld(11, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, make([]byte, 32*32), []sim.Entity{
			{ID: 1, Owner: sim.SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100},
			{ID: 2, Owner: 2, X: c.x, Y: 11, HP: c.hp, MaxHP: 100},
		}, program)
		if err != nil {
			t.Fatal(err)
		}
		if got := wizSteps(w, 3); got != c.outcome {
			t.Errorf("%s (second unit at %d,11): outcome %d, want %d", c.name, c.x, got, c.outcome)
		}
	}
}
