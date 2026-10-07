package sim

// What a group order costs a tick, measured as the two shapes that provoke it.
//
// The first is the owner's own case: several units selected together and sent to
// one distant cell, which is exactly what the front-end issues — one command per
// selected member, all naming the clicked cell. The second is the sealed goal: a
// free cell nothing can reach because units hold every way into it, which is the
// case a search cannot answer cheaply by failing early, because the cell it is
// asked about is perfectly enterable.
//
// Both are benchmarks, so `go test` compiles them and runs neither; the fixtures
// below are what `go test` runs, and they exist because a benchmark that measures
// a group standing still measures nothing. Each asserts the shape the benchmark
// claims to have — that the movers really move, or that the goal really cannot be
// reached — on an instance small enough to be free.
//
// Nothing here reads a game install: every world is built in this file out of
// bounds, positions and a command list.

import (
	"fmt"
	"testing"
	"time"
)

// gopTicks is the window the three shipped shapes are measured over. It is
// stallLimit, so a run that never advances is exactly the sixteen consecutive
// searches a unit spends before it gives up — the whole of what a refused order
// costs.
//
// gopLongTicks is the second window, and it exists because the two costs a
// two-tier search has are not the same cost. The tick an order arrives on pays
// one whole-map sweep per unit ordered; every tick after it pays a windowed
// search per unit and nothing else. Sixteen ticks average the first over the
// second sixteen ways; 256 ticks put the steady state where it can be read.
const gopTicks = stallLimit
const gopLongTicks = 256

// gopBlockWidth is how wide the block a group starts in is laid out. A group is
// a rectangle rather than a line because a line of movers files through one cell
// and a rectangle contends, which is what a selection box actually produces.
const gopBlockWidth = 10

// gopBlock is `units` cells laid out row-major from (x0, y0), which is where the
// group stands before it is ordered anywhere.
func gopBlock(x0, y0 int32, units int, firstID EntityID) []Entity {
	out := make([]Entity, 0, units)
	for k := 0; k < units; k++ {
		out = append(out, Entity{
			ID: firstID + EntityID(k),
			X:  x0 + int32(k%gopBlockWidth),
			Y:  y0 + int32(k/gopBlockWidth),
		})
	}
	return out
}

// gopOrder is one command per entity in ents, every one naming the same cell.
func gopOrder(ents []Entity, x, y int32) []Command {
	out := make([]Command, 0, len(ents))
	for _, e := range ents {
		out = append(out, Command{Entity: e.ID, X: x, Y: y})
	}
	return out
}

// gopGroup is the owner's case over a side x side map with no obstacle on it:
// `units` units standing in a block at the near corner, all ordered to one cell
// at the far one. The order is given on the first tick and never repeated, so
// every later tick is the search alone.
func gopGroup(side int32, units int) (*World, []Command, error) {
	b := Bounds{Width: side, Height: side}
	ents := gopBlock(2, 2, units, 1)
	w, err := NewWorld(1, b, ModeCanonical, nil, ents)
	return w, gopOrder(ents, side-3, side-3), err
}

// gopSealedGoal is the cell the sealed run is ordered onto, and gopSealedStart
// the corner of the movers' block, both on a side x side map.
func gopSealedGoal(side int32) cell  { return cell{x: side - 32, y: side / 2} }
func gopSealedStart(side int32) cell { return cell{x: 8, y: side/2 - 4} }

// gopSealed is the sealed goal: eight units standing on the eight cells around
// gopSealedGoal, holding no order of their own, and `movers` units ordered onto
// the goal from a block across the map.
//
// The goal itself is empty and unblocked, so it is ENTERABLE and no test of the
// goal cell can refuse the order. It is simply not reachable, and the only thing
// that can discover that is a sweep that runs out of frontier or out of
// generations. Every mover therefore spends a whole search on every tick, holds,
// and gives up on the sixteenth — which is the cost this fixture exists to size.
func gopSealed(side int32, movers int) (*World, []Command, error) {
	b := Bounds{Width: side, Height: side}
	g := gopSealedGoal(side)

	ents := make([]Entity, 0, movers+8)
	id := EntityID(1)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			ents = append(ents, Entity{ID: id, X: g.x + dx, Y: g.y + dy})
			id++
		}
	}
	start := gopSealedStart(side)
	group := gopBlock(start.x, start.y, movers, id)
	ents = append(ents, group...)

	w, err := NewWorld(1, b, ModeCanonical, nil, ents)
	return w, gopOrder(group, g.x, g.y), err
}

// gopHeldGoal is the cell the held run is ordered onto, on a side x side map.
func gopHeldGoal(side int32) cell { return cell{x: side - 3, y: side - 3} }

// gopHeld is the group ordered onto a cell that is TAKEN: `movers` units at the
// near corner, one unit standing on the far cell with no order of its own, and
// every mover sent to it.
//
// It is the ordinary end of gopGroup rather than a separate misfortune — the
// first member of a group arrives, and from that tick on every other member is
// asking about a cell it can never enter. The destination cell being unenterable
// is the whole difference from gopSealed, where the destination is free and only
// the way to it is shut.
func gopHeld(side int32, movers int) (*World, []Command, error) {
	b := Bounds{Width: side, Height: side}
	g := gopHeldGoal(side)
	ents := []Entity{{ID: 1, X: g.x, Y: g.y}}
	group := gopBlock(2, 2, movers, 2)
	ents = append(ents, group...)
	w, err := NewWorld(1, b, ModeCanonical, nil, ents)
	return w, gopOrder(group, g.x, g.y), err
}

// gopTerrainSealedGoal is the fifth shape's destination: a cell close to the
// movers that TERRAIN seals off, and gopTerrainSealedStart their block's corner.
//
// Near is the whole point. The old budget was max(5, D>>2) + D, so it was
// SMALLEST where the goal was nearest, and a refused search near at hand used to
// cost a few hundred labelled cells. The flat budget is the same full sweep
// wherever the goal lies, so this is the largest ratio the map admits — a far
// sealed goal shows about one, because the old budget already exceeded the map.
//
// It is sealed by TERRAIN and not by units. The shipped sealed shape is ringed
// by movers, which the far search cannot see: it routes to that goal and the
// group walks. Only terrain can stop a terrain-only search, and only a stopped
// search measures what a refusal costs.
func gopTerrainSealedGoal() cell  { return cell{x: 16, y: 4} }
func gopTerrainSealedStart() cell { return cell{x: 2, y: 2} }

// gopTerrainSealed is `movers` units in a block at the near corner of a
// side x side map, all ordered onto a nearby open cell walled in by blocking
// terrain.
//
// The goal is open, so nothing refuses the order before a sweep; it is
// unreachable, so every sweep runs to exhaustion; and it is near, so the budget
// this revision removed would have stopped that sweep early.
func gopTerrainSealed(side int32, movers int) (*World, []Command, error) {
	b := Bounds{Width: side, Height: side}
	g := gopTerrainSealedGoal()

	grid := make([]byte, side*side)
	for dy := int32(-1); dy <= 1; dy++ {
		for dx := int32(-1); dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				grid[(g.y+dy)*side+(g.x+dx)] = blockGround
			}
		}
	}

	start := gopTerrainSealedStart()
	ents := gopBlock(start.x, start.y, movers, 1)
	w, err := NewWorld(1, b, ModeCanonical, grid, ents)
	return w, gopOrder(ents, g.x, g.y), err
}

// gopRun advances w for the given number of ticks, applying cmds on the first
// and nothing after it, and reports the milliseconds one tick cost.
//
// The tick count is a parameter and everything else about this harness is not:
// the four shipped shapes keep their builders, their sizes and their group
// counts, so their figures stay comparable with the ones already recorded.
func gopRun(b *testing.B, ticks int, build func() (*World, []Command, error)) {
	b.Helper()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		w, cmds, err := build()
		if err != nil {
			b.Fatalf("NewWorld: %v", err)
		}
		b.StartTimer()
		for t := 0; t < ticks; t++ {
			if t == 0 {
				Step(w, cmds)
			} else {
				Step(w, nil)
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*ticks)/1e6, "ms/tick")
}

// BenchmarkAGroupOrderedAcrossTheMap is the owner's case at the two map sizes
// and the group sizes a selection box produces.
func BenchmarkAGroupOrderedAcrossTheMap(b *testing.B) {
	for _, side := range []int32{128, 256} {
		for _, units := range []int{1, 5, 10, 20, 40} {
			b.Run(fmt.Sprintf("%dx%d/%dunits", side, side, units), func(b *testing.B) {
				gopRun(b, gopTicks, func() (*World, []Command, error) { return gopGroup(side, units) })
			})
		}
	}
}

// BenchmarkASealedGroupOrderedOntoOnePoint is the second shape: eighty movers
// ordered onto a cell they cannot reach, every one of them paying a full sweep
// on every one of the sixteen ticks it takes to give up.
func BenchmarkASealedGroupOrderedOntoOnePoint(b *testing.B) {
	gopRun(b, gopTicks, func() (*World, []Command, error) { return gopSealed(128, 80) })
}

// BenchmarkAGroupOrderedOntoAHeldCell is the third: the same eighty movers, and
// a destination that is not sealed off but simply occupied.
func BenchmarkAGroupOrderedOntoAHeldCell(b *testing.B) {
	gopRun(b, gopTicks, func() (*World, []Command, error) { return gopHeld(128, 80) })
}

// BenchmarkAGroupOrderedAcrossTheMapOverALongRun is the same forty units on the
// same 256x256, measured over sixteen times as many ticks. It is the fourth
// shape and it exists to separate the order tick's whole-map sweeps from what a
// tick costs afterwards: divided over 256 ticks the sweep is a sixteenth of what
// it is divided over 16, so the two figures either side of that ratio say how
// much of a run's cost is the order and how much is the walk.
func BenchmarkAGroupOrderedAcrossTheMapOverALongRun(b *testing.B) {
	gopRun(b, gopLongTicks, func() (*World, []Command, error) { return gopGroup(256, 40) })
}

// gopRunWorst is gopRun with one figure more: the WORST single tick, beside the
// mean.
//
// The clock is read per tick rather than per run, which is the only way a worst
// tick can be reported at all. It reads no game install and decides nothing: the
// world is advanced by the same calls in the same order whether or not anything
// is timed.
func gopRunWorst(b *testing.B, ticks int, build func() (*World, []Command, error)) {
	b.Helper()
	var worst time.Duration
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		w, cmds, err := build()
		if err != nil {
			b.Fatalf("NewWorld: %v", err)
		}
		b.StartTimer()
		for t := 0; t < ticks; t++ {
			at := time.Now()
			if t == 0 {
				Step(w, cmds)
			} else {
				Step(w, nil)
			}
			if d := time.Since(at); d > worst {
				worst = d
			}
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*ticks)/1e6, "ms/tick")
	b.ReportMetric(float64(worst.Nanoseconds())/1e6, "ms/worst")
}

// BenchmarkAGroupOrderedOntoATerrainSealedCellNearby is the FIFTH shape, added
// beside the four rather than replacing one so that every figure already
// recorded stays comparable.
//
// Forty units on 256x256 ordered to a near cell terrain seals off: the case
// where the two budgets differ most, and the one this revision has to answer
// for.
func BenchmarkAGroupOrderedOntoATerrainSealedCellNearby(b *testing.B) {
	gopRunWorst(b, gopTicks, func() (*World, []Command, error) { return gopTerrainSealed(256, 40) })
}

// ---------------------------------------------------------------- the fixtures

// The three fixtures below assert what each benchmark's own world does, and two
// of the three say something different from what they said before the two-tier
// arrangement landed. That is the contract changing, not the fixtures drifting:
// a mover's route is now searched over TERRAIN, which is blind to the units
// standing on and around a destination — so a group ordered onto a sealed goal
// or onto a cell somebody is standing on walks there and crowds, where it used
// to stand still from the first tick and give up sixteen ticks later.

// TestTheGroupFixtureIsActuallyWalking is what makes the first benchmark a
// measurement of movement rather than of a standstill: on a small instance of
// the same builder, every unit advances over the window measured, none of them
// arrives inside it, and none of them loses its order.
//
// A unit may now stall for a tick or two on the way and that is not a jam: a
// group sent to one cell converges, and a near search blocked by a neighbour
// this tick is a near search that succeeds once the neighbour moves. What would
// make the benchmark measure a standstill is a unit that never advances at all,
// which is what the comparison at the end is for.
func TestTheGroupFixtureIsActuallyWalking(t *testing.T) {
	w, cmds, err := gopGroup(48, 12)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	before := w.Entities()

	for tick := 1; tick <= gopTicks; tick++ {
		if tick == 1 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}
		for _, e := range w.Entities() {
			if !e.HasTarget {
				t.Fatalf("tick %d: unit %d holds no order — it arrived or gave up inside the measured window",
					tick, e.ID)
			}
		}
	}

	after := w.Entities()
	for i := range after {
		if after[i].X == before[i].X && after[i].Y == before[i].Y {
			t.Errorf("unit %d is still at (%d,%d) after %d ticks", after[i].ID, after[i].X, after[i].Y, gopTicks)
		}
	}
}

// TestTheSealedFixtureIsSealedAndItsGoalIsEnterable is the second benchmark's
// claim about its own world: the goal is ENTERABLE, so no search refuses the
// order out of hand, and it is UNREACHABLE, so nothing ever stands on it.
//
// What the movers do in between is the change this story discloses. The far
// search reads terrain alone and the ring is made of units, so the goal is
// reachable as far as that search can tell: every mover is routed to it and
// walks. It is the NEAR search that meets the ring, and only once a mover
// arrives at it. Inside the window measured here they are still walking.
func TestTheSealedFixtureIsSealedAndItsGoalIsEnterable(t *testing.T) {
	const side = 64
	w, cmds, err := gopSealed(side, 12)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	g := gopSealedGoal(side)

	for _, e := range w.Entities() {
		if e.X == g.x && e.Y == g.y {
			t.Fatalf("unit %d stands on the goal (%d,%d) — this fixture would measure the early refusal",
				e.ID, g.x, g.y)
		}
	}

	before := w.Entities()
	walked := 0
	for tick := 1; tick <= gopTicks; tick++ {
		if tick == 1 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}
		for i, e := range w.Entities() {
			if e.X == g.x && e.Y == g.y {
				t.Fatalf("tick %d: unit %d reached the goal (%d,%d), which is meant to be unreachable",
					tick, e.ID, g.x, g.y)
			}
			// The eight units around the goal hold no order and are not part of
			// what is measured: they stand where they were put.
			if i < 8 && (e.X != before[i].X || e.Y != before[i].Y) {
				t.Fatalf("tick %d: the ring unit %d moved to (%d,%d)", tick, e.ID, e.X, e.Y)
			}
		}
	}

	after := w.Entities()
	for i := 8; i < len(after); i++ {
		if after[i].X != before[i].X || after[i].Y != before[i].Y {
			walked++
		}
	}
	if walked != len(cmds) {
		t.Errorf("%d of %d movers walked toward the sealed goal — under this contract every one of them does",
			walked, len(cmds))
	}
	if got := len(cmds); got != 12 {
		t.Errorf("the fixture ordered %d movers, want 12", got)
	}
}

// TestTheHeldFixtureIsHeldAndItsMoversWalkToIt is the third benchmark's claim,
// and the second of the two this story changes. The destination is occupied, so
// no mover can ever enter it — but the far search cannot see the unit standing
// there, so every mover is routed to it and walks until the near search meets
// the crowd. Where this fixture used to measure sixteen refused ticks and one
// give-up each, it now measures an ordinary walk.
func TestTheHeldFixtureIsHeldAndItsMoversWalkToIt(t *testing.T) {
	const side = 64
	w, cmds, err := gopHeld(side, 12)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	g := gopHeldGoal(side)

	before := w.Entities()
	if before[0].X != g.x || before[0].Y != g.y {
		t.Fatalf("the goal (%d,%d) is not held; unit %d is at (%d,%d)",
			g.x, g.y, before[0].ID, before[0].X, before[0].Y)
	}

	for tick := 1; tick <= gopTicks; tick++ {
		if tick == 1 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}
		got := w.Entities()
		if got[0] != before[0] {
			t.Fatalf("tick %d: the unit holding the goal is %+v, want it where it was put", tick, got[0])
		}
		for _, e := range got[1:] {
			if e.X == g.x && e.Y == g.y {
				t.Fatalf("tick %d: unit %d entered the held cell (%d,%d)", tick, e.ID, g.x, g.y)
			}
		}
	}

	after := w.Entities()
	walked := 0
	for i := 1; i < len(after); i++ {
		if after[i].X != before[i].X || after[i].Y != before[i].Y {
			walked++
		}
	}
	if walked != len(cmds) {
		t.Errorf("%d of %d movers walked toward the held cell — under this contract every one of them does",
			walked, len(cmds))
	}
}

// TestTheTerrainSealedFixtureIsUnreachableAndItsMoversWalkToTheSeal is the fifth
// benchmark's claim about its own world, and 0037 changes its second half the
// way it changed the third benchmark's above.
//
// One, unchanged: the goal is OPEN and UNREACHABLE. Open, so nothing refuses the
// order before a sweep and the benchmark measures a sweep rather than a refusal;
// and walled in by terrain, so every sweep runs to exhaustion. That is what
// makes this shape the one the flat budget costs the most.
//
// Two: the exhausted sweep no longer ends the order — it SETTLES for the
// labelled cell nearest the seal, so every mover walks there. What still
// lands on ONE tick is the sweep itself, and that is what this shape's cost
// was ever about: the order is re-pointed at the cell the search settled
// for, so from the second tick on every mover holds a route that serves and
// no mover sweeps the map again. A mover left pointing at the sealed cell
// would buy that sweep every tick for as long as it walked, which is the
// regression this asserts against.
func TestTheTerrainSealedFixtureIsUnreachableAndItsMoversWalkToTheSeal(t *testing.T) {
	const side = 64
	w, cmds, err := gopTerrainSealed(side, 12)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	g := gopTerrainSealedGoal()

	// Open, and ringed by terrain that is not.
	if !w.terrainOpen(DomainGround, g.x, g.y) {
		t.Fatalf("the goal (%d,%d) blocks ground, so it is refused before any sweep", g.x, g.y)
	}
	for dy := int32(-1); dy <= 1; dy++ {
		for dx := int32(-1); dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if w.terrainOpen(DomainGround, g.x+dx, g.y+dy) {
				t.Fatalf("(%d,%d) is open, so the goal is not sealed", g.x+dx, g.y+dy)
			}
		}
	}
	// And no unit is standing on it, which would measure the early refusal.
	for _, e := range w.Entities() {
		if e.X == g.x && e.Y == g.y {
			t.Fatalf("unit %d stands on the goal", e.ID)
		}
	}
	// Near: the ratio this shape exists for comes from the goal being close.
	start := gopTerrainSealedStart()
	if d := maxAbs(g.x-start.x, g.y-start.y); d > 32 {
		t.Errorf("the goal is %d from the block's corner, and this shape is the NEAR one", d)
	}

	before := w.Entities()
	Step(w, cmds)
	// The sweep ran once, on this tick, and every mover that still holds an
	// order holds one pointing somewhere it can reach. That is the whole of the
	// cost claim: an order still naming the sealed cell is an order whose route
	// can never serve, and so a whole-map sweep on every tick that follows.
	for _, e := range w.Entities() {
		if e.HasTarget && e.TargetX == g.x && e.TargetY == g.y {
			t.Errorf("unit %d still points at the sealed cell (%d,%d) after the sweep, so its route "+
				"cannot serve and it will sweep the map again next tick", e.ID, g.x, g.y)
		}
	}

	for tick := 2; tick <= gopTicks; tick++ {
		Step(w, nil)
	}
	after := w.Entities()
	walked := 0
	for i, e := range after {
		if e.X == g.x && e.Y == g.y {
			t.Errorf("unit %d stands on the sealed cell (%d,%d), which nothing can enter", e.ID, g.x, g.y)
		}
		if e.HasTarget && e.TargetX == g.x && e.TargetY == g.y {
			t.Errorf("unit %d points at the sealed cell at the end of the window", e.ID)
		}
		if e.X != before[i].X || e.Y != before[i].Y {
			walked++
		}
	}
	if walked != len(cmds) {
		t.Errorf("%d of %d movers walked toward the seal — under this contract every one of them does",
			walked, len(cmds))
	}
	if got := len(cmds); got != 12 {
		t.Errorf("the fixture ordered %d movers, want 12", got)
	}
}

// TestAFullMapGroupOrderHoldsARouteOfTheMapsDiagonal fixes the number the path
// overlay's own cost measurement is built on (0037 AC-13's benchmark, pkg/ui).
//
// The overlay strokes one leg per route cell, so what a frame pays for is the
// SUM of the routes a selection holds — and the length of one is not a figure to
// assume. It is the whole diagonal here: gopGroup starts its block at (2,2) and
// orders it to (side-3, side-3), so a mover on the block's own corner walks 251
// cells on a 256x256 map, and nothing shorter is the worst case.
//
// It runs one tick, because a route exists from the tick the order is taken and
// shortens from the next one.
func TestAFullMapGroupOrderHoldsARouteOfTheMapsDiagonal(t *testing.T) {
	const side, units = 256, 40
	w, cmds, err := gopGroup(side, units)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	Step(w, cmds)

	longest, total, holding := 0, 0, 0
	for k := 0; k < units; k++ {
		r := w.Route(EntityID(1 + k))
		if len(r) == 0 {
			continue
		}
		holding++
		total += len(r)
		if len(r) > longest {
			longest = len(r)
		}
	}
	if holding != units {
		t.Fatalf("%d of %d movers hold a route; the overlay's cost is measured over all of them", holding, units)
	}
	if want := side - 5; longest != want {
		t.Errorf("the longest route is %d cells, want %d — the diagonal from (2,2) to (%d,%d)",
			longest, want, side-3, side-3)
	}
	// The quantity the overlay actually pays for, stated so a change to it is a
	// test failure and not a silent drift in a benchmark's premise.
	if total < units*(side-10) {
		t.Errorf("%d route cells over %d movers; the group is not walking the map's own diagonal", total, units)
	}
}
