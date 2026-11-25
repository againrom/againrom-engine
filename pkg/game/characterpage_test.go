package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

// The character list page's data is the game's row: the engine's title and
// Ok in the install language, and Delete in the first game alone.
func TestCharacterPageIsTheGamesRow(t *testing.T) {
	for _, tc := range []struct {
		id, lang  string
		del       bool
		title, ok string
	}{
		{base.ROM1EN, "english", true, "Characters", "Accept"},
		{base.ROM1RU, "russian", true, "Персонажи", "Принять"},
		{base.ROM2EN, "english", false, "Characters", "Accept"},
		{base.ROM2RU, "russian", false, "Персонажи", "Принять"},
	} {
		profile, ok := base.Find(tc.id)
		if !ok {
			t.Fatalf("missing profile %s", tc.id)
		}
		f := &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Base: base.Match{Profile: profile}}}}
		f.Words.Engine = words.For(tc.lang)
		want := ui.CharacterPage{Title: encodeEngineWord(tc.title, 0), OK: encodeEngineWord(tc.ok, 0), Delete: tc.del}
		if got := f.characterPage(); got != want {
			t.Errorf("%s: character page %+v, want %+v", tc.id, got, want)
		}
	}
}
