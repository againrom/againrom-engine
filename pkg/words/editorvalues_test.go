package words

import (
	"strings"
	"testing"
)

func TestEditorUnitValueMeaningsExistInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		b := For(lang)
		for _, key := range []string{"owner", "classkey", "tier", "class", "definition", "unitdefinition", "npcdefinition", "group", "gear", "skipgear", "ignored", "result"} {
			id := "editor.unit.meaning." + key
			text := b.Text(id)
			if text == id || text == "" || strings.ContainsAny(text, "ёЁ") {
				t.Fatalf("%s: %s=%q", lang, id, text)
			}
		}
	}
}

func TestEditorUnitBlockWordsExistInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		b := For(lang)
		for _, id := range []string{"editor.unit.group.vitals", "editor.unit.group.attributes", "editor.unit.group.defence",
			"editor.unit.group.skills", "editor.unit.group.protections", "editor.unit.group.equipment", "editor.unit.clear",
			"editor.unit.zero.allowed", "editor.unit.zero.refused", "editor.unit.zero.error"} {
			text := b.Text(id)
			if text == id || text == "" || strings.ContainsAny(text, "ёЁ") {
				t.Fatalf("%s: %s=%q", lang, id, text)
			}
		}
		for _, id := range []string{"editor.unit.meaning.template", "editor.unit.meaning.explicit"} {
			if b.Text(id) != id {
				t.Fatalf("%s still carries the retired %s", lang, id)
			}
		}
	}
}
