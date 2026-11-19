package sim

import "testing"

func TestSetVariableCommandWritesTheRegisterAScriptReads(t *testing.T) {
	w := engWorld(t, engRel(t), engFighter(10, SelfSlot, 5, 5))
	Step(w, []Command{SetVariable(10, 2), SetVariable(11, -7)})
	if got := w.ScriptRegister(10); got != 2 {
		t.Errorf("variable 10 = %d, want 2", got)
	}
	if got := w.ScriptRegister(11); got != -7 {
		t.Errorf("variable 11 = %d, want -7", got)
	}
}

func TestSetVariableCommandIgnoresAVariableTheProgramLacks(t *testing.T) {
	w := engWorld(t, engRel(t), engFighter(10, SelfSlot, 5, 5))
	before := w.ScriptRegisters()
	Step(w, []Command{SetVariable(-1, 5), SetVariable(scriptRegisters, 5)})
	if w.ScriptRegisters() != before || w.ScriptRegister(scriptRegisters) != 0 {
		t.Errorf("a variable the program lacks changed the registers")
	}
}

func TestSetVariableCommandIsReplayedToTheSameWorld(t *testing.T) {
	build := func() *World { return engWorld(t, engRel(t), engFighter(10, SelfSlot, 5, 5)) }
	a, b, c := build(), build(), build()
	Step(a, []Command{SetVariable(10, 3)})
	Step(b, []Command{SetVariable(10, 3)})
	Step(c, nil)
	ha, hb, hc := a.Hash(), b.Hash(), c.Hash()
	if ha != hb {
		t.Errorf("the same command gave hashes %x and %x", ha, hb)
	}
	if ha == hc {
		t.Errorf("the command left the world hash unchanged: %x", ha)
	}
}
