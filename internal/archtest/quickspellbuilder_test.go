package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The shortcut cells have one reader and one writer in quickspells.go, and
// each row one home in pkg/game.
func TestShortcutQuickSpellBuilderTakesTheLayout(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "game")
	src, err := os.ReadFile(filepath.Join(dir, "quickspells.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, sig := range []string{
		"func quickSpellsToOriginalIndices(slots ui.ShortcutSlots, campaign campaignService)",
		"func quickSpellsFromOriginalIndices(indices []int32, layout ui.ShortcutLayout)",
	} {
		if !strings.Contains(string(src), sig) {
			t.Errorf("quickspells.go lacks the one-builder signature %q", sig)
		}
	}
	home := map[string]string{
		"ui.FirstShortcutLayout":       "campaignfirst.go",
		"ui.SecondShortcutLayout":      "campaignsecond.go",
		"ui.OlderSecondShortcutLayout": "quickspells.go",
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for row, file := range home {
			if n := strings.Count(string(b), row); n > 0 && name != file || n > 1 {
				t.Errorf("pkg/game/%s names %s %d times; its one home is %s", name, row, n, file)
			}
		}
	}
}
