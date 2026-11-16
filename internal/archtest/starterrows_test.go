package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStarterSettingsRowsHaveOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "cmd", "starter", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("cmd/starter has no source; the scan guards nothing")
	}
	y := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		y += strings.Count(src, "66 + 24*")
		for _, bad := range []string{"advancedRow", "row(16", "row(17", "row(18"} {
			if strings.Contains(src, bad) {
				t.Errorf("%s: %q is a second route to a settings view row", filepath.Base(f), bad)
			}
		}
	}
	if y != 1 {
		t.Errorf("the settings row y has %d homes in cmd/starter, want 1", y)
	}
}
