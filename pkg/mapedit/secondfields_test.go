package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

func fieldKeys(fs []mapedit.Field) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Key
	}
	return out
}

func secondRecordOffset(t *testing.T, raw []byte, tid uint32) int {
	t.Helper()
	doc, err := alm.OpenDocument(raw, alm.SecondFormat)
	if err != nil {
		t.Fatal(err)
	}
	off := alm.FileHeaderSize
	for i, id := range doc.RecordTypeIDs() {
		size := len(doc.RecordPayload(i))
		if id == tid {
			return off + alm.RecordHeaderSize
		}
		off += alm.RecordHeaderSize + size
	}
	t.Fatalf("no type-%d record", tid)
	return -1
}

func assertOnlyByteRangeChanged(t *testing.T, what string, before, after []byte, lo, hi int) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("%s: length %d -> %d", what, len(before), len(after))
	}
	changed := false
	for i := range before {
		if before[i] != after[i] {
			if i < lo || i >= hi {
				t.Fatalf("%s: byte %#x changed outside [%#x, %#x)", what, i, lo, hi)
			}
			changed = true
		}
	}
	if !changed {
		t.Fatalf("%s: no byte changed in [%#x, %#x)", what, lo, hi)
	}
}

func TestSecondUnitFieldsEditSingleFieldSaveReloadAndUndo(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)

	fs, err := e.Fields(mapedit.KindUnit, 0)
	if err != nil {
		t.Fatal(err)
	}
	wantKeys := []string{"owner", "definition", "hp", "hpmax", "flags", "group", "field08", "field0a", "field10", "field20"}
	if got := fieldKeys(fs); !slices.Equal(got, wantKeys) {
		t.Fatalf("unit fields = %v, want %v", got, wantKeys)
	}

	unitBase := secondRecordOffset(t, raw, alm.TypeUnits)
	cases := []struct {
		key   string
		value int64
		off   int
		width int
	}{
		{"owner", 2, alm.SecondUnitOwner, 4},
		{"definition", 601, alm.SecondUnitDef, 4},
		{"hp", 120, alm.SecondUnitHP, 2},
		{"hpmax", 240, alm.SecondUnitHPMax, 2},
		{"flags", 0x10, alm.SecondUnitFlags, 4},
		{"group", 19, alm.SecondUnitGroup, 4},
		{"field08", 12, alm.SecondUnitClass, 2},
		{"field0a", 3, alm.SecondUnitField0A, 2},
		{"field10", 8, alm.SecondUnitFlags10, 4},
		{"field20", 10, alm.SecondUnitField20, 4},
	}
	for _, tc := range cases {
		if err := e.SetField(mapedit.KindUnit, 0, tc.key, tc.value); err != nil {
			t.Fatalf("SetField(%s=%d): %v", tc.key, tc.value, err)
		}
		saved, err := e.SaveBytes()
		if err != nil {
			t.Fatalf("%s SaveBytes: %v", tc.key, err)
		}
		assertOnlyByteRangeChanged(t, tc.key, raw, saved, unitBase+tc.off, unitBase+tc.off+tc.width)
		if _, err := alm.OpenROM2(saved); err != nil {
			t.Fatalf("%s OpenROM2: %v", tc.key, err)
		}
		reloaded, err := mapedit.New(saved)
		if err != nil {
			t.Fatalf("%s mapedit.New: %v", tc.key, err)
		}
		if got, err := reloaded.FieldValue(mapedit.KindUnit, 0, tc.key); err != nil || got != tc.value {
			t.Fatalf("%s reloaded = %d (%v), want %d", tc.key, got, err, tc.value)
		}
		if !e.Undo() {
			t.Fatalf("%s Undo returned false", tc.key)
		}
		if got, _ := e.SaveBytes(); !bytes.Equal(got, raw) {
			t.Fatalf("%s Undo did not restore original bytes", tc.key)
		}
	}

	// The first game's unit field table is unchanged.
	first := mapedit.UnitFields()
	if len(first) == 0 || first[0].Key != "owner" || first[1].Key != "class" || first[4].Key != "hp" {
		t.Fatalf("first-game unit fields changed: %v", fieldKeys(first))
	}
}

func TestSecondStructureFieldsEditSingleFieldReshapeSaveReloadAndUndo(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)

	// Append a 28-byte kind 0x1000021 structure with extension dwords (4, 5).
	bridge := make([]byte, alm.ObjectSize+alm.ObjectExtSize)
	secondLE.PutUint32(bridge[alm.ObjectX:], 16<<8|0x80)
	secondLE.PutUint32(bridge[alm.ObjectY:], 16<<8|0x80)
	secondLE.PutUint32(bridge[alm.ObjectKind:], alm.SecondObjectExtBit|alm.ObjectExtKind)
	secondLE.PutUint16(bridge[alm.ObjectPlacement:], 100)
	secondLE.PutUint16(bridge[alm.ObjectOwner:], 1)
	secondLE.PutUint16(bridge[alm.ObjectIdentity:], 10)
	secondLE.PutUint32(bridge[alm.ObjectWidth:], 4)
	secondLE.PutUint32(bridge[alm.ObjectHeight:], 5)
	if _, err := e.Place(mapedit.KindStructure, bridge); err != nil {
		t.Fatal(err)
	}
	base, err := e.SaveBytes()
	if err != nil {
		t.Fatal(err)
	}
	e, _ = secondOpen(t, base)

	plainFields, err := e.Fields(mapedit.KindStructure, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldKeys(plainFields); !slices.Equal(got, []string{"kind", "owner", "field0c"}) {
		t.Fatalf("20-byte structure fields = %v", got)
	}
	extFields, err := e.Fields(mapedit.KindStructure, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldKeys(extFields); !slices.Equal(got, []string{"kind", "owner", "field0c", "field14", "field18"}) {
		t.Fatalf("28-byte structure fields = %v", got)
	}

	structBase := secondRecordOffset(t, base, alm.TypeObjects)
	fixedCases := []struct {
		rec   int
		key   string
		value int64
		lo    int
		hi    int
	}{
		{0, "owner", 2, structBase + alm.ObjectOwner, structBase + alm.ObjectOwner + 2},
		{0, "field0c", 50, structBase + alm.ObjectPlacement, structBase + alm.ObjectPlacement + 2},
		{0, "kind", 5, structBase + alm.ObjectKind, structBase + alm.ObjectKind + 4},
		{1, "field14", 7, structBase + alm.ObjectSize + alm.ObjectWidth, structBase + alm.ObjectSize + alm.ObjectWidth + 4},
		{1, "field18", 9, structBase + alm.ObjectSize + alm.ObjectHeight, structBase + alm.ObjectSize + alm.ObjectHeight + 4},
	}
	for _, tc := range fixedCases {
		if err := e.SetField(mapedit.KindStructure, tc.rec, tc.key, tc.value); err != nil {
			t.Fatalf("SetField(%d, %s=%d): %v", tc.rec, tc.key, tc.value, err)
		}
		saved, err := e.SaveBytes()
		if err != nil {
			t.Fatal(err)
		}
		assertOnlyByteRangeChanged(t, tc.key, base, saved, tc.lo, tc.hi)
		if _, err := alm.OpenROM2(saved); err != nil {
			t.Fatalf("%s reload: %v", tc.key, err)
		}
		reloaded, err := mapedit.New(saved)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := reloaded.FieldValue(mapedit.KindStructure, tc.rec, tc.key); err != nil || got != tc.value {
			t.Fatalf("%s reloaded = %d (%v), want %d", tc.key, got, err, tc.value)
		}
		if !e.Undo() {
			t.Fatalf("%s Undo returned false", tc.key)
		}
		if got, _ := e.SaveBytes(); !bytes.Equal(got, base) {
			t.Fatalf("%s Undo did not restore base", tc.key)
		}
	}

	// Changing kind across 0x21 / 0x1000021 reshapes the record cleanly.
	if err := e.SetField(mapedit.KindStructure, 0, "kind", alm.ObjectExtKind); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	if m.Objects[0].Kind != alm.ObjectExtKind || len(m.Objects[0].Ext) != alm.ObjectExtSize {
		t.Fatalf("grown structure: %+v", m.Objects[0])
	}
	if err := e.SetField(mapedit.KindStructure, 1, "kind", 5); err != nil {
		t.Fatal(err)
	}
	m = secondReload(t, e)
	if m.Objects[1].Kind != 5 || len(m.Objects[1].Ext) != 0 {
		t.Fatalf("shrunk structure: %+v", m.Objects[1])
	}
	e.Undo()
	e.Undo()
	if got, _ := e.SaveBytes(); !bytes.Equal(got, base) {
		t.Fatal("Undo of reshape edits did not restore base")
	}
}

func TestSecondSackFieldsEditItemsSaveReloadAndUndo(t *testing.T) {
	raw := secondPlacedMap()
	e, _ := secondOpen(t, raw)

	fs, err := e.Fields(mapedit.KindLoot, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldKeys(fs); !slices.Equal(got, []string{"field10", "item1", "field04_1", "additem"}) {
		t.Fatalf("sack fields = %v", got)
	}

	// Ground sack 1 follows unit sack 0 (20 + 2*10 = 40 bytes).
	sack1Base := secondRecordOffset(t, raw, alm.TypeLoot) + alm.LootHeadWide + 2*alm.LootElementSize
	fixedCases := []struct {
		key   string
		value int64
		off   int
		width int
	}{
		{"field10", 500, alm.LootGold, 4},
		{"item1", 0x0104, alm.LootHeadWide, 2},
		{"field04_1", 1, alm.LootHeadWide + alm.LootElementField04, 2},
	}
	for _, tc := range fixedCases {
		if err := e.SetField(mapedit.KindLoot, 1, tc.key, tc.value); err != nil {
			t.Fatalf("SetField(%s=%d): %v", tc.key, tc.value, err)
		}
		saved, err := e.SaveBytes()
		if err != nil {
			t.Fatal(err)
		}
		assertOnlyByteRangeChanged(t, tc.key, raw, saved, sack1Base+tc.off, sack1Base+tc.off+tc.width)
		m, err := alm.OpenROM2(saved)
		if err != nil {
			t.Fatal(err)
		}
		loot, err := m.Loot()
		if err != nil {
			t.Fatal(err)
		}
		// Binding of item 1 stays untouched.
		if loot.Records[1].Elements[0].TileMarkerIndex != 2 {
			t.Fatalf("%s disturbed item binding: %+v", tc.key, loot.Records[1].Elements[0])
		}
		if !e.Undo() {
			t.Fatalf("%s Undo failed", tc.key)
		}
		if got, _ := e.SaveBytes(); !bytes.Equal(got, raw) {
			t.Fatalf("%s Undo did not restore original bytes", tc.key)
		}
	}

	// Add an item and remove it; check count and payloadSize survive reload.
	if err := e.SetField(mapedit.KindLoot, 1, "additem", 0x0e41); err != nil {
		t.Fatal(err)
	}
	m := secondReload(t, e)
	loot, err := m.Loot()
	if err != nil || len(loot.Records[1].Elements) != 2 || loot.Records[1].Elements[1].ItemCode() != 0x0e41 {
		t.Fatalf("added item: %+v, %v", loot.Records[1], err)
	}
	if err := e.SetField(mapedit.KindLoot, 1, "item2", 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := e.SaveBytes(); !bytes.Equal(got, raw) {
		t.Fatal("adding then removing an item did not restore original bytes")
	}
	e.Undo()
	e.Undo()
	if got, _ := e.SaveBytes(); !bytes.Equal(got, raw) {
		t.Fatal("Undo of add/remove did not restore original bytes")
	}
	_ = binary.LittleEndian
}
