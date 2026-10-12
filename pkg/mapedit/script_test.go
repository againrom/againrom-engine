package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
)

const (
	testNode    = 796
	testTrigger = 184
)

// testNodeRecord builds one 796-byte node by the layout of ALM-TRIG-045.
func testNodeRecord(opcode, id uint32, params ...[2]uint32) []byte {
	rec := make([]byte, testNode)
	binary.LittleEndian.PutUint32(rec[0x40:], opcode)
	binary.LittleEndian.PutUint32(rec[0x44:], id)
	for i, p := range params {
		binary.LittleEndian.PutUint32(rec[0x4c+4*i:], p[1])
		binary.LittleEndian.PutUint32(rec[0x74+4*i:], p[0])
	}
	for k := 0x9c; k < 0x9c+8; k++ {
		rec[k] = byte(k) // carried bytes no field names
	}
	return rec
}

func testArray(records ...[]byte) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(len(records)))
	for _, r := range records {
		out = append(out, r...)
	}
	return out
}

// scriptFixture is a 16x16 map whose script holds two actions (a drop at 5,6
// and a win), two conditions (a unit-alive check on unit 7 and a constant)
// and one trigger.
func scriptFixture(actions ...[]byte) []byte {
	return synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: scriptPayload(actions...), Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40},
	}})
}

func scriptPayload(actions ...[]byte) []byte {
	if actions == nil {
		actions = [][]byte{testNodeRecord(0x10002, 1, [2]uint32{5, 5}, [2]uint32{6, 6}), testNodeRecord(4, 2)}
	}
	trigger := make([]byte, testTrigger)
	copy(trigger, "Win")
	binary.LittleEndian.PutUint32(trigger[0x80:], 1)
	binary.LittleEndian.PutUint32(trigger[0x84:], 2)
	binary.LittleEndian.PutUint32(trigger[0x98:], 2)
	payload := append(testArray(actions...), testArray(
		testNodeRecord(5, 1, [2]uint32{4, 7}),
		testNodeRecord(0x10002, 2, [2]uint32{7, 0}))...)
	return append(payload, testArray(trigger)...)
}

// firstCondition is the offset of the first condition node in b: the first
// 796-byte node whose opcode is 5.
func firstCondition(b []byte) int {
	return bytes.Index(b, []byte{5, 0, 0, 0, 1, 0, 0, 0}) - 0x40
}

func diffRegion(a, b []byte) (lo, hi int) {
	lo, hi = -1, -1
	for i := range a {
		if i < len(b) && a[i] != b[i] {
			if lo < 0 {
				lo = i
			}
			hi = i + 1
		}
	}
	return lo, hi
}

func TestScriptNodesAreRowsOfTheSectionTable(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	if e.Count(mapedit.KindAction) != 2 || e.Count(mapedit.KindCondition) != 2 {
		t.Fatalf("counts %d actions, %d conditions", e.Count(mapedit.KindAction), e.Count(mapedit.KindCondition))
	}
	if !bytes.Equal(e.Bytes(), b) {
		t.Fatal("the unedited map does not write the bytes it was given")
	}
	fields, err := e.Fields(mapedit.KindCondition, 0)
	if err != nil || len(fields) != 1 || fields[0].Key != "value0" || fields[0].Check != mapedit.CheckScriptUnit {
		t.Fatalf("fields of the unit-alive check: %+v, %v", fields, err)
	}
	if fields, _ = e.Fields(mapedit.KindAction, 0); len(fields) != 2 || fields[0].Max != 255 {
		t.Fatalf("fields of the drop action: %+v", fields)
	}
}

func TestSetScriptTargetWritesOneWordAndUndoRestoresTheBytes(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	if err := e.SetField(mapedit.KindCondition, 0, "value0", 9); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	lo, hi := diffRegion(b, after)
	if len(after) != len(b) || lo != 0x4c+firstCondition(b) || hi-lo > 4 || binary.LittleEndian.Uint32(after[lo:]) != 9 {
		t.Fatalf("the edit changed bytes %d..%d of %d, want one word", lo, hi, len(after))
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) {
		t.Fatal("undo did not restore the bytes")
	}
	if !e.Redo() || !bytes.Equal(e.Bytes(), after) {
		t.Fatal("redo did not restore the edited bytes")
	}
}

func TestSetScriptFieldRefusesWhatItMustAndWritesNothing(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	for name, c := range map[string]func() error{
		"unit id 0":           func() error { return e.SetField(mapedit.KindCondition, 0, "value0", 0) },
		"a slot with no type": func() error { return e.SetField(mapedit.KindCondition, 0, "value3", 1) },
		"drop X over 255":     func() error { return e.SetField(mapedit.KindAction, 0, "value0", 256) },
		"an unknown node":     func() error { return e.SetField(mapedit.KindCondition, 5, "value0", 1) },
		"a repeated key": func() error {
			return e.SetFields(mapedit.KindAction, 0, mapedit.FieldSet{Key: "value0", Value: 1}, mapedit.FieldSet{Key: "value0", Value: 2})
		},
		"no field": func() error { return e.SetFields(mapedit.KindAction, 0) },
	} {
		if err := c(); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused edit wrote or recorded something")
	}
}

func TestSetDropCellIsOneHistoryEntry(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	if err := e.SetDropCell(9, 11); err != nil {
		t.Fatal(err)
	}
	after := e.Bytes()
	if len(after) != len(b) || bytes.Equal(after, b) {
		t.Fatal("the drop cell did not move in place")
	}
	// Value 0 and value 1 are the only words that differ.
	for i := range b {
		if b[i] != after[i] && !inDropWords(b, i) {
			t.Fatalf("byte %d changed outside the drop node's two values", i)
		}
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("one undo did not restore both coordinates")
	}
}

// inDropWords reports whether byte i lies in value 0 or value 1 of the first
// action node of the fixture.
func inDropWords(b []byte, i int) bool {
	first := bytes.Index(b, []byte{0x02, 0x00, 0x01, 0x00}) // opcode 0x10002 at node +0x40
	return first >= 0 && i >= first+0xc && i < first+0xc+8
}

func TestSetDropCellAddsAnActionWhenTheMapHasNone(t *testing.T) {
	b := scriptFixture([][]byte{testNodeRecord(4, 3)}...)
	e := newEditor(t, b)
	if err := e.SetDropCell(9, 11); err != nil {
		t.Fatal(err)
	}
	if e.Count(mapedit.KindAction) != 2 || e.Count(mapedit.KindCondition) != 2 {
		t.Fatalf("counts %d actions, %d conditions", e.Count(mapedit.KindAction), e.Count(mapedit.KindCondition))
	}
	rec, err := e.Record(mapedit.KindAction, 1)
	if err != nil || binary.LittleEndian.Uint32(rec[0x40:]) != 0x10002 || binary.LittleEndian.Uint32(rec[0x44:]) != 4 ||
		binary.LittleEndian.Uint32(rec[0x4c:]) != 9 || binary.LittleEndian.Uint32(rec[0x50:]) != 11 {
		t.Fatalf("the new action: %v %v", rec[0x40:0x58], err)
	}
	if got := e.Bytes(); len(got) != len(b)+testNode {
		t.Fatalf("length %d, want %d", len(got), len(b)+testNode)
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) {
		t.Fatal("undo did not remove the new action")
	}
	if !e.Redo() || e.Count(mapedit.KindAction) != 2 {
		t.Fatal("redo did not restore the new action")
	}
	if err := e.SetDropCell(300, 1); err == nil {
		t.Fatal("a drop cell over 255 was accepted")
	}
}

func TestPlaceAConditionLeavesTheOtherArraysUntouched(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	id, err := e.NextScriptID(mapedit.KindCondition)
	if err != nil || id != 3 {
		t.Fatalf("next condition id %d, %v", id, err)
	}
	rec, err := mapedit.NewScriptNode("Protect unit", 18, id, mapedit.ScriptParam{Type: 4, Value: 7})
	if err != nil {
		t.Fatal(err)
	}
	if i, err := e.Place(mapedit.KindCondition, rec); err != nil || i != 2 {
		t.Fatalf("place answered %d, %v", i, err)
	}
	after := e.Bytes()
	if len(after) != len(b)+testNode || e.Count(mapedit.KindCondition) != 3 || e.Count(mapedit.KindAction) != 2 {
		t.Fatalf("length %d, conditions %d, actions %d", len(after), e.Count(mapedit.KindCondition), e.Count(mapedit.KindAction))
	}
	// The trigger array is the last 188 bytes in both buffers.
	tail := 4 + testTrigger
	if !bytes.Equal(after[len(after)-tail:], b[len(b)-tail:]) {
		t.Fatal("the trigger array changed")
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) {
		t.Fatal("undo did not restore the bytes")
	}
}

func TestAScriptThatDoesNotTileIsNotEdited(t *testing.T) {
	for name, payload := range map[string][]byte{
		"a trailing byte":   append(scriptPayload(), 0),
		"a short payload":   scriptPayload()[:len(scriptPayload())-1],
		"a count past EOF":  append(binary.LittleEndian.AppendUint32(nil, 1<<30), make([]byte, 8)...),
		"only a count word": make([]byte, 4),
	} {
		b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{{X: 256, Y: 256, ClassID: 40}}})
		e := newEditor(t, b)
		if e.Count(mapedit.KindAction) != 0 || e.SetDropCell(3, 3) == nil || e.CanUndo() {
			t.Errorf("%s: the script was edited", name)
		}
		if !bytes.Equal(e.Bytes(), b) {
			t.Errorf("%s: bytes changed", name)
		}
	}
}
