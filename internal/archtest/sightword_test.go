package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckSightWordBuilderNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a game call through data", "pkg/game/a.go",
			"package game\nfunc f() int32 { return data.SightSubCells(1, 2) }\n", "SightSubCells outside"},
		{"a bare call", "pkg/ui/a.go",
			"package ui\nfunc f() int32 { return SightSubCells(1, 2) }\n", "SightSubCells outside"},
		{"the rule", "pkg/rules/sight.go",
			"package rules\nfunc f() int32 { return SightSubCells(1, 2) }\n", ""},
		{"the data alias", "pkg/data/recompute.go",
			"package data\nfunc f() int32 { return rules.SightSubCells(1, 2) }\n", ""},
		{"the sim reader", "pkg/sim/sight.go",
			"package sim\nfunc f() int32 { return rules.SightSubCells(1, 2) }\n", ""},
		{"the builder's caller", "pkg/game/a.go",
			"package game\nfunc f() { rules.SightWordIn(1, 2, 3) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckSightWordBuilder(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckSightWordBuilder = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckSightWordBuilder = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckSightWordBuilder(nil); len(vs) != 1 {
		t.Fatalf("CheckSightWordBuilder(nil) = %v, want one violation", vs)
	}
}

func TestSightWordHasOneBuilder(t *testing.T) {
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
	if _, ok := files["pkg/sim/sight.go"]; !ok {
		t.Fatal("pkg/sim/sight.go is missing; the scan guards nothing")
	}
	for _, v := range CheckSightWordBuilder(files) {
		t.Errorf("%s", v)
	}
}
