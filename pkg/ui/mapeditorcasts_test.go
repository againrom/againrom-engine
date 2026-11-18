package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestEditorNativeCellSpellOpensPalette(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	updateEditorKeys(t, e, ebiten.KeyF3)

	if e.filter != editorCellSpellsFilter || !e.listShown || e.prompt.open() {
		t.Fatal("F3 did not open the Traps and spells palette")
	}
}

func TestEditorNativeSpellCloneWhilePlacing(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.doc.Records = append(e.doc.Records, InspectionRecord{
		Kind: KindEnchantment, Index: 0, Spatial: true, Cell: image.Pt(8, 8), Size: image.Pt(1, 1),
	})
	updateEditorKeys(t, e, ebiten.KeyF3)
	e.Select(len(e.doc.Records)-1, false)
	e.placeChoose(0)
	if !e.addPicking() {
		t.Fatal("spell choice did not arm placement")
	}
	updateEditorKeys(t, e, ebiten.KeyC)
	if e.pick != pickClone {
		t.Fatalf("C while placing armed %q", e.pick)
	}
}
