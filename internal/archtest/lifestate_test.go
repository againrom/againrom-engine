package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckLifeStateNamesRawHealthTests(t *testing.T) {
	const want = "compared with a constant"
	cases := []struct {
		name, file, src, want string
	}{
		{"the query", "pkg/sim/world.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP < 0 }\n", ""},
		{"a sim site", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", want},
		{"a game site", "pkg/game/a.go",
			"package game\nfunc f(e sim.Entity) bool { return e.HP > 0 }\n", want},
		{"a ui site", "pkg/ui/a.go",
			"package ui\nfunc f(e Entity) bool { return e.HP > 0 }\n", want},
		{"zero on the left", "pkg/game/a.go",
			"package game\nfunc f(e sim.Entity) bool { return 0 >= e.HP }\n", want},
		{"a nested selector", "pkg/game/a.go",
			"package game\nfunc f(m *M) bool { return m.unit.HP >= 0 }\n", want},
		{"a negated literal", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP <= -10 }\n", want},
		{"a literal one", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP < 1 }\n", want},
		{"a const identifier", "pkg/sim/a.go",
			"package sim\nconst floor = -600\nfunc f(e *Entity) bool { return e.HP < floor }\n", want},
		{"a named threshold", "pkg/sim/a.go",
			"package sim\nconst decayBonesHP = -10\nfunc f(e *Entity) bool { return e.HP > decayBonesHP }\n", want},
		{"constant arithmetic", "pkg/sim/a.go",
			"package sim\nconst floor = 600\nfunc f(e *Entity) bool { return e.HP < -floor-1 }\n", want},
		{"a converted health", "pkg/game/a.go",
			"package game\nfunc f(r Record) bool { return int16(r.HP) <= 0 }\n", want},
		{"a doubly converted health", "pkg/game/a.go",
			"package game\nfunc f(r Record) bool { return int(int16(r.HP)) > -10 }\n", want},
		{"a converted constant", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return int64(e.HP) < int64(-5) }\n", want},
		{"a local copied from health", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { x := e.HP; return x >= 0 }\n", want},
		{"a local copied through a conversion", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { var x = int64(e.HP); y := x; return y > 5 }\n", want},
		{"a local copied in a pair", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { a, b := e.HP, e.Decay; _ = b; return a >= 0 }\n", want},
		{"a lower-case health field", "pkg/ui/a.go",
			"package ui\nfunc f(m memory) bool { return m.hp < 0 }\n", want},
		{"a health named identifier", "pkg/game/a.go",
			"package game\nfunc f(health int32) bool { return int16(health) > -10 }\n", want},
		{"the saved stats word", "pkg/game/a.go",
			"package game\nfunc f(s Source) bool { return int16(s.Stats[8]) <= 0 }\n", want},
		{"a switch case", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) int {\n\tswitch {\n\tcase e.HP < -600:\n\t\treturn 1\n\t}\n\treturn 0\n}\n", want},
		{"a helper wrapping the comparison", "pkg/sim/a.go",
			"package sim\nfunc below(hp int32) bool { return hp < -10 }\n", want},
		{"two health values", "pkg/sim/a.go",
			"package sim\nfunc f(a, b *Entity) bool { return a.HP < b.HP }\n", ""},
		{"a health and a variable", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity, w int32) bool { return e.HP <= w }\n", ""},
		{"a maximum", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.MaxHP <= 0 }\n", ""},
		{"a non-health local", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { x := e.Decay; return x >= 0 }\n", ""},
		{"equality is not an ordering", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP == 0 }\n", ""},
		{"the query's caller", "pkg/sim/a.go",
			"package sim\nfunc f(e *Entity) bool { return e.InLife(LifeDecaying) }\n", ""},
		{"a test", "pkg/sim/a_test.go",
			"package sim\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", ""},
		{"another package", "pkg/render/a.go",
			"package render\nfunc f(e *Entity) bool { return e.HP <= 0 }\n", ""},
		{"a kept range check", "pkg/game/worldsave.go",
			"package game\nfunc projectDeadActors(t T) bool { return t.HP < -32768 || t.HP > 32767 }\n", ""},
		{"a kept range check in another function", "pkg/game/worldsave.go",
			"package game\nfunc other(t T) bool { return t.HP < -32768 }\n", want},
		{"a kept check that is gone", "pkg/ui/sound.go",
			"package ui\nfunc stepSound() {}\n", "no longer exists"},
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
