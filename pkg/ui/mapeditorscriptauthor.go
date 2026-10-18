package ui

import (
	"fmt"
	"image"
	"sort"
)

const editorScriptsFilter = 7

func scriptRecord(kind string) bool {
	return kind == "Action" || kind == "Condition" || kind == "Trigger"
}

func (e *MapEditor) keyScripts(string) error {
	e.showRecords(editorScriptsFilter)
	return nil
}

func (e *MapEditor) scriptSections() [4][]int {
	var sections [4][]int
	if e.doc == nil {
		return sections
	}
	m := e.doc.Script
	conditions, actions, variables := map[uint32]bool{}, map[uint32]bool{}, map[int]bool{}
	if m != nil {
		for _, t := range m.Triggers {
			for _, ids := range [][3]uint32{t.Left, t.Right} {
				for _, id := range ids {
					if id != 0 {
						conditions[id] = true
					}
				}
			}
			for _, id := range t.Acts {
				if id != 0 {
					actions[id] = true
				}
			}
		}
		for i, v := range e.scriptVariables() {
			sections[1] = append(sections[1], -2-i)
			if v.constant >= 0 {
				variables[v.constant] = true
			}
		}
	}
	for id, r := range e.doc.Records {
		section := -1
		switch r.Kind {
		case "Trigger":
			section = 0
		case "Condition":
			v, ok := e.scriptViewOf(id)
			if !ok || !variables[id] && !conditions[v.node.ID] {
				section = 2
			}
		case "Action":
			v, ok := e.scriptViewOf(id)
			if !ok || !actions[v.node.ID] {
				section = 3
			}
		}
		if section >= 0 {
			sections[section] = append(sections[section], id)
		}
	}
	return sections
}

func (e *MapEditor) scriptRowText(id, section int) string {
	if v, ok := e.scriptVariableRow(id); ok {
		return e.variableTitle(v)
	}
	v, ok := e.scriptViewOf(id)
	if !ok {
		return e.recordSummary(id)
	}
	if v.trigger != nil {
		return fmt.Sprintf(e.word("editor.trigger.title"), v.index+1) + "  " + e.editorHostText(v.trigger.Name)
	}
	title := fmt.Sprintf(e.word("editor.node.condition"), v.index+1)
	if !v.condition {
		title = fmt.Sprintf(e.word("editor.node.action"), v.index+1)
	}
	return title + "  " + e.editorHostText(v.node.Label)
}

type scriptVariableView struct {
	register         uint32
	constant, shared int
	readers, writers []int
}

func (e *MapEditor) scriptVariables() []scriptVariableView {
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	m := e.doc.Script
	registers := map[uint32]bool{}
	for _, nodes := range [][]InspectionScriptNode{m.Conditions, m.Actions} {
		for _, n := range nodes {
			if n.AddressesVariable {
				registers[n.VariableRegister] = true
			}
		}
	}
	var out []scriptVariableView
	for register := range registers {
		v := scriptVariableView{register: register, constant: -1, shared: -1}
		if uint64(register) < uint64(len(m.Conditions)) {
			n := m.Conditions[register]
			if n.Opcode == 0x10002 {
				v.constant = n.Record
			} else {
				v.shared = n.Record
			}
		}
		for _, t := range m.Triggers {
			uses := func(nodes []InspectionScriptNode, ids []uint32) bool {
				for _, n := range nodes {
					if !n.AddressesVariable || n.VariableRegister != register || n.ID == 0 {
						continue
					}
					for _, id := range ids {
						if id == n.ID {
							return true
						}
					}
				}
				return false
			}
			if uses(m.Actions, t.Acts[:]) {
				v.writers = append(v.writers, t.Record)
			}
			if uses(m.Conditions, append(append([]uint32{}, t.Left[:]...), t.Right[:]...)) {
				v.readers = append(v.readers, t.Record)
			}
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].register < out[j].register })
	return out
}

func (e *MapEditor) scriptVariableRow(id int) (scriptVariableView, bool) {
	if id <= -2 {
		if rows := e.scriptVariables(); -2-id < len(rows) {
			return rows[-2-id], true
		}
	}
	return scriptVariableView{}, false
}

func (e *MapEditor) variableTitle(v scriptVariableView) string {
	title := fmt.Sprintf(e.word("editor.scripts.variable"), v.register)
	if v.constant >= 0 {
		n, _ := e.scriptViewOf(v.constant)
		if n.node.Label != "" {
			title += "  " + e.editorHostText(n.node.Label)
		}
	}
	if v.shared >= 0 {
		n, _ := e.scriptViewOf(v.shared)
		title += fmt.Sprintf(e.word("editor.scripts.shared"), n.index+1)
	}
	return title
}

func (e *MapEditor) variableSpec(v scriptVariableView) (popupSpec, []int, bool) {
	e.loadObject()
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-8-420, e.barHeight()+8), true
	}
	s := popupSpec{Title: e.variableTitle(v), At: e.objectAt, Width: 420, Bounds: e.popupBounds(), Buttons: []string{e.word("editor.button.cancel")}}
	var links []int
	link := func(id int, title string) {
		for _, part := range editorWrap(e.bodyFont(), title, s.Width-2*popupPad-8) {
			s.Rows, links = append(s.Rows, popupRow{Kind: popupLink, Label: part}), append(links, -id-1)
		}
	}
	for _, group := range []struct {
		title string
		ids   []int
	}{
		{e.word("editor.scripts.writers"), v.writers}, {e.word("editor.scripts.readers"), v.readers},
	} {
		title := group.title
		if len(group.ids) == 0 {
			title = fmt.Sprintf(e.word("editor.scripts.emptyheading"), title)
		}
		s.Rows, links = append(s.Rows, popupRow{Kind: popupText, Label: title}), append(links, 0)
		for _, id := range group.ids {
			link(id, e.scriptRowText(id, 0))
		}
	}
	if v.constant >= 0 {
		link(v.constant, e.scriptRowText(v.constant, 2))
	}
	s.MaxRows = e.objectRowsShown(true)
	e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	s.Top = e.objectTop
	return s, links, true
}
