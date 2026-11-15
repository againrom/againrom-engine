package sim

import "testing"

// wideFoeWorld is moveOntoEnemyWorld with the enemy's token side set, so the
// enemy covers side x side cells from its anchor (foeX, foeY).
func wideFoeWorld(t *testing.T, foeX, foeY int32, side uint8) *World {
	t.Helper()
	hero := engFighter(1, SelfSlot, 4, 5)
	hero.HP, hero.MaxHP = 2000, 2000
	foe := engFighter(2, 3, foeX, foeY)
	foe.HP, foe.MaxHP = 2000, 2000
	foe.DamageBase = 0
	foe.ScanRange = 0
	foe.TokenSize = side
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 1})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{hero, foe}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	wakeGroups(w)
	return w
}

// walkEndsInStand orders the hero to (9,5) and runs 60 ticks. The move order
// must end once, at the stand order beside the held cell (8,5), and must not be
// issued again afterwards.
func walkEndsInStand(t *testing.T, w *World, label string) {
	t.Helper()
	Step(w, []Command{MoveTo(1, CellPoint{X: 9, Y: 5})})
	ended := -1
	for k := 0; k < 60; k++ {
		Step(w, nil)
		h := laEnt(t, w, 1)
		if ended < 0 {
			if h.HasTarget {
				continue
			}
			ended = k
			if h.ActorState != actorStateAcquire || h.X != 8 || h.Y != 5 {
				t.Fatalf("%s tick %d: hero at (%d,%d) state %#x; want the stand order beside the held cell at (8,5)", label, k, h.X, h.Y, h.ActorState)
			}
			continue
		}
		if h.HasTarget {
			t.Fatalf("%s tick %d: the move order to (9,5) was issued again after the walk ended at tick %d; want the stand order to hold", label, k, ended)
		}
	}
	if ended < 0 {
		t.Fatalf("%s: the hero still walks to (9,5) after 60 ticks; want the stand order beside the held cell", label)
	}
}

// A player's walk whose goal is a non-anchor cell of a wide enemy ends beside
// that enemy in the stand order. The 2x2 enemy anchored at (9,4) covers (9,5).
func TestWalkGoalInsideWideEnemyBecomesStand(t *testing.T) {
	walkEndsInStand(t, wideFoeWorld(t, 9, 4, 2), "wide enemy anchored at (9,4)")
}

// A player's walk whose goal a one-cell enemy holds still ends beside it in the
// stand order, as it did before the footprint query.
func TestWalkGoalOnOneCellEnemyBecomesStand(t *testing.T) {
	walkEndsInStand(t, wideFoeWorld(t, 9, 5, 1), "one-cell enemy at (9,5)")
}
