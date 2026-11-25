package archtest

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConditionShapeLivesInOpRows guards that condition shapes live on Op.Shape.
func TestConditionShapeLivesInOpRows(t *testing.T) {
	gone := []string{"AreaOps", "FirstAreaOps", "SecondAreaOps", "areaOps"}
	root := filepath.Join("..", "..")
	for _, dir := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, g := range gone {
				if strings.Contains(string(src), g) {
					t.Errorf("%s holds %q; condition shapes live in scriptprofile.Op.Shape", path, g)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	triggers, err := os.ReadFile(filepath.Join(root, "pkg", "game", "mapinspectiontriggers.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, lit := range []string{"n.Opcode == 2", "n.Opcode == 3"} {
		if strings.Contains(string(triggers), lit) {
			t.Errorf("pkg/game/mapinspectiontriggers.go holds %q; read scriptprofile.Op.Shape", lit)
		}
	}
	areas, err := os.ReadFile(filepath.Join(root, "pkg", "mapload", "scriptareas.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range []string{"OpsFirst", "OpsSecond"} {
		if strings.Contains(string(areas), branch) {
			t.Errorf("pkg/mapload/scriptareas.go names %q; read scriptprofile.OpRow", branch)
		}
	}
	if !strings.Contains(string(areas), ".Shape") || !strings.Contains(string(triggers), ".Shape") {
		t.Error("expected ScriptAreas and mapinspectiontriggers to read Op.Shape")
	}
}
