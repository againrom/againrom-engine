package sim

import (
	"bytes"
	"testing"
)

// closedCellRouteWorld returns a world whose mover holds a stored route and
// whose Wall of Earth has just landed across the middle of it.
func closedCellRouteWorld(t *testing.T) *World {
	t.Helper()
	mover := Entity{ID: 1, X: 4, Y: 20, HP: 100, MaxHP: 100, Speed: 10, TokenSize: 1,
		ScanRange: 8, HasTarget: true, TargetX: 34, TargetY: 20}
	w, err := NewStockedSpelledWorld(0xb4, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, []Entity{mover}, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 15}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4 && len(w.routes[0]) == 0; i++ {
		Step(w, nil)
	}
	route := w.routes[0]
	if len(route) == 0 {
		t.Fatal("the mover holds no stored route")
	}
	mid := route[len(route)/2]
	if !w.landArea(w.spells[0], 0, 0, false, mid.x-4, mid.y, mid.x, mid.y, nil) {
		t.Fatal("wall did not land")
	}
	if w.routeTerrainOpen(w.entities[0], w.routes[0]) {
		t.Fatal("the wall closed no cell of the stored route")
	}
	return w
}

// TestAWallLandingKeepsTheStoredRouteAcrossTheClosedCell: nothing invalidates a
// stored route on a terrain change, so the landing leaves the route as it was
// and the mover, which walks it by near searches, never stands in the closed
// cell.
func TestAWallLandingKeepsTheStoredRouteAcrossTheClosedCell(t *testing.T) {
	w := closedCellRouteWorld(t)
	kept := append([]cell(nil), w.routes[0]...)
	Step(w, nil)
	if len(w.routes[0]) == 0 {
		t.Fatalf("the route was discarded after the landing; it held %v", kept)
	}
	for tick := 0; tick < 120; tick++ {
		e := w.entities[0]
		if !w.terrainOpenFootprint(e, e.X, e.Y) {
			t.Fatalf("tick %d: the mover stands in a closed cell at (%d,%d)", tick, e.X, e.Y)
		}
		Step(w, nil)
	}
}

// TestARouteAcrossAClosedCellSurvivesSaveAndNativeContinuation: the byte form
// holds the state a landing produces, a cold load reproduces it exactly, and
// the loaded world and the live world then advance identically. The two are
// separate proofs: bytes and hash first, then every tick after.
func TestARouteAcrossAClosedCellSurvivesSaveAndNativeContinuation(t *testing.T) {
	live := closedCellRouteWorld(t)
	form, err := live.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("a world holding a route across a closed cell did not load: %v", err)
	}
	if len(loaded.routes[0]) != len(live.routes[0]) {
		t.Fatalf("the loaded route holds %d cells, the saved one %d", len(loaded.routes[0]), len(live.routes[0]))
	}
	again, err := loaded.MarshalBinary()
	if err != nil || !bytes.Equal(again, form) {
		t.Fatalf("the loaded world does not marshal back to the saved bytes: %v", err)
	}
	if loaded.Hash() != live.Hash() {
		t.Fatal("the loaded world's hash differs from the saved world's")
	}
	for tick := 0; tick < 120; tick++ {
		Step(live, nil)
		Step(&loaded, nil)
		if live.Hash() != loaded.Hash() {
			t.Fatalf("tick %d after the load: the loaded world diverged from the live one", tick)
		}
		// A save taken mid-walk, while the route still crosses the closed cell,
		// loads too.
		if tick%15 == 0 {
			mid, err := live.MarshalBinary()
			if err != nil {
				t.Fatalf("tick %d: MarshalBinary: %v", tick, err)
			}
			var cold World
			if err := cold.UnmarshalBinary(mid); err != nil {
				t.Fatalf("tick %d: a mid-walk save did not load: %v", tick, err)
			}
		}
	}
}

// TestAMoverReachesItsDestinationPastALongLivedWallOnItsRoute: a wall that
// stands for the rest of the walk covers a route cell within the first four
// cells ahead of the mover. The stored route is kept at the landing, and the
// mover still arrives instead of circling the closed cell until the stall count
// ends the order.
func TestAMoverReachesItsDestinationPastALongLivedWallOnItsRoute(t *testing.T) {
	mover := Entity{ID: 1, X: 4, Y: 20, HP: 100, MaxHP: 100, Speed: 10, TokenSize: 1,
		ScanRange: 8, HasTarget: true, TargetX: 34, TargetY: 20}
	w, err := NewStockedSpelledWorld(0xb4, Bounds{Width: 40, Height: 40}, ModeCanonical,
		Terrain{}, []Entity{mover}, nil, Relations{}, nil, nil,
		[]SpellRule{{ID: 19, Area: true, Distribution: 4, Radius: 2, AreaDuration: 5000}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4 && len(w.routes[0]) == 0; i++ {
		Step(w, nil)
	}
	if len(w.routes[0]) == 0 {
		t.Fatal("the mover holds no stored route")
	}
	landed := false
	for tick := 0; tick < 3000; tick++ {
		e := w.entities[0]
		if !landed {
			// Land the wall when the mover is within two cells of a route cell
			// it is about to aim at, so the closed cell is among the first four.
			route := w.routes[0]
			if len(route) > 0 && e.X >= 16 {
				c := route[min(2, len(route)-1)]
				if !w.landArea(w.spells[0], 0, 0, false, c.x-4, c.y, c.x, c.y, nil) {
					t.Fatal("wall did not land")
				}
				if w.routeTerrainOpen(w.entities[0], w.routes[0]) {
					t.Fatal("the wall closed no cell of the stored route")
				}
				landed = true
			}
		}
		if !w.entities[0].HasTarget {
			break
		}
		Step(w, nil)
	}
	if !landed {
		t.Fatal("the wall never landed")
	}
	if e := w.entities[0]; e.X != 34 || e.Y != 20 {
		t.Fatalf("the mover stopped at (%d,%d) short of (34,20); order held: %v", e.X, e.Y, e.HasTarget)
	}
}
