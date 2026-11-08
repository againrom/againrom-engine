package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestSecondGameAliveCheckMarksABodyPlacedAtZeroHealth(t *testing.T) {
	m := &alm.Map{Units: []alm.Unit{
		{UnitID: 7, CurrentHP: 0, HasCurrentHP: true},
		{UnitID: 8, CurrentHP: 20, HasCurrentHP: true},
		{UnitID: 9, CurrentHP: -1},
	}}
	units := mapload.ScriptUnits(m, nil)
	refs := mapload.ScriptRefs{Units: units, ZeroHealthBodies: mapload.ScriptZeroHealthBodies(m, units)}
	src := alm.Script{Conditions: []alm.ScriptNode{
		node("placed at zero", 5, 1, par(4, 7)),
		node("placed wounded", 5, 2, par(4, 8)),
		node("placed whole", 5, 3, par(4, 9)),
	}}
	want := []int32{1, 0, 0}

	second, _, err := mapload.CompileROM2ScriptFrom(src, refs)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range second.Checks() {
		if c.Args[sim.ScriptAliveHeldBody] != want[i] {
			t.Errorf("second game check %d flag = %d, want %d", i, c.Args[sim.ScriptAliveHeldBody], want[i])
		}
	}

	first, _, err := mapload.CompileScriptFrom(src, refs)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range first.Checks() {
		if c.Args[sim.ScriptAliveHeldBody] != 0 {
			t.Errorf("first game check %d carries the flag", i)
		}
	}
}
