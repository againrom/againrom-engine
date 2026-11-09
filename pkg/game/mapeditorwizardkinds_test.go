package game

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// The script's slot types, by their ALM numbers.
const (
	kInt, kGroup, kPlayer, kUnit, kX, kY, kConst, kItem, kStructure uint32 = 1, 2, 3, 4, 5, 6, 7, 8, 9
)

// kindNode is a check or action node of a kind fixture: its operation, slot
// types and values.
type kindNode struct {
	op           uint32
	tags, values []uint32
}

// kindForm is one shipped form of an editor census kind: a pair of the
// check and a constant (or, with other, a second check node; with self, the
// check compared with itself; with swap, the constant on the left), or an
// action. item is the label word of the wizard item it reads as, "" for a
// row the wizard keeps as Other; lines are EN popup lines the row shows.
type kindForm struct {
	name          string
	item          string
	action        bool
	node          kindNode
	other         *kindNode
	constTag      uint32
	constVal, cmp uint32
	swap, self    bool
	unresolved    bool
	lines         []string
}

// kindForms are the shipped forms of the census kinds C1 to C15 and A1 to
// A17, one row each, and the two rows no item states: the C2 row over two
// units and a C10 pair of unbuilt node IDs.
var kindForms = []kindForm{
	{name: "C1 Int ==", item: "editor.wizard.c.variable", node: kindNode{19, []uint32{kInt}, []uint32{51}}, constTag: kInt, cmp: 0, lines: []string{"Value 0", "Comparison equals"}},
	{name: "C1 Const !=", item: "editor.wizard.c.variable", node: kindNode{19, []uint32{kInt}, []uint32{52}}, constTag: kConst, constVal: 3, cmp: 1, lines: []string{"Value 3", "Comparison does not equal"}},
	{name: "C1 itself, never", item: "editor.wizard.c.variables", node: kindNode{19, []uint32{kInt}, []uint32{51}}, self: true, cmp: 0xFFFFFFFF},
	{name: "C2 Int <", item: "editor.wizard.c.unitcell", node: kindNode{7, []uint32{kUnit, kX, kY}, []uint32{10, 4, 5}}, constTag: kInt, constVal: 5, cmp: 3, lines: []string{"Point 4,5", "Radius 5", "Comparison is less than"}},
	{name: "C2 Const <=", item: "editor.wizard.c.unitcell", node: kindNode{7, []uint32{kUnit, kX, kY}, []uint32{10, 4, 5}}, constTag: kConst, constVal: 4, cmp: 5},
	{name: "C2 Int ==", item: "editor.wizard.c.unitcell", node: kindNode{7, []uint32{kUnit, kX, kY}, []uint32{10, 4, 5}}, constTag: kInt, constVal: 0, cmp: 0},
	{name: "C3 Int itself ==", item: "editor.wizard.c.always", node: kindNode{0x10002, []uint32{kInt}, []uint32{1}}, self: true, cmp: 0},
	{name: "C3 Const itself <=", item: "editor.wizard.c.always", node: kindNode{0x10002, []uint32{kConst}, []uint32{1}}, self: true, cmp: 5},
	{name: "C3 reversed", item: "editor.wizard.c.unitnear", node: kindNode{3, []uint32{kUnit, kX, kY, kInt}, []uint32{10, 6, 7, 2}}, constTag: kInt, constVal: 1, cmp: 0, swap: true, lines: []string{"Point 6,7", "Radius 2"}},
	{name: "C4 Int == 0", item: "editor.wizard.c.sackgone", node: kindNode{14, []uint32{kX, kY}, []uint32{8, 9}}, constTag: kInt, cmp: 0, lines: []string{"Point 8,9"}},
	{name: "C4 Const == 0", item: "editor.wizard.c.sackgone", node: kindNode{14, []uint32{kX, kY}, []uint32{8, 9}}, constTag: kConst, cmp: 0},
	{name: "C4 Const != 1", item: "editor.wizard.c.sackgone", node: kindNode{14, []uint32{kX, kY}, []uint32{8, 9}}, constTag: kConst, constVal: 1, cmp: 1},
	{name: "C4 Int == 1", item: "editor.wizard.c.sacklies", node: kindNode{14, []uint32{kX, kY}, []uint32{8, 9}}, constTag: kInt, constVal: 1, cmp: 0},
	{name: "C5 Int <", item: "editor.wizard.c.playernear", node: kindNode{15, []uint32{kPlayer, kX, kY}, []uint32{1, 3, 3}}, constTag: kInt, constVal: 5, cmp: 3, lines: []string{"Radius 5", "Comparison is less than"}},
	{name: "C5 Int !=", item: "editor.wizard.c.playernear", node: kindNode{15, []uint32{kPlayer, kX, kY}, []uint32{1, 3, 3}}, constTag: kInt, constVal: 5, cmp: 1},
	{name: "C5 Int ==", item: "editor.wizard.c.playernear", node: kindNode{15, []uint32{kPlayer, kX, kY}, []uint32{1, 3, 3}}, constTag: kInt, constVal: 5, cmp: 0},
	{name: "C5 Const ==", item: "editor.wizard.c.playernear", node: kindNode{15, []uint32{kPlayer, kX, kY}, []uint32{1, 3, 3}}, constTag: kConst, constVal: 2, cmp: 0},
	{name: "C6 == 1", item: "editor.wizard.c.hasitem", node: kindNode{17, []uint32{kUnit, kItem}, []uint32{10, 7}}, constTag: kConst, constVal: 1, cmp: 0, lines: []string{"Item 7"}},
	{name: "C6 != 1", item: "editor.wizard.c.lacksitem", node: kindNode{17, []uint32{kUnit, kItem}, []uint32{10, 7}}, constTag: kConst, constVal: 1, cmp: 1},
	{name: "C6 == 0", item: "editor.wizard.c.lacksitem", node: kindNode{17, []uint32{kUnit, kItem}, []uint32{10, 7}}, constTag: kConst, cmp: 0},
	{name: "C7 < 1", item: "editor.wizard.c.structure", node: kindNode{21, []uint32{kStructure}, []uint32{4}}, constTag: kInt, constVal: 1, cmp: 3, lines: []string{"Value 1", "Comparison is less than"}},
	{name: "C7 Const == 0", item: "editor.wizard.c.structure", node: kindNode{21, []uint32{kStructure}, []uint32{4}}, constTag: kConst, cmp: 0},
	{name: "C8 >", item: "editor.wizard.c.health", node: kindNode{4, []uint32{kUnit, kInt}, []uint32{10, 6}}, constTag: kInt, constVal: 40, cmp: 2, lines: []string{"Health 40", "Comparison is greater than"}},
	{name: "C8 Const <=", item: "editor.wizard.c.health", node: kindNode{4, []uint32{kUnit, kInt}, []uint32{10, 6}}, constTag: kConst, constVal: 40, cmp: 5},
	{name: "C8 >=", item: "editor.wizard.c.health", node: kindNode{4, []uint32{kUnit, kInt}, []uint32{10, 6}}, constTag: kInt, constVal: 40, cmp: 4},
	{name: "C9 <", item: "editor.wizard.c.unitnearunit", node: kindNode{6, []uint32{kUnit, kUnit}, []uint32{10, 11}}, constTag: kInt, constVal: 3, cmp: 3, lines: []string{"Radius 3", "Comparison is less than"}},
	{name: "C9 >", item: "editor.wizard.c.unitnearunit", node: kindNode{6, []uint32{kUnit, kUnit}, []uint32{10, 11}}, constTag: kInt, constVal: 3, cmp: 2},
	{name: "C11 two checks !=", item: "editor.wizard.c.losetwo", node: kindNode{18, []uint32{kUnit}, []uint32{21}}, other: &kindNode{18, []uint32{kUnit}, []uint32{51}}, cmp: 1, lines: []string{"Comparison does not equal"}},
	{name: "C11 two checks <", item: "editor.wizard.c.losetwo", node: kindNode{18, []uint32{kUnit}, []uint32{21}}, other: &kindNode{18, []uint32{kUnit}, []uint32{51}}, cmp: 3},
	{name: "C11 constant >", item: "editor.wizard.c.lose", node: kindNode{18, []uint32{kUnit}, []uint32{21}}, constTag: kInt, constVal: 1, cmp: 2},
	{name: "C12 != 0", item: "editor.wizard.c.unitalive", node: kindNode{5, []uint32{kUnit}, []uint32{10}}, constTag: kInt, cmp: 1},
	{name: "C12 != 1", item: "editor.wizard.c.unitdead", node: kindNode{5, []uint32{kUnit}, []uint32{10}}, constTag: kInt, constVal: 1, cmp: 1},
	{name: "C13 != 0", item: "editor.wizard.c.heroarea", node: kindNode{2, []uint32{kUnit, kX, kY, kX, kY}, []uint32{10, 2, 2, 5, 5}}, constTag: kInt, cmp: 1, lines: []string{"Area 2,2,5,5"}},
	{name: "C14 == 1", item: "editor.wizard.c.target", node: kindNode{9, []uint32{kUnit}, []uint32{10}}, constTag: kInt, constVal: 1, cmp: 0, lines: []string{"Attacked unit's number 1"}},
	{name: "C15 <= 6", item: "editor.wizard.c.itemnear", node: kindNode{16, []uint32{kUnit, kItem, kX, kY}, []uint32{10, 7, 9, 9}}, constTag: kInt, constVal: 6, cmp: 5, lines: []string{"Radius 6"}},
	{name: "C2 two units", node: kindNode{7, []uint32{kUnit, kUnit}, []uint32{10, 11}}, constTag: kInt, constVal: 3, cmp: 3},
	{name: "C10 unbuilt IDs", unresolved: true, cmp: 0},
	{name: "A1", item: "editor.wizard.a.addvar", action: true, node: kindNode{8, []uint32{kInt}, []uint32{51}}},
	{name: "A2", item: "editor.wizard.a.takeitem", action: true, node: kindNode{13, []uint32{kUnit, kItem}, []uint32{10, 7}}, lines: []string{"Item 7"}},
	{name: "A3", item: "editor.wizard.a.order", action: true, node: kindNode{6, []uint32{kConst, 0, 0, 0, 0, 0, 0, 0, 0, kGroup}, []uint32{3, 0, 0, 0, 0, 0, 0, 0, 0, 22}}, lines: []string{"Command Stand Ground"}},
	{name: "A4", item: "editor.wizard.a.order", action: true, node: kindNode{6, []uint32{kConst, kGroup, kUnit, kInt}, []uint32{15, 22, 10, 4}}, lines: []string{"Command Follow", "Range 4"}},
	{name: "A5", item: "editor.wizard.a.order", action: true, node: kindNode{6, []uint32{kConst, kUnit, 0, 0, 0, 0, 0, 0, 0, kGroup}, []uint32{10, 10, 0, 0, 0, 0, 0, 0, 0, 22}}, lines: []string{"Command Attack"}},
	{name: "A6", item: "editor.wizard.a.order", action: true, node: kindNode{6, []uint32{kConst, kUnit, kInt, 0, 0, 0, 0, 0, 0, kGroup}, []uint32{11, 10, 5, 0, 0, 0, 0, 0, 0, 22}}, lines: []string{"Command Defend", "Range 5"}},
	{name: "A7", item: "editor.wizard.a.castcell", action: true, node: kindNode{21, []uint32{kX, kY, kX, kY, kInt, kInt}, []uint32{1, 2, 3, 4, 3, 0}}, lines: []string{"From point 1,2", "At point 3,4", "Power 0"}},
	{name: "A8", item: "editor.wizard.a.castunit", action: true, node: kindNode{24, []uint32{kX, kY, kUnit, kInt, kInt}, []uint32{1, 2, 10, 3, 5}}, lines: []string{"Power 5"}},
	{name: "A9", item: "editor.wizard.a.celltime", action: true, node: kindNode{29, []uint32{kX, kY, kInt, kInt}, []uint32{5, 6, 3, 200}}, lines: []string{"Time 200"}},
	{name: "A10", item: "editor.wizard.a.setvar", action: true, node: kindNode{3, []uint32{kInt, kInt}, []uint32{51, 2}}, lines: []string{"Value 2"}},
	{name: "A11", item: "editor.wizard.a.property", action: true, node: kindNode{34, []uint32{kUnit, kInt, kInt}, []uint32{10, 16, 30}}, lines: []string{"Property Absorption (16)", "Value 30"}},
	{name: "A12", item: "editor.wizard.a.giveitem", action: true, node: kindNode{12, []uint32{kUnit, kItem}, []uint32{10, 7}}},
	{name: "A13", item: "editor.wizard.a.replace", action: true, node: kindNode{18, []uint32{kUnit, kUnit}, []uint32{10, 11}}},
	{name: "A14", item: "editor.wizard.a.giveall", action: true, node: kindNode{28, []uint32{kUnit, kUnit}, []uint32{10, 11}}},
	{name: "A15", item: "editor.wizard.a.trap", action: true, node: kindNode{25, []uint32{kInt, kInt, kX, kY}, []uint32{3, 4, 7, 8}}, lines: []string{"Power 4", "Point 7,8"}},
	{name: "A16", item: "editor.wizard.a.unittime", action: true, node: kindNode{30, []uint32{kUnit, kInt, kInt}, []uint32{10, 3, 50}}, lines: []string{"Time 50"}},
	{name: "A17", item: "editor.wizard.a.action7", action: true, node: kindNode{7, []uint32{kPlayer, kInt}, []uint32{1, 2}}, lines: []string{"Value 2"}},
}

// kindsMap holds one trigger per kind form, in kindForms' order.
func kindsMap(t *testing.T) []byte {
	t.Helper()
	var conditions, actions []scriptNode
	var triggers []goalTrigger
	add := func(nodes *[]scriptNode, n kindNode, label string) uint32 {
		id := uint32(len(*nodes) + 1)
		*nodes = append(*nodes, scriptNode{label: label, opcode: n.op, id: id, tags: n.tags, values: n.values})
		return id
	}
	for _, f := range kindForms {
		tr := goalTrigger{name: f.name}
		switch {
		case f.action:
			tr.actions = []uint32{add(&actions, f.node, f.name)}
		case f.unresolved:
			tr.left, tr.right = 0xFFFFFFFF, 0xFFFFFFFF
		default:
			tr.left = add(&conditions, f.node, f.name)
			switch {
			case f.self:
				tr.right = tr.left
			case f.other != nil:
				tr.right = add(&conditions, *f.other, f.name)
			default:
				tr.right = add(&conditions, kindNode{0x10002, []uint32{f.constTag}, []uint32{f.constVal}}, f.name)
			}
			if f.swap {
				tr.left, tr.right = tr.right, tr.left
			}
		}
		tr.cmp = f.cmp
		triggers = append(triggers, tr)
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, triggers)}
	return built(t, o, nil, nil).Bytes()
}

// kindsEditor opens b at 1920x1080 with the editor's words in lang ("" or
// "russian"), the host's texts passed through as they are.
func kindsEditor(t *testing.T, b []byte, lang string) (*ui.MapEditor, string) {
	t.Helper()
	e, path := openGoalMap(t, b)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string { return s })
	e.Layout(1920, 1080)
	return e, path
}

// Every shipped form of every census kind reads as its wizard item with its
// parameters, in EN and RU words; only the C2 row over two units and the C10
// pair of unbuilt IDs stay Other.
func TestTriggerWizardReadsEveryCensusKindAsItsItem(t *testing.T) {
	b := kindsMap(t)
	for _, lang := range []string{"", "russian"} {
		book := words.For(lang)
		e, _ := kindsEditor(t, b, lang)
		for i, f := range kindForms {
			e.Select(trigRecord(t, e, "Trigger", i), true)
			rows, kept := e.WizardRowCounts()
			lines := trigLines(e)
			if f.item == "" {
				if rows != 1 || kept != 1 || !strings.Contains(lines, book.Text("editor.wizard.condition")+" "+fmt.Sprintf(book.Text("editor.wizard.other"), "")) {
					t.Errorf("%q %s: %d rows, %d kept; want the row kept as Other:\n%s", lang, f.name, rows, kept, lines)
				}
				continue
			}
			family := "editor.wizard.condition"
			if f.action {
				family = "editor.wizard.action"
			}
			want := book.Text(family) + " " + book.Text(f.item)
			if rows != 1 || kept != 0 || !slices.Contains(e.PopupLines(), want) {
				t.Errorf("%q %s: %d rows, %d kept; want the row %q:\n%s", lang, f.name, rows, kept, want, lines)
				continue
			}
			if lang != "" {
				continue
			}
			for _, line := range f.lines {
				if !slices.Contains(e.PopupLines(), line) {
					t.Errorf("%s: the row lacks %q:\n%s", f.name, line, lines)
				}
			}
		}
	}
}

// A row of each census kind read from a map and not edited writes its nodes
// and pair unchanged: the trigger renamed and applied, then named back and
// applied, leaves the map byte for byte as it was, a Const-typed constant, a
// widened comparison and a group command's slot layout included.
func TestTriggerWizardKeepsEveryUneditedCensusRow(t *testing.T) {
	b := kindsMap(t)
	e, _ := kindsEditor(t, b, "")
	for i, f := range kindForms {
		e.Select(trigRecord(t, e, "Trigger", i), true)
		if _, kept := e.WizardRowCounts(); f.item != "" && kept != 0 {
			t.Fatalf("%s: the row is kept as Other, not read as its item:\n%s", f.name, trigLines(e))
		}
		wizType(t, e, "name", "Renamed")
		wizApply(t, e)
		sc := wizScript(t, e.Document().SourceBytes)
		if sc.Triggers[i].Name != "Renamed" {
			t.Fatalf("%s: the rename was not written:\n%s", f.name, trigLines(e))
		}
		e.Select(trigRecord(t, e, "Trigger", i), true)
		wizType(t, e, "name", f.name)
		wizApply(t, e)
	}
	if !bytes.Equal(e.Document().SourceBytes, b) {
		before, after := wizScript(t, b), wizScript(t, e.Document().SourceBytes)
		t.Fatalf("the map changed: %d conditions and %d actions before, %d and %d after", len(before.Conditions), len(before.Actions), len(after.Conditions), len(after.Actions))
	}
}

// newForm is the shipped form a new wizard row of item label writes: its
// node's operation and slot types, and for a condition the right node (a
// constant of constTag, or with twin a second node like the first) and the
// comparison. command is a group command to choose after the item.
type newForm struct {
	label, command string
	condition      bool
	op             uint32
	tags           []uint32
	twin           bool
	constTag, cmp  uint32
}

var newForms = []newForm{
	{label: "Variable compared with a number", condition: true, op: 19, tags: []uint32{kInt}, constTag: kInt, cmp: 0},
	{label: "Two variables compared", condition: true, op: 19, tags: []uint32{kInt}, twin: true, cmp: 0},
	{label: "Distance from a unit to a point", condition: true, op: 7, tags: []uint32{kUnit, kX, kY}, constTag: kInt, cmp: 5},
	{label: "Sack on a cell is picked up", condition: true, op: 14, tags: []uint32{kX, kY}, constTag: kConst, cmp: 0},
	{label: "Sack lies on a cell", condition: true, op: 14, tags: []uint32{kX, kY}, constTag: kConst, cmp: 0},
	{label: "Any unit of a player near a point", condition: true, op: 15, tags: []uint32{kPlayer, kX, kY}, constTag: kInt, cmp: 5},
	{label: "Unit holds an item", condition: true, op: 17, tags: []uint32{kUnit, kItem}, constTag: kConst, cmp: 0},
	{label: "Unit does not hold an item", condition: true, op: 17, tags: []uint32{kUnit, kItem}, constTag: kConst, cmp: 0},
	{label: "Structure's value", condition: true, op: 21, tags: []uint32{kStructure}, constTag: kInt, cmp: 3},
	{label: "Unit's health", condition: true, op: 4, tags: []uint32{kUnit, kInt}, constTag: kInt, cmp: 2},
	{label: "Unit near unit", condition: true, op: 6, tags: []uint32{kUnit, kUnit}, constTag: kInt, cmp: 5},
	{label: "Lose if either of two units dies", condition: true, op: 18, tags: []uint32{kUnit}, twin: true, cmp: 1},
	{label: "Lose if the unit dies", condition: true, op: 18, tags: []uint32{kUnit}, constTag: kInt, cmp: 2},
	{label: "Unit is alive", condition: true, op: 5, tags: []uint32{kUnit}, constTag: kInt, cmp: 1},
	{label: "Unit is dead", condition: true, op: 5, tags: []uint32{kUnit}, constTag: kInt, cmp: 0},
	{label: "Hero enters area", condition: true, op: 2, tags: []uint32{kUnit, kX, kY, kX, kY}, constTag: kInt, cmp: 0},
	{label: "Whom a unit attacks", condition: true, op: 9, tags: []uint32{kUnit}, constTag: kInt, cmp: 0},
	{label: "Unit with an item: distance to a point", condition: true, op: 16, tags: []uint32{kUnit, kItem, kX, kY}, constTag: kInt, cmp: 5},
	{label: "Add 1 to a variable", op: 8, tags: []uint32{kInt}},
	{label: "Take an item from a unit", op: 13, tags: []uint32{kUnit, kItem}},
	{label: "Order a group", command: "Stand Ground", op: 6, tags: []uint32{kConst, 0, 0, 0, 0, 0, 0, 0, 0, kGroup}},
	{label: "Order a group", command: "Follow", op: 6, tags: []uint32{kConst, kGroup, kUnit, kInt}},
	{label: "Order a group", command: "Attack", op: 6, tags: []uint32{kConst, kUnit, 0, 0, 0, 0, 0, 0, 0, kGroup}},
	{label: "Order a group", command: "Defend", op: 6, tags: []uint32{kConst, kUnit, kInt, 0, 0, 0, 0, 0, 0, kGroup}},
	{label: "Cast a spell from a point at a point", op: 21, tags: []uint32{kX, kY, kX, kY, kInt, kInt}},
	{label: "Cast a spell from a point at a unit", op: 24, tags: []uint32{kX, kY, kUnit, kInt, kInt}},
	{label: "Set the remaining time of effects on a cell", op: 29, tags: []uint32{kX, kY, kInt, kInt}},
	{label: "Set a variable", op: 3, tags: []uint32{kInt, kInt}},
	{label: "Set a unit's property", op: 34, tags: []uint32{kUnit, kInt, kInt}},
	{label: "Give an item to a unit", op: 12, tags: []uint32{kUnit, kItem}},
	{label: "Put a unit in another unit's place", op: 18, tags: []uint32{kUnit, kUnit}},
	{label: "Give all of a unit's items to another unit", op: 28, tags: []uint32{kUnit, kUnit}},
	{label: "Set a spell trap on a cell", op: 25, tags: []uint32{kInt, kInt, kX, kY}},
	{label: "Set the remaining time of effects on a unit", op: 30, tags: []uint32{kUnit, kInt, kInt}},
	{label: "Action 7", op: 7, tags: []uint32{kPlayer, kInt}},
}

// oneTriggerMap holds one trigger "T" that runs instant 4.
func oneTriggerMap(t *testing.T) []byte {
	t.Helper()
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, nil, []goalTrigger{{name: "T", actions: []uint32{1}}})}
	return built(t, o, nil, nil).Bytes()
}

// kindNodeIs reports node n has operation op and slot types tags, every
// other slot empty.
func kindNodeIs(n alm.ScriptNode, op uint32, tags []uint32) bool {
	var want [10]uint32
	copy(want[:], tags)
	return n.Opcode == op && n.Type == want
}

// A new wizard row of each item a census kind reads as, applied with its
// defaults, writes the census's shipped form, and the trigger reads back
// with the row as the same item.
func TestTriggerWizardNewRowOfEveryCensusKindWritesTheShippedForm(t *testing.T) {
	for _, f := range newForms {
		name := f.label + " " + f.command
		e, _ := kindsEditor(t, oneTriggerMap(t), "")
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		key, add := "a1", "+action"
		if f.condition {
			key, add = "c0", ""
		}
		wizPress(t, e, add, 0)
		trigChoose(t, e, wizField(t, e, key+".item"), f.label)
		if f.command != "" {
			trigChoose(t, e, wizField(t, e, key+".p1"), f.command)
		}
		wizApply(t, e)
		sc := wizScript(t, e.Document().SourceBytes)
		tr := sc.Triggers[0]
		if f.condition {
			left, right := wizNode(t, sc.Conditions, tr.Left[0]), wizNode(t, sc.Conditions, tr.Right[0])
			wantRight := kindNodeIs(right, 0x10002, []uint32{f.constTag})
			if f.twin {
				wantRight = kindNodeIs(right, f.op, f.tags) && right.ID != left.ID
			}
			if !kindNodeIs(left, f.op, f.tags) || !wantRight || tr.Cmp[0] != f.cmp {
				t.Errorf("%s: wrote %d %v %v cmp %d against %d %v %v", name, left.Opcode, left.Type, left.Value, tr.Cmp[0], right.Opcode, right.Type, right.Value)
				continue
			}
		} else if n := wizNode(t, sc.Actions, tr.Acts[1]); !kindNodeIs(n, f.op, f.tags) {
			t.Errorf("%s: wrote %d %v %v", name, n.Opcode, n.Type, n.Value)
			continue
		}
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		lines := trigLines(e)
		if rows, kept := e.WizardRowCounts(); rows != 2 || kept != 0 || !strings.Contains(lines, f.label) || f.command != "" && !strings.Contains(lines, "Command "+f.command) {
			t.Errorf("%s: the written trigger reads %d rows, %d kept:\n%s", name, rows, kept, lines)
		}
	}
}

// An edited row read with its sides exchanged writes its item's form: the
// reversed unit-near pair, its radius edited, puts the check on the left.
func TestTriggerWizardEditedReversedPairWritesTheCheckOnTheLeft(t *testing.T) {
	b := kindsMap(t)
	i := slices.IndexFunc(kindForms, func(f kindForm) bool { return f.swap })
	e, _ := kindsEditor(t, b, "")
	e.Select(trigRecord(t, e, "Trigger", i), true)
	wizType(t, e, "c0.p2", "4")
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	tr := sc.Triggers[i]
	left, right := wizNode(t, sc.Conditions, tr.Left[0]), wizNode(t, sc.Conditions, tr.Right[0])
	if left.Opcode != 3 || left.Value[3] != 4 || right.Opcode != 0x10002 || right.Value[0] != 1 || tr.Cmp[0] != 0 {
		t.Fatalf("the edited pair wrote %+v cmp %d %+v", left, tr.Cmp[0], right)
	}
}

// variablesMap holds count constants, one trigger "T" that runs instant 4
// and, with withVariable, an action setting variable 5 that no trigger runs.
func variablesMap(t *testing.T, count int, withVariable bool) []byte {
	t.Helper()
	conditions := make([]scriptNode, count)
	for i := range conditions {
		conditions[i] = scriptNode{label: "Constant", opcode: 0x10002, id: uint32(i + 1), tags: []uint32{kInt}}
	}
	actions := []scriptNode{{label: "win", opcode: 4, id: 1}}
	if withVariable {
		actions = append(actions, scriptNode{label: "set", opcode: 3, id: 2, tags: []uint32{kInt, kInt}, values: []uint32{5, 1}})
	}
	o := synth.ALMOptions{Width: 40, Height: 40, Type7Payload: goalScript(actions, conditions, []goalTrigger{{name: "T", actions: []uint32{1}}})}
	return built(t, o, nil, nil).Bytes()
}

// fieldChoices are the labels wizard field key's list shows.
func fieldChoices(t *testing.T, e *ui.MapEditor, key string) []string {
	t.Helper()
	n := wizField(t, e, key)
	r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
	trigClick(t, e, r, ok, key)
	choices := e.DropdownChoices()
	r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
	trigClick(t, e, r, ok, key)
	return choices
}

// The variable of a variable condition, of Add 1 and of Set a variable
// offers the map's variables and New variable from the host's choices; a
// new variable, then a new condition, are both accepted, the variable on a
// register the condition does not take.
func TestTriggerWizardNewVariableThenANewConditionAreAccepted(t *testing.T) {
	e, _ := kindsEditor(t, variablesMap(t, 2, true), "")
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizPress(t, e, "+action", 0)
	trigChoose(t, e, wizField(t, e, "a1.item"), "Add 1 to a variable")
	wizPress(t, e, "", 0)
	trigChoose(t, e, wizField(t, e, "c0.item"), "Variable compared with a number")
	for _, key := range []string{"a1.p0", "c0.p0"} {
		if got := fieldChoices(t, e, key); !slices.Contains(got, "Variable 5") || !slices.Contains(got, "New variable") {
			t.Fatalf("%s offers %q; want the map's variable 5 and New variable", key, got)
		}
	}
	trigChoose(t, e, wizField(t, e, "a1.item"), "Set a variable")
	if got := fieldChoices(t, e, "a1.p0"); !slices.Contains(got, "Variable 5") || !slices.Contains(got, "New variable") {
		t.Fatalf("Set a variable offers %q", got)
	}
	trigChoose(t, e, wizField(t, e, "a1.p0"), "New variable")
	wizPress(t, e, "c0.item", 0) // Remove: the variable is applied first
	wizApply(t, e)
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizPress(t, e, "", 0)
	trigChoose(t, e, wizField(t, e, "c0.item"), "Unit is dead")
	wizApply(t, e)
	sc := wizScript(t, e.Document().SourceBytes)
	tr := sc.Triggers[0]
	set := wizNode(t, sc.Actions, tr.Acts[1])
	if set.Opcode != 3 || tr.Left[0] == 0 {
		t.Fatalf("the trigger wrote %+v, the action %+v:\n%s", tr, set, trigLines(e))
	}
	register := set.Value[0]
	for k, n := range sc.Conditions {
		if uint32(k) == register {
			t.Fatalf("the new variable %d shares the register of check %d (%+v)", register, k, n)
		}
	}
	if register == 5 || register < uint32(len(sc.Conditions)) {
		t.Fatalf("the new variable took register %d over %d conditions", register, len(sc.Conditions))
	}
}

// When the 100 registers cannot hold a new variable and a new condition both,
// the condition is refused in the editor's words, EN and RU, and nothing is
// written.
func TestTriggerWizardRefusesAConditionOnTheLastRegisterInTheEditorsWords(t *testing.T) {
	for _, c := range []struct {
		profile base.Profile
		sel     int
	}{{base.Profile{Language: "english"}, 0}, {base.Profile{Language: "russian"}, text.SelectorConverting}} {
		x := heroLabelInspector(t, c.profile, c.sel)
		e := openHeroLabelMap(t, x, variablesMap(t, 99, false))
		e.Layout(1920, 1080)
		closeMapPanel(t, e)
		book := words.For(c.profile.Language)
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		wizPress(t, e, "+action", 0)
		trigChoose(t, e, wizField(t, e, "a1.item"), x.inspectionUTF8(book.Text("editor.wizard.a.addvar")))
		wizApply(t, e)
		sc := wizScript(t, e.Document().SourceBytes)
		if n := wizNode(t, sc.Actions, sc.Triggers[0].Acts[1]); n.Opcode != 8 || n.Value[0] != 99 {
			t.Fatalf("%s: the new variable wrote %+v", c.profile.Language, n)
		}
		before := e.Document().SourceBytes
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		wizPress(t, e, "", 0)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlPopupButton, N: 0})
		trigClick(t, e, r, ok, "Apply")
		want := x.inspectionUTF8(fmt.Sprintf(book.Text("editor.wizard.registers"), 99))
		if lines := trigLines(e); !strings.Contains(lines, want) || strings.Contains(lines, "would share") || !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatalf("%s: the condition on the last register was not refused in the editor's words:\n%s", c.profile.Language, lines)
		}
	}
}

// At 1920x1080 the grouped item lists of a condition row and of an action
// row open whole: every heading and item shows at once inside the window,
// in EN and RU.
func TestTriggerWizardItemListsOpenWholeOnTheSmallestScreen(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e, _ := kindsEditor(t, oneTriggerMap(t), lang)
		e.Select(trigRecord(t, e, "Trigger", 0), true)
		wizPress(t, e, "", 0)
		wizPress(t, e, "+action", 0)
		for _, key := range []string{"c0.item", "a1.item"} {
			n := wizField(t, e, key)
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
			trigClick(t, e, r, ok, key)
			choices := e.DropdownChoices()
			book := words.For(lang)
			if !slices.Contains(choices, fmt.Sprintf(book.Text("editor.wizard.subject"), book.Text("editor.wizard.s.units"))) {
				t.Fatalf("%q %s: the list has no Units heading: %q", lang, key, choices)
			}
			for row := range choices {
				rr, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlDropdownRow, N: row})
				if !ok || rr.Max.Y > 1080 || rr.Min.Y < 0 {
					r0, _ := e.FindControl(ui.ControlQuery{Role: ui.ControlDropdownRow, N: 0})
					t.Fatalf("%q %s: row %d of %d (%q) does not show at once: %v %v; field %v, first row %v", lang, key, row, len(choices), choices[row], rr, ok, r, r0)
				}
			}
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
			trigClick(t, e, r, ok, key)
		}
	}
}
