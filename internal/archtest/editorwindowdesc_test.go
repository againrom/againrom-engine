package archtest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// codedEditorWindows counts popupSpec literals in pkg/ui outside
// popupdesc.go; it may only fall.
const codedEditorWindows = 43

func TestEditorWindowLiteralsDoNotRise(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	var all []byte
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, src...)
		if base != "popupdesc.go" {
			n += bytes.Count(src, []byte("popupSpec{"))
		}
	}
	screens, err := filepath.Glob(filepath.Join(root, "pkg", "ui", "screens", "*.json"))
	if err != nil || len(screens) == 0 {
		t.Fatalf("no window description: %v", err)
	}
	for _, f := range screens {
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		if !bytes.Contains(all, []byte(`embeddedPopupDesc("`+name+`")`)) {
			t.Errorf("pkg/ui/screens/%s.json is read by no editor window", name)
		}
	}
	switch {
	case n > codedEditorWindows:
		t.Errorf("%d popupSpec literals in pkg/ui outside popupdesc.go, more than %d: describe the window in pkg/ui/screens", n, codedEditorWindows)
	case n < codedEditorWindows:
		t.Errorf("%d popupSpec literals in pkg/ui outside popupdesc.go: lower codedEditorWindows from %d", n, codedEditorWindows)
	}
}
