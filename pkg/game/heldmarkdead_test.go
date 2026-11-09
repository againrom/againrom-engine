package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestHeldMarkIsNotDrawnOverADeadUnit: a unit killed while its held mark is
// live keeps the attached effect but no longer draws the mark, in either game.
// The living unit's picture is the control.
func TestHeldMarkIsNotDrawnOverADeadUnit(t *testing.T) {
	cases := map[string]*mapWorld{
		"second game protection": secondHeldMarkWorld(t, 4, sim.EffectProtectionFire),
		"second game bless":      secondHeldMarkWorld(t, 20, sim.EffectBless),
		"first game":             nil,
	}
	w := emWorld(t, 5, protectionFireRule())
	emCast(t, w, 5)
	pictures := make([]int, 0x51)
	for i := range pictures {
		pictures[i] = i
	}
	cases["first game"] = &mapWorld{world: w, projectiles: emSheets(pictures...)}

	for name, mw := range cases {
		mw.advanceEffectMarks()
		if len(mw.markDraws(2, 1)) == 0 {
			t.Fatalf("%s: a living unit draws no held mark", name)
		}
		if err := mw.world.HeadlessKill(2); err != nil {
			t.Fatalf("%s: kill: %v", name, err)
		}
		sim.Step(mw.world, nil)
		mw.advanceEffectMarks()
		if mw.world.EntityAlive(2) {
			t.Fatalf("%s: the unit is still alive", name)
		}
		if len(mw.world.ActiveEffects()) == 0 {
			t.Fatalf("%s: the attached effect ended with the unit; the test needs it to outlive", name)
		}
		if got := mw.markDraws(2, 1); len(got) != 0 {
			t.Errorf("%s: a dead unit draws %d held marks", name, len(got))
		}
	}
}
