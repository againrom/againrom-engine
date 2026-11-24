package game

import (
	"testing"

	"againrom/pkg/sim"
)

// A decaying corpse that still carries poison draws no held mark: the poison
// keeps ticking on the body, the picture does not.
func TestPoisonMarkIsNotDrawnOverADecayingCorpse(t *testing.T) {
	mw := secondHeldMarkWorld(t, 6, sim.EffectHealth)
	mw.projectiles = emSheets(20, 24, 62, 64)
	mw.advanceEffectMarks()
	if len(mw.markDraws(2, 1)) == 0 {
		t.Fatal("a living poisoned unit draws no held mark")
	}
	if err := mw.world.HeadlessKill(2); err != nil {
		t.Fatalf("kill: %v", err)
	}
	sim.Step(mw.world, nil)
	mw.advanceEffectMarks()
	if mw.world.EntityAlive(2) {
		t.Fatal("the unit is still alive")
	}
	if len(mw.world.ActiveEffects()) == 0 {
		t.Fatal("the poison ended with the unit; the test needs it to outlive")
	}
	if got := mw.markDraws(2, 1); len(got) != 0 {
		t.Errorf("a decaying corpse draws %d held marks", len(got))
	}
}
