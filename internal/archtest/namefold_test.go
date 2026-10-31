package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestResourceNamesFoldThroughNamefold(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := HandFoldSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s folds ASCII case by hand; use pkg/namefold", s)
	}
	for file := range HandFoldAllowed {
		if _, err := os.Stat(root + "/" + file); err != nil {
			t.Errorf("allowed file %s: %v", file, err)
		}
	}
}

func TestHandFoldScanSeesEachShape(t *testing.T) {
	const src = `package x

func a(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b - 'A' + 'a'
	}
	return b
}

func b(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func c(s string) string {
	t := []byte(s)
	for i, c := range t {
		switch {
		case c >= 'A' && c <= 'Z':
			t[i] = c + 32
		}
	}
	return string(t)
}

func d(b byte) bool { return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' }

func e(b byte) byte { return b + 1 }
`
	got := strings.Join(handFoldInSource("x.go", []byte(src)), ", ")
	if got != "x.go:3 a, x.go:10 b, x.go:17 c" {
		t.Fatalf("hits = %s", got)
	}
}
