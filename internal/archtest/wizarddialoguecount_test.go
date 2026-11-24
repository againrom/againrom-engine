package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWizardDialogueCountHasOneHome pins that the trigger wizard asks the
// fold's interactive count and keeps no dialogue predicate of its own.
func TestWizardDialogueCountHasOneHome(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	wizard, err := os.ReadFile(filepath.Join(root, "pkg", "game", "mapinspectionwizard.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wizard), "a.Opcode == 2") {
		t.Error("mapinspectionwizard.go keeps its own dialogue predicate; ask mapedit.CountInteractive")
	}
	if !strings.Contains(string(wizard), "mapedit.CountInteractive(") {
		t.Error("mapinspectionwizard.go does not ask mapedit.CountInteractive")
	}
}
