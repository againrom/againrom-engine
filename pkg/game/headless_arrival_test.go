package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestHeadlessWalkWaitsForCompletedArrival(t *testing.T) {
	for _, route := range []string{"live", "world"} {
		t.Run(route, func(t *testing.T) {
			w, mw := pickupArrivalWorld(t)
			if route == "live" {
				liveWalk(t, mw, 7, 4, 3)
			} else {
				worldWalk(t, w, 7, 4, 3)
			}
			e, _ := w.Entity(7)
			if w.Tick() != 16 || e.X != 4 || e.Y != 3 || e.Transit != 0 || e.TransitTotal != 16 {
				t.Fatalf("walk returned before all sixteen crossing payments: tick=%d actor=%+v", w.Tick(), e)
			}
			requirePickupArrivalGround(t, w, mw)
		})
	}
}

func TestHeadlessTakePaysAlreadyClaimedCell(t *testing.T) {
	for _, route := range []string{"live", "world"} {
		t.Run(route, func(t *testing.T) {
			w, mw := pickupArrivalWorld(t)
			sim.Step(w, []sim.Command{sim.MoveTo(7, sim.CellPoint{X: 4, Y: 3})})
			e, _ := w.Entity(7)
			if e.X != 4 || e.Y != 3 || e.Transit != 15 {
				t.Fatal("fixture did not claim the cell on its first crossing payment", e)
			}
			if route == "live" {
				liveTakeAt(t, mw, 7, 4, 3)
			} else {
				worldTakeAt(t, &Mission{World: w}, 7, 4, 3)
			}
			e, _ = w.Entity(7)
			if w.Tick() != 16 || e.Transit != 0 {
				t.Fatalf("take did not pay the existing crossing exactly: tick=%d transit=%d", w.Tick(), e.Transit)
			}
			requirePickupArrivalTaken(t, w)
		})
	}
}
