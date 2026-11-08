package sim

import "testing"

func TestAliveCheckHoldsABodyPlacedAtZeroHealth(t *testing.T) {
	const dying = 12
	lost := func(flag, hp int32) uint32 {
		alive := ScriptCheck{Op: ScriptCheckAlive, Register: 0, Unit: 1, HasUnit: true}
		alive.Args[ScriptAliveHeldBody] = flag
		s := mustScript(t,
			[]ScriptCheck{alive, constCheck(1, 0)},
			[]ScriptInstant{{Op: ScriptInstantLose}},
			[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
		body := Entity{ID: 1, HP: hp, MaxHP: 333, DyingTime: dying}
		PrepareAuthoredBody(&body)
		w := scriptWorld(t, s, []Entity{body})
		scriptTicks(w, dying+40, nil)
		_, n := w.ScriptCounters()
		return n
	}
	if n := lost(1, 0); n != 0 {
		t.Errorf("a marked body at health 0 lost the mission after its dwell: lost %d", n)
	}
	if n := lost(0, 0); n == 0 {
		t.Error("an unmarked body at health 0 stopped reading dead after its dwell")
	}
	if n := lost(1, -1); n == 0 {
		t.Error("a marked body below health 0 stopped reading dead after its dwell")
	}
}
