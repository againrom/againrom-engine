package game

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// makeMapLangs are the editor's two languages and what each says of a map
// that is already named, in the refusal's own words.
var makeMapLangs = []struct{ book, taken, number string }{
	{"", "is already named by a mission of the mod", "is not a mission of this install's campaign"},
	{"ru", "уже названа миссией мода", "не миссия кампании этой установки"},
}

// makeMapEditor is the editor over a mod with an unused map, spare.alm, a map
// that replaces mission 20 by its name and a map two rows name; the unused
// map is open as a tab. It answers the unused map's edits and the folder the
// editor saves the mod in.
func makeMapEditor(t *testing.T, book string) (*testPlayFixture, *ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	f, e := workspaceFixture(t)
	f.x.archives.Base.Profile.Language = map[string]string{"": "english", "ru": "russian"}[book]
	e.SetWords(words.For(book))
	e.Layout(1280, 800)
	dir := modMapsMod(t, f.x, e, e.Document().SourceBytes)
	if err := modMapsOpen(t, e, 4); err != nil {
		t.Fatal(err)
	}
	spare := e.Document().Edits.(*mapInspectionEdits)
	// The row that names a map the mod has no file of is not under test.
	delete(spare.workspace.Project.Files, openMapsFile(t, tree(t, dir), "gone.alm"))
	return f, e, spare, dir
}

// actionOf is the entry of the Missions page's map actions that names map
// rel, and the captions of its buttons.
func actionOf(t *testing.T, s *mapInspectionEdits, rel string) (ui.ModMapAction, bool) {
	t.Helper()
	actions, open := s.ModMapActions()
	if !open {
		t.Fatal("the Missions page lists no map actions")
	}
	for _, a := range actions {
		if strings.HasPrefix(a.Label, strings.TrimPrefix(rel, mod.MapsFolder+"/")+":") {
			return a, true
		}
	}
	return ui.ModMapAction{}, false
}

// reopened is the mod saved at dir as the game reads it, its rows in the
// editor's language and the install's answer, checked as the editor checks
// them before it plays them.
func reopened(t *testing.T, f *testPlayFixture, dir string) mod.MissionData {
	t.Helper()
	d, _, err := mod.DescribeMissions(dir, "ambush", "en")
	if err != nil {
		t.Fatal("the saved mod does not describe:", err)
	}
	d, _, err = d.Listed(f.x.missionAnswer())
	if err != nil {
		t.Fatal("the saved mod's rows are refused:", err)
	}
	if err := f.x.checkMissionSet(d.Missions); err != nil {
		t.Fatal("the saved mod does not check clean:", err)
	}
	return d
}

func saveMod(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	if err := pastNotice(e, "saveas"); err != nil {
		t.Fatal(err)
	}
}

func rowOf(d mod.MissionData, rel string) (mod.Mission, bool) {
	for _, m := range d.Missions {
		if m.Map == rel {
			return m, true
		}
	}
	return mod.Mission{}, false
}

// Make mission turns the unused map into a guest row of the editor's default
// guest, or into a replacement of a campaign number the install holds, in one
// Undo step; the saved mod checks clean and reopens with the row.
func TestMakeMissionOfAnUnusedMapGuestAndReplace(t *testing.T) {
	for _, lang := range makeMapLangs {
		t.Run("lang="+lang.book, func(t *testing.T) {
			f, e, spare, dir := makeMapEditor(t, lang.book)
			a, ok := actionOf(t, spare, "maps/spare.alm")
			if !ok || len(a.Buttons) < 1 || a.Buttons[0].Key != "makemission" {
				t.Fatalf("no Make mission for the unused map: %+v", a)
			}
			if _, err := spare.SetWorkspaceField("package", "makemission", "maps/spare.alm"); err != nil {
				t.Fatal(err)
			}
			d := spare.workspace.Project
			row := d.Missions[len(d.Missions)-1]
			if row.Map != "maps/spare.alm" || row.Kind != mod.MissionGuest || row.Key != "spare" || len(row.Party) != 1 || d.Selected != len(d.Missions)-1 || d.Loose != "" {
				t.Fatalf("the guest row %+v, selected %d, loose %q", row, d.Selected, d.Loose)
			}
			if _, ok := actionOf(t, spare, "maps/spare.alm"); ok {
				t.Fatal("the unused entry stays after Make mission")
			}
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			if _, ok := actionOf(t, spare, "maps/spare.alm"); !ok || len(spare.workspace.Project.Missions) != len(d.Missions)-1 {
				t.Fatal("one Undo did not take the row away")
			}
			if err := e.Key("redo"); err != nil {
				t.Fatal(err)
			}
			saveMod(t, e)
			saved, ok := rowOf(reopened(t, f, dir), "maps/spare.alm")
			if !ok || saved.Kind != mod.MissionGuest || saved.Number != row.Number {
				t.Fatalf("the reopened row %+v", saved)
			}
		})
	}
}

func TestMakeMissionReplacesACampaignNumberTheInstallHolds(t *testing.T) {
	for _, lang := range makeMapLangs {
		t.Run("lang="+lang.book, func(t *testing.T) {
			f, e, spare, dir := makeMapEditor(t, lang.book)
			raw := spare.model.Bytes()
			f.x.archives.Containers.Overlay("scenario/30.alm", func() ([]byte, error) { return raw, nil })
			f.x.missionHeld = nil
			const n = 30
			a, _ := actionOf(t, spare, "maps/spare.alm")
			if len(a.Buttons) != 2 || !strings.Contains(a.Buttons[1].Caption, strconv.Itoa(n)) || a.Buttons[1].Key != "makemission" {
				t.Fatalf("the unused map offers %+v", a.Buttons)
			}
			if _, err := spare.SetWorkspaceField("package", "makemission", a.Buttons[1].Value); err != nil {
				t.Fatal(err)
			}
			saveMod(t, e)
			saved, ok := rowOf(reopened(t, f, dir), "maps/spare.alm")
			if !ok || saved.Kind != mod.MissionReplace || saved.Number != n {
				t.Fatalf("the reopened row %+v", saved)
			}
		})
	}
}

// A map that replaces a mission by its file name gets a row that names the
// same map under the same number.
func TestMakeRowNamesTheMapThatReplacesByFileName(t *testing.T) {
	f, e, spare, dir := makeMapEditor(t, "")
	a, ok := actionOf(t, spare, "20.alm")
	if !ok || len(a.Buttons) != 1 || a.Buttons[0].Key != "makerow" {
		t.Fatalf("the by-name map offers %+v", a)
	}
	if _, err := spare.SetWorkspaceField("package", "makerow", a.Buttons[0].Value); err != nil {
		t.Fatal(err)
	}
	if _, ok := actionOf(t, spare, "20.alm"); ok {
		t.Fatal("the by-name entry stays after Make row")
	}
	saveMod(t, e)
	saved, ok := rowOf(reopened(t, f, dir), "20.alm")
	if !ok || saved.Kind != mod.MissionReplace || saved.Number != 20 || saved.ByFileName() {
		t.Fatalf("the reopened row %+v", saved)
	}
}

// Each refusal speaks the editor's words and leaves no draft and no history.
func TestMapActionsRefuseInTheEditorsWordsAndLeaveNoDraft(t *testing.T) {
	for _, lang := range makeMapLangs {
		t.Run("lang="+lang.book, func(t *testing.T) {
			_, _, spare, _ := makeMapEditor(t, lang.book)
			before, steps := spare.workspaceImage, len(spare.undoWorkspace)
			for _, c := range []struct{ key, value, want string }{
				{"makemission", spare.modMaps()[0].Path, lang.taken},
				{"makemission", "999|maps/spare.alm", lang.number},
				{"makerow", "maps/spare.alm", ""},
				{"makemission", "maps/none.alm", ""},
				{"addmaps", filepath.Join(t.TempDir(), "absent"), ""},
				{"addmaps", t.TempDir(), ""},
			} {
				_, err := spare.SetWorkspaceField("package", c.key, c.value)
				if err == nil || !strings.Contains(err.Error(), c.want) {
					t.Fatalf("%s %q: %v", c.key, c.value, err)
				}
			}
			if string(spare.workspaceImage) != string(before) || len(spare.undoWorkspace) != steps {
				t.Fatal("a refusal left a draft or a history step")
			}
		})
	}
}

// Add map copies the open map under its own stem, with a -2 when the stem is
// held, as a draft no mission names, in one Undo step.
func TestAddMapCopiesTheOpenMapUnderItsOwnStem(t *testing.T) {
	_, e, spare, _ := makeMapEditor(t, "")
	if _, err := spare.SetWorkspaceField("package", "addmap", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := spare.SetWorkspaceField("package", "addmap", ""); err != nil {
		t.Fatal(err)
	}
	d := spare.workspace.Project
	if d.Files["maps/spare-2.alm"] == nil {
		t.Fatalf("the copy is not at its own stem: %v", slices.Sorted(mapKeys(d.Files)))
	}
	second := "maps/spare-3.alm"
	if d.Files[second] == nil {
		t.Fatalf("the second copy takes no -3: %v", slices.Sorted(mapKeys(d.Files)))
	}
	if _, ok := actionOf(t, spare, second); !ok {
		t.Fatal("the copy is not an unused entry")
	}
	if err := e.Key("undo"); err != nil || spare.workspace.Project.Files[second] != nil {
		t.Fatal("one Undo did not take the second copy away:", err)
	}
}

// Add maps from folder adds every map of a folder with its pages and labels
// as one Undo step, and the saved mod reopens with the seven maps unused.
func TestAddMapsFromFolderAddsSevenMapsInOneUndoStep(t *testing.T) {
	f, e, spare, dir := makeMapEditor(t, "")
	raw := spare.model.Bytes()
	folder := t.TempDir()
	for _, stem := range []string{"alpha", "beta", "gamma", "delta", "eps", "zeta", "eta"} {
		for name, body := range map[string][]byte{stem + ".alm": raw, stem + ".text/en/title.txt": []byte("Title " + stem), stem + ".variables": []byte{}} {
			p := filepath.Join(folder, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, body, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	steps := len(spare.undoWorkspace)
	if _, err := spare.SetWorkspaceField("package", "addmaps", folder); err != nil {
		t.Fatal(err)
	}
	if len(spare.undoWorkspace) != steps+1 {
		t.Fatalf("the folder took %d steps", len(spare.undoWorkspace)-steps)
	}
	d := spare.workspace.Project
	for _, stem := range []string{"alpha", "beta", "gamma", "delta", "eps", "zeta", "eta"} {
		for _, rel := range []string{"maps/" + stem + ".alm", "maps/" + stem + ".text/en/title.txt"} {
			if d.Files[rel] == nil {
				t.Fatalf("%s is not a draft", rel)
			}
		}
		if _, ok := d.Files["maps/"+stem+".variables"]; !ok {
			t.Fatalf("%s has no labels draft", stem)
		}
		if _, ok := actionOf(t, spare, "maps/"+stem+".alm"); !ok {
			t.Fatalf("%s is not an unused entry", stem)
		}
	}
	if err := e.Key("undo"); err != nil || spare.workspace.Project.Files["maps/alpha.alm"] != nil || spare.workspace.Project.Files["maps/eta.text/en/title.txt"] != nil {
		t.Fatal("one Undo did not take the folder away:", err)
	}
	if err := e.Key("redo"); err != nil {
		t.Fatal(err)
	}
	saveMod(t, e)
	got := tree(t, dir)
	for _, rel := range []string{"maps/alpha.alm", "maps/eta.text/en/title.txt", "maps/zeta.variables"} {
		if _, ok := got[rel]; !ok {
			t.Fatalf("the save did not write %s", rel)
		}
	}
	list := mod.MapList(slices.Sorted(mapKeys(stringBytes(got))), "ambush", reopened(t, f, dir))
	unused := 0
	for _, m := range list {
		if m.Role() == mod.MapUnused {
			unused++
		}
	}
	if unused != 8 {
		t.Fatalf("the reopened mod has %d unused maps, want 8", unused)
	}
}

func stringBytes(m map[string]string) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range m {
		out[k] = []byte(v)
	}
	return out
}

// F5 on an unused map plays it as a guest mission of the mod, with the mod
// active, and writes no row into the project.
func TestTestPlayOfAnUnusedMapStartsItWithTheModActive(t *testing.T) {
	f, _, spare, _ := makeMapEditor(t, "")
	rows := len(spare.workspace.Project.Missions)
	if !spare.testPlayTakesParty() {
		t.Fatal("the party window does not open for an unused map")
	}
	if _, err := spare.TestPlay(); err != nil {
		t.Fatal("test play of an unused map:", err)
	}
	if len(f.started) != 1 {
		t.Fatalf("the game started %d times", len(f.started))
	}
	if got := len(spare.workspace.Project.Missions); got != rows || spare.workspace.Project.Selected != -1 {
		t.Fatalf("test play changed the project: %d rows of %d, selected %d", got, rows, spare.workspace.Project.Selected)
	}
	args := strings.Join(f.started[0], " ")
	if !strings.Contains(args, "ambush") {
		t.Fatalf("the game starts without the mod: %s", args)
	}
}

func hasReplacing(d mod.MissionData, n int) bool {
	_, ok := d.Replacing(n)
	return ok
}
