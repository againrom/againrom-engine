package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -dump-base stands alone: it excludes -check, -headless and -mods.
func TestDumpBaseFlagExcludesOtherModes(t *testing.T) {
	o, err := parse([]string{"-dump-base", "out"})
	if err != nil || o.dumpBase != "out" {
		t.Fatalf("parsed %q: %v", o.dumpBase, err)
	}
	for _, args := range [][]string{
		{"-dump-base", "out", "-check"},
		{"-dump-base", "out", "--headless", "s.json"},
		{"-dump-base", "out", "-mods", "a"},
	} {
		if _, err := parse(args); err == nil || !strings.Contains(err.Error(), "-dump-base excludes") {
			t.Errorf("%v: %v", args, err)
		}
	}
}

// -dump-base writes the mod folder and names each file; a second run into
// the same folder and a run into the install are refused, naming why, and
// write nothing.
func TestDumpBaseWritesAModFolderAndRefuses(t *testing.T) {
	root := defaultInstall(t)
	dir := filepath.Join(t.TempDir(), "base-dump")
	code, out, stderr := checkRun(t, []string{"-assets", root, "-dump-base", dir}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, file := range []string{"mod.toml", "main.star", "data/towns.toml"} {
		if !strings.Contains(out, "againrom: wrote "+file+"\n") {
			t.Errorf("the report does not name %s:\n%s", file, out)
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(file))); err != nil {
			t.Error(err)
		}
	}
	code, _, stderr = checkRun(t, []string{"-assets", root, "-dump-base", dir}, "")
	if code != 1 || !strings.Contains(stderr, "a base dump writes only into a new or empty folder") {
		t.Fatalf("a second dump into %s: exit %d %q", dir, code, stderr)
	}
	inside := filepath.Join(root, "base-dump")
	code, _, stderr = checkRun(t, []string{"-assets", root, "-dump-base", inside}, "")
	if code != 1 || !strings.Contains(stderr, "a base dump never writes into an install") {
		t.Fatalf("a dump into the install: exit %d %q", code, stderr)
	}
	if _, err := os.Lstat(inside); !os.IsNotExist(err) {
		t.Fatalf("the refused dump left %s: %v", inside, err)
	}
}

// On the install the dumped folder is a mod -check accepts: the summary is
// the one without it, followed by the mod's lines, one town record and the
// scenario's tavern rows the dump writes (22 on EN and RU, as
// TestReleaseTavernRowKeyIsUnique measures), and no override is reported.
func TestReleaseDumpBaseWritesAModTheCheckAccepts(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the base dump needs the install")
	}
	mods := t.TempDir()
	code, out, stderr := checkRun(t, []string{"-assets", root, "-dump-base", filepath.Join(mods, "base-dump")}, "")
	if code != 0 || !strings.Contains(out, "againrom: wrote data/towns.toml") || strings.Contains(out, "note:") {
		t.Fatalf("dump: exit %d out %q err %q", code, out, stderr)
	}
	code, plain, _ := checkRun(t, []string{"-assets", root, "-check", "-picker"}, "")
	if code != 0 {
		t.Fatalf("plain check exit %d", code)
	}
	code, withMod, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "base-dump", "-mods-dir", mods}, "")
	if code != 0 || !strings.HasPrefix(withMod, plain) || !strings.Contains(withMod, "againrom: town records=1 tavern rows=22\n") || strings.Contains(stderr, "override") {
		t.Fatalf("check with the dump: exit %d\n%s\nvs\n%s\nerr %q", code, withMod, plain, stderr)
	}
}
