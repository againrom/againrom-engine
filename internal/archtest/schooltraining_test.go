package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckSchoolTrainingNamesEachViolation drives the scan with synthetic
// sources, one case per arm.
func TestCheckSchoolTrainingNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a literal curve base", "pkg/game/a.go",
			"package game\nfunc f(n float64) float64 { return math.Pow(1.1, n) }\n", "math.Pow(1.1, n) outside"},
		{"a price beside the screen", "pkg/game/a.go",
			"package game\nfunc f(n float64) int { return int(math.Pow(statBase, n) * 200) }\n", "times 200"},
		{"an experience from pow11", "pkg/data/a.go",
			"package data\nfunc f(n int32) float64 { return (pow11(n) - 1) * 1000 }\n", "times 1000"},
		{"a scale on the left", "pkg/data/a.go",
			"package data\nfunc f(n int32) float64 { return 200 * pow11(n) }\n", "times 200"},
		{"a purchase threshold", "pkg/game/a.go",
			"package game\nfunc f(n int32) int32 { return data.SkillXPFor(n) + 1 }\n", "plus 1"},
		{"a purchase threshold on a rules value", "pkg/mapload/a.go",
			"package mapload\nfunc f(r rules.Rules, n int32) int32 { return 1 + r.SkillXP(n) }\n", "plus 1"},
		{"the rule itself", "pkg/rules/school.go",
			"package rules\nfunc f(n float64) float64 { return (math.Pow(1.1, n) - 1) * 1000 }\n", ""},
		{"a stat power without a school scale", "pkg/data/a.go",
			"package data\nfunc f(n int32) float64 { return pow11(n) / damageDivisor }\n", ""},
		{"another curve times 200", "pkg/game/a.go",
			"package game\nfunc f(n float64) float64 { return math.Pow(1.2, n) * 200 }\n", ""},
		{"a threshold without the step", "pkg/game/a.go",
			"package game\nfunc f(n int32) int32 { return data.SkillXPFor(n) + 2 }\n", ""},
		{"a source outside pkg", "cmd/a/a.go",
			"package main\nfunc f(n float64) float64 { return math.Pow(1.1, n) * 200 }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckSchoolTraining(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckSchoolTraining = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckSchoolTraining = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckSchoolTraining(nil); len(vs) != 1 {
		t.Fatalf("CheckSchoolTraining(nil) = %v, want one violation", vs)
	}
}

// TestSchoolTrainingHasOneRule is the live-tree half.
func TestSchoolTrainingHasOneRule(t *testing.T) {
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
	if _, ok := files[schoolTrainingRuleFile]; !ok {
		t.Fatalf("%s is missing; the scan guards nothing", schoolTrainingRuleFile)
	}
	for _, v := range CheckSchoolTraining(files) {
		t.Errorf("%s", v)
	}
}
