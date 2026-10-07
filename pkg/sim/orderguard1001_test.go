package sim

import "testing"

// ogGroupMove is one member's share of a group move command carrying tag.
func ogGroupMove(id EntityID, x, y int32, tag uint32) Command {
	return Command{Kind: KindGroupMoveTo, Entity: id, X: x, Y: y, Group: tag}
}

// ogArrow is the damaging row these fixtures cast: cheap, unit-targeted, and
// with reach enough that no case turns on range.
func ogArrow() SpellRule {
	return SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 12,
		DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
}

// ogDestinations is every named entity's destination, in the order asked.
func ogDestinations(t *testing.T, w *World, ids ...EntityID) [][3]int32 {
	t.Helper()
	out := make([][3]int32, 0, len(ids))
	for _, id := range ids {
		e := cbAt(t, w, id)
		held := int32(0)
		if e.HasTarget {
			held = 1
		}
		out = append(out, [3]int32{held, e.TargetX, e.TargetY})
	}
	return out
}

// TestAGroupMoveTakesEveryAddressedMemberAndKeepsTheCentroid is A1's group
// half. Three members are addressed and one of them is mid-fight. Every one of
// them must take a destination, and the destinations must be the ones the same
// order produces when nobody is fighting: the centroid, the formation offsets
// and the rate term are all computed over the ADDRESSED set, so dropping the
// fighting member both loses that member and moves the other two. The fighting
// member's cycle is loaded, so it keeps its victim and its destination waits
// behind the cycle (DIV-1563); the two idle members hold no victim.
func TestAGroupMoveTakesEveryAddressedMemberAndKeepsTheCentroid(t *testing.T) {
	build := func(t *testing.T, fight bool) *World {
		t.Helper()
		m1, m2 := cbEnt(1, 2, 2), cbEnt(2, 2, 4)
		m3 := cbFighter(3, 2, 6, 30, 0)
		victim := cbEnt(4, 2, 7)
		w := cbWorld(t, 7, m1, m2, m3, victim)
		if fight {
			Step(w, []Command{cbOrder(3, 4)})
		} else {
			Step(w, nil)
		}
		Step(w, nil)
		return w
	}

	fighting := build(t, true)
	if e := cbAt(t, fighting, 3); e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
		t.Fatalf("the fixture produced no loaded attack cycle on member 3: %+v", e)
	}

	control := build(t, false)
	for _, w := range []*World{fighting, control} {
		Step(w, []Command{
			ogGroupMove(1, 10, 10, 1),
			ogGroupMove(2, 10, 10, 1),
			ogGroupMove(3, 10, 10, 1),
		})
	}

	for _, id := range []EntityID{1, 2, 3} {
		e := cbAt(t, fighting, id)
		if !e.HasTarget {
			t.Errorf("member %d took no destination from the group move it was addressed by", id)
		}
		if loaded := id == 3; e.HasAttackTarget != loaded {
			t.Errorf("member %d holds a victim after the group move: %t, want %t", id, e.HasAttackTarget, loaded)
		}
	}

	got := ogDestinations(t, fighting, 1, 2, 3)
	want := ogDestinations(t, control, 1, 2, 3)
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("the fighting member changed the order's own distribution: got %v, want %v — "+
			"the centroid and the formation are the addressed set's, not the idle subset's", got, want)
	}
}

// TestGroupStanceAndPatrolReachAFightingMember is the same membership rule at
// the other two group kinds. Guard, Stand Ground and Patrol were dropped on
// exactly the actors a player wants to plant: the ones already fighting.
func TestGroupStanceAndPatrolReachAFightingMember(t *testing.T) {
	build := func(t *testing.T) *World {
		t.Helper()
		member := cbFighter(1, 2, 6, 30, 0)
		member.Owner = 9
		w := cbWorld(t, 7, member, cbEnt(2, 2, 7))
		Step(w, []Command{cbOrder(1, 2)})
		Step(w, nil)
		if e := cbAt(t, w, 1); e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
			t.Fatalf("the fixture produced no loaded attack cycle: %+v", e)
		}
		return w
	}

	t.Run("stance", func(t *testing.T) {
		w := build(t)
		Step(w, []Command{{Kind: KindGroupStance, Entity: 1, X: int32(OrderGuard), Group: 1}})
		e := cbAt(t, w, 1)
		if e.PostX != e.X || e.PostY != e.Y {
			t.Errorf("a fighting member's post is (%d,%d) and it stands at (%d,%d) — the stance "+
				"order did not reach it", e.PostX, e.PostY, e.X, e.Y)
		}
		if e.CommandGroup == 0 {
			t.Error("a fighting member was not put into the stance order's own fresh command group")
		}
	})

	t.Run("patrol", func(t *testing.T) {
		w := build(t)
		Step(w, []Command{{Kind: KindGroupPatrolTo, Entity: 1, X: 9, Y: 9, Group: 1}})
		e := cbAt(t, w, 1)
		if e.PatrolTailX != 9 || e.PatrolTailY != 9 {
			t.Errorf("a fighting member's patrol ring ends at (%d,%d), want the ordered (9,9)",
				e.PatrolTailX, e.PatrolTailY)
		}
	})
}

// TestAScriptedGroupMarchTakesItsFightingMembers is the shipped-script half.
// cmdGroupCommandedMove is script group sub-commands 4 and 5, and it hands the
// group's living members to the same distribution the player's group move uses,
// so a trigger that marches a group left its fighting members standing. The
// fighting member's cycle is loaded, so it keeps its victim and its destination
// waits behind the cycle (DIV-1563).
func TestAScriptedGroupMarchTakesItsFightingMembers(t *testing.T) {
	m1 := cbEnt(1, 2, 2)
	m1.Owner = 9
	m2 := cbFighter(2, 2, 6, 30, 0)
	m2.Owner = 9
	victim := cbEnt(3, 2, 7)
	w := cbWorld(t, 7, m1, m2, victim)
	Step(w, []Command{cbOrder(2, 3)})
	Step(w, nil)
	if e := cbAt(t, w, 2); e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
		t.Fatalf("the fixture produced no loaded attack cycle on member 2: %+v", e)
	}

	w.cmdGroupCommandedMove(0, orderMove, 12, 12)

	for _, id := range []EntityID{1, 2} {
		if e := cbAt(t, w, id); !e.HasTarget {
			t.Errorf("member %d took no destination from the scripted march: %+v", id, e)
		}
	}
	if e := cbAt(t, w, 2); !e.HasAttackTarget || e.AttackTarget != 3 {
		t.Errorf("the scripted march took member 2's loaded cycle away: victim %t/%d", e.HasAttackTarget, e.AttackTarget)
	}
}

// TestACastOwnsItsActorWhileAMoveOrderStillLands is the contention half, and it
// states this round's decision about a cast in progress: the cast is NOT
// interruptible and the move order is NOT dropped. The order attaches on the
// tick it arrives, the body does not advance until the cast releases, and the
// walk begins then.
func TestACastOwnsItsActorWhileAMoveOrderStillLands(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, victim)

	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 {
		t.Fatalf("the fixture admitted no cast: %d pending", len(w.bookCasts))
	}

	// An attack order is taken beside the cast and does not cancel it: the
	// victim is attached now and the fight begins once the cast is over.
	Step(w, []Command{cbOrder(1, 2)})
	if e := spAt(t, w, 1); !e.HasAttackTarget || e.AttackTarget != 2 || len(w.bookCasts) != 1 {
		t.Errorf("an attack order beside a pending cast: victim %v/%d, %d casts pending",
			e.HasAttackTarget, e.AttackTarget, len(w.bookCasts))
	}
	// So does a second cast, and it spends nothing.
	manaBefore := spAt(t, w, 1).Mana
	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 || spAt(t, w, 1).Mana != manaBefore {
		t.Errorf("a second cast was admitted beside the first: %d pending, mana %d want %d",
			len(w.bookCasts), spAt(t, w, 1).Mana, manaBefore)
	}

	// The movement order is taken whole.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 6}})
	e := spAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 0 || e.TargetY != 6 {
		t.Fatalf("the move order was dropped by the pending cast: held=%v (%d,%d)",
			e.HasTarget, e.TargetX, e.TargetY)
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the move order cancelled the pending cast: %d pending", len(w.bookCasts))
	}
	if e.X != 0 || e.Y != 0 {
		t.Errorf("the caster advanced to (%d,%d) while its cast still stood", e.X, e.Y)
	}

	for n := 0; n < 16 && len(w.bookCasts) != 0; n++ {
		Step(w, nil)
	}
	if spAt(t, w, 2).HP >= 100 {
		t.Error("the cast never released")
	}
	for n := 0; n < 16 && spAt(t, w, 1).Y != 6; n++ {
		Step(w, nil)
	}
	if e := spAt(t, w, 1); e.X != 0 || e.Y != 6 {
		t.Errorf("the caster is at (%d,%d) after its cast released, want the move order's own (0,6)",
			e.X, e.Y)
	}
}

// TestAnAttackCycleRefusesAnAutomaticCastAndAdmitsAnotherAttackOrder is the
// non-preempting AI admission path.
func TestAnAttackCycleRefusesAnAutomaticCastAndAdmitsAnotherAttackOrder(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	caster.AttackCharge, caster.DamageBase, caster.AlwaysHits = 30, 5, true
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, spEnt(2, 1, 0), spEnt(3, 0, 1))

	Step(w, []Command{cbOrder(1, 2)})
	Step(w, nil)
	if e := spAt(t, w, 1); e.AttackPhase == AttackReady && e.AttackCountdown == 0 {
		t.Fatalf("the fixture produced no loaded attack cycle: %+v", e)
	}

	manaBefore := spAt(t, w, 1).Mana
	if w.beginBookSpell(0, 3, 1) {
		t.Fatal("automatic cast preempted an attack")
	}
	Step(w, nil)
	if len(w.bookCasts) != 0 || spAt(t, w, 1).Mana != manaBefore {
		t.Errorf("a cast overlapped a loaded attack cycle: %d pending, mana %d want %d",
			len(w.bookCasts), spAt(t, w, 1).Mana, manaBefore)
	}

	control := worldRoundTripForTest(t, w)
	Step(control, nil)
	Step(w, []Command{cbOrder(1, 3)})
	e := spAt(t, w, 1)
	if !e.HasAttackTarget || e.AttackTarget != 2 || !e.HasPendingAttackTarget || e.PendingAttackTarget != 3 {
		t.Error("a fighting unit lost its active victim or refused its requested victim")
	}
	if c := spAt(t, control, 1); e.AttackPhase != c.AttackPhase || e.AttackCountdown != c.AttackCountdown {
		t.Error("the explicit retarget reset the loaded cycle")
	}
}

// TestTheAIArmsCannotInterruptACastTheyDoNotOwn is G3. armSwarm reaches
// orderAttack and escortClose writes a destination directly; neither may take a
// cast away from its actor. The destination write is allowed, because it is the
// same write a player's move order performs and it does not cancel the cast.
func TestTheAIArmsCannotInterruptACastTheyDoNotOwn(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, spEnt(2, 3, 0), spEnt(3, 4, 0))
	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 {
		t.Fatalf("the fixture admitted no cast: %d pending", len(w.bookCasts))
	}

	w.orderAttack(0, 3)
	if e := spAt(t, w, 1); e.HasAttackTarget {
		t.Errorf("the AI's own retarget took a casting actor: victim %d", e.AttackTarget)
	}

	w.escortClose(0, 2)
	if len(w.bookCasts) != 1 {
		t.Fatalf("the escort arm cancelled a pending cast: %d pending", len(w.bookCasts))
	}
	for n := 0; n < 16 && len(w.bookCasts) != 0; n++ {
		Step(w, nil)
	}
	if spAt(t, w, 2).HP >= 100 {
		t.Error("a cast the escort arm walked away from never released")
	}
}

func TestAnArmedAutocastReachesAMovingActorOnTheTickItIsOrdered(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, 1<<1, 1)
	caster.AttackCharge, caster.AttackRelax = 12, 4
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	w := hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 2, Y: 9}})

	e := spAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 2 || e.TargetY != 9 {
		t.Fatalf("the move order was not attached: held=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the armed autocast did not reach an actor on the tick its move order arrived: "+
			"%d pending", len(w.bookCasts))
	}

	// And the order survives the release.
	for n := 0; n < 32 && len(w.bookCasts) != 0; n++ {
		Step(w, nil)
	}
	if spAt(t, w, 2).HP >= 100 {
		t.Fatal("the autocast never released")
	}
	if e := spAt(t, w, 1); !e.HasTarget || e.TargetX != 2 || e.TargetY != 9 {
		t.Errorf("the release consumed the movement order: held=%v (%d,%d)",
			e.HasTarget, e.TargetX, e.TargetY)
	}
}

// TestACommandedAutocasterWalksAndShoots is the movement half of the same
// fixture, and it is the deeper form of the order regression: attaching the
// destination is not obeying the order if the actor never takes a step.
//
// A book cast owns the body through its WIND-UP alone. Recovery gates the next
// action, which is actorActionBusy's business, and not the walk. While the
// mover stood down for recovery as well, an armed row with a target in range
// re-armed on the tick recovery reached zero — stepAutoCasts runs before the
// mover — and the actor covered no ground at all. The control is the same
// world with nothing armed.
func TestACommandedAutocasterWalksAndShoots(t *testing.T) {
	t.Parallel()

	build := func(auto uint16) *World {
		caster := acCaster(1, 2, 2, 5000, 1<<1, auto)
		caster.AttackCharge, caster.AttackRelax = 12, 4
		caster.Owner = 1
		enemy := spEnt(2, 4, 4)
		enemy.Owner, enemy.HP, enemy.MaxHP = 2, 100000, 100000
		return hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)
	}

	armed, control := build(1), build(0)
	for _, w := range []*World{armed, control} {
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 2, Y: 9}})
		for n := 0; n < 200; n++ {
			Step(w, nil)
		}
	}

	if e := spAt(t, control, 1); e.X != 2 || e.Y != 9 {
		t.Fatalf("the control mage is at (%d,%d) — the fixture itself does not walk", e.X, e.Y)
	}
	if e := spAt(t, armed, 1); e.X != 2 || e.Y != 9 {
		t.Errorf("the commanded autocaster is at (%d,%d) after 201 ticks, want the order's own "+
			"(2,9) — an armed row with a target in range must not cost the actor its walk", e.X, e.Y)
	}
	if got := spAt(t, armed, 1).Mana; got >= 5000 {
		t.Errorf("the commanded autocaster spent no mana (%d) — it walked instead of walking and "+
			"casting, which is not what this pins", got)
	}
}

// TestAMoveOrderDoesNotHoldCastRecoveryStill keeps DIV-028 while the recovery
// is now carried by the retained lifecycle: a producer that can claim a
// recovering actor must neither freeze the interval nor freeze the walk.
func TestAMoveOrderDoesNotHoldCastRecoveryStill(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	caster.AttackRelax = 12
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, spEnt(2, 3, 0))

	w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: 1,
		Phase: bookRelaxing, Remaining: 12, Complete: true, Retained: true}}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Phase != bookRelaxing || w.bookCasts[0].Remaining == 0 {
		t.Fatalf("the fixture produced no retained recovery interval: %+v", w.bookCasts)
	}

	for n := 0; n < 40 && len(w.bookCasts) != 0; n++ {
		Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: int32(n % 3), Y: 5}})
	}
	if len(w.bookCasts) != 0 {
		t.Errorf("retained recovery still stands after 40 ticks of movement orders: %+v", w.bookCasts)
	}
	if e := spAt(t, w, 1); e.Y == 0 {
		t.Error("the recovering caster did not begin its commanded walk")
	}
}

// TestACastingMemberStillSuppliesItsGroupsSight is G1. decide reads the group's
// member list for the guard centroid, and candidates reads it for the decider
// and for the shared group sight stamp. Removing a member that began a cast
// therefore took that member's vision out of the same pass: a group whose only
// long-sighted unit is its mage stopped seeing anything the mage alone could
// see, on every tick the mage cast.
func TestACastingMemberStillSuppliesItsGroupsSight(t *testing.T) {
	mage := Entity{ID: 1, X: 5, Y: 5, Owner: 9, HP: 20, MaxHP: 20, ScanRange: 10,
		Mind: 30, MaxMana: 50, Mana: 50, KnownSpells: 1 << 1, DyingTime: 200}
	fighter := engFighter(2, 9, 5, 6)
	fighter.ScanRange = 2
	hostile := engFighter(3, SelfSlot, 5, 12)
	hostile.ScanRange = 1

	rel := engRel(t, [3]uint32{9, SelfSlot, relationHostile}, [3]uint32{SelfSlot, 9, relationHostile})
	w, err := NewStockedSpelledWorld(1, engBounds, ModeCanonical, Terrain{},
		[]Entity{mage, fighter, hostile}, nil, rel, nil, nil, []SpellRule{ogArrow()})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}

	engRun(w, 1)

	if len(w.bookCasts) == 0 && spAt(t, w, 1).CastWait == 0 {
		t.Fatalf("the fixture's mage never cast, so it cannot show what a casting member costs "+
			"its group: %+v", spAt(t, w, 1))
	}
	if v, held := engVictim(w, 2); !held || v != 3 {
		t.Errorf("the short-sighted member holds victim %v/%v, want the hostile (3) its casting "+
			"mage could see for it", v, held)
	}
}
