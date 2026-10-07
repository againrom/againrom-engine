package sim

import "testing"

// The group-count arm: what it measures, what it refuses to measure, and the
// win chain it unblocks.

// groupCheck is one check of the arm, naming a group.
func groupCheck(reg int32, group uint32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckGroupCount, Register: reg, Group: group, HasGroup: true}
}

// killGroupMember puts one entity below zero health, which is dead by every
// reading. It writes the field rather than resolving a blow, because what is
// under test is the count and not the fight.
func killGroupMember(t *testing.T, w *World, id EntityID) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("no entity %d", id)
	}
	w.entities[i].HP = -1
}

// downGroupMember puts one entity at exactly zero against a positive maximum:
// this tree's third state, which the original does not have and which falls on
// the dead side of every script arm.
func downGroupMember(t *testing.T, w *World, id EntityID) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("no entity %d", id)
	}
	w.entities[i].HP = 0
}

// TestTheArmCountsTheGroupsLivingMembers is AC-3 and SC-5.
//
// The transition THROUGH zero is what is asserted, not a single value. Every
// shipped use of this arm compares the count against zero, so a build that
// counted the whole membership would be right about the first case here and
// wrong about the one that matters — and a test pinning one number could not
// tell the two apart.
func TestTheArmCountsTheGroupsLivingMembers(t *testing.T) {
	t.Parallel()

	// Three members of group 3, two of group 8, and one of group 0 — which is a
	// group like any other and is here so that a build treating it as "no group"
	// is caught by a count and not by an absence.
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Group: 3},
		{ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10, Group: 3},
		{ID: 3, X: 3, Y: 1, HP: 10, MaxHP: 10, Group: 3},
		{ID: 4, X: 4, Y: 1, HP: 10, MaxHP: 10, Group: 8},
		{ID: 5, X: 5, Y: 1, HP: 10, MaxHP: 10, Group: 8},
		{ID: 6, X: 6, Y: 1, HP: 10, MaxHP: 10, Group: 0},
	}
	s := mustScript(t, []ScriptCheck{
		groupCheck(0, 3),
		groupCheck(1, 8),
		groupCheck(2, 0),
		groupCheck(3, 99),
	}, nil, nil)
	w := scriptWorld(t, s, ents)

	read := func(what string, want ...int32) {
		t.Helper()
		scriptTicks(w, 16, nil)
		for r, v := range want {
			if got := w.ScriptRegister(int32(r)); got != v {
				t.Errorf("%s: register %d is %d, want %d", what, r, got, v)
			}
		}
	}
	// A second check of the arm in the same pass writes its own register, and a
	// group no entity carries answers zero rather than failing.
	read("all alive", 3, 2, 1, 0)

	// Two of the three killed, then the third. The last step is the one every
	// shipped trigger of this arm is waiting for.
	killGroupMember(t, w, 1)
	killGroupMember(t, w, 2)
	read("two of three felled", 1, 2, 1, 0)
	killGroupMember(t, w, 3)
	read("the group wiped", 0, 2, 1, 0)

	// A DOWNED member — health at exactly zero against a positive maximum — is
	// out of the fight and out of the count, which is the same sense of dead
	// every other arm of this runtime uses.
	downGroupMember(t, w, 4)
	read("one of two downed", 0, 1, 1, 0)
}

// TestACheckNamingNoGroupWritesNoRegister is AC-4 and SC-4's second half.
//
// It is NOT the zero of an empty group, and the difference is the whole of why
// the compiled check carries a presence flag rather than a reserved id. The
// register is preset to a value no count of this world could produce, so a
// measurement taken would be visible; and the trigger reading it is a LIVE
// trigger, evaluated and latched like any other — this arm is implemented, so
// nothing here is inert.
func TestACheckNamingNoGroupWritesNoRegister(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{77}},
		{Op: ScriptCheckGroupCount, Register: 1},
		groupCheck(2, 5),
	}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{pair(1, 1, ScriptCmpEQ)},
		Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone},
		Latch:    0}})
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("this script holds only implemented arms; report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := scriptWorld(t, s, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 9, MaxHP: 9, Group: 5},
		{ID: 2, X: 2, Y: 1, HP: 9, MaxHP: 9, Group: 5},
	})
	// Preset register 1 to something no count of this world is.
	w.registers[1] = 77
	scriptTicks(w, 16, nil)

	if got := w.ScriptRegister(1); got != 77 {
		t.Errorf("a check naming no group wrote %d into its register, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(2); got != 2 {
		t.Errorf("the check beside it measured %d, want 2 — the arm still works", got)
	}
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading that register was not evaluated; this arm is implemented, " +
			"so nothing about it is inert")
	}
}

// TestTheFirstMissionsWinChainReachesAWin is AC-7 and SC-7.
//
// It is the SHAPE of the campaign's first mission's chain, on a hand-built
// script and a synthetic world: a guard group that has to be cleared, an
// escortee that has to arrive, and a hero that has to reach a cell with the
// variable the arrival set. No arm outside this build's supported set is needed
// to advance it — where the shipped map hands the escortee to the player, this
// moves it directly.
//
// What it witnesses is that the group count composes with the rest of the
// runtime into a mission that ENDS, which no single arm's own test can say.
func TestTheFirstMissionsWinChainReachesAWin(t *testing.T) {
	t.Parallel()

	const (
		rZero     = 0 // the constant 0 every comparison is made against
		rThree    = 1 // the constant 3, the chain's own distance
		rGuards   = 2 // how many of the guard group still stand
		rEscortee = 3 // the escortee's distance to its cell
		rHero     = 4 // the hero's distance to its cell
		rVar      = 5 // the mission variable the arrival sets
	)
	checks := []ScriptCheck{
		{Op: ScriptCheckConstant, Register: rZero},
		{Op: ScriptCheckConstant, Register: rThree, Args: [scriptParams]int32{3}},
		groupCheck(rGuards, 1),
		{Op: ScriptCheckDistance, Register: rEscortee, Unit: 10, HasUnit: true,
			Args: [scriptParams]int32{20, 20}},
		{Op: ScriptCheckDistance, Register: rHero, Unit: 20, HasUnit: true,
			Args: [scriptParams]int32{40, 40}},
		{Op: ScriptCheckVariable, Register: rVar, Args: [scriptParams]int32{50}},
	}
	instants := []ScriptInstant{
		{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{50}},
		{Op: ScriptInstantWin},
	}
	triggers := []ScriptTrigger{
		// The escortee arriving sets the variable — and the guard group being
		// wiped is what gates the step before it in the shipped map, so the
		// count is on this trigger rather than left out of the chain.
		{Pairs: [3]ScriptPair{pair(rGuards, rZero, ScriptCmpEQ), pair(rEscortee, rThree, ScriptCmpLE)},
			Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone}, Once: true, Latch: 0},
		// The hero reaching its cell with that variable set wins.
		{Pairs: [3]ScriptPair{pair(rHero, rThree, ScriptCmpLE), pair(rVar, rZero, ScriptCmpNE)},
			Instants: [4]int32{1, ScriptNone, ScriptNone, ScriptNone}, Once: true, Latch: 1},
	}
	s := mustScript(t, checks, instants, triggers)
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("the chain needs an arm this build does not have: report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := scriptWorld(t, s, []Entity{
		{ID: 1, X: 5, Y: 5, HP: 10, MaxHP: 10, Group: 1},
		{ID: 2, X: 6, Y: 5, HP: 10, MaxHP: 10, Group: 1},
		{ID: 10, X: 20, Y: 20, HP: 10, MaxHP: 10, Group: 2},
		{ID: 20, X: 40, Y: 40, HP: 10, MaxHP: 10, Group: 3},
	})
	// The escortee and the hero are already standing on their cells, so the ONLY
	// thing between this world and a win is the guard group.
	scriptTicks(w, 64, nil)
	if w.Outcome() != OutcomeUndecided {
		t.Fatalf("the mission was decided with the guard group still standing: %d", w.Outcome())
	}
	if got := w.ScriptRegister(rGuards); got != 2 {
		t.Fatalf("the guard count is %d, want 2", got)
	}

	killGroupMember(t, w, 1)
	scriptTicks(w, 32, nil)
	if w.Outcome() != OutcomeUndecided {
		t.Fatalf("one guard down decided the mission: %d", w.Outcome())
	}
	killGroupMember(t, w, 2)
	scriptTicks(w, 64, nil)

	if got := w.ScriptRegister(rGuards); got != 0 {
		t.Errorf("the guard count is %d after both were felled, want 0", got)
	}
	if won, lost := w.ScriptCounters(); won != 1 || lost != 0 {
		t.Errorf("the counters are %d/%d, want one win and no loss", won, lost)
	}
	if w.Outcome() != OutcomeWon {
		t.Errorf("the mission came to %d, want won — the chain the count unblocks does not close",
			w.Outcome())
	}
}
