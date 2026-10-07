package sav

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestBuildings1114AliasAndLateScalarStores(t *testing.T) {
	f, offsets := buildingDocumentFixture()
	// Replace the second full object by a plain MFC reference to the first.
	// Header count stays three; the unique roster has two objects.
	start, end := offsets[1]-2, offsets[1]+77
	b := append([]byte(nil), f.Body[:start]...)
	b = binary.LittleEndian.AppendUint16(b, 2)
	b = append(b, f.Body[end:]...)
	if len(b)&1 != 0 {
		b = append(b, 0)
	}
	for i := 0; i < 22; i++ {
		b[offsets[0]+37+i] = byte(80 + i)
	}
	rows, _, err := (&File{Body: b}).Buildings()
	if err != nil || len(rows) != 2 {
		t.Fatalf("alias: %+v %v", rows, err)
	}
	if rows[0].Base52[13] != 93 || rows[0].Base52[14] != 3 || rows[0].Base52[15] != 2 ||
		binary.LittleEndian.Uint32(rows[0].Base52[18:]) != 5 {
		t.Fatal("late scalar stores did not win")
	}
	if rows[0].ArchiveIndex != 2 || rows[1].ArchiveIndex != 3 {
		t.Fatalf("archive identity: %+v", rows)
	}
}

func TestBuildings1114SubclassPayloadsRemainDetached(t *testing.T) {
	f, _ := groundFixture(true, 0)
	rows, _, err := f.Buildings()
	if err != nil || len(rows) != 3 {
		t.Fatal(err)
	}
	if len(rows[0].OutpostRecords) != 2 {
		t.Fatalf("Outpost count %d", len(rows[0].OutpostRecords))
	}
	rows[0].OutpostRecords[1][7] = 90
	again, _, err := f.Buildings()
	if err != nil || again[0].OutpostRecords[1][7] != 0 {
		t.Fatalf("aliased array %v", err)
	}
}

func TestStructureCells1114LiteralOrderAndZeroClear(t *testing.T) {
	f, _ := buildingDocumentFixture()
	doc, _, err := f.exactDocument()
	if err != nil {
		t.Fatal(err)
	}
	off := doc.world.CellRecOff
	b := append([]byte(nil), f.Body[:off]...)
	b = binary.LittleEndian.AppendUint16(b, 3)
	for i, key := range []uint16{0x0c0d, 0xffff, 0x0c0d} {
		b = binary.LittleEndian.AppendUint16(b, key)
		cell := [52]byte{byte(6 + i), byte(1 + i)}
		if i != 2 {
			binary.LittleEndian.PutUint32(cell[12:], 0xa000005a)
		}
		b = append(b, cell[:]...)
	}
	b = append(b, f.Body[off+2:]...)
	got, present, err := (&File{Body: b}).StructureCells()
	want := []StructureCell{{0x0c0d, 6, 1, 0xa000005a}, {0xffff, 7, 2, 0xa000005a}, {0x0c0d, 8, 3, 0}}
	if err != nil || !present || !reflect.DeepEqual(got, want) {
		t.Fatalf("cells %v/%t %v", got, present, err)
	}
	got[0].BuildingKey = 0
	again, _, _ := (&File{Body: b}).StructureCells()
	if again[0] != want[0] {
		t.Fatal("projection aliases caller")
	}
}
