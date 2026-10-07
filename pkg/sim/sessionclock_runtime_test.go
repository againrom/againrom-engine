package sim

import (
	"fmt"
	"testing"
)

func clockWorld1112(t *testing.T, sub, full uint32, ents []Entity) *World {
	t.Helper()
	w := scriptWorld(t, nil, ents)
	s := originalSessionFixture()
	s.HasClock, s.Clock = true, SessionClock{sub, full}
	if err := w.ImportOriginalSession(s); err != nil {
		t.Fatal(err)
	}
	return w
}

func assertClock1112(t *testing.T, w *World, sub, full uint32) {
	t.Helper()
	if got, present := w.SessionClock(); !present || got != (SessionClock{sub, full}) || w.Tick() != uint64(sub) {
		t.Fatalf("clock=%+v present=%t Tick=%d want %d/%d", got, present, w.Tick(), sub, full)
	}
}

// Literal cases exercise the public StepTraced wrapper, not the model helper.
func TestSessionClockRuntimePhasesAndWrap(t *testing.T) {
	for _, tc := range []struct {
		sub, full, nextS, nextF uint32
		pass, report            bool
	}{
		{5, 584, 6, 584, false, false}, {6, 584, 7, 584, true, false},
		{11, 584, 12, 584, false, false}, {12, 584, 13, 584, false, false},
		{14, 584, 15, 585, false, true}, {15, 584, 16, 584, false, false},
		{9343, 584, 9344, 584, false, false},
		{14, 0xffffffff, 15, 0, false, true},
		{0xffffffff, 584, 0, 584, false, false},
		{0x80000006, 584, 0x80000007, 584, false, false},
		{0x8000000c, 584, 0x8000000d, 584, false, false},
		{0x8000000e, 584, 0x8000000f, 584, false, false},
		{0xfffffff6, 584, 0xfffffff7, 584, false, false},
		{0xfffffffc, 584, 0xfffffffd, 584, false, false},
	} {
		t.Run(fmt.Sprintf("%x/%x", tc.sub, tc.full), func(t *testing.T) {
			w := clockWorld1112(t, tc.sub, tc.full, nil)
			tr := StepTraced(w, nil)
			assertClock1112(t, w, tc.nextS, tc.nextF)
			if tr.Tick != uint64(tc.sub) || tr.Pass != tc.pass || tr.Report != tc.report {
				t.Fatalf("trace=%+v want pass=%t report=%t", tr, tc.pass, tc.report)
			}
		})
	}
}

func TestSessionClockScriptReadsBeforeCommand(t *testing.T) {
	script := mustScript(t, []ScriptCheck{{Op: ScriptCheckHealth, Register: 2, Unit: 1, HasUnit: true, Args: [scriptParams]int32{6}}}, nil, nil)
	w := scriptWorld(t, script, []Entity{{ID: 1, HP: 100, MaxHP: 100}})
	s := originalSessionFixture()
	s.HasClock, s.Clock = true, SessionClock{6, 584}
	if err := w.ImportOriginalSession(s); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 30}})
	if w.ScriptRegister(2) != 100 || w.entities[0].HP != 70 {
		t.Fatalf("script/command order: register=%d HP=%d", w.ScriptRegister(2), w.entities[0].HP)
	}
}

func TestSessionClockReportPublishesOnlyAfterEntry14(t *testing.T) {
	for _, tc := range []struct {
		sub  uint32
		want Outcome
	}{{14, OutcomeWon}, {15, OutcomeUndecided}} {
		w := clockWorld1112(t, tc.sub, 584, nil)
		w.won = 1
		Step(w, nil)
		if w.Outcome() != tc.want {
			t.Fatalf("entry%d published outcome%v want%v", tc.sub, w.Outcome(), tc.want)
		}
	}
}

func TestSessionClockSavedFullControlsPoolsBeforeCommand(t *testing.T) {
	for _, tc := range []struct {
		sub, full uint32
		hp, mp    int32
	}{
		{12, 584, 52, 51}, {12, 585, 50, 51}, {12, 586, 50, 51},
		{12, 587, 50, 51}, {12, 588, 52, 51},
		{12, 0x80000000, 52, 51}, {12, 0xffffffff, 50, 51},
		{11, 584, 50, 50}, {13, 584, 50, 50}, {0x8000000c, 584, 50, 50},
	} {
		w := clockWorld1112(t, tc.sub, tc.full, []Entity{{ID: 1, HP: 50, MaxHP: 100, Mana: 50, MaxMana: 100, HealthRegenPeriod: 100, ManaRegenPeriod: 100}})
		w.entities[0].CurrentProfileBasis = ProfileOriginalCurrent
		Step(w, nil)
		if e := w.entities[0]; e.HP != tc.hp || e.Mana != tc.mp {
			t.Fatalf("S/F=%x/%x HP/MP=%d/%d want%d/%d", tc.sub, tc.full, e.HP, e.Mana, tc.hp, tc.mp)
		}
	}
	// Full health at entry means no regen. Damage in the body must leave50,
	// not52: evaluating phase12 after commands is observably wrong.
	w := clockWorld1112(t, 12, 584, []Entity{{ID: 1, HP: 100, MaxHP: 100, HealthRegenPeriod: 100}})
	Step(w, []Command{{Kind: KindDamage, Entity: 1, X: 50}})
	if w.entities[0].HP != 50 {
		t.Fatal("regeneration ran after the command", w.entities[0].HP)
	}
}

func TestSessionClockLiveAndDeadDecayHaveDifferentFullFilters(t *testing.T) {
	for _, tc := range []struct {
		full       uint32
		live, dead int32
	}{
		{584, -3, -21}, {585, -2, -20}, {586, -2, -21}, {587, -2, -20}, {588, -3, -21},
	} {
		w := clockWorld1112(t, 12, tc.full, []Entity{
			{ID: 1, HP: -2, MaxHP: 100, DyingTime: 8},
			{ID: 2, HP: -20, MaxHP: 100, Decay: DecayBones},
			{ID: 3, HP: 0, MaxHP: 100, DyingTime: 8},
		})
		w.entities[0].Dwell = 3 // retained countdown, not a constructor claim
		Step(w, nil)
		if w.entities[0].HP != tc.live || w.entities[1].HP != tc.dead || w.entities[2].HP != 0 || w.entities[0].Dwell != 2 {
			t.Fatalf("F%d live/dead/zero/dwell=%d/%d/%d/%d", tc.full, w.entities[0].HP, w.entities[1].HP, w.entities[2].HP, w.entities[0].Dwell)
		}
	}
	// On an even full tick, the phase12 live loss reaching-10 is followed
	// by the body's teardown call to the dead ladder, producing-11.
	w := clockWorld1112(t, 12, 584, []Entity{{ID: 1, HP: -9, MaxHP: 100}})
	Step(w, nil)
	if e := w.entities[0]; e.HP != -11 || e.Decay != DecayBones {
		t.Fatal("phase12 live loss -> body teardown -> dead call", e)
	}
}

func TestSessionClockLateSessionFailureDoesNotReplaceEitherWord(t *testing.T) {
	w := clockWorld1112(t, 9343, 584, nil)
	s := originalSessionFixture()
	s.HasClock, s.Clock = true, SessionClock{0xffffffff, 99}
	s.Diplomacy = s.Diplomacy[:2499]
	before := w.Hash()
	if err := w.ImportOriginalSession(s); err == nil || w.Hash() != before {
		t.Fatalf("late session error changed canonical state: %v", err)
	}
	assertClock1112(t, w, 9343, 584)
}
