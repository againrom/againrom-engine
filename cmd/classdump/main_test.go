package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// The tool over a synthetic graphics.res: internal/synth writes the archive and
// its three .reg entries, the real reader opens the file, and the real loader
// resolves it — so the whole path a developer runs is exercised with no game
// present (golden rule 2). The archive is written to the test's own temp dir,
// which is the only file this test touches.

func regDir(name string, children ...synth.RegNode) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x01, Children: children}
}

func regInt(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func regStr(name, s string) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x00, Str: s}
}

func regInts(name string, v ...int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x06, Ints: v}
}

// descText carries a byte outside 0x20-0x7E, written as bytes and never as
// literal non-ASCII text (golden rule 2): "hu" + 0x8f. A registry string is in
// an encoding this project maps nowhere, so the dump must escape it.
var descText = string([]byte{0x68, 0x75, 0x8f})

// graphicsArchive builds the archive the tool expects: the three registries at
// the entry paths spec.md's source contract names, each dense from index 0 and
// each carrying the inheritance its registry allows — a unit inheriting File,
// DescText and an array from its parent, an object whose Parent is 0 (a real
// reference to the class whose ID is 0), and structures, which inherit nothing.
func graphicsArchive() []byte {
	units := synth.Reg(0x11, []synth.RegNode{
		regDir("Global", regInt("UnitCount", 2), regInt("FileCount", 2)),
		regDir("Files", regStr("File0", "unused"), regStr("File1", `hu\Man`)),
		regDir("Unit0",
			regInt("ID", 1), regInt("File", 1), regStr("DescText", descText),
			regInts("Sound", 1, 2, 3, 4, 5), regInt("AttackDelay", 4)),
		regDir("Unit1", regInt("ID", 5), regInt("Parent", 1), regInt("AttackDelay", 0)),
	})
	objects := synth.Reg(0x11, []synth.RegNode{
		regDir("Global", regInt("ObjectCount", 2), regInt("FileCount", 2)),
		regDir("Files", regStr("File0", "tree"), regStr("File1", "rock")),
		regDir("Object0", regInt("ID", 0), regInt("File", 0), regStr("DescText", "tree")),
		regDir("Object1", regInt("ID", 1), regInt("File", 1), regInt("Parent", 0)),
	})
	structures := synth.Reg(0x11, []synth.RegNode{
		regDir("Global", regInt("Count", 2)),
		regDir("Structure0", regInt("ID", 1), regStr("File", `hou\Se`), regStr("DescText", "hut"),
			regInt("TileWidth", 2), regInt("FullHeight", 3)),
		regDir("Structure1", regInt("ID", 2), regStr("File", "wall"), regStr("DescText", "wall")),
	})
	return synth.Archive([]synth.File{
		{Path: unitsEntry, Data: units},
		{Path: objectsEntry, Data: objects},
		{Path: structuresEntry, Data: structures},
	})
}

func writeArchive(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "graphics.res")
	if err := os.WriteFile(path, graphicsArchive(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// block returns the lines of the "[name]" class block, up to the next block or
// header line, so one class's printed keys can be asserted on alone.
func block(t *testing.T, out, name string) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if l != "["+name+"]" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if !strings.HasPrefix(lines[j], "  ") {
				return lines[i+1 : j]
			}
		}
		return lines[i+1:]
	}
	t.Fatalf("no [%s] block in:\n%s", name, out)
	return nil
}

func has(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

func TestDumpPrintsAllThreeCollections(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{writeArchive(t)}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	out := buf.String()

	// The header line per registry, carrying the class count a manual run reads
	// off as 34 / 82 / 66.
	for _, want := range []string{
		"units/units.reg: 2 classes",
		"objects/objects.reg: 2 classes",
		"structures/structures.reg: 2 classes",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("no header line %q in:\n%s", want, out)
		}
	}

	// One block per class, named by numeric section order — All()'s order.
	for _, name := range []string{"Unit0", "Unit1", "Object0", "Object1", "Structure0", "Structure1"} {
		if !strings.Contains(out, "\n["+name+"]\n") {
			t.Errorf("no [%s] block in:\n%s", name, out)
		}
	}

	// A class's own keys at their three kinds, the derived paths beside them,
	// and the byte escape over a registry string.
	unit0 := block(t, out, "Unit0")
	for _, want := range []string{
		`  ID = 1`,
		`  File = 1`,
		`  DescText = "hu\x8f"`,
		`  Sound = [1 2 3 4 5]`,
		`  AttackDelay = 4`,
		`  ShootOffset = []`, // an array that resolves to nil prints as the empty list
		`  sprite = "units/hu/Man.256"`,
		`  overlay = "units/hu/Manb.256"`,
	} {
		if !has(unit0, want) {
			t.Errorf("[Unit0]: no line %q in:\n%s", want, strings.Join(unit0, "\n"))
		}
	}

	// The printed values are the RESOLVED ones: Unit1 sets neither File nor
	// DescText and prints its parent's, writes AttackDelay = 0 over its parent's
	// 4 and prints 0.
	unit1 := block(t, out, "Unit1")
	for _, want := range []string{
		`  ID = 5`,
		`  DescText = "hu\x8f"`,
		`  AttackDelay = 0`,
		`  sprite = "units/hu/Man.256"`,
	} {
		if !has(unit1, want) {
			t.Errorf("[Unit1]: no line %q in:\n%s", want, strings.Join(unit1, "\n"))
		}
	}

	object1 := block(t, out, "Object1")
	for _, want := range []string{
		`  ID = 1`,
		`  Parent = 0`,
		`  DescText = "tree"`, // inherited from the class whose ID is 0
		`  sprite = "objects/rock.256"`,
	} {
		if !has(object1, want) {
			t.Errorf("[Object1]: no line %q in:\n%s", want, strings.Join(object1, "\n"))
		}
	}

	structure0 := block(t, out, "Structure0")
	for _, want := range []string{
		`  ID = 1`,
		`  File = "hou\Se"`, // a structure's File is the stored path, backslash and all
		`  sprite = "structures/hou/Se.256"`,
		`  overlay = "structures/hou/Seb.256"`,
	} {
		if !has(structure0, want) {
			t.Errorf("[Structure0]: no line %q in:\n%s", want, strings.Join(structure0, "\n"))
		}
	}

	// The whole dump is pure ASCII: every registry byte outside 0x20-0x7E left
	// the tool as \xNN, so no output byte carries an encoding.
	for i := 0; i < len(out); i++ {
		if out[i] > 0x7e {
			t.Fatalf("output byte %#x at offset %d is above 0x7E; registry bytes must be escaped", out[i], i)
		}
	}
}

// A wrong argument count is a usage error, not a dump: run reports it and reads
// nothing.
func TestRunRejectsAWrongArgumentCount(t *testing.T) {
	for _, args := range [][]string{{}, {"a.res", "b.res"}} {
		var buf bytes.Buffer
		if err := run(args, &buf); err == nil {
			t.Errorf("run(%q) = nil, want a usage error", args)
		}
		if buf.Len() != 0 {
			t.Errorf("run(%q) wrote %q, want nothing", args, buf.String())
		}
	}
}

// An archive missing a registry fails the run rather than printing two
// collections: the tool loads all three or none.
func TestMissingRegistryFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "graphics.res")
	if err := os.WriteFile(path, synth.Archive([]synth.File{{Path: unitsEntry, Data: []byte{}}}), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run([]string{path}, &buf); err == nil {
		t.Fatal("run over an archive with no objects/structures registry = nil, want an error")
	}
}
