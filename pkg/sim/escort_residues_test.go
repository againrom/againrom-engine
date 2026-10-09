package sim

import "testing"

func residueDefender(t *testing.T, hp, mana int32, percent uint32) *World {
	t.Helper()
	charge := laFighter(1, 2, 7, 20, 20)
	charge.HP, charge.MaxHP = hp, 100
	caster := spMage(2, 22, 20, 100, 50, mana, 1<<6)
	caster.Owner, caster.Group, caster.ScanRange = 2, 7, 10
	enemy := laFighter(3, 3, 9, 23, 20)
	w, err := NewStockedSpelledWorld(7, engBounds, ModeCanonical, Terrain{},
		[]Entity{charge, caster, enemy}, nil, engRel(t, [3]uint32{2, 3, 1}), nil, nil, []SpellRule{hlHeal()})
	if err != nil {
		t.Fatal(err)
	}
	w.ImportAutoHealing(2, percent)
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Unit: 1, HasUnit: true, Args: [scriptParams]int32{subCommandDefend, 3}})
	return w
}

func TestDefenderHealConditionsAndFightFallback(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hp, mana int32
		percent  uint32
		cast     bool
	}{
		{"clear reserve any damage", 99, 10, 0, true},
		{"nonzero reserve below half", 49, 10, 100, true},
		{"half health boundary", 50, 10, 100, false},
		{"full health", 100, 10, 0, false},
		{"insufficient mana", 49, 9, 0, false},
		{"reserve upper boundary", 49, 10, 106, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := residueDefender(t, tc.hp, tc.mana, tc.percent)
			w.armDefend(1)
			_, casting := w.bookCastIndex(2)
			if casting != tc.cast {
				t.Fatalf("casting=%v want %v; victim=%d", casting, tc.cast, w.entities[1].AttackTarget)
			}
			if tc.cast {
				c := w.bookCasts[0]
				if c.Target != 1 || c.Spell != 6 || w.entities[1].HasAttackTarget {
					t.Fatalf("heal did not precede cover: cast=%+v", c)
				}
				w.armDefend(1)
				if w.entities[1].HasAttackTarget {
					t.Fatal("cover overwrote an admitted heal")
				}
			} else if !w.entities[1].HasAttackTarget || w.entities[1].AttackTarget != 3 {
				t.Fatal("refused heal did not fall through to cover")
			}
		})
	}
}

func TestIdleEscortTurnsBetweenActorPasses(t *testing.T) {
	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 22, 20))
	w.armFollow(1)
	before := w.entities[1].Facing
	for range 640 {
		Step(w, nil)
		if w.entities[1].Facing != before {
			return
		}
	}
	t.Fatal("idle escort retained its facing for 640 sub-ticks")
}

func TestClosingEscortReaimsAndStopsBetweenActorPasses(t *testing.T) {
	for _, sub := range []int32{subCommandDefend, subCommandFollow} {
		w := esWorld(t, sub, 3, engRel(t),
			laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 12, 20))
		w.actorPass()
		w.entities[0].Y = 24
		Step(w, nil)
		if e := w.entities[1]; !e.HasTarget || e.TargetY != 24 {
			t.Fatalf("state %d retained stale destination (%d,%d)", sub, e.TargetX, e.TargetY)
		}
		w.entities[1].X, w.entities[1].Y = 17, 24
		w.entities[1].Transit, w.entities[1].TransitTotal = 0, 0
		w.entities[1].clearTurn()
		Step(w, nil)
		if w.entities[1].HasTarget {
			t.Fatal("closing escort kept walking inside its stop distance")
		}
		w.entities[0].X = 24
		w.entities[1].clearTurn()
		Step(w, nil)
		if !w.entities[1].HasTarget || w.entities[1].TargetX != 24 {
			t.Fatal("stopped close order did not resume on the next sub-tick")
		}
	}
}

func TestEscortIdleBlowBypassesRandomGateAndClearsAlarm(t *testing.T) {
	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 22, 20), laFighter(3, 3, 9, 40, 40))
	w.armFollow(1)
	w.flipOnBlow(2, 1)
	if !w.entities[1].EscortTurnPending {
		t.Fatal("blow did not set the idle alarm")
	}
	before, state := w.entities[1].Facing, w.RandomState()
	Step(w, nil)
	if w.entities[1].EscortTurnPending || w.entities[1].Facing == before || w.RandomState() != state+gamma {
		t.Fatal("forced idle turn did not consume exactly one draw and clear the alarm")
	}
}

func TestEscortCloseDoesNotReaimDuringTransitOrCast(t *testing.T) {
	w := esWorld(t, subCommandFollow, 3, engRel(t),
		laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 12, 20))
	w.actorPass()
	w.entities[1].Transit, w.entities[1].TransitTotal = 2, 3
	w.entities[0].Y = 24
	Step(w, nil)
	if w.entities[1].TargetY != 20 || w.entities[1].Transit != 1 {
		t.Fatal("escort executor replaced a crossing")
	}
	w.entities[1].Transit, w.entities[1].TransitTotal = 0, 0
	w.entities[1].CastWait = 2
	Step(w, nil)
	if w.entities[1].TargetY != 20 {
		t.Fatal("escort executor replaced cast-owned facing")
	}
}

func TestFollowNeverHealsAndOutOfRangeDefendCloses(t *testing.T) {
	for _, outside := range []bool{false, true} {
		w := residueDefender(t, 49, 20, 0)
		if outside {
			w.entities[1].X = 30
			w.armDefend(1)
		} else {
			w.entities[1].ActorState = actorStateFollow
			w.armFollow(1)
		}
		if len(w.bookCasts) != 0 {
			t.Fatal("heal escaped the in-range defend fork")
		}
	}
}

func TestDefenderHealManaGateUsesBookInstanceForNonMage(t *testing.T) {
	for _, mana := range []int32{10, 11} {
		w := residueDefender(t, 49, mana, 0)
		e := &w.entities[1]
		e.MaxMana = 0
		e.CreatureSpells[0] = CreatureSpell{ID: 6, Threshold: 100}
		e.Book = Spellbook{State: BookPresent}
		e.Book.Slots[5] = BookSpell{Range: 6, ManaCost: 11}
		w.armDefend(1)
		if _, cast := w.bookCastIndex(2); cast != (mana == 11) {
			t.Fatalf("mana %d: cast %v; instance cost 11, table cost 10", mana, cast)
		}
	}
}

func TestSavedEscortDispatchKeepsPendingCastOutsideRange(t *testing.T) {
	for _, state := range []uint8{actorStateDefend, actorStateFollow} {
		w := esWorld(t, subCommandFollow, 3, engRel(t),
			laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, 12, 20))
		w.entities[1].ActorState = state
		pending := PendingOrder{Kind: PendingActorCast, Target: 1, Spell: 6}
		w.entities[1].PendingOrder = pending
		w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{{Entity: 2, State: uint32(state), Authored: true}}}
		w.savedActorDispatch(1)
		if w.entities[1].PendingOrder != pending || w.entities[1].HasTarget || w.savedGroups.Orders[0].Raw[8] != 0 {
			t.Fatal("source escort close replaced a pending cast")
		}
	}
}
