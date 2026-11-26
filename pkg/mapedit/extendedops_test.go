package mapedit_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/scriptprofile"
)

// The one refusal check counts a row's nodes against the script held before:
// a script that adds a node of a row is refused with the row, one that holds
// no new node of any row is not.
func TestExtendedOnlyRefusesARiseOfARow(t *testing.T) {
	teleport := alm.ScriptNode{Opcode: scriptprofile.TeleportCellOp}
	global := alm.ScriptNode{Opcode: scriptprofile.ReadGlobalOp}
	held := alm.Script{Actions: []alm.ScriptNode{teleport}}
	limit := mapedit.ExtendedOnly(scriptprofile.First, alm.Script{}, held)
	if limit == nil || limit.Limit != mapedit.LimitExtendedOnly || limit.Op.Name != "teleport" || limit.Got != 1 || limit.Max != 0 || !limit.Lifted() {
		t.Fatalf("a teleport node answered %+v", limit)
	}
	if limit := mapedit.ExtendedOnly(scriptprofile.First, held, held); limit != nil {
		t.Fatalf("a script that adds no node answered %+v", limit)
	}
	more := alm.Script{Conditions: []alm.ScriptNode{global}, Actions: []alm.ScriptNode{teleport, teleport}}
	if limit := mapedit.ExtendedOnly(scriptprofile.First, held, more); limit == nil || limit.Op.Name != "global" || limit.Got != 1 {
		t.Fatalf("the first row that rises is the table's first: %+v", limit)
	}
	plain := alm.Script{Actions: []alm.ScriptNode{{Opcode: 2}}, Conditions: []alm.ScriptNode{{Opcode: scriptprofile.TeleportCellOp}}}
	if limit := mapedit.ExtendedOnly(scriptprofile.First, alm.Script{}, plain); limit != nil {
		t.Fatalf("a plain message and a check of an action's opcode answered %+v", limit)
	}
}

func TestExtendedOnlyMatchesCompilerDialogueBinding(t *testing.T) {
	thirdSlot := alm.ScriptNode{Label: "elder", Opcode: 2, ID: 1,
		Type: [alm.ScriptParams]uint32{1, 0, 1}, Value: [alm.ScriptParams]uint32{7, 0, 1}}
	unsetSecond := alm.ScriptNode{Label: "elder", Opcode: 2, ID: 1,
		Type: [alm.ScriptParams]uint32{1, 0}, Value: [alm.ScriptParams]uint32{7, 1}}
	if limit := mapedit.ExtendedOnly(scriptprofile.Second, alm.Script{}, alm.Script{Actions: []alm.ScriptNode{thirdSlot}}); limit != nil {
		t.Fatalf("ROM2 non-dialogue [1,0,1]/[7,0,1] was refused: %+v", limit)
	}
	if limit := mapedit.ExtendedOnly(scriptprofile.Second, alm.Script{}, alm.Script{Actions: []alm.ScriptNode{unsetSecond}}); limit == nil || limit.Op.Name != "dialogue" {
		t.Fatalf("ROM2 dialogue [1,0]/[7,1] was not refused: %+v", limit)
	}
	if limit := mapedit.ExtendedOnly(scriptprofile.First, alm.Script{}, alm.Script{Actions: []alm.ScriptNode{thirdSlot}}); limit == nil || limit.Op.Name != "dialogue" {
		t.Fatalf("ROM1 dialogue [1,0,1]/[7,0,1] was not refused: %+v", limit)
	}
	if limit := mapedit.ExtendedOnly(scriptprofile.First, alm.Script{}, alm.Script{Actions: []alm.ScriptNode{unsetSecond}}); limit != nil {
		t.Fatalf("ROM1 non-dialogue [1,0]/[7,1] was refused: %+v", limit)
	}
}

func TestExtendedOnlyRefusesAddedNodeWhenAnotherIsRemoved(t *testing.T) {
	for _, p := range []scriptprofile.Profile{scriptprofile.First, scriptprofile.Second} {
		readGlobal := alm.ScriptNode{Opcode: scriptprofile.ReadGlobalOp, ID: 1}
		setGlobal := alm.ScriptNode{Opcode: scriptprofile.SetGlobalOp, ID: 2}
		if limit := mapedit.ExtendedOnly(p, alm.Script{Conditions: []alm.ScriptNode{readGlobal}}, alm.Script{Actions: []alm.ScriptNode{setGlobal}}); limit == nil || limit.Op.Name != "global" || limit.Got != 1 {
			t.Fatalf("ops %d: removing global check and adding global action answered %+v", p.Ops, limit)
		}
		tp1 := alm.ScriptNode{Opcode: scriptprofile.TeleportCellOp, ID: 1}
		tp2 := alm.ScriptNode{Opcode: scriptprofile.TeleportCellOp, ID: 2}
		if limit := mapedit.ExtendedOnly(p, alm.Script{Actions: []alm.ScriptNode{tp1}}, alm.Script{Actions: []alm.ScriptNode{tp2}}); limit == nil || limit.Op.Name != "teleport" || limit.Got != 1 {
			t.Fatalf("ops %d: removing teleport 1 and adding teleport 2 answered %+v", p.Ops, limit)
		}
		both := alm.Script{Conditions: []alm.ScriptNode{readGlobal}, Actions: []alm.ScriptNode{setGlobal}}
		if limit := mapedit.ExtendedOnly(p, both, alm.Script{Actions: []alm.ScriptNode{setGlobal}}); limit != nil {
			t.Fatalf("ops %d: removing a global check with no addition answered %+v", p.Ops, limit)
		}
	}
}

// A test-only row of the table is refused by the fold with no other code.
func TestFoldRefusesATestOnlyExtendedRow(t *testing.T) {
	defer func(rows []scriptprofile.ExtendedOp) { scriptprofile.ExtendedOps = rows }(scriptprofile.ExtendedOps)
	probe := scriptprofile.ExtendedOp{Name: "probe", Ops: []scriptprofile.OpID{{ID: 0x41ff}}, Word: "editor.script.limit.teleport", Counted: true}
	scriptprofile.ExtendedOps = append(slices.Clone(scriptprofile.ExtendedOps), probe)
	e := newEditor(t, synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload(
		testNodeRecord(0x10002, 1, [2]uint32{5, 5}, [2]uint32{6, 6}), testNodeRecord(4, 2), testNodeRecord(0x41ff, 3))}))
	if err := e.ExtendScript(); err != nil {
		t.Fatal(err)
	}
	held := e.Bytes()
	var limit *mapedit.ScriptLimitError
	if err := e.FoldScript(); !errors.As(err, &limit) || limit.Op.Name != "probe" || limit.Got != 1 || !bytes.Equal(e.Bytes(), held) {
		t.Fatalf("a fold with the test-only row's node answered %v", err)
	}
}
