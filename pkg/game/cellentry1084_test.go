package game

import (
	"image"
	"testing"

	"againrom/pkg/sim"
)

func TestCellEntry1084DirectLightningHasNoActorAnimationAndExpires(t *testing.T) {
	mw := spWorld(t)
	mw.observeScriptCasts([]sim.ScriptCastEvent{{Spell: 13, FromX: 3, FromY: 4, ToX: 7, ToY: 6}})
	records := flightRecords(mw)
	if len(records) != 1 || len(mw.castRun) != 0 {
		t.Fatal("temporary caster acquired an actor animation or built no record")
	}
	// The client arm's record: 5 segments from the source cell's centre,
	// started at actionphase -1, dir 0 (ANIM-144, SAV-1199).
	p := records[0]
	if p.ActionSegments != 4 || p.ActionPhase != 0 || p.Dir != 0 || p.X != 3*256+128 || p.Y != 4*256+128 ||
		p.ActionX != 7*256+128 || p.ActionY != 6*256+128 {
		t.Fatalf("direct-client record = %+v", p)
	}
	// Independent endpoints: no north-facing hand offset on a source cell. The
	// direct route's five calls draw phases 0,4,3,2,1 (MAGIC-281).
	for i, want := range []int{0, 4, 3, 2, 1} {
		b := mw.recordPaths(flightRecords(mw)[0], 0)[0]
		_, frame, points := mw.pathFigure(b)
		if frame != want || len(points) < 2 || !withinPixel(points[0], image.Pt(3*32+16, 4*32+16)) {
			t.Fatalf("direct call %d: frame %d points %v, want frame %d from the source cell centre", i, frame, points, want)
		}
		flightStep(mw)
	}
	if len(flightRecords(mw)) != 0 {
		t.Fatal("completed direct record stayed live")
	}
}
