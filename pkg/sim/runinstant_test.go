package sim

import "testing"

func runInstantWorld(t *testing.T) *World {
	t.Helper()
	checks := []ScriptCheck{
		{Op: ScriptCheckVariable, Register: 0, Args: [scriptParams]int32{10}},
		constCheck(1, 3),
	}
	instants := []ScriptInstant{
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{10, 3}},
		{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{11}},
	}
	return scriptWorld(t, mustScript(t, checks, instants, nil), nil)
}

func TestRunInstantCommandRunsTheCompiledAction(t *testing.T) {
	w := runInstantWorld(t)
	Step(w, []Command{RunInstant(0), RunInstant(1), RunInstant(1)})
	if got := w.ScriptRegister(10); got != 3 {
		t.Errorf("variable 10 = %d, want 3", got)
	}
	if got := w.ScriptRegister(11); got != 2 {
		t.Errorf("variable 11 = %d, want 2", got)
	}
}

func TestRunInstantCommandIgnoresAnInstantTheScriptLacks(t *testing.T) {
	w := runInstantWorld(t)
	before := w.ScriptRegisters()
	Step(w, []Command{RunInstant(-1), RunInstant(2), RunInstant(99)})
	if w.ScriptRegisters() == before {
		return
	}
	after := w.ScriptRegisters()
	for i := range after {
		if i != 93 && after[i] != before[i] {
			t.Errorf("register %d changed to %d", i, after[i])
		}
	}
}

func TestRunInstantCommandIsReplayedToTheSameWorld(t *testing.T) {
	a, b, c := runInstantWorld(t), runInstantWorld(t), runInstantWorld(t)
	Step(a, []Command{RunInstant(0)})
	Step(b, []Command{RunInstant(0)})
	Step(c, nil)
	if a.Hash() != b.Hash() || a.Hash() == c.Hash() {
		t.Errorf("hashes %x %x %x: the replay must agree and the command must reach the hash", a.Hash(), b.Hash(), c.Hash())
	}
}

// A reply's availability asks the pair evaluator a trigger's pass uses, on the
// check results the last pass left.
func TestConditionHoldsReadsTheLastPassLikeATrigger(t *testing.T) {
	w := runInstantWorld(t)
	p := ScriptPair{Left: 0, Right: 1, Cmp: ScriptCmpEQ, Used: true}
	if w.ConditionHolds(p) {
		t.Fatal("the pair held before any variable was set")
	}
	scriptTicks(w, 1, []Command{RunInstant(0)})
	scriptTicks(w, 20, nil)
	if !w.ConditionHolds(p) {
		t.Error("the pair did not hold after the variable took 3")
	}
	if w.ConditionHolds(ScriptPair{Left: 0, Right: 7, Cmp: ScriptCmpEQ, Used: true}) || w.ConditionHolds(ScriptPair{Left: -1, Right: 1}) {
		t.Error("a pair naming a check the program lacks held")
	}
	if (&World{}).ConditionHolds(p) {
		t.Error("a world with no program held a pair")
	}
}
