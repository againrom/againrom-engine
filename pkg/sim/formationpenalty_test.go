package sim

import "testing"

// TestAFormationMovesAtItsSlowestMembersAloneSpeed: the term a formation order
// stores is the minimum of each member's alone speed, overload penalty
// included, and every member moves at that term raw (MOVE-GROUP-030,
// MOVE-RATE-029).
func TestAFormationMovesAtItsSlowestMembersAloneSpeed(t *testing.T) {
	type source struct {
		raw  int16
		load int32
	}
	cases := []struct {
		name string
		ents []Entity
		src  map[EntityID]source
		want int32
	}{
		{"source Humans, the slowest overloaded", []Entity{
			{ID: 1, X: 4, Y: 8, Humanoid: true, Speed: 19, Capacity: 411},
			{ID: 2, X: 5, Y: 8, Humanoid: true, Speed: 20, Capacity: 1000},
		}, map[EntityID]source{1: {18, 640}, 2: {20, 100}}, 18},
		{"native Humanoids, one overloaded", []Entity{
			{ID: 1, X: 4, Y: 8, Humanoid: true, Speed: 20, Load: 900, Capacity: 300},
			{ID: 2, X: 5, Y: 8, Humanoid: true, Speed: 19, Load: 10, Capacity: 300},
		}, nil, 17},
		{"a Unit and an overloaded native Humanoid", []Entity{
			{ID: 1, X: 4, Y: 8, Speed: 18},
			{ID: 2, X: 5, Y: 8, Humanoid: true, Speed: 20, Load: 900, Capacity: 300},
		}, nil, 17},
		{"control: no member overloaded", []Entity{
			{ID: 1, X: 4, Y: 8, Humanoid: true, Speed: 20, Load: 10, Capacity: 300},
			{ID: 2, X: 5, Y: 8, Speed: 16},
		}, nil, 16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustWorld(t, 1, fmBounds, c.ents)
			for i, e := range c.ents {
				w.entities[i].Load = e.Load // construction recomputes Load from an empty inventory
			}
			for id, s := range c.src {
				if !w.SetHumanMovement(id, s.raw, s.load) {
					t.Fatal("projection refused")
				}
			}
			alone := make([]int32, len(c.ents))
			for i, e := range w.Entities() {
				alone[i] = moverSpeed(e)
			}
			Step(w, []Command{fmMove(1, 40, 40), fmMove(2, 40, 40)})
			if m := min(alone[0], alone[1]); m != c.want {
				t.Fatalf("alone speeds %v, minimum %d, want %d", alone, m, c.want)
			}
			for _, e := range w.Entities() {
				if got := moverSpeed(w.groupRateEntity(e)); got != c.want {
					t.Errorf("id %d rate term %d, want %d (alone speeds %v)", e.ID, got, c.want, alone)
				}
			}
		})
	}
}
