package sim

import "testing"

// trapNode is a second-game Set trap node: Teleport at power 255 on (10,10),
// source (10,10), target (30,40).
func trapNode() ScriptInstant {
	return ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{secondCellEvent, 255, 10, 10, 10, 10, 30, 40}}
}

func trapWorld(t *testing.T, ents ...Entity) *World {
	t.Helper()
	s := secondGameScript(t, nil, []ScriptInstant{trapNode()}, nil)
	return scriptWorld(t, s, ents)
}

func TestSecondGameTrapStoresSourceAndTarget(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 5, Y: 5})
	w.setCellTail(trapNode())
	want := [cellTailLen]byte{secondCellEvent, 255, 10, 10, 30, 40}
	if got := w.CellTails(); len(got) != 1 || got[0].X != 10 || got[0].Y != 10 || got[0].Bytes != want {
		t.Fatalf("tails %v, want one at (10,10) with %v", got, want)
	}
}

func TestFirstGameTrapKeepsItsOwnBytes(t *testing.T) {
	w := scriptWorld(t, mustScript(t, nil, nil, nil), []Entity{{ID: 1, HP: 50, MaxHP: 50, X: 5, Y: 5}})
	w.setCellTail(trapNode())
	want := [cellTailLen]byte{secondCellEvent, 255, 0, 10, 0, 10}
	if got := w.CellTails(); len(got) != 1 || got[0].Bytes != want {
		t.Fatalf("tails %v, want %v", got, want)
	}
}

func TestSecondGameTrapMovesTheUnitWhoseStepEndsOnIt(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround})
	w.setCellTail(trapNode())
	w.requestCellCast(0, 10, 10)
	if casts := w.ScriptCasts(); len(casts) != 0 {
		t.Fatalf("entering the cell queued %v", casts)
	}
	w.stepCentred(0)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
		t.Fatalf("unit at (%d,%d), want the target (30,40)", e.X, e.Y)
	}
	if casts := w.ScriptCasts(); len(casts) != 0 {
		t.Fatalf("pending records left: %v", casts)
	}
}

func TestSecondGameTrapLeavesARefusedUnitInPlace(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 9, Y: 10, Domain: DomainGround})
	w.setCellTail(trapNode())
	w.requestCellCast(0, 10, 10)
	w.stepCentred(0)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 9 || e.Y != 10 {
		t.Fatalf("a unit refused the cell moved to (%d,%d)", e.X, e.Y)
	}
}

func TestSecondGameTrapRefusedTargetKeepsTheUnit(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround},
		Entity{ID: 2, HP: 50, MaxHP: 50, X: 30, Y: 40, Domain: DomainGround})
	w.setCellTail(trapNode())
	w.stepCentred(0)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 10 || e.Y != 10 {
		t.Fatalf("unit at (%d,%d), want it kept on a refused target", e.X, e.Y)
	}
}

func TestRepairCellEventsRewritesOnlyTheOlderForm(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 5, Y: 5})
	older := firstGameCellTailBytes(trapNode().Args)
	other := [cellTailLen]byte{9, 1, 2, 3, 4, 5}
	w.writeCellTail(tailKey(10, 10), older)
	w.writeCellTail(tailKey(20, 20), other)
	w.RepairCellEvents()
	w.RepairCellEvents()
	want := [cellTailLen]byte{secondCellEvent, 255, 10, 10, 30, 40}
	got := w.CellTails()
	if len(got) != 2 || got[0].Bytes != want || got[1].Bytes != other {
		t.Fatalf("tails after repair: %v", got)
	}
}

// The operands the editor writes for a teleport from entry (15, 121) to exit
// (13, 22) move the unit that reaches the entry cell to the exit.
func TestSecondGameEditorTeleportOperandsMoveTheUnit(t *testing.T) {
	node := ScriptInstant{Op: ScriptInstantCellTail, Args: [scriptParams]int32{secondCellEvent, 255, 15, 121, 15, 121, 13, 22}}
	s := secondGameScript(t, nil, []ScriptInstant{node}, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1, HP: 50, MaxHP: 50, X: 15, Y: 121, Domain: DomainGround}})
	w.setCellTail(node)
	w.requestCellCast(0, 15, 121)
	w.stepCentred(0)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 13 || e.Y != 22 {
		t.Fatalf("unit at (%d,%d), want the exit (13,22)", e.X, e.Y)
	}
}

// The second game queues a record whose target is (0,0) and relocates refuses
// it, so the ordinary cast of the event's spell follows (R2-ENGINE-1778 states
// no refusal of (0,0) at the centre).
func TestSecondGameTrapToCellZeroStillQueuesTheRecord(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround})
	w.writeCellTail(tailKey(10, 10), [cellTailLen]byte{secondCellEvent, 0, 10, 10, 0, 0})
	w.stepCentred(0)
	if casts := w.ScriptCasts(); len(casts) != 1 || casts[0].ToX|casts[0].ToY != 0 {
		t.Fatalf("a trap to (0,0) queued %v, want one record", casts)
	}
}
