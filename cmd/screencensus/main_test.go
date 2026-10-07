package main

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

// This file is cmd/screencensus's first test file (1024 round 3 punch
// list, item 7: "? againrom/cmd/screencensus [no test files]" before this
// round). unwitnessedRemainder and geometryGaps are pure functions over
// []ui.ScreenCensusEntry and need no subprocess, no install, and no
// go/parser scan of the running binary's own source, so they are witnessed
// directly. listScreens (the subprocess/regexp half) is not: it shells out
// to "go run ./cmd/screenshot -list", which is exactly what a synthetic,
// install-free test in this module must not depend on doing successfully
// inside every CI/sandbox invocation (AGENTS.md gate 2's own bar). Its own
// regexp (listCountRe) and its per-line pass-through are simple enough that
// this round leaves them unwitnessed rather than building a fake
// "go run" stand-in for a two-line parser.

func TestUnwitnessedRemainderReportsOnlyScreensWithNoEffectiveGeometryTest(t *testing.T) {
	census := []ui.ScreenCensusEntry{
		{Name: "witnessed", Composer: "compose1", Test: "TestSelect1", EffectiveGeometryTests: []string{"TestGeom1"}},
		{Name: "bare", Composer: "compose2", Test: "TestSelect2", EffectiveGeometryTests: nil},
		{Name: "refuses", Composer: "", Reason: "no CPU composite"},
		{Name: "shared", Composer: "compose3", Test: "TestSelect3", SharedWith: "witnessed", EffectiveGeometryTests: []string{"TestGeom1"}},
	}
	got := unwitnessedRemainder(census)
	want := []string{"bare: composes (compose2), selection witnessed (TestSelect2), no geometry witness"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unwitnessedRemainder(census) = %#v, want %#v", got, want)
	}
}

// TestUnwitnessedRemainderIsEmptyWhenEveryComposerHasAnEffectiveWitness is
// the shape this whole mechanism exists for (1024 round 3, finding 1): a
// screen with an empty OWN GeometryTests but a non-empty EFFECTIVE one
// (SharesGeometry resolved it, ui.ScreenCensus's own job) must not be
// reported as unwitnessed.
func TestUnwitnessedRemainderIsEmptyWhenEveryComposerHasAnEffectiveWitness(t *testing.T) {
	census := []ui.ScreenCensusEntry{
		{Name: "town", Composer: "composeTownScreen", Test: "TestSelectTown", GeometryTests: []string{"TestGeom1"}, EffectiveGeometryTests: []string{"TestGeom1"}},
		{Name: "gamemenu", Composer: "composeTownScreen", Test: "TestSelectTown", SharedWith: "town", EffectiveGeometryTests: []string{"TestGeom1"}},
	}
	if got := unwitnessedRemainder(census); len(got) != 0 {
		t.Fatalf("unwitnessedRemainder(census) = %#v, want an empty slice", got)
	}
}

func TestUnwitnessedRemainderSkipsRefusingScreens(t *testing.T) {
	census := []ui.ScreenCensusEntry{
		{Name: "picker", Composer: "", Reason: "no CPU composite"},
	}
	if got := unwitnessedRemainder(census); len(got) != 0 {
		t.Fatalf("unwitnessedRemainder(census) = %#v, want an empty slice for a refusing (non-composing) screen", got)
	}
}

func TestGeometryGapsReportsOnlyEntriesCarryingAGapNote(t *testing.T) {
	census := []ui.ScreenCensusEntry{
		{Name: "chargen", GeometryGapNote: "composeChargenDetailedPage has no geometry witness"},
		{Name: "town", GeometryGapNote: ""},
		{Name: "gamemenu", GeometryGapNote: ""},
	}
	got := geometryGaps(census)
	want := []string{"chargen: composeChargenDetailedPage has no geometry witness"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("geometryGaps(census) = %#v, want %#v", got, want)
	}
}

func TestGeometryGapsIsEmptyWhenNoEntryCarriesANote(t *testing.T) {
	census := []ui.ScreenCensusEntry{
		{Name: "town", GeometryGapNote: ""},
	}
	if got := geometryGaps(census); len(got) != 0 {
		t.Fatalf("geometryGaps(census) = %#v, want an empty slice", got)
	}
}

func TestUnwitnessedRemainderAndGeometryGapsAgainstTheLiveRegistry(t *testing.T) {
	census := ui.ScreenCensus()
	for _, rows := range [][]string{unwitnessedRemainder(census), geometryGaps(census)} {
		if len(rows) != 2 || !strings.HasPrefix(rows[0], "cutscenes: ") || !strings.HasPrefix(rows[1], "credits: ") {
			t.Errorf("media geometry remainder = %#v, want only cutscenes and credits", rows)
		}
	}
}
