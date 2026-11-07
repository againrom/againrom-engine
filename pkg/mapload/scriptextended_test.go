package mapload_test

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// An extended trigger compiles every condition with its group and every
// action in order: a condition id naming no check reads ScriptNone, and an
// action id naming a build-time node or no node runs nothing.
func TestExtendedRecordCompilesGroupsListsAndMisses(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("Drop location", 0x10002, 99, par(5, 16), par(6, 16)),
			node("Set", 3, 1, par(1, 140), par(1, 11)),
			node("Inc", 8, 2, par(1, 141)),
		},
		Conditions: []alm.ScriptNode{
			node("One", 0x10002, 1, par(1, 1)),
			node("Var", 19, 2, par(1, 120)),
		},
		Triggers: []alm.ScriptTrigger{{Name: "T", Once: 1, Group: []uint32{0, 1, 1},
			Left: []uint32{2, 1, 9}, Right: []uint32{1, 1, 1}, Cmp: []uint32{0, 0, 1},
			Acts: []uint32{2, 99, 7, 1, 2}}},
		Extended: true, Variables: 150,
	}
	s, rep, err := mapload.CompileScriptFrom(src, mapload.ScriptRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Extended() || s.Variables() != 150 || len(s.Triggers()) != 1 || len(rep.DropCells) != 1 {
		t.Fatalf("compiled %v %d, report %+v", s.Extended(), s.Variables(), rep)
	}
	tr := s.Triggers()[0]
	want := []sim.ScriptPair{
		{Left: 1, Right: 0, Cmp: 0, Used: true, Group: 0},
		{Left: 0, Right: 0, Cmp: 0, Used: true, Group: 1},
		{Left: sim.ScriptNone, Right: 0, Cmp: 1, Used: true, Group: 1},
	}
	if !slices.Equal(tr.Conditions, want) || !slices.Equal(tr.Actions, []int32{1, 0, 1}) || !tr.Once || tr.Latch != 0 {
		t.Fatalf("trigger %+v", tr)
	}
	if tr.Pairs != [3]sim.ScriptPair{} || tr.Instants != [4]int32{sim.ScriptNone, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone} {
		t.Fatalf("the record slots of an extended trigger are used: %+v", tr)
	}
	if _, _, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{}); err == nil || !strings.Contains(err.Error(), "first game's") {
		t.Fatal("the second game compiled the extended record", err)
	}
}
