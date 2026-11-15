package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMapDraftNamesAskTheCompanionRule(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "pkg", "game", "mapeditoropenmaps.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	if !strings.Contains(src, ".Companion(") {
		t.Error("mapDraftNames does not ask MissionPlace.Companion")
	}
	for _, banned := range []string{"mod.MapLabelFile(", "mapTextFolder("} {
		if strings.Contains(src, banned) {
			t.Errorf("mapeditoropenmaps.go names %s; the rule is MissionPlace.Companion", banned)
		}
	}
}
