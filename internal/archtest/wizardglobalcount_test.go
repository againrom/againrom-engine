package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGlobalNodeCountHasOneHome pins that the trigger wizard asks the fold's
// global-node count and counts no global node of its own.
func TestGlobalNodeCountHasOneHome(t *testing.T) {
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
	if strings.Contains(string(wizard), "AddressesGlobal(") {
		t.Error("mapinspectionwizard.go counts global nodes; ask mapedit.CountGlobalNodes")
	}
	if !strings.Contains(string(wizard), "mapedit.CountGlobalNodes(") {
		t.Error("mapinspectionwizard.go does not ask mapedit.CountGlobalNodes")
	}
}
