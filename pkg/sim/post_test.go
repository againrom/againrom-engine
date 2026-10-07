package sim

// The post: the constructor's half (0106 T1).
//
// This file witnesses AC-7 and AC-9's constructor clause only. The stance
// setter that is the second writer (the script's group-order command) and the
// walk home that reads the post are later tasks' — see spec.md's scope. Every
// fixture here is built the way this package's other tests already are, and
// nothing reads a game install (SC-4).

import "testing"

// -------------------------------------------------------------------- AC-7

// TestAFreshWorldsPostIsEveryEntitysOwnCell is AC-7: every entity a freshly
// constructed world holds has its own cell as its post, including one in no
// group (Owner 0, which freezeGroups itself skips), one not alive, and one
// whose group's order — forced past what construction would ever give it,
// the way setGroupOrder's own doc says a T2 command will — is not a
// stance.
func TestAFreshWorldsPostIsEveryEntitysOwnCell(t *testing.T) {
	t.Parallel()

	ents := []Entity{
		// Owned, alive, ordinary: the control.
		{ID: 1, X: 3, Y: 4, HP: 10, MaxHP: 10, Owner: 1, Group: 1},
		// In no group: Owner 0, which groupKeys skips outright.
		{ID: 2, X: 7, Y: 2, HP: 10, MaxHP: 10},
		// Not alive: HP below zero.
		{ID: 3, X: 5, Y: 5, HP: -3, MaxHP: 10, Owner: 1, Group: 1},
		// Owned by a second (owner, group) pair, so its order can be forced
		// away from what construction gives it without touching entity 1's.
		{ID: 4, X: 9, Y: 1, HP: 10, MaxHP: 10, Owner: 2, Group: 1},
	}
	w := mustWorld(t, 1, Bounds{Width: 32, Height: 32}, ents)

	// Force (owner 2, group 1)'s order to orderSwarm — not a stance — the way
	// setGroupOrder's own doc says a T2 command eventually will. If writing
	// the post consulted the group's order at all, this would be the fixture
	// that would catch it.
	setGroupOrder(t, w, 2, 1, orderSwarm)

	for _, tc := range ents {
		got := entityAt(t, w, tc.ID)
		if got.PostX != tc.X || got.PostY != tc.Y {
			t.Errorf("entity %d: post is (%d,%d), want its own cell (%d,%d)",
				tc.ID, got.PostX, got.PostY, tc.X, tc.Y)
		}
	}

	// And the fixture actually exercises what it claims to: entity 4's group
	// really is under a non-stance order at the point its post was checked.
	if order, _, ok := w.groupState(2, 1); !ok || order != orderSwarm {
		t.Fatalf("fixture: (owner 2, group 1)'s order is %d ok=%v, want orderSwarm (%d) — "+
			"this case does not test a non-stance order", order, ok, orderSwarm)
	}
}

func TestTheConstructorOverwritesWhateverPostACallerNamed(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 32, Height: 32}, []Entity{
		{ID: 1, X: 4, Y: 6, HP: 10, MaxHP: 10, PostX: 999, PostY: -999},
	})

	got := entityAt(t, w, 1)
	if got.PostX != 4 || got.PostY != 6 {
		t.Errorf("post is (%d,%d), want the entity's own cell (4,6), not the (999,-999) the literal named",
			got.PostX, got.PostY)
	}
}

// -------------------------------------------------------------------- AC-9

func TestAMidCrossingEntitysPostIsItsOwnCellAtConstruction(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 32, Height: 32}, []Entity{
		// Mid-crossing: two of maxTransit ticks owed, still holding the
		// destination that put it there.
		{ID: 1, X: 6, Y: 5, HP: 10, MaxHP: 10, TargetX: 9, TargetY: 5, HasTarget: true,
			Transit: 2, TransitTotal: maxTransit},
		// Idle, holding no order at all, for contrast.
		{ID: 2, X: 1, Y: 1, HP: 10, MaxHP: 10},
	})

	if got := entityAt(t, w, 1); got.PostX != 6 || got.PostY != 5 {
		t.Errorf("the mid-crossing entity's post is (%d,%d), want its own cell (6,5)", got.PostX, got.PostY)
	}
	if got := entityAt(t, w, 2); got.PostX != 1 || got.PostY != 1 {
		t.Errorf("the idle entity's post is (%d,%d), want its own cell (1,1)", got.PostX, got.PostY)
	}
}
