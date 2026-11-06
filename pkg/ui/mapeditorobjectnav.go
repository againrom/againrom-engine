package ui

import (
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/render/text"
)

// personHidden are the keys of the person window's fields it shows in
// neither view: the facing, the speaker line and the map keys. Their values
// stay on the unit; Apply writes only the fields a row edits.
var personHidden = []string{"facing", "speaker", "keys"}

// personQuiet are the keys of the person window's fields shown without the
// lines that explain them.
var personQuiet = []string{"name"}

// hiddenField reports that the window of record r does not show field f:
// a field of the person window, which alone gives fields a section, named
// in personHidden.
func hiddenField(r InspectionRecord, f InspectionField) bool {
	return r.Kind == KindUnit && f.Section != "" && slices.Contains(personHidden, f.Key)
}

// linkedRecord is the document index of the record ref names, -1 for none.
func (e *MapEditor) linkedRecord(ref RecordRef, identity ...uint64) int {
	if len(identity) > 0 && identity[0] != 0 {
		ref = RecordRef{}
		for current, key := range e.barRecords.current {
			if key == identity[0] {
				ref = current
				break
			}
		}
	}
	if ref.Kind == "" || e.doc == nil {
		return -1
	}
	return slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == ref.Kind && r.Index == ref.Index })
}

type objectPlace struct {
	kind       RecordKind
	index, top int
	group      editorGroup
	register   uint32
	draft      *objectEdit
	goals      bool
	listTop    int
	goalRow    int
	hidden     bool
	details    int // the trigger details view shown, as triggerDetails
	section    int // the person window's tab shown
	identity   uint64
}

func objectParameter(t uint32) bool {
	return t == scriptUnit || t == scriptGroup || t == scriptPlayer || t == scriptStructure || t == scriptItem || t == ScriptVariableParameter || t == ScriptMessageParameter
}

func (e *MapEditor) objectPlace() objectPlace {
	p := objectPlace{index: -1, top: e.objectTop, group: e.groupSel, details: e.triggerDetails, section: e.obj.section}
	if v, ok := e.scriptVariableRow(e.selected); ok {
		p.kind, p.register = KindVariable, v.register
		return p
	}
	if e.selected >= 0 && e.selected < len(e.doc.Records) {
		r := e.doc.Records[e.selected]
		p.kind, p.index = r.Kind, r.Index
	}
	return p
}

func (e *MapEditor) objectMayLeave() bool {
	e.loadObject()
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return false
	}
	return true
}

func (e *MapEditor) openObjectRecord(record int, jump bool) {
	if !e.objectMayLeave() {
		return
	}
	back := append(e.objectBack, e.objectPlace())
	if e.Select(record, jump) {
		e.objectBack = back
	}
}

func (e *MapEditor) objectBackButton(s *popupSpec) {
	if len(e.objectBack) == 0 {
		return
	}
	s.Buttons = append(s.Buttons, e.word("editor.trigger.back"))
	s.Disabled = append(s.Disabled, false)
	fitButtons(s, e.bodyFont())
}

// fitButtons widens s so its whole button row lies inside the popup, each
// button at the width popupButtonWidths gives it in the font body.
func fitButtons(s *popupSpec, body *text.Font) {
	s.Width = max(s.Width, buttonRowWidth(*s, body))
}

// buttonRowWidth is the width the popup needs for its button row.
func buttonRowWidth(s popupSpec, body *text.Font) int {
	row := popupPad
	for _, bw := range popupButtonWidths(s, body) {
		row += bw + popupPad
	}
	if sw := switchButtonWidth(s, body); sw > 0 {
		row += sw + popupPad
	}
	return row
}

func (e *MapEditor) backObject() {
	if len(e.objectBack) == 0 || !e.objectMayLeave() {
		return
	}
	p := e.objectBack[len(e.objectBack)-1]
	back := e.objectBack[:len(e.objectBack)-1]
	e.messagePage = 0
	if p.goals {
		if s, ok := e.script(); ok && !e.report.open {
			e.openGoals(s, p.goalRow)
			e.listSpec()
		}
		e.selectFinding(p.goalRow)
		e.objectBack, e.objectHidden, e.objectTop = back, p.hidden, p.top
		e.obj.open, e.obj.focus = -1, -1
		if e.model != nil {
			top, _ := e.model.Visible()
			e.model.Scroll(p.listTop - top)
		}
		return
	}
	if p.kind == KindVariable {
		for i, v := range e.scriptVariables() {
			if v.register == p.register {
				e.Select(-2-i, false)
				break
			}
		}
	} else if p.group.set {
		e.chooseGroup(p.group, true)
		e.refreshGroup()
	} else {
		if i := e.linkedRecord(RecordRef{Kind: p.kind, Index: p.index}, p.identity); i >= 0 {
			e.Select(i, false)
		}
	}
	e.objectBack, e.objectTop, e.triggerDetails = back, p.top, p.details
	e.obj.doc = nil
	e.loadObject()
	e.obj.section = p.section
	if p.draft != nil {
		for i, f := range p.draft.fields {
			if p.draft.buf[i] == fieldInput(f) {
				continue
			}
			for j, current := range e.obj.fields {
				if current.Key == f.Key {
					e.obj.buf[j] = p.draft.buf[i]
				}
			}
		}
	}
}

func (e *MapEditor) parameterTarget(field int) (InspectionScriptTarget, bool) {
	o := &e.obj
	if field < 0 || field >= len(o.fields) {
		return InspectionScriptTarget{}, false
	}
	f := o.fields[field]
	t := ScriptMessageParameter
	if !e.messageField(field) {
		slot, err := strconv.Atoi(strings.TrimPrefix(f.Key, "value"))
		if err != nil || slot < 0 || slot >= len(o.types) {
			return InspectionScriptTarget{}, false
		}
		vw, ok := e.scriptPopup()
		if !ok || vw.node == nil {
			return InspectionScriptTarget{}, false
		}
		t = e.nodeParameterKind(vw, uint32(o.signature[0]), o.types, slot)
	}
	v, err := strconv.ParseUint(strings.TrimSpace(o.buf[field]), 10, 32)
	if err != nil {
		return InspectionScriptTarget{}, false
	}
	if target := e.doc.Script.OpenTarget(t, uint32(v)); target != nil {
		return *target, true
	}
	return InspectionScriptTarget{}, false
}

func (e *MapEditor) openParameter(field int) {
	target, ok := e.parameterTarget(field)
	if !ok || !e.objectMayLeave() {
		return
	}
	if e.messageField(field) {
		e.openMessage(target.Value)
		return
	}
	if target.Record >= 0 || target.Record <= -2 {
		e.openObjectRecord(target.Record, false)
		return
	}
	g := editorGroup{group: int(target.Value), set: true}
	for i, r := range e.doc.Records {
		if r.Kind == KindUnit && r.Group == g.group {
			g.members = append(g.members, i)
			g.player = r.Player
		}
	}
	if len(g.members) > 0 {
		back := append(e.objectBack, e.objectPlace())
		e.chooseGroup(g, true)
		e.objectBack = back
	}
}

func (e *MapEditor) nodeParameterKind(v scriptView, opcode uint32, types [10]uint32, slot int) uint32 {
	if f := e.doc.Script.Catalog.ParameterKind; f != nil {
		return f(v.condition, opcode, types, slot)
	}
	if !v.condition && opcode == 2 && slot == 0 && types[slot] == scriptInt {
		return ScriptMessageParameter
	}
	if types[slot] == ScriptVariableParameter || types[slot] == ScriptMessageParameter {
		return scriptNone
	}
	return types[slot]
}

func (e *MapEditor) parameterChoices(kind uint32, value int64) []InspectionChoice {
	var out []InspectionChoice
	found := false
	for _, target := range e.doc.Script.Targets[kind] {
		out = append(out, InspectionChoice{target.ChoiceValue(), target.ChoiceText()})
		found = found || target.ChoiceValue() == value
	}
	if !found {
		out = append(out, InspectionChoice{value, strconv.FormatInt(value, 10)})
	}
	return out
}

func (e *MapEditor) parameterRows(field int, s *popupSpec, add func(popupRow, int, ...string), text func(string, string)) {
	if body, ok := e.messageText(field); ok {
		for _, line := range strings.Split(body, "\n") {
			text("  "+line, "")
		}
	}
	if _, ok := e.parameterTarget(field); ok {
		e.obj.rowOpen[len(s.Rows)] = field
		add(popupRow{Kind: popupLink, Label: e.word("editor.param.open") + " " + e.obj.fields[field].Label}, -1)
	}
}
