package ui

import (
	"slices"
	"testing"
)

// mapRowKeys are the Map menu's rows in order: "" is the Generate heading
// and then the rule over Exit.
var mapRowKeys = []string{"new", "open", "saveas", "saveto", "catalog", "testplay", "panel", "records", "", "randomterrain", "randomheights", "", "close"}

// The Map menu's rows are rows of the toolbar's one table, with no toolbar
// button, and read as they always did: Create new map, Open, Save, Save as,
// Built-in maps, Test play, Map panel, All records, the Generate heading over
// the two random commands, a rule, Exit; in EN and RU.
func TestMapEditorMapMenuRowsAreRowsOfTheTable(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _, _ := menuEditor(t, lang)
		items := e.menuItems("map")
		if got := menuKeys(e, "map"); !slices.Equal(got, mapRowKeys) {
			t.Fatalf("%q: the Map menu holds %v, want %v", lang, got, mapRowKeys)
		}
		words := []string{"editor.menu.new", "editor.menu.open", "editor.menu.save", "editor.menu.saveas", "editor.menu.builtin", "editor.menu.testplay", "editor.menu.panel", "editor.view.records", "editor.menu.generate", "editor.tools.randomterrain", "editor.tools.randomheights", "", "editor.menu.exit"}
		for i, w := range words {
			want := ""
			if w != "" {
				want = e.word(w)
			}
			if items[i].label != want {
				t.Errorf("%q: Map row %d reads %q, want %q", lang, i, items[i].label, want)
			}
		}
		table := map[string]toolButton{}
		for _, b := range toolButtons {
			table[b.key] = b
		}
		for _, k := range mapRowKeys {
			if k == "" {
				continue
			}
			if b, ok := table[k]; !ok || b.menu != "map" || b.onToolbar() {
				t.Errorf("%q: Map row %s is not a menu-only row of the table", lang, k)
			}
		}
	}
}

// A Map row is drawn disabled exactly when its action would do nothing: Save,
// Save as and Test play on a map that cannot be edited, the Map panel on a map
// with no panel, All records with no map, the random commands with no map or
// in the built-in list; New, Open, Built-in maps and Exit always act.
func TestMapEditorMapRowIsDisabledWhenItsActionDoesNothing(t *testing.T) {
	off := func(e *MapEditor) map[string]bool {
		out := map[string]bool{}
		for _, it := range e.menuItems("map") {
			if it.key != "" {
				out[it.key] = it.off
			}
		}
		return out
	}
	always := []string{"new", "open", "catalog", "close"}
	e, _, _ := menuEditor(t, "")
	for k, o := range off(e) {
		if o && k != "panel" {
			t.Errorf("an editable map shows %s disabled", k)
		}
	}
	e.doc.Edits = nil
	got := off(e)
	for _, k := range []string{"saveas", "saveto", "testplay", "panel"} {
		if !got[k] {
			t.Errorf("a map that cannot be edited shows %s choosable", k)
		}
	}
	for _, k := range always {
		if got[k] {
			t.Errorf("a map that cannot be edited shows %s disabled", k)
		}
	}
	if got["records"] || got["randomterrain"] {
		t.Errorf("a map that is open shows records or the random commands disabled")
	}
	e.doc = nil
	got = off(e)
	for _, k := range []string{"saveas", "saveto", "testplay", "panel", "records", "randomterrain", "randomheights"} {
		if !got[k] {
			t.Errorf("no map shows %s choosable", k)
		}
	}
	for _, k := range always {
		if got[k] {
			t.Errorf("no map shows %s disabled", k)
		}
	}
	c, _, _ := menuEditor(t, "")
	c.catalog = true
	if got := off(c); !got["randomterrain"] || !got["randomheights"] || got["saveas"] {
		t.Errorf("the built-in list shows random %v/%v, save %v", got["randomterrain"], got["randomheights"], got["saveas"])
	}
}
