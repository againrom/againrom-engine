package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/game"
)

func TestMapEditor1092ArgumentFailuresDoNotOpenWindow(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"one.alm", "two.alm"}, "unexpected arguments"},
		{[]string{"one.alm", "-check"}, "unexpected arguments"},
		{[]string{"-map", "scenario/10.alm", "one.alm"}, "excludes -map and -new"},
		{[]string{"-new", "32x32", "one.alm"}, "excludes -map and -new"},
		{[]string{"-size", "800x600garbage"}, "invalid size"},
		{[]string{"-size", "639x480"}, "invalid size"},
		{[]string{"-check"}, "requires -map"},
		{[]string{"-scroll-details", "1"}, "requires -snapshot"},
		{[]string{"-scroll-details", "-1"}, "range"},
		{[]string{"-focus-reference", "1"}, "requires -select"},
		{[]string{"-focus-reference", "-1"}, "positive reference"},
		{[]string{"-new", "32x32", "-map", "scenario/10.alm"}, "exclude each other"},
		{[]string{"-out", "x.alm"}, "-out requires -new"},
	} {
		if err := run(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%v: %v", c.args, err)
		}
	}
}

// installFolder makes a folder that holds the archive names of an install; the
// files are empty, so it is a root to the discovery and nothing more.
func installFolder(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range game.RequiredArchives() {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeFile(t *testing.T, path, text string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

var errStopped = errors.New("stopped at the install")

// rootChosen runs the program at self with args and answers the root it handed
// to the install reader; the reader is replaced, so no install is read.
func rootChosen(t *testing.T, self string, args ...string) string {
	t.Helper()
	saved := newInspector
	t.Cleanup(func() { newInspector = saved })
	var root string
	newInspector = func(r string) (*game.MapInspector, error) { root = r; return nil, errStopped }
	if err := runAs(self, args, io.Discard); !errors.Is(err, errStopped) {
		t.Fatalf("%v: %v", args, err)
	}
	return root
}

// A map path given as the one argument is the map -map names: it reaches the
// same root choice, here through the install that holds the map.
func TestPositionalMapPathIsTheMapFlag(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	base := resolvedTempDir(t)
	install := installFolder(t, filepath.Join(base, "install"))
	m := writeFile(t, filepath.Join(install, "maps", "a.alm"), "x")
	self := filepath.Join(base, "program", "mapedit.exe")
	for _, args := range [][]string{{m}, {"-map", m}, {"-check", m}} {
		if got := rootChosen(t, self, args...); got != install {
			t.Errorf("%v: root %q, want %q", args, got, install)
		}
	}
}

// With no flag and no variable the root comes from the program's folder first.
func TestRootComesFromTheProgramsFolder(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	base := t.TempDir()
	beside := installFolder(t, filepath.Join(base, "game"))
	other := installFolder(t, filepath.Join(base, "other"))
	m := writeFile(t, filepath.Join(other, "maps", "a.alm"), "x")
	self := filepath.Join(beside, "mapedit.exe")
	if got := rootChosen(t, self, m); got != beside {
		t.Errorf("root %q, want the program's folder %q", got, beside)
	}
}

// A program outside any install reads the install that holds the map.
func TestRootComesFromTheInstallHoldingTheMap(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	base := resolvedTempDir(t)
	install := installFolder(t, filepath.Join(base, "install"))
	m := writeFile(t, filepath.Join(install, "deep", "er", "a.alm"), "x")
	self := filepath.Join(base, "tools", "mapedit.exe")
	if got := rootChosen(t, self, m); got != install {
		t.Errorf("root %q, want %q", got, install)
	}
}

// A map outside any install, with a starter beside the program, reads the
// base the starter selected; without one there is no root and the error says so.
func TestRootComesFromTheStartersSelectedBase(t *testing.T) {
	t.Setenv("AGAINROM_ASSETS", "")
	base := t.TempDir()
	en := installFolder(t, filepath.Join(base, "en"))
	ru := installFolder(t, filepath.Join(base, "ru"))
	tools := filepath.Join(base, "tools")
	self := filepath.Join(tools, "mapedit.exe")
	m := writeFile(t, filepath.Join(base, "mods", "a.alm"), "x")
	if err := runAs(self, []string{m}, io.Discard); err == nil || !strings.Contains(err.Error(), "no asset root") {
		t.Fatalf("no starter: %v", err)
	}
	ini := "[starter]\nlast-base=ru\n[bases]\nen=" + en + "\nru=" + ru + "\n"
	writeFile(t, filepath.Join(tools, "starter.ini"), ini)
	if got := rootChosen(t, self, m); got != ru {
		t.Errorf("root %q, want the selected base %q", got, ru)
	}
	writeFile(t, filepath.Join(tools, "starter.ini"), "[bases]\nen="+en+"\n")
	if got := rootChosen(t, self, m); got != en {
		t.Errorf("root %q, want the first base %q", got, en)
	}
	writeFile(t, filepath.Join(tools, "starter.ini"), "[bases]\nen="+filepath.Join(base, "missing")+"\n")
	if err := runAs(self, []string{m}, io.Discard); err == nil || !strings.Contains(err.Error(), "no asset root") {
		t.Fatalf("base is no install: %v", err)
	}
}

// The editor's order is the flag, the variable, the program's folders, the
// install holding the map, then the starter's base; each step is removed in
// turn and the next one takes over.
func TestAssetRootOrder(t *testing.T) {
	base := resolvedTempDir(t)
	process := installFolder(t, filepath.Join(base, "process"))
	opened := installFolder(t, filepath.Join(base, "opened"))
	starter := installFolder(t, filepath.Join(base, "starter"))
	m := writeFile(t, filepath.Join(opened, "a.alm"), "x")
	tools := filepath.Join(base, "tools")
	writeFile(t, filepath.Join(tools, "starter.ini"), "[bases]\na="+starter+"\n")
	inProcess := filepath.Join(process, "mapedit.exe")
	inTools := filepath.Join(tools, "mapedit.exe")
	steps := []struct {
		flag, env, mapPath, self, want string
	}{
		{"f", "e", m, inProcess, "f"},
		{"", "e", m, inProcess, "e"},
		{"", "", m, inProcess, process},
		{"", "", m, inTools, opened},
		{"", "", "", inTools, starter},
		{"", "", "", filepath.Join(base, "none", "mapedit.exe"), ""},
	}
	for _, s := range steps {
		if got := assetRoot(s.flag, s.env, s.mapPath, s.self); got != s.want {
			t.Errorf("assetRoot(%q, %q, %q, %q) = %q, want %q", s.flag, s.env, s.mapPath, s.self, got, s.want)
		}
	}
}

// The flag and the variable keep their place ahead of every discovery.
func TestRootFlagAndVariableWin(t *testing.T) {
	base := t.TempDir()
	beside := installFolder(t, filepath.Join(base, "game"))
	flagRoot := filepath.Join(base, "flag")
	envRoot := filepath.Join(base, "env")
	self := filepath.Join(beside, "mapedit.exe")
	t.Setenv("AGAINROM_ASSETS", envRoot)
	if got := rootChosen(t, self, "a.alm"); got != envRoot {
		t.Errorf("root %q, want the variable %q", got, envRoot)
	}
	if got := rootChosen(t, self, "-assets", flagRoot, "a.alm"); got != flagRoot {
		t.Errorf("root %q, want the flag %q", got, flagRoot)
	}
}

// A failed start reaches stderr; with no stderr it reaches a log beside the
// player profile.
func TestReportFailureWritesTheLogWhenThereIsNoConsole(t *testing.T) {
	failure := errors.New("no asset root")
	var shown strings.Builder
	dir := t.TempDir()
	reportFailure(&shown, dir, failure)
	if shown.String() != "mapedit: no asset root\n" {
		t.Errorf("stderr got %q", shown.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "Againrom", errorLogName)); err == nil {
		t.Error("a log was written although stderr took the error")
	}
	reportFailure(brokenWriter{}, dir, failure)
	b, err := os.ReadFile(filepath.Join(dir, "Againrom", errorLogName))
	if err != nil || string(b) != "mapedit: no asset root\n" {
		t.Errorf("log %q, %v", b, err)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, os.ErrInvalid }
