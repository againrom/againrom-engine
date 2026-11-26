package mapload_test

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/scriptprofile"
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
	second, _, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{})
	if err != nil || second.Dialect() != sim.ScriptROM2 || !second.Extended() || second.Variables() != 150 ||
		!slices.Equal(second.Triggers()[0].Conditions, want) || !slices.Equal(second.Triggers()[0].Actions, tr.Actions) {
		t.Fatal("the second game did not compile the extended record on its own operations", err)
	}
}

// An interactive message action of an extended script raises the event of the
// dialogue its label names; a legacy one raises its event text number, and an
// original script reads no choice at all.
func TestInteractiveMessageRaisesItsDialogue(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("elder", 2, 1, par(1, 4), par(1, 1)),
			node("", 2, 2, par(1, 5), par(1, 0)),
			node("elder", 2, 3, par(1, 6), par(1, 1)),
			node("Set", 3, 4, par(1, 140), par(1, 1)),
		},
		Conditions: []alm.ScriptNode{node("One", 0x10002, 1, par(1, 1))},
		Triggers: []alm.ScriptTrigger{{Name: "T", Left: []uint32{1}, Right: []uint32{1}, Cmp: []uint32{0},
			Acts: []uint32{1, 2, 3}}},
		Extended: true, Variables: 150,
	}
	_, rep, err := mapload.CompileScriptFrom(src, mapload.ScriptRefs{})
	if err != nil {
		t.Fatal(err)
	}
	want := []mapload.ScriptRaise{{Event: mapload.InteractiveEvent(0)}, {Event: 5}, {Event: mapload.InteractiveEvent(0)}}
	if !slices.Equal(rep.Raises, want) || !slices.Equal(rep.Dialogues, []string{"elder"}) {
		t.Errorf("raises %+v dialogues %q", rep.Raises, rep.Dialogues)
	}
	if rep.ActionOf[4] != 3 || rep.CheckOf[1] != 0 {
		t.Errorf("node maps %v %v", rep.ActionOf, rep.CheckOf)
	}
	if i, ok := mapload.InteractiveIndex(rep.Raises[0].Event); !ok || i != 0 {
		t.Errorf("index %d %v", i, ok)
	}
	if _, ok := mapload.InteractiveIndex(255); ok {
		t.Error("an event text number read as a dialogue")
	}
	src.Extended = false
	src.Triggers = []alm.ScriptTrigger{{Name: "T", Left: []uint32{1}, Right: []uint32{1}, Cmp: []uint32{0}, Acts: []uint32{1, 2}}}
	_, rep, err = mapload.CompileScriptFrom(src, mapload.ScriptRefs{})
	if err != nil || len(rep.Dialogues) != 0 || len(rep.Raises) != 2 || rep.Raises[0].Event != 4 {
		t.Errorf("original script: %+v %v", rep, err)
	}
}

func TestInteractiveMessageMatchesCompilerInBothGames(t *testing.T) {
	cases := []struct {
		name      string
		types     [alm.ScriptParams]uint32
		values    [alm.ScriptParams]uint32
		wantFirst bool
		wantSec   bool
	}{
		{"third-slot-after-unset", [alm.ScriptParams]uint32{1, 0, 1}, [alm.ScriptParams]uint32{7, 0, 1}, true, false},
		{"unset-second-slot", [alm.ScriptParams]uint32{1, 0}, [alm.ScriptParams]uint32{7, 1}, false, true},
	}
	for _, tc := range cases {
		n := alm.ScriptNode{Label: "elder", Opcode: 2, ID: 1, Type: tc.types, Value: tc.values}
		src := alm.Script{
			Actions:    []alm.ScriptNode{n},
			Conditions: []alm.ScriptNode{node("One", 0x10002, 1, par(1, 1))},
			Triggers:   []alm.ScriptTrigger{{Name: "T", Left: []uint32{1}, Right: []uint32{1}, Cmp: []uint32{0}, Acts: []uint32{1}}},
			Extended:   true,
			Variables:  100,
		}
		_, rep1, err := mapload.CompileScriptFrom(src, mapload.ScriptRefs{})
		if err != nil {
			t.Fatalf("%s ROM1: %v", tc.name, err)
		}
		_, rep2, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{})
		if err != nil {
			t.Fatalf("%s ROM2: %v", tc.name, err)
		}
		if got := len(rep1.Dialogues) > 0; got != tc.wantFirst || n.InteractiveMessage(scriptprofile.First.UntypedAsInt) != got {
			t.Errorf("%s ROM1: compiler dialogue %v, predicate %v, want %v", tc.name, got, n.InteractiveMessage(scriptprofile.First.UntypedAsInt), tc.wantFirst)
		}
		if got := len(rep2.Dialogues) > 0; got != tc.wantSec || n.InteractiveMessage(scriptprofile.Second.UntypedAsInt) != got {
			t.Errorf("%s ROM2: compiler dialogue %v, predicate %v, want %v", tc.name, got, n.InteractiveMessage(scriptprofile.Second.UntypedAsInt), tc.wantSec)
		}
	}
}
