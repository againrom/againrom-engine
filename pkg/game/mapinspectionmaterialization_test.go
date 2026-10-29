package game

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func equalNativeItems(a, b []sim.ItemInstance) bool {
	if len(a) != len(b) {
		return false
	}
	for i, item := range a {
		if !sim.StackStateEqual(sim.StackItem(item, 1), sim.StackItem(b[i], 1)) {
			return false
		}
	}
	return true
}

func mustEditorMap(t *testing.T, s *mapInspectionEdits) *alm.Map {
	t.Helper()
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func spellEquipmentEditor(t *testing.T) (*FrontEnd, *mapInspectionEdits, []byte) {
	t.Helper()
	f, s, _ := nativeEquipmentEditor(t)
	params := make([]int32, 22)
	params[20], params[21] = 80, 735
	f.Table.Spells = dbCollection{{}, {name: "Fire Arrow", params: params}}
	weapons := make(dbCollection, f.Table.Weapons.Len())
	for i := range weapons {
		weapons[i] = dbEntry{name: f.Table.Weapons.EntryName(i), params: append([]int32(nil), f.Table.Weapons.EntryParams(i)...)}
	}
	weapons[1].params[2], weapons[1].params[3] = 250, 600
	f.Table.Weapons = weapons
	row := f.Table.Humans.(dbCollection)[1]
	row.strings[0] = "Rare Steel Sword{speed=5,castSpell=Fire_Arrow:15,defence=2}"
	row.strings[2] = "Iron Mail{speed=5,skillpike=5,healthMax=10}"
	f.Table.Humans.(dbCollection)[1] = row
	s.unitViews = nil
	return f, s, s.model.Bytes()
}

func TestEditorOrdinaryEmptyRetainsSpellWeaponAndInherit(t *testing.T) {
	_, s, raw := spellEquipmentEditor(t)
	w0, err := mapload.FromALMWith(mustEditorMap(t, s), s.inspector.table, mapload.DifficultyNormal)
	if err != nil || w0.Entities()[0].Load <= w0.Entities()[0].Capacity {
		t.Fatal("fixture needs an overloaded person", err)
	}
	before, err := unitSheets(mustEditorMap(t, s), s.inspector.table)
	if err != nil || before[0].Worn[0].Code != 0x1121 || before[0].Worn[0].Price <= 0 || len(before[0].Worn[0].Effects) != 3 || before[0].Worn[0].Effects[1] != (sim.ItemEffect{Kind: 41, Operand: 1 | 15<<16}) {
		t.Fatal("fixture needs a spell weapon", err)
	}
	for _, change := range []ui.InspectionFieldValue{{Key: "wear7", Value: 0}, {Key: "wear1", Value: -1}} {
		if _, err := s.SetField("Unit", 0, change.Key, change.Value); err != nil {
			t.Fatal(err)
		}
		after, err := unitSheets(mustEditorMap(t, s), s.inspector.table)
		want := before[0].Worn
		want[6] = sim.ItemInstance{}
		if err != nil || !equalNativeItems(want[:], after[0].Worn[:]) || !equalNativeItems(before[0].Carried, after[0].Carried) {
			t.Fatalf("%s changed retained items: %v\nwant %+v\ngot %+v", change.Key, err, want, after[0].Worn)
		}
	}
	if _, err := s.SetField("Unit", 0, "wear7", -1); err != nil {
		t.Fatal(err)
	}
	after, err := unitSheets(mustEditorMap(t, s), s.inspector.table)
	if err != nil || !equalNativeItems(before[0].Worn[:], after[0].Worn[:]) || !reflect.DeepEqual(before[0].Values, after[0].Values) {
		t.Fatalf("Empty then Inherit changed the original unit: %v\nbefore %+v\nafter %+v", err, before[0].Values, after[0].Values)
	}
	w3, err := mapload.FromALMWith(mustEditorMap(t, s), s.inspector.table, mapload.DifficultyNormal)
	if err != nil || !reflect.DeepEqual(w0.Entities(), w3.Entities()) {
		t.Fatal("restoring an overloaded person's gear changed its actor", err)
	}
	for s.CanUndo() {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("materialization Undo changed bytes")
	}
}

func TestEditorNativeRecipeRetainsBookAndRejectsLoss(t *testing.T) {
	f, s, _ := spellEquipmentEditor(t)
	magic := make(dbCollection, 24)
	magic[23] = dbEntry{name: "Book Astral", params: []int32{200}}
	f.Table.MagicItems = magic
	stock, err := s.model.AddUnitStock(0)
	if err != nil {
		t.Fatal(err)
	}
	book := mapload.ItemInstanceFromCode(0x0e17, f.Table)
	book.Effects = []sim.ItemEffect{{Kind: 42, Operand: 1}}
	book.Price = 735
	if err := s.writeStockItem(s.model, stock, book, 0, false); err != nil {
		t.Fatal(err)
	}
	m := mustEditorMap(t, s)
	loot, _ := m.Loot()
	got, err := mapload.ItemInstanceFromLoot(loot.Records[stock].Elements[0], m, f.Table)
	if err != nil || !reflect.DeepEqual(book, got) || m.Enchantments[0].SpellRaw != 1 || len(m.Enchantments[0].Elements) != 0 {
		t.Fatal("book did not roundtrip through the spell head", got, err)
	}
	for _, suffix := range []string{"speed=5:duration60", "castSpell=Fire_Arrow:15,castSpell=Fire_Arrow:20"} {
		_, trial, _ := spellEquipmentEditor(t)
		row := trial.inspector.table.Humans.(dbCollection)[1]
		row.strings[0] = "Iron Sword{" + suffix + "}"
		trial.inspector.table.Humans.(dbCollection)[1] = row
		raw := trial.model.Bytes()
		_, _, _, err := trial.UnitSettings(0, []ui.InspectionFieldValue{{Key: "wear7", Value: 0}})
		if err == nil || !strings.Contains(err.Error(), "native recipe cannot retain") || !bytes.Equal(raw, trial.model.Bytes()) || trial.CanUndo() {
			t.Fatal("loss was not explained before Apply", suffix, err)
		}
		if _, err := trial.SetField("Unit", 0, "wear7", 0); err == nil || !bytes.Equal(raw, trial.model.Bytes()) {
			t.Fatal("Apply accepted a lossy recipe", suffix, err)
		}
	}
}
