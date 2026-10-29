package game

import (
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func editorChainReleaseProject(t *testing.T) *ui.MapEditor {
	t.Helper()
	parent := assembledMods(t, chainModID, chainModID)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 0, filepath.Join(parent, chainModID)); err != nil {
		t.Fatal(err)
	}
	working := filepath.Join(t.TempDir(), chainModID)
	if err := workspaceField(e, "package", 0, working); err != nil {
		t.Fatal(err)
	}
	if e.Dirty() {
		t.Fatal("complete copy did not checkpoint the project")
	}
	return e
}

func TestReleaseEditorPublishedModChainSAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	e := editorChainReleaseProject(t)
	out := t.TempDir()
	publishEditorFixture(t, e, out)
	releaseMissionChainSAV(t, out)
}
