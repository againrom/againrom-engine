package mod

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func entry(id, version string, mut func(*Manifest)) Entry {
	m := Manifest{ID: id, Title: id, Version: version, AppliesTo: []string{"common"}, API: 1}
	if mut != nil {
		mut(&m)
	}
	return Entry{Folder: id, Dir: "/mods/" + id, Manifest: m}
}

func ids(es []Entry) []string {
	var out []string
	for _, e := range es {
		out = append(out, e.Manifest.ID)
	}
	return out
}

func TestOrderPutsCommonModsFirstThenTheBaseModsByID(t *testing.T) {
	mods := []Entry{
		entry("zeta", "1", func(m *Manifest) { m.AppliesTo = []string{"rom1-en"} }),
		entry("beta", "1", nil),
		entry("alpha", "1", func(m *Manifest) { m.AppliesTo = []string{"rom1"} }),
		entry("omega", "1", nil),
	}
	got, err := Order(mods, "rom1-en")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"beta", "omega", "alpha", "zeta"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("got %v, want %v", ids(got), want)
	}
}

func TestOrderFollowsRequiresAndLoadAfterOverTheGroups(t *testing.T) {
	mods := []Entry{
		entry("a-common", "1", func(m *Manifest) { m.LoadAfter = []string{"z-base", "absent"} }),
		entry("z-base", "2.1", func(m *Manifest) { m.AppliesTo = []string{"rom1-ru"} }),
		entry("m-mid", "1", func(m *Manifest) { m.Requires = []string{"z-base>=2.0"} }),
	}
	got, err := Order(mods, "rom1-ru")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"z-base", "a-common", "m-mid"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("got %v, want %v", ids(got), want)
	}
}

func TestOrderIgnoresTheOrderTheModsWereNamedIn(t *testing.T) {
	a := []Entry{entry("a", "1", nil), entry("b", "1", nil), entry("c", "1", nil)}
	b := []Entry{a[2], a[0], a[1]}
	x, _ := Order(a, "rom1-en")
	y, _ := Order(b, "rom1-en")
	if !reflect.DeepEqual(ids(x), ids(y)) {
		t.Fatalf("%v %v", ids(x), ids(y))
	}
}

func TestOrderRefusalsNameTheMods(t *testing.T) {
	cases := []struct {
		name string
		mods []Entry
		base string
		want string
	}{
		{"api", []Entry{entry("a", "1", func(m *Manifest) { m.API = 2 })}, "rom1-en", `mod "a": api 2 is not supported`},
		{"wrong base", []Entry{entry("a", "1", func(m *Manifest) { m.AppliesTo = []string{"rom1-ru"} })}, "rom1-en", `mod "a": applies-to rom1-ru does not include the active base rom1-en`},
		{"rom2 mod on rom1", []Entry{entry("a", "1", func(m *Manifest) { m.AppliesTo = []string{"rom2"} })}, "rom1-en", `mod "a"`},
		{"conflict", []Entry{entry("a", "1", func(m *Manifest) { m.Conflicts = []string{"b"} }), entry("b", "1", nil)}, "rom1-en", `mod "a" conflicts with the enabled mod "b"`},
		{"missing requirement", []Entry{entry("a", "1", func(m *Manifest) { m.Requires = []string{"core"} })}, "rom1-en", `mod "a" requires "core", which is not enabled`},
		{"old requirement", []Entry{entry("a", "1", func(m *Manifest) { m.Requires = []string{"core>=1.2"} }), entry("core", "1.1", nil)}, "rom1-en", `mod "a" requires "core>=1.2" but "core" 1.1 is enabled`},
		{"bad requirement", []Entry{entry("a", "1", func(m *Manifest) { m.Requires = []string{"core>=x"} })}, "rom1-en", `mod "a": requires`},
		{"cycle", []Entry{entry("a", "1", func(m *Manifest) { m.LoadAfter = []string{"b"} }), entry("b", "1", func(m *Manifest) { m.LoadAfter = []string{"a"} })}, "rom1-en", "a, b require or load after each other in a cycle"},
	}
	for _, c := range cases {
		_, err := Order(c.mods, c.base)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
}

func TestAppliesAndVersionComparison(t *testing.T) {
	for _, c := range []struct {
		list []string
		base string
		want bool
	}{
		{[]string{"common"}, "rom2", true},
		{[]string{"rom1"}, "rom1-demo", true},
		{[]string{"rom1"}, "rom2", false},
		{[]string{"rom1-en"}, "rom1-ru", false},
		{[]string{"rom2", "rom1-ru"}, "rom1-ru", true},
	} {
		if Applies(c.list, c.base) != c.want {
			t.Errorf("%v on %s", c.list, c.base)
		}
	}
	for _, c := range []struct {
		a, b string
		want int
	}{{"1.0", "1", 0}, {"1.2.0", "1.10", -1}, {"2", "1.9.9", 1}} {
		if got, err := CompareVersions(c.a, c.b); err != nil || got != c.want {
			t.Errorf("%s %s: %d %v", c.a, c.b, got, err)
		}
	}
	if _, err := CompareVersions("1.a", "1"); err == nil {
		t.Error("a non-numeric version compared")
	}
}

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestContentDigestCoversFilesAndIgnoresReadmeAndLineEnds(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeTree(t, a, map[string]string{"mod.toml": "id = \"x\"\n", "main.star": "def init(g, s):\n  pass\n", "scripts/a.star": "x = 1\n", "README.md": "one", ".hidden": "1"})
	writeTree(t, b, map[string]string{"mod.toml": "id = \"x\"\r\n", "main.star": "def init(g, s):\r\n  pass\r\n", "scripts/a.star": "x = 1\n", "README.md": "two"})
	da, err := ContentDigest(a)
	if err != nil {
		t.Fatal(err)
	}
	db, _ := ContentDigest(b)
	if da != db || len(da) != 64 {
		t.Fatalf("%s %s", da, db)
	}
	writeTree(t, b, map[string]string{"scripts/a.star": "x = 2\n"})
	if dc, _ := ContentDigest(b); dc == da {
		t.Fatal("a changed script kept the digest")
	}
	writeTree(t, b, map[string]string{"scripts/a.star": "x = 1\n", "data/new.toml": ""})
	if dd, _ := ContentDigest(b); dd == da {
		t.Fatal("an added file kept the digest")
	}
}

func sampleSet() Set {
	return Set{Base: "rom1-en", Mods: []SetEntry{
		{ID: "a", Version: "1", Digest: "d1", Settings: []SettingValue{{Key: "skill_cap", Value: Value{Kind: KindInt, Int: 150}}}},
		{ID: "b", Version: "2", Digest: "d2"},
	}}
}

func TestSetDigestAndDifferences(t *testing.T) {
	s := sampleSet()
	if s.Digest() != sampleSet().Digest() || len(s.Digest()) != 64 {
		t.Fatal("digest is not a function of the set")
	}
	if got := Differences(s, sampleSet()); len(got) != 0 {
		t.Fatalf("equal sets differ: %v", got)
	}
	other := sampleSet()
	other.Mods[0].Settings[0].Value.Int = 120
	if other.Digest() == s.Digest() {
		t.Fatal("a setting is outside the digest")
	}
	cases := []struct {
		name string
		mut  func(*Set)
		want string
	}{
		{"setting", func(x *Set) { x.Mods[0].Settings[0].Value.Int = 120 }, `mod "a" setting skill_cap is 120, the saved game used 150`},
		{"files", func(x *Set) { x.Mods[1].Digest = "zz" }, `mod "b" has different files`},
		{"version", func(x *Set) { x.Mods[1].Version = "3" }, `mod "b" is version 3, the saved game used 2`},
		{"missing", func(x *Set) { x.Mods = x.Mods[:1] }, `mod "b" 2 is missing`},
		{"extra", func(x *Set) { x.Mods = append(x.Mods, SetEntry{ID: "c", Version: "1"}) }, `mod "c" is active but the saved game did not use it`},
		{"order", func(x *Set) { x.Mods[0], x.Mods[1] = x.Mods[1], x.Mods[0] }, "load in the order b, a"},
		{"base", func(x *Set) { x.Base = "rom1-ru" }, "the base game is rom1-ru"},
	}
	for _, c := range cases {
		have := sampleSet()
		c.mut(&have)
		got := strings.Join(Differences(s, have), "|")
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
