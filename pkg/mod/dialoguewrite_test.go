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

func TestDialogueKeepsASpeakerHeroVariantsAndAContinuation(t *testing.T) {
	d := Dialogue{File: "v.toml", Nodes: []DialogueNode{
		{ID: 1, Speaker: 21, Speaks: true, Next: 2, Text: DialogueText{"en": "all", "ru": "все"},
			Variants: map[string]DialogueText{"woman": {"en": "she", "ru": "она"}, "man": {"en": "he"}}},
		{ID: 2, Speaker: 0, Speaks: true, Text: DialogueText{"en": "end"},
			Variants: map[string]DialogueText{"man_mage": {"en": "a"}, "man_fighter": {"en": "b"}, "woman_mage": {"en": "c"}, "woman_fighter": {"en": "d"}}},
	}}
	out, err := WriteDialogue(d)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseDialogue("v.toml", out)
	if err != nil || !SameDialogue(d, back) {
		t.Fatalf("reads back %v, %+v\n%s", err, back, out)
	}
	n := back.Nodes[0]
	for _, c := range []struct {
		lang         string
		female, mage bool
		want         string
	}{{"en", true, false, "she"}, {"en", false, true, "he"}, {"ru", true, true, "она"}, {"ru", false, false, "все"}} {
		if got := n.TextFor(c.lang, c.female, c.mage); got != c.want {
			t.Errorf("%s female=%v mage=%v: %q, want %q", c.lang, c.female, c.mage, got, c.want)
		}
	}
	if got := back.Nodes[1].TextFor("en", true, true); got != "c" {
		t.Errorf("woman mage: %q", got)
	}
	plain, err := ParseDialogue("p.toml", []byte("[[node]]\nid = 1\nen = \"a\"\n"))
	if err != nil || plain.Nodes[0].Speaks || plain.Nodes[0].Variants != nil || plain.Nodes[0].Next != 0 {
		t.Errorf("a file without them reads as before: %v %+v", err, plain)
	}
	for name, src := range map[string]string{
		"mixed kinds":   "[[node]]\nid = 1\nen = \"a\"\nen_woman = \"b\"\nen_mage = \"c\"\n",
		"next nowhere":  "[[node]]\nid = 1\nen = \"a\"\nnext = 5\n",
		"empty variant": "[[node]]\nid = 1\nen = \"a\"\nen_woman = \"\"\n",
	} {
		if _, err := ParseDialogue("x.toml", []byte(src)); err == nil {
			t.Errorf("%s: read", name)
		}
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
