package ui

import (
	"image"
	"slices"
	"testing"
)

func TestSpellPaletteUsesHostTemplateChoices(t *testing.T) {
	e := newPlacementLayoutEditor(t, "en")
	h := e.doc.Edits.(*placementLayoutHost)
	h.templates = InspectionNewStep{
		Value: -70,
		Choices: []InspectionChoice{
			{Value: -70, Label: "Host unit"}, {Value: 0, Label: "Existing unit spell"},
			{Value: -30, Label: "Host cell"},
		},
		Templates: map[int64]InspectionSpellTemplate{
			-70: {Constructor: -70, Target: KindUnit}, 0: {Constructor: -70, Target: KindUnit},
			-30: {Constructor: -30},
		},
	}
	h.power = InspectionNewStep{Value: 10, Label: "Host power", Bounds: &InspectionStepBounds{Min: 9, Max: 12}}
	paletteGrid(t, e, "cellspells")
	for _, n := range []int{0, 2, 0} {
		want := h.templates.Choices[n]
		spec := mustListSpec(t, e)
		if spec.Rows[0].Value != want.Label || e.castTemplate() != want.Value {
			t.Fatalf("Where = %q / %d, want %+v", spec.Rows[0].Value, e.castTemplate(), want)
		}
		next, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: 0, K: 1})
		if !ok {
			t.Fatal("the Where row has no next button")
		}
		chromeClick(e, center(next))
	}
	spec := mustListSpec(t, e)
	power := slices.IndexFunc(spec.Rows, func(r popupRow) bool { return r.Label == h.power.Label })
	for _, step := range []struct {
		button int
		want   string
	}{{1, "12"}, {0, "9"}} {
		r, ok := e.FindControl(ControlQuery{Role: ControlListButton, N: power, K: step.button})
		if power < 0 || !ok {
			t.Fatal("the host's power row has no bound controls")
		}
		chromeClick(e, center(r))
		if got := mustListSpec(t, e).Rows[power].Value; got != step.want {
			t.Fatalf("power = %q, want host bound %q", got, step.want)
		}
	}
}

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
	if _, ok := e.listAdd(); ok {
		t.Fatal("spells shows Add")
	}
}
