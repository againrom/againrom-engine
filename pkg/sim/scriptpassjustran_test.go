package sim

import "testing"

func TestScriptPassJustRanFollowsTheStepJustCompleted(t *testing.T) {
	var w World
	for tick := uint64(0); tick < 40; tick++ {
		w.tick = tick
		want := tick > 0 && (tick-1)%scriptCycle == scriptPassPhase
		if got := w.ScriptPassJustRan(); got != want {
			t.Fatalf("legacy clock, tick %d: %v, want %v", tick, got, want)
		}
	}
	w.hasSessionClock = true
	for _, entry := range []uint32{0, 5, 6, 7, 22, 0x7ffffff6, 0xfffffff0, 0xfffffff6, 0xffffffff} {
		w.tick = uint64(entry + 1)
		want := SessionClock{SubTick: entry}.scriptDue()
		if got := w.ScriptPassJustRan(); got != want {
			t.Fatalf("session clock entered at %#x: %v, want %v", entry, got, want)
		}
	}
}
