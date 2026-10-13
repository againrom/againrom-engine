package mapedit

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
)

func TestNewMapOpensAsAnyMapAndWritesTheConstructorsBytes(t *testing.T) {
	spec := alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64}
	doc, err := alm.NewMap(spec)
	if err != nil {
		t.Fatal(err)
	}
	e, err := NewMap(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Bytes(), doc.Write()) {
		t.Fatal("the session's bytes are not the constructor's")
	}
	// The ordinary route: New of the saved bytes is accepted and writes them back.
	again, err := New(e.Bytes())
	if err != nil || !bytes.Equal(again.Bytes(), e.Bytes()) {
		t.Fatalf("reopening the new map: %v", err)
	}
	if e.CanUndo() || e.UnitCount() != 0 || e.Count(KindStructure) != 0 || e.Count(KindLoot) != 0 {
		t.Fatal("a new map holds history or records")
	}
	if _, err := NewMap(alm.NewMapSpec{Width: 7, Height: 32, Tile: 0x11}); err == nil {
		t.Fatal("a 7-wide map was built")
	}
}

func TestNewMapTakesUnitsStructuresAndSacksThroughTheSectionTable(t *testing.T) {
	e, err := NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	blank := e.Bytes()
	rec, err := e.NewUnitRecord(5<<8|128, 6<<8|128, 27, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	i, err := e.PlaceUnit(rec)
	if err != nil || i != 0 {
		t.Fatalf("PlaceUnit: %d %v", i, err)
	}
	rec2, err := e.NewUnitRecord(7<<8|128, 6<<8|128, 27, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.PlaceUnit(rec2); err != nil {
		t.Fatal(err)
	}
	m, err := e.Map()
	if err != nil || len(m.Units) != 2 {
		t.Fatalf("map after two units: %v, %d units", err, len(m.Units))
	}
	u := m.Units[1]
	if u.X>>8 != 7 || u.Y>>8 != 6 || u.ClassID != 27 || u.ClassSubID != 3 || u.Owner != 2 || u.UnitID != 2 || m.Units[0].UnitID != 1 {
		t.Fatalf("unit %+v", u)
	}
	if u.HasCurrentHP || u.DefID != 0 || u.GroupID != 0 {
		t.Fatalf("unit overrides are not absent: %+v", u)
	}
	for _, off := range []int{0x20, 0x22, 0x24, 0x26} {
		if binary.LittleEndian.Uint16(rec[off:]) != 0xFFFF {
			t.Errorf("+%#x is not the absent word", off)
		}
	}
	if !bytes.Equal(rec[0x2c:0x40], make([]byte, 0x14)) {
		t.Error("the override bytes are not zero")
	}
	// A sack goes in on the same map, and every edit undoes to the blank bytes.
	if _, err := e.Place(KindLoot, e.GroundSack(3<<8, 3<<8, 1)); err != nil {
		t.Fatal(err)
	}
	for e.Undo() {
	}
	if !bytes.Equal(e.Bytes(), blank) {
		t.Fatal("undoing every edit does not restore the blank map")
	}
}

func TestNewUnitRecordRefusesWhatItCannotBuild(t *testing.T) {
	e, err := NewMap(alm.DefaultNewMap())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.NewUnitRecord(0, 0, 1, 0, 0); err == nil {
		t.Error("owner slot 0 was accepted")
	}
}

// ALM-SAVE-256: the written file binds a unit whose owner matches no player to
// player 1; the session keeps the owner it was given.
func TestSaveBytesWritesTheOriginalsFormAndKeepsTheSession(t *testing.T) {
	e, err := NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	rec := make([]byte, 70)
	binary.LittleEndian.PutUint32(rec[0x14:], 9)
	if _, err := e.PlaceUnit(rec); err != nil {
		t.Fatal(err)
	}
	saved, err := e.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	want, err := alm.SaveForm(e.Bytes())
	if err != nil || !bytes.Equal(saved, want) {
		t.Fatalf("SaveBytes is not the save form of the session (%v)", err)
	}
	m, err := alm.Open(saved)
	if err != nil || len(m.Units) != 1 || m.Units[0].Owner != 1 {
		t.Fatalf("saved units %v (%v), want owner 1", m.Units, err)
	}
	if m, err := e.Map(); err != nil || m.Units[0].Owner != 9 {
		t.Fatalf("the session's owner changed (%v)", err)
	}
}
