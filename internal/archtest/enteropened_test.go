package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUIEntersOpenedMapsOnlyThroughEnterOpened keeps the open-then-enter
// sequence in one builder: no non-test file of pkg/ui calls f.enter( except
// listrole.go, where enterOpened lives.
func TestUIEntersOpenedMapsOnlyThroughEnterOpened(t *testing.T) {
	dir := filepath.Join("..", "..", "pkg", "ui")
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		base := filepath.Base(name)
		if strings.HasSuffix(base, "_test.go") || base == "listrole.go" {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(src), "f.enter(") {
			t.Errorf("%s calls f.enter( outside enterOpened", base)
		}
	}
}
