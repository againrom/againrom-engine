package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// InspectionMessages edits a map's message pages. MessageFields of
// NewMessagePage are the empty fields of a page not yet made; CreateMessage
// makes that page with its text and names it in field key of the record, or
// in a new action for kind "Message", as one edit, and returns its number.
type InspectionMessages interface {
	MessageFields(page uint32) []InspectionField
	SetMessage(page uint32, body string, speaker int64) (*InspectionDocument, error)
	CreateMessage(kind string, index int, key, body string, speaker int64) (*InspectionDocument, uint32, error)
}

// NewMessagePage is the message popup's page while it writes a new message:
// the page is made only by Apply.
const NewMessagePage uint32 = 1<<32 - 1

// messageOwner is the field a new message will be named in: field key of
// record index of kind, or a new action for kind "Message".
type messageOwner struct {
	kind  string
	index int
	key   string
}

// beginMessage opens the message popup on a message not yet made.
func (e *MapEditor) beginMessage(owner messageOwner, back bool) {
	e.messageNew = owner
	if back {
		e.openMessage(NewMessagePage)
		return
	}
	e.messagePage, e.objectHidden, e.objectTop = NewMessagePage, false, 0
	e.obj.doc = nil
	e.loadObject()
}

func (e *MapEditor) messageField(field int) bool {
	v, ok := e.scriptPopup()
	return ok && v.node != nil && field >= 0 && field < len(e.obj.fields) && e.obj.fields[field].Key == "value0" && e.nodeParameterKind(v, uint32(e.obj.signature[0]), e.obj.types, 0) == ScriptMessageParameter
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
	if len(o.fields) != 2 {
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
	o.rowField = map[int]int{0: 0, 1: 1}
	lines := strings.Split(o.buf[0], "\n")
	for i := range lines {
		lines[i] = e.editorHostText(strings.TrimSuffix(lines[i], "\r"))
	}
	s.Rows = []popupRow{{Kind: popupMultiline, Label: o.fields[0].Label, Value: strings.Join(lines, "\n"), Focus: o.focus == 0},
		{Kind: popupChoice, Label: o.fields[1].Label, Value: choiceText(o.fields[1], o.buf[1]), Focus: o.focus == 1}}
	if o.note != "" {
		s.Notes = []string{e.editorHostText(o.note)}
	}
	e.objectBackButton(&s)
	if font := e.bodyFont(); font != nil {
		for _, caption := range s.Buttons {
			s.ButtonW = max(s.ButtonW, font.Advance(caption)+16)
		}
	}
	e.fitMessageText(&s)
	return s, []int{0, 0}, true
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
	if !ok || len(e.obj.buf) != 2 {
		return
	}
	speaker, err := strconv.ParseInt(e.obj.buf[1], 10, 32)
	if err == nil && e.messagePage == NewMessagePage {
		e.createMessage(host, speaker)
		return
	}
	var next *InspectionDocument
	if err == nil {
		next, err = host.SetMessage(e.messagePage, e.obj.buf[0], speaker)
	}
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	_ = e.adoptEdit(next, nil)
	e.obj.doc = nil
	e.loadObject()
}

// createMessage makes the new message with the popup's text as one edit and
// shows its page; a new action made for it becomes the popup's way back.
func (e *MapEditor) createMessage(host InspectionMessages, speaker int64) {
	owner := e.messageNew
	next, page, err := host.CreateMessage(owner.kind, owner.index, owner.key, e.obj.buf[0], speaker)
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	if e.adoptEdit(next, nil) != nil || next == nil {
		return
	}
	back := e.objectBack
	if owner.kind == "Message" {
		e.selectNewest("Action")
		e.obj.doc = nil
		e.loadObject()
		back = append(e.objectBack, e.objectPlace())
	}
	e.objectBack, e.messagePage, e.objectHidden = back, page, false
	e.messageNew = messageOwner{}
	e.obj.doc = nil
	e.loadObject()
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
	e.beginMessage(messageOwner{kind: kind, index: index, key: e.obj.fields[field].Key}, true)
}
