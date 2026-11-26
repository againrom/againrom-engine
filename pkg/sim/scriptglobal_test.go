package sim

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

// The global operations' opcodes as the extended-op table names them.
const (
	readGlobalOp int32 = 0x4102
	setGlobalOp  int32 = 0x4103
	addGlobalOp  int32 = 0x4104
)

// globalProgram is an extended program whose setup trigger sets global 2 to
// 40, adds 2 to it, adds 1 to global 1, and sets and adds to global 3, which
// the World does not hold. Its second trigger sets variable 50 to 1 once
// global 1 is below 0 and global 3 reads 0; its third sets variable 51 to 1
// once global 2 is 42.
func globalProgram(t *testing.T) *Script {
	t.Helper()
	s, err := NewExtendedScript(ScriptROM1, 100, []ScriptCheck{
		{Op: readGlobalOp, Register: 0, Args: [scriptParams]int32{1}},
		constCheck(1, 0),
		{Op: readGlobalOp, Register: 2, Args: [scriptParams]int32{3}},
		{Op: readGlobalOp, Register: 3, Args: [scriptParams]int32{2}},
		constCheck(4, 42),
	}, []ScriptInstant{
		{Op: setGlobalOp, Args: [scriptParams]int32{2, 40}},
		{Op: addGlobalOp, Args: [scriptParams]int32{2, 2}},
		{Op: addGlobalOp, Args: [scriptParams]int32{1, 1}},
		{Op: setGlobalOp, Args: [scriptParams]int32{3, 9}},
		{Op: addGlobalOp, Args: [scriptParams]int32{3, 9}},
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{50, 1}},
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{51, 1}},
	}, []ScriptTrigger{
		{Instants: acts(), Once: true, Latch: 0, Actions: []int32{0, 1, 2, 3, 4}},
		{Instants: acts(), Once: true, Latch: 1, Actions: []int32{5}, Conditions: []ScriptPair{
			pair(0, 1, ScriptCmpLT), pair(2, 1, ScriptCmpEQ)}},
		{Instants: acts(), Once: true, Latch: 2, Actions: []int32{6}, Conditions: []ScriptPair{pair(3, 4, ScriptCmpEQ)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// globalWorld is globalProgram's World seated with global 1 at the largest
// int32 and global 2 at 0.
func globalWorld(t *testing.T) *World {
	t.Helper()
	w := scriptWorld(t, globalProgram(t), nil)
	if err := w.SeatScriptGlobals([]ScriptGlobal{{ID: 2}, {ID: 1, Value: math.MaxInt32}}); err != nil {
		t.Fatal(err)
	}
	return w
}

// Set and add write the globals the World holds, add wraps as a register
// does, a comparison reads a global as any check result, and a global the
// World does not hold reads 0 and takes no write.
func TestScriptGlobalsSetAddWrapAndCompare(t *testing.T) {
	w := globalWorld(t)
	scriptTicks(w, 96, nil)
	want := []ScriptGlobal{{ID: 1, Value: math.MinInt32}, {ID: 2, Value: 42}}
	if got := w.ScriptGlobals(); !reflect.DeepEqual(got, want) {
		t.Fatalf("globals %v, want %v", got, want)
	}
	if w.ScriptRegister(50) != 1 || w.ScriptRegister(51) != 1 {
		t.Fatalf("the comparisons left variables 50 and 51 at %d and %d", w.ScriptRegister(50), w.ScriptRegister(51))
	}
	if w.globalAt(3) != 0 {
		t.Fatal("a global the World does not hold reads", w.globalAt(3))
	}
}

// A World holding globals carries them in the outermost form wrapper; the
// form decodes to the same World, every value enters the hash, and a cut
// form is refused.
func TestScriptGlobalsCrossTheByteForm(t *testing.T) {
	w := globalWorld(t)
	scriptTicks(w, 40, nil)
	form := mustMarshal(t, w)
	if !bytes.HasSuffix(form, []byte("GLB1")) || form[0] != scriptGlobalFormVersion {
		t.Fatal("the globals wrapper is not outermost")
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() || !bytes.Equal(mustMarshal(t, &back), form) || !reflect.DeepEqual(back.ScriptGlobals(), w.ScriptGlobals()) {
		t.Fatal("the round trip changed the World")
	}
	before := back.Hash()
	back.setGlobal(2, 43)
	if back.Hash() == before {
		t.Fatal("the hash does not see a global's value")
	}
	if w.globalAt(2) != 42 {
		t.Fatal("a write into a decoded copy reached the World it came from")
	}
	for n := 0; n < len(form); n += 1 + len(form)/700 {
		var cut World
		if cut.UnmarshalBinary(form[:n]) == nil {
			t.Fatalf("a form cut at %d of %d decoded", n, len(form))
		}
	}
}

// A World without globals writes the form it wrote before: no wrapper, the
// same bytes and hash, whether no seat ran or a seat gave it none.
func TestWorldWithoutGlobalsKeepsItsForm(t *testing.T) {
	plain := scriptWorld(t, globalProgram(t), nil)
	seated := scriptWorld(t, globalProgram(t), nil)
	if err := seated.SeatScriptGlobals(nil); err != nil {
		t.Fatal(err)
	}
	scriptTicks(plain, 40, nil)
	scriptTicks(seated, 40, nil)
	a, b := mustMarshal(t, plain), mustMarshal(t, seated)
	if !bytes.Equal(a, b) || plain.Hash() != seated.Hash() || a[0] == scriptGlobalFormVersion || bytes.Contains(a, []byte("GLB1")) {
		t.Fatal("a World without globals changed its form")
	}
}

// The same start and steps give the same globals and hash.
func TestScriptGlobalsReplayEqual(t *testing.T) {
	a, b := globalWorld(t), globalWorld(t)
	for i := 0; i < 96; i++ {
		Step(a, nil)
		Step(b, nil)
		if a.Hash() != b.Hash() {
			t.Fatal("the replay diverged at tick", i)
		}
	}
	if !reflect.DeepEqual(a.ScriptGlobals(), b.ScriptGlobals()) {
		t.Fatal("the replay's globals differ")
	}
}

func TestSeatScriptGlobalsRefusesABadList(t *testing.T) {
	w := globalWorld(t)
	held := w.ScriptGlobals()
	for _, list := range [][]ScriptGlobal{{{ID: 0}}, {{ID: 1025}}, {{ID: 3}, {ID: 3, Value: 1}}} {
		if w.SeatScriptGlobals(list) == nil {
			t.Errorf("seated %v", list)
		}
	}
	if !reflect.DeepEqual(w.ScriptGlobals(), held) {
		t.Fatal("a refused seat changed the globals")
	}
}

// The extended state carries the globals: a restore puts the saved values
// back, refuses a bad list unchanged, and an empty list holds none.
func TestScriptExtendedStateCarriesTheGlobals(t *testing.T) {
	w := globalWorld(t)
	st := w.ScriptExtendedState()
	if !reflect.DeepEqual(st.Globals, w.ScriptGlobals()) {
		t.Fatalf("state %+v", st)
	}
	st.Globals[1].Value = -4
	if err := w.RestoreScriptExtendedState(st); err != nil || w.ScriptGlobals()[1].Value != -4 {
		t.Fatal("the restore did not put the value back", err, w.ScriptGlobals())
	}
	held := w.ScriptGlobals()
	for _, list := range [][]ScriptGlobal{{{ID: 0}}, {{ID: 1025}}, {{ID: 3}, {ID: 2}}} {
		st.Globals = list
		if w.RestoreScriptExtendedState(st) == nil {
			t.Errorf("restored %v", list)
		}
	}
	if !reflect.DeepEqual(w.ScriptGlobals(), held) {
		t.Fatal("a refused restore changed the globals")
	}
	st.Globals = nil
	if err := w.RestoreScriptExtendedState(st); err != nil || w.ScriptGlobals() != nil {
		t.Fatal("an empty state left globals", err, w.ScriptGlobals())
	}
}
