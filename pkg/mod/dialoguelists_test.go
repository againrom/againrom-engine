package mod

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/scriptprofile"
)

// replyLists is a one-node dialogue whose reply tests conds condition pairs
// and runs acts actions, in the writer's form.
func replyLists(conds, acts int) string {
	when := make([]string, conds)
	for i := range when {
		when[i] = fmt.Sprintf("{ left = %d, right = %d, cmp = %d }", i+1, i+2, i%6)
	}
	do := make([]string, acts)
	for i := range do {
		do[i] = fmt.Sprint(100 + i)
	}
	return "[[node]]\nid = 1\nen = \"a\"\n\n[[reply]]\nen = \"b\"\nwhen = [" + strings.Join(when, ", ") + "]\ndo = [" + strings.Join(do, ", ") + "]\n"
}

// A reply holds as many conditions and actions as an extended trigger, in
// order, and writes back to the same bytes; one more of either is refused.
func TestDialogueReplyHoldsAnExtendedTriggersLists(t *testing.T) {
	most := scriptprofile.Extended
	src := replyLists(most.Pairs, most.Actions)
	d, err := ParseDialogue("long.toml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	r := d.Nodes[0].Replies[0]
	if len(r.When) != most.Pairs || len(r.Do) != most.Actions {
		t.Fatalf("the reply holds %d conditions and %d actions", len(r.When), len(r.Do))
	}
	for i, c := range r.When {
		if c != (DialogueCondition{Left: i + 1, Right: i + 2, Cmp: i % 6}) {
			t.Fatalf("condition %d is %+v", i, c)
		}
	}
	for i, id := range r.Do {
		if id != 100+i {
			t.Fatalf("action %d is %d", i, id)
		}
	}
	if b, err := WriteDialogue(d); err != nil || string(b) != src {
		t.Fatalf("the long reply wrote (%v):\n%s", err, b)
	}
	for _, c := range []struct {
		src, msg string
	}{
		{replyLists(most.Pairs+1, 1), fmt.Sprintf("when holds more than %d", most.Pairs)},
		{replyLists(1, most.Actions+1), fmt.Sprintf("do holds more than %d", most.Actions)},
	} {
		_, err := ParseDialogue("long.toml", []byte(c.src))
		var fe *ItemFileError
		if !errors.As(err, &fe) || fe.Line != 7 && fe.Line != 8 || !strings.Contains(fe.Msg, c.msg) {
			t.Errorf("a list past the limit: %v, want %q", err, c.msg)
		}
	}
}

// A list may run over lines inside its brackets, with comments and a
// trailing comma; it reads as the one-line form and keeps its first line.
// An array the file never closes is refused at the line it opens on.
func TestDialogueListsRunOverLines(t *testing.T) {
	head := "[[node]]\nid = 1\nen = \"a\"\n\n[[reply]]\nen = \"b # [not a comment]\"\n"
	one, err := ParseDialogue("x.toml", []byte(head+"when = [{ left = 1, right = 2, cmp = 0 }, { left = 3, right = 4, cmp = 1 }]\ndo = [5, 6, 7]\n"))
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := ParseDialogue("x.toml", []byte(head+"when = [ # all must hold\n  { left = 1, right = 2, cmp = 0 },\n  { left = 3, right = 4, cmp = 1 }, # the second\n]\ndo = [\n  5,\n  6, 7,\n]\nnext = 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !SameDialogue(one, wrapped) {
		t.Fatalf("the wrapped lists read %+v, want %+v", wrapped.Nodes[0].Replies[0], one.Nodes[0].Replies[0])
	}
	if got := wrapped.Nodes[0].Replies[0].Text.Words("en"); got != "b # [not a comment]" {
		t.Fatalf("the reply text read %q", got)
	}
	_, err = ParseDialogue("x.toml", []byte(head+"do = [5,\n6\n"))
	var fe *ItemFileError
	if !errors.As(err, &fe) || fe.Line != 7 || !strings.Contains(fe.Msg, "not closed") {
		t.Fatalf("an unclosed list: %v", err)
	}
}

// A dialogue whose lists run over lines is saved as the writer's one-line
// form for each list, an edited list included, and rereads as the edit.
func TestDialogueWrappedListsSaveAsEdited(t *testing.T) {
	src := "[[node]]\nid = 1\nen = \"a\"\n\n[[reply]]\nen = \"b\"\nwhen = [\n  { left = 1, right = 2, cmp = 0 },\n]\ndo = [\n  5, # first\n  6,\n]\n"
	d, err := ParseDialogue("x.toml", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	d.Nodes[0].Replies[0].Do = []int{5, 9, 7}
	out, err := WriteDialogue(d)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseDialogue("x.toml", out)
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if r := back.Nodes[0].Replies[0]; len(r.Do) != 3 || r.Do[1] != 9 || len(r.When) != 1 {
		t.Fatalf("the saved reply is %+v\n%s", r, out)
	}
}
