package sim

import "testing"

func changeScript(t *testing.T, dialect ScriptDialect, args [10]int32) *Script {
	t.Helper()
	cs := []ScriptCheck{{Op: 0x4105, HasUnit: true, Unit: 1, Args: args}, constCheck(1, 1)}
	in := []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}}
	s, err := NewExtendedScript(dialect, 1000, cs, in, []ScriptTrigger{
		{Instants: acts(), Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ)}, Actions: []int32{0}}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func moveTo(w *World, x, y int32) {
	w.entities[0].X, w.entities[0].Y = x, y
}

func passResult(w *World) int32 {
	w.scriptPass(nil)
	return w.checkResult(0)
}

func TestUnitChangesItsCellWritesPerPass(t *testing.T) {
	for _, dialect := range []ScriptDialect{ScriptROM1, ScriptROM2} {
		w := scriptWorld(t, changeScript(t, dialect, [10]int32{10, 10}), []Entity{{ID: 1, Owner: SelfSlot, X: 30, Y: 40, HP: 100, MaxHP: 100}})
		steps := []struct {
			x, y, want int32
			what       string
		}{
			{30, 40, 0, "first pass"}, {30, 40, 0, "standing"}, {30, 41, 1, "step"}, {30, 41, 0, "standing after the step"},
			{30, 42, 1, "second step"}, {5, 6, 1, "teleport"}, {5, 6, 0, "standing"}, {9, 9, 1, "several cells at once"},
		}
		for _, s := range steps {
			moveTo(w, s.x, s.y)
			if got := passResult(w); got != s.want {
				t.Fatalf("dialect %v, %s: wrote %d, want %d", dialect, s.what, got, s.want)
			}
		}
		if got := w.ScriptRegister(7); got != 4 {
			t.Fatalf("dialect %v: the repeating trigger fired %d times, want 4", dialect, got)
		}
	}
}

func TestUnitChangesItsCellAtOneShotWithVariable(t *testing.T) {
	cs := []ScriptCheck{
		{Op: 0x4105, HasUnit: true, Unit: 1}, constCheck(1, 1),
		{Op: ScriptCheckVariable, Register: 2, Args: [10]int32{6}}, {Op: ScriptCheckVariable, Register: 3, Args: [10]int32{9}},
	}
	in := []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}, {Op: ScriptInstantSetVariable, Args: [10]int32{6, 1}}}
	s, err := NewExtendedScript(ScriptROM1, 1000, cs, in, []ScriptTrigger{
		{Instants: acts(), Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ), pair(2, 1, ScriptCmpEQ)}, Actions: []int32{0}, Once: true, Latch: 0},
		{Instants: acts(), Conditions: []ScriptPair{pair(3, 1, ScriptCmpEQ)}, Actions: []int32{1}, Once: true, Latch: 1}})
	if err != nil {
		t.Fatal(err)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1, Owner: SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100}})
	for y := int32(10); y < 14; y++ {
		moveTo(w, 10, y)
		w.scriptPass(nil)
	}
	if w.ScriptRegister(7) != 0 || w.ScriptRegister(6) != 0 {
		t.Fatalf("walking with N=0 fired: fired %d, N %d", w.ScriptRegister(7), w.ScriptRegister(6))
	}
	w.setRegisterAt(9, 1)
	w.scriptPass(nil)
	w.scriptPass(nil)
	if w.ScriptRegister(6) != 1 || w.ScriptRegister(7) != 0 {
		t.Fatalf("N set while standing: fired %d, N %d", w.ScriptRegister(7), w.ScriptRegister(6))
	}
	moveTo(w, 10, 20)
	w.scriptPass(nil)
	if w.ScriptRegister(7) != 1 {
		t.Fatalf("the first step fired %d times, want 1", w.ScriptRegister(7))
	}
	for y := int32(21); y < 25; y++ {
		moveTo(w, 10, y)
		w.scriptPass(nil)
	}
	if w.ScriptRegister(7) != 1 {
		t.Fatalf("more steps fired %d times in all, want 1", w.ScriptRegister(7))
	}
}

func TestUnitChangesItsCellSurvivesAStateRestore(t *testing.T) {
	path := [][2]int32{{3, 3}, {3, 3}, {3, 4}, {3, 4}, {4, 4}, {4, 4}, {9, 1}}
	run := func(w *World, from int) []int32 {
		var out []int32
		for _, p := range path[from:] {
			moveTo(w, p[0], p[1])
			out = append(out, passResult(w))
		}
		return out
	}
	s := changeScript(t, ScriptROM1, [10]int32{})
	ents := func() []Entity { return []Entity{{ID: 1, Owner: SelfSlot, X: 3, Y: 3, HP: 100, MaxHP: 100}} }
	whole := run(scriptWorld(t, s, ents()), 0)
	first := scriptWorld(t, s, ents())
	for _, p := range path[:3] {
		moveTo(first, p[0], p[1])
		first.scriptPass(nil)
	}
	state := first.ScriptExtendedState()
	second := scriptWorld(t, s, ents())
	if err := second.RestoreScriptExtendedState(state); err != nil {
		t.Fatal(err)
	}
	tail := run(second, 3)
	for i, got := range tail {
		if got != whole[3+i] {
			t.Fatalf("pass %d after the restore wrote %d, the run without it %d", 3+i, got, whole[3+i])
		}
	}
	if !state.Cells[0].Seen {
		t.Fatal("the state did not carry the remembered cell")
	}
}

func TestUnitChangesItsCellSurvivesTheByteForm(t *testing.T) {
	w := scriptWorld(t, changeScript(t, ScriptROM1, [10]int32{}), []Entity{{ID: 1, Owner: SelfSlot, X: 3, Y: 3, HP: 100, MaxHP: 100}})
	passResult(w)
	back := worldRoundTripForTest(t, w)
	moveTo(back, 3, 4)
	if got := passResult(back); got != 1 {
		t.Fatalf("after the byte form the step wrote %d, want 1", got)
	}
}

func TestUnitChangesItsCellOldRecordAndOldSaveStartAtTheFirstPass(t *testing.T) {
	w := scriptWorld(t, changeScript(t, ScriptROM1, [10]int32{10, 10}), []Entity{{ID: 1, Owner: SelfSlot, X: 12, Y: 12, HP: 100, MaxHP: 100}})
	if err := w.RestoreScriptExtendedState(ScriptExtendedState{Variables: w.ScriptExtendedState().Variables, Results: w.ScriptExtendedState().Results, Latches: w.ScriptExtendedState().Latches}); err != nil {
		t.Fatal(err)
	}
	if got := passResult(w); got != 0 || w.ScriptRegister(7) != 0 {
		t.Fatalf("a record with a stored cell fired on its first pass: %d", got)
	}
}

func TestUnitChangesItsCellRunsOnTheSecondGamesExtendedMachine(t *testing.T) {
	s := changeScript(t, ScriptROM2, [10]int32{})
	if s.Dialect() != ScriptROM2 || !s.Extended() || len(s.gaps) != 0 {
		t.Fatalf("dialect %v, extended %t, gaps %v", s.Dialect(), s.Extended(), s.gaps)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1, Owner: SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100}})
	scriptTicks(w, 32, nil)
	if w.ScriptRegister(7) != 0 {
		t.Fatal("fired on the first cell")
	}
	scriptTicks(w, 16, []Command{MoveTo(1, CellPoint{X: 10, Y: 11})})
	if w.ScriptRegister(7) != 1 {
		t.Fatalf("one step fired %d times, want 1", w.ScriptRegister(7))
	}
}

func TestUnitChangesItsCellIsInertOnPlainMachines(t *testing.T) {
	checks := []ScriptCheck{{Op: 0x4105, Unit: 1, HasUnit: true}, constCheck(1, 0)}
	trigger := ScriptTrigger{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0)}
	for _, build := range []func([]ScriptCheck, []ScriptInstant, []ScriptTrigger) (*Script, error){NewScript, NewROM2Script} {
		s, err := build(checks, []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}}, []ScriptTrigger{trigger})
		if err != nil {
			t.Fatal(err)
		}
		if !s.Triggers()[0].Inert {
			t.Fatalf("dialect %v executed the extension in an original-format trigger", s.Dialect())
		}
	}
}

func TestUnitChangesItsCellUsesTheRetainedCell(t *testing.T) {
	for _, removed := range []bool{false, true} {
		s := changeScript(t, ScriptROM1, [10]int32{})
		w := scriptWorld(t, s, []Entity{{ID: 1, X: 10, Y: 10, HP: 100, MaxHP: 100}})
		passResult(w)
		if removed {
			w.currentTerminalActors = []CurrentTerminalActor{{ID: 1, Cell: uint16(11<<8 | 10), Stage: uint8(DecayBones), HP: -10}}
			w.entities = nil
		} else {
			w.entities[0].Y, w.entities[0].HP = 11, -10
		}
		if got := passResult(w); got != 1 {
			t.Fatalf("removed=%t: the retained cell wrote %d, want 1", removed, got)
		}
	}
}

func TestExtendedProgramKeepsItsLateChecksAcrossTheByteForm(t *testing.T) {
	cs := []ScriptCheck{{Op: 0x4105, HasUnit: true, Unit: 1, Late: true}, constCheck(1, 1)}
	in := []ScriptInstant{{Op: ScriptInstantIncVariable, Args: [10]int32{7}}}
	s, err := NewExtendedScript(ScriptROM2, 1000, cs, in, []ScriptTrigger{
		{Instants: acts(), Conditions: []ScriptPair{pair(0, 1, ScriptCmpEQ)}, Actions: []int32{0}}})
	if err != nil {
		t.Fatal(err)
	}
	w := scriptWorld(t, s, []Entity{{ID: 1, Owner: SelfSlot, X: 10, Y: 10, HP: 100, MaxHP: 100}})
	scriptTicks(w, 8, nil)
	back := worldRoundTripForTest(t, w)
	if !back.Script().Checks()[0].Late || back.Hash() != w.Hash() {
		t.Fatal("the byte form lost the late flag of an extended program's check")
	}
	scriptTicks(w, 64, []Command{MoveTo(1, CellPoint{X: 10, Y: 11})})
	scriptTicks(back, 64, []Command{MoveTo(1, CellPoint{X: 10, Y: 11})})
	if w.ScriptRegister(7) != back.ScriptRegister(7) || w.Hash() != back.Hash() {
		t.Fatalf("live fired %d times, loaded %d", w.ScriptRegister(7), back.ScriptRegister(7))
	}
}
