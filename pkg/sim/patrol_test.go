package sim

// The Patrol sub-command, the actor pass and the patrol arm (0099 T2).
//
// Every fixture here is built the way this package's other engagement and
// script tests already are — engFighter/engWorld/engRel, or a hand-built
// world over a compiled script — and nothing reads a game install.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestSubCommand14IsSupportedAndReachesTheArm(t *testing.T) {
	t.Parallel()

	s := mustScript(t, nil, []ScriptInstant{
		{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
			Args: [scriptParams]int32{subCommandPatrol, 9, 9}},
	}, nil)
	if got := s.Unsupported(); len(got) != 0 {
		t.Fatalf("the compile-time report names sub-command 14 unsupported: %+v", got)
	}

	a := engFighter(1, 2, 5, 5)
	a.Group = 1
	w := engWorld(t, engRel(t), a)
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 9, 9}})

	e := entityAt(t, w, 1)
	if e.ActorState != actorStatePatrol {
		t.Fatalf("entity 1's actor state is %d after sub-command 14, want patrol (%d) — "+
			"the dispatch did not reach the arm", e.ActorState, actorStatePatrol)
	}
}

func TestAPatrolNodeNamingNoGroupOrAnAbsentGroupChangesNothing(t *testing.T) {
	t.Parallel()

	t.Run("no group named", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		before, _ := w.MarshalBinary()
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, HasGroup: false,
			Args: [scriptParams]int32{subCommandPatrol, 9, 9}})
		after, _ := w.MarshalBinary()
		if string(before) != string(after) {
			t.Error("a Patrol node naming no group changed the world")
		}
	})

	t.Run("a group no entity carries", func(t *testing.T) {
		t.Parallel()
		a := engFighter(1, 2, 5, 5)
		a.Group = 1
		w := engWorld(t, engRel(t), a)
		before, _ := w.MarshalBinary()
		w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 999, HasGroup: true,
			Args: [scriptParams]int32{subCommandPatrol, 9, 9}})
		after, _ := w.MarshalBinary()
		if string(before) != string(after) {
			t.Error("a group id no entity carries changed the world")
		}
	})
}

func TestPatrolClearsTheGroupsOrderAndLeavesTheCommandedCellAlone(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 2, 5, 5)
	a.Group = 1
	w := engWorld(t, engRel(t), a)
	setGroupCommand(t, w, 2, 1, orderSwarm, 77, 88)

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 9, 9}})

	order, _, ok := w.groupState(2, 1)
	if !ok || order != orderNone {
		t.Fatalf("the group's order is %d ok=%v, want orderNone (0)", order, ok)
	}
	cx, cy, ok := w.groupCommandedCell(2, 1)
	if !ok || cx != 77 || cy != 88 {
		t.Errorf("the commanded cell is (%d,%d) ok=%v, want the untouched (77,88)", cx, cy, ok)
	}
}

func TestPatrolStopsAMemberHoldingEverything(t *testing.T) {
	t.Parallel()

	victim := engFighter(2, 3, 6, 5)
	a := engFighter(1, 2, 5, 5)
	a.Group = 1
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), a, victim)
	i := indexOfEntity(w.entities, 1)

	// The victim first: orderAttack itself calls clearOrder, so the
	// destination, the stall count and the route below must be set AFTER
	// it or this fixture would prove nothing about them.
	w.orderAttack(i, 2)
	w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 40, 40, true
	w.entities[i].Stall = 3
	w.routes[i] = []cell{{x: 6, y: 5}, {x: 7, y: 5}}
	w.entities[i].GroupSpeed = 12

	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 9, 9}})

	e := w.entities[i]
	if e.HasTarget {
		t.Errorf("the member still holds a destination (%d,%d)", e.TargetX, e.TargetY)
	}
	if e.Stall != 0 {
		t.Errorf("the member still holds a stall count of %d", e.Stall)
	}
	if len(w.routes[i]) != 0 {
		t.Errorf("the member still holds a stored route of %d cell(s)", len(w.routes[i]))
	}
	if e.HasAttackTarget {
		t.Errorf("the member still holds a victim (%d)", e.AttackTarget)
	}
	if e.GroupSpeed != 0 {
		t.Errorf("the member still carries a group rate term of %d", e.GroupSpeed)
	}
}

func TestPatrolBuildsARingPerMemberFromWhereItStands(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 9, 5, 5)
	a.Group = 1
	b := engFighter(2, 9, 30, 40)
	b.Group = 1
	w := engWorld(t, engRel(t), a, b)

	// The node's cell is off the map on the X axis, so this also witnesses
	// D-5's clamp: the ring the command builds is in bounds by construction.
	nodeX, nodeY := engBounds.Width+50, int32(9)
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, nodeX, nodeY}})

	wantTailX, wantTailY := w.bounds.clamp(nodeX, nodeY)
	if wantTailX == nodeX {
		t.Fatalf("the node's own X (%d) is already in bounds — this fixture does not exercise the clamp", nodeX)
	}

	e1 := entityAt(t, w, 1)
	e2 := entityAt(t, w, 2)
	for _, tc := range []struct {
		name   string
		e      Entity
		hx, hy int32
	}{
		{"member 1", e1, 5, 5},
		{"member 2", e2, 30, 40},
	} {
		if tc.e.PatrolHeadX != tc.hx || tc.e.PatrolHeadY != tc.hy {
			t.Errorf("%s: head is (%d,%d), want its own cell (%d,%d)",
				tc.name, tc.e.PatrolHeadX, tc.e.PatrolHeadY, tc.hx, tc.hy)
		}
		if tc.e.PatrolTailX != wantTailX || tc.e.PatrolTailY != wantTailY {
			t.Errorf("%s: tail is (%d,%d), want the clamped node cell (%d,%d)",
				tc.name, tc.e.PatrolTailX, tc.e.PatrolTailY, wantTailX, wantTailY)
		}
		if tc.e.PatrolLeg != patrolLegTail {
			t.Errorf("%s: leg is %d, want tail (%d)", tc.name, tc.e.PatrolLeg, patrolLegTail)
		}
	}
	if e1.PatrolHeadX == e2.PatrolHeadX && e1.PatrolHeadY == e2.PatrolHeadY {
		t.Fatalf("both members' heads are (%d,%d) — this fixture does not separate them",
			e1.PatrolHeadX, e1.PatrolHeadY)
	}
}

func TestArmPatrolsThreeSteps(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 2, 5, 5)
	w := engWorld(t, engRel(t), a)
	i := indexOfEntity(w.entities, 1)

	e := &w.entities[i]
	e.ActorState = actorStatePatrol
	e.PatrolHeadX, e.PatrolHeadY = 5, 5
	e.PatrolTailX, e.PatrolTailY = 9, 5
	e.PatrolLeg = patrolLegTail
	// Residue step 1 must clear: a stall count and a stored route left
	// over from something else.
	e.Stall = 4
	w.routes[i] = []cell{{x: 6, y: 5}}

	// The actor stands on the head, not the tail (its current waypoint):
	// the leg must not advance, and the destination is the tail.
	w.armPatrol(i)
	got := w.entities[i]
	if got.Stall != 0 || len(w.routes[i]) != 0 {
		t.Fatalf("step 1 did not clear residue: stall=%d route=%v", got.Stall, w.routes[i])
	}
	if got.PatrolLeg != patrolLegTail {
		t.Fatalf("the leg advanced to %d while the actor stood off its waypoint, want it to stay tail (%d)",
			got.PatrolLeg, patrolLegTail)
	}
	if !got.HasTarget || got.TargetX != 9 || got.TargetY != 5 {
		t.Fatalf("the destination is holding=%v (%d,%d), want (9,5)", got.HasTarget, got.TargetX, got.TargetY)
	}

	// Walk the actor onto its waypoint by hand — the arrival test is
	// position equality, so this alone provokes it, with no route search
	// needed — and call armPatrol again: the leg advances to the head and
	// the destination becomes it.
	w.entities[i].X, w.entities[i].Y = 9, 5
	w.armPatrol(i)
	got = w.entities[i]
	if got.PatrolLeg != patrolLegHead {
		t.Fatalf("arriving at the tail left the leg at %d, want it to advance to head (%d)",
			got.PatrolLeg, patrolLegHead)
	}
	if !got.HasTarget || got.TargetX != 5 || got.TargetY != 5 {
		t.Fatalf("after the arrival flip the destination is holding=%v (%d,%d), want (5,5)",
			got.HasTarget, got.TargetX, got.TargetY)
	}

	// A THIRD call, still standing at (9,5) but with the leg now at head
	// (whose cell is (5,5)), must NOT flip again: the actor is off its
	// (new) current waypoint.
	w.armPatrol(i)
	if w.entities[i].PatrolLeg != patrolLegHead {
		t.Errorf("the leg flipped a second time while the actor still stood off its new waypoint")
	}
}

func TestAPatrollerWalksTheRing(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 9, 5, 5)
	a.Group = 1
	w := engWorld(t, engRel(t), a)
	w.runInstant(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 1, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 15, 5}})

	reachedTail, reachedHeadAgain := false, false
	for i := 0; i < 20000 && !reachedHeadAgain; i++ {
		Step(w, nil)
		e := entityAt(t, w, 1)
		if e.X == 15 && e.Y == 5 {
			reachedTail = true
		}
		if reachedTail && e.X == 5 && e.Y == 5 {
			reachedHeadAgain = true
		}
	}
	if !reachedTail {
		t.Fatal("the patroller never reached its tail (15,5)")
	}
	if !reachedHeadAgain {
		t.Fatal("the patroller reached its tail but never walked back to its head (5,5)")
	}
}

func TestOrderNoneIsNotDecidedOver(t *testing.T) {
	t.Parallel()

	t.Run("a hostile candidate in plain sight is not scored", func(t *testing.T) {
		t.Parallel()
		m := engFighter(1, 2, 5, 5)
		cand := engFighter(2, 3, 6, 5) // adjacent: inside reach, well inside sight
		w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), m, cand)
		setGroupOrder(t, w, 2, 0, orderNone)
		i := indexOfEntity(w.entities, 1)
		before := w.entities[i]

		w.engagementPass()

		after := w.entities[i]
		if after != before {
			t.Errorf("a member of an order-0 group moved under the engagement decision:\n before %+v\n after  %+v",
				before, after)
		}
	})

	t.Run("an empty candidate list does not release it either", func(t *testing.T) {
		t.Parallel()
		m := engFighter(1, 2, 5, 5)
		w := engWorld(t, engRel(t), m) // no relation at all: candidates() is empty
		setGroupOrder(t, w, 2, 0, orderNone)
		i := indexOfEntity(w.entities, 1)
		// A victim it should not hold in this state, poked in directly so a
		// spurious releaseAttack call is visible rather than a no-op.
		w.entities[i].AttackTarget, w.entities[i].HasAttackTarget = 2, true

		w.engagementPass()

		if !w.entities[i].HasAttackTarget {
			t.Error("a member of an order-0 group with an empty candidate list was released by the engagement decision")
		}
	})
}

// ------------------------------------------------------------------- SC-1

// TestAnEntityInAnUnknownActorStateIsLeftInEveryField is SC-1's
// behavioural half: an actor state actorPass's switch has no case for
// reaches no arm, exactly the "not implemented, not merely inert"
// treatment script.go's own dispatch gives an unimplemented opcode — the
// consequence that makes deleting the one case in actor.go behaviour-free
// rather than merely something that compiles.
func TestAnEntityInAnUnknownActorStateIsLeftInEveryField(t *testing.T) {
	t.Parallel()

	a := engFighter(1, 2, 5, 5)
	w := engWorld(t, engRel(t), a)
	i := indexOfEntity(w.entities, 1)
	w.entities[i].ActorState = 0xff // no arm this build has a case for
	before := w.entities[i]

	w.actorPass()

	if w.entities[i] != before {
		t.Errorf("an entity in an unhandled actor state was changed by actorPass:\n before %+v\n after  %+v",
			before, w.entities[i])
	}
}

// TestTheActorPassSwitchHasNoDefaultArm is SC-1's structural half,
// asserted on the source itself: D-1's seam is that the dispatch has no
// default, so deleting every case leaves a switch with none — still
// compiling, still reaching no arm for any state. Go gives no runtime
// reflection over a switch statement's own cases, so this parses actor.go
// and inspects the AST rather than asserting a property nothing else in
// this repo's tests can check honestly.
//
// THE NUMBER OF CASES IS NOT ASSERTED AND IS NOT IN THIS TEST'S NAME.
// It was both until 0167, which added two arms and had to edit the count
// and the name together — the same staleness a test named after a version
// number carries. What D-1 claims is the absence of a default, and that
// claim does not move when an arm is added.
func TestTheActorPassSwitchHasNoDefaultArm(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "actor.go", nil, 0)
	if err != nil {
		t.Fatalf("parsing actor.go: %v", err)
	}

	var sw *ast.SwitchStmt
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "actorPass" {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if s, ok := n.(*ast.SwitchStmt); ok && sw == nil {
				sw = s
			}
			return true
		})
	}
	if sw == nil {
		t.Fatal("actorPass holds no switch statement — the seam SC-1 asks about is gone")
	}
	if len(sw.Body.List) == 0 {
		t.Fatal("the switch holds no clause at all — nothing dispatches")
	}
	for i, cl := range sw.Body.List {
		if cc, ok := cl.(*ast.CaseClause); !ok || cc.List == nil {
			t.Errorf("clause %d is a default — SC-1 asks for no default arm", i)
		}
	}
}
