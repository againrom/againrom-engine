package sim

import "testing"

// standWorld is a warrior of the local participant and a hostile that never
// acts: the relation locks the hostile's side to not hostile, so the world holds
// only what a test orders.
func standWorld(t *testing.T, warrior, hostile Entity) *World {
	t.Helper()
	rel := engRel(t, [3]uint32{SelfSlot, 3, 1}, [3]uint32{3, SelfSlot, 2})
	return engWorld(t, rel, warrior, hostile)
}

// stepUntilTick advances w until its next Step would run tick n.
func stepUntilTick(w *World, n uint64) {
	for w.tick < n {
		Step(w, nil)
	}
}

// TestHoldPressedDuringAChaseEndsThePursuit: a member on its way to a victim out
// of reach holds the pending order 0 once Stand Ground is pressed, and the
// executor runs no arm for 0 (AI-351, AI-350). It stops and stays.
func TestHoldPressedDuringAChaseEndsThePursuit(t *testing.T) {
	t.Parallel()

	w := standWorld(t, swarmFighter(1, SelfSlot, 5, 10), swarmFighter(2, 3, 30, 10))
	Step(w, []Command{Attack(1, 2)})
	for range 4 {
		Step(w, nil)
	}
	going := entityAt(t, w, 1)
	if !going.HasAttackTarget || !going.HasTarget || going.X <= 5 {
		t.Fatalf("fixture: the warrior is not walking to its victim: %+v", going)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	held := entityAt(t, w, 1)
	if held.HasAttackTarget || held.HasTarget {
		t.Fatalf("after Hold the warrior still holds its order: victim %v, destination %v (%d,%d)",
			held.HasAttackTarget, held.HasTarget, held.TargetX, held.TargetY)
	}
	for n := 0; n < 200; n++ {
		Step(w, nil)
		e := entityAt(t, w, 1)
		if e.X != held.X || e.Y != held.Y || e.HasAttackTarget || e.HasTarget {
			t.Fatalf("tick %d after Hold the warrior is at (%d,%d), victim %v, destination %v; it stood at (%d,%d)",
				n, e.X, e.Y, e.HasAttackTarget, e.HasTarget, held.X, held.Y)
		}
	}
}

// TestHoldPressedDuringAWalkStopsTheWalk: a member walking to a cell holds the
// pending order 0 once Stand Ground is pressed (AI-351, AI-350).
func TestHoldPressedDuringAWalkStopsTheWalk(t *testing.T) {
	t.Parallel()

	w := standWorld(t, swarmFighter(1, SelfSlot, 5, 10), swarmFighter(2, 3, 30, 30))
	Step(w, []Command{MoveTo(1, CellPoint{X: 40, Y: 10})})
	for range 4 {
		Step(w, nil)
	}
	going := entityAt(t, w, 1)
	if !going.HasTarget || going.X <= 5 {
		t.Fatalf("fixture: the warrior is not walking: %+v", going)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	held := entityAt(t, w, 1)
	if held.HasTarget {
		t.Fatalf("after Hold the warrior still holds a destination (%d,%d)", held.TargetX, held.TargetY)
	}
	for n := 0; n < 200; n++ {
		Step(w, nil)
		if e := entityAt(t, w, 1); e.X != held.X || e.Y != held.Y || e.HasTarget {
			t.Fatalf("tick %d after Hold the warrior is at (%d,%d), destination %v; it stood at (%d,%d)",
				n, e.X, e.Y, e.HasTarget, held.X, held.Y)
		}
	}
}

// TestHoldPressedMidStepEndsTheStepAtTheNextCellCentre: the setters store no
// progress, so a step under way is paid out and the member stands on the cell it
// entered (AI-352).
func TestHoldPressedMidStepEndsTheStepAtTheNextCellCentre(t *testing.T) {
	t.Parallel()

	walker := swarmFighter(1, SelfSlot, 5, 10)
	walker.Speed = 32 // eight ticks to a cell
	w := standWorld(t, walker, swarmFighter(2, 3, 30, 30))
	Step(w, []Command{MoveTo(1, CellPoint{X: 40, Y: 10})})
	Step(w, nil)
	Step(w, nil)
	crossing := entityAt(t, w, 1)
	if crossing.Transit < 3 || !crossing.HasTarget {
		t.Fatalf("fixture: the warrior is not mid-step: %+v", crossing)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	held := entityAt(t, w, 1)
	if held.HasTarget || held.Transit != crossing.Transit-1 || held.X != crossing.X || held.Y != crossing.Y {
		t.Fatalf("Hold mid-step: destination %v, transit %d (was %d), at (%d,%d) (was (%d,%d)); the step must be paid out",
			held.HasTarget, held.Transit, crossing.Transit, held.X, held.Y, crossing.X, crossing.Y)
	}
	for n := 0; n < 100; n++ {
		Step(w, nil)
		if e := entityAt(t, w, 1); e.X != held.X || e.Y != held.Y || e.HasTarget {
			t.Fatalf("tick %d after Hold the warrior left the cell it had entered: (%d,%d), destination %v", n, e.X, e.Y, e.HasTarget)
		}
	}
	if e := entityAt(t, w, 1); e.Transit != 0 {
		t.Errorf("the step was not paid out: transit %d", e.Transit)
	}
}

// TestHoldPressedMidCycleLetsTheLoadedCycleEndAndIdlesUntilTheNextDecision: a
// loaded attack cycle completes and no new cycle loads, since the setters store
// the pending order 0 and no progress (AI-352, AI-350). The next decision takes
// what stands in reach again (AI-353). Whether the blow of the loaded cycle
// lands is not established, so the test bounds the blows and does not count one.
func TestHoldPressedMidCycleLetsTheLoadedCycleEndAndIdlesUntilTheNextDecision(t *testing.T) {
	t.Parallel()

	warrior := swarmFighter(1, SelfSlot, 5, 10)
	warrior.AttackCharge, warrior.AttackRelax = 3, 1
	w := standWorld(t, warrior, swarmFighter(2, 3, 6, 10))
	engRun(w, 1)
	loaded := entityAt(t, w, 1)
	if !loaded.HasAttackTarget || loaded.AttackPhase != AttackCharging {
		t.Fatalf("fixture: the first decision left the warrior at phase %v, victim %v", loaded.AttackPhase, loaded.HasAttackTarget)
	}
	hp := entityAt(t, w, 2).HP
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	blows := 0
	count := func() {
		if now := entityAt(t, w, 2).HP; now < hp {
			blows++
			hp = now
		}
	}
	count()
	for w.tick < scriptPassPhase+scriptCycle {
		Step(w, nil)
		count()
	}
	t.Logf("%d blow(s) landed between the press and the next decision", blows)
	if blows > 1 {
		t.Errorf("between the press and the next decision %d blows landed; only the loaded cycle may complete", blows)
	}
	if e := entityAt(t, w, 1); e.HasAttackTarget || e.HasTarget {
		t.Errorf("before the next decision the warrior holds victim %v, destination %v; the order ends with the cycle", e.HasAttackTarget, e.HasTarget)
	}
	Step(w, nil)
	if e := entityAt(t, w, 1); !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Errorf("the next decision did not take the hostile in reach again: victim %v/%v", e.AttackTarget, e.HasAttackTarget)
	}
}

// TestHoldPressedInsideALoadedCycleIsAnsweredByTheNextEvaluation: an ordered
// victim in reach is scored again once Hold is pressed, so a decision inside the
// cycle issues the attack again and the fight goes on without a gap (AI-353,
// AI-CMD-054).
func TestHoldPressedInsideALoadedCycleIsAnsweredByTheNextEvaluation(t *testing.T) {
	t.Parallel()

	warrior := swarmFighter(1, SelfSlot, 5, 10)
	warrior.AttackCharge, warrior.AttackRelax = scriptCycle*2, 1
	w := standWorld(t, warrior, swarmFighter(2, 3, 6, 10))
	Step(w, []Command{Attack(1, 2)})
	for range 4 {
		Step(w, nil)
	}
	if e := entityAt(t, w, 1); e.AttackPhase != AttackCharging {
		t.Fatalf("fixture: the ordered attack is at phase %v", e.AttackPhase)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	stepUntilTick(w, scriptPassPhase+1)
	e := entityAt(t, w, 1)
	if !e.HasAttackTarget || e.AttackTarget != 2 || e.HasTarget {
		t.Errorf("the decision inside the loaded cycle left victim %v/%v, destination %v; the victim stands in reach",
			e.AttackTarget, e.HasAttackTarget, e.HasTarget)
	}
	if e.AttackPhase == AttackReady {
		t.Errorf("the cycle was cancelled: phase %v", e.AttackPhase)
	}
}

// TestStandGroundMemberDropsAVictimBeyondSightAtTheNextDecision: the arm hands a
// participant's member that scores nothing to the routine that stores 0 (AI-349,
// AI-353), so a pursuit ends at the next decision instead of following the victim
// until it dies.
func TestStandGroundMemberDropsAVictimBeyondSightAtTheNextDecision(t *testing.T) {
	t.Parallel()

	slow := swarmFighter(1, SelfSlot, 5, 10)
	slow.Speed = 16
	w := standWorld(t, slow, swarmFighter(2, 3, 6, 10))
	engRun(w, 1)
	if v, held := engVictim(w, 1); !held || v != 2 {
		t.Fatalf("fixture: the first decision left victim %v/%v", v, held)
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 40, Y: 10})})
	stepUntilTick(w, scriptPassPhase+scriptCycle)
	before := entityAt(t, w, 1)
	if !before.HasAttackTarget || before.X <= 5 {
		t.Fatalf("fixture: between decisions the member does not follow its victim: %+v", before)
	}
	Step(w, nil)
	after := entityAt(t, w, 1)
	if after.HasAttackTarget || after.HasTarget {
		t.Errorf("at the next decision the victim is out of sight and the member still holds victim %v, destination %v",
			after.HasAttackTarget, after.HasTarget)
	}
	for range 200 {
		Step(w, nil)
	}
	final := entityAt(t, w, 1)
	if final.HasAttackTarget || final.X > before.X+1 {
		t.Errorf("the member kept chasing: at (%d,%d) with victim %v, it stood at (%d,%d) when the decision came",
			final.X, final.Y, final.HasAttackTarget, before.X, before.Y)
	}
}

// TestStandGroundMemberDropsAVictimPastReachAtTheNextDecision is the same for a
// victim the group still sees: the scorer refuses every candidate past reach
// (AI-REACH-072), so the arm stores 0 for the member (AI-349, AI-353).
func TestStandGroundMemberDropsAVictimPastReachAtTheNextDecision(t *testing.T) {
	t.Parallel()

	slow := swarmFighter(1, SelfSlot, 5, 10)
	slow.Speed = 16
	w := standWorld(t, slow, swarmFighter(2, 3, 6, 10))
	engRun(w, 1)
	Step(w, []Command{MoveTo(2, CellPoint{X: 10, Y: 10})})
	stepUntilTick(w, scriptPassPhase+scriptCycle)
	before := entityAt(t, w, 1)
	victim := entityAt(t, w, 2)
	if !before.HasAttackTarget || before.X <= 5 || cellOf(before).chebyshevTo(cellOf(victim)) < 3 {
		t.Fatalf("fixture: member %+v, victim at (%d,%d)", before, victim.X, victim.Y)
	}
	Step(w, nil)
	if after := entityAt(t, w, 1); after.HasAttackTarget || after.HasTarget {
		t.Errorf("at the next decision the victim stands past reach and the member still holds victim %v, destination %v",
			after.HasAttackTarget, after.HasTarget)
	}
	for range 200 {
		Step(w, nil)
	}
	final := entityAt(t, w, 1)
	if final.HasAttackTarget || final.X > before.X+1 {
		t.Errorf("the member approached a hostile that stood still past reach: (%d,%d) with victim %v, it stood at (%d,%d)",
			final.X, final.Y, final.HasAttackTarget, before.X, before.Y)
	}
}

// TestHoldNeverApproachesAHostileBeyondReach is a control: what the claims
// already decide is unchanged. A member holding its ground takes only what
// stands next to it, however near the hostile stands inside sight
// (AI-STAND-076, AI-REACH-072).
func TestHoldNeverApproachesAHostileBeyondReach(t *testing.T) {
	t.Parallel()

	w := standWorld(t, swarmFighter(1, SelfSlot, 5, 10), swarmFighter(2, 3, 8, 10))
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	for range 200 {
		Step(w, nil)
		if e := entityAt(t, w, 1); e.X != 5 || e.Y != 10 || e.HasAttackTarget || e.HasTarget {
			t.Fatalf("a member holding its ground moved or took a victim: %+v", e)
		}
	}
}

// TestHoldPressedAgainstAHostileInReachKeepsTheFight is a control: a member
// striking a hostile that stands in reach when Hold is pressed takes it again at
// the next evaluation and the fight goes on (AI-353).
func TestHoldPressedAgainstAHostileInReachKeepsTheFight(t *testing.T) {
	t.Parallel()

	w := standWorld(t, swarmFighter(1, SelfSlot, 5, 10), swarmFighter(2, 3, 6, 10))
	Step(w, []Command{Attack(1, 2)})
	for range 20 {
		Step(w, nil)
	}
	before := entityAt(t, w, 2).HP
	if before >= entityAt(t, w, 2).MaxHP {
		t.Fatalf("fixture: no blow landed in 20 ticks")
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	for range 200 {
		Step(w, nil)
	}
	if after := entityAt(t, w, 2).HP; after >= before-3 {
		t.Errorf("the fight stopped at Hold: the hostile stands at %d hit points, it had %d", after, before)
	}
}

// standSavedWorld is standWorld with the warrior's group carried as a SAV
// dispatcher group at Stand Ground, so the record mirror can be read.
func standSavedWorld(t *testing.T, warrior, hostile Entity) *World {
	t.Helper()
	w := standWorld(t, warrior, hostile)
	g := SavedGroup{ID: 71, Selector: 19}
	g.Members = []SavedGroupMember{{Entity: warrior.ID, Bound: true}}
	g.AI[0x20], g.AI[0x45], g.AI[0x38] = orderStandGround, 1, 9
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{{Entity: warrior.ID}}); err != nil {
		t.Fatalf("ImportSavedGroups: %v", err)
	}
	return w
}

// TestHoldPressedInASavedWorldStoresPendingOrderZeroAndReach: the setter stores
// ord+0x08 = 0 and ord+0x14 = the member's reach in the SAV dispatcher record
// (AI-351), where the member held a pending pursuit before.
func TestHoldPressedInASavedWorldStoresPendingOrderZeroAndReach(t *testing.T) {
	t.Parallel()

	warrior := swarmFighter(1, SelfSlot, 5, 10)
	warrior.Reach = 3
	w := standSavedWorld(t, warrior, swarmFighter(2, 3, 30, 10))
	Step(w, []Command{Attack(1, 2)})
	for range 4 {
		Step(w, nil)
	}
	Step(w, []Command{GroupStance(1, OrderStandGround, 1)})
	o := w.savedOrder(1)
	if o == nil || o.Raw[8] != 0 || o.Raw[0x14] != 3 {
		t.Fatalf("after Hold the record holds %+v, want ord+0x08 0 and ord+0x14 3", o)
	}
	if e := entityAt(t, w, 1); e.HasAttackTarget || e.HasTarget {
		t.Errorf("after Hold the warrior holds victim %v, destination %v", e.HasAttackTarget, e.HasTarget)
	}
}

// TestSavedStandGroundMemberIdlesWithPendingOrderZero: a participant's member
// that scores nothing idles with 0, not the idle turn an AI owner's member takes
// (AI-349, AI-353), and the pursuit ends.
func TestSavedStandGroundMemberIdlesWithPendingOrderZero(t *testing.T) {
	t.Parallel()

	slow := swarmFighter(1, SelfSlot, 5, 10)
	slow.Speed = 16
	w := standSavedWorld(t, slow, swarmFighter(2, 3, 6, 10))
	engRun(w, 1)
	if o := w.savedOrder(1); o == nil || o.Raw[8] != 5 {
		t.Fatalf("fixture: the first decision left the record %+v, want ord+0x08 5", o)
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 40, Y: 10})})
	stepUntilTick(w, scriptPassPhase+scriptCycle+1)
	if o := w.savedOrder(1); o == nil || o.Raw[8] != 0 {
		t.Errorf("after the victim left, the record holds %+v, want ord+0x08 0", o)
	}
	if e := entityAt(t, w, 1); e.HasAttackTarget || e.HasTarget {
		t.Errorf("the member still holds victim %v, destination %v", e.HasAttackTarget, e.HasTarget)
	}
}

// walkingMember is a participant's member the map or a script sent to a cell:
// it holds a destination and no victim, and no player's command grouped it.
func walkingMember() Entity {
	e := swarmFighter(1, SelfSlot, 5, 10)
	e.Speed = 16
	e.TargetX, e.TargetY, e.HasTarget = 40, 10, true
	return e
}

// TestSavedStandGroundDecisionLeavesAWalkingMemberAlone: a participant's member
// walking to a cell keeps its walk through the decisions of its saved group at
// Stand Ground, as it does in a native group, where a member that holds a victim
// it cannot score is stood down (AI-349).
func TestSavedStandGroundDecisionLeavesAWalkingMemberAlone(t *testing.T) {
	t.Parallel()

	w := standSavedWorld(t, walkingMember(), swarmFighter(2, 3, 30, 30))
	stepUntilTick(w, scriptPassPhase+3*scriptCycle+1)
	if e := entityAt(t, w, 1); !e.HasTarget || e.TargetX != 40 || e.TargetY != 10 {
		t.Errorf("three decisions on, the member holds destination %v (%d,%d) at (%d,%d), want its walk to (40,10) kept",
			e.HasTarget, e.TargetX, e.TargetY, e.X, e.Y)
	}
}

// TestStandGroundDecisionLeavesAWalkingMemberAlone is the same in a native group.
func TestStandGroundDecisionLeavesAWalkingMemberAlone(t *testing.T) {
	t.Parallel()

	w := standWorld(t, walkingMember(), swarmFighter(2, 3, 30, 30))
	stepUntilTick(w, scriptPassPhase+3*scriptCycle+1)
	if e := entityAt(t, w, 1); !e.HasTarget || e.TargetX != 40 || e.TargetY != 10 {
		t.Errorf("three decisions on, the member holds destination %v (%d,%d) at (%d,%d), want its walk to (40,10) kept",
			e.HasTarget, e.TargetX, e.TargetY, e.X, e.Y)
	}
}

// TestStandDownOnAMemberHoldingNothingLeavesItsLoadedRouteRecordAlone: a member
// that holds neither a victim nor a destination already holds the pending order
// 0, so standing it down writes nothing. Clearing an order that is not there
// would mark the route continuation loaded with it as superseded.
func TestStandDownOnAMemberHoldingNothingLeavesItsLoadedRouteRecordAlone(t *testing.T) {
	t.Parallel()

	actor := Entity{ID: 41, X: 4, Y: 5, HP: 30, MaxHP: 30}
	w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{actor})
	if err != nil {
		t.Fatal(err)
	}
	motion := SavedActorMotion{Entity: actor.ID, Position: SavedActorPosition{
		Cell: 0x0504, PackedCell: 0x0504, FineX: 128, FineY: 128, Residue: 0x136a, TerrainKey: 0x56473829,
	}}
	motion.Mover[10], motion.Mover[31] = 18, 0xa7
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	before, _, _, _ := w.SavedActorMotions()
	if e := w.Entities()[0]; e.HasTarget || e.HasAttackTarget || w.motionActive(e.ID) || !before[0].Current {
		t.Fatalf("fixture: the member holds a destination %v or victim %v, or its record is active %v or not current %v",
			e.HasTarget, e.HasAttackTarget, w.motionActive(e.ID), before[0].Current)
	}
	w.standDown(0)
	after, _, _, _ := w.SavedActorMotions()
	if !after[0].Current || after[0].Issue != before[0].Issue {
		t.Errorf("standing down a member that holds nothing left its record current %v with issue %q, want current and %q",
			after[0].Current, after[0].Issue, before[0].Issue)
	}
}
