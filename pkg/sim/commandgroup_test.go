package sim

import "testing"

// This file covers 0117 T2: the writer — group.go's commandGroup, commandFloor
// and freeCommandGroup — and the two call sites that reach it, groupOrder and
// stepWorld's KindMoveTo arm. T1's own AC-9 (round trip with an unwritten
// field) and the version-26 refusal already live elsewhere; this file is
// AC-2 through AC-8. AC-1 is cmd/missionrun's, against a lawful install; AC-10
// is a deletion check run by hand rather than a test (plan.md T2).
//
// It builds on engage_test.go's and commanded_test.go's own helpers
// (engWorld, engRel, engFighter, cmdEntity) rather than inventing a second
// set. No test here reads a game install, and none draws from w.rng.

// -------------------------------------------------------------------- AC-2

// TestACommandedGuardIsNotWalkedHomeButAnUncommandedOneIs is AC-2, and the
// PAIR is the test: a single commanded actor standing still after arriving
// proves nothing about whether the walk home's own arm ran at all, only that
// nothing sent it anywhere — which an arm that never fires would also give.
// The uncommanded actor of the SAME world, decided in the SAME pass, is what
// tells "the arm did not run" apart from "the arm ran and correctly did
// nothing".
//
// Both actors are owned by slot 9 — not SelfSlot, so freezeGroups gives both
// Guard by construction — and both are built off their own post, on
// walkhome_test.go's own convention: the constructor anchors a post at
// construction, so overriding it directly is what a member that walked away
// under some earlier order, with nothing since re-anchoring it, looks like.
// Relations{} (no hostility named at all) keeps both groups' candidate lists
// empty, so any destination either actor ends the decision holding can only
// be the walk home's.
func TestACommandedGuardIsNotWalkedHomeButAnUncommandedOneIs(t *testing.T) {
	t.Parallel()

	const owner = uint32(9)
	commanded := engFighter(1, owner, 8, 5)
	free := engFighter(2, owner, 20, 20)
	w := engWorld(t, Relations{}, commanded, free)
	ci, fi := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	w.entities[ci].PostX, w.entities[ci].PostY = 5, 5
	w.entities[fi].PostX, w.entities[fi].PostY = 15, 20

	// Entity 1 is commanded a few cells off its placement — far enough (10
	// cells) that the single Step below cannot walk the whole distance in
	// one advance, so the order is still open when this checks it stuck.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 8, Y: 15}})
	if e := cmdEntity(t, w, 1); !e.HasTarget || e.TargetX != 8 || e.TargetY != 15 {
		t.Fatalf("fixture: the move order did not stick: %+v", e)
	}
	if id := w.entities[indexOfEntity(w.entities, 1)].CommandGroup; id == 0 {
		t.Fatalf("fixture: the move order built no command group at all")
	}

	// Arrive by hand: cmdToDecision/Step timing is not what this test is
	// about, and walkhome_test.go's own AC-2 case takes the same shortcut.
	w.entities[ci].X, w.entities[ci].Y = 8, 15
	w.entities[ci].TargetX, w.entities[ci].TargetY, w.entities[ci].HasTarget = 0, 0, false

	for _, g := range w.aiGroups() {
		w.decide(g)
	}

	after1 := cmdEntity(t, w, 1)
	if after1.HasTarget {
		t.Errorf("the commanded actor was given a destination after arriving: (%d,%d) — it holds a "+
			"move order now, which has no walk home (FR-8)", after1.TargetX, after1.TargetY)
	}
	after2 := cmdEntity(t, w, 2)
	if !after2.HasTarget || after2.TargetX != 15 || after2.TargetY != 20 {
		t.Errorf("the UNCOMMANDED actor of the same world was not walked home: destination held=%v "+
			"(%d,%d), want its post (15,20) — without this half the pair proves nothing",
			after2.HasTarget, after2.TargetX, after2.TargetY)
	}
}

// -------------------------------------------------------------------- AC-3

// TestACommandedActorsPlacedGroupIsUnchangedAndItsAliveCountHolds is AC-3: a
// move order does not move Group, and the mission script's alive-count check
// — which reads Group and not effectiveGroup, script.go's own corrected note
// — counts the commanded actor exactly as it did before the order.
func TestACommandedActorsPlacedGroupIsUnchangedAndItsAliveCountHolds(t *testing.T) {
	t.Parallel()

	const owner, placedGroup, reg = uint32(9), uint32(4), int32(0)
	a := engFighter(1, owner, 5, 5)
	a.Group = placedGroup
	w := engWorld(t, Relations{}, a)

	before := w.entities[indexOfEntity(w.entities, 1)].Group
	w.runCheck(ScriptCheck{Op: ScriptCheckGroupCount, Group: placedGroup, HasGroup: true, Register: reg}, 0, nil)
	countBefore := w.registerAt(reg)
	if countBefore != 1 {
		t.Fatalf("fixture: the alive count over the placed group is %d before any order, want 1", countBefore)
	}

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 20}})

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after.Group != before {
		t.Errorf("the placed group moved from %d to %d after a move order — FR-3 says nothing writes it", before, after.Group)
	}
	if after.CommandGroup == 0 {
		t.Fatalf("fixture: the order built no command group at all")
	}

	w.runCheck(ScriptCheck{Op: ScriptCheckGroupCount, Group: placedGroup, HasGroup: true, Register: reg}, 0, nil)
	countAfter := w.registerAt(reg)
	if countAfter != countBefore {
		t.Errorf("the alive count over the placed group moved from %d to %d after a command — FR-3 "+
			"says a player's click may not reach it", countBefore, countAfter)
	}
}

// -------------------------------------------------------------------- AC-4

// TestAGroupMoveBuildsOneCommandGroupWithARecordPerOwner is AC-4: two actors
// of two DIFFERENT owners, commanded by one group move, end in the SAME
// command group id, both records at the move order, one record per distinct
// owner.
func TestAGroupMoveBuildsOneCommandGroupWithARecordPerOwner(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 9, 5, 5)
	b := engFighter(2, 11, 6, 5)
	w := engWorld(t, Relations{}, a, b)

	Step(w, []Command{
		{Kind: KindGroupMoveTo, Entity: 1, X: 20, Y: 20, Group: 7},
		{Kind: KindGroupMoveTo, Entity: 2, X: 20, Y: 20, Group: 7},
	})

	e1 := w.entities[indexOfEntity(w.entities, 1)]
	e2 := w.entities[indexOfEntity(w.entities, 2)]
	if e1.CommandGroup == 0 || e2.CommandGroup == 0 {
		t.Fatalf("fixture: the group move built no command group at all: e1=%d e2=%d", e1.CommandGroup, e2.CommandGroup)
	}
	if e1.CommandGroup != e2.CommandGroup {
		t.Errorf("two actors commanded by one group move hold different command groups: %d vs %d", e1.CommandGroup, e2.CommandGroup)
	}

	var records []groupAI
	for _, g := range w.groups {
		if g.group == e1.CommandGroup {
			records = append(records, g)
		}
	}
	if len(records) != 2 {
		t.Fatalf("the command group has %d record(s), want one per distinct owner (2): %+v", len(records), records)
	}
	seenOwners := map[uint32]bool{}
	for _, rec := range records {
		if rec.order != orderMove {
			t.Errorf("the record for owner %d holds order %d, want orderMove (%d)", rec.owner, rec.order, orderMove)
		}
		if rec.commandedX != 20 || rec.commandedY != 20 {
			t.Errorf("the record for owner %d holds cell (%d,%d), want the ordered (20,20)", rec.owner, rec.commandedX, rec.commandedY)
		}
		seenOwners[rec.owner] = true
	}
	if !seenOwners[9] || !seenOwners[11] {
		t.Errorf("the two records are keyed by owners %v, want 9 and 11", seenOwners)
	}
}

// -------------------------------------------------------------------- AC-5

func TestACommandedTwiceActorKeepsTheSameIDAndNothingAccumulates(t *testing.T) {
	t.Parallel()

	w := engWorld(t, Relations{}, engFighter(1, 9, 5, 5))
	before := len(w.groups)

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 8, Y: 5}})
	firstID := w.entities[indexOfEntity(w.entities, 1)].CommandGroup
	if firstID == 0 {
		t.Fatalf("fixture: the first order built no command group at all")
	}
	afterFirst := len(w.groups)
	if afterFirst != before+1 {
		t.Fatalf("the first order added %d record(s) (%d -> %d), want exactly one", afterFirst-before, before, afterFirst)
	}

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 9, Y: 6}})
	secondID := w.entities[indexOfEntity(w.entities, 1)].CommandGroup
	afterSecond := len(w.groups)

	if secondID != firstID {
		t.Errorf("a second order gave the actor id %d, want the same id %d released by the first (DD-4)", secondID, firstID)
	}
	if afterSecond != afterFirst {
		t.Errorf("the group record count moved from %d to %d on a second order to the same lone actor "+
			"— DD-4 says nothing should accumulate", afterFirst, afterSecond)
	}
}

// -------------------------------------------------------------------- AC-6

// TestNoCommandGroupIDEqualsAPlacedOrScriptNamedID is AC-6: the id a command
// builds sits above both the actor's own placed group id and the highest id
// the mission script names.
func TestNoCommandGroupIDEqualsAPlacedOrScriptNamedID(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 9, 5, 5)
	a.Group = 1
	w := engWorld(t, Relations{}, a)
	// A script naming group 3 — above the actor's own placed group (1), so a
	// floor that read only the map's own ids would miss it.
	w.script = &Script{checks: []ScriptCheck{{Op: ScriptCheckGroupCount, Group: 3, HasGroup: true}}}

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 8, Y: 5}})

	id := w.entities[indexOfEntity(w.entities, 1)].CommandGroup
	if id == 0 {
		t.Fatalf("fixture: the order built no command group at all")
	}
	if id == 1 {
		t.Errorf("the command group id %d equals the actor's own placed group id", id)
	}
	if id <= 3 {
		t.Errorf("the command group id %d does not sit above the script's own named group (3) — FR-5's floor", id)
	}
}

// -------------------------------------------------------------------- AC-7

// TestASlotlessActorGainsNoCommandGroupOrRecord is AC-7: an actor in no slot
// — owner 0, unset — is untouched by the writer, on the same ground it
// belongs to no group at all today.
func TestASlotlessActorGainsNoCommandGroupOrRecord(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, engBounds, []Entity{{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10}})
	before := len(w.groups)
	if before != 0 {
		t.Fatalf("fixture: a slotless actor's world already carries %d group record(s), want 0", before)
	}

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 8, Y: 5}})

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after.CommandGroup != 0 {
		t.Errorf("a slotless actor gained command group %d, want none", after.CommandGroup)
	}
	if len(w.groups) != before {
		t.Errorf("a slotless actor's order added a group record: %d before, %d after", before, len(w.groups))
	}
	if !after.HasTarget || after.TargetX != 8 || after.TargetY != 5 {
		t.Errorf("fixture: the move order itself did not stick: %+v", after)
	}
}

// -------------------------------------------------------------------- AC-8

func assertGroupsAscending(t *testing.T, w *World) {
	t.Helper()
	for i := 1; i < len(w.groups); i++ {
		prev, cur := w.groups[i-1], w.groups[i]
		if !(prev.owner < cur.owner || (prev.owner == cur.owner && prev.group < cur.group)) {
			t.Fatalf("w.groups is not strictly ascending at index %d: (%d,%d) does not precede (%d,%d)",
				i, prev.owner, prev.group, cur.owner, cur.group)
		}
	}
}

// TestGroupRecordsStayAscendingAndRoundTripAfterOrders is AC-8: several
// orders, over several owners and with a repeat onto an already-commanded
// actor, leave w.groups strictly ascending after every one of them, and the
// resulting world survives a round trip through its own byte form unchanged.
func TestGroupRecordsStayAscendingAndRoundTripAfterOrders(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 11, 5, 5)
	b := engFighter(2, 9, 30, 30)
	c := engFighter(3, 20, 6, 5)
	w := engWorld(t, Relations{}, a, b, c)

	// Deliberately out of owner order: entity 3's owner (20) sorts last, so
	// commanding it FIRST is what would expose an upsert that just appended
	// instead of inserting in place.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 3, X: 7, Y: 5}})
	assertGroupsAscending(t, w)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 6, Y: 5}})
	assertGroupsAscending(t, w)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 2, X: 31, Y: 30}})
	assertGroupsAscending(t, w)
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 6, Y: 6}})
	assertGroupsAscending(t, w)

	got, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(got); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, err := back.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (round trip): %v", err)
	}
	if string(again) != string(got) {
		t.Error("a world holding several command groups does not round-trip through its byte form")
	}
	if back.Hash() != w.Hash() {
		t.Error("a round-tripped world with several command groups hashes differently from the original")
	}
}
