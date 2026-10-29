package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
)

func TestUnitLookMasksPreserveUnrelatedBytes(t *testing.T) {
	e, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{ClassID: 3, ClassSubID: 0xab80, Flags: 0x5a123480}}}))
	if err != nil {
		t.Fatal(err)
	}
	raw := e.Bytes()
	before, _ := e.UnitRecord(0)
	if err := e.SetFields(mapedit.KindUnit, 0, mapedit.FieldSet{Key: "face", Value: 7}, mapedit.FieldSet{Key: "sex", Value: 1}); err != nil {
		t.Fatal(err)
	}
	got, _ := e.UnitRecord(0)
	want := append([]byte(nil), before...)
	want[0x0a] = 0x87
	binary.LittleEndian.PutUint32(want[0x0c:], 0x5a123484)
	if !bytes.Equal(got, want) {
		t.Fatalf("look touched unrelated bytes: %x / %x", got, want)
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), raw) {
		t.Fatal("look Undo is not exact")
	}
	if err := e.SetUnitField(0, "face", 128); err == nil || !bytes.Equal(e.Bytes(), raw) {
		t.Fatal("out-of-range face changed the map")
	}
}
