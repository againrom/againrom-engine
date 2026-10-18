package words

import (
	"strings"
	"testing"
)

func TestEditorUnitValueMeaningsExistInBothLanguages(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		b := For(lang)
		for _, key := range []string{"owner", "classkey", "tier", "class", "definition", "unitdefinition", "npcdefinition", "group", "gear", "skipgear", "template", "ignored", "result", "explicit"} {
			id := "editor.unit.meaning." + key
			text := b.Text(id)
			if text == id || text == "" || strings.ContainsAny(text, "ёЁ") {
				t.Fatalf("%s: %s=%q", lang, id, text)
			}
		}
	}
}

func TestEditorInheritedMeaningNamesTheInGameValue(t *testing.T) {
	for _, c := range []struct{ lang, want string }{
		{"en", "Inherited; in game: %d"},
		{"ru", "Наследуется; в игре: %d"},
	} {
		if got := For(c.lang).Text("editor.unit.meaning.template"); got != c.want {
			t.Errorf("%s inherited meaning %q, want %q", c.lang, got, c.want)
		}
	}
}
