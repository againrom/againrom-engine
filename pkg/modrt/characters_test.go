package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const charactersMain = "def init(game, settings):\n    game.data.add(\"data/characters.toml\")\n"

const characterRowSrc = "[[character]]\ntarget = \"NPC06\"\nname = \"c.girl\"\nstrip = [\"armour\"]\n"

func loadCharacterMod(t *testing.T, base string, files map[string]string) (Result, error) {
	t.Helper()
	files["main.star"] = charactersMain
	return Load(makeMods(t, map[string]map[string]string{"m": files}), base, nil, Options{})
}

func TestExampleCharacterModNamesTheGirlInTheLanguageOfTheBase(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"archer-girl"})
	if err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string]string{"rom1-en": "Archer girl", "rom1-ru": "Лучница", "rom1": "Archer girl"} {
		res, err := Load(entries, base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", base, err)
		}
		if len(res.Characters.Rows) != 1 || !res.Items.Empty() {
			t.Fatalf("%s: %+v", base, res)
		}
		r := res.Characters.Rows[0]
		if r.Name != want || r.Mod != "archer-girl" || r.Target != "NPC06" || r.Kind != "A_PeasantGuard" ||
			len(r.Strip) != 1 || r.Strip[0] != mod.StripArmour {
			t.Fatalf("%s: %+v", base, r)
		}
	}
}

func TestCharacterDataFileRefusalsNameModFileAndLine(t *testing.T) {
	en := map[string]string{"text/en/strings.toml": "\"c.girl\" = \"G\"\n"}
	with := func(src string, extra map[string]string) map[string]string {
		files := map[string]string{"data/characters.toml": src}
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
		{"unknown strip group", with(strings.Replace(characterRowSrc, `"armour"`, `"boots"`, 1), nil), `mod "m": data/characters.toml:4: unknown strip group "boots"`},
		{"missing text key", with(strings.Replace(characterRowSrc, `c.girl`, `c.none`, 1), nil), `mod "m": data/characters.toml:3: text key "c.none" is not in text/en/strings.toml`},
		{"missing file", map[string]string{}, `mod "m": main.star:2: game.data.add("data/characters.toml"): data/characters.toml: no such file in the mod folder`},
	}
	for _, c := range cases {
		_, err := loadCharacterMod(t, "rom1-en", c.files)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestCharacterFileIsLoadedOnceAndInLoadOrder(t *testing.T) {
	files := map[string]string{"data/characters.toml": characterRowSrc, "text/en/strings.toml": "\"c.girl\" = \"G\"\n",
		"main.star": charactersMain + "    game.data.add(\"data/characters.toml\")\n"}
	_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "the file is already loaded") {
		t.Fatalf("a second load: %v", err)
	}
	mk := func(id, target string) map[string]string {
		return map[string]string{
			"mod.toml":             "id = \"" + id + "\"\ntitle = \"t\"\nversion = \"1\"\n",
			"main.star":            charactersMain,
			"data/characters.toml": strings.Replace(characterRowSrc, "NPC06", target, 1),
			"text/en/strings.toml": "\"c.girl\" = \"G\"\n",
		}
	}
	second := mk("z", "NPC07")
	second["mod.toml"] = "id = \"z\"\ntitle = \"t\"\nversion = \"1\"\nload-after = [\"a\"]\n"
	res, err := Load(makeMods(t, map[string]map[string]string{"z": second, "a": mk("a", "NPC06")}), "rom1-en", nil, Options{})
	if err != nil || len(res.Characters.Rows) != 2 || res.Characters.Rows[0].Target != "NPC06" || res.Characters.Rows[1].Target != "NPC07" {
		t.Fatalf("%v %+v", err, res.Characters.Rows)
	}
}
