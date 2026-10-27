package ui

import (
	"fmt"
	"image"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// InspectionMessages edits message pages and their owner as one history entry.
type InspectionMessages interface {
	MessageFields(page uint32, source ...int) []InspectionField
	MessageParts(page uint32) []InspectionMessagePart
	SetMessageParts(page uint32, parts []InspectionMessagePart) (*InspectionDocument, error)
	CreateMessageParts(owner InspectionMessageOwner, parts []InspectionMessagePart) (*InspectionDocument, uint32, error)
	// PointMessageParts makes owner name the existing page and writes its
	// parts, as one history entry.
	PointMessageParts(owner InspectionMessageOwner, page uint32, parts []InspectionMessagePart) (*InspectionDocument, error)
	SetMessage(page uint32, body string, speaker int64) (*InspectionDocument, error)
	CreateMessage(owner InspectionMessageOwner, body string, speaker int64) (*InspectionDocument, uint32, error)
}

type InspectionMessagePart struct {
	Source, Number int
	Body, Tag      string
	Speaker        int64
}

// NewMessagePage is the message popup's page while it writes a new message:
// the page is made only by Apply.
const NewMessagePage uint32 = 1<<32 - 1

// InspectionMessageOwner names the field or wizard action receiving a page.
type InspectionMessageOwner struct {
	Kind    string
	Index   int
	Key     string
	Trigger *WizardTrigger
	Action  int
}

// The dialog editor: the message popup as one card per page, in order. A
// card holds the page's number on a grip, its speaker and its text; the
// grip drags the card, Alt+Up and Alt+Down move it, its cross strikes it
// until Apply or Cancel, and "+" adds an empty card at the end. Cards of one
// page number move and are struck together. Opened from a trigger's Start
// dialog, it also chooses which dialog the action starts.

// beginMessage opens the message popup on a message not yet made.
func (e *MapEditor) beginMessage(owner InspectionMessageOwner, back bool) {
	e.messageNew, e.messageFor = owner, nil
	if back {
		e.openMessage(NewMessagePage)
		return
	}
	e.messagePage, e.objectHidden, e.objectTop = NewMessagePage, false, 0
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}

func (e *MapEditor) messageField(field int) bool {
	v, ok := e.scriptPopup()
	if !ok || field < 0 || field >= len(e.obj.fields) {
		return false
	}
	if v.trigger != nil && e.obj.wiz != nil {
		r, p := e.wizardRowOf(field)
		if r < 0 || p < 0 {
			return false
		}
		row := e.obj.wiz.rows[r]
		return !row.condition && row.item >= 0 && p < len(wizItems(false)[row.item].params) && wizItems(false)[row.item].params[p].kind == wpMessage
	}
	return v.node != nil && e.obj.fields[field].Key == "value0" && e.nodeParameterKind(v, uint32(e.obj.signature[0]), e.obj.types, 0) == ScriptMessageParameter
}

func (e *MapEditor) messageChoiceAlone(field int, value int64) bool {
	if !e.messageField(field) {
		return false
	}
	found := false
	for _, target := range e.doc.Script.Targets[ScriptMessageParameter] {
		found = found || !target.NewMessage && int64(target.Value) == value
	}
	if !found {
		return false
	}
	old := e.obj.buf[field]
	e.obj.buf[field] = fieldInput(e.obj.fields[field])
	dirty := e.objectDirty()
	e.obj.buf[field] = old
	return !dirty
}

// messageText is the whole text of the page a message field names.
func (e *MapEditor) messageText(field int) (string, bool) {
	if !e.messageField(field) || e.doc.Script == nil {
		return "", false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(e.obj.buf[field]), 10, 64)
	if err != nil {
		return "", false
	}
	for _, t := range e.doc.Script.Targets[ScriptMessageParameter] {
		if !t.NewMessage && int64(t.Value) == v {
			return t.Text, t.Text != ""
		}
	}
	return "", false
}

func (e *MapEditor) openMessage(page uint32) {
	e.messageFor = nil
	e.objectBack = append(e.objectBack, e.objectPlace())
	e.messagePage, e.objectHidden, e.objectTop = page, false, 0
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}

func (e *MapEditor) refreshMessagePage() {
	if e.messagePage == 0 {
		return
	}
	if host, ok := e.doc.Edits.(InspectionMessages); ok && len(host.MessageFields(e.messagePage)) != 0 {
		return
	}
	e.messagePage = 0
	e.obj.doc = nil
	if len(e.objectBack) != 0 {
		e.backObject()
	} else {
		e.closeObject()
	}
}

// messageTextField reports that field i is a card's text.
func (e *MapEditor) messageTextField(i int) bool {
	return e.messagePage != 0 && i >= 0 && i < 2*len(e.obj.messageParts) && i%2 == 0
}

// dialogChoiceField is the field of the dialog choice, -1 when the popup
// was not opened from a trigger's action.
func (e *MapEditor) dialogChoiceField() int {
	if f := 2 * len(e.obj.messageParts); e.messagePage != 0 && e.messageFor != nil && f < len(e.obj.fields) {
		return f
	}
	return -1
}

// DialogChoiceField is the field of the dialog editor's dialog choice, -1
// for none.
func (e *MapEditor) DialogChoiceField() int { return e.dialogChoiceField() }

// messageLive is how many page numbers have a card not struck.
func (e *MapEditor) messageLive() int {
	numbers := map[int]bool{}
	for _, part := range e.obj.messageParts {
		if !e.obj.messageGone[part.Number] {
			numbers[part.Number] = true
		}
	}
	return len(numbers)
}

func (e *MapEditor) messageSpec() (popupSpec, []int, bool) {
	e.loadObject()
	o := &e.obj
	if len(o.fields) == 0 {
		return popupSpec{}, nil, false
	}
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-668, e.barHeight()+8), true
	}
	title := fmt.Sprintf(e.word("editor.message.page"), e.messagePage)
	if e.messagePage == NewMessagePage {
		title = e.word("editor.message.newdialog")
	}
	s := popupSpec{Title: title, At: e.objectAt, Width: e.popupWide(popObject, 660), Bounds: e.popupBounds(), LabelWidth: 64, NoteInk: editorWarning,
		Buttons: []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}}
	o.rowField, o.rowOpen = map[int]int{}, map[int]int{}
	if f := e.dialogChoiceField(); f >= 0 {
		o.rowField[len(s.Rows)] = f
		s.Rows = append(s.Rows, popupRow{Kind: popupChoice, Label: o.fields[f].Label, Value: choiceText(o.fields[f], o.buf[f]), Focus: o.focus == f})
	}
	live := e.messageLive()
	numbers := map[int]int{}
	textRow := -1
	for i, part := range o.messageParts {
		o.rowOpen[len(s.Rows)] = i
		if o.messageGone[part.Number] {
			s.Rows = append(s.Rows, popupRow{Kind: popupCard, Struck: true, Label: e.word("editor.message.removed"), Buttons: []string{e.word("editor.message.restore")}})
			continue
		}
		number := part.Number
		if o.messageChanged {
			if _, ok := numbers[number]; !ok {
				numbers[number] = len(numbers) + 1
			}
			number = numbers[number]
		}
		speaker := 2*i + 1
		head := popupRow{Kind: popupCard, Label: strconv.Itoa(number), Value: choiceText(o.fields[speaker], o.buf[speaker]), Focus: o.focus == speaker, Span: 1}
		if live > 1 {
			head.Buttons = []string{"x"}
		}
		o.rowField[len(s.Rows)] = speaker
		s.Rows = append(s.Rows, head)
		lines := strings.Split(o.buf[2*i], "\n")
		for k := range lines {
			lines[k] = e.editorHostText(strings.TrimSuffix(lines[k], "\r"))
		}
		textRow = len(s.Rows)
		o.rowField[textRow] = 2 * i
		s.Rows = append(s.Rows, popupRow{Kind: popupMultiline, Value: strings.Join(lines, "\n"), Focus: o.focus == 2*i, Lines: 3})
	}
	o.rowOpen[len(s.Rows)] = -1
	s.Rows = append(s.Rows, popupRow{Kind: popupText, Buttons: []string{"+"}})
	if note := e.dialogSharedNote(); note != "" {
		s.Notes = append(s.Notes, note)
	}
	if o.note != "" {
		s.Notes = append(s.Notes, e.editorHostText(o.note))
	}
	if e.dragCard > 0 {
		s.Drag = &popupRowDrag{Row: e.dragCard - 1, At: e.pointer}
	}
	e.objectBackButton(&s)
	if font := e.bodyFont(); font != nil {
		for _, caption := range s.Buttons {
			s.ButtonW = max(s.ButtonW, font.Advance(caption)+16)
		}
	}
	if len(o.messageParts) == 1 && textRow >= 0 {
		e.fitMessageText(&s, textRow)
	} else {
		s.MaxRows = max(3, e.objectRowsShown(true)/2)
		e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
		s.Top = e.objectTop
	}
	return s, make([]int, len(s.Rows)), true
}

// fitMessageText grows the text box of row k until it shows the whole
// message, as far as the window allows; a longer text scrolls by the wheel,
// from its first line.
func (e *MapEditor) fitMessageText(s *popupSpec, k int) {
	body := e.bodyFont()
	if body == nil {
		return
	}
	l := layoutPopup(*s, e.font, body)
	if len(l.Rows) <= k {
		return
	}
	width := l.Rows[k].Control.Dx() - 8
	count := 0
	for _, line := range strings.Split(s.Rows[k].Value, "\n") {
		count += len(editorWrap(body, line, width))
	}
	pitch, line := l.Pitch, fontHeight(body, 15)
	spare := (s.Bounds.Dy() - l.Rect.Dy()) / pitch
	rows := min(max(multilineRows(s.Rows[k]), (count*line+8+pitch-1)/pitch), multilineRows(s.Rows[k])+max(0, spare))
	s.Rows[k].Lines = rows
	visible := max(1, (rows*pitch-2-8)/line)
	e.objectTop = max(0, min(e.objectTop, count-visible))
	s.Rows[k].First = e.objectTop
}

// dialogStarters are the triggers, by index, with an action that starts
// page.
func (e *MapEditor) dialogStarters(page uint32) []int {
	var out []int
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	for i, t := range e.doc.Script.Triggers {
		for _, id := range t.Acts {
			if n, ok := e.nodeByID(false, id); id != 0 && ok && n.Values[0] == page && e.nodeParameterKind(scriptView{}, n.Opcode, n.Types, 0) == ScriptMessageParameter {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// dialogSharedNote names the other triggers that start the shown dialog.
func (e *MapEditor) dialogSharedNote() string {
	if e.messagePage == NewMessagePage {
		return ""
	}
	var others []string
	for _, t := range e.dialogStarters(e.messagePage) {
		if e.messageFor == nil || t != e.messageFor.Index {
			others = append(others, strconv.Itoa(t+1))
		}
	}
	if len(others) == 0 || e.messageFor == nil && len(others) < 2 {
		return ""
	}
	return fmt.Sprintf(e.word("editor.message.shared"), strings.Join(others, ", "))
}

// messageDraftDirty reports a card changed, added, moved or struck.
func (e *MapEditor) messageDraftDirty() bool {
	o := &e.obj
	if o.messageChanged {
		return true
	}
	for i := 0; i < 2*len(o.messageParts) && i < len(o.fields); i++ {
		if o.buf[i] != fieldInput(o.fields[i]) {
			return true
		}
	}
	return false
}

// chooseDialog shows the dialog the choice names, or a new one; the action
// is pointed at it by Apply. A draft with changes stays until it is applied
// or cancelled.
func (e *MapEditor) chooseDialog(value int64) {
	o := &e.obj
	o.open = -1
	page := uint32(value)
	if value == NewMessageChoice {
		page = NewMessagePage
	}
	if page == e.messagePage || value <= 0 || value > NewMessageChoice {
		return
	}
	if e.messageDraftDirty() {
		o.note = e.word("editor.message.switchfirst")
		return
	}
	if page == NewMessagePage {
		owner, err := e.dialogOwner()
		if err != nil {
			o.note = err.Error()
			return
		}
		e.messageNew = owner
	}
	e.messagePage, e.objectTop = page, 0
	o.doc = nil
	e.loadObject()
	o.focus = 0
}

// dialogOwner is the action the dialog editor was opened for, with its
// trigger as it stands.
func (e *MapEditor) dialogOwner() (InspectionMessageOwner, error) {
	owner := *e.messageFor
	if owner.Index < 0 || owner.Index >= len(e.doc.Script.Triggers) {
		return owner, fmt.Errorf("%s", e.word("editor.message.unavailable"))
	}
	plan, err := e.keptPlan(owner.Index)
	if err != nil {
		return owner, err
	}
	owner.Trigger = &plan
	return owner, nil
}

// liveMessageParts are the parts Apply writes: every part not struck.
func (e *MapEditor) liveMessageParts() []InspectionMessagePart {
	var out []InspectionMessagePart
	for _, part := range e.obj.messageParts {
		if !e.obj.messageGone[part.Number] {
			out = append(out, part)
		}
	}
	return out
}

func (e *MapEditor) applyMessage() {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok || !e.captureMessageParts() {
		return
	}
	parts := e.liveMessageParts()
	if e.messagePage == NewMessagePage {
		e.createMessage(host, parts)
		return
	}
	if e.messageFor != nil && e.messagePage != e.messageFrom {
		owner, err := e.dialogOwner()
		var next *InspectionDocument
		if err == nil {
			next, err = host.PointMessageParts(owner, e.messagePage, parts)
		}
		if err != nil {
			e.obj.note = err.Error()
			return
		}
		e.messageFrom = e.messagePage
		_ = e.adoptEdit(next, nil)
		e.obj.doc = nil
		e.loadObject()
		return
	}
	if !e.objectDirty() {
		return
	}
	next, err := host.SetMessageParts(e.messagePage, parts)
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	_ = e.adoptEdit(next, nil)
	e.obj.doc = nil
	e.loadObject()
}

// createMessage writes the draft and its owner; Back returns to the owner.
func (e *MapEditor) createMessage(host InspectionMessages, parts []InspectionMessagePart) {
	owner := e.messageNew
	if e.messageFor != nil {
		var err error
		if owner, err = e.dialogOwner(); err != nil {
			e.obj.note = err.Error()
			return
		}
	}
	next, page, err := host.CreateMessageParts(owner, parts)
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	if e.adoptEdit(next, nil) != nil || next == nil {
		return
	}
	back := e.objectBack
	if owner.Kind == "Message" {
		e.selectNewest("Action")
		e.obj.doc = nil
		e.loadObject()
		back = append(e.objectBack, e.objectPlace())
	}
	e.objectBack, e.messagePage, e.objectHidden = back, page, false
	e.messageNew, e.messageFrom = InspectionMessageOwner{}, page
	e.obj.doc = nil
	e.loadObject()
}

func (e *MapEditor) loadMessageFields(parts []InspectionMessagePart) {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok {
		return
	}
	o := &e.obj
	o.messageParts, o.fields, o.buf = slices.Clone(parts), nil, nil
	for _, part := range parts {
		fields := host.MessageFields(e.messagePage, part.Source)
		if len(fields) != 2 {
			continue
		}
		fields[0].TextValue, fields[1].Value = part.Body, part.Speaker
		o.fields = append(o.fields, fields...)
		for _, field := range fields {
			o.buf = append(o.buf, fieldInput(field))
		}
	}
	if e.messageFor != nil && len(o.fields) == 2*len(parts) {
		value := int64(e.messagePage)
		if e.messagePage == NewMessagePage {
			value = NewMessageChoice
		}
		f := InspectionField{Key: "dialog", Label: e.word("editor.message.dialog"), Value: value, Choices: e.parameterChoices(ScriptMessageParameter, value)}
		for k := range f.Choices {
			if f.Choices[k].Value == NewMessageChoice {
				f.Choices[k].Label = e.word("editor.message.newdialog")
			}
		}
		o.fields, o.buf = append(o.fields, f), append(o.buf, fieldInput(f))
	}
	e.input = editorInputState{}
}

func (e *MapEditor) captureMessageParts() bool {
	o := &e.obj
	if len(o.buf) < len(o.messageParts)*2 {
		return false
	}
	for i := range o.messageParts {
		speaker, err := strconv.ParseInt(o.buf[2*i+1], 10, 32)
		if err != nil {
			o.note = e.word("editor.message.badspeaker")
			return false
		}
		o.messageParts[i].Body, o.messageParts[i].Speaker = o.buf[2*i], speaker
	}
	return true
}

// messageRowButton is a press on a card's cross or Restore, or on "+".
func (e *MapEditor) messageRowButton(row, button int) {
	o := &e.obj
	at, ok := o.rowOpen[row]
	if !ok || button != 0 || !e.captureMessageParts() {
		return
	}
	if at < 0 {
		e.addMessageCard()
		return
	}
	number := o.messageParts[at].Number
	// A new map each time: a Cancel checkpoint keeps the one it saw.
	gone := maps.Clone(o.messageGone)
	if gone[number] {
		delete(gone, number)
		o.messageGone = gone
		return
	}
	if e.messageLive() < 2 {
		return
	}
	if gone == nil {
		gone = map[int]bool{}
	}
	gone[number], o.messageGone, o.messageChanged = true, gone, true
	if f := o.focus; f >= 0 && f < 2*len(o.messageParts) && o.messageParts[f/2].Number == number {
		o.focus = -1
	}
	o.open = -1
}

// addMessageCard adds an empty card at the end, the caret in its text.
func (e *MapEditor) addMessageCard() {
	parts := slices.Clone(e.obj.messageParts)
	id := 0
	for _, part := range parts {
		id = min(id, part.Number)
	}
	parts = append(parts, InspectionMessagePart{Source: -1, Number: id - 1, Speaker: -1})
	gone := e.obj.messageGone
	e.loadMessageFields(parts)
	e.obj.messageGone, e.obj.messageChanged, e.obj.focus, e.obj.open = gone, true, 2*(len(parts)-1), -1
	e.objectTop = len(parts) * 2
}

// moveMessageCard moves card at, with every card of its page number, to
// before card to; to is the card count for after the last.
func (e *MapEditor) moveMessageCard(at, to int) bool {
	o := &e.obj
	if at < 0 || at >= len(o.messageParts) || !e.captureMessageParts() {
		return false
	}
	number := o.messageParts[at].Number
	var group, rest []InspectionMessagePart
	before, lead := 0, 0
	for i, part := range o.messageParts {
		if part.Number == number {
			if i < at {
				lead++
			}
			group = append(group, part)
			continue
		}
		rest = append(rest, part)
		if i < to {
			before++
		}
	}
	moved := slices.Insert(rest, before, group...)
	if slices.Equal(moved, o.messageParts) {
		return false
	}
	gone := o.messageGone
	focus := o.focus
	e.loadMessageFields(moved)
	o.messageGone, o.messageChanged, o.open, o.focus = gone, true, -1, -1
	if focus >= 0 && focus < 2*len(moved) && focus/2 == at {
		o.focus = 2*(before+lead) + focus%2
	}
	return true
}

// keyCard moves the focused card one place up or down, past the whole of
// the neighbouring page number.
func (e *MapEditor) keyCard(key string) error {
	o := &e.obj
	f := o.focus
	if e.messagePage == 0 || f < 0 || f >= 2*len(o.messageParts) {
		return nil
	}
	at := f / 2
	parts := o.messageParts
	number := parts[at].Number
	if key == "cardup" {
		for i := at - 1; i >= 0; i-- {
			if parts[i].Number != number {
				e.moveMessageCard(at, slices.IndexFunc(parts, func(p InspectionMessagePart) bool { return p.Number == parts[i].Number }))
				return nil
			}
		}
		return nil
	}
	for i := at + 1; i < len(parts); i++ {
		if parts[i].Number != number {
			next := parts[i].Number
			last := i
			for k := i; k < len(parts); k++ {
				if parts[k].Number == next {
					last = k
				}
			}
			e.moveMessageCard(at, last+1)
			return nil
		}
	}
	return nil
}

// dropCard ends a card drag: the held card lands where the marker stood.
func (e *MapEditor) dropCard() {
	s, _, ok := e.objectSpec()
	row := e.dragCard - 1
	e.dragCard = 0
	if !ok || e.messagePage == 0 {
		return
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if at, ok := e.obj.rowOpen[row]; ok && at >= 0 && l.Drop >= 0 {
		e.moveMessageCard(at, l.Drop)
	}
}

// CardRects are the window rectangles of card c of the dialog editor, when
// it is shown: the whole card and its grip, empty for a struck card.
func (e *MapEditor) CardRects(c int) (card, grip image.Rectangle, ok bool) {
	s, _, shown := e.objectSpec()
	if !shown || e.messagePage == 0 {
		return card, grip, false
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if at, has := e.obj.rowOpen[rr.Index]; has && at == c && s.Rows[rr.Index].Kind == popupCard {
			return rr.Card, rr.Grip, true
		}
	}
	return card, grip, false
}

// newMessage opens the message popup for a new message the field will
// name; nothing is made until its Apply.
func (e *MapEditor) newMessage(field int) {
	if _, ok := e.doc.Edits.(InspectionMessages); !ok || field < 0 || field >= len(e.obj.fields) {
		return
	}
	if e.obj.buf[field] == strconv.FormatInt(NewMessageChoice, 10) {
		e.obj.buf[field] = fieldInput(e.obj.fields[field])
	}
	if !e.objectMayLeave() {
		return
	}
	kind, index := e.selectedRecord()
	owner := InspectionMessageOwner{Kind: kind, Index: index, Key: e.obj.fields[field].Key}
	if v, ok := e.scriptPopup(); ok && v.trigger != nil && e.obj.wiz != nil {
		plan, err := e.wizardPlan(v)
		if err != nil {
			e.obj.note = err.Error()
			return
		}
		row, _ := e.wizardRowOf(field)
		for r := 0; r < row; r++ {
			if !e.obj.wiz.rows[r].condition {
				owner.Action++
			}
		}
		owner.Trigger = &plan
	}
	e.beginMessage(owner, true)
}

// configureDialog opens the dialog editor for the Start dialog action whose
// message field is field: on the dialog it starts, or on a new one when it
// starts none. A trigger with changes not yet applied is applied first.
func (e *MapEditor) configureDialog(field int) {
	v, ok := e.scriptPopup()
	if !ok || v.trigger == nil || e.obj.wiz == nil || field < 0 || field >= len(e.obj.fields) {
		return
	}
	key := e.obj.fields[field].Key
	if e.objectDirty() {
		e.applyWizard(v)
		if e.obj.note != "" {
			return
		}
		e.loadObject()
		if field = slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key }); field < 0 {
			return
		}
	}
	row, _ := e.wizardRowOf(field)
	action := 0
	for r := 0; r < row; r++ {
		if !e.obj.wiz.rows[r].condition {
			action++
		}
	}
	owner := &InspectionMessageOwner{Kind: "Trigger", Index: v.index, Action: action}
	if target, ok := e.parameterTarget(field); ok {
		e.openMessage(target.Value)
		e.messageFor, e.messageFrom = owner, target.Value
	} else {
		e.newMessage(field)
		if e.messagePage != NewMessagePage {
			return
		}
		e.messageFor, e.messageFrom = owner, NewMessagePage
	}
	e.obj.doc = nil
	e.loadObject()
	e.obj.focus = 0
}
