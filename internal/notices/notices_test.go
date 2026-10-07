package notices

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoticesMatchGoMod asserts that THIRD_PARTY_NOTICES.md lists exactly the
// third-party modules declared in go.mod's require directives — no missing and
// no stale entries (spec AC-3). At zero dependencies both sets are empty; the
// test starts failing the moment a require is added without a matching notice,
// or a notice is added for a module that is not required.
func TestNoticesMatchGoMod(t *testing.T) {
	root := repoRoot(t)
	required := requireModules(t, root)
	listed := noticeModules(t, root)

	for m := range required {
		if !listed[m] {
			t.Errorf("module %q is required in go.mod but missing from THIRD_PARTY_NOTICES.md", m)
		}
	}
	for m := range listed {
		if !required[m] {
			t.Errorf("module %q is listed in THIRD_PARTY_NOTICES.md but not required in go.mod (stale)", m)
		}
	}
}

// portedSource names one third-party algorithm ported by hand into this
// tree (no go.mod require line, so TestNoticesMatchGoMod cannot see it).
type portedSource struct {
	path           string // package directory, relative to repo root
	noticeFragment string // substring that must appear in THIRD_PARTY_NOTICES.md
	licenseFile    string // file under LICENSES/ carrying the full license text
}

// portedSources is the registry TestPortedSourceNoticesPresent checks.
// Add an entry here whenever a third-party algorithm is ported by hand
// instead of vendored as a Go module (see THIRD_PARTY_NOTICES.md's own
// "Source-derived (ported) code" section and its regeneration note).
var portedSources = []portedSource{
	{
		path:           "pkg/video/smacker",
		noticeFragment: "libsmacker",
		licenseFile:    "LGPL-2.1.txt",
	},
}

// TestPortedSourceNoticesPresent is the module-less counterpart to
// TestNoticesMatchGoMod: for each entry in portedSources it requires the
// ported package to actually exist, THIRD_PARTY_NOTICES.md to name it, and
// its license text to be present under LICENSES/, so a ported tree cannot
// silently lose its attribution the way a go.mod dependency cannot.
func TestPortedSourceNoticesPresent(t *testing.T) {
	root := repoRoot(t)
	notices, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range portedSources {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(s.path))); err != nil || !info.IsDir() {
			t.Errorf("ported source %q: package directory not found: %v", s.path, err)
		}
		if !strings.Contains(string(notices), s.noticeFragment) {
			t.Errorf("ported source %q: THIRD_PARTY_NOTICES.md does not mention %q", s.path, s.noticeFragment)
		}
		licensePath := filepath.Join(root, "LICENSES", s.licenseFile)
		if info, err := os.Stat(licensePath); err != nil || info.IsDir() {
			t.Errorf("ported source %q: license text not found at %s: %v", s.path, licensePath, err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// requireModules returns the set of module paths declared in go.mod require
// directives (both the block and single-line forms), ignoring comments.
func requireModules(t *testing.T, root string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if s == "" || strings.HasPrefix(s, "//") {
			continue
		}
		if inBlock {
			if s == ")" {
				inBlock = false
				continue
			}
			if m := firstField(s); m != "" {
				set[m] = true
			}
			continue
		}
		switch {
		case s == "require (":
			inBlock = true
		case strings.HasPrefix(s, "require "):
			if m := firstField(strings.TrimPrefix(s, "require ")); m != "" {
				set[m] = true
			}
		}
	}
	return set
}

// noticeModules returns the set of module paths listed in THIRD_PARTY_NOTICES.md
// table rows. A cell counts as a module path only if it contains both "/" and
// "." so header, separator, and placeholder rows are skipped.
func noticeModules(t *testing.T, root string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY_NOTICES.md"))
	if err != nil {
		t.Fatal(err)
	}
	set := map[string]bool{}
	for _, raw := range strings.Split(string(data), "\n") {
		s := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if !strings.HasPrefix(s, "|") {
			continue
		}
		cells := strings.Split(s, "|")
		if len(cells) < 2 {
			continue
		}
		mod := strings.Trim(cells[1], " `*_")
		if strings.Contains(mod, ".") && !strings.ContainsAny(mod, " ") {
			set[mod] = true
		}
	}
	return set
}

func firstField(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
