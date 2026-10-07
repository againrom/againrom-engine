package sim

import "testing"

// pocketWorld stands unit at (20,20) beside a walled pocket that holds its
// decoded flee cell for a hostile to the east, or at whatever cell the caller
// places it. The cells within two of (17,20) are all closed, so the far search
// from (20,20) labels nothing near the flee cell and settles on the unit's own
// cell: it comes back with no route.
func pocketWorld(t *testing.T, unit, intruder Entity, rel Relations) *World {
	t.Helper()
	grid := make([]byte, engBounds.Width*engBounds.Height)
	for y := int32(18); y <= 22; y++ {
		for x := int32(15); x <= 19; x++ {
			grid[y*engBounds.Width+x] = blockGround
		}
	}
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: grid},
		[]Entity{unit, intruder}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	return w
}

// sturdy gives e health enough to outlast a long fight, so a count of blows read
// from its health is never cut short by its death.
func sturdy(e Entity, hp int32) Entity {
	e.HP, e.MaxHP = hp, hp
	return e
}

const sturdyHP = 1000

func blowsOn(w *World, victim int) int32 { return (sturdyHP - w.entities[victim].HP) / 5 }

// TestExplicitRetreatWhoseFleeCellHasNoRouteKeepsItsPickAcrossCycles is a
// retreating unit with a wall at its back and a hostile beside it. The pursuit
// order the reacquisition writes has nothing in it that measures elapsed time or
// ends with a cycle (AI-PURSUE-040, AI-BREAK-041), and the state arm rewrites it
// only at its next dispatch, so the unit strikes again as soon as each cycle
// returns to ready: it never holds no order while the hostile stays beside it,
// and it lands more blows than one per decision period.
func TestExplicitRetreatWhoseFleeCellHasNoRouteKeepsItsPickAcrossCycles(t *testing.T) {
	const span = 320
	w := pocketWorld(t, retreatFighter(1, 20, 20), sturdy(retreatIntruder(2, 21, 20), sturdyHP), retreatRelations(t))
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	idle := stepIdle(t, w, span, 20, 20)
	if idle != 0 {
		t.Errorf("unit held no order for %d consecutive ticks beside its hostile, want none", idle)
	}
	if blows, periods := blowsOn(w, 1), int32(span/scriptCycle); blows <= periods {
		t.Errorf("unit landed %d blow(s) in %d ticks, want more than the %d decision periods it spans", blows, span, periods)
	}
}

// TestExplicitRetreatCorneredAtThePlayableEdgeStandsAfterTheCentredShortcut is
// the same unit against the west edge of the playable rectangle: the decoded
// flee cell clamps onto its own cell. A request equal to the cell of a centred
// mover returns before any search and raises no failure (MOVE-081, MOVE-083),
// so no pick is taken and the unit stands on its cell. DIV-1639.
func TestExplicitRetreatCorneredAtThePlayableEdgeStandsAfterTheCentredShortcut(t *testing.T) {
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{},
		[]Entity{retreatFighter(1, 8, 20), sturdy(retreatIntruder(2, 9, 20), sturdyHP)}, nil, retreatRelations(t))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	if e := w.entities[0]; e.X != 8 || e.Y != 20 || e.HasTarget || e.HasAttackTarget {
		t.Fatalf("unit = (%d,%d) move %t attack %t, want standing on (8,20) with neither", e.X, e.Y, e.HasTarget, e.HasAttackTarget)
	}
}

// TestExplicitRetreatWhoseFleeCellStaysServedEndsTheOrderWithTheCycle is the
// control for the two above: where a route serves the flee cell the unit's next
// decision is a walk, so the retained order of a cycle loaded before the
// decision ends with the cycle and the unit strikes once before it leaves.
func TestExplicitRetreatWhoseFleeCellStaysServedEndsTheOrderWithTheCycle(t *testing.T) {
	w := engWorld(t, retreatRelations(t), retreatFighter(1, 20, 20), sturdy(retreatIntruder(2, 21, 20), sturdyHP))
	w.orderAttack(0, 2)
	for n := 0; n < 4 && w.entities[0].AttackPhase == AttackReady; n++ {
		Step(w, nil)
	}
	if w.entities[0].AttackPhase == AttackReady {
		t.Fatal("fixture did not load the cycle")
	}
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	for n := 0; n < 80; n++ {
		Step(w, nil)
	}
	if blows := blowsOn(w, 1); blows != 1 {
		t.Errorf("unit landed %d blow(s), want the one it had loaded", blows)
	}
	if e := w.entities[0]; e.X >= 20 {
		t.Errorf("unit stands at (%d,%d), want it walked west of (20,20)", e.X, e.Y)
	}
}

// mageOf marks e as a spellcaster, the class bit whose short reach a human
// participant's unit does not fight with on its own (AI-GUARD-007).
func mageOf(e Entity) Entity {
	e.MaxMana, e.Mana = 10, 10
	return e
}

// TestExplicitRetreatOfAHumanMageOfReachOneTakesNoPick is the suppression clause
// both reacquisition and engagement carry: a human participant's mage of reach
// below two does not take the pick, so the refused flee leaves it standing with
// no order (AI-GUARD-007, AI-328).
func TestExplicitRetreatOfAHumanMageOfReachOneTakesNoPick(t *testing.T) {
	w := pocketWorld(t, mageOf(retreatFighter(1, 20, 20)), sturdy(retreatIntruder(2, 21, 20), sturdyHP), retreatRelations(t))
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	for n := 0; n < 96; n++ {
		Step(w, nil)
		if e := w.entities[0]; e.HasAttackTarget || e.HasTarget || e.X != 20 || e.Y != 20 {
			t.Fatalf("tick %d: mage at (%d,%d) holds attack %t destination %t, want it standing with no order",
				w.tick, e.X, e.Y, e.HasAttackTarget, e.HasTarget)
		}
	}
	if blows := blowsOn(w, 1); blows != 0 {
		t.Errorf("mage landed %d blow(s), want none", blows)
	}
}

// TestExplicitRetreatOfAHumanMageOfReachTwoTakesThePick is the control: the
// clause tests reach below two.
func TestExplicitRetreatOfAHumanMageOfReachTwoTakesThePick(t *testing.T) {
	mage := mageOf(retreatFighter(1, 20, 20))
	mage.Reach = 2
	w := pocketWorld(t, mage, sturdy(retreatIntruder(2, 21, 20), sturdyHP), retreatRelations(t))
	w.tick = scriptPassPhase
	Step(w, []Command{GroupRetreat(1, SelfSlot, 91)})
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("mage of reach two holds attack %t on %d, want its victim 2", e.HasAttackTarget, e.AttackTarget)
	}
	stepIdle(t, w, 96, 20, 20)
	if blows := blowsOn(w, 1); blows < 3 {
		t.Errorf("mage landed %d blow(s), want at least 3", blows)
	}
}

// TestWithdrawalOfAHumanMageOfReachOneTakesNoPick is the same clause on the
// automatic arm: the fixed-radius withdrawal writes a flee cell nothing leads to
// and the unit is left with no order.
func TestWithdrawalOfAHumanMageOfReachOneTakesNoPick(t *testing.T) {
	mage := mageOf(withdrawalFighter(1, SelfSlot, 20, 20, 30))
	mage.Withdraw = 30
	w := pocketWorld(t, mage, sturdy(withdrawalFighter(2, 2, 21, 20, sturdyHP), sturdyHP),
		engRel(t, [3]uint32{SelfSlot, 2, 1}, [3]uint32{2, SelfSlot, 2}))
	for n := 0; n < 96; n++ {
		Step(w, nil)
		if e := w.entities[0]; e.HasAttackTarget || e.HasTarget || e.X != 20 || e.Y != 20 {
			t.Fatalf("tick %d: mage at (%d,%d) holds attack %t destination %t, want it standing with no order",
				w.tick, e.X, e.Y, e.HasAttackTarget, e.HasTarget)
		}
	}
}

// TestWithdrawalOfAMapMageOfReachOneTakesThePick is the control: the clause
// names a human participant, so a map-owned mage of reach one fights on.
func TestWithdrawalOfAMapMageOfReachOneTakesThePick(t *testing.T) {
	mage := mageOf(withdrawalFighter(1, 2, 20, 20, 30))
	mage.Withdraw = 30
	w := pocketWorld(t, mage, sturdy(withdrawalFighter(2, SelfSlot, 21, 20, sturdyHP), sturdyHP),
		engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}))
	engRun(w, 1)
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatalf("map mage holds attack %t on %d, want its victim 2", e.HasAttackTarget, e.AttackTarget)
	}
}

// TestWithdrawalWhoseFleeCellHasNoRouteLeavesAFlierAloneBesideAGroundUnit: the
// reacquisition's candidates carry no domain veto (AI-327), and the owner rule
// refuses a melee unit every flier, so a ground unit of reach one beside one
// after a refused flee takes no victim (DIV-2372).
func TestWithdrawalWhoseFleeCellHasNoRouteLeavesAFlierAloneBesideAGroundUnit(t *testing.T) {
	self := withdrawalFighter(1, 2, 20, 20, 30)
	self.Withdraw = 30
	flier := sturdy(withdrawalFighter(2, SelfSlot, 21, 20, sturdyHP), sturdyHP)
	flier.Domain = DomainAir
	w := pocketWorld(t, self, flier, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}))
	engRun(w, 1)
	if e := w.entities[0]; e.HasAttackTarget || e.HasTarget {
		t.Fatalf("unit after the refused flee holds attack %t on %d with destination %t, want neither",
			e.HasAttackTarget, e.AttackTarget, e.HasTarget)
	}
	for n := 0; n < 96; n++ {
		Step(w, nil)
	}
	if blows := (sturdyHP - w.entities[1].HP); blows != 0 {
		t.Errorf("flier lost %d health in 96 ticks beside the ground unit, want none", blows)
	}
}
