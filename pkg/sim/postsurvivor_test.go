package sim

import (
	"encoding/binary"
	"testing"
)

// The order 0 block walks a member home when no survivor stands within
// radius 5 of the post (AI-704): a dying hostile (health 0 through -9) on the
// post is not engaged. The guard stands two cells off its post at (10,10) and
// the only hostile in the block is a dying body one cell from the post. Both
// sides of the routine are driven: the per-actor arm and the saved guard.
func TestThePostBlockEngagesNoDyingBody(t *testing.T) {
	t.Parallel()
	dying := pxHostile(2, 11, 10)
	dying.HP = -5
	check := func(name string, got Entity) {
		t.Helper()
		if got.HasAttackTarget {
			t.Errorf("%s: the guard holds victim %d, want no victim for a dying body", name, got.AttackTarget)
		}
		if !got.HasTarget || got.TargetX != 10 || got.TargetY != 10 {
			t.Errorf("%s: destination %v (%d,%d), want the walk home to (10,10)", name, got.HasTarget, got.TargetX, got.TargetY)
		}
	}

	w := pxGuard(t, 12, 10, dying)
	w.entities[0].PostX, w.entities[0].PostY = 10, 10
	w.armGuard(0)
	check("per-actor arm", entityAt(t, w, 1))

	s := pxGuard(t, 12, 10, dying)
	o := &SavedActorOrder{Entity: 1, State: 0xb}
	binary.LittleEndian.PutUint16(o.Raw[:], 10|10<<8)
	s.savedGuard(0, o)
	check("saved guard", entityAt(t, s, 1))

	living := pxHostile(2, 11, 10)
	l := pxGuard(t, 12, 10, living)
	l.entities[0].PostX, l.entities[0].PostY = 10, 10
	l.armGuard(0)
	if got := entityAt(t, l, 1); !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Errorf("control: the guard holds victim %v/%d, want the living hostile 2", got.HasAttackTarget, got.AttackTarget)
	}
}

// walkMaxStall steps w until walker 1 holds no destination and reports it
// then, with the most stalled passes it counted on the way.
func walkMaxStall(t *testing.T, w *World) (Entity, int) {
	t.Helper()
	most := 0
	for k := 0; k < 200; k++ {
		Step(w, nil)
		e := laEnt(t, w, 1)
		most = max(most, int(e.Stall))
		if !e.HasTarget {
			return e, most
		}
	}
	t.Fatal("the walk never ended")
	return Entity{}, 0
}

// A first-game AI-owned walk home whose near search aims at its own
// destination and finds nothing is refused on that first answer and takes the
// refused walk's answer, the hostile within reach (AI-709). A walk whose
// search aims at a waypoint, and a human participant's walk, keep the stall
// count to the sixteenth pass.
func TestAnAIWalkHomeAimedAtItsPostIsRefusedAtOnce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		owner  uint32
		postX  int32
		stalls bool
	}{
		{"AI walk aimed at its post", 2, 9, false},
		{"AI walk aimed at a waypoint", 2, 20, true},
		{"human participant's walk aimed at its destination", SelfSlot, 9, true},
	} {
		w := jammedCorridor(t, 3, 5)
		i := indexOfEntity(w.entities, 1)
		if tc.owner == SelfSlot {
			w.entities[i].Owner = SelfSlot
			Step(w, []Command{MoveTo(1, CellPoint{X: tc.postX, Y: 4})})
		} else {
			w.entities[i].PostX, w.entities[i].PostY = tc.postX, 4
			w.walkHome([]int{i})
		}
		got, most := walkMaxStall(t, w)
		if got.X != 7 {
			t.Fatalf("%s: the walk ended at (%d,%d), want column 7 before the jam", tc.name, got.X, got.Y)
		}
		if stalled := most >= stallLimit-1; stalled != tc.stalls {
			t.Errorf("%s: the walk counted %d stalled passes, want the stall count %v", tc.name, most, tc.stalls)
		}
		if tc.owner == 2 && (!got.HasAttackTarget || got.AttackTarget != 3) {
			t.Errorf("%s: victim %v/%d, want the hostile within reach", tc.name, got.HasAttackTarget, got.AttackTarget)
		}
	}
}
