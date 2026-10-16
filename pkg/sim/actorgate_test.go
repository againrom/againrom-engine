package sim

import "testing"

// gateArms are the six per-actor arms the dispatch enters on the actor state.
var gateArms = []struct {
	name  string
	state uint8
}{
	{"guard", actorStateGuard},
	{"acquire", actorStateAcquire},
	{"patrol", actorStatePatrol},
	{"defend", actorStateDefend},
	{"follow", actorStateFollow},
	{"retreat", actorStateRetreat},
}

// gateOrders are the Group orders other than 0: the five arms, 0x11, 0xff,
// and 6 for the band that does nothing (AI-ORDER-010).
var gateOrders = []uint8{orderGuard, orderSwarm, orderStandGround, orderMove, orderSwarm2, orderRoam, 6, 0xff}

// gateWorld is a warrior of the local participant in the given actor state,
// a hostile two cells east that never acts, and an ally twelve cells east the
// escort states name. Each state is reached through its own setter where the
// player has one; acquire and follow are written the way their script setters
// leave them.
func gateWorld(t *testing.T, state uint8) *World {
	t.Helper()
	m := withdrawalFighter(1, SelfSlot, 20, 20, 100)
	m.Group = 1
	ally := withdrawalFighter(3, SelfSlot, 32, 20, 100)
	ally.Group = 9
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2})
	w := engWorld(t, rel, m, withdrawalFighter(2, 3, 22, 20, 100), ally)
	w.tick = scriptPassPhase
	i := indexOfEntity(w.entities, 1)
	switch state {
	case actorStatePatrol:
		Step(w, []Command{GroupPatrolTo(1, CellPoint{X: 26, Y: 26}, 1)})
	case actorStateDefend, actorStateFollow:
		Step(w, []Command{GroupDefend(1, 3, 1)})
	case actorStateRetreat:
		Step(w, []Command{GroupRetreat(1, SelfSlot, 1)})
	default:
		Step(w, []Command{GroupStance(1, int32(orderGuard), 1)})
	}
	if state == actorStateAcquire || state == actorStateFollow {
		w.entities[i].ActorState = state
	}
	o := w.ensureSavedOrder(i)
	o.State = uint32(state)
	o.authorNative()
	if w.entities[i].ActorState != state {
		t.Fatalf("fixture: the warrior holds state %#x, want %#x", w.entities[i].ActorState, state)
	}
	return w
}

// gateSetOrder writes the order of the Group holding the warrior, the byte a
// later Group command writes without touching its members' states.
func gateSetOrder(t *testing.T, w *World, order uint8) {
	t.Helper()
	g := w.savedGroupFor(1)
	if g == nil {
		t.Fatal("fixture: the warrior stands in no Group")
	}
	g.AI[0x20] = order
}

// gateReads runs fn with the dispatch observed and returns the states it read
// for entity id, in order.
func gateReads(id EntityID, fn func()) []uint8 {
	var got []uint8
	actorStateRead = func(e EntityID, state uint8) {
		if e == id {
			got = append(got, state)
		}
	}
	defer func() { actorStateRead = nil }()
	fn()
	return got
}

// TestFirstGameArmsRunUnderGroupOrderZeroAlone: the per-actor machine runs under
// Group order 0 alone (AI-ORDER-010, AI-POST-095). For each arm, a warrior in
// that state has it read and its arm entered under order 0, and under every
// other order no dispatch reads it.
func TestFirstGameArmsRunUnderGroupOrderZeroAlone(t *testing.T) {
	for _, arm := range gateArms {
		for _, order := range append([]uint8{orderNone}, gateOrders...) {
			w := gateWorld(t, arm.state)
			if w.groupDialectNative() {
				t.Fatal("fixture: a first-game World holds a Group registry")
			}
			gateSetOrder(t, w, order)
			got := gateReads(1, func() { engRun(w, 3) })
			if order == orderNone && (len(got) == 0 || got[0] != arm.state) {
				t.Fatalf("%s, order 0: the dispatch read %#x, want %#x", arm.name, got, arm.state)
			}
			if order != orderNone && len(got) != 0 {
				t.Fatalf("%s, order %#x: the dispatch read the member's state %#x", arm.name, order, got)
			}
		}
	}
}

// TestScriptGuardOverAPatrolStopsThePatrolArm: a script Guard against a Group
// an earlier node put on patrol raises the order to 1 and leaves the members
// in the patrol state. The patrol arm no longer runs, so the member holds the
// post Guard anchored; the control, left on patrol, walks its ring.
func TestScriptGuardOverAPatrolStopsThePatrolArm(t *testing.T) {
	build := func() *World {
		archer := lcArcher(lcArcherID, 3, 20, 20)
		archer.Group = lcGroup
		w := engWorld(t, engRel(t), archer)
		w.runInstant(lcGroupOrder(lcGroup, 0, subCommandPatrol, 30, 20))
		return w
	}
	w, control := build(), build()
	w.runInstant(lcGroupOrder(lcGroup, 0, int32(orderGuard)))
	i := indexOfEntity(w.entities, lcArcherID)
	if order, ok := registryOrder(w, i); !ok || order != orderGuard {
		t.Fatalf("after Guard the Group stands at order %d (known %v), want Guard", order, ok)
	}
	if e := w.entities[i]; e.ActorState != actorStatePatrol || !w.nativePatrol(i) {
		t.Fatalf("fixture: Guard rewrote the member's state to %#x", e.ActorState)
	}
	got := gateReads(lcArcherID, func() { engRun(w, 4) })
	engRun(control, 4)
	if len(got) != 0 {
		t.Fatalf("under Guard the dispatch read the patroller's state %#x", got)
	}
	if e := laEnt(t, w, lcArcherID); e.X != 20 || e.Y != 20 || e.HasTarget {
		t.Fatalf("under Guard the patroller moved to (%d,%d), destination %v", e.X, e.Y, e.HasTarget)
	}
	if e := laEnt(t, control, lcArcherID); e.X <= 20 {
		t.Fatalf("control: the patroller left on patrol stayed at (%d,%d)", e.X, e.Y)
	}
}

// TestSecondGameDispatchKeepsItsGuardOnlyGate: the second game's order machine
// is its own, so its dispatch is unchanged. A patroller whose Group order is
// raised to 1 is still read and still walks its ring.
func TestSecondGameDispatchKeepsItsGuardOnlyGate(t *testing.T) {
	m := withdrawalFighter(1, SelfSlot, 20, 20, 100)
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{m}, secondGameScript(t, nil, nil, nil), engRel(t))
	if err != nil {
		t.Fatal(err)
	}
	if !w.SecondGame() || !w.groupDialectNative() {
		t.Fatal("fixture: not a second-game World")
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupPatrolTo(1, CellPoint{X: 30, Y: 20}, 1)})
	e := laEnt(t, w, 1)
	raised := false
	for gi := range w.groups {
		if w.groups[gi].owner == e.Owner && w.groups[gi].group == effectiveGroup(&e) {
			w.groups[gi].order, raised = orderGuard, true
		}
	}
	if !raised || e.ActorState != actorStatePatrol {
		t.Fatalf("fixture: Group record %v, state %#x", raised, e.ActorState)
	}
	at := e.X
	got := gateReads(1, func() { engRun(w, 4) })
	if len(got) == 0 || got[0] != actorStatePatrol {
		t.Fatalf("the second-game dispatch read %#x, want the patrol state", got)
	}
	if e := laEnt(t, w, 1); e.X <= at {
		t.Fatalf("the second-game patroller stayed at (%d,%d)", e.X, e.Y)
	}
}
