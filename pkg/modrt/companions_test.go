package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const companionsMain = "def init(game, settings):\n    game.data.add(\"data/companions.toml\")\n"

const joinSrc = "[[join]]\nkey = \"a\"\ncompanion = 22\nchapter = 30\nbuilding = \"tavern\"\ntalk = 22\n"

func TestExampleModHoldsTheTownCompanionUntilTheTalk(t *testing.T) {
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"reniesta-joins-on-talk"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Load(entries, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Companions.Joins) != 1 {
		t.Fatalf("%+v", res.Companions)
	}
	j := res.Companions.Joins[0]
	if j.Mod != "reniesta-joins-on-talk" || j.Key != "plagat_mage" || j.Companion != 22 || j.Chapter != 30 || j.Talk != 22 || j.Building != mod.BuildingTavern {
		t.Fatalf("%+v", j)
	}
	if !res.Items.Empty() || !res.Screens.Empty() {
		t.Fatal("the example mod adds more than its condition")
	}
}

func TestCompanionFileRefusalsNameModFileAndLine(t *testing.T) {
	load := func(files map[string]string) error {
		files["main.star"] = companionsMain
		_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
		return err
	}
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"unknown key":  {map[string]string{"data/companions.toml": joinSrc + "when = 1\n"}, `mod "m": data/companions.toml:7: unknown key "when"`},
		"missing file": {map[string]string{}, `mod "m": main.star:2: game.data.add("data/companions.toml"): data/companions.toml: no such file in the mod folder`},
	} {
		if err := load(c.files); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestTwoModsCannotHoldTheSameCompanion(t *testing.T) {
	second := strings.Replace(joinSrc, `"a"`, `"b"`, 1)
	mods := makeMods(t, map[string]map[string]string{
		"m1": {"main.star": companionsMain, "data/companions.toml": joinSrc},
		"m2": {"main.star": companionsMain, "data/companions.toml": second},
	})
	_, err := Load(mods, "rom1-en", nil, Options{})
	want := `data/companions.toml:1: companion 22 of chapter 30 already has a join condition from mod "m1"`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%v, want %q", err, want)
	}
}
