package modrt

import (
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

const bodiesMain = "def init(game, settings):\n    game.data.add(\"data/bodies.toml\")\n    game.data.add(\"data/weapon-bodies.toml\")\n"

const bodySrc = "[[body]]\nname = \"hammer2h\"\nheroes = \"a.png\"\nheroes_l = \"b.png\"\nframe = [8, 8]\norigin = [4, 7]\ndirections = 5\nmove = { frames = 1, ticks = 1 }\nattack = { frames = 1, ticks = 1 }\n"

const weaponSrc = "[[weapon]]\nweapon = \"War Hammer\"\nbody = \"hammer2h\"\n"

func TestExampleModsChooseHeroBodies(t *testing.T) {
	for _, base := range []string{"rom1-en", "rom1-ru", "rom2-en", "rom2-ru"} {
		entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"war-hammer-mace", "war-hammer-body"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Load(entries, base, nil, Options{}); err != nil {
			t.Fatalf("%s: %v", base, err)
		}
	}
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"war-hammer-mace"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Load(entries, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Bodies.Weapons) != 1 || len(res.Bodies.Bodies) != 0 {
		t.Fatalf("%+v", res.Bodies)
	}
	if w := res.Bodies.Weapons[0]; w.Mod != "war-hammer-mace" || w.Weapon != "War Hammer" || w.Body != "clubman" || w.File != mod.WeaponBodiesFile || w.Line != 3 {
		t.Fatalf("%+v", w)
	}
	if !res.Items.Empty() || !res.Companions.Empty() {
		t.Fatal("the example mod adds more than its mapping")
	}

	entries, err = mod.Resolve(filepath.Join("testdata", "mods"), []string{"war-hammer-body"})
	if err != nil {
		t.Fatal(err)
	}
	res, err = Load(entries, "rom2-ru", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Bodies.Weapons) != 1 || len(res.Bodies.Bodies) != 1 {
		t.Fatalf("%+v", res.Bodies)
	}
	b := res.Bodies.Bodies[0]
	if b.Name != "hammer2h" || b.Dir != entries[0].Dir || b.Width != 64 || b.Height != 80 || !b.WeaponLast || b.Attack.Frames != 5 {
		t.Fatalf("%+v", b)
	}
}

func TestBodyFileRefusalsNameModFileAndLine(t *testing.T) {
	load := func(files map[string]string) error {
		files["main.star"] = bodiesMain
		_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
		return err
	}
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"body key":    {map[string]string{"data/bodies.toml": bodySrc + "colour = 1\n", "data/weapon-bodies.toml": weaponSrc}, `mod "m": data/bodies.toml:10: unknown key "colour" in [[body]]`},
		"weapon key":  {map[string]string{"data/bodies.toml": bodySrc, "data/weapon-bodies.toml": weaponSrc + "hand = 2\n"}, `mod "m": data/weapon-bodies.toml:4: unknown key "hand" in [[weapon]]`},
		"no mappings": {map[string]string{"data/bodies.toml": bodySrc}, `mod "m": main.star:3: game.data.add("data/weapon-bodies.toml"): data/weapon-bodies.toml: no such file in the mod folder`},
	} {
		if err := load(c.files); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestTwoModsCannotSupplyOneBody(t *testing.T) {
	only := "def init(game, settings):\n    game.data.add(\"data/bodies.toml\")\n"
	mods := makeMods(t, map[string]map[string]string{
		"m1": {"main.star": only, "data/bodies.toml": bodySrc},
		"m2": {"main.star": only, "data/bodies.toml": bodySrc},
	})
	_, err := Load(mods, "rom1-en", nil, Options{})
	want := `data/bodies.toml:1: body "hammer2h" is already supplied by mod "m1"`
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("%v, want %q", err, want)
	}
}
