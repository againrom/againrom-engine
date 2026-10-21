package modrt

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mod"
)

// packedFixture is a mod with a setting, a rule set from it, two data files,
// texts in both languages and install files.
var packedFixture = map[string]string{
	"mod.toml":                              "id = \"fix\"\ntitle = \"Fix\"\nversion = \"1.0\"\n",
	"settings.toml":                         "[cap]\nlabel = \"Cap\"\ntype = \"int\"\ndefault = 110\nmin = 100\nmax = 150\n",
	"main.star":                             "def init(game, settings):\n    game.rules.skill_cap = settings.cap\n    game.data.add(\"data/items.toml\")\n    game.data.add(\"data/towns.toml\")\n",
	"data/items.toml":                       itemRowSrc,
	"data/towns.toml":                       "[[town]]\nid = 1\nname = \"Home\"\n",
	"text/en/strings.toml":                  "\"item.a\" = \"A\"\n",
	"text/ru/strings.toml":                  "\"item.a\" = \"Б\"\n",
	"files/common/graphics/interface/a.bmp": "BM",
	"files/en/main/text/sites.txt":          "East\n",
	"files/ru/main/text/sites.txt":          "Восток\n",
}

// withoutDirs is r with every field that names where the mod lies cleared:
// the rest is what the mod gave.
func withoutDirs(r Result) Result {
	r.Ordered = nil
	for i := range r.Items.Rows {
		r.Items.Rows[i].Dir = ""
	}
	for i := range r.Towns.Records {
		r.Towns.Records[i].Dir = ""
		if d := r.Towns.Records[i].Description; d != nil {
			d.Dir = ""
		}
	}
	for i := range r.Bodies.Bodies {
		r.Bodies.Bodies[i].Dir = ""
	}
	return r
}

// A mod run from its packed RES gives what it gives from its folder on both
// languages: rules from a setting, the set and its digest, items, towns and
// install files. Without the setting flag the rule differs, so the
// comparison can fail.
func TestPackedModRunsAsItsFolder(t *testing.T) {
	folder := makeMods(t, map[string]map[string]string{"fix": packedFixture})
	m, b, err := mod.Pack(folder[0].Dir)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, mod.PackedName(m.ID)), b, 0o644); err != nil {
		t.Fatal(err)
	}
	packed, err := mod.Resolve(dir, []string{"fix"})
	if err != nil {
		t.Fatal(err)
	}
	given := []mod.SettingFlag{{Mod: "fix", Key: "cap", Value: "130"}}
	for _, base := range []string{"rom1-en", "rom1-ru"} {
		want, err := Load(folder, base, given, Options{})
		if err != nil {
			t.Fatal(err)
		}
		got, err := Load(packed, base, given, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Ordered[0].Dir != packed[0].Dir || want.Ordered[0].Dir != folder[0].Dir {
			t.Fatalf("%s: dirs %s %s", base, got.Ordered[0].Dir, want.Ordered[0].Dir)
		}
		if got.Rules.SkillCap() != 130 || len(got.Items.Rows) != 1 || len(got.Towns.Records) != 1 || len(got.Files.Files) != 2 {
			t.Fatalf("%s: the packed mod gave too little: %+v", base, got)
		}
		if !reflect.DeepEqual(withoutDirs(want), withoutDirs(got)) || want.Set.Digest() != got.Set.Digest() {
			t.Fatalf("%s: the packed mod runs otherwise than its folder:\nfolder %+v\npacked %+v", base, want, got)
		}
		plain, err := Load(packed, base, nil, Options{})
		if err != nil || plain.Rules.SkillCap() == got.Rules.SkillCap() || plain.Set.Digest() == got.Set.Digest() {
			t.Fatalf("%s: the setting changes nothing: %v", base, err)
		}
	}
}
