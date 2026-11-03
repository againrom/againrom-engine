package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRaisedActorsUseTheSharedCreationBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "pkg/sim/controlspirit.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if strings.Count(source, "w.createActor(") != 1 {
		t.Fatal("corpse raise must use the shared runtime actor builder")
	}
	for _, duplicate := range []string{"append(w.entities", "w.constructMover(", "w.commandGroup(", "w.newSavedCommandGroup("} {
		if strings.Contains(source, duplicate) {
			t.Fatalf("corpse raise duplicates actor construction: %s", duplicate)
		}
	}
}
