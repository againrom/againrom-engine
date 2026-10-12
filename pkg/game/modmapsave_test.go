package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
)

const editorModManifest = "id = \"ambush\"\ntitle = \"Ambush\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\n" +
	"description = \"Mission 20 on another map.\"\napi = 1\napplies-to = [\"rom1\"]\nentry = \"main.star\"\n"

// writeEditorMod writes the mod folder ambush under parent with the given
// data/missions.toml; an empty missions writes none.
func writeEditorMod(t *testing.T, parent, missions string) string {
	t.Helper()
	dir := filepath.Join(parent, "ambush")
	files := map[string]string{"mod.toml": editorModManifest,
		"main.star": "def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"}
	if missions != "" {
		files["data/missions.toml"] = missions
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const editorModMissions = "[[mission]]\nkind = \"replace\"\nkey = \"ambush\"\nnumber = 20\nmap = \"mission/{lang}/map.alm\"\n"

// editorEdits is an opened map of source, as the editor holds it.
func editorEdits(t *testing.T, x *MapInspector, source string) *mapInspectionEdits {
	t.Helper()
	model, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "edited"}))
	if err != nil {
		t.Fatal(err)
	}
	return &mapInspectionEdits{inspector: x, model: model, source: source}
}

func TestEditorSavesIntoTheModMissionPath(t *testing.T) {
	dir := writeEditorMod(t, t.TempDir(), editorModMissions)
	x := &MapInspector{}
	if err := x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "mission", "en", "map.alm")
	s := editorEdits(t, x, "scenario/20.alm")
	if s.ModTarget() != "ambush" {
		t.Fatalf("target %q", s.ModTarget())
	}
	got, err := s.SaveIntoMod()
	if err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(want)
	if err != nil || !bytes.Equal(written, s.model.Bytes()) || s.Dirty() {
		t.Fatalf("saved to %s: %v, dirty %v", got, err, s.Dirty())
	}
	// The saved map is now the mod's file; Save again replaces it.
	if err := os.WriteFile(want, []byte("older"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatalf("second save: %v", err)
	}
	if written, _ := os.ReadFile(want); !bytes.Equal(written, s.model.Bytes()) {
		t.Fatal("the second save did not replace the mod's map")
	}
	entries, _ := os.ReadDir(filepath.Dir(want))
	if len(entries) != 1 {
		t.Fatalf("the mod's map folder holds %d files", len(entries))
	}
}

func TestEditorRefusesAModThatNamesNoEntryForTheMap(t *testing.T) {
	dir := writeEditorMod(t, t.TempDir(), editorModMissions)
	x := &MapInspector{}
	if err := x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"scenario/10.alm", "loose/20.alm", filepath.Join(t.TempDir(), "other.alm")} {
		s := editorEdits(t, x, source)
		if _, err := s.SaveIntoMod(); err == nil || !strings.Contains(err.Error(), `no mission of mod "ambush" names this map`) {
			t.Errorf("%s: %v", source, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "mission")); err == nil {
		t.Fatal("a refused save wrote into the mod")
	}
	// A mod that declares no missions is refused when it is named.
	if err := (&MapInspector{}).SetMod(writeEditorMod(t, t.TempDir(), "")); err == nil || !strings.Contains(err.Error(), `mod "ambush" declares no missions`) {
		t.Fatalf("no missions: %v", err)
	}
	// A mod folder inside the install is refused.
	root := t.TempDir()
	inside := writeEditorMod(t, root, editorModMissions)
	if err := (&MapInspector{archives: &Archives{Root: root}}).SetMod(inside); err == nil || !strings.Contains(err.Error(), "is inside the install") {
		t.Fatalf("inside the install: %v", err)
	}
	// Without a mod there is no mod target and Save asks for a path.
	if s := editorEdits(t, &MapInspector{}, "scenario/20.alm"); s.ModTarget() != "" {
		t.Fatal("a mod target without a mod")
	}
}

func TestWriteModMapRefusesATargetOutsideTheModFolderWithoutCreatingADirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "mod")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "elsewhere")
	for _, rel := range []string{"../elsewhere/deep/map.alm", "../elsewhere/map.alm"} {
		if _, err := writeModMap("", dir, rel, []byte("x")); err == nil || !strings.Contains(err.Error(), "outside the mod folder") {
			t.Errorf("%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("a refused target created a directory outside the mod folder")
	}
}
