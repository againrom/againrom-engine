package sim

import "testing"

// This file covers 0096 T2: the script's group command (opcode 6, its five
// sub-command writers) and the three engagement arms the new orders reach —
// Swarm, Move and Swarm 2's gate. T1's own AC-1, AC-11 through AC-16, AC-19
// and AC-21 already live in engage_test.go, binary_test.go and
// release_test.go; this file does not repeat them.
//
// The command tests call w.runInstant directly rather than going through a
// compiled script and a firing trigger: AC-1..AC-20 are claims about
// pkg/sim's own state machine, and a fixture that routed through
// mapload.CompileScript would be measuring the binder as well. AC-10, the
// one claim that IS about the compiled report, is proven the same way
// script_test.go already proves the others — over NewScript directly.

// setGroupCommand sets a group record's order AND commanded cell directly,
// bypassing runInstant, for the engagement-arm tests that need a group
// already under a given order without exercising the command dispatch a
// second time — release_test.go's own setGroupOrder does the same for the
// order alone; this is its two-field twin.
func setGroupCommand(t *testing.T, w *World, owner, group uint32, order uint8, cx, cy int32) {
	t.Helper()
	for i := range w.groups {
		if w.groups[i].owner == owner && w.groups[i].group == group {
			w.groups[i].order = order
			w.groups[i].commandedX, w.groups[i].commandedY = cx, cy
			return
		}
	}
	t.Fatalf("no group record for (owner %d, group %d) — this test's fixture is wrong", owner, group)
}

// ---------------------------------------------------------------- AC-2, AC-3

// TestAGroupCommandMovesExactlyOneOrder is AC-2: two groups, one command, and
// the record it names is the only thing that changes — checked over the
// WHOLE byte form, not just the two touched fields, so a command that leaked
// into a third field would fail this even if the order and cell alone looked
// right.
func TestAGroupCommandMovesExactlyOneOrder(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 2, 5, 5)
	a.Group = 10
	b := engFighter(2, 3, 30, 30)
	b.Group = 20
	w := engWorld(t, engRel(t), a, b)

	before, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 10, HasGroup: true,
		Args: [scriptParams]int32{int32(orderStandGround)}})

	if order, _, ok := w.groupState(2, 10); !ok || order != orderStandGround {
		t.Fatalf("the named record reads order=%d ok=%v, want Stand Ground", order, ok)
	}
	if order, _, ok := w.groupState(3, 20); !ok || order != orderGuard {
		t.Fatalf("the untouched record moved to order=%d, want Guard (its construction default)", order)
	}

	// Diff the marshalled bytes against a hand-built "before" world carrying
	// the SAME single change, so the comparison names more than "the two
	// bytes I already checked moved" — any other byte moving fails it too.
	wantAfter := engWorld(t, engRel(t), a, b)
	setGroupOrder(t, wantAfter, 2, 10, orderStandGround)
	got, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (after): %v", err)
	}
	want, err := wantAfter.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary (want): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("the command's own bytes are not exactly the hand-built single-field change")
	}
	if string(got) == string(before) {
		t.Errorf("the command changed nothing at all — this test would pass for the wrong reason")
	}
}

// TestTheFiveSubCommandsWriteTheOrderTheyName is AC-3's positive half: each
// of the five implemented sub-commands leaves the group under the order the
// table in plan.md names.
func TestTheFiveSubCommandsWriteTheOrderTheyName(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sub  uint8
		want uint8
	}{
		{"1 Guard", orderGuard, orderGuard},
		{"2 Swarm", orderSwarm, orderSwarm},
		{"3 Stand Ground", orderStandGround, orderStandGround},
		{"4 Move", orderMove, orderMove},
		{"5 Swarm 2", orderSwarm2, orderSwarm2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := engFighter(1, 2, 5, 5)
			a.Group = 1
			w := engWorld(t, engRel(t), a)
			w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
				Args: [scriptParams]int32{int32(tc.sub), 9, 9}})
			if order, _, ok := w.groupState(2, 1); !ok || order != tc.want {
				t.Errorf("sub-command %d leaves order=%d ok=%v, want %d", tc.sub, order, ok, tc.want)
			}
		})
	}
}

// TestAnUnimplementedSubCommandChangesNothing is AC-3's negative half: every
// value the law does not act on leaves the world bit-for-bit unchanged,
// checked over the whole byte form.
//
// Command17 is implemented by1148. Catalogue18 is inert in the original;
// 250 is outside the catalogue. Neither has a runtime handler.
func TestAnUnimplementedSubCommandChangesNothing(t *testing.T) {
	t.Parallel()

	for _, sub := range []int32{18, 250} {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			a := engFighter(1, 2, 5, 5)
			a.Group = 1
			w := engWorld(t, engRel(t), a)
			before, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
				Args: [scriptParams]int32{sub, 40, 40}})
			after, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("MarshalBinary: %v", err)
			}
			if string(before) != string(after) {
				t.Errorf("sub-command %d changed the world, want bit-for-bit unchanged", sub)
			}
		})
	}
}

// TestANodeNamingNoGroupOrAnAbsentGroupChangesNothing is AC-3's other two
// no-op cases: a node with HasGroup false, and one naming a group id no
// entity carries at all.
func TestANodeNamingNoGroupOrAnAbsentGroupChangesNothing(t *testing.T) {
	t.Parallel()

	t.Run("no group named", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		before, _ := w.MarshalBinary()
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: false,
			Args: [scriptParams]int32{int32(orderStandGround), 1, 1}})
		after, _ := w.MarshalBinary()
		if string(before) != string(after) {
			t.Error("a node naming no group changed the world")
		}
	})

	t.Run("a group no entity carries", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		before, _ := w.MarshalBinary()
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 999, HasGroup: true,
			Args: [scriptParams]int32{int32(orderStandGround), 1, 1}})
		after, _ := w.MarshalBinary()
		if string(before) != string(after) {
			t.Error("a group id no entity carries changed the world")
		}
	})
}

// ---------------------------------------------------------------------- AC-5

// TestSubCommandOneReFreezesTheNoticeBase is AC-5: after a member has moved
// since construction, sub-command 1 re-freezes the base from the group's
// PRESENT geometry, and the next Guard decision's clip uses the new number —
// not the one construction gave it.
func TestSubCommandOneReFreezesTheNoticeBase(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(2), uint32(1)
	a := engFighter(1, owner, 5, 5)
	a.Group = group
	w := engWorld(t, engRel(t), a)

	_, constructedBase, ok := w.groupState(owner, group)
	if !ok {
		t.Fatal("the fixture's own construction-time record is missing")
	}

	// Widen the member's own reach so a re-freeze must move the base. 18 and
	// not something larger: the sight march's own window is a FIXED 20-cell
	// half-width regardless of ScanRange (sightHalf in sight.go) — a wider
	// ScanRange widens the notice base just as well, but a candidate placed
	// out where only a wide ScanRange could matter would sit outside the
	// window and be invisible however the base moved, which would make this
	// fixture prove nothing.
	i := indexOfEntity(w.entities, 1)
	w.entities[i].ScanRange = 18

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderGuard)}})

	order, gotBase, ok := w.groupState(owner, group)
	if !ok || order != orderGuard {
		t.Fatalf("sub-command 1 left order=%d ok=%v, want Guard", order, ok)
	}
	wantBase := noticeBase(w.entities, []int{i}, w.entities[i].X, w.entities[i].Y)
	if gotBase != wantBase {
		t.Fatalf("the re-frozen base is %d, want %d (the present geometry's own answer)", gotBase, wantBase)
	}
	if gotBase == constructedBase {
		t.Fatalf("the base did not move at all (%d) — this test's own fixture does not separate "+
			"the two geometries", gotBase)
	}

	// And the NEXT decision's clip uses the new base, not the old one: a
	// candidate between the old radius and the new one — and safely inside
	// the sight window — is acquired.
	oldRadius := int64(noticeRadius(constructedBase))
	newRadius := int64(noticeRadius(gotBase))
	const candDist = 16
	if candDist <= oldRadius || candDist > newRadius || candDist >= 20 {
		t.Fatalf("the candidate distance %d does not sit strictly between the old radius %d and the "+
			"new radius %d, inside the sight window — this test's own arithmetic is wrong", candDist, oldRadius, newRadius)
	}
	cand := engFighter(2, 9, 5+candDist, 5)
	w2 := engWorld(t, engRel(t, [3]uint32{owner, 9, 1}), w.entities[i], cand)
	setGroupOrder(t, w2, owner, group, orderGuard)
	// Force the re-frozen base directly, as the command above already
	// proved it produces.
	for k := range w2.groups {
		if w2.groups[k].owner == owner && w2.groups[k].group == group {
			w2.groups[k].base = gotBase
		}
	}
	w2.engagementPass()
	if v, held := engVictim(w2, 1); !held || v != 2 {
		t.Errorf("the re-frozen radius did not reach the candidate at distance %d: held=%v/%v", candDist, held, v)
	}
}

// -------------------------------------------------------------------- AC-6

// TestSubCommandsFourAndFiveDistributeLikeThePlayersOwnGroupMove is AC-6: the
// script's Move setter hands the group's living members to the SAME
// distribution the player's own group move performs — in formation, each
// member's own displacement from the centroid and the slowest member's rate
// term; out of formation, the bare cell and no term — and sub-command 5
// gives identical destinations under a different order.
func TestSubCommandsFourAndFiveDistributeLikeThePlayersOwnGroupMove(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(9), uint32(1)
	const slow, fast = int32(8), int32(40)

	t.Run("in formation: offset destinations and the slowest rate", func(t *testing.T) {
		t.Parallel()
		w := mustWorld(t, 1, fmBounds, []Entity{
			{ID: 1, X: 20, Y: 20, Owner: owner, Group: group, Speed: slow},
			{ID: 2, X: 21, Y: 20, Owner: owner, Group: group, Speed: fast},
		})
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
			Args: [scriptParams]int32{int32(orderMove), 21, 50}})

		want := [][2]int32{{20, 50}, {21, 50}}
		for i, e := range w.Entities() {
			if e.TargetX != want[i][0] || e.TargetY != want[i][1] || !e.HasTarget {
				t.Errorf("id %d holds destination=%v (%d,%d), want (%d,%d)",
					e.ID, e.HasTarget, e.TargetX, e.TargetY, want[i][0], want[i][1])
			}
			if e.GroupSpeed != uint8(slow) {
				t.Errorf("id %d carries a term of %d, want the slow member's %d", e.ID, e.GroupSpeed, slow)
			}
		}
		if order, _, ok := w.groupState(owner, group); !ok || order != orderMove {
			t.Errorf("the group's stored order is %d ok=%v, want Move", order, ok)
		}
	})

	t.Run("out of formation: the bare cell and no term", func(t *testing.T) {
		t.Parallel()
		w := mustWorld(t, 1, fmBounds, []Entity{
			{ID: 1, X: 20, Y: 20, Owner: owner, Group: group, Speed: slow},
			{ID: 2, X: 20, Y: 26, Owner: owner, Group: group, Speed: fast},
		})
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
			Args: [scriptParams]int32{int32(orderMove), 20, 50}})
		for _, e := range w.Entities() {
			if e.TargetX != 20 || e.TargetY != 50 || !e.HasTarget {
				t.Errorf("id %d holds destination=%v (%d,%d), want (20,50)", e.ID, e.HasTarget, e.TargetX, e.TargetY)
			}
			if e.GroupSpeed != 0 {
				t.Errorf("id %d carries a term of %d, want none out of formation", e.ID, e.GroupSpeed)
			}
		}
	})

	t.Run("sub-command 5 gives identical destinations, a different order", func(t *testing.T) {
		t.Parallel()
		w := mustWorld(t, 1, fmBounds, []Entity{
			{ID: 1, X: 20, Y: 20, Owner: owner, Group: group, Speed: slow},
			{ID: 2, X: 21, Y: 20, Owner: owner, Group: group, Speed: fast},
		})
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
			Args: [scriptParams]int32{int32(orderSwarm2), 21, 50}})
		want := [][2]int32{{20, 50}, {21, 50}}
		for i, e := range w.Entities() {
			if e.TargetX != want[i][0] || e.TargetY != want[i][1] || !e.HasTarget {
				t.Errorf("id %d holds destination=%v (%d,%d), want (%d,%d)",
					e.ID, e.HasTarget, e.TargetX, e.TargetY, want[i][0], want[i][1])
			}
		}
		if order, _, ok := w.groupState(owner, group); !ok || order != orderSwarm2 {
			t.Errorf("the group's stored order is %d ok=%v, want Swarm 2", order, ok)
		}
	})
}

// -------------------------------------------------------------------- AC-7

// TestASwarmCommandGivesNoDestinationUntilTheNextDecision is AC-7: issuing
// sub-command 2 alone gives no member a destination; the NEXT decision, with
// no candidate anywhere, sends every member to the commanded cell unoffset —
// two members standing apart converge on the SAME cell.
func TestASwarmCommandGivesNoDestinationUntilTheNextDecision(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(9), uint32(1)
	a := engFighter(1, owner, 5, 5)
	a.Group = group
	b := engFighter(2, owner, 15, 5)
	b.Group = group
	// No relation at all: candidates() is empty for this group on every
	// decision, so the walk below is provably "scored nothing" and not a
	// side effect of an engagement.
	w := engWorld(t, engRel(t), a, b)

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderSwarm), 30, 30}})

	for _, id := range []EntityID{1, 2} {
		e := entityAt(t, w, id)
		if e.HasTarget {
			t.Fatalf("entity %d got a destination the moment Swarm was commanded: (%d,%d)",
				id, e.TargetX, e.TargetY)
		}
	}
	if order, _, ok := w.groupState(owner, group); !ok || order != orderSwarm {
		t.Fatalf("the command left order=%d ok=%v, want Swarm", order, ok)
	}

	cmdToDecision(w)
	Step(w, nil)

	for _, id := range []EntityID{1, 2} {
		e := entityAt(t, w, id)
		if !e.HasTarget || e.TargetX != 30 || e.TargetY != 30 {
			t.Errorf("entity %d holds destination=%v (%d,%d) after the decision, want (30,30) unoffset",
				id, e.HasTarget, e.TargetX, e.TargetY)
		}
	}
}

// -------------------------------------------------------------------- AC-8

// TestSwarmEngagesWhatGuardWouldHaveClipped is AC-8: the same group, the same
// candidate standing past the notice circle but inside sight, is taken under
// Swarm and released under Guard.
func TestSwarmEngagesWhatGuardWouldHaveClipped(t *testing.T) {
	t.Parallel()

	// build takes its OWN t (the calling subtest's), never the outer
	// function's: a helper that closed over the outer t and then called
	// Fatalf on it from inside a t.Parallel() subtest's own goroutine would
	// panic testing's own goroutine tracking rather than fail cleanly.
	build := func(t *testing.T, order uint8) (*World, EntityID) {
		t.Helper()
		// The member's ScanRange at CONSTRUCTION freezes the notice base: a lone
		// member's base is max(minimalGuardRange, 0+ScanRange), so building at
		// engSight=5 freezes base=8, radius=12. Widening ScanRange AFTER
		// construction moves live SIGHT alone — noticeBase is never recomputed
		// off a tick path — so this fixture can put a candidate past the frozen
		// clip and still inside sight, which no lone-member fixture built at one
		// ScanRange throughout can do (a lone member's own radius is always
		// ScanRange+noticeMargin, at least as wide as its own sight).
		m := engFighter(1, 2, 5, 5)
		cand := engFighter(2, 3, 5+15, 5) // distance 15: past a 12-radius clip
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, cand)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].ScanRange = 18 // sight alone; comfortably inside the march's window
		if _, base, ok := w.groupState(2, 0); !ok || base != 8 {
			t.Fatalf("the fixture's own frozen base is %d, want 8 (radius 12) — this test's arithmetic assumes it", base)
		}
		setGroupOrder(t, w, 2, 0, order)
		return w, 2
	}

	t.Run("guard clips it away", func(t *testing.T) {
		t.Parallel()
		w, cand := build(t, orderGuard)
		w.engagementPass()
		if v, held := engVictim(w, 1); held {
			t.Fatalf("guard acquired candidate %v/%v — this fixture must clip it for the contrast to mean anything", v, held)
		}
		_ = cand
	})

	t.Run("swarm engages it", func(t *testing.T) {
		t.Parallel()
		w, cand := build(t, orderSwarm)
		w.engagementPass()
		if v, held := engVictim(w, 1); !held || v != cand {
			t.Errorf("swarm holds victim %v/%v, want %d acquired — swarm clips nothing", v, held, cand)
		}
	})
}

// -------------------------------------------------------------------- AC-9

// TestMoveScoresOnlyArrivedMembers is AC-9: a member still holding a
// destination takes no decision at all, even with a candidate inside its
// reach; once arrived it engages that candidate; and a candidate one cell
// past reach is refused exactly as it would be under Stand Ground.
func TestMoveScoresOnlyArrivedMembers(t *testing.T) {
	t.Parallel()

	t.Run("still walking: no decision at all", func(t *testing.T) {
		t.Parallel()
		m := engFighter(1, 2, 5, 5)
		cand := engFighter(2, 3, 6, 5) // adjacent: inside reach
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, cand)
		setGroupOrder(t, w, 2, 0, orderMove)
		i := indexOfEntity(w.entities, 1)
		w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 40, 40, true
		w.entities[i].Stall = 2
		before := w.entities[i]

		w.decide(aiGroup{owner: 2, group: 0, members: []int{i}})

		after := w.entities[indexOfEntity(w.entities, 1)]
		if after != before {
			t.Errorf("a still-walking Move member's fields moved though it should take no decision:\n"+
				" before %+v\n after  %+v", before, after)
		}
	})

	t.Run("arrived: engages the candidate", func(t *testing.T) {
		t.Parallel()
		m := engFighter(1, 2, 5, 5)
		cand := engFighter(2, 3, 6, 5)
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, cand)
		setGroupOrder(t, w, 2, 0, orderMove)
		i := indexOfEntity(w.entities, 1)

		w.decide(aiGroup{owner: 2, group: 0, members: []int{i}})

		if v, held := engVictim(w, 1); !held || v != 2 {
			t.Errorf("an arrived Move member holds victim %v/%v, want entity 2 engaged", v, held)
		}
	})

	t.Run("past reach: refused exactly as under Stand Ground", func(t *testing.T) {
		t.Parallel()
		build := func(order uint8) *World {
			m := engFighter(1, 2, 5, 5)
			cand := engFighter(2, 3, 5+groupScorerReach+1, 5) // one cell past reach
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, cand)
			setGroupOrder(t, w, 2, 0, order)
			return w
		}
		move := build(orderMove)
		move.decide(aiGroup{owner: 2, group: 0, members: []int{indexOfEntity(move.entities, 1)}})
		stand := build(orderStandGround)
		stand.decide(aiGroup{owner: 2, group: 0, members: []int{indexOfEntity(stand.entities, 1)}})

		mv, mvHeld := engVictim(move, 1)
		sg, sgHeld := engVictim(stand, 1)
		if mvHeld {
			t.Errorf("Move acquired a candidate past reach: %v", mv)
		}
		if sgHeld {
			t.Errorf("Stand Ground acquired a candidate past reach: %v — this fixture's own control is wrong", sg)
		}
		if mvHeld != sgHeld {
			t.Errorf("Move and Stand Ground disagree on a candidate past reach: held=%v vs held=%v", mvHeld, sgHeld)
		}
	})
}

// ------------------------------------------------------------------- AC-10

// TestTheGapReportNamesTheSubCommand is AC-10: a script authoring an
// implemented and an unimplemented sub-command reports exactly one gap,
// naming the unimplemented one.
func TestTheGapReportNamesTheSubCommand(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1)},
		[]ScriptInstant{
			{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
				Args: [scriptParams]int32{int32(orderMove), 5, 5}}, // implemented
			// Catalogue18 remains inert; command17 has a runtime handler.
			{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
				Args: [scriptParams]int32{18, 5, 5}}, // not implemented
		}, nil)

	gaps := s.Unsupported()
	want := []ScriptGap{{Kind: ScriptGapInstant, Op: ScriptInstantGroupOrder, Index: 1, Sub: 18}}
	if len(gaps) != len(want) || gaps[0] != want[0] {
		t.Fatalf("Unsupported() is %+v, want %+v", gaps, want)
	}
}

// ------------------------------------------------------------------- AC-17

// TestSwarm2SeesOnlyACorpseAndRunsItsOwnBody is AC-17: a group under Swarm 2
// whose only visible candidate is a corpse runs its OWN body — the ordinary,
// non-empty branch — rather than Move's arm: the corpse keeps the candidate
// list non-empty (candidates() moves the dead list back when the live one is
// empty), so the whole-count gate above does not fire. No member gets a
// destination either way, so the corpse is placed PAST REACH: Move's arm
// would refuse it (Stand Ground's own reach veto) where Swarm 2's own
// ordinary cost does not, which is what tells the two branches apart.
func TestSwarm2SeesOnlyACorpseAndRunsItsOwnBody(t *testing.T) {
	t.Parallel()

	m := engFighter(1, 5, 5, 5)
	m.ScanRange = engSight
	c := engFighter(2, 9, 5+groupScorerReach+1, 5) // past reach, inside sight
	c.HP, c.MaxHP = 0, 5                           // downed: candidates() carries it as a corpse
	w := engWorld(t, engRel(t, [3]uint32{5, 9, 1}), m, c)
	setGroupOrder(t, w, 5, 0, orderSwarm2)

	i := indexOfEntity(w.entities, 1)
	w.decide(aiGroup{owner: 5, group: 0, members: []int{i}})

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after.HasTarget {
		t.Errorf("a Swarm 2 member scoring only a corpse got a destination: (%d,%d)",
			after.TargetX, after.TargetY)
	}
	if !after.HasAttackTarget || after.AttackTarget != 2 {
		t.Errorf("a Swarm 2 member did not engage the corpse past reach (HasAttackTarget=%v "+
			"AttackTarget=%v) — a gate reading the LIVING candidates alone would find this list "+
			"empty and take Move's arm instead, whose reach-vetoing scorer refuses anything past "+
			"reach; the corpse fallback keeps candidates() non-empty and Swarm 2 must run its own "+
			"ordinary-cost body, which does not veto by distance",
			after.HasAttackTarget, after.AttackTarget)
	}
}

// ------------------------------------------------------------------- AC-18

// TestSwarm2LeavesAnAllVetoedGroupUntouchedWhereSwarmWalks is AC-18: a group
// whose one visible candidate the preference table vetoes outright — a
// ground member against a flier — leaves every field untouched under Swarm
// 2 (it does not walk), where the identical fixture under Swarm sends the
// member to the commanded cell.
func TestSwarm2LeavesAnAllVetoedGroupUntouchedWhereSwarmWalks(t *testing.T) {
	t.Parallel()

	build := func(order uint8) *World {
		m := engFighter(1, 5, 5, 5)
		flier := engFighter(2, 9, 6, 5) // adjacent, so only the domain veto explains no engagement
		flier.Domain = DomainAir
		w := engWorld(t, engRel(t, [3]uint32{5, 9, 1}), m, flier)
		setGroupCommand(t, w, 5, 0, order, 40, 40)
		return w
	}

	t.Run("swarm 2: untouched, no walk", func(t *testing.T) {
		t.Parallel()
		w := build(orderSwarm2)
		before := w.entities[indexOfEntity(w.entities, 1)]
		w.decide(aiGroup{owner: 5, group: 0, members: []int{indexOfEntity(w.entities, 1)}})
		after := w.entities[indexOfEntity(w.entities, 1)]
		if after != before {
			t.Errorf("a Swarm 2 member with every candidate vetoed moved:\n before %+v\n after  %+v", before, after)
		}
	})

	t.Run("swarm: walks to the commanded cell", func(t *testing.T) {
		t.Parallel()
		w := build(orderSwarm)
		w.decide(aiGroup{owner: 5, group: 0, members: []int{indexOfEntity(w.entities, 1)}})
		after := w.entities[indexOfEntity(w.entities, 1)]
		if !after.HasTarget || after.TargetX != 40 || after.TargetY != 40 {
			t.Errorf("a Swarm member with every candidate vetoed holds destination=%v (%d,%d), want (40,40)",
				after.HasTarget, after.TargetX, after.TargetY)
		}
	})
}

// ------------------------------------------------------------------- AC-20

func TestAMoveCommandedGroupKeepsDecidingWhileItWalks(t *testing.T) {
	t.Parallel()

	const owner, group = uint32(9), uint32(1)
	m := engFighter(1, owner, 5, 5)
	m.Group = group
	m.Speed = 30
	cand := engFighter(2, 3, 6, 20) // adjacent to the ordered destination alone
	w := engWorld(t, engRel(t, [3]uint32{owner, 3, 1}), m, cand)

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: group, HasGroup: true,
		Args: [scriptParams]int32{int32(orderMove), 5, 20}})

	if e := entityAt(t, w, 1); !e.HasTarget {
		t.Fatal("the move command did not give the member a destination — this fixture's own setup is wrong")
	}

	// One tick later, still walking: the group still names it.
	Step(w, nil)
	found := false
	for _, g := range w.aiGroups() {
		if g.owner != owner || g.group != group {
			continue
		}
		for _, mi := range g.members {
			if w.entities[mi].ID == 1 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("a member walking under its own group's Move order is excluded from aiGroups — DD-18's skip is too wide")
	}

	// Run to arrival AND the end of the crossing it was paying for: HasTarget
	// clears on the tick the mover reaches its cell, but a rated mover keeps
	// owing transit ticks past that (step.go's own transit-first rule), and
	// arrived() — moveArm's own arrival test — is !HasTarget && Transit==0.
	// Scoring the moment HasTarget clears alone would catch a decision phase
	// that lands mid-crossing, where the member is correctly not yet scored.
	settled := false
	for i := 0; i < 20000; i++ {
		Step(w, nil)
		e := entityAt(t, w, 1)
		if !e.HasTarget && e.Transit == 0 {
			settled = true
			break
		}
	}
	if !settled {
		t.Fatal("the member never finished arriving within the tick budget")
	}
	for i := 0; i <= scriptCycle; i++ {
		Step(w, nil)
	}
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Errorf("on arrival the member holds victim %v/%v, want entity 2 engaged", v, held)
	}

	// Contrast — REVERSED BY 0117, and said so explicitly rather than edited
	// quietly: until 0117 a unit under a PLAYER's own move order stayed
	// excluded from aiGroups (0097's own rule), because the actor itself never
	// left the GUARD or STAND GROUND group the map placed it in — neither of
	// which issues a destination, so the narrowed skip above never admitted it.
	// In bounds, unlike the group above's own destination need not be: an
	// off-map order gives up on its own first tick (step.go's far-search
	// failure), which clears HasTarget again before aiGroups ever gets asked
	// and would make this contrast pass for the wrong reason.
	p := engFighter(3, 2, 5, 5)
	pw := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), p, engFighter(4, 3, 6, 5))
	Step(pw, []Command{{Kind: KindMoveTo, Entity: 3, X: 5, Y: 20}})
	if e := entityAt(t, pw, 3); !e.HasTarget {
		t.Fatal("the contrast fixture's own player order did not stick — this test would pass for the wrong reason")
	}
	foundPlayerCommanded := false
	for _, g := range pw.aiGroups() {
		for _, mi := range g.members {
			if pw.entities[mi].ID == 3 {
				foundPlayerCommanded = true
			}
		}
	}
	if !foundPlayerCommanded {
		t.Fatal("a unit under a player's own move order is no longer named by aiGroups — 0117 FR-4's " +
			"command group should keep it visible under the move order, on DD-18's own terms, exactly " +
			"as the script-commanded group above stays visible")
	}
}
