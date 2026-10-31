package ui

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"
)

// The trigger wizard: a trigger read as When (its first condition), And if
// (the others) and Do (its actions), each row an item of a ready list in
// plain words with its parameters in the row. Picking an item makes its
// nodes; the host writes the row's check, the constant it is compared with
// and the pair, or its action, on Apply. A pair or action no item reads
// stays as it is and shows as "Other"; Details opens the pair view.

// wizParamKind is what a row parameter names, and so how it is edited.
type wizParamKind uint8

const (
	wpUnit    wizParamKind = iota + 1 // a choice of units, or a click on one
	wpPlayer                          // a choice of players
	wpGroup                           // a choice of groups
	wpNumber                          // a typed number
	wpCell                            // "x,y", or a click on the map
	wpBox                             // "x1,y1,x2,y2", or a box drawn on the map
	wpMessage                         // a choice of message pages
	wpCommand                         // a choice of the group commands that take a cell
	wpRelation
)

// wizParam is one parameter of an item: its kind, its label's word, its
// default number, and whether a node slot holds it (else the constant the
// check is compared with does). A last parameter stands in the node's last
// slots, as a group command's target group does (ALM-TRIG-045); the others
// fill the slots from the first on.
type wizParam struct {
	kind      wizParamKind
	labelWord string
	def       int64
	inNode    bool
	last      bool
	slot      uint32
}

// wizItem is one entry of the ready list. A condition is its check compared
// with a constant: by cmp with right, or, cmp -1, by the comparison the row
// chooses; rightParam names the parameter that holds the constant, -1 for
// right. fixed are node slots after the parameters', as type and value.
// onePerTrigger is an item a trigger holds at most once: another row of the
// trigger shows it shut.
type wizItem struct {
	labelWord     string
	opcode        uint32
	params        []wizParam
	cmp           int
	right         int64
	rightParam    int
	fixed         [][2]uint32
	onePerTrigger bool
	hintWord      string
}

var (
	pUnit   = wizParam{kind: wpUnit, labelWord: "editor.wizard.p.unit", inNode: true}
	pUnit2  = wizParam{kind: wpUnit, labelWord: "editor.wizard.p.unit2", inNode: true}
	pPlayer = wizParam{kind: wpPlayer, labelWord: "editor.wizard.p.player", def: 1, inNode: true}
	pGroup  = wizParam{kind: wpGroup, labelWord: "editor.wizard.p.group", inNode: true}
	pPoint  = wizParam{kind: wpCell, labelWord: "editor.wizard.p.point", inNode: true}
	pArea   = wizParam{kind: wpBox, labelWord: "editor.wizard.p.area", inNode: true}

	pToPlayer = wizParam{kind: wpPlayer, labelWord: "editor.wizard.p.player2", def: 2, inNode: true}
)

func numbered(p wizParam) wizParam {
	p.slot = scriptInt
	return p
}

func (p wizParam) types() []uint32 {
	if p.slot != 0 {
		return []uint32{p.slot}
	}
	return paramTypes(p.kind)
}

var wizardRelationWords = []string{"editor.wizard.relation.peace", "editor.wizard.relation.hostile", "editor.wizard.relation.alliance"}

// wizConditions are the ready conditions. Check 2 is a unit in a box
// (TRIG-CHECKGEOM-130), 3 a unit within a radius of a cell and 6 the
// distance between two units, 255 when either is dead (TRIG-DIST-014),
// 15 the distance from a cell to a player's nearest unit (TRIG-NEAREST-029),
// 5 a unit alive, 1 a group's living count (TRIG-COUNT-015), 8 a player's
// (TRIG-LISTREAD-131); a constant is preset to its value (TRIG-COND-003).
var wizConditions = []wizItem{
	{labelWord: "editor.wizard.c.heroarea", opcode: 2, params: []wizParam{pUnit, pArea}, cmp: 0, right: 1, rightParam: -1},
	{labelWord: "editor.wizard.c.playernear", opcode: 15, params: []wizParam{pPlayer, pPoint, {kind: wpNumber, labelWord: "editor.wizard.p.radius", def: 3}}, cmp: 5, rightParam: 2},
	{labelWord: "editor.wizard.c.unitnear", opcode: 3, params: []wizParam{pUnit, pPoint, {kind: wpNumber, labelWord: "editor.wizard.p.radius", def: 3, inNode: true}}, cmp: 0, right: 1, rightParam: -1},
	{labelWord: "editor.wizard.c.unitnearunit", opcode: 6, hintWord: "editor.wizard.c.unitnearunit.hint", params: []wizParam{pUnit, pUnit2, {kind: wpNumber, labelWord: "editor.wizard.p.radius", def: 3}}, cmp: 5, rightParam: 2},
	{labelWord: "editor.wizard.c.unitdead", opcode: 5, params: []wizParam{pUnit}, cmp: 0, right: 0, rightParam: -1},
	{labelWord: "editor.wizard.c.groupcount", opcode: 1, params: []wizParam{pGroup, {kind: wpNumber, labelWord: "editor.wizard.p.count"}}, cmp: -1, rightParam: 1},
	{labelWord: "editor.wizard.c.playercount", opcode: 8, params: []wizParam{pPlayer, {kind: wpNumber, labelWord: "editor.wizard.p.count"}}, cmp: -1, rightParam: 1},
	{labelWord: "editor.wizard.c.relation", opcode: 10, hintWord: "editor.wizard.c.relation.hint", cmp: -1, rightParam: 2, params: []wizParam{pPlayer, pToPlayer, {kind: wpRelation, labelWord: "editor.wizard.p.relation"}}},
	{labelWord: "editor.wizard.c.always", opcode: scriptConstantOp, cmp: 0, right: 1, rightParam: -1, fixed: [][2]uint32{{scriptInt, 1}}},
}

// wizActions are the ready actions, with the claims that state their
// parameters: give money (TRIG-MONEY-028), win and lose (TRIG-ACT-004), a
// dialog (TRIG-MSG-023; the player sees only a trigger's first, so one per
// trigger), give a unit or group to a player
// (PARTY-JOIN-025), move a unit (AI-SCRIPTGUARD-625), a unit or group off
// and back on the map (TRIG-OFFMAP-041, TRIG-RETURN-042, TRIG-MAPGROUP-043)
// and the drop cell the builder takes (TRIG-DROP-013), and a group order to
// a cell (TRIG-GROUP-005, TRIG-GROUPBOUNDS-136).
var wizActions = []wizItem{
	{labelWord: "editor.wizard.a.gold", opcode: 23, params: []wizParam{pPlayer, {kind: wpNumber, labelWord: "editor.wizard.p.gold", def: 500, inNode: true}}},
	{labelWord: "editor.wizard.a.win", opcode: 4},
	{labelWord: "editor.wizard.a.lose", opcode: 5},
	{labelWord: "editor.wizard.a.dialog", opcode: 2, params: []wizParam{{kind: wpMessage, labelWord: "editor.wizard.p.message", inNode: true}}, onePerTrigger: true},
	{labelWord: "editor.wizard.a.giveunit", opcode: 19, params: []wizParam{pUnit, pPlayer}},
	{labelWord: "editor.wizard.a.givegroup", opcode: 22, params: []wizParam{pGroup, pPlayer}},
	{labelWord: "editor.wizard.a.move", opcode: 27, params: []wizParam{pUnit, pPoint}},
	{labelWord: "editor.wizard.a.unitoff", opcode: 16, params: []wizParam{pUnit}},
	{labelWord: "editor.wizard.a.unitback", opcode: 17, params: []wizParam{pUnit}},
	{labelWord: "editor.wizard.a.groupoff", opcode: 32, params: []wizParam{pGroup}},
	{labelWord: "editor.wizard.a.groupback", opcode: 33, params: []wizParam{pGroup}},
	{labelWord: "editor.wizard.a.drop", opcode: scriptConstantOp, params: []wizParam{pPoint}},
	{labelWord: "editor.wizard.a.order", opcode: scriptGroupCommand, params: []wizParam{
		{kind: wpGroup, labelWord: "editor.wizard.p.group", inNode: true, last: true},
		{kind: wpCommand, labelWord: "editor.wizard.p.command", def: 4, inNode: true}, pPoint}},
	{labelWord: "editor.wizard.a.relation", opcode: 10, hintWord: "editor.wizard.a.relation.hint", params: []wizParam{numbered(pPlayer), numbered(pToPlayer), {kind: wpRelation, labelWord: "editor.wizard.p.relation", inNode: true}}},
}

// scriptConstantOp is the build-time operation: a preset condition, or the
// drop cell action.
const scriptConstantOp = 0x10002

func wizItems(condition bool) []wizItem {
	if condition {
		return wizConditions
	}
	return wizActions
}

// paramTypes are the slot types a parameter kind fills.
func paramTypes(k wizParamKind) []uint32 {
	switch k {
	case wpUnit:
		return []uint32{scriptUnit}
	case wpPlayer:
		return []uint32{scriptPlayer}
	case wpGroup:
		return []uint32{scriptGroup}
	case wpCell:
		return []uint32{scriptX, scriptY}
	case wpBox:
		return []uint32{scriptX, scriptY, scriptX, scriptY}
	case wpCommand:
		return []uint32{scriptConst}
	}
	return []uint32{scriptInt}
}

// hasRadius reports an item whose point has a radius: a square on the map.
func (it wizItem) hasRadius() bool {
	for _, p := range it.params {
		if p.labelWord == wizRadiusWord {
			return true
		}
	}
	return false
}

// wizRadiusWord is the label word of a square's radius parameter.
const wizRadiusWord = "editor.wizard.p.radius"

// itemSlots are the slot types an item's node carries, the slot each
// parameter starts at (-1 for one no slot holds) and the slot its fixed
// values start at.
func (it wizItem) itemSlots() (types [10]uint32, at []int, fixed int) {
	for _, p := range it.params {
		t := p.types()
		switch {
		case !p.inNode:
			at = append(at, -1)
		case p.last:
			at = append(at, len(types)-len(t))
			copy(types[len(types)-len(t):], t)
		default:
			at = append(at, fixed)
			copy(types[fixed:], t)
			fixed += len(t)
		}
	}
	for k, f := range it.fixed {
		types[fixed+k] = f[0]
	}
	return types, at, fixed
}

// wizRow is one row: its item (-1 a pair or action kept as it stands), the
// nodes it names now and, for a kept pair, its comparison; vals are its
// field texts (the item, each parameter, then a chosen comparison) and orig
// those it was read with, so an unchanged row keeps its nodes. origItem is
// the item it was read with, -2 for a row added in the window: only a row
// read as kept offers "Other" back.
type wizRow struct {
	condition        bool
	item, origItem   int
	left, right, act uint32
	cmp              uint32
	vals, orig       []string
}

// wizardState is the wizard's rows, whether a row was added or removed,
// and the fields a drawing on the map fills (-1 none).
type wizardState struct {
	rows       []wizRow
	changed    bool
	draw       int
	drawRadius int
	square     bool
}

// triggerDetailsShown reports that trigger v shows the pair view.
func (e *MapEditor) triggerDetailsShown(v scriptView) bool {
	return v.trigger != nil && e.triggerDetails == v.index+1
}

// wizardOpen is the wizard of the shown trigger popup, nil for none.
func (e *MapEditor) wizardOpen() *wizardState {
	if v, ok := e.scriptPopup(); ok && v.trigger != nil && e.obj.doc == e.doc && e.obj.record == e.selected {
		return e.obj.wiz
	}
	return nil
}

// loadWizard reads trigger v into rows.
func (e *MapEditor) loadWizard(v scriptView) {
	e.obj.wiz = &wizardState{rows: e.wizardRows(v.trigger), draw: -1, drawRadius: -1}
	e.wizardFields(v)
}

// wizardRows are trigger t's pairs and actions as rows, as it stands.
func (e *MapEditor) wizardRows(t *InspectionTrigger) []wizRow {
	var rows []wizRow
	for k := range t.Left {
		if t.Left[k] == 0 && t.Right[k] == 0 {
			continue
		}
		r := wizRow{condition: true, item: -1, left: t.Left[k], right: t.Right[k], cmp: t.Cmp[k]}
		r.item, r.vals = e.readCondition(t.Left[k], t.Right[k], t.Cmp[k])
		rows = append(rows, r)
	}
	for _, id := range t.Acts {
		if id == 0 {
			continue
		}
		r := wizRow{item: -1, act: id}
		r.item, r.vals = e.readAction(id)
		rows = append(rows, r)
	}
	for i := range rows {
		rows[i].origItem = rows[i].item
		rows[i].orig = append([]string(nil), rows[i].vals...)
	}
	return rows
}

// keptPlan is trigger index as it stands, every pair and action kept.
func (e *MapEditor) keptPlan(index int) (WizardTrigger, error) {
	t := &e.doc.Script.Triggers[index]
	return e.rowsPlan(WizardTrigger{Index: index, Name: t.Name, Once: t.Once}, e.wizardRows(t))
}

// constantValue is the value of the constant condition id, false when id
// names no single constant.
func (e *MapEditor) constantValue(id uint32) (int64, bool) {
	n, ok := e.nodeByID(true, id)
	if !ok || n.Opcode != scriptConstantOp || n.Types[0] != scriptInt && n.Types[0] != scriptConst {
		return 0, false
	}
	return int64(int32(n.Values[0])), true
}

// readParams are a node's parameter texts as item it reads them, false when
// its slots are not the item's or its group command is not one the item
// offers.
func (e *MapEditor) readParams(it wizItem, n InspectionScriptNode) ([]string, bool) {
	want, at, fixed := it.itemSlots()
	if n.Opcode != it.opcode || n.Types != want {
		return nil, false
	}
	var out []string
	for k, p := range it.params {
		if at[k] < 0 {
			out = append(out, "")
			continue
		}
		v := n.Values[at[k]:]
		switch p.kind {
		case wpCell:
			out = append(out, fmt.Sprintf("%d,%d", v[0], v[1]))
		case wpBox:
			out = append(out, fmt.Sprintf("%d,%d,%d,%d", v[0], v[1], v[2], v[3]))
		case wpNumber, wpMessage, wpRelation:
			out = append(out, strconv.FormatInt(int64(int32(v[0])), 10))
		case wpCommand:
			if !slices.Contains(e.itemCommands(it), v[0]) {
				return nil, false
			}
			out = append(out, strconv.FormatUint(uint64(v[0]), 10))
		default:
			out = append(out, strconv.FormatUint(uint64(v[0]), 10))
		}
	}
	for k, f := range it.fixed {
		if n.Values[fixed+k] != f[1] {
			return nil, false
		}
	}
	return out, true
}

// itemCommands are the group commands whose signature, where a claim states
// one, is item it's slots: for a target cell, 2, 4, 5 and 14
// (TRIG-GROUPBOUNDS-136).
func (e *MapEditor) itemCommands(it wizItem) []uint32 {
	want, _, _ := it.itemSlots()
	var out []uint32
	for _, c := range e.doc.Script.Catalog.GroupCommands {
		sig, ok := e.doc.Script.Catalog.signature(false, scriptGroupCommand, c)
		var got [10]uint32
		for s := range sig {
			got[s] = sig[s].Type
		}
		if ok && got == want {
			out = append(out, c)
		}
	}
	return out
}

// readCondition is the item and field texts of the pair left cmp right, -1
// when no item reads it.
func (e *MapEditor) readCondition(left, right, cmp uint32) (int, []string) {
	n, ok := e.nodeByID(true, left)
	v, constant := e.constantValue(right)
	if !ok || !constant {
		return -1, []string{"-1"}
	}
	for i, it := range wizConditions {
		params, ok := e.readParams(it, n)
		if !ok || it.cmp >= 0 && uint32(it.cmp) != cmp || it.cmp < 0 && cmp > 5 {
			continue
		}
		if it.rightParam < 0 {
			if v != it.right {
				continue
			}
		} else {
			params[it.rightParam] = strconv.FormatInt(v, 10)
		}
		out := append([]string{strconv.Itoa(i)}, params...)
		if it.cmp < 0 {
			out = append(out, strconv.FormatUint(uint64(cmp), 10))
		}
		return i, out
	}
	return -1, []string{"-1"}
}

// readAction is the item and field texts of action id, -1 when no item
// reads it.
func (e *MapEditor) readAction(id uint32) (int, []string) {
	n, ok := e.nodeByID(false, id)
	if ok {
		for i, it := range wizActions {
			if params, ok := e.readParams(it, n); ok {
				return i, append([]string{strconv.Itoa(i)}, params...)
			}
		}
	}
	return -1, []string{"-1"}
}

// wizardDefaults are a fresh row's field texts for item i.
func (e *MapEditor) wizardDefaults(condition bool, i int) []string {
	it := wizItems(condition)[i]
	out := []string{strconv.Itoa(i)}
	mid := image.Pt(min(e.doc.Width, 256)/2, min(e.doc.Height, 256)/2)
	for _, p := range it.params {
		switch p.kind {
		case wpUnit:
			out = append(out, strconv.FormatInt(e.defaultUnit(), 10))
		case wpGroup:
			out = append(out, strconv.FormatInt(e.firstTarget(scriptGroup, 0), 10))
		case wpCell:
			out = append(out, fmt.Sprintf("%d,%d", mid.X, mid.Y))
		case wpBox:
			out = append(out, fmt.Sprintf("%d,%d,%d,%d", max(0, mid.X-2), max(0, mid.Y-2), mid.X+2, mid.Y+2))
		default:
			out = append(out, strconv.FormatInt(p.def, 10))
		}
	}
	if it.cmp < 0 {
		out = append(out, "0")
	}
	return out
}

// multiplayer reports a map whose player capacity is above one: a hero
// number names no unit there (TRIG-HEROORD-075).
func (e *MapEditor) multiplayer() bool {
	p, ok := e.mapPanel()
	return ok && p.MapFacts().Mode > 1
}

// defaultUnit is hero 1, or on a multiplayer map the first placed unit a
// script can name.
func (e *MapEditor) defaultUnit() int64 {
	if e.multiplayer() {
		for _, t := range e.doc.Script.Targets[scriptUnit] {
			if t.Record >= 0 {
				return int64(t.Value)
			}
		}
	}
	return 10001
}

// firstTarget is the first value parameter kind can name, else def.
func (e *MapEditor) firstTarget(kind uint32, def int64) int64 {
	for _, t := range e.doc.Script.Targets[kind] {
		if !t.NewMessage {
			return int64(t.Value)
		}
	}
	return def
}

// wizardKey is the key of field part of row r: "c0.item", "a1.p0", "c2.cmp".
func (w *wizardState) wizardKey(r int, part string) string {
	n := 0
	for k := 0; k < r; k++ {
		if w.rows[k].condition == w.rows[r].condition {
			n++
		}
	}
	family := "a"
	if w.rows[r].condition {
		family = "c"
	}
	return fmt.Sprintf("%s%d.%s", family, n, part)
}

// pullWizard copies the field texts into the rows.
func (e *MapEditor) pullWizard() {
	o := &e.obj
	w := o.wiz
	at := map[string]int{}
	for i, f := range o.fields {
		at[f.Key] = i
	}
	for r := range w.rows {
		for k := range w.rows[r].vals {
			if i, ok := at[w.wizardKey(r, wizPart(w.rows[r], k))]; ok && i < len(o.buf) {
				w.rows[r].vals[k] = o.buf[i]
			}
		}
	}
}

// wizPart is the key part of field text k of row r.
func wizPart(r wizRow, k int) string {
	switch {
	case k == 0:
		return "item"
	case r.item >= 0 && k > len(wizItems(r.condition)[r.item].params):
		return "cmp"
	}
	return fmt.Sprintf("p%d", k-1)
}

// wizardFields builds the fields from the rows: the name, the once switch,
// then each row's item, parameters and comparison. A field's value is the
// text the row was read with, so only an edit makes it dirty.
func (e *MapEditor) wizardFields(v scriptView) {
	o := &e.obj
	w := o.wiz
	t := v.trigger
	o.fields = []InspectionField{{Key: "name", Label: e.word("editor.trigger.name"), Text: true, TextValue: t.Name},
		{Key: "once", Label: e.word("editor.trigger.once"), Value: int64(t.Once), Choices: []InspectionChoice{{1, e.word("editor.trigger.once.yes")}, {0, e.word("editor.trigger.once.no")}}}}
	keep := []string{o.bufAt(0, t.Name), o.bufAt(1, strconv.FormatUint(uint64(t.Once), 10))}
	o.buf = o.buf[:0]
	o.buf = append(o.buf, keep...)
	for r, row := range w.rows {
		items := wizItems(row.condition)
		var choices []InspectionChoice
		var shut []int64
		for i, it := range items {
			label := e.word(it.labelWord)
			if w.shut(r, i) {
				label = fmt.Sprintf(e.word("editor.wizard.onlyone"), label)
				shut = append(shut, int64(i))
			}
			choices = append(choices, InspectionChoice{int64(i), label})
		}
		if row.origItem == -1 {
			choices = append(choices, InspectionChoice{-1, e.keptText(row)})
		}
		orig := row.orig
		if row.item != row.origItem {
			orig = nil
		}
		text := func(k int) string {
			if k < len(orig) {
				return orig[k]
			}
			return row.vals[k]
		}
		label := e.word("editor.wizard.condition")
		if !row.condition {
			label = e.word("editor.wizard.action")
		}
		o.fields = append(o.fields, InspectionField{Key: w.wizardKey(r, "item"), Label: label, Value: int64(row.origItem), Choices: choices, Shut: shut})
		o.buf = append(o.buf, row.vals[0])
		if row.item < 0 {
			continue
		}
		it := items[row.item]
		for k, p := range it.params {
			f := InspectionField{Key: w.wizardKey(r, fmt.Sprintf("p%d", k)), Label: e.word(p.labelWord)}
			value := text(k + 1)
			switch p.kind {
			case wpUnit, wpPlayer, wpGroup, wpMessage, wpRelation:
				n, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
				f.Value = n
				f.Choices = e.wizardChoices(p.kind, n)
			case wpCommand:
				n, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
				f.Value = n
				for _, c := range e.itemCommands(it) {
					f.Choices = append(f.Choices, InspectionChoice{int64(c), e.commandName(c)})
				}
			default:
				f.Text, f.TextValue = true, value
			}
			o.fields = append(o.fields, f)
			o.buf = append(o.buf, row.vals[k+1])
		}
		if it.cmp < 0 {
			var cmps []InspectionChoice
			for c := int64(0); c <= 5; c++ {
				cmps = append(cmps, InspectionChoice{c, e.cmpWord(c)})
			}
			n, _ := strconv.ParseInt(text(len(it.params)+1), 10, 64)
			o.fields = append(o.fields, InspectionField{Key: w.wizardKey(r, "cmp"), Label: e.word("editor.trigger.comparison"), Value: n, Choices: cmps})
			o.buf = append(o.buf, row.vals[len(it.params)+1])
		}
	}
}

// shut reports that row r may not take item i: the item is one per trigger
// and another row of the trigger has it.
func (w *wizardState) shut(r, i int) bool {
	row := w.rows[r]
	if i < 0 || i >= len(wizItems(row.condition)) || !wizItems(row.condition)[i].onePerTrigger || row.item == i {
		return false
	}
	for k, other := range w.rows {
		if k != r && other.condition == row.condition && other.item == i {
			return true
		}
	}
	return false
}

// bufAt is field i's text, or def when there is none yet.
func (o *objectEdit) bufAt(i int, def string) string {
	if i < len(o.buf) && i < len(o.fields) {
		return o.buf[i]
	}
	return def
}

// wizardChoices are the values a row parameter of kind can name: for a
// unit the placed units a script can name and heroes 1 to 8, with value
// kept when it is none of them.
func (e *MapEditor) wizardChoices(kind wizParamKind, value int64) []InspectionChoice {
	switch kind {
	case wpMessage:
		return e.parameterChoices(ScriptMessageParameter, value)
	}
	out := e.wizardTargets(kind)
	if kind == wpUnit {
		out = slices.DeleteFunc(out, func(c InspectionChoice) bool { return c.Value >= 10001 && c.Value <= 11000 })
		for h := int64(1); h <= 8; h++ {
			out = append(out, InspectionChoice{10000 + h, e.heroLabel(h)})
		}
	}
	for _, c := range out {
		if c.Value == value {
			return out
		}
	}
	label := strconv.FormatInt(value, 10)
	if kind == wpUnit && value > 10000 {
		label = e.heroLabel(value - 10000)
	}
	return append(out, InspectionChoice{value, label})
}

// wizardTargets are the values the wizard lists for a row parameter of kind:
// for a unit the placed units a script can name and the hero band, for a
// player or a group the script's targets of that kind, for a relation its
// three words.
func (e *MapEditor) wizardTargets(kind wizParamKind) []InspectionChoice {
	var out []InspectionChoice
	add := func(v int64, label string) { out = append(out, InspectionChoice{v, label}) }
	switch kind {
	case wpUnit:
		for _, t := range e.doc.Script.Targets[scriptUnit] {
			if t.Record >= 0 || t.Value >= 10001 && t.Value <= 11000 {
				add(int64(t.Value), t.ChoiceText())
			}
		}
	case wpPlayer:
		for _, t := range e.doc.Script.Targets[scriptPlayer] {
			add(int64(t.Value), t.ChoiceText())
		}
	case wpGroup:
		for _, t := range e.doc.Script.Targets[scriptGroup] {
			add(int64(t.Value), t.ChoiceText())
		}
	case wpRelation:
		for v, word := range wizardRelationWords {
			add(int64(v), e.word(word))
		}
	}
	return out
}

// wizardTargetAllowed reports that a unit or group parameter names value: a
// value wizardTargets lists.
func (e *MapEditor) wizardTargetAllowed(kind wizParamKind, value int64) bool {
	for _, c := range e.wizardTargets(kind) {
		if c.Value == value {
			return true
		}
	}
	return false
}

// keptText is a kept row in words: "Other: #3 Distance ... is at most #4
// Number", or the action's title.
func (e *MapEditor) keptText(r wizRow) string {
	if !r.condition {
		title, _ := e.slotText(false, strconv.FormatUint(uint64(r.act), 10))
		return fmt.Sprintf(e.word("editor.wizard.other"), title)
	}
	left, _ := e.slotText(true, strconv.FormatUint(uint64(r.left), 10))
	right, _ := e.slotText(true, strconv.FormatUint(uint64(r.right), 10))
	return fmt.Sprintf(e.word("editor.wizard.other"), left+" "+e.cmpWord(int64(r.cmp))+" "+right)
}

// rewizard follows a changed item: the row takes the item's fields, a row
// back on the item it was read with its texts.
func (e *MapEditor) rewizard() {
	o := &e.obj
	w := o.wiz
	v, ok := e.scriptPopup()
	if !ok || v.trigger == nil || w == nil {
		return
	}
	e.pullWizard()
	rebuilt := false
	for r := range w.rows {
		row := &w.rows[r]
		item, err := strconv.Atoi(strings.TrimSpace(row.vals[0]))
		if err != nil || item < -1 || item >= len(wizItems(row.condition)) || item == -1 && row.origItem != -1 {
			item = row.item
		}
		if item == row.item {
			continue
		}
		rebuilt = true
		if w.shut(r, item) {
			row.vals[0] = strconv.Itoa(row.item)
			continue
		}
		row.item = item
		switch {
		case item == row.origItem:
			row.vals = append([]string(nil), row.orig...)
		default:
			row.vals = e.wizardDefaults(row.condition, item)
		}
	}
	if rebuilt {
		e.wizardFields(v)
		o.open = -1
	}
}

// wizardDirty reports a row added, removed or given another item.
func (w *wizardState) dirty() bool {
	if w.changed {
		return true
	}
	for _, r := range w.rows {
		if r.item != r.origItem {
			return true
		}
	}
	return false
}

// Row actions of the wizard.
const (
	actRemoveRow = "removerow"
	actDraw      = "draw"
	actPickParam = "pickparam" // a unit, structure or group clicked on the map
	actPickArea  = "pickarea"  // a trigger area clicked on the map
	actConfigure = "configure"
)

const actionGapRows = 1

func actionGap(add func(popupRow, int, ...string), shown int) {
	if shown == 0 {
		return
	}
	for range actionGapRows {
		add(popupRow{Kind: popupText}, -1)
	}
}

// wizardSpec is the wizard as a popup: the name, the once switch, When, And
// if and Do with their rows, and the buttons Apply, Cancel, New, Delete and
// Details.
func (e *MapEditor) wizardSpec(v scriptView, s *popupSpec, add func(popupRow, int, ...string), text func(string, string)) {
	o := &e.obj
	w := o.wiz
	s.Title = fmt.Sprintf(e.word("editor.trigger.title"), v.index+1)
	add(popupRow{Kind: popupEdit, Label: o.fields[trigName].Label, Value: e.editorHostText(o.buf[trigName])}, trigName)
	once := e.word("editor.trigger.once.no")
	if strings.TrimSpace(o.buf[trigOnce]) != "0" {
		once = e.word("editor.trigger.once.yes")
	}
	add(popupRow{Kind: popupSwitch, Label: o.fields[trigOnce].Label, Value: once}, trigOnce)
	at := map[string]int{}
	for i, f := range o.fields {
		at[f.Key] = i
	}
	conds, acts, heroes, drops := 0, 0, false, 0
	row := func(r int) {
		wr := w.rows[r]
		f := at[w.wizardKey(r, "item")]
		label := o.fields[f].Label
		ink := editorCondition
		if !wr.condition {
			ink = editorGold
		}
		acts := []string{actRemoveRow}
		if wr.item >= 0 && slices.ContainsFunc(wizItems(wr.condition)[wr.item].params, func(p wizParam) bool { return p.kind == wpMessage }) {
			acts = []string{actConfigure, actRemoveRow}
		}
		add(popupRow{Kind: popupChoice, Label: label, Value: choiceText(o.fields[f], o.buf[f]), Ink: ink}, f, acts...)
		if wr.item < 0 {
			if wr.condition {
				_, lp := e.slotText(true, strconv.FormatUint(uint64(wr.left), 10))
				_, rp := e.slotText(true, strconv.FormatUint(uint64(wr.right), 10))
				for _, p := range []string{lp, rp} {
					if p != "" {
						text("    "+p, "")
					}
				}
			} else if _, p := e.slotText(false, strconv.FormatUint(uint64(wr.act), 10)); p != "" {
				text("    "+p, "")
			}
			text("    "+e.word("editor.wizard.otherhint"), "")
			return
		}
		it := wizItems(wr.condition)[wr.item]
		if !wr.condition && it.opcode == scriptConstantOp {
			drops++
		}
		for k, p := range it.params {
			if p.kind == wpMessage {
				continue // the item row's Configure edits the dialog
			}
			i := at[w.wizardKey(r, fmt.Sprintf("p%d", k))]
			pr := popupRow{Kind: popupEdit, Label: "  " + o.fields[i].Label, Value: o.buf[i]}
			var buttons []string
			switch p.kind {
			case wpUnit, wpPlayer, wpGroup, wpCommand, wpRelation:
				pr.Kind, pr.Value = popupChoice, choiceText(o.fields[i], o.buf[i])
				if p.kind == wpUnit || p.kind == wpGroup {
					buttons = []string{actPickParam}
				}
				if p.kind == wpUnit {
					if n, err := strconv.ParseInt(strings.TrimSpace(o.buf[i]), 10, 64); err == nil && n > 10000 {
						heroes = true
					}
				}
			case wpCell:
				buttons = []string{actPick}
				if wr.condition && it.hasRadius() {
					buttons = []string{actDraw, actPickArea}
				}
			case wpBox:
				buttons = []string{actDraw, actPickArea}
			}
			add(pr, i, buttons...)
		}
		if it.cmp < 0 {
			i := at[w.wizardKey(r, "cmp")]
			add(popupRow{Kind: popupChoice, Label: "  " + o.fields[i].Label, Value: choiceText(o.fields[i], o.buf[i])}, i)
		}
		if it.hintWord != "" {
			text("    "+e.word(it.hintWord), "")
		}
	}
	for r, wr := range w.rows {
		if !wr.condition {
			continue
		}
		switch conds {
		case 0:
			add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.when")}, -1)
		case 1:
			add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.andif")}, -1)
		}
		conds++
		row(r)
	}
	if conds == 0 {
		add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.when")}, -1)
	}
	if conds < 3 {
		add(popupRow{Kind: popupText}, -1, actAddPair)
	}
	add(popupRow{Kind: popupHeading, Label: e.word("editor.wizard.do")}, -1)
	for r, wr := range w.rows {
		if !wr.condition {
			actionGap(add, acts)
			acts++
			row(r)
		}
	}
	if acts < 4 {
		add(popupRow{Kind: popupText}, -1, actAddAct)
	}
	if conds == 0 && acts > drops {
		text(e.word("editor.wizard.nocondition"), "warning")
	}
	if heroes && e.multiplayer() {
		text(e.word("editor.wizard.multiplayer"), "warning")
	}
	s.Buttons = []string{e.word("editor.button.apply"), e.word("editor.button.cancel"), e.word("editor.trigger.new"), e.word("editor.trigger.delete"), e.word("editor.wizard.details")}
	s.Disabled = []bool{!e.objectDirty(), false, false, false, false}
	fitButtons(s)
}

// wizardRowButton is a press on button k of a wizard row: remove the row,
// add one, draw its area, or pick its cell or unit on the map.
func (e *MapEditor) wizardRowButton(act string, f int, hasField bool) {
	o := &e.obj
	w := o.wiz
	v, _ := e.scriptPopup()
	e.pullWizard()
	switch act {
	case actRemoveRow:
		key := o.fields[f].Key
		for r := range w.rows {
			if w.wizardKey(r, "item") == key {
				w.rows = append(w.rows[:r], w.rows[r+1:]...)
				w.changed = true
				break
			}
		}
		o.focus = -1
		e.wizardFields(v)
	case actAddPair, actAddAct:
		condition := act == actAddPair
		r := wizRow{condition: condition, origItem: -2}
		r.item, r.vals = 0, e.wizardDefaults(condition, 0)
		// A condition goes after the last condition, an action last.
		at := len(w.rows)
		if condition {
			at = 0
			for k, x := range w.rows {
				if x.condition {
					at = k + 1
				}
			}
		}
		w.rows = append(w.rows[:at], append([]wizRow{r}, w.rows[at:]...)...)
		w.changed = true
		e.wizardFields(v)
	case actDraw:
		if !hasField {
			return
		}
		w.draw, w.drawRadius, w.square = f, -1, e.wizardSquare(f)
		if w.square {
			w.drawRadius = e.wizardRadiusField(f)
		}
		o.focus = f
		e.leave(leaveDrag)
		e.armPick(pickArea)
	case actConfigure:
		key := o.fields[f].Key
		for r := range w.rows {
			if w.wizardKey(r, "item") != key || w.rows[r].item < 0 {
				continue
			}
			for k, p := range wizItems(w.rows[r].condition)[w.rows[r].item].params {
				if p.kind == wpMessage {
					e.configureDialog(slices.IndexFunc(o.fields, func(x InspectionField) bool { return x.Key == w.wizardKey(r, fmt.Sprintf("p%d", k)) }))
					return
				}
			}
		}
	case actPickParam, actPickArea:
		if hasField {
			e.armParamPick(f, act == actPickArea)
		}
	case actPick:
		if hasField {
			o.focus, o.pickField = f, f
			e.leave(leaveDrag)
			e.armPick(pickScriptCell)
		}
	}
}

// wizardRowOf is the row and parameter index field f edits, -1 for none.
func (e *MapEditor) wizardRowOf(f int) (int, int) {
	w := e.obj.wiz
	key := e.obj.fields[f].Key
	for r := range w.rows {
		for k := range w.rows[r].vals {
			if w.wizardKey(r, wizPart(w.rows[r], k)) == key {
				return r, k - 1
			}
		}
	}
	return -1, -1
}

// wizardSquare reports that field f is the point of a point-and-radius
// condition: a drawing gives it its radius too.
func (e *MapEditor) wizardSquare(f int) bool {
	r, p := e.wizardRowOf(f)
	if r < 0 || p < 0 {
		return false
	}
	row := e.obj.wiz.rows[r]
	it := wizItems(row.condition)[row.item]
	return it.params[p].kind == wpCell && e.wizardRadiusField(f) >= 0
}

// wizardRadiusField is the radius field of the row field f is in, -1 for
// none.
func (e *MapEditor) wizardRadiusField(f int) int {
	r, _ := e.wizardRowOf(f)
	if r < 0 {
		return -1
	}
	w := e.obj.wiz
	row := w.rows[r]
	if !row.condition || row.item < 0 {
		return -1
	}
	for k, p := range wizItems(true)[row.item].params {
		if p.labelWord == wizRadiusWord {
			key := w.wizardKey(r, fmt.Sprintf("p%d", k))
			for i, fl := range e.obj.fields {
				if fl.Key == key {
					return i
				}
			}
		}
	}
	return -1
}

// wizardDrawing reports that the next drag on the map draws a wizard area.
func (e *MapEditor) wizardDrawing() (*wizardState, bool) {
	w := e.obj.wiz
	if e.pick != pickArea || w == nil || w.draw < 0 || e.obj.doc != e.doc || e.obj.record != e.selected || w.draw >= len(e.obj.buf) {
		return nil, false
	}
	return w, true
}

// wizardAreaDrawn writes the drawn box, or the point and radius, into the
// row's fields and returns to the wizard.
func (e *MapEditor) wizardAreaDrawn(h *areaHandle) {
	w, ok := e.wizardDrawing()
	e.leave(leavePick)
	if !ok {
		return
	}
	o := &e.obj
	if h.kind == handleDrawSquare {
		o.buf[w.draw] = fmt.Sprintf("%d,%d", h.from[0], h.from[1])
		if w.drawRadius >= 0 {
			o.buf[w.drawRadius] = strconv.Itoa(h.radius)
		}
	} else {
		o.buf[w.draw] = fmt.Sprintf("%d,%d,%d,%d", h.box[0], h.box[1], h.box[2], h.box[3])
	}
	w.draw, w.drawRadius = -1, -1
}

// wizardPlan is the trigger the rows say, for the host to write.
func (e *MapEditor) wizardPlan(v scriptView) (WizardTrigger, error) {
	o := &e.obj
	w := o.wiz
	e.pullWizard()
	once, _ := strconv.ParseInt(strings.TrimSpace(o.buf[trigOnce]), 10, 64)
	return e.rowsPlan(WizardTrigger{Index: v.index, Name: o.buf[trigName], Once: uint32(once)}, w.rows)
}

// rowsPlan is trigger t with the pairs and actions rows say.
func (e *MapEditor) rowsPlan(t WizardTrigger, rows []wizRow) (WizardTrigger, error) {
	for r, row := range rows {
		unchanged := row.item == row.origItem && len(row.vals) == len(row.orig)
		for k := range row.vals {
			unchanged = unchanged && strings.TrimSpace(row.vals[k]) == strings.TrimSpace(row.orig[k])
		}
		if row.item < 0 || unchanged {
			if row.condition {
				t.Pairs = append(t.Pairs, WizardPair{Left: WizardNode{ID: row.left, Keep: true}, Right: WizardNode{ID: row.right, Keep: true}, Cmp: row.cmp})
			} else {
				t.Actions = append(t.Actions, WizardNode{ID: row.act, Keep: true})
			}
			continue
		}
		node, right, cmp, err := e.wizardNode(r, row)
		if err != nil {
			return t, err
		}
		if row.condition {
			p := WizardPair{Left: node, Cmp: cmp, Right: WizardNode{ID: row.right, Opcode: scriptConstantOp}}
			p.Left.ID = row.left
			p.Right.Types[0], p.Right.Values[0] = scriptInt, uint32(int32(right))
			t.Pairs = append(t.Pairs, p)
		} else {
			node.ID = row.act
			t.Actions = append(t.Actions, node)
		}
	}
	if len(t.Pairs) > 3 || len(t.Actions) > 4 {
		return t, fmt.Errorf("%s", e.word("editor.wizard.full"))
	}
	return t, nil
}

// wizardNode is the node row r says, the constant it is compared with and
// the comparison.
func (e *MapEditor) wizardNode(r int, row wizRow) (WizardNode, int64, uint32, error) {
	it := wizItems(row.condition)[row.item]
	n := WizardNode{Opcode: it.opcode}
	label := func(k int) string { return e.word(it.params[k].labelWord) }
	_, at, fixed := it.itemSlots()
	var right int64
	for k, p := range it.params {
		text := strings.TrimSpace(row.vals[k+1])
		var nums []int64
		for _, part := range strings.Split(text, ",") {
			x, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil {
				nums = nil
				break
			}
			if x < math.MinInt32 || x > math.MaxInt32 {
				return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.wizard.range"))
			}
			nums = append(nums, x)
		}
		want := len(p.types())
		if len(nums) != want {
			switch p.kind {
			case wpCell:
				return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.node.cell"))
			case wpBox:
				return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.wizard.badbox"))
			}
			return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.object.number"))
		}
		switch p.kind {
		case wpCell, wpBox:
			for c, x := range nums {
				limit := e.doc.Width
				if c%2 == 1 {
					limit = e.doc.Height
				}
				if x < 0 || x >= int64(min(limit, 256)) {
					return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.wizard.outside"))
				}
			}
			if p.kind == wpBox && (nums[0] > nums[2] || nums[1] > nums[3]) {
				return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.wizard.badbox"))
			}
		case wpUnit, wpPlayer:
			if nums[0] <= 0 {
				return n, 0, 0, fmt.Errorf("%s: %s", label(k), e.word("editor.node.unset"))
			}
		}
		if !p.inNode {
			if k == it.rightParam {
				right = nums[0]
			}
			continue
		}
		for c, t := range p.types() {
			n.Types[at[k]+c], n.Values[at[k]+c] = t, uint32(int32(nums[c]))
		}
	}
	for k, f := range it.fixed {
		n.Types[fixed+k], n.Values[fixed+k] = f[0], f[1]
	}
	if !row.condition {
		return n, 0, 0, nil
	}
	cmp := uint32(max(it.cmp, 0))
	if it.rightParam < 0 {
		right = it.right
	}
	if it.cmp < 0 {
		c, err := strconv.ParseInt(strings.TrimSpace(row.vals[len(it.params)+1]), 10, 64)
		if err != nil || c < 0 || c > 5 {
			return n, 0, 0, fmt.Errorf("%s: %s", e.word("editor.trigger.comparison"), e.word("editor.object.number"))
		}
		cmp = uint32(c)
	}
	return n, right, cmp, nil
}

// applyWizard writes the rows through the host as one edit.
func (e *MapEditor) applyWizard(v scriptView) {
	o := &e.obj
	if _, ok := e.doc.Edits.(InspectionTriggerWizard); !ok {
		return
	}
	t, err := e.wizardPlan(v)
	var ch editorChange
	if err == nil {
		ch, err = e.commandWriteTrigger(t)
	}
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	o.open, o.note = -1, ""
	if ch.doc == nil {
		e.obj.doc = nil // nothing written: read the trigger again
	}
}

// wizardButton is a press on the wizard's button row.
func (e *MapEditor) wizardButton(v scriptView, n int) {
	switch n {
	case 0:
		e.applyWizard(v)
	case 1:
		e.cancelPopup()
	case 2:
		e.newTrigger()
	case 3:
		e.deleteTrigger(v)
	case 4:
		if e.objectDirty() {
			e.obj.note = e.word("editor.trigger.applyfirst")
			return
		}
		e.triggerDetails = v.index + 1
		e.obj.doc = nil
	}
}

// wizardAreaLines outline the areas the open wizard's rows draw, in gold.
func (e *MapEditor) wizardAreaLines() []editorLine {
	w := e.wizardOpen()
	if w == nil || e.objectHidden {
		return nil
	}
	e.pullWizard()
	var out []editorLine
	bounds := image.Rect(0, 0, e.doc.Width, e.doc.Height)
	for _, row := range w.rows {
		if !row.condition || row.item < 0 {
			continue
		}
		it := wizConditions[row.item]
		var r image.Rectangle
		var centre image.Point
		radius, hasCell := -1, false
		for k, p := range it.params {
			var nums []int
			for _, part := range strings.Split(row.vals[k+1], ",") {
				x, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil {
					nums = nil
					break
				}
				nums = append(nums, x)
			}
			switch {
			case p.kind == wpBox && len(nums) == 4:
				r = image.Rect(nums[0], nums[1], nums[2]+1, nums[3]+1)
			case p.kind == wpCell && len(nums) == 2:
				centre, hasCell = image.Pt(nums[0], nums[1]), true
				r = image.Rect(centre.X, centre.Y, centre.X+1, centre.Y+1)
			case p.labelWord == wizRadiusWord && len(nums) == 1:
				radius = nums[0]
				if it.cmp == 3 || it.cmp == 4 {
					radius--
				}
			}
		}
		if hasCell && radius >= 0 && radius <= maxAreaRadius { // a radius around two units draws nothing
			r = image.Rect(centre.X-radius, centre.Y-radius, centre.X+radius+1, centre.Y+radius+1)
		}
		if r = r.Intersect(bounds); !r.Empty() {
			out = append(out, e.areaOutline(r, editorGold)...)
		}
	}
	return out
}

// WizardShown reports that the selected trigger's popup is the wizard.
func (e *MapEditor) WizardShown() bool {
	e.loadObject()
	return e.wizardOpen() != nil
}

// WizardField is the field index of wizard field key ("c0.item", "a1.p0",
// "c0.cmp"), -1 when the wizard shows none.
func (e *MapEditor) WizardField(key string) int {
	if !e.WizardShown() {
		return -1
	}
	for i, f := range e.obj.fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

// WizardRowCounts are the shown wizard's rows and those of them it keeps as
// they stand: a trigger shows fully as rows when kept is 0.
func (e *MapEditor) WizardRowCounts() (rows, kept int) {
	if !e.WizardShown() {
		return 0, 0
	}
	for _, r := range e.obj.wiz.rows {
		rows++
		if r.item < 0 {
			kept++
		}
	}
	return rows, kept
}

// ShowTriggerDetails switches the shown trigger popup between the wizard
// and the pair view, as the Details button and its way back do.
func (e *MapEditor) ShowTriggerDetails(on bool) {
	if v, ok := e.scriptPopup(); ok && v.trigger != nil {
		e.triggerDetails = 0
		if on {
			e.triggerDetails = v.index + 1
		}
		e.obj.doc = nil
	}
}
