package main

// The readout, witnessed with NO INSTALL PRESENT. Everything below is a world
// built in test code, so what is measured is the rendering and not a mission.

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// trcWorld is one world carrying a script with every shape a line has to
// render: a firing whose pair reads a measured register against a constant, a
// loss counted by a check on a unit that is not alive, and a check that writes
// nothing because it names no group.
//
// The two entities are named u7 and u9 by the map's own script, so a line that
// printed an internal entity id instead would be visible rather than plausible.
func trcWorld(t *testing.T) (*sim.World, map[uint16]sim.EntityID) {
	t.Helper()
	s, err := sim.NewScript(
		[]sim.ScriptCheck{
			{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}},
			{Op: sim.ScriptCheckAlive, Register: 1, Unit: 1, HasUnit: true},
			{Op: sim.ScriptCheckVIP, Register: 2, Unit: 2, HasUnit: true},
			{Op: sim.ScriptCheckGroupCount, Register: 3},
		},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantLose}},
		[]sim.ScriptTrigger{{
			Pairs:    [3]sim.ScriptPair{{Left: 1, Right: 0, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, -1, -1, -1},
			Latch:    5,
		}},
	)
	if err != nil {
		t.Fatalf("NewScript: %v", err)
	}
	w, err := sim.NewScriptedWorld(3, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: 1, X: 1, Y: 1, HP: 10, MaxHP: 10},
			{ID: 2, X: 4, Y: 4, HP: -10, MaxHP: 10},
		}, s)
	if err != nil {
		t.Fatalf("NewScriptedWorld: %v", err)
	}
	return w, map[uint16]sim.EntityID{7: 1, 9: 2}
}

// trcRun advances one full cycle through the tracer and returns what it printed.
func trcRun(t *testing.T, on bool) string {
	t.Helper()
	w, names := trcWorld(t)
	var buf bytes.Buffer
	tr := newTracer(on, &buf, names, w.Script())
	tr.preamble()
	for i := 0; i < 32; i++ {
		tr.step(w, nil)
	}
	return buf.String()
}

func TestTheReadoutNamesEachEventInTheMapsOwnTerms(t *testing.T) {
	got := trcRun(t, true)
	for _, want := range []string{
		"script  4 checks, 1 instants, 1 triggers",
		"trigger 0 FIRED (map latch 5)",
		"r1[check 1 alive(u7)]",                      // the register, through its check, in map terms
		"r0[check 0 const(1)]",                       // the constant it was compared against
		"->  1 == 1  = true",                         // the values actually compared
		"instant 0 LOSE",                             // what the firing ran
		"check 2 vip(u9) COUNTED A LOSS",             // a loss belonging to no trigger
		"(-10 hp)",                                   // reached the completed-body boundary
		"check 3 groupcount(NO GROUP) WROTE NOTHING", // the silent check
		"it names no group",                          // and why
		"counters won=0 lost=2",                      // the firing's LOSE and the check's own
		"REPORT won=0 lost=2 -> undecided",           // two losses in one pass make no report
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the readout does not carry %q:\n%s", want, got)
		}
	}
	// A standing silence prints once, not on every pass. Two passes run in 32
	// ticks and the group check is silent on both.
	if n := strings.Count(got, "WROTE NOTHING"); n != 1 {
		t.Errorf("the silent check printed %d times; a standing condition prints on change", n)
	}
	// A firing is an event and prints every time its trigger holds.
	if n := strings.Count(got, "FIRED"); n != 2 {
		t.Errorf("the firing printed %d times over two passes", n)
	}
}

func TestTheReadoutIsSilentWhenItIsOff(t *testing.T) {
	if got := trcRun(t, false); got != "" {
		t.Errorf("the readout printed with the flag off:\n%s", got)
	}
}

// TestAnUnnamedUnitIsNotPrintedAsAMapNumber pins the honesty of the one
// substitution the readout makes. A party member has no script number at all, so
// naming it uNN would invent a reference a reader could look up and not find.
func TestAnUnnamedUnitIsNotPrintedAsAMapNumber(t *testing.T) {
	w, names := trcWorld(t)
	tr := newTracer(true, io.Discard, names, w.Script())
	if got := tr.unitText(1, true); got != "u7" {
		t.Errorf("unitText(named) = %q, want the map's own number", got)
	}
	if got := tr.unitText(41, true); !strings.Contains(got, "41") || strings.Contains(got, "u41") {
		t.Errorf("unitText(unnamed) = %q; an entity the script never named must not read as one", got)
	}
	if got := tr.unitText(0, false); got != "NO UNIT" {
		t.Errorf("unitText(absent) = %q, want the word", got)
	}
}

func TestBothItemTestOpcodesUseTheSameTraceName(t *testing.T) {
	for _, op := range []int32{sim.ScriptCheckItemTestAlias, sim.ScriptCheckItemTest} {
		if got := checkName(op); got != "itemtest" {
			t.Errorf("checkName(%d) = %q, want itemtest", op, got)
		}
	}
}
