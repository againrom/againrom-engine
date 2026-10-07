package modrt

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const spellsMain = "def init(game, settings):\n    game.data.add(\"data/spells.toml\")\n"

const spellsSettings = "[spell_mana]\ntype = \"int\"\ndefault = 6\nmin = 0\nmax = 100\n\n[mana_num]\ntype = \"int\"\ndefault = 1\nmin = 1\nmax = 10\n\n[mana_den]\ntype = \"int\"\ndefault = 2\nmin = 1\nmax = 10\n"

const spellsSrc = "[global]\nmana_mul = [\"@mana_num\", \"@mana_den\"]\n\n[[spell]]\ntarget = \"Fire_Ball\"\nmana = \"@spell_mana\"\nradius = 4\n"

func TestSpellsFileReachesTheResultWithSettings(t *testing.T) {
	entries := makeMods(t, map[string]map[string]string{"m": {
		"main.star": spellsMain, "settings.toml": spellsSettings, "data/spells.toml": spellsSrc,
	}})
	res, err := Load(entries, "rom1-en", []mod.SettingFlag{{Mod: "m", Key: "spell_mana", Value: "9"}, {Mod: "m", Key: "mana_num", Value: "3"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	d := res.Spells
	if len(d.Rows) != 1 || d.Rows[0].Mod != "m" || d.Rows[0].Target != "Fire_Ball" || d.Rows[0].Mana.Val != 9 || d.Rows[0].Radius.Val != 4 {
		t.Fatalf("rows %+v", d.Rows)
	}
	if g := d.Global.Mana; !g.Set || g.Num != 3 || g.Den != 2 {
		t.Errorf("mana_mul %+v", g)
	}
	if !res.Items.Empty() || !res.Characters.Empty() {
		t.Error("the spells file added more than spells")
	}
	res, err = Load(entries, "rom1-en", nil, Options{})
	if err != nil || res.Spells.Rows[0].Mana.Val != 6 || res.Spells.Global.Mana.Num != 1 {
		t.Errorf("defaults: %v %+v", err, res.Spells)
	}
}

func TestSpellsFileRefusalsNameModFileAndLine(t *testing.T) {
	load := func(files map[string]string) error {
		files["main.star"] = spellsMain
		_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
		return err
	}
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"unknown key":      {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nluck = 1\n"}, `data/spells.toml:3: unknown key "luck"`},
		"out of range":     {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nmana = 99999\n"}, "data/spells.toml:3: mana is 99999, outside 0..32767"},
		"no such setting":  {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nmana = \"@speed\"\n"}, `data/spells.toml:3: mana names the setting "speed"`},
		"setting is bool":  {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nmana = \"@flag\"\n", "settings.toml": "[flag]\ntype = \"bool\"\ndefault = true\n"}, `data/spells.toml:3: mana names the setting "flag"`},
		"duplicate field":  {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nmana = 1\n\n[[spell]]\ntarget = \"Heal\"\nmana = 2\n"}, `data/spells.toml:7: mana of "Heal" is already set by mod "m"`},
		"duplicate filter": {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nheal_hostile = true\n\n[[spell]]\ntarget = \"Heal\"\nheal_hostile = false\n"}, `data/spells.toml:7: heal_hostile of "Heal" is already set by mod "m"`},
		"duplicate rays":   {map[string]string{"data/spells.toml": "[[spell]]\ntarget = \"Prismatic_Spray\"\nrays = 9\n\n[[spell]]\ntarget = \"Prismatic_Spray\"\nrays = 10\n"}, `data/spells.toml:7: rays of "Prismatic_Spray" is already set by mod "m"`},
		"missing file":     {map[string]string{}, `main.star:2: game.data.add("data/spells.toml"): data/spells.toml: no such file in the mod folder`},
	} {
		if err := load(c.files); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestTwoModsCannotSetTheSameSpellField(t *testing.T) {
	a := "[global]\ndamage_mul = [2, 1]\n\n[[spell]]\ntarget = \"Fire Ball\"\nmana = 4\nrange = 3\n"
	b := "[[spell]]\ntarget = \"Fire_Ball\"\nradius = 3\nrange = 9\n"
	mods := makeMods(t, map[string]map[string]string{
		"m1": {"main.star": spellsMain, "data/spells.toml": a},
		"m2": {"main.star": spellsMain, "data/spells.toml": b},
	})
	_, err := Load(mods, "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), `data/spells.toml:4: range of "Fire_Ball" is already set by mod "m1"`) {
		t.Errorf("%v", err)
	}
	b = "[global]\nmana_mul = [1, 2]\n\n[[spell]]\ntarget = \"Fire_Ball\"\nradius = 3\n"
	mods = makeMods(t, map[string]map[string]string{
		"m1": {"main.star": spellsMain, "data/spells.toml": a},
		"m2": {"main.star": spellsMain, "data/spells.toml": b},
	})
	res, err := Load(mods, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	g := res.Spells.Global
	if !g.Damage.Set || !g.Mana.Set || len(res.Spells.Rows) != 2 {
		t.Errorf("%+v", res.Spells)
	}
	b = "[global]\ndamage_mul = [3, 1]\n"
	mods = makeMods(t, map[string]map[string]string{
		"m1": {"main.star": spellsMain, "data/spells.toml": a},
		"m2": {"main.star": spellsMain, "data/spells.toml": b},
	})
	if _, err = Load(mods, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), `data/spells.toml:2: damage_mul is already set by mod "m1"`) {
		t.Errorf("%v", err)
	}
}

func TestNoSpellsFileLeavesTheResultEmpty(t *testing.T) {
	res, err := run(t, "def init(game, settings):\n    pass\n")
	if err != nil || !res.Spells.Empty() {
		t.Errorf("%v %+v", err, res.Spells)
	}
}
