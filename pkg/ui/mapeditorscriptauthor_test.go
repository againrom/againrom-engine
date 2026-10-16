package ui

import "testing"

func TestEditorFieldChoicesCycleWithoutCommitting(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.openPrompt(editorPrompt{kind: promptNewScript, path: "0"})
	if e.fieldTitle() != "Action / Up, Down / Enter: add" {
		t.Fatal(e.fieldTitle())
	}
	_ = e.Key("down")
	if e.prompt.path != "1" || e.fieldTitle() != "Condition / Up, Down / Enter: add" {
		t.Fatalf("choice %q %s", e.prompt.path, e.fieldTitle())
	}
	_ = e.Key("up")
	_ = e.Key("up")
	if e.prompt.path != "2" {
		t.Fatalf("wrapped choice %q", e.prompt.path)
	}
	if e.Dirty() {
		t.Fatal("choosing wrote to model")
	}
	_ = e.Key("escape")
	if e.prompt.open() {
		t.Fatal("choice prompt kept mode")
	}
}
