package game

import (
	"bytes"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

func TestEditorInlineRecipePrivatelyCopiesASharedNativeLink(t *testing.T) {
	_, s, _ := nativeInventoryEditor(t)
	for _, v := range []ui.InspectionFieldValue{{Key: "wear1", Value: 0x102}, {Key: "wearrecipe1", Value: -1}, {Key: "wearrecipe1.addeffect", Value: 17}, {Key: "wearrecipe1.low1", Value: 5}, {Key: "carryadd", Value: 0x102}} {
		if _, err := s.SetField("Unit", 0, v.Key, v.Value); err != nil {
			t.Fatal(err)
		}
	}
	stock, _ := s.model.UnitStock(0)
	if err := s.model.SetField(mapedit.KindLoot, stock, "recipe2", 1); err != nil {
		t.Fatal(err)
	}
	before := s.model.Bytes()
	if _, err := s.SetField("Unit", 0, "carryrecipe2.low1", 9); err != nil {
		t.Fatal(err)
	}
	m, _ := s.model.Map()
	loot, _ := m.Loot()
	links := loot.Records[stock].Elements
	if links[0].TileMarkerIndex == links[1].TileMarkerIndex || len(m.Enchantments) != 2 || m.Enchantments[0].Elements[0].Low != 5 || m.Enchantments[1].Elements[0].Low != 9 {
		t.Fatal("inline recipe changed the other item", m.Enchantments)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("private inline recipe Undo", err)
	}
	if _, err := s.SetField("Unit", 0, "carryrecipe2.notafield", 3); err == nil || !bytes.Equal(before, s.model.Bytes()) {
		t.Fatal("invalid recipe field wrote bytes", err)
	}
}

func TestEditorSequentialNativeDraftNoOpAndFailureAreAtomic(t *testing.T) {
	_, s, raw := nativeInventoryEditor(t)
	values := []ui.InspectionFieldValue{{Key: "group", Value: 4, Step: 1}, {Key: "group", Value: 0, Step: 2}}
	if next, err := s.SetFields("Unit", 0, values); err != nil || next != nil || !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("roundtrip draft created history", err)
	}
	values = []ui.InspectionFieldValue{{Key: "group", Value: 4, Step: 2}, {Key: "group", Value: 0, Step: 1}}
	if _, err := s.SetFields("Unit", 0, values); err == nil || !bytes.Equal(raw, s.model.Bytes()) {
		t.Fatal("out of order draft wrote bytes", err)
	}
	values = []ui.InspectionFieldValue{{Key: "carryadd", Value: 0x102, Step: 1}, {Key: "body", Value: 256, Step: 2}}
	if _, err := s.SetFields("Unit", 0, values); err == nil || !bytes.Equal(raw, s.model.Bytes()) || s.CanUndo() {
		t.Fatal("failed complete draft wrote bytes", err)
	}
}
