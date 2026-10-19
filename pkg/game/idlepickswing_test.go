package game

import (
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

// idlePickWorld holds unit 1 on an idle pick of unit 2 beside it: the attack
// target is held, the pursuit is idle and nothing walks or strikes.
func idlePickWorld(t *testing.T, idle bool) *mapWorld {
	t.Helper()
	pick := swingUnit(1, 4, 4)
	pick.HasAttackTarget, pick.AttackTarget, pick.AttackTargetKind, pick.PursuitIdle = true, 2, sim.AttackTargetUnit, idle
	return swingWorld(t, swingAnimDesc(), pick, swingUnit(2, 5, 4))
}

func inIdleBlock(frames []*terrain.StaticFrame, f *terrain.StaticFrame) bool {
	for i := swingIdleBase; i < swingIdleBase+8; i++ {
		if frames[i] == f {
			return true
		}
	}
	return false
}

// A unit that holds a victim on an idle pursuit (AI-542: the staff-less
// mage's pick stores order 0 and installs no strike) draws the standing
// frame on every tick, never the attack block.
func TestAnIdlePickDrawsTheStandingFrameNotASwing(t *testing.T) {
	mw := idlePickWorld(t, true)
	art := mw.units.Classes[1]
	for tick := 0; tick < swingRun+6; tick++ {
		mw.tick()
		d := swingDraw(t, mw, 1)
		if !inIdleBlock(art.Frames, d.Frame) {
			t.Fatalf("tick %d: the idle pick draws frame %p, outside the standing block", tick, d.Frame)
		}
	}
}

// Loss control: the same unit not idled strikes, and draws the swing's own
// frames first.
func TestAHeldVictimWithoutIdlePursuitStillDrawsTheSwing(t *testing.T) {
	mw := idlePickWorld(t, false)
	art := mw.units.Classes[1]
	mw.tick()
	d := swingDraw(t, mw, 1)
	if inIdleBlock(art.Frames, d.Frame) {
		t.Fatal("the unit that holds its victim without idling draws the standing frame, want the swing")
	}
}
