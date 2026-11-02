package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckHeroBandNamesEachCopy plants each shape a second hero-band
// binder takes, and each site the binder keeps.
func TestCheckHeroBandNamesEachCopy(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"an edition test", "pkg/game/a.go",
			"package game\nfunc f(e E) bool { return e.BindsByIdentity() }\n", "edition test"},
		{"the edition test declared again", "pkg/base/a.go",
			"package base\nfunc (e Edition) BindsByIdentity() bool { return true }\n", "edition test"},
		{"a zero hero identity", "pkg/mapload/a.go",
			"package mapload\nfunc f(e E) bool { return e.HeroIdentity != 0 }\n", "hero identity of zero"},
		{"a zero hero identity, reversed", "pkg/game/a.go",
			"package game\nfunc f(e E) bool { return 0 == e.HeroIdentity }\n", "hero identity of zero"},
		{"the source row read by a caller", "pkg/game/a.go",
			"package game\nfunc f(t T) bool { return t.Edition.HeroBand == 1 }\n", "identity source is read"},
		{"the source row read in the binder's file, outside the binder", heroBandBinder,
			"package mapload\nfunc heroBand(t T) S { return sources[t.Edition.HeroBand] }\nfunc g(t T) bool { return t.Edition.HeroBand == 0 }\nfunc (roleBand) bind(n N) { n.ResolveHero(0, p, r) }\n", "identity source is read"},
		{"a source named by a caller", "pkg/game/a.go",
			"package game\nfunc f(e E) bool { return e.Band() == base.HeroBandIdentity }\n", "identity source is named"},
		{"the role test run by a caller", "pkg/game/a.go",
			"package game\nfunc f(n N) { n.ResolveHero(0, p, r) }\n", "role test is run"},
		{"the role test as a method value", "pkg/mapload/a.go",
			"package mapload\nvar f = npc.ResolveHero\n", "role test is run"},
		{"a hero ordinal tested by a caller", "pkg/game/a.go",
			"package game\nfunc f(n N) bool { return n.HeroOrdinalMatches(1, p, a) }\n", "hero ordinal is tested"},
		{"the binder's sites", heroBandBinder,
			"package mapload\nvar heroBandSources = [...]S{base.HeroBandRole: roleBand{}, base.HeroBandIdentity: identityBand{}}\nfunc heroBand(t T) S { return heroBandSources[t.Edition.HeroBand] }\nfunc (roleBand) bind(n N) { n.ResolveHero(0, p, r) }\nfunc TableEdition(t *Table) *base.Edition { first := base.Game(\"\").Edition(); return &first }\n", ""},
		{"a table fallback of its own", "pkg/game/a.go",
			"package game\nfunc f(t *T) base.Edition { if t.Edition == nil { return base.Game(\"\").Edition() }\n return *t.Edition }\n", "mapload.TableEdition"},
		{"a table fallback by the first game's name", "pkg/mapload/a.go",
			"package mapload\nvar first = base.GameROM1.Edition()\n", "mapload.TableEdition"},
		{"the first game's cutscene bank", "pkg/game/cutscene.go",
			"package game\nfunc OpenCutscenes() B { return open(base.Game(\"\").Edition().Cutscenes) }\n", ""},
		{"another game's edition", "pkg/game/a.go",
			"package game\nvar second = base.GameROM2.Edition()\n", ""},
		{"the binder without its role site", heroBandBinder,
			"package mapload\nfunc heroBand(t T) S { return sources[t.Edition.HeroBand] }\n", "ResolveHero site is gone"},
		{"the edition rows", "pkg/base/a.go",
			"package base\nconst (\n HeroBandRole HeroBand = iota\n HeroBandIdentity\n)\nvar first = Edition{HeroBand: HeroBandRole}\n", ""},
		{"ResolveHero's own test", "pkg/data/heroordinal.go",
			"package data\nfunc (n *NPCDefs) ResolveHero(k int) bool { return n.HeroOrdinalMatches(k, p, a) }\n", ""},
		{"a test file", "pkg/game/a_test.go",
			"package game\nfunc f(n N) { n.ResolveHero(0, p, r) }\n", ""},
		{"outside pkg", "cmd/x/a.go",
			"package main\nfunc f(n N) { n.ResolveHero(0, p, r) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckHeroBand(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckHeroBand = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file) {
				t.Fatalf("CheckHeroBand = %v, want one in %s naming %q", vs, tc.file, tc.want)
			}
		})
	}
	if vs := CheckHeroBand(nil); len(vs) != 1 {
		t.Fatalf("CheckHeroBand(nil) = %v, want one violation", vs)
	}
}

// TestHeroBandHasOneBinder is the live-tree half.
func TestHeroBandHasOneBinder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[heroBandBinder]; !ok {
		t.Fatalf("the binder %s is not among the scanned sources", heroBandBinder)
	}
	for _, v := range CheckHeroBand(files) {
		t.Errorf("%s", v)
	}
}
