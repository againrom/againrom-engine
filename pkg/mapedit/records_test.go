package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// recordAt walks the physical record headers itself and returns the absolute
// offsets of the header and payload of the record with typeId tid,
// independent of the model.
func recordAt(t *testing.T, b []byte, tid uint32) (header, payload int) {
	t.Helper()
	off := 20
	for n := uint32(0); n < binary.LittleEndian.Uint32(b[12:]); n++ {
		size := int(binary.LittleEndian.Uint32(b[off+8:]))
		if binary.LittleEndian.Uint32(b[off+12:]) == tid {
			return off, off + 20
		}
		off += 20 + size
	}
	t.Fatalf("fixture has no type-%d record", tid)
	return 0, 0
}

// metaAt is the absolute offset of a type-0 word.
func metaAt(t *testing.T, b []byte, word int) int {
	_, p := recordAt(t, b, 0)
	return p + word
}

// splice is the independent expectation: b with n bytes at off replaced by in.
func splice(b []byte, off, n int, in []byte) []byte {
	out := append([]byte(nil), b[:off]...)
	out = append(out, in...)
	return append(out, b[off+n:]...)
}

func addWord(b []byte, off int, delta int) {
	binary.LittleEndian.PutUint32(b[off:], uint32(int64(binary.LittleEndian.Uint32(b[off:]))+int64(delta)))
}

// recordsFixture is a 16x16 map with one unit (id 7), a plain structure of
// kind 5 and a kind-0x21 structure with a 2x3 override, and an empty type-8
// record.
func recordsFixture(t *testing.T) []byte {
	t.Helper()
	b := synth.ALM(synth.ALMOptions{Width: 16, Height: 16, Type7Payload: make([]byte, 12),
		Units: []synth.ALMUnit{{X: 2*256 + 128, Y: 3*256 + 128, ClassID: 40, ClassSubID: 2}},
		Objects: []synth.ALMObject{
			{X: 3*256 + 128, Y: 4*256 + 128, Kind: 5, Field0C: 11, Field0E: 2, Field12: 7},
			{X: 8*256 + 64, Y: 9*256 + 32, Kind: 0x21, Field0C: 12, Field0E: 1, Field12: 9, Ext: []byte{2, 0xa1, 0, 0, 3, 0xb2, 0, 0}},
		}})
	_, p := recordAt(t, b, 6)
	binary.LittleEndian.PutUint16(b[p+0x40:], 7)
	return b
}

// undoRedoExact undoes n edits back to start and redoes them to end, byte
// for byte.
func undoRedoExact(t *testing.T, e *mapedit.Editor, n int, start, end []byte) {
	t.Helper()
	for i := 0; i < n; i++ {
		if !e.Undo() {
			t.Fatalf("undo %d refused", i)
		}
	}
	if !bytes.Equal(e.Bytes(), start) || e.CanUndo() {
		t.Fatal("undo did not restore the exact bytes")
	}
	for i := 0; i < n; i++ {
		e.Redo()
	}
	if !bytes.Equal(e.Bytes(), end) {
		t.Fatal("redo did not restore the exact bytes")
	}
}

func TestStructureMoveWritesOnlyItsCoordinateWords(t *testing.T) {
	b := recordsFixture(t)
	e := newEditor(t, b)
	if err := e.Move(mapedit.KindStructure, 1, 11*256+64, 12*256+32); err != nil {
		t.Fatal(err)
	}
	_, p := recordAt(t, b, 4)
	want := append([]byte(nil), b...)
	binary.LittleEndian.PutUint32(want[p+20:], 11*256+64)
	binary.LittleEndian.PutUint32(want[p+24:], 12*256+32)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("the move wrote bytes other than structure 1's X and Y words")
	}
	m, err := alm.Open(e.Bytes())
	if err != nil || m.Objects[1].X>>8 != 11 || m.Objects[1].Y>>8 != 12 || !bytes.Equal(m.Objects[1].Ext, []byte{2, 0xa1, 0, 0, 3, 0xb2, 0, 0}) {
		t.Fatalf("moved structure %+v, %v", m.Objects, err)
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestStructureCloneCopiesEveryByteButTheCell(t *testing.T) {
	b := recordsFixture(t)
	e := newEditor(t, b)
	i, err := e.CloneRecord(mapedit.KindStructure, 1, 13*256+64, 2*256+32)
	if err != nil || i != 2 {
		t.Fatalf("clone index %d, %v", i, err)
	}
	h, p := recordAt(t, b, 4)
	rec := append([]byte(nil), b[p+20:p+48]...)
	binary.LittleEndian.PutUint32(rec[0:], 13*256+64)
	binary.LittleEndian.PutUint32(rec[4:], 2*256+32)
	want := splice(b, p+48, 0, rec)
	addWord(want, h+8, 28)
	addWord(want, metaAt(t, b, 0x20), 1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("the clone is not the source's 28 bytes at the new cell plus the size and count words")
	}
	m, err := alm.Open(e.Bytes())
	if err != nil || len(m.Objects) != 3 || m.Objects[2].Field12 != 9 || m.Objects[2].Field0C != 12 || m.Objects[2].Field0E != 1 {
		t.Fatalf("clone %+v, %v", m.Objects, err)
	}
	undoRedoExact(t, e, 1, b, want)
}

func TestStructureDeleteRemovesOneRecord(t *testing.T) {
	b := recordsFixture(t)
	e := newEditor(t, b)
	if err := e.Delete(mapedit.KindStructure, 0); err != nil {
		t.Fatal(err)
	}
	h, p := recordAt(t, b, 4)
	want := splice(b, p, 20, nil)
	addWord(want, h+8, -20)
	addWord(want, metaAt(t, b, 0x20), -1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("the delete is not the first 20-byte record plus the size and count words")
	}
	m, err := alm.Open(e.Bytes())
	if err != nil || len(m.Objects) != 1 || m.Objects[0].Kind != 0x21 {
		t.Fatalf("after delete %+v, %v", m.Objects, err)
	}
	undoRedoExact(t, e, 1, b, want)
	if err := e.Delete(mapedit.KindStructure, 1); err == nil {
		t.Fatal("a delete past the last structure was accepted")
	}
}

func TestStructureKindAcrossTheExtensionReshapesTheRecord(t *testing.T) {
	b := recordsFixture(t)
	e := newEditor(t, b)
	keys := func(i int) string {
		fs, err := e.Fields(mapedit.KindStructure, i)
		if err != nil {
			t.Fatal(err)
		}
		s := ""
		for _, f := range fs {
			s += f.Key + " "
		}
		return s
	}
	if keys(0) != "kind " || keys(1) != "kind width height " {
		t.Fatalf("structure fields %q / %q", keys(0), keys(1))
	}
	if err := e.SetField(mapedit.KindStructure, 0, "kind", 0x21); err != nil {
		t.Fatal(err)
	}
	h, p := recordAt(t, b, 4)
	want := splice(b, p+20, 0, make([]byte, 8))
	binary.LittleEndian.PutUint32(want[p+8:], 0x21)
	addWord(want, h+8, 8)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("kind 0x21 did not add one zero extension")
	}
	if err := e.SetField(mapedit.KindStructure, 1, "kind", 6); err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(e.Bytes())
	if err != nil || !bytes.Equal(m.Objects[0].Ext, make([]byte, 8)) || m.Objects[1].Ext != nil || m.Objects[1].Kind != 6 {
		t.Fatalf("reshaped %+v, %v", m.Objects, err)
	}
	if err := e.SetField(mapedit.KindStructure, 0, "width", 4); err != nil {
		t.Fatal(err)
	}
	if v, err := e.FieldValue(mapedit.KindStructure, 0, "width"); err != nil || v != 4 {
		t.Fatalf("width %d, %v", v, err)
	}
	end := e.Bytes()
	for _, c := range []struct {
		key string
		v   int64
	}{{"kind", 0}, {"kind", 256}, {"width", 256}, {"field12", 1}} {
		if err := e.SetField(mapedit.KindStructure, 0, c.key, c.v); err == nil {
			t.Fatalf("%s=%d accepted", c.key, c.v)
		}
	}
	if !bytes.Equal(e.Bytes(), end) {
		t.Fatal("a refused structure field wrote")
	}
	undoRedoExact(t, e, 3, b, end)
}

func lootElements(t *testing.T, b []byte) []alm.LootRecord {
	t.Helper()
	m, err := alm.Open(b)
	if err != nil {
		t.Fatal(err)
	}
	l, err := m.Loot()
	if err != nil {
		t.Fatal(err)
	}
	return l.Records
}

func TestGroundSackPlaceFieldsAndDelete(t *testing.T) {
	b := recordsFixture(t)
	e := newEditor(t, b)
	rec := e.GroundSack(5*256+128, 6*256+128, 0x0101)
	if len(rec) != 30 {
		t.Fatalf("a one-item ground record is %d bytes", len(rec))
	}
	if i, err := e.Place(mapedit.KindLoot, rec); err != nil || i != 0 {
		t.Fatalf("place %d, %v", i, err)
	}
	h, p := recordAt(t, b, 8)
	want := splice(b, p, 0, rec)
	addWord(want, h+8, 30)
	addWord(want, metaAt(t, b, 0x2c), 1)
	if !bytes.Equal(e.Bytes(), want) {
		t.Fatal("the placed sack is not its 30 bytes plus the size and count words")
	}
	steps := []struct {
		key   string
		v     int64
		codes []uint16
		gold  uint32
	}{
		{"gold", 250, []uint16{0x0101}, 250},
		{"additem", 0x0202, []uint16{0x0101, 0x0202}, 250},
		{"item1", 0, []uint16{0x0202}, 250},
		{"item1", 0x0303, []uint16{0x0303}, 250},
	}
	for _, s := range steps {
		before := e.Bytes()
		if err := e.SetField(mapedit.KindLoot, 0, s.key, s.v); err != nil {
			t.Fatalf("%s=%d: %v", s.key, s.v, err)
		}
		r := lootElements(t, e.Bytes())
		if len(r) != 1 || !r[0].Ground() || r[0].CellX() != 5 || r[0].CellY() != 6 || r[0].Gold != s.gold || len(r[0].Elements) != len(s.codes) {
			t.Fatalf("%s=%d: %+v", s.key, s.v, r)
		}
		for k, c := range s.codes {
			if r[0].Elements[k].ItemCode() != c {
				t.Fatalf("%s=%d: element %d is %#x", s.key, s.v, k, r[0].Elements[k].ItemCode())
			}
		}
		if s.key == "gold" || s.key == "item1" && s.v != 0 {
			if diff := diffSpan(before, e.Bytes()); diff > 4 {
				t.Fatalf("%s=%d changed %d bytes", s.key, s.v, diff)
			}
		}
	}
	end := e.Bytes()
	for _, c := range []struct {
		key string
		v   int64
	}{{"gold", -1}, {"gold", 1 << 31}, {"additem", 0}, {"item1", 70000}, {"item2", 1}} {
		if err := e.SetField(mapedit.KindLoot, 0, c.key, c.v); err == nil {
			t.Fatalf("%s=%d accepted", c.key, c.v)
		}
	}
	if !bytes.Equal(e.Bytes(), end) {
		t.Fatal("a refused sack field wrote")
	}
	undoRedoExact(t, e, 5, b, end)
	if err := e.Delete(mapedit.KindLoot, 0); err != nil || !bytes.Equal(e.Bytes(), b) {
		t.Fatalf("deleting the only sack did not restore the fixture: %v", err)
	}
}

// diffSpan counts the bytes in which two equal-length buffers differ.
func diffSpan(a, b []byte) int {
	if len(a) != len(b) {
		return 1 << 30
	}
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

func TestGroundSackCloneAndMoveKeepItsItems(t *testing.T) {
	e := newEditor(t, recordsFixture(t))
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(5*256+128, 6*256+128, 0x0101)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.CloneRecord(mapedit.KindLoot, 0, 9*256+128, 9*256+128); err != nil {
		t.Fatal(err)
	}
	if err := e.Move(mapedit.KindLoot, 0, 1*256+128, 2*256+128); err != nil {
		t.Fatal(err)
	}
	r := lootElements(t, e.Bytes())
	if len(r) != 2 || r[0].CellX() != 1 || r[0].CellY() != 2 || r[1].CellX() != 9 || r[1].Elements[0].ItemCode() != 0x0101 {
		t.Fatalf("sacks %+v", r)
	}
}

func TestStockHasNoCellAndKeepsItsItems(t *testing.T) {
	e := newEditor(t, recordsFixture(t))
	stock := e.GroundSack(0, 0, 0x0101)
	binary.LittleEndian.PutUint32(stock[4:], 7) // owner: unit id 7
	if _, err := e.Place(mapedit.KindLoot, stock); err != nil {
		t.Fatal(err)
	}
	b := e.Bytes()
	if _, err := e.CloneRecord(mapedit.KindLoot, 0, 256, 256); err == nil {
		t.Fatal("a stock clone was accepted")
	}
	if err := e.Move(mapedit.KindLoot, 0, 256, 256); err == nil {
		t.Fatal("a stock move was accepted")
	}
	if _, _, err := e.Position(mapedit.KindLoot, 0); err == nil {
		t.Fatal("a stock has a position")
	}
	if !bytes.Equal(e.Bytes(), b) {
		t.Fatal("a refused stock edit wrote")
	}
	fs, _ := e.Fields(mapedit.KindLoot, 0)
	if len(fs) != 2 || fs[0].Key != "item1" || fs[1].Key != "additem" {
		t.Fatalf("stock fields %+v", fs)
	}
	if err := e.SetField(mapedit.KindLoot, 0, "additem", 0x0202); err != nil {
		t.Fatal(err)
	}
	if r := lootElements(t, e.Bytes()); len(r) != 1 || r[0].Owner != 7 || len(r[0].Elements) != 2 {
		t.Fatalf("stock %+v", r)
	}
}

func TestShortLootHeadHasNoGold(t *testing.T) {
	b := recordsFixture(t)
	binary.LittleEndian.PutUint32(b[0x10:], 988)
	e := newEditor(t, b)
	rec := e.GroundSack(256, 256, 0x0101)
	if len(rec) != 26 {
		t.Fatalf("a one-item short-head record is %d bytes", len(rec))
	}
	if _, err := e.Place(mapedit.KindLoot, rec); err != nil {
		t.Fatal(err)
	}
	fs, _ := e.Fields(mapedit.KindLoot, 0)
	if len(fs) != 2 || fs[0].Key != "item1" {
		t.Fatalf("short-head fields %+v", fs)
	}
	if r := lootElements(t, e.Bytes()); len(r) != 1 || r[0].Elements[0].ItemCode() != 0x0101 {
		t.Fatalf("short-head sack %+v", r)
	}
}

func TestALootSectionThatDoesNotWalkRefusesEveryEdit(t *testing.T) {
	b := recordsFixture(t)
	binary.LittleEndian.PutUint32(b[metaAt(t, b, 0x2c):], 1) // one record, empty payload
	e := newEditor(t, b)
	if n := e.Count(mapedit.KindLoot); n != 0 {
		t.Fatalf("count %d", n)
	}
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(256, 256, 0x0101)); err == nil {
		t.Fatal("a place over a section that does not walk was accepted")
	}
	if _, err := e.Place(mapedit.KindStructure, make([]byte, 21)); err == nil {
		t.Fatal("a 21-byte structure was placed")
	}
	if !bytes.Equal(e.Bytes(), b) || e.CanUndo() {
		t.Fatal("a refused edit wrote or recorded")
	}
}

func TestSectionFieldRowsLieInsideTheirRecords(t *testing.T) {
	e := newEditor(t, recordsFixture(t))
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(256, 256, 0x0101)); err != nil {
		t.Fatal(err)
	}
	for _, k := range []mapedit.Kind{mapedit.KindUnit, mapedit.KindStructure, mapedit.KindLoot} {
		for i := 0; i < e.Count(k); i++ {
			rec, _ := e.Record(k, i)
			fs, err := e.Fields(k, i)
			if err != nil || len(fs) == 0 {
				t.Fatalf("kind %d record %d: %v", k, i, err)
			}
			for _, f := range fs {
				end := f.Offset + f.Width
				if f.Element > 0 && f.Offset == len(rec) {
					end = f.Offset // the appending row
				}
				if f.Offset < 8 || end > len(rec) || f.Min > f.Max {
					t.Fatalf("kind %d row %+v outside a %d-byte record", k, f, len(rec))
				}
			}
		}
	}
}
