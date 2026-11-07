package sim

import (
	"bytes"
	"reflect"
	"testing"
)

// extendedProgram is an extended program of 150 variables: checks 0, 1 and
// 4 are the constants 1, 0 and 7, check 2 reads variable 120 and check 3
// variable 5. Trigger 0 runs setup; trigger 1 fires once when variable 120
// is 1 (group 0) or variable 5 is 7 (group 1); trigger 2 never holds and
// latches at position 1500, past the session block's thousand.
func extendedProgram(t *testing.T, setup ...int32) *Script {
	t.Helper()
	group := func(g, left, right, cmp int32) ScriptPair {
		p := pair(left, right, cmp)
		p.Group = g
		return p
	}
	s, err := NewExtendedScript(150, []ScriptCheck{
		constCheck(0, 1), constCheck(1, 0),
		{Op: ScriptCheckVariable, Register: 2, Args: [scriptParams]int32{120}},
		{Op: ScriptCheckVariable, Register: 3, Args: [scriptParams]int32{5}},
		constCheck(4, 7),
	}, []ScriptInstant{
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{140, 11}},
		{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{141}},
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{120, 1}},
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{5, 7}},
	}, []ScriptTrigger{
		{Instants: acts(), Once: true, Latch: 0, Actions: setup},
		{Instants: acts(), Once: true, Latch: 1, Actions: []int32{0, 1}, Conditions: []ScriptPair{
			group(0, 2, 0, ScriptCmpEQ), group(0, 4, 1, ScriptCmpGT),
			group(1, 3, 4, ScriptCmpEQ), group(1, 0, 1, ScriptCmpGT),
		}},
		{Instants: acts(), Latch: 1500, Conditions: []ScriptPair{group(0, 0, 1, ScriptCmpEQ)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestExtendedTriggerFiresWhenAnyGroupHolds(t *testing.T) {
	for _, c := range []struct {
		name  string
		setup []int32
		fires bool
	}{{"group 0", []int32{2}, true}, {"group 1", []int32{3}, true}, {"neither", nil, false}} {
		w := scriptWorld(t, extendedProgram(t, c.setup...), nil)
		scriptTicks(w, 96, nil)
		want := map[int32]int32{140: 0, 141: 0}
		if c.fires {
			want = map[int32]int32{140: 11, 141: 1}
		}
		for v, n := range want {
			if got := w.ScriptRegister(v); got != n {
				t.Errorf("%s: variable %d is %d, want %d", c.name, v, got, n)
			}
		}
		if w.ScriptLatched(1) != c.fires || w.ScriptLatched(1500) {
			t.Errorf("%s: latches 1 and 1500 are %v and %v", c.name, w.ScriptLatched(1), w.ScriptLatched(1500))
		}
		// The variables are their own array: check 0's result is 1 and
		// variable 0 stays 0, while variable 93 counts the passes.
		if w.ScriptRegister(0) != 0 || w.checkResult(0) != 1 || w.ScriptRegister(93) < 5 {
			t.Errorf("%s: variable 0 %d, result 0 %d, passes %d", c.name, w.ScriptRegister(0), w.checkResult(0), w.ScriptRegister(93))
		}
	}
}

func TestExtendedWorldCrossesTheByteFormWithItsState(t *testing.T) {
	w := scriptWorld(t, extendedProgram(t, 2), nil)
	scriptTicks(w, 40, nil)
	w.setRegisterAt(149, -3)
	w.setLatch(1500, 1)
	form := mustMarshal(t, w)
	if !bytes.Contains(form, []byte("XSC1")) {
		t.Fatal("the extended form wrapper is absent")
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() || !bytes.Equal(mustMarshal(t, &back), form) {
		t.Fatal("the round trip changed the World")
	}
	s := back.Script()
	if !s.Extended() || s.Variables() != 150 || !EqualTriggers(s.Triggers(), w.Script().Triggers()) ||
		!reflect.DeepEqual(back.ScriptExtendedState(), w.ScriptExtendedState()) || back.ScriptRegister(149) != -3 || !back.ScriptLatched(1500) {
		t.Fatal("the program or its extended state changed", back.ScriptExtendedState())
	}
	for i := 0; i < 64; i++ {
		Step(w, nil)
		Step(&back, nil)
		if back.Hash() != w.Hash() {
			t.Fatal("the decoded World diverged at tick", i)
		}
	}
	// The extended state enters the hash.
	for name, edit := range map[string]func(*World){
		"variable": func(x *World) { x.setRegisterAt(130, 9) },
		"latch":    func(x *World) { x.latchTail[1] = 1 },
		"result":   func(x *World) { x.scriptResults[2]++ },
	} {
		var other World
		if err := other.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		before := other.Hash()
		edit(&other)
		if other.Hash() == before {
			t.Errorf("%s: the hash does not see the change", name)
		}
	}
	// Every cut of the form is refused.
	for n := 0; n < len(form); n += 1 + len(form)/700 {
		var cut World
		if cut.UnmarshalBinary(form[:n]) == nil {
			t.Fatalf("a form cut at %d of %d decoded", n, len(form))
		}
	}
}

func TestOriginalWorldWritesNoExtendedForm(t *testing.T) {
	s := mustScript(t, []ScriptCheck{constCheck(0, 1)}, []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{7}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 0, ScriptCmpEQ)}, Instants: acts(0), Latch: 0}})
	w := scriptWorld(t, s, nil)
	scriptTicks(w, 40, nil)
	if bytes.Contains(mustMarshal(t, w), []byte("XSC1")) || w.scriptVars != nil || w.scriptResults != nil || w.latchTail != nil {
		t.Fatal("an original program carries extended state")
	}
	if w.ScriptRegister(0) != 1 || w.ScriptRegister(7) != w.ScriptRegister(93) || w.ScriptRegister(7) < 2 || w.ScriptRegister(100) != 0 {
		t.Fatal("an original program no longer shares its registers", w.ScriptRegister(0), w.ScriptRegister(7))
	}
}
