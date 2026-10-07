package sim

import (
	"bytes"
	"strings"
	"testing"
)

func originalSessionFixture() OriginalSession {
	return OriginalSession{
		TriggerLatches: make([]byte, scriptLatches),
		Diplomacy:      make([]byte, relationLen),
	}
}

func originalSessionWorld(t *testing.T) *World {
	t.Helper()
	script := mustScript(t, []ScriptCheck{constCheck(5, 73)},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{{Instants: acts(0), Once: true, Latch: 7}})
	w, err := NewRelatedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 2, Y: 2, Owner: 1}}, script, Relations{})
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

func TestOriginalSessionRestoresSpentTriggersAndDirectionalDiplomacy(t *testing.T) {
	w := originalSessionWorld(t)
	state := originalSessionFixture()
	state.TriggerLatches[7] = 1
	state.Diplomacy[0] = 0xa5
	state.Diplomacy[4] = 0x5a
	state.Diplomacy[2*relationSlots] = 0x3c
	state.Diplomacy[3*relationSlots+4] = 1
	state.Diplomacy[4*relationSlots+3] = 2

	if err := w.ImportOriginalSession(state); err != nil {
		t.Fatalf("ImportOriginalSession: %v", err)
	}
	if !w.ScriptLatched(7) {
		t.Fatal("the saved one-shot latch was not restored")
	}
	if got := w.ScriptRegister(5); got != 73 {
		t.Fatalf("session import overlaid a builder-owned result register with %d, want 73", got)
	}
	if !bytes.Equal(w.relations.cells, state.Diplomacy) {
		t.Fatal("the complete 50x50 diplomacy matrix was not restored byte-for-byte")
	}
	if !w.Relations().Hostile(3, 4) || w.Relations().Hostile(4, 3) || !w.Relations().Locked(4, 3) {
		t.Fatalf("directional relation reads %#x/%#x, want hostile/locked",
			w.Relations().Byte(3, 4), w.Relations().Byte(4, 3))
	}

	// The compiled trigger is unconditionally true. A fresh world would win on
	// its first script pass; the imported latch makes it spent, proving the
	// foreign state gates the programme instead of merely being reported.
	for i := 0; i < 16; i++ {
		Step(w, nil)
	}
	if w.Outcome() != OutcomeUndecided {
		t.Fatalf("the already-spent one-shot ran again: outcome %v", w.Outcome())
	}
	if won, lost := w.ScriptCounters(); won != 0 || lost != 0 {
		t.Fatalf("the already-spent one-shot moved counters to %d/%d", won, lost)
	}
	if !w.ScriptLatched(7) {
		t.Fatal("the spent one-shot cleared its imported latch")
	}

	// Import owns a copy. The save decoder's buffers cannot mutate a running
	// world after the handoff returns.
	state.TriggerLatches[7] = 0
	state.Diplomacy[3*relationSlots+4] = 0
	if !w.ScriptLatched(7) || !w.Relations().Hostile(3, 4) {
		t.Fatal("mutating the caller's session changed the imported world")
	}
}

func TestOriginalSessionRestoresRegistersPerSlotClass(t *testing.T) {
	w := originalSessionWorld(t)
	state := originalSessionFixture()
	state.Registers[5] = 999 // a constant-owned slot: must not survive
	state.Registers[50] = 4242
	state.RawHead[0], state.RawHead[47] = 0x11, 0x22
	state.RawMid[0], state.RawMid[399] = 0x33, 0x44

	if err := w.ImportOriginalSession(state); err != nil {
		t.Fatalf("ImportOriginalSession: %v", err)
	}
	if got := w.ScriptRegister(5); got != 73 {
		t.Fatalf("constant-owned register 5 = %d after import, want 73 (the compiled constant, not the file's 999)", got)
	}
	if got := w.ScriptRegister(50); got != 4242 {
		t.Fatalf("authored variable register 50 = %d after import, want 4242 (the file's own value)", got)
	}
	regs := w.ScriptRegisters()
	if regs[5] != 73 || regs[50] != 4242 {
		t.Fatalf("ScriptRegisters()[5]/[50] = %d/%d, want 73/4242", regs[5], regs[50])
	}
	for i, v := range regs {
		if i == 5 || i == 50 {
			continue
		}
		if v != 0 {
			t.Fatalf("register %d = %d, want 0 (only 5 and 50 were given non-zero input)", i, v)
		}
	}
	if got := w.RawSessionHead(); got != state.RawHead {
		t.Fatalf("RawSessionHead() = %v, want %v", got, state.RawHead)
	}
	if got := w.RawSessionMid(); got != state.RawMid {
		t.Fatalf("RawSessionMid() = %v, want %v", got, state.RawMid)
	}
}

// TestOriginalSessionRegistersSurviveARealCheckOnlyUntilItRuns is the
// ordinary check-owned slot class: TRIG-SAVE-008/MISSION-SLOT-008 state the
// original neither presets nor protects it, so a restored value there is
// live only until that check's own next full tick recomputes it. A
// ScriptCheckAlive on the one entity always answers 1 (ALIVE), so the
// mismatch between an implausible restored value and the check's own answer
// is the witness that the recompute, not the restore, wins after a tick.
func TestOriginalSessionRegistersSurviveARealCheckOnlyUntilItRuns(t *testing.T) {
	script := mustScript(t,
		[]ScriptCheck{{Op: ScriptCheckAlive, Register: 9, Unit: 1, HasUnit: true}},
		nil, nil)
	w, err := NewRelatedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 2, Y: 2, Owner: 1}}, script, Relations{})
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	state := originalSessionFixture()
	state.Registers[9] = -777
	if err := w.ImportOriginalSession(state); err != nil {
		t.Fatalf("ImportOriginalSession: %v", err)
	}
	if got := w.ScriptRegister(9); got != -777 {
		t.Fatalf("check-owned register 9 = %d immediately after import, want the file's -777 (untouched until the check runs)", got)
	}
	// scriptPass runs once per scriptCycle (16) sub-ticks, at phase
	// scriptPassPhase; one full cycle guarantees the check has run once.
	for i := 0; i < scriptCycle; i++ {
		Step(w, nil)
	}
	if got := w.ScriptRegister(9); got != 1 {
		t.Fatalf("check-owned register 9 = %d after one full tick, want 1 (the check's own recomputed answer, not the file's -777)", got)
	}
}

// TestValidateOriginalSessionAcceptsAnyRegisterOrRawContent proves Registers,
// RawHead and RawMid carry no value check: every array is fixed-width, so
// there is no length to police, and the original file can legitimately hold
// any int32 or byte pattern in any of the three.
func TestValidateOriginalSessionAcceptsAnyRegisterOrRawContent(t *testing.T) {
	s := originalSessionFixture()
	for i := range s.Registers {
		s.Registers[i] = int32(i)*1000 - 12345
	}
	for i := range s.RawHead {
		s.RawHead[i] = byte(i)
	}
	for i := range s.RawMid {
		s.RawMid[i] = byte(i)
	}
	if err := ValidateOriginalSession(s); err != nil {
		t.Fatalf("ValidateOriginalSession rejected an arbitrary Registers/RawHead/RawMid: %v", err)
	}
}

func TestOriginalSessionImportIsTransactional(t *testing.T) {
	cases := []struct {
		name string
		edit func(*OriginalSession)
		want string
	}{
		{"short latches", func(s *OriginalSession) { s.TriggerLatches = s.TriggerLatches[:999] }, "999 trigger latch"},
		{"long latches", func(s *OriginalSession) { s.TriggerLatches = append(s.TriggerLatches, 0) }, "1001 trigger latch"},
		{"invalid latch", func(s *OriginalSession) { s.TriggerLatches[999] = 2 }, "latch 999 is 2"},
		{"short diplomacy", func(s *OriginalSession) { s.Diplomacy = s.Diplomacy[:2499] }, "2499 diplomacy"},
		{"long diplomacy", func(s *OriginalSession) { s.Diplomacy = append(s.Diplomacy, 0) }, "2501 diplomacy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := originalSessionWorld(t)
			valid := originalSessionFixture()
			valid.TriggerLatches[2] = 1
			valid.Diplomacy[1*relationSlots+2] = 3
			if err := w.ImportOriginalSession(valid); err != nil {
				t.Fatalf("seed import: %v", err)
			}
			before, err := w.MarshalBinary()
			if err != nil {
				t.Fatalf("marshal before: %v", err)
			}

			bad := originalSessionFixture()
			tc.edit(&bad)
			err = w.ImportOriginalSession(bad)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ImportOriginalSession error = %v, want text %q", err, tc.want)
			}
			after, marshalErr := w.MarshalBinary()
			if marshalErr != nil {
				t.Fatalf("marshal after: %v", marshalErr)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("a refused original session partially changed canonical state")
			}
		})
	}
}

func TestValidateOriginalSessionMatchesTheImportBoundary(t *testing.T) {
	valid := originalSessionFixture()
	if err := ValidateOriginalSession(valid); err != nil {
		t.Fatalf("valid session: %v", err)
	}
	valid.TriggerLatches[4] = 9
	if err := ValidateOriginalSession(valid); err == nil {
		t.Fatal("ValidateOriginalSession accepted a latch the byte form refuses")
	}
	var nilWorld *World
	if err := nilWorld.ImportOriginalSession(originalSessionFixture()); err == nil {
		t.Fatal("a nil world accepted an original session")
	}
}
