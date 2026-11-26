package game

import (
	"fmt"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// dialoguesHold is whether the mod window's Dialogues page shows: a map is
// open in a mod's project.
func (s *mapInspectionEdits) dialoguesHold() bool {
	return s.model != nil && s.modPageHolds()
}

// dialoguesPage is the Dialogues page row, shown only while a map is open.
func dialoguesPage(r workspaceScope) workspaceScope {
	r.holds = (*mapInspectionEdits).dialoguesHold
	return r
}

// dialoguesFields is the Dialogues page: one row for each dialogue of the
// open map with its nodes and the triggers that start it and an Open button,
// then the New dialogue button.
func (s *mapInspectionEdits) dialoguesFields() []ui.InspectionField {
	if !s.dialoguesHold() {
		return nil
	}
	x := s.inspector
	var out []ui.InspectionField
	held := s.dialogueHeld()
	for _, name := range s.DialogueNames() {
		nodes := 0
		if f, err := mod.ParseDialogue(name+".toml", held[name]); err == nil {
			nodes = len(f.Nodes)
		}
		users := s.DialogueUsers(name)
		used := x.word("editor.dialogues.unused")
		if len(users) > 0 {
			used = fmt.Sprintf(x.word("editor.dialogues.users"), strings.Join(users, ", "))
		}
		out = append(out,
			ui.InspectionField{Key: ui.DialogueInfoKey + name, Label: name, Text: true, ReadOnly: true,
				TextValue: fmt.Sprintf(x.word("editor.dialogues.summary"), nodes, used)},
			ui.InspectionField{Key: ui.DialogueOpenKey + name, Label: x.word("editor.dialogues.open"), Action: true, Column: 1})
	}
	return append(out, ui.InspectionField{Key: ui.DialogueNewKey, Label: x.word("editor.dialogues.new"), Action: true})
}

// dialoguesGate refuses a save while a dialogue file breaks a check or an
// interactive action starts a dialogue that has no file.
func (s *mapInspectionEdits) dialoguesGate() error {
	x := s.inspector
	held := s.dialogueHeld()
	for _, name := range s.DialogueNames() {
		d, err := s.ReadDialogue(name)
		if err != nil {
			return wordErr("editor.dialogue.err.save", name, err.Error())
		}
		for _, i := range s.DialogueIssues(d) {
			return wordErr("editor.dialogue.err.save", name, ui.IssueText(x.word, i))
		}
	}
	if s.model == nil {
		return nil
	}
	if _, sc, err := wizardScript(s.model); err == nil {
		untyped := s.model.Profile().UntypedAsInt
		for _, a := range sc.Actions {
			if a.InteractiveMessage(untyped) && held[a.Label] == nil {
				return wordErr("editor.dialogue.err.nofile", a.Label)
			}
		}
	}
	return nil
}

// setDialoguesField takes no value: the page's buttons open the dialogue
// window, which the editor itself shows.
func (s *mapInspectionEdits) setDialoguesField(string, string) (*ui.InspectionDocument, error) {
	return nil, nil
}
