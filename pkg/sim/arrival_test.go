package sim

// Arrival on open ground, and two units whose lines cross.
//
// Both are runs. The arrival case holds ONE unit and nothing else on the map, so
// the tick count it pins is the contract's — a second unit anywhere near the
// path would make it a detour's, and the case would then measure the detour.

import "testing"

// arrivalBounds is roomy enough that no walk below reaches an edge: a clamped
// walk would be measuring the clamp, which has cases of its own.
var arrivalBounds = Bounds{Width: 12, Height: 12}

// TestAUnitArrivesAfterTheChebyshevDistanceUnderBothModes is AC-10. A tick moves
// a unit at most one cell on each axis, so max(|dx|,|dy|) ticks is the fewest
// any walk can take; that both modes take exactly that many on open ground is
// the claim — neither wastes a tick, and neither takes a cheaper route that is
// longer in steps.
func TestAUnitArrivesAfterTheChebyshevDistanceUnderBothModes(t *testing.T) {
	for _, tc := range []struct {
		what  string
		from  cell
		to    cell
		ticks int
	}{
		{"due east", cell{1, 1}, cell{9, 1}, 8},
		{"due north-east", cell{1, 1}, cell{9, 9}, 8},
		{"east twice as far as north", cell{1, 1}, cell{9, 5}, 8},
		{"back the other way", cell{9, 9}, cell{1, 1}, 8},
		{"west and north together", cell{9, 5}, cell{2, 9}, 7},
		{"one cell diagonally", cell{5, 5}, cell{6, 6}, 1},
	} {
		for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
			w := mustWorldGrid(t, 1, arrivalBounds, mode, openGrid(arrivalBounds),
				[]Entity{{ID: 1, X: tc.from.x, Y: tc.from.y}})

			prev := tc.from
			arrived := 0
			for tick := 1; tick <= tc.ticks+4 && arrived == 0; tick++ {
				if tick == 1 {
					Step(w, []Command{{Entity: 1, X: tc.to.x, Y: tc.to.y}})
				} else {
					Step(w, nil)
				}
				e := w.Entities()[0]

				if abs32(e.X-prev.x) > 1 || abs32(e.Y-prev.y) > 1 {
					t.Fatalf("%s mode %d tick %d: the unit went (%d,%d) to (%d,%d), more than one cell on an axis",
						tc.what, mode, tick, prev.x, prev.y, e.X, e.Y)
				}
				was := prev
				prev = cell{e.X, e.Y}
				if prev == tc.to {
					arrived = tick
					// The arrived unit is the whole entity and not a position, so this
					// fails on any field the walk left residue in. Its FACING is the LAST
					// STEP'S — read off the step the loop just watched rather than
					// written out, because which cell a mode enters the target from is the
					// mode's business and this case is not about that. What is asserted is
					// that the facing is that step's, whichever step it was.
					f, ok := facingToward(e.X-was.x, e.Y-was.y)
					if !ok {
						t.Fatalf("%s mode %d: arrived without a step", tc.what, mode)
					}
					if got := (Entity{ID: 1, X: tc.to.x, Y: tc.to.y, Facing: f, DesiredFacing: f, ActorState: actorStateGuard, Reach: 1,
						PostX: tc.from.x, PostY: tc.from.y}); e != got {
						t.Errorf("%s mode %d: the arrived unit is %+v, want %+v", tc.what, mode, e, got)
					}
				}
			}
			if arrived != tc.ticks {
				t.Errorf("%s mode %d: arrived at tick %d, want %d", tc.what, mode, arrived, tc.ticks)
			}
		}
	}
}

// TestTwoCrossingUnitsNeverShareACellAndReplayTheSame is AC-4. The two direct
// lines meet at (4,4), and the lower id reaches it first, so the higher one is
// resolved against a cell it may not enter and has to give way. What is asserted
// is the invariant and the reproduction, never the detour each mode picks.
func TestTwoCrossingUnitsNeverShareACellAndReplayTheSame(t *testing.T) {
	b := Bounds{Width: 9, Height: 9}
	units := func() []Entity {
		return []Entity{{ID: 1, X: 0, Y: 4}, {ID: 2, X: 4, Y: 0}}
	}
	orders := []Command{{Entity: 1, X: 8, Y: 4}, {Entity: 2, X: 4, Y: 8}}
	const ticks = 20

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		// The trajectories of one run, then of a second run of the same world
		// and the same commands. Equal sequences are the reproduction.
		run := func() [][]Entity {
			w := mustWorldGrid(t, 1, b, mode, openGrid(b), units())
			out := make([][]Entity, 0, ticks)
			for tick := 1; tick <= ticks; tick++ {
				if tick == 1 {
					Step(w, orders)
				} else {
					Step(w, nil)
				}
				ents := w.Entities()
				if ents[0].X == ents[1].X && ents[0].Y == ents[1].Y {
					t.Fatalf("mode %d tick %d: both units stand on (%d,%d)", mode, tick, ents[0].X, ents[0].Y)
				}
				out = append(out, ents)
			}
			return out
		}

		first, second := run(), run()
		for tick := range first {
			for i := range first[tick] {
				if first[tick][i] != second[tick][i] {
					t.Fatalf("mode %d tick %d: entity %d is %+v on the first run and %+v on the second",
						mode, tick+1, first[tick][i].ID, first[tick][i], second[tick][i])
				}
			}
		}

		// The fixture is only a crossing if both units get across it. A pair
		// that gave up would satisfy every assertion above and measure nothing.
		last := first[len(first)-1]
		for _, e := range last {
			if e.HasTarget {
				t.Errorf("mode %d: entity %d still holds a target after %d ticks: %+v", mode, e.ID, ticks, e)
			}
		}
		if last[0].X != 8 || last[0].Y != 4 || last[1].X != 4 || last[1].Y != 8 {
			t.Errorf("mode %v: the pair finished at %+v and %+v, want them across", mode, last[0], last[1])
		}
	}
}
