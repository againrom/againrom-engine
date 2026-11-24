package mod

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/scriptprofile"
)

// A dialogue file is one interactive dialogue of a mission, read from
// maps/<stem>.dialogs/<name>.toml. It holds [[node]] tables, each followed by
// the [[reply]] tables that belong to it. A text is written once per language
// under the language's code. A node may name its speaker (speaker = the
// speaker record), tell its words apart by the hero's sex or class under keys
// such as en_woman, en_mage or en_man_fighter, and continue to a node with
// next when it has no replies. A reply's next is the node it pages to (0 or
// absent ends the dialogue); a node may carry the name the editor shows; do
// lists action node ids and when lists condition pairs by condition node ids
// and comparison code, as a trigger does; once offers the reply until it is chosen and records the choice in
// script variable mark; unavailable is hidden (default) or disabled. A reply
// holds as many conditions and actions as an extended trigger does.

// The dialogue file's bounds.
const (
	// MaxDialogueReplies is the most replies one node holds.
	MaxDialogueReplies = 4
	// MaxDialogueNodes is the most nodes one file holds.
	MaxDialogueNodes = 256
	// MaxDialogueVariable is the highest script variable a mark names.
	MaxDialogueVariable = 4095
	// maxDialogueID bounds a node id and a node reference.
	maxDialogueID = 1 << 20
)

// dialogueName is the name of a dialogue file, without its extension.
var dialogueName = regexp.MustCompile(`^[a-z0-9_]{1,32}$`)

// ValidDialogueName reports that name can name a dialogue file.
func ValidDialogueName(name string) bool { return dialogueName.MatchString(name) }

// DialogueFiles is the dialogue files one map folder holds, keyed by name.
// entries are the folder's names, a nested path is not one, and read gives
// the bytes of an entry, false when the view holds none. A file over limit
// bytes is left out; limit 0 bounds none (an editor draft is never dropped).
// The game, the editor and the Globals page all read the folder here.
func DialogueFiles(entries []string, limit int, read func(entry string) ([]byte, bool)) map[string][]byte {
	files := map[string][]byte{}
	for _, e := range entries {
		name, isTOML := strings.CutSuffix(e, ".toml")
		if !isTOML || strings.Contains(e, "/") || !ValidDialogueName(name) {
			continue
		}
		if b, ok := read(e); ok && (limit == 0 || len(b) <= limit) {
			files[name] = b
		}
	}
	return files
}

// DialogueText is one text in each language the file wrote it in.
type DialogueText map[string]string

// Words is the text in lang, else in the language paired with it, else the first
// language the file wrote by code order, else "".
func (t DialogueText) Words(lang string) string {
	if s, ok := t[lang]; ok {
		return s
	}
	if s, ok := t[locale.Other(lang)]; ok {
		return s
	}
	for _, code := range slices.Sorted(maps.Keys(t)) {
		return t[code]
	}
	return ""
}

// DialogueCondition is one condition pair: the two condition node ids a
// trigger's pair names and its comparison code.
type DialogueCondition struct{ Left, Right, Cmp int }

// DialogueReply is one answer of a node.
type DialogueReply struct {
	Line     int
	Text     DialogueText
	Next     int
	Once     bool
	Mark     int
	Disabled bool
	When     []DialogueCondition
	Do       []int
}

// DialogueNode is one page of a dialogue and the replies it offers. Speaker
// is the speaker record whose face the page shows when Speaks. Variants are
// the page's words for the hero's sex or class, by variant key, each in the
// languages the file wrote; Text is the words for every other hero. Next is the
// node a page with no replies continues to (0 ends the dialogue).
type DialogueNode struct {
	Line     int
	ID       int
	Name     string
	Text     DialogueText
	Speaker  int
	Speaks   bool
	Variants map[string]DialogueText
	Next     int
	Replies  []DialogueReply
}

// The hero variant keys, by the hero's sex, class or both. A node uses the
// keys of one family.
var dialogueVariantFamilies = [][]string{
	{"woman", "man"},
	{"mage", "fighter"},
	{"woman_mage", "woman_fighter", "man_mage", "man_fighter"},
}

// DialogueVariantKeys are the keys a node's Variants may hold, family by family.
func DialogueVariantKeys() []string {
	return slices.Concat(dialogueVariantFamilies...)
}

func dialogueVariantKnown(key string) bool { return slices.Contains(DialogueVariantKeys(), key) }

// TextFor is the page's words in lang for a hero of the sex and class given:
// the variant of the hero's sex and class, of the sex, of the class, in that
// order, in lang, else the page's own words.
func (n DialogueNode) TextFor(lang string, female, mage bool) string {
	sex, class := "man", "fighter"
	if female {
		sex = "woman"
	}
	if mage {
		class = "mage"
	}
	for _, key := range []string{sex + "_" + class, sex, class} {
		if s, ok := n.Variants[key][lang]; ok {
			return s
		}
	}
	return n.Text.Words(lang)
}

// Dialogue is a parsed dialogue file; its first node opens it.
type Dialogue struct {
	File  string
	Nodes []DialogueNode
}

// Node is the node with id.
func (d Dialogue) Node(id int) (DialogueNode, bool) {
	i := slices.IndexFunc(d.Nodes, func(n DialogueNode) bool { return n.ID == id })
	if i < 0 {
		return DialogueNode{}, false
	}
	return d.Nodes[i], true
}

// ParseDialogue reads the dialogue file named file. A refusal names the file
// and the line.
func ParseDialogue(file string, data []byte) (Dialogue, error) {
	tables, err := parseTOMLWith(data, tomlOptions{arrays: true})
	if err != nil {
		return Dialogue{}, fileError(file, err)
	}
	bad := func(line int, format string, args ...any) error {
		return &ItemFileError{File: file, Line: line, Msg: fmt.Sprintf(format, args...)}
	}
	if len(tables[0].pairs) != 0 {
		p := tables[0].pairs[0]
		return Dialogue{}, bad(p.line, "%s is outside a [[node]] table", p.key)
	}
	d := Dialogue{File: file}
	for _, t := range tables[1:] {
		switch {
		case !t.array || t.name != "node" && t.name != "reply":
			return Dialogue{}, bad(t.line, "unknown table %q (this file holds [[node]] and [[reply]])", t.name)
		case t.name == "node":
			if len(d.Nodes) >= MaxDialogueNodes {
				return Dialogue{}, bad(t.line, "more than %d nodes", MaxDialogueNodes)
			}
			n, err := readDialogueNode(t, bad)
			if err != nil {
				return Dialogue{}, err
			}
			if slices.ContainsFunc(d.Nodes, func(o DialogueNode) bool { return o.ID == n.ID }) {
				return Dialogue{}, bad(t.line, "node id %d is given twice", n.ID)
			}
			d.Nodes = append(d.Nodes, n)
		default:
			if len(d.Nodes) == 0 {
				return Dialogue{}, bad(t.line, "[[reply]] comes before any [[node]]")
			}
			r, err := readDialogueReply(t, bad)
			if err != nil {
				return Dialogue{}, err
			}
			last := &d.Nodes[len(d.Nodes)-1]
			if len(last.Replies) == MaxDialogueReplies {
				return Dialogue{}, bad(t.line, "node %d holds more than %d replies", last.ID, MaxDialogueReplies)
			}
			last.Replies = append(last.Replies, r)
		}
	}
	if len(d.Nodes) == 0 {
		return Dialogue{}, bad(0, "no [[node]]")
	}
	for _, n := range d.Nodes {
		if _, ok := d.Node(n.Next); n.Next != 0 && !ok {
			return Dialogue{}, bad(n.Line, "next = %d names no node", n.Next)
		}
		for _, r := range n.Replies {
			if _, ok := d.Node(r.Next); r.Next != 0 && !ok {
				return Dialogue{}, bad(r.Line, "next = %d names no node", r.Next)
			}
		}
	}
	return d, nil
}

type dialogueBad func(line int, format string, args ...any) error

// dialogueInt reads an integer key within 0..max.
func dialogueInt(p tomlPair, max int64, bad dialogueBad) (int, error) {
	if p.val.kind != tomlInt {
		return 0, bad(p.line, "%s must be an integer, not %s", p.key, p.val.kind)
	}
	if p.val.num < 0 || p.val.num > max {
		return 0, bad(p.line, "%s is %d; use 0 to %d", p.key, p.val.num, max)
	}
	return int(p.val.num), nil
}

// dialogueText reads a language key; ok is false for any other key.
func dialogueText(p tomlPair, text *DialogueText, bad dialogueBad) (ok bool, err error) {
	if !locale.Known(p.key) {
		return false, nil
	}
	if p.val.kind != tomlString {
		return true, bad(p.line, "%s must be a string, not %s", p.key, p.val.kind)
	}
	if p.val.str == "" {
		return true, bad(p.line, "%s is empty", p.key)
	}
	if *text == nil {
		*text = DialogueText{}
	}
	(*text)[p.key] = p.val.str
	return true, nil
}

func readDialogueNode(t tomlTable, bad dialogueBad) (DialogueNode, error) {
	n := DialogueNode{Line: t.line}
	have := false
	for _, p := range t.pairs {
		if ok, err := dialogueText(p, &n.Text, bad); ok || err != nil {
			if err != nil {
				return n, err
			}
			continue
		}
		if lang, variant, ok := strings.Cut(p.key, "_"); ok && locale.Known(lang) && dialogueVariantKnown(variant) {
			var t DialogueText
			if err := readVariantText(p, lang, n.Variants[variant], &t, bad); err != nil {
				return n, err
			}
			if n.Variants == nil {
				n.Variants = map[string]DialogueText{}
			}
			n.Variants[variant] = t
			continue
		}
		switch p.key {
		case "speaker":
			v, err := dialogueInt(p, maxDialogueID, bad)
			if err != nil {
				return n, err
			}
			n.Speaker, n.Speaks = v, true
			continue
		case "next":
			v, err := dialogueInt(p, maxDialogueID, bad)
			if err != nil {
				return n, err
			}
			n.Next = v
			continue
		}
		if p.key == "name" {
			if p.val.kind != tomlString || p.val.str == "" {
				return n, bad(p.line, "name must be a non-empty string")
			}
			n.Name = p.val.str
			continue
		}
		if p.key != "id" {
			return n, bad(p.line, "unknown key %q in [[node]]", p.key)
		}
		id, err := dialogueInt(p, maxDialogueID, bad)
		if err != nil {
			return n, err
		}
		if id < 1 {
			return n, bad(p.line, "id is 0; use 1 or more")
		}
		n.ID, have = id, true
	}
	if err := checkVariantFamily(n.Variants, t.line, bad); err != nil {
		return n, err
	}
	switch {
	case !have:
		return n, bad(t.line, "[[node]] has no id")
	case len(n.Text) == 0:
		return n, bad(t.line, "node %d has no text (write one key per language, such as en = \"...\")", n.ID)
	}
	return n, nil
}

// readVariantText reads one language of a variant's text, adding it to the
// languages already read in have.
func readVariantText(p tomlPair, lang string, have DialogueText, out *DialogueText, bad dialogueBad) error {
	if p.val.kind != tomlString || p.val.str == "" {
		return bad(p.line, "%s must be a non-empty string", p.key)
	}
	*out = DialogueText{}
	maps.Copy(*out, have)
	(*out)[lang] = p.val.str
	return nil
}

// checkVariantFamily refuses a node whose variant keys mix families.
func checkVariantFamily(v map[string]DialogueText, line int, bad dialogueBad) error {
	for _, family := range dialogueVariantFamilies {
		for key := range v {
			if slices.Contains(family, key) {
				for other := range v {
					if !slices.Contains(family, other) {
						return bad(line, "variants %q and %q are of different kinds (use woman and man, mage and fighter, or the four of both)", key, other)
					}
				}
				return nil
			}
		}
	}
	return nil
}

func readDialogueReply(t tomlTable, bad dialogueBad) (DialogueReply, error) {
	r := DialogueReply{Line: t.line}
	hasMark := false
	for _, p := range t.pairs {
		ok, err := dialogueText(p, &r.Text, bad)
		if err != nil {
			return r, err
		}
		if ok {
			continue
		}
		switch p.key {
		case "next":
			r.Next, err = dialogueInt(p, maxDialogueID, bad)
		case "mark":
			r.Mark, err = dialogueInt(p, MaxDialogueVariable, bad)
			hasMark = true
		case "once":
			if p.val.kind != tomlBool {
				err = bad(p.line, "once must be true or false, not %s", p.val.kind)
			}
			r.Once = p.val.flag
		case "unavailable":
			switch {
			case p.val.kind != tomlString:
				err = bad(p.line, "unavailable must be a string, not %s", p.val.kind)
			case p.val.str == "disabled":
				r.Disabled = true
			case p.val.str != "hidden":
				err = bad(p.line, "unavailable is %q; use \"hidden\" or \"disabled\"", p.val.str)
			}
		case "do":
			r.Do, err = dialogueList(p, scriptprofile.Extended.Actions, bad, func(v tomlValue, line int) (int, error) {
				if v.kind != tomlInt || v.num < 1 || v.num > maxDialogueID {
					return 0, bad(line, "do holds action node ids, whole numbers from 1")
				}
				return int(v.num), nil
			})
		case "when":
			var pairs []DialogueCondition
			pairs, err = dialogueList(p, scriptprofile.Extended.Pairs, bad, func(v tomlValue, line int) (DialogueCondition, error) {
				return readDialogueCondition(v, line, bad)
			})
			r.When = pairs
		default:
			err = bad(p.line, "unknown key %q in [[reply]]", p.key)
		}
		if err != nil {
			return r, err
		}
	}
	switch {
	case len(r.Text) == 0:
		return r, bad(t.line, "[[reply]] has no text (write one key per language, such as en = \"...\")")
	case r.Once && !hasMark:
		return r, bad(t.line, "once = true needs mark = the script variable that records the choice")
	case hasMark && !r.Once:
		return r, bad(t.line, "mark is set but once is not true")
	}
	return r, nil
}

func dialogueList[T any](p tomlPair, max int, bad dialogueBad, item func(tomlValue, int) (T, error)) ([]T, error) {
	if p.val.kind != tomlArray {
		return nil, bad(p.line, "%s must be an array, not %s", p.key, p.val.kind)
	}
	if len(p.val.list) > max {
		return nil, bad(p.line, "%s holds more than %d entries", p.key, max)
	}
	var out []T
	for _, v := range p.val.list {
		x, err := item(v, p.line)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func readDialogueCondition(v tomlValue, line int, bad dialogueBad) (DialogueCondition, error) {
	var c DialogueCondition
	if v.kind != tomlInline {
		return c, bad(line, "when holds { left = N, right = N, cmp = N } tables")
	}
	seen := map[string]bool{}
	for _, p := range v.tbl {
		p.line = line
		var err error
		switch p.key {
		case "left":
			c.Left, err = dialogueInt(p, maxDialogueID, bad)
		case "right":
			c.Right, err = dialogueInt(p, maxDialogueID, bad)
		case "cmp":
			c.Cmp, err = dialogueInt(p, 5, bad)
		default:
			err = bad(line, "unknown key %q in a when table", p.key)
		}
		if err != nil {
			return c, err
		}
		seen[p.key] = true
	}
	if !seen["left"] || !seen["right"] {
		return c, bad(line, "a when table needs left and right condition node ids")
	}
	return c, nil
}
