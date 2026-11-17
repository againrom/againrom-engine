package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

func TestEditorNativeChoiceWordsUseFontBytesInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, _ := nativeInventoryEditor(t)
			x := s.inspector
			x.archives.Base.Profile.Language = lang
			x.font = &text.Font{Selector: 0}
			check := func(key string, choice int, word string) {
				t.Helper()
				fields := s.Fields("Unit", 0)
				f := fields[fieldIndex(t, fields, key)]
				if choice >= len(f.Choices) {
					t.Fatal("missing choice", key, choice)
				}
				want, _ := textinput.EncodeText(words.For(lang).Text(word), 0, textinput.Substitute)
				if f.Choices[choice].Label != want {
					t.Fatalf("%s choice %d is not font-encoded: %q != %q", key, choice, f.Choices[choice].Label, want)
				}
			}
			check("identity", 0, "editor.unit.identity.persons")
			check("face", 0, "editor.unit.look.inherit")
			check("sex", 0, "editor.unit.look.male")
			check("sex", 1, "editor.unit.look.female")
			check("gear", 0, "editor.unit.meaning.gear")
			check("wear1", 0, "editor.equipment.inherit")
			check("wear1", 1, "editor.equipment.empty")
			check("carryadd", 0, "editor.inventory.choose")
			if _, err := s.SetField("Unit", 0, "carryadd", 0x102); err != nil {
				t.Fatal(err)
			}
			check("carry1", 0, "editor.inventory.remove")
			check("carry1", 1, "editor.inventory.duplicate")
			check("carryrecipe1", 0, "editor.equipment.newrecipe")
			if _, err := s.SetField("Unit", 0, "identity", namedIdentity(8)); err != nil {
				t.Fatal(err)
			}
			check("wear1", 1, "editor.equipment.npcempty")
			// A sack item's recipe list opens with the plain item, in the
			// map maker's language.
			plain := map[string]string{"en": "No recipe (plain item)", "ru": "Без рецепта (простой предмет)"}[lang]
			want, _ := textinput.EncodeText(plain, 0, textinput.Substitute)
			if c := recipeChoices(s, fieldChoiceAsk{key: "recipe"}); len(c) == 0 || c[0].Label != want {
				t.Fatalf("recipe choices %v, want %q first", c, plain)
			}
		})
	}
}

func TestEditorRecipeChoicesKeepCatalogNamesAndUnknownSpellNumbers(t *testing.T) {
	for _, tc := range []struct {
		lang     string
		selector int
		name     string
		unknown  string
	}{{"en", 0, "Catalog spell name", "Spell 29"}, {"ru", 1, "Имя заклинания из каталога", "Заклинание 29"}} {
		t.Run(tc.lang, func(t *testing.T) {
			s, _, recipe := sackRecipeMap(t, tc.lang, false)
			x := s.inspector
			x.font = &text.Font{Selector: tc.selector}
			x.recordOptions = map[string][]ui.InspectionChoice{"spell": {{Value: 5, Label: "5: " + x.inspectionUTF8(tc.name)}}}
			for _, spell := range []int64{5, 29} {
				if err := s.model.SetField(mapedit.KindEnchantment, recipe, "spell", spell); err != nil {
					t.Fatal(err)
				}
				want := tc.name
				if spell == 29 {
					want = tc.unknown
				}
				found := 0
				for _, c := range s.RecordChoices(ui.KindSack, groundSack(t, s), "recipe1") {
					if c.Value != int64(recipe+1) && c.Value != -int64(recipe+1)-1 {
						continue
					}
					if !strings.Contains(c.Label, x.inspectionUTF8(want)) || strings.Contains(c.Label, "?") || strings.Contains(c.Label, fmt.Sprintf("spell%d", spell)) {
						t.Errorf("spell %d recipe choice %d: %q, want intact %q", spell, c.Value, c.Label, want)
					}
					found++
				}
				if found != 2 {
					t.Errorf("spell %d: checked %d choices, want shared and private copy", spell, found)
				}
			}
		})
	}
}
