package modrt

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const screensMain = "def init(game, settings):\n    game.data.add(\"data/screens.toml\")\n"

const screenSrc = "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t\"\nmenu = \"m\"\nplace = \"main\"\ntext = [\"p\"]\n"

const screenStrings = "\"t\" = \"Title\"\n\"m\" = \"Entry\"\n\"p\" = \"Body\"\n"

func loadScreenMod(t *testing.T, base string, files map[string]string) (Result, error) {
	t.Helper()
	files["main.star"] = screensMain
	return Load(makeMods(t, map[string]map[string]string{"m": files}), base, nil, Options{})
}

func TestExampleModDeclaresItsScreenInTheLanguageOfTheBase(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"heavy-armor"})
	if err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string]string{"rom1-en": "About heavy armour", "rom1-ru": "О тяжелых доспехах", "rom1": "About heavy armour"} {
		res, err := Load(entries, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if len(res.Screens.Screens) != 1 {
			t.Fatalf("%s: %+v", base, res.Screens)
		}
		s := res.Screens.Screens[0]
		if s.Mod != "heavy-armor" || s.Key != "about" || s.Kind != mod.ScreenInfo || s.Title != want || s.MenuLabel != want ||
			!s.Main || !s.Game || len(s.Paragraphs) != 3 || s.File != mod.ScreensFile || s.Line != 2 {
			t.Fatalf("%s: %+v", base, s)
		}
	}
}

func TestScreenDataFileRefusalsNameModFileAndLine(t *testing.T) {
	with := func(src string) map[string]string {
		return map[string]string{"data/screens.toml": src, "text/en/strings.toml": screenStrings}
	}
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"unknown kind", with(strings.Replace(screenSrc, `"info"`, `"wheel"`, 1)), `mod "m": data/screens.toml:3: unknown screen kind "wheel"`},
		{"missing text key", with(strings.Replace(screenSrc, `["p"]`, `["zz"]`, 1)), `mod "m": data/screens.toml:7: text key "zz" is not in text/en/strings.toml`},
		{"missing file", map[string]string{}, `mod "m": main.star:2: game.data.add("data/screens.toml"): data/screens.toml: no such file in the mod folder`},
		{"no strings file", map[string]string{"data/screens.toml": screenSrc}, `mod "m": data/screens.toml:4: text key "t" is not in text/en/strings.toml`},
	}
	for _, c := range cases {
		_, err := loadScreenMod(t, "rom1-en", c.files)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestMenuSlotsAreSharedAcrossMods(t *testing.T) {
	// Two mods each fill part of the main menu; the second does not fit.
	mk := func(id string, n int) map[string]string {
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteString(strings.Replace(screenSrc, `key = "a"`, fmt.Sprintf(`key = "s%d"`, i), 1))
		}
		return map[string]string{
			"mod.toml":             "id = \"" + id + "\"\ntitle = \"t\"\nversion = \"1\"\n",
			"main.star":            screensMain,
			"data/screens.toml":    b.String(),
			"text/en/strings.toml": screenStrings,
		}
	}
	_, err := Load(makeMods(t, map[string]map[string]string{"a": mk("a", 2), "b": mk("b", 2)}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), `mod "b": data/screens.toml:`) || !strings.Contains(err.Error(), `does not fit: the main menu has 3 slots`) {
		t.Fatalf("%v", err)
	}
}

func TestAScreenFileIsLoadedOnce(t *testing.T) {
	script := "def init(game, settings):\n    game.data.add(\"data/screens.toml\")\n    game.data.add(\"data/screens.toml\")\n"
	files := map[string]string{"main.star": script, "data/screens.toml": screenSrc, "text/en/strings.toml": screenStrings}
	_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "the file is already loaded") {
		t.Fatalf("%v", err)
	}
}

func TestExampleAbandonModDeclaresItsActionInTheLanguageOfTheBase(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"mission-abandon"})
	if err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string][2]string{
		"rom1-en": {"Abandon mission", "Abandon, return to town"},
		"rom1-ru": {"Отказаться от миссии", "Отказаться и уйти в город"},
		"rom1":    {"Abandon mission", "Abandon, return to town"},
	} {
		res, err := Load(entries, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if len(res.Screens.Screens) != 1 {
			t.Fatalf("%s: %+v", base, res.Screens)
		}
		s := res.Screens.Screens[0]
		if s.Mod != "mission-abandon" || s.Key != "abandon" || s.Kind != mod.ActionAbandon || s.MenuLabel != want[0] ||
			s.Title != want[1] || s.Main || !s.Game || s.File != mod.ScreensFile || s.Line != 2 {
			t.Fatalf("%s: %+v", base, s)
		}
	}
}

func TestTwoModsCannotAddTheSameAction(t *testing.T) {
	action := "[[action]]\nkey = \"a\"\naction = \"abandon\"\nmenu = \"m\"\n"
	files := func() map[string]string {
		return map[string]string{"main.star": screensMain, "data/screens.toml": action, "text/en/strings.toml": "\"m\" = \"Leave\"\n"}
	}
	_, err := Load(makeMods(t, map[string]map[string]string{"a": files(), "b": files()}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), `action "abandon" is already added by mod "a"`) {
		t.Fatalf("%v", err)
	}
}
