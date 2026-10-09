package sim

import "testing"

func TestTurnRetargetConsumesAtMostOneRateStepPerSubTick(t *testing.T) {
	b := Bounds{Width: 9, Height: 9}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{turnActor(1, 4, 4, 0, 16)})
	Step(w, []Command{MoveTo(1, CellPoint{X: 4, Y: 6})})
	before := w.entities[0].Facing
	beforeCounter := w.entities[0].TurnState.Counter
	Step(w, []Command{MoveTo(1, CellPoint{X: 6, Y: 4})})
	e := w.entities[0]
	if step := facingArc(before, e.Facing); step > 16 {
		t.Fatalf("retarget consumed facing arc %d, want at most rate 16", step)
	}
	if e.TurnState.Counter != beforeCounter+1 {
		t.Fatalf("retarget consumed counter %d, want %d", e.TurnState.Counter, beforeCounter+1)
	}
	if e.DesiredFacing != 64 || !e.HasTarget || e.TargetX != 6 || e.TargetY != 4 {
		t.Fatalf("retarget did not replace heading and destination: %+v", e)
	}
	if w.turnStepScope || len(w.turnSteps) != 0 {
		t.Fatal("turn scheduler retained a completed sub-tick")
	}
}

func TestRepeatedMoveCommandsContinueTurningAcrossSubTicks(t *testing.T) {
	b := Bounds{Width: 9, Height: 9}
	w := mustWorldGrid(t, 1, b, ModeCanonical, openGrid(b), []Entity{turnActor(1, 4, 4, 0, 16)})
	for call := 1; call <= 5; call++ {
		Step(w, []Command{MoveTo(1, CellPoint{X: 4, Y: 6})})
		if e := w.entities[0]; int(e.Facing) != call*16 || e.DesiredFacing != 128 || e.X != 4 || e.Y != 4 {
			t.Fatalf("call %d: repeated destination changed turn progress: %+v", call, e)
		}
	}
}

func TestRepeatedFacingRequestsRewriteTheTargetWithoutASecondStep(t *testing.T) {
	w := &World{entities: []Entity{turnActor(1, 4, 4, 0, 16)}}
	w.beginTurnSubTick()
	defer w.endTurnSubTick()
	if !w.turnToward(0, 0, 1) || !w.turnToward(0, 1, 0) {
		t.Fatal("facing request did not hold the actor")
	}
	e := w.entities[0]
	if e.Facing != 16 || e.DesiredFacing != 64 {
		t.Fatalf("two requests left current/desired facing %d/%d, want 16/64", e.Facing, e.DesiredFacing)
	}
	if e.TurnState.Counter != 1 {
		t.Fatalf("two requests consumed counter %d, want one", e.TurnState.Counter)
	}
	if !w.turnAlreadyStepped(e.ID) {
		t.Fatal("facing step was not recorded")
	}
}

func TestTurnHeadPassDoesNotRepeatAnEarlierProducerStep(t *testing.T) {
	w := &World{entities: []Entity{turnActor(1, 4, 4, 0, 16)}}
	w.beginTurnSubTick()
	defer w.endTurnSubTick()
	w.turnToward(0, 0, 1)
	w.advanceTurns()
	if got := w.entities[0].Facing; got != 16 {
		t.Fatalf("producer and head pass advanced facing to %d, want one rate step 16", got)
	}
}
