package game

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// TestEditorUnitHealthMaxGoldAndLootReachTheMap: the detailed view of the
// unit window, in English and Russian, sets the unit's maximum health, its
// gold and its loot switch; each writes the map field that holds it, the
// map spawns the unit with that maximum, its current health at most the
// maximum, that purse and its loot discarded, and Undo restores the map
// (DIV-4945, DIV-4946).
func TestEditorUnitHealthMaxGoldAndLootReachTheMap(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(lang, func(t *testing.T) {
			s, raw := viewsMap(t)
			s.inspector.archives.Base.Profile.Language = lang
			s.inspector.recordOptions = nil
			sheet := s.unitViewSheets()[0].Values
			e, c := motionWindow(t, s, lang, false, false)
			fields := e.ObjectFields()
			if f := fields[fieldIndex(t, fields, "hpmax")]; !f.Inherit || f.Value != 0 || f.Inherited != strconv.Itoa(int(sheet.MaxHP)) {
				t.Fatalf("empty maximum health: inherit %v value %d shows %q, want the kind's %d", f.Inherit, f.Value, f.Inherited, sheet.MaxHP)
			}
			if f := fields[fieldIndex(t, fields, unitGoldField)]; f.Value != 0 || f.Label != s.inspector.word("editor.unit.field.gold") {
				t.Fatalf("gold row %q at %d, want 0", f.Label, f.Value)
			}
			if f := fields[fieldIndex(t, fields, "noloot")]; len(f.Choices) != 2 || f.Value != 0 {
				t.Fatalf("loot switch offers %v at %d, want two choices at 0", f.Choices, f.Value)
			}
			typeField(t, e, c, "hpmax", 640)
			typeField(t, e, c, unitGoldField, 75)
			if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "noloot", Value: 1, Step: 1}, {Key: "hp", Value: 900, Step: 2}}); err != nil {
				t.Fatal(err)
			}
			rec, _ := s.model.UnitRecord(0)
			if v := binary.LittleEndian.Uint16(rec[alm.UnitHealthMax:]); v != 640 || binary.LittleEndian.Uint32(rec[alm.UnitFlags:])&alm.UnitFlagNoLoot == 0 {
				t.Fatalf("record maximum %d flags %#x after the edits", v, binary.LittleEndian.Uint32(rec[alm.UnitFlags:]))
			}
			m, err := alm.Open(s.model.Bytes())
			if err != nil {
				t.Fatal(err)
			}
			stock, err := s.model.UnitStock(0)
			if err != nil || stock < 0 {
				t.Fatalf("unit stock %d: %v", stock, err)
			}
			loot, err := m.Loot()
			if err != nil || loot.Records[stock].Gold != 75 || m.Units[0].Tail.HealthMax != 640 || !m.Units[0].Tail.NoLoot {
				t.Fatalf("saved map: gold %d maximum %d no loot %v (%v)", loot.Records[stock].Gold, m.Units[0].Tail.HealthMax, m.Units[0].Tail.NoLoot, err)
			}
			w, err := mapload.FromALMWith(m, s.inspector.table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			if got := w.Entities()[0]; got.MaxHP != 640 || got.HP != 640 || got.Purse != 75 || !got.SuppressCorpseLoot {
				t.Fatalf("spawned health %d/%d purse %d suppress %v, want 640/640, 75 and true", got.HP, got.MaxHP, got.Purse, got.SuppressCorpseLoot)
			}
			for range 3 {
				if _, err := s.Undo(); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(s.model.Bytes(), raw) {
				t.Fatal("Undo did not restore the map")
			}
		})
	}
}

// TestEditorUnitGoldAddsTheUnitsOwnLootRecord: gold on a unit with no loot
// record of its own adds one; gold 0 there writes nothing.
func TestEditorUnitGoldAddsTheUnitsOwnLootRecord(t *testing.T) {
	_, s, raw := nativeInventoryEditor(t)
	if stock, err := s.model.UnitStock(0); err != nil || stock >= 0 {
		t.Fatalf("fixture unit stock %d: %v, want none", stock, err)
	}
	if _, err := s.SetField(ui.KindUnit, 0, unitGoldField, 0); err != nil || !bytes.Equal(s.model.Bytes(), raw) {
		t.Fatalf("gold 0 on a unit without a record changed the map (%v)", err)
	}
	if _, err := s.SetField(ui.KindUnit, 0, unitGoldField, 50); err != nil {
		t.Fatal(err)
	}
	stock, err := s.model.UnitStock(0)
	if err != nil || stock < 0 {
		t.Fatalf("unit stock %d: %v", stock, err)
	}
	if v, err := s.model.FieldValue(mapedit.KindLoot, stock, "gold"); err != nil || v != 50 {
		t.Fatalf("the unit's record holds gold %d (%v), want 50", v, err)
	}
}
