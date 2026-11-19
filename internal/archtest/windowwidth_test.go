package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditorWindowWidthHasOneStep(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"fitListWidth(", "listFit(", "listColumnsFit(", "listFitIn{"} {
			if strings.Contains(string(b), forbidden) {
				t.Errorf("%s retains %s: naturalWidth is the one width step", filepath.Base(name), forbidden)
			}
		}
		if filepath.Base(name) == "mapeditorquestion.go" && strings.Contains(string(b), "buttonRowWidth(") {
			t.Errorf("mapeditorquestion.go measures its own width; it asks naturalWidth")
		}
	}
}
