package ui

import (
	"strconv"
	"strings"
)

type objectPlace struct {
	kind       string
	index, top int
	group      editorGroup
	register   uint32
	draft      *objectEdit
}

func objectParameter(t uint32) bool {
	return t == scriptUnit || t == scriptGroup || t == scriptPlayer || t == scriptStructure || t == scriptItem || t == ScriptVariableParameter
}

func (e *MapEditor) objectPlace() objectPlace {
	p := objectPlace{index: -1, top: e.objectTop, group: e.groupSel}
	if v, ok := e.scriptVariableRow(e.selected); ok {
		p.kind, p.register = "Variable", v.register
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
}

func (e *MapEditor) backObject() {
	if len(e.objectBack) == 0 || !e.objectMayLeave() {
		return
	}
	p := e.objectBack[len(e.objectBack)-1]
	back := e.objectBack[:len(e.objectBack)-1]
	if p.kind == "Variable" {
		for i, v := range e.scriptVariables() {
			if v.register == p.register {
				e.Select(-2-i, false)
				break
			}
		}
	} else if p.group.set {
		e.chooseGroup(p.group)
		e.refreshGroup()
	} else {
		for i, r := range e.doc.Records {
			if r.Kind == p.kind && r.Index == p.index {
				e.Select(i, false)
				break
			}
		}
	}
	e.objectBack, e.objectTop = back, p.top
	e.obj.doc = nil
	e.loadObject()
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
	slot, err := strconv.Atoi(strings.TrimPrefix(f.Key, "value"))
	if err != nil || slot < 0 || slot >= len(o.types) {
		return InspectionScriptTarget{}, false
	}
	vw, ok := e.scriptPopup()
	if !ok || vw.node == nil {
		return InspectionScriptTarget{}, false
	}
	t := e.nodeParameterKind(vw, uint32(o.signature[0]), o.types, slot)
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
	if target.Record >= 0 || target.Record <= -2 {
		e.openObjectRecord(target.Record, false)
		return
	}
	g := editorGroup{group: int(target.Value), set: true}
	for i, r := range e.doc.Records {
		if r.Kind == "Unit" && r.Group == g.group {
			g.members = append(g.members, i)
			g.player = r.Player
		}
	}
	if len(g.members) > 0 {
		back := append(e.objectBack, e.objectPlace())
		e.chooseGroup(g)
		e.objectBack = back
	}
}

func (e *MapEditor) nodeParameterKind(v scriptView, opcode uint32, types [10]uint32, slot int) uint32 {
	if f := e.doc.Script.Catalog.ParameterKind; f != nil {
		return f(v.condition, opcode, types, slot)
	}
	if types[slot] == ScriptVariableParameter {
		return scriptNone
	}
	return types[slot]
}

func (e *MapEditor) parameterChoices(kind uint32, value int64) []InspectionChoice {
	var out []InspectionChoice
	found := false
	for _, target := range e.doc.Script.Targets[kind] {
		out = append(out, InspectionChoice{int64(target.Value), target.ChoiceText()})
		found = found || int64(target.Value) == value
	}
	if !found {
		out = append(out, InspectionChoice{value, strconv.FormatInt(value, 10)})
	}
	return out
}
