package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/mapedit"
)

func TestDuplicateLootElementPreservesNativeWordsAndUndo(t *testing.T) {
	e := freshRecords(t)
	record := e.GroundSack(0, 0, 0x0101)
	binary.LittleEndian.PutUint32(record[20:], 0xaabb0101)
	binary.LittleEndian.PutUint16(record[24:], 0x1234)
	index, err := e.Place(mapedit.KindLoot, record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddItemRecipe(index, 1); err != nil {
		t.Fatal(err)
	}
	before := e.Bytes()
	if err := e.SetField(mapedit.KindLoot, index, "duplicate1", 1); err != nil {
		t.Fatal(err)
	}
	m, _ := e.Map()
	loot, _ := m.Loot()
	a, b := loot.Records[index].Elements[0], loot.Records[index].Elements[1]
	if a != b || b.Code != 0xaabb0101 || b.Field04 != 0x1234 || b.TileMarkerIndex != 1 {
		t.Fatal("duplicate changed native words", a, b)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("duplicate Undo was not exact")
	}
}
