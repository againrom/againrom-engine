package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

type InspectionMessages interface {
	MessageFields(page uint32) []InspectionField
	SetMessage(page uint32, body string, speaker int64) (*InspectionDocument, error)
	NewMessage(kind string, index int, key string) (*InspectionDocument, error)
}

func (e *MapEditor) messageField(field int) bool {
	v, ok := e.scriptPopup()
	return ok && v.node != nil && field >= 0 && field < len(e.obj.fields) && e.obj.fields[field].Key == "value0" && e.nodeParameterKind(v, uint32(e.obj.signature[0]), e.obj.types, 0) == ScriptMessageParameter
}

func (e *MapEditor) openMessage(page uint32) {
	e.objectBack = append(e.objectBack, e.objectPlace())
	e.messagePage, e.objectHidden = page, false
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
	s := popupSpec{Title: fmt.Sprintf(e.word("editor.message.page"), e.messagePage), At: e.objectAt, Width: 660, Bounds: e.popupBounds(), LabelWidth: 135, NoteInk: editorWarning,
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
	return s, []int{0, 0}, true
}

func (e *MapEditor) applyMessage() {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok || len(e.obj.buf) != 2 {
		return
	}
	speaker, err := strconv.ParseInt(e.obj.buf[1], 10, 32)
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

func (e *MapEditor) newMessage(field int) {
	host, ok := e.doc.Edits.(InspectionMessages)
	if !ok || field < 0 || field >= len(e.obj.fields) {
		return
	}
	if e.obj.buf[field] == strconv.FormatInt(NewMessageChoice, 10) {
		e.obj.buf[field] = fieldInput(e.obj.fields[field])
	}
	if !e.objectMayLeave() {
		return
	}
	kind, index := e.selectedRecord()
	key := e.obj.fields[field].Key
	next, err := host.NewMessage(kind, index, key)
	if err != nil {
		e.obj.note = err.Error()
		return
	}
	if e.adoptEdit(next, nil) != nil || next == nil {
		return
	}
	e.obj.doc = nil
	e.loadObject()
	for i, f := range e.obj.fields {
		if f.Key == key {
			e.openParameter(i)
			return
		}
	}
}
