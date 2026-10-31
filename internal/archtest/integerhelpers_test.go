package archtest

import (
	"os"
	"testing"
)

func TestCheckIntegerHelpersNamesEachCopy(t *testing.T) {
	for _, tc := range []struct {
		name, file, src string
		want            int
	}{
		{"abs", "pkg/a/a.go", "package a\nfunc abs(v int) int { return v }\n", 1},
		{"absInt", "cmd/t/m.go", "package main\nfunc absInt(v int) int { return v }\n", 1},
		{"abs64 in a test", "pkg/a/a_test.go", "package a\nfunc abs64(v int64) int64 { return v }\n", 1},
		{"a named copy", "pkg/a/a.go", "package a\nfunc cellEffectAbs(v int32) int32 { return v }\n", 1},
		{"minInt", "pkg/a/a.go", "package a\nfunc minInt(a, b int) int { return a }\n", 1},
		{"a method", "pkg/a/a.go", "package a\nfunc (s T) abs(rel int) int { return rel }\n", 0},
		{"another shape", "pkg/a/a.go", "package a\nfunc absorbedDeadPayload() []byte { return nil }\nfunc absolute(p string) string { return p }\n", 0},
		{"the leaf", "pkg/intmath/intmath.go", "package intmath\nfunc abs(v int) int { return v }\n", 0},
		{"another tree", "internal/x/x.go", "package x\nfunc abs(v int) int { return v }\n", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CheckIntegerHelpers(map[string]string{tc.file: tc.src}); len(got) != tc.want {
				t.Fatalf("CheckIntegerHelpers = %v, want %d", got, tc.want)
			}
		})
	}
	if vs := CheckIntegerHelpers(nil); len(vs) != 1 {
		t.Fatalf("CheckIntegerHelpers(nil) = %v, want one violation", vs)
	}
}

func TestIntegerHelpersHaveOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadTreeSources(root, "pkg", "cmd")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["pkg/intmath/intmath.go"]; !ok {
		t.Fatal("pkg/intmath/intmath.go is missing; the scan guards nothing")
	}
	for _, v := range CheckIntegerHelpers(files) {
		t.Errorf("%s", v)
	}
}
