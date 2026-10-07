package sim

// The script trace, witnessed. Every world below is built in test code and no
// file is read: what is measured is the runtime's own account of itself.

import (
	"bytes"
	"testing"
)

// sttBounds and sttEnts are one small world: two entities, the second of which a
// test may fell to drive the loss-counting arm.
var sttBounds = Bounds{Width: 8, Height: 8}

func sttEnts() []Entity {
	return []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10},
		{ID: 2, X: 5, Y: 5, HP: 10, MaxHP: 10},
	}
}

// sttWorld builds a world running s, refusing rather than reporting a build
// error so that no test proceeds against a world it did not ask for.
func sttWorld(t *testing.T, s *Script, ents []Entity) *World {
	t.Helper()
	w, err := NewScriptedWorld(7, sttBounds, ModeCanonical, nil, ents, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w
}

// sttScript compiles, refusing on error for the same reason.
func sttScript(t *testing.T, checks []ScriptCheck, instants []ScriptInstant, trigs []ScriptTrigger) *Script {
	t.Helper()
	s, err := NewScript(checks, instants, trigs)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	return s
}

// sttPass advances to the next evaluation pass and returns the trace of it. It
// steps plainly up to the phase and observed on it, so what comes back is one
// pass and not an accumulation.
func sttPass(w *World) ScriptTrace {
	for w.tick%scriptCycle != scriptPassPhase {
		Step(w, nil)
	}
	return StepTraced(w, nil)
}

// TestObservingAWorldMovesNoByteOfIt is the whole basis on which this seam is
// allowed to exist. Two worlds are built identically; one is advanced observed
// at every tick and the other is not, and their bytes and digests are compared
// AT EVERY TICK rather than at the end — an equality only at the end would pass
// a divergence that healed.
//
// The run crosses sixteen full cycles, so several evaluation passes and several
// outcome reports are inside it, and the script it runs both fires a trigger and
// counts a loss so that the observed run has something to record.
func TestObservingAWorldMovesNoByteOfIt(t *testing.T) {
	build := func() *World {
		s := sttScript(t,
			[]ScriptCheck{
				{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{3}},
				{Op: ScriptCheckAlive, Register: 1, Unit: 1, HasUnit: true},
				{Op: ScriptCheckVIP, Register: 2, Unit: 2, HasUnit: true},
			},
			[]ScriptInstant{{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{5}}},
			[]ScriptTrigger{{
				Pairs:    [3]ScriptPair{{Left: 1, Right: 1, Cmp: ScriptCmpEQ, Used: true}},
				Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone},
				Latch:    0,
			}},
		)
		ents := sttEnts()
		ents[1].HP = -10 // the loss-counting arm has a finished body to count
		return sttWorld(t, s, ents)
	}
	plain, observed := build(), build()
	for i := 0; i < scriptCycle*16; i++ {
		Step(plain, nil)
		StepTraced(observed, nil)
		pb, err := plain.MarshalBinary()
		if err != nil {
			t.Fatalf("tick %d: MarshalBinary: %v", i, err)
		}
		ob, err := observed.MarshalBinary()
		if err != nil {
			t.Fatalf("tick %d: MarshalBinary: %v", i, err)
		}
		if !bytes.Equal(pb, ob) {
			t.Fatalf("tick %d: observing moved %d bytes of the world's form", i, len(pb))
		}
		if plain.Hash() != observed.Hash() {
			t.Fatalf("tick %d: observing moved the digest: %#x vs %#x", i, plain.Hash(), observed.Hash())
		}
	}
	if plain.Outcome() == OutcomeUndecided {
		t.Fatal("the run decided nothing, so it witnessed no report phase")
	}
}

// ---------------------------------------------------------------- AC-2

func TestATickWithNoScriptPhaseObservesNothing(t *testing.T) {
	w := sttWorld(t, sttScript(t, nil, nil, nil), sttEnts())
	// Tick 0 is neither the pass phase nor the report phase.
	tr := StepTraced(w, nil)
	if !tr.Empty() {
		t.Errorf("a tick outside both script phases reported %+v", tr)
	}
	if tr.Tick != 0 {
		t.Errorf("the trace records tick %d, want the tick it ran on", tr.Tick)
	}
}

// ---------------------------------------------------------------- AC-3

// TestAFiringNamesItsTriggerItsValuesAndItsInstants drives one trigger whose two
// used pairs hold and whose instant slots hold one arm this build runs and one
// it does not.
func TestAFiringNamesItsTriggerItsValuesAndItsInstants(t *testing.T) {
	const unsupportedInstant int32 = 0x7f
	s := sttScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{1}},
			{Op: ScriptCheckAlive, Register: 1, Unit: 1, HasUnit: true},
			{Op: ScriptCheckDistance, Register: 2, Unit: 1, HasUnit: true,
				Args: [scriptParams]int32{5, 5}},
			{Op: ScriptCheckConstant, Register: 3, Args: [scriptParams]int32{9}},
		},
		[]ScriptInstant{
			{Op: ScriptInstantLose},
			{Op: unsupportedInstant},
		},
		[]ScriptTrigger{{
			Pairs: [3]ScriptPair{
				{Left: 1, Right: 0, Cmp: ScriptCmpEQ, Used: true},
				{Left: 2, Right: 3, Cmp: ScriptCmpLT, Used: true},
			},
			Instants: [4]int32{0, 1, ScriptNone, ScriptNone},
			Once:     true,
			Latch:    12,
		}},
	)
	w := sttWorld(t, s, sttEnts())
	tr := sttPass(w)
	if len(tr.Firings) != 1 {
		t.Fatalf("the pass recorded %d firings, want 1: %+v", len(tr.Firings), tr.Firings)
	}
	f := tr.Firings[0]
	if f.Trigger != 0 || f.Latch != 12 || !f.Once {
		t.Errorf("firing = trigger %d latch %d once %v, want 0/12/true", f.Trigger, f.Latch, f.Once)
	}
	// Entity 1 is alive at (1,1) and the distance check measures Chebyshev 4 to
	// (5,5), so the recorded values are the ones the comparisons were made on.
	if f.Pairs[0].Left != 1 || f.Pairs[0].Right != 1 {
		t.Errorf("pair 0 recorded %d vs %d, want the alive check's 1 against the constant 1",
			f.Pairs[0].Left, f.Pairs[0].Right)
	}
	if f.Pairs[1].Left != 4 || f.Pairs[1].Right != 9 {
		t.Errorf("pair 1 recorded %d vs %d, want the measured distance 4 against the constant 9",
			f.Pairs[1].Left, f.Pairs[1].Right)
	}
	if f.Pairs[2].Pair.Used {
		t.Errorf("an unused pair slot was recorded as used")
	}
	if len(f.Instants) != 2 {
		t.Fatalf("the firing recorded %d instants, want the two slots it names", len(f.Instants))
	}
	if !f.Instants[0].Supported || f.Instants[0].Op != ScriptInstantLose {
		t.Errorf("slot 0 = %+v, want the implemented lose arm", f.Instants[0])
	}
	if f.Instants[1].Supported {
		t.Errorf("slot 1 names op %d, which this build does not run, yet was marked supported",
			f.Instants[1].Op)
	}
	if tr.Lost != 1 {
		t.Errorf("the lose counter reads %d after a firing that ran the lose arm", tr.Lost)
	}
}

// ---------------------------------------------------------------- AC-3a

func TestAPairIsRecordedAtTheValueItCompared(t *testing.T) {
	s := sttScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{0}},
			{Op: ScriptCheckConstant, Register: 5, Args: [scriptParams]int32{0}},
			{Op: ScriptCheckConstant, Register: 6, Args: [scriptParams]int32{1}},
		},
		[]ScriptInstant{{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{5, 1}}},
		[]ScriptTrigger{
			{
				Pairs:    [3]ScriptPair{{Left: 5, Right: 0, Cmp: ScriptCmpEQ, Used: true}},
				Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone},
				Latch:    0,
			},
			{
				Pairs:    [3]ScriptPair{{Left: 5, Right: 6, Cmp: ScriptCmpEQ, Used: true}},
				Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone},
				Latch:    1,
			},
		},
	)
	w := sttWorld(t, s, sttEnts())
	tr := sttPass(w)
	if len(tr.Firings) != 2 {
		t.Fatalf("%d firings, want both triggers: %+v", len(tr.Firings), tr.Firings)
	}
	if got := tr.Firings[0].Pairs[0].Left; got != 0 {
		t.Errorf("the first trigger read register 5 as %d; it held 0 when the condition was "+
			"tested and reached 1 only through that trigger's own instant", got)
	}
	if got := tr.Firings[1].Pairs[0].Left; got != 1 {
		t.Errorf("the second trigger read register 5 as %d; the first trigger's instant had "+
			"already set it to 1", got)
	}
	if got := w.ScriptRegister(5); got != 1 {
		t.Fatalf("register 5 ends the pass at %d; this test measures nothing unless it is 1", got)
	}
}

// ---------------------------------------------------------------- AC-4

// TestALossCountedByACheckBelongsToNoTrigger is the shape the tenth mission is
// lost in, and the one a firing-only record cannot see.
func TestALossCountedByACheckBelongsToNoTrigger(t *testing.T) {
	s := sttScript(t,
		[]ScriptCheck{{Op: ScriptCheckVIP, Register: 4, Unit: 2, HasUnit: true}},
		nil, nil)
	ents := sttEnts()
	ents[1].HP = -10
	w := sttWorld(t, s, ents)
	tr := sttPass(w)
	if len(tr.Firings) != 0 {
		t.Errorf("a script with no trigger recorded %d firings", len(tr.Firings))
	}
	if len(tr.VIP) != 1 || tr.VIP[0].Check != 0 || tr.VIP[0].Unit != 2 {
		t.Fatalf("the self-counted loss reads %+v, want check 0 on entity 2", tr.VIP)
	}
	if tr.Lost != 1 {
		t.Errorf("the lose counter reads %d, want the one this check counted", tr.Lost)
	}
	if len(tr.Silent) != 0 {
		t.Errorf("a loss-counting check that found its unit was recorded as silent: %+v", tr.Silent)
	}
}

// ---------------------------------------------------------------- AC-5

// TestLegacySilentProjectionKeepsFailuresAndOmitsDesignSilences pins the old
// projection's distinction: these three failures are reported, while a
// constant and a living VIP are visible only in the complete check records.
func TestLegacySilentProjectionKeepsFailuresAndOmitsDesignSilences(t *testing.T) {
	const unsupportedCheck int32 = 0x7e
	s := sttScript(t,
		[]ScriptCheck{
			{Op: unsupportedCheck, Register: 0},                                  // 0: unsupported
			{Op: ScriptCheckAlive, Register: 1, Unit: 99, HasUnit: true},         // 1: no such entity
			{Op: ScriptCheckGroupCount, Register: 2},                             // 2: names no group
			{Op: ScriptCheckConstant, Register: 3, Args: [scriptParams]int32{1}}, // by design
			{Op: ScriptCheckVIP, Register: 4, Unit: 1, HasUnit: true},            // by design, alive
			{Op: ScriptCheckWithin, Register: 5, Unit: 1, HasUnit: true},         // writes a register
		},
		nil, nil)
	w := sttWorld(t, s, sttEnts())
	tr := sttPass(w)
	want := map[int32]ScriptSilence{
		0: ScriptSilenceUnsupported,
		1: ScriptSilenceNoUnit,
		2: ScriptSilenceNoGroup,
	}
	if len(tr.Silent) != len(want) {
		t.Fatalf("%d silences recorded, want %d: %+v", len(tr.Silent), len(want), tr.Silent)
	}
	for _, s := range tr.Silent {
		why, ok := want[s.Check]
		if !ok {
			t.Errorf("check %d was recorded silent and writes no register by design", s.Check)
			continue
		}
		if s.Why != why {
			t.Errorf("check %d recorded reason %d, want %d", s.Check, s.Why, why)
		}
		if s.Register != s.Check {
			t.Errorf("check %d recorded register %d, want its own", s.Check, s.Register)
		}
	}
}

// ---------------------------------------------------------------- AC-6

func TestARegisterNamesTheCheckThatWritesIt(t *testing.T) {
	s := sttScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckAlive, Register: 3, Unit: 1, HasUnit: true},
			{Op: ScriptCheckDistance, Register: 8, Unit: 1, HasUnit: true},
		},
		nil, nil)
	c, i, ok := s.RegisterOwner(8)
	if !ok || i != 1 || c.Op != ScriptCheckDistance {
		t.Errorf("RegisterOwner(8) = %+v, %d, %v; want the second check", c, i, ok)
	}
	if _, i, ok := s.RegisterOwner(9); ok || i != ScriptNone {
		t.Errorf("RegisterOwner(9) claimed an owner; register 9 is an authored variable")
	}
	if _, _, ok := (*Script)(nil).RegisterOwner(0); ok {
		t.Errorf("a nil script claimed to own a register")
	}
}

// ---------------------------------------------------------------- 1041 B2

func TestEveryCheckDispositionIsRecordedAtItsOwnBoundary(t *testing.T) {
	const unsupportedCheck int32 = 0x7e
	s := sttScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{1}},
			{Op: ScriptCheckAlive, Register: 1, Unit: 1, HasUnit: true},
			{Op: ScriptCheckVIP, Register: 2, Unit: 1, HasUnit: true},
			{Op: ScriptCheckDead11, Register: 3},
			{Op: ScriptCheckHealth, Register: 4, Unit: 1, HasUnit: true},
			{Op: unsupportedCheck, Register: 5},
			{Op: ScriptCheckGroupCount, Register: 6},
		}, nil, nil)
	w := sttWorld(t, s, sttEnts())
	first := sttPass(w)
	if len(first.Checks) != len(s.Checks()) {
		t.Fatalf("the pass recorded %d checks, want every one of %d", len(first.Checks), len(s.Checks()))
	}
	want := []struct {
		dispatched bool
		wrote      bool
		reason     ScriptSilence
	}{
		{false, false, ScriptSilenceBuildTimeConstant},
		{true, true, ScriptSilenceNone},
		{true, false, ScriptSilenceVIPNoValue},
		{true, false, ScriptSilenceDeadArm},
		{true, false, ScriptSilenceHealthGate},
		{false, false, ScriptSilenceUnsupported},
		{true, false, ScriptSilenceNoGroup},
	}
	for i, w := range want {
		got := first.Checks[i]
		if got.Check != int32(i) || got.Dispatched != w.dispatched || got.Wrote != w.wrote {
			t.Errorf("check %d disposition = %+v, want dispatched=%v wrote=%v", i, got, w.dispatched, w.wrote)
		}
		if got.Wrote {
			if got.HasSilence || got.Silence != ScriptSilenceNone {
				t.Errorf("writing check %d also claims silence %+v", i, got)
			}
		} else if !got.HasSilence || got.Silence != w.reason {
			t.Errorf("check %d silence = (%v,%d), want (true,%d)", i, got.HasSilence, got.Silence, w.reason)
		}
	}
	if got := first.Checks[0].Value; got != 1 {
		t.Errorf("constant's builder-owned register reads %d, want its preset 1", got)
	}
	if first.Checks[2].LostBefore != first.Checks[2].LostAfter {
		t.Errorf("the living VIP moved the loss counter: %+v", first.Checks[2])
	}

	// The alive arm writes 1 again on the second pass. Equality of before and
	// after cannot erase the write that occurred at the dispatch site.
	second := sttPass(w)
	got := second.Checks[1]
	if !got.Wrote || got.Before != 1 || got.Value != 1 {
		t.Errorf("same-value write recorded as %+v, want wrote=true and 1 -> 1", got)
	}
}

func TestEveryTriggerDecisionAndShortCircuitIsRecorded(t *testing.T) {
	const unsupportedCheck int32 = 0x7e
	s := sttScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{1}},
			{Op: ScriptCheckConstant, Register: 1, Args: [scriptParams]int32{2}},
			{Op: unsupportedCheck, Register: 2},
		}, nil,
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{{Left: 2, Right: 0, Cmp: ScriptCmpEQ, Used: true}}, Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone}, Latch: 0},
			{Pairs: [3]ScriptPair{{Left: 0, Right: 0, Cmp: ScriptCmpEQ, Used: true}}, Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone}, Once: true, Latch: 1},
			{Pairs: [3]ScriptPair{
				{Left: 0, Right: 0, Cmp: ScriptCmpEQ, Used: true},
				{Left: 0, Right: 1, Cmp: ScriptCmpEQ, Used: true},
				{Left: 1, Right: 1, Cmp: ScriptCmpEQ, Used: true},
			}, Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone}, Latch: 2},
			{Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone}, Latch: 3},
		})
	w := sttWorld(t, s, sttEnts())
	first := sttPass(w)
	if len(first.Triggers) != 4 {
		t.Fatalf("the pass recorded %d trigger decisions, want 4", len(first.Triggers))
	}
	if first.Triggers[0].Decision != ScriptTriggerInert || len(first.Triggers[0].Pairs) != 0 {
		t.Errorf("inert trigger = %+v", first.Triggers[0])
	}
	if first.Triggers[1].Decision != ScriptTriggerFired || first.Triggers[1].LatchBefore != 0 || first.Triggers[1].LatchAfter != 1 {
		t.Errorf("one-shot's first decision = %+v", first.Triggers[1])
	}
	failed := first.Triggers[2]
	if failed.Decision != ScriptTriggerFailed || failed.FirstFailed != 1 || len(failed.Pairs) != 2 {
		t.Fatalf("short-circuited trigger = %+v", failed)
	}
	if !failed.Pairs[0].Result || failed.Pairs[1].Result {
		t.Errorf("compared results = %+v, want true then false", failed.Pairs)
	}
	if first.Triggers[3].Decision != ScriptTriggerFired || len(first.Triggers[3].Pairs) != 0 {
		t.Errorf("always trigger = %+v", first.Triggers[3])
	}
	second := sttPass(w)
	if got := second.Triggers[1]; got.Decision != ScriptTriggerSpent || got.LatchBefore != 1 || got.LatchAfter != 1 || len(got.Pairs) != 0 {
		t.Errorf("spent one-shot = %+v", got)
	}
	if got := second.Triggers[3]; got.Decision != ScriptTriggerHeld || got.LatchBefore != 1 || got.LatchAfter != 1 {
		t.Errorf("held repeating trigger = %+v", got)
	}
}

func TestInstantDispatchRecordsSubcommandAndStateOutcome(t *testing.T) {
	const unsupportedInstant int32 = 0x7f
	s := sttScript(t,
		[]ScriptCheck{{Op: ScriptCheckConstant, Register: 5, Args: [scriptParams]int32{1}}},
		[]ScriptInstant{
			{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{5, 1}},
			{Op: ScriptInstantIncVariable, Args: [scriptParams]int32{5}},
			{Op: unsupportedInstant},
			{Op: ScriptInstantGroupOrder, Args: [scriptParams]int32{subCommandPatrol}},
		},
		[]ScriptTrigger{{
			Instants: [4]int32{0, 1, 2, 3}, Once: true, Latch: 0,
		}})
	w := sttWorld(t, s, sttEnts())
	tr := sttPass(w)
	if len(tr.Firings) != 1 || len(tr.Firings[0].Instants) != 4 {
		t.Fatalf("instant trace = %+v, want one firing with four slots", tr.Firings)
	}
	runs := tr.Firings[0].Instants
	if runs[0].Outcome != ScriptInstantNoStateChange || runs[0].BeforeHash != runs[0].AfterHash {
		t.Errorf("same-value setter = %+v", runs[0])
	}
	if runs[1].Outcome != ScriptInstantStateChanged || runs[1].BeforeHash == runs[1].AfterHash {
		t.Errorf("increment = %+v", runs[1])
	}
	if runs[2].Supported || runs[2].Outcome != ScriptInstantUnsupported || runs[2].BeforeHash != runs[2].AfterHash {
		t.Errorf("unsupported arm = %+v", runs[2])
	}
	if !runs[3].Supported || !runs[3].HasSubCommand || runs[3].SubCommand != subCommandPatrol || runs[3].Outcome != ScriptInstantNoStateChange {
		t.Errorf("group sub-dispatch = %+v", runs[3])
	}
}
