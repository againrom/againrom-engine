package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// openMapsMod writes a mod of count missions, numbered from 21, each on its
// own copy of the editor's open map, and opens the editor's own copy of it
// in W. It answers the copy's folder.
func openMapsMod(t *testing.T, e *ui.MapEditor, count int) string {
	t.Helper()
	var missions strings.Builder
	for i := range count {
		fmt.Fprintf(&missions, "[[mission]]\nkey='m%d'\nnumber=%d\nmap='map%d.alm'\nafter=%d\nparty=[{character='npc22'}]\n", i+1, 21+i, i+1, 20+i)
	}
	dir := writeEditorMod(t, t.TempDir(), missions.String())
	for i := range count {
		writeTestPlayInput(t, filepath.Join(dir, fmt.Sprintf("map%d.alm", i+1)), e.Document().SourceBytes)
	}
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	for _, folder := range []string{dir, own} {
		if err := workspaceField(e, "package", 0, folder); err != nil {
			t.Fatal(err)
		}
	}
	return own
}

// openMapsFile is the name in tree files of the map file named base.
func openMapsFile(t *testing.T, files map[string]string, base string) string {
	t.Helper()
	for name := range files {
		if name == base || strings.HasSuffix(name, "/"+base) {
			return name
		}
	}
	t.Fatalf("the mod holds no %s", base)
	return ""
}

// openMapsClick presses and releases the left button at p.
func openMapsClick(e *ui.MapEditor, p image.Point) {
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Unix(1, 0))
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
}

// openMapsTab clicks tab n of the strip.
func openMapsTab(t *testing.T, e *ui.MapEditor, n int) {
	t.Helper()
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlMapTab, N: n})
	if !ok {
		t.Fatalf("the strip has %d tabs, no tab %d", openMapsTabs(e), n)
	}
	openMapsClick(e, r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)))
	if e.ShownMap() != n {
		t.Fatalf("a click on tab %d shows map %d", n, e.ShownMap())
	}
}

// openMapsTabs counts the strip's tabs.
func openMapsTabs(e *ui.MapEditor) int {
	n := 0
	for ; ; n++ {
		if _, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlMapTab, N: n}); !ok {
			return n
		}
	}
}

// openMapsState is each open map's mission number, with a star on the ones
// with unsaved changes of their own and brackets on the one shown.
func openMapsState(e *ui.MapEditor) string {
	var out []string
	for _, r := range e.OpenMapRows() {
		s := fmt.Sprint(r.Mission)
		if r.Dirty != strings.HasSuffix(r.Caption, " *") || !strings.HasPrefix(r.Caption, s+". ") {
			s += "?" + r.Caption
		}
		if r.Dirty {
			s += "*"
		}
		if r.Shown {
			s = "[" + s + "]"
		}
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// openMapsAnswer presses button n of the open question.
func openMapsAnswer(t *testing.T, e *ui.MapEditor, n int) {
	t.Helper()
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPromptButton, N: n})
	if !ok {
		t.Fatalf("the question has no button %d: %q %q", n, e.PromptTitle(), e.PromptNotes())
	}
	openMapsClick(e, r.Min.Add(image.Pt(r.Dx()/2, r.Dy()/2)))
}

// Three maps of one mod stay open as tabs. Each keeps its view, its changes
// and its undo history; Save writes the map shown and no other map's file; a
// map already open is shown, never opened twice; closing a tab and leaving
// the mod ask the unsaved-changes question about each changed map.
func TestEditorKeepsSeveralMapsOfOneModOpen(t *testing.T) {
	for _, lang := range []struct{ book, name, closeTitle, changed string }{
		{"", "Ford", "Close the map", "%s has unsaved changes."},
		{"ru", "Брод", "Закрыть карту", ""},
	} {
		t.Run("lang="+lang.book, func(t *testing.T) {
			_, e := workspaceFixture(t)
			e.SetWords(words.For(lang.book))
			e.Layout(1280, 800)
			if openMapsTabs(e) != 0 || e.OpenMapCount() != 1 {
				t.Fatal("one map with no mod shows a strip")
			}
			dir := openMapsMod(t, e, 3)
			if openMapsTabs(e) != 0 || openMapsState(e) != "[21]" {
				t.Fatalf("the mod's first map: %d tabs, %s", openMapsTabs(e), openMapsState(e))
			}
			x1, y1, z1 := e.MapView()
			for _, n := range []string{"2", "3"} {
				if err := workspaceField(e, "package", 9, n); err != nil {
					t.Fatal(err)
				}
			}
			if openMapsTabs(e) != 3 || openMapsState(e) != "21 22 [23]" {
				t.Fatalf("three maps open: %d tabs, %s", openMapsTabs(e), openMapsState(e))
			}
			name := func(n int) string { return fmt.Sprintf("%s %d", lang.name, n) }
			rename := func(n int) {
				t.Helper()
				if err := workspaceField(e, "metadata", 0, name(n)); err != nil {
					t.Fatal(err)
				}
			}

			// The third map: its own view and its own change.
			_ = e.Key("escape")
			fx, fy, fz := e.MapView()
			e.Pointer(ui.Input{CursorX: 900, CursorY: 500, WheelY: 2}, time.Unix(1, 0))
			e.Pointer(ui.Input{CursorX: 900, CursorY: 500}, time.Unix(1, 0))
			x3, y3, z3 := e.MapView()
			if x3 == fx && y3 == fy && z3 == fz {
				t.Fatal("the wheel did not move the third map's view")
			}
			rename(3)
			third := e.Document().Edits
			if !e.Select(0, false) || e.Selected() != 0 {
				t.Fatal("the third map has no record to select")
			}
			if openMapsState(e) != "21 22 [23*]" {
				t.Fatal("the third map's change:", openMapsState(e))
			}

			// The first map has its own view, and takes its own change.
			openMapsTab(t, e, 0)
			if x, y, z := e.MapView(); x != x1 || y != y1 || z != z1 || (x == x3 && y == y3 && z == z3) || e.Selected() == 0 {
				t.Fatal("the first map took the third map's view or selection")
			}
			rename(1)
			if openMapsState(e) != "[21*] 22 23*" || e.Document().Name != name(1) {
				t.Fatal("the first map's change:", openMapsState(e), e.Document().Name)
			}
			first := e.Document().Edits.(*mapInspectionEdits)

			// Back on the third: its view, its change and its own history.
			openMapsTab(t, e, 2)
			if x, y, z := e.MapView(); x != x3 || y != y3 || z != z3 || e.Document().Edits != third || e.Document().Name != name(3) || e.Selected() != 0 {
				t.Fatal("the third map did not come back as it was left")
			}
			// The one project came with the map shown: W, the graph and the
			// checker read every open map's draft from it.
			held := third.(*mapInspectionEdits).workspace.Project
			if draft := held.Files[held.Missions[0].Map]; len(held.Missions) != 3 || !bytes.Equal(draft, first.model.Bytes()) || bytes.Equal(draft, first.saved) {
				t.Fatal("the project of the map shown does not hold the first map's draft")
			}
			if err := e.Key("undo"); err != nil || e.Document().Name == name(3) || openMapsState(e) != "21* 22 [23]" {
				t.Fatal("undo on the third map:", err, e.Document().Name, openMapsState(e))
			}
			if err := e.Key("redo"); err != nil || e.Document().Name != name(3) || openMapsState(e) != "21* 22 [23*]" {
				t.Fatal("redo on the third map:", err, e.Document().Name, openMapsState(e))
			}

			// Save writes the map shown; the first map's file stays.
			before := tree(t, dir)
			if err := pastNotice(e, "saveas"); err != nil {
				t.Fatal(err, e.Message())
			}
			after := tree(t, dir)
			var changed []string
			for file := range after {
				if after[file] != before[file] {
					changed = append(changed, file)
				}
			}
			slices.Sort(changed)
			if !slices.Equal(changed, []string{openMapsFile(t, before, "map3.alm")}) || len(after) != len(before) {
				t.Fatal("Save on the third map changed", changed, e.Message())
			}
			if openMapsState(e) != "21* 22 [23]" || !e.Dirty() {
				t.Fatal("after Save on the third map:", openMapsState(e), e.Dirty())
			}

			// A map already open is shown, from the mission list and by its file.
			if err := workspaceField(e, "package", 9, "1"); err != nil || openMapsState(e) != "[21*] 22 23" {
				t.Fatal("the mission list on an open map:", err, openMapsState(e))
			}
			if err := e.Open(e.OpenMapRows()[1].Path); err != nil || openMapsState(e) != "21* [22] 23" {
				t.Fatal("Open on an open map:", err, openMapsState(e))
			}
			rename(2)
			openMapsTab(t, e, 2)
			rename(4)
			if err := e.Key("undo"); err != nil || openMapsState(e) != "21* 22* [23]" {
				t.Fatal("undo after the save:", err, openMapsState(e))
			}
			if err := e.Key("redo"); err != nil || openMapsState(e) != "21* 22* [23*]" {
				t.Fatal("redo after the save:", err, openMapsState(e))
			}

			// Closing a tab with changes asks the unsaved-changes question.
			openMapsTab(t, e, 0)
			closeAt, _ := e.FindControl(ui.ControlQuery{Role: ui.ControlMapTab, Flag: true})
			closeTab := closeAt.Min.Add(image.Pt(closeAt.Dx()/2, closeAt.Dy()/2))
			openMapsClick(e, closeTab)
			asked := func(file string) bool {
				// The Russian line is in the font's bytes; the file's name reads in both.
				notes := e.PromptNotes()
				return len(notes) == 1 && strings.Contains(notes[0], file) && (lang.book != "" || notes[0] == fmt.Sprintf(lang.changed, file))
			}
			if e.PromptTitle() != lang.closeTitle || !asked("map1.alm") || e.OpenMapCount() != 3 {
				t.Fatalf("closing a changed tab: %q %q", e.PromptTitle(), e.PromptNotes())
			}
			_ = e.Key("escape")
			if _, open := e.PromptRect(); open || openMapsState(e) != "[21*] 22* 23*" {
				t.Fatal("Cancel closed the tab:", openMapsState(e))
			}
			openMapsClick(e, closeTab)
			openMapsAnswer(t, e, 1)
			if _, open := e.PromptRect(); open || openMapsState(e) != "[22*] 23*" {
				t.Fatal("closing without saving:", openMapsState(e))
			}
			// A closed tab's map opens again from its file, as it was saved.
			if err := workspaceField(e, "package", 9, "1"); err != nil || openMapsState(e) != "22* 23* [21]" || e.Document().Name == name(1) {
				t.Fatal("the closed map opened again:", err, openMapsState(e), e.Document().Name)
			}
			_ = e.Key("escape")
			closeAt, _ = e.FindControl(ui.ControlQuery{Role: ui.ControlMapTab, Flag: true})
			openMapsClick(e, closeAt.Min.Add(image.Pt(closeAt.Dx()/2, closeAt.Dy()/2)))
			if _, open := e.PromptRect(); open || openMapsState(e) != "22* [23*]" {
				t.Fatal("closing a clean tab:", openMapsState(e))
			}

			// Leaving the mod asks about each changed map, showing it.
			other := writeNamedEditorMod(t, t.TempDir(), "other", "[[mission]]\nkey='solo'\nnumber=31\nmap='solo.alm'\nafter=20\nparty=[{character='npc22'}]\n")
			writeTestPlayInput(t, filepath.Join(other, "solo.alm"), bytes.Clone(e.Document().SourceBytes))
			if err := e.Key("package"); err != nil {
				t.Fatal(err)
			}
			if err := workspaceField(e, "package", 0, other); err != nil {
				t.Fatal(err)
			}
			if !asked("map3.alm") || openMapsState(e) != "22* [23*]" {
				t.Fatalf("leaving the mod, first question: %q %s", e.PromptNotes(), openMapsState(e))
			}
			openMapsAnswer(t, e, 1)
			if !asked("map2.alm") || openMapsState(e) != "[22*] 23*" {
				t.Fatalf("leaving the mod, second question: %q %s", e.PromptNotes(), openMapsState(e))
			}
			openMapsAnswer(t, e, 1)
			if _, open := e.PromptRect(); open || e.OpenMapCount() != 1 || filepath.Base(e.Document().Source) != "solo.alm" {
				t.Fatalf("the other mod did not open alone: %q %s %s", e.PromptNotes(), openMapsState(e), e.Document().Source)
			}
			if openMapsTabs(e) != 0 {
				t.Fatal("one open map shows a strip")
			}
			if got := tree(t, dir); !reflect.DeepEqual(got, after) {
				t.Fatal("answers without saving wrote into the mod")
			}
		})
	}
}

// At most eight maps are open at once: the ninth is refused in the editor's
// words, and opens once a tab is closed. Closing the window asks about each
// changed map.
func TestEditorOpenMapsStopAtTheLimit(t *testing.T) {
	_, e := workspaceFixture(t)
	e.SetWords(words.For(""))
	e.Layout(1280, 800)
	openMapsMod(t, e, 9)
	for n := 2; n <= 8; n++ {
		if err := workspaceField(e, "package", 9, fmt.Sprint(n)); err != nil {
			t.Fatal(err)
		}
	}
	if e.OpenMapCount() != 8 {
		t.Fatal("eight maps did not open:", openMapsState(e))
	}
	if err := workspaceField(e, "package", 9, "9"); err == nil || !strings.Contains(err.Error(), "At most 8 maps") || e.OpenMapCount() != 8 || e.ShownMap() != 7 {
		t.Fatal("the ninth map:", err, openMapsState(e))
	}
	_ = e.Key("escape")
	e.CloseOpenMap()
	if err := workspaceField(e, "package", 9, "9"); err != nil || openMapsState(e) != "21 22 23 24 25 26 27 [29]" {
		t.Fatal("the ninth map after a close:", err, openMapsState(e))
	}
	if err := workspaceField(e, "metadata", 0, "Last"); err != nil {
		t.Fatal(err)
	}
	openMapsTab(t, e, 0)
	if err := workspaceField(e, "metadata", 0, "First"); err != nil {
		t.Fatal(err)
	}
	openMapsTab(t, e, 3)
	if err := e.Key("close"); err != nil || e.ExitAsked() || openMapsState(e) != "[21*] 22 23 24 25 26 27 29*" {
		t.Fatal("close with two changed maps:", err, openMapsState(e))
	}
	openMapsAnswer(t, e, 1)
	if e.ExitAsked() || openMapsState(e) != "21* 22 23 24 25 26 27 [29*]" {
		t.Fatal("the second question:", openMapsState(e))
	}
	openMapsAnswer(t, e, 1)
	if !e.ExitAsked() {
		t.Fatal("the window did not close after both answers")
	}
}

// An agent reads the open maps and shows one by its place; every other tool
// reads and edits the map shown.
func TestEditorMCPListsAndShowsOpenMaps(t *testing.T) {
	_, e := workspaceFixture(t)
	e.SetWords(words.For(""))
	e.Layout(1280, 800)
	openMapsMod(t, e, 3)
	if err := workspaceField(e, "package", 9, "2"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "metadata", 0, "Ford"); err != nil {
		t.Fatal(err)
	}
	tool, must := editorMCP(t, e)
	rows := func() []ui.OpenMapRow {
		t.Helper()
		var got []ui.OpenMapRow
		if err := json.Unmarshal([]byte(must("open_maps", map[string]any{}).Content[0].Text), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	got := rows()
	if len(got) != 2 || got[0].Mission != 21 || got[0].Dirty || got[0].Shown || got[1].Mission != 22 || !got[1].Dirty || !got[1].Shown || got[1].Caption != "22. Ford *" {
		t.Fatalf("open_maps: %+v", got)
	}
	var state map[string]any
	if err := json.Unmarshal([]byte(must("show_map", map[string]any{"map": 0, "verbose": true}).Content[0].Text), &state); err != nil {
		t.Fatal(err)
	}
	if got = rows(); !got[0].Shown || got[1].Shown || !got[1].Dirty || state["path"] != got[0].Path || state["dirty"] != true {
		t.Fatalf("show_map: %+v path %v dirty %v", got, state["path"], state["dirty"])
	}
	if r := tool("show_map", map[string]any{"map": 2}); !r.IsError || !strings.Contains(r.Content[0].Text, "open_maps lists 2") {
		t.Fatalf("show_map outside the open maps: %+v", r)
	}
}
