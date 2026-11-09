package ui

import "testing"

func TestFindControlByRoleAndCaption(t *testing.T) {
	e := groupOrderEditor(t)
	e.openPrompt(editorPrompt{kind: promptOpen, path: "abcd"})
	s, ok := e.promptSpec()
	if !ok || len(s.Buttons) == 0 {
		t.Fatal("setup: no prompt with buttons")
	}
	byRole, ok := e.FindControl(ControlQuery{Role: ControlPromptButton, N: 0})
	if !ok || byRole.Empty() {
		t.Fatalf("role lookup = %v, %v", byRole, ok)
	}
	byCaption, ok := e.FindControl(ControlQuery{Role: ControlPromptButton, Caption: s.Buttons[0]})
	if !ok || byCaption != byRole {
		t.Fatalf("caption lookup = %v, %v; role %v", byCaption, ok, byRole)
	}
	if r, ok := e.FindControl(ControlQuery{Role: ControlPromptButton, Caption: "no such button"}); ok {
		t.Errorf("unknown caption found %v", r)
	}
	if r, ok := e.FindControl(ControlQuery{Role: ControlRowButton, Caption: "no such button"}); ok {
		t.Errorf("a role with no caption row found %v", r)
	}
	if r, ok := e.FindControl(ControlQuery{Role: controlRoleCount}); ok {
		t.Errorf("unknown role found %v", r)
	}
}
