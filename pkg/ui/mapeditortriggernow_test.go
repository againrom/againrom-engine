package ui

import (
	"strings"
	"testing"

	"againrom/pkg/words"
)

// New trigger is one command row: the one key table row on F4, one help
// line, one toolbar table row naming that key row and one Tools menu row
// reading its word and key, in no other menu. EN and RU.
func TestMapEditorNewTriggerIsOneCommandRow(t *testing.T) {
	rows, f4 := 0, 0
	for _, r := range editorKeyRows {
		if r.key == "newtrigger" {
			rows++
		}
		if r.label == "F4" {
			f4++
		}
	}
	if rows != 1 || f4 != 1 || editorKeyByName["newtrigger"] == nil || editorKeyByName["newtrigger"].label != "F4" {
		t.Fatalf("key table rows %d, rows on F4 %d", rows, f4)
	}
	buttons := 0
	for _, b := range toolButtons {
		if b.key == "newtrigger" {
			buttons++
			if b.keyRow != "newtrigger" || b.menu != "tools" || b.icon != nil {
				t.Fatalf("the toolbar table row %+v", b)
			}
		}
	}
	if buttons != 1 {
		t.Fatalf("toolbar table rows %d", buttons)
	}
	for _, lang := range []string{"", "russian"} {
		line := words.For(lang).Text("editor.keys.newtrigger")
		help := 0
		for _, l := range EditorHelpLinesIn(lang) {
			if l == line {
				help++
			}
		}
		if help != 1 || line == "editor.keys.newtrigger" {
			t.Fatalf("%q: help lines %q: %d", lang, line, help)
		}
		e, _, _ := menuEditor(t, lang)
		for _, b := range e.barEntries() {
			if !b.menu {
				continue
			}
			n := 0
			for _, it := range e.menuItems(b.key) {
				if it.key != "newtrigger" {
					continue
				}
				n++
				if word := e.word("editor.tool.newtrigger"); !strings.Contains(it.label, word) || !strings.Contains(it.label, "F4") || it.off {
					t.Fatalf("%q: the %s menu row reads %q off %v, want %q with F4", lang, b.key, it.label, it.off, word)
				}
			}
			if want := map[bool]int{true: 1, false: 0}[b.key == "tools"]; n != want {
				t.Fatalf("%q: the %s menu holds %d New trigger rows, want %d", lang, b.key, n, want)
			}
		}
	}
}
