package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// shotUnit is swingUnit with an explicit reach, for the tests below that
// depend on it: a cycle long enough (AttackCharge 8) that a mid-cycle
// progress is unambiguous, and a reach the caller states rather than the
// zero swingUnit leaves for NewWorld to normalise to 1.
func shotUnit(id sim.EntityID, x, y int32, reach uint8) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: 1, HP: 200, MaxHP: 200,
		Reach: reach, AttackCharge: 8, AttackRelax: 8}
}

// TestARangedAttackerDrawsAnAdvancingShot is AC-6's positive half: a reach-3
// attacker mid-cycle against a target three cells away carries exactly one
// Shot, it lies strictly between the two cells on the segment the closed
// form gives, and a later tick carries it further along the same segment.
func TestARangedAttackerDrawsAnAdvancingShot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(1, 4, 4, 3), shotUnit(2, 7, 4, 1))
	mw.strike(1, 2)

	// Four ticks: the first enters the charging phase fresh (swing clock 0),
	// each after it advances the clock by one — swing_test.go's own witnessed
	// relationship between tick count and mw.swing — so the fourth tick
	// leaves the clock at 3, over an 8-tick charge: mid-cycle, not at either
	// end.
	for i := 0; i < 4; i++ {
		mw.tick()
	}

	d := swingDraw(t, mw, 1)
	if d.Shot == nil {
		t.Fatalf("a reach-3 attacker mid-cycle against a target 3 cells away carries no Shot")
	}

	const num, den = 3, 8
	fromX, toX := 4*ui.ShotScale, 7*ui.ShotScale
	wantX := fromX + (toX-fromX)*num/den
	wantY := 4 * ui.ShotScale
	if d.Shot.X != wantX || d.Shot.Y != wantY {
		t.Errorf("Shot = (%d,%d), want the closed form (%d,%d)", d.Shot.X, d.Shot.Y, wantX, wantY)
	}
	if !(d.Shot.X > fromX && d.Shot.X < toX) {
		t.Errorf("Shot.X = %d, want strictly between the attacker's %d and the victim's %d",
			d.Shot.X, fromX, toX)
	}

	// The victim carries no shot of its own — exactly one entity does.
	if v := swingDraw(t, mw, 2); v.Shot != nil {
		t.Errorf("the victim carries a Shot too: %v", *v.Shot)
	}

	// It advances: one more tick moves it further along the same segment.
	prevX := d.Shot.X
	mw.tick()
	d2 := swingDraw(t, mw, 1)
	if d2.Shot == nil {
		t.Fatalf("the shot vanished on the very next tick")
	}
	if d2.Shot.X <= prevX {
		t.Errorf("Shot.X = %d after another tick, want more than %d — the mark should advance",
			d2.Shot.X, prevX)
	}
}

// TestAReachOneAttackerDrawsNoShot is AC-6's first negative clause. A reach-1
// attacker can only ever fight from adjacency — inReach (pkg/sim, unexported)
// refuses anything farther and the approach walks it closer — so this is the
// one fixture a reach-1 attacker's own cycle can be witnessed in without
// conflating the reach gate with the attacker having drifted off its cell.
func TestAReachOneAttackerDrawsNoShot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(1, 4, 4, 1), shotUnit(2, 5, 4, 1))
	mw.strike(1, 2)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if d := swingDraw(t, mw, 1); d.Shot != nil {
		t.Errorf("a reach-1 attacker carries a Shot: %v", *d.Shot)
	}
}

// TestAnAdjacentAttackerDrawsNoShotAtAnyReach is AC-6's second negative
// clause, and the reach it is given (3, past the threshold the test above
// covers) is what isolates it from that one: an attacker that COULD carry a
// shot at range still carries none beside its own victim.
func TestAnAdjacentAttackerDrawsNoShotAtAnyReach(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(1, 4, 4, 3), shotUnit(2, 5, 4, 3))
	mw.strike(1, 2)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if d := swingDraw(t, mw, 1); d.Shot != nil {
		t.Errorf("an adjacent reach-3 attacker carries a Shot: %v", *d.Shot)
	}
}

// TestAnUnengagedReachAttackerDrawsNoShot is AC-6's third negative clause: an
// entity with no attack target — reach above 1 and all — carries no mark.
func TestAnUnengagedReachAttackerDrawsNoShot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(1, 4, 4, 3), shotUnit(2, 7, 4, 1))
	// No strike: entity 1 never holds an attack target.
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if d := swingDraw(t, mw, 1); d.Shot != nil {
		t.Errorf("an entity with no attack target carries a Shot: %v", *d.Shot)
	}
}

// TestADeadReachAttackerDrawsNoShot is AC-6's fourth negative clause: felling
// the attacker mid-cycle clears the mark on the very next push, exactly as
// swing_test.go's TestACorpseNeverSwings pins the frame it rides.
func TestADeadReachAttackerDrawsNoShot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(1, 4, 4, 3), shotUnit(2, 7, 4, 1))
	mw.strike(1, 2)
	mw.tick()
	mw.tick()
	if d := swingDraw(t, mw, 1); d.Shot == nil {
		t.Fatalf("fixture: a live reach-3 attacker mid-cycle at range 3 carries no Shot to begin with")
	}
	mw.affect(1, true) // fell the attacker mid-run
	mw.tick()          // the queue is applied by the advance, like any order
	if d := swingDraw(t, mw, 1); d.Shot != nil {
		t.Errorf("a dead attacker carries a Shot: %v", *d.Shot)
	}
}

func TestNoReachDrawsNoShot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingUnit(1, 4, 4), swingUnit(2, 5, 4))
	mw.strike(1, 2)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	for _, d := range mw.entityDraws() {
		if d.Shot != nil {
			t.Errorf("entity %d carries a Shot with no reach above 1 anywhere in the world: %v",
				d.ID, *d.Shot)
		}
	}
}
