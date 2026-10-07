package mapload_test

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"testing"
)

func TestSecondGameCompilerSpecialActionDoesNotAssignSpawnMeaning(t *testing.T) {
	src := alm.Script{Actions: []alm.ScriptNode{node("pair", 65538, 1, par(5, 267), par(6, 297))}}
	_, rep, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.DropCells) != 0 {
		t.Fatalf("ROM2 special action acquired unproved spawn meaning: %+v", rep.DropCells)
	}
	if len(rep.PackedCoordinates) != 1 || rep.PackedCoordinates[0] != 0x290b {
		t.Fatalf("packed collection = %x", rep.PackedCoordinates)
	}
}

func TestSecondGameCompilerPreservesBankAndObjectiveOperands(t *testing.T) {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("set", 35, 20, par(1, 779), par(1, 0xfffffff9)), node("goal", 36, 21, par(1, 1), par(1, 2)),
			node("bad", 37, 22, par(5, 8), par(6, 9), par(1, 1), par(1, 2), par(1, 3), par(1, 4), par(1, 5), par(1, 6)),
			node("take", 38, 23, par(8, 6)), node("clear", 39, 24, par(2, 9)),
		},
		Conditions: []alm.ScriptNode{
			node("constant", 65538, 2, par(1, 0xfffffffe)), node("bank", 23, 4, par(1, 779)),
			node("goal", 24, 6, par(1, 1)), node("layer", 25, 7, par(5, 3), par(6, 4), par(1, 2)),
			node("effect", 26, 8, par(4, 12), par(1, 44)), node("center", 27, 9, par(4, 12), par(5, 3), par(6, 4)),
		},
		Triggers: []alm.ScriptTrigger{trg("compare", [3]uint32{2}, [3]uint32{4}, [3]uint32{uint32(sim.ScriptCmpLT)}, [4]uint32{20, 21, 22, 23}, 1)},
	}
	s, rep, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{12: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Dialect() != sim.ScriptROM2 || len(s.Unsupported()) != 0 || !rep.Empty() {
		t.Fatalf("program/compile=%+v/%+v", s.Unsupported(), rep)
	}
	checks := s.Checks()
	if checks[0].Op != 65538 || checks[0].Args[0] != -2 || checks[4].Args[0] != 44 || checks[5].Args[0] != 3 || checks[5].Args[1] != 4 || checks[5].Args[2] != 0 {
		t.Fatalf("compiled checks=%+v", checks)
	}
	instants := s.Instants()
	if instants[0].Args[1] != -7 || instants[1].Args[0] != 1 || instants[2].Args[7] != 6 || instants[3].Item != 0xe1e || !instants[3].HasItem || !instants[4].HasGroup || instants[4].Group != 9 {
		t.Fatalf("compiled instants=%+v", instants)
	}
	triggers := s.Triggers()
	if triggers[0].Pairs[0].Left != 0 || triggers[0].Pairs[0].Right != 1 || triggers[0].Instants != ([4]int32{0, 1, 2, 3}) || !triggers[0].Once {
		t.Fatalf("trigger=%+v", triggers)
	}
}

func TestSecondGameCompilerOmitsUnresolvedReferencesAndDependentTriggers(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{node("constant", 65538, 2, par(1, 0)), node("missing", 26, 7, par(4, 6), par(1, 12)), node("dynamic", 27, 8, par(4, 8001), par(5, 3), par(6, 4))}, Actions: []alm.ScriptNode{node("missing", 12, 6, par(4, 7), par(8, 6))}, Triggers: []alm.ScriptTrigger{trg("missing condition", [3]uint32{7}, [3]uint32{2}, [3]uint32{0}, [4]uint32{6}, 1)}}
	s, rep, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{Units: map[uint16]sim.EntityID{8001: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Checks()) != 1 || len(s.Instants()) != 0 || len(s.Triggers()) != 0 || len(rep.Unresolved) != 3 || len(rep.OmittedActions) != 1 || len(rep.OmittedChecks) != 2 || len(rep.OmittedTriggers) != 1 {
		t.Fatalf("program arrays%d/%d/%d report%+v", len(s.Checks()), len(s.Instants()), len(s.Triggers()), rep)
	}
}

func TestSecondGameCompilerPartyBridgeUsesOnlyExplicitBindings(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{node("hero", 26, 1, par(4, 10001), par(1, 12)), node("missing", 26, 2, par(4, 10003), par(1, 12))}}
	s, rep, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{Hero: 1, HasHero: true})
	if err != nil {
		t.Fatal(err)
	}
	if checks := s.Checks(); len(checks) != 1 || checks[0].Unit != 1 || len(rep.OmittedChecks) != 1 {
		t.Fatalf("party bridge=%+v report%+v", checks, rep)
	}
	s, _, err = mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{Hero: 1, HasHero: true, Units: map[uint16]sim.EntityID{10001: 9}, Roles: map[uint32]sim.EntityID{10003: 2}})
	if err != nil {
		t.Fatal(err)
	}
	if checks := s.Checks(); len(checks) != 2 || checks[0].Unit != 9 || checks[1].Unit != 2 {
		t.Fatalf("explicit priority=%+v", checks)
	}
}

func TestSecondGameInitializerCopiesRawFirstOperandBeforeReferenceBinding(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{node("raw", 65538, 1, par(4, 8001), par(1, 9))}, Actions: []alm.ScriptNode{node("raw pair", 65538, 2, par(4, 267), par(4, 297))}}
	s, rep, err := mapload.CompileROM2ScriptFrom(src, mapload.ScriptRefs{})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Checks()[0]; got.Args[0] != 8001 || got.HasUnit {
		t.Fatalf("initializer=%+v", got)
	}
	if len(rep.Unresolved) != 0 || len(rep.PackedCoordinates) != 1 || rep.PackedCoordinates[0] != 0x290b {
		t.Fatalf("initializer bound references: %+v", rep)
	}
}
