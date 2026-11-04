package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// Over an installed original, every structure kind of the Add window and
// every object class of the object brush lists in exactly one of the bar's
// three palettes, and each palette holds the census count (EN = RU): 48
// structure kinds in Structures, 33 object classes and 18 structure kinds in
// Objects (the Magic Symbol, kind 27, among them), 42 object classes in
// Trees. The four kinds the class data does not settle list in Structures.
func TestReleaseMapEditorEachPaletteRecordListsInOnePalette(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	s := objectEdits(t, e)
	kinds, objects := map[int64]int{}, map[int]int{}
	want := map[ui.Palette][2]int{ui.PaletteStructures: {47, 0}, ui.PaletteObjects: {18, 33}, ui.PaletteTrees: {0, 42}}
	for _, p := range []ui.Palette{ui.PaletteStructures, ui.PaletteObjects, ui.PaletteTrees} {
		k, o := e.PaletteHolds(p)
		if got := [2]int{len(k), len(o)}; got != want[p] {
			t.Errorf("palette %d holds %d structure kinds and %d object classes, want %v", p, got[0], got[1], want[p])
		}
		for _, v := range k {
			kinds[v]++
		}
		for _, c := range o {
			objects[c]++
		}
	}
	step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindStructure}, "kind")
	for _, c := range step.Choices {
		if kinds[c.Value] != 1 {
			t.Errorf("structure kind %s lists in %d palettes", c.Label, kinds[c.Value])
		}
	}
	for _, r := range s.BrushObjects()[1:] {
		if objects[r.Code] != 1 {
			t.Errorf("object class %d %s lists in %d palettes", r.Code, r.Name, objects[r.Code])
		}
	}
	if len(step.Choices) != 65 || len(s.BrushObjects()) != 76 {
		t.Errorf("the Add window offers %d structure kinds and the brush %d rows, want 65 and 76", len(step.Choices), len(s.BrushObjects()))
	}
	_, unsettled := structurePalettes(x.structures, x.table)
	var ids []int64
	for id := range unsettled {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	structures, _ := e.PaletteHolds(ui.PaletteStructures)
	for _, id := range ids {
		t.Logf("unsettled kind %d: %s", id, unsettled[id])
		if !slices.Contains(structures, id) {
			t.Errorf("unsettled kind %d does not list in Structures", id)
		}
	}
	if objectKinds, _ := e.PaletteHolds(ui.PaletteObjects); !slices.Contains(objectKinds, 8) {
		t.Error("Well 1 (kind 8) does not list in Objects")
	}
	if objectKinds, _ := e.PaletteHolds(ui.PaletteObjects); !slices.Contains(objectKinds, 27) {
		t.Error("the Magic Symbol (kind 27) does not list in Objects")
	}
	if !slices.Equal(ids, []int64{38, 48, 52, 53}) {
		t.Errorf("the unsettled kinds are %v, want 38, 48, 52 and 53", ids)
	}
}
