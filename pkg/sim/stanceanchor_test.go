package sim

// This file covers 0106 T2: the second post writer, the script's two stance
// commands (cmdGroupGuard, cmdGroupStandGround). The constructor's own
// writer is post_test.go's (T1); the guard arm that reads the post is a
// later task's.
//
// Every fixture below first gives a living member a cell away from the one
// the constructor posted it at, THEN issues the stance — a fixture that
// never moves a member cannot tell "the command anchored it" apart from
// "the constructor already had", per this task's own brief. No test reads a
// game install (SC-4).

import "testing"

// ---------------------------------------------------------------------- AC-8

// TestCmdGroupGuardAnchorsEveryLivingMembersCurrentCell is AC-8's guard
// half: two living members of the named group are moved off their
// constructed posts, and a dead third member is moved too but left
// unwalked-home — its post stays at the STALE cell the constructor gave it,
// which is what proves the write is gated on Alive() and not merely absent
// by coincidence.
func TestCmdGroupGuardAnchorsEveryLivingMembersCurrentCell(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(2), uint32(1)
	a := engFighter(1, owner, 5, 5)
	a.Group = group
	b := engFighter(2, owner, 6, 6)
	b.Group = group
	c := engFighter(3, owner, 7, 7)
	c.Group = group
	c.HP = -3 // dead: excluded from groupLivingMembers
	w := engWorld(t, engRel(t), a, b, c)

	ia, ib, ic := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2), indexOfEntity(w.entities, 3)
	w.entities[ia].X, w.entities[ia].Y = 20, 21
	w.entities[ib].X, w.entities[ib].Y = 22, 23
	w.entities[ic].X, w.entities[ic].Y = 24, 25

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderGuard)}})

	if got := entityAt(t, w, 1); got.PostX != 20 || got.PostY != 21 {
		t.Errorf("entity 1 (living): post is (%d,%d), want its current cell (20,21)", got.PostX, got.PostY)
	}
	if got := entityAt(t, w, 2); got.PostX != 22 || got.PostY != 23 {
		t.Errorf("entity 2 (living): post is (%d,%d), want its current cell (22,23)", got.PostX, got.PostY)
	}
	if got := entityAt(t, w, 3); got.PostX != 7 || got.PostY != 7 {
		t.Errorf("entity 3 (dead): post is (%d,%d), want the constructed cell (7,7) unchanged — "+
			"a member that is not alive must not be written", got.PostX, got.PostY)
	}
}

// TestCmdGroupStandGroundAnchorsEveryLivingMembersCurrentCell is AC-8's
// stand-ground half: the walk this command gains beside its existing
// single-line order write, over a living and a dead member exactly as the
// guard case above.
func TestCmdGroupStandGroundAnchorsEveryLivingMembersCurrentCell(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(2), uint32(1)
	a := engFighter(1, owner, 5, 5)
	a.Group = group
	b := engFighter(2, owner, 6, 6)
	b.Group = group
	b.HP = -3 // dead: excluded
	w := engWorld(t, engRel(t), a, b)

	ia, ib := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	w.entities[ia].X, w.entities[ia].Y = 30, 31
	w.entities[ib].X, w.entities[ib].Y = 32, 33

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderStandGround)}})

	if order, _, ok := w.groupState(owner, group); !ok || order != orderStandGround {
		t.Fatalf("the fixture's own command did not set Stand Ground — order=%d ok=%v, "+
			"this test would pass for the wrong reason", order, ok)
	}
	if got := entityAt(t, w, 1); got.PostX != 30 || got.PostY != 31 {
		t.Errorf("entity 1 (living): post is (%d,%d), want its current cell (30,31)", got.PostX, got.PostY)
	}
	if got := entityAt(t, w, 2); got.PostX != 6 || got.PostY != 6 {
		t.Errorf("entity 2 (dead): post is (%d,%d), want the constructed cell (6,6) unchanged — "+
			"a member that is not alive must not be written", got.PostX, got.PostY)
	}
}

// ---------------------------------------------------------------------- AC-9

// TestAMidCrossingMembersPostFollowsTheCellItHolds is AC-9's command half:
// entering a stance while a member owes crossing ticks anchors it at the
// cell it presently occupies — the cell it stepped INTO, and in this tree
// that is the same field as its X, Y — not the cell it was posted at by
// construction and not the destination it is still walking toward.
func TestAMidCrossingMembersPostFollowsTheCellItHolds(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(2), uint32(1)
	a := engFighter(1, owner, 5, 5)
	a.Group = group
	w := engWorld(t, engRel(t), a)

	if got := entityAt(t, w, 1); got.PostX != 5 || got.PostY != 5 {
		t.Fatalf("fixture: constructed post is (%d,%d), want (5,5) — setup is wrong", got.PostX, got.PostY)
	}

	// Simulate the member mid-crossing: it has stepped into (6,5) and still
	// owes transit ticks toward a further destination (9,5) — the same
	// shape post_test.go's own mid-crossing fixture uses for the
	// constructor's half of AC-9.
	i := indexOfEntity(w.entities, 1)
	w.entities[i].X, w.entities[i].Y = 6, 5
	w.entities[i].TargetX, w.entities[i].TargetY = 9, 5
	w.entities[i].HasTarget = true
	w.entities[i].Transit = 2
	w.entities[i].TransitTotal = maxTransit

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderGuard)}})

	got := entityAt(t, w, 1)
	if got.PostX != 6 || got.PostY != 5 {
		t.Errorf("post is (%d,%d), want the cell the member currently occupies (6,5) — "+
			"not its construction post (5,5) and not its destination (9,5)", got.PostX, got.PostY)
	}
}

// TestOnlyTheTwoStancesWriteAPost is the negative twin of the two AC-8 tests
// above: Swarm, Move and Swarm 2 each move a member off its constructed post
// and issue their own sub-command, and every one leaves the post exactly
// where the constructor put it. None of the three gains the write this task
// adds to Guard and Stand Ground alone (plan.md's own "no other sub-command
// gains anything").
//
// PATROL LEFT THIS SET IN 1141, and the change is a corrected expectation
// rather than a regression. Its positive assertion is
// TestPatrolCommandAnchorsThePost (patrolfight1141_test.go).
func TestOnlyTheTwoStancesWriteAPost(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sub  int32
	}{
		{"Swarm", int32(orderSwarm)},
		{"Move", int32(orderMove)},
		{"Swarm 2", int32(orderSwarm2)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			const owner, group = uint32(2), uint32(1)
			a := engFighter(1, owner, 5, 5)
			a.Group = group
			w := engWorld(t, engRel(t), a)

			i := indexOfEntity(w.entities, 1)
			w.entities[i].X, w.entities[i].Y = 40, 41

			w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
				Args: [scriptParams]int32{tc.sub, 9, 9}})

			if got := entityAt(t, w, 1); got.PostX != 5 || got.PostY != 5 {
				t.Errorf("sub-command %d: post is (%d,%d), want the constructed cell (5,5) unchanged — "+
					"only Guard and Stand Ground may anchor", tc.sub, got.PostX, got.PostY)
			}
		})
	}
}
