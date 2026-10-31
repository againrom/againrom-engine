package archtest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func pictureTests(t *testing.T, files map[string]string) string {
	t.Helper()
	got, err := PictureConstantTests(files)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(got, ",")
}

const pictureHeader = "package game\nimport pkgrules \"againrom/pkg/rules\"\nconst deformation = 7\nconst (\n\tfirst = iota\n\tsecond\n)\n"

func TestPictureConstantScanNamesEachShape(t *testing.T) {
	shapes := []struct {
		name, source string
		line         int
	}{
		{"equality with a named constant", "func f(p P) bool { return p.Picture == deformation }", 1},
		{"equality with a literal", "func f(p P) bool { return p.Picture == 20 }", 1},
		{"inequality with a literal", "func f(p P) bool { return 51 != picture }", 1},
		{"inequality with a named constant", "func f(picture int) bool { return second != picture }", 1},
		{"constant on the left", "func f(p P) bool { return deformation == p.Picture }", 1},
		{"conversion around the picture", "func f(p P) bool { return int(p.Picture) == deformation }", 1},
		{"conversion around the constant", "func f(p P) bool { return p.Picture == int32(deformation) }", 1},
		{"parenthesised picture", "func f(p P) bool { return (picture) == 7 }", 1},
		{"case arm with a literal", "func f(p P) int {\nswitch p.Picture {\ncase 7:\nreturn 1\n}\nreturn 0 }", 3},
		{"case arm with a named constant", "func f(picture int) int {\nswitch picture {\ncase deformation:\nreturn 1\n}\nreturn 0 }", 3},
		{"constant reached through an import", "func f(p P) bool { return p.Picture == pkgrules.Deformation }", 0},
		{"ordered range checks", "func f(p P) bool { return p.Picture < 0 || p.Picture < 1 }", 0},
		{"nil test", "func f(picture *int) bool { return picture == nil }", 0},
		{"two values", "func f(a, b P) bool { return a.Picture != b.Picture }", 0},
		{"a variable that is no constant", "func f(p P, n int) bool { return p.Picture == n }", 0},
		{"a selector that is no import", "func f(p, q P) bool { return p.Picture == q.deformation }", 0},
		{"another name", "func f(p P) bool { return p.Kind == 7 }", 0},
		{"switch on another name", "func f(p P) int {\nswitch p.Kind {\ncase 7:\nreturn 1\n}\nreturn 0 }", 0},
		{"a comment", "// p.Picture == 20\nfunc f() {}", 0},
	}
	for _, s := range shapes {
		want := ""
		if s.line > 0 {
			want = fmt.Sprintf("pkg/game/x.go:%d", 7+s.line)
		}
		if got := pictureTests(t, map[string]string{"pkg/game/x.go": pictureHeader + s.source + "\n"}); got != want {
			t.Errorf("%s: PictureConstantTests = %q, want %q", s.name, got, want)
		}
	}
}

func TestPictureConstantScanReadsAConstantThroughAnImport(t *testing.T) {
	got := pictureTests(t, map[string]string{
		"pkg/rules/a.go": "package rules\nconst Deformation = 7\nfunc f(p P) bool { return p.Picture == 30 }\n",
		"pkg/game/x.go":  pictureHeader + "func f(p P) bool { return p.Picture == pkgrules.Deformation }\n",
		"pkg/sim/b.go":   "package sim\nfunc g(p P) bool { return p.Picture == 51 }\n",
		"cmd/tool/c.go":  "package main\nconst k = 4\nfunc h(picture int) bool { return picture != k }\n",
		"pkg/ui/d.go":    "package ui\nfunc i(picture int) bool { return picture == k }\n",
	})
	if want := "cmd/tool/c.go:3,pkg/game/x.go:8,pkg/sim/b.go:2"; got != want {
		t.Fatalf("PictureConstantTests = %q, want %q (pkg/rules is exempt, and k is no constant of pkg/ui)", got, want)
	}
}

func TestPictureCapabilitiesAreRows(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["pkg/sim/savedcontinuation.go"]; !ok {
		t.Fatal("pkg/sim/savedcontinuation.go is missing; the scan guards nothing")
	}
	got, err := PictureConstantTests(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range got {
		t.Errorf("%s tests a projectile picture against a number or a constant; read rules.PictureCapabilityOf", at)
	}
}
