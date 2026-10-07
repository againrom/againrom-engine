package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

// swingSoundUnit is swingUnit's own shape (swing_test.go) with a much longer
// charge: 20 ticks of AttackCharging rather than 8, so a small AttackDelay —
// this file uses 5 — falls well inside the charging window regardless of the
// exact tick advanceAttack (pkg/sim/combat.go) transitions out of it on, and
// well clear of AttackRelaxing's own jitter, which is drawn only at the
// charge's own end and cannot reach backward into it.
func swingSoundUnit(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Class: 1, HP: 200, MaxHP: 200,
		AttackCharge: 20, AttackRelax: 8}
}

// swingSoundPlay is one recorded call to the swing's play callback.
type swingSoundPlay struct {
	slot int
	cell image.Point
}

// TestSwingSoundFiresOncePerRun is AC-14: a swing fires on the tick the
// attack-run counter reaches the class's attack delay, and on no other tick
// of that run.
func TestSwingSoundFiresOncePerRun(t *testing.T) {
	const (
		swingSoundSlot  = 77
		swingSoundDelay = 5
	)
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))

	var plays []swingSoundPlay
	mw.setSwingSound(
		map[int32]UnitSound{1: {Slots: []int32{swingSoundSlot, 0, 0}, AttackDelay: swingSoundDelay}},
		func(slot int, owner uint32, cell image.Point) { plays = append(plays, swingSoundPlay{slot, cell}) },
	)

	mw.strike(1, 2)
	// 25 ticks is comfortably inside the first run's charge (20) and short of
	// even the SHORTEST possible relax-plus-ready gap back to a second
	// charging start (AttackRelax 8, jitter >= 0), so mw.swing's own
	// strictly-increasing count (advanceSwings' header comment) cannot reach
	// AttackDelay a second time inside this loop — the loop bound is what
	// makes "exactly one" a property of the run and not of how far this test
	// happens to drive it.
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}

	if len(plays) != 1 {
		t.Fatalf("%d emission(s), want exactly 1: %+v", len(plays), plays)
	}
	if plays[0].slot != swingSoundSlot {
		t.Errorf("slot = %d, want the class's own slot 0, %d", plays[0].slot, swingSoundSlot)
	}
	if want := (image.Point{X: 4, Y: 4}); plays[0].cell != want {
		t.Errorf("cell = %v, want the attacker's own cell %v", plays[0].cell, want)
	}
}

// TestSwingSoundSilentOnAZeroSlot is spec Terms' "a zero element is silence
// and not an error": AC-14's own class emits nothing when slot 0 of its
// Sound array is 0, even though the tick the counter reaches its delay still
// arrives and is still tested.
func TestSwingSoundSilentOnAZeroSlot(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))

	var plays []swingSoundPlay
	mw.setSwingSound(
		map[int32]UnitSound{1: {Slots: []int32{0, 0, 0}, AttackDelay: 5}},
		func(slot int, owner uint32, cell image.Point) { plays = append(plays, swingSoundPlay{slot, cell}) },
	)

	mw.strike(1, 2)
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}
	if len(plays) != 0 {
		t.Errorf("%d emission(s) from a class whose swing slot is 0, want 0: %+v", len(plays), plays)
	}
}

// THE OWNER'S REPORT, and it is a count of zero rather than "fewer": «он
// проигрывается, но сильно раньше чем стартовала битва — враги еще идут, а
// свинги уже слышны». An attacker that has acquired a victim it cannot reach
// runs a full attack cycle — pkg/sim advances one for every alive entity with
// a target and applies no distance test at all — and every charge tick of it
// used to be voiced. It is now voiced on none of them.
//
// THE CYCLE IS ASSERTED TO STILL RUN, in the same test and by value: the run
// counter must pass the class's own AttackDelay, so this witnesses a
// NARROWING OF THE VOICING and not an attacker that stopped charging. If the
// cycle itself were what changed, the counter would not get there and the
// second assertion would redden instead of the first.
//
// TO CONFIRM IT WITNESSES THE FIX, delete `&& swingInReach(ents, e)` from
// advanceSwings (world.go): this test reddens at "1 emission(s) ... want 0"
// while the in-reach test above stays green, which is the pair — the far
// swing was heard and the near one was always right.
func TestSwingSoundIsSilentWhileTheVictimIsOutOfReach(t *testing.T) {
	const (
		swingSoundSlot  = 77
		swingSoundDelay = 5
	)
	// Eight cells apart on one row, against a reach the world normalises to
	// 1 — far enough that no footprint term in strikeDistance could close it.
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 12, 4))

	var plays []swingSoundPlay
	mw.setSwingSound(
		map[int32]UnitSound{1: {Slots: []int32{swingSoundSlot, 0, 0}, AttackDelay: swingSoundDelay}},
		func(slot int, owner uint32, cell image.Point) { plays = append(plays, swingSoundPlay{slot, cell}) },
	)

	mw.strike(1, 2)
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}

	if len(plays) != 0 {
		t.Errorf("%d emission(s) from an attacker 8 cells from its victim, want 0: %+v", len(plays), plays)
	}
	if mw.swing[1] <= swingSoundDelay {
		t.Errorf("the attack run counter is %d, want it past the class's delay %d — "+
			"the cycle must still run out of reach; only the SOUND is narrowed",
			mw.swing[1], swingSoundDelay)
	}
}

// The far case's own control, so the pair discriminates distance and nothing
// else: the same fixture, the same delay, the same 25 ticks, the victim
// adjacent — exactly ONE emission. Without this a swingInReach that answered
// false for everything would pass the test above.
func TestSwingSoundStillFiresWithTheVictimAdjacent(t *testing.T) {
	const swingSoundDelay = 5
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))

	var plays []swingSoundPlay
	mw.setSwingSound(
		map[int32]UnitSound{1: {Slots: []int32{77, 0, 0}, AttackDelay: swingSoundDelay}},
		func(slot int, owner uint32, cell image.Point) { plays = append(plays, swingSoundPlay{slot, cell}) },
	)

	mw.strike(1, 2)
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}
	if len(plays) != 1 {
		t.Errorf("%d emission(s) from an attacker standing next to its victim, want exactly 1: %+v",
			len(plays), plays)
	}
}

// A victim the world no longer holds is the conservative arm of the same
// rule, asserted rather than assumed: swingInReach answers false, so nothing
// is voiced for an attack target that has left the world.
func TestSwingSoundIsSilentForAVictimTheWorldNoLongerHolds(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))

	var plays []swingSoundPlay
	mw.setSwingSound(
		map[int32]UnitSound{1: {Slots: []int32{77, 0, 0}, AttackDelay: 5}},
		func(slot int, owner uint32, cell image.Point) { plays = append(plays, swingSoundPlay{slot, cell}) },
	)

	// An id no entity carries: sim's own KindAttack is total over an absent
	// target, so the attacker holds the order and charges at nothing.
	mw.strike(1, 99)
	for tick := 0; tick < 25; tick++ {
		mw.tick()
	}
	if len(plays) != 0 {
		t.Errorf("%d emission(s) for an attack target the world does not hold, want 0: %+v",
			len(plays), plays)
	}
}

// TestSwingSoundNilCallbackCostsNothing is the other half of "the tier stays
// testable with no viewer" (plan T3): every hand-built mapWorld in this
// package's OTHER test files never calls setSwingSound at all, and this
// asserts the state such a world is actually left in — advanceSwings runs,
// the counter still counts, and nothing panics or fires.
func TestSwingSoundNilCallbackCostsNothing(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), swingSoundUnit(1, 4, 4), swingSoundUnit(2, 5, 4))
	mw.strike(1, 2)
	for tick := 0; tick < 10; tick++ {
		mw.tick()
	}
	if mw.swing[1] == 0 {
		t.Errorf("swing[1] = 0 after 10 ticks of a live attack run, want it still counting")
	}
}
