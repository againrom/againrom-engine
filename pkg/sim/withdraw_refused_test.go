package sim

import "testing"

// pocketedArcher is the archer of withdraw_cycle_test.go standing beside a
// walled pocket that holds its decoded flee cell. The cells within two of
// (17,20) are all closed, so the far search from (20,20) labels nothing near the
// flee cell and settles on the archer's own cell: it comes back with no route.
func pocketedArcher(t *testing.T, intruder Entity) *World {
	t.Helper()
	archer := withdrawalArcher(1, 2, 20, 20)
	grid := make([]byte, engBounds.Width*engBounds.Height)
	for y := int32(18); y <= 22; y++ {
		for x := int32(15); x <= 19; x++ {
			grid[y*engBounds.Width+x] = blockGround
		}
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: grid},
		[]Entity{archer, intruder}, nil, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

// blowsLanded counts the blows the archer's cycle has resolved on the intruder,
// read from the intruder's health: the archer's blow always lands for 5.
func blowsLanded(w *World, startHP int32) int32 {
	return (startHP - w.entities[1].HP) / 5
}

// TestWithdrawalWhoseFleeCellHasNoRouteTakesTheStandingPickAndShoots is the
// ordinary-play shape of a ranged creature beside a hostile it cannot back away
// from: the tail writes the decoded flee cell, the far search comes back with no
// route, and the original's order machine then clears the pending move and
// reacquires a victim within reach (AI-ROUTE-045, AI-335, MOVE-072). The archer
// holds its victim and its cycles run; before, it stood with no order for as
// long as the hostile stayed inside the block.
func TestWithdrawalWhoseFleeCellHasNoRouteTakesTheStandingPickAndShoots(t *testing.T) {
	intruder := withdrawalFighter(2, SelfSlot, 22, 20, 100)
	w := pocketedArcher(t, intruder)
	engRun(w, 1)

	got := w.entities[0]
	if got.X != 20 || got.Y != 20 {
		t.Fatalf("archer left its cell for a flee cell nothing leads to: (%d,%d)", got.X, got.Y)
	}
	if got.HasTarget {
		t.Fatalf("archer kept the destination (%d,%d) that no route serves", got.TargetX, got.TargetY)
	}
	if !got.HasAttackTarget || got.AttackTarget != intruder.ID {
		t.Fatalf("archer after the refused flee holds attack %t on %d, want its victim %d",
			got.HasAttackTarget, got.AttackTarget, intruder.ID)
	}

	for i := 0; i < 160; i++ {
		Step(w, nil)
		if e := w.entities[0]; e.X != 20 || e.Y != 20 {
			t.Fatalf("tick %d: archer moved to (%d,%d)", w.tick, e.X, e.Y)
		}
	}
	if blows := blowsLanded(w, 100); blows < 2 {
		t.Errorf("archer landed %d blow(s) in 160 ticks beside its hostile, want at least 2 cycles", blows)
	}
	if e := w.entities[0]; !e.HasAttackTarget {
		t.Error("archer ended the span with no attack order")
	}
}

// TestWithdrawalWhoseFleeCellHasNoRouteLeavesALoadedCycleAlone is the same
// creature caught mid-shot: the cycle it loaded resolves against its victim, no
// move is held behind it, and the next cycle follows without the archer ever
// standing without an order. The original's pending move executes only once the
// cycle ends and is refused there, so the archer does not walk (AI-ORDER-039,
// AI-RETREAT-272).
func TestWithdrawalWhoseFleeCellHasNoRouteLeavesALoadedCycleAlone(t *testing.T) {
	intruder := withdrawalFighter(2, SelfSlot, 24, 20, 100)
	w := pocketedArcher(t, intruder)
	engRun(w, 1)
	loaded := w.entities[0]
	if !loaded.HasAttackTarget || loaded.AttackPhase != AttackCharging || loaded.HasTarget {
		t.Fatalf("archer after the first decision = attack %t phase %d destination %t, want a loaded charge",
			loaded.HasAttackTarget, loaded.AttackPhase, loaded.HasTarget)
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 22, Y: 20})})
	stepToFullTick(w)

	held := w.entities[0]
	if held.HasTarget {
		t.Fatalf("a move to (%d,%d) that no route serves was held behind the loaded cycle", held.TargetX, held.TargetY)
	}
	if !held.HasAttackTarget || held.AttackPhase == AttackReady {
		t.Fatalf("archer at the tail = attack %t phase %d, want its loaded cycle kept",
			held.HasAttackTarget, held.AttackPhase)
	}
	for i := 0; i < 240; i++ {
		Step(w, nil)
		if e := w.entities[0]; e.X != 20 || e.Y != 20 {
			t.Fatalf("tick %d: archer moved to (%d,%d)", w.tick, e.X, e.Y)
		}
	}
	if blows := blowsLanded(w, 100); blows < 2 {
		t.Errorf("archer landed %d blow(s) in 240 ticks, want the loaded cycle and at least one more", blows)
	}
}

// TestWithdrawalCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut is
// the same archer against the west edge of the playable rectangle with its
// hostile two cells to the east. The decoded flee cell clamps onto the archer's
// own cell. A request equal to the cell of a centred mover returns before any
// search and raises no failure (MOVE-081, MOVE-083), so the refused-flee
// reacquisition does not run and the movement pass ends the arrived move.
// DIV-1639.
func TestWithdrawalCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut(t *testing.T) {
	archer := withdrawalArcher(1, 2, 8, 20)
	intruder := withdrawalFighter(2, SelfSlot, 10, 20, 100)
	w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}), archer, intruder)
	engRun(w, 1)

	got := w.entities[0]
	if got.X != 8 || got.Y != 20 || got.HasTarget || got.HasAttackTarget {
		t.Fatalf("archer = (%d,%d) move %t attack %t, want standing on (8,20) with neither", got.X, got.Y, got.HasTarget, got.HasAttackTarget)
	}
}

// TestFleeRequestForTheOwnCellIsRefusedOnlyBetweenCells separates the two
// seed requests: a centred mover's request returns before search and is not
// refused (MOVE-083); the same request from a mover between cells is searched
// with the seed as goal, ends with no node and is refused (MOVE-080,
// MOVE-082). A request to a different open cell is served. DIV-1639.
func TestFleeRequestForTheOwnCellIsRefusedOnlyBetweenCells(t *testing.T) {
	archer := withdrawalArcher(1, 2, 20, 20)
	w := engWorld(t, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}), archer,
		withdrawalFighter(2, SelfSlot, 30, 30, 100))
	if w.fleeRefusedAt(0, 20, 20) {
		t.Error("a centred mover's request for its own cell was refused")
	}
	if w.fleeRefusedAt(0, 17, 20) {
		t.Error("a request for an open cell was refused")
	}
	w.entities[0].Transit, w.entities[0].TransitTotal = 2, 4
	if !w.fleeRefusedAt(0, 20, 20) {
		t.Error("a mover between cells was not refused for its own cell")
	}
}
