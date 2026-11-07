package ui

import (
	"fmt"
	"image"
	"math"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/scriptprofile"
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
	wpCommand                         // a choice of the group commands whose slots are the item's
	wpRelation
	wpVariable  // a choice of the map's variables and a new one
	wpItem      // a choice of the item kinds
	wpStructure // a choice of structures, or a click on one
	wpSpell     // a choice of the spells the slot takes
	wpProperty  // a choice of the unit properties a selector names
)

// wizParam is one parameter of an item: its kind, its label's word, its
// default number, and whether a node slot holds it (else the constant the
// check is compared with does). A last parameter stands in the node's last
// slots, as a group command's target group does (ALM-TRIG-045); at, when
// set, is the slot after the one it starts at; the others fill the slots
// from the first free one on. A right parameter stands in the pair's right
// node, a second node of the item's check.
type wizParam struct {
	kind      wizParamKind
	labelWord string
	def       int64
	inNode    bool
	last      bool
	right     bool
	at        int
	slot      uint32
}

// wizItem is one entry of the ready list. A condition is its check compared
// with a constant: by cmp with right, or, with chooseCmp, by the comparison
// the row chooses, cmp its first; rightParam names the parameter that holds
// the constant, -1 for right, and rightType is the constant's slot type, 0
// for a number. also are other comparisons and constants that state the same
// for a check that writes only 0 or 1. An item with right parameters
// compares two nodes of its check. holds marks the item a pair of one node
// with itself by ==, >= or <= reads as. fixed are node slots after the
// parameters', as type and value. onePerTrigger is an item a trigger holds at
// most once: another row shows it shut. subject is its list heading. A
// variant is the nearest earlier item with its opcode under another command:
// the list shows the two as one entry; the row's command chooses.
type wizItem struct {
	labelWord     string
	subject       int
	variant       bool
	opcode        uint32
	params        []wizParam
	cmp           int
	chooseCmp     bool
	right         int64
	rightParam    int
	rightType     uint32
	also          [][2]int64
	holds         bool
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

	pItem      = wizParam{kind: wpItem, labelWord: "editor.wizard.p.item", def: 1, inNode: true}
	pStructure = wizParam{kind: wpStructure, labelWord: "editor.wizard.p.structure", inNode: true}
	pVariable  = wizParam{kind: wpVariable, labelWord: "editor.wizard.p.variable", inNode: true}
	pSpell     = wizParam{kind: wpSpell, labelWord: "editor.wizard.p.spell", def: 1, inNode: true}
	pPower     = wizParam{kind: wpNumber, labelWord: "editor.wizard.p.power", def: 1, inNode: true}
	pTime      = wizParam{kind: wpNumber, labelWord: "editor.wizard.p.time", def: 100, inNode: true}
	pFrom      = wizParam{kind: wpCell, labelWord: "editor.wizard.p.from", inNode: true}
	pTo        = wizParam{kind: wpCell, labelWord: "editor.wizard.p.to", inNode: true}
	pRadius    = wizParam{kind: wpNumber, labelWord: "editor.wizard.p.radius", def: 3}
	pCompared  = wizParam{kind: wpNumber, labelWord: "editor.wizard.p.count"}
	pRange     = wizParam{kind: wpNumber, labelWord: "editor.wizard.p.range", def: 3, inNode: true}
	pCommand   = wizParam{kind: wpCommand, labelWord: "editor.wizard.p.command", inNode: true}
	pLastGroup = wizParam{kind: wpGroup, labelWord: "editor.wizard.p.group", inNode: true, last: true}
)

// command is the command parameter with default c.
func command(c int64) wizParam {
	p := pCommand
	p.def = c
	return p
}

// onRight is p standing in the pair's right node.
func onRight(p wizParam) wizParam {
	p.right = true
	return p
}

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

// The subjects the item list groups its items under, in list order, and
// their heading words.
const (
	subjectMission = iota
	subjectUnits
	subjectPlayers
	subjectGroups
	subjectCells
	subjectItems
	subjectStructures
	subjectSpells
	subjectVariables
)

var wizSubjectWords = []string{"editor.wizard.s.mission", "editor.wizard.s.units", "editor.wizard.s.players", "editor.wizard.s.groups", "editor.wizard.s.cells", "editor.wizard.s.items", "editor.wizard.s.structures", "editor.wizard.s.spells", "editor.wizard.s.variables"}

// wizConditions are the ready conditions. Check 2 is a unit in a box, 5 a
// unit alive, each writing 0 or 1 (TRIG-CHECKGEOM-130); 3 a unit within a
// radius of a cell; 6 the distance between two units, 255 when either is
// dead, and 7 from a unit to a cell (TRIG-DIST-014); 15 the distance from a
// cell to a player's nearest unit (TRIG-NEAREST-029); 1 a group's living
// count (TRIG-COUNT-015), 8 a player's (TRIG-LISTREAD-131); a constant is
// preset to its value and 19 copies a variable (TRIG-COND-003); 4 with 6 a
// unit's health (TRIG-CHECK-051); 9 the unit a unit pursues
// (TRIG-TARGETID-032); 18 a loss when its unit dies, written into no slot
// (MISSION-VIP-004); 14 a sack on a cell (TRIG-SACK-022) and 17 an item a
// unit holds (TRIG-ITEMTEST-040), each 0 or 1; 21 a structure's value
// (TRIG-CHECK-053); 16 the distance from a unit with an item (TRIG-CHECK-052).
var wizConditions = []wizItem{
	{labelWord: "editor.wizard.c.heroarea", subject: subjectUnits, opcode: 2, params: []wizParam{pUnit, pArea}, cmp: 0, right: 1, rightParam: -1, also: [][2]int64{{1, 0}}},
	{labelWord: "editor.wizard.c.playernear", subject: subjectPlayers, opcode: 15, params: []wizParam{pPlayer, pPoint, pRadius}, cmp: 5, chooseCmp: true, rightParam: 2},
	{labelWord: "editor.wizard.c.unitnear", subject: subjectUnits, opcode: 3, params: []wizParam{pUnit, pPoint, {kind: wpNumber, labelWord: "editor.wizard.p.radius", def: 3, inNode: true}}, cmp: 0, right: 1, rightParam: -1},
	{labelWord: "editor.wizard.c.unitnearunit", subject: subjectUnits, opcode: 6, hintWord: "editor.wizard.c.unitnearunit.hint", params: []wizParam{pUnit, pUnit2, pRadius}, cmp: 5, chooseCmp: true, rightParam: 2},
	{labelWord: "editor.wizard.c.unitdead", subject: subjectUnits, opcode: 5, params: []wizParam{pUnit}, cmp: 0, right: 0, rightParam: -1, also: [][2]int64{{1, 1}}},
	{labelWord: "editor.wizard.c.groupcount", subject: subjectGroups, opcode: 1, params: []wizParam{pGroup, pCompared}, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.playercount", subject: subjectPlayers, opcode: 8, params: []wizParam{pPlayer, pCompared}, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.relation", subject: subjectPlayers, opcode: 10, hintWord: "editor.wizard.c.relation.hint", chooseCmp: true, rightParam: 2, params: []wizParam{pPlayer, pToPlayer, {kind: wpRelation, labelWord: "editor.wizard.p.relation"}}},
	{labelWord: "editor.wizard.c.always", subject: subjectMission, opcode: scriptConstantOp, cmp: 0, right: 1, rightParam: -1, fixed: [][2]uint32{{scriptInt, 1}}, holds: true},
	{labelWord: "editor.wizard.c.unitalive", subject: subjectUnits, opcode: 5, params: []wizParam{pUnit}, cmp: 1, right: 0, rightParam: -1, also: [][2]int64{{0, 1}}},
	{labelWord: "editor.wizard.c.unitcell", subject: subjectUnits, opcode: 7, params: []wizParam{pUnit, pPoint, pRadius}, cmp: 5, chooseCmp: true, rightParam: 2},
	{labelWord: "editor.wizard.c.health", subject: subjectUnits, opcode: 4, params: []wizParam{pUnit, {kind: wpNumber, labelWord: "editor.wizard.p.health"}}, fixed: [][2]uint32{{scriptInt, 6}}, cmp: 2, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.target", subject: subjectUnits, opcode: 9, hintWord: "editor.wizard.c.target.hint", params: []wizParam{pUnit, {kind: wpNumber, labelWord: "editor.wizard.p.targetunit", def: 1}}, cmp: 0, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.lose", subject: subjectMission, opcode: 18, hintWord: "editor.wizard.c.lose.hint", params: []wizParam{pUnit, {kind: wpNumber, labelWord: "editor.wizard.p.value", def: 1}}, cmp: 2, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.losetwo", subject: subjectMission, opcode: 18, hintWord: "editor.wizard.c.lose.hint", params: []wizParam{pUnit, onRight(pUnit2)}, cmp: 1, chooseCmp: true, rightParam: -1},
	{labelWord: "editor.wizard.c.sackgone", subject: subjectCells, opcode: 14, params: []wizParam{pPoint}, cmp: 0, right: 0, rightParam: -1, rightType: scriptConst, also: [][2]int64{{1, 1}}},
	{labelWord: "editor.wizard.c.sacklies", subject: subjectCells, opcode: 14, params: []wizParam{pPoint}, cmp: 0, right: 1, rightParam: -1, rightType: scriptConst, also: [][2]int64{{1, 0}}},
	{labelWord: "editor.wizard.c.hasitem", subject: subjectItems, opcode: 17, params: []wizParam{pUnit, pItem}, cmp: 0, right: 1, rightParam: -1, rightType: scriptConst, also: [][2]int64{{1, 0}}},
	{labelWord: "editor.wizard.c.lacksitem", subject: subjectItems, opcode: 17, params: []wizParam{pUnit, pItem}, cmp: 0, right: 0, rightParam: -1, rightType: scriptConst, also: [][2]int64{{1, 1}}},
	{labelWord: "editor.wizard.c.itemnear", subject: subjectItems, opcode: 16, hintWord: "editor.wizard.c.itemnear.hint", params: []wizParam{pUnit, pItem, pPoint, pRadius}, cmp: 5, chooseCmp: true, rightParam: 3},
	{labelWord: "editor.wizard.c.structure", subject: subjectStructures, opcode: 21, params: []wizParam{pStructure, {kind: wpNumber, labelWord: "editor.wizard.p.value", def: 1}}, cmp: 3, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.variable", subject: subjectVariables, opcode: 19, params: []wizParam{pVariable, {kind: wpNumber, labelWord: "editor.wizard.p.value"}}, chooseCmp: true, rightParam: 1},
	{labelWord: "editor.wizard.c.variables", subject: subjectVariables, opcode: 19, params: []wizParam{pVariable, onRight(wizParam{kind: wpVariable, labelWord: "editor.wizard.p.variable2", inNode: true})}, chooseCmp: true, rightParam: -1},
}

// wizActions are the ready actions, with the claims that state their
// parameters: give money (TRIG-MONEY-028), win, lose, set or add 1 to a
// variable (TRIG-ACT-004), a dialog (TRIG-MSG-023; the player sees only a
// trigger's first, so one per trigger), give a unit or group to a player
// (PARTY-JOIN-025), move a unit (AI-SCRIPTGUARD-625), a unit or group off
// and back on the map and one unit in another's place (TRIG-OFFMAP-041,
// TRIG-RETURN-042, TRIG-MAPGROUP-043), the drop cell the builder takes
// (TRIG-DROP-013), a group order by its command (TRIG-GROUP-005,
// TRIG-GROUPBOUNDS-136), an item made, taken or all given (TRIG-ADDITEM-027,
// TRIG-TAKEITEM-038, TRIG-GIVEALL-025), a spell cast, its effect's time and a
// trap (TRIG-CAST-033, TRIG-EFFECTTIME-034, TRIG-CELLTAIL-035), a unit's
// property (TRIG-PROPERTY-036) and action 7, which no claim names.
var wizActions = []wizItem{
	{labelWord: "editor.wizard.a.gold", subject: subjectPlayers, opcode: 23, params: []wizParam{pPlayer, {kind: wpNumber, labelWord: "editor.wizard.p.gold", def: 500, inNode: true}}},
	{labelWord: "editor.wizard.a.win", subject: subjectMission, opcode: 4},
	{labelWord: "editor.wizard.a.lose", subject: subjectMission, opcode: 5},
	{labelWord: "editor.wizard.a.dialog", subject: subjectMission, opcode: 2, params: []wizParam{{kind: wpMessage, labelWord: "editor.wizard.p.message", inNode: true}}, onePerTrigger: true},
	{labelWord: "editor.wizard.a.giveunit", subject: subjectUnits, opcode: 19, params: []wizParam{pUnit, pPlayer}},
	{labelWord: "editor.wizard.a.givegroup", subject: subjectGroups, opcode: 22, params: []wizParam{pGroup, pPlayer}},
	{labelWord: "editor.wizard.a.move", subject: subjectUnits, opcode: 27, params: []wizParam{pUnit, pPoint}},
	{labelWord: "editor.wizard.a.unitoff", subject: subjectUnits, opcode: 16, params: []wizParam{pUnit}},
	{labelWord: "editor.wizard.a.unitback", subject: subjectUnits, opcode: 17, params: []wizParam{pUnit}},
	{labelWord: "editor.wizard.a.groupoff", subject: subjectGroups, opcode: 32, params: []wizParam{pGroup}},
	{labelWord: "editor.wizard.a.groupback", subject: subjectGroups, opcode: 33, params: []wizParam{pGroup}},
	{labelWord: "editor.wizard.a.drop", subject: subjectMission, opcode: scriptConstantOp, params: []wizParam{pPoint}},
	{labelWord: "editor.wizard.a.order", subject: subjectGroups, opcode: scriptGroupCommand, params: []wizParam{pLastGroup, command(4), pPoint}},
	{labelWord: "editor.action.op10", subject: subjectPlayers, opcode: 10, hintWord: "editor.wizard.a.relation.hint", params: []wizParam{numbered(pPlayer), numbered(pToPlayer), {kind: wpRelation, labelWord: "editor.wizard.p.relation", inNode: true}}},
	{labelWord: "editor.wizard.a.addvar", subject: subjectVariables, opcode: 8, params: []wizParam{pVariable}},
	{labelWord: "editor.wizard.a.setvar", subject: subjectVariables, opcode: 3, params: []wizParam{pVariable, {kind: wpNumber, labelWord: "editor.wizard.p.value", inNode: true}}},
	{labelWord: "editor.wizard.a.takeitem", subject: subjectItems, opcode: 13, params: []wizParam{pUnit, pItem}},
	{labelWord: "editor.wizard.a.giveitem", subject: subjectItems, opcode: 12, params: []wizParam{pUnit, pItem}},
	{labelWord: "editor.wizard.a.giveall", subject: subjectItems, opcode: 28, params: []wizParam{pUnit, {kind: wpUnit, labelWord: "editor.wizard.p.receiver", inNode: true}}},
	{labelWord: "editor.wizard.a.replace", subject: subjectUnits, opcode: 18, hintWord: "editor.wizard.a.replace.hint", params: []wizParam{pUnit, pUnit2}},
	{labelWord: "editor.wizard.a.property", subject: subjectUnits, opcode: 34, params: []wizParam{pUnit, {kind: wpProperty, labelWord: "editor.wizard.p.property", def: 6, inNode: true}, {kind: wpNumber, labelWord: "editor.wizard.p.value", inNode: true}}},
	{labelWord: "editor.wizard.a.castcell", subject: subjectSpells, opcode: 21, hintWord: "editor.wizard.a.cast.hint", params: []wizParam{pFrom, pTo, pSpell, pPower}},
	{labelWord: "editor.wizard.a.castunit", subject: subjectSpells, opcode: 24, params: []wizParam{pFrom, pUnit, pSpell, pPower}},
	{labelWord: "editor.wizard.a.trap", subject: subjectSpells, opcode: 25, params: []wizParam{pSpell, pPower, pPoint}},
	{labelWord: "editor.wizard.a.celltime", subject: subjectSpells, opcode: 29, params: []wizParam{pPoint, pSpell, pTime}},
	{labelWord: "editor.wizard.a.unittime", subject: subjectSpells, opcode: 30, params: []wizParam{pUnit, pSpell, pTime}},
	{labelWord: "editor.wizard.a.action7", subject: subjectPlayers, opcode: 7, params: []wizParam{pPlayer, {kind: wpNumber, labelWord: "editor.wizard.p.value", inNode: true}}},
	{labelWord: "editor.wizard.a.order.stand", variant: true, subject: subjectGroups, opcode: scriptGroupCommand, params: []wizParam{pLastGroup, command(3)}},
	{labelWord: "editor.wizard.a.order.attack", variant: true, subject: subjectGroups, opcode: scriptGroupCommand, params: []wizParam{pLastGroup, command(10), pUnit}},
	{labelWord: "editor.wizard.a.order.defend", variant: true, subject: subjectGroups, opcode: scriptGroupCommand, params: []wizParam{pLastGroup, command(11), pUnit, pRange}},
	{labelWord: "editor.wizard.a.order.follow", variant: true, subject: subjectGroups, opcode: scriptGroupCommand, params: []wizParam{{kind: wpGroup, labelWord: "editor.wizard.p.group", inNode: true, at: 2}, {kind: wpCommand, labelWord: "editor.wizard.p.command", def: 15, inNode: true, at: 1}, pUnit, pRange}},
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
	case wpItem:
		return []uint32{scriptItem}
	case wpStructure:
		return []uint32{scriptStructure}
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

// itemSlots are the slot types the item's node on one side of the pair
// carries (right: the right node of an item that compares two nodes), the
// slot each parameter starts at (-1 for one no slot of that node holds) and
// the slot the left node's fixed values start at.
func (it wizItem) itemSlots(right bool) (types [10]uint32, at []int, fixed int) {
	for _, p := range it.params {
		t := p.types()
		switch {
		case !p.inNode || p.right != right:
			at = append(at, -1)
		case p.last:
			at = append(at, len(types)-len(t))
			copy(types[len(types)-len(t):], t)
		default:
			start := fixed
			if p.at > 0 {
				start = p.at - 1
			}
			at = append(at, start)
			copy(types[start:], t)
			fixed = max(fixed, start+len(t))
		}
	}
	if !right {
		for k, f := range it.fixed {
			types[fixed+k] = f[0]
		}
	}
	return types, at, fixed
}

// twoNodes reports an item whose pair compares two nodes of its check.
func (it wizItem) twoNodes() bool {
	return slices.ContainsFunc(it.params, func(p wizParam) bool { return p.right })
}

// wizRow is one row: its item (-1 a pair or action kept as it stands), the
// nodes it names now and, for a kept pair, its comparison; vals are its
// field texts (the item, each parameter, then a chosen comparison) and orig
// those it was read with, so an unchanged row keeps its nodes. origItem is
// the item it was read with, -2 for a row added in the window: only a row
// read as kept offers "Other" back. swapped marks a pair read with its
// sides exchanged: its check stands on the right in the map.
type wizRow struct {
	condition        bool
	item, origItem   int
	left, right, act uint32
	cmp              uint32
	swapped          bool
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
		r.item, r.vals, r.swapped = e.readCondition(t.Left[k], t.Right[k], t.Cmp[k])
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

// keptPlan is trigger index as it stands, every pair and action kept, with
// the rows added after them.
func (e *MapEditor) keptPlan(index int, added ...wizRow) (WizardTrigger, error) {
	t := &e.doc.Script.Triggers[index]
	return e.rowsPlan(WizardTrigger{Index: index, Name: t.Name, Once: t.Once}, append(e.wizardRows(t), added...))
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

// readParams fills out with the texts of the parameters item it's node on
// one side of the pair holds, false when the node's slots are not the item's
// or its group command is not one the item offers.
func (e *MapEditor) readParams(it wizItem, n InspectionScriptNode, right bool, out []string) bool {
	want, at, fixed := it.itemSlots(right)
	if n.Opcode != it.opcode || n.Types != want {
		return false
	}
	for k, p := range it.params {
		if at[k] < 0 {
			continue
		}
		v := n.Values[at[k]:]
		switch p.kind {
		case wpCell:
			out[k] = fmt.Sprintf("%d,%d", v[0], v[1])
		case wpBox:
			out[k] = fmt.Sprintf("%d,%d,%d,%d", v[0], v[1], v[2], v[3])
		case wpNumber, wpMessage, wpRelation, wpVariable, wpSpell, wpProperty:
			out[k] = strconv.FormatInt(int64(int32(v[0])), 10)
		case wpCommand:
			if !slices.Contains(e.itemCommands(it), v[0]) {
				return false
			}
			out[k] = strconv.FormatUint(uint64(v[0]), 10)
		default:
			out[k] = strconv.FormatUint(uint64(v[0]), 10)
		}
	}
	for k, f := range it.fixed {
		if !right && n.Values[fixed+k] != f[1] {
			return false
		}
	}
	return true
}

// itemCommands are the group commands whose signature, where a claim states
// one, is item it's slots: for a target cell, 2, 4, 5 and 14
// (TRIG-GROUPBOUNDS-136).
func (e *MapEditor) itemCommands(it wizItem) []uint32 {
	want, _, _ := it.itemSlots(false)
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

// mirroredCmps are, by code, the comparison of b with a that holds when a
// compared with b does (TRIG-CMP-006).
var mirroredCmps = [scriptprofile.MaxCompare + 1]uint32{0, 1, 3, 2, 5, 4}

// readCondition is the item and field texts of the pair left cmp right, -1
// when no item reads it, and whether it read the sides exchanged: a pair
// whose left side is a constant reads as its mirror, the check on the left.
func (e *MapEditor) readCondition(left, right, cmp uint32) (int, []string, bool) {
	if i, out := e.readPair(left, right, cmp); i >= 0 {
		return i, out, false
	}
	if _, constant := e.constantValue(left); constant && cmp < uint32(len(mirroredCmps)) {
		if i, out := e.readPair(right, left, mirroredCmps[cmp]); i >= 0 {
			return i, out, true
		}
	}
	return -1, []string{"-1"}, false
}

// readPair is the item and field texts of the pair left cmp right read as it
// stands, -1 when no item reads it.
func (e *MapEditor) readPair(left, right, cmp uint32) (int, []string) {
	n, ok := e.nodeByID(true, left)
	if !ok {
		return -1, nil
	}
	v, constant := e.constantValue(right)
	rn, rok := e.nodeByID(true, right)
	for i, it := range wizConditions {
		params := make([]string, len(it.params))
		switch {
		case it.holds && left == right && n.Opcode == it.opcode && (cmp == 0 || cmp == 4 || cmp == 5):
		case !e.readParams(it, n, false, params):
			continue
		case it.twoNodes():
			if !rok || !e.readParams(it, rn, true, params) || !it.reads(cmp, 0) {
				continue
			}
		case !constant || !it.reads(cmp, v):
			continue
		case it.rightParam >= 0:
			params[it.rightParam] = strconv.FormatInt(v, 10)
		}
		out := append([]string{strconv.Itoa(i)}, params...)
		if it.chooseCmp {
			out = append(out, strconv.FormatUint(uint64(cmp), 10))
		}
		return i, out
	}
	return -1, nil
}

// reads reports that the item states comparison cmp with constant v: any
// comparison when the row chooses it, and v when the item fixes it.
func (it wizItem) reads(cmp uint32, v int64) bool {
	fixedRight := it.rightParam < 0 && !it.twoNodes()
	switch {
	case it.chooseCmp:
		return !fixedRight || v == it.right
	case !fixedRight:
		return cmp == uint32(it.cmp)
	case cmp == uint32(it.cmp) && v == it.right:
		return true
	}
	return slices.ContainsFunc(it.also, func(f [2]int64) bool { return uint32(f[0]) == cmp && f[1] == v })
}

// readAction is the item and field texts of action id, -1 when no item
// reads it.
func (e *MapEditor) readAction(id uint32) (int, []string) {
	n, ok := e.nodeByID(false, id)
	if ok {
		for i, it := range wizActions {
			params := make([]string, len(it.params))
			if e.readParams(it, n, false, params) {
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
		case wpVariable, wpItem, wpStructure, wpSpell, wpProperty:
			out = append(out, strconv.FormatInt(e.registryChoices(condition, it, len(out)-1, p.def)[0].Value, 10))
		default:
			out = append(out, strconv.FormatInt(p.def, 10))
		}
	}
	if it.chooseCmp {
		out = append(out, strconv.Itoa(it.cmp))
	}
	return out
}

// registryChoices are the choices of parameter k of item it: the script
// targets the host lists for the slot that holds it, as Details offers
// them, with value kept when it is none of them.
func (e *MapEditor) registryChoices(condition bool, it wizItem, k int, value int64) []InspectionChoice {
	types, at, _ := it.itemSlots(it.params[k].right)
	return e.parameterChoices(e.nodeParameterKind(scriptView{condition: condition}, it.opcode, types, at[k]), value)
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
		choices, shut := e.itemChoices(w, r)
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
		o.fields = append(o.fields, InspectionField{Key: w.wizardKey(r, "item"), Label: label, Value: int64(row.origItem), Choices: choices, Shut: shut, ListAll: true})
		o.buf = append(o.buf, row.vals[0])
		if row.item < 0 {
			continue
		}
		it := items[row.item]
		for k := range it.params {
			f := e.paramField(row.condition, row.item, k, text(k+1))
			f.Key = w.wizardKey(r, fmt.Sprintf("p%d", k))
			o.fields = append(o.fields, f)
			o.buf = append(o.buf, row.vals[k+1])
		}
		if it.chooseCmp {
			f := e.cmpField(text(len(it.params) + 1))
			f.Key = w.wizardKey(r, "cmp")
			o.fields = append(o.fields, f)
			o.buf = append(o.buf, row.vals[len(it.params)+1])
		}
	}
}

// paramField is the field of parameter k of item i whose text is value: a
// choice of what the parameter can name, or typed text. The wizard's rows
// and the Add window's condition steps ask a parameter through it.
func (e *MapEditor) paramField(condition bool, i, k int, value string) InspectionField {
	it := wizItems(condition)[i]
	p := it.params[k]
	f := InspectionField{Label: e.word(p.labelWord)}
	n, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	switch p.kind {
	case wpUnit, wpPlayer, wpGroup, wpMessage, wpRelation:
		f.Value = n
		f.Choices = e.wizardChoices(p.kind, n)
	case wpVariable, wpItem, wpStructure, wpSpell, wpProperty:
		f.Value = n
		f.Choices = e.registryChoices(condition, it, k, n)
	case wpCommand:
		f.Value = n
		for _, c := range e.doc.Script.Catalog.GroupCommands {
			if e.commandItem(condition, i, c) >= 0 {
				f.Choices = append(f.Choices, InspectionChoice{int64(c), e.commandName(c)})
			}
		}
	default:
		f.Text, f.TextValue = true, value
	}
	return f
}

// cmpField is the comparison field whose text is value: the six codes, and a
// code outside them the row was read with.
func (e *MapEditor) cmpField(value string) InspectionField {
	var cmps []InspectionChoice
	for c := int64(0); c <= scriptprofile.MaxCompare; c++ {
		cmps = append(cmps, InspectionChoice{c, e.cmpWord(c)})
	}
	n, _ := strconv.ParseInt(value, 10, 64)
	if n < 0 || n > scriptprofile.MaxCompare {
		cmps = append(cmps, InspectionChoice{n, e.cmpWord(n)})
	}
	return InspectionField{Label: e.word("editor.trigger.comparison"), Value: n, Choices: cmps}
}

// wizHeading is the choice value of the first subject heading; the others
// count down from it, below the kept row's -1.
const wizHeading = -10

// itemChoices are row r's item list and the choices it shuts: each
// subject's heading, then its items, a variant shown as its family's one
// entry, and Other for a row read as kept.
func (e *MapEditor) itemChoices(w *wizardState, r int) ([]InspectionChoice, []int64) {
	row := w.rows[r]
	items := wizItems(row.condition)
	var choices []InspectionChoice
	var shut []int64
	for s, subject := range wizSubjectWords {
		heading := false
		for i, it := range items {
			if it.subject != s || it.variant {
				continue
			}
			if !heading {
				h := int64(wizHeading - s)
				choices = append(choices, InspectionChoice{h, fmt.Sprintf(e.word("editor.wizard.subject"), e.word(subject))})
				shut, heading = append(shut, h), true
			}
			value := int64(i)
			if row.item >= 0 && familyOf(items, row.item) == i {
				value = int64(row.item)
			}
			label := e.word(it.labelWord)
			if w.shut(r, i) {
				label = fmt.Sprintf(e.word("editor.wizard.onlyone"), label)
				shut = append(shut, value)
			}
			choices = append(choices, InspectionChoice{value, label})
		}
	}
	if row.origItem == -1 {
		choices = append(choices, InspectionChoice{-1, e.keptText(row)})
	}
	return choices, shut
}

// familyOf is the item whose list entry item i shows under: i, or for a
// variant the nearest earlier item with its opcode that is none.
func familyOf(items []wizItem, i int) int {
	for k := i; k >= 0; k-- {
		if items[k].opcode == items[i].opcode && !items[k].variant {
			return k
		}
	}
	return i
}

// commandItem is the item of item i's family that offers group command c,
// -1 for none.
func (e *MapEditor) commandItem(condition bool, i int, c uint32) int {
	items := wizItems(condition)
	for k := range items {
		if familyOf(items, k) == familyOf(items, i) && slices.Contains(e.itemCommands(items[k]), c) {
			return k
		}
	}
	return -1
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
		for v, r := range relationValues {
			add(int64(v), e.word(r.nameWord))
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
			if to, vals, ok := e.commandSwitch(*row); ok {
				row.item, row.vals, rebuilt = to, vals, true
			}
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

// commandSwitch is the variant of row's item whose command the row chose
// and its texts, each parameter carried over from the row's first unused one
// of its kind, false when the item offers the command itself.
func (e *MapEditor) commandSwitch(row wizRow) (int, []string, bool) {
	if row.item < 0 {
		return 0, nil, false
	}
	items := wizItems(row.condition)
	k := slices.IndexFunc(items[row.item].params, func(p wizParam) bool { return p.kind == wpCommand })
	if k < 0 {
		return 0, nil, false
	}
	c, err := strconv.ParseUint(strings.TrimSpace(row.vals[k+1]), 10, 32)
	if err != nil || slices.Contains(e.itemCommands(items[row.item]), uint32(c)) {
		return 0, nil, false
	}
	to := e.commandItem(row.condition, row.item, uint32(c))
	if to < 0 {
		return 0, nil, false
	}
	vals := e.wizardDefaults(row.condition, to)
	if to == row.origItem {
		vals = append([]string(nil), row.orig...)
	}
	used := map[int]bool{}
	for j, p := range items[to].params {
		for s, q := range items[row.item].params {
			if !used[s] && q.kind == p.kind {
				used[s], vals[j+1] = true, row.vals[s+1]
				break
			}
		}
	}
	return to, vals, true
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
			case wpUnit, wpPlayer, wpGroup, wpCommand, wpRelation, wpVariable, wpItem, wpStructure, wpSpell, wpProperty:
				pr.Kind, pr.Value = popupChoice, choiceText(o.fields[i], o.buf[i])
				if p.kind == wpUnit || p.kind == wpGroup || p.kind == wpStructure {
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
		if it.chooseCmp {
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
	if conds < scriptprofile.RecordPairs {
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
	if acts < scriptprofile.RecordActions {
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
	fitButtons(s, e.bodyFont())
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
	for _, row := range rows {
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
		node, right, cmp, err := e.wizardNode(row)
		if err != nil {
			return t, err
		}
		if row.condition {
			node.ID, right.ID = row.left, row.right
			if row.swapped { // the check is written on the left
				node.ID, right.ID = row.right, row.left
			}
			t.Pairs = append(t.Pairs, WizardPair{Left: node, Cmp: cmp, Right: right})
		} else {
			node.ID = row.act
			t.Actions = append(t.Actions, node)
		}
	}
	if len(t.Pairs) > scriptprofile.RecordPairs || len(t.Actions) > scriptprofile.RecordActions {
		return t, fmt.Errorf("%s", e.word("editor.wizard.full"))
	}
	return t, nil
}

// wizardNode is the node row says, the node it is compared with (the
// constant of the item's type, or the item's second check node) and the
// comparison.
func (e *MapEditor) wizardNode(row wizRow) (WizardNode, WizardNode, uint32, error) {
	it := wizItems(row.condition)[row.item]
	n, rn := WizardNode{Opcode: it.opcode}, WizardNode{Opcode: scriptConstantOp}
	_, at, fixed := it.itemSlots(false)
	_, rat, _ := it.itemSlots(true)
	right := it.right
	for k, p := range it.params {
		nums, err := e.wizardParam(it, k, row.vals[k+1])
		if err != nil {
			return n, rn, 0, err
		}
		switch {
		case p.right:
			rn.Opcode = it.opcode
			for c, t := range p.types() {
				rn.Types[rat[k]+c], rn.Values[rat[k]+c] = t, uint32(int32(nums[c]))
			}
		case p.inNode:
			for c, t := range p.types() {
				n.Types[at[k]+c], n.Values[at[k]+c] = t, uint32(int32(nums[c]))
			}
		case k == it.rightParam:
			right = nums[0]
		}
	}
	for k, f := range it.fixed {
		n.Types[fixed+k], n.Values[fixed+k] = f[0], f[1]
	}
	if !row.condition {
		return n, rn, 0, nil
	}
	if !it.twoNodes() {
		rn.Types[0], rn.Values[0] = scriptInt, uint32(int32(right))
		if it.rightType != 0 {
			rn.Types[0] = it.rightType
		}
	}
	cmp := uint32(it.cmp)
	if it.chooseCmp {
		// A code outside the alphabet the row was read with stays choosable.
		c, err := strconv.ParseInt(strings.TrimSpace(row.vals[len(it.params)+1]), 10, 64)
		if err != nil || c < 0 || c > math.MaxUint32 || c > scriptprofile.MaxCompare && uint32(c) != row.cmp {
			return n, rn, 0, fmt.Errorf("%s: %s", e.word("editor.trigger.comparison"), e.word("editor.object.number"))
		}
		cmp = uint32(c)
	}
	return n, rn, cmp, nil
}

// wizardParam is the numbers text gives parameter k of item it, or why the
// parameter refuses it: a number out of range, a cell or box off the map, an
// unset unit or player.
func (e *MapEditor) wizardParam(it wizItem, k int, text string) ([]int64, error) {
	p := it.params[k]
	label := e.word(p.labelWord)
	var nums []int64
	for _, part := range strings.Split(strings.TrimSpace(text), ",") {
		x, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			nums = nil
			break
		}
		if x < math.MinInt32 || x > math.MaxInt32 {
			return nil, fmt.Errorf("%s: %s", label, e.word("editor.wizard.range"))
		}
		nums = append(nums, x)
	}
	if len(nums) != len(p.types()) {
		switch p.kind {
		case wpCell:
			return nil, fmt.Errorf("%s: %s", label, e.word("editor.node.cell"))
		case wpBox:
			return nil, fmt.Errorf("%s: %s", label, e.word("editor.wizard.badbox"))
		}
		return nil, fmt.Errorf("%s: %s", label, e.word("editor.object.number"))
	}
	switch p.kind {
	case wpCell, wpBox:
		for c, x := range nums {
			limit := e.doc.Width
			if c%2 == 1 {
				limit = e.doc.Height
			}
			if x < 0 || x >= int64(min(limit, 256)) {
				return nil, fmt.Errorf("%s: %s", label, e.word("editor.wizard.outside"))
			}
		}
		if p.kind == wpBox && (nums[0] > nums[2] || nums[1] > nums[3]) {
			return nil, fmt.Errorf("%s: %s", label, e.word("editor.wizard.badbox"))
		}
	case wpUnit, wpPlayer:
		if nums[0] <= 0 {
			return nil, fmt.Errorf("%s: %s", label, e.word("editor.node.unset"))
		}
	}
	return nums, nil
}

// loneRow is the wizard row a condition no pair names reads as: the first
// item that reads node n as its check, alone on its side, and writes it back
// as it stands, with the item's own comparison and the defaults of the
// parameters the node does not hold. The row names n, so the trigger takes n
// itself; false when no item reads n so.
func (e *MapEditor) loneRow(n InspectionScriptNode) (wizRow, bool) {
	for i, it := range wizConditions {
		if it.twoNodes() || it.holds {
			continue
		}
		vals := e.wizardDefaults(true, i)
		params := make([]string, len(it.params))
		if !e.readParams(it, n, false, params) {
			continue
		}
		_, at, _ := it.itemSlots(false)
		for k := range it.params {
			if at[k] >= 0 {
				vals[k+1] = params[k]
			}
		}
		row := wizRow{condition: true, item: i, origItem: -2, left: n.ID, vals: vals}
		node, _, _, err := e.wizardNode(row)
		if err == nil && node.Opcode == n.Opcode && node.Types == n.Types && node.Values == n.Values {
			return row, true
		}
	}
	return wizRow{}, false
}

// applyWizard writes the rows through the host as one edit.
func (e *MapEditor) applyWizard(v scriptView) {
	o := &e.obj
	if _, ok := e.doc.Edits.(InspectionTriggerWizard); !ok {
		return
	}
	t, err := e.wizardPlan(v)
	if err != nil {
		o.note = e.editorHostText(err.Error())
		return
	}
	t.Join = e.joinsAdd(v.index)
	ch, err := e.commandWriteTrigger(t)
	if err != nil {
		o.note = err.Error() // host text: the popup encodes its note
		return
	}
	o.open, o.note = -1, ""
	if ch.doc != nil {
		e.fresh = freshTrigger{}
	}
	if ch.doc == nil {
		e.obj.doc = nil // nothing written: read the trigger again
		if t.Join {
			// The session goes on with the rows read again.
			e.loadObject()
			e.fresh.wiz = e.obj.wiz
		}
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
				if cmp := wizRowCmp(it, row); cmp == 3 || cmp == 4 {
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

// wizRowCmp is the comparison row of item it states: the one it chose, else
// the item's.
func wizRowCmp(it wizItem, row wizRow) int {
	if it.chooseCmp && len(row.vals) > len(it.params)+1 {
		if c, err := strconv.Atoi(strings.TrimSpace(row.vals[len(it.params)+1])); err == nil {
			return c
		}
	}
	return it.cmp
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
