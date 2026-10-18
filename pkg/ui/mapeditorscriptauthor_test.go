package ui

import "testing"

// Up and Down cycle the Add kind without writing; Esc leaves.
func TestEditorFieldChoicesCycleWithoutCommitting(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("scripts")
	_ = e.Key("player")
	r, ok := e.addRunning()
	if !ok || r.kind != "Script" || e.prompt.path != "0" {
		t.Fatalf("A in the script list opened %v", e.prompt.kind)
	}
	_ = e.Key("down")
	if e.prompt.path != "1" {
		t.Fatalf("choice %q", e.prompt.path)
	}
	_ = e.Key("up")
	_ = e.Key("up")
	if e.prompt.path != "3" {
		t.Fatalf("wrapped choice %q", e.prompt.path)
	}
	if e.Dirty() || len(calls.lines) != 0 {
		t.Fatal("choosing wrote to model")
	}
	_ = e.Key("escape")
	if e.prompt.open() {
		t.Fatal("choice window kept mode")
	}
}
