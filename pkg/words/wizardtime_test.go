package words

import "testing"

// The two effect-time actions only rewrite the remaining time of effects
// already present, and the item words say so.
func TestWizardEffectTimeActionsSayTheyRewriteTheRemainingTime(t *testing.T) {
	want := map[string]map[string]string{
		"en": {
			"editor.wizard.a.celltime": "Set the remaining time of effects on a cell",
			"editor.wizard.a.unittime": "Set the remaining time of effects on a unit",
		},
		"ru": {
			"editor.wizard.a.celltime": "Задать оставшееся время эффектов на клетке",
			"editor.wizard.a.unittime": "Задать оставшееся время эффектов на отряде",
		},
	}
	for lang, ids := range want {
		for id, text := range ids {
			if got := For(lang).Text(id); got != text {
				t.Errorf("%s %s = %q, want %q", lang, id, got, text)
			}
		}
	}
}
