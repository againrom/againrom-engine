package modrt

import (
	"testing"

	"againrom/pkg/mod"
)

func TestAbandonRuleUsesTheResolvedModSetting(t *testing.T) {
	entries := makeMods(t, map[string]map[string]string{"leave": {
		"settings.toml": "[enabled]\ntype = \"bool\"\ndefault = true\n",
		"main.star":     "def init(game, settings):\n    game.rules.abandon = 1 if settings.enabled else 0\n",
	}})
	for _, base := range []string{"rom1-en", "rom1-ru", "rom2-en", "rom2-ru"} {
		for _, value := range []string{"true", "false"} {
			r, err := Load(entries, base, []mod.SettingFlag{{Mod: "leave", Key: "enabled", Value: value}}, Options{})
			if err != nil || r.Rules.Abandon() != (value == "true") {
				t.Fatalf("%s %s: abandon=%t %v", base, value, r.Rules.Abandon(), err)
			}
		}
	}
}
