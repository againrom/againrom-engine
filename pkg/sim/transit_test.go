package sim

// The rate where a tick meets it: one cell per transit, nothing in between, and
// two movers of different speeds arriving in the ratio their transits predict.
//
// rate_test.go measures the law. This file measures what a WORLD does with it,
// which is a different question and the one a rate wired to the wrong term would
// pass the other and fail here.

import "testing"

// trBounds is a corridor wide enough for a straight walk and one diagonal, with
// no obstacle anywhere: what is being measured is time, so nothing here may make
// a mover take a detour.
var trBounds = Bounds{Width: 24, Height: 5}

// walkTicks steps w until the entity stands on (tx,ty) and returns how many
// ticks that took, counting the tick the order was given as tick 0. It gives up
// well past any transit this file uses, so a mover that stops moving fails with
// a count rather than by running forever.
func walkTicks(t *testing.T, w *World, id EntityID, tx, ty int32) int {
	t.Helper()
	cmds := []Command{{Entity: id, X: tx, Y: ty}}
	for n := 0; n < 20000; n++ {
		Step(w, cmds)
		cmds = nil
		e := w.Entities()[indexOfEntity(w.Entities(), id)]
		if e.X == tx && e.Y == ty {
			return n
		}
	}
	t.Fatalf("entity %d never reached (%d,%d)", id, tx, ty)
	return 0
}

// TestARatedMoverCrossesOneCellPerTransit is AC-5's first half: the cadence
// itself, tick by tick, with the cell and the stored route asked on every tick
// of a crossing.
//
// A mover that re-searched between cells would still arrive; what says it does
// not is that nothing about it moves on those ticks.
func TestARatedMoverCrossesOneCellPerTransit(t *testing.T) {
	const speed = 16
	want := transitOf(rateOf(DomainGround, speed, 0, 0, 0, 0), false)
	if want != 16 {
		t.Fatalf("the fixture's transit is %d ticks, want the worked example's 16", want)
	}

	w := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 2, Speed: speed}})
	Step(w, []Command{{Entity: 1, X: 10, Y: 2}})

	e := w.Entities()[0]
	if e.X != 1 || e.Y != 2 {
		t.Fatalf("after the ordering tick the mover is at (%d,%d), want (1,2)", e.X, e.Y)
	}
	if int32(e.TransitTotal) != want || int32(e.Transit) != want-1 {
		t.Fatalf("the crossing is %d/%d, want %d owed of %d", e.Transit, e.TransitTotal, want-1, want)
	}

	// Every tick of the crossing but its last: the cell, the route and the
	// target all still. Only the owed count moves.
	route := w.Route(1)
	for k := int32(1); k < want; k++ {
		Step(w, nil)
		e = w.Entities()[0]
		if e.X != 1 || e.Y != 2 {
			t.Fatalf("tick %d of the crossing moved the mover to (%d,%d)", k, e.X, e.Y)
		}
		if int32(e.Transit) != want-1-k {
			t.Fatalf("tick %d of the crossing left %d owed, want %d", k, e.Transit, want-1-k)
		}
		if got := w.Route(1); len(got) != len(route) {
			t.Fatalf("tick %d of the crossing changed the stored route", k)
		}
	}
	// And the tick after it steps again, which is what makes the cadence the
	// transit's own length rather than one more than it.
	Step(w, nil)
	if e = w.Entities()[0]; e.X != 2 {
		t.Fatalf("the tick after the crossing left the mover at (%d,%d), want column 2", e.X, e.Y)
	}
}

// TestTwoSpeedsArriveInTheRatioTheirTransitsPredict is AC-5's second half, and
// the check that would catch a rate wired to the wrong term: a ratio of 1:1
// passes every arithmetic case in rate_test.go and fails here.
//
// It PRINTS the two counts, because that measurement is what this story owes a
// reader who wants to know what the owner will see.
func TestTwoSpeedsArriveInTheRatioTheirTransitsPredict(t *testing.T) {
	const dist = 10
	for _, speed := range []int32{8, 16, 35, 0} {
		w := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
			[]Entity{{ID: 1, X: 0, Y: 2, Speed: speed}})
		got := walkTicks(t, w, 1, dist, 2)

		// An unrated mover is a cell a tick, so it arrives on the tick its
		// distance names; a rated one owes a transit for every cell but the
		// tick it stepped on.
		want := dist - 1
		transit := 1
		if speed > 0 {
			transit = int(transitOf(rateOf(DomainGround, speed, 0, 0, 0, 0), false))
			want = (dist - 1) * transit
		}
		t.Logf("speed %2d: transit %2d tick(s) a cell, %d cells in %d ticks", speed, transit, dist, got)
		if got != want {
			t.Errorf("speed %d crossed %d cells in %d ticks, want %d", speed, dist, got, want)
		}
	}
}

// TestADiagonalCellCostsMoreInAWorldToo carries the diagonal into a tick: the
// same mover over the same number of cells takes longer on the diagonal, and it
// takes exactly the number the law names.
func TestADiagonalCellCostsMoreInAWorldToo(t *testing.T) {
	const speed = 16
	v := rateOf(DomainGround, speed, 0, 0, 0, 0)
	straight, diagonal := transitOf(v, false), transitOf(v, true)

	// Four cells each way. The straight walk runs along a row; the diagonal one
	// climbs a column at the same time, so every step of it is a diagonal.
	sw := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 0, Speed: speed}})
	dw := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 0, Speed: speed}})

	gotS := walkTicks(t, sw, 1, 4, 0)
	gotD := walkTicks(t, dw, 1, 4, 4)
	t.Logf("speed %d over 4 cells: straight %d ticks, diagonal %d ticks", speed, gotS, gotD)

	if want := 3 * int(straight); gotS != want {
		t.Errorf("the straight walk took %d ticks, want %d", gotS, want)
	}
	if want := 3 * int(diagonal); gotD != want {
		t.Errorf("the diagonal walk took %d ticks, want %d", gotD, want)
	}
	if gotD <= gotS {
		t.Errorf("the diagonal walk (%d) was not slower than the straight one (%d)", gotD, gotS)
	}
}

// TestAnOrderArrivingMidCrossingWaitsForIt: a command lands in phase 1 like any
// other, but a mover half-way across a cell has already committed to it, so the
// new order takes effect when the crossing ends and not before.
func TestAnOrderArrivingMidCrossingWaitsForIt(t *testing.T) {
	const speed = 16
	w := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 5, Y: 2, Speed: speed}})
	Step(w, []Command{{Entity: 1, X: 10, Y: 2}})
	if e := w.Entities()[0]; e.X != 6 {
		t.Fatalf("the mover is at column %d, want 6", e.X)
	}

	// Re-ordered the other way, two ticks into a sixteen-tick crossing.
	Step(w, nil)
	Step(w, []Command{{Entity: 1, X: 0, Y: 2}})
	e := w.Entities()[0]
	if e.X != 6 {
		t.Errorf("the re-order moved a mover mid-crossing, to column %d", e.X)
	}
	if e.TargetX != 0 {
		t.Errorf("the re-order did not reach the target: it is %d", e.TargetX)
	}
	// It turns round on the tick the crossing ends, and not one earlier.
	for k := 0; k < 13; k++ {
		Step(w, nil)
		if got := w.Entities()[0].X; got != 6 {
			t.Fatalf("the mover left column 6 on tick %d of the crossing, at column %d", k+3, got)
		}
	}
	Step(w, nil)
	if got := w.Entities()[0].X; got != 5 {
		t.Errorf("the mover is at column %d after the crossing ended, want 5 — back the way it came", got)
	}
}

// TestAMoverFelledMidCrossingCarriesNone: the crossing is dropped where the blow
// lands, which is the one site that drops it, and the world the blow leaves is
// one the byte form will read back.
func TestAMoverFelledMidCrossingCarriesNone(t *testing.T) {
	w := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 5, Y: 2, Speed: 16, HP: 10, MaxHP: 10}})
	Step(w, []Command{{Entity: 1, X: 10, Y: 2}})
	if e := w.Entities()[0]; e.Transit == 0 {
		t.Fatalf("the mover owes nothing, so this fixture witnesses nothing")
	}

	Step(w, []Command{{Entity: 1, Kind: KindKill}})
	e := w.Entities()[0]
	if e.Transit != 0 || e.TransitTotal != 0 {
		t.Errorf("the corpse carries a crossing of %d/%d", e.Transit, e.TransitTotal)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if err := (&World{}).UnmarshalBinary(form); err != nil {
		t.Errorf("the world a kill left cannot be read back: %v", err)
	}
}

// TestAMoverOwingACrossingTakesNoStallCount: a stall counts consecutive ticks on
// which a near search found nothing, and a mover that made no search has not
// been held up. Without this the give-up would fire on a mover that was simply
// slow.
func TestAMoverOwingACrossingTakesNoStallCount(t *testing.T) {
	w := mustWorldGrid(t, 1, trBounds, ModeCanonical, nil,
		[]Entity{{ID: 1, X: 0, Y: 2, Speed: 8}})
	Step(w, []Command{{Entity: 1, X: 10, Y: 2}})
	// A transit of 32 ticks is twice the give-up limit, so a mover taking a
	// stall count per owed tick would lose its order inside one crossing.
	for k := 0; k < 40; k++ {
		Step(w, nil)
		if e := w.Entities()[0]; e.Stall != 0 || !e.HasTarget {
			t.Fatalf("tick %d: stall %d, target held %v", k, e.Stall, e.HasTarget)
		}
	}
}
