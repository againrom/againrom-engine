package sim

import (
	"reflect"
	"testing"
)

// Check opcodes 11 and 13: two DEAD ARMS (TRIG-COND-003), reproduced as dead
// rather than treated as unimplemented. scriptCheckSupported is the ONLY
// place that answer is given, so these tests take it from there: both
// opcodes are supported, neither is a gap, and a trigger reading either
// register is live — evaluated against whatever the register already held,
// never poisoned and never inerted.

// deadCheck is one check of arm 11 or 13, naming neither a unit, a group nor
// a player — the two dead arms read nothing at all.
func deadCheck(reg, op int32) ScriptCheck {
	return ScriptCheck{Op: op, Register: reg}
}

// ---------------------------------------------------------------- AC-5

// TestBothDeadArmsAreSupportedAndReportNoGap is SC-1 and AC-5's first two
// clauses: scriptCheckSupported answers true for both opcodes directly, and a
// script naming either in a check reports no gap for it.
func TestBothDeadArmsAreSupportedAndReportNoGap(t *testing.T) {
	t.Parallel()

	if !scriptCheckSupported(ScriptCheckDead11) {
		t.Error("ScriptCheckDead11 (11) is not in scriptCheckSupported")
	}
	if !scriptCheckSupported(ScriptCheckDead13) {
		t.Error("ScriptCheckDead13 (13) is not in scriptCheckSupported")
	}

	s := mustScript(t, []ScriptCheck{
		deadCheck(0, ScriptCheckDead11),
		deadCheck(1, ScriptCheckDead13),
	}, nil, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("Unsupported() is %+v, want none — both opcodes are dead, not unimplemented", gaps)
	}
}

// TestADeadArmWritesNoRegisterAndItsTriggerIsLive is AC-5 end to end: the
// register a dead-arm check owns keeps exactly the value it held before the
// pass, and the trigger reading it is still evaluated — which an inert
// trigger could not be, latch or no latch, no matter what the register held.
func TestADeadArmWritesNoRegisterAndItsTriggerIsLive(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		deadCheck(0, ScriptCheckDead11),
		deadCheck(1, ScriptCheckDead13),
	}, nil, []ScriptTrigger{
		// Each pair compares a register against ITSELF: always EQ, so the
		// latch moves if and only if the trigger was evaluated at all — an
		// inert trigger is skipped before its latch is ever touched, on any
		// register value whatsoever (script.go's scriptPass).
		{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(), Latch: 0},
		{Pairs: [3]ScriptPair{pair(1, 1, ScriptCmpEQ)}, Instants: acts(), Latch: 1},
	})
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("this script holds only supported arms; report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := scriptWorld(t, s, nil)
	// The instant set: register 0 to the value opcode 11's check would have
	// to leave alone, register 1 to the value opcode 13's check would have
	// to leave alone. Nothing in this script writes either again.
	w.registers[0] = 42
	w.registers[1] = 43
	scriptTicks(w, 16, nil)

	if got := w.ScriptRegister(0); got != 42 {
		t.Errorf("opcode 11's register is %d after a pass, want the preset 42 left untouched", got)
	}
	if got := w.ScriptRegister(1); got != 43 {
		t.Errorf("opcode 13's register is %d after a pass, want the preset 43 left untouched", got)
	}
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading opcode 11's register did not latch; a dead arm's reader is not inert")
	}
	if !w.ScriptLatched(1) {
		t.Errorf("the trigger reading opcode 13's register did not latch; a dead arm's reader is not inert")
	}
}

// TestDeadArmsAreDistinguishedFromAGenuinelyUnimplementedCheck puts a dead
// arm beside scriptCheckSentinelOp — the suite's own "a real arm this build
// does not evaluate" (script_test.go)
// — in one script: the dead arm's reader is live and the dead arm is not a
// gap, while the unimplemented arm's reader is inert and IS the one gap
// reported. Same script, same pass, so the two answers cannot be confused for
// each other by accident.
func TestDeadArmsAreDistinguishedFromAGenuinelyUnimplementedCheck(t *testing.T) {
	t.Parallel()

	const notAnArm = scriptCheckSentinelOp

	s := mustScript(t, []ScriptCheck{
		deadCheck(0, ScriptCheckDead11),
		deadCheck(1, ScriptCheckDead13),
		{Op: notAnArm, Register: 2},
	}, nil, []ScriptTrigger{
		{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(), Latch: 0},
		{Pairs: [3]ScriptPair{pair(1, 1, ScriptCmpEQ)}, Instants: acts(), Latch: 1},
		{Pairs: [3]ScriptPair{pair(2, 2, ScriptCmpEQ)}, Instants: acts(), Latch: 2},
	})

	gaps := s.Unsupported()
	want := []ScriptGap{{Kind: ScriptGapCheck, Op: notAnArm, Index: 2}}
	if !reflect.DeepEqual(sortedGaps(gaps), want) {
		t.Fatalf("Unsupported() is %+v, want %+v — neither dead arm is a gap", gaps, want)
	}
	if inert := s.InertTriggers(); !reflect.DeepEqual(inert, []int32{2}) {
		t.Fatalf("InertTriggers() is %v, want [2] — only the unimplemented arm's reader is inert", inert)
	}

	w := scriptWorld(t, s, nil)
	scriptTicks(w, 16, nil)
	if !w.ScriptLatched(0) {
		t.Errorf("the opcode-11 trigger did not latch; it is not inert")
	}
	if !w.ScriptLatched(1) {
		t.Errorf("the opcode-13 trigger did not latch; it is not inert")
	}
	if w.ScriptLatched(2) {
		t.Errorf("the check-9 trigger latched; it should be inert and never evaluated")
	}
}
