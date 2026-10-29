package game

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func nativeInventoryEditor(t *testing.T) (*FrontEnd, *mapInspectionEdits, []byte) {
	t.Helper()
	f, s, raw := nativeEquipmentEditor(t)
	items := make(dbCollection, 9)
	params := make([]int32, 16)
	params[0], params[3] = 10, 1
	items[8] = dbEntry{name: "PotionHealth", params: params, strings: []string{"health=5"}}
	f.Table.MagicItems = items
	f.Table.Names[0x0e08] = "Health potion"
	return f, s, raw
}

func TestEditorCarriedDraftAndDerivedLootHaveNoWriteControls(t *testing.T) {
	_, s, raw := nativeInventoryEditor(t)
	_, fields, _, err := s.UnitSettings(0, []ui.InspectionFieldValue{{Key: "carryadd", Value: 0x0e08}})
	if err != nil {
		t.Fatal(err)
	}
	f := fields[fieldIndex(t, fields, "holdings")]
	text := ""
	for _, line := range f.Lines {
		text += line.Text + "\n"
	}
	if !f.ReadOnly || !strings.Contains(text, "Health potion") || !strings.Contains(text, "Corpse") || !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("draft loot is missing or mutates the map", f, text)
	}
	if _, err := s.SetField("Unit", 0, "holdings", 0); err == nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("derived loot accepted a write", err)
	}
	if _, err := s.SetField("Unit", 0, "carryadd", 0x0e08); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	sheets, err := unitSheets(m, s.inspector.table)
	if err != nil || len(sheets[0].Carried) != 1 || len(sheets[0].Carried[0].Effects) != 1 {
		t.Fatal("potion payload duplicated by the stock writer", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("Add carried Undo was not exact", err)
	}
}

func TestEditorCarriedEditsKeepIndicesAcrossWornMaterialization(t *testing.T) {
	_, s, _ := nativeInventoryEditor(t)
	for _, code := range []int64{0x102, 0x0e08} {
		if _, err := s.SetField("Unit", 0, "carryadd", code); err != nil {
			t.Fatal(err)
		}
	}
	before := s.model.Bytes()
	if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{{Key: "wear7", Value: 0}, {Key: "carry1", Value: 0}, {Key: "carry2", Value: 0x101}}); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	sheets, err := unitSheets(m, s.inspector.table)
	if err != nil || len(sheets[0].Carried) != 1 || sheets[0].Carried[0].Code != 0x101 || sheets[0].Worn[6].Code != 0 {
		t.Fatal("compound edit shifted a carried index", err)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("compound carried/worn Undo was not exact", err)
	}
}

func TestEditorCarriedDuplicateOwnsItsRecipe(t *testing.T) {
	_, s, _ := nativeInventoryEditor(t)
	if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Unit", 0, "carryrecipe1", -1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Enchantment", 0, "addeffect", 17); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Enchantment", 0, "low1", 5); err != nil {
		t.Fatal(err)
	}
	before := s.model.Bytes()
	if _, err := s.SetField("Unit", 0, "carry1", -1); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	loot, _ := m.Loot()
	if len(m.Enchantments) != 2 || len(loot.Records[0].Elements) != 2 || loot.Records[0].Elements[0].TileMarkerIndex == loot.Records[0].Elements[1].TileMarkerIndex {
		t.Fatal("duplicate shared its recipe")
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("duplicate Undo was not exact", err)
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetField("Enchantment", 1, "low1", 9); err != nil {
		t.Fatal(err)
	}
	m, _ = s.model.Map()
	if m.Enchantments[0].Elements[0].Low != 5 || m.Enchantments[1].Elements[0].Low != 9 {
		t.Fatal("editing the duplicate changed its source")
	}
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "inventory.alm"))
	if err != nil {
		t.Fatal(err, s.Report())
	}
	doc, err := s.inspector.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := doc.Edits.(*mapInspectionEdits).model.Map()
	if err != nil {
		t.Fatal(err)
	}
	items, err := unitSheets(reopened, s.inspector.table)
	if err != nil || len(items[0].Carried) != 2 || len(items[0].Carried[0].Effects) != 1 || len(items[0].Carried[1].Effects) != 1 || items[0].Carried[0].Effects[0].Operand != 5 || items[0].Carried[1].Effects[0].Operand != 9 {
		t.Fatal("Save/reopen lost independent carried recipes", err)
	}
}

func TestEditorRemovingCarriedItemDiscardsItsPendingRecipe(t *testing.T) {
	_, s, _ := nativeInventoryEditor(t)
	if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{{Key: "carryrecipe1", Value: -1}, {Key: "carry1", Value: 0}}); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	loot, _ := m.Loot()
	if len(loot.Records[0].Elements) != 0 || len(m.Enchantments) != 0 {
		t.Fatal("removed item left a pending recipe")
	}
}

func TestEditorCarriedItemsSurviveMissionSAV(t *testing.T) {
	f, s, _ := nativeInventoryEditor(t)
	for _, code := range []int64{0x102, 0x0e08} {
		if _, err := s.SetField("Unit", 0, "carryadd", code); err != nil {
			t.Fatal(err)
		}
	}
	cold := nativeEditorMissionSAV(t, f, s, nativeInventoryEditor)
	items, _ := cold.live.world.CarriedItems(0)
	if len(items) != 2 || items[0].Code != 0x102 || items[1].Code != 0x0e08 || len(items[1].Effects) != 1 {
		t.Fatal("cold LOAD lost carried item state", items)
	}
}

func TestEditorDerivedLootMatchesTerminalDeathAndSuppression(t *testing.T) {
	for _, suppressed := range []bool{false, true} {
		f, s, _ := nativeInventoryEditor(t)
		if suppressed {
			row := f.Table.Humans.(dbCollection)[1]
			row.name = "NPCGuard"
			f.Table.Humans.(dbCollection)[1] = row
		}
		if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
			t.Fatal(err)
		}
		m, _ := s.model.Map()
		sheets, err := unitSheets(m, f.Table)
		if err != nil || sheets[0].Suppressed != suppressed {
			t.Fatal("wrong preview suppression", err)
		}
		w, err := mapload.FromALMWith(m, f.Table, mapload.DifficultyNormal)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.HeadlessKill(0); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 128; i++ {
			sim.Step(w, nil)
		}
		if suppressed {
			if len(w.Sacks()) != 0 {
				t.Fatal("suppressed template dropped an item", w.Sacks())
			}
		} else {
			if len(w.Sacks()) != 1 || len(w.Sacks()[0].ItemInstances) != 4 {
				t.Fatal("terminal loot differs from carried plus eligible worn", w.Sacks())
			}
		}
	}
}
