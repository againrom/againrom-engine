package game

import (
	"bytes"
	"testing"

	"againrom/pkg/ui"
)

// dialogueHost opens an empty 40 by 40 map in the editor and returns the host
// its dialogue methods are on.
func dialogueHost(t *testing.T) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	e, _ := openGoalMap(t, emptyScriptMap(t, 40))
	s, ok := e.Document().Edits.(*mapInspectionEdits)
	if !ok {
		t.Fatal("the host is not the map inspector's")
	}
	return e, s
}

func setVariableNode(variable, value uint32) ui.WizardNode {
	return ui.WizardNode{Opcode: 3, Types: [10]uint32{1, 1}, Values: [10]uint32{variable, value}}
}

func twoNodeDialogue() ui.DialogueData {
	return ui.DialogueData{Nodes: []ui.DialogueNodeData{
		{ID: 1, Name: "start", Text: map[string]string{"en": "Hello", "ru": "Привет"}, Replies: []ui.DialogueReplyData{
			{Text: map[string]string{"en": "Yes"}, Next: 2, Actions: []ui.WizardNode{setVariableNode(10, 1)}},
			{Text: map[string]string{"en": "Once", "ru": "Раз"}, Once: true, Mark: 99},
		}},
		{ID: 2, Name: "end", Text: map[string]string{"en": "Bye"}},
	}}
}

// A once mark past the map's script variables is refused in words.
func TestDialogueRefusesAOnceMarkPastTheScriptVariables(t *testing.T) {
	_, s := dialogueHost(t)
	bad := twoNodeDialogue()
	bad.Nodes[0].Replies[1].Mark = 5000
	issues := s.DialogueIssues(bad)
	if len(issues) != 1 || issues[0].Rule != "editor.dialogue.check.mark" || issues[0].Other != 5000 {
		t.Fatalf("issues %+v", issues)
	}
	if _, err := s.WriteDialogue("mill", bad); err == nil {
		t.Fatal("a mark past the variables was written")
	}
}

func TestDialogueWriteReadsBackAndUndoRestoresTheMap(t *testing.T) {
	_, s := dialogueHost(t)
	before := s.model.Bytes()
	if _, err := s.WriteDialogue("mill", twoNodeDialogue()); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadDialogue("mill")
	if err != nil || len(got.Nodes) != 2 || got.Nodes[0].Name != "start" || got.Nodes[0].Text["ru"] != "Привет" {
		t.Fatalf("read back %+v %v", got, err)
	}
	r := got.Nodes[0].Replies[0]
	if len(r.Actions) != 1 || !r.Actions[0].Keep || r.Next != 2 || !got.Nodes[0].Replies[1].Once || got.Nodes[0].Replies[1].Mark != 99 {
		t.Fatalf("reply %+v", got.Nodes[0].Replies)
	}
	sc := wizScript(t, s.model.Bytes())
	if n := wizNode(t, sc.Actions, r.Actions[0].ID); n.Opcode != 3 || n.Value[0] != 10 {
		t.Fatalf("the outcome node %+v", n)
	}
	if names := s.DialogueNames(); len(names) != 1 || names[0] != "mill" {
		t.Fatalf("names %v", names)
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, s.model.Bytes()) || len(s.DialogueNames()) != 0 {
		t.Fatalf("undo left the map or the dialogue: %v", s.DialogueNames())
	}
}

func TestDialogueRewriteDropsTheNodesItLeavesUnnamed(t *testing.T) {
	_, s := dialogueHost(t)
	if _, err := s.WriteDialogue("mill", twoNodeDialogue()); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ReadDialogue("mill")
	got.Nodes[0].Replies[0].Actions = nil
	if _, err := s.WriteDialogue("mill", got); err != nil {
		t.Fatal(err)
	}
	if sc := wizScript(t, s.model.Bytes()); len(sc.Actions) != 0 {
		t.Fatalf("the outcome node stayed: %+v", sc.Actions)
	}
}

func TestDialogueRefusesWhatTheChecksRefuseAndAUsedDialogueStays(t *testing.T) {
	_, s := dialogueHost(t)
	bad := twoNodeDialogue()
	bad.Nodes[0].Replies[0].Next = 7
	if _, err := s.WriteDialogue("mill", bad); err == nil {
		t.Fatal("a next node that does not exist was written")
	}
	issues := s.DialogueIssues(bad)
	if len(issues) != 1 || issues[0].Rule != "editor.dialogue.check.next" || issues[0].Other != 7 {
		t.Fatalf("issues %+v", issues)
	}
	if _, err := s.WriteDialogue("mill", twoNodeDialogue()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RemoveDialogue("mill"); err != nil || len(s.DialogueNames()) != 0 {
		t.Fatalf("an unused dialogue stays: %v %v", err, s.DialogueNames())
	}
	if _, err := s.RemoveDialogue("mill"); err != nil {
		t.Fatal(err)
	}
	if sc := wizScript(t, s.model.Bytes()); len(sc.Actions) != 0 {
		t.Fatalf("the removed dialogue left nodes: %+v", sc.Actions)
	}
	if mark := s.FreeMark(ui.DialogueData{}); mark != 99 {
		t.Fatalf("free mark %d", mark)
	}
}
