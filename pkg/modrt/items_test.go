package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const itemsMain = "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n"

const itemRowSrc = "[[item]]\nkey = \"a\"\nname = \"item.a\"\nslot = \"body\"\nstand-in = \"Soft Mail\"\nprice = 5\n"

func loadItemMod(t *testing.T, base string, files map[string]string) (Result, error) {
	t.Helper()
	files["main.star"] = itemsMain
	return Load(makeMods(t, map[string]map[string]string{"m": files}), base, nil, Options{})
}

func TestExampleItemModAddsItemsInTheLanguageOfTheBase(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"heavy-armor"})
	if err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string]string{"rom1-en": "Gambeson", "rom1-ru": "Поддоспешник", "rom1": "Gambeson"} {
		res, err := Load(entries, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if len(res.Items.Rows) != 3 || len(res.Items.Changes) != 1 {
			t.Fatalf("%s: %+v", base, res.Items)
		}
		r := res.Items.Rows[0]
		if r.Name != want || r.Mod != "heavy-armor" || r.Key != "gambeson" || r.SlotNo != 7 || r.Defence != 6 || r.Sprite == "" || r.Dir != entries[0].Dir || !r.Stock {
			t.Fatalf("%s: %+v", base, r)
		}
		for i, layer := range []struct {
			key, placement, anchor string
			anchorNo               int
		}{{"gambeson", "under", "body", 7}, {"cloak", "over", "body", 7}, {"boots", "over", "legs", 11}} {
			got := res.Items.Rows[i]
			if got.Key != layer.key || got.Layer != layer.placement || got.Anchor != layer.anchor || got.AnchorNo != layer.anchorNo {
				t.Fatalf("%s: row %d is %+v, want layer %+v", base, i, got, layer)
			}
		}
		if cloak := res.Items.Rows[1]; cloak.Suit != mod.SuitFighter || cloak.Slot != "shield" || cloak.Figure != "Cloak" {
			t.Fatalf("%s: the cloak is %+v", base, cloak)
		}
	}
}

func TestLanguageFor(t *testing.T) {
	for base, want := range map[string]string{"rom1-ru": "ru", "rom1-en": "en", "rom1": "en", "rom2": "en", "": "en"} {
		if got := LanguageFor(base); got != want {
			t.Errorf("%q: %q, want %q", base, got, want)
		}
	}
}

func TestItemDataFileRefusalsNameModFileAndLine(t *testing.T) {
	en := map[string]string{"text/en/strings.toml": "\"item.a\" = \"A\"\n"}
	with := func(src string, extra map[string]string) map[string]string {
		files := map[string]string{"data/items.toml": src}
		for k, v := range en {
			files[k] = v
		}
		for k, v := range extra {
			files[k] = v
		}
		return files
	}
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"unknown slot", with(strings.Replace(itemRowSrc, `"body"`, `"belt"`, 1), nil), `mod "m": data/items.toml:4: unknown slot "belt"`},
		{"missing text key", with(strings.Replace(itemRowSrc, `item.a`, `item.zzz`, 1), nil), `mod "m": data/items.toml:3: text key "item.zzz" is not in text/en/strings.toml`},
		{"unknown key", with(itemRowSrc+"colour = 1\n", nil), `mod "m": data/items.toml:7: unknown key "colour"`},
		{"missing file", map[string]string{}, `mod "m": main.star:2: game.data.add("data/items.toml"): data/items.toml: no such file in the mod folder`},
		{"strings syntax", with(itemRowSrc, map[string]string{"text/en/strings.toml": "\"item.a\" = 3\n"}), `mod "m": text/en/strings.toml:1: "item.a" must be a string`},
	}
	for _, c := range cases {
		_, err := loadItemMod(t, "rom1-en", c.files)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestDataLoadRefusesOtherFilesAndSecondLoads(t *testing.T) {
	check := func(script, want string) {
		t.Helper()
		files := map[string]string{"main.star": script, "data/items.toml": itemRowSrc, "data/skills.toml": "x = 1\n", "text/en/strings.toml": "\"item.a\" = \"A\"\n"}
		_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), `mod "m": main.star:`) {
			t.Errorf("%q: %v", script, err)
		}
	}
	check("def init(game, settings):\n    game.data.add(\"data/skills.toml\")\n", "no data file of that name is loaded (loadable: data/bodies.toml, data/characters.toml, data/companions.toml, data/items.toml, data/screens.toml, data/spells.toml, data/weapon-bodies.toml)")
	check("def init(game, settings):\n    game.data.add(\"data/items.toml\")\n    game.data.add(\"data/items.toml\")\n", "the file is already loaded")
	check("def init(game, settings):\n    game.data.add(\"../x.toml\")\n", "not a path inside the mod folder")
	check("def init(game, settings):\n    game.data.add()\n", "missing argument")
	check("def init(game, settings):\n    game.data.nothing()\n", "nothing")
}

func TestItemRussianTextFallsBackToEnglishAndTheEnglishBaseIgnoresRussian(t *testing.T) {
	files := map[string]string{"data/items.toml": itemRowSrc, "text/en/strings.toml": "\"item.a\" = \"A\"\n"}
	res, err := loadItemMod(t, "rom1-ru", files)
	if err != nil || res.Items.Rows[0].Name != "A" {
		t.Fatalf("%v %+v", err, res.Items)
	}
	files = map[string]string{"data/items.toml": itemRowSrc, "text/ru/strings.toml": "\"item.a\" = \"Б\"\n"}
	if _, err := loadItemMod(t, "rom1-en", files); err == nil || !strings.Contains(err.Error(), `text key "item.a" is not in text/en/strings.toml`) {
		t.Fatalf("an English game read Russian text: %v", err)
	}
}

func TestItemDataIsLoadedInLoadOrder(t *testing.T) {
	mk := func(id, key string) map[string]string {
		return map[string]string{
			"mod.toml":             "id = \"" + id + "\"\ntitle = \"t\"\nversion = \"1\"\nload-after = [\"b\"]\n",
			"main.star":            itemsMain,
			"data/items.toml":      strings.Replace(itemRowSrc, `key = "a"`, `key = "`+key+`"`, 1),
			"text/en/strings.toml": "\"item.a\" = \"A\"\n",
		}
	}
	b := mk("b", "first")
	b["mod.toml"] = "id = \"b\"\ntitle = \"t\"\nversion = \"1\"\n"
	res, err := Load(makeMods(t, map[string]map[string]string{"c": mk("c", "second"), "b": b}), "rom1-en", nil, Options{})
	if err != nil || len(res.Items.Rows) != 2 || res.Items.Rows[0].Key != "first" || res.Items.Rows[1].Key != "second" {
		t.Fatalf("%v %+v", err, res.Items.Rows)
	}
}

func TestStringsFileIsReadThroughTheBoundedReader(t *testing.T) {
	big := "\"item.a\" = \"" + strings.Repeat("x", mod.MaxFileBytes) + "\"\n"
	_, err := loadItemMod(t, "rom1-en", map[string]string{
		"data/items.toml":      itemRowSrc,
		"text/en/strings.toml": big,
	})
	want := `mod "m": text/en/strings.toml: text/en/strings.toml is larger than`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("an oversize strings file: %v", err)
	}
}
