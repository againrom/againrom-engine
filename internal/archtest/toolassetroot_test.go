package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestToolsTakeTheAssetRootFromTheOneResolver holds the program tier to the
// one asset root precedence: no cmd file discovers an install or looks one up
// by an opened file itself; they pass their sources to game.FindAssetRoot.
func TestToolsTakeTheAssetRootFromTheOneResolver(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	banned := []string{"game.DiscoverAssetRoot(", "game.InstallHolding(", "game.ProcessDirs("}
	err = filepath.WalkDir(filepath.Join(root, "cmd"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, call := range banned {
			if strings.Contains(string(src), call) {
				t.Errorf("%s calls %s: take the root through game.FindAssetRoot", path, call)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
