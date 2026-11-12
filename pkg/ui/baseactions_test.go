package ui

import (
	"testing"

	"againrom/pkg/words"
)

func TestBaseActionRegistrationAndModReplacement(t *testing.T) {
	for _, lang := range []string{"english", "russian"} {
		a := newTestApp(t, appRows(1), okLoader(t))
		a.flow.words.Engine = words.For(lang)
		data := ModScreen{Key: "abandon", Kind: ModActionAbandon, Game: true,
			MenuLabel: words.For(lang).Text("mission.abandon"), Title: words.For(lang).Text("mission.abandon.confirm")}
		if err := a.SetModScreens([]ModScreen{data}); err != nil {
			t.Fatal(err)
		}
		base := a.flow.modUI.screens
		if len(base) != 1 || base[0].Kind != ModActionAbandon || base[0].MenuLabel != words.For(lang).Text("mission.abandon") {
			t.Fatalf("%s base action: %+v", lang, base)
		}
		if err := a.SetModScreens(nil); err != nil || len(a.flow.modUI.screens) != 0 {
			t.Fatalf("%s off: %+v %v", lang, a.flow.modUI.screens, err)
		}
		for _, withBase := range []bool{true, false} {
			replacement := modActionScreens()[:1]
			if withBase {
				replacement = append(replacement, data)
			}
			if err := a.SetModScreens(replacement); err != nil {
				t.Fatal(err)
			}
			got := a.flow.modUI.screens
			if len(got) != 1 || got[0].Mod != "m" || got[0].Title != "Leave it" {
				t.Fatalf("%s replacement: %+v", lang, got)
			}
		}
	}
}
