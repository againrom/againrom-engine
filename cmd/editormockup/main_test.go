package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/game"
)

// fakeInstall is a temporary folder holding the install archive names.
func fakeInstall(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{game.MainArchive, game.GraphicsArchive, game.ScenarioArchive, game.WorldArchive, game.MoviesArchive} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// The tool writes nowhere the engine's install fence refuses, and writes a
// file outside every install.
func TestToolRefusesWhereTheInstallFenceDoes(t *testing.T) {
	install := fakeInstall(t)
	t.Setenv("AGAINROM_ASSETS", "")
	refused := []struct {
		name string
		cwd  string
		args []string
		file string
	}{
		{"relative out, working directory inside the install", install,
			[]string{"page", "-assets", install, "-out", "rel.html"}, filepath.Join(install, "rel.html")},
		{"out named with leading dots", t.TempDir(),
			[]string{"page", "-assets", install, "-out", filepath.Join(install, "..x.json")}, filepath.Join(install, "..x.json")},
		{"out inside an install, no assets", t.TempDir(),
			[]string{"page", "-out", filepath.Join(install, "abs.html")}, filepath.Join(install, "abs.html")},
	}
	for _, c := range refused {
		t.Chdir(c.cwd)
		err := run(c.args)
		if err == nil || !strings.Contains(err.Error(), "inside the install") {
			t.Errorf("%s: err %v", c.name, err)
		}
		if exists(c.file) {
			t.Errorf("%s: %s was written", c.name, c.file)
		}
	}
	outside := filepath.Join(t.TempDir(), "page.html")
	t.Chdir(t.TempDir())
	if err := run([]string{"page", "-out", outside}); err != nil || !exists(outside) {
		t.Errorf("out outside the install: err %v, written %v", err, exists(outside))
	}
}
