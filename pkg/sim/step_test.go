package sim

import (
	"reflect"
	"testing"
)

// stepBounds is the extent every walk in this file stays inside. It IS a
// movement constraint — a unit never leaves it — so a fixture that ran into an
// edge would be measuring the clamp rather than the walk, and the clamp has a
// case of its own below.
var stepBounds = Bounds{Width: 8, Height: 6}

// abs32 and dist are the test's OWN arithmetic. AC-1's arrival tick is
// max(|dx|,|dy|); the step computes no such quantity, so the test must, and the
// hand-written paths below are checked against it rather than against the step.
func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func dist(from Entity, to Command) int {
	d := int(abs32(to.X - from.X))
	if v := int(abs32(to.Y - from.Y)); v > d {
		d = v
	}
	return d
}

// walk is one AC-1 case. path is written out cell by cell by hand — the position
// after tick 1, tick 2, and so on to arrival — never recomputed from the step's
// own rule, and its length is cross-checked against dist.
type walk struct {
	name  string
	start Entity
	cmd   Command
	path  [][2]int32
}

var walks = []walk{
	{
		name:  "straight along x",
		start: Entity{ID: 1, X: 2, Y: 3},
		cmd:   Command{Entity: 1, X: 5, Y: 3},
		path:  [][2]int32{{3, 3}, {4, 3}, {5, 3}},
	},
	{
		name:  "straight along y, descending",
		start: Entity{ID: 1, X: 5, Y: 5},
		cmd:   Command{Entity: 1, X: 5, Y: 2},
		path:  [][2]int32{{5, 4}, {5, 3}, {5, 2}},
	},
	{
		name:  "diagonal",
		start: Entity{ID: 1, X: 0, Y: 0},
		cmd:   Command{Entity: 1, X: 3, Y: 3},
		path:  [][2]int32{{1, 1}, {2, 2}, {3, 3}},
	},
	{
		name:  "diagonal until the short axis arrives, then straight",
		start: Entity{ID: 1, X: 0, Y: 0},
		cmd:   Command{Entity: 1, X: 4, Y: 2},
		path:  [][2]int32{{1, 1}, {2, 2}, {3, 2}, {4, 2}},
	},
	{
		name:  "both axes negative, into the origin",
		start: Entity{ID: 1, X: 2, Y: 2},
		cmd:   Command{Entity: 1, X: 0, Y: 0},
		path:  [][2]int32{{1, 1}, {0, 0}},
	},
	{
		name:  "along the top edge, where one whole row of neighbours is off the map",
		start: Entity{ID: 1, X: 2, Y: 0},
		cmd:   Command{Entity: 1, X: 5, Y: 0},
		path:  [][2]int32{{3, 0}, {4, 0}, {5, 0}},
	},
}

func TestStepWalksOneCellPerTickAndClearsTheTargetOnArrival(t *testing.T) {
	for _, tc := range walks {
		t.Run(tc.name, func(t *testing.T) {
			if d := dist(tc.start, tc.cmd); d != len(tc.path) {
				t.Fatalf("max(|dx|,|dy|) is %d but the path names %d cell(s)", d, len(tc.path))
			}
			w := mustWorld(t, 1, stepBounds, []Entity{tc.start})

			for i, want := range tc.path {
				if i == 0 {
					Step(w, []Command{tc.cmd})
				} else {
					Step(w, nil)
				}
				got := w.Entities()
				if len(got) != 1 {
					t.Fatalf("after %d tick(s) the world holds %d entities", i+1, len(got))
				}
				e := got[0]
				if e.X != want[0] || e.Y != want[1] {
					t.Fatalf("after %d tick(s): at (%d,%d), want (%d,%d)", i+1, e.X, e.Y, want[0], want[1])
				}
				if w.Tick() != uint64(i+1) {
					t.Fatalf("after %d Step call(s) the tick is %d", i+1, w.Tick())
				}
				arrived := i == len(tc.path)-1
				if e.HasTarget == arrived {
					t.Fatalf("after %d tick(s) HasTarget is %v (arrived=%v)", i+1, e.HasTarget, arrived)
				}
				if arrived {
					if e.TargetX != 0 || e.TargetY != 0 {
						t.Errorf("a cleared target left residue (%d,%d)", e.TargetX, e.TargetY)
					}
				} else if e.TargetX != tc.cmd.X || e.TargetY != tc.cmd.Y {
					t.Errorf("after %d tick(s) the target is (%d,%d), want (%d,%d)",
						i+1, e.TargetX, e.TargetY, tc.cmd.X, tc.cmd.Y)
				}
			}

			// Arrived and still: further ticks move nothing.
			at := w.Entities()[0]
			for i := 0; i < 3; i++ {
				Step(w, nil)
				if got := w.Entities()[0]; got != at {
					t.Fatalf("%d idle tick(s) after arrival moved the entity: %+v, want %+v", i+1, got, at)
				}
			}
		})
	}
}

// TestStepClearsATargetNamingTheEntitysOwnCell is AC-1's d == 0 edge: the
// arrival check runs after the move, not instead of it, so an order to stand
// where the entity already stands is satisfied by the step that applies it.
func TestStepClearsATargetNamingTheEntitysOwnCell(t *testing.T) {
	w := mustWorld(t, 1, stepBounds, []Entity{{ID: 1, X: 3, Y: 3}})
	Step(w, []Command{{Entity: 1, X: 3, Y: 3}})

	want := Entity{ID: 1, X: 3, Y: 3, ActorState: actorStateGuard, Reach: 1, PostX: 3, PostY: 3}
	if got := w.Entities()[0]; got != want {
		t.Errorf("entity is %+v, want %+v", got, want)
	}
}

func TestStepAppliesCommandsBeforeMoving(t *testing.T) {
	w := mustWorld(t, 1, stepBounds, []Entity{{ID: 1, X: 0, Y: 0}})
	Step(w, []Command{{Entity: 1, X: 3, Y: 0}})

	if got := w.Entities()[0]; got.X != 1 {
		t.Errorf("after the tick that carried the order the entity is at X=%d, want 1 "+
			"(X=0 means the command was applied after the move phase)", got.X)
	}
}

// TestStepIgnoresACommandNamingAnAbsentEntity is AC-2's first half. The absent
// ids bracket and interleave the present ones, so a search that walked off the
// slice, or a loop that gave up on the first miss, fails here.
func TestStepIgnoresACommandNamingAnAbsentEntity(t *testing.T) {
	ents := []Entity{{ID: 2, X: 1, Y: 1}, {ID: 5, X: 4, Y: 4}}
	w := mustWorld(t, 0xC0FFEE, stepBounds, ents)
	before := snap(w)

	Step(w, []Command{{Entity: 0, X: 7, Y: 7}, {Entity: 3, X: 7, Y: 7}, {Entity: 99, X: 7, Y: 7}})

	after := snap(w)
	if after.tick != before.tick+1 {
		t.Errorf("tick is %d, want %d", after.tick, before.tick+1)
	}
	after.tick = before.tick
	if !reflect.DeepEqual(after, before) {
		t.Errorf("commands naming absent entities changed the world:\n before %+v\n after  %+v", before, after)
	}

	// A miss must not stop the slice: the order after the two misses is obeyed.
	// What that shows is that the unit MOVED and still carries the order it was
	// given — which cell it took is the search's answer and is pinned where the
	// search is.
	Step(w, []Command{{Entity: 0, X: 9, Y: 9}, {Entity: 3, X: 9, Y: 9}, {Entity: 5, X: 7, Y: 5}})
	got := w.Entities()[1]
	if got.X == 4 && got.Y == 4 {
		t.Errorf("the order after two misses was dropped: the unit is still at its start, %+v", got)
	}
	if !got.HasTarget || got.TargetX != 7 || got.TargetY != 5 {
		t.Errorf("the order after two misses was dropped: %+v", got)
	}
}

// TestStepLaterCommandForTheSameEntityWins is AC-2's second half. The two orders
// point opposite ways along x, so the unit's single cell of movement says which
// one took effect — whatever the search does with the other axis.
func TestStepLaterCommandForTheSameEntityWins(t *testing.T) {
	w := mustWorld(t, 1, stepBounds, []Entity{{ID: 2, X: 4, Y: 3}})
	Step(w, []Command{{Entity: 2, X: 7, Y: 5}, {Entity: 2, X: 1, Y: 1}})

	got := w.Entities()[0]
	if got.X >= 4 {
		t.Errorf("the unit is at X=%d, want less than 4 — the earlier order was the one obeyed", got.X)
	}
	if !got.HasTarget || got.TargetX != 1 || got.TargetY != 1 {
		t.Errorf("the unit holds target %v (%d,%d), want the later order (1,1)",
			got.HasTarget, got.TargetX, got.TargetY)
	}
}

func TestStepAdvancesTheTickExactlyOnceAndDrawsNoRandomness(t *testing.T) {
	const seed = 0xDEADBEEF
	w := mustWorld(t, seed, stepBounds, []Entity{{ID: 1, X: 0, Y: 0}, {ID: 4, X: 7, Y: 5}})

	for i := 1; i <= 6; i++ {
		Step(w, []Command{{Entity: 1, X: 6, Y: 6}, {Entity: 4, X: 0, Y: 0}})
		if w.Tick() != uint64(i) {
			t.Fatalf("after %d Step call(s) the tick is %d", i, w.Tick())
		}
		if w.rng.state != seed {
			t.Fatalf("after %d Step call(s) the rng state is %#x, want the seed %#x", i, w.rng.state, seed)
		}
	}
}

// TestStepIsClampedByTheBounds replaces what this file used to say, which was
// that bounds are recorded state and not a movement constraint. They are one
// now: a unit never leaves them, so the SAME unit under the SAME order walks in
// one world and does not walk in the other, and the only difference between the
// two worlds is the extent.
//
// The order is deliberately not refused when it is given. It is accepted and
// searched for, the target is never labelled because nothing off the map is
// open to either relation, and the search SETTLES for the labelled cell
// nearest what was asked for. So "as far as the map goes" is exactly what
// happens: the mover walks to the in-bounds corner nearest the cell it was
// sent to and stops there, where the roomy world's mover walks the whole
// way.
//
// That is a stronger reading of the clamp than the one this test used to make,
// and it is the same clamp. The unit is held inside the extent by the cells a
// search may label and by nothing else — no coordinate is trimmed anywhere — so
// the two worlds still differ in nothing but their extent, and the one whose
// extent contains the target is the one that reaches it.
func TestStepIsClampedByTheBounds(t *testing.T) {
	ents := []Entity{{ID: 1, X: 6, Y: 4}}
	roomy := mustWorld(t, 1, Bounds{Width: 64, Height: 48}, ents)
	cramped := mustWorld(t, 1, Bounds{Width: 8, Height: 6}, ents)

	cmds := []Command{{Entity: 1, X: 11, Y: 9}}
	for i := 1; i <= 6; i++ {
		Step(roomy, cmds)
		Step(cramped, cmds)
		cmds = nil
	}

	if got := roomy.Entities()[0]; got.X != 11 || got.Y != 9 || got.HasTarget {
		t.Errorf("in the roomy world the target was not reached: %+v", got)
	}
	got := cramped.Entities()[0]
	if got.X != 7 || got.Y != 5 {
		t.Errorf("in the cramped world the unit is at (%d,%d), want the in-bounds corner (7,5) nearest "+
			"the cell it was sent to — the extent is what it settles against", got.X, got.Y)
	}
	if got.HasTarget || got.TargetX != 0 || got.TargetY != 0 || got.Stall != 0 {
		t.Errorf("the settled unit is %+v, want its order ended on arrival with no residue", got)
	}
	if reflect.DeepEqual(roomy.Entities(), cramped.Entities()) {
		t.Error("the two worlds agree, so this case cannot see the clamp at all")
	}
}

func TestStepIsAPureFunctionOfStateAndCommands(t *testing.T) {
	b, ents := sample()
	schedule := [][]Command{
		{{Entity: 2, X: -4, Y: 9}},
		nil,
		{{Entity: 7, X: 0, Y: 0}, {Entity: 5, X: 3, Y: 3}},
		{{Entity: 2, X: 20, Y: 20}},
		nil,
	}
	one := mustWorld(t, 12345, b, ents)
	two := mustWorld(t, 12345, b, ents)

	for i, cmds := range schedule {
		Step(one, cmds)
		Step(two, cmds)
		if a, c := snap(one), snap(two); !reflect.DeepEqual(a, c) {
			t.Fatalf("tick %d: two runs of the same inputs diverged:\n %+v\n %+v", i+1, a, c)
		}
	}
}

func TestStepIsIndependentOfInsertionOrder(t *testing.T) {
	b, ents := sample()
	reversed := make([]Entity, len(ents))
	for i, e := range ents {
		reversed[len(ents)-1-i] = e
	}
	forward := mustWorld(t, 99, b, ents)
	backward := mustWorld(t, 99, b, reversed)

	cmds := []Command{{Entity: 7, X: 12, Y: 12}, {Entity: 2, X: 11, Y: 12}, {Entity: 5, X: -6, Y: 20}}
	for i := 1; i <= 8; i++ {
		Step(forward, cmds)
		Step(backward, cmds)
		cmds = nil
		if a, c := forward.Entities(), backward.Entities(); !reflect.DeepEqual(a, c) {
			t.Fatalf("tick %d: insertion order is observable: %v vs %v", i, a, c)
		}
	}
}

func TestStepMutatesTheWorldInPlace(t *testing.T) {
	b, ents := sample()
	w := mustWorld(t, 1, b, ents)
	array := &w.entities[0]

	Step(w, []Command{{Entity: 2, X: 40, Y: 40}})

	if &w.entities[0] != array {
		t.Error("Step replaced the world's entity slice instead of writing through it")
	}
	if w.entities[0].X == 11 && w.entities[0].Y == 12 {
		t.Error("the world's own storage did not move")
	}
}

func TestStepIsIndependentOfEveryClassID(t *testing.T) {
	b, ents := sample()
	classes := map[EntityID]int32{2: 34, 5: -5, 7: 80}
	classed := make([]Entity, len(ents))
	for i, e := range ents {
		e.Class = classes[e.ID]
		classed[i] = e
	}

	plain := mustWorld(t, 12345, b, ents)
	rich := mustWorld(t, 12345, b, classed)

	schedule := [][]Command{
		{{Entity: 2, X: -4, Y: 9}},
		nil,
		{{Entity: 7, X: 0, Y: 0}, {Entity: 5, X: 3, Y: 3}},
		{{Entity: 2, X: 20, Y: 20}},
		nil,
	}
	for i, cmds := range schedule {
		Step(plain, cmds)
		Step(rich, cmds)

		a, c := snap(plain), snap(rich)
		for j := range c.entities {
			if got, want := c.entities[j].Class, classes[c.entities[j].ID]; got != want {
				t.Fatalf("tick %d: entity %d's class id is %d, construction set %d",
					i+1, c.entities[j].ID, got, want)
			}
			if a.entities[j].Class != 0 {
				t.Fatalf("tick %d: the class-less world's entity %d gained class %d",
					i+1, a.entities[j].ID, a.entities[j].Class)
			}
			// The one intended difference, verified above and masked here so the
			// comparison below covers every OTHER field of the stepped state.
			c.entities[j].Class = 0
		}
		if !reflect.DeepEqual(a, c) {
			t.Fatalf("tick %d: class ids reached the step:\n %+v\n %+v", i+1, a, c)
		}
	}
}
