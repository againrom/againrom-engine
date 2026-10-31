package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckLifeStateNamesRawHealthTests(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"the query", "pkg/sim/world.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP < 0 }\n", ""},
		{"a sim site", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", "raw .HP comparison"},
		{"a game site", "pkg/game/a.go",
			"package game\nfunc f(e sim.Entity) bool { return e.HP > 0 }\n", "raw .HP comparison"},
		{"zero on the left", "pkg/game/a.go",
			"package game\nfunc f(e sim.Entity) bool { return 0 >= e.HP }\n", "raw .HP comparison"},
		{"a nested selector", "pkg/game/a.go",
			"package game\nfunc f(m *M) bool { return m.unit.HP >= 0 }\n", "raw .HP comparison"},
		{"equality is not an ordering", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP == 0 }\n", ""},
		{"another bound", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP > decayBonesHP }\n", ""},
		{"the query's caller", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.InLife(LifeDecaying) }\n", ""},
		{"a test", "pkg/sim/a_test.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", ""},
		{"another package", "pkg/render/a.go",
			"package render\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckLifeState(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckLifeState = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckLifeState = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

func TestLifeStateHasOneQuery(t *testing.T) {
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
	if !strings.Contains(files["pkg/sim/world.go"], "func (s LifeState) Holds(") {
		t.Fatal("pkg/sim/world.go holds no LifeState query; the scan guards nothing")
	}
	for _, v := range CheckLifeState(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
