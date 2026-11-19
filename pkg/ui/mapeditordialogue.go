package ui

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/locale"
	"againrom/pkg/words"
)

// The dialogue window: one interactive dialogue as nodes, each with its
// texts in two languages and up to four replies. A reply names the node it
// leads to, whether it is once and how it shows when it is not available,
// and holds one condition and one outcome, edited as the trigger wizard's
// rows: the window keeps the selected reply's rows as the wizard's.

// dialogueObjectRecord is the object popup's record while the dialogue
// window shows.
const dialogueObjectRecord = -4

// dialogueReplies is the most replies a node holds.
const dialogueReplies = 4

// dialogueLanguages are the languages the window edits a text in, as the
// dialogue file's codes.
var dialogueLanguages = func() (codes []string) {
	for _, l := range locale.All() {
		codes = append(codes, l.Code)
	}
	return codes
}()

// The dialogue window's fields.
const (
	dlgFile        = "d.file"
	dlgNode        = "d.node"
	dlgName        = "d.name"
	dlgText        = "d.text."
	dlgReplyField  = "d.reply"
	dlgReplyText   = "d.r."
	dlgNext        = "d.next"
	dlgOnce        = "d.once"
	dlgUnavailable = "d.unavailable"
)

// The row actions of the dialogue window.
const (
	actAddNode     = "addnode"
	actRemoveNode  = "removenode"
	actNodeUp      = "nodeup"
	actNodeDown    = "nodedown"
	actAddReply    = "addreply"
	actRemoveReply = "removereply"
)

type dlgReply struct {
	text     map[string]string
	next     int
	once     bool
	mark     int
	markSet  bool // mark came from the file
	disabled bool
	rows     []wizRow
}

type dlgNodeDraft struct {
	id      int
	name    string
	text    map[string]string
	replies []dlgReply
}

// dialogueSession is the dialogue being edited: its nodes as drafts and the
// selected node and reply (-1 for no reply).
type dialogueSession struct {
	name    string
	file    string // the name a new dialogue is typed under
	fresh   bool   // not written yet
	fromMod bool   // opened from the mod window, which it returns to
	nodes   []dlgNodeDraft
	node    int
	reply   int
	saved   string
}

// key is the draft as one text, equal for equal drafts.
func (d *dialogueSession) key() string {
	return fmt.Sprintf("%s|%v", d.file, d.nodes)
}

func (d *dialogueSession) current() *dlgNodeDraft { return &d.nodes[d.node] }

func (d *dialogueSession) shown() *dlgReply {
	n := d.current()
	if d.reply < 0 || d.reply >= len(n.replies) {
		return nil
	}
	return &n.replies[d.reply]
}

func (d *dialogueSession) title() string {
	if d.name != "" {
		return d.name
	}
	return d.file
}

func (e *MapEditor) dialogueHost() (InspectionDialogues, bool) {
	if e.doc == nil {
		return nil, false
	}
	h, ok := e.doc.Edits.(InspectionDialogues)
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

// DialogueOpen reports that the dialogue window shows.
func (e *MapEditor) DialogueOpen() bool { return e.dlg != nil }

// DialogueField is the dialogue window's field named key, -1 for none, for a
// headless driver.
func (e *MapEditor) DialogueField(key string) int {
	if e.dlg == nil {
		return -1
	}
	e.objectSpec()
	for i, f := range e.obj.fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

// DialogueRowActions are the actions of the buttons shown on the row of the
// dialogue window's field key, in button order.
func (e *MapEditor) DialogueRowActions(key string) []string {
	i := e.DialogueField(key)
	if i < 0 {
		return nil
	}
	return slices.Clone(e.obj.rowActs[e.fieldRow(i)])
}

// DialogueText is the text of the dialogue window's field key as typed.
func (e *MapEditor) DialogueText(key string) string {
	if i := e.DialogueField(key); i >= 0 {
		return e.obj.buf[i]
	}
	return ""
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

// dialogueNodes are the drafts of data read from the host.
func (e *MapEditor) dialogueNodes(data DialogueData) []dlgNodeDraft {
	var out []dlgNodeDraft
	for _, n := range data.Nodes {
		node := dlgNodeDraft{id: n.ID, name: n.Name, text: maps.Clone(n.Text)}
		for _, r := range n.Replies {
			node.replies = append(node.replies, dlgReply{text: maps.Clone(r.Text), next: r.Next, once: r.Once, mark: r.Mark, markSet: r.Once, disabled: r.Disabled, rows: e.replyRows(r)})
		}
		out = append(out, node)
	}
	return out
}

// openDialogue shows dialogue name; a new one starts with one empty node.
func (e *MapEditor) openDialogue(name string, fresh, fromMod bool) error {
	h, ok := e.dialogueHost()
	if !ok {
		return errNoEditHost
	}
	d := &dialogueSession{name: name, fresh: fresh, fromMod: fromMod, reply: -1}
	if fresh {
		d.name = ""
		d.file = name
		d.nodes = []dlgNodeDraft{{id: 1, name: fmt.Sprintf(e.word("editor.dialogue.nodeword"), 1), text: map[string]string{}}}
	} else {
		data, err := h.ReadDialogue(name)
		if err != nil {
			return err
		}
		d.nodes = e.dialogueNodes(data)
		if len(d.nodes) == 0 {
			return errNoEditHost
		}
		if len(d.nodes[0].replies) > 0 {
			d.reply = 0
		}
	}
	e.dlg = d
	e.modWindow = false
	e.objectHidden, e.objectTop = false, 0
	e.obj.doc = nil
	e.loadObject()
	if !fresh {
		d.saved = e.dialogueKey()
	}
	return nil
}

// openDialogueFrom opens the dialogue the wizard's field f names, after
// applying the wizard's own edits.
func (e *MapEditor) openDialogueFrom(f int) {
	o := &e.obj
	name := strings.TrimSpace(o.buf[f])
	if !validDialogueName(name) {
		o.note = e.word("editor.dialogue.err.name")
		return
	}
	v, ok := e.scriptPopup()
	if !ok {
		return
	}
	if e.objectDirty() {
		e.applyWizard(v)
		if e.prompt.open() || e.obj.note != "" {
			return
		}
		e.loadObject()
		if e.objectDirty() {
			return
		}
	}
	back := append(slices.Clone(e.objectBack), e.objectPlace())
	if err := e.openDialogue(name, !e.dialogueExists(name), false); err != nil {
		e.obj.note = e.editorHostText(err.Error())
		return
	}
	e.objectBack = back
}

// OpenDialogue shows dialogue name, or a new one of that name, from the mod
// window.
func (e *MapEditor) OpenDialogue(name string) error {
	if !validDialogueName(name) {
		return fmt.Errorf("%s", e.word("editor.dialogue.err.name"))
	}
	return e.openDialogue(name, !e.dialogueExists(name), true)
}

// NewDialogue shows a new dialogue the window asks a name for.
func (e *MapEditor) NewDialogue() error {
	if err := e.openDialogue("", true, true); err != nil {
		return err
	}
	return nil
}

func (e *MapEditor) loadDialogueObject() { e.dialogueFields() }

// dialogueLabel is node n as the lists name it.
func (e *MapEditor) dialogueLabel(n dlgNodeDraft) string {
	if n.name != "" {
		return e.editorHostText(n.name)
	}
	return fmt.Sprintf(e.word("editor.dialogue.nodeword"), n.id)
}

// hostUTF8 is text typed or kept in the font's encoding as UTF-8.
func (e *MapEditor) hostUTF8(s string) string {
	if e.hostDecode != nil {
		return e.hostDecode(s)
	}
	return s
}

// dialogueDrawable reports that the editor's face draws language lang's
// letters: a text it cannot draw is shown and kept as it is, never typed
// over.
func (e *MapEditor) dialogueDrawable(lang string) bool {
	l, ok := locale.ByCode(lang)
	if !ok || !locale.FontRemaps(l.Selector) {
		return true
	}
	probe := string(rune(0x042F)) // a capital letter of the Cyrillic alphabet
	return e.hostUTF8(e.editorHostText(probe)) == probe
}

// dialogueText is the field of one language's text of m.
func (e *MapEditor) dialogueText(m map[string]string, lang string) InspectionField {
	return InspectionField{Text: true, TextValue: e.editorHostText(m[lang]), ReadOnly: !e.dialogueDrawable(lang)}
}

// dialogueFields builds the window's fields from the draft: the node and
// its texts, the reply and its settings, then the reply's condition and
// outcome rows as the wizard's fields.
func (e *MapEditor) dialogueFields() {
	o, d := &e.obj, e.dlg
	o.fields, o.buf, o.wiz = nil, nil, nil
	add := func(f InspectionField) {
		o.fields = append(o.fields, f)
		o.buf = append(o.buf, fieldInput(f))
	}
	if d.fresh && d.fromMod {
		add(InspectionField{Key: dlgFile, Label: e.word("editor.dialogue.file"), Text: true, TextValue: d.file})
	}
	n := d.current()
	var nodes []InspectionChoice
	for i, x := range d.nodes {
		nodes = append(nodes, InspectionChoice{Value: int64(i), Label: e.dialogueLabel(x)})
	}
	add(InspectionField{Key: dlgNode, Label: e.word("editor.dialogue.node"), Value: int64(d.node), Choices: nodes})
	add(InspectionField{Key: dlgName, Label: e.word("editor.dialogue.nodename"), Text: true, TextValue: e.editorHostText(n.name), ReadOnly: e.hostUTF8(e.editorHostText(n.name)) != n.name})
	for _, lang := range dialogueLanguages {
		f := e.dialogueText(n.text, lang)
		f.Key = dlgText + lang
		add(f)
	}
	var replies []InspectionChoice
	for i, r := range n.replies {
		replies = append(replies, InspectionChoice{Value: int64(i), Label: fmt.Sprintf("%d. %s", i+1, e.replyHead(r))})
	}
	if len(replies) == 0 {
		replies = []InspectionChoice{{Value: -1, Label: e.word("editor.dialogue.noreply")}}
	}
	add(InspectionField{Key: dlgReplyField, Label: e.word("editor.dialogue.reply"), Value: int64(d.reply), Choices: replies})
	r := d.shown()
	if r == nil {
		return
	}
	for _, lang := range dialogueLanguages {
		f := e.dialogueText(r.text, lang)
		f.Key = dlgReplyText + lang
		add(f)
	}
	next := []InspectionChoice{{Value: 0, Label: e.word("editor.dialogue.end")}}
	for _, x := range d.nodes {
		next = append(next, InspectionChoice{Value: int64(x.id), Label: e.dialogueLabel(x)})
	}
	if !slices.ContainsFunc(next, func(c InspectionChoice) bool { return c.Value == int64(r.next) }) {
		next = append(next, InspectionChoice{Value: int64(r.next), Label: fmt.Sprintf(e.word("editor.dialogue.nodeword"), r.next)})
	}
	add(InspectionField{Key: dlgNext, Label: e.word("editor.dialogue.next"), Value: int64(r.next), Choices: next})
	add(InspectionField{Key: dlgOnce, Label: e.word("editor.dialogue.once"), Value: boolInt(r.once), Choices: []InspectionChoice{{Value: 1, Label: e.word("editor.dialogue.once.yes")}, {Value: 0, Label: e.word("editor.dialogue.once.no")}}})
	add(InspectionField{Key: dlgUnavailable, Label: e.word("editor.dialogue.unavailable"), Value: boolInt(r.disabled), Choices: []InspectionChoice{{Value: 0, Label: e.word("editor.dialogue.hidden")}, {Value: 1, Label: e.word("editor.dialogue.disabled")}}})
	o.wiz = &wizardState{rows: r.rows, draw: -1, drawRadius: -1}
	e.wizardRowFields()
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// replyHead is the words a reply's list entry shows: its text in the editor's
// language, else in the other.
func (e *MapEditor) replyHead(r dlgReply) string {
	for _, lang := range append([]string{e.dialogueLanguage()}, dialogueLanguages...) {
		if s := strings.TrimSpace(r.text[lang]); s != "" {
			return e.editorHostText(s)
		}
	}
	return e.word("editor.dialogue.noreply.text")
}

// dialogueLanguage is the dialogue code of the editor's language.
func (e *MapEditor) dialogueLanguage() string {
	b := e.book
	if !e.hasBook {
		b = words.For("")
	}
	return b.Lang()
}

// dialoguePull copies the typed texts and the choices into the draft.
func (e *MapEditor) dialoguePull() {
	o, d := &e.obj, e.dlg
	if d == nil || len(o.fields) != len(o.buf) {
		return
	}
	if o.wiz != nil {
		e.pullWizard()
		if r := d.shown(); r != nil {
			r.rows = o.wiz.rows
		}
	}
	n, r := d.current(), d.shown()
	set := func(m *map[string]string, lang, v string) {
		if *m == nil {
			*m = map[string]string{}
		}
		if strings.TrimSpace(v) == "" {
			delete(*m, lang)
			return
		}
		(*m)[lang] = v
	}
	for i, f := range o.fields {
		v := o.buf[i]
		num, _ := strconv.Atoi(strings.TrimSpace(v))
		if f.ReadOnly {
			continue
		}
		switch {
		case f.Key == dlgFile:
			d.file = strings.TrimSpace(v)
		case f.Key == dlgName:
			n.name = strings.TrimSpace(e.hostUTF8(v))
		case strings.HasPrefix(f.Key, dlgText):
			set(&n.text, strings.TrimPrefix(f.Key, dlgText), e.hostUTF8(v))
		case r == nil:
		case strings.HasPrefix(f.Key, dlgReplyText):
			set(&r.text, strings.TrimPrefix(f.Key, dlgReplyText), e.hostUTF8(v))
		case f.Key == dlgNext:
			r.next = num
		case f.Key == dlgOnce:
			r.once = num == 1
		case f.Key == dlgUnavailable:
			r.disabled = num == 1
		}
	}
}

// dialogueKey is the draft as it stands, typed text included.
func (e *MapEditor) dialogueKey() string {
	e.dialoguePull()
	return e.dlg.key()
}

func (e *MapEditor) dialogueDirty() bool {
	return e.dlg != nil && e.dialogueKey() != e.dlg.saved
}

// dialogueRetype follows a changed choice: another node or reply, or a
// wizard row's item.
func (e *MapEditor) dialogueRetype() {
	o, d := &e.obj, e.dlg
	if len(o.fields) != len(o.buf) {
		return
	}
	pick := func(key string) (int, bool) {
		for i, f := range o.fields {
			if f.Key == key {
				n, err := strconv.Atoi(strings.TrimSpace(o.buf[i]))
				return n, err == nil
			}
		}
		return 0, false
	}
	node, okNode := pick(dlgNode)
	reply, okReply := pick(dlgReplyField)
	switch {
	case okNode && node != d.node && node >= 0 && node < len(d.nodes):
		e.dialoguePull()
		d.node, d.reply = node, -1
		if len(d.current().replies) > 0 {
			d.reply = 0
		}
		e.dialogueFields()
		o.open = -1
	case okReply && reply != d.reply && reply < len(d.current().replies):
		e.dialoguePull()
		d.reply = reply
		e.dialogueFields()
		o.open = -1
	default:
		e.rewizard()
		e.dialoguePull()
	}
}

// dialogueData is the draft as the host writes it: the rows of every reply
// as nodes, each once reply with its mark.
func (e *MapEditor) dialogueData() (DialogueData, error) {
	h, _ := e.dialogueHost()
	e.dialoguePull()
	var data DialogueData
	for _, n := range e.dlg.nodes {
		node := DialogueNodeData{ID: n.id, Name: n.name, Text: maps.Clone(n.text)}
		for k, r := range n.replies {
			reply := DialogueReplyData{Text: maps.Clone(r.text), Next: r.next, Once: r.once, Mark: r.mark, Disabled: r.disabled}
			plan, err := e.rowsPlan(WizardTrigger{}, r.rows)
			if err != nil {
				return data, fmt.Errorf(e.word("editor.dialogue.err.reply"), e.dialogueLabel(n), k+1, err.Error())
			}
			reply.Pairs, reply.Actions = plan.Pairs, plan.Actions
			node.Replies = append(node.Replies, reply)
		}
		data.Nodes = append(data.Nodes, node)
	}
	for i := range data.Nodes {
		for k := range data.Nodes[i].Replies {
			r := &data.Nodes[i].Replies[k]
			if r.Once && r.Mark == 0 && !e.dlg.nodes[i].replies[k].markSet {
				mark := h.FreeMark(data)
				if mark < 0 {
					return data, fmt.Errorf("%s", e.word("editor.dialogue.err.nomark"))
				}
				r.Mark = mark
			}
		}
	}
	return data, nil
}

// dialogueIssueText is a failed check in words.
func (e *MapEditor) dialogueIssueText(i DialogueIssue) string {
	return IssueText(e.word, i)
}

// IssueText is a failed dialogue check in the words word gives.
func IssueText(word func(string) string, i DialogueIssue) string {
	if i.Rule == "editor.dialogue.check.notext" && i.Reply == 0 {
		return fmt.Sprintf(word("editor.dialogue.check.notextnode"), i.Node, i.Reply, i.Other)
	}
	return fmt.Sprintf(word(i.Rule), i.Node, i.Reply, i.Other)
}

// DialogueNote is what the dialogue window says about the draft: every
// failed check and the first refusal of a row, as the Apply button reports.
func (e *MapEditor) DialogueChecks() []string {
	h, ok := e.dialogueHost()
	if !ok || e.dlg == nil {
		return nil
	}
	data, err := e.dialogueData()
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	for _, i := range h.DialogueIssues(data) {
		out = append(out, e.dialogueIssueText(i))
	}
	return out
}

// applyDialogue writes the dialogue as one edit, after its checks.
func (e *MapEditor) applyDialogue() {
	o, d := &e.obj, e.dlg
	h, ok := e.dialogueHost()
	if !ok {
		return
	}
	if checks := e.DialogueChecks(); len(checks) > 0 {
		o.note = strings.Join(checks, "\n")
		return
	}
	name := d.name
	if d.fresh {
		name = d.file
	}
	if !validDialogueName(name) {
		o.note = e.word("editor.dialogue.err.name")
		return
	}
	if d.fresh && e.dialogueExists(name) {
		o.note = fmt.Sprintf(e.word("editor.dialogue.err.exists"), name)
		return
	}
	data, _ := e.dialogueData()
	next, err := h.WriteDialogue(name, data)
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	_, _ = e.adopted(next, nil)
	back, err := h.ReadDialogue(name)
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	node := d.current().id
	d.name, d.fresh = name, false
	d.nodes = e.dialogueNodes(back)
	d.node, d.reply = 0, -1
	if at := slices.IndexFunc(d.nodes, func(n dlgNodeDraft) bool { return n.id == node }); at >= 0 {
		d.node = at
	}
	if len(d.current().replies) > 0 {
		d.reply = 0
	}
	o.note = ""
	e.obj.doc = nil
	e.loadObject()
	d.saved = e.dialogueKey()
}

// dialogueDiscard returns the draft to the dialogue as written.
func (e *MapEditor) dialogueDiscard() {
	d := e.dlg
	if d == nil || d.fresh {
		return
	}
	h, ok := e.dialogueHost()
	if !ok {
		return
	}
	data, err := h.ReadDialogue(d.name)
	if err != nil {
		return
	}
	d.nodes = e.dialogueNodes(data)
	d.node, d.reply = 0, -1
	if len(d.current().replies) > 0 {
		d.reply = 0
	}
}

// closeDialogue leaves the dialogue window, back to the mod window when it
// came from there.
func (e *MapEditor) closeDialogue() {
	d := e.dlg
	e.dlg = nil
	e.obj.doc = nil
	if d != nil && d.fromMod {
		e.modWindow = true
	}
}

// dialogueWindowButton is a press on button n of the window's button row:
// Apply, Cancel or Delete.
func (e *MapEditor) dialogueWindowButton(n int) {
	d := e.dlg
	switch n {
	case 0:
		e.applyDialogue()
	case 1:
		e.cancelDialogue()
	case 2:
		if d.fresh {
			return
		}
		h, ok := e.dialogueHost()
		if !ok {
			return
		}
		next, err := h.RemoveDialogue(d.name)
		if err != nil {
			e.obj.note = e.editorHostText(err.Error())
			return
		}
		_, _ = e.adopted(next, nil)
		e.dlg.saved = e.dlg.key()
		e.cancelDialogue()
	}
}

// cancelDialogue drops the draft and closes the window.
func (e *MapEditor) cancelDialogue() {
	e.dialogueDiscard()
	if e.dlg != nil {
		e.dlg.saved = e.dlg.key()
	}
	if len(e.objectBack) > 0 && !e.dlg.fromMod {
		e.backObject()
		return
	}
	e.closeDialogue()
}

// dialogueRowButton is a press on button k of spec row r.
func (e *MapEditor) dialogueRowButton(r, k int) {
	o, d := &e.obj, e.dlg
	acts := o.rowActs[r]
	if k < 0 || k >= len(acts) {
		return
	}
	f, hasField := o.rowField[r]
	o.note, o.open = "", -1
	e.dialoguePull()
	switch acts[k] {
	case actAddNode:
		id := 1
		for _, n := range d.nodes {
			id = max(id, n.id+1)
		}
		d.nodes = append(d.nodes, dlgNodeDraft{id: id, name: fmt.Sprintf(e.word("editor.dialogue.nodeword"), id), text: map[string]string{}})
		d.node, d.reply = len(d.nodes)-1, -1
		e.dialogueFields()
	case actRemoveNode:
		if len(d.nodes) < 2 {
			return
		}
		d.nodes = slices.Delete(d.nodes, d.node, d.node+1)
		d.node = min(d.node, len(d.nodes)-1)
		d.reply = -1
		if len(d.current().replies) > 0 {
			d.reply = 0
		}
		e.dialogueFields()
	case actNodeUp, actNodeDown:
		e.dialogueMoveNode(map[string]int{actNodeUp: -1, actNodeDown: 1}[acts[k]])
	case actAddReply:
		n := d.current()
		if len(n.replies) >= dialogueReplies {
			o.note = fmt.Sprintf(e.word("editor.dialogue.check.five"), e.dialogueLabel(*n))
			return
		}
		n.replies = append(n.replies, dlgReply{text: map[string]string{}})
		d.reply = len(n.replies) - 1
		e.dialogueFields()
	case actRemoveReply:
		n := d.current()
		if d.reply < 0 || d.reply >= len(n.replies) {
			return
		}
		n.replies = slices.Delete(n.replies, d.reply, d.reply+1)
		d.reply = min(d.reply, len(n.replies)-1)
		e.dialogueFields()
	default:
		e.wizardRowButton(acts[k], f, hasField)
	}
}

// dialogueMoveNode moves the selected node one place; the first node starts
// the dialogue.
func (e *MapEditor) dialogueMoveNode(by int) {
	d := e.dlg
	to := d.node + by
	if to < 0 || to >= len(d.nodes) {
		return
	}
	e.dialoguePull()
	d.nodes[d.node], d.nodes[to] = d.nodes[to], d.nodes[d.node]
	d.node = to
	e.dialogueFields()
}

// dialogueScope is the dialogue window's bindings.
func (e *MapEditor) dialogueScope() *popupScope {
	o, d := &e.obj, e.dlg
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
	top.Values["name"] = d.title()
	bind("file", dlgFile)
	bind("node", dlgNode)
	bind("nodename", dlgName)
	bind("reply", dlgReplyField)
	bind("next", dlgNext)
	bind("once", dlgOnce)
	bind("unavailable", dlgUnavailable)
	langWord := func(lang string) string { return e.word("editor.dialogue.lang." + lang) }
	for _, lang := range dialogueLanguages {
		if i, ok := at[dlgText+lang]; ok {
			top.Lists["texts"] = append(top.Lists["texts"], popupScope{Values: map[string]string{"label": fmt.Sprintf(e.word("editor.dialogue.text"), langWord(lang)), "value": o.buf[i]}, Fields: map[string]int{"text": i}})
		}
		if i, ok := at[dlgReplyText+lang]; ok {
			top.Lists["replytexts"] = append(top.Lists["replytexts"], popupScope{Values: map[string]string{"label": fmt.Sprintf(e.word("editor.dialogue.replytext"), langWord(lang)), "value": o.buf[i]}, Fields: map[string]int{"text": i}})
		}
	}
	for _, f := range []struct {
		name string
		on   bool
	}{
		{"naming", d.fresh && d.fromMod},
		{"fresh", d.fresh},
		{"hasreply", d.shown() != nil},
		{"canremovenode", len(d.nodes) > 1},
		{"canup", d.node > 0},
		{"candown", d.node < len(d.nodes)-1},
		{"clean", !e.dialogueDirty()},
	} {
		if f.on {
			top.flag(f.name)
		}
	}
	if o.wiz != nil {
		heroes, drops := false, 0
		cond, act := -1, -1
		for r, wr := range o.wiz.rows {
			switch {
			case wr.condition && cond < 0:
				cond = r
			case !wr.condition && act < 0:
				act = r
			}
		}
		if cond >= 0 {
			item := e.wizardItemScope(cond, at, &heroes, &drops)
			top.Lists["conditions"] = append(top.Lists["conditions"], item)
		} else {
			top.flag("nocondition")
		}
		if act >= 0 {
			item := e.wizardItemScope(act, at, &heroes, &drops)
			top.Lists["actions"] = append(top.Lists["actions"], item)
		} else {
			top.flag("noaction")
		}
	}
	return top
}

// dialogueSpec is the dialogue window as data.
func (e *MapEditor) dialogueSpec() (popupSpec, []int, bool) {
	e.loadObject()
	s, made := dialogueDesc().build(e.dialogueScope(), e.word)
	s.NoteInk = editorWarning
	s, links := e.madeSpec(s, made)
	e.objectBackButton(&s)
	return s, links, true
}

var dialogueDesc = embeddedPopupDesc("dialogue")
