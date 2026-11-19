package mod

import (
	"os"
	"testing"
)

func TestWriteDialogueReadsBackEqualThroughTheLoader(t *testing.T) {
	data, err := os.ReadFile("testdata/dialogue/elder.toml")
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseDialogue("elder.toml", data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := WriteDialogue(d)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseDialogue("elder.toml", out)
	if err != nil || !SameDialogue(d, back) {
		t.Fatalf("the written file reads back %v, %+v", err, back)
	}
	named := Dialogue{File: "named.toml", Nodes: []DialogueNode{
		{ID: 4, Name: "start", Text: DialogueText{"en": "Say \"hi\"\n", "ru": "Привет"}, Replies: []DialogueReply{
			{Text: DialogueText{"ru": "Да"}, Next: 9, Once: true, Mark: 12, Disabled: true,
				When: []DialogueCondition{{Left: 1, Right: 2, Cmp: 3}}, Do: []int{5, 6}}}},
		{ID: 9, Name: "end", Text: DialogueText{"en": "Bye"}},
	}}
	out, err = WriteDialogue(named)
	if err != nil {
		t.Fatal(err)
	}
	if back, err = ParseDialogue("named.toml", out); err != nil || !SameDialogue(named, back) {
		t.Fatalf("the named file reads back %v, %+v\n%s", err, back, out)
	}
}

func TestCheckDialogueRefusesEachRuleWithItsIssue(t *testing.T) {
	text := DialogueText{"en": "x"}
	reply := DialogueReply{Text: text}
	five := []DialogueReply{reply, reply, reply, reply, reply}
	for name, c := range map[string]struct {
		d    Dialogue
		vars int
		rule DialogueRule
	}{
		"no node":       {Dialogue{}, 0, RuleNoNode},
		"five replies":  {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text, Replies: five}}}, 0, RuleReplies},
		"missing next":  {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text, Replies: []DialogueReply{{Text: text, Next: 7}}}}}, 0, RuleNextMissing},
		"empty name":    {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text}, {ID: 2, Text: text}}}, 0, RuleNameEmpty},
		"repeated name": {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text}, {ID: 2, Name: "a", Text: text}}}, 0, RuleNameTwice},
		"no text":       {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text, Replies: []DialogueReply{{}}}}}, 0, RuleNoText},
		"mark past":     {Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: text, Replies: []DialogueReply{{Text: text, Once: true, Mark: 120}}}}}, 100, RuleMarkPast},
	} {
		issues := CheckDialogue(c.d, c.vars, true)
		if len(issues) == 0 || issues[0].Rule != c.rule {
			t.Errorf("%s: issues %+v, want rule %d", name, issues, c.rule)
		}
		if _, err := WriteDialogue(c.d); err == nil && c.rule != RuleMarkPast && c.rule != RuleNameEmpty {
			t.Errorf("%s: the writer wrote it", name)
		}
	}
	ok := Dialogue{Nodes: []DialogueNode{{ID: 1, Name: "a", Text: DialogueText{"ru": "я"}, Replies: []DialogueReply{{Text: DialogueText{"en": "b"}, Once: true, Mark: 99}}}}}
	if issues := CheckDialogue(ok, 100, true); len(issues) != 0 {
		t.Errorf("a text in one language and a mark below the variables is refused: %+v", issues)
	}
}
