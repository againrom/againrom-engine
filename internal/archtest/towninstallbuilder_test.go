package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTownInstallHasOneBuilder reads every production file of pkg/game and
// checks that the townInstall composite literal has one home: the builder
// newTownInstall in campaignreturn.go.
func TestTownInstallHasOneBuilder(t *testing.T) {
	root := moduleRootForTest(t)
	dir := filepath.Join(root, "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	const literal = "townInstall{"
	var homes []string
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		scanned++
		for i := 0; i < strings.Count(string(src), literal); i++ {
			homes = append(homes, name)
		}
	}
	if scanned == 0 {
		t.Fatal("pkg/game holds no production Go file; the scan went blind")
	}
	if len(homes) != 1 || homes[0] != "campaignreturn.go" {
		t.Errorf("townInstall literal homes = %v, want exactly one in campaignreturn.go", homes)
	}
}
