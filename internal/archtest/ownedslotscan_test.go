package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOwnedSlotPairScanHasOneHome(t *testing.T) {
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
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "ownedslot.go" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		for _, banned := range []string{".quick-owner", ".timed-owner", "readQuickSaveSlots", "OwnerExtension"} {
			if strings.Contains(src, banned) {
				t.Errorf("pkg/game/%s names %q: the pair scan and the sidecar extensions live in ownedslot.go", name, banned)
			}
		}
	}
}
