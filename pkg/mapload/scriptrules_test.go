package mapload

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// The late-reference band is the profile's data: the second game's profile
// gives 8000..9999, and a profile with another band moves the band with it.
func TestLateReferenceReadsProfileBand(t *testing.T) {
	second := scriptprofile.Second
	for _, c := range []struct {
		name string
		p    scriptprofile.Profile
		typ  uint32
		v    uint32
		want bool
	}{
		{"second 8000 group", second, typeGroup, 8000, true},
		{"second 9999 unit", second, typeUnit, 9999, true},
		{"second 7999 player", second, typePlayer, 7999, false},
		{"second 10000 group", second, typeGroup, 10000, false},
		{"second 8000 int", second, typeInt, 8000, false},
		{"first game 8000 group", scriptprofile.First, typeGroup, 8000, false},
		{"extended 8000 group", scriptprofile.Extended, typeGroup, 8000, false},
		{"moved band 150 group", moved(second, 100, 200), typeGroup, 150, true},
		{"moved band 8000 group", moved(second, 100, 200), typeGroup, 8000, false},
	} {
		if got := lateReference(c.p, c.typ, c.v); got != c.want {
			t.Errorf("%s: lateReference = %v, want %v", c.name, got, c.want)
		}
	}
}

func moved(p scriptprofile.Profile, low, high uint32) scriptprofile.Profile {
	p.LateLow, p.LateHigh = low, high
	return p
}

// Each game's compile route reaches its own empty-script rule: the first
// game's empty script is no script, the second game's is a script with no
// records. Both routes go through compileScriptFrom.
func TestCompileRoutesApplyProfileEmptyRule(t *testing.T) {
	var empty alm.Script
	first, _, err := CompileScriptFrom(empty, ScriptRefs{})
	if err != nil || first != nil {
		t.Fatalf("first game empty script: got %v, %v; want nil, nil", first, err)
	}
	second, _, err := CompileROM2ScriptFrom(empty, ScriptRefs{})
	if err != nil || second == nil {
		t.Fatalf("second game empty script: got %v, %v; want a script, nil", second, err)
	}
	if second.Dialect() != sim.ScriptROM2 {
		t.Fatalf("second game script dialect = %d, want the second game's", second.Dialect())
	}
}
