package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
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
	// Independent endpoints: no north-facing hand offset on a source cell. The
	// direct route's five calls draw phases 0,4,3,2,1 (MAGIC-281).
	for i, want := range []int{0, 4, 3, 2, 1} {
		b := mw.bolts[0]
		_, frame, points := mw.pathFigure(b)
		if frame != want || len(points) < 2 || !withinPixel(points[0], image.Pt(3*32+16, 4*32+16)) {
			t.Fatalf("direct call %d: frame %d points %v, want frame %d from the source cell centre", i, frame, points, want)
		}
		mw.advanceBolts()
	}
	if len(mw.bolts) != 0 || mw.world.Hash() != before {
		t.Fatal("completed direct bolt stayed live or changed hashed state")
	}
}
