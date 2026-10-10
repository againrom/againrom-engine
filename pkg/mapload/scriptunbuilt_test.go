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

// mission90Source is 90.alm's two win triggers in miniature: subscript 0 is
// message 1, T4 runs the win and then takes water from 10001, 10002 and
// 10003, and T19 takes it from 10005. T5 names an id the map does not hold.
// Triggers 0..3 and 6..18 are dropped placeholders, so the latches keep the
// map's positions.
func mission90Source() alm.Script {
	src := alm.Script{
		Actions: []alm.ScriptNode{
			node("Send message hello", 2, 2, par(1, 1)),
			node("Force Mission Complete", 4, 8),
			node("remove water from Danath", 13, 43, par(4, 10001), par(8, 10)),
			node("remove water from Reniesta", 13, 44, par(4, 10002), par(8, 10)),
			node("remove water from Naira", 13, 45, par(4, 10003), par(8, 10)),
			node("remove water from Paladin", 13, 46, par(4, 10005), par(8, 10)),
		},
		Conditions: []alm.ScriptNode{
			node("Constant Value=0", 0x10002, 6, par(1, 0)),
		},
	}
	src.Triggers = make([]alm.ScriptTrigger, 20)
	src.Triggers[4] = trg("Mission complit", [3]uint32{6, 0, 0}, [3]uint32{6, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{8, 43, 44, 45}, 1)
	src.Triggers[5] = trg("names nothing", [3]uint32{6, 0, 0}, [3]uint32{6, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{99, 0, 0, 0}, 1)
	src.Triggers[19] = trg("Mission complit 2", [3]uint32{6, 0, 0}, [3]uint32{6, 0, 0}, [3]uint32{0, 0, 0}, [4]uint32{46, 0, 0, 0}, 1)
	return src
}

func TestASlotNamingAnUnbuiltNodeRaisesSubscriptZerosMessage(t *testing.T) {
	refs := mapload.ScriptRefs{Hero: 9, HasHero: true, Roster: true}
	s, rep := compile(t, mission90Source(), refs)
	if !slices.Equal(rep.OmittedActions, []uint32{44, 45, 46}) {
		t.Fatalf("omitted actions %v, want [44 45 46]", rep.OmittedActions)
	}
	if in := s.Instants()[0]; in.Op != 2 || in.Args[0] != 1 {
		t.Fatalf("instant 0 %+v, want message 1", in)
	}
	byLatch := map[int32][]int32{}
	for _, r := range rep.Raises {
		byLatch[r.Latch] = append(byLatch[r.Latch], r.Event)
	}
	if got := byLatch[4]; !slices.Equal(got, []int32{1, 1}) {
		t.Errorf("T4 raises %v, want message 1 for each of its two unbuilt slots", got)
	}
	if got := byLatch[19]; !slices.Equal(got, []int32{1}) {
		t.Errorf("T19 raises %v, want message 1", got)
	}
	if got := byLatch[5]; len(got) != 0 {
		t.Errorf("T5 raises %v, want none: an id the map does not hold is not an unbuilt node", got)
	}
	if len(rep.Raises) != 3 {
		t.Errorf("raises %+v, want three", rep.Raises)
	}
}

func TestAResolvedRosterRaisesNoSubscriptZeroMessage(t *testing.T) {
	refs := mapload.ScriptRefs{Hero: 9, HasHero: true, Companion: 10, HasCompanion: true,
		Roles: map[uint32]sim.EntityID{10003: 11, 10005: 12}, Roster: true}
	_, rep := compile(t, mission90Source(), refs)
	if len(rep.OmittedActions) != 0 || len(rep.Raises) != 0 {
		t.Errorf("omitted %v raises %+v, want every node built and no raise", rep.OmittedActions, rep.Raises)
	}
}

func TestAnUnbuiltSlotRaisesNothingWhenSubscriptZeroIsNoMessage(t *testing.T) {
	src := mission90Source()
	src.Actions[0], src.Actions[1] = src.Actions[1], src.Actions[0]
	_, rep := compile(t, src, mapload.ScriptRefs{Hero: 9, HasHero: true, Roster: true})
	if len(rep.Raises) != 0 {
		t.Errorf("raises %+v, want none: subscript 0 is the win, not a message", rep.Raises)
	}
}
