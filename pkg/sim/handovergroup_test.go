package sim

import "testing"

// What a hand-over does to an actor's GROUP, beside what it does to its owner.
// The owner half is scriptowner_test.go's; nothing here re-measures it except
// where a case would otherwise pass by writing nothing at all.

// groupsOf reads the placed group and the command-group overlay back by id.
func groupsOf(w *World) map[EntityID][2]uint32 {
	out := make(map[EntityID][2]uint32, len(w.entities))
	for _, e := range w.Entities() {
		out[e.ID] = [2]uint32{e.Group, e.CommandGroup}
	}
	return out
}

// TestTheGroupArmGivesEachMemberAGroupOfItsOwn is 0159 AC-3.
//
// Three entities stand in group 4. After the hand-over all three are under the
// new slot, none of them is still in group 4, and no two of them share a group.
// That last clause is the one that separates this build from one that moved the
// whole group across intact: a single fresh group for all three would satisfy
// every other assertion here.
func TestTheGroupArmGivesEachMemberAGroupOfItsOwn(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
	}, 0)
	w := ownerWorld(t, s)
	runPass(t, w)

	got := groupsOf(w)
	seen := map[uint32]EntityID{}
	for _, id := range []EntityID{1, 2, 3} {
		g := got[id][0]
		if g == 4 {
			t.Errorf("entity %d is still in group 4 after the hand-over", id)
		}
		if other, dup := seen[g]; dup {
			t.Errorf("entities %d and %d share group %d; each is owed one of its own",
				other, id, g)
		}
		seen[g] = id
	}
	// The two entities the arm did not name keep the group they were built in,
	// so "every group changed" cannot pass this.
	for _, id := range []EntityID{4, 5} {
		if got[id][0] != 7 {
			t.Errorf("entity %d left group 7 for %d and the arm never named it", id, got[id][0])
		}
	}
	// And the owner half still holds, so a group write that lost the owner write
	// is not a pass.
	for _, id := range []EntityID{1, 2, 3} {
		if o := owners(w)[id]; o != 9 {
			t.Errorf("entity %d is under slot %d, want 9", id, o)
		}
	}
}

// TestAHandOverClearsTheCommandGroup is 0159 AC-4.
//
// The named entity carries a command-group overlay when the trigger fires. The
// overlay is the group word every reader that asks the EFFECTIVE group reads
// first, so leaving it standing would keep the actor inside the group it was
// just removed from.
func TestAHandOverClearsTheCommandGroup(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 2, HasUnit: true, Player: 9, HasPlayer: true},
	}, 0)
	w := scriptWorld(t, s, []Entity{
		{ID: 1, HP: 5, MaxHP: 5, Group: 4, Owner: 1},
		{ID: 2, HP: 5, MaxHP: 5, Group: 4, Owner: 2, CommandGroup: 77},
	})
	runPass(t, w)

	got := groupsOf(w)
	if got[2][1] != 0 {
		t.Errorf("entity 2 kept command group %d across the hand-over", got[2][1])
	}
	if got[2][0] == 4 {
		t.Error("entity 2 is still in group 4 after the hand-over")
	}
	if got[1] != [2]uint32{4, 0} {
		t.Errorf("entity 1 is at %v and the arm never named it", got[1])
	}
}

// TestAnAbsentUnitReferenceWritesNoGroupEither is 0159 AC-5.
//
// The unit arm already refused an absent reference for the owner write; this
// says the group write did not become a second, unguarded one below it.
func TestAnAbsentUnitReferenceWritesNoGroupEither(t *testing.T) {
	t.Parallel()

	for _, in := range []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 4242, HasUnit: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveUnit, Unit: 2, HasUnit: true},
		{Op: ScriptInstantGiveGroup, Group: 4242, HasGroup: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveGroup, Player: 9, HasPlayer: true},
	} {
		w := ownerWorld(t, fireOnce(t, []ScriptInstant{in}, 0))
		before := groupsOf(w)
		beforeOwners := owners(w)
		runPass(t, w)
		for id, g := range groupsOf(w) {
			if g != before[id] {
				t.Errorf("%v moved entity %d from group %v to %v", in.Op, id, before[id], g)
			}
		}
		for id, o := range owners(w) {
			if o != beforeOwners[id] {
				t.Errorf("%v moved entity %d from slot %d to %d", in.Op, id, beforeOwners[id], o)
			}
		}
	}
}

func TestASecondHandOverKeepsTheSlotAndTakesASecondGroup(t *testing.T) {
	t.Parallel()

	w := scriptWorld(t, nil, []Entity{
		{ID: 1, HP: 5, MaxHP: 5, Group: 4, Owner: 1},
	})
	w.handOver(0, 9)
	first := w.entities[0].Group
	w.handOver(0, 9)
	second := w.entities[0].Group

	if w.entities[0].Owner != 9 {
		t.Errorf("after two hand-overs the slot is %d, want 9", w.entities[0].Owner)
	}
	if second == first {
		t.Errorf("the second hand-over reused group %d; each call takes a fresh one", first)
	}
	if first == 4 || second == 4 {
		t.Error("a hand-over left the actor in the group it started in")
	}
}

// TestAFreshGroupIsAboveEveryGroupTheScriptNames guards the allocator's floor:
// a hand-over must not put an actor into a group a later script node counts,
// which is what would happen if the fresh id were simply "one more than the
// highest in use".
func TestAFreshGroupIsAboveEveryGroupTheScriptNames(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Unit: 1, HasUnit: true, Player: 9, HasPlayer: true},
		{Op: ScriptInstantGiveGroup, Group: 4242, HasGroup: true, Player: 8, HasPlayer: true},
	}, 0)
	w := scriptWorld(t, s, []Entity{{ID: 1, HP: 5, MaxHP: 5, Group: 4, Owner: 1}})
	runPass(t, w)

	if g := w.entities[0].Group; g <= 4242 {
		t.Errorf("the fresh group is %d, which the script's own nodes reach; want above 4242", g)
	}
}

// A handed-over actor stands in a new group at order 0, and order 0 gives the
// member to its own actor state: an enemy inside the guard block is engaged
// without an order from the player.
func TestAHandedOverGuardEngagesAnEnemyWithoutAnOrder(t *testing.T) {
	t.Parallel()

	build := func(t *testing.T) *World {
		t.Helper()
		a := engFighter(1, 6, 10, 10)
		a.Group = 1
		foe := engFighter(2, 7, 13, 10)
		rel := engRel(t, [3]uint32{1, 7, relationHostile}, [3]uint32{7, 1, relationHostile},
			[3]uint32{6, 7, relationHostile}, [3]uint32{7, 6, relationHostile})
		w, err := NewRelatedWorld(1, pxBounds, ModeCanonical, Terrain{}, []Entity{a, foe}, nil, rel)
		if err != nil {
			t.Fatalf("NewRelatedWorld: %v", err)
		}
		return w
	}
	engaged := func(w *World) bool {
		for range 40 {
			Step(w, nil)
			if entityAt(t, w, 1).HasAttackTarget {
				return true
			}
		}
		return false
	}

	t.Run("handed over", func(t *testing.T) {
		t.Parallel()
		w := build(t)
		w.handOver(indexOfEntity(w.entities, 1), 1)
		e := entityAt(t, w, 1)
		if order, _, ok := w.groupState(1, e.Group); !ok || order != orderNone {
			t.Fatalf("the new group (1,%d) holds order %d, record %v; want a record at order 0", e.Group, order, ok)
		}
		if !engaged(w) {
			t.Error("a handed-over guard did not engage an enemy three cells away within 40 ticks")
		}
	})

	t.Run("loss control: no record for the new group", func(t *testing.T) {
		t.Parallel()
		w := build(t)
		w.handOver(indexOfEntity(w.entities, 1), 1)
		e := entityAt(t, w, 1)
		for i := len(w.groups) - 1; i >= 0; i-- {
			if w.groups[i].owner == 1 && w.groups[i].group == e.Group {
				w.groups = append(w.groups[:i], w.groups[i+1:]...)
			}
		}
		if engaged(w) {
			t.Error("the actor engaged with no group record, so the test does not measure the record")
		}
	})
}
