package sim

// The mission script runtime. Every fixture is built here out of compiled
// records; nothing reads a map, and nothing outside the standard library is
// imported.

import (
	"bytes"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------- fixtures

func mustScript(t *testing.T, cs []ScriptCheck, is []ScriptInstant, ts []ScriptTrigger) *Script {
	t.Helper()
	s, err := NewScript(cs, is, ts)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

func scriptWorld(t *testing.T, s *Script, ents []Entity) *World {
	t.Helper()
	w, err := NewScriptedWorld(1, Bounds{Width: 80, Height: 80}, ModeCanonical, nil, ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w
}

// scriptTicks advances w n ticks, issuing cmds on the first of them alone.
func scriptTicks(w *World, n int, cmds []Command) {
	for i := 0; i < n; i++ {
		if i == 0 {
			Step(w, cmds)
			continue
		}
		Step(w, nil)
	}
}

// pair is one used condition pair.
func pair(left, right, cmp int32) ScriptPair {
	return ScriptPair{Left: left, Right: right, Cmp: cmp, Used: true}
}

// acts fills a trigger's four instant slots, unused ones first cleared to none.
func acts(is ...int32) [4]int32 {
	out := [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone}
	copy(out[:], is)
	return out
}

// constCheck is the build-time constant form: a register preset to v.
func constCheck(reg, v int32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckConstant, Register: reg, Args: [scriptParams]int32{v}}
}

// distCheck is check arm 7 over a unit and a point.
func distCheck(reg int32, unit EntityID, x, y int32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckDistance, Register: reg, Unit: unit, HasUnit: true,
		Args: [scriptParams]int32{x, y}}
}

// ---------------------------------------------------------------- the pass

// TestThePassRunsOncePerFullTickOnItsOwnPhase is AC-4. The script is evaluated
// on phase 6 of the sixteen-tick cycle and on no other, so a condition that
// becomes true at tick 0 is not seen until tick 6 and a trigger that could fire
// on every tick fires once every sixteen.
func TestThePassRunsOncePerFullTickOnItsOwnPhase(t *testing.T) {
	// A repeating trigger with no condition at all: it fires on every pass and
	// on nothing else, so the counter IS the pass count.
	s := mustScript(t,
		nil,
		[]ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{7}}},
		[]ScriptTrigger{{Instants: acts(0), Once: false, Latch: 0}})

	w := scriptWorld(t, s, []Entity{{ID: 1}})
	for tick := 0; tick < 40; tick++ {
		before := w.ScriptRegister(7)
		Step(w, nil)
		after := w.ScriptRegister(7)
		ranPass := after != before
		wantPass := tick%16 == 6
		if ranPass != wantPass {
			t.Errorf("tick %d: the pass %v, want %v (phase %d)",
				tick, map[bool]string{true: "ran", false: "did not run"}[ranPass],
				map[bool]string{true: "it to", false: "it not to"}[wantPass], tick%16)
		}
	}
	// Forty ticks hold phases 6, 22 and 38: three passes.
	if got := w.ScriptRegister(7); got != 3 {
		t.Errorf("after 40 ticks the pass has run %d time(s), want 3", got)
	}
}

// TestThePassPrecedesTheMovement is AC-4's second half: a condition measures the
// world as the tick FOUND it, not as the tick left it. A unit one step from the
// cell it is walking to is measured at its old cell on the tick it arrives.
func TestThePassPrecedesTheMovement(t *testing.T) {
	// Fires when the unit is exactly on (5,5): distance == 0.
	s := mustScript(t,
		[]ScriptCheck{distCheck(0, 1, 5, 5), constCheck(1, 0)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})

	// It stands at (5,6) and is ordered to (5,5), so it arrives on tick 0 — six
	// ticks before the first pass. What decides the test is the REGISTER at the
	// pass, which must be the distance from where it stood when the pass ran.
	w := scriptWorld(t, s, []Entity{{ID: 1, X: 5, Y: 6}})
	Step(w, []Command{{Entity: 1, X: 5, Y: 5}})
	if got := w.Entities()[0]; got.X != 5 || got.Y != 5 {
		t.Fatalf("after one tick the unit is at (%d,%d), want (5,5)", got.X, got.Y)
	}
	if won, _ := w.ScriptCounters(); won != 0 {
		t.Fatalf("the win counter moved on tick 0, before any pass")
	}
	scriptTicks(w, 6, nil) // ticks 1..6, the last of them the pass
	if got := w.ScriptRegister(0); got != 0 {
		t.Errorf("the distance register is %d at the pass, want 0", got)
	}
	if won, _ := w.ScriptCounters(); won != 1 {
		t.Errorf("the win counter is %d after the first pass, want 1", won)
	}
}

// ---------------------------------------------------------------- comparison

// TestTheSixComparisonCodes is AC-6: the alphabet, in the order the engine's own
// table puts it, and everything above it permanently false.
func TestTheSixComparisonCodes(t *testing.T) {
	// Registers 0 and 1 are set by two constants, so the comparison is the only
	// thing under test.
	table := []struct {
		code       int32
		a, b       int32
		wantFiring bool
	}{
		{ScriptCmpEQ, 5, 5, true}, {ScriptCmpEQ, 5, 6, false},
		{ScriptCmpNE, 5, 6, true}, {ScriptCmpNE, 5, 5, false},
		{ScriptCmpGT, 6, 5, true}, {ScriptCmpGT, 5, 5, false},
		{ScriptCmpLT, 5, 6, true}, {ScriptCmpLT, 5, 5, false},
		{ScriptCmpGE, 5, 5, true}, {ScriptCmpGE, 4, 5, false},
		{ScriptCmpLE, 5, 5, true}, {ScriptCmpLE, 6, 5, false},
		// Above the alphabet: the engine's own bound sends these to the failing
		// arm, and one shipped map carries the second of them.
		{6, 5, 5, false},
		{-1, 5, 5, false},
		{0x7fffffff, 5, 5, false},
		// Signed, not unsigned: a negative register is BELOW a positive one.
		{ScriptCmpLT, -1, 1, true},
		{ScriptCmpGT, -1, 1, false},
	}
	for _, tc := range table {
		s := mustScript(t,
			[]ScriptCheck{constCheck(0, tc.a), constCheck(1, tc.b)},
			[]ScriptInstant{{Op: ScriptInstantWin}},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, tc.code)}, Instants: acts(0), Once: true}})
		w := scriptWorld(t, s, []Entity{{ID: 1}})
		scriptTicks(w, 7, nil)
		won, _ := w.ScriptCounters()
		if (won == 1) != tc.wantFiring {
			t.Errorf("code %d on (%d,%d): the trigger %sfired", tc.code, tc.a, tc.b,
				map[bool]string{true: "", false: "did not "}[won == 1])
		}
	}
}

// TestThePairsAreANDedAndTheANDOfNothingIsTrue is AC-6's other half.
func TestThePairsAreANDedAndTheANDOfNothingIsTrue(t *testing.T) {
	// Registers: 0 = 1, 1 = 1, 2 = 2. So (0 == 1) holds and (0 == 2) does not.
	consts := []ScriptCheck{constCheck(0, 1), constCheck(1, 1), constCheck(2, 2)}
	win := []ScriptInstant{{Op: ScriptInstantWin}}

	tests := []struct {
		name  string
		pairs [3]ScriptPair
		fires bool
	}{
		{"no pair at all — the AND of nothing", [3]ScriptPair{}, true},
		{"one holding pair", [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, true},
		{"two holding pairs", [3]ScriptPair{pair(0, 1, ScriptCmpEQ), pair(1, 0, ScriptCmpEQ)}, true},
		{"the second pair fails", [3]ScriptPair{pair(0, 1, ScriptCmpEQ), pair(0, 2, ScriptCmpEQ)}, false},
		{"the first pair fails", [3]ScriptPair{pair(0, 2, ScriptCmpEQ), pair(0, 1, ScriptCmpEQ)}, false},
		// An UNUSED slot is not a comparison against register 0: were it one,
		// this trigger would have to satisfy (0 == 0) as well and would still
		// fire, so the discriminating case is the one where register 0 is not 0.
		{"an unused slot beside a holding one",
			[3]ScriptPair{pair(0, 1, ScriptCmpEQ), {}, {}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t, consts, win,
				[]ScriptTrigger{{Pairs: tc.pairs, Instants: acts(0), Once: true}})
			w := scriptWorld(t, s, []Entity{{ID: 1}})
			scriptTicks(w, 7, nil)
			won, _ := w.ScriptCounters()
			if (won == 1) != tc.fires {
				t.Errorf("the trigger %sfired", map[bool]string{true: "", false: "did not "}[won == 1])
			}
		})
	}
}

// ---------------------------------------------------------------- the latch

// TestTheFireOnceFlagIsTheWholeDifference is AC-5's first half: two triggers
// alike in every particular but the flag, over a condition that holds forever.
func TestTheFireOnceFlagIsTheWholeDifference(t *testing.T) {
	for _, once := range []bool{true, false} {
		s := mustScript(t,
			[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
			[]ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{9}}},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
				Instants: acts(0), Once: once, Latch: 4}})
		w := scriptWorld(t, s, []Entity{{ID: 1}})
		scriptTicks(w, 40, nil) // three passes

		want := int32(3)
		if once {
			want = 1
		}
		if got := w.ScriptRegister(9); got != want {
			t.Errorf("once=%v: the trigger fired %d time(s) over three passes, want %d", once, got, want)
		}
		if got := w.ScriptLatched(4); !got {
			t.Errorf("once=%v: the latch at the trigger's own map position is clear", once)
		}
		// The latch is indexed by the trigger's MAP position, not by its
		// position in the compiled slice — this trigger is compiled at 0 and
		// latches at 4.
		if w.ScriptLatched(0) {
			t.Errorf("once=%v: the latch at index 0 is set; the trigger latches at 4", once)
		}
	}
}

// ---------------------------------------------------------------- registers

// TestTheRegisterFileIsOneArrayForBothIdSpaces is AC-3: a compiled check result
// and an authored mission variable share one array, so a map may name a register
// a check overwrites every pass. The collision is the map's to make.
func TestTheRegisterFileIsOneArrayForBothIdSpaces(t *testing.T) {
	// The check at register 0 measures a distance; the instant writes 42 into
	// register 0 as if it were a variable. The next pass overwrites it.
	s := mustScript(t,
		[]ScriptCheck{distCheck(0, 1, 0, 0), constCheck(1, 0), constCheck(2, 0)},
		[]ScriptInstant{{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{0, 42}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(1, 2, ScriptCmpEQ)}, Instants: acts(0), Once: false}})

	w := scriptWorld(t, s, []Entity{{ID: 1, X: 5, Y: 0}})
	scriptTicks(w, 7, nil)
	if got := w.ScriptRegister(0); got != 42 {
		t.Fatalf("after the first pass register 0 is %d, want the 42 the instant wrote", got)
	}
	scriptTicks(w, 16, nil)
	if got := w.ScriptRegister(0); got != 42 {
		t.Errorf("after the second pass register 0 is %d — the check overwrote it and the "+
			"instant wrote 42 back, which is the collision", got)
	}
	// And the check really is writing it: with no instant the register reads the
	// distance.
	s2 := mustScript(t, []ScriptCheck{distCheck(0, 1, 0, 0)}, nil, nil)
	w2 := scriptWorld(t, s2, []Entity{{ID: 1, X: 5, Y: 0}})
	scriptTicks(w2, 7, nil)
	if got := w2.ScriptRegister(0); got != 5 {
		t.Errorf("the check writes %d, want the distance 5", got)
	}
}

// TestAConstantIsPresetOnceAndNeverAgain is AC-3's other half, and it is the
// whole of what a mission variable's initial value is.
func TestAConstantIsPresetOnceAndNeverAgain(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 3), constCheck(1, 3)},
		[]ScriptInstant{{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{0, 99}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(1, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	// Preset at construction, before any tick.
	if got := w.ScriptRegister(0); got != 3 {
		t.Fatalf("register 0 is %d before any tick, want the constant's 3", got)
	}
	scriptTicks(w, 40, nil)
	if got := w.ScriptRegister(0); got != 99 {
		t.Errorf("register 0 is %d after three passes, want the 99 the instant wrote — a constant "+
			"that were re-applied every pass would read 3", got)
	}
}

func TestAnAuthoredRegisterSubscriptOutsideTheFileIsRefused(t *testing.T) {
	for _, bad := range []int32{-1, scriptRegisters, scriptRegisters + 500, 1 << 30} {
		s := mustScript(t,
			[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
			[]ScriptInstant{
				{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{bad, 7}},
				{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{bad}},
			},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
				Instants: acts(0, 1), Once: false}})
		w := scriptWorld(t, s, []Entity{{ID: 1}})
		before, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		scriptTicks(w, 40, nil)
		after, err := w.MarshalBinary()
		if err != nil {
			t.Fatalf("MarshalBinary: %v", err)
		}
		// Only the tick and script clock may move: no other register or latch beyond the
		// trigger's own, and above all no byte of the latch array standing in
		// for a register the original would have overrun into.
		if got := w.ScriptRegister(bad); got != 0 {
			t.Errorf("subscript %d reads %d, want 0", bad, got)
		}
		for r := int32(2); r < scriptRegisters; r++ {
			want := int32(0)
			if r == 93 {
				want = 3 // the three script passes, independently of the refused writes
			}
			if got := w.ScriptRegister(r); got != want {
				t.Errorf("subscript %d: register %d is %d, want %d — the write escaped its bound", bad, r, got, want)
			}
		}
		if bytes.Equal(before, after) {
			t.Errorf("subscript %d: the form did not move at all, so the trigger never fired "+
				"and the test proves nothing", bad)
		}
	}
}

// ---------------------------------------------------------------- the outcome

// TestTheOutcomeIsTwoCountersAndAReporter is AC-7. Lose is tested first, both
// tests are for exactly one, and the outcome latches.
func TestTheOutcomeIsTwoCountersAndAReporter(t *testing.T) {
	// oneShot builds a script whose single always-true trigger runs the instants
	// named, once.
	oneShot := func(ops ...int32) *Script {
		is := make([]ScriptInstant, len(ops))
		slots := make([]int32, len(ops))
		for i, op := range ops {
			is[i] = ScriptInstant{Op: op}
			slots[i] = int32(i)
		}
		return mustScript(t, []ScriptCheck{constCheck(0, 1), constCheck(1, 1)}, is,
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
				Instants: acts(slots...), Once: true}})
	}

	tests := []struct {
		name       string
		ops        []int32
		want       Outcome
		wantWonLos [2]uint32
	}{
		{"a win", []int32{ScriptInstantWin}, OutcomeWon, [2]uint32{1, 0}},
		{"a loss", []int32{ScriptInstantLose}, OutcomeLost, [2]uint32{0, 1}},
		// LOSE IS TESTED FIRST, so one pass raising both is a loss.
		{"both in one pass", []int32{ScriptInstantWin, ScriptInstantLose}, OutcomeLost, [2]uint32{1, 1}},
		// BOTH TESTS ARE FOR EXACTLY ONE. Two losing arms in one pass take the
		// counter from 0 to 2 and no report is ever made — the reporter's own
		// comparison, and a map can reach it.
		{"two losses in one pass", []int32{ScriptInstantLose, ScriptInstantLose},
			OutcomeUndecided, [2]uint32{0, 2}},
		{"two wins in one pass", []int32{ScriptInstantWin, ScriptInstantWin},
			OutcomeUndecided, [2]uint32{2, 0}},
		// A win skipped past by a doubled loss still leaves the mission running.
		{"two losses and a win", []int32{ScriptInstantLose, ScriptInstantLose, ScriptInstantWin},
			OutcomeWon, [2]uint32{1, 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := scriptWorld(t, oneShot(tc.ops...), []Entity{{ID: 1}})
			if w.Outcome() != OutcomeUndecided {
				t.Fatalf("a fresh world is already at outcome %d", w.Outcome())
			}
			scriptTicks(w, 16, nil) // the pass at 6 and the report at 15
			if got := w.Outcome(); got != tc.want {
				t.Errorf("the outcome is %d, want %d", got, tc.want)
			}
			won, lost := w.ScriptCounters()
			if [2]uint32{won, lost} != tc.wantWonLos {
				t.Errorf("the counters are (%d,%d), want %v", won, lost, tc.wantWonLos)
			}
		})
	}
}

// TestTheOutcomeLatches: a decided mission is never re-decided, however the
// counters move afterwards.
func TestTheOutcomeLatches(t *testing.T) {
	// A REPEATING trigger that loses on every pass: the counter climbs past one
	// and the outcome, set on the pass that took it to one, stays put.
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
		[]ScriptInstant{{Op: ScriptInstantLose}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: false}})
	w := scriptWorld(t, s, []Entity{{ID: 1}})
	scriptTicks(w, 40, nil)
	if got := w.Outcome(); got != OutcomeLost {
		t.Errorf("the outcome is %d, want lost", got)
	}
	if _, lost := w.ScriptCounters(); lost != 3 {
		t.Errorf("the lose counter is %d after three passes, want 3", lost)
	}
}

// ---------------------------------------------------------------- check arms

// TestTheCheckArmsThisBuildEvaluates is AC-3: each implemented arm against the
// register it writes.
func TestTheCheckArmsThisBuildEvaluates(t *testing.T) {
	ents := []Entity{
		{ID: 1, X: 10, Y: 10, HP: 5, MaxHP: 5},  // alive
		{ID: 2, X: 14, Y: 12, HP: -1, MaxHP: 5}, // dead
		{ID: 3, X: 25, Y: 10, HP: 0, MaxHP: 5},  // downed
		{ID: 4, X: 20, Y: 10, HP: 5, MaxHP: 5},  // alive, ten cells from id 1
	}
	tests := []struct {
		name  string
		check ScriptCheck
		want  int32
		// unwritten says the arm writes NO register, so the register keeps the
		// value it was preset to.
		unwritten bool
	}{
		{"distance to a point", distCheck(0, 1, 13, 14), 4, false},
		{"distance truncated to a byte", distCheck(0, 1, 10+300, 10), 300 & 0xff, false},
		{"within a radius, inside", ScriptCheck{Op: ScriptCheckWithin, Register: 0, Unit: 1, HasUnit: true,
			Args: [scriptParams]int32{13, 10, 3}}, 1, false},
		{"within a radius, outside", ScriptCheck{Op: ScriptCheckWithin, Register: 0, Unit: 1, HasUnit: true,
			Args: [scriptParams]int32{14, 10, 3}}, 0, false},
		{"in a box, inside", ScriptCheck{Op: ScriptCheckInBox, Register: 0, Unit: 1, HasUnit: true,
			Args: [scriptParams]int32{9, 9, 11, 11}}, 1, false},
		{"in a box, outside", ScriptCheck{Op: ScriptCheckInBox, Register: 0, Unit: 1, HasUnit: true,
			Args: [scriptParams]int32{0, 0, 9, 9}}, 0, false},
		{"alive", ScriptCheck{Op: ScriptCheckAlive, Register: 0, Unit: 1, HasUnit: true}, 1, false},
		{"dead", ScriptCheck{Op: ScriptCheckAlive, Register: 0, Unit: 2, HasUnit: true}, 0, false},
		{"downed counts as dead", ScriptCheck{Op: ScriptCheckAlive, Register: 0, Unit: 3, HasUnit: true}, 0, false},
		{"distance between two units", ScriptCheck{Op: ScriptCheckUnitDistance, Register: 0,
			Unit: 1, HasUnit: true, Unit2: 4, HasUnit2: true}, 10, false},
		{"distance to a downed unit saturates too", ScriptCheck{Op: ScriptCheckUnitDistance,
			Register: 0, Unit: 1, HasUnit: true, Unit2: 3, HasUnit2: true}, 0xff, false},
		{"distance to a dead unit saturates", ScriptCheck{Op: ScriptCheckUnitDistance, Register: 0,
			Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true}, 0xff, false},
		{"an unresolved reference measures nothing",
			ScriptCheck{Op: ScriptCheckDistance, Register: 0}, 0, true},
		{"a reference to an entity the world does not hold",
			ScriptCheck{Op: ScriptCheckDistance, Register: 0, Unit: 99, HasUnit: true}, 0, true},
		{"the VIP arm writes no register", ScriptCheck{Op: ScriptCheckVIP, Register: 0,
			Unit: 1, HasUnit: true}, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Register 0 is the arm's; register 1 is a constant preset to a
			// value no arm here writes, so "wrote nothing" is distinguishable
			// from "wrote zero" — the check reads its own register back after a
			// preset only when the arm leaves it alone.
			c := tc.check
			c.Register = 0
			s := mustScript(t, []ScriptCheck{c}, nil, nil)
			w := scriptWorld(t, s, ents)
			// Seed register 0 through an instant-free route: set it by hand is
			// not available, so the marker is that an unwritten register stays
			// at its zero. What separates the two cases below is the VIP arm's
			// side effect, checked separately.
			scriptTicks(w, 7, nil)
			if got := w.ScriptRegister(0); got != tc.want {
				t.Errorf("register 0 is %d, want %d", got, tc.want)
			}
			if tc.unwritten && tc.want != 0 {
				t.Fatalf("the case is inconsistent: an unwritten register cannot want %d", tc.want)
			}
		})
	}
}

// TestTheVIPArmCountsALossAndWritesNothing is AC-3's sharpest arm: a "protect
// this unit" objective is a trigger with no action at all, and what arms it is
// being AUTHORED rather than being referenced.
func TestTheVIPArmCountsALossAndWritesNothing(t *testing.T) {
	// Two VIP checks compared with != , and no instant — the shipped shape. Both
	// registers are written by nothing, so the pattern is permanently false and
	// the trigger's whole effect is its checks' side effect.
	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckVIP, Register: 0, Unit: 1, HasUnit: true},
			{Op: ScriptCheckVIP, Register: 1, Unit: 2, HasUnit: true},
		},
		nil,
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpNE)},
			Instants: acts(), Once: false}})

	w := scriptWorld(t, s, []Entity{{ID: 1, HP: 5, MaxHP: 5}, {ID: 2, HP: 5, MaxHP: 5}})

	// Alive: no loss, and the pattern never fires because 0 != 0 is false.
	scriptTicks(w, 7, nil)
	if _, lost := w.ScriptCounters(); lost != 0 {
		t.Fatalf("the lose counter is %d with both units alive", lost)
	}
	if w.ScriptLatched(0) {
		t.Errorf("the permanently false pattern latched")
	}

	// Finish one of them at the teardown boundary: the check counts a loss on
	// every pass from now on. An exact-zero body is intentionally still
	// rescuable and does not satisfy this arm.
	// world is at tick 7, so the next pass is 22 and the report that reads it is
	// 31 — the report at 15 runs BEFORE the pass that raises the counter, which
	// is why this runs to 32 and not to 16.
	Step(w, []Command{{Kind: KindKill, Entity: 1}})
	i := indexOfEntity(w.entities, 1)
	w.entities[i].HP, w.entities[i].Decay, w.entities[i].Dwell = -10, DecayBones, 0
	scriptTicks(w, 24, nil)
	if _, lost := w.ScriptCounters(); lost != 1 {
		t.Fatalf("the lose counter is %d after one pass with a dead VIP, want 1", lost)
	}
	if got := w.Outcome(); got != OutcomeLost {
		t.Errorf("the outcome is %d, want lost — the trigger has no action and the loss is the "+
			"check's own side effect", got)
	}
	// The registers were never written.
	if a, b := w.ScriptRegister(0), w.ScriptRegister(1); a != 0 || b != 0 {
		t.Errorf("the VIP registers read %d and %d, want 0 and 0", a, b)
	}
}

// ---------------------------------------------------------------- loudness

// scriptCheckSentinelOp is the ONE opcode this suite uses wherever a test needs
// "a real arm of the vocabulary that this build does not evaluate". It was
// written out at each site as a literal, and each site carried its own note
// saying the literal moves the day that arm is implemented; 1029 implemented
// checks 9 and 17 and had to move four of them at once. One constant is what
// makes the next such story a one-line change.
//
// It is opcode 20 rather than 4, 16 or 21. All four are real arms of the
// 22-entry condition table (TRIG-COND-003) that this build does not evaluate,
// but 4, 16 and 21 are the whole of the shipped census's remaining count and a
// research experiment on those three is queued. No shipped campaign map authors
// opcode 20, so nothing measured against the corpus depends on it.
//
// TestTheSentinelIsStillUnimplemented is the guard that this stays true.
const scriptCheckSentinelOp int32 = 20

// TestTheSentinelIsStillUnimplemented fails the day the sentinel opcode is
// implemented, which is the day every test using it stops testing what it
// claims to. Each of those tests also guards itself; this one names the reason.
func TestTheSentinelIsStillUnimplemented(t *testing.T) {
	t.Parallel()

	if scriptCheckSupported(scriptCheckSentinelOp) {
		t.Fatalf("check opcode %d is implemented; move scriptCheckSentinelOp to an arm that is not, "+
			"and re-read every test that reads it", scriptCheckSentinelOp)
	}
}

// TestAnUnimplementedCheckMakesItsReadersInert is AC-8 and AC-9, and it is the
// one outcome this story must not ship.
//
// The shape is the shipped one: an unimplemented check compared with the
// constant 0. The check is an arm this build does not evaluate, so its
// register is never written — and if the trigger were evaluated anyway it would
// compare that untouched zero with the authored zero, hold, and fire on the
// first pass of the mission. It must not.
//
// The arm was the GROUP COUNT, then the SACK QUERY, then the POPULATION COUNT
// and the NEAREST-UNIT DISTANCE (0122), then check 9 until 1029 implemented it,
// and the case moved on rather than being deleted: what is under test is the
// rule, and the rule needs an arm that is genuinely absent to be tested
// against. The sentinel is now scriptCheckSentinelOp, declared once above.
func TestAnUnimplementedCheckMakesItsReadersInert(t *testing.T) {
	const notAnArm = scriptCheckSentinelOp

	s := mustScript(t,
		[]ScriptCheck{{Op: notAnArm, Register: 0}, constCheck(1, 0)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0),
			Once: true, Latch: 2}})

	// Reported before a single tick has run.
	gaps := s.Unsupported()
	want := []ScriptGap{{Kind: ScriptGapCheck, Op: notAnArm, Index: 0}}
	if !reflect.DeepEqual(sortedGaps(gaps), want) {
		t.Errorf("Unsupported() is %+v, want %+v", gaps, want)
	}
	if got := s.InertTriggers(); !reflect.DeepEqual(got, []int32{0}) {
		t.Errorf("InertTriggers() is %v, want [0]", got)
	}
	if !s.Triggers()[0].Inert {
		t.Errorf("the trigger is not marked inert")
	}

	// And behaviourally: it never fires, and it leaves no trace that could be
	// read as "evaluated and failed".
	w := scriptWorld(t, s, []Entity{{ID: 1}})
	scriptTicks(w, 40, nil)
	if won, _ := w.ScriptCounters(); won != 0 {
		t.Errorf("the inert trigger fired: the win counter is %d", won)
	}
	if w.ScriptLatched(2) {
		t.Errorf("the inert trigger touched its latch")
	}
	if got := w.Outcome(); got != OutcomeUndecided {
		t.Errorf("the outcome is %d, want undecided", got)
	}
}

// TestARegisterOwnedByNoCheckIsNotPoisoned is the negative half of the rule
// above: a mission variable is legitimately zero until an instant writes it, and
// a trigger reading one is doing exactly what the map asked for. Poisoning every
// unwritten register would make an ordinary variable trigger inert.
func TestARegisterOwnedByNoCheckIsNotPoisoned(t *testing.T) {
	// Register 50 is owned by nothing. The trigger compares it with a constant
	// 0 and fires, which is what a map that has not yet set the variable means.
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 0)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(50, 0, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	if got := s.InertTriggers(); len(got) != 0 {
		t.Fatalf("InertTriggers() is %v on a script with no unimplemented arm", got)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1}})
	scriptTicks(w, 7, nil)
	if won, _ := w.ScriptCounters(); won != 1 {
		t.Errorf("the trigger did not fire; a variable register is not a poisoned one")
	}
}

// TestAnUnimplementedInstantIsSkippedAndReported is AC-8's other half: an arm
// this build does not run does NOT hold the rest of its trigger hostage.
func TestAnUnimplementedInstantIsSkippedAndReported(t *testing.T) {
	// Opcode 11 is the item TRANSFER arm: a real operation in the 34-arm table,
	// unimplemented here and authored by no shipped map. It replaces opcode 2,
	// which carried this role until 0169 implemented it.
	const transferArm int32 = 11

	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
		[]ScriptInstant{
			{Op: transferArm, Args: [scriptParams]int32{15}},
			{Op: ScriptInstantWin},
		},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
			Instants: acts(0, 1), Once: true}})

	want := []ScriptGap{{Kind: ScriptGapInstant, Op: transferArm, Index: 0}}
	if got := sortedGaps(s.Unsupported()); !reflect.DeepEqual(got, want) {
		t.Errorf("Unsupported() is %+v, want %+v", got, want)
	}
	if got := s.InertTriggers(); len(got) != 0 {
		t.Errorf("InertTriggers() is %v — an unimplemented INSTANT makes no trigger inert", got)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1}})
	scriptTicks(w, 16, nil)
	if got := w.Outcome(); got != OutcomeWon {
		t.Errorf("the outcome is %d, want won — the skipped instant took the win with it", got)
	}
}

// ---------------------------------------------------------------- NewScript

func TestNewScriptRefusesWhatABinderCanOnlyReachByBeingWrong(t *testing.T) {
	ok := []ScriptCheck{constCheck(0, 1)}
	tests := []struct {
		name     string
		checks   []ScriptCheck
		instants []ScriptInstant
		triggers []ScriptTrigger
	}{
		{"a register subscript below the file", []ScriptCheck{constCheck(-1, 1)}, nil, nil},
		{"a register subscript above the file", []ScriptCheck{constCheck(scriptRegisters, 1)}, nil, nil},
		{"two checks owning one register", []ScriptCheck{constCheck(3, 1), constCheck(3, 2)}, nil, nil},
		{"a latch below the array", ok, nil, []ScriptTrigger{{Latch: -1}}},
		{"a latch above the array", ok, nil, []ScriptTrigger{{Latch: scriptLatches}}},
		{"a pair reading a register outside the file", ok, nil,
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, scriptRegisters, ScriptCmpEQ)}}}},
		{"an instant slot naming no instant", ok, nil,
			[]ScriptTrigger{{Instants: acts(0)}}},
		{"an instant slot below none", ok, []ScriptInstant{{Op: ScriptInstantWin}},
			[]ScriptTrigger{{Instants: [4]int32{-2, ScriptNone, ScriptNone, ScriptNone}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if s, err := NewScript(tc.checks, tc.instants, tc.triggers); err == nil {
				t.Errorf("accepted: %+v", s)
			}
		})
	}

	// An UNIMPLEMENTED OPCODE is not among the refusals: a map is entitled to
	// author the whole vocabulary, and refusing would take the arms this build
	// does implement down with it.
	if _, err := NewScript([]ScriptCheck{{Op: 12345, Register: 0}}, nil, nil); err != nil {
		t.Errorf("an unimplemented opcode was refused: %v", err)
	}
}

// TestAnEmptyScriptAndNoneAreOneWorld: an empty script has nothing to evaluate,
// so keeping it apart from none would be a distinction the byte form, the digest
// and every tick are blind to.
func TestGiveMoneyCreditsTheNamedPlayersPurseWithWrapping(t *testing.T) {
	s := fireOnce(t, []ScriptInstant{{
		Op: ScriptInstantGiveMoney, Args: [scriptParams]int32{500},
		Player: SelfSlot, HasPlayer: true,
	}}, 0)
	w := scriptWorld(t, s, nil)
	w.purses[SelfSlot] = ^uint32(0) - 99

	scriptTicks(w, 32, nil)

	if got := w.Purse(SelfSlot); got != 400 {
		t.Fatalf("Purse(SelfSlot) = %d, want 400 after a wrapping 500-gold grant", got)
	}
}

func TestGiveMoneyWithoutAPlayerChangesNoPurse(t *testing.T) {
	s := fireOnce(t, []ScriptInstant{{
		Op: ScriptInstantGiveMoney, Args: [scriptParams]int32{500},
	}}, 0)
	w := scriptWorld(t, s, nil)

	scriptTicks(w, 32, nil)

	if got := w.Purse(SelfSlot); got != 0 {
		t.Fatalf("Purse(SelfSlot) = %d, want 0 when the node names no player", got)
	}
}

func TestAnEmptyScriptAndNoneAreOneWorld(t *testing.T) {
	empty := mustScript(t, nil, nil, nil)
	a := scriptWorld(t, nil, []Entity{{ID: 1}})
	b := scriptWorld(t, empty, []Entity{{ID: 1}})

	if a.Script() != nil || b.Script() != nil {
		t.Errorf("a world given an empty script holds one")
	}
	fa, err := a.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	fb, err := b.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if !bytes.Equal(fa, fb) || a.Hash() != b.Hash() {
		t.Errorf("the two worlds differ in form or digest")
	}
}

// ---------------------------------------------------------------- the form

// TestTheScriptAndItsStateCrossTheByteForm is AC-5: a mission resumed from its
// bytes does not re-fire a one-shot trigger it has already spent, and the
// compiled program crosses with it.
func TestTheScriptAndItsStateCrossTheByteForm(t *testing.T) {
	build := func() *Script {
		return mustScript(t,
			[]ScriptCheck{
				constCheck(0, 1), constCheck(1, 1),
				distCheck(2, 7, 3, 4),
				{Op: ScriptCheckVIP, Register: 3, Unit: 7, HasUnit: true},
			},
			[]ScriptInstant{
				{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{60}},
				{Op: ScriptInstantWin},
			},
			[]ScriptTrigger{
				{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0),
					Once: true, Latch: 11},
			})
	}
	w := scriptWorld(t, build(), []Entity{{ID: 7, X: 9, Y: 9, HP: 3, MaxHP: 3}})
	scriptTicks(w, 7, nil)
	if got := w.ScriptRegister(60); got != 1 {
		t.Fatalf("the one-shot trigger fired %d time(s) before the cut, want 1", got)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var resumed World
	if err := resumed.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	// The compiled program crossed, record for record.
	if got, want := resumed.Script().Checks(), w.Script().Checks(); !reflect.DeepEqual(got, want) {
		t.Errorf("the decoded checks are\n %+v\nwant\n %+v", got, want)
	}
	if got, want := resumed.Script().Instants(), w.Script().Instants(); !reflect.DeepEqual(got, want) {
		t.Errorf("the decoded instants are\n %+v\nwant\n %+v", got, want)
	}
	if got, want := resumed.Script().Triggers(), w.Script().Triggers(); !reflect.DeepEqual(got, want) {
		t.Errorf("the decoded triggers are\n %+v\nwant\n %+v", got, want)
	}
	if !resumed.ScriptLatched(11) {
		t.Errorf("the spent latch did not cross the form")
	}

	// And the state: advancing both worlds another three passes leaves them
	// identical, which is what "it does not re-fire" means where a replay would
	// meet it.
	for i := 0; i < 40; i++ {
		Step(w, nil)
		Step(&resumed, nil)
		if w.Hash() != resumed.Hash() {
			t.Fatalf("the resumed world diverged at tick %d: %#016x vs %#016x",
				w.Tick(), w.Hash(), resumed.Hash())
		}
	}
	if got := w.ScriptRegister(60); got != 1 {
		t.Errorf("the one-shot trigger has now fired %d time(s), want the 1 it fired before the cut", got)
	}

	// A world whose script is REPLACED is a different world: the program is
	// canonical state, so two worlds alike in every other field differ here.
	other := mustScript(t, []ScriptCheck{constCheck(0, 2)}, nil, nil)
	w2 := scriptWorld(t, other, []Entity{{ID: 7, X: 9, Y: 9, HP: 3, MaxHP: 3}})
	w3 := scriptWorld(t, build(), []Entity{{ID: 7, X: 9, Y: 9, HP: 3, MaxHP: 3}})
	if w2.Hash() == w3.Hash() {
		t.Errorf("two worlds running different scripts hash equal")
	}
}

func TestTheScriptSectionRefusesTheBytesNoTickCanLeave(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1), constCheck(1, 1)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0),
			Once: true, Latch: 3}})
	w := scriptWorld(t, s, []Entity{{ID: 1}})
	scriptTicks(w, 16, nil)
	valid, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	// The script section begins here: the header, no grid at these bounds
	// (80x80 gives 6400 cells), one record, one route count, the empty group
	// section this entity's Owner of 0 writes — a count of zero and no record
	// — the empty sack section this world names none of (0103), the same bare
	// four-byte zero count, the empty carry section and purse (0112) — this
	// entity carries nothing and no gold is credited, so a bare four-byte zero
	// count and purseLen zeros — and the empty equipment section (0124 T2),
	// between the two: this entity equips nothing, so a bare EquipSlots-wide
	// zero record and no count at all. The record is 230 bytes since 0166's own
	// escort triple and widened spell mark grew it past the width this comment
	// used to name beyond the width this comment used to name; this number is
	// written out by hand, so a record that widens again moves it here and
	// nowhere else. spellCountLen is added for the same reason: this world
	// names no spell table, so the section right after the purse is its own
	// bare two-byte zero count. structureCountLen is added on the same reason
	// (1033 B3): this world declares no structure, so the section right after
	// the script-state section is its own bare four-byte zero count.
	base := 34 + 3*80*80 + 492 + 4 + groupCountLen + sackCountLen + carryCountLen + equipRecordLen + treasureRecordLen + purseLen + spellCountLen + itemWeightCountLen + 2*castingCountLen + relationSlots + tailCountLen + structureCountLen + len(emptyItemStatePin(1)) + 4
	// The relation closes the form BEHIND the script section, so it is subtracted
	// here rather than cut off the fixture: a spoiled copy has to differ from a
	// well-formed one in the one byte this test writes, and a copy missing the
	// block would be refused for its length before the spoiling was ever read.
	if len(valid)-entityIDFloorLen-spellDeliverySpanLen-65-base-relationLen-12 != scriptStateLen+scriptCountsLen+2*scriptCheckLen+
		scriptInstantLen+scriptTriggerLen {
		t.Fatalf("the script section is %d byte(s), which is not the shape this test spoils",
			len(valid)-base-relationLen)
	}

	spoil := func(off int, b byte) []byte {
		out := append([]byte(nil), valid...)
		out[base+off] = b
		return out
	}
	tests := []struct {
		name string
		data []byte
	}{
		{"a latch byte that is neither 0 nor 1", spoil(4*scriptRegisters, 2)},
		{"an outcome this build does not define", spoil(scriptStateLen-1, 3)},
		{"a check's unit-presence byte", spoil(scriptStateLen+scriptCountsLen+56, 2)},
		// The instant record's own three presence bytes, at its +56, +57 and +58 —
		// the tail the owner story appended. Each is refused outside its value set
		// rather than read as truthy, or two forms would decode to one world.
		{"an instant's unit-presence byte", spoil(scriptStateLen+scriptCountsLen+
			2*scriptCheckLen+56, 2)},
		{"an instant's group-presence byte", spoil(scriptStateLen+scriptCountsLen+
			2*scriptCheckLen+57, 0xff)},
		{"an instant's player-presence byte", spoil(scriptStateLen+scriptCountsLen+
			2*scriptCheckLen+58, 2)},
		{"a trigger's pair-use byte", spoil(scriptStateLen+scriptCountsLen+
			2*scriptCheckLen+scriptInstantLen+12, 2)},
		{"a trigger's fire-once byte", spoil(scriptStateLen+scriptCountsLen+
			2*scriptCheckLen+scriptInstantLen+55, 2)},
		{"one byte too many", append(append([]byte(nil), valid...), 0)},
		{"one byte too few", valid[:len(valid)-1]},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got World
			if err := got.UnmarshalBinary(tc.data); err == nil {
				t.Errorf("accepted")
			}
		})
	}

	// Original Player outcome is independent from the map-lifetime counters.
	won := append([]byte(nil), valid...)
	for i := 0; i < 4; i++ {
		won[base+4*scriptRegisters+scriptLatches+i] = 0 // the win counter back to 0
	}
	var got World
	if err := got.UnmarshalBinary(won); err != nil {
		t.Errorf("refused a saved Player outcome with a win counter of 0: %v", err)
	}
}

// ---------------------------------------------------------------- the chain

// TestTheCampaignsFirstMissionWinChainFires is AC-7's whole point and the
// milestone's: a map's own authored script, compiled and run, ends the mission.
//
// It is the campaign's first mission's win chain, reduced to the two triggers
// that carry it and driven with the coordinates that map authors:
//
//	trigger 3 — the escortee within 3 of (56,21): raise a message and set the
//	            mission variable at register 50
//	trigger 4 — the hero within 3 of (66,16) AND that variable not zero: raise a
//	            message and force the "Mission Complete" state
//
// The two message arms are in the fixture ON PURPOSE. They were an instant
// this build did not run, and the chain had to complete around them while
// the report named them; 0169 implemented the arm, so the same authored
// chain now reports no gap at all. Keeping the nodes in place is what makes
// that a measured change rather than a fixture edited to agree.
//
// THE DISTANCES ARE DRIVEN TO ZERO rather than to three, and that is deliberate.
// Which metric these arms measure is the one thing in the runtime the evidence
// does not fix; at distance zero every candidate metric agrees, so this witness
// does not rest on the choice.
func TestTheCampaignsFirstMissionWinChainFires(t *testing.T) {
	const (
		hero     EntityID = 1
		escortee EntityID = 2

		varSlot int32 = 50
	)

	s := mustScript(t,
		[]ScriptCheck{
			distCheck(0, escortee, 56, 21), // "distance from point to unit"
			constCheck(1, 3),               // the authored "3"
			distCheck(2, hero, 66, 16),
			{Op: ScriptCheckVariable, Register: 3, Args: [scriptParams]int32{varSlot}},
			constCheck(4, 0), // the authored "FALSE"
		},
		[]ScriptInstant{
			{Op: ScriptInstantMessage, Args: [scriptParams]int32{2}}, // send message
			{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{varSlot}},
			{Op: ScriptInstantMessage, Args: [scriptParams]int32{3}}, // send message
			{Op: ScriptInstantWin},
		},
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpLE)},
				Instants: acts(0, 1), Once: true, Latch: 3},
			{Pairs: [3]ScriptPair{pair(2, 1, ScriptCmpLE), pair(3, 4, ScriptCmpNE)},
				Instants: acts(2, 3), Once: true, Latch: 4},
		})

	// The report names nothing: every arm this authored chain uses is one this
	// build runs, and no trigger is inert.
	if got := sortedGaps(s.Unsupported()); len(got) != 0 {
		t.Fatalf("Unsupported() is %+v, want none — every arm of this chain is implemented", got)
	}
	if got := s.InertTriggers(); len(got) != 0 {
		t.Fatalf("InertTriggers() is %v, want none", got)
	}

	w := scriptWorld(t, s, []Entity{
		{ID: hero, X: 63, Y: 16},
		{ID: escortee, X: 53, Y: 21},
	})

	// Both walk to their cells, three cells each, one cell a tick.
	scriptTicks(w, 6, []Command{
		{Entity: hero, X: 66, Y: 16},
		{Entity: escortee, X: 56, Y: 21},
	})
	for _, e := range w.Entities() {
		if e.HasTarget {
			t.Fatalf("entity %d has not arrived by tick %d", e.ID, w.Tick())
		}
	}
	if won, _ := w.ScriptCounters(); won != 0 {
		t.Fatalf("the mission was won before the first pass")
	}

	// Tick 6, the first pass. Trigger 3 fires and sets the variable — and
	// trigger 4 does NOT, because every check is evaluated BEFORE any trigger
	// runs, so its variable register still reads the pass's own zero.
	Step(w, nil)
	if got := w.ScriptRegister(varSlot); got != 1 {
		t.Fatalf("after the first pass the mission variable is %d, want 1", got)
	}
	if !w.ScriptLatched(3) {
		t.Errorf("trigger 3's latch is clear after it fired")
	}
	if won, _ := w.ScriptCounters(); won != 0 {
		t.Fatalf("the win fired in the same pass that set the variable it waits on")
	}

	// Tick 22, the second pass: the variable is read, trigger 4 fires, the win
	// counter moves. Tick 31 reports it.
	scriptTicks(w, 25, nil)
	if won, _ := w.ScriptCounters(); won != 1 {
		t.Fatalf("the win counter is %d after the second pass", won)
	}
	if got := w.Outcome(); got != OutcomeWon {
		t.Fatalf("the outcome is %d, want won", got)
	}
	// Trigger 3 fired ONCE, not twice: it is a one-shot and its latch held.
	if got := w.ScriptRegister(varSlot); got != 1 {
		t.Errorf("the mission variable is %d, want 1 — the one-shot trigger fired again", got)
	}
	// And it stays won, however long it runs.
	scriptTicks(w, 40, nil)
	if got := w.Outcome(); got != OutcomeWon {
		t.Errorf("the outcome moved to %d after the mission was won", got)
	}
}

// ---------------------------------------------------------------------------
// The compiled instant's three references, across the byte form — and the two
// hand-over arms still UNIMPLEMENTED, which is what makes the state's arrival a
// coherent change on its own.
// ---------------------------------------------------------------------------

// TestTheInstantRecordCarriesItsThreeReferences is AC-7 over the script
// section's own widened record: the unit, group and player an instant names, and
// the three bytes that say whether it names them, all cross the form.
//
// The two records that differ ONLY in a presence flag are the point. Group zero
// named and no group at all are different instants, and so are player zero named
// and no player; if either pair came back alike the form would have stopped being
// injective and two byte forms would decode to one world.
func TestTheInstantRecordCarriesItsThreeReferences(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1)},
		[]ScriptInstant{
			{Op: ScriptInstantWin, Unit: 4294967295, HasUnit: true,
				Group: 4294967295, HasGroup: true, Player: 4294967295, HasPlayer: true},
			{Op: ScriptInstantLose, Group: 0, HasGroup: true, Player: 0, HasPlayer: true},
			{Op: ScriptInstantLose},
			{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{7, 9},
				Unit: 0, HasUnit: true},
		}, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1, Owner: 3}})

	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got, want := back.Script().Instants(), s.Instants()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("instant %d crosses the form as %+v, want %+v", i, got[i], want[i])
		}
	}
	if got[1] == got[2] {
		t.Errorf("an instant naming group 0 and player 0 and one naming neither come back alike")
	}
	// Entity id zero is a real entity, so the same separation is owed to the unit
	// reference and is asked of the fourth record against the third.
	if !got[3].HasUnit || got[3].Unit != 0 {
		t.Errorf("the instant naming entity 0 comes back as %d/%v, want 0/true",
			got[3].Unit, got[3].HasUnit)
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
}

// TestTheReportStopsNamingTheTwoHandOverArms replaces
// TestTheTwoHandOverArmsAreStillUnimplemented, which asserted the opposite and
// was written to have exactly this shelf life: the state's arrival was a change
// that had to stand on its own, and the way to say so was to show the arms it
// was for still absent. They are here now, so the claim inverts.
func TestTheReportStopsNamingTheTwoHandOverArms(t *testing.T) {
	t.Parallel()

	const stillUnimplemented int32 = 11 // the item transfer arm
	if scriptInstantSupported(ScriptInstant{Op: stillUnimplemented}) {
		t.Fatal("the transfer arm is implemented; this test needs one that is not")
	}
	s := mustScript(t,
		[]ScriptCheck{constCheck(0, 1)},
		[]ScriptInstant{
			{Op: ScriptInstantGiveGroup, Group: 4, HasGroup: true, Player: 9, HasPlayer: true},
			{Op: stillUnimplemented},
			{Op: ScriptInstantGiveUnit, Unit: 1, HasUnit: true, Player: 9, HasPlayer: true},
		}, nil)

	gaps := s.Unsupported()
	if len(gaps) != 1 || gaps[0].Kind != ScriptGapInstant ||
		gaps[0].Op != stillUnimplemented || gaps[0].Index != 1 {
		t.Errorf("the report names %+v, want only the transfer arm at subscript 1", gaps)
	}
}
