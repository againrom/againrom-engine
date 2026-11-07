package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// testTriggerRecord lays out one trigger record by hand at the offsets of
// ALM-TRIG-047: the name, the pairs as left and right words from +0x80, the
// actions from +0x98, the comparison codes from +0xa8 and once at +0xb4.
func testTriggerRecord(name string, left, right, cmp [3]uint32, acts [4]uint32, once uint32) []byte {
	rec := make([]byte, testTrigger)
	copy(rec, name)
	for k := range 3 {
		binary.LittleEndian.PutUint32(rec[0x80+8*k:], left[k])
		binary.LittleEndian.PutUint32(rec[0x84+8*k:], right[k])
		binary.LittleEndian.PutUint32(rec[0xa8+4*k:], cmp[k])
	}
	for k := range 4 {
		binary.LittleEndian.PutUint32(rec[0x98+4*k:], acts[k])
	}
	binary.LittleEndian.PutUint32(rec[0xb4:], once)
	return rec
}

func TestNewScriptTriggerLaysOutTheRecord(t *testing.T) {
	for name, c := range map[string]struct {
		in   alm.ScriptTrigger
		want []byte
	}{
		"every slot": {alm.ScriptTrigger{Name: "Every", Left: []uint32{1, 2, 3}, Right: []uint32{4, 5, 6}, Cmp: []uint32{0, 5, 2},
			Acts: []uint32{7, 8, 9, 5000}, Once: 1},
			testTriggerRecord("Every", [3]uint32{1, 2, 3}, [3]uint32{4, 5, 6}, [3]uint32{0, 5, 2}, [4]uint32{7, 8, 9, 5000}, 1)},
		"short slices are empty slots": {alm.ScriptTrigger{Name: "Short", Left: []uint32{9}, Right: []uint32{10, 11}, Acts: []uint32{12}},
			testTriggerRecord("Short", [3]uint32{9}, [3]uint32{10, 11}, [3]uint32{}, [4]uint32{12}, 0)},
		"no slot": {alm.ScriptTrigger{Name: "Trigger", Once: 1}, testTriggerRecord("Trigger", [3]uint32{}, [3]uint32{}, [3]uint32{}, [4]uint32{}, 1)},
	} {
		got, err := mapedit.NewScriptTrigger(c.in)
		if err != nil || !bytes.Equal(got, c.want) {
			t.Errorf("%s: %v, % x", name, err, got)
		}
	}
}

// TestDecodedTriggersEncodeToTheirRecords reads a record the format decodes
// and builds it again: the slot bytes are the record's.
func TestDecodedTriggersEncodeToTheirRecords(t *testing.T) {
	e := newEditor(t, scriptFixture())
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil || len(sc.Triggers) != 1 {
		t.Fatalf("script %v, %d triggers", err, len(sc.Triggers))
	}
	raw, err := e.Record(mapedit.KindTrigger, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := mapedit.NewScriptTrigger(sc.Triggers[0])
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatalf("%v: built % x, the record % x", err, got, raw)
	}
}

func TestTriggerWritersRefuseWhatTheRecordCannotHold(t *testing.T) {
	b := scriptFixture()
	e := newEditor(t, b)
	for name, c := range map[string]struct {
		in   alm.ScriptTrigger
		want mapedit.ScriptLimit
	}{
		"four pairs":   {alm.ScriptTrigger{Left: []uint32{1, 1, 1, 1}, Right: []uint32{2, 2, 2, 2}}, mapedit.LimitPairs},
		"four codes":   {alm.ScriptTrigger{Cmp: []uint32{0, 0, 0, 0}}, mapedit.LimitPairs},
		"five actions": {alm.ScriptTrigger{Acts: []uint32{1, 2, 1, 2, 1}}, mapedit.LimitActions},
		"code 6":       {alm.ScriptTrigger{Left: []uint32{1}, Right: []uint32{2}, Cmp: []uint32{6}}, mapedit.LimitCompare},
		"left id 5001": {alm.ScriptTrigger{Left: []uint32{5001}, Right: []uint32{2}}, mapedit.LimitID},
		"act id 5001":  {alm.ScriptTrigger{Acts: []uint32{2, 5001}}, mapedit.LimitID},
	} {
		rec, err := mapedit.NewScriptTrigger(c.in)
		var limit *mapedit.ScriptLimitError
		if rec != nil || !errors.As(err, &limit) || limit.Limit != c.want {
			t.Errorf("%s: built %d bytes, %v", name, len(rec), err)
		}
		err = e.SetTriggerSlots(0, c.in)
		if !errors.As(err, &limit) || limit.Limit != c.want {
			t.Errorf("%s: the field setter answered %v", name, err)
		}
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused trigger wrote or recorded something")
	}
}

// TestSetTriggerSlotsKeepsAHeldCode writes another slot of a trigger whose
// first comparison code lies outside the alphabet: the code the record holds
// stays, the changed slot is one edit, and undo restores the bytes.
func TestSetTriggerSlotsKeepsAHeldCode(t *testing.T) {
	payload := scriptPayload()
	binary.LittleEndian.PutUint32(payload[len(payload)-testTrigger+0xa8:], 0xffffffff)
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: payload, Units: []synth.ALMUnit{
		{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40},
	}})
	e := newEditor(t, b)
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	sc, err := m.Script()
	if err != nil {
		t.Fatal(err)
	}
	tr := sc.Triggers[0]
	tr.Acts = []uint32{2, 1}
	if err := e.SetTriggerSlots(0, tr); err != nil {
		t.Fatal(err)
	}
	raw, err := e.Record(mapedit.KindTrigger, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := testTriggerRecord("Win", [3]uint32{1}, [3]uint32{2}, [3]uint32{0xffffffff}, [4]uint32{2, 1}, 0)
	if !bytes.Equal(raw, want) {
		t.Fatalf("the record % x, want % x", raw, want)
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("undo left another map or more history")
	}
	tr.Cmp = []uint32{0xfffffffe}
	var limit *mapedit.ScriptLimitError
	if err := e.SetTriggerSlots(0, tr); !errors.As(err, &limit) || limit.Limit != mapedit.LimitCompare {
		t.Fatalf("a changed code outside the alphabet answered %v", err)
	}
}
