package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/launch"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
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

// The chain mod the editor publishes with no path typed lands in the mods
// folder the one resolver gives a program folder with no starter.ini. The
// starter reads that folder: it lists the mod, and the mission its list would
// name to the game is the chain's first. The chain then plays from that folder.
func TestReleaseEditorPublishedModChainSAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	e := editorChainReleaseProject(t)
	mods := launch.ModsFolder("", t.TempDir())
	s := e.Document().Edits.(*mapInspectionEdits)
	s.inspector.SetTestPlay(TestPlayHost{ModsFolder: mods})
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	if !e.FocusModField("publish") {
		t.Fatal("missing Publish")
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(mods, mod.PackedName(chainModID))
	if st, err := os.Stat(file); err != nil || !st.Mode().IsRegular() {
		t.Fatalf("Publish wrote no %s: %v", file, err)
	}
	listed, err := mod.Scan(mods)
	if err != nil || len(listed) != 1 || listed[0].Err != nil || listed[0].Manifest.ID != chainModID {
		t.Fatalf("the mods folder lists %+v, %v", listed, err)
	}
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	d, _, err := mod.DescribeMissions(listed[0].Dir, chainModID, lang)
	if err != nil {
		t.Fatal(err)
	}
	if missions := d.Listed(); len(missions) == 0 || missions[0].Number != guestMissionNo || missions[0].Title == "" {
		t.Fatalf("the starter would list %+v, want mission %d first with its title", missions, guestMissionNo)
	}
	releaseMissionChainSAV(t, mods)
}
