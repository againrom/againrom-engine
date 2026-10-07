package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func rsbPrograms(t *testing.T) (*Script, *Script) {
	t.Helper()
	legacy := mustScript(t, []ScriptCheck{
		constCheck(4, 3),
		{Op: ScriptCheckUnitDistance, Register: 7, Unit: 7, HasUnit: true},
		{Op: ScriptCheckDistance, Register: 8, Args: [scriptParams]int32{12, 9}},
		{Op: 999, Register: 9},
	}, []ScriptInstant{
		{Op: ScriptInstantGiveUnit, Player: 2, HasPlayer: true},
		{Op: ScriptInstantGiveAll, Unit: 7, HasUnit: true},
		{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{60}},
		{Op: ScriptInstantWin},
	}, []ScriptTrigger{
		{Pairs: [3]ScriptPair{pair(4, 4, ScriptCmpEQ)}, Instants: acts(2, 3), Once: true, Latch: 17},
		{Pairs: [3]ScriptPair{pair(7, 4, ScriptCmpEQ)}, Instants: acts(0, 1), Once: true, Latch: 18},
		{Pairs: [3]ScriptPair{pair(9, 4, ScriptCmpEQ)}, Instants: acts(2), Once: true, Latch: 19},
	})
	checks, instants := legacy.Checks(), legacy.Instants()
	checks[1].Unit2, checks[1].HasUnit2 = 0, true
	checks[2].Unit, checks[2].HasUnit = 7, true
	instants[0].Unit, instants[0].HasUnit = 7, true
	instants[1].Unit2, instants[1].HasUnit2 = 0, true
	return legacy, mustScript(t, checks, instants, legacy.Triggers())
}

func rsbWorld(t *testing.T, script *Script) *World {
	t.Helper()
	w := scriptWorld(t, script, []Entity{
		{ID: 0, Owner: 1, X: 10, Y: 9, HP: 13, MaxHP: 13},
		{ID: 7, Owner: 1, X: 20, Y: 9, HP: 17, MaxHP: 17},
	})
	scriptTicks(w, 16, nil)
	if !w.ScriptLatched(17) || w.ScriptRegister(60) != 1 || w.Outcome() != OutcomeWon {
		t.Fatal("fixture did not reach a spent trigger and reported victory")
	}
	w.registers[4], w.registers[93], w.registers[99] = 12345, 814, -27
	w.latches[999] = 1
	w.won, w.lost = 9, 3
	w.purses[1] = 9876
	w.rawSessionHead[3], w.rawSessionMid[255] = 0x91, 0x37
	return w
}

func rsbForm(t *testing.T, w *World) []byte {
	t.Helper()
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return form
}

func TestRestoreScriptBindingsPreservesLiveStateAndNativeRoundTrip(t *testing.T) {
	legacy, current := rsbPrograms(t)
	legacyCopy := mustScript(t, legacy.Checks(), legacy.Instants(), legacy.Triggers())
	var resumed World
	if err := resumed.UnmarshalBinary(rsbForm(t, rsbWorld(t, legacy))); err != nil {
		t.Fatal(err)
	}
	expected := resumed
	expected.script = current
	want := rsbForm(t, &expected)

	if !resumed.RestoreScriptBindings(legacy, current) {
		t.Fatal("recognized saved program was not restored")
	}
	if !bytes.Equal(rsbForm(t, &resumed), want) {
		t.Fatal("repair changed state beyond the compiled unit bindings")
	}
	if !reflect.DeepEqual(legacy, legacyCopy) {
		t.Fatal("repair mutated the legacy program")
	}
	if resumed.script == current || resumed.script == legacy {
		t.Fatal("restored program aliases a caller's program")
	}

	var cold World
	if err := cold.UnmarshalBinary(rsbForm(t, &resumed)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rsbForm(t, &cold), want) {
		t.Fatal("restored bindings or live state failed to cross the byte form")
	}
	program := resumed.script
	if resumed.RestoreScriptBindings(legacy, current) || resumed.script != program ||
		!bytes.Equal(rsbForm(t, &resumed), want) {
		t.Fatal("repeated restore changed an already-current program")
	}
	current.checks[1].Unit2 = 99
	current.instants[0].Unit = 99
	if !bytes.Equal(rsbForm(t, &resumed), want) {
		t.Fatal("caller mutation reached the restored program")
	}
}

func TestRestoreScriptBindingsAcceptsEntityZero(t *testing.T) {
	legacy := mustScript(t, []ScriptCheck{{Op: ScriptCheckAlive, Register: 0}},
		[]ScriptInstant{{Op: ScriptInstantGiveUnit, Player: 2, HasPlayer: true}}, nil)
	checks, instants := legacy.Checks(), legacy.Instants()
	checks[0].HasUnit, instants[0].HasUnit = true, true
	current := mustScript(t, checks, instants, nil)
	w := scriptWorld(t, legacy, []Entity{{ID: 0, Owner: 1, X: 3, Y: 4, HP: 5, MaxHP: 5}})
	if !w.RestoreScriptBindings(legacy, current) {
		t.Fatal("entity zero was treated as an absent reference")
	}
	scriptTicks(w, 7, nil)
	if w.ScriptRegister(0) != 1 || !w.script.instants[0].HasUnit || w.script.instants[0].Unit != 0 {
		t.Fatal("restored entity-zero reference is not usable")
	}
}

func TestRestoreScriptBindingsRejectsNonBindingChangesAtomically(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Script)
	}{
		{"later instant argument", func(s *Script) { s.instants[3].Args[9] = 1 }},
		{"check opcode", func(s *Script) { s.checks[2].Op = ScriptCheckAlive }},
		{"constant preset", func(s *Script) { s.checks[0].Args[0]++ }},
		{"check register", func(s *Script) { s.checks[3].Register = 20 }},
		{"player", func(s *Script) { s.instants[0].Player++ }},
		{"item", func(s *Script) { s.instants[1].Item, s.instants[1].HasItem = 6, true }},
		{"group", func(s *Script) { s.checks[2].Group, s.checks[2].HasGroup = 5, true }},
		{"structure", func(s *Script) { s.checks[2].Structure, s.checks[2].HasStructure = 5, true }},
		{"trigger condition", func(s *Script) { s.triggers[2].Pairs[0].Cmp = ScriptCmpNE }},
		{"trigger action", func(s *Script) { s.triggers[2].Instants[0] = 3 }},
		{"trigger latch", func(s *Script) { s.triggers[2].Latch++ }},
		{"trigger once", func(s *Script) { s.triggers[2].Once = false }},
		{"check count", func(s *Script) { s.checks = s.checks[:3] }},
		{"instant count", func(s *Script) { s.instants = s.instants[:3] }},
		{"trigger count", func(s *Script) { s.triggers = s.triggers[:2] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy, current := rsbPrograms(t)
			w := rsbWorld(t, legacy)
			before, program := rsbForm(t, w), w.script
			tc.change(current)
			if w.RestoreScriptBindings(legacy, current) || w.script != program ||
				!bytes.Equal(rsbForm(t, w), before) {
				t.Fatal("incompatible current program was accepted or partially applied")
			}
		})
	}
}

func TestRestoreScriptBindingsRejectsChangedOrPartiallyRestoredSave(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Script)
	}{
		{"late saved instant", func(s *Script) { s.instants[3].Args[9] = 1 }},
		{"late saved trigger", func(s *Script) { s.triggers[2].Latch = 25 }},
		{"one binding already restored", func(s *Script) { s.checks[1].HasUnit2 = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy, current := rsbPrograms(t)
			saved := mustScript(t, legacy.Checks(), legacy.Instants(), legacy.Triggers())
			tc.change(saved)
			w := rsbWorld(t, saved)
			before, program := rsbForm(t, w), w.script
			if w.RestoreScriptBindings(legacy, current) || w.script != program ||
				!bytes.Equal(rsbForm(t, w), before) {
				t.Fatal("customized or partially restored save was changed")
			}
		})
	}
}

func TestRestoreScriptBindingsRefusesToChangeExistingReferences(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Script)
	}{
		{"check unit", func(s *Script) { s.checks[1].Unit = 0 }},
		{"instant unit", func(s *Script) { s.instants[1].Unit = 0 }},
		{"check presence", func(s *Script) { s.checks[1].HasUnit = false }},
		{"instant presence", func(s *Script) { s.instants[1].HasUnit = false }},
		{"still-absent value", func(s *Script) { s.checks[3].Unit = 7 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy, current := rsbPrograms(t)
			w := rsbWorld(t, legacy)
			before := rsbForm(t, w)
			tc.change(current)
			if w.RestoreScriptBindings(legacy, current) || !bytes.Equal(rsbForm(t, w), before) {
				t.Fatal("an existing or still-absent reference was rewritten")
			}
		})
	}
}

func TestRestoreScriptBindingsRejectsInertMismatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		saved bool
		index int
		inert bool
	}{
		{"saved adds inert", true, 0, true},
		{"saved clears inert", true, 2, false},
		{"current adds inert", false, 0, true},
		{"current clears inert", false, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			legacy, current := rsbPrograms(t)
			w := rsbWorld(t, mustScript(t, legacy.Checks(), legacy.Instants(), legacy.Triggers()))
			subject := current
			if tc.saved {
				subject = w.script
			}
			subject.triggers[tc.index].Inert = tc.inert
			before, program, triggers := rsbForm(t, w), w.script, w.script.Triggers()
			if w.RestoreScriptBindings(legacy, current) || w.script != program ||
				!bytes.Equal(rsbForm(t, w), before) || !reflect.DeepEqual(w.script.Triggers(), triggers) {
				t.Fatal("Inert mismatch was accepted or changed the saved program")
			}
		})
	}
}

func TestRestoreScriptBindingsRebuildsStaleGapLists(t *testing.T) {
	legacy, current := rsbPrograms(t)
	expected := mustScript(t, current.Checks(), current.Instants(), current.Triggers())
	w := rsbWorld(t, mustScript(t, legacy.Checks(), legacy.Instants(), legacy.Triggers()))
	current.gaps, current.inert = nil, nil
	if !w.RestoreScriptBindings(legacy, current) || !reflect.DeepEqual(w.script, expected) {
		t.Fatal("stale gap lists were copied instead of rebuilt")
	}
}

func TestRestoreScriptBindingsNoOp(t *testing.T) {
	legacy, current := rsbPrograms(t)
	var absent *World
	if absent.RestoreScriptBindings(legacy, current) {
		t.Fatal("nil world reported a repair")
	}
	for _, tc := range []struct {
		name                   string
		saved, legacy, current *Script
	}{
		{"missing saved program", nil, legacy, current},
		{"missing legacy", legacy, nil, current},
		{"missing current", legacy, legacy, nil},
		{"no changed binding", legacy, legacy, legacy},
		{"already current", current, legacy, current},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := scriptWorld(t, tc.saved, nil)
			before, program := rsbForm(t, w), w.script
			if w.RestoreScriptBindings(tc.legacy, tc.current) || w.script != program ||
				!bytes.Equal(rsbForm(t, w), before) {
				t.Fatal("no-op input changed the world")
			}
		})
	}
}
