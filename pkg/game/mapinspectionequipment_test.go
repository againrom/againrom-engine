package game

import (
	"bytes"
	"image"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

func nativeEquipmentEditor(t *testing.T) (*FrontEnd, *mapInspectionEdits, []byte) {
	t.Helper()
	f, s, _ := nativeIdentityEditor(t)
	gear := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials, f.Table.Weapons, f.Table.Shields = gear.Shapes, gear.Materials, gear.Weapons, gear.Shields
	f.Table.Shapes = fixtureScale{names: []string{"Common", "Rare"}, doubles: [][]float64{{1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 1, 1, 1, 1}}}
	f.Table.Materials = fixtureScale{names: []string{"Iron", "Steel"}, doubles: [][]float64{{1, 1, 1, 1, 1, 1, 1, 1, 1}, {1, 1, 1, 1, 1, 1, 1, 1, 1}}}
	params := make([]int32, 17)
	params[4], params[9], params[10], params[15] = 7, 12, 2, 3
	f.Table.Armors = dbCollection{{}, {name: "Mail", params: params}}
	for _, row := range []int{1, 2} {
		e := f.Table.Humans.(dbCollection)[row]
		e.strings = []string{"Iron Sword", "Iron Ward", "Iron Mail{speed=5}"}
		f.Table.Humans.(dbCollection)[row] = e
	}
	f.Table.Names = data.ItemNames{0x101: "Sword", 0x102: "Mace", 0x201: "Ward", 0x701: "Mail", 0x1102: "Steel Mace", 0x122: "Rare Mace", 0x1122: "Rare Steel Mace"}
	s.unitViews, s.inspector.recordOptions = nil, nil
	return f, s, s.model.Bytes()
}

func TestEditorWornRowsPreviewDisplacementAndNPCEmpty(t *testing.T) {
	_, s, raw := nativeEquipmentEditor(t)
	_, fields, _, err := s.UnitSettings(0, []ui.InspectionFieldValue{{Key: "wear1", Value: 0x102}})
	if err != nil || !strings.Contains(fields[fieldIndex(t, fields, "wear1")].Meaning, "Sword") || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("draft did not disclose displacement without writes", err)
	}
	if _, err := s.SetField("Unit", 0, "npc", 8); err != nil {
		t.Fatal(err)
	}
	named := s.model.Bytes()
	fields = s.Fields("Unit", 0)
	row := fields[fieldIndex(t, fields, "wear7")]
	if len(row.Shut) != 1 || row.Shut[0] != 0 || !strings.Contains(strings.ToLower(row.Choices[1].Label), "npc") {
		t.Fatal("NPC Empty has no disabled reason", row)
	}
	if _, err := s.SetField("Unit", 0, "wear7", 0); err == nil || !bytes.Equal(named, s.model.Bytes()) {
		t.Fatal("NPC Empty changed native identity or records", err)
	}
	if _, err := s.SetField("Unit", 0, "wear1", 0x102); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	if m.Units[0].Flags != 0x2001 || m.Units[0].ClassSubID != 8 {
		t.Fatal("equipment changed NPC identity")
	}
}

func TestEditorWornEquipmentSurvivesMissionSAV(t *testing.T) {
	for _, npc := range []bool{false, true} {
		f, s, _ := nativeEquipmentEditor(t)
		if npc {
			if _, err := s.SetField("Unit", 0, "npc", 8); err != nil {
				t.Fatal(err)
			}
		} else if _, err := s.SetField("Unit", 0, "wear7", 0); err != nil {
			t.Fatal(err)
		}
		if _, err := s.SetField("Unit", 0, "wear1", 0x102); err != nil {
			t.Fatal(err)
		}
		cold := nativeEditorMissionSAV(t, f, s, nativeEquipmentEditor)
		items, _ := cold.live.world.Equipped(0)
		pack, _ := cold.live.world.Carried(0)
		if items[0] != 0x102 || npc && (len(pack) != 1 || pack[0] != 0x101) || !npc && items[6] != 0 {
			t.Fatal("SAV lost authored gear or displacement", npc, items, pack)
		}
	}
}

func TestEditorWornVariantsKeepTheirPrivateRecipe(t *testing.T) {
	_, s, _ := nativeEquipmentEditor(t)
	for _, change := range []ui.InspectionFieldValue{{Key: "wear1", Value: 0x102}, {Key: "wearrecipe1", Value: -1}, {Key: "wearmaterial1", Value: 1}, {Key: "weartier1", Value: 1}} {
		if _, err := s.SetFields("Unit", 0, []ui.InspectionFieldValue{change}); err != nil {
			t.Fatal(change, err)
		}
	}
	m, _ := s.model.Map()
	loot, _ := m.Loot()
	stock, _ := s.model.UnitStock(0)
	last := loot.Records[stock].Elements[len(loot.Records[stock].Elements)-1]
	if last.ItemCode() != 0x1122 || last.TileMarkerIndex != 1 || len(m.Enchantments) != 1 {
		t.Fatal("variant lost recipe", last)
	}
	before := s.model.Bytes()
	for _, index := range []int{-1, 1} {
		if _, err := s.SetField("Unit", index, "wear1", 0x101); err == nil || !bytes.Equal(before, s.model.Bytes()) {
			t.Fatal("invalid placement mutated records", index, err)
		}
	}
	if _, err := s.SetField("Unit", 0, "wearmaterial1", 15); err == nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("missing variant mutated records", err)
	}
}

func TestEditorOrdinaryEmptyMaterializesNativeStockAndRecipe(t *testing.T) {
	_, s, raw := nativeEquipmentEditor(t)
	if _, err := s.SetField("Unit", 0, "wear1", 0); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	stock, err := s.model.UnitStock(0)
	if err != nil || stock < 0 || m.Units[0].Flags != 0x2080 || m.Units[0].ClassSubID != 0xab80 {
		t.Fatal("wrong skip bit or native identity", err)
	}
	loot, _ := m.Loot()
	if len(loot.Records[stock].Elements) != 2 || len(m.Enchantments) != 1 {
		t.Fatal("retained row items or magic were lost")
	}
	for _, item := range loot.Records[stock].Elements {
		if item.Field04 != 1 {
			t.Fatal("worn writer did not use shipped value 1")
		}
	}
	sheets, err := unitSheets(m, s.inspector.table)
	if err != nil || sheets[0].Worn[0].Code != 0 || sheets[0].Worn[1].Code != 0x201 || sheets[0].Worn[6].Code != 0x701 || len(sheets[0].Worn[6].Effects) != 1 {
		t.Fatalf("materialized items differ from row: err=%v elements=%+v", err, loot.Records[stock].Elements)
	}
	after := s.model.Bytes()
	if _, err := s.Undo(); err != nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("Empty Undo was not exact", err)
	}
	if _, err := s.Redo(); err != nil || !bytes.Equal(after, s.model.Bytes()) {
		t.Fatal("Empty Redo was not exact", err)
	}
	path, err := s.SaveAs(filepath.Join(t.TempDir(), "equipment.alm"))
	if err != nil {
		t.Fatal(err, s.Report())
	}
	doc, err := s.inspector.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := doc.Edits.(*mapInspectionEdits)
	if _, err := reopened.Clone("Unit", 0, image.Pt(12, 8)); err != nil {
		t.Fatal(err)
	}
	m, _ = reopened.model.Map()
	if len(m.Units) != 2 || len(m.Enchantments) != 2 {
		t.Fatal("clone shared its item recipe")
	}
	if _, err := reopened.SetField("Unit", 1, "wear1", -1); err != nil {
		t.Fatal(err)
	}
	m, _ = reopened.model.Map()
	sheets, err = unitSheets(m, s.inspector.table)
	if err != nil || sheets[0].Worn[0].Code != 0 || sheets[1].Worn[0].Code != 0x101 {
		t.Fatal("inherit changed another placement", err)
	}
	if _, err := reopened.SetField("Unit", 1, "wear7", 0x101); err == nil {
		t.Fatal("weapon admitted into body armour slot")
	}
}
