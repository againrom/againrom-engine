package modrt

import (
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// Install files are read for the base's language, in load order, with the
// order the mods state. A mod changes game state when its script sets a rule
// or adds a data file, or serves a text file; a mod that carries pictures and
// sounds alone, or nothing, does not.
func TestInstallFilesAndTheStateFreeMods(t *testing.T) {
	mods := makeMods(t, map[string]map[string]string{
		"pic": {
			"files/common/graphics/interface/a.bmp": "BM",
			"files/en/main/text/sites.txt":          "East\n",
			"files/ru/main/text/sites.txt":          "Восток\n",
		},
		"pic-script": {
			"main.star":                    "def init(game, settings):\n    pass\n",
			"files/en/main/text/sites.txt": "Later\n",
			"mod.toml":                     "id = \"pic-script\"\ntitle = \"p\"\nversion = \"1\"\nload-after = [\"pic\"]\n",
		},
		"art": {
			"files/common/graphics/interface/b.bmp": "BM",
			"files/EN/sfx/ui/click.wav":             "RIFF",
		},
		"rule": {"main.star": "def init(game, settings):\n    game.rules.skill_cap = 120\n"},
		"data": {
			"main.star":       "def init(game, settings):\n    game.data.add(\"data/towns.toml\")\n",
			"data/towns.toml": "[[town]]\nid = 1\nname = \"Home\"\n",
		},
	})
	for _, c := range []struct{ base, sites string }{{"rom1-en", "East\n"}, {"rom1-ru", "Восток\n"}} {
		res, err := Load(mods, c.base, nil, Options{})
		if err != nil {
			t.Fatalf("%s: %v", c.base, err)
		}
		free := map[string]bool{}
		for _, m := range res.Set.Mods {
			free[m.ID] = m.StateFree
		}
		// pic-script serves a text on English installs only.
		if free["pic"] || free["pic-script"] != (c.base == "rom1-ru") || !free["art"] || free["rule"] || free["data"] {
			t.Errorf("%s: state-free %v", c.base, free)
		}
		var got []string
		var sites string
		for _, f := range res.Files.Files {
			if f.Mod != "art" {
				got = append(got, f.Mod+":"+f.File)
			}
			if f.Mod == "pic" && f.Kind == mod.FileText {
				sites = string(f.Bytes)
			}
		}
		want := "pic:files/common/graphics/interface/a.bmp pic:files/" + c.base[5:] + "/main/text/sites.txt"
		if c.base == "rom1-en" {
			want += " pic-script:files/en/main/text/sites.txt"
		}
		if strings.Join(got, " ") != want || sites != c.sites {
			t.Errorf("%s: files %v, want %s", c.base, got, want)
		}
		if !res.Files.Order.After("pic-script", "pic") {
			t.Errorf("%s: the order is lost: %v", c.base, res.Files.Order)
		}
		if marked, want := res.Set.Marked(), map[string]int{"rom1-en": 4, "rom1-ru": 3}[c.base]; len(marked.Mods) != want {
			t.Errorf("%s: marked %+v", c.base, marked.Mods)
		}
	}
	bad := makeMods(t, map[string]map[string]string{"bad": {"files/xx/main/a.txt": "x"}})
	if _, err := Load(bad, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), `mod "bad": files/xx:`) {
		t.Fatalf("an unknown language folder: %v", err)
	}
}
