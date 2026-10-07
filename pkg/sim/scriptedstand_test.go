package sim

import (
	"encoding/binary"
	"testing"
)

const scriptStandOrder = int32(orderStandGround)

// scriptStandWalker is a unit on a walk with no hostile in sight, native or
// saved.
func scriptStandWalker(t *testing.T, saved bool) *World {
	t.Helper()
	archer := lcArcher(lcArcherID, 3, 20, 20)
	archer.Group = lcGroup
	w := engWorld(t, engRel(t), archer)
	if saved {
		g := SavedGroup{ID: 71, Selector: lcGroup, Words: []uint16{97}, Path: []uint16{0x1111}}
		g.Members = []SavedGroupMember{{1, lcArcherID, true}}
		g.AI[0x20], g.AI[0x45], g.AI[0x38], g.AI[0x44] = 0, 1, 9, 7
		orders := []SavedActorOrder{{Entity: lcArcherID}}
		if err := w.ImportSavedGroups([]SavedGroup{g}, orders); err != nil {
			t.Fatalf("ImportSavedGroups: %v", err)
		}
	}
	w.runInstant(lcGroupOrder(lcGroup, 0, int32(orderMove), 40, 20))
	for range 3 {
		Step(w, nil)
	}
	if e := laEnt(t, w, lcArcherID); !e.HasTarget || e.X == 20 {
		t.Fatalf("fixture: the member holds destination %t at %d, want a walk under way", e.HasTarget, e.X)
	}
	return w
}

// The script's Stand Ground ends a member's walk (AI-370).
func TestScriptStandGroundStandsAMemberOnAWalk(t *testing.T) {
	for _, saved := range []bool{false, true} {
		w := scriptStandWalker(t, saved)
		control := worldRoundTripForTest(t, w)
		w.runInstant(lcGroupOrder(lcGroup, 0, scriptStandOrder))
		e := laEnt(t, w, lcArcherID)
		if e.HasTarget || e.HasAttackTarget || e.ActorState != actorStateGuard {
			t.Fatalf("saved=%t: member holds destination %t victim %t state %#x, want none and guard", saved, e.HasTarget, e.HasAttackTarget, e.ActorState)
		}
		if saved {
			if o := w.savedOrder(lcArcherID); o == nil || o.Raw[8] != 0 || o.State != uint32(actorStateGuard) {
				t.Fatalf("saved order after Stand Ground: %+v", o)
			}
		} else if order, _, ok := w.groupState(3, lcGroup); !ok || order != orderStandGround {
			t.Fatalf("group order %d known %t, want Stand Ground", order, ok)
		}
		at := e.X
		for range 120 {
			Step(w, nil)
			Step(control, nil)
		}
		if got := laEnt(t, w, lcArcherID); got.X != at || got.HasTarget {
			t.Fatalf("saved=%t: member moved from %d to %d after Stand Ground", saved, at, got.X)
		}
		if got := laEnt(t, control, lcArcherID); got.X <= at {
			t.Fatalf("saved=%t: control without Stand Ground stayed at %d", saved, got.X)
		}
	}
}

// A patrolling or escorting member leaves that state (AI-370).
func TestScriptStandGroundEndsAPatrolAndAnEscort(t *testing.T) {
	for _, name := range []string{"patrol", "escort"} {
		archer := lcArcher(lcArcherID, 3, 20, 20)
		archer.Group = lcGroup
		w := engWorld(t, engRel(t), archer, lcArcher(3, 3, 22, 20))
		if name == "patrol" {
			w.runInstant(lcGroupOrder(lcGroup, 0, subCommandPatrol, 30, 20))
		} else {
			w.runInstant(lcGroupOrder(lcGroup, 3, subCommandDefend, 3))
		}
		want := actorStatePatrol
		if name == "escort" {
			want = actorStateDefend
		}
		if e := laEnt(t, w, lcArcherID); e.ActorState != want && !(name == "escort" && e.ActorState == actorStateAcquire) {
			t.Fatalf("%s: fixture state %#x", name, e.ActorState)
		}
		w.runInstant(lcGroupOrder(lcGroup, 0, scriptStandOrder))
		if e := laEnt(t, w, lcArcherID); e.ActorState != actorStateGuard || e.HasTarget {
			t.Fatalf("%s: after Stand Ground state %#x destination %t, want guard and none", name, e.ActorState, e.HasTarget)
		}
	}
}

// A loaded cycle is kept (AI-352, AI-354).
func TestScriptStandGroundKeepsALoadedCycle(t *testing.T) {
	for _, saved := range []bool{false, true} {
		w := lcBuild(t, 3, saved, 0)
		w.runInstant(lcGroupOrder(lcGroup, 0, scriptStandOrder))
		lcRequireRetained(t, w, lcArcherID, lcVictimID)
		lcBlowFirst(t, w, lcArcherID, lcVictimID)
	}
}

// The saved completion word is 0 after Stand Ground (AI-351, AI-370).
func TestStandGroundWritesTheSavedCompletionWordZero(t *testing.T) {
	w := lcBuild(t, 3, true, 0)
	o := w.savedOrder(lcArcherID)
	binary.LittleEndian.PutUint32(o.Raw[0x50:], 1)
	w.runInstant(lcGroupOrder(lcGroup, 0, scriptStandOrder))
	if got := binary.LittleEndian.Uint32(w.savedOrder(lcArcherID).Raw[0x50:]); got != 0 {
		t.Fatalf("completion word %d after the script's Stand Ground, want 0", got)
	}
	w = lcBuild(t, SelfSlot, true, 0)
	binary.LittleEndian.PutUint32(w.savedOrder(lcArcherID).Raw[0x50:], 1)
	Step(w, []Command{GroupStance(lcArcherID, OrderStandGround, SelfSlot)})
	if got := binary.LittleEndian.Uint32(w.savedOrder(lcArcherID).Raw[0x50:]); got != 0 {
		t.Fatalf("completion word %d after Hold, want 0", got)
	}
}
