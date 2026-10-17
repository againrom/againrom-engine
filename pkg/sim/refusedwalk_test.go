package sim

import "testing"

// jammedCorridor is a corridor two cells high (rows 4 and 5) with neutral
// bodies standing across both rows at columns 8 to 12: a walk east from the
// west end has a terrain route and no free step past column 7. The walker 1
// (owner 2) stands at (2,4); a hostile 3 (owner 3) stands at hostileX on row 5.
// No Group is woken, so no Group decision runs and only the walk's own
// refusal can give the walker a victim.
func jammedCorridor(t *testing.T, reach uint8, hostileX int32, extra ...Entity) *World {
	t.Helper()
	const width, height = 30, 10
	b, grid := escortWallGrid(width, height)
	walker := engFighter(1, 2, 2, 4)
	walker.Reach = reach
	ents := []Entity{walker, engFighter(3, 3, hostileX, 5)}
	for k, x := range []int32{8, 9, 10, 11, 12} {
		for r, y := range []int32{4, 5} {
			body := engFighter(EntityID(10+2*k+r), 5, x, y)
			body.ScanRange = 0
			ents = append(ents, body)
		}
	}
	ents = append(ents, extra...)
	cells := [][3]uint32{{2, 3, 1}, {3, 2, 1}}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

// stepUntilWalkEnds steps w until the walker no longer holds a destination and
// reports the walker then. It fails if the walker held a victim earlier.
func stepUntilWalkEnds(t *testing.T, w *World) Entity {
	t.Helper()
	for k := 0; k < 200; k++ {
		before := laEnt(t, w, 1)
		if before.HasAttackTarget {
			t.Fatalf("tick %d: the walker holds victim %d before its walk ended", k, before.AttackTarget)
		}
		Step(w, nil)
		if e := laEnt(t, w, 1); !e.HasTarget {
			return e
		}
	}
	t.Fatal("the walk never ended")
	return Entity{}
}

func TestARefusedGuardWalkHomeTakesTheHostileWithinReach(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		reach uint8
		pick  bool
	}{
		"hostile within reach": {reach: 3, pick: true},
		"hostile past reach":   {reach: 1, pick: false},
	} {
		w := jammedCorridor(t, tc.reach, 5)
		i := indexOfEntity(w.entities, 1)
		if w.entities[i].ActorState != actorStateGuard {
			t.Fatalf("%s: fixture walker state %#x, want guard", name, w.entities[i].ActorState)
		}
		w.entities[i].PostX, w.entities[i].PostY = 20, 4
		w.walkHome([]int{i})
		e := stepUntilWalkEnds(t, w)
		if e.X != 7 {
			t.Fatalf("%s: the walk ended at (%d,%d), want column 7 before the jam", name, e.X, e.Y)
		}
		got := e.HasAttackTarget && e.AttackTarget == 3 && e.AcquirePursuit
		if got != tc.pick {
			t.Errorf("%s: victim %v/%d acquisition %v; want the pick %v", name, e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, tc.pick)
		}
	}
}

func TestARefusedEscortCloseTakesTheHostileWithinReach(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		reach uint8
		pick  bool
	}{
		"hostile within reach": {reach: 3, pick: true},
		"hostile past reach":   {reach: 1, pick: false},
	} {
		w := jammedCorridor(t, tc.reach, 5, engFighter(4, 2, 20, 4))
		i, ti := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 4)
		e := &w.entities[i]
		e.ActorState, e.EscortTarget, e.HasEscortTarget, e.EscortRange = actorStateDefend, 4, true, 1
		w.escortClose(i, ti)
		got := stepUntilWalkEnds(t, w)
		pick := got.HasAttackTarget && got.AttackTarget == 3 && got.AcquirePursuit
		if pick != tc.pick {
			t.Errorf("%s: victim %v/%d acquisition %v at (%d,%d); want the pick %v", name, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit, got.X, got.Y, tc.pick)
		}
	}
}

// A Move member's refused walk is the tail's state 1: it takes no pick, and
// the Group's Move arm sends it again to its stored cell.
func TestARefusedMoveMemberTakesNoPickAndIsSentAgain(t *testing.T) {
	t.Parallel()
	w := jammedCorridor(t, 3, 5)
	wakeGroups(w)
	i := indexOfEntity(w.entities, 1)
	w.commandGroup([]int{i}, orderMove, cell{x: 20, y: 4})
	w.issueGroupDestination([]int{i}, cell{x: 20, y: 4})
	if g := w.savedGroupFor(1); g == nil || g.AI[0x20] != orderMove || g.AI[0x44] != 0 {
		t.Fatalf("fixture: the walker's Group is not a Move Group without a rate byte")
	}
	e := stepUntilWalkEnds(t, w)
	if e.HasAttackTarget {
		t.Fatalf("the refused Move member took victim %d", e.AttackTarget)
	}
	for k := 0; k < 2*scriptCycle; k++ {
		Step(w, nil)
		if e := laEnt(t, w, 1); e.HasTarget {
			if e.TargetX != 20 || e.TargetY != 4 {
				t.Fatalf("the member was sent to (%d,%d), want its stored cell (20,4)", e.TargetX, e.TargetY)
			}
			return
		}
	}
	t.Error("the Move arm never sent the refused member again")
}
