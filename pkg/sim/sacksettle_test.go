package sim

import (
	"slices"
	"testing"
)

// settleWorld is collectWorld with a second hero, entity 8 at (x8, y8).
func settleWorld(t *testing.T, sacks []Sack, x7, y7, x8, y8 int32) *World {
	t.Helper()
	hero := func(id EntityID, x, y int32) Entity {
		return Entity{ID: id, X: x, Y: y, TypeID: HumanTypeID, Humanoid: true, Owner: SelfSlot, HP: 100, MaxHP: 100, ScanRange: 3}
	}
	w := secondWorld(t, 1, Bounds{Width: 24, Height: 24}, nil, hero(7, x7, y7), hero(8, x8, y8))
	if err := w.ReplaceGroundSacks(sacks); err != nil {
		t.Fatal(err)
	}
	return w
}

// settleOnce is the driver's settle after one Step: the cells taken and the
// orders it queues for the next Step.
func settleOnce(w *World, docs SackDocuments) (taken []CellPoint, next []Command) {
	for _, s := range w.SettleSackPickup(docs) {
		if s.Outcome == SackPickupTaken {
			taken = append(taken, CellPoint{s.Sack.X, s.Sack.Y})
		}
		if s.Next {
			next = append(next, s.Order)
		}
	}
	return taken, next
}

// arrivedTakers answers each actor that holds a pending pickup and stands
// arrived on its cell, with that cell.
func arrivedTakers(w *World) map[EntityID]CellPoint {
	at := map[EntityID]CellPoint{}
	for _, e := range w.Entities() {
		if e.PendingOrder.Kind == PendingPickup && e.X == e.PendingOrder.X && e.Y == e.PendingOrder.Y && w.SackPickupArrived(e.ID) {
			at[e.ID] = CellPoint{e.PendingOrder.X, e.PendingOrder.Y}
		}
	}
	return at
}

func TestSettleServesEveryArrivedTakerInTheStepHeArrives(t *testing.T) {
	for _, tc := range []struct {
		name  string
		sacks []Sack
		first func(w *World) []Command
		want  int
	}{
		{"a collector does not hold back a plain pickup",
			[]Sack{{X: 12, Y: 10, Gold: 1}, {X: 15, Y: 10, Gold: 2}, {X: 18, Y: 10, Gold: 4}, {X: 12, Y: 16, Gold: 8}},
			func(w *World) []Command {
				c, _ := w.OrderSackCollect(7)
				return []Command{c, PickUp(8, CellPoint{X: 12, Y: 16})}
			}, 15},
		{"a long walk does not hold back a collector",
			[]Sack{{X: 22, Y: 22, Gold: 1}, {X: 12, Y: 16, Gold: 2}, {X: 14, Y: 16, Gold: 4}},
			func(w *World) []Command {
				c, _ := w.OrderSackCollect(8)
				return []Command{PickUp(7, CellPoint{X: 22, Y: 22}), c}
			}, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := settleWorld(t, tc.sacks, 10, 10, 10, 16)
			cmds := tc.first(w)
			for k := 0; k < 600; k++ {
				Step(w, cmds)
				arrived := arrivedTakers(w)
				_, cmds = settleOnce(w, SackDocuments{})
				for id, cell := range arrived {
					if _, still := w.groundSackAt(cell.X, cell.Y); still {
						t.Fatalf("tick %d: hero %d arrived at %+v and the settle left the sack", k, id, cell)
					}
				}
			}
			if len(w.Sacks()) != 0 || w.Purse(SelfSlot) != uint32(tc.want) {
				t.Fatalf("sacks left %v, purse %d, want %d", w.Sacks(), w.Purse(SelfSlot), tc.want)
			}
		})
	}
}

// Two heroes sent to one sack: the one standing on it takes it, and the one
// who cannot enter that cell does not wait for ever (sacks at (16,10) and
// (18,10), hero 7 at (13,10), hero 8 at (15,11)).
func TestTwoHeroesSentToOneSackBothEnd(t *testing.T) {
	for _, tc := range []struct {
		name    string
		collect [2]bool
	}{
		{"two plain pickups", [2]bool{false, false}},
		{"two collects", [2]bool{true, true}},
		{"collect and plain", [2]bool{true, false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := settleWorld(t, []Sack{{X: 16, Y: 10, Gold: 1}, {X: 18, Y: 10, Gold: 2}}, 13, 10, 15, 11)
			var cmds []Command
			for k, id := range []EntityID{7, 8} {
				c := PickUp(id, CellPoint{X: 16, Y: 10})
				if tc.collect[k] {
					var ok bool
					if c, ok = w.OrderSackCollect(id); !ok || c.X != 16 || c.Y != 10 {
						t.Fatalf("hero %d collect %+v/%v, want (16,10)", id, c, ok)
					}
				}
				cmds = append(cmds, c)
			}
			var taken []CellPoint
			for k := 0; k < 600; k++ {
				Step(w, cmds)
				var got []CellPoint
				got, cmds = settleOnce(w, SackDocuments{})
				taken = append(taken, got...)
			}
			e7, _ := w.Entity(7)
			e8, _ := w.Entity(8)
			t.Logf("took %v; hero 7 at (%d,%d) state %d; hero 8 at (%d,%d) state %d; sacks %v", taken, e7.X, e7.Y, e7.ActorState, e8.X, e8.Y, e8.ActorState, w.Sacks())
			if len(taken) == 0 || taken[0] != (CellPoint{16, 10}) {
				t.Fatalf("took %v, want (16,10) first", taken)
			}
			for _, e := range []Entity{e7, e8} {
				if e.PendingOrder.Kind != PendingNone || e.HasTarget || e.ActorState != actorStateAcquire || e.PostX != e.X || e.PostY != e.Y {
					t.Fatalf("hero %d still waits: %+v", e.ID, e)
				}
			}
			want := []CellPoint{{16, 10}}
			if tc.collect[0] || tc.collect[1] {
				want = append(want, CellPoint{18, 10})
			}
			if !slices.Equal(taken, want) {
				t.Fatalf("took %v, want %v", taken, want)
			}
		})
	}
}
