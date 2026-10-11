package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckHeadingLawsNamesEachViolation drives the scan with synthetic
// sources, one case per arm.
func TestCheckHeadingLawsNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a converted byte", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(i int) bool { return w.turnTo(i, heading(64)) }\n", "conversion to heading"},
		{"a second branch tree", "pkg/sim/a.go",
			"package sim\nfunc f(dx, dy int64) uint8 { return headingOf(dx, dy) }\n", "headingOf outside"},
		{"a second sign table", "pkg/sim/a.go",
			"package sim\nfunc f(dx, dy int32) uint8 { h, _ := facingToward(dx, dy); return h }\n", "facingToward outside"},
		{"a direct request", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(i int) bool { return w.entities[i].requestFacing(32) }\n", "requestFacing outside"},
		{"the rule file", "pkg/sim/heading.go",
			"package sim\nfunc (w *World) f(i int) bool { return w.turnTo(i, heading(headingOf(1, 2))) }\n", ""},
		{"the turn core", "pkg/sim/facing.go",
			"package sim\nfunc (w *World) f(i int) bool { return w.entities[i].requestFacing(32) }\n", ""},
		{"a law's caller", "pkg/sim/a.go",
			"package sim\nfunc (w *World) f(i, j int) bool { return w.turnToActor(i, j) }\n", ""},
		{"another package", "pkg/game/a.go",
			"package game\nfunc f() uint8 { return heading(1) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckHeadingLaws(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckHeadingLaws = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckHeadingLaws = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckHeadingLaws(nil); len(vs) != 1 {
		t.Fatalf("CheckHeadingLaws(nil) = %v, want one violation", vs)
	}
}

// TestHeadingLawsHaveOneBuilderEach is the live-tree half.
func TestHeadingLawsHaveOneBuilderEach(t *testing.T) {
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
	if _, ok := files[headingRuleFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", headingRuleFile)
	}
	for _, v := range CheckHeadingLaws(files) {
		t.Errorf("%s", v)
	}
}
