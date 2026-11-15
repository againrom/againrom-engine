package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The class cut and the source bound of a spell effect's magnitude are applied
// only by the one landing builder in spellarms.go.
func TestSpellEffectMagnitudeHasOneLandingBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "sim")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "spellarms.go" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, call := range []string{".sightCut(", ".sourceSightCut("} {
			if strings.Contains(string(b), call) {
				t.Errorf("%s calls %s; the magnitude where an effect lands comes from landingMagnitude", name, call)
			}
		}
	}
}
