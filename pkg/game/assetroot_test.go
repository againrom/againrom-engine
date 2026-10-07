package game

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestResolveAssetRoot is a synthetic unit test: it exercises the precedence
// rule with in-code values only and never reads a game installation.
func TestResolveAssetRoot(t *testing.T) {
	cases := []struct {
		name, flag, env, want string
	}{
		{"flag wins over env", "/flag/root", "/env/root", "/flag/root"},
		{"env used when no flag", "", "/env/root", "/env/root"},
		{"empty when neither set", "", "", ""},
		{"whitespace flag falls back to env", "   ", "/env/root", "/env/root"},
		{"surrounding space trimmed", "  /flag/root  ", "", "/flag/root"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveAssetRoot(tc.flag, tc.env); got != tc.want {
				t.Errorf("ResolveAssetRoot(%q, %q) = %q, want %q", tc.flag, tc.env, got, tc.want)
			}
		})
	}
}

// installArchiveNames is the set of file names a directory must hold before
// DiscoverAssetRoot will call it an install.
//
// It is spelt out here rather than read from RequiredArchives on purpose. The
// expectation has to be independent of the list under test: built from
// RequiredArchives, every case below would keep passing after a renamed constant
// left the discovery recognising a file no install ships.
var installArchiveNames = []string{"main.res", "graphics.res", "scenario.res", "world.res", "movies.res"}

// writeInstallNames creates dir and writes an empty file for each name.
//
// It writes no game data. An install is recognised by the names it ships, and
// these are names; the files are zero bytes, so nothing here is an asset and
// boundary 1 is untouched.
func writeInstallNames(t *testing.T, dir string, names []string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// TestRequiredArchivesMatchesTheDiscoveredSet pins the coupling the discovery
// depends on: the archives OpenArchives opens are the archives DiscoverAssetRoot
// requires. A directory the discovery accepts is one the front end can open, and
// that holds only while the two lists are the same list.
func TestRequiredArchivesMatchesTheDiscoveredSet(t *testing.T) {
	if got := RequiredArchives(); !slices.Equal(got, installArchiveNames) {
		t.Errorf("RequiredArchives() = %v, want %v", got, installArchiveNames)
	}
}

// TestDiscoverAssetRootAcceptsOnlyACompleteInstall covers what makes a directory
// an install: all five archives, in whatever case the install ships them.
//
// The uppercase case is not hypothetical. The two lawful roots disagree — the EN
// root ships graphics.res and the RU root ships GRAPHICS.RES — so a comparison
// that did not fold case would answer differently for the two.
func TestDiscoverAssetRootAcceptsOnlyACompleteInstall(t *testing.T) {
	upper := make([]string, len(installArchiveNames))
	for i, name := range installArchiveNames {
		upper[i] = strings.ToUpper(name)
	}
	mixed := slices.Clone(installArchiveNames)
	mixed[1] = "Graphics.RES"

	t.Run("all five present", func(t *testing.T) {
		dir := writeInstallNames(t, t.TempDir(), installArchiveNames)
		if got := DiscoverAssetRoot(dir); got != dir {
			t.Errorf("DiscoverAssetRoot = %q, want %q", got, dir)
		}
	})

	t.Run("all five present in upper case", func(t *testing.T) {
		dir := writeInstallNames(t, t.TempDir(), upper)
		if got := DiscoverAssetRoot(dir); got != dir {
			t.Errorf("DiscoverAssetRoot = %q, want %q", got, dir)
		}
	})

	t.Run("all five present in mixed case", func(t *testing.T) {
		dir := writeInstallNames(t, t.TempDir(), mixed)
		if got := DiscoverAssetRoot(dir); got != dir {
			t.Errorf("DiscoverAssetRoot = %q, want %q", got, dir)
		}
	})

	// One missing archive is checked for EVERY member rather than for a
	// convenient one: a discovery that looked at four of the five would pass a
	// single case and fail exactly one of these.
	for _, missing := range installArchiveNames {
		t.Run("without "+missing, func(t *testing.T) {
			names := make([]string, 0, len(installArchiveNames)-1)
			for _, name := range installArchiveNames {
				if name != missing {
					names = append(names, name)
				}
			}
			dir := writeInstallNames(t, t.TempDir(), names)
			if got := DiscoverAssetRoot(dir); got != "" {
				t.Errorf("DiscoverAssetRoot = %q, want %q rejected for the missing %s", got, dir, missing)
			}
		})
	}

	t.Run("a required name that is a directory does not count", func(t *testing.T) {
		names := slices.Clone(installArchiveNames)
		dir := writeInstallNames(t, t.TempDir(), names[1:])
		if err := os.MkdirAll(filepath.Join(dir, names[0]), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := DiscoverAssetRoot(dir); got != "" {
			t.Errorf("DiscoverAssetRoot = %q, want a directory named %s rejected", got, names[0])
		}
	})

	t.Run("an empty directory is not an install", func(t *testing.T) {
		dir := t.TempDir()
		if got := DiscoverAssetRoot(dir); got != "" {
			t.Errorf("DiscoverAssetRoot = %q, want %q rejected", got, dir)
		}
	})
}

// TestDiscoverAssetRootTakesTheFirstQualifyingCandidate covers the ordering the
// caller relies on, and the candidates it cannot supply: an empty string when a
// lookup failed, and a path that does not exist.
func TestDiscoverAssetRootTakesTheFirstQualifyingCandidate(t *testing.T) {
	first := writeInstallNames(t, filepath.Join(t.TempDir(), "first"), installArchiveNames)
	second := writeInstallNames(t, filepath.Join(t.TempDir(), "second"), installArchiveNames)
	partial := writeInstallNames(t, filepath.Join(t.TempDir(), "partial"), installArchiveNames[:3])
	absent := filepath.Join(t.TempDir(), "nothing-here")

	t.Run("the earlier install wins", func(t *testing.T) {
		if got := DiscoverAssetRoot(first, second); got != first {
			t.Errorf("DiscoverAssetRoot = %q, want %q", got, first)
		}
	})

	t.Run("non-installs are skipped, not fatal", func(t *testing.T) {
		if got := DiscoverAssetRoot("", "   ", absent, partial, second); got != second {
			t.Errorf("DiscoverAssetRoot = %q, want %q", got, second)
		}
	})

	t.Run("no candidates at all", func(t *testing.T) {
		if got := DiscoverAssetRoot(); got != "" {
			t.Errorf("DiscoverAssetRoot() = %q, want empty", got)
		}
	})

	t.Run("no candidate qualifies", func(t *testing.T) {
		if got := DiscoverAssetRoot(absent, partial); got != "" {
			t.Errorf("DiscoverAssetRoot = %q, want empty", got)
		}
	})
}
