package game

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// modMapsMod writes a mod whose maps have each role: one map two rows name,
// one a row names with no file, one named by its file name, one no mission
// names. It opens the editor's own copy of it in W and answers its folder.
func modMapsMod(t *testing.T, e *ui.MapEditor, raw []byte) string {
	t.Helper()
	var missions strings.Builder
	for i, name := range []string{"map1.alm", "map1.alm", "gone.alm"} {
		fmt.Fprintf(&missions, "[[mission]]\nkey='m%d'\nnumber=%d\nmap='%s'\nafter=%d\nparty=[{character='npc22'}]\n", i+1, 21+i, name, 20+i)
	}
	dir := writeEditorMod(t, t.TempDir(), missions.String())
	for _, name := range []string{"map1.alm", "gone.alm", "20.alm", "maps/spare.alm"} {
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), raw)
	}
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	for _, folder := range []string{dir, own} {
		if err := workspaceField(e, "package", 0, folder); err != nil {
			t.Fatal(err)
		}
	}
	// The third row's file is taken out of the project.
	d := e.Document().Edits.(*mapInspectionEdits).workspace.Project
	d.Files[openMapsFile(t, tree(t, own), "gone.alm")] = nil
	return own
}

// modMapsField is W's map list as the host gives it.
func modMapsField(t *testing.T, e *ui.MapEditor) ui.InspectionField {
	t.Helper()
	f := e.Document().Edits.(ui.InspectionModMaps).ModMaps()
	if f.Key != "map" || len(f.Choices) == 0 {
		t.Fatal("W has no map list")
	}
	return f
}

// modMapsRows is the list's rows, the map shown in brackets.
func modMapsRows(t *testing.T, e *ui.MapEditor) string {
	t.Helper()
	f := modMapsField(t, e)
	var out []string
	for _, c := range f.Choices {
		s := c.Label
		if c.Value == f.Value {
			s = "[" + s + "]"
		}
		out = append(out, s)
	}
	return strings.Join(out, " | ")
}

// modMapsPage turns W, opened when it is shut, to its page of the mod's maps:
// the window's key turns its pages, and that page is the one whose first row
// carries an Open button.
func modMapsPage(t *testing.T, e *ui.MapEditor) {
	t.Helper()
	for turn := 0; !e.ModWindowOpen() || len(e.PopupRowButtons(0)) != 1; turn++ {
		if err := e.Key("package"); err != nil || turn > 5 {
			t.Fatalf("W has no page of the mod's maps: %v\n%s", err, strings.Join(e.PopupLines(), "\n"))
		}
	}
}

// modMapsOpen opens row n of the map list, as the list's Open button does.
func modMapsOpen(t *testing.T, e *ui.MapEditor, n int) error {
	t.Helper()
	if !e.ModWindowOpen() {
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
	}
	return e.OpenModMap(n)
}

// W lists every map of the mod once with its role, in the editor's
// language. A map no row names opens as a tab through the switch the
// mission choice uses; its change is a draft the project keeps, with its
// Undo, across a switch to a map with a row and back; Save writes it at its
// own path and no row.
func TestEditorListsAndOpensEveryMapOfTheMod(t *testing.T) {
	for _, lang := range []struct{ book, group, rows, missing, play string }{
		{"", "Maps of the mod", "[map1.alm: guest 21 after 20, guest 22 after 21] | gone.alm: missing | 20.alm: replaces 20 by name | spare.alm: unused",
			"holds no file of this map", "no mission of the mod names this map"},
		{"ru", "Карты мода", "[map1.alm: гостевая 21 после 20, гостевая 22 после 21] | gone.alm: файла нет | 20.alm: заменяет 20 по имени | spare.alm: не используется",
			"нет файла этой карты", "ни одна миссия"},
	} {
		t.Run("lang="+lang.book, func(t *testing.T) {
			f, e := workspaceFixture(t)
			f.x.archives.Base.Profile.Language = map[string]string{"": "english", "ru": "russian"}[lang.book]
			e.SetWords(words.For(lang.book))
			e.Layout(1280, 800)
			dir := modMapsMod(t, e, e.Document().SourceBytes)
			m1 := openMapsFile(t, tree(t, dir), "map1.alm")
			if got := modMapsRows(t, e); got != lang.rows {
				t.Fatalf("the map list:\n got %q\nwant %q", got, lang.rows)
			}
			if err := modMapsOpen(t, e, 2); err == nil || !strings.Contains(err.Error(), lang.missing) {
				t.Fatal("a map with no file opened:", err)
			}

			// The unused map opens as a second tab and takes a change.
			if err := modMapsOpen(t, e, 4); err != nil {
				t.Fatal(err)
			}
			if e.OpenMapCount() != 2 || e.ShownMap() != 1 || !strings.Contains(modMapsRows(t, e), "[spare.alm") {
				t.Fatalf("the unused map: %d maps, shown %d, %s", e.OpenMapCount(), e.ShownMap(), modMapsRows(t, e))
			}
			spare := e.Document().Edits.(*mapInspectionEdits)
			if d := spare.workspace.Project; d.Selected != -1 || d.Loose != "maps/spare.alm" || len(d.Missions) < 3 {
				t.Fatalf("the unused map's project: row %d, path %q, %d rows", d.Selected, d.Loose, len(d.Missions))
			}
			if err := workspaceField(e, "metadata", 0, "Spare"); err != nil {
				t.Fatal(err)
			}
			if _, err := spare.TestPlay(); err == nil || !strings.Contains(err.Error(), lang.play) {
				t.Fatal("test play of a map no mission names:", err)
			}

			// To the map with rows: the project keeps the unused map's draft.
			if err := modMapsOpen(t, e, 1); err != nil {
				t.Fatal(err)
			}
			first := e.Document().Edits.(*mapInspectionEdits)
			if e.OpenMapCount() != 2 || e.ShownMap() != 0 || first == spare || first.workspace.Project.Files["maps/spare.alm"] == nil {
				t.Fatalf("back on the first map: %d maps, shown %d", e.OpenMapCount(), e.ShownMap())
			}
			if err := workspaceField(e, "metadata", 0, "First"); err != nil {
				t.Fatal(err)
			}

			// And back: the change, its Undo and the first map's draft.
			if err := modMapsOpen(t, e, 4); err != nil {
				t.Fatal(err)
			}
			if e.Document().Edits != ui.InspectionEdits(spare) || e.Document().Name != "Spare" || e.OpenMapCount() != 2 {
				t.Fatal("the unused map did not come back as it was left:", e.Document().Name)
			}
			if d := spare.workspace.Project; d.Loose != "maps/spare.alm" || d.Files[m1] == nil || d.Files["maps/spare.alm"] == nil {
				t.Fatal("the project lost a draft on the way back")
			}
			_ = e.Key("escape")
			if err := e.Key("undo"); err != nil || e.Document().Name == "Spare" {
				t.Fatal("undo on the unused map:", err, e.Document().Name)
			}
			if d := spare.workspace.Project; d.Files[m1] == nil {
				t.Fatal("undo on the unused map dropped the first map's draft")
			}
			if err := e.Key("redo"); err != nil || e.Document().Name != "Spare" {
				t.Fatal("redo on the unused map:", err, e.Document().Name)
			}

			// Save writes the unused map at its own path and adds no row.
			// The save refuses a row with no file; the file comes back first.
			if err := pastNotice(e, "saveas"); err == nil || !strings.Contains(err.Error(), "gone.alm") {
				t.Fatal("a save with a row's map missing:", err)
			}
			delete(spare.workspace.Project.Files, openMapsFile(t, tree(t, dir), "gone.alm"))
			before := tree(t, dir)
			if err := pastNotice(e, "saveas"); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			if after["maps/spare.alm"] == before["maps/spare.alm"] || after["data/missions.toml"] != before["data/missions.toml"] {
				for name := range after {
					if after[name] != before[name] {
						t.Log("changed:", name)
					}
				}
				t.Fatal("save of the unused map did not write its file alone", e.Message())
			}
			if got := modMapsRows(t, e); !strings.Contains(got, "[spare.alm") || strings.Count(got, "|") != 3 {
				t.Fatal("the list after the save:", got)
			}

			// The Open window leads with the mod's maps. One already open is
			// shown; another opens as a third tab; the project stays.
			choose := func(base string) {
				t.Helper()
				if e.ModWindowOpen() {
					_ = e.Key("escape")
				}
				if err := e.Key("catalog"); err != nil {
					t.Fatal(err)
				}
				lines := e.ListLines()
				if len(lines) < 5 || strings.TrimSpace(lines[0]) != f.x.inspectionUTF8(lang.group) {
					t.Fatalf("the Open window's first group: %q", lines)
				}
				for n, line := range lines[:5] {
					if !strings.Contains(line, base) {
						continue
					}
					r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlListRow, N: n})
					if !ok {
						t.Fatalf("the Open window shows no row %d", n)
					}
					openMapsClick(e, r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)))
					if err := e.Key("enter"); err != nil {
						t.Fatal(err)
					}
					return
				}
				t.Fatalf("the Open window lists no %s: %q", base, lines)
			}
			choose("map1.alm")
			if e.OpenMapCount() != 2 || e.ShownMap() != 0 || e.Document().Edits != ui.InspectionEdits(first) {
				t.Fatalf("the Open window on an open map: %d maps, shown %d", e.OpenMapCount(), e.ShownMap())
			}
			choose("20.alm")
			third := e.Document().Edits.(*mapInspectionEdits)
			if e.OpenMapCount() != 3 || e.ShownMap() != 2 || !third.InMod() || len(third.workspace.Project.Missions) != len(first.workspace.Project.Missions) {
				t.Fatalf("the Open window on a third map: %d maps, shown %d", e.OpenMapCount(), e.ShownMap())
			}
			if lines := e.ListLines(); len(lines) > 0 && strings.TrimSpace(lines[0]) == f.x.inspectionUTF8(lang.group) {
				t.Fatal("the Open window stayed after its choice")
			}

			// Ctrl+Tab and Ctrl+Shift+Tab turn the tabs and wrap.
			var seen []int
			for _, key := range []string{"nextmap", "nextmap", "nextmap", "prevmap", "prevmap"} {
				if err := e.Key(key); err != nil {
					t.Fatal(err)
				}
				seen = append(seen, e.ShownMap())
			}
			if fmt.Sprint(seen) != "[0 1 2 1 0]" || e.Document().Edits != ui.InspectionEdits(first) {
				t.Fatal("the tab keys showed", seen)
			}
			if err := e.Key("prevmap"); err != nil || e.ShownMap() != 2 {
				t.Fatal("Ctrl+Shift+Tab from the first tab:", err, e.ShownMap())
			}
		})
	}
}
