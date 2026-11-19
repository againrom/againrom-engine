package ui

import (
	"bytes"
	"encoding/json"
	"testing"

	"againrom/pkg/words"
)

func TestViewOnlyMapRefusesEveryEditCommand(t *testing.T) {
	e, host := popupEditor(t)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1920, 1080)
	e.doc.Edits = nil
	before := bytes.Clone(e.doc.SourceBytes)
	for _, lang := range []string{"en", "ru"} {
		e.SetWords(words.For(lang))
		e.doc.ReadOnly = words.For(lang).Text("editor.map.secondtool")
		for _, cmd := range editorCommandRows() {
			if cmd.opens {
				continue
			}
			if _, err := cmd.mcpRun(e, json.RawMessage(`{}`)); err == nil || err.Error() != e.doc.ReadOnly {
				t.Fatalf("%s %s: %v", lang, cmd.name, err)
			}
		}
		if err := e.Key("saveas"); err == nil || err.Error() != e.doc.ReadOnly {
			t.Fatalf("%s save: %v", lang, err)
		}
		if !bytes.Equal(before, e.doc.SourceBytes) || len(host.calls) != 0 || e.Dirty() {
			t.Fatal("view-only command reached a writer or changed the document")
		}
	}
}
