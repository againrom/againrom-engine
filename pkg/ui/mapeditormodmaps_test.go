package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/words"
	"github.com/hajimehoshi/ebiten/v2"
)

// W's last page lists every map of the mod, one row per map with its role
// and an Open button, at the small window and the large and in each
// language: a role too long for the row goes on under it whole, the rows
// scroll to the map the keyboard stands on, and Enter there opens that map.
func TestModWindowMapsPageListsEveryMapOfTheMod(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range []image.Point{{X: 640, Y: 480}, {X: 1280, Y: 800}} {
			e, calls := newModesEditor(t, "workspace")
			e.SetWords(words.For(lang))
			e.Layout(size.X, size.Y)
			_ = e.Key("package")
			s, _, _ := e.modSpec()
			if len(s.Tabs) != 2 || s.Tabs[1] != e.word("editor.mod.mapstab") {
				t.Fatalf("%s tabs %q", lang, s.Tabs)
			}
			_ = e.Key("package")
			if e.modPage != modPageMaps {
				t.Fatalf("the window's key shows page %d", e.modPage)
			}
			s, _, _ = e.modSpec()
			list := e.doc.Edits.(*modesWorkspace).ModMaps()
			buttons := 0
			var drawn []string
			for i, r := range s.Rows {
				drawn = append(drawn, strings.TrimSpace(r.Label))
				if len(r.Buttons) == 1 {
					buttons++
					if e.obj.rowActs[i] == nil {
						t.Fatalf("row %d has a button and no map", i)
					}
				}
			}
			all := strings.Join(drawn, " ")
			for _, c := range list.Choices {
				for _, w := range strings.Fields(c.Label) {
					if !strings.Contains(" "+all+" ", " "+w+" ") {
						t.Fatalf("%s %v: %q of %q is not whole in %q", lang, size, w, c.Label, drawn)
					}
				}
			}
			if buttons != len(list.Choices) || !strings.HasPrefix(s.Rows[0].Label, "> ") {
				t.Fatalf("%d Open buttons for %d maps; first row %q", buttons, len(list.Choices), s.Rows[0].Label)
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			if !l.Rect.In(image.Rectangle{Max: size}) || len(l.Rows) < 3 {
				t.Fatalf("%s %v: the page is %v with %d rows", lang, size, l.Rect, len(l.Rows))
			}
			// Up from the first button is the last map; its row scrolls in.
			_ = e.Key("up")
			s, _, _ = e.modSpec()
			l = layoutPopup(s, e.font, e.bodyFont())
			last, seen := e.modMapLine[len(e.modMapLine)-1], false
			for _, r := range l.Rows {
				seen = seen || r.Index == last
			}
			if !seen || !s.Rows[last].Armed[0] || s.ButtonFocus != nil {
				t.Fatalf("%s %v: the last map's row %d is not the keyboard's shown place", lang, size, last)
			}
			_ = e.Key("enter")
			want := `Workspace package map "5"`
			if got := calls.lines[len(calls.lines)-1]; got != want || !e.modWindow || e.modPage != modPageMaps {
				t.Fatalf("Enter on the last map: %q, window %v page %d", got, e.modWindow, e.modPage)
			}
			// A click on a row's Open button opens that row's map; the
			// window's own buttons answer on this page as on the others.
			_ = e.Key("home")
			e.objectTop = 0
			s, _, _ = e.modSpec()
			for _, r := range layoutPopup(s, e.font, e.bodyFont()).Rows {
				if r.Index == e.modMapLine[1] {
					modesClick(e, center(r.Buttons[0]))
				}
			}
			if got := calls.lines[len(calls.lines)-1]; got != `Workspace package map "2"` {
				t.Fatalf("%s %v: a click on the second map's Open went %q", lang, size, got)
			}
			s, _, _ = e.modSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[modButtonIndex("close")]))
			if e.modWindow {
				t.Fatalf("%s %v: Close left the window open", lang, size)
			}
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
