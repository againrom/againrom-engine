package sim

import "testing"

// refusedRouteEntries are the four entries that answer a refused route, each
// called as its production caller calls it once the walk has ended.
var refusedRouteEntries = []struct {
	name string
	call func(w *World, i int)
}{
	{"pursuit", func(w *World, i int) { w.answerRefusedPursuit(i) }},
	{"walk", func(w *World, i int) { w.answerRefusedWalk(i) }},
	{"flee", func(w *World, i int) { w.reacquireWithinReach(i) }},
	{"retreat failure", func(w *World, i int) {
		w.ensureRetreatContinuation(i)
		w.entities[i].Retreat.Failure = true
		w.retreatFailureTail(i)
	}},
}

// refusedRouteWalker is jammedCorridor's walker 1 at (2,4) with reach 3 and
// the hostile 3 at (hostileX,5), plus a second hostile 4 at (25,4) past reach.
// The walker stands in Guard, or in Retreat with the continuation r, and holds
// victim when victim is nonzero.
func refusedRouteWalker(t *testing.T, hostileX int32, victim EntityID, retreat *RetreatContinuation) (*World, int) {
	t.Helper()
	w := jammedCorridor(t, 3, hostileX, engFighter(4, 3, 25, 4))
	i := indexOfEntity(w.entities, 1)
	w.entities[i].ActorState = actorStateGuard
	if victim != 0 && !w.orderAttack(i, victim) {
		t.Fatalf("fixture: the walker takes no victim %d", victim)
	}
	if retreat != nil {
		w.entities[i].ActorState = actorStateRetreat
		w.entities[i].Retreat = *retreat
	}
	return w, i
}

// With no hostile in reach each walk state leaves its own outcome: a pursuit
// goes idle and keeps its victim (AI-328), a walk stays (AI-649), a flee and
// the retreat failure tail clear the order and the victim.
func TestARefusedRouteWithNoneInReachLeavesTheOutcomeOfItsWalkState(t *testing.T) {
	t.Parallel()
	for _, tc := range refusedRouteEntries {
		victim := EntityID(4)
		var retreat *RetreatContinuation
		switch tc.name {
		case "walk":
			victim = 0
		case "retreat failure":
			retreat = &RetreatContinuation{Known: true, Progress: 1, Counter: 2}
		}
		w, i := refusedRouteWalker(t, 20, victim, retreat)
		w.entities[i].TargetX, w.entities[i].TargetY, w.entities[i].HasTarget = 6, 4, true
		tc.call(w, i)
		got := w.entities[i]
		switch tc.name {
		case "pursuit":
			if !got.PursuitIdle || !got.HasAttackTarget || got.AttackTarget != 4 || got.AcquirePursuit {
				t.Errorf("%s: idle %v victim %v/%d acquisition %v; want idle with victim 4 kept", tc.name, got.PursuitIdle, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit)
			}
		case "walk":
			if got.PursuitIdle || got.HasAttackTarget || !got.HasTarget {
				t.Errorf("%s: idle %v victim %v destination %v; want the walker as it stood", tc.name, got.PursuitIdle, got.HasAttackTarget, got.HasTarget)
			}
		default:
			if got.PursuitIdle || got.HasAttackTarget || got.HasTarget {
				t.Errorf("%s: idle %v victim %v destination %v; want order and victim cleared", tc.name, got.PursuitIdle, got.HasAttackTarget, got.HasTarget)
			}
		}
		if retreat != nil && (got.Retreat.Progress != 1 || got.Retreat.Counter != 2 || got.Retreat.Failure) {
			t.Errorf("%s: retreat %+v, want progress 1 counter 2 kept and the flag consumed", tc.name, got.Retreat)
		}
	}
}

// A stone-cursed walker names the hostile within reach but its order is not
// written. The pursuit goes idle; the flee and the retreat failure tail keep
// the victim they held, and the tail still restarts the strike progress.
func TestARefusedRouteWhoseOrderIsNotWrittenIdlesOnlyAPursuit(t *testing.T) {
	t.Parallel()
	for _, tc := range refusedRouteEntries {
		if tc.name == "walk" {
			continue
		}
		var retreat *RetreatContinuation
		if tc.name == "retreat failure" {
			retreat = &RetreatContinuation{Known: true, Progress: 1, Counter: 2, Complete: true}
		}
		w, i := refusedRouteWalker(t, 5, 4, retreat)
		w.attached = []attachedEffect{{Target: 1, Spell: 20, Kind: EffectAbsorption, Mode: EffectDuration, Remaining: 20}}
		if !w.stoneCursed(i) {
			t.Fatal("fixture: the walker is not stone-cursed")
		}
		tc.call(w, i)
		got := w.entities[i]
		if !got.HasAttackTarget || got.AttackTarget != 4 {
			t.Errorf("%s: victim %v/%d, want victim 4 kept", tc.name, got.HasAttackTarget, got.AttackTarget)
		}
		if idle := tc.name == "pursuit"; got.PursuitIdle != idle {
			t.Errorf("%s: idle %v, want %v", tc.name, got.PursuitIdle, idle)
		}
		if retreat != nil && (got.Retreat.Progress != 1 || got.Retreat.Counter != 0 || got.Retreat.Complete) {
			t.Errorf("%s: retreat %+v, want progress 1, counter 0, completion cleared", tc.name, got.Retreat)
		}
	}
}

// Only the flee and the retreat failure tail restart a Retreat actor's strike
// progress on a pick, through the one reset writer: progress 1, counter 0,
// completion cleared (AI-365). A pursuit and a pickup walk in Retreat take the
// pick and leave the progress as it was.
func TestOnlyTheRetreatRowsRestartStrikeProgressOnAPick(t *testing.T) {
	t.Parallel()
	for _, tc := range refusedRouteEntries {
		victim := EntityID(0)
		if tc.name == "pursuit" {
			victim = 4
		}
		held := RetreatContinuation{Known: true, Progress: 1, Counter: 2, Complete: true}
		w, i := refusedRouteWalker(t, 5, victim, &held)
		if tc.name == "walk" {
			w.entities[i].PendingOrder = PendingOrder{Kind: PendingPickup}
		}
		tc.call(w, i)
		got := w.entities[i]
		if !got.HasAttackTarget || got.AttackTarget != 3 || !got.AcquirePursuit {
			t.Fatalf("%s: victim %v/%d acquisition %v, want the pick 3", tc.name, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit)
		}
		want := held
		if tc.name == "flee" || tc.name == "retreat failure" {
			want = RetreatContinuation{Known: true, Progress: 1}
		}
		if got.Retreat != want {
			t.Errorf("%s: retreat %+v, want %+v", tc.name, got.Retreat, want)
		}
	}
}

// The retreat failure tail keeps the action phase the new order reset
// (AI-365); a flee takes the order's ready phase.
func TestOnlyTheRetreatFailureTailKeepsThePhaseOnAPick(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"flee", "retreat failure"} {
		var call func(w *World, i int)
		for _, tc := range refusedRouteEntries {
			if tc.name == name {
				call = tc.call
			}
		}
		w, i := refusedRouteWalker(t, 5, 4, &RetreatContinuation{Known: true})
		w.entities[i].AttackPhase, w.entities[i].AttackCountdown = AttackBoundaryOne, 2
		call(w, i)
		got := w.entities[i]
		if !got.HasAttackTarget || got.AttackTarget != 3 {
			t.Fatalf("%s: victim %v/%d, want the pick 3", name, got.HasAttackTarget, got.AttackTarget)
		}
		phase, countdown := AttackReady, int32(0)
		if name == "retreat failure" {
			phase, countdown = AttackBoundaryOne, 2
		}
		if got.AttackPhase != phase || got.AttackCountdown != countdown {
			t.Errorf("%s: phase %v countdown %d, want %v %d", name, got.AttackPhase, got.AttackCountdown, phase, countdown)
		}
	}
}
