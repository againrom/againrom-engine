package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/mapedit"
)

func castRecord(kind uint16) []byte {
	b := make([]byte, 38)
	binary.LittleEndian.PutUint32(b, 0xaabbccdd)
	binary.LittleEndian.PutUint32(b[4:], 12)
	binary.LittleEndian.PutUint32(b[8:], 13)
	binary.LittleEndian.PutUint16(b[12:], kind)
	binary.LittleEndian.PutUint32(b[14:], 0x11223344)
	binary.LittleEndian.PutUint32(b[18:], 7|50<<16)
	binary.LittleEndian.PutUint32(b[22:], 2)
	for i, v := range []uint16{4, 5, 0xdead, 6, 7, 0xbeef} {
		binary.LittleEndian.PutUint16(b[26+2*i:], v)
	}
	return b
}

func TestCellSpellFieldsPreserveOpaqueWords(t *testing.T) {
	e := freshRecords(t)
	for _, kind := range []uint16{0, 1, 2, 3, 4, 8, 12} {
		rec := castRecord(kind)
		i, err := e.Place(mapedit.KindEnchantment, rec)
		if err != nil {
			t.Fatal(err)
		}
		before := e.Bytes()
		fields, err := e.Fields(mapedit.KindEnchantment, i)
		if err != nil {
			t.Fatal(err)
		}
		if kind == 12 {
			if len(fields) != 0 || e.Delete(mapedit.KindEnchantment, i) == nil || e.SetField(mapedit.KindEnchantment, i, "spell", 8) == nil || !bytes.Equal(before, e.Bytes()) {
				t.Fatal("unknown kind was writable")
			}
			continue
		}
		if err := e.SetFields(mapedit.KindEnchantment, i, mapedit.FieldSet{Key: "spell", Value: 13}, mapedit.FieldSet{Key: "power", Value: 100}); err != nil {
			t.Fatal(err)
		}
		want := bytes.Clone(rec)
		binary.LittleEndian.PutUint32(want[18:], 13|100<<16)
		got, _ := e.Record(mapedit.KindEnchantment, i)
		if !bytes.Equal(got, want) {
			t.Fatalf("kind %d: got %x want %x", kind, got, want)
		}
		for _, v := range []int64{-1, 101, 4294967396} {
			if e.SetField(mapedit.KindEnchantment, i, "power", v) == nil {
				t.Fatal("power accepted", v)
			}
		}
		if !e.Undo() || !bytes.Equal(before, e.Bytes()) {
			t.Fatal("field undo")
		}
	}
}

func TestCellSpellTemplateAndDeleteRenumberLinks(t *testing.T) {
	e := freshRecords(t)
	trap := castRecord(3)
	if _, err := e.Place(mapedit.KindEnchantment, trap); err != nil {
		t.Fatal(err)
	}
	sack, err := e.Place(mapedit.KindLoot, e.GroundSack(10*256, 11*256, 257))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddItemRecipe(sack, 1); err != nil {
		t.Fatal(err)
	}
	before := e.Bytes()
	i, err := e.AddSpellFrom(0, 20, 21, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := bytes.Clone(trap)
	binary.LittleEndian.PutUint32(want[4:], 20)
	binary.LittleEndian.PutUint32(want[8:], 21)
	binary.LittleEndian.PutUint32(want[14:], 0)
	got, _ := e.Record(mapedit.KindEnchantment, i)
	if !bytes.Equal(got, want) {
		t.Fatalf("template %x want %x", got, want)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("template undo")
	}
	if err := e.Delete(mapedit.KindEnchantment, 0); err != nil {
		t.Fatal(err)
	}
	m, _ := e.Map()
	l, _ := m.Loot()
	if l.Records[0].Elements[0].TileMarkerIndex != 1 {
		t.Fatal("later recipe link was not renumbered", l)
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("delete undo")
	}
}

func TestCellSpellDeleteUsesLastLoadedBackReference(t *testing.T) {
	e := freshRecords(t)
	if _, err := e.Place(mapedit.KindEnchantment, castRecord(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Place(mapedit.KindEnchantment, make([]byte, 26)); err != nil {
		t.Fatal(err)
	}
	for _, link := range []uint32{2, 2, 1, 99} {
		rec := e.GroundSack(10*256, 11*256, 257)
		binary.LittleEndian.PutUint32(rec[26:], link)
		if _, err := e.Place(mapedit.KindLoot, rec); err != nil {
			t.Fatal(err)
		}
	}
	before := e.Bytes()
	if err := e.Delete(mapedit.KindEnchantment, 0); err != nil {
		t.Fatal(err)
	}
	for i, want := range []uint32{2, 1, 1, 99} {
		rec, err := e.Record(mapedit.KindLoot, i)
		if err != nil || binary.LittleEndian.Uint32(rec[26:]) != want {
			t.Fatal("back-reference", i, rec, want, err)
		}
	}
	if !e.Undo() || !bytes.Equal(before, e.Bytes()) {
		t.Fatal("back-reference undo")
	}
}
