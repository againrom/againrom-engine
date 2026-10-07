package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
)

// The fixture registry's class ids: the same descriptor over two sheets, one
// holding exactly the predicted total and one deliberately short of it.
const (
	auditFullID  = 3 // File 0 — units/full/walk.256, 54 frames: the predicted total
	auditShortID = 7 // File 1 — units/short/walk.256, 20 frames: deliberately short
)

// auditClass is one [UnitN] section carrying the 0024 spec's worked-example
// scalars — Flip 1, MB 1, MV 2, AT 2, DY 2, BN 2, ID 0, Move Time [2,1] /
// Frame [0,1] — so the loaded descriptor is the one the spec itself states as
// literals: bases 9/24/34/44, track [0,0,1], predicted total 54. Both classes
// share it; only their sheets differ.
func auditClass(id, file int32) []synth.RegNode {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	a := func(name string, vs ...int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x06, Ints: vs}
	}
	return []synth.RegNode{
		i("ID", id), i("File", file), i("Flip", 1),
		i("Width", 32), i("Height", 40), i("CenterX", 16), i("CenterY", 36),
		i("MoveBeginPhases", 1), i("MovePhases", 2), i("AttackPhases", 2),
		i("DyingPhases", 2), i("BonePhases", 2), i("IdlePhases", 0),
		a("MoveAnimTime", 2, 1), a("MoveAnimFrame", 0, 1),
	}
}

// auditSheet builds a palette-bearing sheet of n 1x1 wholly transparent
// frames. The audit reads the sheet as its frame COUNT, never a pixel, so the
// smallest drawable frame per slot is all a fixture needs.
func auditSheet(n int) []byte {
	frames := make([]synth.Frame256, n)
	for i := range frames {
		frames[i] = synth.Frame256{Width: 1, Height: 1}
	}
	return synth.Sheet256(synth.Sheet256Options{Frames: frames})
}

// auditArchiveFiles is the fixture graphics archive's content: the registry
// and the two sheets, 54 frames against the predicted 54 and 20 against it.
func auditArchiveFiles(t *testing.T) []synth.File {
	t.Helper()
	return []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: synth.UnitsReg([]string{`full\walk`, `short\walk`},
			auditClass(auditFullID, 0), auditClass(auditShortID, 1))},
		{Path: "units/full/walk.256", Data: auditSheet(54)},
		{Path: "units/short/walk.256", Data: auditSheet(20)},
	}
}

// writeAuditArchive lays the fixture graphics.res into dir.
func writeAuditArchive(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "graphics.res"),
		synth.Archive(auditArchiveFiles(t)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// unitAnimWant is the EXACT output the fixture must print — one row per
// class, ascending by id, and the summary — asserted by whole-string
// equality, so a missing line, a changed token or a swapped count all fail.
//
// The figures are worked out BY HAND from the spec's arithmetic, never
// recomputed through the code under test. The domain per class is 2 states *
// 8 octants * the track period 3 = 48. Moving: index 9 + slot*3 + 1 +
// track[step], slots at D 5 per octant 0,1,2,3,4,3,2,1, track values 0,0,1 —
// per octant {10,10,11}, {13,13,14}, {16,16,17}, {19,19,20}, {22,22,23},
// {19,19,20}, {16,16,17}, {13,13,14}. Idle fails its gate (ID 0), so the
// idle state is standing at S 9: frames 0,2,4,6,8,6,4,2 at any tick.
//
// Against 54 frames every index is in range: 48/0. Against 20, an index of
// 20 or more guards — one each at octants 3 and 5, all three at octant 4 —
// so moving is 19 in range and 5 guarded, standing all 24 in range: 43/5.
// The summary sums the rows: 48+43 = 91 in range, 5 guarded, and one class
// of the two mismatching its predicted total.
const unitAnimWant = "class 3: predicted 54, frames 54, in-range 48, guarded 0\n" +
	"class 7: predicted 54, frames 20, in-range 43, guarded 5\n" +
	"unitanim: 2 classes, 1 mismatched, 91 in-range, 5 guarded\n"

// TestUnitAnimLines — SC-9: the subcommand prints exactly the audit's rows
// and summary, and a mismatched sheet is DATA — the run with a guarded count
// on it still exits zero.
func TestUnitAnimLines(t *testing.T) {
	dir := t.TempDir()
	writeAuditArchive(t, dir)

	t.Run("-assets", func(t *testing.T) {
		var buf bytes.Buffer
		if err := run([]string{"unitanim", "-assets", dir}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != unitAnimWant {
			t.Errorf("output = %q, want %q", got, unitAnimWant)
		}
	})

	t.Run("-graphics", func(t *testing.T) {
		// The archive in a directory no asset root names, and no -assets given:
		// only -graphics can be what found it. It keeps the canonical FILE NAME
		// for the reason the census subcommand's twin does — the identity the
		// bundle's addresses carry is that name's.
		alt := t.TempDir()
		archivePath := filepath.Join(alt, graphicsArchive)
		if err := os.WriteFile(archivePath, synth.Archive(auditArchiveFiles(t)), 0o644); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := run([]string{"unitanim", "-graphics", archivePath}, &buf); err != nil {
			t.Fatalf("run: %v", err)
		}
		if got := buf.String(); got != unitAnimWant {
			t.Errorf("output = %q, want %q", got, unitAnimWant)
		}
	})
}

// TestUnitAnimErrors — the one error class: a LOAD failure exits non-zero —
// main exits non-zero on any error run returns, the shipped wiring — and a
// failed run prints NOTHING, so a recorded audit can never be a partial one.
func TestUnitAnimErrors(t *testing.T) {
	dir := t.TempDir()
	writeAuditArchive(t, dir)

	// The broken install: a graphics.res that opens but holds no unit
	// registry, so the archive read succeeds and the bundle load fails.
	noReg := t.TempDir()
	if err := os.WriteFile(filepath.Join(noReg, "graphics.res"), synth.Archive([]synth.File{
		{Path: "units/full/walk.256", Data: auditSheet(54)},
	}), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		args  []string
		names string // a substring the error must carry, "" for none pinned
	}{
		{"no asset root", []string{"unitanim"}, ""},
		// -map is units' flag, not this subcommand's: unitanim takes exactly
		// its two, and the audit involves no map.
		{"unknown flag", []string{"unitanim", "-assets", dir, "-map", "x.alm"}, ""},
		{"archive not found", []string{"unitanim", "-assets", filepath.Join(dir, "nope")}, ""},
		{"no unit registry in the archive", []string{"unitanim", "-assets", noReg}, game.UnitRegistry},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Hermetic even on a machine whose environment names a real
			// install: the no-asset-root case must not find one there.
			t.Setenv("AGAINROM_ASSETS", "")
			var stdout bytes.Buffer
			err := run(tc.args, &stdout)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if stdout.Len() != 0 {
				t.Errorf("a failed run wrote output: %q", stdout.String())
			}
			if tc.names != "" && !strings.Contains(err.Error(), tc.names) {
				t.Errorf("error %q does not name %q", err, tc.names)
			}
		})
	}
}
