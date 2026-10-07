package ui

// The displacement over a crossing that takes many ticks.
//
// shift_test.go measures it over ONE tick, which is what every mover did before
// a rate existed and what every mover with no rate still does. What is measured
// here is that the fraction is taken over the crossing instead — and that at a
// crossing of one, or of none, the two are the same arithmetic.

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// crossingSpan is the crossing this file draws over: sixteen ticks, which is
// what the rate law makes of the speed the original's own worked example uses.
const crossingSpan = 16

// crossingEnts is shiftEntities with a crossing on both units: owed ticks of a
// span, the pair the simulation carries.
func crossingEnts(step image.Point, owed, span int) []MapEntity {
	ents := shiftEntities(step)
	for i := range ents {
		ents[i].Transit, ents[i].TransitSpan = owed, span
	}
	return ents
}

// TestTheDisplacementIsTakenOverTheCrossing walks one crossing tick by tick at
// each tick's start, and asks for the very end at the last tick's end.
//
// The offset must run from a whole cell BEHIND the entity's own cell to zero,
// never backwards, and — the point of the whole thing — must not be home after
// the crossing's first tick.
func TestTheDisplacementIsTakenOverTheCrossing(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every offset here is the contract's own literal", terrain.CellSize)
	}

	var xs []int
	for owed := crossingSpan - 1; owed >= 0; owed-- {
		ents := crossingEnts(shiftEast, owed, crossingSpan)
		v := shiftViewer(t, false, ents, 0, shiftPeriod)
		xs = append(xs, v.entityShift(ents[0]).X)
	}
	last := crossingEnts(shiftEast, 0, crossingSpan)
	xs = append(xs, shiftViewer(t, false, last, shiftPeriod, shiftPeriod).entityShift(last[0]).X)

	if want := -terrain.CellSize; xs[0] != want {
		t.Errorf("the crossing starts drawn at %d, want a whole cell back at %d", xs[0], want)
	}
	if end := xs[len(xs)-1]; end != 0 {
		t.Errorf("the crossing ends drawn at %d, want 0 — on the cell it entered", end)
	}
	if xs[1] == 0 {
		t.Errorf("the entity is drawn home after one tick of a %d-tick crossing", crossingSpan)
	}
	for i := 1; i < len(xs); i++ {
		if xs[i] < xs[i-1] {
			t.Fatalf("the drawn offset went backwards at sample %d: %d after %d", i, xs[i], xs[i-1])
		}
	}
	// Every sample distinct enough to be a walk rather than a jump: a
	// sixteen-tick crossing over a 32-pixel cell moves two pixels a tick.
	for i := 1; i < len(xs)-1; i++ {
		if xs[i] == xs[i-1] {
			t.Errorf("samples %d and %d are both %d, so the crossing is drawn in jumps", i-1, i, xs[i])
		}
	}
}

// TestACrossingOfOneOrNoneDrawsWhatItAlwaysDrew is the compatibility clause: a
// span of 0 — every entity a front-end built before this story pushes — and a
// span of 1 both reduce to the one-tick fraction, exactly.
func TestACrossingOfOneOrNoneDrawsWhatItAlwaysDrew(t *testing.T) {
	for _, ph := range shiftPhases {
		t.Run(ph.name, func(t *testing.T) {
			for _, span := range []int{0, 1} {
				ents := crossingEnts(shiftEast, 0, span)
				v := shiftViewer(t, false, ents, ph.elapsed, shiftPeriod)
				if got := v.entityShift(ents[0]).X; got != ph.wantX {
					t.Errorf("a span of %d drew at %d, want the one-tick %d", span, got, ph.wantX)
				}
			}
		})
	}
}

// TestAnImpossibleCrossingIsBroughtInsideItsOwnSpan: a draw path is handed
// whatever it is handed, so an owed count outside its span is clamped rather
// than refused — the trade the elapsed clamp beside it already makes. The
// simulation writes no such pair; a front-end that made one up still draws
// somewhere between the two cells.
func TestAnImpossibleCrossingIsBroughtInsideItsOwnSpan(t *testing.T) {
	full := crossingEnts(shiftEast, crossingSpan-1, crossingSpan)
	want := shiftViewer(t, false, full, 0, shiftPeriod).entityShift(full[0])

	for _, owed := range []int{crossingSpan, crossingSpan * 4} {
		ents := crossingEnts(shiftEast, owed, crossingSpan)
		if got := shiftViewer(t, false, ents, 0, shiftPeriod).entityShift(ents[0]); got != want {
			t.Errorf("an owed count of %d in a span of %d drew at %v, want the span's own start %v",
				owed, crossingSpan, got, want)
		}
	}
	neg := crossingEnts(shiftEast, -3, crossingSpan)
	zero := crossingEnts(shiftEast, 0, crossingSpan)
	if got, w := shiftViewer(t, false, neg, 0, shiftPeriod).entityShift(neg[0]),
		shiftViewer(t, false, zero, 0, shiftPeriod).entityShift(zero[0]); got != w {
		t.Errorf("a negative owed count drew at %v, want the span's own end %v", got, w)
	}
}
