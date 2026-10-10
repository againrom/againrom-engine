package sim

// The indexed predicate against the one it replaced.
//
// enterable used to walk the entity slice on every call. It now reads a count off
// the tick's scratch, and "the two answer the same thing" is the whole of what
// makes that a speed-up rather than a change. So the walk is kept HERE, in a test
// file, as a reference implementation nothing in production calls, and the two are
// compared cell for cell.
//
// Two things make the comparison worth running. It covers every unit as the asker,
// not one: the predicate takes an identity, and an index that forgot whose turn it
// was would still agree for whichever unit happened to be first. And it covers
// MID-TICK states — units walked one at a time, exactly as Step resolves them —
// because the contract is that the predicate answers over the entities as they
// stand at the moment it is asked, not over the entities as they stood at the top
// of the tick. An index built once and read all tick passes a comparison that only
// ever looks at a fresh world.
//
// The corpus is generated from the package's own generator, so it is the same
// corpus on every machine and every run, and it is asserted to contain the three
// awkward cases rather than assumed to: a cell two units stand on, a unit off the
// map, and a cell the grid blocks.

import "testing"

// entNaive is an independent linear implementation of the footprint relation:
// every covered cell must be in bounds and terrain-open, and no other counted
// same-layer actor may cover one of those cells.
func (w *World) entNaive(self int, x, y int32) bool {
	e := w.entities[self]
	side := int32(e.TokenSize)
	if side == 0 {
		side = 1
	}
	for dy := int32(0); dy < side; dy++ {
		for dx := int32(0); dx < side; dx++ {
			i, ok := w.cellIndex(x+dx, y+dy)
			if !ok || w.grid[i]&e.Domain.blocks() != 0 {
				return false
			}
		}
	}
	for j := range w.entities {
		if j == self || !counted(&w.entities[j]) || w.entities[j].Domain.layer() != e.Domain.layer() {
			continue
		}
		o := w.entities[j]
		otherSide := int32(o.TokenSize)
		if otherSide == 0 {
			otherSide = 1
		}
		if x < o.X+otherSide && o.X < x+side && y < o.Y+otherSide && o.Y < y+side {
			return false
		}
	}
	return true
}

// entMargin is how far past the bounds the comparison sweeps. A search asks about
// cells its start is next to, and a start may be off the map, so the two
// predicates have to agree about cells that have no slot at all.
const entMargin = 2

// entAgree compares the two over every unit and every cell of the swept
// rectangle, and reports the first disagreement with enough of the state to read
// it — a coordinate alone would say nothing about which unit was asking.
func entAgree(t *testing.T, what string, w *World, s *routeScratch) {
	t.Helper()
	for self := range w.entities {
		for y := int32(-entMargin); y < w.bounds.Height+entMargin; y++ {
			for x := int32(-entMargin); x < w.bounds.Width+entMargin; x++ {
				got, want := w.enterable(s, self, x, y), w.entNaive(self, x, y)
				if got != want {
					t.Fatalf("%s: unit %d at (%d,%d) is told (%d,%d) is enterable=%v, the walk says %v\n entities %+v",
						what, w.entities[self].ID, w.entities[self].X, w.entities[self].Y,
						x, y, got, want, w.entities)
				}
			}
		}
	}
}

// entCorpus is one generated world: bounds, a grid and a handful of units placed
// with no regard for whether the cell is free, on the map, or passable.
func entCorpus(t *testing.T, r *rng) *World {
	t.Helper()
	b := Bounds{Width: 1 + int32(r.next()%9), Height: 1 + int32(r.next()%7)}

	grid := make([]byte, gridCells(b))
	for i := range grid {
		switch r.next() % 4 {
		case 0:
			grid[i] = blockGround
		case 1:
			grid[i] = blockAir
		}
	}

	ents := make([]Entity, 1+int(r.next()%6))
	for i := range ents {
		ents[i] = Entity{
			ID:        EntityID(i + 1),
			X:         int32(r.next()%uint64(b.Width+2*entMargin)) - entMargin,
			Y:         int32(r.next()%uint64(b.Height+2*entMargin)) - entMargin,
			Domain:    Domain(r.next() % 3),
			TokenSize: uint8(r.next() % 4),
		}
	}

	w, err := NewWorld(r.next(), b, ModeCanonical, grid, ents)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

// entShapes counts what a corpus actually produced, so the comparison's reach is
// asserted and not hoped for.
type entShapes struct {
	shared, offMap, blocked, large int
}

func (sh *entShapes) note(w *World) {
	for i := range w.entities {
		if w.entities[i].TokenSize > 1 {
			sh.large++
		}
		if _, ok := w.cellIndex(w.entities[i].X, w.entities[i].Y); !ok {
			sh.offMap++
			continue
		}
		for j := range w.entities {
			if j != i && w.entities[j].X == w.entities[i].X && w.entities[j].Y == w.entities[i].Y {
				sh.shared++
			}
		}
	}
	for _, c := range w.grid {
		if c&blockGround != 0 {
			sh.blocked++
		}
	}
}

// TestTheIndexedPredicateAnswersWhatTheWalkAnswered is SC-11's oracle half. Each
// generated world is compared as built, and then again after each unit is moved to
// a cell chosen with no regard for the rules — which is what makes the comparison
// cover a half-resolved tick and not just a fresh one.
func TestTheIndexedPredicateAnswersWhatTheWalkAnswered(t *testing.T) {
	r := &rng{state: 0x5eed}
	var sh entShapes

	for run := 0; run < 400; run++ {
		w := entCorpus(t, r)
		sh.note(w)
		s := newRouteScratch(w)
		entAgree(t, "as built", w, s)

		// The mid-tick walk: units resolved in ascending id, each one moved and
		// the counts moved with it, exactly as Step does it.
		for i := range w.entities {
			e := &w.entities[i]
			from := cell{x: e.X, y: e.Y}
			to := cell{
				x: int32(r.next()%uint64(w.bounds.Width+2*entMargin)) - entMargin,
				y: int32(r.next()%uint64(w.bounds.Height+2*entMargin)) - entMargin,
			}
			e.X, e.Y = to.x, to.y
			s.moved(w, *e, from, to)
			sh.note(w)
			entAgree(t, "mid-tick", w, s)
		}
	}

	if sh.shared == 0 || sh.offMap == 0 || sh.blocked == 0 || sh.large == 0 {
		t.Errorf("the corpus held %d shared cells, %d units off the map, %d blocked cells and %d large actors — "+
			"a comparison that never met one of those has not measured it", sh.shared, sh.offMap, sh.blocked, sh.large)
	}
}

// TestAReOccupiedScratchAnswersForTheWorldItWasPointedAt is the other half of the
// scratch's occupancy contract: the counts belong to one world's entities, and
// pointing the scratch at a second world replaces them rather than adding to them.
// Without that a reused scratch would answer for a union of two worlds, which is
// no world at all.
func TestAReOccupiedScratchAnswersForTheWorldItWasPointedAt(t *testing.T) {
	b := Bounds{Width: 4, Height: 4}
	first := mustWorld(t, 1, b, []Entity{{ID: 1, X: 0, Y: 0}, {ID: 2, X: 1, Y: 1}})
	second := mustWorld(t, 1, b, []Entity{{ID: 1, X: 3, Y: 3}})

	s := newRouteScratch(first)
	entAgree(t, "the first world", first, s)

	s.occupy(second)
	entAgree(t, "the second world", second, s)
	if !second.enterable(s, 0, 1, 1) {
		t.Error("(1,1) is not enterable in the second world, where nothing stands on it")
	}

	s.occupy(first)
	entAgree(t, "the first world again", first, s)
}
