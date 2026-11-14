package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A package manifest has one constructor, defaultPackageManifest, which takes
// the game; no caller in pkg/game names the family literal or a second
// applies-to builder.
func TestPackageManifestHasOneConstructor(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	constructors := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(src)
		if strings.Contains(text, "packageBase(") {
			t.Fatalf("%s: packageBase is gone; the constructor takes the game", entry.Name())
		}
		if strings.Contains(text, "func defaultPackageManifest(") {
			constructors++
		}
		if strings.Contains(text, "AppliesTo: []string{base.ROM1}") {
			t.Fatalf("%s: a package manifest names the family literal; use defaultPackageManifest(game)", entry.Name())
		}
	}
	if constructors != 1 {
		t.Fatalf("package manifest constructors: %d, want 1", constructors)
	}
}
