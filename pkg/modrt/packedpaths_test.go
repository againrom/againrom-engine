package modrt

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// A script path a folder mod loads loads the same from its packed file, or
// neither loads it and both say which path. A name starting with a dot is
// read from neither; a Cyrillic name matches in another case in both, as
// the Windows folder matches it and the packed index folds it.
func TestPackedModLoadsScriptPathsAsItsFolder(t *testing.T) {
	for _, c := range []struct {
		name, stored, queried, refuse string
		windowsFolderLoads            bool
	}{
		{name: "dotted", stored: ".cap.star", queried: ".cap.star", refuse: ".cap.star"},
		{name: "dotted folder", stored: ".hid/cap.star", queried: ".hid/cap.star", refuse: ".hid/cap.star"},
		{name: "cyrillic case", stored: "карта.star", queried: "КАРТА.star", windowsFolderLoads: true},
		{name: "yo case", stored: "ёлка.star", queried: "ЁЛКА.star", windowsFolderLoads: true},
		{name: "ascii case", stored: "map.star", queried: "MAP.star", windowsFolderLoads: true},
	} {
		t.Run(c.name, func(t *testing.T) {
			folder := makeMods(t, map[string]map[string]string{"probe": {
				"main.star": "load(\"" + c.queried + "\", \"cap\")\ndef init(game, settings):\n    game.rules.skill_cap = cap\n",
				c.stored:    "cap = 137\n",
			}})
			want, ferr := Load(folder, "rom1-en", nil, Options{})
			if c.refuse != "" {
				if ferr == nil || !strings.Contains(ferr.Error(), c.refuse) {
					t.Fatalf("the folder must refuse a dotted path naming it: %v", ferr)
				}
			} else if ferr != nil && c.windowsFolderLoads && runtime.GOOS == "windows" {
				t.Fatalf("the folder: %v", ferr)
			}
			m, b, err := mod.Pack(folder[0].Dir)
			if err != nil {
				t.Fatalf("pack: %v", err)
			}
			out := t.TempDir()
			if err := os.WriteFile(filepath.Join(out, mod.PackedName(m.ID)), b, 0o600); err != nil {
				t.Fatal(err)
			}
			packed, err := mod.Resolve(out, []string{"probe"})
			if err != nil {
				t.Fatal(err)
			}
			got, perr := Load(packed, "rom1-en", nil, Options{})
			if (ferr == nil) != (perr == nil) {
				t.Fatalf("folder error %v, packed error %v", ferr, perr)
			}
			if c.refuse != "" {
				if !strings.Contains(perr.Error(), c.refuse) {
					t.Fatalf("the packed mod must refuse naming %s: %v", c.refuse, perr)
				}
				return
			}
			if ferr == nil && got.Rules.SkillCap() != want.Rules.SkillCap() {
				t.Fatalf("skill_cap %d from the folder, %d from the RES", want.Rules.SkillCap(), got.Rules.SkillCap())
			}
		})
	}
}
