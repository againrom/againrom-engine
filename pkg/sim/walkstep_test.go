package sim

import "testing"

// The walk step's cell rules, each under both games: the logical cell is the
// departure until the crossing call (MOVE-STEP-010, MOVE-088); a step holds
// the cell it leaves until it lands and the cell it enters from its start
// (MOVE-CLAIM-007); a listed near step is searched again only once the
// dynamic counter passes 32, and a stored route once more than sixteen near
// searches have served it (MOVE-130, MOVE-REFRESH-012); a participant's unit
// waits behind a walker and goes round a standing unit (MOVE-WAIT-008,
// MOVE-132). Only fields the earlier form also has are read here.

func walkStepWorld(t *testing.T, second bool, ents ...Entity) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{Width: 60, Height: 40}, ents)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

func walkStepGames(t *testing.T, run func(t *testing.T, second bool)) {
	for _, second := range []bool{false, true} {
		name := "first"
		if second {
			name = "second"
		}
		t.Run(name, func(t *testing.T) { run(t, second) })
	}
}

func walkStepUnit(id EntityID, x, y int32, speed int32) Entity {
	return Entity{ID: id, Owner: SelfSlot, X: x, Y: y, HP: 100, MaxHP: 100, Speed: speed, RotationSpeed: 255, Facing: 64}
}

func TestAStepCommitsItsCellOnTheCrossingCall(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		for _, tc := range []struct {
			name string
			to   CellPoint
			axis func(NativeStride) int8
		}{
			{"east", CellPoint{X: 6, Y: 5}, func(s NativeStride) int8 { return s.StepX }},
			{"north-east", CellPoint{X: 6, Y: 4}, func(s NativeStride) int8 { return s.StepY }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				w := walkStepWorld(t, second, walkStepUnit(1, 5, 5, 16))
				Step(w, []Command{MoveTo(1, tc.to)})
				for k := 0; k < 8 && !w.entities[0].Stride.Present; k++ {
					Step(w, nil)
				}
				e := w.entities[0]
				if !e.Stride.Present || e.Transit == 0 || e.Stride.ToX != tc.to.X || e.Stride.ToY != tc.to.Y {
					t.Fatalf("the step retained no stride to %+v: %+v", tc.to, e.Stride)
				}
				call := crossingCall(tc.axis(e.Stride))
				if call < 2 {
					t.Fatalf("stride %+v crosses on its start call", e.Stride)
				}
				for c := 1; ; c++ {
					e = w.entities[0]
					want := cell{x: 5, y: 5}
					if c >= call {
						want = cell{x: tc.to.X, y: tc.to.Y}
					}
					if e.X != want.x || e.Y != want.y {
						t.Fatalf("call %d (crossing call %d): logical cell (%d,%d), want (%d,%d)", c, call, e.X, e.Y, want.x, want.y)
					}
					if e.Transit == 0 {
						break
					}
					Step(w, nil)
				}
			})
		}
	})
}

func TestAStepHoldsTheCellItLeavesUntilItLands(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		w := walkStepWorld(t, second, walkStepUnit(1, 5, 5, 16), walkStepUnit(2, 4, 5, 16), walkStepUnit(3, 7, 5, 16))
		Step(w, []Command{MoveTo(1, CellPoint{X: 6, Y: 5}), MoveTo(2, CellPoint{X: 5, Y: 5}), MoveTo(3, CellPoint{X: 6, Y: 5})})
		entered := func(e Entity, c cell) bool {
			return e.X == c.x && e.Y == c.y || e.Transit > 0 && e.Stride.Present && e.Stride.ToX == c.x && e.Stride.ToY == c.y
		}
		landed := false
		for tick := 1; tick <= 80; tick++ {
			a, b, c := w.entities[0], w.entities[1], w.entities[2]
			if a.Transit > 0 {
				if entered(b, cell{x: 5, y: 5}) {
					t.Fatalf("tick %d: unit 2 entered the cell unit 1 is leaving: %+v", tick, b)
				}
				if entered(c, cell{x: 6, y: 5}) {
					t.Fatalf("tick %d: unit 3 entered the cell unit 1 is entering: %+v", tick, c)
				}
			} else {
				landed = true
			}
			if landed && b.X == 5 && b.Y == 5 && b.Transit == 0 {
				return
			}
			Step(w, nil)
		}
		t.Fatalf("unit 2 never reached the cell unit 1 left: %+v", w.entities[1])
	})
}

// A centred unit whose listed step is held stays on its cell while the
// dynamic counter is 32 or less and searches again on the pass it reaches 33.
// The unit lists its step east while it still faces north, so a standing unit
// placed on that cell during the turn holds the listed step; the counter
// counts the walk passes after the turn.
func TestABlockedListedStepIsSearchedAgainOnlyPastTheDynamicCounter(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		a := walkStepUnit(1, 5, 5, 16)
		a.Facing, a.RotationSpeed = 0, 8
		w := walkStepWorld(t, second, a, walkStepUnit(2, 20, 20, 16))
		Step(w, []Command{MoveTo(1, CellPoint{X: 10, Y: 5})})
		if e := w.entities[0]; e.X != 5 || e.Y != 5 || e.Transit != 0 || e.Facing == 0 || e.Facing == 64 {
			t.Fatalf("the fixture is not turning toward its first step: %+v", e)
		}
		w.entities[1].X, w.entities[1].Y = 6, 5
		for k := 0; k < 16 && w.entities[0].Facing != 64; k++ {
			Step(w, nil)
		}
		for pass := 1; pass <= 32; pass++ {
			Step(w, nil)
			if e := w.entities[0]; e.X != 5 || e.Y != 5 || e.Transit != 0 || e.Facing != 64 {
				t.Fatalf("counted pass %d: the unit left its listed step before the counter passed 32: %+v", pass, e)
			}
		}
		Step(w, nil)
		if f := w.entities[0].Facing; f == 64 {
			t.Fatal("the pass the counter reaches 33 did not search round the standing unit")
		}
		for tick := 0; tick <= 200; tick++ {
			if e := w.entities[0]; e.X == 10 && e.Y == 5 {
				return
			}
			Step(w, nil)
		}
		t.Fatalf("the unit never reached its goal round the standing unit: %+v", w.entities[0])
	})
}

// A stored route serves sixteen near searches; the seventeenth makes the next
// centred pass search the far route again. The stored route here runs south
// before it turns north to the goal, which a fresh far search does not.
func TestAStoredRouteIsSearchedAgainAfterSixteenNearSearches(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		a := walkStepUnit(1, 2, 5, 0)
		a.HasTarget, a.TargetX, a.TargetY, a.Facing = true, 52, 5, 96
		w := walkStepWorld(t, second, a)
		var route []cell
		for k := int32(1); k <= 25; k++ {
			route = append(route, cell{x: 2 + k, y: 5 + k})
		}
		for k := int32(1); k <= 25; k++ {
			route = append(route, cell{x: 27 + k, y: 30 - k})
		}
		w.routes[0] = route
		for tick := 1; tick <= 17; tick++ {
			Step(w, nil)
			if e := w.entities[0]; e.X != route[tick-1].x || e.Y != route[tick-1].y {
				t.Fatalf("tick %d: the unit stands at (%d,%d), want the stored route's (%d,%d)", tick, e.X, e.Y, route[tick-1].x, route[tick-1].y)
			}
		}
		// The fresh route turns north-east, so the next step may wait a turn.
		for k := 0; k < 4 && w.entities[0].X == route[16].x; k++ {
			Step(w, nil)
		}
		if e := w.entities[0]; e.Y >= route[16].y {
			t.Fatalf("after seventeen near searches the unit kept the stored route: at (%d,%d)", e.X, e.Y)
		}
	})
}

// A follower whose next route cell a walker holds waits behind it, turned to
// that cell; with a standing unit there it searches round it at once.
func TestAFollowerWaitsBehindAWalkerAndGoesRoundAStandingUnit(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		w := walkStepWorld(t, second, walkStepUnit(1, 5, 5, 16), walkStepUnit(2, 4, 5, 16))
		Step(w, []Command{MoveTo(1, CellPoint{X: 20, Y: 5}), MoveTo(2, CellPoint{X: 19, Y: 5})})
		for w.entities[0].Transit > 0 && w.entities[0].X == 5 || w.entities[0].Stride.FromX == 5 && w.entities[0].Transit > 0 {
			if b := w.entities[1]; b.X != 4 || b.Y != 5 || b.Transit != 0 {
				t.Fatalf("the follower stepped while the walker held its next cell: %+v", b)
			}
			Step(w, nil)
		}

		s := walkStepUnit(1, 5, 5, 16)
		w = walkStepWorld(t, second, s, walkStepUnit(2, 4, 5, 16))
		Step(w, []Command{MoveTo(2, CellPoint{X: 19, Y: 5})})
		moved := false
		for tick := 0; tick < 8 && !moved; tick++ {
			b := w.entities[1]
			moved = b.Transit > 0 || b.X != 4 || b.Y != 5
			Step(w, nil)
		}
		if !moved {
			t.Fatalf("the follower did not go round the standing unit: %+v", w.entities[1])
		}
	})
}
