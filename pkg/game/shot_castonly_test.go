package game

// The shot-marker hotfix (implementation/docs/hotfix/LEDGER.md): a book cast
// alone, with no attack order, must draw no shot marker, even where the
// world's entity id 0 sits far enough away to pass every other gate the
// marker checks. Entity id 0 is an ordinary placed unit (mapload places the
// map's first unit at id 0, pkg/mapload/fromalm.go) and not a reserved
// sentinel, and e.AttackTarget's zero value coincides with it whenever
// e.HasAttackTarget is false — the pair sim's own decode invariant
// (attackFault, pkg/sim/binary.go) requires.

import (
	"testing"
)

// TestABookCastWithNoAttackTargetDrawsNoShotMarker is the regression case.
// Entity 1 enters a cast run directly (mw.startCastRun, no strike, so
// HasAttackTarget stays false and AttackTarget stays its zero value) while
// entity 0 sits three cells away — within a reach-3 marker's range and past
// chebyshevDist's own two-or-more-cells threshold. Before the fix the block
// resolved entity 0 as e.AttackTarget's zero value and drew a marker aimed at
// it; the fix adds the missing e.HasAttackTarget gate.
func TestABookCastWithNoAttackTargetDrawsNoShotMarker(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(0, 7, 4, 1), shotUnit(1, 4, 4, 3))
	mw.startCastRun(1)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if d := swingDraw(t, mw, 1); d.Shot != nil {
		t.Errorf("a book caster with no attack target carries a Shot aimed at entity 0: %v", *d.Shot)
	}
}

// TestAGenuineAttackerStillDrawsAShotMarker is the fix's negative control:
// same geometry as the test above and the same candidate victim, entity 0.
// An attacker that actually holds entity 0 as its attack target must still
// carry a marker, so the added e.HasAttackTarget gate narrows the bug and
// not the feature (weaponBoltDraws, pkg/game/spellbolt.go, gates the sibling
// mark the same way).
func TestAGenuineAttackerStillDrawsAShotMarker(t *testing.T) {
	mw := swingWorld(t, swingAnimDesc(), shotUnit(0, 7, 4, 1), shotUnit(1, 4, 4, 3))
	mw.strike(1, 0)
	for i := 0; i < 4; i++ {
		mw.tick()
	}
	if d := swingDraw(t, mw, 1); d.Shot == nil {
		t.Fatalf("a genuine attacker of entity 0 carries no Shot")
	}
}
