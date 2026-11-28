package sim

import "testing"

func structureOwnerWorld(t *testing.T, script *Script, structures []Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(7, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, MapUnitID: 91, X: 5, Y: 5, HP: 20, MaxHP: 20, TokenSize: 1}},
		script, Relations{}, nil, nil, nil, GhostTemplate{}, structures)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// A mission loader builds a first World with no program and builds the played
// World again from that World's structure list under the program's game. The
// first build must not clear an owner the second game keeps.
func TestStructureOwnerSurvivesTheMissionRebuildInTheSecondGame(t *testing.T) {
	list := []Structure{{ID: 1, Kind: 1, Col: 2, Row: 2, Width: 1, Height: 1, Owner: 3}}
	first := structureOwnerWorld(t, nil, list)
	if got := first.Structures()[0].Owner; got != 3 {
		t.Fatalf("the World with no program holds owner %d, want 3", got)
	}
	second, err := NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := structureOwnerWorld(t, second, first.Structures()).Structures()[0].Owner; got != 3 {
		t.Fatalf("the second game's rebuilt World holds owner %d, want 3", got)
	}
	quiet, err := NewScript(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := structureOwnerWorld(t, quiet, first.Structures()).Structures()[0].Owner; got != 0 {
		t.Fatalf("the first game's rebuilt World holds owner %d, want none", got)
	}
}
