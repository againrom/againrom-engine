package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/mapedit"
)

func TestUnitCloneCopiesItsOwnedStockAsOneIndependentEdit(t *testing.T) {
	e := freshRecords(t)
	u, _ := e.NewUnitRecord(10*256+128, 11*256+128, 40, 2, 2)
	_, _ = e.PlaceUnit(u)
	_, _ = e.AddUnitStock(0)
	_ = e.SetField(mapedit.KindLoot, 0, "additem", 0x0101)
	_, _ = e.AddItemRecipe(0, 1)
	_ = e.SetField(mapedit.KindEnchantment, 0, "addeffect", 12)
	_ = e.SetField(mapedit.KindEnchantment, 0, "low1", 5)
	before := e.Bytes()
	i, err := e.CloneUnit(0, 12*256+128, 13*256+128)
	if err != nil || i != 1 {
		t.Fatal(i, err)
	}
	after := e.Bytes()
	m, _ := e.Map()
	l, _ := m.Loot()
	source, _ := e.UnitRecord(0)
	clone, _ := e.UnitRecord(1)
	if len(l.Records) != 2 || m.Units[0].UnitID != 1 || m.Units[1].UnitID != 2 ||
		l.Records[0].Owner != 1 || l.Records[1].Owner != 2 ||
		binary.LittleEndian.Uint32(source[0x18:]) != 1 || binary.LittleEndian.Uint32(clone[0x18:]) != 2 ||
		l.Records[0].Elements[0].TileMarkerIndex != 1 || l.Records[1].Elements[0].TileMarkerIndex != 2 || len(m.Enchantments) != 2 {
		t.Fatal("the clone did not own a separate stock", m.Units, l)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) || !e.Redo() || !bytes.Equal(after, e.Bytes()) {
		t.Fatal("unit and stock clone are not one byte-exact history entry")
	}
	if err := e.SetField(mapedit.KindLoot, 1, "item1", 0x0202); err != nil {
		t.Fatal(err)
	}
	m, _ = e.Map()
	l, _ = m.Loot()
	if l.Records[0].Elements[0].ItemCode() != 0x0101 || l.Records[1].Elements[0].ItemCode() != 0x0202 {
		t.Fatal("editing cloned stock changed the source", l)
	}
	if err := e.SetField(mapedit.KindEnchantment, 1, "low1", 9); err != nil {
		t.Fatal(err)
	}
	m, _ = e.Map()
	if m.Enchantments[0].Elements[0].Low != 5 || m.Enchantments[1].Elements[0].Low != 9 {
		t.Fatal("editing the cloned item recipe changed the source")
	}
}

func TestUnitCloneDoesNotInheritForeignOrInvalidStockLinks(t *testing.T) {
	for _, link := range []uint32{1, 99, 0xffffffff} {
		e := freshRecords(t)
		u, _ := e.NewUnitRecord(10*256+128, 11*256+128, 40, 2, 2)
		binary.LittleEndian.PutUint32(u[0x18:], link)
		_, _ = e.PlaceUnit(u)
		_, _ = e.Place(mapedit.KindLoot, e.GroundSack(128, 128, 0x0101))
		if _, err := e.CloneUnit(0, 12*256+128, 13*256+128); err != nil {
			t.Fatal(err)
		}
		clone, _ := e.UnitRecord(1)
		source, _ := e.UnitRecord(0)
		if binary.LittleEndian.Uint32(clone[0x18:]) != 0 || binary.LittleEndian.Uint32(source[0x18:]) != link || e.Count(mapedit.KindLoot) != 1 {
			t.Fatal("foreign or invalid stock link survived clone", link)
		}
	}
}

func TestUnitStockCloneRefusesBrokenRecipesWithoutMutation(t *testing.T) {
	e := freshRecords(t)
	u, _ := e.NewUnitRecord(10*256+128, 11*256+128, 40, 2, 2)
	_, _ = e.PlaceUnit(u)
	_, _ = e.AddUnitStock(0)
	_ = e.SetField(mapedit.KindLoot, 0, "additem", 0x0101)
	_ = e.SetField(mapedit.KindLoot, 0, "recipe1", 99)
	before := e.Bytes()
	e = newEditor(t, before)
	if _, err := e.CloneUnit(0, 12*256+128, 13*256+128); err == nil || !bytes.Equal(before, e.Bytes()) || e.CanUndo() {
		t.Fatal("a broken recipe clone wrote bytes or history")
	}
}
