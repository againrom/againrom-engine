package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// One row press focuses an added wizard row, for both windows.
func TestWizardRowFocusLivesInTheSharedRowPress(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	shared := read("pkg/ui/mapeditorwizard.go")
	dialog := read("pkg/ui/mapeditordialogue.go")
	if !messageRouteCalls(shared, "wizardRowButton", "focusObjectField", false) {
		t.Error("wizardRowButton must focus the row an add button adds")
	}
	if messageRouteCalls(dialog, "dialogRowButton", "focusObjectField", false) {
		t.Error("dialogRowButton keeps a second focus of an added row")
	}
}

// One row mover serves both windows: no reply-only mover exists.
func TestWizardRowMoveIsOneEditorMethod(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"mapeditordialogue.go", "mapeditormessage.go"} {
		b, err := os.ReadFile(filepath.Join(root, "pkg", "ui", name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "moveReplyRow") {
			t.Errorf("%s names a reply-only row mover", name)
		}
	}
}
