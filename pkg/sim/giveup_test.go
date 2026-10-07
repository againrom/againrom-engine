package sim

// How an order ends when it cannot be walked, and the two ways that are not the
// same way.
//
// A FAR search reads the terrain and the bounds, and neither changes while a
// world is advanced; a unit whose far search fails has not moved either, so a
// second far search from the same cell is the same search. The order is over in
// the tick that finds it. A NEAR search reads occupancy, which changes every
// tick, so its failure is a delay: the unit keeps everything, counts the tick,
// and gives up only when the count runs out.
//
// Both files' worlds are built here out of bounds, positions and a command list;
// nothing reads a game install.

import "testing"

// ------------------------------------------------- AC-2

// TestAnUnservableOrderIsClearedOnTheFirstTick is AC-2, and 0045 wrote it
// against a tree where the two modes answered it alike. They no longer do: the
// wave SETTLES for the labelled cell nearest what was asked for, so under the
// canonical mode each of these three orders is walked as far as it goes and only
// the optimised mode still ends it where it stands (0037 AC-1, AC-4).
//
// Three orders no exact route serves — a wall with no gap, a target off the map,
// and a target on a cell that blocks ground — each advanced sixteen ticks, which
// is the whole of what the give-up would have cost. Under the optimised mode
// each ends on the first of them; under the canonical one each mover walks to
// the cell named beside its fixture, arrives, and is idle from then on.
//
// Sixteen ticks and not one: what is being measured is that the ticks after the
// order has resolved are uneventful, so a rule that cleared the order LATER — or
// one that let a settled mover set off again — would fail here rather than pass
// by resolving it eventually.
func TestAnUnservableOrderIsClearedOnTheFirstTick(t *testing.T) {
	b := Bounds{Width: 5, Height: 5}
	walled := func() []byte {
		g := make([]byte, 25)
		for y := int32(0); y < 5; y++ {
			g[y*5+2] = blockGround
		}
		return g
	}
	blocked := func() []byte {
		g := make([]byte, 25)
		g[2*5+4] = blockGround
		return g
	}

	for _, tc := range []struct {
		what string
		grid []byte
		// target is the cell ordered; settles is where the canonical mode's
		// mover comes to rest, which is the labelled cell its own failed wave
		// settled for. The optimised mode never leaves (0, 2).
		target  cell
		settles cell
	}{
		{"a wall with no gap in it", walled(), cell{4, 2}, cell{1, 2}},
		{"a target off the map", nil, cell{9, 2}, cell{4, 2}},
		{"a target on a cell that blocks ground", blocked(), cell{4, 2}, cell{3, 2}},
	} {
		for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
			w := mustWorldGrid(t, 1, b, mode, tc.grid, []Entity{{ID: 1, X: 0, Y: 2}})

			// Where this mover ends up, and by when. The optimised mode ends the
			// order at once and never moves; the canonical one walks, so it is
			// settled by the time it could have stepped that far.
			rest := cell{0, 2}
			if mode == ModeCanonical {
				rest = tc.settles
			}
			arrivedBy := int(rest.chebyshevTo(cell{0, 2})) + 1

			for tick := 1; tick <= stallLimit; tick++ {
				if tick == 1 {
					Step(w, []Command{{Entity: 1, X: tc.target.x, Y: tc.target.y}})
				} else {
					Step(w, nil)
				}
				if tick < arrivedBy {
					continue
				}

				// THE FACING IS DROPPED FROM THIS COMPARISON and asserted nowhere
				// here, for the reason wall_test.go's settled case drops it: it is
				// not residue, it depends on which cell each MODE walked in from,
				// and that choice is this case's subject. Every other field is
				// checked whole, which is what makes this a no-residue case; the
				// facing is asserted against the step that produced it in
				// arrival_test.go, over all eight directions.
				got := w.Entities()[0]
				got.Facing = 0
				got.DesiredFacing = 0
				if want := (Entity{ID: 1, X: rest.x, Y: rest.y, ActorState: actorStateGuard, Reach: 1,
					PostX: 0, PostY: 2}); got != want {
					t.Fatalf("%s, mode %d, tick %d: the unit is %+v, want %+v — settled, its target "+
						"cleared and its stall at zero", tc.what, mode, tick, got, want)
				}
				if len(w.routes[0]) != 0 {
					t.Fatalf("%s, mode %d, tick %d: the unit holds the route %s and no target",
						tc.what, mode, tick, fmtRoute(w.routes[0]))
				}
				// A world this path produced is one its own form accepts: no
				// residue anywhere means no refusal on the way back in.
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatalf("%s: MarshalBinary: %v", tc.what, err)
				}
				var back World
				if err := back.UnmarshalBinary(form); err != nil {
					t.Fatalf("%s, mode %d, tick %d: the world does not decode: %v", tc.what, mode, tick, err)
				}
			}
		}
	}
}

// ------------------------------------------------- AC-3

// guGoal is the cell the group below is ordered onto, and guSide the map it
// stands on. The goal is held by a unit with no order of its own, so it can
// never be entered — and the far search cannot see that, which is why the group
// walks the whole way there before anything stops it.
const guSide int32 = 24

var guGoal = cell{12, 12}

// guWorld is eight movers in a block at the near corner, and the unit holding
// the goal.
func guWorld(t *testing.T) (*World, []Command) {
	t.Helper()
	ents := []Entity{{ID: 1, X: guGoal.x, Y: guGoal.y}}
	cmds := make([]Command, 0, 8)
	for k := 0; k < 8; k++ {
		id := EntityID(2 + k)
		ents = append(ents, Entity{ID: id, X: 2 + int32(k%4), Y: 2 + int32(k/4)})
		cmds = append(cmds, Command{Entity: id, X: guGoal.x, Y: guGoal.y})
	}
	return mustWorld(t, 1, Bounds{Width: guSide, Height: guSide}, ents), cmds
}

// guChebyshev is the distance from (x,y) to the goal, in the metric a step
// covers one of per tick.
func guChebyshev(x, y int32) int32 {
	dx, dy := x-guGoal.x, y-guGoal.y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dy > dx {
		return dy
	}
	return dx
}

// TestAGroupOrderedOntoAHeldCellWalksToItCrowdsAndGivesUp is AC-3, and it is one
// of the two behaviours this story changes on purpose. Every mover is routed to
// the held cell by a search that reads terrain alone, so every one of them walks
// there; what stops each of them is the near search meeting the crowd, and only
// once it is standing in it.
//
// Four things are asserted, and the first three are what make the fourth mean
// something: every mover advances, none ever stands on the goal, every mover
// that gives up does so standing where no cell closer to the goal is free, and
// each loses its target AND its route on its sixteenth consecutive stalled tick.
//
// "Can get no closer" is asserted as WHY a mover stops rather than as a claim
// about the cells beside it, and the difference is a measured property of this
// contract. A near search aims at the sub-goal its own route yields and at
// nothing else, so a mover whose sub-goal is taken by the crowd ahead of it
// stalls where it stands — which may be three or four cells short of the nearest
// free cell, since substitute destinations are a non-goal here and a unit walks
// as near as its route takes it. What is asserted instead is that the cell it was
// aiming at was held by another unit on the tick it gave up: it was stopped by
// the crowd and not by anything else.
func TestAGroupOrderedOntoAHeldCellWalksToItCrowdsAndGivesUp(t *testing.T) {
	w, cmds := guWorld(t)
	start := w.Entities()
	held := start[0]

	advanced := make(map[EntityID]bool)
	gaveUp := make(map[EntityID]int)
	prev := start

	for tick := 1; tick <= 80; tick++ {
		// What each mover is aiming at, and where everything stands, as the tick
		// opens: the give-up below is explained against the state the tick was
		// resolved in and not against the one it left.
		subs := make([]cell, len(prev))
		aiming := make([]bool, len(prev))
		taken := make(map[cell]bool, len(prev))
		for i := range prev {
			taken[cell{prev[i].X, prev[i].Y}] = true
			subs[i], aiming[i] = w.subGoal(newRouteScratch(w), i)
		}

		if tick == 1 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}
		now := w.Entities()

		if now[0] != held {
			t.Fatalf("tick %d: the unit holding the goal is %+v, want it where it was put", tick, now[0])
		}

		for i := 1; i < len(now); i++ {
			e, was := now[i], prev[i]
			if e.X == guGoal.x && e.Y == guGoal.y {
				t.Fatalf("tick %d: unit %d entered the held cell (%d,%d)", tick, e.ID, guGoal.x, guGoal.y)
			}
			if e.X != was.X || e.Y != was.Y {
				advanced[e.ID] = true
				continue
			}
			if was.HasTarget && !e.HasTarget {
				if !aiming[i] {
					t.Errorf("tick %d: unit %d gave up holding no usable route", tick, e.ID)
				} else if !taken[subs[i]] {
					t.Errorf("tick %d: unit %d gave up aiming at (%d,%d), which no unit was standing on",
						tick, e.ID, subs[i].x, subs[i].y)
				}
				if was.Stall != stallLimit-1 {
					t.Errorf("tick %d: unit %d lost its order after %d stalled tick(s), want %d",
						tick, e.ID, was.Stall, stallLimit-1)
				}
				if e.Stall != 0 || e.TargetX != 0 || e.TargetY != 0 {
					t.Errorf("tick %d: unit %d gave up as %+v, want no residue", tick, e.ID, e)
				}
				if len(w.routes[i]) != 0 {
					t.Errorf("tick %d: unit %d gave up still holding the route %s",
						tick, e.ID, fmtRoute(w.routes[i]))
				}
				gaveUp[e.ID] = tick
			}
		}
		prev = now
	}

	for _, e := range start[1:] {
		if !advanced[e.ID] {
			t.Errorf("unit %d never advanced, so it did not walk toward the goal at all", e.ID)
		}
		if gaveUp[e.ID] == 0 {
			t.Errorf("unit %d still holds its order after 80 ticks", e.ID)
		}
	}
	// And the crowd really did form: every mover walked the ten cells to the goal
	// and stopped at the press around it, rather than anywhere on the way.
	for i, e := range w.Entities() {
		if i == 0 {
			continue
		}
		if d := guChebyshev(e.X, e.Y); d > 5 {
			t.Errorf("unit %d ended %d cell(s) from the goal, so it did not crowd it", e.ID, d)
		}
	}
}
