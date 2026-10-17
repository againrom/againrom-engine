package ui

import (
	"fmt"
	"image"
	"strconv"
	"strings"
)

// The trigger editor: a trigger's popup reads as the rule it is, its name,
// its condition pairs with the comparison in words, its actions and its once
// switch; a condition or action opens in its own popup with one control per
// parameter by what the parameter names. Both are the object popup: its
// edit state, drop-downs, typing and Apply serve them, over fields this file
// builds from the document's script model.

// scriptPopupWidth is the trigger and node popups' width.
const scriptPopupWidth = 560

// scriptView is the script record the object popup shows: a trigger, a
// condition or an action of the script model.
type scriptView struct {
	trigger   *InspectionTrigger
	node      *InspectionScriptNode
	condition bool
	index     int // the record's index among its kind
}

// scriptViewOf is the script model's view of document record id.
func (e *MapEditor) scriptViewOf(id int) (scriptView, bool) {
	if e.doc == nil || e.doc.Script == nil || id < 0 || id >= len(e.doc.Records) {
		return scriptView{}, false
	}
	m := e.doc.Script
	r := e.doc.Records[id]
	switch r.Kind {
	case "Trigger":
		if r.Index < len(m.Triggers) && m.Triggers[r.Index].Record == id {
			return scriptView{trigger: &m.Triggers[r.Index], index: r.Index}, true
		}
	case "Condition":
		if r.Index < len(m.Conditions) && m.Conditions[r.Index].Record == id {
			return scriptView{node: &m.Conditions[r.Index], condition: true, index: r.Index}, true
		}
	case "Action":
		if r.Index < len(m.Actions) && m.Actions[r.Index].Record == id {
			return scriptView{node: &m.Actions[r.Index], index: r.Index}, true
		}
	}
	return scriptView{}, false
}

// scriptPopup reports that the object popup shows the selected record as a
// trigger or a node.
func (e *MapEditor) scriptPopup() (scriptView, bool) {
	if !e.editable() {
		return scriptView{}, false
	}
	return e.scriptViewOf(e.selected)
}

// The trigger's fields, in this order: name, once, the three pairs as
// left, comparison and right, then the four action slots.
const (
	trigName  = 0
	trigOnce  = 1
	trigPairs = 2
	trigActs  = 11
)

func pairField(p, side int) int { return trigPairs + 3*p + side }

// opName is the operation's word: its name where a claim names it, else
// "Condition N" or "Action N".
func (e *MapEditor) opName(condition bool, opcode uint32) string {
	if condition {
		if w, ok := e.checkWord(opcode); ok && e.doc.Script.Catalog.named(true, opcode) {
			return w
		}
		return fmt.Sprintf(e.word("editor.check.unnamed"), opcode)
	}
	if w, ok := e.actionWord(opcode); ok && e.doc.Script.Catalog.named(false, opcode) {
		return w
	}
	return fmt.Sprintf(e.word("editor.action.unnamed"), opcode)
}

// cmpWord is a comparison code in words (TRIG-CMP-006); a code above 5 never
// holds.
func (e *MapEditor) cmpWord(code int64) string {
	if w, ok := e.cmpWordOf(code); ok {
		return w
	}
	return fmt.Sprintf(e.word("editor.cmp.never"), code)
}

// paramWord is the meaning of a parameter type code.
func (e *MapEditor) paramWord(t uint32) string {
	switch t {
	case scriptInt:
		return e.word("editor.param.number")
	case scriptGroup:
		return e.word("editor.param.group")
	case scriptPlayer:
		return e.word("editor.param.player")
	case scriptUnit:
		return e.word("editor.param.unit")
	case scriptX, scriptY:
		return e.word("editor.param.cell")
	case scriptConst:
		return e.word("editor.param.const")
	case scriptItem:
		return e.word("editor.param.item")
	case scriptStructure:
		return e.word("editor.param.structure")
	}
	return fmt.Sprintf(e.word("editor.param.kind"), t)
}

// slotLabel is the label of slot s of a node: the role a claim gives it, else
// what its type names.
func (e *MapEditor) slotLabel(condition bool, opcode, command uint32, s int, t uint32) string {
	if sig, ok := e.doc.Script.Catalog.signature(condition, opcode, command); ok && sig[s].Type == t && sig[s].Role != "" {
		if w, ok := e.roleWord(sig[s].Role); ok {
			return w
		}
	}
	return e.paramWord(t)
}

// targetText names value v of a parameter of type t: a unit, a group with its
// members, a player, a structure, or the number itself.
func (e *MapEditor) targetText(t, v uint32) string {
	if t == scriptUnit && v >= 10001 {
		if v <= 11000 {
			return fmt.Sprintf(e.word("editor.param.hero"), v-10000)
		}
		return fmt.Sprintf(e.word("editor.param.named"), v)
	}
	if e.doc.Script != nil {
		for _, target := range e.doc.Script.Targets[t] {
			if target.Value != v {
				continue
			}
			switch t {
			case scriptGroup:
				return fmt.Sprintf("%d (%s)", v, fmt.Sprintf(e.word("editor.param.members"), target.Members))
			case scriptPlayer:
				return target.Name
			}
			s := strconv.FormatUint(uint64(v), 10)
			if target.Name != "" {
				s += " " + target.Name
			}
			if target.Record >= 0 && target.Record < len(e.doc.Records) && e.doc.Records[target.Record].Spatial {
				c := e.doc.Records[target.Record].Cell
				s += fmt.Sprintf(" (%d,%d)", c.X, c.Y)
			}
			return s
		}
	}
	return strconv.FormatUint(uint64(v), 10)
}

// command is a group command node's first parameter.
func groupCommand(condition bool, opcode uint32, values [10]uint32) uint32 {
	if !condition && opcode == scriptGroupCommand {
		return values[0]
	}
	return 0
}

// nodeParams are a node's parameters by meaning, one "label value" text each.
func (e *MapEditor) nodeParams(condition bool, opcode uint32, types, values [10]uint32) []string {
	var out []string
	command := groupCommand(condition, opcode, values)
	for s := 0; s < len(types); s++ {
		t, v := types[s], values[s]
		if t == scriptNone {
			continue
		}
		label := e.slotLabel(condition, opcode, command, s, t)
		value := e.targetText(t, v)
		switch {
		case t == scriptX && s+1 < len(types) && types[s+1] == scriptY:
			value = fmt.Sprintf("%d,%d", v, values[s+1])
			s++
		case !condition && opcode == scriptGroupCommand && s == 0 && t == scriptConst:
			value = e.commandName(v)
		}
		out = append(out, label+" "+value)
	}
	return out
}

func (e *MapEditor) commandName(c uint32) string {
	if w, ok := e.commandWord(c); ok {
		return w
	}
	return strconv.FormatUint(uint64(c), 10)
}

// nodeTitle is a node's number and operation: "#3 Units in group".
func (e *MapEditor) nodeTitle(condition bool, n InspectionScriptNode) string {
	return fmt.Sprintf("#%d %s", n.ID, e.opName(condition, n.Opcode))
}

// nodeByID is the one node of the family with id, or false.
func (e *MapEditor) nodeByID(condition bool, id uint32) (InspectionScriptNode, bool) {
	nodes := e.doc.Script.Actions
	if condition {
		nodes = e.doc.Script.Conditions
	}
	var found InspectionScriptNode
	n := 0
	for _, node := range nodes {
		if node.ID == id {
			found, n = node, n+1
		}
	}
	return found, n == 1
}

// nodeChoices are the nodes a trigger slot can name, then a new one.
func (e *MapEditor) nodeChoices(condition bool) []InspectionChoice {
	nodes := e.doc.Script.Actions
	newWord := e.word("editor.trigger.newaction")
	if condition {
		nodes, newWord = e.doc.Script.Conditions, e.word("editor.trigger.newcondition")
	}
	var out []InspectionChoice
	for _, n := range nodes {
		if _, ok := e.nodeByID(condition, n.ID); !ok || n.ID == 0 || !condition && n.Opcode >= 0x10000 {
			continue
		}
		out = append(out, InspectionChoice{Value: int64(n.ID), Label: e.nodeTitle(condition, n)})
	}
	return append(out, InspectionChoice{Value: -1, Label: newWord})
}

// scriptFields are the fields the popup edits for a script record.
func (e *MapEditor) scriptFields(v scriptView) []InspectionField {
	if t := v.trigger; t != nil {
		out := []InspectionField{{Key: "name", Label: e.word("editor.trigger.name"), Text: true, TextValue: t.Name},
			{Key: "once", Label: e.word("editor.trigger.once"), Value: int64(t.Once), Choices: []InspectionChoice{{1, e.word("editor.trigger.once.yes")}, {0, e.word("editor.trigger.once.no")}}}}
		var cmps []InspectionChoice
		for c := int64(0); c <= 5; c++ {
			cmps = append(cmps, InspectionChoice{c, e.cmpWord(c)})
		}
		for p := 0; p < 3; p++ {
			out = append(out, InspectionField{Key: fmt.Sprintf("left%d", p), Value: int64(t.Left[p]), Choices: e.nodeChoices(true)},
				InspectionField{Key: fmt.Sprintf("cmp%d", p), Value: int64(t.Cmp[p]), Choices: cmps},
				InspectionField{Key: fmt.Sprintf("right%d", p), Value: int64(t.Right[p]), Choices: e.nodeChoices(true)})
		}
		for a := 0; a < 4; a++ {
			out = append(out, InspectionField{Key: fmt.Sprintf("action%d", a), Value: int64(t.Acts[a]), Choices: e.nodeChoices(false)})
		}
		return out
	}
	return e.nodeFields(v, e.obj.types, v.node.Opcode, groupCommand(v.condition, v.node.Opcode, v.node.Values))
}

// slotValue is a slot's value as its field reads it: a number or constant
// is signed.
func slotValue(t, v uint32) int64 {
	if t == scriptInt || t == scriptConst {
		return int64(int32(v))
	}
	return int64(v)
}

// nodeFields are a node's fields for its slot types: its label, its
// operation, and one field per parameter, a cell as one "x,y" field.
func (e *MapEditor) nodeFields(v scriptView, types [10]uint32, opcode, command uint32) []InspectionField {
	n := v.node
	out := []InspectionField{{Key: "name", Label: e.word("editor.node.label"), Text: true, TextValue: n.Label}}
	var ops []InspectionChoice
	for _, op := range e.doc.Script.Catalog.operations(v.condition) {
		ops = append(ops, InspectionChoice{int64(op.Opcode), e.opName(v.condition, op.Opcode)})
	}
	out = append(out, InspectionField{Key: "opcode", Label: e.word("editor.node.operation"), Value: int64(n.Opcode), Choices: ops})
	for s := 0; s < len(types); s++ {
		t := types[s]
		if t == scriptNone {
			continue
		}
		f := InspectionField{Key: fmt.Sprintf("value%d", s), Label: e.slotLabel(v.condition, opcode, command, s, t), Value: slotValue(t, n.Values[s])}
		if t != n.Types[s] {
			f.Value = 0
		}
		switch {
		case t == scriptX && s+1 < len(types) && types[s+1] == scriptY:
			f.Key, f.Text = fmt.Sprintf("cell%d", s), true
			f.TextValue = fmt.Sprintf("%d,%d", f.Value, n.Values[s+1])
			if types[s+1] != n.Types[s+1] {
				f.TextValue = fmt.Sprintf("%d,0", f.Value)
			}
			s++
		case !v.condition && opcode == scriptGroupCommand && s == 0 && t == scriptConst:
			for _, c := range e.doc.Script.Catalog.GroupCommands {
				f.Choices = append(f.Choices, InspectionChoice{int64(c), e.commandName(c)})
			}
		case t == scriptUnit || t == scriptGroup || t == scriptPlayer || t == scriptStructure:
			for _, target := range e.doc.Script.Targets[t] {
				f.Choices = append(f.Choices, InspectionChoice{int64(target.Value), e.targetText(t, target.Value)})
			}
			if t == scriptUnit && n.Values[s] >= 10001 && t == n.Types[s] {
				f.Choices = append(f.Choices, InspectionChoice{int64(n.Values[s]), e.targetText(t, n.Values[s])})
			}
		}
		out = append(out, f)
	}
	return out
}

// loadScriptObject fills the edit state of a script record.
func (e *MapEditor) loadScriptObject(v scriptView) {
	o := &e.obj
	o.fontText = true
	if v.node != nil {
		o.types = v.node.Types
		o.signature = [2]int64{int64(v.node.Opcode), int64(groupCommand(v.condition, v.node.Opcode, v.node.Values))}
	}
	o.fields = e.scriptFields(v)
	o.buf = o.buf[:0]
	for _, f := range o.fields {
		o.buf = append(o.buf, fieldInput(f))
	}
}

// retypeNode follows a changed operation or group command: the parameter
// slots become the new operation's signature, where a claim states one; a
// slot keeps its typed text while its type stays.
func (e *MapEditor) retypeNode() {
	o := &e.obj
	v, ok := e.scriptPopup()
	if !ok || v.node == nil {
		return
	}
	text := map[string]string{}
	for i, f := range o.fields {
		text[f.Key] = o.buf[i]
	}
	opcode, _ := strconv.ParseInt(strings.TrimSpace(text["opcode"]), 10, 64)
	command := int64(0)
	if !v.condition && opcode == scriptGroupCommand {
		command, _ = strconv.ParseInt(strings.TrimSpace(text["value0"]), 10, 64)
		if o.types[0] != scriptConst {
			command = 0
		}
	}
	key := [2]int64{opcode, command}
	if key == o.signature {
		return
	}
	o.signature = key
	sig, ok := e.doc.Script.Catalog.signature(v.condition, uint32(opcode), uint32(command))
	if !ok && !v.condition && opcode == scriptGroupCommand {
		command, o.signature[1] = 3, 3 // stand ground, the command without parameters
		text["value0"] = "3"
		sig, ok = e.doc.Script.Catalog.signature(false, scriptGroupCommand, 3)
	}
	old := o.types
	if ok {
		for s := range sig {
			o.types[s] = sig[s].Type
		}
	}
	o.fields = e.nodeFields(v, o.types, uint32(opcode), uint32(command))
	o.buf = o.buf[:0]
	for _, f := range o.fields {
		b := fieldInput(f)
		if t, kept := text[f.Key]; kept && (f.Key == "name" || f.Key == "opcode" || keySlotTypeKept(f.Key, old, o.types) || f.Key == "value0" && command != 0) {
			b = t
		}
		o.buf = append(o.buf, b)
	}
	o.open = -1
}

// keySlotTypeKept reports that the slot field key names keeps its type.
func keySlotTypeKept(key string, old, now [10]uint32) bool {
	for _, prefix := range []string{"value", "cell"} {
		if s, err := strconv.Atoi(strings.TrimPrefix(key, prefix)); err == nil && strings.HasPrefix(key, prefix) && s >= 0 && s < len(now) {
			return old[s] == now[s] && (prefix == "value" || s+1 < len(now) && old[s+1] == now[s+1])
		}
	}
	return false
}

// scriptAdd marks the shown pairs and action slots beyond those the buffer
// names: a pair or action added and not yet chosen.
func (o *objectEdit) shownPair(p int) bool {
	return o.added[p] || o.buf[pairField(p, 0)] != "0" || o.buf[pairField(p, 2)] != "0"
}

func (o *objectEdit) shownAction(a int) bool {
	return o.added[3+a] || o.buf[trigActs+a] != "0"
}

// row actions of the script popups.
const (
	actEdit       = "edit"
	actRemovePair = "removepair"
	actAddPair    = "addpair"
	actRemoveAct  = "removeaction"
	actAddAct     = "addaction"
	actPick       = "pick"
)

// scriptSpec is the trigger or node popup as data.
func (e *MapEditor) scriptSpec(v scriptView) (popupSpec, []int) {
	e.loadObject()
	o := &e.obj
	if !e.objPlaced {
		e.objectAt, e.objPlaced = image.Pt(e.window.X-8-scriptPopupWidth, e.barHeight()+8), true
	}
	s := popupSpec{At: e.objectAt, Width: scriptPopupWidth, Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 150}
	o.rowField, o.rowActs = map[int]int{}, map[int][]string{}
	var links []int
	width := scriptPopupWidth - 2*popupPad - 8
	add := func(row popupRow, field int, acts ...string) {
		if field >= 0 {
			o.rowField[len(s.Rows)] = field
			row.Focus = field == o.focus
			if row.Kind == popupEdit && row.Focus {
				if b := e.bodyFont(); b != nil {
					row.Caret = b.Advance(row.Value)
				}
			}
		}
		if len(acts) > 0 {
			o.rowActs[len(s.Rows)] = acts
		}
		s.Rows, links = append(s.Rows, row), append(links, 0)
	}
	text := func(line string, ink string) {
		// A wrapped line keeps its indent on every part.
		body := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(body)]
		w := width
		if b := e.bodyFont(); b != nil {
			w -= b.Advance(indent)
		}
		for _, part := range editorWrap(e.bodyFont(), body, w) {
			add(popupRow{Kind: popupText, Label: indent + part, Ink: inspectionRoleColor(ink)}, -1)
		}
	}
	if v.trigger != nil {
		s.Title = fmt.Sprintf(e.word("editor.trigger.title"), v.index+1)
		add(popupRow{Kind: popupEdit, Label: o.fields[trigName].Label, Value: o.buf[trigName]}, trigName)
		once := e.word("editor.trigger.once.no")
		if strings.TrimSpace(o.buf[trigOnce]) != "0" {
			once = e.word("editor.trigger.once.yes")
		}
		add(popupRow{Kind: popupSwitch, Label: o.fields[trigOnce].Label, Value: once}, trigOnce)
		add(popupRow{Kind: popupHeading, Label: e.word("editor.trigger.conditions")}, -1)
		pairs := 0
		for p := 0; p < 3; p++ {
			if !o.shownPair(p) {
				continue
			}
			pairs++
			for side := 0; side < 3; side++ {
				f := pairField(p, side)
				if side == 1 {
					code, _ := strconv.ParseInt(strings.TrimSpace(o.buf[f]), 10, 64)
					add(popupRow{Kind: popupChoice, Label: "  " + e.word("editor.trigger.comparison"), Value: e.cmpWord(code)}, f, actRemovePair)
					continue
				}
				label := fmt.Sprintf(e.word("editor.trigger.pair"), p+1)
				if side == 2 {
					label = "  " + e.word("editor.trigger.with")
				}
				title, params := e.slotText(true, o.buf[f])
				add(popupRow{Kind: popupChoice, Label: label, Value: title, Ink: editorCondition}, f, actEdit)
				if params != "" {
					text("    "+params, "")
				}
			}
		}
		if pairs == 0 {
			text(e.word("editor.trigger.nopair"), "warning")
		}
		if pairs < 3 {
			add(popupRow{Kind: popupText}, -1, actAddPair)
		}
		add(popupRow{Kind: popupHeading, Label: e.word("editor.trigger.actions")}, -1)
		acts := 0
		for a := 0; a < 4; a++ {
			if !o.shownAction(a) {
				continue
			}
			acts++
			title, params := e.slotText(false, o.buf[trigActs+a])
			add(popupRow{Kind: popupChoice, Label: fmt.Sprintf(e.word("editor.trigger.action"), acts), Value: title, Ink: editorGold}, trigActs+a, actEdit, actRemoveAct)
			if params != "" {
				text("    "+params, "")
			}
		}
		if acts < 4 {
			add(popupRow{Kind: popupText}, -1, actAddAct)
		}
		s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel"), e.word("editor.trigger.new"), e.word("editor.trigger.delete")}
		s.Disabled = []bool{!e.objectDirty(), false, false, false}
	} else {
		s.Title = fmt.Sprintf(e.word("editor.node.action"), v.node.ID)
		if v.condition {
			s.Title = fmt.Sprintf(e.word("editor.node.condition"), v.node.ID)
		}
		for i, f := range o.fields {
			row := popupRow{Kind: popupEdit, Label: editorClip(e.bodyFont(), f.Label, 140), Value: o.buf[i]}
			var acts []string
			switch {
			case strings.HasPrefix(f.Key, "cell"):
				row.Kind, acts = popupPick, []string{actPick}
			case len(f.Choices) > 0:
				row.Kind, row.Value = popupChoice, choiceText(f, o.buf[i])
			}
			if f.Key == "opcode" {
				add(popupRow{Kind: popupHeading, Label: e.word("editor.node.operation")}, -1)
				row.Label = ""
			}
			add(row, i, acts...)
			if f.Key == "opcode" && len(o.fields) > 2 {
				add(popupRow{Kind: popupHeading, Label: e.word("editor.node.parameters")}, -1)
			}
		}
		s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}
		s.Disabled = []bool{!e.objectDirty(), false}
		if e.backTrigger >= 0 {
			s.Buttons = append(s.Buttons, e.word("editor.trigger.back"))
			s.Disabled = append(s.Disabled, false)
		}
	}
	if o.note != "" {
		s.Notes = append(s.Notes, editorWrap(e.bodyFont(), e.editorHostText(o.note), width)...)
	}
	for r, acts := range o.rowActs {
		for _, a := range acts {
			s.Rows[r].Buttons = append(s.Rows[r].Buttons, e.actWord(a))
		}
	}
	s.MaxRows = e.objectRowsShown(true)
	e.objectTop = max(0, min(e.objectTop, len(s.Rows)-s.MaxRows))
	s.Top = e.objectTop
	return s, links
}

func (e *MapEditor) actWord(a string) string {
	switch a {
	case actEdit:
		return e.word("editor.trigger.edit")
	case actRemovePair, actRemoveAct:
		return e.word("editor.trigger.remove")
	case actAddPair:
		return e.word("editor.trigger.addpair")
	case actAddAct:
		return e.word("editor.trigger.addaction")
	}
	return e.word("editor.object.pick")
}

// slotText is a trigger slot's node as a title and its parameters.
func (e *MapEditor) slotText(condition bool, buf string) (string, string) {
	id, err := strconv.ParseInt(strings.TrimSpace(buf), 10, 64)
	switch {
	case err != nil || id == 0:
		return e.word("editor.trigger.choose"), ""
	case id < 0 && condition:
		return e.word("editor.trigger.newcondition"), ""
	case id < 0:
		return e.word("editor.trigger.newaction"), ""
	}
	n, ok := e.nodeByID(condition, uint32(id))
	if !ok {
		return fmt.Sprintf("#%d ?", id), ""
	}
	return e.nodeTitle(condition, n), strings.Join(e.nodeParams(condition, n.Opcode, n.Types, n.Values), ", ")
}

// scriptRowButton is a press on button k of spec row r of a script popup.
func (e *MapEditor) scriptRowButton(r, k int) {
	o := &e.obj
	acts := o.rowActs[r]
	if k < 0 || k >= len(acts) {
		return
	}
	f, hasField := o.rowField[r]
	o.note, o.open = "", -1
	switch acts[k] {
	case actEdit:
		e.editSlotNode(f)
	case actRemovePair:
		p := (f - trigPairs) / 3
		for q := p; q < 2; q++ {
			for side := 0; side < 3; side++ {
				o.buf[pairField(q, side)] = o.buf[pairField(q+1, side)]
			}
			o.added[q] = o.added[q+1]
		}
		o.buf[pairField(2, 0)], o.buf[pairField(2, 2)], o.added[2] = "0", "0", false
	case actAddPair:
		for p := 0; p < 3; p++ {
			if !o.shownPair(p) {
				o.added[p] = true
				if c, err := strconv.ParseInt(strings.TrimSpace(o.buf[pairField(p, 1)]), 10, 64); err != nil || c < 0 || c > 5 {
					o.buf[pairField(p, 1)] = "0"
				}
				break
			}
		}
	case actRemoveAct:
		a := f - trigActs
		o.buf[f], o.added[3+a] = "0", false
	case actAddAct:
		for a := 0; a < 4; a++ {
			if !o.shownAction(a) {
				o.added[3+a] = true
				break
			}
		}
	case actPick:
		if hasField {
			o.focus = f
			o.pickField = f
			e.leave(leaveDrag)
			e.armPick(pickScriptCell)
		}
	}
}

// editSlotNode opens the node trigger field f names in its own popup, with a
// Back button to the trigger. A trigger with changes not applied keeps its
// popup and says so.
func (e *MapEditor) editSlotNode(f int) {
	o := &e.obj
	if e.objectDirty() {
		o.note = e.word("editor.trigger.applyfirst")
		return
	}
	id, _ := strconv.ParseInt(strings.TrimSpace(o.buf[f]), 10, 64)
	n, ok := e.nodeByID(f < trigActs, uint32(id))
	if id <= 0 || !ok {
		o.note = e.word("editor.trigger.choosefirst")
		return
	}
	back := e.doc.Records[e.selected].Index
	e.Select(n.Record, false)
	e.backTrigger = back
}

// backToTrigger returns from a node popup to the trigger it was opened from.
func (e *MapEditor) backToTrigger() {
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return
	}
	for _, t := range e.doc.Script.Triggers {
		if e.doc.Records[t.Record].Index == e.backTrigger {
			e.Select(t.Record, false)
			return
		}
	}
	e.backTrigger = -1
}

// scriptCellPicked writes the clicked cell into the pick's field.
func (e *MapEditor) scriptCellPicked(cell image.Point) {
	o := &e.obj
	if o.pickField >= 0 && o.pickField < len(o.buf) {
		o.buf[o.pickField] = fmt.Sprintf("%d,%d", cell.X, cell.Y)
	}
	o.pickField = -1
}

// scriptButton is a press on the popup's button row.
func (e *MapEditor) scriptButton(v scriptView, n int) {
	switch {
	case n == 0:
		e.applyObject()
	case n == 1:
		e.obj.doc = nil
		e.loadObject()
	case v.trigger != nil && n == 2:
		e.newTrigger()
	case v.trigger != nil && n == 3:
		e.deleteTrigger(v)
	case v.node != nil && n == 2:
		e.backToTrigger()
	}
}

// newTrigger adds a trigger and opens it.
func (e *MapEditor) newTrigger() {
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return
	}
	host, ok := e.doc.Edits.(InspectionScriptAuthor)
	if !ok {
		return
	}
	next, err := host.AddScript("Trigger")
	if e.adoptEdit(next, err) != nil || next == nil {
		return
	}
	e.selectNewest("Trigger")
	e.Select(e.selected, false)
}

// deleteTrigger deletes the shown trigger; its nodes stay.
func (e *MapEditor) deleteTrigger(v scriptView) {
	next, err := e.doc.Edits.Delete("Trigger", v.index)
	if e.adoptEdit(next, err) == nil && next != nil {
		e.selected = -1
	}
}

// scriptValues are the changed fields of a script popup as the host's field
// values: a new node slot asks for its node; a node's operation goes first,
// then the slot types it brings, then the values, a cell as its two words.
func (e *MapEditor) scriptValues(v scriptView) ([]InspectionFieldValue, error) {
	o := &e.obj
	var out []InspectionFieldValue
	number := func(i int) (int64, error) {
		n, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s: %s", o.fields[i].Label, e.word("editor.object.number"))
		}
		return n, nil
	}
	if v.trigger != nil {
		for p := 0; p < 3; p++ {
			if o.shownPair(p) && (o.buf[pairField(p, 0)] == "0" || o.buf[pairField(p, 2)] == "0") {
				return nil, fmt.Errorf(e.word("editor.trigger.sides"), p+1)
			}
		}
		for a := 0; a < 4; a++ {
			if o.added[3+a] && o.buf[trigActs+a] == "0" {
				return nil, fmt.Errorf(e.word("editor.trigger.noaction"), a+1)
			}
		}
		for i, f := range o.fields {
			if o.buf[i] == fieldInput(f) {
				continue
			}
			if f.Text {
				out = append(out, InspectionFieldValue{Key: f.Key, Text: true, TextValue: o.buf[i]})
				continue
			}
			n, err := number(i)
			if err != nil {
				return nil, err
			}
			out = append(out, InspectionFieldValue{Key: f.Key, Value: n, New: n < 0})
		}
		return out, nil
	}
	var values []InspectionFieldValue
	for i, f := range o.fields {
		if o.buf[i] == fieldInput(f) && !(strings.HasPrefix(f.Key, "value") || strings.HasPrefix(f.Key, "cell")) {
			continue
		}
		switch {
		case f.Text && f.Key == "name":
			out = append(out, InspectionFieldValue{Key: f.Key, Text: true, TextValue: o.buf[i]})
		case strings.HasPrefix(f.Key, "cell"):
			s, _ := strconv.Atoi(strings.TrimPrefix(f.Key, "cell"))
			xs, ys, ok := strings.Cut(o.buf[i], ",")
			x, errX := strconv.ParseInt(strings.TrimSpace(xs), 10, 64)
			y, errY := strconv.ParseInt(strings.TrimSpace(ys), 10, 64)
			if !ok || errX != nil || errY != nil {
				return nil, fmt.Errorf("%s: %s", f.Label, e.word("editor.node.cell"))
			}
			for k, w := range []int64{x, y} {
				if o.types[s+k] != v.node.Types[s+k] || int64(v.node.Values[s+k]) != w {
					values = append(values, InspectionFieldValue{Key: fmt.Sprintf("value%d", s+k), Value: w})
				}
			}
		default:
			n, err := number(i)
			if err != nil {
				return nil, err
			}
			s, _ := strconv.Atoi(strings.TrimPrefix(f.Key, "value"))
			if f.Key == "opcode" {
				out = append(out, InspectionFieldValue{Key: f.Key, Value: n})
			} else if o.types[s] != v.node.Types[s] || slotValue(o.types[s], v.node.Values[s]) != n {
				values = append(values, InspectionFieldValue{Key: f.Key, Value: n})
			}
		}
	}
	// The slot types go first, so each value is checked as its new type; the
	// operation goes last.
	var types []InspectionFieldValue
	for s := range o.types {
		if o.types[s] != v.node.Types[s] {
			types = append(types, InspectionFieldValue{Key: fmt.Sprintf("type%d", s), Value: int64(o.types[s])})
		}
	}
	var last []InspectionFieldValue
	for _, x := range out {
		if x.Key == "opcode" {
			last = append(last, x)
		} else {
			types = append([]InspectionFieldValue{x}, types...)
		}
	}
	return append(append(types, values...), last...), nil
}

// scriptDirty reports a script popup with changes not applied.
func (e *MapEditor) scriptDirty(v scriptView) bool {
	o := &e.obj
	if v.trigger != nil {
		for _, a := range o.added {
			if a {
				return true
			}
		}
		return false
	}
	return o.types != v.node.Types
}

// PopupLines are the shown object popup's rows as text, label then value,
// and its notes: what a headless driver reads.
func (e *MapEditor) PopupLines() []string {
	s, _, ok := e.objectSpec()
	if !ok {
		return nil
	}
	var out []string
	for _, r := range s.Rows {
		out = append(out, strings.TrimSpace(r.Label+" "+r.Value))
	}
	return append(out, s.Notes...)
}

// PopupButtonRect is the window rectangle of button n of the object popup's
// button row.
func (e *MapEditor) PopupButtonRect(n int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if n < 0 || n >= len(l.Buttons) {
		return image.Rectangle{}, false
	}
	return l.Buttons[n], true
}

// RowButtonRect is the window rectangle of button k of the shown row that
// edits field n; n -1 with actions false is the add-pair row's button, with
// actions true the add-action row's.
func (e *MapEditor) RowButtonRect(n, k int, actions bool) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	want := actAddPair
	if actions {
		want = actAddAct
	}
	for _, rr := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		f, has := e.obj.rowField[rr.Index]
		acts := e.obj.rowActs[rr.Index]
		match := has && f == n || n < 0 && !has && len(acts) == 1 && acts[0] == want
		if match && k >= 0 && k < len(rr.Buttons) {
			return rr.Buttons[k], true
		}
	}
	return image.Rectangle{}, false
}

// DropdownRowRect is the window rectangle of row n of the object popup's
// open drop-down, when that row is shown.
func (e *MapEditor) DropdownRowRect(n int) (image.Rectangle, bool) {
	d, ok := e.openDropdown()
	if !ok {
		return image.Rectangle{}, false
	}
	top, count := d.List.Visible()
	if n < top || n >= top+count {
		return image.Rectangle{}, false
	}
	return d.listBox(e.bodyFont()).Row(n - top), true
}

// DropdownChoices are the texts of the open drop-down's rows.
func (e *MapEditor) DropdownChoices() []string {
	d, ok := e.openDropdown()
	if !ok {
		return nil
	}
	var out []string
	for _, r := range d.List.Rows() {
		out = append(out, r.Text)
	}
	return out
}

// ScrollDropdown scrolls the open drop-down by n rows, as its wheel does.
func (e *MapEditor) ScrollDropdown(n int) {
	if _, ok := e.openDropdown(); ok {
		e.obj.choice.Scroll(n)
	}
}
