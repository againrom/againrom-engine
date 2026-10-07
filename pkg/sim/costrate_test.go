package sim

import "testing"

// The two planes REACHING the rate law. The law itself is pinned by
// rate_test.go and is not re-tested here; what is tested is that a transit's
// rate is composed from the two cells it joins and from the right two.

// crBounds is a strip four cells wide, so a mover ordered along it makes three
// transits and each joins a named pair.
var crBounds = Bounds{Width: 4, Height: 1}

// crTransit advances a world one tick with the given order and returns the
// transit length the mover took, which is the whole of what a rate is
// observable as.
//
// It reads TransitTotal rather than the rate, because the total is canonical
// state and the rate is not: a rate that reached no transit would be a number
// nothing in the world can see.
func crTransit(t *testing.T, cost, height []byte, speed int32) uint16 {
	t.Helper()
	w, err := NewTerrainWorld(1, crBounds, ModeCanonical,
		Terrain{Cost: cost, Height: height},
		[]Entity{{ID: 1, X: 0, Y: 0, Speed: speed}}, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	Step(w, []Command{{Entity: 1, X: 3, Y: 0}})
	e := w.Entities()[0]
	if e.X != 1 {
		t.Fatalf("the mover is at x=%d after one tick, want 1 — it took no transit to measure", e.X)
	}
	return e.TransitTotal
}

// TestCheaperGroundIsCrossedFaster — AC-7.
//
// One mover, one order, one speed, and the only difference between the two runs
// is the pair of cost bytes the first step joins. The law divides by their mean,
// so the cheap pair must take strictly fewer ticks; the numbers are the shipped
// corpus's own extremes.
func TestCheaperGroundIsCrossedFaster(t *testing.T) {
	t.Parallel()

	cheap := crTransit(t, []byte{6, 6, 6, 6}, nil, 16)
	dear := crTransit(t, []byte{16, 16, 16, 16}, nil, 16)
	middling := crTransit(t, []byte{8, 8, 8, 8}, nil, 16)

	if !(cheap < middling && middling < dear) {
		t.Errorf("transits over cost 6, 8 and 16 are %d, %d and %d — want strictly increasing",
			cheap, middling, dear)
	}
	// And the middling one is what a world naming NO cost plane gives, which is
	// the continuity claim at the rate's own site.
	if none := crTransit(t, nil, nil, 16); none != middling {
		t.Errorf("no cost plane gives a transit of %d and a uniform default plane %d",
			none, middling)
	}
}

// TestTheSlopeTiltSaturates — AC-8.
//
// The tilt is taken from the two cells' height DIFFERENCE, clamped to plus or
// minus 32, so a drop of 33 and a drop of 200 must give the same transit and
// both must differ from level ground. Uphill and downhill are separated by
// walking the same strip with the heights the other way round.
func TestTheSlopeTiltSaturates(t *testing.T) {
	t.Parallel()

	const speed = 20
	flat := crTransit(t, nil, []byte{100, 100, 100, 100}, speed)

	// Cell 0 higher than cell 1: the mover is going DOWN, which lengthens the
	// step and so shortens the transit.
	down32 := crTransit(t, nil, []byte{132, 100, 100, 100}, speed)
	down99 := crTransit(t, nil, []byte{199, 100, 100, 100}, speed)
	// And the other way about.
	up32 := crTransit(t, nil, []byte{100, 132, 100, 100}, speed)
	up99 := crTransit(t, nil, []byte{100, 199, 100, 100}, speed)

	if down32 >= flat {
		t.Errorf("downhill takes %d tick(s) and level ground %d — downhill must be quicker",
			down32, flat)
	}
	if up32 <= flat {
		t.Errorf("uphill takes %d tick(s) and level ground %d — uphill must be slower",
			up32, flat)
	}
	if down99 != down32 {
		t.Errorf("a drop of 99 takes %d tick(s) and a drop of 32 takes %d — the tilt is not clamped",
			down99, down32)
	}
	if up99 != up32 {
		t.Errorf("a climb of 99 takes %d tick(s) and a climb of 32 takes %d — the tilt is not clamped",
			up99, up32)
	}
}

// TestATransitOffTheMapComposesAsItDidBeforeThePlanes — AC-8a.
//
// A search may begin from a cell outside the bounds, so a transit can join a
// cell the planes describe and one they do not. The rate is then composed from
// four zeros, which is what this call passed before either plane existed — and
// the point of the case is that it is NOT composed from one plane byte and one
// zero, which would halve the cost mean and hand such a transit a rate no
// earlier build gave it.
func TestATransitOffTheMapComposesAsItDidBeforeThePlanes(t *testing.T) {
	t.Parallel()

	const speed = 16
	// The mover stands one cell to the WEST of the map and is ordered onto it.
	offMap := func(cost []byte) uint16 {
		t.Helper()
		w, err := NewTerrainWorld(1, crBounds, ModeCanonical,
			Terrain{Cost: cost},
			[]Entity{{ID: 1, X: -1, Y: 0, Speed: speed}}, nil)
		if err != nil {
			t.Fatalf("NewTerrainWorld: %v", err)
		}
		Step(w, []Command{{Entity: 1, X: 3, Y: 0}})
		e := w.Entities()[0]
		if e.X != 0 {
			t.Fatalf("the mover is at x=%d, want the map's own first column", e.X)
		}
		return e.TransitTotal
	}

	// Whatever the plane says about the cell it stepped ONTO, the transit is the
	// one the substitute gives — the same for a plane of 6s and a plane of 16s,
	// and the same as a world with no plane at all.
	six, sixteen, none := offMap([]byte{6, 6, 6, 6}), offMap([]byte{16, 16, 16, 16}), offMap(nil)
	if six != sixteen || six != none {
		t.Errorf("stepping onto the map from outside it takes %d, %d and %d tick(s) over cost "+
			"planes of 6, 16 and none — all three must be the substitute's", six, sixteen, none)
	}
	// And that answer is the one an in-bounds transit over the default gets,
	// which is what "as it did before the planes" means as a number.
	if want := crTransit(t, nil, nil, speed); six != want {
		t.Errorf("the off-map transit is %d tick(s) and the default in-bounds one is %d", six, want)
	}
}
