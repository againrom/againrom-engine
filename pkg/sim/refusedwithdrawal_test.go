package sim

import "testing"

// retreatCorridor is jammedCorridor's jam, closed at (7,5) by one more body,
// with the walker 1 (owner 2) at (7,4) and the hostile 3 (owner 3), which
// acquires nothing, at (5,4). A Retreat from that hostile flees to (10,4)
// inside the jam: the far search reads terrain and finds a route, and the near
// search reaches no cell nearer the flee cell than the walker's own.
func retreatCorridor(t *testing.T, reach uint8) *World {
	t.Helper()
	const width, height = 30, 10
	b, grid := escortWallGrid(width, height)
	walker := engFighter(1, 2, 7, 4)
	walker.Reach = reach
	hostile := engFighter(3, 3, 5, 4)
	hostile.ScanRange = 0
	plug := engFighter(30, 5, 7, 5)
	plug.ScanRange = 0
	ents := []Entity{walker, hostile, plug}
	for k, x := range []int32{8, 9, 10, 11, 12} {
		for r, y := range []int32{4, 5} {
			body := engFighter(EntityID(10+2*k+r), 5, x, y)
			body.ScanRange = 0
			ents = append(ents, body)
		}
	}
	cells := [][3]uint32{{2, 3, 1}, {3, 2, 1}}
	w, err := NewRelatedWorld(1, b, ModeCanonical, Terrain{Block: grid}, ents, nil, engRel(t, cells...))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

// An engine-made withdrawal whose flee walk is refused takes the retreat
// failure tail's answer: the hostile within reach becomes its victim and the
// retreat progress is 1 (AI-365, AI-649). Past reach it takes none.
func TestARefusedPlayerRetreatWalkTakesTheHostileWithinReach(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		reach uint8
		pick  bool
	}{
		"hostile within reach": {reach: 3, pick: true},
		"hostile past reach":   {reach: 1, pick: false},
	} {
		w := retreatCorridor(t, tc.reach)
		Step(w, []Command{GroupRetreat(1, 2, 91)})
		for k := 0; k < 2*scriptCycle && !laEnt(t, w, 1).HasTarget; k++ {
			Step(w, nil)
		}
		if e := laEnt(t, w, 1); e.ActorState != actorStateRetreat || !e.HasTarget {
			t.Fatalf("%s: fixture: state %#x destination %v, want a Retreat walking", name, e.ActorState, e.HasTarget)
		}
		got := stepUntilWalkEnds(t, w)
		if got.X != 7 || got.Y != 4 {
			t.Fatalf("%s: the walk ended at (%d,%d), want (7,4) before the jam", name, got.X, got.Y)
		}
		pick := got.HasAttackTarget && got.AttackTarget == 3 && got.AcquirePursuit
		if pick != tc.pick {
			t.Errorf("%s: victim %v/%d acquisition %v; want the pick %v", name, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit, tc.pick)
		}
		if tc.pick && (got.Retreat.Progress != 1 || got.Retreat.Counter != 0 || got.Retreat.Failure || got.ActorState != actorStateRetreat) {
			t.Errorf("%s: retreat %+v state %#x, want progress 1, counter 0, no flag, still Retreat", name, got.Retreat, got.ActorState)
		}
	}
}

// Each side of refusedWalkTakesPick the Group order and the withdrawal
// decide, pinned on the answer to one refused walk. The walker stands three
// cells from the hostile with reach 3 and no victim. A Group order 0 guard
// takes the pick; a Move, Swarm (order 2), Swarm 2 or Roam member takes none;
// a withdrawal takes it in any of those Groups, with retreat progress 1
// (AI-365, AI-649).
func TestARefusedWalkTakesThePickByGroupOrderAndWithdrawal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		order    uint8
		withdraw bool
		pick     bool
	}{
		{"order 0", orderNone, false, true},
		{"Move", orderMove, false, false},
		{"Swarm order 2", orderSwarm, false, false},
		{"Swarm 2", orderSwarm2, false, false},
		{"Roam", orderRoam, false, false},
		{"withdrawal in order 0", orderNone, true, true},
		{"withdrawal in Move", orderMove, true, true},
		{"withdrawal in Swarm order 2", orderSwarm, true, true},
		{"withdrawal in Swarm 2", orderSwarm2, true, true},
		{"withdrawal in Roam", orderRoam, true, true},
	} {
		w := jammedCorridor(t, 3, 5)
		i := indexOfEntity(w.entities, 1)
		w.commandGroup([]int{i}, tc.order, cell{x: 20, y: 4})
		g := w.savedGroupFor(1)
		if g == nil {
			t.Fatalf("%s: fixture: the walker has no Group", tc.name)
		}
		g.AI[0x20] = tc.order
		e := &w.entities[i]
		e.ActorState = actorStateGuard
		if tc.withdraw {
			e.ActorState = actorStateRetreat
			e.Retreat = RetreatContinuation{Known: true}
		}
		if o := w.savedOrder(e.ID); o != nil && o.State == 1 {
			t.Fatalf("%s: fixture: the walker holds a loaded state 1 order", tc.name)
		}
		w.answerRefusedWalk(i)
		got := w.entities[i]
		pick := got.HasAttackTarget && got.AttackTarget == 3 && got.AcquirePursuit
		if pick != tc.pick {
			t.Errorf("%s: victim %v/%d acquisition %v; want the pick %v", tc.name, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit, tc.pick)
		}
		if tc.withdraw && (got.Retreat.Progress != 1 || got.Retreat.Counter != 0 || got.Retreat.Failure) {
			t.Errorf("%s: retreat %+v, want progress 1, counter 0 and no flag", tc.name, got.Retreat)
		}
	}
}

// The engine-made refusal and a loaded route-failure flag are one answer. At
// the tick a Retreat's walk is refused by its sixteenth stall, the World's
// own refusal and a twin whose current loaded motion holds the flag give the
// walker the same victim, acquisition and retreat continuation. Loss
// controls: one tick earlier neither the World nor a twin holding the loaded
// motion without the flag has a victim.
func TestAnEngineMadeAndALoadedRefusedRetreatTakeOneAnswer(t *testing.T) {
	t.Parallel()
	w := retreatCorridor(t, 3)
	Step(w, []Command{GroupRetreat(1, 2, 91)})
	for k := 0; k < 4*scriptCycle && laEnt(t, w, 1).Stall < stallLimit-2; k++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, 1); e.Stall != stallLimit-2 || !e.HasTarget || e.HasAttackTarget {
		t.Fatalf("fixture: stall %d destination %v victim %v, want a walk two stalls short of its refusal", e.Stall, e.HasTarget, e.HasAttackTarget)
	}
	loaded := func(w *World, flag byte) *World {
		b := worldRoundTripForTest(t, w)
		motion := SavedActorMotion{Entity: 1, Current: true}
		motion.Mover[0x98] = flag
		b.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{motion}}
		return b
	}
	quiet := loaded(w, 0)
	Step(w, nil)
	Step(quiet, nil)
	for name, c := range map[string]*World{"engine-made": w, "loaded without the flag": quiet} {
		if e := laEnt(t, c, 1); e.HasAttackTarget {
			t.Fatalf("%s: a victim %d one tick before the refusal", name, e.AttackTarget)
		}
	}
	flagged := loaded(w, 1)
	Step(w, nil)
	Step(flagged, nil)
	a, b := laEnt(t, w, 1), laEnt(t, flagged, 1)
	if !a.HasAttackTarget || a.AttackTarget != 3 || !a.AcquirePursuit || a.Retreat.Progress != 1 {
		t.Fatalf("engine-made: victim %v/%d acquisition %v retreat %+v, want the hostile and progress 1", a.HasAttackTarget, a.AttackTarget, a.AcquirePursuit, a.Retreat)
	}
	if a.HasAttackTarget != b.HasAttackTarget || a.AttackTarget != b.AttackTarget || a.AcquirePursuit != b.AcquirePursuit ||
		a.AttackPhase != b.AttackPhase || a.Retreat != b.Retreat || a.HasTarget != b.HasTarget || a.X != b.X || a.Y != b.Y {
		t.Errorf("the answers differ:\nengine-made %v/%d %v %v %+v\nloaded      %v/%d %v %v %+v",
			a.HasAttackTarget, a.AttackTarget, a.AcquirePursuit, a.AttackPhase, a.Retreat,
			b.HasAttackTarget, b.AttackTarget, b.AcquirePursuit, b.AttackPhase, b.Retreat)
	}
	if flagged.motionFor(1).Mover[0x98] != 0 {
		t.Error("the loaded flag was not consumed")
	}
}
