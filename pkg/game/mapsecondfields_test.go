package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func secondFixtureMap() []byte {
	le := binary.LittleEndian
	cell := func(c uint32) uint32 { return c<<8 | 0x80 }
	meta := make([]byte, 660)
	le.PutUint32(meta[0:], 32)
	le.PutUint32(meta[4:], 32)
	le.PutUint32(meta[alm.MetaGroupCount:], 2)
	le.PutUint32(meta[alm.MetaObjectCount:], 2)
	le.PutUint32(meta[alm.MetaUnitCount:], 2)
	le.PutUint32(meta[alm.MetaLootCount:], 1)

	groups := make([]byte, 2*alm.GroupSize)
	copy(groups[alm.GroupName:], "PlayerOne")
	le.PutUint32(groups[alm.GroupColour:], 1)
	copy(groups[alm.GroupSize+alm.GroupName:], "PlayerTwo")
	le.PutUint32(groups[alm.GroupSize+alm.GroupColour:], 2)

	st0 := make([]byte, alm.ObjectSize)
	le.PutUint32(st0[alm.ObjectX:], cell(12))
	le.PutUint32(st0[alm.ObjectY:], cell(12))
	le.PutUint32(st0[alm.ObjectKind:], 34)
	le.PutUint16(st0[alm.ObjectPlacement:], 100)
	le.PutUint16(st0[alm.ObjectOwner:], 1)
	le.PutUint16(st0[alm.ObjectIdentity:], 1)

	st1 := make([]byte, alm.ObjectSize+alm.ObjectExtSize)
	le.PutUint32(st1[alm.ObjectX:], cell(16))
	le.PutUint32(st1[alm.ObjectY:], cell(16))
	le.PutUint32(st1[alm.ObjectKind:], alm.SecondObjectExtBit|alm.ObjectExtKind)
	le.PutUint16(st1[alm.ObjectPlacement:], 100)
	le.PutUint16(st1[alm.ObjectOwner:], 2)
	le.PutUint16(st1[alm.ObjectIdentity:], 2)
	le.PutUint32(st1[alm.ObjectWidth:], 4)
	le.PutUint32(st1[alm.ObjectHeight:], 5)

	u0 := make([]byte, alm.SecondUnitSize)
	le.PutUint32(u0[alm.UnitX:], cell(10))
	le.PutUint32(u0[alm.UnitY:], cell(10))
	le.PutUint16(u0[alm.SecondUnitClass:], 7)
	le.PutUint16(u0[alm.SecondUnitField0A:], 2)
	le.PutUint32(u0[alm.SecondUnitFlags:], 0)
	le.PutUint32(u0[alm.SecondUnitFlags10:], 0)
	le.PutUint32(u0[alm.SecondUnitDef:], 600)
	le.PutUint32(u0[alm.SecondUnitOwner:], 1)
	le.PutUint32(u0[alm.SecondUnitField20:], 10)
	le.PutUint16(u0[alm.SecondUnitHP:], 0xffff)
	le.PutUint16(u0[alm.SecondUnitHPMax:], 0xffff)
	le.PutUint32(u0[alm.SecondUnitTag:], 3|(cell(10)&0xffff)<<16)
	le.PutUint32(u0[alm.SecondUnitGroup:], 5)

	u1 := make([]byte, alm.SecondUnitSize)
	le.PutUint32(u1[alm.UnitX:], cell(14))
	le.PutUint32(u1[alm.UnitY:], cell(14))
	le.PutUint16(u1[alm.SecondUnitClass:], 1)
	le.PutUint16(u1[alm.SecondUnitField0A:], 1)
	le.PutUint32(u1[alm.SecondUnitFlags:], alm.SecondUnitFlagHuman)
	le.PutUint32(u1[alm.SecondUnitDef:], 2113)
	le.PutUint32(u1[alm.SecondUnitOwner:], 2)
	le.PutUint32(u1[alm.SecondUnitField20:], 10)
	le.PutUint16(u1[alm.SecondUnitHP:], 150)
	le.PutUint16(u1[alm.SecondUnitHPMax:], 150)
	le.PutUint32(u1[alm.SecondUnitTag:], 4|(cell(14)&0xffff)<<16)
	le.PutUint32(u1[alm.SecondUnitGroup:], 6)

	sack := make([]byte, alm.LootHeadWide+alm.LootElementSize)
	le.PutUint32(sack[alm.LootCount:], 1)
	le.PutUint32(sack[alm.LootX:], cell(18))
	le.PutUint32(sack[alm.LootY:], cell(18))
	le.PutUint32(sack[alm.LootGold:], 100)
	le.PutUint32(sack[alm.LootHeadWide:], 0x0102)

	payloads := []struct {
		id uint32
		b  []byte
	}{
		{0, meta},
		{1, make([]byte, 2*32*32)},
		{2, make([]byte, 32*32)},
		{3, make([]byte, 32*32)},
		{5, groups},
		{4, append(st0, st1...)},
		{8, sack},
		{6, append(u0, u1...)},
		{7, make([]byte, 12)},
	}
	out := make([]byte, alm.FileHeaderSize)
	copy(out, "M7R\x00")
	le.PutUint32(out[4:], alm.FileHeaderSize)
	le.PutUint32(out[8:], 4*32*32+72)
	le.PutUint32(out[12:], uint32(len(payloads)))
	le.PutUint32(out[16:], 1300)
	for _, p := range payloads {
		h := make([]byte, alm.RecordHeaderSize)
		le.PutUint32(h[0:], 7)
		le.PutUint32(h[4:], alm.RecordHeaderSize)
		le.PutUint32(h[8:], uint32(len(p.b)))
		if p.id == 0 {
			le.PutUint32(h[8:], 644)
		}
		le.PutUint32(h[12:], p.id)
		out = append(out, h...)
		out = append(out, p.b...)
	}
	return out
}

func secondTestInspector(t *testing.T) *MapInspector {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	if err := os.WriteFile(path, synth.Archive(nil), 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	unitRow := func(name string, serverID int32) dbEntry {
		p := make([]int32, 62)
		p[4] = 80
		p[55] = serverID
		return dbEntry{name: name, params: p}
	}
	humanRow := func(name string, serverID int32) dbEntry {
		p := make([]int32, 26)
		p[4] = 150
		p[24] = serverID
		return dbEntry{name: name, params: p}
	}
	buildings := make(dbCollection, 36)
	buildings[5] = dbEntry{name: "Mill", params: []int32{3, 3, 4, 1000, 186, 511}}
	buildings[33] = dbEntry{name: "Vertical Wooden Bridge", params: []int32{1, 1, 0, 0, 1, 1}}
	buildings[34] = dbEntry{name: "Shop", params: []int32{3, 3, 4, 0, 447, 511}}
	weapons := make(dbCollection, 5)
	weapons[2] = dbEntry{name: "Dagger", params: make([]int32, 17)}
	weapons[4] = dbEntry{name: "Long Sword", params: make([]int32, 17)}
	magic := make(dbCollection, 66)
	magic[65] = dbEntry{name: "Potion Body", params: make([]int32, 2)}

	tab := readUnder(base.GameROM2, &mapload.Table{
		Units:      dbCollection{{}, unitRow("Goblin_Pike", 600), unitRow("Goblin_Pike.2", 601)},
		Humans:     dbCollection{{}, humanRow("2F_Knight1", 2113), humanRow("2A_Knight1", 2114)},
		Buildings:  buildings,
		Weapons:    weapons,
		MagicItems: magic,
		Names:      data.ItemNames{0x0102: "Dagger", 0x0104: "Long Sword"},
	})
	return &MapInspector{
		archives:   &Archives{Containers: fsys, Base: base.Match{Profile: base.Profile{Game: base.GameROM2}}},
		tiles:      &terrain.Tileset{},
		structures: &terrain.StructureSet{},
		table:      tab,
		// No campaign to read: the test-play guest number is settled.
		guestNumber: new(int),
	}
}

func recordByKind(t *testing.T, d *ui.InspectionDocument, kind ui.RecordKind, index int) ui.InspectionRecord {
	t.Helper()
	for _, r := range d.Records {
		if r.Kind == kind && r.Index == index {
			return r
		}
	}
	t.Fatalf("no %s record %d", kind, index)
	return ui.InspectionRecord{}
}

func TestSecondEditorFieldsAndCardsForUnitStructureAndSack(t *testing.T) {
	raw := secondFixtureMap()
	x := secondTestInspector(t)
	d, err := x.inspect(raw, "second.alm")
	if err != nil {
		t.Fatal(err)
	}
	host := d.Edits.(*mapInspectionEdits)

	// Record cards come from the field list and carry no technical raw lines.
	forbidden := []string{"X/Y raw", "Flags +10", "Definition unresolved", "Class/sub", "HP raw", "Raw 0C/0E/12", "Extension:", "art key", "raw04", "enchantment"}
	checkCard := func(r ui.InspectionRecord, mustContain ...string) {
		t.Helper()
		joined := strings.Join(r.Details, "\n")
		for _, bad := range forbidden {
			if strings.Contains(joined, bad) || strings.Contains(r.Warning, bad) {
				t.Fatalf("%s %d card carries technical line %q:\n%s\nwarning: %s", r.Kind, r.Index, bad, joined, r.Warning)
			}
		}
		for _, want := range mustContain {
			if !strings.Contains(joined, want) {
				t.Fatalf("%s %d card lacks %q:\n%s", r.Kind, r.Index, want, joined)
			}
		}
	}
	checkCard(recordByKind(t, d, ui.KindUnit, 0), "Owner: 1: PlayerOne", "Definition: 600: Goblin_Pike",
		"Lookup index 0x08 (picks the object the load builds): 7", "Word 0x0a, 0 to 7 in the installed maps (Medium: no game reader found): 2",
		"Flags 0x10: 1, 2, 4 offered at an Inn, 8 drops nothing at death (Medium): 0", "Dword 0x20, 0 to 14 in the installed maps (Medium: no game reader found): 10")
	checkCard(recordByKind(t, d, ui.KindUnit, 1), "Owner: 2: PlayerTwo", "Definition: 2113: 2F_Knight1", "Health (-1 = default): 150", "Maximum health (-1 = default): 150")
	checkCard(recordByKind(t, d, ui.KindStructure, 0), "Type: 34: Shop", "Owner: 1: PlayerOne", "Word 0x0c (no game reader found for a Shop, Inn or Pointer): 100")
	checkCard(recordByKind(t, d, ui.KindStructure, 1), "Type: 33: Vertical Wooden Bridge", "Owner: 2: PlayerTwo",
		"Word 0x0c: 0 clears the building's word +0x42 (no reader of another value found): 100",
		"Size X (replaces the building row's, Medium): 4", "Size Y (replaces the building row's, Medium): 5")
	checkCard(recordByKind(t, d, ui.KindSack, 0), "Word 0x10 (the ground bag is placed with it): 100", "Item 1: 258: Dagger",
		"Item 1: word 0x04 (no game reader found for a ground sack): 0")

	// Unit fields list no level/experience and offer catalogue definition choices per arm.
	u0Fields := host.Fields(ui.KindUnit, 0)
	u0Keys := make([]string, len(u0Fields))
	for i, f := range u0Fields {
		u0Keys[i] = f.Key
		if f.Key == "level" || f.Key == "experience" {
			t.Fatalf("unit field list invented %q", f.Key)
		}
	}
	if !slices.Equal(u0Keys, []string{"owner", "definition", "hp", "hpmax", "flags", "group", "field08", "field0a", "field10", "field20", "spellbook"}) {
		t.Fatalf("unit fields = %v", u0Keys)
	}
	if def, ok := fieldOfKey(u0Fields, "definition"); !ok || len(def.Choices) != 2 || def.Choices[0].Label != "600: Goblin_Pike" {
		t.Fatalf("unit 0 definition choices = %+v", def.Choices)
	}
	if def, ok := fieldOfKey(host.Fields(ui.KindUnit, 1), "definition"); !ok || len(def.Choices) != 2 || def.Choices[0].Label != "2113: 2F_Knight1" {
		t.Fatalf("unit 1 definition choices = %+v", def.Choices)
	}

	// Edit unit, structure and sack fields through the host, verify save/reload and undo.
	if _, err := host.SetField(ui.KindUnit, 0, "owner", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindUnit, 0, "definition", 601); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindUnit, 0, "hp", 75); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindUnit, 0, "field0a", 4); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindStructure, 0, "kind", 99); err == nil {
		t.Fatal("invalid Buildings kind 99 was accepted")
	}
	if _, err := host.SetField(ui.KindStructure, 0, "owner", 9); err == nil {
		t.Fatal("out-of-roster structure owner 9 was accepted")
	}
	if _, err := host.SetField(ui.KindStructure, 0, "kind", 5); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindStructure, 1, "field14", 6); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindSack, 0, "item1", 0x0f01); err == nil {
		t.Fatal("invalid item code A=15 was accepted")
	}
	// 0x0e41 (Potion Body) is not in table.Names, resolved through MagicItems (R2-ENGINE-1513).
	if _, err := host.SetField(ui.KindSack, 0, "additem", 0x0e41); err != nil {
		t.Fatal(err)
	}
	if _, err := host.SetField(ui.KindSack, 0, "field10", 250); err != nil {
		t.Fatal(err)
	}

	target, err := host.SaveAs(filepath.Join(t.TempDir(), "edited.alm"))
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := x.Open(target)
	if err != nil || !bytes.Equal(reloaded.SourceBytes, saved) {
		t.Fatalf("reload: %v", err)
	}
	checkCard(recordByKind(t, reloaded, ui.KindUnit, 0), "Owner: 2: PlayerTwo", "Definition: 601: Goblin_Pike.2", "Health (-1 = default): 75", "Word 0x0a, 0 to 7 in the installed maps (Medium: no game reader found): 4")
	checkCard(recordByKind(t, reloaded, ui.KindStructure, 0), "Type: 5: Mill", "Word 0x0c: 0 clears the building's word +0x42")
	checkCard(recordByKind(t, reloaded, ui.KindStructure, 1), "Type: 33: Vertical Wooden Bridge", "Size X (replaces the building row's, Medium): 6")
	checkCard(recordByKind(t, reloaded, ui.KindSack, 0), "Word 0x10 (the ground bag is placed with it): 250", "Item 1: 258: Dagger", "Item 2: 3649: Potion Body")

	// Undo all 8 edits back to exact original bytes.
	for i := 0; i < 8; i++ {
		if _, err := host.Undo(); err != nil {
			t.Fatalf("Undo %d: %v", i, err)
		}
	}
	if !bytes.Equal(host.model.Bytes(), raw) {
		t.Fatal("Undo did not restore original second-game map bytes")
	}
}

func fieldOfKey(fields []ui.InspectionField, key string) (ui.InspectionField, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return ui.InspectionField{}, false
}

func TestFirstEditorUnitFieldListUnchanged(t *testing.T) {
	b := inspectionFixtureBytes(t)
	_, e, _ := openTestMapEditor(t, b)
	host := e.Document().Edits.(*mapInspectionEdits)
	fields := host.Fields(ui.KindUnit, 0)
	for _, key := range []string{"owner", "group", "hp", "hpmax", "mana", "body", "facing", "name", "speaker", "gold", "loot"} {
		f, ok := fieldOfKey(fields, key)
		if !ok {
			keys := make([]string, len(fields))
			for i, f := range fields {
				keys[i] = f.Key
			}
			t.Fatalf("first-game unit field %q missing from %v", key, keys)
		}
		if f.Section == "" {
			t.Fatalf("first-game unit field %q lost its section", key)
		}
	}
}
