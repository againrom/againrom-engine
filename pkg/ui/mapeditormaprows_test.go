package ui

import (
	"slices"
	"testing"
)

// Owner items 1, 3, 4, 6 move the panel and generation out, history and search in.
var mapRowKeys = []string{"new", "open", "saveas", "saveto", "catalog", "testplay", "undo", "redo", "search", "records", "", "close"}

// Owner items 1, 3, 4, 6: File reads the command table, including history
// with its keys and Search, without the Map panel or Generate group.
func TestMapEditorMapMenuRowsAreRowsOfTheTable(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _, _ := menuEditor(t, lang)
		items := e.menuItems("file")
		if got := menuKeys(e, "file"); !slices.Equal(got, mapRowKeys) {
			t.Fatalf("%q: the Map menu holds %v, want %v", lang, got, mapRowKeys)
		}
		words := []string{"editor.menu.new", "editor.menu.open", "editor.menu.save", "editor.menu.saveas", "editor.menu.builtin", "editor.menu.testplay", "editor.bar.undo", "editor.bar.redo", "editor.bar.search", "editor.view.records", "", "editor.menu.exit"}
		for i, w := range words {
			want := ""
			if w != "" {
				want = e.word(w)
			}
			if items[i].key == "undo" {
				want += " (Ctrl+Z)"
			}
			if items[i].key == "redo" {
				want += " (Ctrl+Y / Ctrl+Shift+Z)"
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
			if b, ok := table[k]; !ok || b.menu != "file" || b.onToolbar() {
				t.Errorf("%q: Map row %s is not a menu-only row of the table", lang, k)
			}
		}
	}
}

// Owner item 4 keeps history's disabled states in File. Save, Save as and
// Test play require edits; Search and All records require a map.
func TestMapEditorMapRowIsDisabledWhenItsActionDoesNothing(t *testing.T) {
	off := func(e *MapEditor) map[string]bool {
		out := map[string]bool{}
		for _, it := range e.menuItems("file") {
			if it.key != "" {
				out[it.key] = it.off
			}
		}
		return out
	}
	always := []string{"new", "open", "catalog", "close"}
	e, _, _ := menuEditor(t, "")
	for k, o := range off(e) {
		if o && k != "undo" && k != "redo" {
			t.Errorf("an editable map shows %s disabled", k)
		}
	}
	e.doc.Edits = nil
	got := off(e)
	for _, k := range []string{"saveas", "saveto", "testplay", "undo", "redo"} {
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
	for _, k := range []string{"saveas", "saveto", "testplay", "undo", "redo", "records", "search"} {
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
	if got := off(c); got["saveas"] {
		t.Errorf("the built-in list shows random %v/%v, save %v", got["randomterrain"], got["randomheights"], got["saveas"])
	}
}
