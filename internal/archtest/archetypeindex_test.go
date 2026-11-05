package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckArchetypeIndexNamesEachCopy plants each shape a second index
// function takes.
func TestCheckArchetypeIndexNamesEachCopy(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"female plus one", "pkg/data/a.go",
			"package data\nfunc f(female bool) int { i := 0\n if female { i++ }\n return i }\n", "herokind.Of"},
		{"mage plus two", "pkg/game/a.go",
			"package game\nfunc f(m T) int { i := 0\n if m.Mage { i += 2 }\n return i }\n", "herokind.Of"},
		{"a method read, or one", "pkg/sim/a.go",
			"package sim\nfunc f(d D) int { i := 0\n if d.Female() { i |= 1 }\n return i }\n", "herokind.Of"},
		{"a token read", "pkg/mapload/a.go",
			"package mapload\nfunc f(t T) int { i := 0\n if t.Has(NPCTokenMage) { i += 2 }\n return i }\n", "herokind.Of"},
		{"a hero field read", "pkg/game/a.go",
			"package game\nfunc f(a A) int { by := 0\n if a.HeroMage { by += 2 }\n return by }\n", "herokind.Of"},
		{"UI mage shift one", "pkg/ui/a.go",
			"package ui\nfunc f(mage bool) int { h := 1\n if !mage { h <<= 1 }\n return h }\n", "herokind.Of"},
		{"UI female shift two", "pkg/ui/a.go",
			"package ui\nfunc f(female bool) int { h := 1\n if !female { h <<= 2 }\n return h }\n", "herokind.Of"},
		{"assigned shift", "pkg/game/a.go",
			"package game\nfunc f(mage bool) int { h := 1\n if mage { h = h << 1 }\n return h }\n", "herokind.Of"},
		{"cell sex argument", "pkg/game/a.go",
			"package game\nfunc f(h int) { plantedHero(h < 2, false) }\n", "herokind.AudienceOrder"},
		{"cell class argument", "pkg/game/a.go",
			"package game\nfunc f(h int) { plantedHero(false, h%2 == 0) }\n", "herokind.AudienceOrder"},
		{"both cell arguments inside another call", "pkg/game/a.go",
			"package game\nfunc f(h int) { cast(plantedHero(h < 2, h%2 == 0)) }\n", "herokind.AudienceOrder"},
		{"UI parenthesized cell argument", "pkg/ui/a.go",
			"package ui\nfunc f(h int) { plantedHero(!((h % 2) != 0)) }\n", "herokind.AudienceOrder"},
		{"order lookup", "pkg/ui/a.go",
			"package ui\nfunc f(h int) { k := herokind.AudienceOrder[h]; plantedHero(k.Female(), k.Mage()) }\n", ""},
		{"cell split by numeric positions", "pkg/game/a.go",
			"package game\nfunc f(got [4]int) { ui.HeroCells(got[0] != got[2] || got[1] != got[3], got[0] != got[1] || got[2] != got[3]) }\n", "herokind.AudienceOrder"},
		{"cell split by the declared order", "pkg/game/a.go",
			"package game\nfunc f(got [4]int) { ui.HeroCells(got[herokind.FemaleMage.In(herokind.AudienceOrder)] != got[herokind.MaleMage.In(herokind.AudienceOrder)], false) }\n", ""},
		{"cell sex in a field", "pkg/game/a.go",
			"package game\nfunc f(h int) A { return A{HeroFemale: h < 2} }\n", "herokind.AudienceOrder"},
		{"cell class in a field", "pkg/game/a.go",
			"package game\nfunc f(h int) A { return A{HeroMage: h%2 == 0} }\n", "herokind.AudienceOrder"},
		{"a non-axis shift", "pkg/ui/a.go",
			"package ui\nfunc f(ok bool) int { h := 1\n if ok { h <<= 2 }\n return h }\n", ""},
		{"an integer column", "pkg/ui/a.go",
			"package ui\nfunc f(h int) { column(h % 2) }\n", ""},
		{"a loop counter under the condition", "pkg/game/a.go",
			"package game\nfunc f(mage bool) { if mage { for id := 1; id < 3; id++ { g(id) } }\n}\n", ""},
		{"herokind itself", "pkg/herokind/a.go",
			"package herokind\nfunc f(female bool) int { i := 0\n if female { i++ }\n return i }\n", ""},
		{"a condition that is not the axis", "pkg/game/a.go",
			"package game\nfunc f(ok bool) int { i := 0\n if ok { i += 2 }\n return i }\n", ""},
		{"a body that is no index", "pkg/game/a.go",
			"package game\nfunc f(mage bool) string { s := \"\"\n if mage { s = \"x\" }\n return s }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckArchetypeIndex(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("CheckArchetypeIndex = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) || !strings.HasPrefix(vs[0].From, tc.file+":") {
				t.Fatalf("CheckArchetypeIndex = %v, want one naming %q", vs, tc.want)
			}
		})
	}
	if vs := CheckArchetypeIndex(nil); len(vs) != 1 {
		t.Fatalf("CheckArchetypeIndex(nil) = %v, want one violation", vs)
	}
}

// TestHeroArchetypeIndexHasOneBuilder is the live-tree half.
func TestHeroArchetypeIndexHasOneBuilder(t *testing.T) {
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
	for _, v := range CheckArchetypeIndex(files) {
		t.Errorf("%s", v)
	}
}
