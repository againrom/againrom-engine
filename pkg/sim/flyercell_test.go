package sim

import "testing"

// flyerCellFrames orders each (id, x, y) and records every entity's cell after
// each tick until nothing holds an order.
func flyerCellFrames(t *testing.T, w *World, orders ...Command) [][]Entity {
	t.Helper()
	Step(w, orders)
	frames := [][]Entity{w.Entities()}
	for i := 0; i < 96; i++ {
		busy := false
		for _, e := range w.Entities() {
			if e.HasTarget {
				busy = true
			}
		}
		if !busy {
			return frames
		}
		Step(w, nil)
		frames = append(frames, w.Entities())
	}
	t.Fatalf("nothing settled in 96 ticks")
	return nil
}

// sharedCellFrame is the first frame in which entities a and b overlap, or -1.
func sharedCellFrame(frames [][]Entity, a, b int) int {
	for k, f := range frames {
		if footprintsOverlap(f[a].X, f[a].Y, f[a].TokenSize, f[b].X, f[b].Y, f[b].TokenSize) {
			return k
		}
	}
	return -1
}

// TestTwoFlyersOrderedOntoOneCellEndOnTwoCells proves two flying units are
// refused one cell as two ground units are: both orders name one cell, and the
// pair ends on two cells and never shares one on the way. A ground pair is the
// control, so the flyer result is the ground result.
func TestTwoFlyersOrderedOntoOneCellEndOnTwoCells(t *testing.T) {
	for _, tc := range []struct {
		what   string
		domain Domain
		size   uint8
	}{
		{"flyers", DomainAir, 0},
		{"ground units", DomainGround, 0},
		{"2x2 flyers", DomainAir, 2},
		{"3x3 flyers", DomainAir, 3},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 14, Height: 9}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 0, Y: 1, Domain: tc.domain, TokenSize: tc.size},
			{ID: 2, X: 0, Y: 6, Domain: tc.domain, TokenSize: tc.size},
		})
		frames := flyerCellFrames(t, w,
			Command{Entity: 1, X: 10, Y: 3}, Command{Entity: 2, X: 10, Y: 3})
		if k := sharedCellFrame(frames, 0, 1); k >= 0 {
			t.Errorf("%s shared a cell after tick %d: (%d,%d) and (%d,%d)", tc.what, k+1,
				frames[k][0].X, frames[k][0].Y, frames[k][1].X, frames[k][1].Y)
		}
		last := frames[len(frames)-1]
		if last[0].X == last[1].X && last[0].Y == last[1].Y {
			t.Errorf("%s ended on one cell (%d,%d)", tc.what, last[0].X, last[0].Y)
		}
	}
}

// TestAFlyerAndAGroundUnitMayShareACell and the ghost case prove the rule is
// per layer: a flyer contends with flyers only, a ghost with ground units only.
func TestAFlyerAndAGroundUnitMayShareACell(t *testing.T) {
	for _, tc := range []struct {
		what  string
		other Domain
	}{
		{"a ground unit", DomainGround},
		{"a ghost", DomainGhost},
	} {
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
			{ID: 1, X: 0, Y: 2, Domain: DomainAir},
			{ID: 2, X: 6, Y: 2, Domain: tc.other},
		})
		frames := flyerCellFrames(t, w, Command{Entity: 1, X: 6, Y: 2})
		last := frames[len(frames)-1][0]
		if last.X != 6 || last.Y != 2 {
			t.Errorf("a flyer ordered onto %s's cell ended at (%d,%d), want (6,2)", tc.what, last.X, last.Y)
		}
	}
}

// TestAGhostAndAGroundUnitStillContend keeps the ghost's own layer: it shares
// the ground layer, so a ghost never ends on a ground unit's cell.
func TestAGhostAndAGroundUnitStillContend(t *testing.T) {
	w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 0, Y: 2, Domain: DomainGhost},
		{ID: 2, X: 6, Y: 2, Domain: DomainGround},
	})
	frames := flyerCellFrames(t, w, Command{Entity: 1, X: 6, Y: 2})
	if k := sharedCellFrame(frames, 0, 1); k >= 0 {
		t.Errorf("a ghost stood on a ground unit's cell after tick %d", k+1)
	}
}

// TestAFlyerOrderedOntoARestingFlyerEndsBeside: a flyer ordered onto the cell a
// peer rests on never stands on it and ends on a neighbouring cell.
func TestAFlyerOrderedOntoARestingFlyerEndsBeside(t *testing.T) {
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, mode, nil, []Entity{
			{ID: 1, X: 0, Y: 2, Domain: DomainAir},
			{ID: 2, X: 6, Y: 2, Domain: DomainAir},
		})
		frames := flyerCellFrames(t, w, Command{Entity: 1, X: 6, Y: 2})
		if k := sharedCellFrame(frames, 0, 1); k >= 0 {
			t.Errorf("mode %v: the flyer stood on the resting flyer's cell after tick %d", mode, k+1)
		}
	}
	w := mustWorldGrid(t, 1, Bounds{Width: 9, Height: 5}, ModeCanonical, nil, []Entity{
		{ID: 1, X: 0, Y: 2, Domain: DomainAir},
		{ID: 2, X: 6, Y: 2, Domain: DomainAir},
	})
	frames := flyerCellFrames(t, w, Command{Entity: 1, X: 6, Y: 2})
	last := frames[len(frames)-1][0]
	if d := (cell{last.X, last.Y}).chebyshevTo(cell{6, 2}); d != 1 {
		t.Errorf("the flyer ended at (%d,%d), %d cells from the occupied cell, want a neighbour", last.X, last.Y, d)
	}
}
