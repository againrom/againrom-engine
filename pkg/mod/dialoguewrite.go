package mod

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/locale"
)

// WriteDialogue is d as a dialogue file ParseDialogue reads back equal. A
// dialogue the file cannot hold is refused with the check's first issue.
func WriteDialogue(d Dialogue) ([]byte, error) {
	if issues := CheckDialogue(d, 0, d.NamedNodes()); len(issues) != 0 {
		return nil, fmt.Errorf("dialogue %s: %s", d.File, issues[0].Text())
	}
	var b strings.Builder
	for i, n := range d.Nodes {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("[[node]]\nid = " + strconv.Itoa(n.ID) + "\n")
		if n.Name != "" {
			b.WriteString("name = " + writeString(n.Name) + "\n")
		}
		if n.Speaks {
			b.WriteString("speaker = " + strconv.Itoa(n.Speaker) + "\n")
		}
		if n.Next != 0 {
			b.WriteString("next = " + strconv.Itoa(n.Next) + "\n")
		}
		writeDialogueText(&b, n.Text)
		for _, key := range DialogueVariantKeys() {
			for _, l := range locale.All() {
				if s, ok := n.Variants[key][l.Code]; ok {
					b.WriteString(l.Code + "_" + key + " = " + writeString(s) + "\n")
				}
			}
		}
		for _, r := range n.Replies {
			b.WriteString("\n[[reply]]\n")
			writeDialogueText(&b, r.Text)
			if r.Next != 0 {
				b.WriteString("next = " + strconv.Itoa(r.Next) + "\n")
			}
			if r.Once {
				b.WriteString("once = true\nmark = " + strconv.Itoa(r.Mark) + "\n")
			}
			if r.Disabled {
				b.WriteString("unavailable = \"disabled\"\n")
			}
			if len(r.When) != 0 {
				parts := make([]string, len(r.When))
				for k, c := range r.When {
					parts[k] = fmt.Sprintf("{ left = %d, right = %d, cmp = %d }", c.Left, c.Right, c.Cmp)
				}
				b.WriteString("when = [" + strings.Join(parts, ", ") + "]\n")
			}
			if len(r.Do) != 0 {
				parts := make([]string, len(r.Do))
				for k, id := range r.Do {
					parts[k] = strconv.Itoa(id)
				}
				b.WriteString("do = [" + strings.Join(parts, ", ") + "]\n")
			}
		}
	}
	out := []byte(b.String())
	back, err := ParseDialogue(d.File, out)
	if err != nil {
		return nil, fmt.Errorf("dialogue %s: written file does not read back: %w", d.File, err)
	}
	if !SameDialogue(d, back) {
		return nil, fmt.Errorf("dialogue %s: written file does not read back as given", d.File)
	}
	return out, nil
}

// writeDialogueText writes the languages of t in the table's order.
func writeDialogueText(b *strings.Builder, t DialogueText) {
	for _, l := range locale.All() {
		if s, ok := t[l.Code]; ok {
			b.WriteString(l.Code + " = " + writeString(s) + "\n")
		}
	}
}

// SameDialogue reports that a and b hold the same nodes and replies, leaving
// out the file name and the source lines.
func SameDialogue(a, b Dialogue) bool {
	return slices.EqualFunc(a.Nodes, b.Nodes, func(x, y DialogueNode) bool {
		return x.ID == y.ID && x.Name == y.Name && sameText(x.Text, y.Text) &&
			x.Speaks == y.Speaks && (!x.Speaks || x.Speaker == y.Speaker) && x.Next == y.Next && sameVariants(x.Variants, y.Variants) &&
			slices.EqualFunc(x.Replies, y.Replies, func(r, s DialogueReply) bool {
				return r.Next == s.Next && r.Once == s.Once && (!r.Once || r.Mark == s.Mark) && r.Disabled == s.Disabled &&
					sameText(r.Text, s.Text) && slices.Equal(r.When, s.When) && slices.Equal(r.Do, s.Do)
			})
	})
}

func sameVariants(a, b map[string]DialogueText) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || !sameText(v, w) {
			return false
		}
	}
	return true
}

func sameText(a, b DialogueText) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}

// DialogueRule names a check of CheckDialogue.
type DialogueRule uint8

const (
	// RuleNoNode: the file has no node.
	RuleNoNode DialogueRule = iota + 1
	// RuleReplies: a node offers more than MaxDialogueReplies replies.
	RuleReplies
	// RuleNextMissing: a reply's next names no node of the file.
	RuleNextMissing
	// RuleNameEmpty: a node has no name.
	RuleNameEmpty
	// RuleNameTwice: two nodes have one name.
	RuleNameTwice
	// RuleNoText: a node or reply has no text in any language.
	RuleNoText
	// RuleMarkPast: a once reply's mark is not one of the map's variables.
	RuleMarkPast
)

// DialogueIssue is one refusal of CheckDialogue. Node is the node's name,
// Reply the reply's number from 1 (0 for the node), Other the next node's id
// or the mark.
type DialogueIssue struct {
	Rule  DialogueRule
	Node  string
	Reply int
	Other int
}

// Text is the issue in plain English, for logs; the editor words it itself.
func (i DialogueIssue) Text() string {
	switch i.Rule {
	case RuleNoNode:
		return "no node"
	case RuleReplies:
		return fmt.Sprintf("node %s offers more than %d replies", i.Node, MaxDialogueReplies)
	case RuleNextMissing:
		if i.Reply == 0 {
			return fmt.Sprintf("node %s: next node %d does not exist", i.Node, i.Other)
		}
		return fmt.Sprintf("node %s reply %d: next node %d does not exist", i.Node, i.Reply, i.Other)
	case RuleNameEmpty:
		return "a node has no name"
	case RuleNameTwice:
		return fmt.Sprintf("node name %s is used twice", i.Node)
	case RuleNoText:
		return fmt.Sprintf("node %s reply %d has no text", i.Node, i.Reply)
	}
	return fmt.Sprintf("node %s reply %d: variable %d is not a variable of the map", i.Node, i.Reply, i.Other)
}

// CheckDialogue lists what keeps d from being written: no node, a fifth
// reply, a next node that does not exist, an empty (when names) or repeated
// node name, a node or reply with no text, and, when variables is not 0, a
// once reply whose mark is not below it. A node with no name is shown by its
// id.
func CheckDialogue(d Dialogue, variables int, names bool) []DialogueIssue {
	var out []DialogueIssue
	if len(d.Nodes) == 0 {
		return []DialogueIssue{{Rule: RuleNoNode}}
	}
	seen := map[string]bool{}
	for _, n := range d.Nodes {
		label := n.Name
		if label == "" {
			label = "#" + strconv.Itoa(n.ID)
		}
		switch {
		case n.Name == "" && names:
			out = append(out, DialogueIssue{Rule: RuleNameEmpty, Node: label})
		case seen[n.Name] && n.Name != "":
			out = append(out, DialogueIssue{Rule: RuleNameTwice, Node: label})
		}
		seen[n.Name] = true
		if len(n.Text) == 0 {
			out = append(out, DialogueIssue{Rule: RuleNoText, Node: label})
		}
		if len(n.Replies) > MaxDialogueReplies {
			out = append(out, DialogueIssue{Rule: RuleReplies, Node: label})
		}
		if _, ok := d.Node(n.Next); n.Next != 0 && !ok {
			out = append(out, DialogueIssue{Rule: RuleNextMissing, Node: label, Other: n.Next})
		}
		for k, r := range n.Replies {
			if len(r.Text) == 0 {
				out = append(out, DialogueIssue{Rule: RuleNoText, Node: label, Reply: k + 1})
			}
			if _, ok := d.Node(r.Next); r.Next != 0 && !ok {
				out = append(out, DialogueIssue{Rule: RuleNextMissing, Node: label, Reply: k + 1, Other: r.Next})
			}
			if r.Once && variables != 0 && r.Mark >= variables {
				out = append(out, DialogueIssue{Rule: RuleMarkPast, Node: label, Reply: k + 1, Other: r.Mark})
			}
		}
	}
	return out
}

// NamedNodes reports that some node of d carries a name, so every node must.
func (d Dialogue) NamedNodes() bool {
	return slices.ContainsFunc(d.Nodes, func(n DialogueNode) bool { return n.Name != "" })
}
