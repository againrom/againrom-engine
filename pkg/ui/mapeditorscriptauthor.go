package ui

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

const editorScriptsFilter = 7

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
		case KindTrigger:
			section = 0
		case KindCondition:
			v, ok := e.scriptViewOf(id)
			if !ok || !variables[id] && !conditions[v.node.ID] {
				section = 2
			}
		case KindAction:
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
	caption := e.editorHostText(v.node.Label)
	if !v.condition && e.nodeParameterKind(v, v.node.Opcode, v.node.Types, 0) == ScriptMessageParameter {
		title = fmt.Sprintf(e.word("editor.node.message"), v.index+1)
		caption = e.messageFirstLine(v.node.Values[0])
	}
	return strings.TrimSpace(title + "  " + caption)
}

// scriptVariableView is one variable: its register, the preset constant or
// the check sharing its register (-1 none), the triggers that change and
// read it, whether any node addresses it, and its label.
type scriptVariableView struct {
	register         uint32
	constant, shared int
	readers, writers []int
	addressed        bool
	label            string
}

// scriptVariables are the variables in register order: each register a node
// addresses and each labelled register, the set the host's variable choices
// list as records -2 and below.
func (e *MapEditor) scriptVariables() []scriptVariableView {
	if e.doc == nil || e.doc.Script == nil {
		return nil
	}
	m := e.doc.Script
	addressed := map[uint32]bool{}
	for _, nodes := range [][]InspectionScriptNode{m.Conditions, m.Actions} {
		for _, n := range nodes {
			if n.AddressesVariable {
				addressed[n.VariableRegister] = true
			}
		}
	}
	registers := maps.Clone(addressed)
	for r := range m.Labels {
		registers[r] = true
	}
	var out []scriptVariableView
	for register := range registers {
		v := scriptVariableView{register: register, constant: -1, shared: -1, addressed: addressed[register], label: m.Labels[register]}
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
	if v.label != "" {
		title += "  " + e.editorHostText(v.label)
	} else if v.constant >= 0 {
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
	s := popupSpec{Title: e.variableTitle(v), At: e.placedCorner(popObject, homeObjectWindow), Width: e.popupWide(popObject, 420), Bounds: e.popupBounds(), Buttons: []string{e.word("editor.button.cancel")}}
	if _, ok := e.variableLabels(); ok {
		s.Buttons = []string{e.word("editor.labels.edit"), e.word("editor.labels.clear"), e.word("editor.button.cancel")}
		s.Disabled = []bool{false, v.label == "", false}
		s.ButtonFit = true
		fitButtons(&s, e.bodyFont())
	}
	var links []int
	if !v.addressed {
		s.Rows, links = append(s.Rows, popupRow{Kind: popupText, Label: e.word("editor.labels.unused")}), append(links, 0)
	}
	for _, group := range []struct {
		title string
		ids   []int
	}{
		{e.word("editor.scripts.writers"), v.writers}, {e.word("editor.scripts.readers"), v.readers},
	} {
		var rows []scriptPopupLink
		for _, id := range group.ids {
			rows = append(rows, scriptPopupLink{record: id, title: e.scriptRowText(id, 0)})
		}
		e.scriptLinkSection(&s, &links, group.title, rows)
	}
	if v.constant >= 0 {
		e.scriptPopupLink(&s, &links, scriptPopupLink{record: v.constant, title: e.scriptRowText(v.constant, 2)})
	}
	s.MaxRows = e.objectRowsShown(true)
	e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	s.Top = e.objectTop
	return s, links, true
}

func (e *MapEditor) messageFirstLine(value uint32) string {
	for _, t := range e.doc.Script.Targets[ScriptMessageParameter] {
		if t.NewMessage || t.Value != value {
			continue
		}
		for _, line := range strings.Split(t.Text, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				return line
			}
		}
	}
	return ""
}
