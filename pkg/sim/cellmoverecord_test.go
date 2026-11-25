package sim

import "testing"

// A type-9 record of event 23 bound at load moves a unit whose step ends on
// its cell to the record's target, and the record stays for the next unit
// (R2-ENGINE-1778, R2-ENGINE-1777 Medium).
func TestSecondGameLoadedCellMoveRecordMovesEveryUnitOnItsCell(t *testing.T) {
	w := trapWorld(t, Entity{ID: 1, HP: 50, MaxHP: 50, X: 10, Y: 10, Domain: DomainGround},
		Entity{ID: 2, HP: 50, MaxHP: 50, X: 20, Y: 20, Domain: DomainGround})
	tail := CellTail{X: 10, Y: 10, Bytes: [cellTailLen]byte{secondCellEvent, 0, 10, 10, 30, 40}}
	if err := w.DeclareCellTails([]CellTail{tail}); err != nil {
		t.Fatal(err)
	}
	w.stepCentred(0)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(1); e.X != 30 || e.Y != 40 {
		t.Fatalf("unit 1 at (%d,%d), want the target (30,40)", e.X, e.Y)
	}
	if got := w.CellTails(); len(got) != 1 || got[0] != tail {
		t.Fatalf("tails after the move %v, want the record kept", got)
	}
	if !w.relocateActor(0, 50, 50, nil) || !w.relocateActor(1, 10, 10, nil) {
		t.Fatal("setup moves refused")
	}
	w.stepCentred(1)
	w.stepScriptCasts(nil)
	if e, _ := w.Entity(2); e.X != 30 || e.Y != 40 {
		t.Fatalf("unit 2 at (%d,%d), want the target (30,40)", e.X, e.Y)
	}
}
