package ui

import (
	"image"
	"testing"
)

func TestEditorCellSpellPaletteUsesTreeSelection(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	for i, label := range []string{"trap", "unit spell", "casting structure"} {
		e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindEnchantment, Index: i, Label: label, Cell: image.Pt(8+i, 8), Spatial: true, Size: image.Pt(1, 1)})
	}
	e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindEnchantment, Index: 3, Label: "item recipe"})
	e.barAct("cellspells")
	s, ok := e.listSpec()
	if !ok || s.List == nil || len(e.rows()) != 3 {
		t.Fatal("palette omits cell spells or includes item recipes")
	}
	for n, id := range e.rows() {
		e.listChoose(n, true)
		if e.selected != id || e.objectHidden {
			t.Fatal("palette did not choose record", n)
		}
	}
	rows := e.treeRowsNow()
	for n, row := range rows {
		if row.of == "cellspells" && row.member {
			e.treeChoose(n)
			if e.selected != row.record {
				t.Fatal("tree did not use shared selection")
			}
		}
	}
	if row, ok := e.listAdd(); !ok || row.Kind != KindEnchantment {
		t.Fatal("palette lost shared Add")
	}
	e.openAdd(KindEnchantment)
	if key, _, ok := e.AddStep(); !ok || key != "template" {
		t.Fatal("palette Add lost three-kind template step")
	}
}
