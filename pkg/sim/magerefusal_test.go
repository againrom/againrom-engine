package sim

import "testing"

// mrWalker is refusedRouteWalker's walker made a human participant's mage of
// the given reach and mana pool, with the hostile 3 beside it at (3, 5).
func mrWalker(t *testing.T, name string, reach uint8, maxMana int32, second bool) (*World, int) {
	t.Helper()
	victim := EntityID(4)
	var retreat *RetreatContinuation
	switch name {
	case "walk":
		victim = 0
	case "retreat failure":
		retreat = &RetreatContinuation{Known: true, Progress: 1, Counter: 2}
	}
	w, i := refusedRouteWalker(t, 3, victim, retreat)
	installAcquireControl(t, w, acquireControlCase{owner: 2, present: true, human: true})
	w.entities[i].Reach, w.entities[i].MaxMana, w.entities[i].Mana = reach, maxMana, maxMana
	if name == "retreat failure" {
		w.entities[i].AttackPhase, w.entities[i].AttackCountdown = AttackBoundaryOne, 2
	}
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	return w, i
}

// A first-game suppressed acquirer's refused route writes the pick and idles
// the order in every walk state: the victim is the hostile within reach, the
// order byte is 0, and neither the strike progress nor the action phase is
// written (AI-GUARD-007, AI-542).
func TestASuppressedMageRefusedRouteWritesThePickAndIdlesTheOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range refusedRouteEntries {
		w, i := mrWalker(t, tc.name, 1, 10, false)
		if !w.suppressedAcquirer(w.entities[i]) {
			t.Fatal("fixture: the walker is not a suppressed acquirer")
		}
		o := w.savedOrder(w.entities[i].ID)
		if o == nil {
			t.Fatal("fixture: the walker has no order block")
		}
		o.Raw[8] = 5
		tc.call(w, i)
		got := w.entities[i]
		if !got.HasAttackTarget || got.AttackTarget != 3 || !got.PursuitIdle || got.AcquirePursuit {
			t.Fatalf("%s: victim %v/%d idle %v acquisition %v; want the pick 3 held with the order idle",
				tc.name, got.HasAttackTarget, got.AttackTarget, got.PursuitIdle, got.AcquirePursuit)
		}
		if o := w.savedOrder(got.ID); o == nil || o.Raw[8] != 0 {
			t.Fatalf("%s: the order byte is not 0", tc.name)
		}
		if tc.name == "retreat failure" {
			if got.Retreat.Progress != 1 || got.Retreat.Counter != 2 {
				t.Errorf("%s: retreat %+v, want progress 1 counter 2 kept", tc.name, got.Retreat)
			}
			if got.AttackPhase != AttackBoundaryOne || got.AttackCountdown != 2 {
				t.Errorf("%s: phase %v countdown %d, want the phase kept", tc.name, got.AttackPhase, got.AttackCountdown)
			}
		}
		hp := laEnt(t, w, 3).HP
		for k := 0; k < 120; k++ {
			Step(w, nil)
		}
		if laEnt(t, w, 3).HP != hp {
			t.Errorf("%s: the mage struck its idled pick", tc.name)
		}
	}
}

// The withdrawal's refused flee idles the pick; the World byte form carries
// it and the cold continuation equals the live one.
func TestAnIdledPickOfARefusedWithdrawalSurvivesTheWorldForm(t *testing.T) {
	t.Parallel()
	mage := mageOf(withdrawalFighter(1, SelfSlot, 20, 20, 30))
	mage.Withdraw = 30
	w := pocketWorld(t, mage, sturdy(withdrawalFighter(2, 2, 21, 20, sturdyHP), sturdyHP),
		engRel(t, [3]uint32{SelfSlot, 2, 1}, [3]uint32{2, SelfSlot, 2}))
	for k := 0; k < 96 && !w.entities[0].PursuitIdle; k++ {
		Step(w, nil)
	}
	if e := w.entities[0]; !e.PursuitIdle || e.AttackTarget != 2 {
		t.Fatal("fixture: the refused withdrawal never idled its pick")
	}
	if o := w.savedOrder(1); o == nil || o.Raw[8] != 0 {
		t.Fatal("the idled pick's order byte is not 0")
	}
	back := worldRoundTripForTest(t, w)
	for k := 0; k < 120; k++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("tick %d: the cold continuation of the idled pick differs", k)
		}
	}
}

// Controls: a mage of reach 2 and a unit with no mana pool take the pick as
// an acquisition; the second game keeps its refusal rows.
func TestARefusedRouteAnswerOutsideTheSuppressionIsUnchanged(t *testing.T) {
	t.Parallel()
	for _, tc := range refusedRouteEntries {
		for _, c := range []struct {
			what  string
			reach uint8
			mana  int32
		}{{"mage of reach 2", 2, 10}, {"no mana pool", 1, 0}} {
			w, i := mrWalker(t, tc.name, c.reach, c.mana, false)
			tc.call(w, i)
			if got := w.entities[i]; !got.HasAttackTarget || got.AttackTarget != 3 || !got.AcquirePursuit || got.PursuitIdle {
				t.Errorf("%s, %s: victim %v/%d acquisition %v idle %v; want the pick 3 acquired",
					tc.name, c.what, got.HasAttackTarget, got.AttackTarget, got.AcquirePursuit, got.PursuitIdle)
			}
		}
		w, i := mrWalker(t, tc.name, 1, 10, true)
		tc.call(w, i)
		got := w.entities[i]
		switch tc.name {
		case "pursuit":
			if !got.PursuitIdle || got.AttackTarget != 4 {
				t.Errorf("second game %s: idle %v victim %d; want the old victim 4 idle", tc.name, got.PursuitIdle, got.AttackTarget)
			}
		default:
			if got.HasAttackTarget {
				t.Errorf("second game %s: victim %d; want none", tc.name, got.AttackTarget)
			}
		}
	}
}

// mrDefend is a unit 1 of owner 3, neutral to everyone, at (16, 16); a
// player's mage 2 of the given reach and mana pool at (16, 17) defending it;
// and a passive hostile 3 of owner 2 at (17, 18), beside the mage but not
// hostile to the defended unit, so the cover block is empty and the
// defender's own acquisition takes the hostile.
func mrDefend(t *testing.T, reach uint8, maxMana int32) *World {
	t.Helper()
	subject := engFighter(1, 3, 16, 16)
	mage := engFighter(2, SelfSlot, 16, 17)
	mage.Reach, mage.MaxMana, mage.Mana, mage.Speed = reach, maxMana, maxMana, 10
	foe := sturdy(engFighter(3, 2, 17, 18), sturdyHP)
	foe.DamageBase = 0
	w := engWorld(t, engRel(t, [3]uint32{SelfSlot, 2, 1}, [3]uint32{2, SelfSlot, 2}), subject, mage, foe)
	Step(w, []Command{GroupDefend(2, 1, 77)})
	return w
}

// A staff-less defender whose acquisition idles its pick has the order 0,
// which falls to the crowding check: standing beside the defended unit it
// steps away instead of standing there holding the pick (AI-FOLLOWTAB-113,
// AI-FOLLOWGAP-114). It never strikes.
func TestAStafflessDefenderWithAnIdledPickStepsAwayFromItsSubject(t *testing.T) {
	t.Parallel()
	w := mrDefend(t, 1, 10)
	left := false
	for k := 0; k < 200; k++ {
		Step(w, nil)
		m := laEnt(t, w, 2)
		if m.HasAttackTarget && !m.PursuitIdle || m.AttackPhase != AttackReady {
			t.Fatalf("tick %d: the staff-less defender engaged", w.Tick())
		}
		if cellOf(&m).chebyshevTo(cellOf(&w.entities[indexOfEntity(w.entities, 1)])) >= escortCrowd {
			left = true
		}
	}
	if !left {
		t.Fatal("the staff-less defender never stepped away from the unit it defends")
	}
	if laEnt(t, w, 3).HP != sturdyHP {
		t.Error("the staff-less defender struck the hostile beside it")
	}
	if e := laEnt(t, w, 2); e.ActorState != actorStateDefend {
		t.Errorf("the defender left Defend: state %d", e.ActorState)
	}
}

// Controls: a defender of reach 2 and one with no mana pool engage the
// hostile beside them, which keeps them busy, and strike it.
func TestADefenderOutsideTheSuppressionEngagesItsPick(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		what  string
		reach uint8
		mana  int32
	}{{"mage of reach 2", 2, 10}, {"no mana pool", 1, 0}} {
		w := mrDefend(t, c.reach, c.mana)
		for k := 0; k < 200; k++ {
			Step(w, nil)
		}
		if laEnt(t, w, 3).HP == sturdyHP {
			t.Errorf("%s: the defender never struck the hostile beside it", c.what)
		}
	}
}

// mrIdle is a player's unit 1 at (10, 10) and an AI unit 2 at (30, 30) that
// see nothing hostile.
func mrIdle(t *testing.T, second bool) *World {
	t.Helper()
	w := engWorld(t, engRel(t), engFighter(1, SelfSlot, 10, 10), engFighter(2, 2, 30, 30))
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	return w
}

// The acquisition with no target sends a human participant's unit to the heal
// arm, which leaves the order 0 when it casts nothing; an AI owner's unit gets
// the guard order 0xb (AI-ACQUIRE-002, AI-349).
func TestAnAcquisitionWithNoTargetLeavesAHumanUnitsOrderZero(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		what   string
		id     EntityID
		second bool
		want   uint8
	}{{"player's unit", 1, false, 0}, {"AI unit", 2, false, 0xb}, {"second-game player's unit", 1, true, 0xb}} {
		w := mrIdle(t, c.second)
		i := indexOfEntity(w.entities, c.id)
		o := &SavedActorOrder{Entity: c.id, State: uint32(actorStateAcquire)}
		o.Raw[8] = 5
		w.savedAcquire(i, o)
		if e := w.entities[i]; e.HasAttackTarget || o.Raw[8] != c.want {
			t.Errorf("%s: victim %v order byte %#x, want none and %#x", c.what, e.HasAttackTarget, o.Raw[8], c.want)
		}
	}
	w := mrIdle(t, false)
	Step(w, nil)
	if o := w.savedOrder(1); o == nil || o.Raw[8] != 0 {
		t.Fatal("after a step the idle player's unit does not hold order byte 0")
	}
	back := worldRoundTripForTest(t, w)
	for k := 0; k < 60; k++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("tick %d: the cold continuation differs", k)
		}
	}
}
