package archtest

import (
	"os"
	"strings"
	"testing"
)

// The live tree asks the World's game only at its named sites.
func TestTheWorldAsksItsGameOnce(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"pkg/sim/gameprofile.go", "pkg/sim/carriers.go", "pkg/game/savactionwire.go"} {
		if _, ok := files[f]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", f)
		}
	}
	found, err := FindWorldGame(files)
	if err != nil {
		t.Fatal(err)
	}
	chooser := 0
	for _, f := range found {
		if f.Shape == worldGameChooser && f.Func == "newGameProfile" {
			chooser++
		}
	}
	if chooser == 0 {
		t.Fatal("no newGameProfile chooser found; the scan guards nothing")
	}
	for _, v := range CheckWorldGame(found, WorldGameAllowed, WorldGameStateCalls) {
		t.Error(v)
	}
}

// worldGamePlanted scans one planted pkg/sim file.
func worldGamePlanted(t *testing.T, file, source string) []string {
	t.Helper()
	found, err := FindWorldGame(map[string]string{file: "package sim\n" + source})
	if err != nil {
		t.Fatal(err)
	}
	return CheckWorldGame(found, WorldGameAllowed, nil)
}

// Each shape planted outside its named sites is refused once; the same
// shape at a named site, in a test file or outside pkg/sim and pkg/game is
// not.
func TestTheWorldGameScanRefusesEachPlantedShape(t *testing.T) {
	for _, tc := range []struct {
		shape, source string
	}{
		{worldGameSecondGame, "func f(w *World) bool { return w.SecondGame() }\n"},
		{worldGameROM2Nil, "func f(w *World) bool { return w.rom2 != nil }\n"},
		{worldGameROM2Nil, "func f(w *World) bool { return nil == w.rom2 }\n"},
		{worldGameSavedGroups, "func f(w *World) bool { return w.savedGroups == nil }\n"},
		{worldGameDialect, "func f(w *World) bool { return w.script.Dialect() == ScriptROM2 }\n"},
		{worldGameDialect, "func f(w *World) int {\n\tswitch w.script.Dialect() {\n\tcase ScriptROM1:\n\t\treturn 1\n\t}\n\treturn 0\n}\n"},
		{worldGameChooser, "func f(w *World) { w.game = newGameProfile(ScriptROM2) }\n"},
		{worldGameChooser, "func f() groupRouteProfile { return secondGameGroupRoute }\n"},
		{worldGameChooser, "func f() *World { return &World{game: gameProfile{}} }\n"},
	} {
		got := worldGamePlanted(t, "pkg/sim/planted.go", tc.source)
		if len(got) == 0 || !strings.Contains(got[0], tc.shape) {
			t.Errorf("planted %s: violations %v", tc.shape, got)
		}
		for _, file := range []string{"pkg/sim/planted_test.go", "pkg/render/planted.go"} {
			if got := worldGamePlanted(t, file, tc.source); len(got) != 0 {
				t.Errorf("%s in %s: violations %v, want none", tc.shape, file, got)
			}
		}
	}
	if got := worldGamePlanted(t, "pkg/sim/rom2script.go", "func ROM2ScenarioState(w *World) bool { return w.rom2 == nil }\n"); len(got) != 0 {
		t.Errorf("a named state site is refused: %v", got)
	}
	if got := worldGamePlanted(t, "pkg/sim/rom2script.go", "func other(w *World) bool { return w.rom2 == nil }\n"); len(got) != 1 {
		t.Errorf("another function of a named file is admitted: %v", got)
	}
	if got := worldGamePlanted(t, "pkg/sim/planted.go", "// w.SecondGame() in a comment\nvar s = \"w.rom2 == nil\"\n"); len(got) != 0 {
		t.Errorf("a comment or a string is refused: %v", got)
	}
	if got := worldGamePlanted(t, "pkg/game/planted.go", "func f(s *Snapshot) { s.game = 1 }\n"); len(got) != 0 {
		t.Errorf("a pkg/game field named game is refused: %v", got)
	}
}

// A spell row's format marker and its writer are refused outside their named
// sites: a row read, a key, a whole-table read through one row, and a use of
// AssignSecondGameArms in any production file, called or taken as a value.
func TestTheWorldGameScanRefusesEachSpellMarkerShape(t *testing.T) {
	planted := func(file, source string) []string {
		t.Helper()
		pkg := file[strings.Index(file, "/")+1 : strings.LastIndex(file, "/")]
		found, err := FindWorldGame(map[string]string{file: "package " + pkg + "\n" + source})
		if err != nil {
			t.Fatal(err)
		}
		return CheckWorldGame(found, WorldGameAllowed, nil)
	}
	for _, tc := range []struct {
		file, shape, source string
	}{
		{"pkg/sim/planted.go", worldGameSpellRow, "func f(r SpellRule) bool { return r.Second }\n"},
		{"pkg/sim/planted.go", worldGameSpellRow, "func f() SpellRule { return SpellRule{Second: true} }\n"},
		{"pkg/sim/planted.go", worldGameSpellTable, "func f(w *World) bool { return len(w.spells) > 0 && w.spells[0].Second }\n"},
		{"pkg/sim/planted.go", worldGameSpellArms, "func f(r []SpellRule) { AssignSecondGameArms(r) }\n"},
		{"pkg/sim/planted.go", worldGameSpellArms, "func f(w *World) { w.rowArms = AssignSecondGameArms }\n"},
		{"pkg/game/planted.go", worldGameSpellArms, "func f(r []sim.SpellRule) { sim.AssignSecondGameArms(r) }\n"},
		{"pkg/mapload/planted.go", worldGameSpellArms, "func f(r []sim.SpellRule) { sim.AssignSecondGameArms(r) }\n"},
		{"cmd/tool/planted.go", worldGameSpellArms, "var f = sim.AssignSecondGameArms\n"},
	} {
		got := planted(tc.file, tc.source)
		if len(got) != 1 || !strings.Contains(got[0], tc.shape) {
			t.Errorf("planted %s in %s: violations %v", tc.shape, tc.file, got)
		}
		test := strings.TrimSuffix(tc.file, ".go") + "_test.go"
		if got := planted(test, tc.source); len(got) != 0 {
			t.Errorf("%s in %s: violations %v, want none", tc.shape, test, got)
		}
	}
	if got := planted("pkg/sim/spellarms.go", "func armSpellID(w *World) bool { return w.spells[0].Second }\n"); len(got) != 0 {
		t.Errorf("the named whole-table site is refused: %v", got)
	}
	if got := planted("pkg/sim/spellarms.go", "func poisonPointMagnitude(w *World) bool { return w.spells[0].Second }\n"); len(got) != 1 {
		t.Errorf("a whole-table read at a row site is admitted: %v", got)
	}
	if got := planted("pkg/sim/spellarms.go", "func armSpellID(r SpellRule) bool { return r.Second }\n"); len(got) != 1 {
		t.Errorf("a row read at the whole-table site is admitted: %v", got)
	}
	if got := planted("pkg/game/planted.go", "func f(s *session) bool { return s.Second != nil || time.Second > 0 }\n"); len(got) != 0 {
		t.Errorf("a pkg/game field named Second is refused: %v", got)
	}
}

// The groupDialectNative state calls hold their count per file: a new
// call, a new file and a fallen count each fail.
func TestTheWorldGameScanRatchetsTheGroupRegistryTest(t *testing.T) {
	found, err := FindWorldGame(map[string]string{
		"pkg/sim/a.go": "package sim\nfunc f(w *World) bool { return w.groupDialectNative() && w.groupDialectNative() }\n",
		"pkg/sim/b.go": "package sim\nfunc g(w *World) bool { return w.groupDialectNative() }\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		state map[string]int
		want  string
	}{
		{map[string]int{"pkg/sim/a.go": 2, "pkg/sim/b.go": 1}, ""},
		{map[string]int{"pkg/sim/a.go": 1, "pkg/sim/b.go": 1}, "above its count"},
		{map[string]int{"pkg/sim/a.go": 2}, "pkg/sim/b.go holds 1 groupDialectNative calls, above its count of 0"},
		{map[string]int{"pkg/sim/a.go": 3, "pkg/sim/b.go": 1}, "below its count"},
	} {
		got := CheckWorldGame(found, nil, tc.state)
		if tc.want == "" && len(got) != 0 || tc.want != "" && (len(got) != 1 || !strings.Contains(got[0], tc.want)) {
			t.Errorf("state %v: violations %v, want %q", tc.state, got, tc.want)
		}
	}
}
