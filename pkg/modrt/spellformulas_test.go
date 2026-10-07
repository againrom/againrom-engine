package modrt

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const formulasSettings = "[top]\ntype = \"int\"\ndefault = 120\nmin = 0\nmax = 255\n"

func TestFormulaTablesReachTheResultWithSettings(t *testing.T) {
	src := "[global]\npower = [0, \"@top\"]\nrange_bonus = [2]\n\n[[spell]]\ntarget = \"Heal\"\ndamage_factor = [30, \"@top\"]\n\n[[spell]]\ntarget = \"Haste\"\nmagnitude = [4]\nduration_factor = [1000]\n"
	entries := makeMods(t, map[string]map[string]string{"m": {
		"main.star": spellsMain, "settings.toml": formulasSettings, "data/spells.toml": src,
	}})
	res, err := Load(entries, "rom1-en", []mod.SettingFlag{{Mod: "m", Key: "top", Value: "77"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	g := res.Spells.Global
	if !g.Power.Set || g.Power.Vals[1] != 77 || !g.RangeBonus.Set || g.RangeBonus.Vals[0] != 2 || g.DamageFactor.Set {
		t.Errorf("globals %+v", g)
	}
	heal, haste := res.Spells.Rows[0], res.Spells.Rows[1]
	if !heal.DamageFactor.Set || heal.DamageFactor.Vals[1] != 77 || !haste.Magnitude.Set || !haste.DurationFactor.Set || haste.Power.Set {
		t.Errorf("rows %+v %+v", heal, haste)
	}
	res, err = Load(entries, "rom1-en", nil, Options{})
	if err != nil || res.Spells.Global.Power.Vals[1] != 120 {
		t.Errorf("default setting: %v %+v", err, res.Spells.Global.Power)
	}
}

func TestTwoModsCannotSetTheSameFormula(t *testing.T) {
	a := "[global]\npower = [10]\n\n[[spell]]\ntarget = \"Heal\"\ndamage_factor = [30]\n"
	load := func(b string) error {
		mods := makeMods(t, map[string]map[string]string{
			"m1": {"main.star": spellsMain, "data/spells.toml": a},
			"m2": {"main.star": spellsMain, "data/spells.toml": b},
		})
		_, err := Load(mods, "rom1-en", nil, Options{})
		return err
	}
	if err := load("[global]\npower = [20]\n"); err == nil || !strings.Contains(err.Error(), `data/spells.toml:2: power is already set by mod "m1"`) {
		t.Errorf("global: %v", err)
	}
	if err := load("[[spell]]\ntarget = \"Heal\"\ndamage_factor = [60]\n"); err == nil || !strings.Contains(err.Error(), `data/spells.toml:3: damage_factor of "Heal" is already set by mod "m1"`) {
		t.Errorf("row: %v", err)
	}
	if err := load("[global]\ndamage_factor = [60]\n\n[[spell]]\ntarget = \"Heal\"\npower = [60]\nrange_bonus = [3]\n"); err != nil {
		t.Errorf("another formula of the same row, and a global against a row table: %v", err)
	}
}

func TestAFormulaTableRefusalNamesModFileAndLine(t *testing.T) {
	entries := makeMods(t, map[string]map[string]string{"m": {
		"main.star": spellsMain, "data/spells.toml": "[[spell]]\ntarget = \"Heal\"\nrange_bonus = [256]\n",
	}})
	_, err := Load(entries, "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "data/spells.toml:3: range_bonus entry 0 is 256, outside 0..255") {
		t.Errorf("%v", err)
	}
}
