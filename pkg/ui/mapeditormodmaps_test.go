package ui

import (
	"strings"
	"testing"

	"againrom/pkg/words"
	"github.com/hajimehoshi/ebiten/v2"
)

// The mod window has no page of the mod's maps: its tabs are the fields, the
// overview and the graph, while the Open window still lists every map.
func TestModWindowHasNoMapsTab(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := newModesEditor(t, "workspace")
		e.SetWords(words.For(lang))
		host := e.doc.Edits.(*modesWorkspace)
		host.overview, host.graph = modesOverview, modesGraph()
		_ = e.Key("package")
		s, _, _ := e.modSpec()
		if len(s.Tabs) != 3 {
			t.Fatalf("%s tabs %q", lang, s.Tabs)
		}
		for _, tab := range s.Tabs {
			if tab == "Maps" || tab == "Карты" {
				t.Fatalf("%s tabs %q hold the maps tab", lang, s.Tabs)
			}
		}
		if rows, maps := e.catalogRows(); len(rows) < 2 || maps[1] >= 0 {
			t.Fatalf("%s: the Open window lost the mod's maps: %v", lang, rows)
		}
	}
}

// The Open window leads with the open mod's maps: a heading, a row per map
// that can be chosen, and each further role of a map on a line of its own
// that cannot. Choosing a map goes the route W's list goes.
func TestOpenWindowLeadsWithTheMapsOfTheMod(t *testing.T) {
	e, calls := newModesEditor(t, "workspace")
	rows, maps := e.catalogRows()
	want := []string{"Maps of the mod", "crossing.alm: guest 25 after 20", "watch.alm: guest 26 after 25,", "replaces 3", "7.alm: replaces 7 by name", "spare.alm: unused", "gone.alm: missing"}
	choose := []bool{false, true, true, false, true, true, true}
	if len(rows) < len(want) {
		t.Fatalf("rows %v", rows)
	}
	for i := range want {
		if strings.TrimSpace(rows[i].Text) != want[i] || rows[i].Choosable != choose[i] {
			t.Fatalf("row %d is %+v, want %q choosable %v", i, rows[i], want[i], choose[i])
		}
	}
	if maps[0] != -1 || maps[2] != -3 || maps[3] != -1 || maps[4] != -4 {
		t.Fatalf("maps %v", maps[:len(want)])
	}
	if err := e.OpenModMap(-maps[4] - 1); err != nil {
		t.Fatal(err)
	}
	if got := calls.lines[len(calls.lines)-1]; got != `Workspace package map "3"` {
		t.Fatalf("the Open window's choice went %q", got)
	}
	plain, _ := newModesEditor(t, "plain")
	if rows, _ := plain.catalogRows(); len(rows) > 0 && rows[0].Text == "Maps of the mod" {
		t.Fatal("with no mod open the Open window has a mod group")
	}
}

// A map's roles are lines as the host lists them, never a split label.
func TestModMapRolesAreAListNotASplitLabel(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	p := e.doc.Edits.(*modesWorkspace)
	p.maps.Choices = []InspectionChoice{{Value: 1, Label: "a.alm: guest 25, the ambush, after 20, replaces 3"}, {Value: 2, Label: "b.alm: guest 26, the watch"}}
	p.maps.ChoiceLines = [][]string{{"a.alm: guest 25, the ambush, after 20,", "replaces 3"}}
	rows, _ := e.catalogRows()
	want := []string{"Maps of the mod", "a.alm: guest 25, the ambush, after 20,", "replaces 3", "b.alm: guest 26, the watch"}
	if len(rows) < len(want) {
		t.Fatalf("rows %v", rows)
	}
	for i := range want {
		if strings.TrimSpace(rows[i].Text) != want[i] {
			t.Fatalf("row %d is %q, want %q", i, rows[i].Text, want[i])
		}
	}
}

// Ctrl+Tab is the key of the open maps, not W's Tab: with one map open it
// leaves the keyboard's place in W where it stands.
func TestModWindowCtrlTabIsNotTheFieldTab(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	_ = e.Key("package")
	e.FocusModField("title")
	at := e.obj.focus
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyTab)
	updateEditorKeys(t, e, ebiten.KeyControl, ebiten.KeyShift, ebiten.KeyTab)
	if e.obj.focus != at || !e.modWindow {
		t.Fatalf("Ctrl+Tab moved W's place from %d to %d", at, e.obj.focus)
	}
	for _, key := range []string{"nextmap", "prevmap"} {
		if row := editorKeyByName[key]; row == nil || !strings.HasSuffix(row.label, "Tab") {
			t.Fatalf("the key table has no %s", key)
		}
	}
}

// The two keys of the open maps have a help line each, shown only while two or
// more maps are open: none with no mod or one map, one line each with two. A
// line naming both keys is wider than the help rail.
func TestOpenMapKeysHaveAHelpLineEach(t *testing.T) {
	count := func(e *MapEditor) (next, back int) {
		for _, l := range e.allHelpLines() {
			switch {
			case strings.HasPrefix(l, "Ctrl+Tab:") && !strings.Contains(l, "Ctrl+Shift+Tab"):
				next++
			case strings.HasPrefix(l, "Ctrl+Shift+Tab:") && !strings.Contains(l, "Ctrl+Tab"):
				back++
			case strings.Contains(l, "Ctrl+Tab") || strings.Contains(l, "Ctrl+Shift+Tab"):
				t.Errorf("the help line %q names both keys", l)
			}
		}
		return next, back
	}
	for _, lang := range []string{"en", "ru"} {
		open := func(host string, acts ...string) *MapEditor {
			e, _ := newModesEditor(t, host)
			e.SetWords(words.For(lang))
			for _, a := range acts {
				modesAct(e, a)
			}
			return e
		}
		noMod := open("plain")
		if n := len(noMod.OpenMapRows()); n != 1 {
			t.Fatalf("%s setup: %d maps open with no mod, want 1", lang, n)
		}
		if next, back := count(noMod); next != 0 || back != 0 {
			t.Errorf("%s no mod: %d Ctrl+Tab and %d Ctrl+Shift+Tab lines, want none", lang, next, back)
		}
		one := open("workspace")
		if n := len(one.OpenMapRows()); n != 1 {
			t.Fatalf("%s setup: %d maps open, want 1", lang, n)
		}
		if next, back := count(one); next != 0 || back != 0 {
			t.Errorf("%s one map: %d Ctrl+Tab and %d Ctrl+Shift+Tab lines, want none", lang, next, back)
		}
		two := open("workspace", "ui:mod-second-map")
		if n := len(two.OpenMapRows()); n != 2 {
			t.Fatalf("%s setup: %d maps open, want 2", lang, n)
		}
		if next, back := count(two); next != 1 || back != 1 {
			t.Errorf("%s two maps: %d Ctrl+Tab and %d Ctrl+Shift+Tab lines, want one each", lang, next, back)
		}
	}
}

// The overview's Files list is folded when the page opens, its heading says
// how many files there are, and a click or Enter on the heading unfolds it
// and folds it again.
func TestModOverviewFilesListStartsFoldedAndAnswersClickAndEnter(t *testing.T) {
	lines := func(e *MapEditor) string { return strings.Join(e.PopupLines(), "\n") }
	e, _ := newModesEditor(t, "workspace")
	e.doc.Edits.(*modesWorkspace).overview = modesOverview
	_ = e.Key("package")
	if got := lines(e); !strings.Contains(got, "Files (2)") || strings.Contains(got, "mod.toml") || !strings.Contains(got, "terrain: data/terrain.toml") {
		t.Fatalf("the page opens with the files list not folded:\n%s", got)
	}
	click := func() {
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		for _, r := range l.Rows {
			if r.Index == e.modFoldRow[0] {
				modesClick(e, center(r.Label))
				return
			}
		}
		t.Fatal("the Files heading is not laid out")
	}
	click()
	if got := lines(e); !strings.Contains(got, "mod.toml") {
		t.Fatalf("a click did not unfold the files:\n%s", got)
	}
	click()
	if got := lines(e); strings.Contains(got, "mod.toml") {
		t.Fatalf("a second click did not fold the files:\n%s", got)
	}
	for i := 0; i < 20 && e.obj.focus != len(e.obj.fields)+e.modButtonCount(); i++ {
		_ = e.Key("tab")
	}
	_ = e.Key("enter")
	if got := lines(e); !strings.Contains(got, "mod.toml") {
		t.Fatalf("Enter on the heading did not unfold the files:\n%s", got)
	}
	_ = e.Key("enter")
	if got := lines(e); strings.Contains(got, "mod.toml") {
		t.Fatalf("a second Enter did not fold the files:\n%s", got)
	}
}
