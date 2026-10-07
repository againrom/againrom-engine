package sim

import "testing"

// pickupBeyondWall is a sack behind a wall no terrain route crosses.
func pickupBeyondWall(t *testing.T, hostileBeside bool, reach uint8, mage bool) *World {
	return pickupBeyondWallAt(t, hostileBeside, 11, 5, reach, mage)
}

func pickupBeyondWallAt(t *testing.T, hostileBeside bool, hx, hy int32, reach uint8, mage bool) *World {
	t.Helper()
	const width, height = 30, 10
	grid := make([]byte, width*height)
	for y := int32(0); y < height; y++ {
		grid[y*width+14] = blockGround
	}
	walker := engFighter(1, SelfSlot, 10, 4)
	walker.Reach = reach
	walker.ScanRange = 100
	if mage {
		walker.MaxMana, walker.Mana = 10, 10
	}
	ents := []Entity{walker}
	cells := [][3]uint32{{SelfSlot, 5, 1}, {5, SelfSlot, 1}}
	if hostileBeside {
		beside := engFighter(11, 5, hx, hy)
		beside.ScanRange = 0
		ents = append(ents, beside)
	}
	w, err := NewLootWorld(1, Bounds{Width: width, Height: height}, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...),
		[]Sack{{X: 20, Y: 4, Gold: 9}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w
}

func stepsUntilWalkEnds(t *testing.T, w *World, cmd Command) Entity {
	t.Helper()
	Step(w, []Command{cmd})
	for k := 0; k < 400; k++ {
		if e := laEnt(t, w, 1); !e.HasTarget && k > 0 {
			return e
		}
		Step(w, nil)
	}
	t.Fatal("the walk never ended")
	return Entity{}
}

// A refused pick-up walk takes the hostile within reach (AI-375, AI-350).
func TestARefusedPickupWalkTakesTheHostileWithinReach(t *testing.T) {
	w := pickupBeyondWall(t, true, 2, false)
	e := stepsUntilWalkEnds(t, w, PickUp(1, CellPoint{X: 20, Y: 4}))
	if !e.AcquirePursuit || !e.HasAttackTarget || e.AttackTarget != 11 || e.PendingOrder.Kind != PendingNone {
		t.Fatalf("acquisition %v victim %v/%d pending %d; want hostile 11 taken in place of the pick-up request",
			e.AcquirePursuit, e.HasAttackTarget, e.AttackTarget, e.PendingOrder.Kind)
	}
	back := worldRoundTripForTest(t, w)
	hp := laEnt(t, w, 11).HP
	for range 160 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("cold continuation of the acquired pick differs")
		}
	}
	if laEnt(t, w, 11).HP >= hp {
		t.Fatal("the unit never struck its pick")
	}
}

// Controls: a refused move stands; an unanswered pick-up keeps its request.
func TestARefusedMoveAndAnUnansweredPickupTakeNoPick(t *testing.T) {
	w := pickupBeyondWall(t, true, 2, false)
	e := stepsUntilWalkEnds(t, w, MoveTo(1, CellPoint{X: 20, Y: 4}))
	if e.AcquirePursuit || e.HasAttackTarget || e.PendingOrder.Kind != PendingNone {
		t.Fatalf("refused move: acquisition %v victim %v pending %d; want a unit that stands", e.AcquirePursuit, e.HasAttackTarget, e.PendingOrder.Kind)
	}
	for name, w := range map[string]*World{
		"no hostile":         pickupBeyondWall(t, false, 2, false),
		"short-reach mage":   pickupBeyondWall(t, true, 1, true),
		"hostile past reach": pickupBeyondWallAt(t, true, 12, 6, 1, false),
	} {
		e := stepsUntilWalkEnds(t, w, PickUp(1, CellPoint{X: 20, Y: 4}))
		if e.AcquirePursuit || e.HasAttackTarget || e.PendingOrder.Kind != PendingPickup {
			t.Fatalf("%s: acquisition %v victim %v pending %d; want the pick-up request kept and no pick", name, e.AcquirePursuit, e.HasAttackTarget, e.PendingOrder.Kind)
		}
	}
}
