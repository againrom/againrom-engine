package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// TestUnitSpeedSightFacingSetClearRefuseAndUndo: a new placement leaves the
// three carrier bytes at 0; each field writes its own byte as one edit,
// clears back to the same bytes, refuses a value outside its range without a
// write, and Undo restores the record.
func TestUnitSpeedSightFacingSetClearRefuseAndUndo(t *testing.T) {
	e := freshRecords(t)
	u, err := e.NewUnitRecord(4*256+128, 5*256+128, 40, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if u[alm.UnitSpeed] != 0 || u[alm.UnitFacing] != 0 || u[alm.UnitSight] != 0 {
		t.Fatalf("new record carries % x at +0x30, +0x33, +0x34", []byte{u[alm.UnitSpeed], u[alm.UnitFacing], u[alm.UnitSight]})
	}
	if _, err := e.PlaceUnit(u); err != nil {
		t.Fatal(err)
	}
	placed := e.Bytes()
	for _, c := range []struct {
		key        string
		offset     int
		value, bad int64
	}{
		{"speed", alm.UnitSpeed, 37, 256},
		{"sight", alm.UnitSight, alm.SightCells, alm.SightCells + 1},
		{"facing", alm.UnitFacing, alm.FacingDirections, alm.FacingDirections + 1},
	} {
		if err := e.SetField(mapedit.KindUnit, 0, c.key, c.bad); err == nil || !bytes.Equal(e.Bytes(), placed) {
			t.Fatalf("%s %d was accepted", c.key, c.bad)
		}
		if err := e.SetField(mapedit.KindUnit, 0, c.key, -1); err == nil || !bytes.Equal(e.Bytes(), placed) {
			t.Fatalf("%s -1 was accepted", c.key)
		}
		if err := e.SetField(mapedit.KindUnit, 0, c.key, c.value); err != nil {
			t.Fatal(err)
		}
		rec, _ := e.UnitRecord(0)
		if got, _ := e.FieldValue(mapedit.KindUnit, 0, c.key); got != c.value || int64(rec[c.offset]) != c.value {
			t.Fatalf("%s reads %d, byte %d, want %d", c.key, got, rec[c.offset], c.value)
		}
		m, _ := e.Map()
		tail := m.Units[0].Tail
		if decoded := map[string]uint8{"speed": tail.Speed, "sight": tail.Sight, "facing": tail.Facing}[c.key]; int64(decoded) != c.value {
			t.Fatalf("%s decodes as %d, want %d", c.key, decoded, c.value)
		}
		if !e.Undo() || !bytes.Equal(e.Bytes(), placed) {
			t.Fatalf("Undo of %s did not restore the record", c.key)
		}
		if err := e.SetField(mapedit.KindUnit, 0, c.key, c.value); err != nil {
			t.Fatal(err)
		}
		if err := e.SetField(mapedit.KindUnit, 0, c.key, 0); err != nil || !bytes.Equal(e.Bytes(), placed) {
			t.Fatalf("clearing %s left % x", c.key, e.Bytes())
		}
	}
}
