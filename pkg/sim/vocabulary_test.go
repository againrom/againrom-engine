package sim

import "testing"

// This file covers 0146: the four group command kinds the PLAYER reaches — the
// move he already had, the two cell-free stances, the aimed march and the aimed
// patrol — and the rule that any of them ends a patrol.
//
// It drives Step with command slices rather than runInstant, because that is
// the surface this story adds: the script's own six arms are 0096's and 0099's
// and are not re-proven here.

// vocabUnit is a plain owned unit at (x, y). It carries no scan range, so
// nothing engages and every assertion below is about the command and not about
// a fight it happened to start.
func vocabUnit(id EntityID, owner uint32, x, y int32) Entity {
	return Entity{ID: id, X: x, Y: y, Owner: owner, HP: 20, MaxHP: 20, Speed: 20}
}

// commandGroupOf is the one command group every member of a stated set is in,
// or a failure naming what it found instead.
func commandGroupOf(t *testing.T, w *World, ids ...EntityID) uint32 {
	t.Helper()
	var id uint32
	for k, want := range ids {
		i := indexOfEntity(w.entities, want)
		if i < 0 {
			t.Fatalf("entity %d is not in the world", want)
		}
		got := w.entities[i].CommandGroup
		if got == 0 {
			t.Fatalf("entity %d holds no command group", want)
		}
		if k == 0 {
			id = got
			continue
		}
		if got != id {
			t.Fatalf("entity %d is in command group %d, want %d — one order, one group", want, got, id)
		}
	}
	return id
}

func TestGuardPutsTheSelectionInOneGroupAndAnchorsItsPosts(t *testing.T) {
	t.Parallel()

	a, b, c := vocabUnit(1, SelfSlot, 5, 5), vocabUnit(2, SelfSlot, 6, 5), vocabUnit(3, SelfSlot, 7, 6)
	a.Group, b.Group, c.Group = 1, 2, 3
	w := engWorld(t, engRel(t), a, b, c)

	Step(w, []Command{
		{Kind: KindGroupStance, Entity: 1, X: OrderGuard, Group: 7},
		{Kind: KindGroupStance, Entity: 2, X: OrderGuard, Group: 7},
		{Kind: KindGroupStance, Entity: 3, X: OrderGuard, Group: 7},
	})

	id := commandGroupOf(t, w, 1, 2, 3)
	order, base, ok := w.groupState(SelfSlot, id)
	if !ok || order != orderGuard {
		t.Fatalf("the command group reads order=%d ok=%v, want Guard (%d)", order, ok, orderGuard)
	}
	if base == 0 {
		t.Fatal("the command group's notice base is 0 — it was never frozen")
	}
	for _, e := range w.entities {
		if e.PostX != e.X || e.PostY != e.Y {
			t.Fatalf("entity %d stands at (%d,%d) with its post at (%d,%d), want the post anchored where it stands",
				e.ID, e.X, e.Y, e.PostX, e.PostY)
		}
	}
}

func TestStandGroundIsItsOwnOrderOverTheSameShape(t *testing.T) {
	t.Parallel()

	a, b := vocabUnit(1, SelfSlot, 9, 9), vocabUnit(2, SelfSlot, 10, 9)
	w := engWorld(t, engRel(t), a, b)

	Step(w, []Command{
		{Kind: KindGroupStance, Entity: 1, X: OrderStandGround, Group: 1},
		{Kind: KindGroupStance, Entity: 2, X: OrderStandGround, Group: 1},
	})

	id := commandGroupOf(t, w, 1, 2)
	if order, _, ok := w.groupState(SelfSlot, id); !ok || order != orderStandGround {
		t.Fatalf("the command group reads order=%d ok=%v, want Stand Ground (%d)", order, ok, orderStandGround)
	}
}

func TestAStanceCommandRefusesAnyOrderButTheTwo(t *testing.T) {
	t.Parallel()

	for _, order := range []int32{0, 2, 4, 5, 14, 99, -1} {
		w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 4, 4))
		before, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		Step(w, []Command{{Kind: KindGroupStance, Entity: 1, X: order, Group: 1}})
		after, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		// The reference takes an EMPTY advance, so the two forms are compared
		// with the tick raised on both: a Step always raises it and this test is
		// not about that.
		ref := mustUnmarshal(t, before)
		Step(ref, nil)
		want, err := ref.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		if string(after) != string(want) {
			t.Fatalf("a stance command carrying order %d changed the world; it names no arm and must change nothing", order)
		}
	}
}

// mustUnmarshal decodes a byte form or fails.
func mustUnmarshal(t *testing.T, b []byte) *World {
	t.Helper()
	var w World
	if err := w.UnmarshalBinary(b); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	return &w
}

func TestMarchIsMoveWithTheOtherOrderByte(t *testing.T) {
	t.Parallel()

	build := func(kind uint8) *World {
		w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5), vocabUnit(2, SelfSlot, 6, 5))
		Step(w, []Command{
			{Kind: kind, Entity: 1, X: 20, Y: 20, Group: 1},
			{Kind: kind, Entity: 2, X: 20, Y: 20, Group: 1},
		})
		return w
	}

	march, move := build(KindGroupSwarmTo), build(KindGroupMoveTo)

	id := commandGroupOf(t, march, 1, 2)
	if order, _, ok := march.groupState(SelfSlot, id); !ok || order != orderSwarm2 {
		t.Fatalf("the marched group reads order=%d ok=%v, want Swarm 2 (%d)", order, ok, orderSwarm2)
	}
	if x, y, ok := march.groupCommandedCell(SelfSlot, id); !ok || x != 20 || y != 20 {
		t.Fatalf("the marched group's commanded cell is (%d,%d) ok=%v, want (20,20)", x, y, ok)
	}
	for k := range march.entities {
		m, v := march.entities[k], move.entities[k]
		if !m.HasTarget {
			t.Fatalf("entity %d holds no destination after a march", m.ID)
		}
		if m.TargetX != v.TargetX || m.TargetY != v.TargetY {
			t.Fatalf("entity %d marched to (%d,%d) and moved to (%d,%d) — the distribution must be the same one",
				m.ID, m.TargetX, m.TargetY, v.TargetX, v.TargetY)
		}
	}
}

func TestPatrolBuildsARingFromWhereEachMemberStands(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5), vocabUnit(2, SelfSlot, 8, 12))
	Step(w, []Command{
		{Kind: KindGroupPatrolTo, Entity: 1, X: 30, Y: 31, Group: 4},
		{Kind: KindGroupPatrolTo, Entity: 2, X: 30, Y: 31, Group: 4},
	})

	id := commandGroupOf(t, w, 1, 2)
	if order, _, ok := w.groupState(SelfSlot, id); !ok || order != orderNone {
		t.Fatalf("the patrolling group reads order=%d ok=%v, want none (%d)", order, ok, orderNone)
	}
	want := []struct{ hx, hy int32 }{{5, 5}, {8, 12}}
	for k, e := range w.entities {
		if e.ActorState != actorStatePatrol {
			t.Fatalf("entity %d is in actor state %d, want patrol (%d)", e.ID, e.ActorState, actorStatePatrol)
		}
		if e.PatrolHeadX != want[k].hx || e.PatrolHeadY != want[k].hy {
			t.Fatalf("entity %d's ring head is (%d,%d), want the cell it stood on (%d,%d)",
				e.ID, e.PatrolHeadX, e.PatrolHeadY, want[k].hx, want[k].hy)
		}
		if e.PatrolTailX != 30 || e.PatrolTailY != 31 {
			t.Fatalf("entity %d's ring tail is (%d,%d), want the ordered cell (30,31)", e.ID, e.PatrolTailX, e.PatrolTailY)
		}
		if e.PatrolLeg != patrolLegTail {
			t.Fatalf("entity %d stands on leg %d, want the tail (%d)", e.ID, e.PatrolLeg, patrolLegTail)
		}
	}
}

func TestPatrolClampsTheFarCellIntoTheMap(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5))
	Step(w, []Command{{Kind: KindGroupPatrolTo, Entity: 1, X: 9999, Y: -4, Group: 1}})

	e := w.entities[0]
	if e.PatrolTailX >= engBounds.Width || e.PatrolTailY < 0 {
		t.Fatalf("the ring tail is (%d,%d), which is outside a %dx%d map",
			e.PatrolTailX, e.PatrolTailY, engBounds.Width, engBounds.Height)
	}
}

func TestAnyPlayerOrderEndsAPatrol(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cmd  Command
	}{
		{"a plain move", Command{Kind: KindMoveTo, Entity: 1, X: 20, Y: 20}},
		{"a group move", Command{Kind: KindGroupMoveTo, Entity: 1, X: 20, Y: 20, Group: 1}},
		{"a march", Command{Kind: KindGroupSwarmTo, Entity: 1, X: 20, Y: 20, Group: 1}},
		{"guard", Command{Kind: KindGroupStance, Entity: 1, X: OrderGuard, Group: 1}},
		{"stand ground", Command{Kind: KindGroupStance, Entity: 1, X: OrderStandGround, Group: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5))
			Step(w, []Command{{Kind: KindGroupPatrolTo, Entity: 1, X: 30, Y: 30, Group: 9}})
			if w.entities[0].ActorState != actorStatePatrol {
				t.Fatal("fixture: the unit is not patrolling")
			}

			Step(w, []Command{tc.cmd})
			// TWO WHOLE DECISION PHASES, not one advance: the actor pass runs
			// on one tick in sixteen, so a build that had left the ring in
			// place would read exactly as this one does until that tick comes
			// round. This is the reason the check is here rather than on the
			// fields alone.
			engRun(w, 2)

			e := w.entities[0]
			if e.ActorState != actorStateGuard {
				t.Fatalf("the unit is still in actor state %d after %s, want guard (%d)",
					e.ActorState, tc.name, actorStateGuard)
			}
			if e.PatrolHeadX != 0 || e.PatrolHeadY != 0 || e.PatrolTailX != 0 || e.PatrolTailY != 0 ||
				e.PatrolLeg != patrolLegHead {
				t.Fatalf("the ring survived %s: head (%d,%d) tail (%d,%d) leg %d",
					tc.name, e.PatrolHeadX, e.PatrolHeadY, e.PatrolTailX, e.PatrolTailY, e.PatrolLeg)
			}
		})
	}
}

// TestAMoveOverAPatrollerReachesTheOrderedCell is AC-5's own observable half:
// not that the fields were cleared, but that the unit ARRIVES where the player
// sent it. It is what the defect actually looked like — the order applied, the
// unit set off, and one decision phase later it turned round and went back to
// its ring.
func TestAMoveOverAPatrollerReachesTheOrderedCell(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5))
	Step(w, []Command{{Kind: KindGroupPatrolTo, Entity: 1, X: 40, Y: 40, Group: 1}})
	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 1, X: 5, Y: 12, Group: 2}})
	engRun(w, 6)

	e := w.entities[0]
	if e.X != 5 || e.Y != 12 {
		t.Fatalf("the unit stands at (%d,%d), want the ordered cell (5,12) — the actor pass took the order back",
			e.X, e.Y)
	}
}

func TestAGroupOrderCollectsByKindAndTag(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5), vocabUnit(2, SelfSlot, 6, 5))
	Step(w, []Command{
		{Kind: KindGroupStance, Entity: 1, X: OrderGuard, Group: 3},
		{Kind: KindGroupMoveTo, Entity: 2, X: 20, Y: 20, Group: 3},
	})

	one, two := w.entities[0].CommandGroup, w.entities[1].CommandGroup
	if one == 0 || two == 0 {
		t.Fatalf("command groups are %d and %d, want both nonzero", one, two)
	}
	if one == two {
		t.Fatalf("both units landed in command group %d — a stance and a move sharing a tag must not merge", one)
	}
	if order, _, ok := w.groupState(SelfSlot, one); !ok || order != orderGuard {
		t.Fatalf("the stance group reads order=%d ok=%v, want Guard", order, ok)
	}
	if order, _, ok := w.groupState(SelfSlot, two); !ok || order != orderMove {
		t.Fatalf("the move group reads order=%d ok=%v, want Move", order, ok)
	}
}

func TestEveryGroupKindIsDispatched(t *testing.T) {
	t.Parallel()

	for _, kind := range []uint8{KindGroupMoveTo, KindGroupSwarmTo, KindGroupStance, KindGroupPatrolTo} {
		if !isGroupKind(kind) {
			t.Fatalf("kind %d is a group kind and isGroupKind refuses it", kind)
		}
		w := engWorld(t, engRel(t), vocabUnit(1, SelfSlot, 5, 5))
		Step(w, []Command{{Kind: kind, Entity: 1, X: OrderGuard, Y: 7, Group: 1}})
		if w.entities[0].CommandGroup == 0 {
			t.Fatalf("kind %d was consumed by the group arm and built no command group", kind)
		}
	}
	for _, kind := range []uint8{KindMoveTo, KindKill, KindDamage, KindAttack, KindEquip, KindCast, 10, 200} {
		if isGroupKind(kind) {
			t.Fatalf("kind %d is not a group kind and isGroupKind admits it", kind)
		}
	}
}
