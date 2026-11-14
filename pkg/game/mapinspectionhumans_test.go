package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// The identity choice's first tab is the creatures block, headed by its
// Humans group, and its second tab is the people block; both names differ by
// language.
func TestEditorIdentityTabAndRowNames(t *testing.T) {
	names := map[string][3]string{
		"en": {"Humans and Creatures", "Named characters", "Humans"},
		"ru": {"Люди и Существа", "Именованные персонажи", "Люди"},
	}
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s := peopleIdentityEditor(t, lang)
			x := s.inspector
			step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
			creatures := x.inspectionUTF8(names[lang][0])
			people := x.inspectionUTF8(names[lang][1])
			if !slices.ContainsFunc(step.Groups, func(g ui.InspectionChoiceGroup) bool { return g.Block == creatures }) {
				t.Errorf("no tab named %q", names[lang][0])
			}
			if !slices.ContainsFunc(step.Groups, func(g ui.InspectionChoiceGroup) bool { return g.Block == people }) {
				t.Errorf("no tab named %q", names[lang][1])
			}
			if len(step.Groups) == 0 || step.Groups[0].Block != creatures || step.Groups[0].Head != x.inspectionUTF8(names[lang][2]) {
				t.Errorf("first row is not %q under %q", names[lang][2], names[lang][0])
			}
		})
	}
}

// The Humans row holds one cell per sprite type of a human that is not a
// hero or a mercenary: the first such row of each class sheet, in row order,
// so Courier (a second row of the peasant's sheet) is not in it.
func TestEditorIdentityHumansRowHoldsOneCellPerSpriteType(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	x := s.inspector
	step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
	creatures := x.inspectionUTF8("Humans and Creatures")
	var humans []int64
	for _, g := range step.Groups {
		if g.Block == creatures && g.Head == x.inspectionUTF8("Humans") {
			humans = g.Values
			break
		}
	}
	want := []int64{personValue(1), personValue(13), personValue(14), personValue(15), personValue(16), personValue(17)}
	if !slices.Equal(humans, want) {
		t.Fatalf("Humans row = %v, want %v", humans, want)
	}
	for _, v := range humans {
		if id, ok := s.identityOf(v); !ok || id.Kind != mapload.UnitPerson {
			t.Errorf("Humans cell %#x is not a person", v)
		}
	}
}

// The second tab keeps every group the people block held: story heroes,
// player characters, mercenaries, and the other people by class, men first.
func TestEditorIdentityNamedTabKeepsEveryGroup(t *testing.T) {
	heads := map[string][]string{
		"en": {"Story heroes", "Player characters", "Mercenaries", "Others: fighters", "Others: mages", "Others: riders", "Others: Odd class"},
		"ru": {"Сюжетные герои", "Персонажи игрока", "Наемники", "Прочие: бойцы", "Прочие: маги", "Прочие: всадники", "Прочие: Odd class"},
	}
	want := [][]int64{{identityValue(identityNamed, 8)}, {personValue(3), personValue(4)},
		{personValue(5), personValue(7), personValue(9), personValue(11), personValue(6), personValue(8), personValue(10), personValue(12)},
		{personValue(2), personValue(1)}, {personValue(14), personValue(13)}, {personValue(16), personValue(15)}, {personValue(17)}}
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			s := peopleIdentityEditor(t, lang)
			x := s.inspector
			step := s.NewRecordStep(ui.InspectionNewRecord{Kind: ui.KindUnit}, "identity")
			people := x.inspectionUTF8(x.word("editor.unit.groups.people"))
			var got []ui.InspectionChoiceGroup
			for _, g := range step.Groups {
				if g.Block == people {
					got = append(got, g)
				}
			}
			if len(got) != len(want) {
				t.Fatalf("named tab has %d groups, want %d", len(got), len(want))
			}
			for i, g := range got {
				if g.Head != x.inspectionUTF8(heads[lang][i]) || !slices.Equal(g.Values, want[i]) {
					t.Errorf("group %d = %q %v, want %q %v", i, g.Head, g.Values, heads[lang][i], want[i])
				}
			}
		})
	}
}

// Pointing at a human cell shows that character's portrait: a named character
// and a Humans row that a named record names both show the record's face; a
// Humans row no record names shows its sprite, not the portrait.
func TestEditorIdentityHumanHoverShowsThePortrait(t *testing.T) {
	s := peopleIdentityEditor(t, "en")
	portrait := s.npcPortraits([]ui.InspectionChoice{{Value: 8}})[8]
	if portrait == nil {
		t.Fatal("record 8 has no portrait in the fixture")
	}
	for _, v := range []int64{identityValue(identityNamed, 8), personValue(2)} {
		if pic := s.NewRecordPreview(ui.KindUnit, "identity", v); pic == nil || !slices.Equal(pic.Pix, portrait.Pix) {
			t.Errorf("hover over %#x does not show record 8's portrait", v)
		}
	}
	if pic := s.NewRecordPreview(ui.KindUnit, "identity", personValue(1)); pic == nil || slices.Equal(pic.Pix, portrait.Pix) {
		t.Errorf("a human no record names shows the portrait or nothing")
	}
}
