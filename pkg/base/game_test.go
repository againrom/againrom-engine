package base

import (
	"strings"
	"testing"
)

func TestRootGameComesFromItsFiles(t *testing.T) {
	cases := []struct {
		name    string
		extra   []string
		lang    string
		id      string
		game    Game
		profile string
	}{
		{"russian second game", []string{"allods2.exe", "Scenario.DLL"}, "russian", ROM2RU, GameROM2, ROM2RU},
		{"english second game", []string{"ALLODS2.EXE", "scenario.dll"}, "english", ROM2EN, GameROM2, ROM2EN},
		{"second game, unknown language", []string{"allods2.exe", "scenario.dll"}, "", ROM2, GameROM2, ROM2},
		{"first game", nil, "russian", ROM1RU, GameROM1, ROM1RU},
		{"one marker is not the second game", []string{"allods2.exe"}, "english", ROM1EN, GameROM1, ROM1EN},
	}
	for _, c := range cases {
		root := fixture(t, "unknown-main", c.extra...)
		m, err := Detect(root, lang(c.lang))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if m.ID() != c.id || m.Exact || m.Profile.GameOf() != c.game {
			t.Errorf("%s: match = %+v, want inexact %s of %s", c.name, m, c.id, c.game)
		}
	}
}

func TestSecondGameProfilesStateTheirLimits(t *testing.T) {
	for _, id := range []string{ROM2EN, ROM2RU, ROM2} {
		p, ok := Find(id)
		if !ok || p.GameOf() != GameROM2 {
			t.Fatalf("Find(%s) = %+v, %v", id, p, ok)
		}
		if p.Mission() != 10 || !p.Limits.NoCharacterGeneration || p.Limits.OriginalSaveRefusal == "" || len(p.Limits.Notes) == 0 {
			t.Errorf("%s: limits %+v", id, p.Limits)
		}
	}
	for _, p := range Profiles {
		if p.GameOf() == GameROM1 && p.Game != "" && p.Game != GameROM1 {
			t.Errorf("%s: game %q", p.ID, p.Game)
		}
	}
}

func TestSecondGameMatchStatement(t *testing.T) {
	loose := Match{Profile: unrecognisedROM2}.String()
	if !strings.Contains(loose, "unrecognised build and language") {
		t.Fatalf("statement = %q", loose)
	}
}
