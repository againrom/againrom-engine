package base

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture writes a root holding the five archives, with main.res set to main
// and the extra files empty.
func fixture(t *testing.T, main string, extra ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range RequiredArchives() {
		body := "x"
		if name == MainArchive {
			body = main
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range extra {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func digestOf(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func profilesFor(main string) []Profile {
	return []Profile{
		{ID: "demo", Title: "Demo", Language: "english", Files: []string{"video4.res"},
			Builds: []Build{{Size: int64(len(main)), SHA256: digestOf(main)}},
			Limits: Limits{NoCharacterGeneration: true, FirstMission: 41}},
		{ID: "en", Title: "English", Language: "english"},
		{ID: "ru", Title: "Russian", Language: "russian"},
	}
}

func lang(s string) Language { return func(string) string { return s } }

func TestDetectExactBuild(t *testing.T) {
	root := fixture(t, "demo-main", "VIDEO4.RES")
	m, err := DetectIn(root, profilesFor("demo-main"), lang("english"))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID() != "demo" || !m.Exact || m.Digest != digestOf("demo-main") {
		t.Fatalf("match = %+v, want exact demo", m)
	}
	if m.Profile.Mission() != 41 {
		t.Fatalf("demo mission = %d, want 41", m.Profile.Mission())
	}
}

func TestDetectByLanguageWhenDigestUnknown(t *testing.T) {
	for _, c := range []struct{ lang, id string }{{"english", "en"}, {"russian", "ru"}} {
		root := fixture(t, "other-main")
		m, err := DetectIn(root, profilesFor("demo-main"), lang(c.lang))
		if err != nil {
			t.Fatal(err)
		}
		if m.ID() != c.id || m.Exact {
			t.Fatalf("language %s: match = %+v, want inexact %s", c.lang, m, c.id)
		}
		if m.Profile.Mission() != DefaultFirstMission || m.Profile.Limits.NoCharacterGeneration {
			t.Fatalf("a release profile carries a limit: %+v", m.Profile.Limits)
		}
	}
}

func TestDetectUnknownLanguageIsGeneric(t *testing.T) {
	root := fixture(t, "other-main")
	for _, l := range []Language{lang("language selector 7"), lang(""), nil} {
		m, err := DetectIn(root, profilesFor("demo-main"), l)
		if err != nil {
			t.Fatal(err)
		}
		if m.ID() != ROM1 || m.Exact {
			t.Fatalf("match = %+v, want the generic profile", m)
		}
	}
}

// A demo lookalike without the demo's extra file is not taken for the demo, and
// is not taken for a release either: it is refused and names what is missing.
func TestDetectPartialDemoRefused(t *testing.T) {
	root := fixture(t, "demo-main")
	_, err := DetectIn(root, profilesFor("demo-main"), lang("english"))
	var partial *PartialError
	if !errors.As(err, &partial) {
		t.Fatalf("err = %v, want *PartialError", err)
	}
	if partial.Profile != "demo" || len(partial.Missing) != 1 || partial.Missing[0] != "video4.res" {
		t.Fatalf("partial = %+v", partial)
	}
	if !strings.Contains(err.Error(), "video4.res") || !strings.Contains(err.Error(), "demo") {
		t.Fatalf("message %q does not name the profile and the file", err)
	}
}

func TestDetectMissingArchivesRefused(t *testing.T) {
	root := fixture(t, "demo-main")
	if err := os.Remove(filepath.Join(root, "graphics.res")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "movies.res")); err != nil {
		t.Fatal(err)
	}
	_, err := DetectIn(root, profilesFor("demo-main"), lang("english"))
	var miss *NotInstallError
	if !errors.As(err, &miss) {
		t.Fatalf("err = %v, want *NotInstallError", err)
	}
	if got := strings.Join(miss.Missing, ","); got != "graphics.res,movies.res" {
		t.Fatalf("missing = %s", got)
	}
}

func TestDetectUnreadableRootRefused(t *testing.T) {
	_, err := DetectIn(filepath.Join(t.TempDir(), "absent"), profilesFor("x"), nil)
	var miss *NotInstallError
	if !errors.As(err, &miss) || len(miss.Missing) != len(RequiredArchives()) {
		t.Fatalf("err = %v, want every archive missing", err)
	}
}

// Detection compares file names with case folded, as the install roots differ in
// case, and reads nothing but the directory and the main archive.
func TestDetectFoldsCase(t *testing.T) {
	dir := t.TempDir()
	for _, name := range RequiredArchives() {
		if err := os.WriteFile(filepath.Join(dir, strings.ToUpper(name)), []byte("m"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m, err := DetectIn(dir, profilesFor("demo-main"), lang("russian"))
	if err != nil || m.ID() != "ru" {
		t.Fatalf("match = %+v, err = %v", m, err)
	}
}

func TestRequire(t *testing.T) {
	m := Match{Profile: Profile{ID: ROM1EN, Title: "English"}, Exact: true}
	if err := Require("", m); err != nil {
		t.Fatal(err)
	}
	if err := Require(ROM1EN, m); err != nil {
		t.Fatal(err)
	}
	err := Require(ROM1Demo, m)
	var mm *MismatchError
	if !errors.As(err, &mm) || !strings.Contains(err.Error(), ROM1Demo) || !strings.Contains(err.Error(), ROM1EN) {
		t.Fatalf("err = %v, want a mismatch naming both profiles", err)
	}
}

func TestKnownProfiles(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Profiles {
		if seen[p.ID] || p.ID == "" || p.Title == "" {
			t.Fatalf("profile %+v: empty or repeated id or title", p)
		}
		seen[p.ID] = true
		for _, b := range p.Builds {
			if b.Size <= 0 || len(b.SHA256) != 64 {
				t.Fatalf("profile %s: build %+v is not a size and a SHA-256", p.ID, b)
			}
		}
		if got, ok := Find(p.ID); !ok || got.ID != p.ID {
			t.Fatalf("Find(%s) = %+v, %v", p.ID, got, ok)
		}
	}
	for _, id := range []string{ROM1EN, ROM1RU, ROM1Demo} {
		if !seen[id] {
			t.Fatalf("profile %s is missing", id)
		}
	}
	if p, _ := Find(ROM1Demo); !p.Limits.NoCharacterGeneration || p.Mission() != 41 || len(p.Limits.Notes) == 0 {
		t.Fatalf("the demo profile states no limit: %+v", p.Limits)
	}
	if _, ok := Find("rom3-ru"); ok {
		t.Fatal("an unknown id was found")
	}
}

func TestMatchString(t *testing.T) {
	if got := (Match{}).String(); got != "no base detected" {
		t.Fatalf("zero match = %q", got)
	}
	exact := Match{Profile: Profile{ID: "a", Title: "A"}, Exact: true}.String()
	loose := Match{Profile: Profile{ID: "a", Title: "A"}}.String()
	generic := Match{Profile: unrecognised}.String()
	for _, s := range []string{exact, loose, generic} {
		if !strings.Contains(s, "(") {
			t.Fatalf("%q lacks a build statement", s)
		}
	}
	if exact == loose || loose == generic {
		t.Fatalf("statements are not distinct: %q %q %q", exact, loose, generic)
	}
}

// TestNameable: a -base flag names every profile Find knows except the first
// game's unrecognised build.
func TestNameable(t *testing.T) {
	for _, id := range IDs() {
		if !Nameable(id) {
			t.Errorf("Nameable(%q) = false, want true", id)
		}
	}
	for id, want := range map[string]bool{ROM2: true, ROM1: false, "": false, "rom3": false} {
		if got := Nameable(id); got != want {
			t.Errorf("Nameable(%q) = %v, want %v", id, got, want)
		}
	}
}
