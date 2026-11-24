package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateChoiceMarkersHaveOneHome pins that the editor's create markers are
// named only where the row table is built; every other reader asks the table.
func TestCreateChoiceMarkersHaveOneHome(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "pkg", "ui")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "mapeditorscriptmodel.go" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, marker := range []string{"NewMessageChoice", "NewGlobalChoice"} {
			if strings.Contains(string(src), marker) {
				t.Errorf("%s names %s; read it from the create row table", name, marker)
			}
		}
	}
}
