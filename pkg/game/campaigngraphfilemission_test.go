package game

import (
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// graphFileMod is a mod of a manifest and loose maps named as the given
// missions of graphRegistry, with a page beside mission 10's, and the
// missions file given when it is not empty.
func graphFileMod(t *testing.T, missions string, numbers ...int) string {
	t.Helper()
	maps := map[string][]byte{}
	for _, n := range numbers {
		maps[fmt.Sprintf("%d.alm", n)] = synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: fmt.Sprintf("map %d", n)})
	}
	dir := fileFormMod(t, maps)
	for _, lang := range []string{"en", "ru"} {
		writeFileAll(t, filepath.Join(dir, "10.text", lang, "title.txt"), []byte("First contact"))
	}
	if missions != "" {
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(mod.MissionsFile)), []byte(missions))
		writeFileAll(t, filepath.Join(dir, "main.star"), []byte("def init(game, settings):\n    game.data.add(\"data/missions.toml\")\n"))
		writeFileAll(t, filepath.Join(dir, "mission", "en", "map.alm"), synth.ALM(synth.ALMOptions{Width: 8, Height: 8, Name: "mission"}))
	}
	return dir
}

// clickGraphMission clicks the box of mission n of the shown graph.
func clickGraphMission(t *testing.T, e *ui.MapEditor, n int) {
	t.Helper()
	texts, rects := e.PopupCanvasBoxes()
	i := slices.IndexFunc(texts, func(s string) bool {
		return strings.Contains(s+" ", fmt.Sprintf(" %d ", n)) && !strings.Contains(s, ",")
	})
	if i < 0 || rects[i].Empty() {
		t.Fatalf("no box of mission %d in %q", n, texts)
	}
	p := rects[i].Min.Add(rects[i].Size().Div(2))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

// A mod that replaces missions by map files named as their numbers, with no
// missions file, lists each as a row of W's mission list; the graph node of
// each opens the mod's map as that row does, in each language of the word
// book. A mission with no map anywhere stays inert, and the page writes
// nothing.
func TestEditorGraphOpensAMissionAMapFileReplaces(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(words.Language(entry), func(t *testing.T) {
			f, e := graphFixtureIn(t, entry, nil, []string{"Ford", "Ambush", "Vault"})
			dir := graphFileMod(t, "", 10, 20, 31)
			files, profile := tree(t, dir), tree(t, f.profile)
			if err := workspaceField(e, "package", 0, dir); err != nil {
				t.Fatal(err)
			}
			if f.x.mod == nil || f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
				t.Fatalf("the mod is not open as the game admits it: %+v", f.x.mod)
			}
			// The first row's map opens with the mod, as a row of a missions
			// file does.
			if got := openSource(e); got != filepath.Join(dir, "10.alm") || f.x.mod.selected != "10.alm" {
				t.Fatalf("the mod opened %q as %q", got, f.x.mod.selected)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			var rows []string
			for _, m := range s.workspace.Project.Missions {
				rows = append(rows, fmt.Sprintf("%s %s %d %t", m.RowID(), m.Kind, m.Number, m.ByFileName()))
			}
			if want := []string{"10.alm replace 10 true", "20.alm replace 20 true", "31.alm replace 31 true"}; !slices.Equal(rows, want) {
				t.Fatalf("mission rows %q, want %q", rows, want)
			}
			got := map[int]int{}
			for _, n := range s.CampaignGraph().Nodes {
				if n.Kind != ui.GraphMission {
					continue
				}
				if n.ModMission != 0 {
					got[n.Number] = n.ModMission
				}
				if held := n.Number == 10 || n.Number == 20 || n.Number == 31; held && n.Source != "" || !held && n.ModMission != 0 {
					t.Fatalf("mission %d: source %q, row %d", n.Number, n.Source, n.ModMission)
				}
			}
			if want := map[int]int{10: 1, 20: 2, 31: 3}; !reflect.DeepEqual(got, want) {
				t.Fatalf("mod mission rows %v, want %v", got, want)
			}

			showGraph(t, e)
			clickGraphMission(t, e, 30)
			if got := openSource(e); got != filepath.Join(dir, "10.alm") {
				t.Fatalf("a mission with no map opened %q", got)
			}
			clickGraphMission(t, e, 20)
			if got := openSource(e); got != filepath.Join(dir, "20.alm") || f.x.mod.selected != "20.alm" || e.Document().Name != "map 20" {
				t.Fatalf("mission 20 opened %q as %q", got, f.x.mod.selected)
			}
			showGraph(t, e)
			clickGraphMission(t, e, 31)
			if got := openSource(e); got != filepath.Join(dir, "31.alm") || f.x.mod.selected != "31.alm" {
				t.Fatalf("mission 31 opened %q as %q", got, f.x.mod.selected)
			}
			// The row is the file: its route is not a field of the list.
			s = e.Document().Edits.(*mapInspectionEdits)
			want := fmt.Sprintf(words.For(entry).Text("editor.mod.err.filemission"), "31.alm")
			for _, key := range []string{"number", "kind", "remove", "up"} {
				if _, err := s.SetWorkspaceField("package", key, "1"); err == nil || err.Error() != want {
					t.Fatalf("%s: %v, want %q", key, err, want)
				}
			}
			for _, field := range s.WorkspaceFields("package") {
				if (field.Key == "number" || field.Key == "kind") && !field.ReadOnly {
					t.Fatalf("field %s of a file mission can be edited", field.Key)
				}
			}
			if e.Dirty() {
				t.Fatal("opening the missions edited the mod")
			}
			if got := tree(t, dir); !reflect.DeepEqual(got, files) {
				t.Fatal("the page wrote into the mod folder")
			}
			if got := tree(t, f.profile); !reflect.DeepEqual(got, profile) {
				t.Fatalf("the page wrote into the profile: %v", got)
			}
		})
	}
}

// A mod holding rows of a missions file and a map file named as a mission
// lists the rows first and the file's mission after them; each graph node
// opens its own.
func TestEditorGraphOfRowsAndAMapFile(t *testing.T) {
	f, e := graphFixture(t)
	source := "[[mission]]\nkind = \"replace\"\nkey = \"ambush\"\nnumber = 20\nmap = \"mission/{lang}/map.alm\"\n"
	dir := graphFileMod(t, source, 10, 31)
	files := tree(t, dir)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if f.x.mod == nil || f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
		t.Fatalf("the mod is not open as the game admits it: %+v", f.x.mod)
	}
	row := filepath.Join(dir, "mission", "en", "map.alm")
	if got := openSource(e); got != row || f.x.mod.selected != "ambush" {
		t.Fatalf("the mod opened %q as %q", got, f.x.mod.selected)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	got := map[int]int{}
	for _, n := range s.CampaignGraph().Nodes {
		if n.Kind == ui.GraphMission && n.ModMission != 0 {
			got[n.Number] = n.ModMission
		}
	}
	if want := map[int]int{20: 1, 10: 2, 31: 3}; !reflect.DeepEqual(got, want) {
		t.Fatalf("mod mission rows %v, want %v", got, want)
	}
	showGraph(t, e)
	clickGraphMission(t, e, 10)
	if got := openSource(e); got != filepath.Join(dir, "10.alm") || f.x.mod.selected != "10.alm" {
		t.Fatalf("mission 10 opened %q as %q", got, f.x.mod.selected)
	}
	showGraph(t, e)
	clickGraphMission(t, e, 20)
	if got := openSource(e); got != row || f.x.mod.selected != "ambush" {
		t.Fatalf("mission 20 opened %q as %q", got, f.x.mod.selected)
	}
	// The writer keeps the missions file's rows and adds none for a file.
	s = e.Document().Edits.(*mapInspectionEdits)
	d := s.workspace.Project
	if b, err := mod.WriteMissions(d.Source, d.Original, d.Missions); err != nil || string(b) != source {
		t.Fatalf("the rows written: %q, %v", b, err)
	}
	if got := tree(t, dir); !reflect.DeepEqual(got, files) {
		t.Fatal("the page wrote into the mod folder")
	}
}

// Saving a mod whose missions are map files writes no missions file: an
// unchanged mod keeps every file, and an edited map changes its own file
// alone, in a mod the game still admits.
func TestSavingAModOfMapFilesWritesNoMissionRow(t *testing.T) {
	f, e := graphFixture(t)
	dir := graphFileMod(t, "", 10, 20, 31)
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if err := e.Open(filepath.Join(dir, "10.alm")); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	files := tree(t, dir)
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if got := tree(t, dir); !reflect.DeepEqual(got, files) {
		t.Fatalf("saving the unchanged mod changed its files: %v", slices.Sorted(mapsKeys(got)))
	}
	if _, err := s.SetWorkspaceField("metadata", "description", "edited"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	got := tree(t, dir)
	if got["10.alm"] == files["10.alm"] {
		t.Fatal("the edited map was not written")
	}
	got["10.alm"] = files["10.alm"]
	if !reflect.DeepEqual(got, files) {
		t.Fatalf("the save changed more than the map: %v", slices.Sorted(mapsKeys(got)))
	}
	if f.x.mod.refusal != "" || f.x.mod.contextIssue != "" || len(f.x.mod.missions.Missions) != 3 {
		t.Fatalf("the saved mod: refusal %q, issue %q, %d missions", f.x.mod.refusal, f.x.mod.contextIssue, len(f.x.mod.missions.Missions))
	}
	// A test play of the file's mission carries the mod as it is: the map
	// file is still the declaration.
	saved := tree(t, dir)
	if _, err := s.TestPlay(); err != nil {
		t.Fatal(err)
	}
	private := tree(t, filepath.Join(f.profile, TestPlayModID, "file"))
	if _, row := private[mod.MissionsFile]; row || private["10.alm"] != saved["10.alm"] || private["20.alm"] != saved["20.alm"] {
		t.Fatalf("the test mod: %v", slices.Sorted(mapsKeys(private)))
	}
	if len(f.started) != 1 || !strings.Contains(strings.Join(f.started[0], "|"), "-mods|file|") || !strings.Contains(strings.Join(f.started[0], "|"), "|10") {
		t.Fatalf("test play started %q", f.started)
	}
	if got := tree(t, dir); !reflect.DeepEqual(got, saved) {
		t.Fatal("test play wrote into the mod folder")
	}
}

func mapsKeys(m map[string]string) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}
