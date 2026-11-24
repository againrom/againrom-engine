package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rowHelpers are the files whose target pickers skip every create row.
var rowHelpers = map[string]bool{"mapeditorwizard.go": true, "mapeditorcommandread.go": true, "mapeditorscriptauthor.go": true, "mapeditorscriptmodel.go": true}

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
		if rowHelpers[name] && (strings.Contains(string(src), "NewChoice != createChoice(") || strings.Contains(string(src), "NewChoice == createChoice(")) {
			t.Errorf("%s compares a target with the message marker; test NewChoice against 0", name)
		}
	}
}

// TestHostCreateIsReadFromTheTable pins that the host's SetField makes a
// message through the create row's host edit, not a branch of its own.
func TestHostCreateIsReadFromTheTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, "pkg", "game", "mapinspectionedit.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"ui.NewMessageChoice", "ui.ScriptMessageParameter", ".NewMessage("} {
		if strings.Contains(string(src), bad) {
			t.Errorf("mapinspectionedit.go names %s; use ui.HostCreate", bad)
		}
	}
}
