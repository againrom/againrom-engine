package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestItemInstanceKeyHasOneBuilder: in pkg/game the only production source
// that formats a value with %#v is the item instance key function and the
// capture comparison in headless.go. An item instance formatted anywhere else
// would key on the native record pointer, which differs per copy.
func TestItemInstanceKeyHasOneBuilder(t *testing.T) {
	dir := filepath.Join(moduleRootForTest(t), "pkg", "game")
	allowed := map[string]bool{"consumables.go": true, "headless.go": true}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		if allowed[d.Name()] {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(src), "%#v") {
			t.Errorf("%s formats a value with %%#v; an item instance key comes from itemCastKey", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
