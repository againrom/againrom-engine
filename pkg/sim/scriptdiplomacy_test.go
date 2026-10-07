package sim

import (
	"bytes"
	"testing"
)

// Check opcode 10 and instant opcode 10: what one player thinks of another,
// and the one arm that changes it. The relation matrix itself is
// relations.go's own and is exercised here only through the two script arms
// that read and write it — relations_test.go already owns the matrix's own
// contract.

// relationCheck is one check of arm 10, naming both players.
func relationCheck(reg int32, p1, p2 uint32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckRelation, Register: reg,
		Player: p1, HasPlayer: true, Player2: p2, HasPlayer2: true}
}

// relationInstant is one instant of arm 10: three plain parameters, no
// player reference at all.
func relationInstant(p0, p1, p2 int32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantRelation, Args: [scriptParams]int32{p0, p1, p2}}
}

// relCell is a relation byte slice of relationLen with exactly one cell set to
// v; every other cell, including the diagonal, stays zero. The caller may set
// further cells in the slice it gets back before handing it to relationWorld.
func relCell(from, to uint32, v byte) []byte {
	cells := make([]byte, relationLen)
	cells[from*relationSlots+to] = v
	return cells
}

// relationWorld is a world built over an explicit initial relation and no
// entities: neither script arm under test names a unit.
func relationWorld(t *testing.T, s *Script, cells []byte) *World {
	t.Helper()
	rel, err := NewRelations(cells)
	if err != nil {
		t.Fatalf("NewRelations: %v", err)
	}
	w, err := NewLootWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, nil, s, rel, nil)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w
}

// diplomacyScript is a script with no checks at all, whose one always-true,
// one-shot trigger runs exactly the named instant of arm 10 — the "no pair at
// all" idiom TestThePairsAreANDedAndTheANDOfNothingIsTrue already establishes.
func diplomacyScript(t *testing.T, p0, p1, p2 int32) *Script {
	t.Helper()
	return mustScript(t, nil, []ScriptInstant{relationInstant(p0, p1, p2)},
		[]ScriptTrigger{{Instants: acts(0), Once: true, Latch: 0}})
}

// ---------------------------------------------------------------- AC-3, check 10

func TestARelationCheckMasksToTheLowTwoBits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		v    byte
		want int32
	}{
		{"every bit set: only the low two survive", 0xff, 3},
		{"high bits set, low two clear", 0xfc, 0},
		{"bit 2 alone must not leak into the mask", 0b0000_0101, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := mustScript(t, []ScriptCheck{relationCheck(0, 1, 2)}, nil, nil)
			w := relationWorld(t, s, relCell(1, 2, tc.v))
			scriptTicks(w, 7, nil)
			if got := w.ScriptRegister(0); got != tc.want {
				t.Errorf("cell %#02x reads register %d, want %d", tc.v, got, tc.want)
			}
		})
	}
}

func TestARelationCheckOnASlotTheMatrixDoesNotHoldReadsZero(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		relationCheck(0, 0, 5),             // slot 0 names no roster entry
		relationCheck(1, relationSlots, 5), // past the matrix's own side
		relationCheck(2, 1, 2),             // the control: a slot the matrix holds
	}, nil, nil)
	w := relationWorld(t, s, relCell(1, 2, 0xff))
	scriptTicks(w, 7, nil)
	if got := w.ScriptRegister(0); got != 0 {
		t.Errorf("slot 0 reads %d, want 0", got)
	}
	if got := w.ScriptRegister(1); got != 0 {
		t.Errorf("a slot past the matrix reads %d, want 0", got)
	}
	if got := w.ScriptRegister(2); got != 3 {
		t.Errorf("the control measured %d, want 3 — the arm still works for a slot the matrix holds", got)
	}
}

func TestARelationCheckNamingFewerThanTwoPlayersWritesNoRegister(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{77}},
		{Op: ScriptCheckRelation, Register: 1, Player: 1, HasPlayer: true},   // no second player
		{Op: ScriptCheckRelation, Register: 2, Player2: 2, HasPlayer2: true}, // no first player
		{Op: ScriptCheckRelation, Register: 3},                               // neither
		relationCheck(4, 1, 2),                                               // the control
	}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{pair(1, 1, ScriptCmpEQ)},
		Instants: acts(),
		Latch:    0}})
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("this script holds only implemented arms; report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := relationWorld(t, s, relCell(1, 2, 0xff))
	w.registers[1], w.registers[2], w.registers[3] = 77, 77, 77
	scriptTicks(w, 16, nil)

	if got := w.ScriptRegister(1); got != 77 {
		t.Errorf("naming a first player alone wrote %d, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(2); got != 77 {
		t.Errorf("naming a second player alone wrote %d, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(3); got != 77 {
		t.Errorf("naming no player wrote %d, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(4); got != 3 {
		t.Errorf("the control naming both players measured %d, want 3 — the arm still works", got)
	}
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading register 1 was not evaluated; this arm is implemented, " +
			"so nothing about it is inert")
	}
}

// TestARelationCheckIsSilencedByDesign is the trace half of the rule above:
// naming no player at all is recorded as ScriptSilenceNoPlayer, the same kind
// checks 8 and 15 already use.
func TestARelationCheckIsSilencedByDesign(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{{Op: ScriptCheckRelation, Register: 0}}, nil, nil)
	w, err := NewLootWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{}, nil, s, Relations{}, nil)
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	for w.tick%scriptCycle != scriptPassPhase {
		Step(w, nil)
	}
	tr := StepTraced(w, nil)
	if len(tr.Silent) != 1 || tr.Silent[0].Check != 0 || tr.Silent[0].Why != ScriptSilenceNoPlayer {
		t.Fatalf("the silence is %+v, want one entry naming check 0 and ScriptSilenceNoPlayer", tr.Silent)
	}
}

// TestBothArm10sAreSupportedAndNotInert is check 10 and instant 10 both
// reported implemented, and a trigger reading either evaluated rather than
// held inert.
func TestBothArm10sAreSupportedAndNotInert(t *testing.T) {
	t.Parallel()

	if !scriptCheckSupported(ScriptCheckRelation) {
		t.Error("ScriptCheckRelation (10) is not in scriptCheckSupported")
	}
	if !scriptInstantSupported(ScriptInstant{Op: ScriptInstantRelation}) {
		t.Error("ScriptInstantRelation (10) is not in scriptInstantSupported")
	}

	s := mustScript(t, []ScriptCheck{relationCheck(0, 1, 2), constCheck(1, 3)}, nil,
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(), Latch: 0}})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("Unsupported() is %+v, want none — the arm is implemented", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Fatalf("InertTriggers() is %v, want none", inert)
	}

	w := relationWorld(t, s, relCell(1, 2, 0xff))
	scriptTicks(w, 7, nil)
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading the relation register did not evaluate")
	}
}

// ---------------------------------------------------------------- AC-4, instant 10

func TestTheDiplomacyWriteTakesOneDirection(t *testing.T) {
	t.Parallel()

	cells := relCell(2, 3, 0)
	cells[3*relationSlots+2] = 0xaa // the mirror cell

	s := diplomacyScript(t, 2, 3, 1)
	w := relationWorld(t, s, cells)
	scriptTicks(w, 16, nil)

	rel := w.Relations()
	if got := rel.Byte(2, 3); got != 1 {
		t.Errorf("cell [2][3] is %#02x, want 1", got)
	}
	if got := rel.Byte(3, 2); got != 0xaa {
		t.Errorf("the mirror cell [3][2] moved to %#02x, want the untouched 0xaa", got)
	}
}

func TestTheDiplomacyWriteIsAReadModifyWriteNotAnAssignment(t *testing.T) {
	t.Parallel()

	s := diplomacyScript(t, 2, 3, 1)
	w := relationWorld(t, s, relCell(2, 3, 0xfc)) // every bit above 1 set, low two clear
	scriptTicks(w, 16, nil)

	if got := w.Relations().Byte(2, 3); got != 0xfd {
		t.Errorf("cell [2][3] is %#02x, want 0xfd — bits 2..7 must survive the write", got)
	}
}

func TestTheDiplomacyWriteAddsRatherThanOrs(t *testing.T) {
	t.Parallel()

	s := diplomacyScript(t, 2, 3, 5)
	w := relationWorld(t, s, relCell(2, 3, 4))
	scriptTicks(w, 16, nil)

	if got := w.Relations().Byte(2, 3); got != 9 {
		t.Errorf("cell [2][3] is %d, want 9 — the parameter is added, not or-ed in", got)
	}
}

func TestTheDiplomacyWritesSumTruncatesInTheByte(t *testing.T) {
	t.Parallel()

	s := diplomacyScript(t, 2, 3, 6)
	w := relationWorld(t, s, relCell(2, 3, 0xfc))
	scriptTicks(w, 16, nil)

	if got := w.Relations().Byte(2, 3); got != 2 {
		t.Errorf("cell [2][3] is %d, want 2 — the sum truncates in the byte", got)
	}
}

func TestTheDiplomacyWriteOverwritesALockedPair(t *testing.T) {
	t.Parallel()

	s := diplomacyScript(t, 2, 3, 1)
	w := relationWorld(t, s, relCell(2, 3, relationLocked))
	scriptTicks(w, 16, nil)

	if got := w.Relations().Byte(2, 3); got != 1 {
		t.Errorf("the locked cell reads %#02x after the write, want 1 — the lock is not respected", got)
	}
}

func TestTheDiplomacyWriteOnASlotTheMatrixDoesNotHoldIsANoOp(t *testing.T) {
	t.Parallel()

	cells := relCell(5, 6, 0x2a) // an unrelated cell, so the matrix is not all-zero to start
	for _, p0 := range []int32{0, relationSlots, relationSlots + 1} {
		s := diplomacyScript(t, p0, 3, 9)
		w := relationWorld(t, s, cells)
		scriptTicks(w, 16, nil)
		if !w.ScriptLatched(0) {
			t.Fatalf("p0=%d: the trigger never fired; the test proves nothing", p0)
		}
		got := w.Relations()
		if !bytes.Equal(got.cells, cells) {
			t.Errorf("p0=%d: the matrix changed to %v, want it untouched at %v", p0, got.cells, cells)
		}
	}
}

func TestTheDiplomacyWriteIsBareBesideItsOwnCell(t *testing.T) {
	t.Parallel()

	s := diplomacyScript(t, 2, 3, 1)
	w := relationWorld(t, s, relCell(2, 3, 0))
	scriptTicks(w, 16, nil)

	if won, lost := w.ScriptCounters(); won != 0 || lost != 0 {
		t.Errorf("the counters moved to (%d,%d); the write is bare and does nothing else", won, lost)
	}
	if got := w.Outcome(); got != OutcomeUndecided {
		t.Errorf("the outcome is %d, want undecided", got)
	}
}
