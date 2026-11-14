package game

import (
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// Pointing at a Humans cell shows the figure the game shows for that human,
// composed from his equipment; the cell keeps his sprite. Every person
// identity has a sprite for its cell, the riders included.
func TestReleaseEditorIdentityHumansHoverShowsTheComposedFigure(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	if err := e.Open(newMapSource(32, 32)); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	choices, _ := x.identityChoices(s.mission())
	groups := x.identityGroups(choices)
	humans := x.inspectionUTF8(x.word("editor.unit.groups.humans"))
	i := slices.IndexFunc(groups, func(g ui.InspectionChoiceGroup) bool { return g.Head == humans })
	if i < 0 || len(groups[i].Values) == 0 {
		t.Fatal("no Humans group")
	}
	for _, v := range groups[i].Values {
		figure := s.NewRecordPreview(ui.KindUnit, "identity", v)
		sprite := s.NewRecordPreview(ui.KindUnit, "identity-grid", v)
		if figure == nil || sprite == nil {
			t.Fatalf("Humans cell %#x: figure %v, sprite %v", v, figure != nil, sprite != nil)
		}
		if figure.Bounds().Dx() <= sprite.Bounds().Dx() || figure.Bounds().Dy() <= sprite.Bounds().Dy() {
			t.Errorf("Humans cell %#x: hover picture %v is no larger than its sprite %v", v, figure.Bounds(), sprite.Bounds())
		}
	}
	for v, id := range x.identities {
		if id.Kind != mapload.UnitCreature && s.NewRecordPreview(ui.KindUnit, "identity-grid", v) == nil {
			t.Errorf("person identity %#x has no picture for its cell", v)
		}
	}
}
