package sim

// The sack query — check opcode 14 (AC-7, 0103 T3).

import "testing"

// sackCheckBounds is the small extent every fixture here is built against.
var sackCheckBounds = Bounds{Width: 10, Height: 10}

// sackCheckWorld builds a world running s over sacks, with no entities: a
// check-14 node names no unit, so nothing here ever needs one.
func sackCheckWorld(t *testing.T, s *Script, sacks []Sack) *World {
	t.Helper()
	w, err := NewLootWorld(1, sackCheckBounds, ModeCanonical, Terrain{}, nil, s, Relations{}, sacks)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w
}

// sackCheck is one compiled check-14 node over (x, y).
func sackCheck(reg, x, y int32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckSackAt, Register: reg, Args: [scriptParams]int32{x, y}}
}

func TestTheSackQueryEvaluatesTheNamedCell(t *testing.T) {
	sacks := []Sack{{X: 5, Y: 3, Gold: 10, Items: []uint16{0x0100}}}

	tests := []struct {
		name  string
		sacks []Sack
		x, y  int32
		want  int32
	}{
		// Gold and Items ride on the sack this case finds, and neither reaches the
		// register: the want below is 1, not 10 or a count.
		{"a cell with a sack", sacks, 5, 3, 1},
		{"a neighbouring cell with none", sacks, 5, 4, 0},
		{"no sacks in the world at all", nil, 5, 3, 0},
		{"a cell outside the world's bounds", sacks, 50, 50, 0},
		{"a coordinate above 255 truncates to its low byte (FR-17)", sacks, 5 + 256, 3, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t, []ScriptCheck{sackCheck(0, tc.x, tc.y)}, nil, nil)
			w := sackCheckWorld(t, s, tc.sacks)
			scriptTicks(w, 7, nil)
			if got := w.ScriptRegister(0); got != tc.want {
				t.Errorf("register 0 is %d, want %d", got, tc.want)
			}
		})
	}
}

func TestASackQueryOutsideBoundsDoesNotStopThePass(t *testing.T) {
	s := mustScript(t, []ScriptCheck{
		sackCheck(0, 200, 200), // outside sackCheckBounds
		sackCheck(1, 5, 3),     // a real sack, evaluated after it
	}, nil, nil)
	w := sackCheckWorld(t, s, []Sack{{X: 5, Y: 3}})
	scriptTicks(w, 7, nil)
	if got := w.ScriptRegister(0); got != 0 {
		t.Errorf("register 0 (outside bounds) is %d, want 0", got)
	}
	if got := w.ScriptRegister(1); got != 1 {
		t.Errorf("register 1 is %d, want 1 — the out-of-bounds check must not have stopped the pass", got)
	}
}

// ---------------------------------------------------------------- AC-7

// TestATriggerConditionedOnTheSackQueryFiresExactlyWhenOneIsThere is AC-7's
// trigger half.
func TestATriggerConditionedOnTheSackQueryFiresExactlyWhenOneIsThere(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{sackCheck(0, 5, 3), constCheck(1, 1)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(0), Once: true, Latch: 0}})

	empty := sackCheckWorld(t, s, nil)
	scriptTicks(empty, 40, nil)
	if won, _ := empty.ScriptCounters(); won != 0 {
		t.Errorf("the trigger fired with no sack on the map: won=%d", won)
	}

	occupied := sackCheckWorld(t, s, []Sack{{X: 5, Y: 3}})
	scriptTicks(occupied, 40, nil)
	if won, _ := occupied.ScriptCounters(); won != 1 {
		t.Errorf("the trigger did not fire with a sack at the named cell: won=%d", won)
	}
}

func TestTheSackQueryIsNoLongerReportedUnsupported(t *testing.T) {
	if !scriptCheckSupported(ScriptCheckSackAt) {
		t.Fatal("scriptCheckSupported(14) is false; the sack query is implemented now")
	}
	s := mustScript(t,
		[]ScriptCheck{sackCheck(0, 1, 1), constCheck(1, 0)},
		nil,
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(), Latch: 0}})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Errorf("Unsupported() is %+v, want none", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Errorf("InertTriggers() is %v, want none — the trigger reads only implemented arms", inert)
	}
}
