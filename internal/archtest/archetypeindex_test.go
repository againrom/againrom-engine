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
