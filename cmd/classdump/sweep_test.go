package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// The sweep over synthetic maps: internal/synth writes each .alm and the
// graphics.res the classes come from, and the real readers, loader and
// classifier resolve them — so the path a developer runs against an install is
// exercised with no game present (golden rule 2). Every file lives in the test's
// own temp dirs, which are the only files these tests touch, and no output of a
// real run appears here (golden rule 1).
//
// The registries are main_test.go's: units carry IDs 1 and 5, objects 0 and 1,
// structures 1 and 2. Every key a test wants to miss is chosen outside those.

// The smallest grid that holds a placed overlay cell at more than one index,
// which is all of the grid the sweep reads.
const mapW, mapH = 2, 2

func writeMap(t *testing.T, dir, name string, o synth.ALMOptions) string {
	t.Helper()
	o.Width, o.Height = mapW, mapH
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, synth.ALM(o), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// sweepRun runs the tool's sweep half through its own argument parsing, so the
// invocation is exercised alongside the sweep.
func sweepRun(t *testing.T, dir string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := run([]string{"-sweep", dir, writeArchive(t)}, &buf)
	return buf.String(), err
}

// quoted is the source field as a census line carries it. The quoting is written
// out here rather than taken from the tool, so a test locating a line agrees
// with the format and not with the code that produced it. Every path these tests
// build is printable ASCII, so nothing in one is escaped.
func quoted(s string) string { return `"` + s + `"` }

// censusOf returns the fixed tokens of the line whose leading field is prefix — a
// quoted source path, or the bare "totals" of the totals block — from "type3"
// on, so an assertion compares the tokens themselves byte for byte.
func censusOf(t *testing.T, out, prefix string) string {
	t.Helper()
	var found string
	seen := 0
	for _, l := range strings.Split(out, "\n") {
		rest, ok := strings.CutPrefix(l, prefix+" ")
		if !ok || !strings.HasPrefix(rest, "type3 ") {
			continue
		}
		seen++
		found = rest
	}
	if seen != 1 {
		t.Fatalf("%d census lines led by %s, want 1, in:\n%s", seen, prefix, out)
	}
	return found
}

// failures returns the per-reference failure lines, in the order printed.
func failures(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "unresolved ") {
			lines = append(lines, l)
		}
	}
	return lines
}

func wantLine(t *testing.T, out, want string) {
	t.Helper()
	if !strings.Contains(out, want+"\n") {
		t.Errorf("no line %q in:\n%s", want, out)
	}
}

// SC-9, first sweep case: a sweep whose only non-direct records are diverted
// exits ZERO with them counted. A diverted record names a table this story does
// not hold, so it is not a failure to resolve — and the npc record here names a
// class key that resolves nowhere, which a tool resolving every record against
// units.reg would report as a failure.
func TestSweepCountsDivertedRecordsAndExitsZero(t *testing.T) {
	dir := t.TempDir()
	src := writeMap(t, dir, "diverted.alm", synth.ALMOptions{
		Overlay: []uint8{1, 2, 0, 0}, // codes 1 and 2: objects.reg IDs 0 and 1
		Objects: []synth.ALMObject{{Kind: 1}, {Kind: 2}},
		Units: []synth.ALMUnit{
			{ClassID: 1},                          // direct, and it resolves
			{ClassID: 99, Flags: 1},               // npc; the key resolves nowhere
			{ClassID: 5, DefID: 0xabcd},           // def; the key would have resolved
			{ClassID: 5, Flags: 1, DefID: 0xabcd}, // both; the key would have resolved
		},
	})

	out, err := sweepRun(t, dir)
	if err != nil {
		t.Fatalf("sweep: %v, want nil — a diverted record is counted, never failed", err)
	}

	want := "type3 nonzero=2 resolved=2 past-registry=0" +
		" type4 refs=2 resolved=2 unresolved=0" +
		" type6 records=4 direct=1 resolved=1 unresolved=0 npc=1 def=1 both=1 diverted-would-resolve=2"
	if got := censusOf(t, out, quoted(src)); got != want {
		t.Errorf("source census:\n got %s\nwant %s", got, want)
	}
	if got := censusOf(t, out, "totals"); got != want {
		t.Errorf("totals census over one source:\n got %s\nwant %s", got, want)
	}
	wantLine(t, out, "totals sources=1 skipped=0")

	if got := failures(out); len(got) != 0 {
		t.Errorf("failure lines %q, want none", got)
	}
	// The sweep is the sweep: the class dump is the other invocation, and 182
	// class blocks ahead of the census would bury it.
	if strings.Contains(out, "[Unit0]") {
		t.Errorf("the sweep printed the class dump:\n%s", out)
	}
	for i := 0; i < len(out); i++ {
		if out[i] > 0x7e {
			t.Fatalf("output byte %#x at offset %d is above 0x7E", out[i], i)
		}
	}
}

// SC-9, second sweep case: one unresolvable direct record exits non-zero, and
// the reference is named. Its key is written 0x8001 by the builder and must be
// reported at the -32767 the file's sign-extended read gives, not at the 32769
// an unsigned read would have missed for.
func TestSweepFailsOnAnUnresolvableDirectRecord(t *testing.T) {
	dir := t.TempDir()
	src := writeMap(t, dir, "direct.alm", synth.ALMOptions{
		Units: []synth.ALMUnit{{ClassID: 1}, {ClassID: -32767}},
	})

	out, err := sweepRun(t, dir)
	if err == nil {
		t.Fatal("sweep = nil, want an error: a direct type-6 reference did not resolve")
	}

	want := "type3 nonzero=0 resolved=0 past-registry=0" +
		" type4 refs=0 resolved=0 unresolved=0" +
		" type6 records=2 direct=2 resolved=1 unresolved=1 npc=0 def=0 both=0 diverted-would-resolve=0"
	if got := censusOf(t, out, quoted(src)); got != want {
		t.Errorf("source census:\n got %s\nwant %s", got, want)
	}

	wantFail := []string{"unresolved " + quoted(src) + " type6 index=1 value=-32767"}
	if got := failures(out); !equalLines(got, wantFail) {
		t.Errorf("failure lines:\n got %q\nwant %q", got, wantFail)
	}
}

// SC-9, third sweep case: a type-3 code resolving past the registry is
// counted and printed, and does not move the exit code. Such cells exist on
// shipped maps and whether the engine reads them is undecided, so a sweep
// that failed here would fail on the owner's install for a residual already
// known.
func TestSweepCountsATypeThreeCodePastTheRegistry(t *testing.T) {
	dir := t.TempDir()
	src := writeMap(t, dir, "overlay.alm", synth.ALMOptions{
		Overlay: []uint8{1, 9, 0, 0}, // code 1: objects.reg ID 0; code 9: ID 8, which no class holds
	})

	out, err := sweepRun(t, dir)
	if err != nil {
		t.Fatalf("sweep: %v, want nil — a type-3 code past the registry must not fail the run", err)
	}

	want := "type3 nonzero=2 resolved=1 past-registry=1" +
		" type4 refs=0 resolved=0 unresolved=0" +
		" type6 records=0 direct=0 resolved=0 unresolved=0 npc=0 def=0 both=0 diverted-would-resolve=0"
	if got := censusOf(t, out, quoted(src)); got != want {
		t.Errorf("source census:\n got %s\nwant %s", got, want)
	}
	if got := failures(out); len(got) != 0 {
		t.Errorf("failure lines %q, want none: a past-registry cell is a measurement, not a fault", got)
	}
}

// A type-4 reference that does not resolve fails the run and is named at its
// key.
func TestSweepFailsOnAnUnresolvedTypeFourReference(t *testing.T) {
	dir := t.TempDir()
	src := writeMap(t, dir, "objects.alm", synth.ALMOptions{
		Objects: []synth.ALMObject{
			{Kind: 1},          // structures.reg ID 1
			{Kind: 0x00010002}, // ID 2: the high half is not part of the key
			{Kind: 0x21},       // ID 33, which no class holds; the record carries an extension
			{Kind: 42},         // ID 42, which no class holds
		},
	})

	out, err := sweepRun(t, dir)
	if err == nil {
		t.Fatal("sweep = nil, want an error: two type-4 references did not resolve")
	}

	want := "type3 nonzero=0 resolved=0 past-registry=0" +
		" type4 refs=4 resolved=2 unresolved=2" +
		" type6 records=0 direct=0 resolved=0 unresolved=0 npc=0 def=0 both=0 diverted-would-resolve=0"
	if got := censusOf(t, out, quoted(src)); got != want {
		t.Errorf("source census:\n got %s\nwant %s", got, want)
	}

	wantFail := []string{
		"unresolved " + quoted(src) + " type4 index=2 value=33",
		"unresolved " + quoted(src) + " type4 index=3 value=42",
	}
	if got := failures(out); !equalLines(got, wantFail) {
		t.Errorf("failure lines:\n got %q\nwant %q", got, wantFail)
	}
}

// Sources are not de-duplicated: the same map bytes loose and inside a .res are
// two sources under two paths, each with its own census line and its own failure
// lines, and the totals are their sum. De-duplicating needs a content
// comparison, and one that answered "same map" would hide a divergence — which
// is the only thing a sweep of two copies can find.
//
// Doubling every token is also what catches a field left out of the running
// totals: a census summed field by field agrees with one source and not with two.
func TestSweepDoesNotDeDuplicateSources(t *testing.T) {
	dir := t.TempDir()
	m := synth.ALM(synth.ALMOptions{
		Width: mapW, Height: mapH,
		Overlay: []uint8{1, 9, 0, 0},
		Objects: []synth.ALMObject{{Kind: 1}, {Kind: 42}},
		Units: []synth.ALMUnit{
			{ClassID: 1},           // direct, resolves
			{ClassID: 99},          // direct, does not
			{ClassID: 5, Flags: 1}, // npc; the key would have resolved
			{ClassID: 7, DefID: 1}, // def; the key would not have
		},
	})
	loose := filepath.Join(dir, "dup.alm")
	if err := os.WriteFile(loose, m, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(dir, "bundle.res")
	if err := os.WriteFile(bundle, synth.Archive([]synth.File{{Path: "maps/dup.alm", Data: m}}), 0o644); err != nil {
		t.Fatal(err)
	}
	archived := bundle + "::maps/dup.alm"

	out, err := sweepRun(t, dir)
	if err == nil {
		t.Fatal("sweep = nil, want an error: each copy places two references that do not resolve")
	}

	perSource := "type3 nonzero=2 resolved=1 past-registry=1" +
		" type4 refs=2 resolved=1 unresolved=1" +
		" type6 records=4 direct=2 resolved=1 unresolved=1 npc=1 def=1 both=0 diverted-would-resolve=1"
	for _, src := range []string{loose, archived} {
		if got := censusOf(t, out, quoted(src)); got != perSource {
			t.Errorf("census of %s:\n got %s\nwant %s", src, got, perSource)
		}
	}

	doubled := "type3 nonzero=4 resolved=2 past-registry=2" +
		" type4 refs=4 resolved=2 unresolved=2" +
		" type6 records=8 direct=4 resolved=2 unresolved=2 npc=2 def=2 both=0 diverted-would-resolve=2"
	if got := censusOf(t, out, "totals"); got != doubled {
		t.Errorf("totals census:\n got %s\nwant %s", got, doubled)
	}
	wantLine(t, out, "totals sources=2 skipped=0")

	got := failures(out)
	if len(got) != 4 {
		t.Fatalf("%d failure lines, want 4 (two per source):\n%s", len(got), out)
	}
	for _, src := range []string{loose, archived} {
		for _, want := range []string{
			"unresolved " + quoted(src) + " type4 index=1 value=42",
			"unresolved " + quoted(src) + " type6 index=1 value=99",
		} {
			if !containsLine(got, want) {
				t.Errorf("no failure line %q in %q", want, got)
			}
		}
	}
}

// A source that does not decode is reported and counted as skipped, and does not
// move the exit code: it was never asked whether its classes resolve. The
// skipped figure is what keeps that visible — a sweep whose sources all failed
// to decode says so on its own output rather than passing silently.
func TestSweepSkipsASourceItCannotDecode(t *testing.T) {
	dir := t.TempDir()
	good := writeMap(t, dir, "good.alm", synth.ALMOptions{Units: []synth.ALMUnit{{ClassID: 1}}})
	bad := filepath.Join(dir, "bad.alm")
	if err := os.WriteFile(bad, []byte("not a map at all"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := sweepRun(t, dir)
	if err != nil {
		t.Fatalf("sweep: %v, want nil — an undecodable source is skipped, not failed", err)
	}
	wantLine(t, out, "totals sources=1 skipped=1")
	if !strings.Contains(out, "skipped "+quoted(bad)+": ") {
		t.Errorf("no skip line for %s in:\n%s", bad, out)
	}
	censusOf(t, out, quoted(good)) // the good source is still swept
}

// The sweep invocation is three arguments led by -sweep; anything else is a
// usage error, which exits 2 rather than 1, and reads nothing.
func TestRunRejectsAMalformedSweepInvocation(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"-sweep"},
		{"-sweep", dir},
		{"-sweep", dir, "graphics.res", "extra"},
		{dir, "graphics.res", "extra"},
	} {
		var buf bytes.Buffer
		err := run(args, &buf)
		if !errors.Is(err, errUsage) {
			t.Errorf("run(%q) = %v, want a usage error", args, err)
		}
		if buf.Len() != 0 {
			t.Errorf("run(%q) wrote %q, want nothing", args, buf.String())
		}
	}
}

func containsLine(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

func equalLines(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
