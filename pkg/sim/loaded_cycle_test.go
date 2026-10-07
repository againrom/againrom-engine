package sim

import "testing"

// Fixtures load cycles through commands and ordinary ticks. A locked ally
// relation keeps the victims from fighting or walking. Setter retention is
// conditional on other active-field writers leaving the cycle alone (AI-355).

const (
	lcArcherID = EntityID(1)
	lcVictimID = EntityID(2)
	lcBlow     = int32(5)
)

// lcArcher is a ranged unit whose charge and relax are long enough for one
// cycle to span several full ticks. Its blow always lands.
func lcArcher(id EntityID, owner uint32, x, y int32) Entity {
	e := withdrawalFighter(id, owner, x, y, 100)
	e.Reach, e.ScanRange, e.DamageBase = 4, 6, lcBlow
	e.AttackCharge, e.AttackRelax = 40, 30
	return e
}

// lcVictim is a target that outlasts any number of blows.
func lcVictim(id EntityID, owner uint32, x, y int32) Entity {
	e := withdrawalFighter(id, owner, x, y, 200)
	e.MaxHP = 200
	return e
}

// lcRequireLoaded fails unless id stands with a charge loaded on victim and no
// destination.
func lcRequireLoaded(t *testing.T, w *World, id, victim EntityID) {
	t.Helper()
	e := laEnt(t, w, id)
	if !e.HasAttackTarget || e.AttackTarget != victim || e.AttackPhase != AttackCharging || e.HasTarget {
		t.Fatalf("fixture: entity %d holds attack %t victim %d phase %d destination %t, want a charge loaded on %d",
			id, e.HasAttackTarget, e.AttackTarget, e.AttackPhase, e.HasTarget, victim)
	}
}

// lcPlayerWorld is the ordinary route to a loaded cycle on the participant's
// own side: the archer is ordered onto a hostile four cells away, inside its
// reach, and loads its charge on that tick. extra entities stand beside them.
func lcPlayerWorld(t *testing.T, extra ...Entity) *World {
	t.Helper()
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	ents := append([]Entity{lcArcher(lcArcherID, SelfSlot, 20, 20), lcVictim(lcVictimID, 2, 24, 20)}, extra...)
	w := engWorld(t, rel, ents...)
	Step(w, []Command{Attack(lcArcherID, lcVictimID)})
	lcRequireLoaded(t, w, lcArcherID, lcVictimID)
	return w
}

// lcHold fails unless id still holds its loaded cycle on victim with the
// destination (dx,dy) stored beside it and its body at (x,y).
func lcHold(t *testing.T, w *World, id, victim EntityID, x, y, dx, dy int32) {
	t.Helper()
	e := laEnt(t, w, id)
	if !e.HasAttackTarget || e.AttackTarget != victim || e.AttackPhase == AttackReady {
		t.Fatalf("the order discarded the loaded cycle: attack %t victim %d phase %d countdown %d",
			e.HasAttackTarget, e.AttackTarget, e.AttackPhase, e.AttackCountdown)
	}
	if !e.HasTarget || e.TargetX != dx || e.TargetY != dy {
		t.Fatalf("destination = (%d,%d,%t), want (%d,%d,true) held for the end of the cycle", e.TargetX, e.TargetY, e.HasTarget, dx, dy)
	}
	if e.X != x || e.Y != y {
		t.Fatalf("entity %d stands at (%d,%d) with its cycle loaded, want (%d,%d)", id, e.X, e.Y, x, y)
	}
}

// lcBlowFirst advances w until victim takes its first blow, which must land
// while id still stands where it was, and returns the tick it landed on.
func lcBlowFirst(t *testing.T, w *World, id, victim EntityID) uint64 {
	t.Helper()
	home := laEnt(t, w, id)
	hp := laEnt(t, w, victim).HP
	for i := 0; i < 240; i++ {
		Step(w, nil)
		got := laEnt(t, w, id)
		if laEnt(t, w, victim).HP < hp {
			if got.X != home.X || got.Y != home.Y {
				t.Fatalf("entity %d left (%d,%d) before its blow landed at tick %d", id, home.X, home.Y, w.tick)
			}
			if lost := hp - laEnt(t, w, victim).HP; lost != lcBlow {
				t.Fatalf("the blow took %d health, want %d", lost, lcBlow)
			}
			return w.tick
		}
		if got.X != home.X || got.Y != home.Y {
			t.Fatalf("entity %d left (%d,%d) at tick %d with no blow landed", id, home.X, home.Y, w.tick)
		}
	}
	t.Fatal("the loaded cycle never resolved its blow")
	return 0
}

// lcResolve advances w until id leaves its cell and checks what the held order
// is owed: the blow lands on victim first, while id stands; the walk starts
// after the cycle has ended; and no second cycle loads before the walk.
func lcResolve(t *testing.T, w *World, id, victim EntityID) {
	t.Helper()
	home := laEnt(t, w, id)
	hp := laEnt(t, w, victim).HP
	blow := lcBlowFirst(t, w, id, victim)
	var left uint64
	for i := 0; i < 240 && left == 0; i++ {
		Step(w, nil)
		if got := laEnt(t, w, id); got.X != home.X || got.Y != home.Y {
			left = w.tick
			if got.HasAttackTarget || got.AttackPhase != AttackReady {
				t.Errorf("entity %d walked while still holding its cycle: attack %t phase %d", id, got.HasAttackTarget, got.AttackPhase)
			}
		}
	}
	if left == 0 {
		t.Fatal("the held destination was never walked after the cycle ended")
	}
	if left <= blow {
		t.Errorf("the walk began at tick %d, not after the blow at tick %d", left, blow)
	}
	if got := laEnt(t, w, victim).HP; got != hp-lcBlow {
		t.Errorf("victim health = %d, want exactly one blow of %d from %d", got, lcBlow, hp)
	}
}

// TestAMoveOrderWaitsBehindALoadedAttackCycle covers the player's move, the
// group move every right-click queues, and the Swarm 2 order: each writes a
// destination and no progress (AI-CMD-033, MOVE-FORM-036, MOVE-GATE-035,
// AI-RETREAT-275), so a charge already loaded resolves its blow first. The
// Swarm 2 member goes on fighting what its group sees once the cycle has
// ended, as an unloaded member does, so only its blow is followed.
func TestAMoveOrderWaitsBehindALoadedAttackCycle(t *testing.T) {
	dest := CellPoint{X: 17, Y: 20}
	for _, tc := range []struct {
		name  string
		order Command
		walks bool
	}{
		{"plain move", MoveTo(lcArcherID, dest), true},
		{"group move", GroupMoveTo(lcArcherID, dest, 1), true},
		{"swarm 2 move", GroupSwarmTo(lcArcherID, dest, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := lcPlayerWorld(t)
			Step(w, []Command{tc.order})
			lcHold(t, w, lcArcherID, lcVictimID, 20, 20, dest.X, dest.Y)
			if tc.walks {
				lcResolve(t, w, lcArcherID, lcVictimID)
			} else {
				lcBlowFirst(t, w, lcArcherID, lcVictimID)
			}
		})
	}
}

// TestAGroupMoveKeepsEachMembersLoadedCycleAndMovesTheRest sends a loaded
// archer and an idle companion in one formation order: the companion walks at
// once and the archer's blow lands before it follows.
func TestAGroupMoveKeepsEachMembersLoadedCycleAndMovesTheRest(t *testing.T) {
	companion := lcArcher(3, SelfSlot, 20, 22)
	w := lcPlayerWorld(t, companion)
	dest := CellPoint{X: 17, Y: 21}
	Step(w, []Command{
		GroupMoveTo(lcArcherID, dest, 1),
		GroupMoveTo(3, dest, 1),
	})
	held := laEnt(t, w, lcArcherID)
	if !held.HasAttackTarget || held.AttackPhase == AttackReady || !held.HasTarget || held.X != 20 || held.Y != 20 {
		t.Fatalf("the loaded member after the order: attack %t phase %d destination %t at (%d,%d)",
			held.HasAttackTarget, held.AttackPhase, held.HasTarget, held.X, held.Y)
	}
	for i := 0; i < 4; i++ {
		Step(w, nil)
	}
	if got := laEnt(t, w, 3); got.X == 20 && got.Y == 22 {
		t.Error("the idle companion did not start walking with the group order")
	}
	lcResolve(t, w, lcArcherID, lcVictimID)
}

// TestAMoveOrderBetweenCyclesStillReplacesTheAttackAtOnce is the control: an
// archer still approaching its victim owes no cycle, so each move order takes
// its attack order away on the tick it is given.
func TestAMoveOrderBetweenCyclesStillReplacesTheAttackAtOnce(t *testing.T) {
	dest := CellPoint{X: 17, Y: 20}
	for _, tc := range []struct {
		name  string
		order Command
	}{
		{"plain move", MoveTo(lcArcherID, dest)},
		{"group move", GroupMoveTo(lcArcherID, dest, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
			w := engWorld(t, rel, lcArcher(lcArcherID, SelfSlot, 20, 20), lcVictim(lcVictimID, 2, 32, 20))
			Step(w, []Command{Attack(lcArcherID, lcVictimID)})
			if e := laEnt(t, w, lcArcherID); !e.HasAttackTarget || e.AttackPhase != AttackReady {
				t.Fatalf("fixture: attack %t phase %d, want an archer still approaching", e.HasAttackTarget, e.AttackPhase)
			}
			Step(w, []Command{tc.order})
			got := laEnt(t, w, lcArcherID)
			if got.HasAttackTarget || got.AttackPhase != AttackReady {
				t.Errorf("the move left attack %t phase %d, want the attack order gone", got.HasAttackTarget, got.AttackPhase)
			}
			if !got.HasTarget || got.TargetX != dest.X || got.TargetY != dest.Y {
				t.Errorf("destination = (%d,%d,%t), want (%d,%d,true)", got.TargetX, got.TargetY, got.HasTarget, dest.X, dest.Y)
			}
		})
	}
}

// TestAMoveHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically:
// the held move is attack fields plus destination fields, both in the byte
// form already, so a world encoded the tick the order arrives decodes and then
// advances exactly as the original does through the blow, the recovery and the
// walk.
func TestAMoveHeldBehindALoadedCycleSurvivesTheByteFormAndResumesIdentically(t *testing.T) {
	w := lcPlayerWorld(t)
	Step(w, []Command{MoveTo(lcArcherID, CellPoint{X: 17, Y: 20})})
	lcHold(t, w, lcArcherID, lcVictimID, 20, 20, 17, 20)
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal a world holding a move behind a loaded cycle: %v", err)
	}
	var copyOf World
	if err := copyOf.UnmarshalBinary(form); err != nil {
		t.Fatalf("decode a world holding a move behind a loaded cycle: %v", err)
	}
	if w.Hash() != copyOf.Hash() {
		t.Fatalf("decoded hash %#x differs from the live hash %#x", copyOf.Hash(), w.Hash())
	}
	for i := 0; i < 150; i++ {
		Step(w, nil)
		Step(&copyOf, nil)
		if w.Hash() != copyOf.Hash() {
			t.Fatalf("tick %d: decoded world diverged: live %#x decoded %#x", w.tick, w.Hash(), copyOf.Hash())
		}
	}
	if got := laEnt(t, w, lcArcherID); got.X == 20 && got.Y == 20 {
		t.Error("the archer never left its cell in the resumed span, so the comparison did not cross the walk")
	}
}

// lcGuardWorld is a ranged guard under group order 0, the one population the
// per-actor guard arm decides, standing three cells off its post with a hostile
// inside the post's five-cell block.
func lcGuardWorld(t *testing.T) *World {
	t.Helper()
	archer := lcArcher(lcArcherID, 2, 20, 20)
	archer.Group = 1
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, 2})
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{archer, lcVictim(lcVictimID, 3, 22, 20)}, nil, rel)
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	for gi := range w.groups {
		if w.groups[gi].owner == 2 && w.groups[gi].group == 1 {
			w.groups[gi].order = orderNone
		}
	}
	i := indexOfEntity(w.entities, lcArcherID)
	w.entities[i].PostX, w.entities[i].PostY = 17, 20
	return w
}

// TestAGuardBreakingOffToWalkHomeWaitsBehindALoadedAttackCycle: the guard
// engages a hostile inside its post's block and loads a charge; the hostile
// then steps out of the block. The arm's walk home is a pending order written
// over the pursuit (AI-BREAK-041, AI-GUARD-012), so the charge resolves before
// the guard turns for its post.
func TestAGuardBreakingOffToWalkHomeWaitsBehindALoadedAttackCycle(t *testing.T) {
	w := lcGuardWorld(t)
	engRun(w, 1)
	lcRequireLoaded(t, w, lcArcherID, lcVictimID)
	Step(w, []Command{MoveTo(lcVictimID, CellPoint{X: 23, Y: 20})})
	stepToFullTick(w)
	lcHold(t, w, lcArcherID, lcVictimID, 20, 20, 17, 20)
	lcResolve(t, w, lcArcherID, lcVictimID)
}

// lcEscortWorld is an archer escorting the unit at (sx,sy) under sub, with a
// hostile in reach: the archer is put on it by the arm's own attack order and
// loads its charge on the first tick.
func lcEscortWorld(t *testing.T, sub int32, sx, sy int32) *World {
	t.Helper()
	const escort, foe = EntityID(2), EntityID(3)
	archer := lcArcher(escort, 2, 20, 20)
	archer.Group = 7
	hostile := lcVictim(foe, 3, 24, 20)
	hostile.Group = 9
	rel := engRel(t, [3]uint32{2, 3, relationHostile}, [3]uint32{3, 2, 2})
	w := esWorld(t, sub, 3, rel, laFighter(1, 2, 7, sx, sy), archer, hostile)
	w.orderAttack(indexOfEntity(w.entities, escort), foe)
	Step(w, nil)
	lcRequireLoaded(t, w, escort, foe)
	return w
}

// TestAnEscortClosingOnItsSubjectWaitsBehindALoadedAttackCycle: an escort out
// of range of its subject closes on it, which the original writes as a pending
// order (AI-DEFEND-111, AI-FOLLOW-112), so the charge it holds resolves first.
func TestAnEscortClosingOnItsSubjectWaitsBehindALoadedAttackCycle(t *testing.T) {
	for _, tc := range []struct {
		name string
		sub  int32
	}{{"defend", subCommandDefend}, {"follow", subCommandFollow}} {
		t.Run(tc.name, func(t *testing.T) {
			w := lcEscortWorld(t, tc.sub, 5, 5)
			stepToFullTick(w)
			lcHold(t, w, 2, 3, 20, 20, 5, 5)
			lcResolve(t, w, 2, 3)
		})
	}
}

// TestAFollowerSteppingAwayFromItsSubjectWaitsBehindALoadedAttackCycle is the
// crowding half: a follower within two cells of the unit it follows steps out
// to the stop distance (AI-FOLLOWGAP-114), after its loaded charge resolves.
func TestAFollowerSteppingAwayFromItsSubjectWaitsBehindALoadedAttackCycle(t *testing.T) {
	w := lcEscortWorld(t, subCommandFollow, 19, 20)
	stepToFullTick(w)
	lcHold(t, w, 2, 3, 20, 20, 22, 20)
	lcResolve(t, w, 2, 3)
}

func TestAnAttackOrderOnAnotherVictimKeepsTheLoadedCycle(t *testing.T) {
	const second = EntityID(3)
	w := lcPlayerWorld(t, lcVictim(second, 2, 24, 21))
	for i := 0; i < 30; i++ {
		Step(w, nil)
	}
	if e := laEnt(t, w, lcArcherID); e.AttackPhase != AttackCharging {
		t.Fatalf("fixture: phase %d after 30 ticks, want the charge still loading", e.AttackPhase)
	}
	firstHP, secondHP := laEnt(t, w, lcVictimID).HP, laEnt(t, w, second).HP
	control := worldRoundTripForTest(t, w)
	Step(control, nil)
	Step(w, []Command{Attack(lcArcherID, second)})
	before := laEnt(t, control, lcArcherID)
	if e := laEnt(t, w, lcArcherID); !e.HasAttackTarget || e.AttackTarget != lcVictimID || e.AttackPhase != before.AttackPhase || e.AttackCountdown != before.AttackCountdown {
		t.Fatalf("command replaced the active cycle: target=%d phase=%d countdown=%d; control=%d/%d/%d", e.AttackTarget, e.AttackPhase, e.AttackCountdown, before.AttackTarget, before.AttackPhase, before.AttackCountdown)
	}
	var first, next uint64
	seenRecovery, seenOne, seenTwo := false, false, false
	for i := 0; i < 240 && next == 0; i++ {
		Step(w, nil)
		e := laEnt(t, w, lcArcherID)
		if first == 0 && laEnt(t, w, lcVictimID).HP < firstHP {
			first = w.tick
		}
		if first != 0 && next == 0 {
			seenRecovery = seenRecovery || e.AttackPhase == AttackRelaxing
			seenOne = seenOne || e.AttackPhase == AttackBoundaryOne
			seenTwo = seenTwo || e.AttackPhase == AttackBoundaryTwo
		}
		if laEnt(t, w, second).HP < secondHP {
			next = w.tick
		}
	}
	if first == 0 || next <= first || !seenRecovery || !seenOne || !seenTwo || firstHP-laEnt(t, w, lcVictimID).HP != lcBlow {
		t.Fatalf("handoff: old=%d next=%d recovery=%v boundaries=%v/%v old damage=%d", first, next, seenRecovery, seenOne, seenTwo, firstHP-laEnt(t, w, lcVictimID).HP)
	}
}
