package archtest

import (
	"os"
	"testing"
)

// TestCheckWordWrapNamesEachShape drives the evaluator with one synthetic
// source per shape it refuses and one per shape it allows.
func TestCheckWordWrapNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src string
		want            int
	}{
		{"a named wrap function", "pkg/ui/a.go", "package ui\nfunc wrapThing(s string) []string { return nil }\n", 1},
		{"a wrap method", "pkg/ui/a.go", "package ui\ntype T struct{}\nfunc (T) saveWrapped(s string) []string { return nil }\n", 1},
		{"Fields", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.Fields(\"a b\")\n", 1},
		{"FieldsFunc", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.FieldsFunc(\"a b\", nil)\n", 1},
		{"Split at a space", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.Split(\"a b\", \" \")\n", 1},
		{"Cut at a space", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _, _, _ = strings.Cut(\"a b\", \" \")\n", 1},
		{"LastIndexByte of a space", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.LastIndexByte(\"a b\", ' ')\n", 1},
		{"Index of a space", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.Index(\"a b\", \" \")\n", 1},
		{"IndexRune of a space", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.IndexRune(\"a b\", ' ')\n", 1},
		{"Split at a newline is not a breaker", "pkg/ui/a.go", "package ui\nimport \"strings\"\nvar _ = strings.Split(\"a\\nb\", \"\\n\")\n", 0},
		{"the width helper hostedWrap", "pkg/ui/a.go", "package ui\nfunc hostedWrap() int { return 0 }\n", 0},
		{"an Unwrap method", "pkg/ui/a.go", "package ui\ntype E struct{}\nfunc (E) Unwrap() error { return nil }\n", 0},
		{"the breaker's own file", "pkg/ui/linebreak.go", "package ui\nimport \"strings\"\nvar _ = strings.Fields(\"a b\")\n", 0},
		{"another package", "pkg/game/a.go", "package game\nimport \"strings\"\nvar _ = strings.Fields(\"a b\")\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{tc.file: tc.src}
			// The scan needs at least one file it reads; the home file and other
			// packages are outside it.
			switch tc.file {
			case "pkg/ui/linebreak.go", "pkg/game/a.go":
				files["pkg/ui/b.go"] = "package ui\n"
			}
			vs := CheckWordWrap(files)
			if len(vs) != tc.want {
				t.Fatalf("CheckWordWrap = %v, want %d violations", vs, tc.want)
			}
		})
	}
}

// TestCheckWordWrapEmptyFileSet pins that reading nothing is a violation.
func TestCheckWordWrapEmptyFileSet(t *testing.T) {
	if vs := CheckWordWrap(map[string]string{"pkg/game/a.go": "package game\n"}); len(vs) != 1 {
		t.Fatalf("CheckWordWrap = %v, want one violation", vs)
	}
}

// TestUITextIsBrokenByTheOneLineBreaker is the live-tree half.
func TestUITextIsBrokenByTheOneLineBreaker(t *testing.T) {
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
	for _, v := range CheckWordWrap(files) {
		t.Errorf("second line breaker: %s", v)
	}
}
