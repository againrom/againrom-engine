package sim

// This file carries the invariant over a whole run rather than the outcome of one
// contest: through a long run in which units genuinely contend, no tick ever
// leaves two of them standing on one cell. Two runs carry it — a named N-way
// convergence and a seeded sweep — and each is paired with a counterfactual,
// because "no two units ever shared a cell" and "no cell gained an occupant" are
// both perfectly true of a rule that moves nobody at all.
//
// Who wins a contest is deliberately asserted nowhere here. That claim belongs to
// the cases that name a winner: everything in this file holds under a descending
// resolution order too, and a criterion here that failed under one would be
// answering a question this file does not ask.
//
// occEntity comes from occupancy_test.go, dist from step_test.go and blkFree from
// blocked_test.go. A second copy of any of them would only be somewhere for the
// two to drift apart.

import "testing"

// conBounds is roomy enough that nothing here walks out of it. These runs are
// about a unit stopped by a unit, and bounds are recorded state rather than a
// movement constraint, so a run that left the map would be staging two things at
// once.
var conBounds = Bounds{Width: 16, Height: 16}

// conShared is the one cell every contender of the convergence is ordered onto.
var conShared = [2]int32{6, 6}

// conContenders are three units the same number of steps from conShared, listed
// in ascending id. Equidistant is what makes this a contest rather than a queue: a
// rule that resolved each unit without reference to the others would put all three
// on that cell on the same tick. Their ids are pairwise non-adjacent with conIdle's
// ids falling between them, so a rule that compared a unit only against its
// neighbour in the slice answers a two-unit contest correctly and this one wrongly.
var conContenders = []Entity{
	{ID: 2, X: 3, Y: 6},
	{ID: 6, X: 9, Y: 6},
	{ID: 10, X: 6, Y: 3},
}

// conIdle stand between the contenders in id and nowhere near them on the map.
// They are given no target in the fixture and no command anywhere in the run.
var conIdle = []Entity{
	{ID: 4, X: 1, Y: 14, ActorState: actorStateGuard, Reach: 1, PostX: 1, PostY: 14},
	{ID: 8, X: 14, Y: 1, ActorState: actorStateGuard, Reach: 1, PostX: 14, PostY: 1},
}

// conNWayTicks runs the convergence well past the tick all three would have
// arrived on, so that what the invariant is asserted over is a standstill that
// held rather than one the run stopped looking at.
const conNWayTicks = 8

const (
	// conSweepSeed is the constant the sweep's own generator is seeded with: the
	// randomness is the test's, never the world's, so the sweep asserts nothing
	// about a generator state it moved itself.
	conSweepSeed = 0x0f1e2d3c4b5a6978
	// conSweepUnits and conSweepSide put twelve units in sixty-four cells, which
	// is dense enough that they cannot walk past each other for twenty-four ticks
	// by luck.
	conSweepUnits = 12
	conSweepSide  = 8
	conSweepTicks = 24
)

// conHeld is the test's own scan for an occupied cell. The package's own predicate
// is the thing under test and must not be borrowed to build a fixture: one that
// reported every cell held would spin the draw loop below forever, and one that
// reported every cell free would hand it duplicate start cells — in both
// directions the fixture would be measuring itself.
func conHeld(ents []Entity, x, y int32) bool {
	for i := range ents {
		if ents[i].X == x && ents[i].Y == y {
			return true
		}
	}
	return false
}

// conDistinct is the invariant itself, and it is asserted after every tick rather
// than once at the end: a state checked only on the last tick says nothing about
// the ticks before it, and a pair that shared a cell and parted again would pass.
func conDistinct(t *testing.T, tick int, ents []Entity) {
	t.Helper()
	for i := range ents {
		for j := i + 1; j < len(ents); j++ {
			if ents[i].X == ents[j].X && ents[i].Y == ents[j].Y {
				t.Fatalf("tick %d: units %d and %d both stand on (%d,%d)",
					tick, ents[i].ID, ents[j].ID, ents[i].X, ents[i].Y)
			}
		}
	}
}

// conFind reads one unit out of a recorded tick by id, so a trajectory is
// compared unit by unit and never by position in a slice.
func conFind(t *testing.T, ents []Entity, id EntityID) Entity {
	t.Helper()
	for _, e := range ents {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("the recorded tick holds no unit %d", id)
	return Entity{}
}

// conRemaining is how many ticks a unit still needs to reach the target it holds —
// the Chebyshev distance, on the test's own arithmetic — and zero for a unit
// holding none.
func conRemaining(e Entity) int {
	if !e.HasTarget {
		return 0
	}
	return dist(e, Command{Entity: e.ID, X: e.TargetX, Y: e.TargetY})
}

// conSolo replays the commands that name one unit, tick for tick, in a world
// holding that unit alone. A world with one unit in it has nothing to contend
// with, so the walk it returns is the one the rule makes when nothing stands in
// the way.
//
// It is the run-length form of the control the single-tick cases carry. Parking
// one obstacle in a checked-empty corner does not survive a run, because which
// unit obstructs which changes tick by tick and there is no one cell to park;
// taking every other unit away is the same idea to the end of the run.
func conSolo(t *testing.T, start Entity, cmds [][]Command) []Entity {
	t.Helper()
	w := mustWorld(t, 1, conBounds, []Entity{start})
	out := make([]Entity, 0, len(cmds))
	for _, tc := range cmds {
		var mine []Command
		for _, c := range tc {
			if c.Entity == start.ID {
				mine = append(mine, c)
			}
		}
		Step(w, mine)
		out = append(out, occEntity(t, w, start.ID))
	}
	return out
}

// conRefused is the assertion that the run actually blocked somebody, and it is
// stated against the solo walks in one direction only: some unit, at some tick, is
// further from the target it is holding than the same unit under the same orders
// with nobody in its way.
//
// Counting the units that did not move would not do, and that is measured rather
// than supposed. With the movement guard weakened so that nothing orthogonal
// moves, every unit stands still on every tick, such a count comes out at its
// maximum, and the invariant holds perfectly — a rule that moves nobody shares no
// cell. Against the solo walks that same rule blocks nobody, because the solo
// walks stand still too, and the comparison goes quiet as it should.
func conRefused(t *testing.T, what string, start []Entity, cells [][]Entity, cmds [][]Command) {
	t.Helper()
	for _, e := range start {
		solo := conSolo(t, e, cmds)
		for tick := range cells {
			if conRemaining(conFind(t, cells[tick], e.ID)) > conRemaining(solo[tick]) {
				return
			}
		}
	}
	t.Errorf("%s: no unit anywhere in the run is further from its target than the same unit walking alone — nothing was ever blocked, so the invariant above held over a run that never contended",
		what)
}

// TestStepKeepsEveryUnitOnACellOfItsOwnThroughAnNWayConvergence sends three units
// the same distance onto one cell and then keeps looking for five more ticks. A
// two-unit contest is the case a wrong rule passes: comparing a unit against the
// one before it in the slice, or arbitrating a pair, answers two correctly and
// three wrongly, which is why the contenders are non-adjacent in id with an
// uninvolved unit between them. Nothing here names a winner — the invariant does
// not care which of the three took the cell, only that the other two did not join
// it — and the run's own claim to have contended at all is the solo comparison and
// not the standstill.
func TestStepKeepsEveryUnitOnACellOfItsOwnThroughAnNWayConvergence(t *testing.T) {
	start := append(append([]Entity{}, conContenders...), conIdle...)

	// The fixture, on the test's own arithmetic. Five things have to hold or the
	// run would pass for a reason that is not the rule: the three converge on one
	// tick, that cell starts empty, no two units start on one cell, the ids are
	// pairwise non-adjacent, and an uninvolved id falls between each pair.
	steps := -1
	for _, e := range conContenders {
		d := dist(e, Command{Entity: e.ID, X: conShared[0], Y: conShared[1]})
		if steps < 0 {
			steps = d
		}
		if d != steps {
			t.Fatalf("fixture: unit %d is %d steps from the shared cell, the first contender is %d — they must converge on one tick",
				e.ID, d, steps)
		}
	}
	if steps >= conNWayTicks {
		t.Fatalf("fixture: the contenders need %d ticks and the run is %d — it must outlast the convergence",
			steps, conNWayTicks)
	}
	blkFree(t, "the shared cell", start, conShared[0], conShared[1])
	conDistinct(t, 0, start)
	for i := 1; i < len(conContenders); i++ {
		lo, hi := conContenders[i-1].ID, conContenders[i].ID
		if lo >= hi || lo+1 == hi {
			t.Fatalf("fixture: contender ids %d and %d must ascend and must not be adjacent", lo, hi)
		}
		between := false
		for _, e := range conIdle {
			between = between || (e.ID > lo && e.ID < hi)
		}
		if !between {
			t.Fatalf("fixture: no uninvolved id falls between %d and %d", lo, hi)
		}
	}

	orders := make([]Command, 0, len(conContenders))
	for _, e := range conContenders {
		orders = append(orders, Command{Entity: e.ID, X: conShared[0], Y: conShared[1]})
	}

	w := mustWorld(t, 1, conBounds, start)
	cmds := make([][]Command, 0, conNWayTicks)
	cells := make([][]Entity, 0, conNWayTicks)
	for tick := 1; tick <= conNWayTicks; tick++ {
		var tc []Command
		if tick == 1 {
			tc = orders
		}
		Step(w, tc)
		ents := w.Entities()
		conDistinct(t, tick, ents)
		cmds, cells = append(cmds, tc), append(cells, ents)
	}
	conRefused(t, "the convergence", start, cells, cmds)

	// The uninvolved units are part of the fixture's claim and not decoration: a
	// rule that moved one of them would be answering the id question with
	// something other than the ids.
	for _, e := range conIdle {
		if got := occEntity(t, w, e.ID); got != e {
			t.Errorf("the uninvolved unit is %+v, want %+v", got, e)
		}
	}
}

// conSweepStart draws conSweepUnits units onto distinct cells of the window. Their
// ids ascend by two, so no two are adjacent and nothing in the sweep can be
// answered by comparing a unit against its neighbour in the slice.
func conSweepStart(r *rng) []Entity {
	ents := make([]Entity, 0, conSweepUnits)
	for id := EntityID(1); len(ents) < conSweepUnits; id += 2 {
		x, y := int32(r.next()%conSweepSide), int32(r.next()%conSweepSide)
		if conHeld(ents, x, y) {
			continue
		}
		ents = append(ents, Entity{ID: id, X: x, Y: y})
	}
	return ents
}

// conSweepOrders gives every unit holding no target a fresh one, in ascending id,
// so that the command stream a run produces follows from the seed and the world's
// state alone. A unit that arrives is sent somewhere else on the next tick, which
// is what keeps the run contending to its end instead of settling after the first
// few ticks.
func conSweepOrders(r *rng, ents []Entity) []Command {
	var out []Command
	for _, e := range ents {
		if e.HasTarget {
			continue
		}
		out = append(out, Command{
			Entity: e.ID,
			X:      int32(r.next() % conSweepSide),
			Y:      int32(r.next() % conSweepSide),
		})
	}
	return out
}

// conSweep is one whole run of the sweep: the start it drew, the commands it
// issued, the cells after every tick and the digest after every tick. rngMoved
// reports whether the world's own generator advanced, which it must not: the
// sweep's randomness is the test's, and a run that had spent the world's
// generator would be comparing digests it had moved itself.
type conSweep struct {
	start    []Entity
	cmds     [][]Command
	cells    [][]Entity
	digests  []uint64
	rngMoved bool
}

// conSweepRun walks the sweep from the seed, asserting the invariant after every
// tick as it goes. Two calls of it are the same seed run twice.
func conSweepRun(t *testing.T) conSweep {
	t.Helper()
	r := rng{state: conSweepSeed}
	start := conSweepStart(&r)
	conDistinct(t, 0, start)

	w := mustWorld(t, 1, conBounds, start)
	before := w.rng
	out := conSweep{start: start}
	for tick := 1; tick <= conSweepTicks; tick++ {
		tc := conSweepOrders(&r, w.Entities())
		Step(w, tc)
		ents := w.Entities()
		conDistinct(t, tick, ents)
		out.cmds, out.cells = append(out.cmds, tc), append(out.cells, ents)
		out.digests = append(out.digests, w.Hash())
	}
	out.rngMoved = w.rng != before
	return out
}

// TestStepKeepsEveryUnitOnACellOfItsOwnThroughASeededSweep is the invariant over a
// run nobody laid out by hand: twelve units in sixty-four cells, targets redrawn
// from a fixed seed as fast as they are reached, twenty-four ticks. The named
// convergence above shows the rule holding a shape chosen to break it; this shows
// it holding shapes nobody chose, and the two together are the sampling the
// contract claims — the invariant is sampled here, never proved.
//
// The seed is the test's own and the world's generator is asserted unmoved, so the
// digests compared below are digests of a state this run did not stir. A second
// run from the same seed reproduces every trajectory and every digest, which is
// what makes a sweep evidence rather than an anecdote.
func TestStepKeepsEveryUnitOnACellOfItsOwnThroughASeededSweep(t *testing.T) {
	first := conSweepRun(t)
	if len(first.start) != conSweepUnits {
		t.Fatalf("fixture: the sweep drew %d units, want %d", len(first.start), conSweepUnits)
	}
	if first.rngMoved {
		t.Error("the world's own generator moved during the sweep: the randomness here is the test's, and a run that spent the world's would be asserting about a state it changed")
	}
	conRefused(t, "the sweep", first.start, first.cells, first.cmds)

	second := conSweepRun(t)
	for i, e := range first.start {
		if got := second.start[i]; got != e {
			t.Fatalf("the re-run drew unit %d at (%d,%d), want (%d,%d) — the same seed must draw the same world",
				e.ID, got.X, got.Y, e.X, e.Y)
		}
	}
	for tick := range first.cells {
		for _, e := range first.cells[tick] {
			if got := conFind(t, second.cells[tick], e.ID); got != e {
				t.Fatalf("tick %d: the re-run holds unit %d as %+v, want %+v", tick+1, e.ID, got, e)
			}
		}
		if got := second.digests[tick]; got != first.digests[tick] {
			t.Fatalf("tick %d: the re-run's digest is %#016x, want %#016x", tick+1, got, first.digests[tick])
		}
	}
}
