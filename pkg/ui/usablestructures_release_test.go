package ui

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/formats/res"
)

func TestReleaseUsableStructureRegistryAdmission(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("AGAINROM_ASSETS not set")
	}
	archive, err := res.OpenFileIndex(filepath.Join(root, "graphics.res"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := reg.Parse(saveDialogInstallEntry(t, archive, "structures/structures.reg"))
	if err != nil {
		t.Fatal(err)
	}
	classes, err := data.LoadStructureClasses(r)
	if err != nil {
		t.Fatal(err)
	}
	var usable []int32
	for _, c := range classes.All() {
		if c.Usable == 0 {
			continue
		}
		if c.ID < 1 || c.ID > 255 {
			t.Fatalf("unreachable usable class ID %d", c.ID)
		}
		usable = append(usable, c.ID)
		checkStructureAdmission(t, uint16(c.ID))
	}
	if len(usable) == 0 {
		t.Fatal("registry has no usable classes")
	}
	t.Logf("registry usable classes (%d): %v", len(usable), usable)
}
