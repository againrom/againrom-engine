package ui

import (
	"fmt"
	"image"
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

// beginMessage opens the message popup on a message not yet made.
func (e *MapEditor) beginMessage(owner InspectionMessageOwner, back bool) {
	e.messageNew = owner
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
		title = e.word("editor.message.new")
	}
	s := popupSpec{Title: title, At: e.objectAt, Width: e.popupWide(popObject, 660), Bounds: e.popupBounds(), LabelWidth: 135, NoteInk: editorWarning,
		Buttons: []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}}
	o.rowField, o.rowOpen = map[int]int{}, map[int]int{}
	numbers := map[int]int{}
	for i, part := range o.messageParts {
		number := part.Number
		if o.messageChanged {
			if _, ok := numbers[number]; !ok {
				numbers[number] = len(numbers) + 1
			}
			number = numbers[number]
		}
		toolbar := func() {
			o.rowOpen[len(s.Rows)] = i
			row := popupRow{Kind: popupText, Label: fmt.Sprintf(e.word("editor.message.part"), number)}
			row.Buttons = []string{e.word("editor.message.add"), e.word("editor.message.end"), e.word("editor.message.remove"), e.word("editor.message.up"), e.word("editor.message.down")}
			s.Rows = append(s.Rows, row)
		}
		if len(o.messageParts) > 1 {
			toolbar()
		}
		for side := range 2 {
			field := 2*i + side
			o.rowField[len(s.Rows)] = field
			lines := strings.Split(o.buf[field], "\n")
			for i := range lines {
				lines[i] = e.editorHostText(strings.TrimSuffix(lines[i], "\r"))
			}
			row := popupRow{Kind: popupMultiline, Label: o.fields[field].Label, Value: strings.Join(lines, "\n"), Focus: o.focus == field, Lines: 3}
			if side == 1 {
				row.Kind, row.Value = popupChoice, choiceText(o.fields[field], o.buf[field])
			}
			s.Rows = append(s.Rows, row)
		}
		if len(o.messageParts) == 1 {
			toolbar()
		}
	}
	if o.note != "" {
		s.Notes = []string{e.editorHostText(o.note)}
	}
	e.objectBackButton(&s)
	if font := e.bodyFont(); font != nil {
		for _, caption := range s.Buttons {
			s.ButtonW = max(s.ButtonW, font.Advance(caption)+16)
		}
	}
	if len(o.messageParts) == 1 {
		e.fitMessageText(&s)
	} else {
		s.MaxRows = max(3, e.objectRowsShown(true)/2)
		e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
		s.Top = e.objectTop
	}
	return s, make([]int, len(s.Rows)), true
}

// fitMessageText grows the text box until it shows the whole message, as far
// as the window allows; a longer text scrolls by the wheel, from its first
// line.
func (e *MapEditor) fitMessageText(s *popupSpec) {
	body := e.bodyFont()
	if body == nil {
		return
	}
	l := layoutPopup(*s, e.font, body)
	if len(l.Rows) == 0 {
		return
	}
	width := l.Rows[0].Control.Dx() - 8
	count := 0
	for _, line := range strings.Split(s.Rows[0].Value, "\n") {
		count += len(editorWrap(body, line, width))
	}
	pitch, line := l.Pitch, fontHeight(body, 15)
	spare := (s.Bounds.Dy() - l.Rect.Dy()) / pitch
	rows := min(max(multilineRows(s.Rows[0]), (count*line+8+pitch-1)/pitch), multilineRows(s.Rows[0])+max(0, spare))
	s.Rows[0].Lines = rows
	visible := max(1, (rows*pitch-2-8)/line)
	e.objectTop = max(0, min(e.objectTop, count-visible))
	s.Rows[0].First = e.objectTop
}

func (e *MapEditor) applyMessage() {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok || !e.captureMessageParts() {
		return
	}
	if e.messagePage == NewMessagePage {
		e.createMessage(host)
		return
	}
	next, err := host.SetMessageParts(e.messagePage, e.obj.messageParts)
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	_ = e.adoptEdit(next, nil)
	e.obj.doc = nil
	e.loadObject()
}

// createMessage writes the draft and its owner; Back returns to the owner.
func (e *MapEditor) createMessage(host InspectionMessages) {
	owner := e.messageNew
	next, page, err := host.CreateMessageParts(owner, e.obj.messageParts)
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
	e.messageNew = InspectionMessageOwner{}
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
	e.input = editorInputState{}
}

func (e *MapEditor) captureMessageParts() bool {
	o := &e.obj
	if len(o.buf) != len(o.messageParts)*2 {
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

func (e *MapEditor) messageRowButton(row, button int) {
	at, ok := e.obj.rowOpen[row]
	if !ok || !e.captureMessageParts() {
		return
	}
	parts := slices.Clone(e.obj.messageParts)
	number := parts[at].Number
	var group, rest []InspectionMessagePart
	start := 0
	for _, part := range parts {
		if part.Number == number {
			if len(group) == 0 {
				start = len(rest)
			}
			group = append(group, part)
		} else {
			rest = append(rest, part)
		}
	}
	switch button {
	case 0, 1:
		id := 0
		for _, part := range parts {
			id = min(id, part.Number)
		}
		at = len(parts)
		if button == 0 {
			at = start + len(group)
			parts = slices.Insert(rest, start, group...)
		}
		parts = slices.Insert(parts, at, InspectionMessagePart{Source: -1, Number: id - 1, Speaker: -1})
	case 2:
		if len(rest) == 0 {
			return
		}
		parts, at = rest, min(start, len(rest)-1)
	case 3, 4:
		if button == 3 {
			if start == 0 {
				return
			}
			start--
			for start > 0 && rest[start-1].Number == rest[start].Number {
				start--
			}
		} else {
			if start == len(rest) {
				return
			}
			next := rest[start].Number
			for start < len(rest) && rest[start].Number == next {
				start++
			}
		}
		parts, at = slices.Insert(rest, start, group...), start
	default:
		return
	}
	e.loadMessageFields(parts)
	e.obj.messageChanged, e.obj.focus, e.obj.open = true, 2*at, -1
	e.objectTop = max(0, 3*at-1)
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
