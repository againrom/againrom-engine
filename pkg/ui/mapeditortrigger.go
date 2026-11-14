package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/scriptprofile"
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
	case KindTrigger:
		if r.Index < len(m.Triggers) && m.Triggers[r.Index].Record == id {
			return scriptView{trigger: &m.Triggers[r.Index], index: r.Index}, true
		}
	case KindCondition:
		if r.Index < len(m.Conditions) && m.Conditions[r.Index].Record == id {
			return scriptView{node: &m.Conditions[r.Index], condition: true, index: r.Index}, true
		}
	case KindAction:
		if r.Index < len(m.Actions) && m.Actions[r.Index].Record == id {
			return scriptView{node: &m.Actions[r.Index], index: r.Index}, true
		}
	}
	return scriptView{}, false
}

// scriptPopup reports that the object popup shows the selected record as a
// trigger or a node.
func (e *MapEditor) scriptPopup() (scriptView, bool) {
	if e.messagePage != 0 {
		return scriptView{}, false
	}
	if !e.editable() {
		return scriptView{}, false
	}
	return e.scriptViewOf(e.selected)
}

// The trigger's fields, in this order: name, once, its pairs as left,
// comparison and right, its action slots, then, on an extended map or once
// an OR group is added, each pair's group. An original trigger spells the
// record's three pairs and four actions; an added slot past them widens
// the fields, and Apply then asks to extend the map.
const (
	trigName  = 0
	trigOnce  = 1
	trigPairs = 2
	pairSides = 3 // left, comparison, right
)

func pairField(p, side int) int { return trigPairs + pairSides*p + side }

// triggerSlots are the pairs and action slots the trigger fields spell and
// whether they spell the pairs' groups.
func (o *objectEdit) triggerSlots() (pairs, acts int, grouped bool) {
	for _, f := range o.fields {
		switch {
		case strings.HasPrefix(f.Key, "left"):
			pairs++
		case strings.HasPrefix(f.Key, "action"):
			acts++
		case strings.HasPrefix(f.Key, "group"):
			grouped = true
		}
	}
	return pairs, acts, grouped
}

// actField is action slot a's field; groupField pair p's group field, -1
// when the fields spell no groups.
func (o *objectEdit) actField(a int) int {
	pairs, _, _ := o.triggerSlots()
	return trigPairs + pairSides*pairs + a
}

func (o *objectEdit) groupField(p int) int {
	pairs, acts, grouped := o.triggerSlots()
	if !grouped {
		return -1
	}
	return trigPairs + pairSides*pairs + acts + p
}

// pairGroup is pair p's group as its field reads, 0 without groups.
func (o *objectEdit) pairGroup(p int) int64 {
	f := o.groupField(p)
	if f < 0 {
		return 0
	}
	g, _ := strconv.ParseInt(strings.TrimSpace(o.buf[f]), 10, 64)
	return g
}

// marked is mark i of marks, false past them.
func marked(marks []bool, i int) bool { return i >= 0 && i < len(marks) && marks[i] }

// mark sets mark i of marks to on, growing them to hold it.
func mark(marks *[]bool, i int, on bool) {
	for len(*marks) <= i {
		*marks = append(*marks, false)
	}
	(*marks)[i] = on
}

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

// HeroLabels builds the label of a hero band value (10001 and up): Format
// with the number shown, or RoleFormat with the number and the role when
// Roles names the hero.
type HeroLabels struct {
	Format, RoleFormat string
	Roles              map[int64]string
}

// Label is hero's label showing the number shown.
func (h HeroLabels) Label(shown, hero int64) string {
	if role, ok := h.Roles[hero]; ok && h.RoleFormat != "" {
		return fmt.Sprintf(h.RoleFormat, shown, role)
	}
	return fmt.Sprintf(h.Format, shown)
}

// heroLabel is hero n's label in the editor's words.
func (e *MapEditor) heroLabel(n int64) string {
	return HeroLabels{Format: e.word("editor.param.hero"), RoleFormat: e.word("editor.param.herorole"), Roles: e.heroRoles}.Label(n, n)
}

// targetText names value v of a parameter of type t: a unit, a group with its
// members, a player, a structure, or the number itself.
func (e *MapEditor) targetText(t, v uint32) string {
	if t == ScriptVariableParameter {
		return strconv.FormatUint(uint64(v), 10)
	}
	if t == scriptUnit && v >= 10001 {
		if v <= 11000 {
			return e.heroLabel(int64(v) - 10000)
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
			case scriptUnit:
				// A placed unit's one name holds its number and cell.
				if r := target.Record; r >= 0 && r < len(e.doc.Records) && e.doc.Records[r].Name != "" {
					return e.doc.Records[r].Name
				}
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
		case t == scriptInt && e.doc.Script.Catalog.againrom(condition, opcode):
			value = strconv.FormatInt(slotValue(t, v), 10)
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
		extended := e.doc.Script != nil && e.doc.Script.Extended
		return e.triggerFields(t, len(t.Left), len(t.Acts), extended)
	}
	return e.nodeFields(v, e.obj.types, v.node.Opcode, groupCommand(v.condition, v.node.Opcode, v.node.Values))
}

// triggerFields are trigger t's fields spelling pairs pairs and acts action
// slots, with the pairs' groups when grouped; a slot past t's is empty.
func (e *MapEditor) triggerFields(t *InspectionTrigger, pairs, acts int, grouped bool) []InspectionField {
	out := []InspectionField{{Key: "name", Label: e.word("editor.trigger.name"), Text: true, TextValue: t.Name},
		{Key: "once", Label: e.word("editor.trigger.once"), Value: int64(t.Once), Choices: []InspectionChoice{{1, e.word("editor.trigger.once.yes")}, {0, e.word("editor.trigger.once.no")}}}}
	var cmps []InspectionChoice
	for c := int64(0); c <= scriptprofile.MaxCompare; c++ {
		cmps = append(cmps, InspectionChoice{c, e.cmpWord(c)})
	}
	slot := func(s []uint32, k int) int64 {
		if k < len(s) {
			return int64(s[k])
		}
		return 0
	}
	for p := range pairs {
		out = append(out, InspectionField{Key: scriptprofile.LeftKey(p), Value: slot(t.Left, p), Choices: e.nodeChoices(true)},
			InspectionField{Key: scriptprofile.CmpKey(p), Value: slot(t.Cmp, p), Choices: cmps},
			InspectionField{Key: scriptprofile.RightKey(p), Value: slot(t.Right, p), Choices: e.nodeChoices(true)})
	}
	for a := range acts {
		out = append(out, InspectionField{Key: scriptprofile.ActionKey(a), Value: slot(t.Acts, a), Choices: e.nodeChoices(false)})
	}
	if grouped {
		for p := range pairs {
			out = append(out, InspectionField{Key: scriptprofile.GroupKey(p), Value: slot(t.Group, p)})
		}
	}
	return out
}

// widenTrigger spells pairs pairs and acts action slots in the shown
// trigger's fields, with groups when grouped, each field keeping its text.
func (e *MapEditor) widenTrigger(v scriptView, pairs, acts int, grouped bool) {
	o := &e.obj
	text := map[string]string{}
	for i, f := range o.fields {
		text[f.Key] = o.buf[i]
	}
	o.fields = e.triggerFields(v.trigger, pairs, acts, grouped)
	o.buf = o.buf[:0]
	for _, f := range o.fields {
		b := fieldInput(f)
		if t, ok := text[f.Key]; ok {
			b = t
		}
		o.buf = append(o.buf, b)
	}
	o.focus, o.open = -1, -1
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
		kind := e.nodeParameterKind(v, opcode, types, s)
		switch {
		case kind == ScriptMessageParameter:
			f.Label = e.word("editor.message.parameter")
			f.Choices = e.parameterChoices(kind, f.Value)
		case kind == ScriptVariableParameter:
			f.Label = e.word("editor.role.variable")
			f.Choices = e.parameterChoices(kind, f.Value)
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
		case objectParameter(kind), kind >= ScriptSpellParameter && kind <= ScriptPropertyParameter:
			f.Choices = e.parameterChoices(kind, f.Value)
		}
		out = append(out, f)
	}
	return out
}

// loadScriptObject fills the edit state of a script record.
func (e *MapEditor) loadScriptObject(v scriptView) {
	o := &e.obj
	if v.trigger != nil && !e.triggerDetailsShown(v) {
		e.loadWizard(v)
		return
	}
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
	if ok && v.trigger != nil && o.wiz != nil {
		e.rewizard()
		return
	}
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
	return marked(o.addedPair, p) || o.buf[pairField(p, 0)] != "0" || o.buf[pairField(p, 2)] != "0"
}

func (o *objectEdit) shownAction(a int) bool {
	return marked(o.addedAct, a) || o.buf[o.actField(a)] != "0"
}

// row actions of the script popups.
const (
	actEdit       = "edit"
	actRemovePair = "removepair"
	actAddPair    = "addpair"
	actRemoveAct  = "removeaction"
	actAddAct     = "addaction"
	actAddGroup   = "addgroup"
	actPick       = "pick"
)

// scriptSpec is the trigger or node popup as data.
func (e *MapEditor) scriptSpec(v scriptView) (popupSpec, []int) {
	e.loadObject()
	o := &e.obj
	s := popupSpec{At: e.placedCorner(popObject, homeTrigger), Width: e.popupWide(popObject, scriptPopupWidth), Bounds: e.popupBounds(), NoteInk: editorWarning, LabelWidth: 150}
	o.rowField, o.rowActs, o.rowOpen = map[int]int{}, map[int][]string{}, map[int]int{}
	var links []int
	width := e.popupWide(popObject, scriptPopupWidth) - 2*popupPad - 8
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
		for _, part := range editorTextLines(e.bodyFont(), body, w) {
			add(popupRow{Kind: popupText, Label: indent + part, Ink: inspectionRoleColor(ink)}, -1)
		}
	}
	if v.trigger != nil && o.wiz != nil {
		e.wizardSpec(v, &s, add, text)
	} else if v.trigger != nil {
		s.Title = fmt.Sprintf(e.word("editor.trigger.title"), v.index+1)
		add(popupRow{Kind: popupEdit, Label: o.fields[trigName].Label, Value: e.editorHostText(o.buf[trigName])}, trigName)
		once := e.word("editor.trigger.once.no")
		if strings.TrimSpace(o.buf[trigOnce]) != "0" {
			once = e.word("editor.trigger.once.yes")
		}
		add(popupRow{Kind: popupSwitch, Label: o.fields[trigOnce].Label, Value: once}, trigOnce)
		add(popupRow{Kind: popupHeading, Label: e.word("editor.trigger.conditions")}, -1)
		pairs, group := 0, int64(0)
		slots, actSlots, _ := o.triggerSlots()
		for p := range slots {
			if !o.shownPair(p) {
				continue
			}
			// A pair in another group than the one before opens an OR group.
			if g := o.pairGroup(p); pairs > 0 && g != group {
				add(popupRow{Kind: popupHeading, Label: e.word("editor.trigger.orgroup")}, -1)
				group = g
			} else if pairs == 0 {
				group = g
			}
			pairs++
			for side := range pairSides {
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
		// The add rows reach the extended record's limits on any map: past
		// the original record's, Apply asks to extend the map.
		if most := scriptprofile.Extended; pairs == 0 {
			add(popupRow{Kind: popupText}, -1, actAddPair)
		} else if pairs < most.Pairs {
			add(popupRow{Kind: popupText}, -1, actAddPair, actAddGroup)
		}
		add(popupRow{Kind: popupHeading, Label: e.word("editor.trigger.actions")}, -1)
		acts := 0
		for a := range actSlots {
			if !o.shownAction(a) {
				continue
			}
			actionGap(add, acts)
			acts++
			f := o.actField(a)
			title, params := e.slotText(false, o.buf[f])
			add(popupRow{Kind: popupChoice, Label: fmt.Sprintf(e.word("editor.trigger.action"), acts), Value: title, Ink: editorGold}, f, actEdit, actRemoveAct)
			if params != "" {
				text("    "+params, "")
			}
		}
		if acts < scriptprofile.Extended.Actions {
			add(popupRow{Kind: popupText}, -1, actAddAct)
		}
		_, copies := e.copyOpen()
		s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel"), e.word("editor.trigger.new"), e.word("editor.trigger.delete"), e.word("editor.wizard.back"), e.word("editor.trigger.copy")}
		s.Disabled = []bool{!e.objectDirty(), false, false, false, false, !copies}
		fitButtons(&s, e.bodyFont())
	} else {
		s.Title = fmt.Sprintf(e.word("editor.node.action"), v.node.ID)
		if v.condition {
			s.Title = fmt.Sprintf(e.word("editor.node.condition"), v.node.ID)
		}
		for i, f := range o.fields {
			row := popupRow{Kind: popupEdit, Label: f.Label, Value: o.buf[i]}
			if f.Key == "name" {
				row.Value = e.editorHostText(o.buf[i])
			}
			var acts []string
			switch {
			case strings.HasPrefix(f.Key, "cell"):
				row.Kind, acts = popupPick, []string{actPick}
			case len(f.Choices) > 0:
				row.Kind, row.Value = popupChoice, choiceText(f, o.buf[i])
			}
			if e.nodeFieldTarget(v, i) != targetNone {
				acts = []string{actPickParam}
			}
			if f.Key == "opcode" {
				add(popupRow{Kind: popupHeading, Label: e.word("editor.node.operation")}, -1)
				row.Label = ""
			}
			add(row, i, acts...)
			e.parameterRows(i, &s, add, text)
			if f.Key == "opcode" && e.doc.Script.Catalog.againrom(v.condition, v.node.Opcode) {
				text(e.word("editor.node.againrom"), "warning")
			}
			if f.Key == "opcode" && len(o.fields) > 2 {
				add(popupRow{Kind: popupHeading, Label: e.word("editor.node.parameters")}, -1)
			}
		}
		s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel")}
		s.Disabled = []bool{!e.objectDirty(), false}
		if e.listedCondition(v) {
			s.Buttons, s.Disabled = append(s.Buttons, e.word("editor.scripts.totrigger")), append(s.Disabled, false)
			fitButtons(&s, e.bodyFont())
		}

	}
	if o.note != "" {
		s.Notes = append(s.Notes, editorTextLines(e.bodyFont(), e.editorHostText(o.note), width)...)
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
	case actRemovePair, actRemoveAct, actRemoveRow:
		return e.word("editor.trigger.remove")
	case actAddPair:
		if e.obj.wiz != nil {
			return e.word("editor.wizard.addcondition")
		}
		return e.word("editor.trigger.addpair")
	case actAddAct:
		if e.obj.wiz != nil {
			return e.word("editor.wizard.addaction")
		}
		return e.word("editor.trigger.addaction")
	case actAddGroup:
		return e.word("editor.trigger.addgroup")
	case actDraw:
		return e.word("editor.wizard.draw")
	case actConfigure:
		return e.word("editor.wizard.configure")
	case actPickArea:
		return e.word("editor.wizard.pickarea")
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
	if o.wiz != nil {
		e.wizardRowButton(acts[k], f, hasField)
		return
	}
	switch acts[k] {
	case actEdit:
		e.editSlotNode(f)
	case actRemovePair:
		p := (f - trigPairs) / pairSides
		pairs, _, _ := o.triggerSlots()
		last := pairs - 1
		for q := p; q < last; q++ {
			for side := range pairSides {
				o.buf[pairField(q, side)] = o.buf[pairField(q+1, side)]
			}
			if g := o.groupField(q); g >= 0 {
				o.buf[g] = o.buf[g+1]
			}
			mark(&o.addedPair, q, marked(o.addedPair, q+1))
		}
		o.buf[pairField(last, 0)], o.buf[pairField(last, 2)] = "0", "0"
		mark(&o.addedPair, last, false)
	case actAddPair, actAddGroup:
		e.addTriggerPair(acts[k] == actAddGroup)
	case actRemoveAct:
		mark(&o.addedAct, f-o.actField(0), false)
		o.buf[f] = "0"
	case actAddAct:
		_, actSlots, _ := o.triggerSlots()
		a := 0
		for a < actSlots && o.shownAction(a) {
			a++
		}
		if a == actSlots {
			if a >= scriptprofile.Extended.Actions {
				return
			}
			v, _ := e.scriptPopup()
			pairs, _, grouped := o.triggerSlots()
			e.widenTrigger(v, pairs, a+1, grouped)
		}
		mark(&o.addedAct, a, true)
	case actPick:
		if hasField {
			o.focus = f
			o.pickField = f
			e.leave(leaveDrag)
			e.armPick(pickScriptCell)
		}
	case actPickParam, actPickArea:
		if hasField {
			e.armParamPick(f, acts[k] == actPickArea)
		}
	}
}

// addTriggerPair shows one more condition pair. Without groups it is the
// first empty pair, as the original record fills its pairs; with groups,
// or for a new OR group, it is the pair after the last one shown, in that
// pair's group or, for a new group, the next. A pair past the fields widens
// them.
func (e *MapEditor) addTriggerPair(newGroup bool) {
	o := &e.obj
	v, _ := e.scriptPopup()
	pairs, acts, grouped := o.triggerSlots()
	if newGroup && !grouped {
		e.widenTrigger(v, pairs, acts, true)
		grouped = true
	}
	p, last := -1, -1
	for q := range pairs {
		if o.shownPair(q) {
			last = q
		} else if p < 0 && !grouped {
			p = q
		}
	}
	if p < 0 {
		p = last + 1
	}
	if p >= pairs {
		if p >= scriptprofile.Extended.Pairs {
			return
		}
		e.widenTrigger(v, p+1, acts, grouped)
	}
	mark(&o.addedPair, p, true)
	if c, err := strconv.ParseInt(strings.TrimSpace(o.buf[pairField(p, 1)]), 10, 64); err != nil || c < 0 || c > scriptprofile.MaxCompare {
		o.buf[pairField(p, 1)] = "0"
	}
	if grouped {
		g := int64(0)
		if last >= 0 {
			g = o.pairGroup(last) + int64(refCount(newGroup))
		}
		o.buf[o.groupField(p)] = strconv.FormatInt(g, 10)
	}
}

func (e *MapEditor) editSlotNode(f int) {
	o := &e.obj
	id, _ := strconv.ParseInt(strings.TrimSpace(o.buf[f]), 10, 64)
	n, ok := e.nodeByID(f < o.actField(0), uint32(id))
	if id <= 0 || !ok {
		o.note = e.word("editor.trigger.choosefirst")
		return
	}
	e.openObjectRecord(n.Record, false)
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
	if v.trigger != nil && e.obj.wiz != nil {
		e.wizardButton(v, n)
		return
	}
	switch {
	case n == 0:
		e.applyObject()
	case n == 1:
		e.cancelPopup()
	case v.trigger != nil && n == 2:
		e.newTrigger()
	case v.trigger != nil && n == 3:
		e.deleteTrigger(v)
	case v.trigger != nil && n == 4:
		if e.objectDirty() {
			e.obj.note = e.word("editor.trigger.applyfirst")
			return
		}
		e.ShowTriggerDetails(false)
	case v.trigger != nil && n == 5:
		e.copyTrigger(v.index)
	case n == 2 && e.listedCondition(v):
		e.addToTrigger(v)
	case v.node != nil && n == 2:
		e.backObject()
	}
}

// newTrigger adds a new trigger in one click, as the Scenario palette's New
// trigger does.
func (e *MapEditor) newTrigger() { e.addNow(KindTrigger) }

// freshTrigger is the trigger the one click added, the document its add made
// and the wizard session that click opened; wiz nil for none.
type freshTrigger struct {
	doc   *InspectionDocument
	index int
	wiz   *wizardState
}

// addNow is the one-click add of a script record, the Scenario palette's New
// trigger button and the New trigger command row: it adds the record the Add
// steps of kind add with every proposal accepted, as one history entry. The
// palette stays open. A trigger opens in the wizard, its name field focused
// with its text selected. A shown popup with unapplied edits keeps them and
// says so.
func (e *MapEditor) addNow(kind RecordKind) {
	host, ok := e.addHost()
	row, _ := addKindOf(kind)
	if !ok || !e.addNowOpen(kind) {
		return
	}
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return
	}
	if owner := e.barWindow(); owner != "" && owner != "scripts" {
		e.closeBarWindow()
	}
	e.barOwner = "scripts"
	e.leave(leaveDrag | leavePick | leavePrompt | leaveBrush)
	run := &addRun{kind: row.Kind}
	run.setSteps(row.Steps)
	record := e.addAccepted(host, run, -1, image.Point{})
	ch, err := e.commandAddRecord(record)
	if e.askScriptForm(err, func() { e.addNow(kind) }) {
		return
	}
	if err != nil {
		e.message = e.editorHostText(err.Error())
		return
	}
	if ch.doc == nil {
		return
	}
	e.selectNewest(KindTrigger, ch.doc.AddedRecord)
	e.obj.doc = nil
	e.loadObject()
	if e.wizardOpen() == nil {
		return
	}
	e.fresh = freshTrigger{doc: e.doc, index: ch.doc.AddedRecord.Index, wiz: e.obj.wiz}
	e.focusObjectField(trigName)
	if m, _ := e.inputModel(); m != nil {
		m.SelectAll()
	}
}

// addNowOpen reports that the open map takes the one-click add of kind: its
// host adds records, as the Add steps need.
func (e *MapEditor) addNowOpen(kind RecordKind) bool {
	_, ok := e.addHost()
	return ok && e.addKindOpen(kind)
}

// joinsAdd reports that the Apply of trigger index is the first one of the
// wizard session the one-click add opened, so the write joins the add's
// history entry and one undo takes the named trigger out. A wizard read
// again (Cancel, another selection, a later opening) is a new session, and
// a written save ends the join.
func (e *MapEditor) joinsAdd(index int) bool {
	f := e.fresh
	return f.wiz != nil && f.wiz == e.obj.wiz && f.doc == e.doc && f.index == index
}

// deleteTrigger deletes the shown trigger; its nodes stay.
func (e *MapEditor) deleteTrigger(v scriptView) {
	ch, err := e.commandDelete(RecordRef{Kind: KindTrigger, Index: v.index})
	if err != nil {
		e.message = err.Error()
	} else if ch.doc != nil {
		e.selected = -1
	}
}

// copyHost reports that the map is editable and its host copies triggers.
func (e *MapEditor) copyHost() bool {
	if !e.editable() {
		return false
	}
	_, ok := e.doc.Edits.(InspectionTriggerCopy)
	return ok
}

// copyOpen is the selected trigger's index when its host copies triggers.
func (e *MapEditor) copyOpen() (int, bool) {
	if !e.copyHost() {
		return 0, false
	}
	v, ok := e.scriptViewOf(e.selected)
	return v.index, ok && v.trigger != nil
}

// copySelectedTrigger is the Copy trigger row of the Tools menu and of the
// key table: it copies the selected trigger, and with none selected says to
// select one.
func (e *MapEditor) copySelectedTrigger() {
	index, ok := e.copyOpen()
	if !ok {
		e.message = e.word("editor.trigger.copyfirst")
		return
	}
	e.copyTrigger(index)
}

// copyTrigger copies trigger index right after it as one edit and selects
// the copy, its popup read afresh. A shown popup with unapplied edits keeps
// them and says so.
func (e *MapEditor) copyTrigger(index int) {
	if e.objectDirty() {
		e.obj.note = e.word("editor.trigger.applyfirst")
		return
	}
	ch, err := e.commandCopyTrigger(index)
	if e.askScriptForm(err, func() { e.copyTrigger(index) }) {
		return
	}
	if err != nil {
		e.message = e.editorHostText(err.Error())
		return
	}
	if ch.record == nil {
		return
	}
	e.selectNewest(KindTrigger, *ch.record)
	e.obj.doc = nil
	e.loadObject()
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
		pairs, acts, grouped := o.triggerSlots()
		for p := range pairs {
			if o.shownPair(p) && (o.buf[pairField(p, 0)] == "0" || o.buf[pairField(p, 2)] == "0") {
				return nil, fmt.Errorf(e.word("editor.trigger.sides"), p+1)
			}
		}
		for a := range acts {
			if marked(o.addedAct, a) && o.buf[o.actField(a)] == "0" {
				return nil, fmt.Errorf(e.word("editor.trigger.noaction"), a+1)
			}
		}
		// Fields wider than the original record, or an extended map's, are
		// written whole: the host writes them as one trigger.
		whole := grouped || pairs > scriptprofile.RecordPairs || acts > scriptprofile.RecordActions || e.doc.Script.Extended
		for i, f := range o.fields {
			slot := i >= trigPairs
			if o.buf[i] == fieldInput(f) && !(whole && slot) {
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
	// A reference slot the user has not chosen is refused here, in the
	// editor's words, before the host sees it.
	for i, f := range o.fields {
		s, err := strconv.Atoi(strings.TrimPrefix(f.Key, "value"))
		if !strings.HasPrefix(f.Key, "value") || err != nil || s < 0 || s >= len(o.types) {
			continue
		}
		switch o.types[s] {
		case scriptPlayer, scriptUnit, scriptStructure: // the host accepts 0 only for a group and an item
			n, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64)
			if err == nil && n == 0 && (o.types[s] != v.node.Types[s] || v.node.Values[s] != 0) {
				return nil, fmt.Errorf("%s: %s", f.Label, e.word("editor.node.unset"))
			}
		}
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
	if v.trigger != nil && o.wiz != nil {
		return o.wiz.dirty()
	}
	if v.trigger != nil {
		return slices.Contains(o.addedPair, true) || slices.Contains(o.addedAct, true)
	}
	return o.types != v.node.Types
}

// PopupLines are the shown object popup's rows as text, label then value,
// the boxes of its canvas, the labels of the canvas's lines and its notes:
// what a headless driver reads.
func (e *MapEditor) PopupLines() []string {
	s, _, ok := e.objectSpec()
	if !ok {
		return nil
	}
	var out []string
	for _, r := range s.Rows {
		mark := ""
		if r.Fold != FoldNone {
			mark = rowFoldMarks[r.Fold] + " " // as TreeLines marks a branch
		}
		out = append(out, strings.TrimSpace(mark+r.Label+" "+r.Value))
	}
	return append(append(append(append(out, s.Canvas.legendWords()...), s.Canvas.texts()...), s.Canvas.labels()...), s.Notes...)
}

// PopupCanvasBoxes are the boxes of the shown object popup's canvas: each
// one's text and its window rectangle, which a headless driver clicks.
func (e *MapEditor) PopupCanvasBoxes() (texts []string, rects []image.Rectangle) {
	s, _, ok := e.objectSpec()
	if !ok || s.Canvas == nil {
		return nil, nil
	}
	boxes := layoutPopup(s, e.font, e.bodyFont()).CanvasBoxes
	rects = make([]image.Rectangle, len(boxes))
	for i, row := range boxes {
		rects[i] = row.rect
	}
	return s.Canvas.texts(), rects
}

// PopupCanvasView is the window rectangle the shown object popup's canvas
// shows its sheet in, empty for none.
func (e *MapEditor) PopupCanvasView() image.Rectangle {
	s, _, ok := e.objectSpec()
	if !ok || s.Canvas == nil {
		return image.Rectangle{}
	}
	return layoutPopup(s, e.font, e.bodyFont()).Canvas
}

// PopupRowButtons are the captions of popup row n's own buttons.
func (e *MapEditor) PopupRowButtons(n int) []string {
	s, _, ok := e.objectSpec()
	if !ok || n < 0 || n >= len(s.Rows) {
		return nil
	}
	return s.Rows[n].Buttons
}

// PopupRecordLinks returns each popup row's record destination, or -1.
func (e *MapEditor) PopupRecordLinks() []int {
	s, links, _ := e.objectSpec()
	out := make([]int, len(s.Rows))
	for i := range out {
		out[i] = -1
		if i < len(links) && links[i] < 0 {
			out[i] = -links[i] - 1
		}
	}
	return out
}

func (e *MapEditor) controlPopupRowRect(n int) (image.Rectangle, bool) {
	s, _, ok := e.objectSpec()
	if !ok {
		return image.Rectangle{}, false
	}
	for _, row := range layoutPopup(s, e.font, e.bodyFont()).Rows {
		if row.Index == n {
			return row.Rect, true
		}
	}
	return image.Rectangle{}, false
}

// PopupTipRects are the boxes of the hover lines ChromeImage draws now.
func (e *MapEditor) PopupTipRects() []image.Rectangle {
	var specs []popupSpec
	if s, ok := e.listSpec(); ok {
		specs = append(specs, s)
	}
	if s, _, ok := e.objectSpec(); ok {
		specs = append(specs, s)
	}
	if s, ok := e.promptSpec(); ok {
		specs = append(specs, s)
	}
	var out []image.Rectangle
	for _, s := range specs {
		l := layoutPopup(s, e.font, e.bodyFont())
		if tip, ok := popupTipAt(image.Rect(0, 0, e.window.X, e.window.Y), e.font, e.bodyFont(), s, l, e.pointer, e.pointerOK); ok {
			out = append(out, tip.Rect)
		}
	}
	return out
}

// controlPopupButtonRect is the window rectangle of button n of the object popup's
// button row.
func (e *MapEditor) controlPopupButtonRect(n int) (image.Rectangle, bool) {
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

// PopupButtons are the captions of the object popup's button row.
func (e *MapEditor) PopupButtons() []string {
	s, _, ok := e.objectSpec()
	if !ok {
		return nil
	}
	return s.Buttons
}

// controlRowButtonRect is the window rectangle of button k of the shown row that
// edits field n; n -1 with actions false is the add-pair row's button (k 1
// adds an OR group), with actions true the add-action row's. In the dialog editor n 2c+1 is card
// c's cross or Restore and n -1 the "+".
func (e *MapEditor) controlRowButtonRect(n, k int, actions bool) (image.Rectangle, bool) {
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
		if part, ok := e.obj.rowOpen[rr.Index]; e.messagePage != 0 && ok {
			f, has = 2*part+1, true
		}
		acts := e.obj.rowActs[rr.Index]
		match := has && f == n || n < 0 && !has && len(acts) > 0 && acts[0] == want
		if match && k >= 0 && k < len(rr.Buttons) {
			return rr.Buttons[k], true
		}
	}
	return image.Rectangle{}, false
}

// controlDropdownRowRect is the window rectangle of row n of the open drop-down,
// the object popup's or the trigger choice's, when that row is shown.
func (e *MapEditor) controlDropdownRowRect(n int) (image.Rectangle, bool) {
	d, ok := e.openDropdown()
	if !ok {
		d, ok = e.openAreaDropdown()
	}
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
		d, ok = e.openAreaDropdown()
	}
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
	} else if d, ok := e.openAreaDropdown(); ok {
		d.List.Scroll(n)
	}
}
