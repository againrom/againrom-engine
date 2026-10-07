package sim

// The walk home: the guard stance's own tail (0106 T3, engage.go's
// walkHome).
//
// Every fixture here builds a member OFF its own post before deciding for
// it — never on the cell it was constructed at — because a constructor posts
// every entity where it stands, so a fixture that never moves a member could
// not tell "the walk home fired" apart from "nothing happened to it" (plan.md
// note 4). No test here reads a game install (SC-4), and none draws from
// w.rng (SC-1) — decide() and walkHome read only entity fields and w.groups.

import "testing"

// -------------------------------------------------------------------- AC-1

func TestAnOffPostGuardWithNoCandidatesWalksHome(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	i := indexOfEntity(w.entities, 1)
	// Off its own post: the constructor anchored it at (8,5); this stance
	// command's own re-anchor never ran, so overriding the field directly is
	// what a member that walked away under some earlier order, with nothing
	// since re-anchoring it, looks like.
	w.entities[i].PostX, w.entities[i].PostY = 5, 5
	rngBefore := w.rng

	w.decide(w.aiGroups()[0])

	got := w.entities[indexOfEntity(w.entities, 1)]
	if !got.HasTarget || got.TargetX != 5 || got.TargetY != 5 {
		t.Errorf("destination is HasTarget=%v (%d,%d), want the post (5,5) held", got.HasTarget, got.TargetX, got.TargetY)
	}
	if got.HasAttackTarget {
		t.Errorf("a member with nothing to see acquired a victim")
	}
	if got.Stall != 0 {
		t.Errorf("stall count is %d, want 0 — clearOrder drops it with the rest of the old order", got.Stall)
	}
	if r := w.routes[indexOfEntity(w.entities, 1)]; len(r) != 0 {
		t.Errorf("stored route is %v, want none — the walk home writes a destination directly and searches no route", r)
	}
	if w.rng != rngBefore {
		t.Errorf("the walk home drew from w.rng: before %+v, after %+v", rngBefore, w.rng)
	}
}

// -------------------------------------------------------------------- AC-2

func TestAnArrivedGuardIsLeftInEveryField(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	before := w.entities[indexOfEntity(w.entities, 1)]
	if before.X != before.PostX || before.Y != before.PostY {
		t.Fatalf("fixture: the member is not on its own post — got (%d,%d) post (%d,%d)",
			before.X, before.Y, before.PostX, before.PostY)
	}

	w.decide(w.aiGroups()[0])

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after != before {
		t.Errorf("an at-post, victimless member changed:\n before %+v\n after  %+v", before, after)
	}
}

// TestAWalkedHomeMemberDoesNotReIssueOnTheNextDecision is AC-2's other half:
// once a member has walked home and arrived — HasTarget cleared by the
// ordinary move loop reaching the target, exactly as any other destination
// is consumed — a second, later decision does not re-issue the same walk or
// touch any other field.
func TestAWalkedHomeMemberDoesNotReIssueOnTheNextDecision(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 5, 5
	w.decide(w.aiGroups()[0])
	if got := w.entities[indexOfEntity(w.entities, 1)]; !got.HasTarget || got.TargetX != 5 || got.TargetY != 5 {
		t.Fatalf("fixture: the first decision did not walk the member home — got HasTarget=%v (%d,%d)",
			got.HasTarget, got.TargetX, got.TargetY)
	}

	// Arrive by hand rather than by stepping: what AC-2 is about is the
	// SECOND decision, not the walk itself, which post_test.go's and this
	// file's own AC-1 case already cover.
	w.entities[i].X, w.entities[i].Y = 5, 5
	w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 0, 0, false
	arrived := w.entities[indexOfEntity(w.entities, 1)]

	w.decide(w.aiGroups()[0])

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after != arrived {
		t.Errorf("a second decision at the post changed the member:\n before %+v\n after  %+v", arrived, after)
	}
}

// -------------------------------------------------------------------- AC-3

func TestAGuardHoldingAVictimGainsNoDestinationEvenOffPost(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), engFighter(1, 2, 8, 5), engFighter(2, 3, 9, 5))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 5, 5

	w.decide(w.aiGroups()[0])

	got := w.entities[indexOfEntity(w.entities, 1)]
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("fixture: the member did not acquire the adjacent candidate — victim %v held=%v", v, held)
	}
	if got.HasTarget {
		t.Errorf("a member holding a victim gained a destination: (%d,%d)", got.TargetX, got.TargetY)
	}
	if got.X != 8 || got.Y != 5 {
		t.Errorf("the member moved from (8,5) to (%d,%d) inside a decision", got.X, got.Y)
	}
}

// -------------------------------------------------------------------- AC-4

func TestAReleasedOffPostGuardIsWalkedHomeInTheSameTick(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 5, 5
	w.entities[i].AttackTarget, w.entities[i].HasAttackTarget = 99, true

	w.decide(w.aiGroups()[0])

	got := w.entities[indexOfEntity(w.entities, 1)]
	if got.HasAttackTarget {
		t.Fatalf("fixture: the release did not fire — the member still holds a victim")
	}
	if !got.HasTarget || got.TargetX != 5 || got.TargetY != 5 {
		t.Errorf("destination is HasTarget=%v (%d,%d), want the post (5,5) held — release then walk home",
			got.HasTarget, got.TargetX, got.TargetY)
	}
}

// -------------------------------------------------------------------- AC-5

// TestAGuardsOwnOffPostDestinationIsReplacedByItsPost is AC-5: a member
// already holding a destination of its own — not its post — has that
// destination REPLACED, not merged or appended to, by clearOrder then the
// post.
func TestAGuardsOwnOffPostDestinationIsReplacedByItsPost(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 5, 5
	w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 20, 20, true
	w.entities[i].Stall = 3
	w.routes[i] = []cell{{9, 5}, {10, 5}}

	w.decide(aiGroup{owner: 2, group: 0, members: []int{i}})

	got := w.entities[indexOfEntity(w.entities, 1)]
	if !got.HasTarget || got.TargetX != 5 || got.TargetY != 5 {
		t.Errorf("destination is HasTarget=%v (%d,%d), want the post (5,5) — the old (20,20) replaced, not kept",
			got.HasTarget, got.TargetX, got.TargetY)
	}
	if got.Stall != 0 {
		t.Errorf("stall count is %d, want 0 — an order replacement discards it", got.Stall)
	}
	if r := w.routes[indexOfEntity(w.entities, 1)]; len(r) != 0 {
		t.Errorf("stored route is %v, want none — an order replacement discards it too", r)
	}
}

// -------------------------------------------------------------------- AC-6

func TestAStandGroundMemberGainsNothingEvenOffPost(t *testing.T) {
	t.Parallel()

	w := engWorld(t, engRel(t), engFighter(1, 2, 8, 5))
	setGroupOrder(t, w, 2, 0, orderStandGround)
	i := indexOfEntity(w.entities, 1)
	w.entities[i].PostX, w.entities[i].PostY = 5, 5
	before := w.entities[indexOfEntity(w.entities, 1)]

	w.decide(w.aiGroups()[0])

	after := w.entities[indexOfEntity(w.entities, 1)]
	if after != before {
		t.Errorf("an off-post stand-ground member changed:\n before %+v\n after  %+v", before, after)
	}

	for k := 0; k < 64; k++ {
		Step(w, nil)
	}
	if final := w.entities[indexOfEntity(w.entities, 1)]; final.X != before.X || final.Y != before.Y {
		t.Errorf("an off-post stand-ground member moved from (%d,%d) to (%d,%d) over 64 ticks",
			before.X, before.Y, final.X, final.Y)
	}
}

func TestNoNonGuardStanceWalksHomeOffTheLoopFoot(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		order uint8
	}{
		{"Stand Ground", orderStandGround},
		{"Swarm 2", orderSwarm2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flier := engFighter(2, 3, 9, 5)
			flier.Domain = DomainAir
			w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}), engFighter(1, 2, 8, 5), flier)
			setGroupOrder(t, w, 2, 0, tc.order)
			i := indexOfEntity(w.entities, 1)
			w.entities[i].PostX, w.entities[i].PostY = 5, 5
			before := w.entities[i]

			w.decide(w.aiGroups()[0])

			after := w.entities[indexOfEntity(w.entities, 1)]
			if after.HasAttackTarget {
				t.Fatalf("fixture: the member scored the candidate, so it never reaches the "+
					"state the walk home acts on — victim %d", after.AttackTarget)
			}
			if after.HasTarget {
				t.Errorf("a %s member off its post gained a destination (%d,%d) — only the guard "+
					"stance reads the post (FR-8, FR-9)", tc.name, after.TargetX, after.TargetY)
			}
			if after != before {
				t.Errorf("a %s member off its post changed:\n before %+v\n after  %+v", tc.name, before, after)
			}
		})
	}
}
