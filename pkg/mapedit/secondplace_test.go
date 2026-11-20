package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

var secondLE = binary.LittleEndian

func secondWord(v uint32) []byte {
	b := make([]byte, 4)
	secondLE.PutUint32(b, v)
	return b
}

func secondCat(parts ...[]byte) []byte { return bytes.Join(parts, nil) }

// secondMarkerBytes is one type-9 element: +0, +8 and +0xc words, the mode,
// the key, the +0x16 word and no entries.
func secondMarkerBytes(first, cellX, cellY uint32, mode uint16, key uint32) []byte {
	b := make([]byte, alm.SecondMarkerHead)
	secondLE.PutUint32(b[0:], first)
	secondLE.PutUint32(b[4:], cellX)
	secondLE.PutUint32(b[8:], cellY)
	secondLE.PutUint16(b[alm.SecondMarkerMode:], mode)
	secondLE.PutUint32(b[alm.SecondMarkerKey:], key)
	return b
}

func secondSackBytes(owner, x, y uint32, items ...[2]uint32) []byte {
	b := make([]byte, alm.LootHeadWide, alm.LootHeadWide+len(items)*alm.LootElementSize)
	secondLE.PutUint32(b[alm.LootCount:], uint32(len(items)))
	secondLE.PutUint32(b[alm.LootOwner:], owner)
	secondLE.PutUint32(b[alm.LootX:], x)
	secondLE.PutUint32(b[alm.LootY:], y)
	for _, it := range items {
		e := make([]byte, alm.LootElementSize)
		secondLE.PutUint32(e, it[0])
		secondLE.PutUint32(e[alm.LootElementMarker:], it[1])
		b = append(b, e...)
	}
	return b
}

func secondUnitBytes(x, y, owner, sack, id, group uint32) []byte {
	b := make([]byte, alm.SecondUnitSize)
	secondLE.PutUint32(b[0:], x)
	secondLE.PutUint32(b[4:], y)
	secondLE.PutUint16(b[8:], 7)
	secondLE.PutUint32(b[0x14:], 600)
	secondLE.PutUint32(b[alm.SecondUnitOwner:], owner)
	secondLE.PutUint32(b[alm.SecondUnitSack:], sack)
	secondLE.PutUint32(b[alm.SecondUnitTag:], id|(x&0xffff)<<16)
	secondLE.PutUint32(b[alm.SecondUnitGroup:], group)
	return b
}

func secondStructureBytes(x, y, kind uint32, id uint16) []byte {
	b := make([]byte, alm.ObjectSize)
	secondLE.PutUint32(b[0:], x)
	secondLE.PutUint32(b[4:], y)
	secondLE.PutUint32(b[alm.ObjectKind:], kind)
	secondLE.PutUint16(b[alm.ObjectPlacement:], 100)
	secondLE.PutUint16(b[alm.ObjectOwner:], 1)
	secondLE.PutUint16(b[alm.ObjectIdentity:], id)
	return b
}

// secondPlacedMap is a 32x32 map with structure 1 (a shop with a type-11a and
// a type-11c entry), units 2 (with a sack whose item is bound to element 1,
// and a spellbook element) and 3, and a ground sack bound to element 2.
// The elements: 1 the bound item of unit 2, 2 the bound item of the ground
// sack, 3 a mode-4 element of structure 1, 4 a mode-8 element keyed to unit 2.
func secondPlacedMap(omit ...uint32) []byte {
	cell := func(c uint32) uint32 { return c<<8 | 0x80 }
	skip := map[uint32]bool{}
	for _, o := range omit {
		skip[o] = true
	}
	meta := make([]byte, 660)
	secondLE.PutUint32(meta[0:], 32)
	secondLE.PutUint32(meta[4:], 32)
	secondLE.PutUint32(meta[alm.MetaGroupCount:], 2)
	secondLE.PutUint32(meta[alm.MetaObjectCount:], 1)
	secondLE.PutUint32(meta[alm.MetaUnitCount:], 2)
	secondLE.PutUint32(meta[alm.MetaLootCount:], 2)
	secondLE.PutUint32(meta[alm.SecondMeta11Count:], 1)
	secondLE.PutUint32(meta[alm.SecondMeta11Count+8:], 1)
	markers := secondCat(secondWord(4),
		secondMarkerBytes(0, 0, 0, 0, 0),
		secondMarkerBytes(1, 0, 0, 0, 0),
		secondMarkerBytes(3, 12, 12, 4, 1),
		secondMarkerBytes(0, 0, 0, 8, 2))
	payloads := map[uint32][]byte{
		0: meta,
		1: make([]byte, 2*32*32),
		2: make([]byte, 32*32),
		3: make([]byte, 32*32),
		5: make([]byte, 2*alm.GroupSize),
		11: secondCat(
			secondWord(1), secondWord(2), secondWord(3), // 11a
			secondWord(1), secondWord(1), secondWord(9)), // 11c
		4: secondStructureBytes(cell(12), cell(12), 0x22, 1),
		9: markers,
		8: secondCat(
			secondSackBytes(2, cell(5), cell(5), [2]uint32{300, 1}, [2]uint32{301, 0}),
			secondSackBytes(0, cell(20), cell(20), [2]uint32{302, 2})),
		6: secondCat(
			secondUnitBytes(cell(5), cell(5), 1, 1, 2, 5),
			secondUnitBytes(cell(8), cell(9), 2, 0, 3, 6)),
		7: make([]byte, 12),
	}
	order := []uint32{0, 1, 2, 3, 5, 11, 4, 9, 8, 6, 7}
	var recs [][]byte
	for _, tid := range order {
		if skip[tid] {
			continue
		}
		h := make([]byte, 20)
		secondLE.PutUint32(h[0:], 7)
		secondLE.PutUint32(h[4:], 20)
		secondLE.PutUint32(h[8:], uint32(len(payloads[tid])))
		if tid == 0 {
			secondLE.PutUint32(h[8:], 644)
		}
		secondLE.PutUint32(h[12:], tid)
		secondLE.PutUint32(h[16:], 0x1234)
		recs = append(recs, h, payloads[tid])
	}
	file := make([]byte, 20)
	copy(file, "M7R\x00")
	secondLE.PutUint32(file[4:], 20)
	secondLE.PutUint32(file[8:], 4*32*32+72)
	secondLE.PutUint32(file[12:], uint32(len(recs)/2))
	secondLE.PutUint32(file[16:], 1300)
	return secondCat(append([][]byte{file}, recs...)...)
}

func secondOpen(t *testing.T, raw []byte) (*mapedit.Editor, *alm.Map) {
	t.Helper()
	e, err := mapedit.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	m, err := e.Map()
	if err != nil {
		t.Fatal(err)
	}
	return e, m
}

func secondReload(t *testing.T, e *mapedit.Editor) *alm.Map {
	t.Helper()
	out, err := e.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.OpenROM2(out)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	return m
}

func secondCounters(t *testing.T, m *alm.Map) {
	t.Helper()
	if int(m.Meta.Count4) != len(m.Objects) || int(m.Meta.Count6) != len(m.Units) {
		t.Fatalf("type-0 counters %d, %d for %d structures, %d units", m.Meta.Count4, m.Meta.Count6, len(m.Objects), len(m.Units))
	}
	loot, err := m.Loot()
	if err != nil || int(m.Meta.Word2C) != len(loot.Records) {
		t.Fatalf("type-0 sack counter %d for %d sacks: %v", m.Meta.Word2C, len(loot.Records), err)
	}
}

func TestSecondPlacedMapSavesUnchangedAndWalks(t *testing.T) {
	raw := secondPlacedMap()
	e, m := secondOpen(t, raw)
	secondCounters(t, m)
	if out, err := e.SaveBytes(); err != nil || !bytes.Equal(out, raw) {
		t.Fatalf("unchanged save differs: %v", err)
	}
	if e.Count(mapedit.KindUnit) != 2 || e.Count(mapedit.KindStructure) != 1 || e.Count(mapedit.KindLoot) != 2 {
		t.Fatal("sections do not walk")
	}
}

func TestSecondGroundSackIsPlacedMovedAndDeleted(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)
	x, y := uint32(3<<8|0x80), uint32(4<<8|0x80)
	i, err := e.Place(mapedit.KindLoot, e.GroundSack(x, y, 258))
	if err != nil || i != 2 {
		t.Fatalf("place: %d %v", i, err)
	}
	m := secondReload(t, e)
	secondCounters(t, m)
	loot, _ := m.Loot()
	r := loot.Records[2]
	if !r.Ground() || r.X != x || r.Y != y || len(r.Elements) != 1 || r.Elements[0].ItemCode() != 258 || r.Elements[0].TileMarkerIndex != 0 {
		t.Fatalf("placed sack: %+v", r)
	}
	if err := e.Move(mapedit.KindLoot, 2, 9<<8|0x80, 9<<8|0x80); err != nil {
		t.Fatal(err)
	}
	loot, _ = secondReload(t, e).Loot()
	if loot.Records[2].X != 9<<8|0x80 {
		t.Fatal("sack did not move")
	}
	if err := e.Delete(mapedit.KindLoot, 2); err != nil {
		t.Fatal(err)
	}
	if out, err := e.SaveBytes(); err != nil || !bytes.Equal(out, raw) {
		t.Fatalf("place and delete of a sack leave other bytes: %v", err)
	}
	if err := e.Delete(mapedit.KindLoot, 0); err == nil {
		t.Fatal("a unit's own sack was deleted by itself")
	}
}

func TestSecondUnitCopyTakesItsOwnIdSackAndBoundElement(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)
	x, y := uint32(15<<8|0x80), uint32(16<<8|0x80)
	i, err := e.CloneRecord(mapedit.KindUnit, 0, x, y)
	if err != nil || i != 2 {
		t.Fatalf("clone: %d %v", i, err)
	}
	if err := e.SetField(mapedit.KindUnit, i, "owner", 2); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	secondCounters(t, m)
	u := m.Units[2]
	if u.UnitID != 4 || u.X != x || u.Y != y || u.Owner != 2 {
		t.Fatalf("new unit: %+v", u)
	}
	rec, _ := e.Record(mapedit.KindUnit, 2)
	if got := secondLE.Uint32(rec[alm.SecondUnitTag:]); got != 4|(x&0xffff)<<16 {
		t.Fatalf("0x28 = %#x", got)
	}
	link := secondLE.Uint32(rec[alm.SecondUnitSack:])
	loot, _ := m.Loot()
	if link != 3 || len(loot.Records) != 3 {
		t.Fatalf("sack link %d with %d sacks", link, len(loot.Records))
	}
	s := loot.Records[2]
	if s.Owner != 4 || s.X != x || s.Y != y || len(s.Elements) != 2 || s.Elements[0].TileMarkerIndex != 5 || s.Elements[1].TileMarkerIndex != 0 {
		t.Fatalf("copied sack: %+v", s)
	}
	if e.NextID() != 5 {
		t.Fatalf("next id %d", e.NextID())
	}
	// The source unit and its sack keep their bytes.
	if got, _ := e.Record(mapedit.KindUnit, 0); !bytes.Equal(got, secondUnitBytes(5<<8|0x80, 5<<8|0x80, 1, 1, 2, 5)) {
		t.Fatal("the source unit changed")
	}
	if !e.CanUndo() {
		t.Fatal("no history")
	}
	e.Undo()
	e.Undo()
	if out, _ := e.SaveBytes(); !bytes.Equal(out, raw) {
		t.Fatal("undo did not restore the bytes")
	}
}

func TestSecondStructureCopyTakesTheLowestFreeId(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)
	if _, err := e.CloneRecord(mapedit.KindUnit, 1, 20<<8|0x80, 20<<8|0x80); err != nil {
		t.Fatal(err)
	}
	if _, err := e.CloneRecord(mapedit.KindStructure, 0, 25<<8|0x80, 25<<8|0x80); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	secondCounters(t, m)
	if m.Units[2].UnitID != 4 || m.Objects[1].Field12 != 5 {
		t.Fatalf("ids: unit %d structure %d", m.Units[2].UnitID, m.Objects[1].Field12)
	}
	// No type-11 entry is written for the copy.
	if m.Meta.Count11 != [3]uint32{1, 0, 1} {
		t.Fatalf("type-11 counts %v", m.Meta.Count11)
	}
}

func TestSecondUnitMoveCarriesItsTagAndSack(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	nx, ny := uint32(0x1234<<8|0x80), uint32(7<<8|0x80)
	if err := e.Move(mapedit.KindUnit, 0, nx, ny); err != nil {
		t.Fatal(err)
	}
	rec, _ := e.Record(mapedit.KindUnit, 0)
	if secondLE.Uint32(rec[alm.SecondUnitTag:]) != 2|(nx&0xffff)<<16 {
		t.Fatal("0x28 high word did not follow x")
	}
	sack, _ := e.Record(mapedit.KindLoot, 0)
	if secondLE.Uint32(sack[alm.LootX:]) != nx || secondLE.Uint32(sack[alm.LootY:]) != ny {
		t.Fatal("sack did not follow its unit")
	}
	if other, _ := e.Record(mapedit.KindLoot, 1); !bytes.Equal(other, secondSackBytes(0, 20<<8|0x80, 20<<8|0x80, [2]uint32{302, 2})) {
		t.Fatal("another sack changed")
	}
}

func TestSecondUnitDeleteTakesSackBoundAndBookElements(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	if err := e.Delete(mapedit.KindUnit, 0); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	secondCounters(t, m)
	if len(m.Units) != 1 || m.Units[0].UnitID != 3 {
		t.Fatalf("units %+v", m.Units)
	}
	loot, _ := m.Loot()
	if len(loot.Records) != 1 || !loot.Records[0].Ground() {
		t.Fatalf("sacks %+v", loot.Records)
	}
	// Elements 1 (bound) and 4 (spellbook) are gone; the ground sack's item
	// that bound element 2 now binds position 1.
	if loot.Records[0].Elements[0].TileMarkerIndex != 1 {
		t.Fatalf("binding %d", loot.Records[0].Elements[0].TileMarkerIndex)
	}
	_, ms := secondMarkerModes(t, e)
	if len(ms) != 2 || ms[0] != 0 || ms[1] != 4 {
		t.Fatalf("elements left: %v", ms)
	}
	// The next id is not lowered by a delete; the freed id is reused.
	if e.NextID() != 4 {
		t.Fatalf("next id %d", e.NextID())
	}
	e.Undo()
	if e.NextID() != 4 {
		t.Fatal("undo lowered the next id")
	}
}

// secondMarkerModes reads the type-9 modes through the document.
func secondMarkerModes(t *testing.T, e *mapedit.Editor) (int, []uint16) {
	t.Helper()
	doc, err := e.Document()
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range doc.RecordTypeIDs() {
		if id != alm.TypeTileMarkers {
			continue
		}
		p := doc.RecordPayload(i)
		n := int(secondLE.Uint32(p))
		off, modes := 4, []uint16(nil)
		for k := 0; k < n; k++ {
			modes = append(modes, secondLE.Uint16(p[off+alm.SecondMarkerMode:]))
			off += alm.SecondMarkerHead + alm.SecondMarkerEntry*int(secondLE.Uint32(p[off+alm.SecondMarkerCount:]))
		}
		if off != len(p) {
			t.Fatal("type-9 walk")
		}
		return n, modes
	}
	t.Fatal("no type-9 record")
	return 0, nil
}

func TestSecondStructureDeleteTakesItsElementsAndListEntries(t *testing.T) {
	e, _ := secondOpen(t, secondPlacedMap())
	if err := e.Delete(mapedit.KindStructure, 0); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	secondCounters(t, m)
	if len(m.Objects) != 0 || m.Meta.Count11 != [3]uint32{0, 0, 0} || len(m.Extension[1]) != 0 {
		t.Fatalf("structure %d, lists %v/%d bytes", len(m.Objects), m.Meta.Count11, len(m.Extension[1]))
	}
	n, modes := secondMarkerModes(t, e)
	if n != 3 || modes[0] != 0 || modes[1] != 0 || modes[2] != 8 {
		t.Fatalf("elements left: %v", modes)
	}
	// The unit ids and the sacks stay.
	if len(m.Units) != 2 || m.Meta.Word2C != 2 {
		t.Fatal("units or sacks changed")
	}
}

func TestSecondPlacementCreatesAbsentRecordsInTheEditorsOrder(t *testing.T) {
	raw := secondPlacedMap(4, 6, 8)
	e, m := secondOpen(t, raw)
	if len(m.Units) != 0 || len(m.Objects) != 0 {
		t.Fatal("fixture holds placed elements")
	}
	if _, err := e.Place(mapedit.KindLoot, e.GroundSack(5<<8|0x80, 5<<8|0x80, 258)); err != nil {
		t.Fatal(err)
	}
	doc, err := e.Document()
	if err != nil {
		t.Fatal(err)
	}
	var got []uint32
	for _, id := range doc.RecordTypeIDs() {
		got = append(got, id)
	}
	want := []uint32{0, 1, 2, 3, 5, 11, 9, 8, 7}
	if len(got) != len(want) {
		t.Fatalf("records %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("records %v, want %v", got, want)
		}
	}
	out, _ := e.SaveBytes()
	if _, err := alm.OpenROM2(out); err != nil {
		t.Fatal(err)
	}
	// The new record carries the map's tag and its word at +0x10.
	hdr := alm.FileHeaderSize
	for i := 0; i < 7; i++ {
		hdr += alm.RecordHeaderSize + len(doc.RecordPayload(i))
	}
	if secondLE.Uint32(out[hdr+alm.RecordWord10:]) != 0x1234 || secondLE.Uint32(out[hdr:]) != 7 || secondLE.Uint32(out[hdr+alm.RecordTypeID:]) != 8 {
		t.Fatal("new record header")
	}
}
