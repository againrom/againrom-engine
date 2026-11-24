package ui

import (
	"fmt"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"againrom/pkg/words"
)

// The reply part of the dialog window. The window is the message window:
// its left column lists the lines, its right column shows the selected one.
// A line says what follows it: the next line, the end of the dialogue or a
// choice of up to four replies. A reply holds its text, the line it leads
// to, whether it is once, how it shows when its condition fails, and one
// condition and one outcome edited as the trigger wizard's rows. A dialogue
// with no reply choice is written as message pages; one with a reply choice
// is written as a dialogue file.

// The reply fields of the window, after the cards' own.
const (
	dlgFile        = "d.file"
	dlgAfter       = "d.after"
	dlgReplyText   = "d.rt"
	dlgNext        = "d.rnext"
	dlgOnce        = "d.ronce"
	dlgUnavailable = "d.runavail"
)

// The row actions of the reply part.
const (
	actAddReply    = "addreply"
	actRemoveReply = "removereply"
)

// afterKind is what follows a line.
type afterKind uint8

const (
	afterNext afterKind = iota
	afterReply
	afterEnd
)

// dlgReply is one reply of a line as drafted. next is the number of the card
// it leads to, 0 for the end of the dialogue.
type dlgReply struct {
	text     map[string]string
	next     int
	once     bool
	mark     int
	markSet  bool // mark came from the file
	disabled bool
	rows     []wizRow
}

// dlgLine is what the window keeps of a line beside its card: what follows
// it, its replies and the one shown open (-1 for none), the node name and
// the node as the file held it, for the languages and variants the window
// does not edit.
type dlgLine struct {
	after   afterKind
	replies []dlgReply
	open    int
	name    string
	keep    *DialogueNodeData
}

// dialogSource is the dialogue file the window shows: its name, whether it
// is not written yet, whether the mod window opened it, and the cards and
// lines read from it.
type dialogSource struct {
	name    string
	file    string // the name a new dialogue is typed under
	fresh   bool
	fromMod bool
	// replies and foreign say the file, as read, has a reply and has a text
	// in a language other than the editor's; both stay false for a new one.
	replies, foreign bool
	parts            []InspectionMessagePart
	lines            map[int]*dlgLine
}

// dialogueHeroKeys are the dialogue file's hero variant keys by the heroes a
// cell holds.
var dialogueHeroKeys = []struct {
	heroes HeroAudience
	key    string
}{
	{HeroWomanMage | HeroWomanFighter, "woman"},
	{HeroManMage | HeroManFighter, "man"},
	{HeroWomanMage | HeroManMage, "mage"},
	{HeroWomanFighter | HeroManFighter, "fighter"},
	{HeroWomanMage, "woman_mage"},
	{HeroWomanFighter, "woman_fighter"},
	{HeroManMage, "man_mage"},
	{HeroManFighter, "man_fighter"},
}

// heroKey is the variant key of cell heroes, empty for none.
func heroKey(h HeroAudience) string {
	for _, k := range dialogueHeroKeys {
		if k.heroes == h {
			return k.key
		}
	}
	return ""
}

// snippetRunes is the most letters of a line's text its list entry shows.
const snippetRunes = 28

func (e *MapEditor) dialogueHost() (InspectionDialogues, bool) {
	if e.doc == nil {
		return nil, false
	}
	h, ok := Seam[InspectionDialogues](e.doc.Edits)
	return h, ok
}

// DialoguesOffered reports that the open map keeps dialogues.
func (e *MapEditor) DialoguesOffered() bool {
	_, ok := e.dialogueHost()
	return ok
}

func (e *MapEditor) dialogueExists(name string) bool {
	h, ok := e.dialogueHost()
	return ok && name != "" && slices.Contains(h.DialogueNames(), name)
}

// DialogueOpen reports that the window shows a dialogue file.
func (e *MapEditor) DialogueOpen() bool { return e.dialog != nil }

// DialogueField is the window's field named key, -1 for none, for a headless
// driver.
func (e *MapEditor) DialogueField(key string) int {
	if e.messagePage == 0 {
		return -1
	}
	e.objectSpec()
	if key == DialogueLineText || key == DialogueLineSpeaker {
		n, ok := e.selectedNumber()
		start, end, found := e.cardOf(n)
		if !ok || !found {
			return -1
		}
		for c := start; c < end; c++ {
			if p := e.obj.messageParts[c]; !p.Never && !p.Ends {
				if key == DialogueLineSpeaker {
					return 2*c + 1
				}
				return 2 * c
			}
		}
		return -1
	}
	for i, f := range e.obj.fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

// The keys DialogueField answers for the line on show.
const (
	DialogueLineText    = "line.text"
	DialogueLineSpeaker = "line.speaker"
)

// DialogueLines is how many lines the window lists, struck ones not counted.
func (e *MapEditor) DialogueLines() int {
	e.objectSpec()
	return len(e.liveNumbers())
}

// DialogueSelect shows the line at place rank, from 1, of the list.
func (e *MapEditor) DialogueSelect(rank int) bool {
	e.objectSpec()
	live := e.liveNumbers()
	if e.messagePage == 0 || rank < 1 || rank > len(live) {
		return false
	}
	e.selectLine(live[rank-1])
	e.objectSpec()
	return true
}

// DialogueListRow is the window rectangle of the list entry of the line at
// place rank, from 1.
func (e *MapEditor) DialogueListRow(rank int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	live := e.liveNumbers()
	if !ok || rank < 1 || rank > len(live) {
		return image.Rectangle{}, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if n, has := e.obj.rowSelect[rr.Index]; has && n == live[rank-1] {
			return rr.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// DialogueReplyRow is the window rectangle of the header of reply i, from 0,
// of the line on show.
func (e *MapEditor) DialogueReplyRow(i int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if n, has := e.obj.rowReply[rr.Index]; has && n == i {
			return rr.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// DialogueButton is the window rectangle of the first shown button whose
// action is act, for a headless driver.
func (e *MapEditor) DialogueButton(act string) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if k := slices.Index(e.obj.rowActs[rr.Index], act); k >= 0 && k < len(rr.Buttons) {
			return rr.Buttons[k], true
		}
	}
	return image.Rectangle{}, false
}

// DialogueRowActions are the actions of the buttons shown on the row of the
// window's field key, in button order.
func (e *MapEditor) DialogueRowActions(key string) []string {
	i := e.DialogueField(key)
	if i < 0 {
		return nil
	}
	return slices.Clone(e.obj.rowActs[e.fieldRow(i)])
}

// DialogueText is the text of the window's field key as typed.
func (e *MapEditor) DialogueText(key string) string {
	if i := e.DialogueField(key); i >= 0 {
		return e.obj.buf[i]
	}
	return ""
}

// replyAllowed reports that the shown message window offers a reply choice:
// it belongs to a trigger's Start dialog action or shows a dialogue file.
func (e *MapEditor) replyAllowed() bool {
	if e.messagePage == 0 {
		return false
	}
	if _, ok := e.dialogueHost(); !ok {
		return false
	}
	return e.dialog != nil || e.messageFor != nil
}

// lineAt is the draft of the line whose card has number.
func (e *MapEditor) lineAt(number int) *dlgLine {
	o := &e.obj
	if o.lines == nil {
		o.lines = map[int]*dlgLine{}
	}
	l := o.lines[number]
	if l == nil {
		l = &dlgLine{open: -1}
		o.lines[number] = l
	}
	return l
}

// liveNumbers are the numbers of the cards not struck, in order.
func (e *MapEditor) liveNumbers() []int {
	var out []int
	for _, p := range e.obj.messageParts {
		if !e.obj.messageGone[p.Number] && !slices.Contains(out, p.Number) {
			out = append(out, p.Number)
		}
	}
	return out
}

// selectedNumber is the number of the line on show: the one chosen while its
// card stands, else the first.
func (e *MapEditor) selectedNumber() (int, bool) {
	live := e.liveNumbers()
	if len(live) == 0 {
		return 0, false
	}
	if !slices.Contains(live, e.obj.sel) {
		e.obj.sel = live[0]
	}
	return e.obj.sel, true
}

// cardOf is the cells [start, end) of the card with number.
func (e *MapEditor) cardOf(number int) (start, end int, ok bool) {
	parts := e.obj.messageParts
	start = slices.IndexFunc(parts, func(p InspectionMessagePart) bool { return p.Number == number })
	if start < 0 {
		return 0, 0, false
	}
	for end = start + 1; end < len(parts) && parts[end].Number == number; end++ {
	}
	return start, end, true
}

// lineSnippet is the first words of the line with number, in the font's
// bytes.
func (e *MapEditor) lineSnippet(number int) string {
	o := &e.obj
	start, end, ok := e.cardOf(number)
	if !ok {
		return ""
	}
	for c := start; c < end; c++ {
		if o.messageParts[c].Never || o.messageParts[c].Ends || 2*c >= len(o.buf) {
			continue
		}
		s, _, _ := strings.Cut(strings.TrimSpace(o.buf[2*c]), "\n")
		s = strings.TrimSpace(s)
		if utf8.RuneCountInString(s) > snippetRunes {
			s = string([]rune(s)[:snippetRunes]) + "..."
		}
		return e.editorHostText(s)
	}
	return ""
}

// lineRanks are the places, from 1, of the numbers of the live cards.
func (e *MapEditor) lineRanks() map[int]int {
	ranks := map[int]int{}
	for k, n := range e.liveNumbers() {
		ranks[n] = k + 1
	}
	return ranks
}

// lineLabel is a line as the lists name it: its place and first words.
func (e *MapEditor) lineLabel(number int) string {
	rank := e.lineRanks()[number]
	if s := e.lineSnippet(number); s != "" {
		return fmt.Sprintf("%d. %s", rank, s)
	}
	return fmt.Sprintf("%d.", rank)
}

// dialogueLanguage is the dialogue code of the editor's language.
func (e *MapEditor) dialogueLanguage() string {
	b := e.book
	if !e.hasBook {
		b = words.For("")
	}
	return b.Lang()
}

// hostUTF8 is text typed or kept in the font's encoding as UTF-8.
func (e *MapEditor) hostUTF8(s string) string {
	if e.hostDecode != nil {
		return e.hostDecode(s)
	}
	return s
}

// nodeName is the default name of node id: the editor's word, in English
// when the face cannot draw the editor's letters.
func (e *MapEditor) nodeName(id int) string {
	s := fmt.Sprintf(e.plainWord("editor.dialogue.nodeword"), id)
	if e.hostUTF8(e.editorHostText(s)) != s {
		return fmt.Sprintf(words.For("").Text("editor.dialogue.nodeword"), id)
	}
	return s
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// replyHead is the words a reply's header shows: its text in the editor's
// language, else in another.
func (e *MapEditor) replyHead(r dlgReply) string {
	lang := e.dialogueLanguage()
	if s := strings.TrimSpace(r.text[lang]); s != "" {
		return e.editorHostText(s)
	}
	for _, s := range slices.Sorted(maps.Keys(r.text)) {
		if t := strings.TrimSpace(r.text[s]); t != "" {
			return e.editorHostText(t)
		}
	}
	return e.word("editor.dialogue.noreply.text")
}

// replyRows are the wizard rows of a reply's condition and outcome as the
// map holds them.
func (e *MapEditor) replyRows(r DialogueReplyData) []wizRow {
	t := &InspectionTrigger{}
	for _, p := range r.Pairs {
		t.Left, t.Right = append(t.Left, p.Left.ID), append(t.Right, p.Right.ID)
		t.Cmp, t.Group = append(t.Cmp, p.Cmp), append(t.Group, p.Group)
	}
	for _, a := range r.Actions {
		t.Acts = append(t.Acts, a.ID)
	}
	return e.wizardRows(t)
}

// dialogueParts are the cards and lines of dialogue data: a card per node,
// holding a cell for each hero the node's variants tell apart, with the
// text of the editor's language.
func (e *MapEditor) dialogueParts(data DialogueData) ([]InspectionMessagePart, map[int]*dlgLine, error) {
	lang := e.dialogueLanguage()
	var parts []InspectionMessagePart
	lines := map[int]*dlgLine{}
	for i, n := range data.Nodes {
		node := n
		line := &dlgLine{open: -1, name: n.Name, keep: &node}
		sex, class := false, false
		for key := range n.Variants {
			switch {
			case strings.Contains(key, "_"):
				sex, class = true, true
			case key == "woman" || key == "man":
				sex = true
			default:
				class = true
			}
		}
		speaker := int64(-1)
		if n.Speaks {
			speaker = int64(n.Speaker)
		}
		for _, h := range HeroCells(sex, class) {
			body := n.Text[lang]
			if key := heroKey(h); key != "" {
				if v, ok := n.Variants[key][lang]; ok {
					body = v
				}
			}
			parts = append(parts, InspectionMessagePart{Source: -1, Number: n.ID, Body: body, Speaker: speaker, Heroes: h})
		}
		switch {
		case len(n.Replies) > 0:
			line.after = afterReply
		case n.Next > 0:
			if i+1 >= len(data.Nodes) || data.Nodes[i+1].ID != n.Next {
				return nil, nil, fmt.Errorf(e.plainWord("editor.dialogue.err.jump"), i+1)
			}
		case i < len(data.Nodes)-1:
			line.after = afterEnd
		}
		for _, r := range n.Replies {
			line.replies = append(line.replies, dlgReply{text: maps.Clone(r.Text), next: r.Next, once: r.Once, mark: r.Mark, markSet: r.Once, disabled: r.Disabled, rows: e.replyRows(r)})
		}
		lines[n.ID] = line
	}
	for _, p := range parts[:min(1, len(parts))] {
		if l := lines[p.Number]; len(l.replies) > 0 {
			l.open = 0
		}
	}
	return parts, lines, nil
}

// dialogueNode is the node a Start dialog action of the interactive kind, or
// of the message kind, is written as; id is the action it rewrites.
func dialogueNode(interactive bool, label string, id uint32) WizardNode {
	for _, it := range wizActions {
		if it.opcode != 2 || it.interactive != interactive || it.subject != subjectMission {
			continue
		}
		types, _, fixed := it.itemSlots(false)
		n := WizardNode{ID: id, Opcode: it.opcode, Types: types, Label: label}
		for k, f := range it.fixed {
			n.Values[fixed+k] = f[1]
		}
		return n
	}
	return WizardNode{ID: id}
}

// dialogData is the draft as dialogue data: a node per live card, in order,
// the cells of a card as the node's text and hero variants, the other
// languages the file held kept as they were.
func (e *MapEditor) dialogData() (DialogueData, error) {
	var data DialogueData
	o := &e.obj
	h, _ := e.dialogueHost()
	if !e.captureMessageParts() {
		return data, fmt.Errorf("%s", o.note)
	}
	e.dialogPull()
	lang := e.dialogueLanguage()
	type card struct {
		number int
		cells  []InspectionMessagePart
	}
	var cards []card
	for _, p := range e.liveMessageParts() {
		if len(cards) == 0 || cards[len(cards)-1].number != p.Number {
			cards = append(cards, card{number: p.Number})
		}
		cards[len(cards)-1].cells = append(cards[len(cards)-1].cells, p)
	}
	ids, most := map[int]int{}, 0
	for _, c := range cards {
		most = max(most, c.number)
	}
	for _, c := range cards {
		if c.number > 0 {
			ids[c.number] = c.number
		} else {
			most++
			ids[c.number] = most
		}
	}
	named := false
	for _, c := range cards {
		named = named || o.lines[c.number] != nil && o.lines[c.number].name != ""
	}
	for k, c := range cards {
		line := o.lines[c.number]
		if line == nil {
			line = &dlgLine{open: -1}
		}
		node := DialogueNodeData{ID: ids[c.number], Name: line.name}
		if named && node.Name == "" {
			node.Name = e.nodeName(node.ID)
		}
		var heroes []HeroAudience
		for _, cell := range c.cells {
			who := ""
			if w := HeroWord(cell.Heroes); w != "" {
				who = " (" + e.plainWord(w) + ")"
			}
			switch {
			case cell.Never:
				return data, fmt.Errorf(e.plainWord("editor.dialogue.err.never"), k+1, who)
			case cell.Ends:
				return data, fmt.Errorf(e.plainWord("editor.dialogue.err.ends"), k+1, who)
			case cell.Other != "":
				return data, fmt.Errorf(e.plainWord("editor.dialogue.err.other"), k+1, who, cell.Other)
			}
			if cell.Speaker != c.cells[0].Speaker {
				return data, fmt.Errorf(e.plainWord("editor.dialogue.err.speakers"), k+1)
			}
			heroes = append(heroes, cell.Heroes)
		}
		if sp := c.cells[0].Speaker; sp >= 0 {
			node.Speaks, node.Speaker = true, int(sp)
		}
		node.Text = map[string]string{}
		sex, class := HeroSplit(heroes)
		set := func(m map[string]string, v string) {
			if strings.TrimSpace(v) != "" {
				m[lang] = v
			}
		}
		if sex || class {
			node.Variants = map[string]map[string]string{}
			for _, cell := range c.cells {
				key := heroKey(cell.Heroes)
				if key == "" {
					return data, fmt.Errorf(e.plainWord("editor.dialogue.err.cells"), k+1)
				}
				if strings.TrimSpace(cell.Body) != "" {
					node.Variants[key] = map[string]string{lang: cell.Body}
				}
			}
		}
		for _, cell := range c.cells {
			if _, ok := node.Text[lang]; !ok {
				set(node.Text, cell.Body)
			}
		}
		if line.keep != nil {
			for l, v := range line.keep.Text {
				if l != lang && v != "" {
					node.Text[l] = v
				}
			}
			for key, m := range line.keep.Variants {
				if _, ok := node.Variants[key]; !ok {
					continue
				}
				for l, v := range m {
					if l != lang && v != "" {
						node.Variants[key][l] = v
					}
				}
			}
		}
		switch {
		case line.after == afterReply && len(line.replies) == 0:
			return data, fmt.Errorf(e.plainWord("editor.dialogue.err.noreplies"), k+1)
		case line.after == afterReply:
			for j, r := range line.replies {
				next := 0
				if r.next != 0 {
					id, ok := ids[r.next]
					if !ok {
						return data, fmt.Errorf(e.plainWord("editor.dialogue.err.gone"), j+1, k+1)
					}
					next = id
				}
				reply := DialogueReplyData{Text: maps.Clone(r.text), Next: next, Once: r.once, Mark: r.mark, Disabled: r.disabled}
				plan, err := e.rowsPlan(WizardTrigger{}, r.rows)
				if err != nil {
					return data, fmt.Errorf(e.plainWord("editor.dialogue.err.reply"), k+1, j+1, err.Error())
				}
				reply.Pairs, reply.Actions = plan.Pairs, plan.Actions
				node.Replies = append(node.Replies, reply)
			}
		case line.after == afterNext && k+1 < len(cards):
			node.Next = ids[cards[k+1].number]
		}
		data.Nodes = append(data.Nodes, node)
	}
	k := 0
	for _, c := range cards {
		line := o.lines[c.number]
		if line == nil || line.after != afterReply {
			k++
			continue
		}
		for j := range line.replies {
			r := &data.Nodes[k].Replies[j]
			if r.Once && r.Mark == 0 && !line.replies[j].markSet {
				mark := h.FreeMark(data)
				if mark < 0 {
					return data, fmt.Errorf("%s", e.plainWord("editor.dialogue.err.nomark"))
				}
				r.Mark = mark
			}
		}
		k++
	}
	return data, nil
}

// dialogueIssueText is a failed check in words, its node named by its place.
func (e *MapEditor) dialogueIssueText(i DialogueIssue, data DialogueData) string {
	if id, ok := strings.CutPrefix(i.Node, "#"); ok {
		n, _ := strconv.Atoi(id)
		for k, node := range data.Nodes {
			if node.ID == n {
				i.Node = strconv.Itoa(k + 1)
			}
		}
	}
	return IssueText(e.plainWord, i)
}

// IssueText is a failed dialogue check in the words word gives.
func IssueText(word func(string) string, i DialogueIssue) string {
	if i.Rule == "editor.dialogue.check.notext" && i.Reply == 0 {
		return fmt.Sprintf(word("editor.dialogue.check.notextnode"), i.Node, i.Reply, i.Other)
	}
	if i.Rule == "editor.dialogue.check.next" && i.Reply == 0 {
		return fmt.Sprintf(word("editor.dialogue.check.nextnode"), i.Node, i.Reply, i.Other)
	}
	return fmt.Sprintf(word(i.Rule), i.Node, i.Reply, i.Other)
}

// DialogueChecks is what the window says about the draft when it is to be
// written as a dialogue file: every failed check and the first refusal of a
// row, as Apply reports it.
func (e *MapEditor) DialogueChecks() []string {
	h, ok := e.dialogueHost()
	if !ok || e.messagePage == 0 {
		return nil
	}
	data, err := e.dialogData()
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, i := range h.DialogueIssues(data) {
		out = append(out, e.dialogueIssueText(i, data))
	}
	return out
}

// dialogueHolding reports that data has a reply and has a text in a language
// other than the editor's.
func (e *MapEditor) dialogueHolding(data DialogueData) (replies, foreign bool) {
	lang := e.dialogueLanguage()
	other := func(m map[string]string) bool {
		for k, v := range m {
			if k != lang && v != "" {
				return true
			}
		}
		return false
	}
	for _, n := range data.Nodes {
		replies = replies || len(n.Replies) > 0
		foreign = foreign || other(n.Text)
		for _, v := range n.Variants {
			foreign = foreign || other(v)
		}
		for _, r := range n.Replies {
			foreign = foreign || other(r.Text)
		}
	}
	return replies, foreign
}

// dialogToFile reports that the draft is written as a dialogue file. A new
// draft is when a line offers a reply choice or a line before the last ends
// the dialogue. A file opened from the mod window stays one. A file opened
// from a trigger stays one when it holds a text in another language or has
// no reply; else it folds into message pages once its last reply is removed.
func (e *MapEditor) dialogToFile() bool {
	if !e.replyAllowed() {
		return false
	}
	if e.dialog != nil && e.messageFor == nil {
		return true
	}
	if src := e.dialog; src != nil && !src.fresh && (src.foreign || !src.replies) {
		return true
	}
	live := e.liveNumbers()
	for k, n := range live {
		l := e.obj.lines[n]
		if l != nil && (l.after == afterReply || l.after == afterEnd && k < len(live)-1) {
			return true
		}
	}
	return false
}

// dialogueTarget is the name the dialogue file is written under.
func (e *MapEditor) dialogueTarget() (string, error) {
	h, _ := e.dialogueHost()
	src := e.dialog
	switch {
	case src == nil:
		for n := 1; ; n++ {
			name := "dialogue"
			if n > 1 {
				name += strconv.Itoa(n)
			}
			if !slices.Contains(h.DialogueNames(), name) {
				return name, nil
			}
		}
	case !src.fresh:
		return src.name, nil
	}
	name := src.name
	if e.messageFor == nil {
		name = src.file
	}
	if !validDialogueName(name) {
		return "", fmt.Errorf("%s", e.plainWord("editor.dialogue.err.name"))
	}
	if e.dialogueExists(name) {
		return "", fmt.Errorf(e.plainWord("editor.dialogue.err.exists"), name)
	}
	return name, nil
}

// applyDialogue writes the draft as a dialogue file, and the trigger's
// action that starts it, as one edit.
func (e *MapEditor) applyDialogue() {
	o := &e.obj
	h, ok := e.dialogueHost()
	if !ok {
		return
	}
	data, err := e.dialogData()
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	var checks []string
	for _, i := range h.DialogueIssues(data) {
		checks = append(checks, e.dialogueIssueText(i, data))
	}
	if len(checks) > 0 {
		o.note = e.editorHostText(strings.Join(checks, "\n"))
		return
	}
	name, err := e.dialogueTarget()
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	var next *InspectionDocument
	if e.messageFor != nil {
		owner, oerr := e.dialogOwner()
		if oerr != nil {
			o.note = e.editorHostText(oerr.Error())
			return
		}
		a := &owner.Trigger.Actions[owner.Action]
		*a = dialogueNode(true, name, a.ID)
		next, err = h.WriteDialogueFor(owner, name, data)
	} else {
		next, err = h.WriteDialogue(name, data)
	}
	if err != nil {
		if !e.askScriptForm(err, e.applyDialogue) {
			o.note = e.editorHostText(err.Error())
		}
		return
	}
	_, _ = e.adopted(next, nil)
	e.reopenDialogue(name)
}

// reopenDialogue shows dialogue name as the file holds it, the line on show
// kept by its place.
func (e *MapEditor) reopenDialogue(name string) {
	h, _ := e.dialogueHost()
	o := &e.obj
	data, err := h.ReadDialogue(name)
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	parts, lines, err := e.dialogueParts(data)
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	place := 0
	if n, ok := e.selectedNumber(); ok {
		place = e.lineRanks()[n] - 1
	}
	fromMod := e.dialog != nil && e.dialog.fromMod
	e.dialog = &dialogSource{name: name, fromMod: fromMod, parts: parts, lines: lines}
	e.dialog.replies, e.dialog.foreign = e.dialogueHolding(data)
	e.messagePage, e.messageFrom = NewMessagePage, NewMessagePage
	e.obj.doc = nil
	e.loadObject()
	if live := e.liveNumbers(); place >= 0 && place < len(live) {
		e.obj.sel = live[place]
		e.dialogFields()
	}
	e.obj.focus = e.firstMessageField()
}

// dropDialogue removes dialogue file name, which no trigger starts any
// more; a file another trigger still starts stays.
func (e *MapEditor) dropDialogue(name string) {
	h, ok := e.dialogueHost()
	if !ok || name == "" || !e.dialogueExists(name) {
		return
	}
	if next, err := h.RemoveDialogue(name); err == nil {
		_, _ = e.adopted(next, nil)
	}
}

// openDialogue shows dialogue name, or a new one of that name, in the
// window, for the owner's action or, with none, from the mod window.
func (e *MapEditor) openDialogue(name string, fresh bool, owner *InspectionMessageOwner) error {
	h, ok := e.dialogueHost()
	if !ok {
		return errNoEditHost
	}
	src := &dialogSource{name: name, fresh: fresh, fromMod: owner == nil}
	if fresh {
		src.parts = []InspectionMessagePart{{Source: -1, Number: 1, Speaker: -1, Heroes: EveryHero}}
		src.lines = map[int]*dlgLine{1: {open: -1}}
		if owner == nil {
			src.name, src.file = "", name
		} else if !validDialogueName(name) {
			n, err := e.dialogueNameFor()
			if err != nil {
				return err
			}
			src.name = n
		}
	} else {
		data, err := h.ReadDialogue(name)
		if err != nil {
			return err
		}
		parts, lines, err := e.dialogueParts(data)
		if err != nil {
			return err
		}
		if len(parts) == 0 {
			return errNoEditHost
		}
		src.parts, src.lines = parts, lines
		src.replies, src.foreign = e.dialogueHolding(data)
	}
	if owner != nil {
		e.objectBack = append(e.objectBack, e.objectPlace())
	}
	e.dialog = src
	e.messageFor, e.messageNew = owner, InspectionMessageOwner{}
	e.messagePage, e.messageFrom = NewMessagePage, NewMessagePage
	e.modWindow = false
	e.objectHidden, e.objectTop = false, 0
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = e.firstMessageField()
	return nil
}

// dialogueNameFor is a name no dialogue has, for a dialogue a trigger starts.
func (e *MapEditor) dialogueNameFor() (string, error) {
	save := e.dialog
	e.dialog = nil
	defer func() { e.dialog = save }()
	return e.dialogueTarget()
}

// OpenDialogue shows dialogue name, or a new one of that name, from the mod
// window.
func (e *MapEditor) OpenDialogue(name string) error {
	if !validDialogueName(name) {
		return fmt.Errorf("%s", e.plainWord("editor.dialogue.err.name"))
	}
	return e.openDialogue(name, !e.dialogueExists(name), nil)
}

// NewDialogue shows a new dialogue the window asks a name for.
func (e *MapEditor) NewDialogue() error { return e.openDialogue("", true, nil) }

// closeDialogue leaves the dialogue file window, back to the mod window when
// it came from there; false when the window was not from the mod window.
func (e *MapEditor) closeDialogue() bool {
	d := e.dialog
	if d == nil || !d.fromMod {
		return false
	}
	e.dialog, e.messagePage, e.messageFor = nil, 0, nil
	e.obj.doc = nil
	e.modWindow = true
	return true
}

// deleteDialogue removes the shown dialogue file when no trigger starts it.
func (e *MapEditor) deleteDialogue() {
	d := e.dialog
	h, ok := e.dialogueHost()
	if !ok || d == nil || d.fresh {
		return
	}
	next, err := h.RemoveDialogue(d.name)
	if err != nil {
		e.obj.note = e.editorHostText(err.Error())
		return
	}
	_, _ = e.adopted(next, nil)
	e.obj.messageChanged = false
	if !e.closeDialogue() {
		e.cancelPopup()
	}
}

// dialogFieldAt is the index of the reply part's field key, -1 for none.
func (e *MapEditor) dialogFieldAt(key string) int {
	o := &e.obj
	for i := max(0, o.extra); i < len(o.fields); i++ {
		if o.fields[i].Key == key {
			return i
		}
	}
	return -1
}

// openReply is the reply of the selected line that shows open, nil for none.
func (e *MapEditor) openReply() *dlgReply {
	n, ok := e.selectedNumber()
	if !ok {
		return nil
	}
	l := e.lineAt(n)
	if l.after != afterReply || l.open < 0 || l.open >= len(l.replies) {
		return nil
	}
	return &l.replies[l.open]
}

// dialogFields builds the reply part's fields after the cards' own: the
// dialogue's name, the line's choice of what follows, and the open reply's
// text, transition, settings and wizard rows.
func (e *MapEditor) dialogFields() {
	o := &e.obj
	o.extra = min(max(o.extra, 0), len(o.fields))
	o.fields, o.buf, o.wiz = o.fields[:o.extra], o.buf[:o.extra], nil
	add := func(f InspectionField) {
		o.fields = append(o.fields, f)
		o.buf = append(o.buf, fieldInput(f))
	}
	if d := e.dialog; d != nil && d.fresh && e.messageFor == nil {
		add(InspectionField{Key: dlgFile, Label: e.word("editor.dialogue.file"), Text: true, TextValue: d.file})
	}
	if !e.replyAllowed() {
		return
	}
	number, ok := e.selectedNumber()
	if !ok {
		return
	}
	line := e.lineAt(number)
	live := e.liveNumbers()
	last := number == live[len(live)-1]
	choices := []InspectionChoice{{Value: int64(afterNext), Label: e.word("editor.dialogue.one.nextline")}, {Value: int64(afterReply), Label: e.word("editor.dialogue.one.replychoice")}, {Value: int64(afterEnd), Label: e.word("editor.dialogue.one.end")}}
	value := int64(line.after)
	if last {
		choices = []InspectionChoice{{Value: int64(afterNext), Label: e.word("editor.dialogue.one.end")}, choices[1]}
		if line.after == afterEnd {
			value = int64(afterNext)
		}
	}
	add(InspectionField{Key: dlgAfter, Label: e.word("editor.dialogue.one.after"), Value: value, Choices: choices})
	r := e.openReply()
	if r == nil {
		return
	}
	lang := e.dialogueLanguage()
	add(InspectionField{Key: dlgReplyText, Label: e.word("editor.dialogue.reply"), Text: true, TextValue: r.text[lang]})
	next := []InspectionChoice{{Value: 0, Label: e.word("editor.dialogue.one.end")}}
	for _, n := range live {
		next = append(next, InspectionChoice{Value: int64(n), Label: e.lineLabel(n)})
	}
	if !slices.ContainsFunc(next, func(c InspectionChoice) bool { return c.Value == int64(r.next) }) {
		next = append(next, InspectionChoice{Value: int64(r.next), Label: strconv.Itoa(r.next)})
	}
	add(InspectionField{Key: dlgNext, Label: e.word("editor.dialogue.transition"), Value: int64(r.next), Choices: next})
	add(InspectionField{Key: dlgOnce, Label: e.word("editor.dialogue.once"), Value: boolInt(r.once), Choices: []InspectionChoice{{Value: 1, Label: e.word("editor.dialogue.once.yes")}, {Value: 0, Label: e.word("editor.dialogue.once.no")}}})
	add(InspectionField{Key: dlgUnavailable, Label: e.word("editor.dialogue.unavailable"), Value: boolInt(r.disabled), Choices: []InspectionChoice{{Value: 0, Label: e.word("editor.dialogue.hidden")}, {Value: 1, Label: e.word("editor.dialogue.disabled")}}})
	o.wiz = &wizardState{rows: r.rows, draw: -1, drawRadius: -1}
	e.wizardRowFields()
}

// dialogSyncRows keeps the open reply's rows as the wizard holds them.
func (e *MapEditor) dialogSyncRows() {
	if w := e.obj.wiz; w != nil {
		if r := e.openReply(); r != nil {
			r.rows = w.rows
		}
	}
}

// dialogPull copies the typed texts and the choices of the reply part into
// the draft; a changed one marks the draft changed.
func (e *MapEditor) dialogPull() {
	o := &e.obj
	if e.messagePage == 0 || o.extra < 0 || o.extra > len(o.fields) || len(o.fields) != len(o.buf) {
		return
	}
	if o.wiz != nil {
		e.pullWizard()
		e.dialogSyncRows()
	}
	number, ok := e.selectedNumber()
	if !ok {
		return
	}
	line, r := e.lineAt(number), e.openReply()
	lang := e.dialogueLanguage()
	for i := o.extra; i < len(o.fields); i++ {
		f, v := o.fields[i], o.buf[i]
		if v != fieldInput(f) {
			o.messageChanged = true
		}
		n, err := strconv.Atoi(strings.TrimSpace(v))
		switch {
		case f.Key == dlgFile && e.dialog != nil:
			e.dialog.file = strings.TrimSpace(v)
		case f.Key == dlgAfter && err == nil && n >= 0 && n <= int(afterEnd):
			line.after = afterKind(n)
		case r == nil:
		case f.Key == dlgReplyText:
			if r.text == nil {
				r.text = map[string]string{}
			}
			if strings.TrimSpace(v) == "" {
				delete(r.text, lang)
			} else {
				r.text[lang] = v
			}
		case f.Key == dlgNext && err == nil:
			r.next = n
		case f.Key == dlgOnce && err == nil:
			r.once = n == 1
		case f.Key == dlgUnavailable && err == nil:
			r.disabled = n == 1
		}
	}
}

// dialogueRetype follows a changed choice of the window: what follows the
// line, or a wizard row's item.
func (e *MapEditor) dialogueRetype() {
	o := &e.obj
	if len(o.fields) != len(o.buf) {
		return
	}
	number, ok := e.selectedNumber()
	if !ok {
		return
	}
	line := e.lineAt(number)
	was := line.after
	e.dialogPull()
	if line.after != was {
		if line.after == afterReply && len(line.replies) == 0 {
			line.replies, line.open = append(line.replies, dlgReply{text: map[string]string{}}), 0
		} else if line.after == afterReply && line.open < 0 {
			line.open = 0
		}
		o.messageChanged = true
		e.dialogFields()
		o.open = -1
		return
	}
	if o.wiz != nil {
		e.rewizard()
		e.dialogPull()
	}
}

// selectLine shows the line with number.
func (e *MapEditor) selectLine(number int) {
	o := &e.obj
	if !e.captureMessageParts() {
		return
	}
	e.dialogPull()
	if e.pick == pickParamUnit && o.pickField >= 0 {
		e.leave(leavePick)
		o.pickField = -1
	}
	o.sel, o.open, o.focus, o.fieldsTop = number, -1, -1, 0
	e.dialogFields()
	o.focus = e.firstMessageField()
	e.keepLineShown(number)
}

// keepLineShown leaves the list where it is scrolled and moves it the least
// that shows the row of line number.
func (e *MapEditor) keepLineShown(number int) {
	s, _, ok := e.objectSpec()
	if !ok || s.MaxRows <= 0 {
		return
	}
	for row, n := range e.obj.rowSelect {
		if n != number || row >= len(s.Rows) {
			continue
		}
		_, line := popupRowPosition(specWindow(s), row)
		if line < s.Top {
			e.objectTop = line
		} else if line >= s.Top+s.MaxRows {
			e.objectTop = line - s.MaxRows + 1
		}
		return
	}
}

// reselect rebuilds the reply part for the line on show after a card was
// added, struck or restored.
func (e *MapEditor) reselect() {
	e.obj.open = -1
	e.dialogFields()
}

// toggleReply opens reply i of the selected line, or closes it when it is
// open; opening one closes the one that was open.
func (e *MapEditor) toggleReply(i int) {
	o := &e.obj
	number, ok := e.selectedNumber()
	if !ok {
		return
	}
	e.dialogPull()
	line := e.lineAt(number)
	if line.open == i {
		line.open = -1
	} else {
		line.open = i
	}
	o.open, o.focus = -1, -1
	e.dialogFields()
}

// dialogRowPress is a press on a list entry or a reply's header.
func (e *MapEditor) dialogRowPress(n int) bool {
	o := &e.obj
	if number, ok := o.rowSelect[n]; ok {
		e.selectLine(number)
		return true
	}
	if i, ok := o.rowReply[n]; ok {
		e.toggleReply(i)
		return true
	}
	return false
}

// wizardRowActs are the row actions of the trigger wizard's rows.
var wizardRowActs = map[string]bool{actRemoveRow: true, actAddPair: true, actAddGroup: true, actAddAct: true, actDraw: true, actPickArea: true, actPick: true, actPickParam: true}

// dialogRowButton is a press on button k of spec row r when it belongs to
// the reply part; false when it is the cards'.
func (e *MapEditor) dialogRowButton(row, k int) bool {
	o := &e.obj
	acts := o.rowActs[row]
	if k < 0 || k >= len(acts) {
		return false
	}
	act := acts[k]
	f, hasField := o.rowField[row]
	switch {
	case act == actAddReply || act == actRemoveReply:
	case wizardRowActs[act] && o.wiz != nil && (!hasField || f >= o.extra):
	default:
		return false
	}
	o.note, o.open = "", -1
	e.dialogPull()
	switch act {
	case actAddReply:
		number, ok := e.selectedNumber()
		if !ok {
			return true
		}
		line := e.lineAt(number)
		if len(line.replies) >= MaxReplies {
			o.note = e.word("editor.dialogue.err.five")
			return true
		}
		line.replies = append(line.replies, dlgReply{text: map[string]string{}})
		line.open = len(line.replies) - 1
		o.messageChanged = true
		e.dialogFields()
	case actRemoveReply:
		number, ok := e.selectedNumber()
		if !ok {
			return true
		}
		line := e.lineAt(number)
		if line.open >= 0 && line.open < len(line.replies) {
			line.replies = slices.Delete(line.replies, line.open, line.open+1)
			line.open = -1
			o.messageChanged = true
		}
		e.dialogFields()
	default:
		e.wizardRowButton(act, f, hasField)
		e.dialogPull()
	}
	return true
}

// fieldsLines is the lines of the right column of the Dialog window s.
func fieldsLines(s popupSpec) int {
	n := 0
	for _, r := range s.Rows {
		if r.Column == 1 && !r.placed() {
			n++
		}
	}
	return n
}

// fieldsArea is the region of the Dialog window's right column while its
// rows pass the window's height: the wheel over it moves that column alone.
func (e *MapEditor) fieldsArea(s popupSpec, l popupLayout) (scrollArea, bool) {
	most := fieldsLines(s) - s.MaxRows
	if e.messagePage == 0 || len(s.ColumnTops) < 2 || s.MaxRows <= 0 || most <= 0 {
		return scrollArea{}, false
	}
	left := l.RowsViewport.Max.X
	for _, rr := range l.Rows {
		if rr.Index < len(s.Rows) && s.Rows[rr.Index].Column == 1 {
			left = min(left, rr.Rect.Min.X)
		}
	}
	over := image.Rect(left, l.RowsViewport.Min.Y, l.RowsViewport.Max.X, l.RowsViewport.Max.Y)
	return rangeArea(image.Rectangle{}, e.obj.fieldsTop, most, s.MaxRows, []image.Rectangle{over}, &e.objectWheel,
		func(n int) { e.obj.fieldsTop = max(0, min(n, most)) }), true
}

// replyScope is the bindings of the open reply: its text and settings and
// every condition and action row, listed as the trigger window lists them.
func (e *MapEditor) replyScope() *popupScope {
	o := &e.obj
	at := map[string]int{}
	for i, f := range o.fields {
		at[f.Key] = i
	}
	top := &popupScope{Values: map[string]string{}, Words: map[string]string{}, Fields: map[string]int{}, Lists: map[string][]popupScope{}}
	shown := func(key string) string {
		i, ok := at[key]
		if !ok {
			return ""
		}
		if len(o.fields[i].Choices) > 0 {
			return choiceText(o.fields[i], o.buf[i])
		}
		return o.buf[i]
	}
	bind := func(name, key string) {
		if i, ok := at[key]; ok {
			top.Fields[name] = i
			top.Values[name] = shown(key)
		}
	}
	bind("text", dlgReplyText)
	// The field holds UTF-8; the row draws in the font's bytes, converted here once.
	top.Values["text"] = e.editorHostText(top.Values["text"])
	bind("next", dlgNext)
	bind("once", dlgOnce)
	bind("unavailable", dlgUnavailable)
	if o.wiz != nil {
		conds, acts, _, _ := e.wizardLists(top, at)
		if conds == 0 {
			top.flag("nocondition")
		}
		if acts == 0 {
			top.flag("noaction")
		}
	}
	return top
}

var dialogueDesc = embeddedPopupDesc("dialogue")

// dialogRows appends the right column's reply part to s: the line's choice
// of what follows, then, for a reply choice, a header row for each reply
// with the open one's rows under it, and the button that adds a reply.
func (e *MapEditor) dialogRows(s *popupSpec, width int) {
	o := &e.obj
	if !e.replyAllowed() {
		return
	}
	e.dialogPull()
	number, ok := e.selectedNumber()
	if !ok {
		return
	}
	line := e.lineAt(number)
	if f := e.dialogFieldAt(dlgAfter); f >= 0 {
		o.rowField[len(s.Rows)] = f
		s.Rows = append(s.Rows, popupRow{Kind: popupChoice, Column: 1, Label: e.word("editor.dialogue.one.after"), Value: choiceText(o.fields[f], o.buf[f]), Focus: o.focus == f, Help: e.word("editor.dialogue.one.after.help")})
	}
	if line.after != afterReply {
		return
	}
	for i, r := range line.replies {
		o.rowReply[len(s.Rows)] = i
		head := blockRow(fmt.Sprintf("%d. %s", i+1, e.replyHead(r)), line.open == i)
		head.Column = 1
		s.Rows = append(s.Rows, head)
		if line.open == i {
			e.spliceReply(s, width)
		}
	}
	if len(line.replies) < MaxReplies {
		o.rowActs[len(s.Rows)] = []string{actAddReply}
		s.Rows = append(s.Rows, popupRow{Kind: popupText, Column: 1, Buttons: []string{e.word("editor.dialogue.addreply")}})
	}
}

// spliceReply appends the rows the reply description builds for the open
// reply to the right column of s, each bound to its field and actions.
func (e *MapEditor) spliceReply(s *popupSpec, width int) {
	o := &e.obj
	sub, made := dialogueDesc().build(e.replyScope(), e.word)
	body := e.bodyFont()
	rows, made := madeRowLines(sub.Rows, made, lineBox{Measure: advanceOf(body), Width: width - 2*popupPad - 8})
	for i, r := range rows {
		r.Column = 1
		at := len(s.Rows)
		if m := made[i]; m.Field >= 0 {
			o.rowField[at] = m.Field
			r.Focus = m.Field == o.focus
			if r.Kind == popupEdit && r.Focus && body != nil {
				r.Caret = body.Advance(r.Value)
			}
		}
		if len(made[i].Acts) > 0 {
			o.rowActs[at] = made[i].Acts
		}
		s.Rows = append(s.Rows, r)
	}
}
