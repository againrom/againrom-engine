package modrt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

// A folder mod and its pack match a script path by one rule, the mod tree's:
// the case of A-Z and of CP866 Cyrillic letters folds, the case of any other
// letter does not. The host file system's own folding of other letters never
// decides, so a path refused in the pack is refused in the folder.
func TestFolderAndPackedMatchScriptNamesByOneRule(t *testing.T) {
	for _, c := range []struct {
		name, stored, queried string
		loads                 bool
	}{
		{"exact", "Éclair.star", "Éclair.star", true},
		{"ascii case", "map.star", "MAP.star", true},
		{"cyrillic case", "карта.star", "КАРТА.star", true},
		{"outside the rule", "Éclair.star", "éclair.star", false},
		{"outside the rule, other way", "éclair.star", "Éclair.star", false},
		{"outside the rule in a folder name", "Ünter/a.star", "ünter/a.star", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			folder := makeMods(t, map[string]map[string]string{"probe": {
				"main.star": "load(\"" + c.queried + "\", \"cap\")\ndef init(game, settings):\n    game.rules.skill_cap = cap\n",
				c.stored:    "cap = 137\n",
			}})
			want, ferr := Load(folder, "rom1-en", nil, Options{})
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
			if c.loads {
				if ferr != nil || perr != nil {
					t.Fatalf("folder error %v, packed error %v", ferr, perr)
				}
				if got.Rules.SkillCap() != want.Rules.SkillCap() {
					t.Fatalf("skill_cap %d from the folder, %d from the pack", want.Rules.SkillCap(), got.Rules.SkillCap())
				}
				return
			}
			if ferr == nil || perr == nil {
				t.Fatalf("both must refuse: folder error %v, packed error %v", ferr, perr)
			}
			if ferr.Error() != perr.Error() || !strings.Contains(perr.Error(), "no such file in the mod folder") {
				t.Fatalf("folder said %q, pack said %q", ferr, perr)
			}
		})
	}
}
