package sim

import "testing"

func TestEntityAliveAnswersAsTheEntityDoes(t *testing.T) {
	w := mustWorld(t, 1, Bounds{Width: 8, Height: 8}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5},
		{ID: 2, X: 2, Y: 2, HP: -1, MaxHP: 5},
		{ID: 3, X: 3, Y: 3, HP: 0, MaxHP: 5},
		{ID: 4, X: 4, Y: 4, HP: 0, MaxHP: 5, Undying: true},
		{ID: 5, X: 5, Y: 5, HP: 0, MaxHP: 0},
	})
	want := map[EntityID]bool{1: true, 2: false, 3: false, 4: true, 5: true, 99: false}
	for id, alive := range want {
		e, held := w.Entity(id)
		if got := held && e.Alive(); got != alive {
			t.Fatalf("entity %d: the copy says alive=%v, the table says %v", id, got, alive)
		}
		if got := w.EntityAlive(id); got != alive {
			t.Errorf("EntityAlive(%d) = %v, want %v", id, got, alive)
		}
	}
}
