package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const routeWalkFixture = "pkg/game/routewitness.go"

// TestCheckRouteWalkNamesEachShape plants each per-game shape the route
// witness's stage walks took; the controls pass.
func TestCheckRouteWalkNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a first-game stage method", "pkg/game/other.go",
			"package game\nfunc (w *routeWitness) firstShop() {}\n", "routeWitness.firstShop is a first-game stage walk"},
		{"a first-game method on a value receiver", "pkg/game/routewitness_first.go",
			"package game\nfunc (w routeWitness) firstMenu() {}\n", "routeWitness.firstMenu is a first-game stage walk"},
		{"a second parameter", routeWalkFixture,
			"package game\nfunc newRouteWitnessOn(second bool) {}\n", "a value named second"},
		{"a branch on a second field", "pkg/game/routewitness_release_test.go",
			"package game\nfunc f(w *routeWitness) { if w.second { return } }\n", "a value named second"},
		{"a comparison with a game", routeWalkFixture,
			"package game\nfunc f(g base.Game) bool { return g == base.GameROM2 }\n", "a comparison with GameROM2"},
		{"a comparison of the game call", "pkg/game/routewitness_release_test.go",
			"package game\nfunc f(a *Archives, g int) bool { return a.Game() != g }\n", "a comparison with Game()"},
		{"a switch on the game", routeWalkFixture,
			"package game\nfunc f(a *Archives) { switch a.Game() { } }\n", "a switch on the game"},
		{"a profile read of the first game's data", routeWalkFixture,
			"package game\nfunc f(f *FrontEnd) bool { return !f.Base().Profile.HasFirstData() }\n", "HasFirstData tells the games apart"},
		{"a nil test of the chapter field", routeWalkFixture,
			"package game\nfunc f(w *routeWitness) bool { return w.chapter == nil }\n", "a nil test of the witness's chapter field"},
		{"a nil test of the town field", routeWalkFixture,
			"package game\nfunc f(w *routeWitness) bool { return nil != w.town }\n", "a nil test of the witness's town field"},
		{"a row with a function literal", routeWalkFixture,
			"package game\nvar rows = []routeStage{{\"shop\", \"town/square\", func(w *routeWitness) { w.shop(true) }}}\n", "a stage row holds a function literal"},
		{"a row with a method", routeWalkFixture,
			"package game\nvar rows = []routeStage{{\"menu\", \"menu\", (*routeWitness).menu}}\n", "a stage row holds the method menu"},
		{"rows naming kinds with data", routeWalkFixture,
			"package game\nvar rows = []routeStage{{\"menu\", \"menu\", routeKindMenu, routeData{play: \"map\"}}}\n", ""},
		{"a table the campaign service answers", routeWalkFixture,
			"package game\nfunc f(f *FrontEnd) []routeStage { return f.campaign().routeStages() }\n", ""},
		{"a first-game method of another type", routeWalkFixture,
			"package game\nfunc (t *townScreen) firstRoom() {}\n", ""},
		{"a game branch outside the witness", "pkg/game/town.go",
			"package game\nfunc f(second bool, g base.Game) bool { return second && g == base.GameROM2 }\n", ""},
		{"a nil test of another field", routeWalkFixture,
			"package game\nfunc f(w *routeWitness) bool { return w.f.Town != nil }\n", ""},
		{"a method named first alone", routeWalkFixture,
			"package game\nfunc (w *routeWitness) first() {}\nfunc (w *routeWitness) firsthand() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{tc.file: tc.src}
			if tc.file != routeWalkFixture {
				files[routeWalkFixture] = "package game\n"
			}
			var got []string
			for _, v := range CheckRouteWalk(files) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckRouteWalk = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckRouteWalk = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckRouteWalk(map[string]string{"pkg/game/town.go": "package game\n"}); len(vs) != 1 || !strings.Contains(vs[0].Reason, "nothing to read") {
		t.Fatalf("CheckRouteWalk without a route witness file = %v, want one violation", vs)
	}
}

// TestRouteWitnessStagesHaveOneWalk scans the live pkg/game.
func TestRouteWitnessStagesHaveOneWalk(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(root, "pkg", "game", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		files["pkg/game/"+filepath.Base(p)] = string(b)
	}
	for _, v := range CheckRouteWalk(files) {
		t.Errorf("%s", v)
	}
}
