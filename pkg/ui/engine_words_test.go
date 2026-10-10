package ui

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"againrom/pkg/render/text"
	"againrom/pkg/words"
)

// A third language is one table: a synthetic xx table reaches the game menu's
// speed rows and the save dialog with no code naming xx, and an id it lacks
// draws the English word.
func TestAThirdLanguageTableReachesTheScreens(t *testing.T) {
	fsys := fstest.MapFS{
		"text/en/strings.toml": {Data: mustEngineTable(t, "en")},
		"text/xx/strings.toml": {Data: []byte("\"speed.label\" = \"XX SPEED: \"\n\"speed.slower\" = \"~XX SLOW\"\n\"save.title\" = \"Xx save\"\n")},
	}
	book, err := words.Load(fsys, "xx")
	if err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, nil, nil)
	w := AuthoredWords()
	w.Engine = book
	a.SetWords(w, nil, nil)
	f := a.flow
	f.menuBack = ScreenTown
	rows := f.gameOptionsRows()
	if !strings.HasPrefix(rows[1].Label, "XX SPEED: ") || rows[2].Label != "~XX SLOW" || rows[3].Label != "~FASTER" {
		t.Fatalf("speed rows %q %q %q", rows[1].Label, rows[2].Label, rows[3].Label)
	}
	if s := f.saveWords(); s.Title != "Xx save" || s.Cancel != "Cancel" {
		t.Fatalf("save dialog title %q cancel %q", s.Title, s.Cancel)
	}
}

// The menu font's selector converts bytes only: a Russian font under English
// words draws English, and Russian words come from the install's language.
func TestTheFontSelectorDoesNotChooseTheLanguage(t *testing.T) {
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), &text.Font{Selector: text.SelectorConverting}, nil)
	if got := a.flow.saveWords().Title; got != "Save game" {
		t.Fatalf("English words under a converting font: %q", got)
	}
	w := AuthoredWords()
	w.Engine = words.For("russian")
	a.SetWords(w, &text.Font{}, nil)
	if got := a.flow.endingWords().Title; got != words.For("russian").Text("ending.title") || got == "Campaign complete" {
		t.Fatalf("Russian install words under an identity font: %q", got)
	}
}

func mustEngineTable(t *testing.T, lang string) []byte {
	t.Helper()
	b, err := fs.ReadFile(words.Engine(), "text/"+lang+"/strings.toml")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
