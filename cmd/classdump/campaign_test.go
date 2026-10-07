package main

// Tests for the campaign census. Every fixture is a synthetic archive written to
// a temp dir; no game install is read.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// campaignRoot lays a synthetic asset root: a campaign container holding one map
// and the NPC registry, a world container holding the definition table, and one
// loose map beside them — which is the shape of the corpus the verb covers.
func campaignRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	embedded := synth.ALM(synth.ALMOptions{
		Width: 32, Height: 32,
		Units: []synth.ALMUnit{
			{X: 0x0A80, Y: 0x0A80, ClassID: 200},          // units, resolving
			{X: 0x0B80, Y: 0x0A80, ClassID: 7},            // humans by type
			{X: 0x0C80, Y: 0x0A80, ClassID: 7, Flags: 1},  // npc — subscript 0, naming nothing
			{X: 0x0D80, Y: 0x0A80, ClassID: 7, DefID: 77}, // server id
		},
	})
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("scenario.res", synth.Archive([]synth.File{
		{Path: "10.alm", Data: embedded},
		{Path: "npc.reg", Data: synth.NPCReg(map[int32]int32{51: 509})},
	}))
	write("world.res", synth.Archive([]synth.File{{Path: "data/data.bin", Data: synthDataBin()}}))
	write("loose.alm", synth.ALM(synth.ALMOptions{
		Width: 32, Height: 32,
		Units: []synth.ALMUnit{{X: 0x0A80, Y: 0x0A80, ClassID: 201}}, // units, matching nothing
	}))
	return dir
}

// Both halves of the corpus are covered, each map's arms are counted, and the
// totals sum to the placements walked.
func TestCampaignVerbCoversBothHalvesOfTheCorpus(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"-campaign", campaignRoot(t)}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()

	// map, type6, units, server-id, npc, humans, then the drop column.
	for _, want := range [][]string{
		{"10.alm", "4", "1", "1", "1", "1", "none"},
		{"loose.alm", "1", "1", "0", "0", "0", "none"},
		{"TOTAL", "5", "2", "1", "1", "1", "0", "cell(s)", "over", "2", "map(s)"},
	} {
		if f := fieldsOfLine(t, got, want[0]); !sameFields(f, want) {
			t.Errorf("row %s reads %q, want %q", want[0], f, want)
		}
	}

	// The per-arm summary reports what each arm reached, which is the half the
	// row counts do not carry.
	for _, want := range []string{
		"arm units      taken      2  reached an entry      1",
		"arm server-id  taken      1  reached an entry      0",
		"arm npc        taken      1  reached an entry      0",
		"arm humans     taken      1  reached an entry      1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the summary has no %q:\n%s", want, got)
		}
	}
}

// A missing archive, a missing registry and a map that will not decode are each
// a failure of the whole run: a census whose denominator silently shrinks agrees
// with nothing and looks like it did.
func TestCampaignVerbRefusesAnIncompleteRoot(t *testing.T) {
	for _, tc := range []struct {
		name string
		mut  func(t *testing.T, dir string)
	}{
		{"no world archive", func(t *testing.T, dir string) {
			os.Remove(filepath.Join(dir, "world.res"))
		}},
		{"no npc registry", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "scenario.res"),
				synth.Archive([]synth.File{{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 32, Height: 32})}}),
				0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"a map that will not decode", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "broken.alm"), []byte("not a map"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := campaignRoot(t)
			tc.mut(t, dir)
			if err := run([]string{"-campaign", dir}, &bytes.Buffer{}); err == nil {
				t.Fatal("the run reported no failure")
			}
		})
	}
}

// The verb's own argument shape is a usage error, not a path that happens to
// miss.
func TestCampaignVerbArgumentShape(t *testing.T) {
	if err := run([]string{"-campaign"}, &bytes.Buffer{}); err == nil {
		t.Fatal("a bare verb was accepted")
	}
}

// The named map's NPC-arm placements are reported down both routes, and the two
// disagree — which is the only shape in which "the npc arm never reads its own
// definition id" is a measurement.
func TestCampaignVerbReportsTheNPCArmsTwoRoutes(t *testing.T) {
	dir := t.TempDir()
	// One placement on the npc arm carrying BOTH an npc subscript the registry
	// names and a definition id of its own, and the two name different entries.
	m := synth.ALM(synth.ALMOptions{
		Width: 32, Height: 32,
		Units: []synth.ALMUnit{
			{X: 0x0A80, Y: 0x0A80, ClassID: 200},                     // units: not on the arm
			{X: 0x0B80, Y: 0x0A80, ClassID: 7, Flags: 1, DefID: 900}, // npc 0: unnamed, but its own id resolves
			{X: 0x0C80, Y: 0x0A80, ClassID: 7, Flags: 1, ClassSubID: 51, DefID: 5},
		},
	})
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("scenario.res", synth.Archive([]synth.File{
		{Path: "10.alm", Data: m},
		{Path: "npc.reg", Data: synth.NPCReg(map[int32]int32{51: 900})},
	}))
	write("world.res", synth.Archive([]synth.File{{Path: "data/data.bin", Data: synthDataBin()}}))

	var out bytes.Buffer
	if err := run([]string{"-campaign", dir, "10.alm"}, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "10.alm: placements on the npc arm") {
		t.Fatalf("no npc-arm report:\n%s", got)
	}
	// record, npc, serverID, entry, own defID, entry.
	// The two rows invert each other, which is what makes them evidence. Record
	// 1's own definition id WOULD have resolved and its npc subscript does not,
	// and the arm still answers nothing; record 2's npc subscript resolves and
	// its own definition id does not, and the arm answers the registry's entry.
	for _, want := range [][]string{
		{"1", "0", "-", "0", "900", "1"},
		{"2", "51", "900", "1", "5", "0"},
	} {
		if f := fieldsOfLine(t, got, want[0]); !sameFields(f, want) {
			t.Errorf("npc row %s reads %q, want %q", want[0], f, want)
		}
	}
}

// Naming a map the corpus does not hold is a failure, not a silent census.
func TestCampaignVerbRefusesAnUnknownMapName(t *testing.T) {
	if err := run([]string{"-campaign", campaignRoot(t), "nosuch.alm"}, &bytes.Buffer{}); err == nil {
		t.Fatal("an unknown map name was accepted")
	}
	if err := run([]string{"-campaign", campaignRoot(t), "a", "b"}, &bytes.Buffer{}); err == nil {
		t.Fatal("four arguments were accepted")
	}
}
