package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckEditorInkNamesEachShape(t *testing.T) {
	files := map[string]string{
		"pkg/ui/editorink.go":        "package ui\nvar a = color.RGBA{1, 2, 3, 255}\n",
		"pkg/ui/mapeditorfoo.go":     "package ui\nvar b = color.RGBA{1, 2, 3, 255}\nvar none = color.RGBA{}\n",
		"pkg/ui/widgetfoo.go":        "package ui\nvar c = color.RGBA{1, 2, 3, 255}\n",
		"pkg/ui/popupfoo_test.go":    "package ui\nvar d = color.RGBA{1, 2, 3, 255}\n",
		"pkg/ui/mapeditorgraph.go":   "package ui\nvar e = color.RGBA{1, 2, 3, 255}\n",
		"pkg/ui/mapeditorcapture.go": "package ui\n",
	}
	got := map[string]string{}
	for _, v := range CheckEditorInk(files) {
		got[v.From] = v.Reason
	}
	want := map[string]string{
		"pkg/ui/mapeditorfoo.go":         "outside the editor ink table",
		"pkg/ui/mapeditorgraph.go":       "changed",
		"pkg/ui/mapeditorcapture.go":     "changed",
		"pkg/ui/mapeditorpassability.go": "not scanned",
	}
	if len(got) != len(want) {
		t.Errorf("violations %v, want one for each of %v", got, want)
	}
	for name, reason := range want {
		if !strings.Contains(got[name], reason) {
			t.Errorf("%s: violation %q, want one naming %q", name, got[name], reason)
		}
	}
}

// TestEditorInkIsOneTable is the live-tree half.
func TestEditorInkIsOneTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[editorInkTable]; !ok {
		t.Fatalf("no %s scanned", editorInkTable)
	}
	for _, v := range CheckEditorInk(files) {
		t.Errorf("%s", v)
	}
}
