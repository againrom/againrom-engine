package game

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// openMapsNamed opens a mod of count maps, each with a tab, named Map1..
// and saved, so each file names its own map. It answers the mod's folder.
func openMapsNamed(t *testing.T, e *ui.MapEditor, count int) string {
	t.Helper()
	dir := openMapsMod(t, e, count)
	for n := 1; n <= count; n++ {
		if err := workspaceField(e, "package", 9, fmt.Sprint(n)); err != nil {
			t.Fatal(err)
		}
		if err := workspaceField(e, "metadata", 0, fmt.Sprintf("Map%d", n)); err != nil {
			t.Fatal(err)
		}
		if err := pastNotice(e, "saveas"); err != nil {
			t.Fatal(err, e.Message())
		}
	}
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	return dir
}

// openMapsHeld is the map names each map file of the mod at dir holds, as
// "map1.alm=Map1 map2.alm=Map2".
func openMapsHeld(t *testing.T, dir string, count int) string {
	t.Helper()
	files := tree(t, dir)
	var out []string
	for n := 1; n <= count; n++ {
		base := fmt.Sprintf("map%d.alm", n)
		held := base + "="
		for name, b := range files {
			if name != base && !strings.HasSuffix(name, "/"+base) {
				continue
			}
			for _, word := range []string{"Map", "Edit", "Dup"} {
				for k := 1; k <= count; k++ {
					if w := fmt.Sprintf("%s%d", word, k); strings.Contains(b, w) {
						held += w
					}
				}
			}
		}
		out = append(out, held)
	}
	return strings.Join(out, " ")
}

// A mission removed on its tab comes back by Undo after another tab was
// shown: the tab holds the removed mission's map under its own row again,
// and Save leaves every map file with its own map.
func TestEditorUndoOfARemovedMissionAfterATabSwitch(t *testing.T) {
	for _, tc := range []struct {
		name        string
		tab         int
		state, held string
	}{
		{"the last mission", 1, "21 [23]", "Map3"},
		{"the first mission", 0, "[21] 23", "Map1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, e := workspaceFixture(t)
			e.SetWords(words.For(""))
			e.Layout(1280, 800)
			dir := openMapsNamed(t, e, 3)
			openMapsTab(t, e, 1)
			e.CloseOpenMap()
			openMapsTab(t, e, tc.tab)
			if err := e.Key("package"); err != nil {
				t.Fatal(err)
			}
			if !e.FocusModField("remove") {
				t.Fatal("the mod window has no remove button")
			}
			rows := func() int { return len(e.Document().Edits.(*mapInspectionEdits).workspace.Project.Missions) }
			for press := 0; press < 2 && rows() == 3; press++ {
				_ = e.Key("enter")
			}
			if rows() != 2 {
				t.Fatalf("the remove left %d missions", rows())
			}
			_ = e.Key("escape")
			back := e.ShownMap()
			openMapsTab(t, e, 1-back)
			openMapsTab(t, e, back)
			if err := e.Key("undo"); err != nil {
				t.Fatal(err)
			}
			s := e.Document().Edits.(*mapInspectionEdits)
			if got := openMapsState(e); strings.ReplaceAll(got, "*", "") != tc.state || e.Document().Name != tc.held || rows() != 3 {
				t.Fatalf("undo after the tab switch: tabs %s, map %q of %s, %d missions; want %s, %q, 3",
					got, e.Document().Name, filepath.Base(s.source), rows(), tc.state, tc.held)
			}
			if m, ok := s.ownMission(s.project0()); !ok || filepath.Base(m.Map) != filepath.Base(s.source) {
				t.Fatalf("the tab holds %s under the row of %s", filepath.Base(s.source), m.Map)
			}
			if err := pastNotice(e, "saveas"); err != nil {
				t.Fatal(err, e.Message())
			}
			if got, want := openMapsHeld(t, dir, 3), "map1.alm=Map1 map2.alm=Map2 map3.alm=Map3"; got != want {
				t.Fatalf("after the save the files hold %s, want %s", got, want)
			}
			if b := tree(t, dir)["data/missions.toml"]; strings.Count(b, "[[mission]]") != 3 {
				t.Fatalf("after the save the mod declares\n%s", b)
			}
		})
	}
}

// A mod saved into another folder keeps one tab per map: every open map is
// then the new folder's, the mission list and Open by path show its tab, and
// a save of each tab writes its own file once.
func TestEditorOpenMapsFollowTheModToAnotherFolder(t *testing.T) {
	_, e := workspaceFixture(t)
	e.SetWords(words.For(""))
	e.Layout(1280, 800)
	dir := openMapsNamed(t, e, 2)
	other := filepath.Join(t.TempDir(), filepath.Base(dir))
	if err := workspaceField(e, "package", 0, other); err != nil {
		t.Fatal(err, e.Message())
	}
	for _, r := range e.OpenMapRows() {
		if rel, err := filepath.Rel(other, r.Path); err != nil || !filepath.IsLocal(rel) {
			t.Fatalf("after the save into %s the tab of mission %d is %s", other, r.Mission, r.Path)
		}
	}
	if err := workspaceField(e, "package", 9, "1"); err != nil {
		t.Fatal(err)
	}
	if got := openMapsState(e); got != "[21] 22" {
		t.Fatalf("mission 1 from the list: tabs %s, want [21] 22", got)
	}
	_ = e.Key("escape")
	if err := e.Open(filepath.Join(other, filepath.FromSlash(openMapsFile(t, tree(t, other), "map2.alm")))); err != nil {
		t.Fatal(err)
	}
	if got := openMapsState(e); got != "21 [22]" {
		t.Fatalf("map 2 opened by its path: tabs %s, want 21 [22]", got)
	}
	for n, name := range []string{"Edit1", "Edit2"} {
		openMapsTab(t, e, n)
		if err := workspaceField(e, "metadata", 0, name); err != nil {
			t.Fatal(err)
		}
		if err := pastNotice(e, "saveas"); err != nil {
			t.Fatal(err, e.Message())
		}
	}
	if got, want := openMapsHeld(t, other, 2), "map1.alm=Edit1 map2.alm=Edit2"; got != want {
		t.Fatalf("the new folder holds %s, want %s", got, want)
	}
	if got, want := openMapsHeld(t, dir, 2), "map1.alm=Map1 map2.alm=Map2"; got != want {
		t.Fatalf("the folder left holds %s, want %s", got, want)
	}
}

// An imported mod is read-only and saves only as a copy under another
// parent. The copy moves its maps; a changed map's tab is the one Open by
// the new path shows, with its unsaved change.
func TestEditorOpenMapsFollowAnImportedModsCopy(t *testing.T) {
	_, e := workspaceFixture(t)
	e.SetWords(words.For(""))
	e.Layout(1280, 800)
	var missions strings.Builder
	for i := range 2 {
		fmt.Fprintf(&missions, "[[mission]]\nkey='m%d'\nnumber=%d\nmap='map%d.alm'\nafter=%d\nparty=[{character='npc22'}]\n", i+1, 21+i, i+1, 20+i)
	}
	dir := writeEditorMod(t, t.TempDir(), missions.String())
	for i := range 2 {
		writeTestPlayInput(t, filepath.Join(dir, fmt.Sprintf("map%d.alm", i+1)), e.Document().SourceBytes)
	}
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 9, "2"); err != nil {
		t.Fatal(err)
	}
	openMapsTab(t, e, 0)
	if err := workspaceField(e, "metadata", 0, "Edit1"); err != nil {
		t.Fatal(err)
	}
	openMapsTab(t, e, 1)
	other := filepath.Join(t.TempDir(), filepath.Base(dir))
	if err := workspaceField(e, "package", 0, other); err != nil {
		t.Fatal(err, e.Message())
	}
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	if err := e.Open(filepath.Join(other, filepath.FromSlash(openMapsFile(t, tree(t, other), "map1.alm")))); err != nil {
		t.Fatal(err)
	}
	if got := openMapsState(e); got != "[21*] 22" || e.Document().Name != "Edit1" {
		t.Fatalf("map 1 opened by its new path: tabs %s showing %q, want [21*] 22 showing Edit1", got, e.Document().Name)
	}
}
