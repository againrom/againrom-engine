package sim

import "testing"

// swarmFighter is engFighter with health enough to outlive a fight, so that a
// test about where a member goes afterwards is not decided by its death.
func swarmFighter(id EntityID, owner uint32, x, y int32) Entity {
	e := engFighter(id, owner, x, y)
	e.HP, e.MaxHP = 5000, 5000
	return e
}

// TestSwarm2WalksOnToItsCellAfterTheFight: an attack move that meets a hostile
// on the way fights it, and once the group sees nothing the member walks on to
// the cell it was commanded to and stays there (AI-SWARM2GATE-107,
// AI-MOVE-023). The fight ends the member's walk and nothing gave the walk
// back, so the member stopped where the last blow landed.
func TestSwarm2WalksOnToItsCellAfterTheFight(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{1, 3, 1}, [3]uint32{3, 1, 1})
	foe := engFighter(2, 3, 12, 12)
	foe.HP, foe.MaxHP = 5, 5
	w := engWorld(t, rel, swarmFighter(1, SelfSlot, 5, 10), foe)
	to := cell{x: 30, y: 10}
	Step(w, []Command{GroupSwarmTo(1, CellPoint{X: to.x, Y: to.y}, 1)})

	fought, ended := false, false
	var shortBy int64
	for n := 0; n < 600; n++ {
		Step(w, nil)
		e := entityAt(t, w, 1)
		fought = fought || e.HasAttackTarget
		if fought && !e.HasAttackTarget && !ended {
			ended, shortBy = true, cellOf(e).chebyshevTo(to)
		}
	}
	if !fought {
		t.Fatal("fixture: the member never engaged the hostile on its way")
	}
	if shortBy <= formationSpread {
		t.Fatalf("fixture: the fight ended %d cells from the commanded cell, within the arrival distance", shortBy)
	}
	final := entityAt(t, w, 1)
	if d := cellOf(final).chebyshevTo(to); d > formationSpread {
		t.Errorf("after the fight the member stands at (%d,%d), %d cells from the commanded cell (%d,%d), and holds no destination (HasTarget=%v)",
			final.X, final.Y, d, to.x, to.y, final.HasTarget)
	}
}

// TestScriptedSwarm2GroupWalksOnToItsCellAfterTheFight is the same walk for a
// group that a script commands and another participant owns. Group
// sub-command 5 is the attack move's setter (AI-GROUPCMD-020, AI-CMDSET45-108),
// so the group resumes after its fight as the player's does
// (AI-SWARM2GATE-107, AI-MOVE-023).
func TestScriptedSwarm2GroupWalksOnToItsCellAfterTheFight(t *testing.T) {
	t.Parallel()

	const ai = uint32(3)
	rel := engRel(t, [3]uint32{SelfSlot, ai, 1}, [3]uint32{ai, SelfSlot, 1})
	foe := engFighter(2, SelfSlot, 12, 12)
	foe.HP, foe.MaxHP = 5, 5
	// A script names a group by its id alone, so the hostile stands in another
	// group and the command does not send it on as well.
	foe.Group = 1
	w := engWorld(t, rel, swarmFighter(1, ai, 5, 10), foe)
	to := cell{x: 30, y: 10}
	w.cmdGroupCommandedMove(0, orderSwarm2, to.x, to.y)

	fought := false
	for n := 0; n < 600; n++ {
		Step(w, nil)
		fought = fought || entityAt(t, w, 1).HasAttackTarget
	}
	if !fought {
		t.Fatal("fixture: the member never engaged the hostile on its way")
	}
	final := entityAt(t, w, 1)
	if d := cellOf(final).chebyshevTo(to); d > formationSpread {
		t.Errorf("after the fight the scripted member stands at (%d,%d), %d cells from the commanded cell (%d,%d), and holds no destination (HasTarget=%v)",
			final.X, final.Y, d, to.x, to.y, final.HasTarget)
	}
}

// TestSwarm2GroupThatReachedItsCellStaysOnIt is the control for the walk above:
// a group with nothing in sight that stands on its commanded cell is left in
// every field, decision after decision.
func TestSwarm2GroupThatReachedItsCellStaysOnIt(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), swarmFighter(1, SelfSlot, 5, 10))
	to := cell{x: 20, y: 10}
	Step(w, []Command{GroupSwarmTo(1, CellPoint{X: to.x, Y: to.y}, 1)})
	for n := 0; n < 200 && cellOf(entityAt(t, w, 1)) != to; n++ {
		Step(w, nil)
	}
	held := entityAt(t, w, 1)
	if cellOf(held) != to {
		t.Fatalf("fixture: the member reached (%d,%d), not the commanded cell", held.X, held.Y)
	}
	for n := 0; n < 200; n++ {
		Step(w, nil)
		if e := entityAt(t, w, 1); e.HasTarget || e.X != held.X || e.Y != held.Y {
			t.Fatalf("tick %d: a member on its commanded cell moved or took a destination: %+v", n, e)
		}
	}
}

// TestHoldPressedMidFightEndsTheAttackMoveResume: a member of an attack-move
// group that is told to hold its ground while it fights belongs to the new
// Stand Ground group, whose arm never walks (AI-STAND-076), so once the fight
// is over it stands where it is and the attack move's cell is not resumed.
func TestHoldPressedMidFightEndsTheAttackMoveResume(t *testing.T) {
	t.Parallel()

	rel := engRel(t, [3]uint32{1, 3, 1}, [3]uint32{3, 1, 1})
	foe := engFighter(2, 3, 12, 12)
	foe.HP, foe.MaxHP = 5, 5
	w := engWorld(t, rel, swarmFighter(1, SelfSlot, 5, 10), foe)
	to := cell{x: 30, y: 10}
	Step(w, []Command{GroupSwarmTo(1, CellPoint{X: to.x, Y: to.y}, 1)})

	held := false
	for n := 0; n < 600; n++ {
		if !held && entityAt(t, w, 1).HasAttackTarget {
			Step(w, []Command{GroupStance(1, OrderStandGround, 2)})
			held = true
			continue
		}
		Step(w, nil)
	}
	if !held {
		t.Fatal("fixture: the member never engaged the hostile on its way")
	}
	final := entityAt(t, w, 1)
	if d := cellOf(final).chebyshevTo(to); d <= formationSpread || final.HasTarget {
		t.Errorf("a member told to hold its ground mid-fight stands at (%d,%d), %d cells from the attack move's cell, HasTarget=%v; it must stay where the fight left it",
			final.X, final.Y, d, final.HasTarget)
	}
}
