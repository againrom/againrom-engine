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
			for _, ids := range [][]uint32{t.Left, t.Right} {
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
// read it, whether any node addresses it, its label, and each place a
// trigger names a node that addresses it.
type scriptVariableView struct {
	register         uint32
	constant, shared int
	readers, writers []int
	addressed        bool
	label            string
	uses             []scriptVariableUse
}

// scriptVariableUse is one place a trigger names a node that addresses a
// variable: the trigger's index, its pair (0..2) or action slot (0..3), the
// side of the pair, and the node. Uses run in trigger order, a trigger's
// pairs before its actions, the left side before the right.
type scriptVariableUse struct {
	trigger, slot int
	action, right bool
	node          InspectionScriptNode
}

// variableNode is the node of nodes with id that addresses register.
func variableNode(nodes []InspectionScriptNode, id, register uint32) (InspectionScriptNode, bool) {
	for _, n := range nodes {
		if id != 0 && n.ID == id && n.AddressesVariable && n.VariableRegister == register {
			return n, true
		}
	}
	return InspectionScriptNode{}, false
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
		for ti, t := range m.Triggers {
			read, written := false, false
			for p := range t.Left {
				for side, id := range [2]uint32{t.Left[p], t.Right[p]} {
					if n, ok := variableNode(m.Conditions, id, register); ok {
						v.uses, read = append(v.uses, scriptVariableUse{trigger: ti, slot: p, right: side == 1, node: n}), true
					}
				}
			}
			for a, id := range t.Acts {
				if n, ok := variableNode(m.Actions, id, register); ok {
					v.uses, written = append(v.uses, scriptVariableUse{trigger: ti, slot: a, action: true, node: n}), true
				}
			}
			if written {
				v.writers = append(v.writers, t.Record)
			}
			if read {
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

// variableTitle is variable v's row in the scripts list: its number, its
// label after two spaces, and the condition it shares its register with.
func (e *MapEditor) variableTitle(v scriptVariableView) string {
	title := fmt.Sprintf(e.word("editor.scripts.variable"), v.register)
	if label := e.variableLabel(v); label != "" {
		title += "  " + label
	}
	if v.shared >= 0 {
		n, _ := e.scriptViewOf(v.shared)
		title += fmt.Sprintf(e.word("editor.scripts.shared"), n.index+1)
	}
	return title
}

// variableLabel is variable v's label as the editor shows it, else the label
// of the constant it names, else empty.
func (e *MapEditor) variableLabel(v scriptVariableView) string {
	if v.label != "" {
		return e.editorHostText(v.label)
	}
	if n, ok := e.scriptViewOf(v.constant); ok && n.node != nil && n.node.VariableLabel != "" {
		return e.editorHostText(n.node.VariableLabel)
	}
	return ""
}

// variableSpec is the variable window: its title names the variable, its
// (?) says what a variable is; the triggers that change it and those that
// read it are links, then the condition it shares its register with, with
// its own (?), and the constant it names. A variable no trigger addresses
// says so in a note at the bottom.
func (e *MapEditor) variableSpec(v scriptVariableView) (popupSpec, []int, bool) {
	e.loadObject()
	title := fmt.Sprintf(e.word("editor.scripts.variable"), v.register)
	if label := e.variableLabel(v); label != "" {
		title = fmt.Sprintf(e.word("editor.scripts.variablenamed"), v.register, label)
	}
	s := popupSpec{Title: title, TitleHelp: e.word("editor.scripts.variable.help"), At: e.keptCorner(popObject), Bounds: e.popupBounds(), NoteInk: editorPaper,
		Buttons: []string{e.word("editor.button.cancel")}}
	if _, ok := e.variableLabels(); ok {
		s.Buttons = []string{e.word("editor.labels.edit"), e.word("editor.labels.clear"), e.word("editor.button.cancel")}
		s.Disabled = []bool{false, v.label == "", false}
	}
	if !v.addressed {
		s.Notes = []string{e.word("editor.labels.unused")}
	}
	// The links wrap to the window's width: the widest a link may ask.
	s.Width = min(e.window.X-16, e.popupWide(popObject, e.window.X-16))
	var links []int
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
		e.scriptLinkSection(&s, &links, group.title, rows, FoldNone)
	}
	if v.shared >= 0 {
		n, _ := e.scriptViewOf(v.shared)
		s.Rows = append(s.Rows, popupRow{Kind: popupLink, Label: fmt.Sprintf(e.word("editor.scripts.sharedlink"), n.index+1), Help: e.word("editor.scripts.sharedlink.help")})
		links = append(links, -v.shared-1)
	}
	if v.constant >= 0 {
		e.scriptPopupLink(&s, &links, scriptPopupLink{record: v.constant, title: e.scriptRowText(v.constant, 2)})
	}
	s.Width = e.popupWide(popObject, min(naturalWidth(&s, e.font, e.bodyFont(), 0), e.window.X-16))
	s.Notes = wrapNotes(s.Notes, advanceOf(e.bodyFont()), s.Width)
	e.pageObjectRows(&s, true)
	return s, links, true
}

func (e *MapEditor) messageFirstLine(value uint32) string {
	for _, t := range e.doc.Script.Targets[ScriptMessageParameter] {
		if t.NewChoice != 0 || t.Value != value {
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
