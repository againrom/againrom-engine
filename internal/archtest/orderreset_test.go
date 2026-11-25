package archtest

import (
	"os"
	"strings"
	"testing"
)

// A cell move resets a scripted unit's order through standScriptedUnit rather
// than clearing Raw[10:12] in cellentry.go.
func TestCellMoveResetsOrderThroughStandScriptedUnit(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	src, ok := files["pkg/sim/cellentry.go"]
	if !ok {
		t.Fatal("pkg/sim/cellentry.go is missing; the scan guards nothing")
	}
	if !strings.Contains(src, "w.standScriptedUnit(") {
		t.Error("pkg/sim/cellentry.go must reset the order through standScriptedUnit")
	}
	for _, bad := range []string{"Raw[10", "ensureSavedOrder("} {
		if strings.Contains(src, bad) {
			t.Errorf("pkg/sim/cellentry.go touches %q directly; reset the order in standScriptedUnit", bad)
		}
	}
}
