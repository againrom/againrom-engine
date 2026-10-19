package installpath

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// installNames is a first-game install as its files are named: the five
// required archives, a loose map, the music archive in its folder, a data
// file and the sound archive.
var installNames = []string{
	"main.res", "graphics.res", "scenario.res", "world.res", "movies.res",
	"kids.alm", "allods/music.res", "famehall.dat", "sfx.res",
}

// spelled builds a case-sensitive install whose every name is spelt by
// spell; each file's bytes are its requested name, so a read proves which
// file answered.
func spelled(spell func(string) string) fstest.MapFS {
	m := fstest.MapFS{}
	for _, n := range installNames {
		m[spell(n)] = &fstest.MapFile{Data: []byte(n)}
	}
	return m
}

// mixed spells each segment differently by its position, so one install
// holds lower, upper and capitalised names side by side.
func mixed(n string) string {
	parts := strings.Split(n, "/")
	for i, p := range parts {
		switch (len(p) + i) % 3 {
		case 0:
			parts[i] = strings.ToUpper(p)
		case 1:
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "/")
}

func TestResolverOpensEveryNameWhateverItsCase(t *testing.T) {
	for _, tc := range []struct {
		label string
		spell func(string) string
	}{
		{"lower", strings.ToLower},
		{"upper", strings.ToUpper},
		{"mixed", mixed},
	} {
		t.Run(tc.label, func(t *testing.T) {
			fsys := spelled(tc.spell)
			r := NewFS(fsys)
			for _, n := range installNames {
				b, err := r.ReadFile(n)
				if err != nil {
					t.Fatalf("ReadFile(%q): %v", n, err)
				}
				if string(b) != n {
					t.Fatalf("ReadFile(%q) read %q", n, b)
				}
				// The same name asked for in another case resolves to the same file.
				if b, err := r.ReadFile(strings.ToUpper(n)); err != nil || string(b) != n {
					t.Fatalf("ReadFile(%q) = %q, %v", strings.ToUpper(n), b, err)
				}
				e, err := r.Resolve(n)
				if err != nil || e.Rel != tc.spell(n) || e.Dir {
					t.Fatalf("Resolve(%q) = %+v, %v; want %q", n, e, err, tc.spell(n))
				}
			}
		})
	}
}

// TestExactNameFailsOnCaseSensitiveFS is the loss control: on the same file
// system, opening the requested name as it stands fails wherever the disk
// spells it otherwise.
func TestExactNameFailsOnCaseSensitiveFS(t *testing.T) {
	fsys := spelled(strings.ToUpper)
	failed := 0
	for _, n := range installNames {
		if _, err := fsys.Open(n); errors.Is(err, fs.ErrNotExist) {
			failed++
		}
	}
	if failed != len(installNames) {
		t.Fatalf("exact-name opens failed for %d of %d names; the control proves nothing", failed, len(installNames))
	}
}

func TestExactSpellingWinsOverFoldedOne(t *testing.T) {
	fsys := fstest.MapFS{
		"graphics.res": {Data: []byte("lower")},
		"GRAPHICS.RES": {Data: []byte("upper")},
	}
	r := NewFS(fsys)
	for name, want := range map[string]string{"graphics.res": "lower", "GRAPHICS.RES": "upper"} {
		if b, err := r.ReadFile(name); err != nil || string(b) != want {
			t.Fatalf("ReadFile(%q) = %q, %v; want %q", name, b, err, want)
		}
	}
}

func TestNamesDifferingOnlyByCaseAreReported(t *testing.T) {
	fsys := fstest.MapFS{
		"GRAPHICS.RES": {Data: []byte("upper")},
		"Graphics.res": {Data: []byte("capital")},
	}
	r := NewFS(fsys)
	_, err := r.ReadFile("graphics.res")
	if !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ReadFile = %v, want an ambiguity", err)
	}
	for _, n := range []string{`"GRAPHICS.RES"`, `"Graphics.res"`} {
		if !strings.Contains(err.Error(), n) {
			t.Fatalf("%v does not name %s", err, n)
		}
	}
	if _, err := r.Path("graphics.res"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Path = %v, want an ambiguity", err)
	}
	// A directory on the way is resolved the same way.
	fsys = fstest.MapFS{"Allods/a.res": {}, "ALLODS/a.res": {}}
	if _, err := NewFS(fsys).Resolve("allods/a.res"); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("Resolve through two directories = %v, want an ambiguity", err)
	}
}

func TestAbsentNameKeepsRequestedSpelling(t *testing.T) {
	r := NewFS(fstest.MapFS{"main.res": {}})
	if _, err := r.Resolve("sfx.res"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Resolve(absent) = %v", err)
	}
	if name, err := r.Name("Sfx.res"); err != nil || name != "Sfx.res" {
		t.Fatalf("Name(absent) = %q, %v", name, err)
	}
	for _, bad := range []string{"", "../main.res", "a//b", "./main.res"} {
		if _, err := r.Name(bad); !errors.Is(err, fs.ErrInvalid) {
			t.Fatalf("Name(%q) = %v, want invalid", bad, err)
		}
	}
}

func TestMissTakesAFreshListing(t *testing.T) {
	fsys := fstest.MapFS{"main.res": {}}
	r := NewFS(fsys)
	if _, err := r.Resolve("main.res"); err != nil {
		t.Fatal(err)
	}
	fsys["SFX.RES"] = &fstest.MapFile{Data: []byte("x")}
	if b, err := r.ReadFile("sfx.res"); err != nil || string(b) != "x" {
		t.Fatalf("ReadFile after the file was added = %q, %v", b, err)
	}
}

func TestHostPathUnderRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Allods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Allods", "VIDEO4.RES"), []byte("v"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := New(dir)
	got, err := r.Path("allods/video4.res")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "Allods", "VIDEO4.RES"); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
	if got, err := r.Path("absent.res"); err != nil || got != filepath.Join(dir, "absent.res") {
		t.Fatalf("Path(absent) = %q, %v", got, err)
	}
	if b, err := r.ReadFile("ALLODS/video4.res"); err != nil || string(b) != "v" {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
}
