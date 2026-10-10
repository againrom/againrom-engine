package sim

import "testing"

// withdrawalArcher is a ranged unit whose health always satisfies its
// Withdraw threshold, with a charge and relax long enough for one attack
// cycle to span several full ticks. Its blow always lands, so the victim's
// health says whether the cycle ever resolved.
func withdrawalArcher(id EntityID, owner uint32, x, y int32) Entity {
	e := withdrawalFighter(id, owner, x, y, 30)
	e.MaxHP, e.Withdraw = 30, 30
	e.Reach, e.ScanRange, e.DamageBase = 4, 6, 5
	e.AttackCharge, e.AttackRelax = 40, 30
	return e
}

// archerAndIntruder is the ordinary route to a loaded cycle: the group
// decision engages a hostile four cells away, inside the archer's reach and
// outside the radius-2 withdrawal block, and the archer loads its charge. The
// intruder is then walked to two cells by a player move, so the next full tick
// finds the archer mid-cycle with a hostile inside the block.
func archerAndIntruder(t *testing.T) *World {
	t.Helper()
	return archerAndIntruderIn(t, DomainGround)
}

func archerAndIntruderIn(t *testing.T, domain Domain) *World {
	t.Helper()
	archer := withdrawalArcher(1, 2, 20, 20)
	archer.Domain = domain
	intruder := withdrawalFighter(2, SelfSlot, 24, 20, 100)
	rel := engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2})
	w := engWorld(t, rel, archer, intruder)
	engRun(w, 1)
	got := w.entities[0]
	if !got.HasAttackTarget || got.AttackPhase != AttackCharging || got.HasTarget {
		t.Fatalf("archer after the first decision = attack %t phase %d destination %t, want a loaded charge",
			got.HasAttackTarget, got.AttackPhase, got.HasTarget)
	}
	Step(w, []Command{MoveTo(2, CellPoint{X: 22, Y: 20})})
	return w
}

// stepToFullTick advances w until the next full-tick boundary has run.
func stepToFullTick(w *World) {
	for i := 0; i < scriptCycle; i++ {
		Step(w, nil)
		if (w.tick-1)%scriptCycle == scriptPassPhase {
			return
		}
	}
}

// TestWithdrawalLetsALoadedAttackCycleFinishBeforeTheWalkStarts is the
// ordinary-play shape of a ranged creature meeting a hostile at two cells:
// the wind-up already loaded resolves its blow, the archer stands through the
// recovery, and only then does the decoded flee move start (AI-WITHDRAW-028,
// AI-RETREAT-272, AI-ORDER-039, HERO-CADENCE-112).
func TestWithdrawalLetsALoadedAttackCycleFinishBeforeTheWalkStarts(t *testing.T) {
	w := archerAndIntruder(t)
	stepToFullTick(w)

	archer := w.entities[0]
	if archer.X != 20 || archer.Y != 20 {
		t.Fatalf("archer walked on the tick the tail decided: (%d,%d), want it standing at (20,20)", archer.X, archer.Y)
	}
	if !archer.HasAttackTarget || archer.AttackPhase == AttackReady {
		t.Fatalf("tail discarded the loaded cycle: attack %t phase %d countdown %d",
			archer.HasAttackTarget, archer.AttackPhase, archer.AttackCountdown)
	}
	if !archer.HasTarget || archer.TargetX != 17 || archer.TargetY != 20 {
		t.Fatalf("flee destination = (%d,%d,%t), want the decoded cell (17,20,true) held for the end of the cycle",
			archer.TargetX, archer.TargetY, archer.HasTarget)
	}

	victimHP := w.entities[1].HP
	blowTick, leftTick := uint64(0), uint64(0)
	for i := 0; i < 200 && leftTick == 0; i++ {
		Step(w, nil)
		if blowTick == 0 && w.entities[1].HP < victimHP {
			blowTick = w.tick
			if archer := w.entities[0]; archer.X != 20 || archer.Y != 20 {
				t.Fatalf("archer left its cell before the blow landed: (%d,%d)", archer.X, archer.Y)
			}
		}
		if got := w.entities[0]; got.X != 20 || got.Y != 20 {
			leftTick = w.tick
			if got.HasAttackTarget || got.AttackPhase != AttackReady {
				t.Errorf("archer walked while still holding its attack order: attack %t phase %d",
					got.HasAttackTarget, got.AttackPhase)
			}
		}
	}
	if blowTick == 0 {
		t.Fatal("the loaded cycle never resolved its blow")
	}
	if leftTick == 0 {
		t.Fatal("the archer never started the flee after its cycle ended")
	}
	if leftTick <= blowTick {
		t.Errorf("flee walk began at tick %d, not after the blow at tick %d", leftTick, blowTick)
	}
	if got := w.entities[1].HP; got != victimHP-5 {
		t.Errorf("victim health = %d, want exactly one blow of 5 from %d (no second cycle before the flee)", got, victimHP)
	}
}

// TestWithdrawalDeferredFleeYieldsToTheNextGroupDecision covers the other end
// of the deferral: the pending flee move is only an ordinary pending order, so
// the group decision on the next full tick replaces it, and an intruder that
// has left the block by then leaves the archer to its own attack.
func TestWithdrawalDeferredFleeYieldsToTheNextGroupDecision(t *testing.T) {
	w := archerAndIntruder(t)
	stepToFullTick(w)
	if got := w.entities[0]; !got.HasTarget || !got.HasAttackTarget {
		t.Fatalf("no deferred flee to replace: attack %t destination %t", got.HasAttackTarget, got.HasTarget)
	}

	Step(w, []Command{MoveTo(2, CellPoint{X: 26, Y: 20})})
	for i := 0; i < scriptCycle; i++ {
		Step(w, nil)
		if (w.tick-1)%scriptCycle == scriptPassPhase {
			break
		}
	}
	got := w.entities[0]
	if got.HasTarget {
		t.Fatalf("stale flee destination (%d,%d) survived a full tick with the intruder outside the block", got.TargetX, got.TargetY)
	}
	if !got.HasAttackTarget {
		t.Fatal("archer lost its attack order although nothing withdrew it")
	}
	if got.X != 20 || got.Y != 20 {
		t.Errorf("archer moved to (%d,%d) with no flee pending", got.X, got.Y)
	}
}

// TestWithdrawalAtReadyStillReplacesTheOrderImmediately is the control: between
// cycles nothing is owed, so the decoded move replaces the pending attack order
// and the walk starts on the tail's own tick.
func TestWithdrawalAtReadyStillReplacesTheOrderImmediately(t *testing.T) {
	archer := withdrawalArcher(1, 2, 20, 20)
	intruder := withdrawalFighter(2, SelfSlot, 22, 20, 100)
	rel := engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2})
	w := engWorld(t, rel, archer, intruder)
	engRun(w, 1)
	got := w.entities[0]
	if got.HasAttackTarget || got.AttackPhase != AttackReady {
		t.Fatalf("archer between cycles kept an attack order: attack %t phase %d", got.HasAttackTarget, got.AttackPhase)
	}
	if !got.HasTarget || got.TargetX != 17 || got.TargetY != 20 || got.X != 19 {
		t.Fatalf("archer after the tail = (%d,%d) destination (%d,%d,%t), want the first flee step to (19,20)",
			got.X, got.Y, got.TargetX, got.TargetY, got.HasTarget)
	}
}

// TestWithdrawalDeferredFleeKeepsAStandingFlyerInThePlane: a flyer is counted
// in the occupancy plane only while it holds no destination, because a moving
// flyer crosses its peers. One standing through a loaded cycle with the move
// held behind it is not moving, so it stays counted until the walk starts.
func TestWithdrawalDeferredFleeKeepsAStandingFlyerInThePlane(t *testing.T) {
	w := archerAndIntruderIn(t, DomainAir)
	stepToFullTick(w)
	held := w.entities[0]
	if !held.HasAttackTarget || held.AttackPhase == AttackReady || !held.HasTarget {
		t.Fatalf("flyer state at the tail = attack %t phase %d destination %t, want a held move behind a loaded cycle",
			held.HasAttackTarget, held.AttackPhase, held.HasTarget)
	}
	if !counted(&held) {
		t.Error("a flyer standing through its cycle left the occupancy plane")
	}
	walked := false
	for i := 0; i < 200 && !walked; i++ {
		Step(w, nil)
		if got := w.entities[0]; got.X != 20 || got.Y != 20 {
			walked = true
			if counted(&got) {
				t.Error("a flyer walking its flee move is still counted")
			}
		}
	}
	if !walked {
		t.Fatal("the flyer never started its flee after the cycle")
	}
}

// TestWithdrawalHeldMoveBehindACycleSurvivesTheByteFormAndResumesIdentically:
// the held move is attack fields plus destination fields, both already in the
// byte form, so a world encoded mid-cycle decodes and then advances exactly as
// the original does through the blow, the recovery and the first flee steps.
func TestWithdrawalHeldMoveBehindACycleSurvivesTheByteFormAndResumesIdentically(t *testing.T) {
	w := archerAndIntruder(t)
	stepToFullTick(w)
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
	if got := w.entities[0]; got.X == 20 && got.Y == 20 {
		t.Error("archer never left its cell in the resumed span, so the comparison did not cross the flee start")
	}
}
