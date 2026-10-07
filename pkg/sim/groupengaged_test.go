package sim

import "testing"

// TestAGroupStruckFromBeyondItsNoticeCircleAnswersWithEveryMember shoots one
// member of a guarding group from a cell far outside the group's frozen notice
// circle and asks whether the members, all of them, take the shooter.
//
// The shooter reaches the member by an ordinary attack order and the hostility
// between the two slots comes from the blow itself, so nothing here is written
// into the world by hand. Before the fix the circle clipped the shooter from
// the candidate list, every member scored nothing, and the group stood while
// one of its own was shot.
func TestAGroupStruckFromBeyondItsNoticeCircleAnswersWithEveryMember(t *testing.T) {
	t.Parallel()

	members := []Entity{engFighter(1, 2, 10, 10), engFighter(2, 2, 11, 10), engFighter(3, 2, 10, 11)}
	shooter := engFighter(4, SelfSlot, 30, 10)
	shooter.Reach = 25
	shooter.DamageBase = 1
	w := engWorld(t, Relations{}, append(members, shooter)...)

	_, base, _ := w.groupState(2, 0)
	if radius := int64(noticeRadius(base)); radius >= 20 {
		t.Fatalf("the fixture's notice radius is %d, want it under the 20 cells the shooter stands at", radius)
	}

	Step(w, []Command{Attack(4, 1)})
	var answered [4]bool
	for i := 0; i < 120; i++ {
		Step(w, nil)
		for _, id := range []EntityID{1, 2, 3} {
			if victim, held := engVictim(w, id); held && victim == 4 {
				answered[id] = true
			}
		}
	}
	if struck := spAt(t, w, 1); struck.HP == struck.MaxHP {
		t.Fatal("the shooter never struck the member, so the fixture proves nothing")
	}
	for _, id := range []EntityID{1, 2, 3} {
		if !answered[id] {
			t.Errorf("member %d never took the shooter that was striking its group from outside the notice circle", id)
		}
	}
}

// TestAGroupWalkingHomeAnswersTheFightItIsDrawnInto lets a guarding group chase
// a hostile decoy off its posts, kill it, and start the walk home, and then has
// a player unit strike one member on the way. A member walking home holds a
// destination and no victim, which the decision pass reads as a commanded unit
// and leaves out, so before the fix a group made only of such members took no
// decision at all and walked on past the fight it was in.
func TestAGroupWalkingHomeAnswersTheFightItIsDrawnInto(t *testing.T) {
	t.Parallel()

	first, second := engFighter(1, 2, 5, 5), engFighter(2, 2, 5, 6)
	first.Speed, second.Speed = 8, 8
	decoy := engFighter(3, 3, 10, 5)
	decoy.HP, decoy.MaxHP = 5, 5
	striker := engFighter(4, SelfSlot, 5, 9)
	striker.HP, striker.MaxHP, striker.DamageBase = 1<<20, 1<<20, 1
	w := engWorld(t, engRel(t, [3]uint32{2, 3, 1}, [3]uint32{3, 2, 2}), first, second, decoy, striker)

	walkingHome := func(id EntityID) bool {
		m := spAt(t, w, id)
		return m.HasTarget && !m.HasAttackTarget && m.TargetX == m.PostX && m.TargetY == m.PostY &&
			(m.X != m.PostX || m.Y != m.PostY)
	}
	for i := 0; i < 1500 && !(walkingHome(1) && walkingHome(2)); i++ {
		Step(w, nil)
	}
	if !walkingHome(1) || !walkingHome(2) {
		t.Fatalf("fixture: the group is not on its way home: %+v %+v", spAt(t, w, 1), spAt(t, w, 2))
	}

	Step(w, []Command{Attack(4, 1)})
	for i := 0; i < 40; i++ {
		Step(w, nil)
	}
	if struck := spAt(t, w, 1); struck.HP == struck.MaxHP {
		t.Fatal("the player unit never struck the member, so the fixture proves nothing")
	}
	for _, id := range []EntityID{1, 2} {
		if m := spAt(t, w, id); m.X == m.PostX && m.Y == m.PostY {
			t.Fatalf("fixture: member %d reached its post before the check", id)
		}
		if victim, held := engVictim(w, id); !held || victim != 4 {
			t.Errorf("member %d walking home holds victim %v/%v, want the unit striking its group (entity 4)", id, victim, held)
		}
	}
}

// TestACasterWalksInUntilItSeesItsVictimThenCasts orders a weapon-spell carrier
// against a victim inside the spell's range and outside the carrier's own
// sight. The release refuses a victim the carrier cannot see, so before the fix
// the carrier stopped at the spell's range and stood there without casting.
func TestACasterWalksInUntilItSeesItsVictimThenCasts(t *testing.T) {
	t.Parallel()

	caster := wpnCaster(1, 0, 0, 1, 30, 1, 0)
	caster.ScanRange = 5
	caster.Speed = 40
	victim := spEnt(2, 7, 0)
	w := spWorld(t, 1, []SpellRule{wpnRule(6, 6, 7)}, caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	for i := 0; i < 240; i++ {
		Step(w, nil)
	}

	a := spAt(t, w, 1)
	dist := (cell{x: a.X, y: a.Y}).chebyshevTo(cell{x: 7, y: 0})
	if dist > 5 {
		t.Errorf("the caster stands %d cells from a victim it cannot see from beyond 5, want it walked into sight", dist)
	}
	if hp := spAt(t, w, 2).HP; hp == 100 {
		t.Error("the caster never cast at a victim inside the spell's range")
	}
}

// TestACasterBesideAHiddenVictimWaitsForSight orders a weapon-spell carrier
// against a victim standing next to it on ground that hides it. The carrier
// cannot come closer, so it stays on its cell, loads no wind-up and casts
// nothing, as it did before a carrier that cannot see its victim began to walk
// in.
func TestACasterBesideAHiddenVictimWaitsForSight(t *testing.T) {
	t.Parallel()

	caster := wpnCaster(1, 10, 10, 1, 30, 2, 1)
	caster.Owner, caster.ScanRange, caster.Facing = 1, 1, 64
	victim := spEnt(2, 11, 10)
	victim.Owner = 2
	w := visibilityWorld(t, heightCell(11, 10, 127), wpnRule(6, 6, 5), caster, victim)

	Step(w, []Command{cbOrder(1, 2)})
	for i := 0; i < 24; i++ {
		Step(w, nil)
		if a := spAt(t, w, 1); a.X != 10 || a.Y != 10 || a.AttackPhase != AttackReady {
			t.Fatalf("tick %d: the caster beside its hidden victim is at %d,%d in phase %d, want it standing on 10,10 and ready",
				i, a.X, a.Y, a.AttackPhase)
		}
	}
	if hp := spAt(t, w, 2).HP; hp != 100 {
		t.Errorf("the victim's health is %d, want 100: a hidden victim is not cast at", hp)
	}
}

// TestAGroupDoesNotAnswerAnAttackOrderTheFoeHasNotYetReached gives a player unit
// an attack order on a member from far outside the group's sight and stops
// before the unit has walked into reach. The group answers a foe striking at
// it, not an order it cannot see, so no member takes the unit yet.
func TestAGroupDoesNotAnswerAnAttackOrderTheFoeHasNotYetReached(t *testing.T) {
	t.Parallel()

	members := []Entity{engFighter(1, 2, 10, 10), engFighter(2, 2, 11, 10), engFighter(3, 2, 10, 11)}
	foe := engFighter(4, SelfSlot, 45, 10)
	rel := engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 1})
	w := engWorld(t, rel, append(members, foe)...)

	Step(w, []Command{Attack(4, 1)})
	for i := 0; i <= scriptPassPhase+scriptCycle; i++ {
		Step(w, nil)
	}
	if f := spAt(t, w, 4); (cell{x: f.X, y: f.Y}).chebyshevTo(cell{x: 10, y: 10}) <= engSight+noticeMargin {
		t.Fatalf("fixture: the foe is already at %d,%d, inside the group's reach of notice", f.X, f.Y)
	}
	for _, id := range []EntityID{1, 2, 3} {
		if victim, held := engVictim(w, id); held {
			t.Errorf("member %d took %d for an attack order that had not reached the group", id, victim)
		}
	}
}
