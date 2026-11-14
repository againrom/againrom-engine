package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapCompanionSuffixesHaveOneHome(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "pkg", "mod"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"mapVariablesSuffix": 3, "mapTextSuffix": 2}
	got := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, "pkg", "mod", name))
		if err != nil {
			t.Fatal(err)
		}
		for suffix := range want {
			got[suffix] += strings.Count(string(src), suffix)
		}
	}
	for suffix, n := range want {
		if got[suffix] != n {
			t.Errorf("pkg/mod spells %s %d times, want %d (MapLabelFile, MissionStem, Pages)", suffix, got[suffix], n)
		}
	}
}
