package mod

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const goodScreens = `# screens
[[screen]]
key   = "about"
kind  = "info"
title = "t.about"
menu  = "m.about"
place = "both"
text  = ["p.one", "p.two"]

[[screen]]
key   = "perks"
kind  = "list"
title = "t.perks"
menu  = "m.perks"
place = "main"
items = ["p.one", "p.two"]

[[screen]]
key   = "stats"
kind  = "table"
title = "t.stats"
menu  = "m.stats"
place = "game"
rows  = [["l.one", "v.one"], ["l.two", "v.two"]]
`

var screenText = map[string]string{
	"t.about": "About", "m.about": "About us", "t.perks": "Perks", "m.perks": "Perks menu",
	"t.stats": "Stats", "m.stats": "Stats menu", "p.one": "First.", "p.two": "Second.",
	"l.one": "Label one", "v.one": "Value one", "l.two": "Label two", "v.two": "Value two",
}

func TestParseScreensReadsEveryKind(t *testing.T) {
	d, err := ParseScreens("m", ScreensFile, []byte(goodScreens), textOf(screenText), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Screens) != 3 {
		t.Fatalf("%d screens", len(d.Screens))
	}
	about, perks, stats := d.Screens[0], d.Screens[1], d.Screens[2]
	if about.Kind != ScreenInfo || about.Title != "About" || about.MenuLabel != "About us" || !about.Main || !about.Game ||
		len(about.Paragraphs) != 2 || about.Paragraphs[1] != "Second." || about.Line != 2 || about.Mod != "m" {
		t.Fatalf("info %+v", about)
	}
	if perks.Kind != ScreenList || !perks.Main || perks.Game || len(perks.Items) != 2 {
		t.Fatalf("list %+v", perks)
	}
	if stats.Kind != ScreenTable || stats.Main || !stats.Game || len(stats.Rows) != 2 || stats.Rows[1] != [2]string{"Label two", "Value two"} {
		t.Fatalf("table %+v", stats)
	}
	if main, game := d.Count(); main != 2 || game != 2 {
		t.Fatalf("slots used: main %d game %d", main, game)
	}
}

func TestParseScreensRefusalsNameFileAndLine(t *testing.T) {
	base := func(extra string) string {
		return "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t.about\"\nmenu = \"m.about\"\nplace = \"main\"\n" + extra
	}
	cases := []struct {
		name, src string
		line      int
		want      string
	}{
		{"unknown kind", strings.Replace(base("text = [\"p.one\"]\n"), `"info"`, `"carousel"`, 1), 3, `unknown screen kind "carousel"`},
		{"missing text key", base("text = [\"p.zzz\"]\n"), 7, `text key "p.zzz" is not in text/en/strings.toml`},
		{"missing menu key", strings.Replace(base("text = [\"p.one\"]\n"), `m.about`, `m.zzz`, 1), 5, `text key "m.zzz" is not in`},
		{"no body", base(""), 1, `kind "info" needs text`},
		{"wrong body", base("items = [\"p.one\"]\n"), 7, `items belongs to kind "list", not to kind "info"`},
		{"no menu", "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t.about\"\nplace = \"main\"\ntext = [\"p.one\"]\n", 1, "[[screen]] has no menu"},
		{"no place", "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t.about\"\nmenu = \"m.about\"\ntext = [\"p.one\"]\n", 1, "[[screen]] has no place"},
		{"bad place", strings.Replace(base("text = [\"p.one\"]\n"), `"main"`, `"top"`, 1), 6, `place is "top"`},
		{"unknown key", base("text = [\"p.one\"]\ncolour = 1\n"), 8, `unknown key "colour"`},
		{"bad key", strings.Replace(base("text = [\"p.one\"]\n"), `key = "a"`, `key = "A b"`, 1), 2, `key "A b" must be`},
		{"empty list", base("text = []\n"), 7, "text is empty"},
		{"row not a pair", strings.Replace(base("rows = [[\"l.one\"]]\n"), `"info"`, `"table"`, 1), 7, "rows entry 1 must be a pair"},
		{"top level pair", "x = 1\n", 1, "x is outside a [[screen]] table"},
		{"unknown table", "[other]\nx = 1\n", 1, `unknown table "other"`},
		{"long menu label", strings.Replace(base("text = [\"p.one\"]\n"), `m.about`, `m.long`, 1), 5, "has 29 characters; a menu entry has 1 to 28"},
	}
	text := map[string]string{}
	for k, v := range screenText {
		text[k] = v
	}
	text["m.long"] = strings.Repeat("x", 29)
	for _, c := range cases {
		_, err := ParseScreens("m", ScreensFile, []byte(c.src), textOf(text), "en")
		var fe *ItemFileError
		if err == nil || !errors.As(err, &fe) || fe.File != ScreensFile || fe.Line != c.line || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v (want line %d, %q)", c.name, err, c.line, c.want)
		}
	}
}

func TestParseScreensDuplicateKey(t *testing.T) {
	src := "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t.about\"\nmenu = \"m.about\"\nplace = \"main\"\ntext = [\"p.one\"]\n"
	_, err := ParseScreens("m", ScreensFile, []byte(src+src), textOf(screenText), "en")
	if err == nil || !strings.Contains(err.Error(), `data/screens.toml:8: screen key "a" is already used at line 1`) {
		t.Fatalf("%v", err)
	}
}

func TestScreenSlotsAreBounded(t *testing.T) {
	one := func(i int, place string) string {
		return fmt.Sprintf("[[screen]]\nkey = \"s%d\"\nkind = \"info\"\ntitle = \"t.about\"\nmenu = \"m.about\"\nplace = %q\ntext = [\"p.one\"]\n", i, place)
	}
	var main, game strings.Builder
	for i := 0; i <= MaxMainMenuScreens; i++ {
		main.WriteString(one(i, "main"))
	}
	for i := 0; i <= MaxGameMenuScreens; i++ {
		game.WriteString(one(i, "game"))
	}
	if _, err := ParseScreens("m", ScreensFile, []byte(main.String()), textOf(screenText), "en"); err == nil ||
		!strings.Contains(err.Error(), `screen "s3" does not fit: the main menu has 3 slots`) || !strings.Contains(err.Error(), "data/screens.toml:") {
		t.Fatalf("main menu: %v", err)
	}
	if _, err := ParseScreens("m", ScreensFile, []byte(game.String()), textOf(screenText), "en"); err == nil ||
		!strings.Contains(err.Error(), `screen "s2" does not fit: the game menu has 2 slots`) {
		t.Fatalf("game menu: %v", err)
	}
	// Exactly the slots is accepted.
	if _, err := ParseScreens("m", ScreensFile, []byte(strings.TrimSuffix(game.String(), one(2, "game"))), textOf(screenText), "en"); err != nil {
		t.Fatalf("full game menu refused: %v", err)
	}
}

const goodActions = `[[action]]
key     = "abandon"
action  = "abandon"
menu    = "m.about"
confirm = "t.about"

[[action]]
key    = "again"
action = "restart"
menu   = "m.perks"
`

func TestParseActionsReadsEntriesForTheGameMenu(t *testing.T) {
	if _, err := ParseScreens("m", ScreensFile, []byte(goodActions+"\n"+goodScreens), textOf(screenText), "en"); err == nil || !strings.Contains(err.Error(), "the game menu has 2 slots") {
		t.Fatalf("two actions and two game screens need 4 game slots: %v", err)
	}
	d, err := ParseScreens("m", ScreensFile, []byte(goodActions), textOf(screenText), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Screens) != 2 {
		t.Fatalf("%+v", d.Screens)
	}
	a, r := d.Screens[0], d.Screens[1]
	if a.Kind != ActionAbandon || a.Key != "abandon" || a.MenuLabel != "About us" || a.Title != "About" || a.Main || !a.Game || a.Line != 1 {
		t.Fatalf("abandon: %+v", a)
	}
	if r.Kind != ActionRestart || r.MenuLabel != "Perks menu" || r.Title != "Perks menu" || !r.Game || r.Main || r.Line != 7 {
		t.Fatalf("restart: confirm defaults to the entry's label: %+v", r)
	}
	if !IsAction(a.Kind) || IsAction(ScreenInfo) {
		t.Fatal("IsAction")
	}
	if main, game := d.Count(); main != 0 || game != 2 {
		t.Fatalf("slots used: main %d game %d", main, game)
	}
}

func TestParseActionsRefusalsNameFileAndLine(t *testing.T) {
	base := "[[action]]\nkey = \"a\"\naction = \"abandon\"\nmenu = \"m.about\"\n"
	cases := []struct {
		name, src string
		line      int
		want      string
	}{
		{"unknown action", strings.Replace(base, `"abandon"`, `"teleport"`, 1), 3, `unknown action "teleport"`},
		{"place", base + "place = \"main\"\n", 5, "takes no place"},
		{"no menu", "[[action]]\nkey = \"a\"\naction = \"abandon\"\n", 1, "[[action]] has no menu"},
		{"no action", "[[action]]\nkey = \"a\"\nmenu = \"m.about\"\n", 1, "[[action]] has no action"},
		{"unknown key", base + "colour = \"red\"\n", 5, `unknown key "colour" in [[action]]`},
		{"missing text", base + "confirm = \"zz\"\n", 5, `text key "zz" is not in text/en/strings.toml`},
		{"long label", strings.Replace(base, "m.about", "m.long", 1), 4, "has 41 characters; a menu entry has 1 to 40"},
		{"not a string", strings.Replace(base, `"m.about"`, `3`, 1), 4, "menu must be a string"},
		{"bad key", strings.Replace(base, `"a"`, `"A b"`, 1), 2, `key "A b" must be`},
		{"duplicate key", base + base, 6, `screen key "a" is already used at line 1`},
	}
	text := map[string]string{"m.long": strings.Repeat("x", 41)}
	for k, v := range screenText {
		text[k] = v
	}
	for _, c := range cases {
		_, err := ParseScreens("m", ScreensFile, []byte(c.src), textOf(text), "en")
		var fe *ItemFileError
		if err == nil || !errors.As(err, &fe) || fe.File != ScreensFile || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v (want %q)", c.name, err, c.want)
			continue
		}
		if c.name != "duplicate key" && fe.Line != c.line {
			t.Errorf("%s: line %d, want %d: %v", c.name, fe.Line, c.line, err)
		}
	}
}

func TestAnActionIsAddedByOneModOnly(t *testing.T) {
	d, err := ParseScreens("one", ScreensFile, []byte("[[action]]\nkey = \"a\"\naction = \"abandon\"\nmenu = \"m.about\"\n"), textOf(screenText), "en")
	if err != nil {
		t.Fatal(err)
	}
	e, err := ParseScreens("two", ScreensFile, []byte("[[action]]\nkey = \"b\"\naction = \"abandon\"\nmenu = \"m.perks\"\n"), textOf(screenText), "en")
	if err != nil {
		t.Fatal(err)
	}
	d.Screens = append(d.Screens, e.Screens...)
	if err := d.CheckSlots(); err == nil || !strings.Contains(err.Error(), `action "abandon" is already added by mod "one"`) {
		t.Fatalf("%v", err)
	}
}
