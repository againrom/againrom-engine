package sim

import "testing"

// groupOwnerWorld carries group 40 under two owners: owner 5 holds three
// members and owner 7 holds two. A script group id resolves to the later owner's
// group alone.
func groupOwnerWorld(t *testing.T, s *Script) *World {
	t.Helper()
	ent := func(id EntityID, owner uint32, x, y int32) Entity {
		return Entity{ID: id, HP: 40, MaxHP: 40, Speed: 20, Group: 40, Owner: owner, X: x, Y: y}
	}
	return scriptWorld(t, s, []Entity{
		ent(1, 5, 10, 10), ent(2, 5, 12, 10), ent(3, 5, 14, 10),
		ent(4, 7, 60, 60), ent(5, 7, 62, 60),
	})
}

func TestScriptGroupCommandReachesTheLaterOwnerOnly(t *testing.T) {
	s := fireOnce(t, []ScriptInstant{{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 40,
		Args: [scriptParams]int32{int32(orderSwarm2), 30, 30}}}, 0)
	w := groupOwnerWorld(t, s)
	runPass(t, w)
	for _, e := range w.Entities() {
		moved := e.X != 10 && e.X != 12 && e.X != 14 && e.X != 60 && e.X != 62 || e.HasTarget
		switch e.Owner {
		case 5:
			if moved {
				t.Errorf("owner 5 entity %d was commanded", e.ID)
			}
		case 7:
			if !moved {
				t.Errorf("owner 7 entity %d was not commanded", e.ID)
			}
		}
	}
	groups := ObserveScriptGroups(w, 40)
	if len(groups) != 1 || groups[0].Owner != 7 || groups[0].Order != uint8(orderSwarm2) {
		t.Fatalf("resolved group record: %+v", groups)
	}
	if _, base, ok := w.groupState(5, 40); !ok || w.groups[0].order == orderSwarm2 {
		t.Fatalf("owner 5 record changed or missing: base %d ok %v order %d", base, ok, w.groups[0].order)
	}
}

func TestScriptGroupCountAndMembersFollowTheLaterOwner(t *testing.T) {
	s := mustScript(t,
		[]ScriptCheck{groupCheck(0, 40), constCheck(1, 2)},
		[]ScriptInstant{{Op: ScriptInstantGroupOffMap, HasGroup: true, Group: 40}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(0, 1, ScriptCmpEQ)},
			Instants: acts(0), Once: true, Latch: 0}})
	w := groupOwnerWorld(t, s)
	runPass(t, w)
	if got := w.ScriptRegister(0); got != 2 {
		t.Fatalf("count of group 40 is %d, want owner 7's 2", got)
	}
	for _, e := range w.Entities() {
		if off := e.OffMap; off != (e.Owner == 7) {
			t.Errorf("entity %d owner %d off the map %v", e.ID, e.Owner, off)
		}
	}
}

func TestScriptGroupIdWithOneOwnerIsUnchanged(t *testing.T) {
	s := fireOnce(t, []ScriptInstant{{Op: ScriptInstantGroupOrder, HasGroup: true, Group: 40,
		Args: [scriptParams]int32{int32(orderSwarm2), 30, 30}}}, 0)
	w := scriptWorld(t, s, []Entity{
		{ID: 1, HP: 40, MaxHP: 40, Speed: 20, Group: 40, Owner: 5, X: 10, Y: 10},
		{ID: 2, HP: 40, MaxHP: 40, Speed: 20, Group: 40, Owner: 5, X: 12, Y: 10},
	})
	runPass(t, w)
	if g := ObserveScriptGroups(w, 40); len(g) != 1 || g[0].Owner != 5 || g[0].Order != uint8(orderSwarm2) {
		t.Fatalf("single-owner group record: %+v", g)
	}
}
