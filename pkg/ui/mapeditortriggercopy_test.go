package ui

import (
	"bytes"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// Copy trigger is one command row: one unlisted key table row, one toolbar
// table row standing in the Tools menu alone, one command table row taking
// the trigger's index, and words for the menu row, the trigger window's
// button, the copy mark and the refusal. With no trigger selected the key
// says to select one, changing nothing; with no map the row is off. EN and
// RU.
func TestMapEditorCopyTriggerIsOneCommandRow(t *testing.T) {
	rows := 0
	for _, r := range editorKeyRows {
		if r.key == "copytrigger" {
			rows++
		}
	}
	if rows != 1 || editorKeyByName["copytrigger"] == nil || editorKeyByName["copytrigger"].label != "" {
		t.Fatalf("key table rows %d", rows)
	}
	buttons := 0
	for _, b := range toolButtons {
		if b.key == "copytrigger" {
			buttons++
			if b.keyRow != "" || b.menu != "tools" || b.icon != nil || b.onToolbar() {
				t.Fatalf("the toolbar table row %+v", b)
			}
		}
	}
	commands := 0
	for _, c := range editorCommandRows() {
		if c.name == "copy_trigger" {
			commands++
			if len(c.args) != 1 || c.args[0].name != "index" || !c.args[0].required {
				t.Fatalf("the command row takes %+v", c.args)
			}
		}
	}
	if buttons != 1 || commands != 1 {
		t.Fatalf("toolbar table rows %d, command table rows %d", buttons, commands)
	}
	for _, lang := range []string{"", "russian"} {
		book := words.For(lang)
		for _, key := range []string{"editor.tool.copytrigger", "editor.trigger.copy", "editor.trigger.copyname", "editor.trigger.copyfirst"} {
			if book.Text(key) == key {
				t.Fatalf("%q: no word %s", lang, key)
			}
		}
		if strings.Count(book.Text("editor.trigger.copyname"), "%s") != 1 {
			t.Fatalf("%q: the copy mark %q", lang, book.Text("editor.trigger.copyname"))
		}
		e, _, _ := menuEditor(t, lang)
		for _, b := range e.barEntries() {
			if !b.menu {
				continue
			}
			n := 0
			for _, it := range e.menuItems(b.key) {
				if it.key != "copytrigger" {
					continue
				}
				n++
				if word := e.word("editor.tool.copytrigger"); !strings.Contains(it.label, word) || it.off {
					t.Fatalf("%q: the %s menu row reads %q off %v, want %q on", lang, b.key, it.label, it.off, word)
				}
			}
			if want := map[bool]int{true: 1, false: 0}[b.key == "tools"]; n != want {
				t.Fatalf("%q: the %s menu holds %d Copy trigger rows, want %d", lang, b.key, n, want)
			}
		}
		start := bytes.Clone(e.doc.SourceBytes)
		if err := e.Key("copytrigger"); err != nil {
			t.Fatal(err)
		}
		if e.message != e.word("editor.trigger.copyfirst") || !bytes.Equal(e.doc.SourceBytes, start) {
			t.Fatalf("%q: the key with no trigger says %q", lang, e.message)
		}
	}
	none := NewMapEditor(panelFont(), nil, nil)
	none.Layout(1920, 1080)
	for _, it := range none.menuItems("tools") {
		if it.key == "copytrigger" && !it.off {
			t.Fatal("the Tools menu offers Copy trigger with no map")
		}
	}
}
