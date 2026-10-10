package mapload_test

import (
	"slices"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// unbuiltSource is the mission-100 shape in miniature: a drop location, a
// group Move, a take naming hero ordinal 10002, a constant, a check naming
// ordinal 10003 and a check naming a placed unit, and one trigger whose slot
// names the take and whose pair names the 10003 check.
func unbuiltSource() alm.Script {
	return alm.Script{
		Actions: []alm.ScriptNode{
			node("drop", 0x10002, 1, par(5, 4), par(6, 4)),
			node("servant flees", 6, 2, par(7, 4), par(5, 13), par(6, 14), par(2, 21)),
			node("take from 10002", 13, 3, par(4, 10002), par(8, 11)),
			node("message", 2, 4, par(1, 7)),
		},
		Conditions: []alm.ScriptNode{
			node("const 0", 0x10002, 1, par(1, 0)),
			node("10003 alive", 5, 2, par(4, 10003)),
			node("unit 77 alive", 5, 3, par(4, 77)),
		},
		Triggers: []alm.ScriptTrigger{
			trg("win", [3]uint32{2, 3, 0}, [3]uint32{1, 1, 0}, [3]uint32{0, 0, 0}, [4]uint32{3, 4, 0, 0}, 1),
		},
	}
}

func TestAnUnresolvedRoleNodeIsNotBuiltUnderTheRoster(t *testing.T) {
	refs := mapload.ScriptRefs{Hero: 9, HasHero: true, Roster: true}
	s, rep := compile(t, unbuiltSource(), refs)
	instants, checks := s.Instants(), s.Checks()
	if len(instants) != 2 || instants[0].Op != 6 || instants[1].Op != 2 {
		t.Fatalf("instants %+v, want the group Move at 0 and the message at 1", instants)
	}
	if len(checks) != 2 || checks[0].Op != int32(0x10002) || checks[1].Op != 5 || checks[1].HasUnit {
		t.Fatalf("checks %+v, want the constant at 0 and the unit-77 check at 1, built with no unit", checks)
	}
	tr := s.Triggers()[0]
	if tr.Instants[0] != 0 || tr.Instants[1] != 1 {
		t.Errorf("slots %v, want the unbuilt take at subscript 0 and the message at 1", tr.Instants)
	}
	if tr.Pairs[0].Left != 0 || tr.Pairs[1].Left != 1 {
		t.Errorf("pairs %+v, want the unbuilt check at register 0 and the unit-77 check at 1", tr.Pairs)
	}
	if !slices.Equal(rep.OmittedActions, []uint32{3}) || !slices.Equal(rep.OmittedChecks, []uint32{2}) || len(rep.OmittedTriggers) != 0 {
		t.Errorf("omitted actions %v checks %v triggers %v, want [3] [2] []", rep.OmittedActions, rep.OmittedChecks, rep.OmittedTriggers)
	}
	if len(rep.Unresolved) != 3 {
		t.Errorf("unresolved %+v, want the two roles and unit 77", rep.Unresolved)
	}
}

func TestAResolvedRosterBuildsEveryNode(t *testing.T) {
	refs := mapload.ScriptRefs{Hero: 9, HasHero: true, Companion: 10, HasCompanion: true,
		Roles: map[uint32]sim.EntityID{10003: 11}, Units: map[uint16]sim.EntityID{77: 3}, Roster: true}
	s, rep := compile(t, unbuiltSource(), refs)
	if len(s.Instants()) != 3 || len(s.Checks()) != 3 || len(rep.OmittedActions)+len(rep.OmittedChecks) != 0 {
		t.Fatalf("instants %d checks %d omitted %v %v, want every node built", len(s.Instants()), len(s.Checks()), rep.OmittedActions, rep.OmittedChecks)
	}
	tr := s.Triggers()[0]
	if tr.Instants[0] != 1 || tr.Pairs[0].Left != 1 {
		t.Errorf("slot %d pair %d, want the take at subscript 1 and the check at register 1", tr.Instants[0], tr.Pairs[0].Left)
	}
}

func TestWithoutTheRosterEveryRoleNodeIsBuilt(t *testing.T) {
	s, rep := compile(t, unbuiltSource(), mapload.ScriptRefs{Hero: 9, HasHero: true})
	if len(s.Instants()) != 3 || len(s.Checks()) != 3 || len(rep.OmittedActions)+len(rep.OmittedChecks) != 0 {
		t.Fatalf("instants %d checks %d omitted %v %v, want every node built", len(s.Instants()), len(s.Checks()), rep.OmittedActions, rep.OmittedChecks)
	}
	if in := s.Instants()[1]; in.Op != 13 || in.HasUnit {
		t.Errorf("instant 1 %+v, want the take built with no unit", in)
	}
}
