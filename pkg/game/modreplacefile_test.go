package game

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// fileFormMod writes a mod "file" of a manifest and maps, each at its path
// in the mod, and returns its folder.
func fileFormMod(t *testing.T, maps map[string][]byte) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "file")
	writeFileAll(t, filepath.Join(dir, mod.ManifestName), []byte("id = \"file\"\ntitle = \"File\"\nversion = \"1.0\"\n"))
	for rel, b := range maps {
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(rel)), b)
	}
	return dir
}

// fileFormMissions loads the mod at dir as the launch does and applies its
// missions to the replace fixture's front end.
func fileFormMissions(t *testing.T, dir string) (*FrontEnd, modrt.Result, []string, error) {
	t.Helper()
	res, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := replaceFixtureFront(t)
	warnings, err := f.SetModMissions(res.Missions)
	return f, res, warnings, err
}

// A mod of a manifest and a map named 20.alm, in its top or in maps/, builds
// campaign mission 20 from that map through the row form's route; its save
// mark is the mod's, whose digest covers the map.
func TestAMapNamedAsACampaignMissionReplacesIt(t *testing.T) {
	modded := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "by name"})
	for _, rel := range []string{"20.alm", "maps/20.alm"} {
		dir := fileFormMod(t, map[string][]byte{rel: modded})
		f, res, warnings, err := fileFormMissions(t, dir)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		// The map rule table judges the map as a row's: its warnings name
		// the file.
		for _, w := range warnings {
			if !strings.HasPrefix(w, `mod "file": `+rel+`: map `+rel+`: warning `) {
				t.Fatalf("%s: %q", rel, w)
			}
		}
		if got, err := f.Archives.Containers.ReadFile("scenario/20.alm"); err != nil || string(got) != string(modded) {
			t.Fatalf("%s: mission 20 is served %d bytes: %v", rel, len(got), err)
		}
		m, ok := f.Table.Mods.Missions.Replacing(20)
		if !ok || m.Mod != "file" || m.Map != rel {
			t.Fatalf("%s: recorded %+v", rel, f.Table.Mods.Missions)
		}
		want := []string{`mission 20 replaced by mod "file": map ` + rel + `, named by its file name`}
		if got := tableMissions(f.Table).Replacements(); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: %q", rel, got)
		}
		// The installed texts stay: the mod serves none.
		if _, err := f.Archives.Containers.ReadFile("main/text/battle/m20/title.txt"); err == nil {
			t.Errorf("%s: a title the mod does not declare was served", rel)
		}
		// The save mark: the mod is marked, a save without a mark is refused
		// in the replaced mission, and the digest covers the map.
		if len(res.Set.Mods) != 1 || res.Set.Mods[0].StateFree {
			t.Fatalf("%s: set %+v", rel, res.Set)
		}
		if err := refuseUnmarkedReplaced([]byte("an unmarked save"), 20, mapload.ModContext{Missions: f.Table.Mods.Missions}); err == nil ||
			!strings.Contains(err.Error(), `mod "file" replaces that mission's map`) {
			t.Fatalf("%s: %v", rel, err)
		}
		other := fileFormMod(t, map[string][]byte{rel: synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "another"})})
		a, errA := mod.ContentDigest(dir)
		b, errB := mod.ContentDigest(other)
		if errA != nil || errB != nil || a == b {
			t.Fatalf("%s: the digest does not cover the map: %s %s %v %v", rel, a, b, errA, errB)
		}
	}
}

// The map editor's mod workspace lists the tables of data/missions.toml; a
// map the file form names is a file of the mod it neither lists nor writes.
// Opening the mod reports no issue, the complete copy and Save keep the file
// byte for byte, and the loader still reads it.
func TestTheEditorKeepsAMapNamedByItsNumberAsAFileOfTheMod(t *testing.T) {
	f, e := workspaceFixture(t)
	declarations := "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'mission/{lang}/first.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n"
	dir := writeEditorMod(t, t.TempDir(), declarations)
	named := e.Document().SourceBytes
	for _, name := range []string{"mission/en/first.alm", "mission/ru/first.alm", "10.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), named)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if s.inspector.mod == nil || s.inspector.mod.contextIssue != "" {
		t.Fatalf("opening the mod: %+v", s.inspector.mod)
	}
	copyDir := filepath.Join(t.TempDir(), "ambush")
	if err := workspaceField(e, "package", 0, copyDir); err != nil {
		t.Fatal(err)
	}
	if w := s.workspace.Project; w == nil || len(w.Missions) != 1 || w.Missions[0].Key != "first" {
		t.Fatalf("the workspace lists %+v", s.workspace.Project)
	}
	if err := workspaceField(e, "package", 8, "73"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, copyDir)["10.alm"]; got != string(named) {
		t.Fatalf("Save changed 10.alm: %d bytes", len(got))
	}
	entries, err := mod.Resolve(filepath.Dir(copyDir), []string{"ambush"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := modrt.Load(entries, BaseID(InspectInstall(f.root)), nil, modrt.Options{})
	if err != nil {
		t.Fatal(err)
	}
	admitted, _, err := loaded.Missions.AdmitMaps(func(int) error { return nil })
	if m, ok := admitted.Replacing(10); err != nil || !ok || m.Map != "10.alm" {
		t.Fatalf("the saved mod's file form: %+v %v", admitted.Missions, err)
	}
}

// A numbered map with no counterpart in the install replaces nothing and
// refuses nothing: the launch warns naming the file.
func TestAMapNamedAsNoCampaignMapWarnsAndReplacesNothing(t *testing.T) {
	broken := []byte("MAP")
	f, _, warnings, err := fileFormMissions(t, fileFormMod(t, map[string][]byte{"31.alm": broken, "maps/50.alm": broken}))
	want := []string{
		`mod "file": 31.alm: the install holds no map for mission 31 (scenario/31.alm); the file replaces nothing`,
		`mod "file": maps/50.alm: number 50 is no mission of this install's campaign; the file replaces nothing`,
	}
	if err != nil || !reflect.DeepEqual(warnings, want) {
		t.Fatalf("%q %v", warnings, err)
	}
	if !f.Table.Mods.Missions.Empty() {
		t.Fatalf("recorded %+v", f.Table.Mods.Missions)
	}
	// Loss control: the same broken map named as mission 20 is refused as a
	// row's map is, naming the file.
	if _, _, _, err := fileFormMissions(t, fileFormMod(t, map[string][]byte{"20.alm": broken})); err == nil ||
		!strings.Contains(err.Error(), `mod "file": 20.alm: map 20.alm: `) {
		t.Fatalf("%v", err)
	}
}
