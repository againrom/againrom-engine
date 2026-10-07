package sim

import "testing"

// retreatFighter is a player unit with a short attack cycle whose blow always
// lands for 5, so the intruder's health says how many cycles resolved. It faces
// north and turns at 32 per tick, so a victim to its east costs it two ticks
// before its cycle can load.
func retreatFighter(id EntityID, x, y int32) Entity {
	e := withdrawalFighter(id, SelfSlot, x, y, 100)
	e.DamageBase = 5
	e.AttackCharge, e.AttackRelax = 4, 4
	e.RotationSpeed = 32
	return e
}

// retreatIntruder is a hostile that never strikes back: the relation from its
// owner to the player is locked non-hostile, so only the retreating unit acts.
func retreatIntruder(id EntityID, x, y int32) Entity {
	return withdrawalFighter(id, 3, x, y, 100)
}

func retreatRelations(t *testing.T) Relations {
	t.Helper()
	return engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2})
}

// pocketedRetreater stands a player unit at (20,20) beside a walled pocket that
// holds its decoded flee cell for a hostile to the east. The cells within two of
// (17,20) are all closed, so the far search from (20,20) labels nothing near the
// flee cell and settles on the unit's own cell: it comes back with no route.
func pocketedRetreater(t *testing.T, intruder Entity) *World {
	t.Helper()
	grid := make([]byte, engBounds.Width*engBounds.Height)
	for y := int32(18); y <= 22; y++ {
		for x := int32(15); x <= 19; x++ {
			grid[y*engBounds.Width+x] = blockGround
		}
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: grid},
		[]Entity{retreatFighter(1, 20, 20), intruder}, nil, retreatRelations(t))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func retreatBlows(w *World) int32 { return (100 - w.entities[1].HP) / 5 }

// stepIdle advances w for n ticks and reports the longest run of ticks on which
// the unit at index 0 held no order of any kind while its cycle was ready.
func stepIdle(t *testing.T, w *World, n int, x, y int32) (longest int) {
	t.Helper()
	run := 0
	for i := 0; i < n; i++ {
		Step(w, nil)
		e := w.entities[0]
		if e.X != x || e.Y != y {
			t.Fatalf("tick %d: unit moved to (%d,%d)", w.tick, e.X, e.Y)
		}
		if e.ActorState != actorStateRetreat {
			t.Fatalf("tick %d: unit left Retreat for state %d", w.tick, e.ActorState)
		}
		if !e.HasAttackTarget && !e.HasTarget && e.AttackPhase == AttackReady {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	return longest
}

// TestExplicitRetreatWhoseFleeCellHasNoRouteTakesTheStandingPickAndFights is the
// ordinary-play shape of a retreating unit with a wall at its back: the state
// arm writes the decoded flee cell, the far search comes back with no route,
// and the original's order machine clears the pending move and reacquires a
// victim within reach (AI-RETREAT-273, AI-ROUTE-045, AI-335, MOVE-072). The
// state stays Retreat and the pick keeps its turn toward the victim and its
// cycle. Before, the movement pass ended the move and the unit stood without an
// order for as long as the hostile stayed beside it.
func TestExplicitRetreatWhoseFleeCellHasNoRouteTakesTheStandingPickAndFights(t *testing.T) {
	w := pocketedRetreater(t, retreatIntruder(2, 21, 20))
	if FacingDir(w.entities[0].Facing) == 2 {
		t.Fatal("fixture: the unit already faces its victim, so its pick would load without a turn")
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})

	got := w.entities[0]
	if got.ActorState != actorStateRetreat {
		t.Fatalf("state after Retreat = %d, want %d", got.ActorState, actorStateRetreat)
	}
	if got.X != 20 || got.Y != 20 {
		t.Fatalf("unit left its cell for a flee cell nothing leads to: (%d,%d)", got.X, got.Y)
	}
	if got.HasTarget {
		t.Fatalf("unit kept the destination (%d,%d) that no route serves", got.TargetX, got.TargetY)
	}
	if !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Fatalf("unit after the refused flee holds attack %t on %d, want its victim 2",
			got.HasAttackTarget, got.AttackTarget)
	}

	idle := stepIdle(t, w, 160, 20, 20)
	if blows := retreatBlows(w); blows < 3 {
		t.Errorf("unit landed %d blow(s) in 160 ticks beside its hostile, want at least 3", blows)
	}
	if idle > scriptCycle+4 {
		t.Errorf("unit held no order for %d consecutive ticks beside its hostile, want at most %d", idle, scriptCycle+4)
	}
}

// TestExplicitRetreatCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut
// is the same unit against the west edge of the playable rectangle with its
// hostile beside it to the east. The decoded flee cell clamps onto the unit's
// own cell. The request equals the cell of a centred mover, which returns
// before search and raises no failure (MOVE-083), so the reacquisition of a
// refused flee does not run and the arrived move ends without a victim.
func TestExplicitRetreatCorneredAtThePlayableEdgeTakesNoPickForTheCentredShortcut(t *testing.T) {
	unit := retreatFighter(1, 8, 20)
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{},
		[]Entity{unit, retreatIntruder(2, 9, 20)}, nil, retreatRelations(t))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})

	got := w.entities[0]
	if got.X != 8 || got.Y != 20 || got.HasTarget {
		t.Fatalf("unit at the edge = (%d,%d) with destination %t, want it standing on (8,20) with none", got.X, got.Y, got.HasTarget)
	}
	if got.HasAttackTarget {
		t.Fatalf("unit cornered at the edge holds attack on %d, want no reacquisition", got.AttackTarget)
	}
}

// TestExplicitRetreatWithALoadedCycleFightsOnAfterTheCycleEnds admits Retreat
// while the unit holds a loaded cycle against a hostile beside it. The cycle
// resolves its blow first (AI-RETREAT-272); the decision that follows meets the
// refused flee cell and the unit takes the standing pick again, so it lands
// more than the one blow it had loaded. Before, it stood after that blow.
func TestExplicitRetreatWithALoadedCycleFightsOnAfterTheCycleEnds(t *testing.T) {
	w := pocketedRetreater(t, retreatIntruder(2, 21, 20))
	w.orderAttack(0, 2)
	for n := 0; n < 4 && w.entities[0].AttackPhase == AttackReady; n++ {
		Step(w, nil)
	}
	if w.entities[0].AttackPhase == AttackReady {
		t.Fatal("fixture did not load the cycle")
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	if e := w.entities[0]; e.ActorState != actorStateRetreat || e.AttackPhase == AttackReady || !e.HasAttackTarget {
		t.Fatalf("Retreat did not leave the loaded cycle to finish: state %d phase %d attack %t",
			e.ActorState, e.AttackPhase, e.HasAttackTarget)
	}
	idle := stepIdle(t, w, 160, 20, 20)
	if blows := retreatBlows(w); blows < 3 {
		t.Errorf("unit landed %d blow(s) in 160 ticks, want the loaded one and at least two more", blows)
	}
	if idle > scriptCycle+4 {
		t.Errorf("unit held no order for %d consecutive ticks beside its hostile, want at most %d", idle, scriptCycle+4)
	}
}

// TestExplicitRetreatWhoseFleeCellHasNoRouteDoesNotWalkAtAHostileOutOfReach is
// the reach bound of the reacquisition (AI-327): the hostile stands three cells
// east, inside the block the flee cell is computed from and outside the unit's
// reach of one. The refused flee ends the move and the unit stands where it is;
// it does not close on a victim it cannot strike.
func TestExplicitRetreatWhoseFleeCellHasNoRouteDoesNotWalkAtAHostileOutOfReach(t *testing.T) {
	w := pocketedRetreater(t, retreatIntruder(2, 23, 20))
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	for i := 0; i < 96; i++ {
		Step(w, nil)
		if e := w.entities[0]; e.X != 20 || e.Y != 20 || e.HasTarget || e.HasAttackTarget {
			t.Fatalf("tick %d: unit at (%d,%d) destination %t attack %t, want it standing with no order",
				w.tick, e.X, e.Y, e.HasTarget, e.HasAttackTarget)
		}
	}
	if w.entities[1].HP != 100 {
		t.Fatalf("hostile out of reach lost health: %d", w.entities[1].HP)
	}
	if got := w.entities[0].ActorState; got != actorStateRetreat {
		t.Fatalf("unit left Retreat for state %d", got)
	}
}
