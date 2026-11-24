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

// TestEditorUnitManaMaxReachesTheMap: the unit window, in English and
// Russian, sets the unit's maximum mana as it sets maximum health; the map
// holds it at +0x26, a reload reads it, the map spawns the unit with that
// maximum and its current mana at most the maximum, 0 keeps the derived
// maximum, and Undo restores the map byte for byte (DIV-5150).
func TestEditorUnitManaMaxReachesTheMap(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		t.Run(lang, func(t *testing.T) {
			s, _ := viewsMap(t)
			// The unit holds the word every installed record holds at +0x26
			// (ALM-TAILU16-082).
			if err := s.model.SetField(mapedit.KindUnit, 0, "manamax", -1); err != nil {
				t.Fatal(err)
			}
			raw := s.model.Bytes()
			s.inspector.archives.Base.Profile.Language = lang
			s.unitViews, s.inspector.recordOptions = nil, nil
			sheet := s.unitViewSheets()[0].Values
			e, c := motionWindow(t, s, lang, false, false)
			fields := e.ObjectFields()
			if f := fields[fieldIndex(t, fields, "manamax")]; !f.Inherit || f.Value != -1 || f.Inherited != strconv.Itoa(int(sheet.MaxMana)) ||
				f.Label != s.inspector.word("editor.unit.override.manamax") {
				t.Fatalf("empty maximum mana %q: inherit %v value %d shows %q, want the kind's %d", f.Label, f.Inherit, f.Value, f.Inherited, sheet.MaxMana)
			}
			typeField(t, e, c, "manamax", 300)
			if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "mana", Value: 900, Step: 1}}); err != nil {
				t.Fatal(err)
			}
			if rec, _ := s.model.UnitRecord(0); binary.LittleEndian.Uint16(rec[alm.UnitManaMax:]) != 300 {
				t.Fatalf("record maximum mana %d after the edit, want 300", binary.LittleEndian.Uint16(rec[alm.UnitManaMax:]))
			}
			// spawned reloads the saved map and spawns it: the tail reads
			// tail, the unit has want mana of want.
			spawned := func(tail uint16, want int32) {
				t.Helper()
				m, err := alm.Open(s.model.Bytes())
				if err != nil {
					t.Fatal(err)
				}
				w, err := mapload.FromALMWith(m, s.inspector.table, mapload.DifficultyNormal)
				if err != nil {
					t.Fatal(err)
				}
				if got := w.Entities()[0]; got.MaxMana != want || got.Mana != want || m.Units[0].Tail.ManaMax != tail {
					t.Fatalf("spawned mana %d/%d tail %d, want %d/%d tail %d", got.Mana, got.MaxMana, m.Units[0].Tail.ManaMax, want, want, tail)
				}
			}
			spawned(300, 300)
			if _, err := s.SetFields(ui.KindUnit, 0, []ui.InspectionFieldValue{{Key: "manamax", Value: 0, Step: 1}, {Key: "mana", Value: -1, Step: 2}}); err != nil {
				t.Fatal(err)
			}
			spawned(0, sheet.MaxMana)
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
