package sim

import "testing"

// Check opcodes 8 and 15: how many living units a player has, and how near
// the nearest of them is.

// populationCheck is one check of arm 8, naming a player.
func populationCheck(reg int32, player uint32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckPopulation, Register: reg, Player: player, HasPlayer: true}
}

// nearestCheck is one check of arm 15, naming a player and the authored cell.
func nearestCheck(reg int32, player uint32, x, y int32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckNearest, Register: reg, Player: player, HasPlayer: true,
		Args: [scriptParams]int32{x, y}}
}

// killOwned puts one entity below zero health, dead by every reading. It
// writes the field rather than resolving a blow, because what is under test is
// the count and the distance, not the fight — the same fixture shape
// scriptgroup_test.go uses for the group count's own arm.
func killOwned(t *testing.T, w *World, id EntityID) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("no entity %d", id)
	}
	w.entities[i].HP = -1
}

// downOwned puts one entity at exactly zero against a positive maximum: this
// tree's third state, which falls on the dead side of every script arm.
func downOwned(t *testing.T, w *World, id EntityID) {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("no entity %d", id)
	}
	w.entities[i].HP = 0
}

// ---------------------------------------------------------------- AC-1, arm 8

func TestThePopulationCountCountsAPlayersLivingEntities(t *testing.T) {
	t.Parallel()

	// Three of player 1, two of player 2, and one of player 0 — a real roster
	// slot and not "no player", so a build that mistook it for absence would
	// be caught by a count and not by a silence.
	ents := []Entity{
		{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10, Owner: 1},
		{ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10, Owner: 1},
		{ID: 3, X: 3, Y: 1, HP: 10, MaxHP: 10, Owner: 1},
		{ID: 4, X: 4, Y: 1, HP: 10, MaxHP: 10, Owner: 2},
		{ID: 5, X: 5, Y: 1, HP: 10, MaxHP: 10, Owner: 2},
		{ID: 6, X: 6, Y: 1, HP: 10, MaxHP: 10, Owner: 0},
	}
	s := mustScript(t, []ScriptCheck{
		populationCheck(0, 1),
		populationCheck(1, 2),
		populationCheck(2, 0),
		populationCheck(3, 99),
	}, nil, nil)
	w := scriptWorld(t, s, ents)

	read := func(what string, want ...int32) {
		t.Helper()
		scriptTicks(w, 16, nil)
		for r, v := range want {
			if got := w.ScriptRegister(int32(r)); got != v {
				t.Errorf("%s: register %d is %d, want %d", what, r, got, v)
			}
		}
	}
	read("all alive", 3, 2, 1, 0)

	killOwned(t, w, 1)
	killOwned(t, w, 2)
	read("two of player 1's three felled", 1, 2, 1, 0)
	killOwned(t, w, 3)
	read("player 1 wiped, player 2 untouched", 0, 2, 1, 0)

	// A DOWNED entity — health at exactly zero against a positive maximum — is
	// out of the fight and out of the count, the same sense of dead every
	// other arm of this runtime uses.
	downOwned(t, w, 4)
	read("one of player 2's two downed", 0, 1, 1, 0)
}

// TestAPopulationCheckNamingNoPlayerWritesNoRegister is AC-1's other half: a
// preset register no count could reach, left alone, and the trigger reading it
// evaluated and latched like any other — this arm is implemented, so nothing
// about it is inert.
func TestAPopulationCheckNamingNoPlayerWritesNoRegister(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{77}},
		{Op: ScriptCheckPopulation, Register: 1},
		populationCheck(2, 5),
	}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{pair(1, 1, ScriptCmpEQ)},
		Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone},
		Latch:    0}})
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("this script holds only implemented arms; report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := scriptWorld(t, s, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 9, MaxHP: 9, Owner: 5},
		{ID: 2, X: 2, Y: 1, HP: 9, MaxHP: 9, Owner: 5},
	})
	w.registers[1] = 77
	scriptTicks(w, 16, nil)

	if got := w.ScriptRegister(1); got != 77 {
		t.Errorf("a check naming no player wrote %d into its register, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(2); got != 2 {
		t.Errorf("the check beside it measured %d, want 2 — the arm still works", got)
	}
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading that register was not evaluated; this arm is implemented, " +
			"so nothing about it is inert")
	}
}

// TestAPopulationCheckIsSilencedByDesign is AC-1's silence: the trace names the
// missing-player case, and it is the only one this arm can reach.
func TestAPopulationCheckIsSilencedByDesign(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{{Op: ScriptCheckPopulation, Register: 0}}, nil, nil)
	w, err := NewScriptedWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil,
		[]Entity{{ID: 1, HP: 5, MaxHP: 5, Owner: 1}}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	for w.tick%scriptCycle != scriptPassPhase {
		Step(w, nil)
	}
	tr := StepTraced(w, nil)
	if len(tr.Silent) != 1 || tr.Silent[0].Check != 0 || tr.Silent[0].Why != ScriptSilenceNoPlayer {
		t.Fatalf("the silence is %+v, want one entry naming check 0 and ScriptSilenceNoPlayer", tr.Silent)
	}
}

// ---------------------------------------------------------------- AC-2, arm 15

// TestTheNearestDistanceIsTheMinimumOverAPlayersLivingEntities is AC-2: the
// minimum moves as the nearer entities are felled, the dead are ignored
// outright, and a player with no living entity answers 0xff.
func TestTheNearestDistanceIsTheMinimumOverAPlayersLivingEntities(t *testing.T) {
	t.Parallel()

	// Player 1 has three entities at Chebyshev distances 2, 5 and 9 from
	// (10,10); player 2 has one, closer than any of player 1's, so the two
	// registers must never agree by accident.
	ents := []Entity{
		{ID: 1, X: 12, Y: 10, HP: 10, MaxHP: 10, Owner: 1}, // distance 2
		{ID: 2, X: 15, Y: 10, HP: 10, MaxHP: 10, Owner: 1}, // distance 5
		{ID: 3, X: 19, Y: 10, HP: 10, MaxHP: 10, Owner: 1}, // distance 9
		{ID: 4, X: 11, Y: 10, HP: 10, MaxHP: 10, Owner: 2}, // distance 1
	}
	s := mustScript(t, []ScriptCheck{
		nearestCheck(0, 1, 10, 10),
		nearestCheck(1, 2, 10, 10),
	}, nil, nil)
	w := scriptWorld(t, s, ents)

	read := func(what string, want0, want1 int32) {
		t.Helper()
		scriptTicks(w, 16, nil)
		if got := w.ScriptRegister(0); got != want0 {
			t.Errorf("%s: register 0 is %d, want %d", what, got, want0)
		}
		if got := w.ScriptRegister(1); got != want1 {
			t.Errorf("%s: register 1 is %d, want %d", what, got, want1)
		}
	}
	read("all alive", 2, 1)

	// Killing the nearest of player 1's three moves the minimum to the next.
	killOwned(t, w, 1)
	read("the nearest felled", 5, 1)
	killOwned(t, w, 2)
	read("the two nearest felled", 9, 1)

	// A DOWNED entity is excluded exactly as a dead one is: player 1's last
	// entity down leaves nothing living to measure, and the seed itself is
	// the answer.
	downOwned(t, w, 3)
	read("player 1's last entity downed: the seed itself", 0xff, 1)
}

// TestANearestCheckNamingNoPlayerWritesNoRegister is AC-2's other half over the
// nearest-distance arm, mirroring the population check's own case.
func TestANearestCheckNamingNoPlayerWritesNoRegister(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{77}},
		{Op: ScriptCheckNearest, Register: 1, Args: [scriptParams]int32{10, 10}},
		nearestCheck(2, 5, 10, 10),
	}, nil, []ScriptTrigger{{
		Pairs:    [3]ScriptPair{pair(1, 1, ScriptCmpEQ)},
		Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone},
		Latch:    0}})
	if len(s.Unsupported()) != 0 || len(s.InertTriggers()) != 0 {
		t.Fatalf("this script holds only implemented arms; report %+v, inert %v",
			s.Unsupported(), s.InertTriggers())
	}

	w := scriptWorld(t, s, []Entity{{ID: 1, X: 11, Y: 10, HP: 9, MaxHP: 9, Owner: 5}})
	w.registers[1] = 77
	scriptTicks(w, 16, nil)

	if got := w.ScriptRegister(1); got != 77 {
		t.Errorf("a check naming no player wrote %d into its register, want the preset 77 left alone", got)
	}
	if got := w.ScriptRegister(2); got != 1 {
		t.Errorf("the check beside it measured %d, want 1 — the arm still works", got)
	}
	if !w.ScriptLatched(0) {
		t.Errorf("the trigger reading that register was not evaluated; this arm is implemented, " +
			"so nothing about it is inert")
	}
}

func TestTheNearestDistanceSeedAndTheAbsentPlayerAreDistinguished(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckConstant, Register: 0, Args: [scriptParams]int32{77}}, // an unreachable marker
		nearestCheck(1, 9, 10, 10), // player 9 owns nothing: 0xff
		{Op: ScriptCheckNearest, Register: 2, Args: [scriptParams]int32{10, 10}}, // no player: nothing
	}, nil, nil)
	w, err := NewScriptedWorld(1, Bounds{Width: 40, Height: 40}, ModeCanonical, nil,
		[]Entity{{ID: 1, HP: 5, MaxHP: 5, Owner: 1}}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	w.registers[2] = 77
	for w.tick%scriptCycle != scriptPassPhase {
		Step(w, nil)
	}
	tr := StepTraced(w, nil)

	if got := w.ScriptRegister(1); got != 0xff {
		t.Errorf("register 1 (player 9, owns nothing) is %d, want the seed 0xff", got)
	}
	if got := w.ScriptRegister(2); got != 77 {
		t.Errorf("register 2 (no player named) is %d, want the preset 77 left alone — "+
			"writing nothing is not writing 0xff", got)
	}
	if len(tr.Silent) != 1 || tr.Silent[0].Check != 2 || tr.Silent[0].Why != ScriptSilenceNoPlayer {
		t.Fatalf("the silence is %+v, want exactly one entry naming check 2 and ScriptSilenceNoPlayer — "+
			"the seeded 0xff at check 1 is a measurement, not a silence", tr.Silent)
	}
}

func TestTheNearestDistancePassesThroughTheSharedByteMask(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{nearestCheck(0, 1, 0, 0)}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1, X: 10 + 300, Y: 10, HP: 5, MaxHP: 5, Owner: 1}})
	scriptTicks(w, 7, nil)
	if got, want := w.ScriptRegister(0), int32((10+300)&0xff); got != want {
		t.Errorf("register 0 is %d, want %d — the byte mask two sibling arms already apply", got, want)
	}
}

// ---------------------------------------------------------------- both arms

// TestBothArmsAreInSupportedAndNoLongerInertTheirReaders is SC-1: the compile
// report names both opcodes among the implemented ones, and a trigger reading
// either register is live.
func TestBothArmsAreInSupportedAndNoLongerInertTheirReaders(t *testing.T) {
	t.Parallel()

	if !scriptCheckSupported(ScriptCheckPopulation) {
		t.Error("ScriptCheckPopulation (8) is not in scriptCheckSupported")
	}
	if !scriptCheckSupported(ScriptCheckNearest) {
		t.Error("ScriptCheckNearest (15) is not in scriptCheckSupported")
	}

	s := mustScript(t, []ScriptCheck{
		populationCheck(0, 1),
		{Op: ScriptCheckConstant, Register: 1},
		nearestCheck(2, 1, 0, 0),
	}, nil, []ScriptTrigger{
		{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)}, Instants: acts(), Latch: 0},
		{Pairs: [3]ScriptPair{pair(2, 1, ScriptCmpGT)}, Instants: acts(), Latch: 1},
	})
	if gaps := s.Unsupported(); len(gaps) != 0 {
		t.Fatalf("Unsupported() is %+v, want none — both arms are implemented", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 0 {
		t.Fatalf("InertTriggers() is %v, want none", inert)
	}

	w := scriptWorld(t, s, []Entity{{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5, Owner: 1}})
	scriptTicks(w, 16, nil)
	if !w.ScriptLatched(1) {
		t.Errorf("the trigger reading the nearest-distance register did not evaluate")
	}
}
