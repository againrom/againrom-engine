package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// A mod's number-named maps are listed as replacements, and Start names one.
func TestMissionListShowsTheMissionsAModReplacesByFileName(t *testing.T) {
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		t.Run(tc.lang, func(t *testing.T) {
			r := rigWithLanguage(t, tc.language)
			writeCampaignInstall(t, r.root, 7, 12)
			dir := filepath.Join(r.dir, "mods", "byname")
			for name, body := range map[string]string{"mod.toml": goodMod("byname"), "7.alm": "map", "maps/12.alm": "map", "maps/notes.alm": "map"} {
				path := filepath.Join(dir, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			a := r.app()
			do(t, a, action{aToggleMod, 0})
			var numbers []string
			for _, row := range missionTexts(a) {
				n, _, _ := strings.Cut(row, ".")
				numbers = append(numbers, n)
			}
			if want := []string{"7", "12"}; !reflect.DeepEqual(numbers, want) {
				t.Fatalf("mission rows %q, want missions %q; the window shows %q", missionTexts(a), want, texts(a))
			}
			if containsText(a, caption(tc.lang, cNoMissions)) || !hasAction(a, aStartMission) {
				t.Fatalf("the window shows %q", texts(a))
			}
			want := []string{"7. " + caption(tc.lang, cMissionByName, "7.alm"), "12. " + caption(tc.lang, cMissionByName, "maps/12.alm")}
			if got := missionTexts(a); !reflect.DeepEqual(got, want) {
				t.Fatalf("mission rows %q, want %q", got, want)
			}
			do(t, a, action{aSelectMission, 1})
			do(t, a, action{kind: aStartMission})
			if len(r.fakes.started) != 1 || !strings.Contains(strings.Join(r.fakes.started[0], " "), " -mission 12") {
				t.Fatalf("Start started %q", r.fakes.started)
			}
		})
	}
}
