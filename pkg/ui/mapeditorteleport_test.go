package ui

import (
	"fmt"
	"image"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/scriptprofile"
)

// teleportEditor is the wizard fixture on a second-game map whose action
// record 4 is node 44 of 53.alm as R2-ASSET-386 states it: action 25 with
// event 23, power 255, cell (15, 121), source (15, 121), target (13, 22).
func teleportEditor(t *testing.T, node InspectionScriptNode) *MapEditor {
	t.Helper()
	e, _ := wizardChoiceEditor(t)
	e.Layout(1920, 1080)
	e.doc.Script.Limits = scriptprofile.Second
	e.doc.Script.Catalog = secondCatalog()
	node.Record = 4
	e.doc.Script.Actions[0] = node
	e.Select(4, false)
	e.loadObject()
	if v, ok := e.scriptPopup(); !ok || v.node == nil || v.node.ID != node.ID {
		t.Fatal("no action popup")
	}
	return e
}

var mission53Teleport = InspectionScriptNode{ID: 44, Opcode: 25,
	Types: [10]uint32{1, 1, 5, 6, 5, 6, 5, 6}, Values: [10]uint32{23, 255, 15, 121, 15, 121, 13, 22}}

func objField(e *MapEditor, key string) int {
	return slices.IndexFunc(e.obj.fields, func(f InspectionField) bool { return f.Key == key })
}

// Mission 53's action 25 node reads as a teleport: the operation is named a
// teleport, the entry cell is (15, 121), the exit (13, 22), the source shown
// as the entry cell and the power named unread; the exit changes by a cell
// pick and Apply sends only the exit's two words.
func TestSecondGameActionTwentyFiveEventTwentyThreeReadsAsTeleport(t *testing.T) {
	e := teleportEditor(t, mission53Teleport)
	if got, want := e.nodeTitle(false, mission53Teleport), "#44 "+e.word("editor.second.action.25.teleport"); got != want {
		t.Errorf("title %q, want %q", got, want)
	}
	params := e.nodeParams(false, 25, mission53Teleport.Types, mission53Teleport.Values)
	want := []string{e.word("editor.role.event") + " 23", e.word("editor.role.unreadpower") + " 255",
		e.word("editor.role.entry") + " 15,121",
		e.word("editor.role.source") + " " + fmt.Sprintf(e.word("editor.node.follows"), "15,121", e.word("editor.role.entry")),
		e.word("editor.role.exit") + " 13,22"}
	if !slices.Equal(params, want) {
		t.Errorf("parameters %q, want %q", params, want)
	}
	op := objField(e, "opcode")
	if op < 0 || e.obj.fields[op].Value != formChoice(25, 23) || choiceText(e.obj.fields[op], e.obj.buf[op]) != e.word("editor.second.action.25.teleport") {
		t.Fatalf("operation field %+v", e.obj.fields)
	}
	entry, source, exit := objField(e, "cell2"), objField(e, "cell4"), objField(e, "cell6")
	if entry < 0 || source < 0 || exit < 0 || e.obj.buf[entry] != "15,121" || e.obj.buf[exit] != "13,22" ||
		e.obj.fields[entry].ReadOnly || e.obj.fields[exit].ReadOnly || !e.obj.fields[source].ReadOnly {
		t.Fatalf("cell fields %+v / %q", e.obj.fields, e.obj.buf)
	}
	e.obj.pickField = exit
	e.scriptCellPicked(image.Pt(20, 30))
	v, _ := e.scriptPopup()
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	if w := []InspectionFieldValue{{Key: "value6", Value: 20}, {Key: "value7", Value: 30}}; !slices.Equal(got, w) {
		t.Fatalf("Apply sends %+v, want %+v", got, w)
	}
}

// A teleport built from the operation list on another node writes the
// original's operands: event 23, power 255, the entry cell twice and the
// exit, as action 25 (R2-ENGINE-1775, R2-ASSET-386).
func TestSecondGameTeleportBuiltFromTheOperationList(t *testing.T) {
	e := teleportEditor(t, InspectionScriptNode{ID: 11, Opcode: 4})
	op := objField(e, "opcode")
	if op < 0 || !slices.ContainsFunc(e.obj.fields[op].Choices, func(c InspectionChoice) bool { return c.Value == formChoice(25, 23) }) {
		t.Fatalf("the operation list offers no teleport: %+v", e.obj.fields)
	}
	e.obj.buf[op] = strconv.FormatInt(formChoice(25, 23), 10)
	e.retypeNode()
	if e.obj.types != mission53Teleport.Types {
		t.Fatalf("slots %v", e.obj.types)
	}
	e.obj.buf[objField(e, "cell2")] = "15,121"
	e.obj.buf[objField(e, "cell6")] = "13,22"
	v, _ := e.scriptPopup()
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	if got[len(got)-1] != (InspectionFieldValue{Key: "opcode", Value: 25}) {
		t.Fatalf("Apply sends %+v, the operation not last as 25", got)
	}
	values := map[string]int64{}
	for _, x := range got {
		values[x.Key] = x.Value
	}
	for s := range 8 {
		key := "value" + string(rune('0'+s))
		if values["type"+string(rune('0'+s))] != int64(mission53Teleport.Types[s]) || values[key] != int64(mission53Teleport.Values[s]) {
			t.Fatalf("Apply sends %+v, want the slots %v of %v", got, mission53Teleport.Types, mission53Teleport.Values)
		}
	}
	// The plain operation picked again leaves the teleport.
	e.obj.buf[op] = "25"
	e.retypeNode()
	if e.obj.buf[objField(e, "value0")] != "0" || objField(e, "cell4") < 0 || e.obj.fields[objField(e, "cell4")].ReadOnly {
		t.Fatalf("plain action 25 fields %+v / %q", e.obj.fields, e.obj.buf)
	}
}

// A plain action 25 node whose cell is set, turned into a teleport from the
// operation list, writes the form's power and the entry cell as its source,
// though its slots already have the form's types (R2-ENGINE-1775,
// R2-ASSET-387).
func TestSecondGameTeleportFromPlainActionTwentyFiveWritesDefaults(t *testing.T) {
	plain := InspectionScriptNode{ID: 12, Opcode: 25, Types: mission53Teleport.Types, Values: [10]uint32{0, 0, 15, 121, 0, 0, 0, 0}}
	e := teleportEditor(t, plain)
	op := objField(e, "opcode")
	e.obj.buf[op] = strconv.FormatInt(formChoice(25, 23), 10)
	e.retypeNode()
	e.obj.buf[objField(e, "cell6")] = "13,22"
	v, _ := e.scriptPopup()
	got, err := e.scriptValues(v)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]int64{}
	for _, x := range got {
		values[x.Key] = x.Value
	}
	for s, want := range []int64{23, 255, 15, 121, 15, 121, 13, 22} {
		key := "value" + string(rune('0'+s))
		if s == 2 || s == 3 {
			continue // unchanged: the node already holds the entry cell
		}
		if values[key] != want {
			t.Fatalf("Apply sends %+v, want %s = %d", got, key, want)
		}
	}
}
