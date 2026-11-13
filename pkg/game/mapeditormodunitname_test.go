package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// A unit's name set in a mission of an editor-owned mod travels as a page
// edit does: test play serves it, a mission switch keeps it, the package
// save writes it and the reopened package shows it, and Undo and Redo move
// the mod's copy with the workspace.
func TestEditorModUnitNameTravelsAsAPageEdit(t *testing.T) {
	f, e := workspaceFixture(t)
	writeTestPlayInput(t, filepath.Join(f.root, MainArchive), synth.Archive([]synth.File{{Path: "id", Data: []byte("english 0")}}))
	for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
		writeTestPlayInput(t, filepath.Join(f.root, name), synth.Archive(nil))
	}
	f.x.archives.Base.Profile.Language = "english"
	s := e.Document().Edits.(*mapInspectionEdits)
	unitMap := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{X: 9*256 + 128, Y: 10*256 + 128, ClassID: 40, ClassSubID: 2, UnitID: 5}}})
	unitSection(t, unitMap)[alm.UnitOwner] = 1 // the unit stands in the roster
	declarations := "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'first.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n"
	dir := writeEditorMod(t, t.TempDir(), declarations)
	writeTestPlayInput(t, filepath.Join(dir, "first.alm"), unitMap)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	copyDir := filepath.Join(t.TempDir(), "ambush")
	if err := workspaceField(e, "package", 0, copyDir); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	const line = "5=Bjorn\r\n"
	if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: unitNameField, Text: true, TextValue: "Bjorn"}}); err != nil {
		t.Fatal(err)
	}
	modPage := func() string {
		d := s.workspace.Project
		for name, b := range d.Files {
			if filepath.Base(name) == unitNamesPage {
				return string(b)
			}
		}
		return ""
	}
	if got := modPage(); got != line {
		t.Fatalf("the mod document holds page %q, want %q", got, line)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if got := modPage(); got != "" {
		t.Fatalf("Undo left the mod document page %q", got)
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if got := modPage(); got != line {
		t.Fatalf("Redo left the mod document page %q", got)
	}
	if _, err := s.TestPlay(); err != nil {
		t.Fatal(err)
	}
	played, err := filepath.Glob(filepath.Join(f.profile, TestPlayModID, "ambush", "maps", "first.text", "*", unitNamesPage))
	if err != nil || len(played) == 0 {
		t.Fatalf("test play wrote no names page: %v", err)
	}
	for _, p := range played {
		if b, _ := os.ReadFile(p); string(b) != line {
			t.Errorf("test play folder %s holds %q", p, b)
		}
	}
	if _, err := s.selectWorkspaceMission(1); err != nil {
		t.Fatal(err)
	}
	if got := string(s.workspace.Pages[unitNamesPage]); got != line {
		t.Fatalf("a mission switch lost the name: %q", got)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	result, err := modrt.Load([]mod.Entry{mod.Load(copyDir)}, "rom1-en", nil, modrt.Options{})
	if err != nil || len(result.Missions.Missions) != 1 {
		t.Fatalf("load %s: %v", copyDir, err)
	}
	if got := string(result.Missions.Missions[0].TextFiles[unitNamesPage]); got != line {
		t.Fatalf("the saved mission's units page = %q, want %q", got, line)
	}
	doc, err := s.SetWorkspaceField("package", "folder-open", copyDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Edits.(*mapInspectionEdits).authoredUnitNames()[5]; got != "Bjorn" {
		t.Fatalf("the reopened package names unit 5 %q", got)
	}
}
