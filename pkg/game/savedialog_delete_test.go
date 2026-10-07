package game

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveDialogDeleteUsesBrowsedDirectoryAndExistingFences(t *testing.T) {
	initial, browsed, fence := t.TempDir(), t.TempDir(), t.TempDir()
	const name = "selected.sav"
	for _, dir := range []string{initial, browsed, fence} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("snapshot"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	seams := (&FrontEnd{}).SaveDialogSeams(SaveStore{Dir: initial}, OriginalStore{Dir: fence})
	if seams.CanDelete == nil || seams.PrepareDelete == nil {
		t.Fatal("SAVE has no selected-file deletion seam")
	}
	for _, target := range []struct{ dir, name string }{{fence, name}, {browsed, "../selected.sav"}, {browsed, "missing.sav"}} {
		if seams.CanDelete(target.dir, target.name) {
			t.Fatalf("unsafe target enabled: %+v", target)
		}
		if _, err := seams.PrepareDelete(target.dir, target.name); err == nil {
			t.Fatalf("unsafe target prepared: %+v", target)
		}
	}
	if !seams.CanDelete(browsed, name) {
		t.Fatal("browsed local save cannot be deleted")
	}
	remove, err := seams.PrepareDelete(browsed, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(browsed, name), []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := remove(); err == nil {
		t.Fatal("confirmation deleted a changed save")
	}
	remove, err = seams.PrepareDelete(browsed, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := remove(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(browsed, name)); !os.IsNotExist(err) {
		t.Fatalf("selected save remains: %v", err)
	}
	for _, dir := range []string{initial, fence} {
		if got, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(got) != "snapshot" {
			t.Fatalf("unselected namesake changed: %s, %v", dir, err)
		}
	}
	legacy := filepath.Join(browsed, "legacy.ags")
	if err := os.WriteFile(legacy, []byte("legacy snapshot"), 0600); err != nil {
		t.Fatal(err)
	}
	// AGS is retired: its files are neither deletable nor touched.
	if seams.CanDelete(browsed, "legacy.ags") {
		t.Fatal("an AGS row is deletable")
	}
	if _, err := seams.PrepareDelete(browsed, "legacy.ags"); err == nil {
		t.Fatal("an AGS delete was prepared")
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("an AGS file was touched", err)
	}
}
