package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestCellEntry1084DirectLightningHasNoActorAnimationAndExpires(t *testing.T) {
	mw := spWorld(t)
	before := mw.world.Hash()
	mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: 13, FromX: 3, FromY: 4, ToX: 7, ToY: 6}})
	if len(mw.bolts) != 1 || len(mw.castRun) != 0 || mw.world.Hash() != before {
		t.Fatal("temporary caster acquired an actor animation or mutated simulation state")
	}
	b := mw.bolts[0]
	if b.life != 5 || b.from != image.Pt(3, 4) || b.to != image.Pt(7, 6) || !b.centered {
		t.Fatalf("direct-client bolt = %+v", b)
	}
	// Independent endpoints: no north-facing hand offset on a source cell.
	points := boltPathFrom(image.Pt(3*ui.ShotScale, 4*ui.ShotScale), b.to, b.seed, 0)
	if len(points) < 2 || points[0] != image.Pt(3*ui.ShotScale, 4*ui.ShotScale) || points[len(points)-1] != image.Pt(7*ui.ShotScale, 6*ui.ShotScale) {
		t.Fatalf("direct bolt endpoints = %v", points)
	}
	for i := 0; i < 5; i++ {
		mw.advanceBolts()
	}
	if len(mw.bolts) != 0 || mw.world.Hash() != before {
		t.Fatal("completed direct bolt stayed live or changed hashed state")
	}
}
