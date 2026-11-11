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
	got := loads(t, mods, "rom1-en")
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
	got := loads(t, mods, "rom1-ru")
	if want := []string{"z-base", "a-common", "m-mid"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("got %v, want %v", ids(got), want)
	}
}

func TestOrderIgnoresTheOrderTheModsWereNamedIn(t *testing.T) {
	a := []Entry{entry("a", "1", nil), entry("b", "1", nil), entry("c", "1", nil)}
	b := []Entry{a[2], a[0], a[1]}
	x := loads(t, a, "rom1-en")
	y := loads(t, b, "rom1-en")
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
		err := Order(c.mods, c.base).Err()
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
}

// loads is the load order of a set Order accepts.
func loads(t *testing.T, mods []Entry, base string) []Entry {
	t.Helper()
	o := Order(mods, base)
	if err := o.Err(); err != nil {
		t.Fatal(err)
	}
	return o.Mods
}

// texts lists the game's text of each refusal.
func texts(rs []Refusal) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Error())
	}
	return out
}

// A set with several refusals names each one on its mod as data, keeps the
// first in the order the launch printed it, and is still put in load order.
func TestOrderNamesEveryRefusalAndKeepsTheFirstText(t *testing.T) {
	mods := []Entry{
		entry("api", "1", func(m *Manifest) { m.API = 2; m.AppliesTo = []string{"rom2"} }),
		entry("clash", "1", func(m *Manifest) { m.Conflicts = []string{"core", "absent"} }),
		entry("core", "1.1", nil),
		entry("loop-a", "1", func(m *Manifest) { m.LoadAfter = []string{"loop-b"} }),
		entry("loop-b", "1", func(m *Manifest) { m.Requires = []string{"loop-a"} }),
		entry("needs", "1", func(m *Manifest) { m.Requires = []string{"core>=1.2", "ghost", "core>=x"} }),
		entry("tail", "1", func(m *Manifest) { m.LoadAfter = []string{"loop-a"} }),
	}
	o := Order(mods, "rom1-en")
	want := []string{
		`mod "api": api 2 is not supported (this game runs api 1)`,
		`mod "api": applies-to rom2 does not include the active base rom1-en`,
		`mod "clash" conflicts with the enabled mod "core"`,
		`mod "needs" requires "core>=1.2" but "core" 1.1 is enabled`,
		`mod "needs" requires "ghost", which is not enabled`,
		`mod "needs": requires: "core>=x": version "x" is not dotted numbers`,
		"mods loop-a, loop-b, tail require or load after each other in a cycle",
	}
	if got := texts(o.Refusals); !reflect.DeepEqual(got, want) {
		t.Fatalf("refusals\n%q\nwant\n%q", got, want)
	}
	if o.Err() == nil || o.Err().Error() != want[0] {
		t.Fatalf("first refusal %v", o.Err())
	}
	kinds := []RefusalKind{RefusedAPI, RefusedBase, RefusedConflict, RefusedRequirementVersion, RefusedRequirementMissing, RefusedRequirementUnreadable, RefusedCycle}
	for i, r := range o.Refusals {
		if r.Kind != kinds[i] {
			t.Errorf("refusal %d kind %d, want %d", i, r.Kind, kinds[i])
		}
	}
	if r := o.Refusals[3]; r.Mod != "needs" || r.Other != "core" || r.Requirement != "core>=1.2" || r.Version != "1.1" {
		t.Errorf("version refusal data %+v", r)
	}
	if r := o.Refusals[1]; r.Base != "rom1-en" || !reflect.DeepEqual(r.AppliesTo, []string{"rom2"}) || o.Refusals[0].API != 2 {
		t.Errorf("fit refusal data %+v %+v", o.Refusals[0], r)
	}
	if r := o.Refusals[2]; r.Other != "core" {
		t.Errorf("conflict names %q", r.Other)
	}
	for id, n := range map[string]int{"api": 2, "clash": 1, "core": 0, "needs": 3, "loop-a": 1, "loop-b": 1, "tail": 1} {
		if got := len(o.Of(id)); got != n {
			t.Errorf("%s stands under %d refusals, want %d", id, got, n)
		}
	}
	if want := []string{"clash", "core", "needs", "api", "loop-a", "loop-b", "tail"}; !reflect.DeepEqual(ids(o.Mods), want) {
		t.Fatalf("order under refusals %v, want %v", ids(o.Mods), want)
	}
}

// Fit is the part of Order a mod raises alone: the api and the base, never
// what other mods decide.
func TestFitIsTheModsOwnRefusalsOnTheBase(t *testing.T) {
	alone := entry("a", "1", func(m *Manifest) { m.AppliesTo = []string{"rom1"}; m.Requires = []string{"ghost"} }).Manifest
	if got := Fit(alone, "rom1-ru"); len(got) != 0 {
		t.Fatalf("a first-game mod on rom1-ru: %v", got)
	}
	got := Fit(alone, "rom2-en")
	if len(got) != 1 || got[0].Kind != RefusedBase || got[0].Mod != "a" || got[0].Error() != texts(Order([]Entry{{Manifest: alone}}, "rom2-en").Refusals)[0] {
		t.Fatalf("a first-game mod on rom2-en: %v", got)
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
	if dd, _ := ContentDigest(b); dd != da {
		t.Fatal("a file outside the tree moved the digest")
	}
	writeTree(t, b, map[string]string{ItemsFile: "# empty\n"})
	if dd, _ := ContentDigest(b); dd == da {
		t.Fatal("an added data kind kept the digest")
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
