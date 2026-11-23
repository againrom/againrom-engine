package sim

import (
	"slices"
	"testing"
)

const secondGameRegisters = 1000

func TestSecondGameSessionStartRegistersAndTick999(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckVariable, Register: 0, Args: [10]int32{999}},
		{Op: ScriptCheckVariable, Register: 1, Args: [10]int32{901}},
		{Op: ScriptCheckVariable, Register: 2, Args: [10]int32{900}},
		{Op: ScriptCheckVariable, Register: 3, Args: [10]int32{902}},
	}, nil, nil)
	w := scriptWorld(t, s, nil)
	if w.ScriptRegister(900) != 0 || w.ScriptRegister(901) != 1 || w.ScriptRegister(902) != 0 || w.ScriptRegister(999) != 0 {
		t.Fatalf("session start registers: 900=%d 901=%d 902=%d 999=%d, want 0 1 0 0",
			w.ScriptRegister(900), w.ScriptRegister(901), w.ScriptRegister(902), w.ScriptRegister(999))
	}
	beyond := w.ScriptRegistersBeyond()
	if len(beyond) != secondGameRegisters-scriptRegisters || beyond[901-scriptRegisters] != 1 {
		t.Fatalf("ScriptRegistersBeyond len=%d slot901=%d", len(beyond), beyond[901-scriptRegisters])
	}
	for step := int32(1); step <= 3; step++ {
		w.scriptPass(nil)
		if w.ScriptRegister(999) != step || w.ScriptRegister(0) != step {
			t.Fatalf("pass %d: reg999=%d reg0=%d, want %d", step, w.ScriptRegister(999), w.ScriptRegister(0), step)
		}
		if w.ScriptRegister(93) != 0 {
			t.Fatalf("pass %d: second game incremented slot 93 to %d", step, w.ScriptRegister(93))
		}
		if w.ScriptRegister(1) != 1 || w.ScriptRegister(2) != 0 || w.ScriptRegister(3) != 0 {
			t.Fatalf("pass %d: reg1=%d reg2=%d reg3=%d", step, w.ScriptRegister(1), w.ScriptRegister(2), w.ScriptRegister(3))
		}
	}
	w.setRegisterAt(999, 2147483647)
	w.scriptPass(nil)
	if got := w.ScriptRegister(999); got != -2147483648 {
		t.Fatalf("wrapped reg999=%d, want -2147483648", got)
	}
	controlled, err := NewControlledScriptWorld(w, s)
	if err != nil {
		t.Fatal(err)
	}
	if controlled.ScriptRegister(901) != 1 || controlled.ScriptRegister(999) != 0 {
		t.Fatalf("controlled start: 901=%d 999=%d, want 1 0", controlled.ScriptRegister(901), controlled.ScriptRegister(999))
	}
}

func TestSecondGameWideRegistersReadWriteAndRoundTrip(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckVariable, Register: 150, Args: [10]int32{500}},
		{Op: ScriptCheckConstant, Register: 950, Args: [10]int32{42}},
	}, []ScriptInstant{
		{Op: ScriptInstantSetVariable, Args: [10]int32{777, -19}},
		{Op: ScriptInstantIncVariable, Args: [10]int32{888}},
	}, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{{Left: 150, Right: 950, Cmp: ScriptCmpEQ, Used: true}},
		Instants: acts(0, 1),
		Once:     true,
		Latch:    2,
	}})
	w := scriptWorld(t, s, nil)
	if got := w.ScriptRegister(950); got != 42 {
		t.Fatalf("constant reg950=%d, want 42", got)
	}
	w.runInstant(ScriptInstant{Op: ScriptInstantSetVariable, Args: [10]int32{500, 42}})
	w.runInstant(ScriptInstant{Op: ScriptInstantSetVariable, Args: [10]int32{901, 0}})
	w.runInstant(ScriptInstant{Op: ScriptInstantSetVariable, Args: [10]int32{1000, 99}})
	w.runInstant(ScriptInstant{Op: ScriptInstantSetVariable, Args: [10]int32{-1, 99}})
	if w.ScriptRegister(1000) != 0 || w.ScriptRegister(-1) != 0 {
		t.Fatalf("out-of-range registers were written: 1000=%d -1=%d", w.ScriptRegister(1000), w.ScriptRegister(-1))
	}
	w.scriptPass(nil)
	if w.ScriptRegister(150) != 42 || w.ScriptRegister(777) != -19 || w.ScriptRegister(888) != 1 || w.ScriptRegister(901) != 0 || w.ScriptRegister(999) != 1 {
		t.Fatalf("after pass: 150=%d 777=%d 888=%d 901=%d 999=%d",
			w.ScriptRegister(150), w.ScriptRegister(777), w.ScriptRegister(888), w.ScriptRegister(901), w.ScriptRegister(999))
	}
	if ext := w.ScriptExtendedState(); len(ext.Variables) != 0 || len(ext.Results) != 0 || len(ext.Latches) != 0 {
		t.Fatalf("second game exposed wide registers as extended state: %+v", ext)
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("round-trip hash %#x != %#x", back.Hash(), w.Hash())
	}
	for _, want := range [][2]int32{{150, 42}, {500, 42}, {777, -19}, {888, 1}, {901, 0}, {950, 42}, {999, 1}} {
		if got := back.ScriptRegister(want[0]); got != want[1] {
			t.Errorf("round-trip reg %d = %d, want %d", want[0], got, want[1])
		}
	}

	wide := make([]int32, secondGameRegisters-scriptRegisters)
	wide[900-scriptRegisters] = 5
	wide[901-scriptRegisters] = 1
	wide[902-scriptRegisters] = 7
	wide[999-scriptRegisters] = 141
	sess := OriginalSession{
		WideRegisters:  wide,
		TriggerLatches: make([]byte, scriptLatches),
		Diplomacy:      make([]byte, relationLen),
		Outcome:        OutcomeUndecided,
	}
	if err := back.ImportOriginalSession(sess); err != nil {
		t.Fatal(err)
	}
	if back.ScriptRegister(900) != 5 || back.ScriptRegister(901) != 1 || back.ScriptRegister(902) != 7 || back.ScriptRegister(950) != 42 || back.ScriptRegister(999) != 141 {
		t.Fatalf("imported wide registers: 900=%d 901=%d 902=%d 950=%d 999=%d",
			back.ScriptRegister(900), back.ScriptRegister(901), back.ScriptRegister(902), back.ScriptRegister(950), back.ScriptRegister(999))
	}
	sess.WideRegisters = []int32{1, 2, 3}
	if err := back.ImportOriginalSession(sess); err == nil {
		t.Fatal("ImportOriginalSession accepted malformed WideRegisters length")
	}
}

func TestFirstGameKeepsHundredRegistersAndSlot93Tick(t *testing.T) {
	if _, err := NewScript([]ScriptCheck{{Op: ScriptCheckVariable, Register: 100}}, nil, nil); err == nil {
		t.Fatal("ROM1 NewScript accepted check owning register 100")
	}
	if _, err := NewScript(nil, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{{Left: 999, Right: 0, Cmp: ScriptCmpEQ, Used: true}},
		Instants: acts(),
	}}); err == nil {
		t.Fatal("ROM1 NewScript accepted trigger reading register 999")
	}
	s, err := NewScript([]ScriptCheck{
		{Op: ScriptCheckVariable, Register: 0, Args: [10]int32{93}},
	}, []ScriptInstant{
		{Op: ScriptInstantSetVariable, Args: [10]int32{100, 55}},
		{Op: ScriptInstantIncVariable, Args: [10]int32{902}},
	}, []ScriptTrigger{{Instants: acts(0, 1)}})
	if err != nil {
		t.Fatal(err)
	}
	w := scriptWorld(t, s, nil)
	if beyond := w.ScriptRegistersBeyond(); beyond != nil {
		t.Fatalf("ROM1 ScriptRegistersBeyond = %v, want nil", beyond)
	}
	if w.ScriptRegister(901) != 0 || w.ScriptRegister(999) != 0 {
		t.Fatalf("ROM1 seeded wide registers: 901=%d 999=%d", w.ScriptRegister(901), w.ScriptRegister(999))
	}
	w.scriptPass(nil)
	if w.ScriptRegister(93) != 1 || w.ScriptRegister(0) != 1 || w.ScriptRegister(100) != 0 || w.ScriptRegister(902) != 0 || w.ScriptRegister(999) != 0 {
		t.Fatalf("ROM1 after pass: 93=%d 0=%d 100=%d 902=%d 999=%d",
			w.ScriptRegister(93), w.ScriptRegister(0), w.ScriptRegister(100), w.ScriptRegister(902), w.ScriptRegister(999))
	}
	wide := make([]int32, secondGameRegisters-scriptRegisters)
	if err := w.ImportOriginalSession(OriginalSession{
		WideRegisters:  wide,
		TriggerLatches: make([]byte, scriptLatches),
		Diplomacy:      make([]byte, relationLen),
		Outcome:        OutcomeUndecided,
	}); err == nil {
		t.Fatal("ROM1 world accepted WideRegisters in ImportOriginalSession")
	}
}

func TestSecondGameCheck20CountsPlayerStructuresWithPositiveHealth(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckPlayerStructures, Register: 0, Player: 2, HasPlayer: true},
		{Op: ScriptCheckPlayerStructures, Register: 1, Player: 3, HasPlayer: true},
		{Op: ScriptCheckPlayerStructures, Register: 2, Player: 4, HasPlayer: true},
		{Op: ScriptCheckPlayerStructures, Register: 3},
	}, []ScriptInstant{
		{Op: ScriptInstantStructField, Structure: 1, HasStructure: true, Args: [10]int32{0}},
	}, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("ROM2 check 20 reported as gap: %+v", gaps)
	}
	w := scriptWorld(t, s, nil)
	w.DeclareStructures([]Structure{
		{ID: 1, Owner: 2, Field42: 100, MaxHealth: 100, Col: 1, Row: 1},
		{ID: 2, Owner: 2, Field42: 1, MaxHealth: 100, Col: 2, Row: 1},
		{ID: 3, Owner: 2, Field42: 0, MaxHealth: 100, Col: 3, Row: 1},
		{ID: 4, Owner: 2, Field42: 0xffff, MaxHealth: 100, Col: 4, Row: 1},
		{ID: 5, Owner: 3, Field42: 500, MaxHealth: 500, Col: 5, Row: 1},
	})
	w.setRegisterAt(3, 77)
	w.scriptPass(nil)
	if w.ScriptRegister(0) != 2 || w.ScriptRegister(1) != 1 || w.ScriptRegister(2) != 0 || w.ScriptRegister(3) != 77 {
		t.Fatalf("check 20 counts: p2=%d p3=%d p4=%d noPlayer=%d, want 2 1 0 77",
			w.ScriptRegister(0), w.ScriptRegister(1), w.ScriptRegister(2), w.ScriptRegister(3))
	}

	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	if back.Hash() != w.Hash() {
		t.Fatalf("structure owner round-trip hash %#x != %#x", back.Hash(), w.Hash())
	}
	back.runInstant(s.Instants()[0])
	back.scriptPass(nil)
	if got := back.ScriptRegister(0); got != 1 {
		t.Fatalf("after zeroing structure 1, check 20 = %d, want 1", got)
	}

	rom1, err := NewScript([]ScriptCheck{{Op: ScriptCheckPlayerStructures, Register: 0, Player: 2, HasPlayer: true}}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{{Left: 0, Right: 0, Cmp: ScriptCmpEQ, Used: true}},
		Instants: acts(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rom1.Unsupported(), []ScriptGap{{Kind: ScriptGapCheck, Op: ScriptCheckPlayerStructures, Index: 0}}) || !slices.Equal(rom1.InertTriggers(), []int32{0}) {
		t.Fatalf("ROM1 check 20 gaps=%+v inert=%v", rom1.Unsupported(), rom1.InertTriggers())
	}
}

func TestSecondGameCheck22CellObjectRelocatesAndReturnsOneOrZero(t *testing.T) {
	s := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckCellObject, Register: 0, Args: [10]int32{7, 7, 1, 2}},
		{Op: ScriptCheckCellObject, Register: 1, Args: [10]int32{4, 4, 1, 2}},
		{Op: ScriptCheckCellObject, Register: 2, Args: [10]int32{5, 5, 1, 2}},
		{Op: ScriptCheckCellObject, Register: 3, Args: [10]int32{1, 1, 1, 2}},
		// p1*256 + p0 = 2*256 + 258 = 3*256 + 2 = cell (2, 3); low8(0x105, 0x204) = (5, 4).
		{Op: ScriptCheckCellObject, Register: 4, Args: [10]int32{258, 2, 0x105, 0x204}},
	}, nil, nil)
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("ROM2 check 22 reported as gap: %+v", gaps)
	}
	grid := make([]byte, 8*8)
	grid[6*8+6] = blockGround
	w, err := NewTerrainWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, Terrain{Block: grid}, []Entity{
		{ID: 1, X: 2, Y: 3, HP: 10, MaxHP: 10, Domain: DomainGround},
		{ID: 2, X: 4, Y: 4, HP: 10, MaxHP: 10, Domain: DomainAir},
		{ID: 3, X: 5, Y: 5, HP: 0, MaxHP: 10, Domain: DomainGround},
		{ID: 4, X: 1, Y: 1, HP: 10, MaxHP: 10, Domain: DomainGround, OffMap: true},
	}, s)
	if err != nil {
		t.Fatal(err)
	}
	w.scriptPass(nil)
	if w.ScriptRegister(0) != 0 || w.ScriptRegister(1) != 0 || w.ScriptRegister(2) != 1 || w.ScriptRegister(3) != 0 || w.ScriptRegister(4) != 1 {
		t.Fatalf("check 22 results: empty=%d air=%d body=%d offmap=%d ground=%d, want 0 0 1 0 1",
			w.ScriptRegister(0), w.ScriptRegister(1), w.ScriptRegister(2), w.ScriptRegister(3), w.ScriptRegister(4))
	}
	u1, _ := w.Entity(1)
	u2, _ := w.Entity(2)
	if u1.X != 5 || u1.Y != 4 || u2.X != 4 || u2.Y != 4 {
		t.Fatalf("positions after check 22: u1=(%d,%d) u2=(%d,%d), want (5,4) (4,4)", u1.X, u1.Y, u2.X, u2.Y)
	}
	// Second pass: (2, 3) is now empty so register 4 becomes 0.
	w.scriptPass(nil)
	if got := w.ScriptRegister(4); got != 0 {
		t.Fatalf("after relocation, check 22 on vacated cell = %d, want 0", got)
	}
	// Occupied cell targeting a blocked cell (6, 6): still returns 1 and unit stays at (5, 4).
	blocked := secondGameScript(t, []ScriptCheck{
		{Op: ScriptCheckCellObject, Register: 0, Args: [10]int32{5, 4, 6, 6}},
	}, nil, nil)
	if err := w.RestoreScriptProgram(blocked); err != nil {
		t.Fatal(err)
	}
	w.scriptPass(nil)
	u1, _ = w.Entity(1)
	if w.ScriptRegister(0) != 1 || u1.X != 5 || u1.Y != 4 {
		t.Fatalf("blocked target: reg0=%d pos=(%d,%d), want 1 (5,4)", w.ScriptRegister(0), u1.X, u1.Y)
	}

	rom1, err := NewScript([]ScriptCheck{{Op: ScriptCheckCellObject, Register: 0, Args: [10]int32{2, 3, 5, 4}}}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{{Left: 0, Right: 0, Cmp: ScriptCmpEQ, Used: true}},
		Instants: acts(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rom1.Unsupported(), []ScriptGap{{Kind: ScriptGapCheck, Op: ScriptCheckCellObject, Index: 0}}) || !slices.Equal(rom1.InertTriggers(), []int32{0}) {
		t.Fatalf("ROM1 check 22 gaps=%+v inert=%v", rom1.Unsupported(), rom1.InertTriggers())
	}
}

func TestSecondGameAction1IncrementsRegister902AndGapsRemain(t *testing.T) {
	s := secondGameScript(t, nil, []ScriptInstant{{Op: ScriptInstantCounter}}, []ScriptTrigger{{Instants: acts(0)}})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("ROM2 action 1 reported as gap: %+v", gaps)
	}
	w := scriptWorld(t, s, nil)
	if got := w.ScriptRegister(902); got != 0 {
		t.Fatalf("initial reg902=%d, want 0", got)
	}
	var firings []ScriptFiring
	for n := 0; n < scriptPassPhase+1; n++ {
		firings = append(firings, StepTraced(w, nil).Firings...)
	}
	if got := w.ScriptRegister(902); got != 1 {
		t.Fatalf("after 1 pass reg902=%d, want 1", got)
	}
	if len(firings) != 1 || len(firings[0].Instants) != 1 || !firings[0].Instants[0].Supported || firings[0].Instants[0].Outcome != ScriptInstantStateChanged {
		t.Fatalf("unexpected trace for action 1: %+v", firings)
	}
	w.scriptPass(nil)
	if got := w.ScriptRegister(902); got != 2 {
		t.Fatalf("after 2 passes reg902=%d, want 2", got)
	}

	rom1, err := NewScript(nil, []ScriptInstant{{Op: ScriptInstantCounter}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(rom1.Unsupported(), []ScriptGap{{Kind: ScriptGapInstant, Op: ScriptInstantCounter, Index: 0}}) {
		t.Fatalf("ROM1 action 1 gaps=%+v", rom1.Unsupported())
	}

	// Actions 11, 27 and 31 remain unsupported gaps in ROM2.
	debt := secondGameScript(t, nil, []ScriptInstant{{Op: 11}, {Op: 27}, {Op: 31}}, nil)
	wantGaps := []ScriptGap{
		{Kind: ScriptGapInstant, Op: 11, Index: 0},
		{Kind: ScriptGapInstant, Op: 27, Index: 1},
		{Kind: ScriptGapInstant, Op: 31, Index: 2},
	}
	if !slices.Equal(debt.Unsupported(), wantGaps) {
		t.Fatalf("ROM2 open-debt action gaps=%+v, want %+v", debt.Unsupported(), wantGaps)
	}
}

func TestSecondGameEmptyProgramCountsRegister999(t *testing.T) {
	w := scriptWorld(t, secondGameScript(t, nil, nil, nil), nil)
	for step := int32(1); step <= 5; step++ {
		w.scriptPass(nil)
		if w.ScriptRegister(999) != step || w.ScriptRegister(901) != 1 || w.ScriptRegister(93) != 0 {
			t.Fatalf("pass %d: 999=%d 901=%d 93=%d", step, w.ScriptRegister(999), w.ScriptRegister(901), w.ScriptRegister(93))
		}
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	if back.ScriptRegister(999) != 5 {
		t.Fatalf("reloaded 999=%d, want 5", back.ScriptRegister(999))
	}
}
