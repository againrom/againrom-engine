package game

// The drawn stride of a mover that takes many ticks to cross one cell.
//
// The world here is built in this file rather than loaded, so the case does not
// depend on what any loader gives a placement: what is measured is that the
// seam holds a crossing's step for its whole length and hands the crossing over,
// and that is a statement about this package and the window tier alone.

import (
	"image"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// strideSpeed is the rate the mover in this file carries, and strideTicks what
// one straight cell then costs it. Both are written out: the rate law is
// pkg/sim's and unexported, and a test that re-derived it here would be
// asserting our own arithmetic twice. At this speed the law's multiplier and its
// substitute for an absent cost plane cancel, so the rate is the speed and 256
// over it rounds up to the count below.
const (
	strideSpeed = 16
	strideTicks = 16
)

// strideWorld is one rated mover on an open plane, ordered east, beside one that
// names no speed at all — so the two cadences stand in one world and the second
// says what an unrated mover still does.
func strideWorld(t *testing.T) (*mapWorld, *ui.Viewer) {
	t.Helper()
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)},
		sim.ModeCanonical, nil, []sim.Entity{
			{ID: 0, X: 21, Y: 23, Speed: strideSpeed},
			{ID: 1, X: 25, Y: 20},
		})
	if err != nil {
		t.Fatalf("building the stride world: %v", err)
	}
	sched := [][]sim.Command{{
		{Entity: 0, X: 31, Y: 23},
		{Entity: 1, X: 35, Y: 20},
	}}
	return newMapWorld(w, sched, nil, v), v
}

func TestACrossingIsDrawnWalkingForItsWholeLength(t *testing.T) {
	mw, _ := strideWorld(t)

	mw.tick()
	first := mw.entityDraws()[0]
	if want := (image.Point{X: 1, Y: 0}); first.Step != want {
		t.Fatalf("the crossing's first tick pushed a step of %v, want %v", first.Step, want)
	}
	if first.TransitSpan != strideTicks || first.Transit != strideTicks-1 {
		t.Fatalf("the crossing crossed as %d of %d, want %d of %d",
			first.Transit, first.TransitSpan, strideTicks-1, strideTicks)
	}

	for k := 1; k < strideTicks; k++ {
		mw.tick()
		d := mw.entityDraws()[0]
		if d.Step != first.Step {
			t.Fatalf("tick %d of the crossing pushed a step of %v, want the crossing's own %v",
				k, d.Step, first.Step)
		}
		if d.Cell != first.Cell {
			t.Fatalf("tick %d of the crossing moved the drawn cell to %v", k, d.Cell)
		}
		if d.TransitSpan != strideTicks || d.Transit != strideTicks-1-k {
			t.Fatalf("tick %d of the crossing crossed as %d of %d, want %d of %d",
				k, d.Transit, d.TransitSpan, strideTicks-1-k, strideTicks)
		}
	}

	// The unrated mover beside it: a cell a tick, so its crossing is nothing and
	// the seam says so.
	if d := mw.entityDraws()[1]; d.TransitSpan != 0 || d.Transit != 0 {
		t.Errorf("the unrated mover crossed as %d of %d, want 0 of 0", d.Transit, d.TransitSpan)
	}
}

// TestTheCrossingCrossesTheSeamWholeOverAWholeWalk carries the pair over more
// than one crossing, so what is measured is not one lucky tick: at every tick of
// the walk the pushed crossing is the simulation's own, and the pushed step is
// the one the crossing began with.
//
// What the window tier then DOES with that pair is measured where the
// arithmetic lives, in pkg/ui — this side owes only that the two numbers cross
// unchanged and that the step outlives the tick it was taken on.
func TestTheCrossingCrossesTheSeamWholeOverAWholeWalk(t *testing.T) {
	mw, _ := strideWorld(t)

	seenMidCrossing, cells := 0, map[image.Point]bool{}
	for k := 0; k < 4*strideTicks; k++ {
		mw.tick()
		d := mw.entityDraws()[0]
		e := mw.world.Entities()[0]
		if d.Transit != int(e.Transit) || d.TransitSpan != int(e.TransitTotal) {
			t.Fatalf("tick %d pushed a crossing of %d/%d for a world holding %d/%d",
				k, d.Transit, d.TransitSpan, e.Transit, e.TransitTotal)
		}
		if d.Step == (image.Point{}) {
			t.Fatalf("tick %d pushed no step, so the mover reads as standing still mid-walk", k)
		}
		if d.Transit > 0 {
			seenMidCrossing++
		}
		cells[d.Cell] = true
	}
	if seenMidCrossing == 0 {
		t.Error("no tick of the walk was pushed mid-crossing")
	}
	if len(cells) < 4 {
		t.Errorf("the walk covered %d cell(s) over four crossings, want at least 4", len(cells))
	}
}
