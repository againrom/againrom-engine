package sim

import (
	"reflect"
	"testing"
)

// What one step onto a named cell costs a named mover.
//
// The law itself is pinned by rate_test.go and the planes reaching it by
// costrate_test.go; neither is re-tested here. What is tested is that the QUERY
// answers with the advance's own number, and that it refuses exactly where the
// advance declines to rate a mover.

// srBounds is wide enough to hold a mover with all eight neighbours in bounds and
// a cell two steps away besides.
var srBounds = Bounds{Width: 5, Height: 3}

// srPlane is a plane of one repeated byte over srBounds, so a fixture that wants
// to vary ONE cell can start from a uniform one and name that cell.
func srPlane(b byte) []byte {
	p := make([]byte, int(srBounds.Width)*int(srBounds.Height))
	for i := range p {
		p[i] = b
	}
	return p
}

func srAt(x, y int32) int { return int(y)*int(srBounds.Width) + int(x) }

func srWorld(t *testing.T, cost, height []byte, ents []Entity) *World {
	t.Helper()
	w, err := NewTerrainWorld(1, srBounds, ModeCanonical,
		Terrain{Cost: cost, Height: height}, ents, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w
}

func TestStepRateIsTheNumberTheAdvanceUses(t *testing.T) {
	t.Parallel()

	down := srPlane(100)
	down[srAt(1, 1)] = 140
	up := srPlane(100)
	up[srAt(2, 1)] = 140

	for _, tc := range []struct {
		what         string
		cost, height []byte
		speed        int32
	}{
		{"a world naming neither plane", nil, nil, 16},
		{"cheap ground", srPlane(6), nil, 16},
		{"dear ground", srPlane(16), nil, 16},
		{"a step downhill", nil, down, 20},
		{"a step uphill", nil, up, 20},
	} {
		t.Run(tc.what, func(t *testing.T) {
			w := srWorld(t, tc.cost, tc.height, []Entity{{ID: 1, X: 1, Y: 1, Speed: tc.speed}})

			_, want, adjacent, ok := w.StepRate(1, 2, 1)
			if !ok {
				t.Fatalf("the query refused a rated live mover and its neighbour")
			}
			if !adjacent {
				t.Errorf("(1,1) to (2,1) reported as no single step")
			}

			Step(w, []Command{{Entity: 1, X: 2, Y: 1}})
			e := w.Entities()[0]
			if e.X != 2 || e.Y != 1 {
				t.Fatalf("the mover stands at (%d,%d) — it did not take the step under test", e.X, e.Y)
			}
			if int32(e.TransitTotal) != want {
				t.Errorf("the query predicted a transit of %d and the advance recorded %d",
					want, e.TransitTotal)
			}
		})
	}
}

// TestStepRateFollowsTheCellUnderTheDestination is AC-2: the figure is a function
// of the terrain being stepped ONTO, not of the mover alone. One cell's cost byte
// is the only difference between the two worlds.
func TestStepRateFollowsTheCellUnderTheDestination(t *testing.T) {
	t.Parallel()

	cheap, dear := srPlane(8), srPlane(8)
	dear[srAt(2, 1)] = 20

	rateOver := func(cost []byte) int32 {
		w := srWorld(t, cost, nil, []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})
		r, _, _, ok := w.StepRate(1, 2, 1)
		if !ok {
			t.Fatal("the query refused a rated live mover and its neighbour")
		}
		return r
	}

	if a, b := rateOver(cheap), rateOver(dear); a <= b {
		t.Errorf("stepping onto cost 8 rates %d and onto cost 20 rates %d — dearer ground must be slower", a, b)
	}
}

// TestStepRateIsRelativeToTheMover is AC-3, and it is the requirement in one
// assertion: two movers, ONE destination cell, uniform terrain, and the only
// difference between them is their speed.
func TestStepRateIsRelativeToTheMover(t *testing.T) {
	t.Parallel()

	w := srWorld(t, srPlane(8), srPlane(100), []Entity{
		{ID: 1, X: 0, Y: 1, Speed: 12},
		{ID: 2, X: 2, Y: 1, Speed: 30},
	})

	slow, _, _, ok1 := w.StepRate(1, 1, 1)
	fast, _, _, ok2 := w.StepRate(2, 1, 1)
	if !ok1 || !ok2 {
		t.Fatalf("the query refused a rated live mover: %v %v", ok1, ok2)
	}
	if slow >= fast {
		t.Errorf("onto one cell the slow mover rates %d and the fast one %d", slow, fast)
	}
}

// TestStepRateTakesTheGroupTermOverTheUnitsOwnSpeed is AC-4. A unit carrying a
// group term moves at the term and its own speed is inert, so the query must
// answer what the term gives and not what the class speed would.
func TestStepRateTakesTheGroupTermOverTheUnitsOwnSpeed(t *testing.T) {
	t.Parallel()

	rateOf := func(e Entity) int32 {
		w := srWorld(t, srPlane(8), nil, []Entity{e})
		r, _, _, ok := w.StepRate(e.ID, 2, 1)
		if !ok {
			t.Fatal("the query refused a rated live mover and its neighbour")
		}
		return r
	}

	held := rateOf(Entity{ID: 1, X: 1, Y: 1, Speed: 40, GroupSpeed: 12})
	term := rateOf(Entity{ID: 1, X: 1, Y: 1, Speed: 12})
	own := rateOf(Entity{ID: 1, X: 1, Y: 1, Speed: 40})

	if held != term {
		t.Errorf("a unit of speed 40 held at a group term of 12 rates %d; a unit of speed 12 rates %d", held, term)
	}
	if held == own {
		t.Errorf("the group term changed nothing — speed 40 rates %d either way", own)
	}
}

// TestStepRateSeparatesUphillFromDownhill is AC-5: the ordered pair is ordered,
// and source and destination are not interchangeable. Two movers on ONE height
// plane ask about each other's cell, so the two calls differ in nothing but which
// way round the pair is.
func TestStepRateSeparatesUphillFromDownhill(t *testing.T) {
	t.Parallel()

	h := srPlane(100)
	h[srAt(1, 1)] = 140

	w := srWorld(t, nil, h, []Entity{
		{ID: 1, X: 1, Y: 1, Speed: 20},
		{ID: 2, X: 2, Y: 1, Speed: 20},
	})

	downRate, downTransit, _, ok1 := w.StepRate(1, 2, 1)
	upRate, upTransit, _, ok2 := w.StepRate(2, 1, 1)
	if !ok1 || !ok2 {
		t.Fatalf("the query refused a rated live mover: %v %v", ok1, ok2)
	}
	if downRate <= upRate {
		t.Errorf("downhill rates %d and uphill %d — downhill must be the faster", downRate, upRate)
	}
	if downTransit >= upTransit {
		t.Errorf("downhill takes %d tick(s) and uphill %d — downhill must be the shorter", downTransit, upTransit)
	}
}

// TestADiagonalStepTakesLongerThanAnOrthogonalOne is AC-6. Same mover, same
// uniform terrain, same rate — the step SHRINKS by the diagonal constant while
// the cell to cross does not, so the transit grows.
func TestADiagonalStepTakesLongerThanAnOrthogonalOne(t *testing.T) {
	t.Parallel()

	w := srWorld(t, srPlane(8), srPlane(100), []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})

	orthRate, orthTransit, _, ok1 := w.StepRate(1, 2, 1)
	diagRate, diagTransit, _, ok2 := w.StepRate(1, 2, 2)
	if !ok1 || !ok2 {
		t.Fatalf("the query refused a rated live mover: %v %v", ok1, ok2)
	}
	if orthRate != diagRate {
		t.Errorf("the rate differs by direction: %d orthogonal, %d diagonal — it is a property of the pair's cells", orthRate, diagRate)
	}
	if diagTransit <= orthTransit {
		t.Errorf("a diagonal transit takes %d tick(s) and an orthogonal one %d — the diagonal must be the longer", diagTransit, orthTransit)
	}
}

func TestStepRateRefusesWhatNoTickWouldRate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what string
		ent  Entity
		id   EntityID
		x, y int32
	}{
		{"an id this world does not hold", Entity{ID: 1, X: 1, Y: 1, Speed: 16}, 9, 2, 1},
		{"a mover of zero effective speed", Entity{ID: 1, X: 1, Y: 1}, 1, 2, 1},
		{"a mover that is downed", Entity{ID: 1, X: 1, Y: 1, Speed: 16, HP: 0, MaxHP: 10}, 1, 2, 1},
		{"a mover that is dead", Entity{ID: 1, X: 1, Y: 1, Speed: 16, HP: -1, MaxHP: 10}, 1, 2, 1},
		{"a destination equal to the source", Entity{ID: 1, X: 1, Y: 1, Speed: 16}, 1, 1, 1},
	} {
		t.Run(tc.what, func(t *testing.T) {
			w := srWorld(t, srPlane(8), nil, []Entity{tc.ent})
			r, transit, adjacent, ok := w.StepRate(tc.id, tc.x, tc.y)
			if ok {
				t.Fatalf("the query answered rate %d over %d tick(s)", r, transit)
			}
			if r != 0 || transit != 0 || adjacent {
				t.Errorf("a refused query still returned %d, %d, %v", r, transit, adjacent)
			}
		})
	}
}

func TestStepRateReportsWhichPairsAreOneStep(t *testing.T) {
	t.Parallel()

	w := srWorld(t, srPlane(8), srPlane(100), []Entity{{ID: 1, X: 2, Y: 1, Speed: 16}})

	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if _, _, adjacent, ok := w.StepRate(1, 2+dx, 1+dy); !ok || !adjacent {
				t.Errorf("the neighbour at (%+d,%+d) reported ok=%v adjacent=%v", dx, dy, ok, adjacent)
			}
		}
	}

	for _, far := range [][2]int32{{0, 1}, {4, 1}, {2, 1 + 2}, {0, 0}} {
		r, transit, adjacent, ok := w.StepRate(1, far[0], far[1])
		if !ok {
			t.Errorf("the pair to (%d,%d) was refused rather than marked", far[0], far[1])
			continue
		}
		if adjacent {
			t.Errorf("(%d,%d) is more than one cell from (2,1) and was reported as a single step", far[0], far[1])
		}
		if r <= 0 || transit <= 0 {
			t.Errorf("a marked pair returned no figure: rate %d over %d tick(s)", r, transit)
		}
	}
}

func TestStepRateWritesNothing(t *testing.T) {
	t.Parallel()

	w := srWorld(t, srPlane(12), srPlane(140), []Entity{{ID: 1, X: 1, Y: 1, Speed: 16}})

	before := snap(w)
	bytesBefore, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	digestBefore := w.Hash()

	if _, _, _, ok := w.StepRate(1, 2, 2); !ok {
		t.Fatal("the query refused a rated live mover and its neighbour")
	}

	if got := snap(w); !reflect.DeepEqual(got, before) {
		t.Errorf("the query mutated the world:\n before %+v\n after  %+v", before, got)
	}
	bytesAfter, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !reflect.DeepEqual(bytesAfter, bytesBefore) {
		t.Error("the world's byte form changed across a query")
	}
	if w.Hash() != digestBefore {
		t.Error("the world's digest changed across a query")
	}
}
