package game

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

func TestEditorSackRecipeChoicesUseInstalledLanguage(t *testing.T) {
	x := installedInspector(t)
	var d *ui.InspectionDocument
	key := ""
	for _, entry := range x.Maps {
		if !entry.FromArchive {
			continue
		}
		candidate, err := x.Open("scenario/" + entry.Source)
		if err != nil {
			t.Fatal(err)
		}
		m, err := candidate.Edits.(*mapInspectionEdits).model.Map()
		if err != nil {
			t.Fatal(err)
		}
		loot, err := m.Loot()
		if err != nil {
			t.Fatal(err)
		}
		for _, sack := range loot.Records {
			if sack.Owner != 0 {
				continue
			}
			for _, item := range sack.Elements {
				if item.TileMarkerIndex > 0 {
					d, key = candidate, "scenario/"+entry.Source
				}
			}
		}
		if d != nil {
			break
		}
	}
	if d == nil {
		t.Fatal("no installed scenario has a ground sack item linked to a recipe")
	}
	s := d.Edits.(*mapInspectionEdits)
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	names := LoadTextTable(x.archives.Containers, SpellNamesTextPath, InstallTextCode(x.archives.Containers, x.archives.Game().Edition()))
	ru := words.Language(x.archives.Base.Profile.Language) == "ru"
	linked, lines := 0, 0
	for _, r := range d.Records {
		if r.Kind != ui.KindSack {
			continue
		}
		for _, f := range s.Fields(r.Kind, r.Index) {
			if !strings.HasPrefix(f.Key, "recipe") || strings.Contains(f.Key, ".") {
				continue
			}
			if f.Value > 0 {
				linked++
			}
			for _, c := range f.Choices {
				if c.Value == 0 || c.Value == -1 {
					continue
				}
				var label strings.Builder
				latin := false
				for _, b := range []byte(c.Label) {
					r, ok := textinput.DecodeByte(b, x.font.Selector)
					if !ok {
						t.Fatalf("recipe choice has invalid font byte %#x", b)
					}
					label.WriteRune(r)
					latin = latin || unicode.In(r, unicode.Latin)
				}
				text := label.String()
				if ru && latin {
					t.Errorf("Russian sack recipe contains Latin letters: %q", text)
				}
				if strings.Contains(text, "?") {
					t.Errorf("recipe choice lost text during font encoding: %q", text)
				}
				if !ru && !strings.Contains(text, "Recipe ") {
					t.Errorf("English sack recipe is not English: %q", text)
				}
				if ru && !strings.Contains(text, "Рецепт ") {
					t.Errorf("Russian sack recipe is not Russian: %q", text)
				}
				recipe := c.Value
				if recipe < 0 {
					recipe = -recipe - 1
				}
				spell := int(uint16(m.Enchantments[recipe-1].SpellRaw))
				if spell > 0 {
					name, ok := names.At(spell - 1)
					if !ok || !strings.Contains(c.Label, name) {
						t.Errorf("recipe choice does not show installed spell %d's name: %q", spell, text)
					}
				}
				if lines == 0 {
					t.Logf("%s sack %d %s: %s", key, r.Index, f.Key, text)
				}
				lines++
			}
		}
	}
	if linked == 0 || lines == 0 {
		t.Fatalf("no installed sack recipe exercised: linked %d, lines %d", linked, lines)
	}
	saved, err := s.encodeForSave()
	if err != nil || !bytes.Equal(saved, d.SourceBytes) {
		t.Fatalf("reading recipe choices changed the untouched map's save bytes: %v", err)
	}
	t.Logf("linked sack items %d, recipe choice lines %d; untouched save byte-equal", linked, lines)
}
