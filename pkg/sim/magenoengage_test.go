package sim

import "testing"

// mnWorld is a hero of the player at (16, 16), a player's mage at (16, 18)
// with Heal and Fire Arrow known, the given reach, mana and armed spell and
// no damage of its own, and a hostile at (17, 16) beside the hero and ordered
// onto it.
func mnWorld(t *testing.T, reach uint8, mana int32, auto uint16, heroHP int32) *World {
	t.Helper()
	hero := spEnt(1, 16, 16)
	hero.Owner, hero.Reach, hero.ScanRange, hero.HP = SelfSlot, 1, sightRings, heroHP
	mage := acCaster(2, 16, 18, mana, 1<<6|1<<1, auto)
	mage.Owner, mage.Reach = SelfSlot, reach
	foe := spEnt(3, 17, 16)
	foe.Owner, foe.Reach = 2, 1
	foe.DamageBase, foe.AlwaysHits, foe.ScanRange = 1, true, 10
	foe.HP, foe.MaxHP = 30000, 30000
	foe.HasAttackTarget, foe.AttackTarget = true, 1
	w, err := NewStockedSpelledWorld(71, Bounds{Width: 32, Height: 32}, ModeCanonical,
		Terrain{}, []Entity{hero, mage, foe}, nil, acEnemies(t), nil, nil, []SpellRule{hlHeal(), hlArrow()})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w
}

// mnMelee reports whether the mage at id holds a live attack order on the
// hostile 3 or runs a strike.
func mnMelee(t *testing.T, w *World, id EntityID) bool {
	t.Helper()
	m := spAt(t, w, id)
	return m.HasAttackTarget && m.AttackTarget == 3 && !m.PursuitIdle || m.AttackPhase != AttackReady
}

func TestAStafflessPlayerMageDefendingTheHeroDoesNotMelee(t *testing.T) {
	t.Parallel()

	w := mnWorld(t, 1, 50, 1, 100)
	Step(w, []Command{GroupDefend(2, 1, 77)})
	arrows := 0
	for k := 0; k < 400; k++ {
		for _, c := range StepObserved(w, nil) {
			if c.Caster == 2 && c.Spell == 1 && c.Target == 3 {
				arrows++
			}
		}
		if mnMelee(t, w, 2) {
			m := spAt(t, w, 2)
			t.Fatalf("tick %d: the staff-less defending mage went into melee: at (%d, %d), victim %v/%d idle %v",
				w.Tick(), m.X, m.Y, m.HasAttackTarget, m.AttackTarget, m.PursuitIdle)
		}
	}
	if e := spAt(t, w, 2); e.ActorState != actorStateDefend || cellOf(&e).chebyshevTo(cellOf(&w.entities[indexOfEntity(w.entities, 1)])) > 3 {
		t.Fatalf("the mage left the hero: state %d at (%d, %d)", e.ActorState, e.X, e.Y)
	}
	// The armed Fire Arrow keeps its automatic cast moments (DIV-3191).
	if arrows == 0 {
		t.Error("the defending mage never cast its armed Fire Arrow at the hero's attacker")
	}
}

func TestAStafflessPlayerMageDefendingAWoundedHeroHealsIt(t *testing.T) {
	t.Parallel()

	w := mnWorld(t, 1, 50, 0, 40)
	Step(w, []Command{GroupDefend(2, 1, 77)})
	healed := false
	for k := 0; k < 200; k++ {
		for _, c := range StepObserved(w, nil) {
			if c.Caster == 2 && c.Spell == 6 && c.Target == 1 {
				healed = true
			}
		}
		if mnMelee(t, w, 2) {
			t.Fatalf("tick %d: the staff-less defending mage went into melee", w.Tick())
		}
	}
	if !healed {
		t.Fatal("the defending mage never healed the wounded hero")
	}
}

func TestAPlayerMageWithReachDefendingTheHeroStillEngages(t *testing.T) {
	t.Parallel()

	w := mnWorld(t, 2, 0, 0, 100)
	Step(w, []Command{GroupDefend(2, 1, 77)})
	for k := 0; k < 200; k++ {
		Step(w, nil)
		if m := spAt(t, w, 2); m.HasAttackTarget && m.AttackTarget == 3 && !m.PursuitIdle {
			return
		}
	}
	t.Fatal("a defending mage of reach 2 never engaged the hero's attacker")
}

// mnStand is a player's mage of reach 1 with no mana, damage 5, in its
// group's default order, and a passive hostile standing beside it.
func mnStand(t *testing.T, reach uint8) *World {
	t.Helper()
	mage := acCaster(2, 16, 18, 0, 0, 0)
	mage.Owner, mage.Reach, mage.DamageBase, mage.AlwaysHits = SelfSlot, reach, 5, true
	foe := spEnt(3, 16, 19)
	foe.Owner, foe.Reach = 2, 1
	foe.HP, foe.MaxHP = 30000, 30000
	w, err := NewStockedSpelledWorld(71, Bounds{Width: 32, Height: 32}, ModeCanonical,
		Terrain{}, []Entity{mage, foe}, nil, acEnemies(t), nil, nil, []SpellRule{hlHeal(), hlArrow()})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w
}

func TestAStafflessPlayerMageUnderStandGroundDoesNotStrikeAnAdjacentHostile(t *testing.T) {
	t.Parallel()

	staffless, reaching := mnStand(t, 1), mnStand(t, 2)
	for k := 0; k < 200; k++ {
		Step(staffless, nil)
		Step(reaching, nil)
		if m := spAt(t, staffless, 2); m.HasAttackTarget && !m.PursuitIdle {
			t.Fatalf("tick %d: the staff-less mage took the adjacent hostile on its own", staffless.Tick())
		}
	}
	if spAt(t, staffless, 3).HP != 30000 {
		t.Error("the staff-less mage struck the adjacent hostile on its own")
	}
	if spAt(t, reaching, 3).HP == 30000 {
		t.Error("control: the mage of reach 2 never struck the adjacent hostile")
	}
}

func TestAPlayersAttackCommandStillSendsAStafflessMage(t *testing.T) {
	t.Parallel()

	w := mnStand(t, 1)
	Step(w, []Command{Attack(2, 3)})
	for k := 0; k < 200 && spAt(t, w, 3).HP == 30000; k++ {
		Step(w, nil)
	}
	if spAt(t, w, 3).HP == 30000 {
		t.Fatal("the player's attack command did not send the staff-less mage at the hostile")
	}
}

func TestAStafflessPlayerMageAcquiringInPlaceIdlesItsPick(t *testing.T) {
	t.Parallel()

	w := mnStand(t, 1)
	Step(w, []Command{GroupDefend(2, 2, 77)})
	idled := false
	for k := 0; k < 200; k++ {
		Step(w, nil)
		m := spAt(t, w, 2)
		if m.HasAttackTarget && !m.PursuitIdle || m.AttackPhase != AttackReady {
			t.Fatalf("tick %d: the acquiring staff-less mage engaged its pick", w.Tick())
		}
		if m.HasAttackTarget && m.AttackTarget == 3 && m.PursuitIdle {
			idled = true
			if o := w.savedOrder(2); o == nil || o.Raw[8] != 0 {
				t.Fatalf("tick %d: the idled pick's order byte is not 0", w.Tick())
			}
		}
	}
	if !idled {
		t.Error("the acquisition never wrote the adjacent hostile as an idle pick")
	}
	if spAt(t, w, 3).HP != 30000 {
		t.Error("the acquiring staff-less mage struck the adjacent hostile")
	}
}

func TestAStafflessPlayerMageAtItsPostRefusesTheBlockAndKeepsTheLeash(t *testing.T) {
	t.Parallel()

	w := mnStand(t, 1)
	i := indexOfEntity(w.entities, 2)
	o := &SavedActorOrder{Entity: 2}
	o.Raw[0], o.Raw[1], o.Raw[8] = 16, 18, 0xb
	w.savedGuard(i, o)
	if e := w.entities[i]; e.HasAttackTarget || e.HasTarget || o.Raw[8] != 0xb {
		t.Fatalf("at its post the mage acted on the refused block: victim %v, walk %v, order %d", e.HasAttackTarget, e.HasTarget, o.Raw[8])
	}
	w.entities[i].X = 16 + int32(w.entities[i].ScanRange)
	w.savedGuard(i, o)
	if e := w.entities[i]; e.HasAttackTarget || !e.HasTarget || e.TargetX != 16 || e.TargetY != 18 || o.Raw[8] != 1 {
		t.Fatalf("past its leash the mage did not walk home: victim %v, walk %v to (%d, %d), order %d", e.HasAttackTarget, e.HasTarget, e.TargetX, e.TargetY, o.Raw[8])
	}
}
